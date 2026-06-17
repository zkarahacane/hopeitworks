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
	// DefaultMemory is the memory limit in bytes (e.g., 4GB = 4294967296).
	DefaultMemory int64
	// DefaultCPUs is the CPU limit (e.g., 2.0).
	DefaultCPUs float64
	// NetworkName is the Docker network for agent containers.
	NetworkName string
	// LogTailLines is the number of log lines to keep for error context.
	LogTailLines int
}

// AgentRunAction implements model.Action for running Claude Code agents in containers.
// It supports two execution modes:
//   - Callback mode: for hopeitworks/agent-* images, uses HTTP callbacks for logs/cost/status
//   - Legacy mode: for other images, uses Docker log streaming and exit code detection
type AgentRunAction struct {
	containerMgr port.ContainerManager
	logStreamer  port.LogStreamer
	eventPub     port.EventPublisher
	storyRepo    port.StoryRepository
	projectRepo  port.ProjectRepository
	runRepo      port.RunRepository
	renderer     port.TemplateRenderer
	costSvc      *service.CostService
	config       AgentConfig
	logger       *slog.Logger
	apiKeySvc    *service.APIKeyService
	tokenStore   port.ContainerTokenStore
	statusStore  port.CallbackStatusStore
	callbackURL  string
}

// NewAgentRunAction creates a new agent run action.
// The apiKeySvc, tokenStore, statusStore, and callbackURL parameters enable callback mode
// for hopeitworks/agent-* container images. Pass nil/empty to disable callback mode.
func NewAgentRunAction(
	containerMgr port.ContainerManager,
	logStreamer port.LogStreamer,
	eventPub port.EventPublisher,
	storyRepo port.StoryRepository,
	projectRepo port.ProjectRepository,
	runRepo port.RunRepository,
	renderer port.TemplateRenderer,
	costSvc *service.CostService,
	config AgentConfig,
	logger *slog.Logger,
	apiKeySvc *service.APIKeyService,
	tokenStore port.ContainerTokenStore,
	statusStore port.CallbackStatusStore,
	callbackURL string,
) *AgentRunAction {
	return &AgentRunAction{
		containerMgr: containerMgr,
		logStreamer:  logStreamer,
		eventPub:     eventPub,
		storyRepo:    storyRepo,
		projectRepo:  projectRepo,
		runRepo:      runRepo,
		renderer:     renderer,
		costSvc:      costSvc,
		config:       config,
		logger:       logger,
		apiKeySvc:    apiKeySvc,
		tokenStore:   tokenStore,
		statusStore:  statusStore,
		callbackURL:  callbackURL,
	}
}

// Name returns the action identifier.
func (a *AgentRunAction) Name() string {
	return "agent_run"
}

// Execute runs the agent in a container: fetches story, renders prompt from
// template_content metadata, creates container with agent_image, streams logs,
// and waits for exit.
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

	// 3. Resolve and render prompt template from metadata
	templateContent, _ := runCtx.Metadata["template_content"].(string)
	branchName, _ := runCtx.Metadata["branch_name"].(string)
	repoURL := ""
	if project.RepoURL != nil {
		repoURL = *project.RepoURL
	}

	var prompt string
	if templateContent != "" {
		tmplCtx := &model.TemplateContext{
			StoryKey:           story.Key,
			StoryTitle:         story.Title,
			StoryObjective:     derefString(story.Objective),
			TargetFiles:        story.TargetFiles,
			AcceptanceCriteria: derefString(story.AcceptanceCriteria),
			BranchName:         branchName,
			RepoURL:            repoURL,
		}

		// Inject retry context if present
		if ec, ok := runCtx.Metadata["error_context"].(string); ok {
			tmplCtx.ErrorContext = ec
		}
		if lt, ok := runCtx.Metadata["log_tail"].(string); ok {
			tmplCtx.LogTail = lt
		}

		prompt, err = a.renderer.Render(templateContent, tmplCtx)
		if err != nil {
			return fmt.Errorf("render prompt template: %w", err)
		}
	}

	// 4. Resolve agent image from metadata (required)
	agentImage, _ := runCtx.Metadata["agent_image"].(string)
	if agentImage == "" {
		return fmt.Errorf("agent_image is required in run metadata but was not set")
	}

	// 5. Detect execution mode: callback for hopeitworks/agent-* images, legacy otherwise
	isCallbackMode := a.isCallbackMode(agentImage)

	// 6. Create container
	containerID, err := a.createContainer(ctx, runCtx, project, story, agentImage, prompt, branchName)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	defer a.cleanupContainer(containerID)

	// 7. Start container
	if err := a.containerMgr.Start(ctx, containerID); err != nil {
		return fmt.Errorf("start container: %w", err)
	}

	// 8. Persist container ID to run step
	a.persistContainerID(ctx, runCtx.RunStep.ID, containerID)

	// 9. Wait for completion using the appropriate mode
	var exitCode int
	if isCallbackMode {
		exitCode, err = a.waitForCallback(ctx, runCtx)
	} else {
		exitCode, err = a.streamAndWait(ctx, containerID, runCtx)
	}
	if err != nil {
		return fmt.Errorf("stream/wait: %w", err)
	}

	// 10. Check exit code
	if exitCode != 0 {
		return fmt.Errorf("agent exited with code %d", exitCode)
	}

	return nil
}

// createContainer builds ContainerOpts and creates the container.
// For callback mode (hopeitworks/agent-* images), it injects CALLBACK_URL, AUTH_TOKEN,
// API_KEY, PROVIDER, MODEL, RUN_ID, and STEP_ID env vars instead of CLAUDE_CODE_OAUTH_TOKEN.
func (a *AgentRunAction) createContainer(
	ctx context.Context,
	runCtx *model.RunContext,
	project *model.Project,
	story *model.Story,
	agentImage, prompt, branchName string,
) (string, error) {
	repoURL := ""
	if project.RepoURL != nil {
		repoURL = *project.RepoURL
	}

	// Resolve git token dynamically from project config
	tokenEnvName := "GITHUB_TOKEN"
	if project.GitTokenEnv != nil && *project.GitTokenEnv != "" {
		tokenEnvName = *project.GitTokenEnv
	}
	gitToken := os.Getenv(tokenEnvName)

	env := []string{
		"REPO_URL=" + repoURL,
		"BRANCH_NAME=" + branchName,
		"STORY_KEY=" + story.Key,
		// PROMPT_CONTENT is consumed by the legacy shell entrypoint (agent/entrypoint.sh).
		// PROMPT is consumed by the agent-runtime Go binary in callback mode (config.Load).
		// Inject both so either image variant receives the rendered prompt under the name it reads.
		"PROMPT_CONTENT=" + prompt,
		"PROMPT=" + prompt,
		"GIT_TOKEN=" + gitToken,
		"GIT_PROVIDER=" + project.GitProvider,
		"GITHUB_TOKEN=" + gitToken,
	}

	isCallback := a.isCallbackMode(agentImage)

	if isCallback {
		// Callback mode: use per-user API key and container token auth
		provider := resolveProvider(runCtx)

		// Resolve the user's API key for the provider
		if a.apiKeySvc != nil && runCtx.UserID != uuid.Nil {
			apiKey, keyErr := a.apiKeySvc.DecryptKeyForUserProvider(ctx, runCtx.UserID, provider)
			if keyErr != nil {
				a.logger.Warn("failed to resolve API key for user/provider, container may fail auth",
					"user_id", runCtx.UserID, "provider", provider, "error", keyErr)
			} else {
				env = append(env, "API_KEY="+apiKey)
			}
		}

		// Generate a short-lived container token for callback auth
		if a.tokenStore != nil {
			token, tokenErr := a.tokenStore.Create(ctx, runCtx.Run.ID, runCtx.RunStep.ID, 2*time.Hour)
			if tokenErr != nil {
				return "", fmt.Errorf("create container token: %w", tokenErr)
			}
			env = append(env, "AUTH_TOKEN="+token)
		}

		env = append(env,
			"CALLBACK_URL="+a.callbackURL,
			"PROVIDER="+provider,
			"RUN_ID="+runCtx.Run.ID.String(),
			"STEP_ID="+runCtx.RunStep.ID.String(),
		)

		// Inject model (prefer per-step, then provider-default)
		if modelVal, ok := runCtx.Metadata["model"].(string); ok && modelVal != "" {
			env = append(env, "MODEL="+modelVal)
		}
	} else {
		// Legacy mode: use shared OAuth token
		env = append(env, "CLAUDE_CODE_OAUTH_TOKEN="+os.Getenv("CLAUDE_CODE_OAUTH_TOKEN"))

		// Inject per-step model when configured in pipeline YAML
		if modelVal, ok := runCtx.Metadata["model"].(string); ok && modelVal != "" {
			env = append(env, "MODEL="+modelVal)
		}
	}

	opts := model.ContainerOpts{
		Image:       agentImage,
		NetworkName: a.config.NetworkName,
		Memory:      a.config.DefaultMemory,
		CPUs:        a.config.DefaultCPUs,
		Env:         env,
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

	// Accumulate cost events emitted by the agent container.
	var costEvents []model.CostEvent

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

			// Accumulate cost events; do not forward cost lines to the event system.
			if logEvent.Type == "cost" {
				costEvents = append(costEvents, model.CostEvent{
					InputTokens:  logEvent.InputTokens,
					OutputTokens: logEvent.OutputTokens,
					Model:        logEvent.Model,
				})
				continue
			}

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

	// Persist the log tail for every step — not just failures — so an agent's
	// output stays visible in the UI after a successful run, and still feeds
	// error context into retries on failure.
	tail := strings.Join(logTail, "\n")
	a.persistLogTail(ctx, runCtx.RunStep.ID, tail)

	// Record accumulated cost events, regardless of exit code.
	// Cost recording failure is non-fatal.
	if len(costEvents) > 0 {
		agentID := extractAgentID(runCtx)
		if err := a.costSvc.RecordStepCost(ctx, runCtx.RunStep.ID, runCtx.ProjectID, costEvents, agentID); err != nil {
			a.logger.Warn("failed to record step cost",
				"step_id", stepID, "error", err)
		}
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

// extractAgentID extracts the agent ID from the run context.
// It checks RunStep.Config["agent_id"] and Metadata["agent_id"] for a valid UUID string.
// Returns nil if no agent ID is available.
func extractAgentID(runCtx *model.RunContext) *uuid.UUID {
	// Check step config first
	if runCtx.RunStep != nil && runCtx.RunStep.Config != nil {
		if raw, ok := runCtx.RunStep.Config["agent_id"]; ok && raw != "" {
			if id, err := uuid.Parse(raw); err == nil {
				return &id
			}
		}
	}

	// Fallback to metadata
	if runCtx.Metadata != nil {
		if raw, ok := runCtx.Metadata["agent_id"].(string); ok && raw != "" {
			if id, err := uuid.Parse(raw); err == nil {
				return &id
			}
		}
	}

	return nil
}

// isCallbackMode returns true if the agent image is a hopeitworks/agent-* image
// that supports HTTP callback mode for logs, cost, and status reporting.
func (a *AgentRunAction) isCallbackMode(agentImage string) bool {
	return a.statusStore != nil && strings.Contains(agentImage, "hopeitworks/agent-")
}

// waitForCallback waits for the agent container to report its exit status via
// the HTTP callback endpoint. Logs and cost events arrive asynchronously via
// separate callback endpoints and do not flow through this method.
func (a *AgentRunAction) waitForCallback(ctx context.Context, runCtx *model.RunContext) (int, error) {
	stepID := runCtx.RunStep.ID

	exitCode, errMsg, err := a.statusStore.WaitForStatus(ctx, stepID, 2*time.Hour)
	if err != nil {
		return -1, err
	}

	// Revoke the container token after completion
	if a.tokenStore != nil {
		revokeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if tokenStr, ok := a.findContainerToken(runCtx); ok {
			if revokeErr := a.tokenStore.Revoke(revokeCtx, tokenStr); revokeErr != nil {
				a.logger.Warn("failed to revoke container token",
					"step_id", stepID, "error", revokeErr)
			}
		}
	}

	if errMsg != "" {
		a.logger.Warn("agent container reported error",
			"step_id", stepID, "exit_code", exitCode, "error", errMsg)
	}

	return exitCode, nil
}

// findContainerToken looks for the AUTH_TOKEN in the run context metadata.
// This is a best-effort lookup used to revoke tokens after completion.
func (a *AgentRunAction) findContainerToken(_ *model.RunContext) (string, bool) {
	// The token is not stored in metadata; revocation is handled by TTL expiry.
	// This method exists as a hook for future token tracking if needed.
	return "", false
}

// resolveProvider resolves the AI provider for the current step from run context metadata.
// It checks step-specific provider metadata first, then falls back to "claude".
func resolveProvider(runCtx *model.RunContext) string {
	stepOrder := runCtx.RunStep.StepOrder
	if p, ok := runCtx.Metadata[fmt.Sprintf("step_%d_provider", stepOrder)].(string); ok && p != "" {
		return p
	}
	return "claude"
}

// derefString safely dereferences a string pointer, returning empty string if nil.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
