package action

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

// AgentConfig holds configuration for agent container execution.
type AgentConfig struct {
	// DefaultImage is the Docker image for agent containers (e.g., "hopeitworks/agent:latest").
	DefaultImage string
	// DefaultMemory is the memory limit in bytes (e.g., 4GB = 4294967296).
	DefaultMemory int64
	// DefaultCPUs is the CPU limit (e.g., 2.0).
	DefaultCPUs float64
	// NetworkName is the Docker network for agent containers.
	NetworkName string
	// LogTailLines is the number of log lines to keep for error context.
	LogTailLines int
	// ClaudeMDPath is the path to the agent/claude-md/ directory.
	ClaudeMDPath string
}

// AgentRunAction implements model.Action for running Claude Code agents in containers.
type AgentRunAction struct {
	containerMgr port.ContainerManager
	logStreamer  port.LogStreamer
	eventPub     port.EventPublisher
	storyRepo    port.StoryRepository
	projectRepo  port.ProjectRepository
	runRepo      port.RunRepository
	templateSvc  *service.TemplateService
	composer     *CLAUDEMDComposer
	config       AgentConfig
	logger       *slog.Logger
}

// NewAgentRunAction creates a new agent run action.
func NewAgentRunAction(
	containerMgr port.ContainerManager,
	logStreamer port.LogStreamer,
	eventPub port.EventPublisher,
	storyRepo port.StoryRepository,
	projectRepo port.ProjectRepository,
	runRepo port.RunRepository,
	templateSvc *service.TemplateService,
	config AgentConfig,
	logger *slog.Logger,
) *AgentRunAction {
	return &AgentRunAction{
		containerMgr: containerMgr,
		logStreamer:  logStreamer,
		eventPub:     eventPub,
		storyRepo:    storyRepo,
		projectRepo:  projectRepo,
		runRepo:      runRepo,
		templateSvc:  templateSvc,
		composer:     NewCLAUDEMDComposer(config.ClaudeMDPath),
		config:       config,
		logger:       logger,
	}
}

// Name returns the action identifier.
func (a *AgentRunAction) Name() string {
	return "agent_run"
}

// Execute runs the agent in a container: fetches story, composes CLAUDE.md,
// renders prompt, creates container, streams logs, and waits for exit.
func (a *AgentRunAction) Execute(ctx context.Context, runCtx *model.RunContext) error {
	// 1. Fetch story
	story, err := a.storyRepo.GetByID(ctx, runCtx.StoryID)
	if err != nil {
		return fmt.Errorf("fetch story: %w", err)
	}

	// 2. Fetch project
	project, err := a.projectRepo.GetByID(ctx, runCtx.ProjectID)
	if err != nil {
		return fmt.Errorf("fetch project: %w", err)
	}

	// 3. Compose CLAUDE.md
	scope := ""
	if story.Scope != nil {
		scope = *story.Scope
	}
	claudeMD, err := a.composer.Compose(scope)
	if err != nil {
		return fmt.Errorf("compose CLAUDE.md: %w", err)
	}

	// 4. Resolve and render prompt template
	templateName := a.resolveTemplateName(runCtx)
	branchName, _ := runCtx.Metadata["branch_name"].(string)
	repoURL := ""
	if project.RepoURL != nil {
		repoURL = *project.RepoURL
	}

	tmplCtx := &model.TemplateContext{
		StoryKey:           story.Key,
		StoryTitle:         story.Title,
		StoryObjective:     derefString(story.Objective),
		TargetFiles:        story.TargetFiles,
		AcceptanceCriteria: derefString(story.AcceptanceCriteria),
		BranchName:         branchName,
		RepoURL:            repoURL,
	}
	prompt, err := a.templateSvc.RenderForStory(ctx, runCtx.ProjectID, templateName, tmplCtx)
	if err != nil {
		return fmt.Errorf("render prompt template %q: %w", templateName, err)
	}

	// 5. Create container
	containerID, err := a.createContainer(ctx, runCtx, project, story, claudeMD, prompt, branchName)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	defer a.cleanupContainer(containerID)

	// 6. Start container
	if err := a.containerMgr.Start(ctx, containerID); err != nil {
		return fmt.Errorf("start container: %w", err)
	}

	// 7. Persist container ID to run step
	a.persistContainerID(ctx, runCtx.RunStep.ID, containerID)

	// 8. Stream logs + wait for exit
	exitCode, err := a.streamAndWait(ctx, containerID, runCtx)
	if err != nil {
		return fmt.Errorf("stream/wait: %w", err)
	}

	// 9. Check exit code
	if exitCode != 0 {
		return fmt.Errorf("agent exited with code %d", exitCode)
	}

	return nil
}

// resolveTemplateName determines which prompt template to use.
func (a *AgentRunAction) resolveTemplateName(runCtx *model.RunContext) string {
	if name, ok := runCtx.Metadata["template_name"].(string); ok && name != "" {
		return name
	}
	return service.TemplateNameImplement
}

// createContainer builds ContainerOpts and creates the container.
func (a *AgentRunAction) createContainer(
	ctx context.Context,
	runCtx *model.RunContext,
	project *model.Project,
	story *model.Story,
	claudeMD, prompt, branchName string,
) (string, error) {
	repoURL := ""
	if project.RepoURL != nil {
		repoURL = *project.RepoURL
	}

	opts := model.ContainerOpts{
		Image:       a.config.DefaultImage,
		NetworkName: a.config.NetworkName,
		Memory:      a.config.DefaultMemory,
		CPUs:        a.config.DefaultCPUs,
		Env: []string{
			"CLAUDE_MD_CONTENT=" + claudeMD,
			"REPO_URL=" + repoURL,
			"BRANCH_NAME=" + branchName,
			"STORY_KEY=" + story.Key,
			"PROMPT_CONTENT=" + prompt,
			"GITHUB_TOKEN=" + os.Getenv("GITHUB_TOKEN"),
			"CLAUDE_CODE_OAUTH_TOKEN=" + os.Getenv("CLAUDE_CODE_OAUTH_TOKEN"),
		},
		Labels: map[string]string{
			"managed_by": "hopeitworks",
			"run_id":     runCtx.Run.ID.String(),
			"step_id":    runCtx.RunStep.ID.String(),
			"story_key":  story.Key,
		},
	}

	return a.containerMgr.Create(ctx, opts)
}

// streamAndWait starts log streaming, waits for container exit, and handles log tail.
func (a *AgentRunAction) streamAndWait(
	ctx context.Context,
	containerID string,
	runCtx *model.RunContext,
) (int, error) {
	runID := runCtx.Run.ID.String()
	stepID := runCtx.RunStep.ID.String()

	logCh, doneCh, err := a.logStreamer.StreamLogs(ctx, containerID, runID, stepID)
	if err != nil {
		return -1, fmt.Errorf("start log streaming: %w", err)
	}

	// Ring buffer for log tail
	tailSize := a.config.LogTailLines
	if tailSize <= 0 {
		tailSize = 50
	}
	logTail := make([]string, 0, tailSize)

	// Consume logs in goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for logEvent := range logCh {
			// Maintain ring buffer
			if len(logTail) >= tailSize {
				logTail = logTail[1:]
			}
			logTail = append(logTail, logEvent.Message)

			// Forward to event system
			a.publishLogEvent(ctx, runCtx, logEvent)
		}
	}()

	// Wait for container exit.
	// When the context is cancelled, the LogStreamer closes doneCh without sending
	// a value, so exitCode will be 0 (zero value). Check ctx.Err() to distinguish
	// a clean exit code 0 from a cancellation.
	exitCode := <-doneCh
	if err := ctx.Err(); err != nil {
		// Context was cancelled or deadline exceeded; propagate the context error.
		wg.Wait()
		return -1, err
	}

	// Wait for log goroutine to finish
	wg.Wait()

	// On failure, persist log tail
	if exitCode != 0 {
		tail := strings.Join(logTail, "\n")
		a.persistLogTail(ctx, runCtx.RunStep.ID, tail)
	}

	return exitCode, nil
}

// cleanupContainer stops and removes the container, logging errors without failing.
// It uses a dedicated timeout context to ensure cleanup never hangs indefinitely.
func (a *AgentRunAction) cleanupContainer(containerID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.containerMgr.Stop(ctx, containerID); err != nil {
		a.logger.Warn("failed to stop container during cleanup",
			"container_id", containerID, "error", err)
	}

	if err := a.containerMgr.Remove(ctx, containerID); err != nil {
		a.logger.Warn("failed to remove container during cleanup",
			"container_id", containerID, "error", err)
	}

	a.logger.Debug("container cleaned up", "container_id", containerID)
}

// persistContainerID saves the container ID to the run step.
func (a *AgentRunAction) persistContainerID(ctx context.Context, stepID uuid.UUID, containerID string) {
	if _, err := a.runRepo.UpdateRunStepContainerInfo(ctx, stepID, &containerID, nil); err != nil {
		a.logger.Warn("failed to persist container ID to run step",
			"step_id", stepID, "container_id", containerID, "error", err)
	}
}

// persistLogTail saves the log tail to the run step.
func (a *AgentRunAction) persistLogTail(ctx context.Context, stepID uuid.UUID, logTail string) {
	if _, err := a.runRepo.UpdateRunStepContainerInfo(ctx, stepID, nil, &logTail); err != nil {
		a.logger.Warn("failed to persist log tail to run step",
			"step_id", stepID, "error", err)
	}
}

// publishLogEvent publishes a log event to the event system.
func (a *AgentRunAction) publishLogEvent(ctx context.Context, runCtx *model.RunContext, logEvent model.LogEvent) {
	payload, err := json.Marshal(logEvent)
	if err != nil {
		a.logger.Error("failed to marshal log event", "error", err)
		return
	}

	event := model.Event{
		ID:         uuid.New(),
		ProjectID:  runCtx.ProjectID,
		EntityType: "log",
		EntityID:   runCtx.RunStep.ID,
		Action:     "emitted",
		Payload:    payload,
	}

	if err := a.eventPub.Publish(ctx, event); err != nil {
		a.logger.Warn("failed to publish log event", "error", err)
	}
}

// derefString safely dereferences a string pointer, returning empty string if nil.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
