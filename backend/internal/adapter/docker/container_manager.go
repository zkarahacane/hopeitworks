package docker

import (
	"context"
	"fmt"
	"log/slog"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// dockerClient defines the subset of the Docker SDK client used by ContainerManager.
// This allows injecting a mock for unit testing.
type dockerClient interface {
	ContainerCreate(ctx context.Context, config *dockercontainer.Config, hostConfig *dockercontainer.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (dockercontainer.CreateResponse, error)
	ContainerStart(ctx context.Context, containerID string, options dockercontainer.StartOptions) error
	ContainerStop(ctx context.Context, containerID string, options dockercontainer.StopOptions) error
	ContainerRemove(ctx context.Context, containerID string, options dockercontainer.RemoveOptions) error
	ContainerWait(ctx context.Context, containerID string, condition dockercontainer.WaitCondition) (<-chan dockercontainer.WaitResponse, <-chan error)
}

// Ensure ContainerManager implements port.ContainerManager at compile time.
var _ port.ContainerManager = (*ContainerManager)(nil)

// ContainerManager implements port.ContainerManager using the Docker SDK.
type ContainerManager struct {
	client dockerClient
	logger *slog.Logger
}

// NewDockerContainerManager creates a ContainerManager that connects to Docker
// via the specified host URL (e.g., "tcp://socket-proxy:2375").
func NewDockerContainerManager(dockerHost string, logger *slog.Logger) (*ContainerManager, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(dockerHost),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}
	return &ContainerManager{client: cli, logger: logger}, nil
}

// stopTimeoutSeconds is the graceful shutdown timeout before SIGKILL.
const stopTimeoutSeconds = 10

// managedByLabel is the value for the managed_by label on containers.
const managedByLabel = "hopeitworks"

// Create creates a container with the specified options, enforcing security constraints.
func (m *ContainerManager) Create(ctx context.Context, opts model.ContainerOpts) (string, error) {
	if opts.Labels == nil {
		opts.Labels = make(map[string]string)
	}
	opts.Labels["managed_by"] = managedByLabel

	config := &dockercontainer.Config{
		Image:  opts.Image,
		Env:    opts.Env,
		Labels: opts.Labels,
	}

	resources := dockercontainer.Resources{}
	if opts.Memory > 0 {
		resources.Memory = opts.Memory
	}
	if opts.CPUs > 0 {
		resources.NanoCPUs = int64(opts.CPUs * 1e9)
	}

	hostConfig := &dockercontainer.HostConfig{
		Privileged: false,
		Binds:      nil,
		Resources:  resources,
	}

	var networkingConfig *network.NetworkingConfig
	if opts.NetworkName != "" {
		networkingConfig = &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				opts.NetworkName: {},
			},
		}
	}

	resp, err := m.client.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, "")
	if err != nil {
		return "", apperrors.NewContainerError(
			fmt.Sprintf("failed to create container with image %s: %v", opts.Image, err),
			err,
		)
	}

	m.logger.Debug("container created",
		slog.String("container_id", resp.ID),
		slog.String("image", opts.Image),
	)

	return resp.ID, nil
}

// Start starts a created container.
func (m *ContainerManager) Start(ctx context.Context, containerID string) error {
	if err := m.client.ContainerStart(ctx, containerID, dockercontainer.StartOptions{}); err != nil {
		return apperrors.NewContainerError(
			fmt.Sprintf("failed to start container %s: %v", containerID, err),
			err,
		)
	}

	m.logger.Debug("container started", slog.String("container_id", containerID))
	return nil
}

// Stop stops a running container gracefully (SIGTERM, 10s timeout, then SIGKILL).
func (m *ContainerManager) Stop(ctx context.Context, containerID string) error {
	timeout := stopTimeoutSeconds
	if err := m.client.ContainerStop(ctx, containerID, dockercontainer.StopOptions{Timeout: &timeout}); err != nil {
		return apperrors.NewContainerError(
			fmt.Sprintf("failed to stop container %s: %v", containerID, err),
			err,
		)
	}

	m.logger.Debug("container stopped", slog.String("container_id", containerID))
	return nil
}

// Remove removes a stopped container and its associated volumes.
func (m *ContainerManager) Remove(ctx context.Context, containerID string) error {
	if err := m.client.ContainerRemove(ctx, containerID, dockercontainer.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	}); err != nil {
		return apperrors.NewContainerError(
			fmt.Sprintf("failed to remove container %s: %v", containerID, err),
			err,
		)
	}

	m.logger.Debug("container removed", slog.String("container_id", containerID))
	return nil
}

// Wait blocks until the container exits and returns its exit code.
func (m *ContainerManager) Wait(ctx context.Context, containerID string) (int, error) {
	statusCh, errCh := m.client.ContainerWait(ctx, containerID, dockercontainer.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		if err != nil {
			return 0, apperrors.NewContainerError(
				fmt.Sprintf("error waiting for container %s: %v", containerID, err),
				err,
			)
		}
	case status := <-statusCh:
		m.logger.Debug("container exited",
			slog.String("container_id", containerID),
			slog.Int("exit_code", int(status.StatusCode)),
		)
		return int(status.StatusCode), nil
	case <-ctx.Done():
		return 0, apperrors.NewContainerError(
			fmt.Sprintf("context cancelled while waiting for container %s", containerID),
			ctx.Err(),
		)
	}

	return 0, nil
}
