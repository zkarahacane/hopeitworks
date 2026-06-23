package docker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
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
	ContainerList(ctx context.Context, options dockercontainer.ListOptions) ([]dockercontainer.Summary, error)
	ContainerInspect(ctx context.Context, containerID string) (dockercontainer.InspectResponse, error)
	NetworkCreate(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error)
	NetworkRemove(ctx context.Context, networkID string) error
	NetworkConnect(ctx context.Context, networkID, containerID string, config *network.EndpointSettings) error
	NetworkList(ctx context.Context, options network.ListOptions) ([]network.Summary, error)
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
	if opts.Healthcheck != nil {
		config.Healthcheck = &dockercontainer.HealthConfig{
			Test:        opts.Healthcheck.Test,
			Interval:    opts.Healthcheck.Interval,
			Timeout:     opts.Healthcheck.Timeout,
			Retries:     opts.Healthcheck.Retries,
			StartPeriod: opts.Healthcheck.StartPeriod,
		}
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

	// Attach to any additional networks. Empty/nil ExtraNetworks leaves the
	// current single-network behaviour untouched.
	for _, netName := range opts.ExtraNetworks {
		var aliases []string
		if alias, ok := opts.Aliases[netName]; ok && alias != "" {
			aliases = []string{alias}
		}
		if err := m.ConnectContainer(ctx, netName, resp.ID, aliases); err != nil {
			return "", err
		}
	}

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

// ListContainers lists all containers matching the specified labels.
func (m *ContainerManager) ListContainers(ctx context.Context, labels map[string]string) ([]port.ContainerInfo, error) {
	filterArgs := filters.NewArgs()
	for key, value := range labels {
		filterArgs.Add("label", key+"="+value)
	}

	containers, err := m.client.ContainerList(ctx, dockercontainer.ListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return nil, apperrors.NewContainerError(
			fmt.Sprintf("failed to list containers: %v", err),
			err,
		)
	}

	result := make([]port.ContainerInfo, 0, len(containers))
	for _, c := range containers {
		result = append(result, port.ContainerInfo{
			ID:        c.ID,
			Labels:    c.Labels,
			CreatedAt: time.Unix(c.Created, 0),
		})
	}

	m.logger.Debug("containers listed",
		slog.Int("count", len(result)),
	)

	return result, nil
}

// CreateNetwork creates a Docker network and returns its ID. It is idempotent:
// if a network with the same name already exists, its ID is returned instead of
// erroring.
func (m *ContainerManager) CreateNetwork(ctx context.Context, name string, labels map[string]string) (string, error) {
	// Idempotency: return the existing network's ID if one already has this name.
	nameFilter := filters.NewArgs()
	nameFilter.Add("name", name)
	existing, err := m.client.NetworkList(ctx, network.ListOptions{Filters: nameFilter})
	if err != nil {
		return "", apperrors.NewContainerError(
			fmt.Sprintf("failed to list networks for %s: %v", name, err),
			err,
		)
	}
	for _, n := range existing {
		// The name filter matches substrings; require an exact name match.
		if n.Name == name {
			m.logger.Debug("network already exists",
				slog.String("network", name),
				slog.String("network_id", n.ID),
			)
			return n.ID, nil
		}
	}

	resp, err := m.client.NetworkCreate(ctx, name, network.CreateOptions{
		Driver: "bridge",
		Labels: labels,
	})
	if err != nil {
		return "", apperrors.NewContainerError(
			fmt.Sprintf("failed to create network %s: %v", name, err),
			err,
		)
	}

	m.logger.Debug("network created",
		slog.String("network", name),
		slog.String("network_id", resp.ID),
	)
	return resp.ID, nil
}

// RemoveNetwork removes a Docker network by name or ID. It is idempotent: a
// network that does not exist is treated as success.
func (m *ContainerManager) RemoveNetwork(ctx context.Context, nameOrID string) error {
	if err := m.client.NetworkRemove(ctx, nameOrID); err != nil {
		if client.IsErrNotFound(err) {
			m.logger.Debug("network already absent", slog.String("network", nameOrID))
			return nil
		}
		return apperrors.NewContainerError(
			fmt.Sprintf("failed to remove network %s: %v", nameOrID, err),
			err,
		)
	}

	m.logger.Debug("network removed", slog.String("network", nameOrID))
	return nil
}

// ConnectContainer attaches a container to a network, optionally registering DNS
// aliases for it on that network.
func (m *ContainerManager) ConnectContainer(ctx context.Context, networkNameOrID, containerID string, aliases []string) error {
	var cfg *network.EndpointSettings
	if len(aliases) > 0 {
		cfg = &network.EndpointSettings{Aliases: aliases}
	}
	if err := m.client.NetworkConnect(ctx, networkNameOrID, containerID, cfg); err != nil {
		return apperrors.NewContainerError(
			fmt.Sprintf("failed to connect container %s to network %s: %v", containerID, networkNameOrID, err),
			err,
		)
	}

	m.logger.Debug("container connected to network",
		slog.String("container_id", containerID),
		slog.String("network", networkNameOrID),
	)
	return nil
}

// ListNetworks lists Docker networks matching the given label filter.
func (m *ContainerManager) ListNetworks(ctx context.Context, labelFilter map[string]string) ([]model.NetworkInfo, error) {
	filterArgs := filters.NewArgs()
	for key, value := range labelFilter {
		filterArgs.Add("label", key+"="+value)
	}

	networks, err := m.client.NetworkList(ctx, network.ListOptions{Filters: filterArgs})
	if err != nil {
		return nil, apperrors.NewContainerError(
			fmt.Sprintf("failed to list networks: %v", err),
			err,
		)
	}

	result := make([]model.NetworkInfo, 0, len(networks))
	for _, n := range networks {
		result = append(result, model.NetworkInfo{
			ID:        n.ID,
			Name:      n.Name,
			Labels:    n.Labels,
			CreatedAt: n.Created,
		})
	}

	m.logger.Debug("networks listed", slog.Int("count", len(result)))
	return result, nil
}
