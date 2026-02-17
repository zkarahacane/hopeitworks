package docker

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

const (
	testContainerID        = "abc123"
	testErrorCodeContainer = "CONTAINER_OPERATION_FAILED"
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

	// Configurable return values.
	createResp dockercontainer.CreateResponse
	createErr  error
	startErr   error
	stopErr    error
	removeErr  error
	waitStatus dockercontainer.WaitResponse
	waitErr    error
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

func newTestManager(mock *mockDockerClient) *ContainerManager {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return &ContainerManager{client: mock, logger: logger}
}

func TestCreate_Success(t *testing.T) {
	mock := &mockDockerClient{
		createResp: dockercontainer.CreateResponse{ID: "testContainerID"},
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
	if id != "testContainerID" {
		t.Fatalf("expected container ID testContainerID, got %s", id)
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
	if domainErr.Code != testErrorCodeContainer {
		t.Errorf("expected error code %s, got %s", testErrorCodeContainer, domainErr.Code)
	}
}

func TestStart_Success(t *testing.T) {
	mock := &mockDockerClient{}
	mgr := newTestManager(mock)

	err := mgr.Start(context.Background(), "testContainerID")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.startID != "testContainerID" {
		t.Errorf("expected startID=testContainerID, got %s", mock.startID)
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
	if domainErr.Code != testErrorCodeContainer {
		t.Errorf("expected error code %s, got %s", testErrorCodeContainer, domainErr.Code)
	}
}

func TestStop_Success(t *testing.T) {
	mock := &mockDockerClient{}
	mgr := newTestManager(mock)

	err := mgr.Stop(context.Background(), "testContainerID")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.stopID != "testContainerID" {
		t.Errorf("expected stopID=testContainerID, got %s", mock.stopID)
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

	err := mgr.Stop(context.Background(), "testContainerID")
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

	err := mgr.Remove(context.Background(), "testContainerID")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.removeID != "testContainerID" {
		t.Errorf("expected removeID=testContainerID, got %s", mock.removeID)
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

	err := mgr.Remove(context.Background(), "testContainerID")
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

	exitCode, err := mgr.Wait(context.Background(), "testContainerID")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if mock.waitID != "testContainerID" {
		t.Errorf("expected waitID=testContainerID, got %s", mock.waitID)
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

	exitCode, err := mgr.Wait(context.Background(), "testContainerID")
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

	_, err := mgr.Wait(context.Background(), "testContainerID")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var domainErr *apperrors.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != testErrorCodeContainer {
		t.Errorf("expected error code %s, got %s", testErrorCodeContainer, domainErr.Code)
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

	_, err := mgr.Wait(ctx, "testContainerID")
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
