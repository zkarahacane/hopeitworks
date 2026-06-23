package action

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
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

	labelManagedBy    = "managed_by"
	labelValueManaged = "hopeitworks"
	labelRunID        = "run_id"

	defaultGitTokenEnv = "GITHUB_TOKEN"
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

	for _, key := range commandOrder {
		command, ok := env.Commands[key]
		if !ok || strings.TrimSpace(command) == "" {
			continue // optional command: skip when absent or blank
		}
		if err := a.runOneCommand(ctx, key, command, image, cloneURL, branchName, gitEnv, extraEnv, sidecarCtx); err != nil {
			return err
		}
	}
	return nil
}

// runOneCommand launches a single ephemeral container that clones the repo and
// runs command, waits for it, and removes the container. A non-zero exit code is
// returned as an error (fail-fast) with the last log lines for diagnosis.
func (a *AgentRunAction) runOneCommand(
	ctx context.Context,
	key, command, image, cloneURL, branchName string,
	gitEnv, extraEnv []string,
	sidecarCtx *port.SidecarContext,
) error {
	cctx, cancel := context.WithTimeout(ctx, commandRunTimeout)
	defer cancel()

	// Clone the repo (replicating the agent-runtime clone) then run the command
	// from the workdir. The authenticated URL is passed via an env var so it
	// never appears in the container's argv / process list.
	script := "git clone --depth 1 --branch \"$BRANCH_NAME\" \"$HOPEITWORKS_CLONE_URL\" " + ephemeralWorkdir +
		" || git clone --depth 1 \"$HOPEITWORKS_CLONE_URL\" " + ephemeralWorkdir +
		"; cd " + ephemeralWorkdir + " && " + command

	cmdEnv := make([]string, 0, len(gitEnv)+len(extraEnv)+2)
	cmdEnv = append(cmdEnv, gitEnv...)
	cmdEnv = append(cmdEnv, extraEnv...)
	cmdEnv = append(cmdEnv, "BRANCH_NAME="+branchName)
	cmdEnv = append(cmdEnv, "HOPEITWORKS_CLONE_URL="+cloneURL)

	opts := model.ContainerOpts{
		Image:  image,
		Env:    cmdEnv,
		Memory: a.config.DefaultMemory,
		CPUs:   a.config.DefaultCPUs,
		Cmd:    append(append([]string(nil), cmdShell...), script),
		Labels: a.commandLabels(sidecarCtx, key),
	}
	// Attach the ephemeral container to the run network so it reaches the
	// sidecars by their DNS alias, exactly like the agent container does.
	if sidecarCtx != nil && sidecarCtx.NetworkName != "" {
		opts.NetworkName = sidecarCtx.NetworkName
	}

	a.logger.Info("running environment command",
		"command_key", key, "image", image, "network", opts.NetworkName)

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
		return fmt.Errorf("environment command %q failed with exit code %d%s",
			key, exitCode, a.commandLogTail(containerID))
	}

	a.logger.Info("environment command succeeded", "command_key", key)
	return nil
}

// commandLabels builds the labels for an ephemeral command container so it is
// attributable to the run and never mistaken for a sidecar or the agent.
func (a *AgentRunAction) commandLabels(sidecarCtx *port.SidecarContext, key string) map[string]string {
	labels := map[string]string{
		labelManagedBy: labelValueManaged,
		"role":         "env_command",
		"command_key":  key,
	}
	if sidecarCtx != nil {
		labels[labelRunID] = sidecarCtx.RunID.String()
	}
	return labels
}

// commandLogTail returns a best-effort, scrubbed tail of the ephemeral
// container's logs for inclusion in a fail-fast error. It returns "" when logs
// cannot be retrieved so the error message stays clean.
func (a *AgentRunAction) commandLogTail(containerID string) string {
	logCh, doneCh, err := a.logStreamer.StreamLogs(
		context.Background(), containerID, "", "")
	if err != nil {
		return ""
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
		return ""
	}
	return ": " + strings.Join(tail, " | ")
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
// agent container uses) when no stack is declared or the stack repo is absent.
// The fallback guarantees the command image always carries a usable toolchain.
func (a *AgentRunAction) resolveCommandImage(ctx context.Context, env *model.Environment, agentImage string) (string, error) {
	if len(env.Stacks) > 0 && a.stackRepo != nil {
		key := env.Stacks[0]
		stack, err := a.stackRepo.GetByKey(ctx, key)
		if err != nil {
			return "", fmt.Errorf("lookup stack %q: %w", key, err)
		}
		if stack != nil && stack.ImageRef != "" {
			return stack.ImageRef, nil
		}
	}
	if agentImage != "" {
		return agentImage, nil
	}
	return "", fmt.Errorf("no stack declared and no fallback agent image available")
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
