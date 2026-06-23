package port

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// P2c2b scaffolding: the SidecarManager brings up an Environment's sidecar
// services on a per-run isolated Docker network, alongside the agent container.
// It is additive and NOT yet wired into the live agent_run flow — conn-string
// injection (P2c2c), command execution (P2c2d) and GC scheduling (P2c2f) land
// later. The invariant is portability: a SidecarContext must stay K8s-Pod
// expressible (services are sidecars-in-Pod on the run network, never DinD).

// SidecarContext is the handle returned by Launch describing the run's sidecar
// topology. It is the unit Stop/Cleanup operate on. A zero-value or empty
// context (no NetworkName / no ContainerIDs) is valid and means "nothing to do".
type SidecarContext struct {
	RunID        uuid.UUID
	NetworkName  string
	ContainerIDs map[string]string // service name -> container id
	ServiceAddrs map[string]string // service name -> hostname (DNS on the run network)
}

// SidecarManager orchestrates an Environment's sidecar services for a single
// run. All methods are nil-safe and idempotent where noted.
type SidecarManager interface {
	// Launch brings up every service declared by env on a fresh per-run network
	// and returns the resulting SidecarContext. If env is nil or declares no
	// services it is a no-op returning an empty context (no network/container is
	// created). On any error it rolls back atomically (stops/removes any started
	// container and removes the network) before returning the error.
	Launch(ctx context.Context, runID uuid.UUID, env *model.Environment) (*SidecarContext, error)

	// Stop stops the sidecar containers described by sc. Best-effort and
	// idempotent: a nil/empty context is a no-op and individual failures are
	// logged, not returned-fatal.
	Stop(ctx context.Context, sc *SidecarContext) error

	// Cleanup stops and removes the sidecar containers and the run network
	// described by sc. Best-effort and idempotent.
	Cleanup(ctx context.Context, sc *SidecarContext) error

	// ListOrphanNetworks lists managed run networks that no longer have any
	// managed sidecar container attached.
	ListOrphanNetworks(ctx context.Context) ([]model.NetworkInfo, error)

	// GC removes orphan run networks older than olderThan. Best-effort.
	GC(ctx context.Context, olderThan time.Duration) error
}
