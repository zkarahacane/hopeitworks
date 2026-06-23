package action_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// Labels and env keys asserted repeatedly across these tests. Declared as
// constants because goconst (which does NOT skip _test.go in this config) flags
// any string literal repeated three or more times.
const (
	labelRole       = "role"
	labelCommandKey = "command_key"
	roleEnvCommand  = "env_command"
	keyGITTOKEN     = "GIT_TOKEN="

	stackImageRef = "ghcr.io/hopeitworks/go@sha256:deadbeef"
	agentImageRef = "hopeitworks/agent:latest"
)

// envWithCommands returns an Environment that has a postgres service (so sidecars
// launch) and the given commands map.
func envWithCommands(projectID uuid.UUID, commands map[string]string, stacks []string) *model.Environment {
	return &model.Environment{
		ProjectID: projectID,
		Stacks:    stacks,
		Services: []model.EnvironmentService{
			{
				Name:  "db",
				Image: "postgres:16",
				Env: map[string]string{
					"POSTGRES_USER":     "app",
					"POSTGRES_PASSWORD": "secret",
					"POSTGRES_DB":       "appdb",
				},
			},
		},
		Commands: commands,
	}
}

// wireEnvironment makes the fixture's repos serve env and a successful sidecar
// launch returning a run network. It returns the run network name.
func wireEnvironment(f *agentRunFixture, env *model.Environment) string {
	runNetwork := "hopeitworks-run-" + f.runID.String()
	f.environmentRepo.getByProjectIDFn = func(_ context.Context, _ uuid.UUID) (*model.Environment, error) {
		return env, nil
	}
	f.sidecarMgr.launchFn = func(_ context.Context, runID uuid.UUID, _ *model.Environment) (*port.SidecarContext, error) {
		return &port.SidecarContext{
			RunID:        runID,
			NetworkName:  runNetwork,
			ContainerIDs: map[string]string{"db": "sidecar-db"},
			ServiceAddrs: map[string]string{"db": "db"},
		}, nil
	}
	return runNetwork
}

// commandCreateCalls returns only the ephemeral command Create calls (role=env_command).
func commandCreateCalls(f *agentRunFixture) []model.ContainerOpts {
	f.containerMgr.mu.Lock()
	defer f.containerMgr.mu.Unlock()
	var out []model.ContainerOpts
	for _, c := range f.containerMgr.createCalls {
		if c.Labels[labelRole] == roleEnvCommand {
			out = append(out, c)
		}
	}
	return out
}

// agentCreateCalls returns only the non-command (agent) Create calls.
func agentCreateCalls(f *agentRunFixture) []model.ContainerOpts {
	f.containerMgr.mu.Lock()
	defer f.containerMgr.mu.Unlock()
	var out []model.ContainerOpts
	for _, c := range f.containerMgr.createCalls {
		if c.Labels[labelRole] != roleEnvCommand {
			out = append(out, c)
		}
	}
	return out
}

// idForCommand encodes the command key into a container id so waitFn can map an
// id back to its command and decide the exit code.
func idForCommand(opts model.ContainerOpts) string {
	if k := opts.Labels[labelCommandKey]; k != "" {
		return "cmd-" + k
	}
	return testContainerID
}

// TestRunEnvironmentCommands_Order proves the commands run in the FIXED
// conventional order build → migrate → seed → test and that a command absent
// from env.Commands is skipped.
func TestRunEnvironmentCommands_Order(t *testing.T) {
	f := newAgentRunFixture(t)
	// Provide commands out of map order, and omit "migrate" to prove it is skipped.
	env := envWithCommands(f.projectID, map[string]string{
		cmdTest():  "make test",
		cmdBuild(): "make build",
		cmdSeed():  "make seed",
	}, nil)
	wireEnvironment(f, env)

	f.containerMgr.createFn = func(_ context.Context, opts model.ContainerOpts) (string, error) {
		return idForCommand(opts), nil
	}

	if err := f.action.Execute(context.Background(), f.newRunContext()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cmds := commandCreateCalls(f)
	if len(cmds) != 3 {
		t.Fatalf("expected 3 command containers (build, seed, test), got %d", len(cmds))
	}
	gotOrder := []string{
		cmds[0].Labels[labelCommandKey],
		cmds[1].Labels[labelCommandKey],
		cmds[2].Labels[labelCommandKey],
	}
	wantOrder := []string{cmdBuild(), cmdSeed(), cmdTest()}
	for i := range wantOrder {
		if gotOrder[i] != wantOrder[i] {
			t.Errorf("command order[%d]: got %q, want %q (full=%v)", i, gotOrder[i], wantOrder[i], gotOrder)
		}
	}

	// The agent container is still created after the commands succeed.
	if len(agentCreateCalls(f)) != 1 {
		t.Errorf("expected 1 agent container created after commands, got %d", len(agentCreateCalls(f)))
	}
}

// TestRunEnvironmentCommands_FailFast proves that a command exiting non-zero
// fails the run, stops further commands, and never creates the agent container.
func TestRunEnvironmentCommands_FailFast(t *testing.T) {
	f := newAgentRunFixture(t)
	env := envWithCommands(f.projectID, map[string]string{
		cmdBuild():   "make build",
		cmdMigrate(): "make migrate",
		cmdSeed():    "make seed",
	}, nil)
	wireEnvironment(f, env)

	f.containerMgr.createFn = func(_ context.Context, opts model.ContainerOpts) (string, error) {
		return idForCommand(opts), nil
	}
	// migrate fails with exit code 2; build succeeds.
	f.containerMgr.waitFn = func(_ context.Context, containerID string) (int, error) {
		if containerID == "cmd-"+cmdMigrate() {
			return 2, nil
		}
		return 0, nil
	}

	err := f.action.Execute(context.Background(), f.newRunContext())
	if err == nil {
		t.Fatal("expected fail-fast error from migrate, got nil")
	}
	if !strings.Contains(err.Error(), cmdMigrate()) || !strings.Contains(err.Error(), "exit code 2") {
		t.Errorf("expected error to mention migrate and exit code 2, got: %v", err)
	}

	// build ran, migrate ran (and failed), seed must NOT have run.
	cmds := commandCreateCalls(f)
	keys := make([]string, len(cmds))
	for i, c := range cmds {
		keys[i] = c.Labels[labelCommandKey]
	}
	if len(keys) != 2 || keys[0] != cmdBuild() || keys[1] != cmdMigrate() {
		t.Errorf("expected [build migrate] before fail-fast, got %v", keys)
	}

	// The agent container must NOT have been created.
	if got := len(agentCreateCalls(f)); got != 0 {
		t.Errorf("expected 0 agent containers after a failed command, got %d", got)
	}
	// Sidecars are still torn down (deferred Cleanup).
	if got := f.sidecarMgr.getCleanupCalls(); got != 1 {
		t.Errorf("expected 1 sidecar Cleanup after fail-fast, got %d", got)
	}
}

// TestRunEnvironmentCommands_EnvWithServicesNoCommands proves that an
// Environment WITH services but WITHOUT commands launches sidecars but runs NO
// ephemeral command container (no-op).
func TestRunEnvironmentCommands_EnvWithServicesNoCommands(t *testing.T) {
	f := newAgentRunFixture(t)
	env := envWithCommands(f.projectID, nil, nil) // services present, commands nil
	wireEnvironment(f, env)

	if err := f.action.Execute(context.Background(), f.newRunContext()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Launch was called (services exist) but no command container was created.
	if got := f.sidecarMgr.getLaunchCalls(); got != 1 {
		t.Errorf("expected 1 Launch call, got %d", got)
	}
	if got := len(commandCreateCalls(f)); got != 0 {
		t.Errorf("expected 0 ephemeral command containers when commands is empty, got %d", got)
	}
	// The agent container is still created.
	if got := len(agentCreateCalls(f)); got != 1 {
		t.Errorf("expected 1 agent container, got %d", got)
	}
}

// TestRunEnvironmentCommands_NoEnvNoOp proves that with no Environment at all,
// no command container is created (back-compat) — Launch is never called.
func TestRunEnvironmentCommands_NoEnvNoOp(t *testing.T) {
	f := newAgentRunFixture(t)
	// Default fixture: GetByProjectID -> NotFound, sidecarMgr guards Launch.
	if err := f.action.Execute(context.Background(), f.newRunContext()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := len(commandCreateCalls(f)); got != 0 {
		t.Errorf("expected 0 ephemeral command containers with no Environment, got %d", got)
	}
}

// TestRunEnvironmentCommands_ConnStringAndGitEnv proves the ephemeral command
// container receives the sidecar conn-strings AND the same git env the agent
// gets, and is attached to the run network.
func TestRunEnvironmentCommands_ConnStringAndGitEnv(t *testing.T) {
	f := newAgentRunFixture(t)
	env := envWithCommands(f.projectID, map[string]string{cmdMigrate(): "make migrate"}, nil)
	runNetwork := wireEnvironment(f, env)

	f.containerMgr.createFn = func(_ context.Context, opts model.ContainerOpts) (string, error) {
		return idForCommand(opts), nil
	}

	if err := f.action.Execute(context.Background(), f.newRunContext()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cmds := commandCreateCalls(f)
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command container, got %d", len(cmds))
	}
	opts := cmds[0]

	// Conn-string injected.
	if got := envValue(opts.Env, "DATABASE_URL"); got != "postgres://app:secret@db:5432/appdb" {
		t.Errorf("expected DATABASE_URL injected into command container, got %q", got)
	}
	// Same git env the agent gets.
	if got := envValue(opts.Env, "REPO_URL"); got != testAgentRepoURL {
		t.Errorf("expected REPO_URL in command container, got %q", got)
	}
	for _, key := range []string{keyGITTOKEN, "GIT_PROVIDER=", "GITHUB_TOKEN="} {
		if !hasEnvKey(opts.Env, key) {
			t.Errorf("expected git env %q in command container", key)
		}
	}
	// Attached to the run network so it reaches the sidecars.
	if opts.NetworkName != runNetwork {
		t.Errorf("expected command container on run network %q, got %q", runNetwork, opts.NetworkName)
	}
	// Has a Cmd override that clones then runs the command.
	if len(opts.Cmd) < 3 || opts.Cmd[0] != "sh" {
		t.Fatalf("expected sh -lc Cmd override, got %v", opts.Cmd)
	}
	if !strings.Contains(opts.Cmd[2], "git clone") || !strings.Contains(opts.Cmd[2], "make migrate") {
		t.Errorf("expected Cmd to clone then run the command, got %q", opts.Cmd[2])
	}
}

// TestRunEnvironmentCommands_ImageResolution proves the command image is the
// stack image when a stack is declared, and falls back to the agent image when
// it is not.
func TestRunEnvironmentCommands_ImageResolution(t *testing.T) {
	t.Run("stack image when stack declared", func(t *testing.T) {
		f := newAgentRunFixture(t)
		env := envWithCommands(f.projectID, map[string]string{cmdBuild(): "make build"}, []string{model.StackKeyGo})
		wireEnvironment(f, env)
		f.stackRepo.getByKeyFn = func(_ context.Context, key string) (*model.Stack, error) {
			if key != model.StackKeyGo {
				return nil, fmt.Errorf("unexpected stack key %q", key)
			}
			return &model.Stack{Key: key, ImageRef: stackImageRef}, nil
		}
		f.containerMgr.createFn = func(_ context.Context, opts model.ContainerOpts) (string, error) {
			return idForCommand(opts), nil
		}

		if err := f.action.Execute(context.Background(), f.newRunContext()); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		cmds := commandCreateCalls(f)
		if len(cmds) != 1 || cmds[0].Image != stackImageRef {
			t.Errorf("expected command image %q from stack, got %v", stackImageRef, cmds)
		}
	})

	t.Run("agent image fallback when no stack", func(t *testing.T) {
		f := newAgentRunFixture(t)
		env := envWithCommands(f.projectID, map[string]string{cmdBuild(): "make build"}, nil)
		wireEnvironment(f, env)
		f.containerMgr.createFn = func(_ context.Context, opts model.ContainerOpts) (string, error) {
			return idForCommand(opts), nil
		}

		runCtx := f.newRunContext()
		runCtx.Metadata["agent_image"] = agentImageRef
		if err := f.action.Execute(context.Background(), runCtx); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		cmds := commandCreateCalls(f)
		if len(cmds) != 1 || cmds[0].Image != agentImageRef {
			t.Errorf("expected command image to fall back to agent image %q, got %v", agentImageRef, cmds)
		}
	})
}

// hasEnvKey reports whether env has any entry starting with key (a KEY= prefix).
func hasEnvKey(env []string, key string) bool {
	for _, e := range env {
		if strings.HasPrefix(e, key) {
			return true
		}
	}
	return false
}

// Command-key helpers keep the literal command keys in one place; goconst does
// not skip test files, so reusing these avoids repeated string literals.
func cmdBuild() string   { return "build" }
func cmdMigrate() string { return "migrate" }
func cmdSeed() string    { return "seed" }
func cmdTest() string    { return "test" }
