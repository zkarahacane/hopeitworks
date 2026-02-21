package port

import (
	"context"

	"github.com/google/uuid"
)

// JobQueue defines the interface for enqueuing async background jobs.
type JobQueue interface {
	// EnqueueExecuteRun enqueues a job to execute a pipeline run asynchronously.
	EnqueueExecuteRun(ctx context.Context, runID uuid.UUID) error
	// EnqueueResumeRun enqueues a job to resume a suspended pipeline run from a specific step.
	EnqueueResumeRun(ctx context.Context, runID, stepID uuid.UUID) error
}
