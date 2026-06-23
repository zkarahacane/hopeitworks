package action

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Environment command keys, run in this FIXED conventional order (never the
// non-deterministic map iteration). Each key is optional: a command absent from
// env.Commands is skipped.
const (
	cmdKeyBuild   = "build"
	cmdKeyMigrate = "migrate"
	cmdKeySeed    = "seed"
	cmdKeyTest    = "test"
)

// commandOrder is the canonical execution order for Environment setup commands.
var commandOrder = []string{cmdKeyBuild, cmdKeyMigrate, cmdKeySeed, cmdKeyTest}

// ephemeralWorkdir is where the repo is cloned inside the ephemeral command
// container before each Environment command runs.
const ephemeralWorkdir = "/workspace/repo"

// gitProviderGitea is the provider key whose token is injected as "oauth2:<token>".
const gitProviderGitea = "gitea"

// Env var KEY= prefixes and label values shared between the agent container and
// the ephemeral command container. Declared as constants because goconst (which
// does NOT skip _test.go) flags any literal repeated three or more times.
const (
	envPrefixRepoURL     = "REPO_URL="
	envPrefixBranchName  = "BRANCH_NAME="
	envPrefixGitToken    = "GIT_TOKEN="
	envPrefixGitProvider = "GIT_PROVIDER="
	envPrefixGitHubToken = "GITHUB_TOKEN="

	defaultGitTokenEnv = "GITHUB_TOKEN"

	// cloneURLEnv is the env var carrying the authenticated clone URL into the
	// ephemeral command container, keeping the token out of the command argv.
	cloneURLEnv = "HOPEITWORKS_CLONE_URL"

	// eventTypePrompt tags the durable prompt-audit LogEvent published on the
	// agnostic event bus, so consumers distinguish it from container stdout.
	eventTypePrompt = "prompt"
)

// cmdShell is the shell used to drive the clone-then-run one-liner.
var cmdShell = []string{"sh", "-lc"}

// commandRunTimeout bounds a single Environment command (clone + command). It is
// generous: builds and migrations can be slow, but the run must never hang
// indefinitely on a stuck command.
const commandRunTimeout = 30 * time.Minute

// runEnvironmentCommands executes the project's Environment setup commands
// (build → migrate → seed → test) AFTER the sidecars are ready and BEFORE the
// agent container starts. Each present command runs in its own EPHEMERAL
// container (Option A: no agent-runtime change) attached to the run network,
// with the sidecar connection strings and the same git env the agent receives.
//
// The repo is cloned inside the ephemeral container exactly the way the
// agent-runtime clones it (authenticated URL, same branch, shallow), so the
// command sees the project's Makefile/migrations. The clone is throwaway; only
// the side effect on the sidecar (migrated/seeded DB) persists.
//
// Fail-fast: the first command with a non-zero exit code returns an error that
// fails the run; no further command runs and the agent container is never
// created. Every ephemeral container is removed best-effort regardless of
// outcome.
//
// Nil-safety / back-compat: env == nil or len(env.Commands) == 0 is a no-op —
// no ephemeral container is created and behaviour is strictly unchanged.
func (a *AgentRunAction) runEnvironmentCommands(
	ctx context.Context,
	env *model.Environment,
	sidecarCtx *port.SidecarContext,
	project *model.Project,
	runID uuid.UUID,
	branchName, agentImage string,
	extraEnv []string,
) error {
	if env == nil || len(env.Commands) == 0 {
		return nil
	}

	image, err := a.resolveCommandImage(ctx, env, agentImage)
	if err != nil {
		return fmt.Errorf("resolve command image: %w", err)
	}

	gitEnv := a.gitEnv(project)
	cloneURL, err := authenticatedRepoURL(project)
	if err != nil {
		return fmt.Errorf("build authenticated repo url: %w", err)
	}

	spec := commandSpec{
		image:      image,
		cloneURL:   cloneURL,
		branchName: branchName,
		gitEnv:     gitEnv,
		extraEnv:   extraEnv,
		runID:      runID,
		sidecarCtx: sidecarCtx,
	}
	for _, key := range commandOrder {
		command, ok := env.Commands[key]
		if !ok || strings.TrimSpace(command) == "" {
			continue // optional command: skip when absent or blank
		}
		if err := a.runOneCommand(ctx, key, command, spec); err != nil {
			return err
		}
	}
	return nil
}

// commandSpec carries the invariant inputs shared by every ephemeral command
// container in a run, so runOneCommand keeps a small signature.
type commandSpec struct {
	image      string
	cloneURL   string
	branchName string
	gitEnv     []string
	extraEnv   []string
	runID      uuid.UUID
	sidecarCtx *port.SidecarContext
}

// runOneCommand launches a single ephemeral container that clones the repo and
// runs command, waits for it, and removes the container. A non-zero exit code is
// returned as a fail-fast error.
//
// SECURITY: the returned error carries ONLY the command key and exit code —
// never the container's stdout/stderr. That error propagates through Execute to
// the pipeline executor, which persists it to run.error_message AND broadcasts
// it over SSE; a command that dumps its env (printenv / set -x / CI tooling)
// would otherwise leak the git token and DB passwords into a persisted, public
// channel. The log tail is emitted server-side via slog only.
func (a *AgentRunAction) runOneCommand(ctx context.Context, key, command string, spec commandSpec) error {
	cctx, cancel := context.WithTimeout(ctx, commandRunTimeout)
	defer cancel()

	// Clone the repo (replicating the agent-runtime clone) then run the command
	// from the workdir. STRICT semantics: `set -e` makes the clone failure abort
	// the script, so a clone error (network/auth/missing branch) NEVER silently
	// falls back to the default branch — migrate/seed must run against the exact
	// requested branch or not at all. The authenticated URL is passed via an env
	// var so it never appears in the container's argv / process list.
	script := "set -e; git clone --depth 1 --branch \"$BRANCH_NAME\" \"$" + cloneURLEnv + "\" " + ephemeralWorkdir +
		"; cd " + ephemeralWorkdir + "; " + command

	cmdEnv := make([]string, 0, len(spec.gitEnv)+len(spec.extraEnv)+2)
	cmdEnv = append(cmdEnv, spec.gitEnv...)
	cmdEnv = append(cmdEnv, spec.extraEnv...)
	cmdEnv = append(cmdEnv, envPrefixBranchName+spec.branchName)
	cmdEnv = append(cmdEnv, cloneURLEnv+"="+spec.cloneURL)

	opts := model.ContainerOpts{
		Image:  spec.image,
		Env:    cmdEnv,
		Memory: a.config.DefaultMemory,
		CPUs:   a.config.DefaultCPUs,
		// Override BOTH entrypoint and command: stack/agent images bake
		// ENTRYPOINT ["agent-runtime"], so setting only Cmd would run
		// `agent-runtime sh -lc <script>` and never execute the shell. Setting
		// Entrypoint to the shell replaces the image entrypoint entirely.
		Entrypoint: append([]string(nil), cmdShell...),
		Cmd:        []string{script},
		Labels:     a.commandLabels(spec.runID, key),
	}
	// Attach the ephemeral container to the run network so it reaches the
	// sidecars by their DNS alias, exactly like the agent container does.
	if spec.sidecarCtx != nil && spec.sidecarCtx.NetworkName != "" {
		opts.NetworkName = spec.sidecarCtx.NetworkName
	}

	a.logger.Info("running environment command",
		"command_key", key, "image", spec.image, "network", opts.NetworkName)

	containerID, err := a.containerMgr.Create(cctx, opts)
	if err != nil {
		return fmt.Errorf("create %s command container: %w", key, err)
	}
	defer a.removeEphemeral(containerID)

	if err := a.containerMgr.Start(cctx, containerID); err != nil {
		return fmt.Errorf("start %s command container: %w", key, err)
	}

	exitCode, err := a.containerMgr.Wait(cctx, containerID)
	if err != nil {
		return fmt.Errorf("wait %s command container: %w", key, err)
	}
	if exitCode != 0 {
		// Server-side diagnostic ONLY: the tail goes to slog (scrubbed by the
		// ScrubHandler), never into the returned/propagated error.
		a.logCommandTail(containerID, key, exitCode)
		return fmt.Errorf("environment command %q failed with exit code %d", key, exitCode)
	}

	a.logger.Info("environment command succeeded", "command_key", key)
	return nil
}

// commandLabels builds the labels for an ephemeral command container so it is
// attributable to the run and reapable by GC. run_id is ALWAYS stamped — a
// project with commands but no sidecar services still reaches this path, and an
// unlabelled container would be invisible to the GC reaper.
func (a *AgentRunAction) commandLabels(runID uuid.UUID, key string) map[string]string {
	return map[string]string{
		model.LabelManagedBy:  model.LabelManagedByValue,
		model.LabelRole:       model.RoleEnvCommand,
		model.LabelCommandKey: key,
		model.LabelRunID:      runID.String(),
	}
}

// logCommandTail emits a best-effort tail of the ephemeral container's logs to
// slog for SERVER-SIDE diagnosis only. It is never returned to the caller, so it
// cannot reach run.error_message / SSE. slog's ScrubHandler additionally redacts
// token/secret/password-shaped fields. Bounded by its own short timeout so a
// stuck log stream never blocks the fail-fast path.
func (a *AgentRunAction) logCommandTail(containerID, key string, exitCode int) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logCh, doneCh, err := a.logStreamer.StreamLogs(ctx, containerID, "", "")
	if err != nil {
		return
	}
	const maxLines = 20
	tail := make([]string, 0, maxLines)
	for ev := range logCh {
		if len(tail) >= maxLines {
			tail = tail[1:]
		}
		tail = append(tail, ev.Message)
	}
	// Drain doneCh without blocking: the LogStreamer may or may not send an exit
	// code once logCh is closed, and this is a best-effort diagnostic path.
	select {
	case <-doneCh:
	default:
	}
	if len(tail) == 0 {
		return
	}
	a.logger.Warn("environment command failed",
		"command_key", key,
		"exit_code", exitCode,
		"log_tail", strings.Join(tail, "\n"))
}

// removeEphemeral stops and removes an ephemeral command container on a bounded,
// detached context so cleanup runs even when the run's context is cancelled.
func (a *AgentRunAction) removeEphemeral(containerID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := a.containerMgr.Stop(ctx, containerID); err != nil {
		a.logger.Warn("failed to stop ephemeral command container",
			"container_id", containerID, "error", err)
	}
	if err := a.containerMgr.Remove(ctx, containerID); err != nil {
		a.logger.Warn("failed to remove ephemeral command container",
			"container_id", containerID, "error", err)
	}
}

// resolveCommandImage resolves the image that runs the Environment commands.
// Preference: the Environment's first Stack (env.Stacks[0]) via the stack
// catalogue. Fallback: the agent's effective image (the same image the run's
// agent container uses) when no stack is declared OR the declared stack key is
// not catalogued (typo / not seeded). The fallback guarantees the command image
// always carries a usable toolchain.
//
// A NotFound from the stack repo is NOT fatal — it degrades to the agent image
// fallback (with a Warn). Any other repo error (transient DB failure, etc.) is
// propagated and fails the run, so a flaky lookup is not silently masked.
func (a *AgentRunAction) resolveCommandImage(ctx context.Context, env *model.Environment, agentImage string) (string, error) {
	if len(env.Stacks) > 0 && a.stackRepo != nil {
		key := env.Stacks[0]
		stack, err := a.stackRepo.GetByKey(ctx, key)
		switch {
		case err == nil && stack != nil && stack.ImageRef != "":
			return stack.ImageRef, nil
		case err != nil && isNotFound(err):
			a.logger.Warn("environment stack not catalogued, falling back to agent image",
				"stack_key", key)
		case err != nil:
			return "", fmt.Errorf("lookup stack %q: %w", key, err)
		}
	}
	if agentImage != "" {
		return agentImage, nil
	}
	return "", fmt.Errorf("no stack declared and no fallback agent image available")
}

// isNotFound reports whether err is (or wraps) a NotFound DomainError.
func isNotFound(err error) bool {
	var de *apperrors.DomainError
	return stderrors.As(err, &de) && de.Category == apperrors.CategoryNotFound
}

// gitEnv returns the git-related env entries the agent container also receives,
// so the ephemeral command container clones identically. It mirrors the env set
// built in createContainer (REPO_URL, GIT_TOKEN, GIT_PROVIDER, GITHUB_TOKEN),
// resolving the token from the project-configured token env var.
func (a *AgentRunAction) gitEnv(project *model.Project) []string {
	repoURL := ""
	if project.RepoURL != nil {
		repoURL = *project.RepoURL
	}
	gitToken := os.Getenv(gitTokenEnvName(project))
	return []string{
		envPrefixRepoURL + repoURL,
		envPrefixGitToken + gitToken,
		envPrefixGitProvider + project.GitProvider,
		envPrefixGitHubToken + gitToken,
	}
}

// gitTokenEnvName resolves the OS env var name holding the git token for the
// project, defaulting to GITHUB_TOKEN when the project does not override it.
func gitTokenEnvName(project *model.Project) string {
	if project.GitTokenEnv != nil && *project.GitTokenEnv != "" {
		return *project.GitTokenEnv
	}
	return defaultGitTokenEnv
}

// authenticatedRepoURL builds the authenticated clone URL exactly the way the
// agent-runtime does (agent-runtime/internal/git/clone.go#injectToken):
//   - gitea:  https://oauth2:<token>@host/...
//   - github: https://<token>@host/...  (also the default)
//
// The token is read from the project-configured env var. Building it here (not
// in the in-container shell) keeps the token out of the command argv; it is
// passed to the container via an env var instead.
func authenticatedRepoURL(project *model.Project) (string, error) {
	if project.RepoURL == nil || *project.RepoURL == "" {
		return "", fmt.Errorf("project has no repo URL")
	}
	token := os.Getenv(gitTokenEnvName(project))
	parsed, err := url.Parse(*project.RepoURL)
	if err != nil {
		return "", fmt.Errorf("parse repo URL: %w", err)
	}
	if project.GitProvider == gitProviderGitea {
		parsed.User = url.UserPassword("oauth2", token)
	} else {
		parsed.User = url.User(token)
	}
	return parsed.String(), nil
}
