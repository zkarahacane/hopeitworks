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

	// ListContainers lists all containers (any state, including exited) matching
	// the specified labels. labels is a map of key-value pairs for filtering
	// (e.g., managed_by=hopeitworks).
	ListContainers(ctx context.Context, labels map[string]string) ([]ContainerInfo, error)

	// ListRunningContainers lists only running containers matching the specified
	// labels. Used by orphan detection, where exited-but-not-removed containers
	// must NOT keep their run's network alive.
	ListRunningContainers(ctx context.Context, labels map[string]string) ([]ContainerInfo, error)

	// CreateNetwork creates a Docker network with the given name and labels and
	// returns its ID. It is idempotent: if a network with the same name already
	// exists, the existing network's ID is returned instead of erroring.
	CreateNetwork(ctx context.Context, name string, labels map[string]string) (string, error)

	// RemoveNetwork removes a Docker network by name or ID. It is idempotent: a
	// network that does not exist is treated as success (returns nil).
	RemoveNetwork(ctx context.Context, nameOrID string) error

	// ConnectContainer attaches an existing container to a network, optionally
	// registering DNS aliases for it on that network. aliases may be nil/empty.
	ConnectContainer(ctx context.Context, networkNameOrID, containerID string, aliases []string) error

	// DisconnectContainer detaches a container from a network. It is idempotent: a
	// container (or network) that is already absent / not attached is treated as
	// success (returns nil). Used by East-West run isolation to detach the API
	// container before removing a per-run network (otherwise Docker rejects the
	// removal with "network has active endpoints").
	DisconnectContainer(ctx context.Context, networkNameOrID, containerID string) error

	// ListNetworks lists managed Docker networks matching the given label
	// filter (e.g. managed_by=hopeitworks). An empty/nil filter lists all.
	ListNetworks(ctx context.Context, labelFilter map[string]string) ([]model.NetworkInfo, error)

	// InspectHealth reports a container's readiness. When the container has a
	// Docker HEALTHCHECK it returns its health status (one of model.Health*).
	// When it has none it falls back to a running/not-running signal
	// (model.HealthRunning / model.HealthNotRunning).
	InspectHealth(ctx context.Context, containerID string) (string, error)
}
