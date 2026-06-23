package docker

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// mockContainerManager is a hand-written test double for port.ContainerManager.
// It records calls and exposes configurable behaviour via function hooks.
type mockContainerManager struct {
	mu sync.Mutex

	createdNetworks []string
	removedNetworks []string
	createdOpts     []model.ContainerOpts
	startedIDs      []string
	stoppedIDs      []string
	removedIDs      []string

	// Hooks. Defaults are "succeed".
	createNetworkFn func(name string) (string, error)
	createFn        func(opts model.ContainerOpts) (string, error)
	startFn         func(id string) error
	inspectHealthFn func(id string) (string, error)
	removeNetworkFn func(nameOrID string) error
	listNetworksFn  func() ([]model.NetworkInfo, error)
	listContainerFn func() ([]port.ContainerInfo, error)
	listRunningFn   func() ([]port.ContainerInfo, error)
}

func newMockCM() *mockContainerManager {
	return &mockContainerManager{}
}

func (m *mockContainerManager) Create(_ context.Context, opts model.ContainerOpts) (string, error) {
	m.mu.Lock()
	m.createdOpts = append(m.createdOpts, opts)
	m.mu.Unlock()
	if m.createFn != nil {
		return m.createFn(opts)
	}
	return "ctr-" + opts.Labels[labelSidecar], nil
}

func (m *mockContainerManager) Start(_ context.Context, containerID string) error {
	m.mu.Lock()
	m.startedIDs = append(m.startedIDs, containerID)
	m.mu.Unlock()
	if m.startFn != nil {
		return m.startFn(containerID)
	}
	return nil
}

func (m *mockContainerManager) getStartedIDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.startedIDs))
	copy(out, m.startedIDs)
	return out
}

func (m *mockContainerManager) Stop(_ context.Context, containerID string) error {
	m.mu.Lock()
	m.stoppedIDs = append(m.stoppedIDs, containerID)
	m.mu.Unlock()
	return nil
}

func (m *mockContainerManager) Remove(_ context.Context, containerID string) error {
	m.mu.Lock()
	m.removedIDs = append(m.removedIDs, containerID)
	m.mu.Unlock()
	return nil
}

func (m *mockContainerManager) Wait(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (m *mockContainerManager) ListContainers(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
	if m.listContainerFn != nil {
		return m.listContainerFn()
	}
	return nil, nil
}

func (m *mockContainerManager) ListRunningContainers(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
	if m.listRunningFn != nil {
		return m.listRunningFn()
	}
	return nil, nil
}

func (m *mockContainerManager) CreateNetwork(_ context.Context, name string, _ map[string]string) (string, error) {
	m.mu.Lock()
	m.createdNetworks = append(m.createdNetworks, name)
	m.mu.Unlock()
	if m.createNetworkFn != nil {
		return m.createNetworkFn(name)
	}
	return "id-" + name, nil
}

func (m *mockContainerManager) RemoveNetwork(_ context.Context, nameOrID string) error {
	m.mu.Lock()
	m.removedNetworks = append(m.removedNetworks, nameOrID)
	m.mu.Unlock()
	if m.removeNetworkFn != nil {
		return m.removeNetworkFn(nameOrID)
	}
	return nil
}

func (m *mockContainerManager) ConnectContainer(_ context.Context, _, _ string, _ []string) error {
	return nil
}

func (m *mockContainerManager) ListNetworks(_ context.Context, _ map[string]string) ([]model.NetworkInfo, error) {
	if m.listNetworksFn != nil {
		return m.listNetworksFn()
	}
	return nil, nil
}

func (m *mockContainerManager) InspectHealth(_ context.Context, containerID string) (string, error) {
	if m.inspectHealthFn != nil {
		return m.inspectHealthFn(containerID)
	}
	return model.HealthHealthy, nil
}

func newTestSidecarManager(cm *mockContainerManager) *SidecarManager {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	s := NewDockerSidecarManager(cm, logger)
	// Tighten timings so tests are fast.
	s.readinessTimeout = 200 * time.Millisecond
	s.readinessInterval = 5 * time.Millisecond
	s.runningGrace = 0
	return s
}

func pgEnv() *model.Environment {
	return &model.Environment{
		ID:        uuid.New(),
		ProjectID: uuid.New(),
		Services: []model.EnvironmentService{
			{Name: "db", Image: "postgres:16", Env: map[string]string{"POSTGRES_PASSWORD": "x"}},
		},
	}
}

func TestSidecarLaunch_NilEnv_NoOp(t *testing.T) {
	cm := newMockCM()
	s := newTestSidecarManager(cm)

	runID := uuid.New()
	sc, err := s.Launch(context.Background(), runID, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sc == nil {
		t.Fatal("expected non-nil empty context")
	}
	if sc.RunID != runID {
		t.Errorf("expected RunID propagated")
	}
	if sc.NetworkName != "" {
		t.Errorf("expected no network, got %s", sc.NetworkName)
	}
	if len(cm.createdNetworks) != 0 || len(cm.createdOpts) != 0 {
		t.Errorf("expected no docker calls: nets=%d ctrs=%d", len(cm.createdNetworks), len(cm.createdOpts))
	}
}

func TestSidecarLaunch_EmptyServices_NoOp(t *testing.T) {
	cm := newMockCM()
	s := newTestSidecarManager(cm)

	env := &model.Environment{ID: uuid.New(), Services: nil}
	sc, err := s.Launch(context.Background(), uuid.New(), env)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sc.NetworkName != "" {
		t.Errorf("expected no network")
	}
	if len(cm.createdNetworks) != 0 || len(cm.createdOpts) != 0 {
		t.Errorf("expected no docker calls")
	}
}

func TestSidecarLaunch_HappyPath(t *testing.T) {
	cm := newMockCM() // defaults: create ok, start ok, health=healthy
	s := newTestSidecarManager(cm)

	runID := uuid.New()
	sc, err := s.Launch(context.Background(), runID, pgEnv())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	wantNet := sidecarNetworkPrefix + runID.String()
	if sc.NetworkName != wantNet {
		t.Errorf("expected network %s, got %s", wantNet, sc.NetworkName)
	}
	if len(cm.createdNetworks) != 1 || cm.createdNetworks[0] != wantNet {
		t.Errorf("expected one network %s, got %v", wantNet, cm.createdNetworks)
	}
	if sc.ServiceAddrs["db"] != "db" {
		t.Errorf("expected db addr = db, got %q", sc.ServiceAddrs["db"])
	}
	if _, ok := sc.ContainerIDs["db"]; !ok {
		t.Errorf("expected db container id recorded")
	}
	if started := cm.getStartedIDs(); len(started) != 1 {
		t.Errorf("expected exactly one container started, got %d", len(started))
	}
	// Postgres has a healthcheck configured.
	if len(cm.createdOpts) != 1 || cm.createdOpts[0].Healthcheck == nil {
		t.Errorf("expected postgres healthcheck to be configured")
	}
	// Isolation: the sidecar is attached to the run network DIRECTLY at creation
	// (NetworkName, not ExtraNetworks) so it never lands on the default bridge.
	if cm.createdOpts[0].NetworkName != wantNet {
		t.Errorf("expected sidecar created on run network via NetworkName, got %q", cm.createdOpts[0].NetworkName)
	}
	if len(cm.createdOpts[0].ExtraNetworks) != 0 {
		t.Errorf("expected no ExtraNetworks (no bridge dual-home), got %v", cm.createdOpts[0].ExtraNetworks)
	}
	// DNS alias svc.Name on the run network (applied on the primary endpoint).
	if cm.createdOpts[0].Aliases[wantNet] != "db" {
		t.Errorf("expected dns alias db on run network, got %v", cm.createdOpts[0].Aliases)
	}
}

func TestSidecarLaunch_UnknownType_FallbackRunning(t *testing.T) {
	cm := newMockCM()
	cm.inspectHealthFn = func(_ string) (string, error) { return model.HealthRunning, nil }
	s := newTestSidecarManager(cm)

	env := &model.Environment{
		Services: []model.EnvironmentService{{Name: "custom", Image: "ghcr.io/acme/thing:1"}},
	}
	sc, err := s.Launch(context.Background(), uuid.New(), env)
	if err != nil {
		t.Fatalf("expected no error for unknown type running fallback, got %v", err)
	}
	if sc.ServiceAddrs["custom"] != "custom" {
		t.Errorf("expected custom addr recorded")
	}
	// No healthcheck for unknown types.
	if cm.createdOpts[0].Healthcheck != nil {
		t.Errorf("expected no healthcheck for unknown service type")
	}
}

func TestSidecarLaunch_RollbackOnReadinessTimeout(t *testing.T) {
	cm := newMockCM()
	// Postgres healthcheck never goes healthy -> timeout -> rollback.
	cm.inspectHealthFn = func(_ string) (string, error) { return model.HealthStarting, nil }
	s := newTestSidecarManager(cm)

	runID := uuid.New()
	sc, err := s.Launch(context.Background(), runID, pgEnv())
	if err == nil {
		t.Fatal("expected readiness timeout error, got nil")
	}
	if sc != nil {
		t.Errorf("expected nil context on error, got %+v", sc)
	}
	// Rollback: the started container was stopped+removed and the network removed.
	if len(cm.stoppedIDs) != 1 || len(cm.removedIDs) != 1 {
		t.Errorf("expected 1 stop + 1 remove on rollback, got stops=%d removes=%d", len(cm.stoppedIDs), len(cm.removedIDs))
	}
	wantNet := sidecarNetworkPrefix + runID.String()
	if len(cm.removedNetworks) != 1 || cm.removedNetworks[0] != wantNet {
		t.Errorf("expected network %s removed on rollback, got %v", wantNet, cm.removedNetworks)
	}
}

func TestSidecarLaunch_RollbackOnCreateError(t *testing.T) {
	cm := newMockCM()
	calls := 0
	cm.createFn = func(opts model.ContainerOpts) (string, error) {
		calls++
		if calls == 2 {
			return "", errors.New("boom on second create")
		}
		return "ctr-" + opts.Labels[labelSidecar], nil
	}
	s := newTestSidecarManager(cm)

	env := &model.Environment{
		Services: []model.EnvironmentService{
			{Name: "a", Image: "redis:7"},
			{Name: "b", Image: "redis:7"},
		},
	}
	runID := uuid.New()
	_, err := s.Launch(context.Background(), runID, env)
	if err == nil {
		t.Fatal("expected create error, got nil")
	}
	// First container "a" was created+started; it must be torn down on rollback.
	if len(cm.removedIDs) != 1 || cm.removedIDs[0] != "ctr-a" {
		t.Errorf("expected ctr-a removed on rollback, got %v", cm.removedIDs)
	}
	if len(cm.removedNetworks) != 1 {
		t.Errorf("expected network removed on rollback, got %v", cm.removedNetworks)
	}
}

func TestSidecarLaunch_RollbackOnUnhealthy(t *testing.T) {
	cm := newMockCM()
	cm.inspectHealthFn = func(_ string) (string, error) { return model.HealthUnhealthy, nil }
	s := newTestSidecarManager(cm)

	_, err := s.Launch(context.Background(), uuid.New(), pgEnv())
	if err == nil {
		t.Fatal("expected unhealthy error, got nil")
	}
	if len(cm.removedIDs) != 1 || len(cm.removedNetworks) != 1 {
		t.Errorf("expected full rollback, got removes=%d nets=%d", len(cm.removedIDs), len(cm.removedNetworks))
	}
}

func TestSidecarLaunch_RollbackWithCancelledContext(t *testing.T) {
	// If the caller's context is cancelled mid-Launch (InspectHealth returns
	// context.Canceled), teardown must still run: it uses a context detached from
	// the caller's. The mock ignores ctx, but the real SDK would not — this proves
	// rollback no longer reuses the dead ctx.
	cm := newMockCM()
	cm.inspectHealthFn = func(_ string) (string, error) { return "", context.Canceled }
	s := newTestSidecarManager(cm)

	_, err := s.Launch(context.Background(), uuid.New(), pgEnv())
	if err == nil {
		t.Fatal("expected error from cancelled readiness, got nil")
	}
	if len(cm.stoppedIDs) != 1 || len(cm.removedIDs) != 1 || len(cm.removedNetworks) != 1 {
		t.Errorf("expected full teardown despite cancellation: stops=%d removes=%d nets=%d",
			len(cm.stoppedIDs), len(cm.removedIDs), len(cm.removedNetworks))
	}
}

func TestSidecarLaunch_BakedImageHealthcheckReady(t *testing.T) {
	// Unknown service type whose IMAGE bakes its own HEALTHCHECK: InspectHealth
	// returns "healthy" -> must be treated as ready (no false timeout/rollback),
	// even though we configured no profile healthcheck.
	cm := newMockCM()
	cm.inspectHealthFn = func(_ string) (string, error) { return model.HealthHealthy, nil }
	s := newTestSidecarManager(cm)

	env := &model.Environment{
		Services: []model.EnvironmentService{{Name: "custom", Image: "ghcr.io/acme/thing:1"}},
	}
	sc, err := s.Launch(context.Background(), uuid.New(), env)
	if err != nil {
		t.Fatalf("expected ready via baked image healthcheck, got %v", err)
	}
	if sc.ServiceAddrs["custom"] != "custom" {
		t.Errorf("expected custom addr recorded")
	}
	// We did not configure a profile healthcheck for the unknown type.
	if cm.createdOpts[0].Healthcheck != nil {
		t.Errorf("expected no profile healthcheck for unknown type")
	}
	if len(cm.removedIDs) != 0 || len(cm.removedNetworks) != 0 {
		t.Errorf("expected no rollback, got removes=%d nets=%d", len(cm.removedIDs), len(cm.removedNetworks))
	}
}

func TestSidecarLaunch_BakedImageUnhealthyRollback(t *testing.T) {
	// Unknown type, image healthcheck reports unhealthy -> failure + rollback.
	cm := newMockCM()
	cm.inspectHealthFn = func(_ string) (string, error) { return model.HealthUnhealthy, nil }
	s := newTestSidecarManager(cm)

	env := &model.Environment{
		Services: []model.EnvironmentService{{Name: "custom", Image: "ghcr.io/acme/thing:1"}},
	}
	_, err := s.Launch(context.Background(), uuid.New(), env)
	if err == nil {
		t.Fatal("expected unhealthy error, got nil")
	}
	if len(cm.removedIDs) != 1 || len(cm.removedNetworks) != 1 {
		t.Errorf("expected full rollback, got removes=%d nets=%d", len(cm.removedIDs), len(cm.removedNetworks))
	}
}

func TestSidecarStop_NilContext_NoOp(t *testing.T) {
	cm := newMockCM()
	s := newTestSidecarManager(cm)

	if err := s.Stop(context.Background(), nil); err != nil {
		t.Fatalf("expected nil for nil context, got %v", err)
	}
	if err := s.Stop(context.Background(), &port.SidecarContext{}); err != nil {
		t.Fatalf("expected nil for empty context, got %v", err)
	}
	if len(cm.stoppedIDs) != 0 {
		t.Errorf("expected no stop calls, got %d", len(cm.stoppedIDs))
	}
}

func TestSidecarCleanup_Idempotent(t *testing.T) {
	cm := newMockCM()
	s := newTestSidecarManager(cm)

	// Nil + empty contexts are no-ops.
	if err := s.Cleanup(context.Background(), nil); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if err := s.Cleanup(context.Background(), &port.SidecarContext{}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if len(cm.removedIDs) != 0 || len(cm.removedNetworks) != 0 {
		t.Errorf("expected no teardown for empty context")
	}

	// Populated context tears down containers + network.
	sc := &port.SidecarContext{
		NetworkName:  "hopeitworks-run-x",
		ContainerIDs: map[string]string{"db": "ctr-db"},
	}
	if err := s.Cleanup(context.Background(), sc); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if len(cm.stoppedIDs) != 1 || len(cm.removedIDs) != 1 || len(cm.removedNetworks) != 1 {
		t.Errorf("expected teardown: stops=%d removes=%d nets=%d", len(cm.stoppedIDs), len(cm.removedIDs), len(cm.removedNetworks))
	}

	// Calling again is harmless (RemoveNetwork/Remove are idempotent at adapter).
	if err := s.Cleanup(context.Background(), sc); err != nil {
		t.Fatalf("expected nil on second cleanup, got %v", err)
	}
}

func TestSidecarGC_Windowed(t *testing.T) {
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	cm := newMockCM()
	cm.listNetworksFn = func() ([]model.NetworkInfo, error) {
		return []model.NetworkInfo{
			{ID: "old", Name: "hopeitworks-run-old", Labels: map[string]string{labelRunID: "run-old"}, CreatedAt: now.Add(-2 * time.Hour)},
			{ID: "recent", Name: "hopeitworks-run-recent", Labels: map[string]string{labelRunID: "run-recent"}, CreatedAt: now.Add(-1 * time.Minute)},
			{ID: "live", Name: "hopeitworks-run-live", Labels: map[string]string{labelRunID: "run-live"}, CreatedAt: now.Add(-2 * time.Hour)},
			{ID: "exited", Name: "hopeitworks-run-exited", Labels: map[string]string{labelRunID: "run-exited"}, CreatedAt: now.Add(-2 * time.Hour)},
		}, nil
	}
	// Orphan detection uses RUNNING containers only: run-live has a running
	// container; run-exited's container is exited (absent from running list) so
	// its old network must be reaped.
	cm.listRunningFn = func() ([]port.ContainerInfo, error) {
		return []port.ContainerInfo{
			{ID: "c1", Labels: map[string]string{labelRunID: "run-live"}},
		}, nil
	}
	s := newTestSidecarManager(cm)
	s.now = func() time.Time { return now }

	if err := s.GC(context.Background(), 30*time.Minute); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// old + exited removed; recent within window kept; live has a running
	// container kept.
	if len(cm.removedNetworks) != 2 {
		t.Fatalf("expected 2 networks removed, got %v", cm.removedNetworks)
	}
	got := map[string]bool{cm.removedNetworks[0]: true, cm.removedNetworks[1]: true}
	if !got["old"] || !got["exited"] {
		t.Errorf("expected 'old' and 'exited' removed, got %v", cm.removedNetworks)
	}
}

// TestSidecarGC_ReapsCommandContainers proves the GC reaper removes ephemeral
// env-command containers (role=env_command) that are exited and older than the
// window, while leaving running ones and recently-created ones alone. This is
// the safety net for an API crash mid-command where the defer removeEphemeral
// never ran.
func TestSidecarGC_ReapsCommandContainers(t *testing.T) {
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	cm := newMockCM()
	// No networks to reap; focus on command containers.
	cm.listNetworksFn = func() ([]model.NetworkInfo, error) { return nil, nil }
	cm.listContainerFn = func() ([]port.ContainerInfo, error) {
		return []port.ContainerInfo{
			{ID: "cmd-old-exited", Labels: map[string]string{model.LabelRole: model.RoleEnvCommand}, CreatedAt: now.Add(-2 * time.Hour)},
			{ID: "cmd-recent-exited", Labels: map[string]string{model.LabelRole: model.RoleEnvCommand}, CreatedAt: now.Add(-1 * time.Minute)},
			{ID: "cmd-old-running", Labels: map[string]string{model.LabelRole: model.RoleEnvCommand}, CreatedAt: now.Add(-2 * time.Hour)},
		}, nil
	}
	cm.listRunningFn = func() ([]port.ContainerInfo, error) {
		return []port.ContainerInfo{
			{ID: "cmd-old-running", Labels: map[string]string{model.LabelRole: model.RoleEnvCommand}},
		}, nil
	}
	s := newTestSidecarManager(cm)
	s.now = func() time.Time { return now }

	if err := s.GC(context.Background(), 30*time.Minute); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Only the old + exited command container is reaped.
	if len(cm.removedIDs) != 1 || cm.removedIDs[0] != "cmd-old-exited" {
		t.Errorf("expected only 'cmd-old-exited' removed, got %v", cm.removedIDs)
	}
}

func TestSidecarListOrphanNetworks(t *testing.T) {
	cm := newMockCM()
	cm.listNetworksFn = func() ([]model.NetworkInfo, error) {
		return []model.NetworkInfo{
			{ID: "n-orphan", Labels: map[string]string{labelRunID: "r1"}},
			{ID: "n-live", Labels: map[string]string{labelRunID: "r2"}},
			{ID: "n-nolabel", Labels: map[string]string{}}, // not a run network
		}, nil
	}
	cm.listRunningFn = func() ([]port.ContainerInfo, error) {
		return []port.ContainerInfo{{ID: "c", Labels: map[string]string{labelRunID: "r2"}}}, nil
	}
	s := newTestSidecarManager(cm)

	orphans, err := s.ListOrphanNetworks(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(orphans) != 1 || orphans[0].ID != "n-orphan" {
		t.Errorf("expected only n-orphan, got %v", orphans)
	}
}

func TestServicePort(t *testing.T) {
	// Ports are owned by the domain (single source of truth); the docker package
	// no longer wraps them. Assert against model.ServicePort directly.
	cases := map[string]int{
		"postgres": 5432,
		"redis":    6379,
		"mysql":    3306,
		"":         0, // unknown
	}
	for svcType, want := range cases {
		if got := model.ServicePort(svcType); got != want {
			t.Errorf("model.ServicePort(%q) = %d, want %d", svcType, got, want)
		}
	}
}

func TestDetectServiceType(t *testing.T) {
	cases := map[string]string{
		"postgres:16":                "postgres",
		"docker.io/library/postgres": "postgres",
		"redis:7-alpine":             "redis",
		"mysql:8":                    "mysql",
		"ghcr.io/acme/custom:latest": "",
		"registry:5000/mariadb:11":   "mariadb",
		"mongo@sha256:abc":           "mongo",
	}
	for image, want := range cases {
		if got := detectServiceType(image); got != want {
			t.Errorf("detectServiceType(%q) = %q, want %q", image, got, want)
		}
	}
}
