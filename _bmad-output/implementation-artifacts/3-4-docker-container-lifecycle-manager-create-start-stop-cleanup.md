# Story 3.4: [BACK] Docker container lifecycle manager (create, start, stop, cleanup)

Status: ready-for-dev

## Story

As a backend developer, I want a container lifecycle manager for agent execution, so that the system can safely create, run, and clean up isolated Docker containers.

## Acceptance Criteria (BDD)

**AC1: ContainerManager port interface defines lifecycle operations**
- **Given** a ContainerManager port interface in `backend/internal/domain/port/container_manager.go`
- **When** the interface is reviewed
- **Then** it declares Create(ctx, opts ContainerOpts) (containerID string, err error)
- **And** it declares Start(ctx, containerID) error
- **And** it declares Stop(ctx, containerID) error
- **And** it declares Remove(ctx, containerID) error
- **And** it declares Wait(ctx, containerID) (exitCode int, err error)
- **And** all methods return domain errors with contextual information

**AC2: ContainerOpts model defines container configuration**
- **Given** a ContainerOpts struct in `backend/internal/domain/model/container.go`
- **When** the struct is reviewed
- **Then** it includes Image string field
- **And** it includes Env []string field for KEY=VALUE environment variables
- **And** it includes NetworkName string field
- **And** it includes Labels map[string]string field
- **And** it includes Memory int64 field (bytes, 0 = unlimited)
- **And** it includes CPUs float64 field (0 = unlimited)

**AC3: Docker SDK adapter implements ContainerManager via socket-proxy**
- **Given** a Docker adapter in `backend/internal/adapter/docker/container_manager.go`
- **When** Create is called with ContainerOpts
- **Then** it creates a container via Docker SDK with specified image, env, network, and labels
- **And** it applies memory and CPU limits if specified (non-zero values)
- **And** it connects to Docker via socket-proxy URL from config
- **And** it wraps errors in DomainError with container details

**AC4: Docker adapter enforces security constraints on container creation**
- **Given** a Docker adapter creating a container
- **When** Create is called
- **Then** it creates the container with no host filesystem mounts
- **And** it creates the container with no privileged mode
- **And** it creates the container attached to the specified agent network
- **And** it adds label `managed_by=hopeitworks` to all containers
- **And** it prevents any binds or volumes from being added

**AC5: Docker adapter starts created containers**
- **Given** a container created via ContainerManager
- **When** Start is called with the container ID
- **Then** it starts the container via Docker SDK
- **And** it wraps errors in DomainError with container ID context

**AC6: Docker adapter stops containers gracefully with timeout**
- **Given** a running container managed by ContainerManager
- **When** Stop is called with the container ID
- **Then** it sends SIGTERM to the container
- **And** it waits up to 10 seconds for graceful shutdown
- **And** it sends SIGKILL if container does not stop within timeout
- **And** it wraps errors in DomainError with container ID context

**AC7: Docker adapter removes stopped containers**
- **Given** a stopped container managed by ContainerManager
- **When** Remove is called with the container ID
- **Then** it removes the container via Docker SDK
- **And** it removes associated volumes (force removal)
- **And** it wraps errors in DomainError with container ID context

**AC8: Docker adapter waits for container exit and returns exit code**
- **Given** a running container managed by ContainerManager
- **When** Wait is called with the container ID
- **Then** it blocks until the container exits
- **And** it returns the container exit code
- **And** it wraps errors in DomainError with container ID context

**AC9: Unit tests verify Docker adapter behavior with mock Docker client**
- **Given** unit tests in `backend/internal/adapter/docker/container_manager_test.go`
- **When** tests are executed
- **Then** Create tests verify correct container config (image, env, network, labels, limits)
- **And** Create tests verify security constraints (no mounts, no privileged)
- **And** Start tests verify correct Docker SDK calls
- **And** Stop tests verify graceful shutdown with timeout
- **And** Remove tests verify force removal with volumes
- **And** Wait tests verify blocking until exit and return exit code
- **And** all tests use mock Docker client to avoid real Docker operations
- **And** error handling tests verify DomainError wrapping

**AC10: docker-compose adds agent network for container isolation**
- **Given** docker-compose.yml in `deploy/docker-compose.yml`
- **When** the file is reviewed
- **Then** it defines a network named `agent-network`
- **And** the network is isolated from the default network
- **And** agent containers will be attached to this network at runtime

## Tasks / Subtasks

- [ ] [BACK] Task 1: Define ContainerOpts and domain models (AC: #2)
  - [ ] Create `backend/internal/domain/model/container.go`
  - [ ] Define ContainerOpts struct with Image, Env, NetworkName, Labels, Memory, CPUs
  - [ ] Add validation helpers if needed (e.g., validate image format)
  - [ ] Document struct fields with godoc comments

- [ ] [BACK] Task 2: Define ContainerManager port interface (AC: #1)
  - [ ] Create `backend/internal/domain/port/container_manager.go`
  - [ ] Define ContainerManager interface with Create, Start, Stop, Remove, Wait methods
  - [ ] Document all interface methods with godoc comments
  - [ ] Add context.Context as first parameter for all methods

- [ ] [BACK] Task 3: Implement Docker SDK adapter for ContainerManager (AC: #3, #4)
  - [ ] Create `backend/internal/adapter/docker/container_manager.go`
  - [ ] Add DockerContainerManager struct with Docker SDK client dependency
  - [ ] Implement Create method with security constraints (no mounts, no privileged)
  - [ ] Apply container labels: managed_by=hopeitworks, run_id, step_id (from opts)
  - [ ] Connect to Docker via socket-proxy URL from config
  - [ ] Wrap all errors in DomainError with container details

- [ ] [BACK] Task 4: Implement Start, Stop, Remove, Wait methods (AC: #5, #6, #7, #8)
  - [ ] Implement Start method using Docker SDK ContainerStart
  - [ ] Implement Stop method with 10s timeout (SIGTERM → SIGKILL)
  - [ ] Implement Remove method with force=true and volumes=true
  - [ ] Implement Wait method using Docker SDK ContainerWait, return exit code
  - [ ] Wrap all errors in DomainError with container ID context

- [ ] [BACK] Task 5: Add agent network to docker-compose (AC: #10)
  - [ ] Edit `deploy/docker-compose.yml`
  - [ ] Add `agent-network` to networks section
  - [ ] Configure network isolation (bridge driver, internal: false for internet access)
  - [ ] Document network purpose in comments

- [ ] [BACK] Task 6: Create mock Docker client for unit tests (AC: #9)
  - [ ] Create mock Docker client in `container_manager_test.go`
  - [ ] Track container operations (Create, Start, Stop, Remove, Wait)
  - [ ] Support configurable return values (containerID, exit code, error)
  - [ ] Verify security constraints in Create calls

- [ ] [BACK] Task 7: Write unit tests for Docker adapter (AC: #9)
  - [ ] Test Create with valid ContainerOpts (image, env, network, labels)
  - [ ] Test Create enforces security constraints (no mounts, no privileged)
  - [ ] Test Create applies memory and CPU limits
  - [ ] Test Start, Stop, Remove, Wait with valid container IDs
  - [ ] Test Stop timeout behavior (mock graceful shutdown vs timeout)
  - [ ] Test error handling and DomainError wrapping
  - [ ] Verify container config passed to mock Docker client

- [ ] [BACK] Task 8: Add integration test with testcontainers-go (optional, bonus)
  - [ ] Create integration test file with `//go:build integration` tag
  - [ ] Use real Docker SDK client against testcontainers Docker daemon
  - [ ] Create, start, wait, stop, remove a real container (e.g., alpine:latest)
  - [ ] Verify container lifecycle and exit code
  - [ ] Clean up containers in test teardown

## Dev Notes

### Dependencies
- Story 1-1: Go project scaffolding (provides base project structure)
- docker-socket-proxy in docker-compose (already exists, verify connection)
- Docker SDK: `github.com/docker/docker/client`
- Docker API types: `github.com/docker/docker/api/types/container`

### Architecture Requirements
- **Hexagonal architecture:** ContainerManager is a port in domain/port, Docker adapter in adapter/docker
- **Testability:** Docker SDK client is injected as dependency, allowing mock client in unit tests
- **Error handling:** All adapter errors wrapped in DomainError via pkg/errors
- **Structured logging:** Use slog to log container lifecycle events (debug level)
- **Security:** No host mounts, no privileged mode, isolated network

### File Paths (exact)
```
backend/internal/domain/port/container_manager.go          # ContainerManager port interface
backend/internal/domain/model/container.go                 # ContainerOpts, ContainerInfo structs
backend/internal/adapter/docker/container_manager.go       # Docker SDK implementation
backend/internal/adapter/docker/container_manager_test.go  # Unit tests with mock Docker client
deploy/docker-compose.yml                                  # Updated with agent-network
```

### Technical Specifications

**ContainerOpts model:**
```go
package model

// ContainerOpts specifies configuration for creating a container.
type ContainerOpts struct {
    // Image is the Docker image to use (e.g., "hopeitworks/agent:latest")
    Image string

    // Env is a list of environment variables in KEY=VALUE format
    Env []string

    // NetworkName is the Docker network to attach the container to
    NetworkName string

    // Labels are key-value pairs for container metadata
    // Standard labels: managed_by, run_id, step_id
    Labels map[string]string

    // Memory is the memory limit in bytes (0 = unlimited)
    Memory int64

    // CPUs is the CPU limit as a float (0 = unlimited, 1.0 = 1 CPU)
    CPUs float64
}
```

**ContainerManager port interface:**
```go
package port

import "context"
import "hopeitworks/backend/internal/domain/model"

// ContainerManager abstracts Docker container lifecycle operations.
type ContainerManager interface {
    // Create creates a container with the specified options.
    // Returns the container ID on success.
    Create(ctx context.Context, opts model.ContainerOpts) (string, error)

    // Start starts a created container.
    Start(ctx context.Context, containerID string) error

    // Stop stops a running container gracefully (SIGTERM, 10s timeout, then SIGKILL).
    Stop(ctx context.Context, containerID string) error

    // Remove removes a stopped container and its volumes.
    Remove(ctx context.Context, containerID string) error

    // Wait blocks until the container exits and returns its exit code.
    Wait(ctx context.Context, containerID string) (int, error)
}
```

**Docker adapter implementation:**
```go
package docker

import (
    "context"
    "fmt"
    "time"

    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/client"

    "hopeitworks/backend/internal/domain/model"
    "hopeitworks/backend/internal/domain/port"
    "hopeitworks/backend/pkg/errors"
)

type DockerContainerManager struct {
    client *client.Client
}

func NewDockerContainerManager(dockerHost string) (*DockerContainerManager, error) {
    cli, err := client.NewClientWithOpts(
        client.WithHost(dockerHost),
        client.WithAPIVersionNegotiation(),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create Docker client: %w", err)
    }
    return &DockerContainerManager{client: cli}, nil
}

func (m *DockerContainerManager) Create(ctx context.Context, opts model.ContainerOpts) (string, error) {
    // Ensure managed_by label is set
    if opts.Labels == nil {
        opts.Labels = make(map[string]string)
    }
    opts.Labels["managed_by"] = "hopeitworks"

    // Build container config
    config := &container.Config{
        Image:  opts.Image,
        Env:    opts.Env,
        Labels: opts.Labels,
    }

    // Build host config with security constraints
    hostConfig := &container.HostConfig{
        // Security: no host mounts, no privileged mode
        Privileged: false,
        Binds:      nil,

        // Resource limits
        Resources: container.Resources{
            Memory:   opts.Memory,
            NanoCPUs: int64(opts.CPUs * 1e9), // Convert float to nanocpus
        },
    }

    // Network config
    networkConfig := &network.NetworkingConfig{
        EndpointsConfig: map[string]*network.EndpointSettings{
            opts.NetworkName: {},
        },
    }

    resp, err := m.client.ContainerCreate(ctx, config, hostConfig, networkConfig, nil, "")
    if err != nil {
        return "", errors.NewDomainError(
            errors.ErrCodeContainerOperationFailed,
            fmt.Sprintf("failed to create container: %v", err),
            map[string]any{"image": opts.Image, "network": opts.NetworkName},
        )
    }

    return resp.ID, nil
}

func (m *DockerContainerManager) Start(ctx context.Context, containerID string) error {
    if err := m.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
        return errors.NewDomainError(
            errors.ErrCodeContainerOperationFailed,
            fmt.Sprintf("failed to start container: %v", err),
            map[string]any{"container_id": containerID},
        )
    }
    return nil
}

func (m *DockerContainerManager) Stop(ctx context.Context, containerID string) error {
    timeout := 10 * time.Second
    if err := m.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
        return errors.NewDomainError(
            errors.ErrCodeContainerOperationFailed,
            fmt.Sprintf("failed to stop container: %v", err),
            map[string]any{"container_id": containerID},
        )
    }
    return nil
}

func (m *DockerContainerManager) Remove(ctx context.Context, containerID string) error {
    if err := m.client.ContainerRemove(ctx, containerID, container.RemoveOptions{
        Force:         true,
        RemoveVolumes: true,
    }); err != nil {
        return errors.NewDomainError(
            errors.ErrCodeContainerOperationFailed,
            fmt.Sprintf("failed to remove container: %v", err),
            map[string]any{"container_id": containerID},
        )
    }
    return nil
}

func (m *DockerContainerManager) Wait(ctx context.Context, containerID string) (int, error) {
    statusCh, errCh := m.client.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
    select {
    case err := <-errCh:
        if err != nil {
            return 0, errors.NewDomainError(
                errors.ErrCodeContainerOperationFailed,
                fmt.Sprintf("error waiting for container: %v", err),
                map[string]any{"container_id": containerID},
            )
        }
    case status := <-statusCh:
        return int(status.StatusCode), nil
    }
    return 0, nil
}
```

**Error codes to add to pkg/errors:**
```go
const (
    ErrCodeContainerOperationFailed = "CONTAINER_OPERATION_FAILED"
)
```

**docker-compose.yml agent network:**
```yaml
networks:
  agent-network:
    driver: bridge
    # internal: false allows containers to access internet for git clone, API calls
```

### Testing Requirements

**Unit tests (container_manager_test.go):**
- Mock Docker client tracks all operations (Create, Start, Stop, Remove, Wait)
- Test Create with valid ContainerOpts (all fields populated)
- Test Create enforces security: no Privileged, no Binds
- Test Create adds managed_by=hopeitworks label
- Test Create applies memory and CPU limits (verify Resources struct)
- Test Start with valid container ID
- Test Stop with timeout (mock timeout scenario vs graceful stop)
- Test Remove with force=true and volumes=true
- Test Wait returns exit code (mock successful exit vs error)
- Test error handling wraps errors in DomainError
- No actual Docker daemon required in unit tests

**Integration tests (optional, bonus):**
- Tag with `//go:build integration`
- Use real Docker SDK client
- Create container with alpine:latest, echo "test", exit 0
- Start container, Wait for exit, verify exit code = 0
- Stop and Remove container
- Verify container is removed from Docker daemon
- Clean up in test teardown

### References
- Story 1-1: Go project scaffolding
- Story 3-5: StreamLogs (Wave 6, not in this story)
- Architecture doc: `_bmad-output/planning-artifacts/architecture.md`
- Docker SDK docs: https://pkg.go.dev/github.com/docker/docker/client
- docker-socket-proxy: already in docker-compose from Story 1-1

## Dev Agent Record

(To be filled during implementation)

## Change Log

- 2026-02-17: Story created for Wave 5 backend infrastructure
