package docker

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// Test-only constants: the shared agent network the adapter is configured with,
// the fixed id the fake Create returns, and capability names reused ≥3× (goconst).
const (
	runtimeSharedNetwork = "hopeitworks-agents"
	fakeContainerID      = "container-xyz"

	skillName    = "review-checklist"
	httpMCPName  = "search"
	stdioMCPName = "secret-fetcher"
)

// fakeRuntimeCM is a hand-written port.ContainerManager double scoped to the
// Runtime adapter tests. It captures the ContainerOpts handed to Create and
// counts Start/Stop/Remove/Wait so the golden can assert on them. It is distinct
// from the package's other mock (mockContainerManager) so neither test file
// depends on the other's behaviour.
type fakeRuntimeCM struct {
	mu sync.Mutex

	createdOpts []model.ContainerOpts
	startCalls  []string
	stopCalls   []string
	removeCalls []string
	waitCalls   []string

	// Hooks. Nil hooks default to "succeed".
	createFn func(opts model.ContainerOpts) (string, error)
	startFn  func(id string) error
	stopFn   func(id string) error
	removeFn func(id string) error
	waitFn   func(id string) (int, error)
}

func (f *fakeRuntimeCM) Create(_ context.Context, opts model.ContainerOpts) (string, error) {
	f.mu.Lock()
	f.createdOpts = append(f.createdOpts, opts)
	f.mu.Unlock()
	if f.createFn != nil {
		return f.createFn(opts)
	}
	return fakeContainerID, nil
}

func (f *fakeRuntimeCM) Start(_ context.Context, containerID string) error {
	f.mu.Lock()
	f.startCalls = append(f.startCalls, containerID)
	f.mu.Unlock()
	if f.startFn != nil {
		return f.startFn(containerID)
	}
	return nil
}

func (f *fakeRuntimeCM) Stop(_ context.Context, containerID string) error {
	f.mu.Lock()
	f.stopCalls = append(f.stopCalls, containerID)
	f.mu.Unlock()
	if f.stopFn != nil {
		return f.stopFn(containerID)
	}
	return nil
}

func (f *fakeRuntimeCM) Remove(_ context.Context, containerID string) error {
	f.mu.Lock()
	f.removeCalls = append(f.removeCalls, containerID)
	f.mu.Unlock()
	if f.removeFn != nil {
		return f.removeFn(containerID)
	}
	return nil
}

func (f *fakeRuntimeCM) Wait(_ context.Context, containerID string) (int, error) {
	f.mu.Lock()
	f.waitCalls = append(f.waitCalls, containerID)
	f.mu.Unlock()
	if f.waitFn != nil {
		return f.waitFn(containerID)
	}
	return 0, nil
}

func (f *fakeRuntimeCM) ListContainers(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
	return nil, nil
}

func (f *fakeRuntimeCM) ListRunningContainers(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
	return nil, nil
}

func (f *fakeRuntimeCM) CreateNetwork(_ context.Context, _ string, _ map[string]string) (string, error) {
	return "", nil
}

func (f *fakeRuntimeCM) RemoveNetwork(_ context.Context, _ string) error { return nil }

func (f *fakeRuntimeCM) ConnectContainer(_ context.Context, _, _ string, _ []string) error {
	return nil
}

func (f *fakeRuntimeCM) ListNetworks(_ context.Context, _ map[string]string) ([]model.NetworkInfo, error) {
	return nil, nil
}

func (f *fakeRuntimeCM) InspectHealth(_ context.Context, _ string) (string, error) {
	return model.HealthRunning, nil
}

func (f *fakeRuntimeCM) snapshot() ([]model.ContainerOpts, []string, []string, []string, []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.createdOpts, f.startCalls, f.stopCalls, f.removeCalls, f.waitCalls
}

func newTestRuntime(cm port.ContainerManager) *Runtime {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewRuntime(cm, runtimeSharedNetwork, logger)
}

// legacyAgentSpec reproduces the no-Environment agent launch: the COMPLETE
// legacy (non-callback) env set, in the exact order buildAgentEnv emits it
// (agent_run.go:393-410,459), plus the managed_by/run_id/step_id/story_key
// labels, the default resource limits, and a ZERO Network (no run network).
// This is the regression oracle: a realistic full 10-entry env, not a partial
// subset, so the golden proves true byte-identical pass-through.
func legacyAgentSpec() port.RunSpec {
	return port.RunSpec{
		Image: "ghcr.io/zakari/hopeitworks-agent:latest",
		// Order matches buildAgentEnv legacy mode exactly: the base block
		// (REPO_URL→CLAUDE_MD_CONTENT) then the legacy OAuth token last.
		Env: []string{
			"REPO_URL=https://github.com/acme/repo.git",
			"BRANCH_NAME=feat/s-42-test",
			"STORY_KEY=S-42",
			"PROMPT_CONTENT=do the thing",
			"PROMPT=do the thing",
			"GIT_TOKEN=gho_token",
			"GIT_PROVIDER=github",
			"GITHUB_TOKEN=gho_token",
			"CLAUDE_MD_CONTENT=# Project context\nrole: dev\n",
			"CLAUDE_CODE_OAUTH_TOKEN=oauth_token",
		},
		Labels: map[string]string{
			model.LabelManagedBy: model.LabelManagedByValue,
			model.LabelRunID:     "run-1",
			"step_id":            "step-1",
			"story_key":          "S-42",
		},
		Memory: 4294967296,
		CPUs:   2.0,
	}
}

// TestRuntime_ImplementsAgentRuntime mirrors the package-level var assertion so
// the conformance is visible to the test suite.
func TestRuntime_ImplementsAgentRuntime(_ *testing.T) {
	var _ port.AgentRuntime = NewRuntime(&fakeRuntimeCM{}, runtimeSharedNetwork, nil)
}

// TestRuntime_Launch_NoEnvironment_Golden is the docker-package golden: Launch on
// a no-Environment agent spec must produce ContainerOpts byte-identical to the
// legacy createContainer path. It calques the invariants 1→6 of the action-package
// golden TestAgentRunAction_NoEnvironment_GoldenBackCompat.
func TestRuntime_Launch_NoEnvironment_Golden(t *testing.T) {
	cm := &fakeRuntimeCM{}
	rt := newTestRuntime(cm)
	spec := legacyAgentSpec()

	h, err := rt.Launch(context.Background(), spec)
	if err != nil {
		t.Fatalf("Launch returned error: %v", err)
	}
	if h.ID != fakeContainerID {
		t.Fatalf("expected handle ID %q, got %q", fakeContainerID, h.ID)
	}

	created, startCalls, _, _, _ := cm.snapshot()
	if len(created) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(created))
	}
	opts := created[0]

	// Invariant 1: ExtraNetworks empty (single-homed exactly like before).
	if len(opts.ExtraNetworks) != 0 {
		t.Errorf("expected empty ExtraNetworks, got %v", opts.ExtraNetworks)
	}
	// Invariant 1b: Aliases nil for an agent launch (registers none of its own).
	if opts.Aliases != nil {
		t.Errorf("expected nil Aliases, got %v", opts.Aliases)
	}
	// Invariant 2: NetworkName = the shared agent network from the constructor.
	if opts.NetworkName != runtimeSharedNetwork {
		t.Errorf("expected NetworkName %q, got %q", runtimeSharedNetwork, opts.NetworkName)
	}
	// Invariant 2b: Image carried through unchanged.
	if opts.Image != spec.Image {
		t.Errorf("expected Image %q, got %q", spec.Image, opts.Image)
	}
	// Invariant 3: Entrypoint/Cmd nil for an agent launch (image entrypoint kept).
	if opts.Entrypoint != nil {
		t.Errorf("expected nil Entrypoint, got %v", opts.Entrypoint)
	}
	if opts.Cmd != nil {
		t.Errorf("expected nil Cmd, got %v", opts.Cmd)
	}

	// Invariant 4: the env slice equals the spec env, element-by-element IN ORDER.
	// Launch is a pure pass-through, so the order must be preserved byte-for-byte
	// (NO sort) — mirroring the action golden's intent at agent_run_test.go:1189.
	wantEnv := spec.Env
	gotEnv := opts.Env
	if len(gotEnv) != len(wantEnv) {
		t.Fatalf("env length mismatch: got %d (%v), want %d (%v)", len(gotEnv), gotEnv, len(wantEnv), wantEnv)
	}
	for i := range gotEnv {
		if gotEnv[i] != wantEnv[i] {
			t.Errorf("env[%d] mismatch:\n got=%q\nwant=%q", i, gotEnv[i], wantEnv[i])
		}
	}

	// Invariant 5: labels carried through unchanged.
	if opts.Labels["managed_by"] != "hopeitworks" ||
		opts.Labels["run_id"] != "run-1" ||
		opts.Labels["step_id"] != "step-1" ||
		opts.Labels["story_key"] != "S-42" {
		t.Errorf("labels changed: %v", opts.Labels)
	}
	// Invariant 6: resource limits carried through unchanged.
	if opts.Memory != 4294967296 || opts.CPUs != 2.0 {
		t.Errorf("resource limits changed: memory=%d cpus=%f", opts.Memory, opts.CPUs)
	}

	// And the container was started exactly once with the returned id.
	if len(startCalls) != 1 || startCalls[0] != fakeContainerID {
		t.Errorf("expected Start called once with %q, got %v", fakeContainerID, startCalls)
	}
}

// TestRuntime_Launch_WithRunNetwork proves a per-run network dual-homes the
// execution: ExtraNetworks carries the run network and Aliases is propagated.
func TestRuntime_Launch_WithRunNetwork(t *testing.T) {
	cm := &fakeRuntimeCM{}
	rt := newTestRuntime(cm)

	const runNet = "hopeitworks-run-xyz"
	aliases := map[string]string{runNet: "agent"}
	spec := legacyAgentSpec()
	spec.Network = port.RunNetwork{Name: runNet, Aliases: aliases}

	if _, err := rt.Launch(context.Background(), spec); err != nil {
		t.Fatalf("Launch returned error: %v", err)
	}

	created, _, _, _, _ := cm.snapshot()
	if len(created) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(created))
	}
	opts := created[0]

	// Shared network stays primary; run network is the extra attachment.
	if opts.NetworkName != runtimeSharedNetwork {
		t.Errorf("expected shared NetworkName %q, got %q", runtimeSharedNetwork, opts.NetworkName)
	}
	if len(opts.ExtraNetworks) != 1 || opts.ExtraNetworks[0] != runNet {
		t.Errorf("expected ExtraNetworks=[%s], got %v", runNet, opts.ExtraNetworks)
	}
	if got := opts.Aliases[runNet]; got != "agent" {
		t.Errorf("expected Aliases[%s]=agent, got %q (full: %v)", runNet, got, opts.Aliases)
	}
}

// TestRuntime_Launch_StartFails proves a Start failure surfaces as a Launch error
// even though Create succeeded, AND that the created-but-unstarted container is
// torn down inline (Stop then Remove) so it does not leak — mirroring the legacy
// deferred cleanupContainer.
func TestRuntime_Launch_StartFails(t *testing.T) {
	cm := &fakeRuntimeCM{
		startFn: func(_ string) error { return errors.New("docker start error") },
	}
	rt := newTestRuntime(cm)

	if _, err := rt.Launch(context.Background(), legacyAgentSpec()); err == nil {
		t.Fatal("expected Launch to return an error when Start fails, got nil")
	}

	created, startCalls, stopCalls, removeCalls, _ := cm.snapshot()
	if len(created) != 1 {
		t.Errorf("expected Create to have run once, got %d", len(created))
	}
	if len(startCalls) != 1 {
		t.Errorf("expected Start to have been attempted once, got %d", len(startCalls))
	}
	// Inline best-effort teardown of the leaked container.
	if len(stopCalls) != 1 || stopCalls[0] != fakeContainerID {
		t.Errorf("expected Stop called once with %q after Start failure, got %v", fakeContainerID, stopCalls)
	}
	if len(removeCalls) != 1 || removeCalls[0] != fakeContainerID {
		t.Errorf("expected Remove called once with %q after Start failure, got %v", fakeContainerID, removeCalls)
	}
}

// TestRuntime_Launch_CreateFails proves a Create failure surfaces as a Launch
// error and Start is never attempted.
func TestRuntime_Launch_CreateFails(t *testing.T) {
	cm := &fakeRuntimeCM{
		createFn: func(_ model.ContainerOpts) (string, error) { return "", errors.New("docker create error") },
	}
	rt := newTestRuntime(cm)

	if _, err := rt.Launch(context.Background(), legacyAgentSpec()); err == nil {
		t.Fatal("expected Launch to return an error when Create fails, got nil")
	}
	if _, startCalls, _, _, _ := cm.snapshot(); len(startCalls) != 0 {
		t.Errorf("expected Start NOT to be attempted when Create fails, got %d calls", len(startCalls))
	}
}

// TestRuntime_Wait maps the container exit code into RunResult.ExitCode.
func TestRuntime_Wait(t *testing.T) {
	cm := &fakeRuntimeCM{
		waitFn: func(_ string) (int, error) { return 42, nil },
	}
	rt := newTestRuntime(cm)

	res, err := rt.Wait(context.Background(), port.RunHandle{ID: fakeContainerID})
	if err != nil {
		t.Fatalf("Wait returned error: %v", err)
	}
	if res.ExitCode != 42 {
		t.Errorf("expected ExitCode 42, got %d", res.ExitCode)
	}
	if _, _, _, _, waitCalls := cm.snapshot(); len(waitCalls) != 1 || waitCalls[0] != fakeContainerID {
		t.Errorf("expected Wait called once with %q, got %v", fakeContainerID, waitCalls)
	}
}

// TestRuntime_Wait_Error wraps a containerMgr.Wait error.
func TestRuntime_Wait_Error(t *testing.T) {
	cm := &fakeRuntimeCM{
		waitFn: func(_ string) (int, error) { return 0, errors.New("wait boom") },
	}
	rt := newTestRuntime(cm)

	if _, err := rt.Wait(context.Background(), port.RunHandle{ID: fakeContainerID}); err == nil {
		t.Fatal("expected Wait to return an error, got nil")
	}
}

// TestRuntime_Stop calls Stop then Remove on the containerMgr, mirroring the
// legacy cleanupContainer.
func TestRuntime_Stop(t *testing.T) {
	cm := &fakeRuntimeCM{}
	rt := newTestRuntime(cm)

	if err := rt.Stop(context.Background(), port.RunHandle{ID: fakeContainerID}); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}

	_, _, stopCalls, removeCalls, _ := cm.snapshot()
	if len(stopCalls) != 1 || stopCalls[0] != fakeContainerID {
		t.Errorf("expected Stop called once with %q, got %v", fakeContainerID, stopCalls)
	}
	if len(removeCalls) != 1 || removeCalls[0] != fakeContainerID {
		t.Errorf("expected Remove called once with %q, got %v", fakeContainerID, removeCalls)
	}
}

// TestRuntime_Stop_RemoveError returns the Remove error but still attempts Stop
// first (best-effort, Stop errors are logged not returned).
func TestRuntime_Stop_RemoveError(t *testing.T) {
	cm := &fakeRuntimeCM{
		stopFn:   func(_ string) error { return errors.New("stop boom") },
		removeFn: func(_ string) error { return errors.New("remove boom") },
	}
	rt := newTestRuntime(cm)

	if err := rt.Stop(context.Background(), port.RunHandle{ID: fakeContainerID}); err == nil {
		t.Fatal("expected Stop to return the Remove error, got nil")
	}
	_, _, stopCalls, removeCalls, _ := cm.snapshot()
	if len(stopCalls) != 1 {
		t.Errorf("expected Stop attempted once despite error, got %d", len(stopCalls))
	}
	if len(removeCalls) != 1 {
		t.Errorf("expected Remove attempted once, got %d", len(removeCalls))
	}
}

func TestRuntime_SupportedCapabilities(t *testing.T) {
	got := newTestRuntime(&fakeRuntimeCM{}).SupportedCapabilities()
	want := model.CapabilitySet{
		Skills:          true,
		MCPServersHTTP:  true,
		MCPServersStdio: true,
		ToolPolicy:      true,
	}
	if got != want {
		t.Fatalf("SupportedCapabilities() = %+v, want %+v", got, want)
	}
}

// TestRuntime_Provision_AllSupported proves Docker — a full Pod-expressible
// substrate — applies every capability with zero warnings on a mixed spec, and
// never returns a blocking error.
func TestRuntime_Provision_AllSupported(t *testing.T) {
	tests := []struct {
		name        string
		spec        model.CapabilitySpec
		wantApplied []string
	}{
		{
			name:        "empty spec applies nothing",
			spec:        model.CapabilitySpec{},
			wantApplied: nil,
		},
		{
			name: "skill applied",
			spec: model.CapabilitySpec{Skills: []model.SkillSpec{{Name: skillName}}},
			wantApplied: []string{
				model.CapabilityKindSkill + "/" + skillName,
			},
		},
		{
			name: "stdio mcp is supported by docker (unlike microsandbox)",
			spec: model.CapabilitySpec{
				MCPServers: []model.MCPServerSpec{{Name: stdioMCPName, Transport: transportStdio}},
			},
			wantApplied: []string{
				model.CapabilityKindMCPServer + "/" + stdioMCPName,
			},
		},
		{
			name: "mixed: skills + http mcp + stdio mcp + tool policy all applied, 0 warnings",
			spec: model.CapabilitySpec{
				Skills: []model.SkillSpec{{Name: skillName}},
				MCPServers: []model.MCPServerSpec{
					{Name: httpMCPName, Transport: transportHTTP},
					{Name: stdioMCPName, Transport: transportStdio},
				},
				ToolPolicy: &model.ToolPolicySpec{Allow: []string{"Bash"}, Deny: []string{"WebFetch"}},
			},
			wantApplied: []string{
				model.CapabilityKindSkill + "/" + skillName,
				model.CapabilityKindMCPServer + "/" + httpMCPName,
				model.CapabilityKindMCPServer + "/" + stdioMCPName,
				model.CapabilityKindToolPolicy,
			},
		},
	}

	rt := newTestRuntime(&fakeRuntimeCM{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := rt.Provision(context.Background(), tt.spec)
			if err != nil {
				t.Fatalf("Provision returned error %v; warn+skip must never error", err)
			}
			if len(res.Warnings) != 0 {
				t.Errorf("expected 0 warnings (docker supports all), got %v", res.Warnings)
			}
			assertRuntimeRefs(t, "applied", res.Applied, tt.wantApplied)
		})
	}
}

// TestRuntime_Provision_UnknownTransport warns (never errors) on an MCP transport
// docker does not recognise, and the warning reason names the offending transport.
func TestRuntime_Provision_UnknownTransport(t *testing.T) {
	const badTransport = "grpc"
	rt := newTestRuntime(&fakeRuntimeCM{})
	res, err := rt.Provision(context.Background(), model.CapabilitySpec{
		MCPServers: []model.MCPServerSpec{{Name: "weird", Transport: badTransport}},
	})
	if err != nil {
		t.Fatalf("Provision returned error %v; warn+skip must never error", err)
	}
	if len(res.Applied) != 0 {
		t.Errorf("expected nothing applied for unknown transport, got %v", res.Applied)
	}
	if len(res.Warnings) != 1 {
		t.Fatalf("expected exactly 1 warning, got %v", res.Warnings)
	}
	if !strings.Contains(res.Warnings[0].Reason, badTransport) {
		t.Errorf("expected warning reason to name the transport %q, got %q", badTransport, res.Warnings[0].Reason)
	}
}

// assertRuntimeRefs compares an unordered set of refs ignoring order.
func assertRuntimeRefs(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %v (len %d), want %v (len %d)", label, got, len(got), want, len(want))
	}
	gotSet := make(map[string]int, len(got))
	for _, g := range got {
		gotSet[g]++
	}
	for _, w := range want {
		if gotSet[w] == 0 {
			t.Fatalf("%s: missing %q in %v", label, w, got)
		}
		gotSet[w]--
	}
}
