package river

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

// WriteBackArgs is the River job payload for a status write-back to the tracker.
type WriteBackArgs struct {
	ProjectID uuid.UUID `json:"project_id"`
	StoryID   uuid.UUID `json:"story_id"`
	RunID     uuid.UUID `json:"run_id"` // uuid.Nil => no run
	Status    string    `json:"status"`
}

// Kind returns the unique job kind identifier used by River.
func (WriteBackArgs) Kind() string { return "planning_write_back" }

// WriteBackWorker processes planning_write_back jobs by calling the write-back service.
type WriteBackWorker struct {
	river.WorkerDefaults[WriteBackArgs]
	svc *service.PlanningWriteBackService
}

// NewWriteBackWorker creates a new WriteBackWorker.
func NewWriteBackWorker(svc *service.PlanningWriteBackService) *WriteBackWorker {
	return &WriteBackWorker{svc: svc}
}

// Timeout bounds a single write-back attempt (a couple of GraphQL round-trips).
func (w *WriteBackWorker) Timeout(_ *river.Job[WriteBackArgs]) time.Duration {
	return 2 * time.Minute
}

// Work performs the write-back. A returned error triggers River's exponential
// backoff retry (used for transient GitHub failures: rate limit / 5xx).
func (w *WriteBackWorker) Work(ctx context.Context, job *river.Job[WriteBackArgs]) error {
	var runID *uuid.UUID
	if job.Args.RunID != uuid.Nil {
		r := job.Args.RunID
		runID = &r
	}
	return w.svc.SyncStatus(ctx, job.Args.StoryID, runID, job.Args.Status)
}
