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
	// IsolateRuns mirrors cfg.Docker.IsolateRuns (DOCKER_ISOLATE_RUNS). When true,
	// the agent is single-homed on its per-run network instead of the shared
	// NetworkName, so the per-run network must ALWAYS be created — even for a
	// project with no sidecars. The Action therefore launches the SidecarManager
	// unconditionally under this flag; the manager creates the network and wires
	// the API callback in. When false, behaviour is unchanged (no per-run network
	// unless there are sidecars). The substrate (docker.Runtime) is configured with
	// the same flag so it makes the per-run network primary.
	IsolateRuns bool
	// LogTailLines is the number of log lines to keep for error context.
	LogTailLines int
	// CrashGrace bounds how long the Action waits for a callback status AFTER the
	// substrate process has exited, before declaring a crash. 0 => default (5s).
	CrashGrace time.Duration
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
	stackRepo       port.StackRepository
	renderer        port.TemplateRenderer
	costSvc         *service.CostService
	config          AgentConfig
	logger          *slog.Logger
	apiKeySvc       *service.APIKeyService
	tokenStore      port.ContainerTokenStore
	statusStore     port.CallbackStatusStore
	callbackURL     string

	// runtime, when non-nil, is the execution substrate the run is realised on
	// (port.AgentRuntime). main.go injects docker.Runtime by DEFAULT (Docker is no
	// longer special — it is an adapter behind the port), and may inject an
	// alternative substrate (e.g. microsandbox) under SUBSTRATE selection. When nil
	// the Action drives the ContainerManager directly via the legacy path, kept
	// byte-identical for back-compat. Set via WithAgentRuntime.
	runtime port.AgentRuntime
}

// Option configures optional behaviour of an AgentRunAction without widening the
// (already large) NewAgentRunAction positional signature. The legacy path uses no
// options; its constructor call-sites stay unchanged (variadic empty).
type Option func(*AgentRunAction)

// WithAgentRuntime selects the execution substrate the run dispatches through.
// When set, Execute realises the run via this port.AgentRuntime (Launch/Wait/Stop
// — Docker-as-adapter by default) while keeping callback-wait + token mint/revoke
// + the outcome model in the Action. Passing nil is a no-op (legacy direct path).
func WithAgentRuntime(rt port.AgentRuntime) Option {
	return func(a *AgentRunAction) {
		a.runtime = rt
	}
}

// NewAgentRunAction creates a new agent run action.
// The apiKeySvc, tokenStore, statusStore, and callbackURL parameters enable callback mode
// for the claude_code/opencode/cma runtimes. Pass nil/empty to disable callback mode.
//
// environmentRepo and sidecarMgr drive the per-run Environment: when the project has an
// Environment with sidecar services, sidecarMgr brings them up on an isolated per-run
// network and their connection strings are injected into the agent container. Both are
// nil-safe at the call sites; a project without an Environment behaves exactly as before.
//
// stackRepo resolves the image that runs the Environment's setup commands (build/migrate/
// seed/test): it is keyed by the Environment's first Stack, falling back to the agent
// image when the Environment declares no stack. It is nil-safe: when there are no commands
// to run it is never consulted.
func NewAgentRunAction(
	containerMgr port.ContainerManager,
	logStreamer port.LogStreamer,
	eventPub port.EventPublisher,
	storyRepo port.StoryRepository,
	projectRepo port.ProjectRepository,
	runRepo port.RunRepository,
	environmentRepo port.EnvironmentRepository,
	sidecarMgr port.SidecarManager,
	stackRepo port.StackRepository,
	renderer port.TemplateRenderer,
	costSvc *service.CostService,
	config AgentConfig,
	logger *slog.Logger,
	apiKeySvc *service.APIKeyService,
	tokenStore port.ContainerTokenStore,
	statusStore port.CallbackStatusStore,
	callbackURL string,
	opts ...Option,
) *AgentRunAction {
	a := &AgentRunAction{
		containerMgr:    containerMgr,
		logStreamer:     logStreamer,
		eventPub:        eventPub,
		storyRepo:       storyRepo,
		projectRepo:     projectRepo,
		runRepo:         runRepo,
		environmentRepo: environmentRepo,
		sidecarMgr:      sidecarMgr,
		stackRepo:       stackRepo,
		renderer:        renderer,
		costSvc:         costSvc,
		config:          config,
		logger:          logger,
		apiKeySvc:       apiKeySvc,
		tokenStore:      tokenStore,
		statusStore:     statusStore,
		callbackURL:     callbackURL,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
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
	//
	// Under East-West run isolation we launch the SidecarManager even when there
	// are no sidecars: the manager must still create the per-run network (and wire
	// the API callback into it) because the agent is single-homed on it. The
	// manager's Launch is a no-op only when isolation is OFF and there are no
	// services, so the guard here mirrors that to avoid a pointless call.
	var sidecarCtx *port.SidecarContext
	if (env != nil && len(env.Services) > 0) || a.config.IsolateRuns {
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

	// 6b. Run the Environment's setup commands (build → migrate → seed → test)
	// against the ready sidecars, BEFORE the agent container starts. No-op when
	// env==nil or env.Commands is empty (no ephemeral container is created). On
	// failure we return immediately: the deferred sidecar Cleanup runs and no
	// agent container is ever created.
	if err := a.runEnvironmentCommands(ctx, env, sidecarCtx, project, runCtx.Run.ID, branchName, agentImage, extraEnv); err != nil {
		return fmt.Errorf("run environment commands: %w", err)
	}

	// 6c. Resolve the role and mint the callback token ONCE, before either dispatch
	// path. Hoisting the mint here (out of buildAgentEnv) means the token lifecycle
	// is substrate-agnostic: minted once per run, then revoked with the SAME token
	// after the callback resolves (see waitForCallback). Returns nil for legacy /
	// non-callback runs.
	role := resolveRole(runCtx)
	callback, err := a.prepareCallback(ctx, runCtx, role)
	if err != nil {
		return fmt.Errorf("prepare callback: %w", err)
	}

	// 6d. Audit the rendered prompt on the agnostic channel (durable event bus +
	// operational log), once, before dispatch — so every substrate and both paths
	// audit it identically. Skipped when the prompt is empty.
	a.auditPrompt(ctx, runCtx, prompt, role)

	// 7. Substrate dispatch. When a runtime is injected (SUBSTRATE selection in
	// main.go, DockerRuntime by default), the run is realised THROUGH
	// port.AgentRuntime — Docker is no longer special. The callback-wait + token
	// mint/revoke + outcome model stay in the Action (see executeViaRuntime), so
	// this is NOT the rejected branch-beside-Docker fork. nil keeps the legacy
	// direct path (back-compat).
	if a.runtime != nil {
		return a.executeViaRuntime(ctx, runCtx, project, story, agentImage, prompt, branchName, extraEnv, sidecarCtx, callback)
	}

	// 7b. Legacy direct path (unchanged) — createContainer + Start + persist + wait.
	containerID, err := a.createContainer(ctx, runCtx, project, story, agentImage, prompt, branchName, extraEnv, sidecarCtx, callback)
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
		exitCode, err = a.waitForCallback(ctx, runCtx, callback)
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
	callback *port.CallbackSpec,
) (string, error) {
	env, err := a.buildAgentEnv(ctx, runCtx, project, story, agentImage, prompt, branchName, extraEnv, callbackToken(callback))
	if err != nil {
		return "", err
	}

	opts := model.ContainerOpts{
		Image:       agentImage,
		NetworkName: a.config.NetworkName,
		Memory:      a.config.DefaultMemory,
		CPUs:        a.config.DefaultCPUs,
		Env:         env,
		Labels:      a.buildAgentLabels(runCtx, story),
	}

	switch {
	case a.config.IsolateRuns && sidecarCtx != nil && sidecarCtx.NetworkName != "":
		// East-West isolation: the per-run network is the agent's primary and only
		// network; the shared network is dropped so agents from different runs never
		// share an L2 segment. The API is attached to the per-run network by the
		// SidecarManager, keeping the callback reachable.
		opts.NetworkName = sidecarCtx.NetworkName
	case sidecarCtx != nil && sidecarCtx.NetworkName != "":
		// Dual-home (default): keep the shared NetworkName (API callback / egress)
		// AND attach to the run network so the agent can reach sidecars by their
		// service-name DNS alias. Only set when a run network actually exists, so a
		// project without an Environment keeps identical ContainerOpts.
		opts.ExtraNetworks = []string{sidecarCtx.NetworkName}
	}

	return a.containerMgr.Create(ctx, opts)
}

// executeViaRuntime realises the agent run THROUGH port.AgentRuntime instead of
// driving the Docker ContainerManager directly. Docker is no longer special: the
// injected runtime (docker.Runtime by default, microsandbox under SUBSTRATE) owns
// only Launch/Stop/Wait, while THIS Action keeps the substrate-agnostic outcome
// model — callback-wait in callback mode, streamAndWait in legacy mode — plus the
// token mint (inside buildAgentEnv) and persistContainerID timing.
//
// It deliberately does NOT read result.ExitCode off the runtime's Wait as the
// outcome source: that was the rejected P3c branch-beside-Docker fork, which let
// substrate runs skip callback-wait and token-revoke. Here the outcome flows from
// exactly the same place as the legacy path.
func (a *AgentRunAction) executeViaRuntime(
	ctx context.Context,
	runCtx *model.RunContext,
	project *model.Project,
	story *model.Story,
	agentImage, prompt, branchName string,
	extraEnv []string,
	sidecarCtx *port.SidecarContext,
	callback *port.CallbackSpec,
) error {
	// The callback token is minted once upstream (prepareCallback) and threaded in
	// via callback; buildAgentEnv only emits AUTH_TOKEN from it. The token lifecycle
	// is unchanged from — and now identical to — the legacy path.
	env, err := a.buildAgentEnv(ctx, runCtx, project, story, agentImage, prompt, branchName, extraEnv, callbackToken(callback))
	if err != nil {
		return fmt.Errorf("build agent env: %w", err)
	}

	runtimeKind, _ := runCtx.Metadata["runtime_kind"].(string)
	modelID, _ := runCtx.Metadata["model"].(string)

	spec := port.RunSpec{
		RuntimeKind:  runtimeKind,
		Model:        modelID,
		Provider:     resolveProvider(runCtx),
		Image:        agentImage,
		Prompt:       prompt,
		Env:          env,
		Labels:       a.buildAgentLabels(runCtx, story),
		Capabilities: model.CapabilitySpec{}, // materialised by the harness at startup (fetch-at-startup)
		Memory:       a.config.DefaultMemory,
		CPUs:         a.config.DefaultCPUs,
		Network:      buildRunNetwork(sidecarCtx),
		Callback:     callback,
	}

	handle, err := a.runtime.Launch(ctx, spec)
	if err != nil {
		return fmt.Errorf("launch agent on substrate: %w", err)
	}
	// Teardown guaranteed, mirroring the legacy defer cleanupContainer. A detached
	// context lets Stop run even when ctx is already cancelled.
	defer func() {
		if stopErr := a.runtime.Stop(context.Background(), handle); stopErr != nil {
			a.logger.Warn("failed to stop agent substrate handle",
				"run_id", runCtx.Run.ID, "handle", handle.ID, "error", stopErr)
		}
	}()

	// Persist the substrate handle as the run-step container id (same timing as the
	// legacy path persists it after Start), so the out-of-band reapers keep working.
	a.persistContainerID(ctx, runCtx.RunStep.ID, handle.ID)

	// OUTCOME — IDENTICAL to the legacy path. We do NOT read result.ExitCode off the
	// runtime's Wait as the outcome (that is the rejected P3c fork). In callback mode
	// the callback channel is the source of truth (awaitCallbackOrCrash); runtime.Wait
	// is consulted ONLY to detect a substrate process that exits without ever reporting
	// a callback status (crash detection, ADR §2d) — never as the outcome. In legacy
	// mode we streamAndWait on the handle id (= container id for Docker).
	isCallbackMode := a.isCallbackMode(runtimeKind, agentImage)
	var exitCode int
	if isCallbackMode {
		exitCode, err = a.awaitCallbackOrCrash(ctx, runCtx, handle, callback)
	} else {
		exitCode, err = a.streamAndWait(ctx, handle.ID, runCtx)
	}
	if err != nil {
		return fmt.Errorf("stream/wait: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("agent exited with code %d", exitCode)
	}
	return nil
}

// buildRunNetwork maps the per-run sidecar network onto the agnostic
// port.RunNetwork. nil/empty sidecarCtx => zero RunNetwork (no extra attachment),
// so a project without an Environment yields a launch byte-identical to the
// single-homed legacy case.
func buildRunNetwork(sidecarCtx *port.SidecarContext) port.RunNetwork {
	if sidecarCtx != nil && sidecarCtx.NetworkName != "" {
		return port.RunNetwork{Name: sidecarCtx.NetworkName}
	}
	return port.RunNetwork{}
}

// buildAgentLabels returns the bookkeeping labels stamped onto the agent
// execution (managed_by/run_id/step_id/story_key). Extracted so the label set
// lives in one place independent of the execution substrate.
func (a *AgentRunAction) buildAgentLabels(runCtx *model.RunContext, story *model.Story) map[string]string {
	return map[string]string{
		model.LabelManagedBy: model.LabelManagedByValue,
		model.LabelRunID:     runCtx.Run.ID.String(),
		"step_id":            runCtx.RunStep.ID.String(),
		"story_key":          story.Key,
	}
}

// buildAgentEnv assembles the full KEY=value environment for one agent run: the
// repo/branch/prompt base block, the resolved git token, and either the
// callback-mode auth block (API_KEY, the AUTH_TOKEN minted upstream by
// prepareCallback, CALLBACK_URL/PROVIDER/RUN_ID/STEP_ID, MODEL) or the legacy
// OAuth block, followed by any sidecar connection strings in extraEnv. The slice
// order is preserved exactly, so the resulting env is byte-for-byte identical to
// the previous inline assembly.
//
// authToken is the per-run callback token minted once in Execute (prepareCallback)
// and threaded in; it is "" for legacy/non-callback runs. buildAgentEnv no longer
// mints the token itself — the mint is hoisted so it happens once per run for all
// substrates and the matching revoke can run with the same token.
func (a *AgentRunAction) buildAgentEnv(
	ctx context.Context,
	runCtx *model.RunContext,
	project *model.Project,
	story *model.Story,
	agentImage, prompt, branchName string,
	extraEnv []string,
	authToken string,
) ([]string, error) {
	repoURL := ""
	if project.RepoURL != nil {
		repoURL = *project.RepoURL
	}

	// Resolve git token dynamically from project config
	gitToken := os.Getenv(gitTokenEnvName(project))

	// Resolve role from step config and build per-run CLAUDE.md context.
	role := resolveRole(runCtx)

	env := []string{
		envPrefixRepoURL + repoURL,
		envPrefixBranchName + branchName,
		"STORY_KEY=" + story.Key,
		// PROMPT_CONTENT is consumed by the legacy shell entrypoint (agent/entrypoint.sh).
		// PROMPT is consumed by the agent-runtime Go binary in callback mode (config.Load).
		// Inject both so either image variant receives the rendered prompt under the name it reads.
		"PROMPT_CONTENT=" + prompt,
		"PROMPT=" + prompt,
		envPrefixGitToken + gitToken,
		envPrefixGitProvider + project.GitProvider,
		envPrefixGitHubToken + gitToken,
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

		// AUTH_TOKEN for callback auth. The token is minted ONCE upstream in Execute
		// (prepareCallback) and threaded in, so the same token can be revoked after
		// the callback resolves. Empty when no tokenStore is configured (back-compat:
		// behaves exactly as before).
		if authToken != "" {
			env = append(env, "AUTH_TOKEN="+authToken)
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

	return env, nil
}

// prepareCallback mints the per-run callback token ONCE and returns the typed
// CallbackSpec carried on the RunSpec, so the lifecycle — mint at launch, revoke
// after the callback resolves — runs identically on every substrate.
//
// It returns (nil, nil) for non-callback runs: when callback mode is off
// (statusStore nil or a non-callback runtime kind) there is no token to mint and
// the legacy OAuth env is used. Gated on tokenStore != nil so a callback-capable
// run with no token store still launches (back-compat: AUTH_TOKEN simply absent).
func (a *AgentRunAction) prepareCallback(ctx context.Context, runCtx *model.RunContext, role string) (*port.CallbackSpec, error) {
	runtimeKind, _ := runCtx.Metadata["runtime_kind"].(string)
	agentImage, _ := runCtx.Metadata["agent_image"].(string)
	if !a.isCallbackMode(runtimeKind, agentImage) {
		return nil, nil
	}

	var token string
	if a.tokenStore != nil {
		// Bind the token to the agent so the fetch-at-startup bundle endpoint can
		// resolve this agent's capabilities server-side. agentID is uuid.Nil when no
		// agent is bound, which yields an empty bundle (back-compat).
		agentID := uuid.Nil
		if id := extractAgentID(runCtx); id != nil {
			agentID = *id
		}
		t, err := a.tokenStore.Create(ctx, runCtx.Run.ID, runCtx.RunStep.ID, agentID, role, 2*time.Hour)
		if err != nil {
			return nil, fmt.Errorf("create container token: %w", err)
		}
		token = t
	}

	return &port.CallbackSpec{
		URL:       a.callbackURL,
		AuthToken: token,
		RunID:     runCtx.Run.ID,
		StepID:    runCtx.RunStep.ID,
	}, nil
}

// callbackToken returns the minted auth token from a CallbackSpec, or "" when
// there is no callback (legacy/non-callback run).
func callbackToken(callback *port.CallbackSpec) string {
	if callback == nil {
		return ""
	}
	return callback.AuthToken
}

// auditPrompt makes the rendered prompt auditable through the substrate-agnostic
// channel, independent of which substrate runs the agent or the (Docker-only)
// log stream. It does two things:
//   - Durable audit: publishes the prompt on the Postgres event bus as a LogEvent
//     of Type "prompt" (→ events table + SSE). This survives the retirement of the
//     Docker log-stream mechanism (Stage 5).
//   - Operational signal: a structured Info log carrying only run/step/agent/role
//     and the prompt length (the prompt body stays out of the bounded log_tail).
//
// A prompt is a rendered task template, not a secret. The ScrubHandler still
// redacts any token-shaped field on the slog signal; the durable event-bus audit
// is intentionally non-redacted (see in-body comment). An empty prompt is skipped
// (no empty event).
func (a *AgentRunAction) auditPrompt(ctx context.Context, runCtx *model.RunContext, prompt, role string) {
	if prompt == "" {
		return
	}

	agentID := ""
	if id := extractAgentID(runCtx); id != nil {
		agentID = id.String()
	}

	a.logger.Info("agent prompt dispatched",
		"run_id", runCtx.Run.ID,
		"step_id", runCtx.RunStep.ID,
		"agent_id", agentID,
		"role", role,
		"prompt_len", len(prompt),
	)

	// Durable audit on the agnostic event bus — same mechanism as publishLogEvent,
	// with Type "prompt" so consumers can distinguish it from container stdout.
	//
	// The prompt is published NON-redacted to the event bus (Postgres events table
	// + SSE) ON PURPOSE: this is the auditable record of what the agent was asked to
	// do (Decision #2, ADR §7#2). The ScrubHandler only wraps slog, not the event
	// bus, so it does not touch this — which is intended; the prompt is a rendered
	// task template, not a secret.
	//
	// The role is carried on the operational slog above, not duplicated into the
	// LogEvent (no model field added — avoids scope creep). An event-bus consumer
	// recovers the role by joining on run_id/step_id.
	a.publishLogEvent(ctx, runCtx, model.LogEvent{
		RunID:   runCtx.Run.ID.String(),
		StepID:  runCtx.RunStep.ID.String(),
		Message: prompt,
		Type:    eventTypePrompt,
	})
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
//
// It REVOKES the callback token — the real one minted in Execute (prepareCallback)
// and carried on callback — on EVERY exit path (success, WaitForStatus error,
// timeout, cancel, panic) via defer, so the token dies with the run instead of
// lingering until its 2h TTL. (Before Stage 3a the revoke was dead: it looked the
// token up via a stub that always returned none.)
//
// This is the LEGACY callback-wait path (Execute with runtime==nil): there is no
// substrate handle here, so it cannot watch the process for crash detection. The
// substrate path uses awaitCallbackOrCrash instead.
func (a *AgentRunAction) waitForCallback(ctx context.Context, runCtx *model.RunContext, callback *port.CallbackSpec) (int, error) {
	stepID := runCtx.RunStep.ID

	// Revoke via defer, registered BEFORE WaitForStatus, so it fires on every exit
	// path — including a WaitForStatus error, timeout, or context cancellation, none
	// of which would reach a post-wait revoke.
	defer a.revokeCallbackToken(stepID, callback)

	exitCode, errMsg, err := a.statusStore.WaitForStatus(ctx, stepID, 2*time.Hour)
	return a.finishCallbackStatus(stepID, exitCode, errMsg, err)
}

// revokeCallbackToken revokes the minted token best-effort on a bounded detached
// context. No-op when there is no token. Shared by every callback-wait path so the
// token dies with the run regardless of how the wait ends.
func (a *AgentRunAction) revokeCallbackToken(stepID uuid.UUID, callback *port.CallbackSpec) {
	if a.tokenStore == nil || callback == nil || callback.AuthToken == "" {
		return
	}
	revokeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.tokenStore.Revoke(revokeCtx, callback.AuthToken); err != nil {
		a.logger.Warn("failed to revoke container token", "step_id", stepID, "error", err)
	}
}

// awaitCallbackOrCrash waits for the agent's callback status (the AUTHORITATIVE
// outcome) while concurrently watching the substrate process via runtime.Wait for
// CRASH DETECTION. The callback is the source of truth; runtime.Wait only tells us
// the exec finished. If the process exits NON-ZERO WITHOUT a callback status
// arriving within a short grace, the run is declared a crash instead of blocking on
// the 2h status timeout (ADR §2d). A CLEAN exit (code 0) without a status is treated
// as an in-flight callback and keeps waiting on the authoritative status (bounded by
// the 2h WaitForStatus), never a false crash. The reverse — status arrives, Wait
// never returns — also stays bounded by that 2h timeout.
//
// Concurrency: both goroutines send on buffered (cap 1) channels and both their
// contexts are cancelled by defer, so neither leaks regardless of which path wins
// (the buffered send never blocks even after this method returns; the deferred
// runtime.Stop in executeViaRuntime guarantees runtime.Wait unblocks).
func (a *AgentRunAction) awaitCallbackOrCrash(ctx context.Context, runCtx *model.RunContext, handle port.RunHandle, callback *port.CallbackSpec) (int, error) {
	stepID := runCtx.RunStep.ID
	defer a.revokeCallbackToken(stepID, callback)

	type statusResult struct {
		exitCode int
		errMsg   string
		err      error
	}
	statusCh := make(chan statusResult, 1)
	statusCtx, cancelStatus := context.WithCancel(ctx)
	defer cancelStatus()
	go func() {
		ec, em, err := a.statusStore.WaitForStatus(statusCtx, stepID, 2*time.Hour)
		statusCh <- statusResult{exitCode: ec, errMsg: em, err: err}
	}()

	waitCh := make(chan port.RunResult, 1)
	waitCtx, cancelWait := context.WithCancel(context.Background())
	defer cancelWait()
	go func() {
		// This runtime.Wait(waitCtx, handle) may run CONCURRENTLY with the deferred
		// runtime.Stop(handle) in executeViaRuntime, on the SAME handle. That is safe:
		// the Docker client tolerates concurrent Wait+Stop on one container, and it is
		// the expected contract for any substrate adapter (Stop is precisely what
		// unblocks a pending Wait). So even if this goroutine is still parked in Wait
		// when the method returns, the deferred Stop releases it and the buffered send
		// below never blocks — no leak.
		res, werr := a.runtime.Wait(waitCtx, handle)
		if werr != nil {
			a.logger.Debug("runtime wait ended with error (crash detector)", "handle", handle.ID, "error", werr)
		}
		waitCh <- res
	}()

	select {
	case s := <-statusCh:
		return a.finishCallbackStatus(stepID, s.exitCode, s.errMsg, s.err)
	case w := <-waitCh:
		// The substrate process exited. The AUTHORITATIVE outcome is still the
		// callback status; runtime.Wait only signals the exec finished.
		//
		// ADR §2d: a crash is declared ONLY for a NON-ZERO exit without a status. A
		// clean exit (0) without a status is treated as a delayed/in-flight callback
		// — keep waiting on the authoritative status (bounded by the goroutine's own
		// 2h WaitForStatus), never declaring a false crash on a clean exit.
		if w.ExitCode == 0 {
			s := <-statusCh
			return a.finishCallbackStatus(stepID, s.exitCode, s.errMsg, s.err)
		}
		// Non-zero exit: give the (possibly in-flight) callback a brief grace, then
		// declare a crash. A context deadline (not time.After) bounds the grace and
		// is cleaned up by defer — no leaked timer.
		graceCtx, cancelGrace := context.WithTimeout(context.Background(), a.crashGrace())
		defer cancelGrace()
		select {
		case s := <-statusCh:
			return a.finishCallbackStatus(stepID, s.exitCode, s.errMsg, s.err)
		case <-graceCtx.Done():
			// Grace elapsed. Final NON-BLOCKING check: the status may have landed in
			// the same scheduling tick the deadline fired (select is random when both
			// are ready). Prefer the authoritative status if it is already buffered.
			select {
			case s := <-statusCh:
				return a.finishCallbackStatus(stepID, s.exitCode, s.errMsg, s.err)
			default:
				return -1, fmt.Errorf("agent process exited (code %d) on the substrate without reporting a callback status", w.ExitCode)
			}
		}
	}
}

// finishCallbackStatus turns a WaitForStatus result into the outcome shared by
// the legacy and substrate callback-wait paths: propagate any wait error, warn on
// a non-empty container error message, and return the reported exit code.
func (a *AgentRunAction) finishCallbackStatus(stepID uuid.UUID, exitCode int, errMsg string, err error) (int, error) {
	if err != nil {
		return -1, err
	}
	if errMsg != "" {
		a.logger.Warn("agent container reported error", "step_id", stepID, "exit_code", exitCode, "error", errMsg)
	}
	return exitCode, nil
}

// crashGrace returns the configured grace period the substrate callback-wait gives
// an in-flight callback after the process exits, or the 5s default when unset.
func (a *AgentRunAction) crashGrace() time.Duration {
	if a.config.CrashGrace > 0 {
		return a.config.CrashGrace
	}
	return 5 * time.Second
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
