package port

import (
	"context"
	"time"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// ContainerInfo represents metadata about a managed container.
type ContainerInfo struct {
	ID        string
	Labels    map[string]string
	CreatedAt time.Time
}

// ContainerManager abstracts Docker container lifecycle operations.
//
// As of the runtime rework this is no longer the domain's agent-execution port —
// AgentRuntime is (see agent_runtime.go). ContainerManager is being reclassified
// into an internal dependency of the Docker substrate adapter: a low-level
// container CRUD that AgentRuntime adapters build on, not an agent abstraction.
// It is kept here (not removed) because the live agent_run flow still drives it
// directly until the AgentRuntime adapter takes over in a later phase.
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

	// ListContainers lists all containers matching the specified labels.
	// labels is a map of key-value pairs for filtering (e.g., managed_by=hopeitworks).
	ListContainers(ctx context.Context, labels map[string]string) ([]ContainerInfo, error)
}
