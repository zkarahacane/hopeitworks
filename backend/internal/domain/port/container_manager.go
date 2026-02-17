package port

import (
	"context"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

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
