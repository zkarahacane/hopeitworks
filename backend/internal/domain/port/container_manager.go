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
