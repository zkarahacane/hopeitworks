package port

import (
	"context"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// WatchdogRepository exposes the read model the guard watchdog needs: the set of
// currently-running steps enriched with the timing/cost context required to
// evaluate log_silence, wallclock and cost_batch probes (INC 4a).
type WatchdogRepository interface {
	// ListRunningSteps returns every running step of a running run across all
	// projects, with its last log-event timestamp, start time and run context.
	ListRunningSteps(ctx context.Context) ([]*model.RunningStep, error)
}
