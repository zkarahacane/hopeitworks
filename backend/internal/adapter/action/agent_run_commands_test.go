package action_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
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

	// BLOCKER #1 regression guard: the ENTRYPOINT must be overridden to the
	// shell. Stack/agent images bake ENTRYPOINT ["agent-runtime"]; without this
	// override Docker would run `agent-runtime sh -lc <script>` and the command
	// would NEVER execute (and agent-runtime would receive the token+conn-strings
	// in env). The shell goes in Entrypoint, the script in Cmd[0].
	wantEntrypoint := []string{"sh", "-lc"}
	if len(opts.Entrypoint) != len(wantEntrypoint) {
		t.Fatalf("expected Entrypoint %v, got %v", wantEntrypoint, opts.Entrypoint)
	}
	for i := range wantEntrypoint {
		if opts.Entrypoint[i] != wantEntrypoint[i] {
			t.Fatalf("expected Entrypoint %v, got %v", wantEntrypoint, opts.Entrypoint)
		}
	}
	if len(opts.Cmd) != 1 {
		t.Fatalf("expected Cmd to be a single script arg, got %v", opts.Cmd)
	}
	script := opts.Cmd[0]
	if !strings.Contains(script, "git clone") || !strings.Contains(script, "make migrate") {
		t.Errorf("expected script to clone then run the command, got %q", script)
	}
	// STRICT clone (#4): set -e, no silent default-branch fallback.
	if !strings.HasPrefix(script, "set -e;") {
		t.Errorf("expected script to start with 'set -e;' (strict clone), got %q", script)
	}
	if strings.Contains(script, "|| git clone") {
		t.Errorf("script must NOT fall back to default branch on clone failure, got %q", script)
	}
	// The token must NOT appear in argv: the clone URL is referenced via env var.
	if !strings.Contains(script, "$HOPEITWORKS_CLONE_URL") {
		t.Errorf("expected clone URL referenced via env var, got %q", script)
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

	// #3: a declared-but-uncatalogued stack key (typo / not seeded) yields a
	// NotFound from the stack repo. That must NOT fail the run — it degrades to
	// the agent image fallback.
	t.Run("agent image fallback when stack NotFound", func(t *testing.T) {
		f := newAgentRunFixture(t)
		env := envWithCommands(f.projectID, map[string]string{cmdBuild(): "make build"}, []string{"typo-stack"})
		wireEnvironment(f, env)
		f.stackRepo.getByKeyFn = func(_ context.Context, key string) (*model.Stack, error) {
			return nil, errors.NewNotFound("stack", key)
		}
		f.containerMgr.createFn = func(_ context.Context, opts model.ContainerOpts) (string, error) {
			return idForCommand(opts), nil
		}

		runCtx := f.newRunContext()
		runCtx.Metadata["agent_image"] = agentImageRef
		if err := f.action.Execute(context.Background(), runCtx); err != nil {
			t.Fatalf("expected NotFound to fall back, not fail the run, got %v", err)
		}
		cmds := commandCreateCalls(f)
		if len(cmds) != 1 || cmds[0].Image != agentImageRef {
			t.Errorf("expected fallback to agent image on stack NotFound, got %v", cmds)
		}
	})

	// A non-NotFound stack repo error (transient DB failure) MUST fail the run —
	// it is not masked by the fallback.
	t.Run("non-NotFound stack error fails the run", func(t *testing.T) {
		f := newAgentRunFixture(t)
		env := envWithCommands(f.projectID, map[string]string{cmdBuild(): "make build"}, []string{model.StackKeyGo})
		wireEnvironment(f, env)
		f.stackRepo.getByKeyFn = func(_ context.Context, _ string) (*model.Stack, error) {
			return nil, fmt.Errorf("connection refused")
		}

		err := f.action.Execute(context.Background(), f.newRunContext())
		if err == nil {
			t.Fatal("expected a non-NotFound stack error to fail the run, got nil")
		}
		if got := len(commandCreateCalls(f)); got != 0 {
			t.Errorf("expected no command container when image resolution errors, got %d", got)
		}
		if got := len(agentCreateCalls(f)); got != 0 {
			t.Errorf("expected no agent container when image resolution errors, got %d", got)
		}
	})
}

// TestRunEnvironmentCommands_ErrorOmitsLogTail is the BLOCKER #2 regression
// guard: when a command fails, the error that PROPAGATES (and is persisted to
// run.error_message + broadcast over SSE) must contain ONLY the command key and
// exit code — never the container's stdout/stderr, which could carry a dumped
// token or DB password.
func TestRunEnvironmentCommands_ErrorOmitsLogTail(t *testing.T) {
	f := newAgentRunFixture(t)
	env := envWithCommands(f.projectID, map[string]string{cmdMigrate(): "make migrate"}, nil)
	wireEnvironment(f, env)

	f.containerMgr.createFn = func(_ context.Context, opts model.ContainerOpts) (string, error) {
		return idForCommand(opts), nil
	}
	f.containerMgr.waitFn = func(_ context.Context, _ string) (int, error) {
		return 1, nil // command fails
	}
	// The log stream emits a "secret" line that would leak if embedded in the error.
	const secretLine = "GITHUB_TOKEN=ghp_supersecrettoken_should_never_propagate"
	f.logStreamer.streamLogsFn = func(_ context.Context, _, _, _ string) (<-chan model.LogEvent, <-chan int, error) {
		logCh := make(chan model.LogEvent, 1)
		doneCh := make(chan int, 1)
		logCh <- model.LogEvent{Message: secretLine}
		close(logCh)
		doneCh <- 1
		return logCh, doneCh, nil
	}

	err := f.action.Execute(context.Background(), f.newRunContext())
	if err == nil {
		t.Fatal("expected fail-fast error, got nil")
	}
	if strings.Contains(err.Error(), secretLine) || strings.Contains(err.Error(), "ghp_") {
		t.Fatalf("error leaked the log tail / secret: %q", err.Error())
	}
	// It must still carry the actionable bits.
	if !strings.Contains(err.Error(), cmdMigrate()) || !strings.Contains(err.Error(), "exit code 1") {
		t.Errorf("expected error to name the command and exit code, got %q", err.Error())
	}
}

// TestRunEnvironmentCommands_RunIDLabelWithoutServices is the #5 regression
// guard: a project with commands but NO sidecar services still stamps run_id on
// the ephemeral container so the GC reaper can find it.
func TestRunEnvironmentCommands_RunIDLabelWithoutServices(t *testing.T) {
	f := newAgentRunFixture(t)
	// Environment with a command but ZERO services -> no sidecar launch path.
	env := &model.Environment{
		ProjectID: f.projectID,
		Commands:  map[string]string{cmdBuild(): "make build"},
	}
	f.environmentRepo.getByProjectIDFn = func(_ context.Context, _ uuid.UUID) (*model.Environment, error) {
		return env, nil
	}
	f.containerMgr.createFn = func(_ context.Context, opts model.ContainerOpts) (string, error) {
		return idForCommand(opts), nil
	}

	if err := f.action.Execute(context.Background(), f.newRunContext()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// No sidecar launch (no services), but the command still ran.
	if got := f.sidecarMgr.getLaunchCalls(); got != 0 {
		t.Errorf("expected 0 Launch calls (no services), got %d", got)
	}
	cmds := commandCreateCalls(f)
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command container, got %d", len(cmds))
	}
	if got := cmds[0].Labels[model.LabelRunID]; got != f.runID.String() {
		t.Errorf("expected run_id label %q on ephemeral container, got %q", f.runID.String(), got)
	}
	// No run network to attach to: NetworkName stays empty.
	if cmds[0].NetworkName != "" {
		t.Errorf("expected no run network without services, got %q", cmds[0].NetworkName)
	}
}

// TestRunEnvironmentCommands_WaitError is the #7 guard: a Wait error (timeout /
// daemon error) fails the run, creates no agent container, removes the ephemeral
// container, and still tears down the sidecars.
func TestRunEnvironmentCommands_WaitError(t *testing.T) {
	f := newAgentRunFixture(t)
	env := envWithCommands(f.projectID, map[string]string{cmdMigrate(): "make migrate"}, nil)
	wireEnvironment(f, env)

	f.containerMgr.createFn = func(_ context.Context, opts model.ContainerOpts) (string, error) {
		return idForCommand(opts), nil
	}
	f.containerMgr.waitFn = func(_ context.Context, _ string) (int, error) {
		return 0, fmt.Errorf("docker wait failed")
	}

	err := f.action.Execute(context.Background(), f.newRunContext())
	if err == nil {
		t.Fatal("expected error from Wait failure, got nil")
	}
	// The ephemeral container was removed (defer removeEphemeral -> Stop+Remove).
	f.containerMgr.mu.Lock()
	removeCalls := append([]string(nil), f.containerMgr.removeCalls...)
	f.containerMgr.mu.Unlock()
	if len(removeCalls) == 0 {
		t.Error("expected the ephemeral command container to be removed")
	}
	// No agent container was created.
	if got := len(agentCreateCalls(f)); got != 0 {
		t.Errorf("expected 0 agent containers after a Wait error, got %d", got)
	}
	// Sidecars still torn down (deferred Cleanup).
	if got := f.sidecarMgr.getCleanupCalls(); got != 1 {
		t.Errorf("expected 1 sidecar Cleanup, got %d", got)
	}
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
