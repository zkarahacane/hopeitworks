package port

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CallbackStatusStore manages status channels for waiting on agent container results.
type CallbackStatusStore interface {
	// WaitForStatus blocks until a status is set for the given step or the context is cancelled.
	// Returns the exit code and an optional error message.
	WaitForStatus(ctx context.Context, stepID uuid.UUID, timeout time.Duration) (exitCode int, errMsg string, err error)

	// SetStatus sets the final status for a step, unblocking any WaitForStatus call.
	SetStatus(ctx context.Context, stepID uuid.UUID, exitCode int, errMsg string) error
}
