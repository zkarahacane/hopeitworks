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

// AgentRunAction implements model.Action for running coding agents in containers.
// It supports two execution modes, selected by the agent's runtime kind:
//   - Callback mode: claude_code/opencode/cma runtimes use HTTP callbacks for logs/cost/status
//   - Legacy mode: no runtime kind (older images) uses Docker log streaming and exit code detection
type AgentRunAction struct {
	containerMgr    port.ContainerManager
	logStreamer     port.LogStreamer
	eventPub        port.EventPublisher
	storyRepo       port.StoryRepository
	projectRepo     port.ProjectRepository
	runRepo         port.RunRepository
	environmentRepo port.EnvironmentRepository
	sidecarMgr      port.SidecarManager
	renderer        port.TemplateRenderer
	costSvc         *service.CostService
	config          AgentConfig
	logger          *slog.Logger
	apiKeySvc       *service.APIKeyService
	tokenStore      port.ContainerTokenStore
	statusStore     port.CallbackStatusStore
	callbackURL     string
}

// NewAgentRunAction creates a new agent run action.
// The apiKeySvc, tokenStore, statusStore, and callbackURL parameters enable callback mode
// for the claude_code/opencode/cma runtimes. Pass nil/empty to disable callback mode.
//
// environmentRepo and sidecarMgr drive the per-run Environment: when the project has an
// Environment with sidecar services, sidecarMgr brings them up on an isolated per-run
// network and their connection strings are injected into the agent container. Both are
// nil-safe at the call sites; a project without an Environment behaves exactly as before.
func NewAgentRunAction(
	containerMgr port.ContainerManager,
	logStreamer port.LogStreamer,
	eventPub port.EventPublisher,
	storyRepo port.StoryRepository,
	projectRepo port.ProjectRepository,
	runRepo port.RunRepository,
	environmentRepo port.EnvironmentRepository,
	sidecarMgr port.SidecarManager,
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
		containerMgr:    containerMgr,
		logStreamer:     logStreamer,
		eventPub:        eventPub,
		storyRepo:       storyRepo,
		projectRepo:     projectRepo,
		runRepo:         runRepo,
		environmentRepo: environmentRepo,
		sidecarMgr:      sidecarMgr,
		renderer:        renderer,
		costSvc:         costSvc,
		config:          config,
		logger:          logger,
		apiKeySvc:       apiKeySvc,
		tokenStore:      tokenStore,
		statusStore:     statusStore,
		callbackURL:     callbackURL,
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

	// 5. Detect execution mode from the agent's runtime kind (not the image string).
	runtimeKind, _ := runCtx.Metadata["runtime_kind"].(string)
	isCallbackMode := a.isCallbackMode(runtimeKind, agentImage)

	// 6. Resolve the project's Environment and bring up its sidecar services.
	// Back-compat is HARD: a project without an Environment (GetByProjectID ->
	// NotFound) leaves env nil, no network/sidecar is created, and the container
	// is built exactly as before. Only env-level errors other than NotFound fail.
	env, err := a.resolveEnvironment(ctx, runCtx.ProjectID)
	if err != nil {
		return fmt.Errorf("resolve environment: %w", err)
	}

	// Launch sidecars on a per-run isolated network. Nil-safe: env==nil or no
	// services yields a nil sidecarCtx and is a no-op. Fail-fast on launch error.
	var sidecarCtx *port.SidecarContext
	if env != nil && len(env.Services) > 0 {
		sc, launchErr := a.sidecarMgr.Launch(ctx, runCtx.Run.ID, env)
		if launchErr != nil {
			return fmt.Errorf("launch sidecars: %w", launchErr)
		}
		sidecarCtx = sc
		// Teardown is GUARANTEED even if a later step fails. Cleanup runs on its
		// own detached, bounded context inside the SidecarManager, so it works
		// even when ctx is already cancelled.
		defer func() {
			if cleanupErr := a.sidecarMgr.Cleanup(context.Background(), sidecarCtx); cleanupErr != nil {
				a.logger.Warn("failed to clean up sidecars",
					"run_id", runCtx.Run.ID, "error", cleanupErr)
			}
		}()
	}

	// Build connection strings for the sidecar services (DATABASE_URL, etc.).
	// nil when there is no Environment — extraEnv stays nil and the container env
	// is byte-for-byte identical to the pre-Environment behaviour.
	extraEnv := buildConnStrings(env)

	// 7. Create container
	containerID, err := a.createContainer(ctx, runCtx, project, story, agentImage, prompt, branchName, extraEnv, sidecarCtx)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	defer a.cleanupContainer(containerID)

	// 8. Start container
	if err := a.containerMgr.Start(ctx, containerID); err != nil {
		return fmt.Errorf("start container: %w", err)
	}

	// 9. Persist container ID to run step
	a.persistContainerID(ctx, runCtx.RunStep.ID, containerID)

	// 10. Wait for completion using the appropriate mode
	var exitCode int
	if isCallbackMode {
		exitCode, err = a.waitForCallback(ctx, runCtx)
	} else {
		exitCode, err = a.streamAndWait(ctx, containerID, runCtx)
	}
	if err != nil {
		return fmt.Errorf("stream/wait: %w", err)
	}

	// 11. Check exit code
	if exitCode != 0 {
		return fmt.Errorf("agent exited with code %d", exitCode)
	}

	return nil
}

// buildClaudeMD generates a per-run CLAUDE.md string injected as CLAUDE_MD_CONTENT.
// The agent-runtime writes this to /workspace/repo/.claude/CLAUDE.md so Claude Code
// picks it up as project-level context.
//
// Role is resolved from RunStep.Config["role"] (set in the pipeline YAML config map).
// Known roles: "dev" / "implement" → implementer framing; "review" → reviewer framing.
// Unknown or empty role defaults to the implementer framing.
func buildClaudeMD(project *model.Project, role string, story *model.Story) string {
	projectName := project.Name
	projectDescription := ""
	if project.Description != nil {
		projectDescription = *project.Description
	}
	repoURL := ""
	if project.RepoURL != nil {
		repoURL = *project.RepoURL
	}

	storyRef := story.Key
	if story.Title != "" {
		storyRef = story.Key + " — " + story.Title
	}

	switch role {
	case "review", "reviewer":
		return "# Agent context — " + projectName + "\n\n" +
			"You are the **reviewer** agent. Review the changes made for story " + story.Key + " against the acceptance criteria.\n\n" +
			"## Project\n" +
			projectDescription + "\n" +
			"Repository: " + repoURL + "\n\n" +
			"## The changes to review\n" +
			"Inspect the diff for this story yourself with `git diff origin/main...HEAD` and `git log -p origin/main..HEAD`. Review only those changes.\n\n" +
			"## Review checklist\n" +
			"- Acceptance criteria are met.\n" +
			"- The project builds and tests pass (verify or note if you cannot).\n" +
			"- No secrets, tokens, or credentials are committed.\n" +
			"- No unrelated changes are included.\n" +
			"- Code follows existing style and conventions.\n\n" +
			"Be concise and specific. Flag blockers clearly.\n"
	default:
		// Covers "dev", "implement", "" and any unknown role.
		return "# Agent context — " + projectName + "\n\n" +
			"You are the **implementer** agent. Implement the task described in your prompt, scoped to story " + storyRef + ".\n\n" +
			"## Project\n" +
			projectDescription + "\n" +
			"Repository: " + repoURL + "\n\n" +
			"## Conventions\n" +
			"- Follow the existing structure, style, and conventions of the repository.\n" +
			"- Keep your changes scoped to the story. Do not refactor unrelated code.\n" +
			"- Never commit secrets, tokens, or credentials.\n\n" +
			"## Definition of done — MANDATORY before you finish\n" +
			"1. The project BUILDS. Run the build and make it succeed.\n" +
			"2. The TESTS pass. Run the test suite; everything must be green.\n" +
			"3. If the build or tests fail, FIX the code and re-run until they pass. Do NOT finish with a broken build or failing tests.\n"
	}
}

// resolveRole returns the agent role for the current run step.
// It reads RunStep.Config["role"] (set in the pipeline YAML config map, e.g. role: "dev").
// Falls back to an empty string (treated as implementer by buildClaudeMD).
func resolveRole(runCtx *model.RunContext) string {
	if runCtx.RunStep != nil && runCtx.RunStep.Config != nil {
		if r, ok := runCtx.RunStep.Config["role"]; ok {
			return r
		}
	}
	return ""
}

// createContainer builds ContainerOpts and creates the container.
// In callback mode (claude_code/opencode/cma runtimes), it injects CALLBACK_URL, AUTH_TOKEN,
// API_KEY, PROVIDER, MODEL, RUN_ID, and STEP_ID env vars instead of CLAUDE_CODE_OAUTH_TOKEN.
func (a *AgentRunAction) createContainer(
	ctx context.Context,
	runCtx *model.RunContext,
	project *model.Project,
	story *model.Story,
	agentImage, prompt, branchName string,
	extraEnv []string,
	sidecarCtx *port.SidecarContext,
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

	// Resolve role from step config and build per-run CLAUDE.md context.
	role := resolveRole(runCtx)

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
		// CLAUDE_MD_CONTENT is written to /workspace/repo/.claude/CLAUDE.md by the agent-runtime,
		// giving Claude Code role-aware project context for every run.
		// priorFailureContext appends a "Previous attempt" block when there is a prior
		// failed run for this story, so the agent can learn from earlier mistakes.
		"CLAUDE_MD_CONTENT=" + buildClaudeMD(project, role, story) + a.priorFailureContext(ctx, story.ID, runCtx.Run.ID),
	}

	runtimeKind, _ := runCtx.Metadata["runtime_kind"].(string)
	isCallback := a.isCallbackMode(runtimeKind, agentImage)

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

		// Generate a short-lived container token for callback auth. The token is bound
		// to the agent so the fetch-at-startup bundle endpoint can resolve this agent's
		// capabilities server-side. agentID is uuid.Nil when no agent is bound, which
		// yields an empty bundle (back-compat: behaves exactly as before).
		if a.tokenStore != nil {
			agentID := uuid.Nil
			if id := extractAgentID(runCtx); id != nil {
				agentID = *id
			}
			token, tokenErr := a.tokenStore.Create(ctx, runCtx.Run.ID, runCtx.RunStep.ID, agentID, 2*time.Hour)
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

	// Append sidecar connection strings LAST, preserving the existing env order so
	// the slice is byte-for-byte identical to before when extraEnv is nil.
	env = append(env, extraEnv...)

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

	// Dual-home the agent container: it keeps its shared NetworkName (API callback
	// / egress) AND attaches to the run network so it can reach the sidecars by
	// their service-name DNS alias. Only set when a run network actually exists,
	// so a project without an Environment keeps identical ContainerOpts.
	if sidecarCtx != nil && sidecarCtx.NetworkName != "" {
		opts.ExtraNetworks = []string{sidecarCtx.NetworkName}
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

// isCallbackMode returns true when the run should use HTTP callback mode for
// logs, cost, and status reporting.
//
// The primary signal is the agent's runtime kind: claude_code, opencode and cma
// all execute via the agent-runtime binary and report over the callback channel.
// Callback mode also requires a configured statusStore.
//
// Back-compat: when runtimeKind is empty — runs launched before the runtime_kind
// migration and resumed after deploy carry no runtime_kind in their persisted
// metadata — we fall back to the legacy image-substring heuristic one last time.
// New runs always thread runtime_kind, so this fallback fades out on its own.
func (a *AgentRunAction) isCallbackMode(runtimeKind, agentImage string) bool {
	if a.statusStore == nil {
		return false
	}
	switch runtimeKind {
	case model.RuntimeKindClaudeCode, model.RuntimeKindOpenCode, model.RuntimeKindCMA:
		return true
	case "":
		return strings.Contains(agentImage, "hopeitworks/agent-")
	default:
		return false
	}
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

// priorFailureContext looks up the most recent failed run for storyID (excluding
// currentRunID) and returns a markdown block describing the failure so the agent
// can avoid repeating the same mistake. Returns "" when there is no prior failure
// or when any lookup error occurs (memory is best-effort and must never fail the run).
func (a *AgentRunAction) priorFailureContext(ctx context.Context, storyID, currentRunID uuid.UUID) string {
	// Fetch a small window of recent runs for this story (descending by created_at).
	// A limit of 10 is generous enough to find the last failed run without a large scan.
	runs, err := a.runRepo.ListRunsByStory(ctx, storyID, 10, 0)
	if err != nil {
		a.logger.Debug("priorFailureContext: failed to list runs by story",
			"story_id", storyID, "error", err)
		return ""
	}

	// Find the most recent failed run that is not the current run.
	var failedRun *model.Run
	for _, r := range runs {
		if r.ID == currentRunID {
			continue
		}
		if r.Status == model.RunStatusFailed {
			failedRun = r
			break // runs are ordered DESC — first match is the most recent
		}
	}
	if failedRun == nil {
		return ""
	}

	// Retrieve the steps for the failed run to find the failure detail.
	steps, err := a.runRepo.ListRunStepsByRun(ctx, failedRun.ID)
	if err != nil {
		a.logger.Debug("priorFailureContext: failed to list run steps",
			"run_id", failedRun.ID, "error", err)
		return ""
	}

	// Pick the first step with status "failed"; fall back to the last step.
	var targetStep *model.RunStep
	for _, s := range steps {
		if s.Status == model.StepStatusFailed {
			targetStep = s
			break
		}
	}
	if targetStep == nil && len(steps) > 0 {
		targetStep = steps[len(steps)-1]
	}

	// Build the reason and log snippet.
	reason := "(no error message recorded)"
	if failedRun.ErrorMessage != nil && *failedRun.ErrorMessage != "" {
		reason = *failedRun.ErrorMessage
	} else if targetStep != nil && targetStep.ErrorMessage != nil && *targetStep.ErrorMessage != "" {
		reason = *targetStep.ErrorMessage
	}

	logSnippet := ""
	if targetStep != nil && targetStep.LogTail != nil && *targetStep.LogTail != "" {
		tail := *targetStep.LogTail
		const maxLogChars = 1500
		if len(tail) > maxLogChars {
			tail = tail[len(tail)-maxLogChars:]
		}
		logSnippet = tail
	}

	block := "\n\n## Previous attempt for this story FAILED — learn from it\n" +
		"Reason: " + reason + "\n"
	if logSnippet != "" {
		block += "Recent log output from the failed attempt:\n" + logSnippet + "\n"
	}
	block += "Do not repeat the mistake that caused this failure.\n"
	return block
}

// derefString safely dereferences a string pointer, returning empty string if nil.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
