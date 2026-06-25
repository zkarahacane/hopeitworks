//nolint:goconst // Test file with many repeated test IDs and error codes
package docker

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

const (
	testContainerID = "testContainerID"
	testErrCode     = "CONTAINER_OPERATION_FAILED"
)

// mockDockerClient is a test double for the Docker SDK client.
type mockDockerClient struct {
	// Captured arguments from the last call.
	createConfig     *dockercontainer.Config
	createHostConfig *dockercontainer.HostConfig
	createNetConfig  *network.NetworkingConfig
	startID          string
	stopID           string
	stopOpts         dockercontainer.StopOptions
	removeID         string
	removeOpts       dockercontainer.RemoveOptions
	waitID           string
	waitCondition    dockercontainer.WaitCondition

	// List captured args.
	listOpts dockercontainer.ListOptions

	// Network captured args.
	netCreateName    string
	netCreateOpts    network.CreateOptions
	netRemoveID      string
	netConnectNet    string
	netConnectCtr    string
	netConnectCfg    *network.EndpointSettings
	netDisconnectNet string
	netDisconnectCtr string
	netDisconnectFrc bool

	// Configurable return values.
	createResp     dockercontainer.CreateResponse
	createErr      error
	startErr       error
	stopErr        error
	removeErr      error
	waitStatus     dockercontainer.WaitResponse
	waitErr        error
	listContainers []dockercontainer.Summary
	listErr        error

	netCreateResp    network.CreateResponse
	netCreateErr     error
	netRemoveErr     error
	netConnectErr    error
	netDisconnectErr error
	netList          []network.Summary
	netListErr       error
	inspectResp      dockercontainer.InspectResponse
	inspectErr       error
}

func (m *mockDockerClient) ContainerCreate(
	_ context.Context,
	config *dockercontainer.Config,
	hostConfig *dockercontainer.HostConfig,
	networkingConfig *network.NetworkingConfig,
	_ *ocispec.Platform,
	_ string,
) (dockercontainer.CreateResponse, error) {
	m.createConfig = config
	m.createHostConfig = hostConfig
	m.createNetConfig = networkingConfig
	return m.createResp, m.createErr
}

func (m *mockDockerClient) ContainerStart(_ context.Context, containerID string, _ dockercontainer.StartOptions) error {
	m.startID = containerID
	return m.startErr
}

func (m *mockDockerClient) ContainerStop(_ context.Context, containerID string, opts dockercontainer.StopOptions) error {
	m.stopID = containerID
	m.stopOpts = opts
	return m.stopErr
}

func (m *mockDockerClient) ContainerRemove(_ context.Context, containerID string, opts dockercontainer.RemoveOptions) error {
	m.removeID = containerID
	m.removeOpts = opts
	return m.removeErr
}

func (m *mockDockerClient) ContainerWait(_ context.Context, containerID string, condition dockercontainer.WaitCondition) (<-chan dockercontainer.WaitResponse, <-chan error) {
	m.waitID = containerID
	m.waitCondition = condition

	statusCh := make(chan dockercontainer.WaitResponse, 1)
	errCh := make(chan error, 1)

	if m.waitErr != nil {
		errCh <- m.waitErr
	} else {
		statusCh <- m.waitStatus
	}

	return statusCh, errCh
}

func (m *mockDockerClient) ContainerList(_ context.Context, opts dockercontainer.ListOptions) ([]dockercontainer.Summary, error) {
	m.listOpts = opts
	return m.listContainers, m.listErr
}

func (m *mockDockerClient) ContainerInspect(_ context.Context, _ string) (dockercontainer.InspectResponse, error) {
	return m.inspectResp, m.inspectErr
}

func (m *mockDockerClient) NetworkCreate(_ context.Context, name string, options network.CreateOptions) (network.CreateResponse, error) {
	m.netCreateName = name
	m.netCreateOpts = options
	return m.netCreateResp, m.netCreateErr
}

func (m *mockDockerClient) NetworkRemove(_ context.Context, networkID string) error {
	m.netRemoveID = networkID
	return m.netRemoveErr
}

func (m *mockDockerClient) NetworkConnect(_ context.Context, networkID, containerID string, config *network.EndpointSettings) error {
	m.netConnectNet = networkID
	m.netConnectCtr = containerID
	m.netConnectCfg = config
	return m.netConnectErr
}

func (m *mockDockerClient) NetworkDisconnect(_ context.Context, networkID, containerID string, force bool) error {
	m.netDisconnectNet = networkID
	m.netDisconnectCtr = containerID
	m.netDisconnectFrc = force
	return m.netDisconnectErr
}

func (m *mockDockerClient) NetworkList(_ context.Context, _ network.ListOptions) ([]network.Summary, error) {
	return m.netList, m.netListErr
}

func newTestManager(mock *mockDockerClient) *ContainerManager {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return &ContainerManager{client: mock, logger: logger}
}

func TestCreate_Success(t *testing.T) {
	mock := &mockDockerClient{
		createResp: dockercontainer.CreateResponse{ID: testContainerID},
	}
	mgr := newTestManager(mock)

	opts := model.ContainerOpts{
		Image:       "managedByLabel/agent:latest",
		Env:         []string{"API_KEY=secret", "MODE=dev"},
		NetworkName: "agent-network",
		Labels:      map[string]string{"run_id": "r1", "step_id": "s1"},
		Memory:      512 * 1024 * 1024, // 512MB
		CPUs:        1.5,
	}

	id, err := mgr.Create(context.Background(), opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != testContainerID {
		t.Fatalf("expected container ID %s, got %s", testContainerID, id)
	}

	// Verify container config.
	if mock.createConfig.Image != "managedByLabel/agent:latest" {
		t.Errorf("expected image managedByLabel/agent:latest, got %s", mock.createConfig.Image)
	}
	if len(mock.createConfig.Env) != 2 {
		t.Errorf("expected 2 env vars, got %d", len(mock.createConfig.Env))
	}

	// Verify managed_by label is always set.
	if mock.createConfig.Labels["managed_by"] != managedByLabel {
		t.Errorf("expected managed_by=%s label, got %s", managedByLabel, mock.createConfig.Labels["managed_by"])
	}
	if mock.createConfig.Labels["run_id"] != "r1" {
		t.Errorf("expected run_id=r1 label, got %s", mock.createConfig.Labels["run_id"])
	}
}

func TestCreate_SecurityConstraints(t *testing.T) {
	mock := &mockDockerClient{
		createResp: dockercontainer.CreateResponse{ID: "sec123"},
	}
	mgr := newTestManager(mock)

	opts := model.ContainerOpts{
		Image:       "alpine:latest",
		NetworkName: "agent-network",
	}

	_, err := mgr.Create(context.Background(), opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify no privileged mode.
	if mock.createHostConfig.Privileged {
		t.Error("expected Privileged=false, got true")
	}

	// Verify no host mounts / binds.
	if mock.createHostConfig.Binds != nil {
		t.Error("expected Binds=nil, got non-nil")
	}
}

func TestCreate_ManagedByLabelAddedWhenLabelsNil(t *testing.T) {
	mock := &mockDockerClient{
		createResp: dockercontainer.CreateResponse{ID: "lbl123"},
	}
	mgr := newTestManager(mock)

	opts := model.ContainerOpts{
		Image:  "alpine:latest",
		Labels: nil, // nil labels
	}

	_, err := mgr.Create(context.Background(), opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if mock.createConfig.Labels["managed_by"] != managedByLabel {
		t.Errorf("expected managed_by=%s label when Labels is nil, got %s", managedByLabel, mock.createConfig.Labels["managed_by"])
	}
}

func TestCreate_MemoryAndCPULimits(t *testing.T) {
	mock := &mockDockerClient{
		createResp: dockercontainer.CreateResponse{ID: "res123"},
	}
	mgr := newTestManager(mock)

	opts := model.ContainerOpts{
		Image:  "alpine:latest",
		Memory: 1024 * 1024 * 1024, // 1GB
		CPUs:   2.0,
	}

	_, err := mgr.Create(context.Background(), opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if mock.createHostConfig.Resources.Memory != 1024*1024*1024 {
		t.Errorf("expected Memory=1073741824, got %d", mock.createHostConfig.Resources.Memory)
	}
	if mock.createHostConfig.Resources.NanoCPUs != 2_000_000_000 {
		t.Errorf("expected NanoCPUs=2000000000, got %d", mock.createHostConfig.Resources.NanoCPUs)
	}
}

func TestCreate_ZeroLimitsNotApplied(t *testing.T) {
	mock := &mockDockerClient{
		createResp: dockercontainer.CreateResponse{ID: "nolim"},
	}
	mgr := newTestManager(mock)

	opts := model.ContainerOpts{
		Image:  "alpine:latest",
		Memory: 0,
		CPUs:   0,
	}

	_, err := mgr.Create(context.Background(), opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if mock.createHostConfig.Resources.Memory != 0 {
		t.Errorf("expected Memory=0 for unlimited, got %d", mock.createHostConfig.Resources.Memory)
	}
	if mock.createHostConfig.Resources.NanoCPUs != 0 {
		t.Errorf("expected NanoCPUs=0 for unlimited, got %d", mock.createHostConfig.Resources.NanoCPUs)
	}
}

func TestCreate_NetworkConfig(t *testing.T) {
	mock := &mockDockerClient{
		createResp: dockercontainer.CreateResponse{ID: "net123"},
	}
	mgr := newTestManager(mock)

	opts := model.ContainerOpts{
		Image:       "alpine:latest",
		NetworkName: "my-network",
	}

	_, err := mgr.Create(context.Background(), opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if mock.createNetConfig == nil {
		t.Fatal("expected networkingConfig to be set")
	}
	if _, ok := mock.createNetConfig.EndpointsConfig["my-network"]; !ok {
		t.Error("expected EndpointsConfig to include my-network")
	}
}

func TestCreate_NoNetworkWhenEmpty(t *testing.T) {
	mock := &mockDockerClient{
		createResp: dockercontainer.CreateResponse{ID: "nonet"},
	}
	mgr := newTestManager(mock)

	opts := model.ContainerOpts{
		Image:       "alpine:latest",
		NetworkName: "",
	}

	_, err := mgr.Create(context.Background(), opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if mock.createNetConfig != nil {
		t.Error("expected nil networkingConfig when NetworkName is empty")
	}
}

func TestCreate_Error(t *testing.T) {
	mock := &mockDockerClient{
		createErr: errors.New("image not found"),
	}
	mgr := newTestManager(mock)

	opts := model.ContainerOpts{Image: "nonexistent:latest"}
	_, err := mgr.Create(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var domainErr *apperrors.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != testErrCode {
		t.Errorf("expected error code %s, got %s", testErrCode, domainErr.Code)
	}
}

func TestStart_Success(t *testing.T) {
	mock := &mockDockerClient{}
	mgr := newTestManager(mock)

	err := mgr.Start(context.Background(), testContainerID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.startID != testContainerID {
		t.Errorf("expected startID=%s, got %s", testContainerID, mock.startID)
	}
}

func TestStart_Error(t *testing.T) {
	mock := &mockDockerClient{
		startErr: errors.New("container not found"),
	}
	mgr := newTestManager(mock)

	err := mgr.Start(context.Background(), "bad")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var domainErr *apperrors.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != testErrCode {
		t.Errorf("expected error code %s, got %s", testErrCode, domainErr.Code)
	}
}

func TestStop_Success(t *testing.T) {
	mock := &mockDockerClient{}
	mgr := newTestManager(mock)

	err := mgr.Stop(context.Background(), testContainerID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.stopID != testContainerID {
		t.Errorf("expected stopID=%s, got %s", testContainerID, mock.stopID)
	}

	// Verify 10-second timeout is set.
	if mock.stopOpts.Timeout == nil {
		t.Fatal("expected timeout to be set")
	}
	if *mock.stopOpts.Timeout != 10 {
		t.Errorf("expected timeout=10, got %d", *mock.stopOpts.Timeout)
	}
}

func TestStop_Error(t *testing.T) {
	mock := &mockDockerClient{
		stopErr: errors.New("timeout exceeded"),
	}
	mgr := newTestManager(mock)

	err := mgr.Stop(context.Background(), testContainerID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var domainErr *apperrors.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
}

func TestRemove_Success(t *testing.T) {
	mock := &mockDockerClient{}
	mgr := newTestManager(mock)

	err := mgr.Remove(context.Background(), testContainerID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.removeID != testContainerID {
		t.Errorf("expected removeID=%s, got %s", testContainerID, mock.removeID)
	}

	// Verify force removal with volumes.
	if !mock.removeOpts.Force {
		t.Error("expected Force=true")
	}
	if !mock.removeOpts.RemoveVolumes {
		t.Error("expected RemoveVolumes=true")
	}
}

func TestRemove_Error(t *testing.T) {
	mock := &mockDockerClient{
		removeErr: errors.New("container in use"),
	}
	mgr := newTestManager(mock)

	err := mgr.Remove(context.Background(), testContainerID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var domainErr *apperrors.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
}

func TestWait_SuccessExitZero(t *testing.T) {
	mock := &mockDockerClient{
		waitStatus: dockercontainer.WaitResponse{StatusCode: 0},
	}
	mgr := newTestManager(mock)

	exitCode, err := mgr.Wait(context.Background(), testContainerID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if mock.waitID != testContainerID {
		t.Errorf("expected waitID=%s, got %s", testContainerID, mock.waitID)
	}
	if mock.waitCondition != dockercontainer.WaitConditionNotRunning {
		t.Errorf("expected WaitConditionNotRunning, got %v", mock.waitCondition)
	}
}

func TestWait_SuccessNonZeroExit(t *testing.T) {
	mock := &mockDockerClient{
		waitStatus: dockercontainer.WaitResponse{StatusCode: 1},
	}
	mgr := newTestManager(mock)

	exitCode, err := mgr.Wait(context.Background(), testContainerID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestWait_Error(t *testing.T) {
	mock := &mockDockerClient{
		waitErr: errors.New("connection lost"),
	}
	mgr := newTestManager(mock)

	_, err := mgr.Wait(context.Background(), testContainerID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var domainErr *apperrors.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != testErrCode {
		t.Errorf("expected error code %s, got %s", testErrCode, domainErr.Code)
	}
}

func TestWait_ContextCancelled(t *testing.T) {
	// Create a mock that sends nothing (both channels empty and blocking).
	mock := &mockDockerClient{}
	// Override the mock's ContainerWait to return blocking channels.
	blockingMock := &blockingWaitMock{mockDockerClient: mock}
	mgr := &ContainerManager{
		client: blockingMock,
		logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := mgr.Wait(ctx, testContainerID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var domainErr *apperrors.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
}

// blockingWaitMock wraps mockDockerClient but returns empty (blocking) channels for Wait.
type blockingWaitMock struct {
	*mockDockerClient
}

func (m *blockingWaitMock) ContainerWait(_ context.Context, containerID string, condition dockercontainer.WaitCondition) (<-chan dockercontainer.WaitResponse, <-chan error) {
	m.waitID = containerID
	m.waitCondition = condition
	return make(chan dockercontainer.WaitResponse), make(chan error)
}

func TestListContainers_Success(t *testing.T) {
	mock := &mockDockerClient{
		listContainers: []dockercontainer.Summary{
			{
				ID:      "container-1",
				Labels:  map[string]string{"managed_by": "hopeitworks", "run_id": "r1"},
				Created: 1700000000,
			},
			{
				ID:      "container-2",
				Labels:  map[string]string{"managed_by": "hopeitworks", "run_id": "r2"},
				Created: 1700001000,
			},
		},
	}
	mgr := newTestManager(mock)

	result, err := mgr.ListContainers(context.Background(), map[string]string{"managed_by": "hopeitworks"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(result))
	}
	if result[0].ID != "container-1" {
		t.Errorf("expected ID container-1, got %s", result[0].ID)
	}
	if result[0].Labels["run_id"] != "r1" {
		t.Errorf("expected run_id=r1, got %s", result[0].Labels["run_id"])
	}
	if result[1].ID != "container-2" {
		t.Errorf("expected ID container-2, got %s", result[1].ID)
	}

	// Verify All=true is set.
	if !mock.listOpts.All {
		t.Error("expected ListOptions.All=true")
	}
}

func TestListContainers_Empty(t *testing.T) {
	mock := &mockDockerClient{
		listContainers: []dockercontainer.Summary{},
	}
	mgr := newTestManager(mock)

	result, err := mgr.ListContainers(context.Background(), map[string]string{"managed_by": "hopeitworks"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 containers, got %d", len(result))
	}
}

func TestListContainers_Error(t *testing.T) {
	mock := &mockDockerClient{
		listErr: errors.New("docker daemon unreachable"),
	}
	mgr := newTestManager(mock)

	_, err := mgr.ListContainers(context.Background(), map[string]string{"managed_by": "hopeitworks"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var domainErr *apperrors.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != testErrCode {
		t.Errorf("expected error code %s, got %s", testErrCode, domainErr.Code)
	}
}

// --- Network method tests (P2c2a) ---

func TestCreateNetwork_New(t *testing.T) {
	mock := &mockDockerClient{
		netCreateResp: network.CreateResponse{ID: "net-new"},
	}
	mgr := newTestManager(mock)

	id, err := mgr.CreateNetwork(context.Background(), "hopeitworks-run-1", map[string]string{"managed_by": "hopeitworks"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != "net-new" {
		t.Errorf("expected net-new, got %s", id)
	}
	if mock.netCreateName != "hopeitworks-run-1" {
		t.Errorf("expected create name hopeitworks-run-1, got %s", mock.netCreateName)
	}
	if mock.netCreateOpts.Labels["managed_by"] != "hopeitworks" {
		t.Errorf("expected managed_by label propagated, got %v", mock.netCreateOpts.Labels)
	}
}

func TestCreateNetwork_IdempotentExisting(t *testing.T) {
	// A network with the exact name already exists: return its ID, no create.
	mock := &mockDockerClient{
		netList: []network.Summary{
			{ID: "net-existing", Name: "hopeitworks-run-1"},
			{ID: "net-other", Name: "hopeitworks-run-1-suffix"},
		},
		// Make NetworkCreate fail loudly so the test proves it is NOT called.
		netCreateErr: errors.New("create should not be called"),
	}
	mgr := newTestManager(mock)

	id, err := mgr.CreateNetwork(context.Background(), "hopeitworks-run-1", nil)
	if err != nil {
		t.Fatalf("expected no error for existing network, got %v", err)
	}
	if id != "net-existing" {
		t.Errorf("expected existing ID net-existing, got %s", id)
	}
	if mock.netCreateName != "" {
		t.Errorf("expected NetworkCreate not called, but was called with %s", mock.netCreateName)
	}
}

func TestCreateNetwork_Error(t *testing.T) {
	mock := &mockDockerClient{
		netCreateErr: errors.New("daemon error"),
	}
	mgr := newTestManager(mock)

	_, err := mgr.CreateNetwork(context.Background(), "n", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var domainErr *apperrors.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
}

func TestRemoveNetwork_Success(t *testing.T) {
	mock := &mockDockerClient{}
	mgr := newTestManager(mock)

	if err := mgr.RemoveNetwork(context.Background(), "net-1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.netRemoveID != "net-1" {
		t.Errorf("expected removeID net-1, got %s", mock.netRemoveID)
	}
}

func TestRemoveNetwork_IdempotentNotFound(t *testing.T) {
	// Absent network -> NetworkRemove returns a NotFound error -> treated as nil.
	mock := &mockDockerClient{
		netRemoveErr: errdefs.NotFound(errors.New("network n missing")),
	}
	mgr := newTestManager(mock)

	if err := mgr.RemoveNetwork(context.Background(), "n"); err != nil {
		t.Fatalf("expected nil for absent network (idempotent), got %v", err)
	}
}

func TestRemoveNetwork_RealError(t *testing.T) {
	mock := &mockDockerClient{
		netRemoveErr: errors.New("network in use"),
	}
	mgr := newTestManager(mock)

	err := mgr.RemoveNetwork(context.Background(), "n")
	if err == nil {
		t.Fatal("expected error for non-not-found failure, got nil")
	}
	var domainErr *apperrors.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
}

func TestConnectContainer_WithAliases(t *testing.T) {
	mock := &mockDockerClient{}
	mgr := newTestManager(mock)

	err := mgr.ConnectContainer(context.Background(), "net-1", "ctr-1", []string{"postgres"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.netConnectNet != "net-1" || mock.netConnectCtr != "ctr-1" {
		t.Errorf("expected connect net-1/ctr-1, got %s/%s", mock.netConnectNet, mock.netConnectCtr)
	}
	if mock.netConnectCfg == nil || len(mock.netConnectCfg.Aliases) != 1 || mock.netConnectCfg.Aliases[0] != "postgres" {
		t.Errorf("expected alias postgres, got %v", mock.netConnectCfg)
	}
}

func TestConnectContainer_NoAliases(t *testing.T) {
	mock := &mockDockerClient{}
	mgr := newTestManager(mock)

	if err := mgr.ConnectContainer(context.Background(), "net-1", "ctr-1", nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.netConnectCfg != nil {
		t.Errorf("expected nil endpoint config when no aliases, got %v", mock.netConnectCfg)
	}
}

func TestConnectContainer_Error(t *testing.T) {
	mock := &mockDockerClient{
		netConnectErr: errors.New("no such network"),
	}
	mgr := newTestManager(mock)

	err := mgr.ConnectContainer(context.Background(), "net-1", "ctr-1", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var domainErr *apperrors.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
}

func TestListNetworks_Success(t *testing.T) {
	created := time.Unix(1700000000, 0)
	mock := &mockDockerClient{
		netList: []network.Summary{
			{ID: "n1", Name: "hopeitworks-run-1", Labels: map[string]string{"managed_by": "hopeitworks"}, Created: created},
		},
	}
	mgr := newTestManager(mock)

	result, err := mgr.ListNetworks(context.Background(), map[string]string{"managed_by": "hopeitworks"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 network, got %d", len(result))
	}
	if result[0].ID != "n1" || result[0].Name != "hopeitworks-run-1" {
		t.Errorf("unexpected network info: %+v", result[0])
	}
	if !result[0].CreatedAt.Equal(created) {
		t.Errorf("expected CreatedAt %v, got %v", created, result[0].CreatedAt)
	}
}

func TestListNetworks_Error(t *testing.T) {
	mock := &mockDockerClient{
		netListErr: errors.New("daemon unreachable"),
	}
	mgr := newTestManager(mock)

	_, err := mgr.ListNetworks(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var domainErr *apperrors.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
}

func TestCreate_ExtraNetworksConnected(t *testing.T) {
	mock := &mockDockerClient{
		createResp: dockercontainer.CreateResponse{ID: "ctr-multi"},
	}
	mgr := newTestManager(mock)

	opts := model.ContainerOpts{
		Image:         "alpine:latest",
		NetworkName:   "primary",
		ExtraNetworks: []string{"run-net"},
		Aliases:       map[string]string{"run-net": "myalias"},
	}

	id, err := mgr.Create(context.Background(), opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != "ctr-multi" {
		t.Errorf("expected ctr-multi, got %s", id)
	}
	// Primary network still set via networkingConfig.
	if _, ok := mock.createNetConfig.EndpointsConfig["primary"]; !ok {
		t.Error("expected primary network in networkingConfig")
	}
	// Extra network connected via NetworkConnect with the alias.
	if mock.netConnectNet != "run-net" || mock.netConnectCtr != "ctr-multi" {
		t.Errorf("expected connect run-net/ctr-multi, got %s/%s", mock.netConnectNet, mock.netConnectCtr)
	}
	if mock.netConnectCfg == nil || len(mock.netConnectCfg.Aliases) != 1 || mock.netConnectCfg.Aliases[0] != "myalias" {
		t.Errorf("expected alias myalias, got %v", mock.netConnectCfg)
	}
}

func TestCreate_NoExtraNetworksUnchanged(t *testing.T) {
	mock := &mockDockerClient{
		createResp: dockercontainer.CreateResponse{ID: "ctr-plain"},
	}
	mgr := newTestManager(mock)

	opts := model.ContainerOpts{
		Image:       "alpine:latest",
		NetworkName: "primary",
	}

	if _, err := mgr.Create(context.Background(), opts); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// No extra networks -> NetworkConnect must not be called.
	if mock.netConnectNet != "" {
		t.Errorf("expected no NetworkConnect call, got network %s", mock.netConnectNet)
	}
}

func TestCreate_AtomicOnConnectFailure(t *testing.T) {
	// ConnectContainer fails after ContainerCreate succeeds: the freshly-created
	// container must be force-removed so Create stays all-or-nothing (the caller
	// never learns the id and cannot clean up).
	mock := &mockDockerClient{
		createResp:    dockercontainer.CreateResponse{ID: "ctr-leak"},
		netConnectErr: errors.New("no such network"),
	}
	mgr := newTestManager(mock)

	opts := model.ContainerOpts{
		Image:         "alpine:latest",
		ExtraNetworks: []string{"run-net"},
	}

	id, err := mgr.Create(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error from connect failure, got nil")
	}
	if id != "" {
		t.Errorf("expected empty id on failure, got %q", id)
	}
	if mock.removeID != "ctr-leak" {
		t.Errorf("expected ContainerRemove on ctr-leak, got %q", mock.removeID)
	}
	if !mock.removeOpts.Force {
		t.Error("expected force removal during create rollback")
	}
}

func TestCreate_PrimaryNetworkAlias(t *testing.T) {
	// An alias declared for the primary NetworkName is applied on that endpoint
	// at creation, no ExtraNetworks / NetworkConnect needed.
	mock := &mockDockerClient{
		createResp: dockercontainer.CreateResponse{ID: "ctr-alias"},
	}
	mgr := newTestManager(mock)

	opts := model.ContainerOpts{
		Image:       "postgres:16",
		NetworkName: "run-net",
		Aliases:     map[string]string{"run-net": "db"},
	}

	if _, err := mgr.Create(context.Background(), opts); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	ep := mock.createNetConfig.EndpointsConfig["run-net"]
	if ep == nil || len(ep.Aliases) != 1 || ep.Aliases[0] != "db" {
		t.Errorf("expected alias db on primary endpoint, got %+v", ep)
	}
	if mock.netConnectNet != "" {
		t.Errorf("expected no NetworkConnect, got %q", mock.netConnectNet)
	}
}

func TestInspectHealth(t *testing.T) {
	running := &dockercontainer.State{Running: true}
	stopped := &dockercontainer.State{Running: false}
	withHealth := func(status string) *dockercontainer.State {
		return &dockercontainer.State{Running: true, Health: &dockercontainer.Health{Status: status}}
	}

	tests := []struct {
		name    string
		resp    dockercontainer.InspectResponse
		respErr error
		want    string
		wantErr bool
	}{
		{name: "healthcheck healthy", resp: inspectWith(withHealth(model.HealthHealthy)), want: model.HealthHealthy},
		{name: "healthcheck unhealthy", resp: inspectWith(withHealth(model.HealthUnhealthy)), want: model.HealthUnhealthy},
		{name: "healthcheck starting", resp: inspectWith(withHealth(model.HealthStarting)), want: model.HealthStarting},
		{name: "no healthcheck running", resp: inspectWith(running), want: model.HealthRunning},
		{name: "no healthcheck not running", resp: inspectWith(stopped), want: model.HealthNotRunning},
		{name: "nil state", resp: inspectWith(nil), want: model.HealthNotRunning},
		{name: "inspect error", respErr: errors.New("daemon down"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDockerClient{inspectResp: tt.resp, inspectErr: tt.respErr}
			mgr := newTestManager(mock)

			got, err := mgr.InspectHealth(context.Background(), "c")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var domainErr *apperrors.DomainError
				if !errors.As(err, &domainErr) {
					t.Fatalf("expected DomainError, got %T", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got != tt.want {
				t.Errorf("InspectHealth = %q, want %q", got, tt.want)
			}
		})
	}
}

// inspectWith builds an InspectResponse carrying the given state.
func inspectWith(state *dockercontainer.State) dockercontainer.InspectResponse {
	return dockercontainer.InspectResponse{
		ContainerJSONBase: &dockercontainer.ContainerJSONBase{State: state},
	}
}
