package river

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// Ensure JobQueue implements port.JobQueue at compile time.
var _ port.JobQueue = (*JobQueue)(nil)

// JobQueue implements port.JobQueue using River backed by Postgres.
type JobQueue struct {
	client *river.Client[pgx.Tx]
}

// NewJobQueue creates a new JobQueue.
// workers must have all job types registered before calling NewClient.
// River tables are auto-migrated on creation.
func NewJobQueue(pool *pgxpool.Pool, workers *river.Workers) (*JobQueue, error) {
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return nil, fmt.Errorf("create river migrator: %w", err)
	}
	if _, err := migrator.Migrate(context.Background(), rivermigrate.DirectionUp, nil); err != nil {
		return nil, fmt.Errorf("run river migrations: %w", err)
	}

	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers: workers,
	})
	if err != nil {
		return nil, fmt.Errorf("create river client: %w", err)
	}
	return &JobQueue{client: client}, nil
}

// Client returns the underlying River client for lifecycle management (Start/Stop).
func (q *JobQueue) Client() *river.Client[pgx.Tx] {
	return q.client
}

// EnqueueExecuteRun enqueues a job to execute a pipeline run asynchronously.
// MaxAttempts is set to 1 to prevent River from auto-retrying the job if ExecuteRun returns
// an error. The terminal guard in ExecuteRun will skip re-execution of already-terminal runs,
// and the step-level retry_policy in the action layer handles step-specific retries.
func (q *JobQueue) EnqueueExecuteRun(ctx context.Context, runID uuid.UUID) error {
	_, err := q.client.Insert(ctx, ExecuteRunArgs{RunID: runID}, &river.InsertOpts{
		MaxAttempts: 1,
	})
	if err != nil {
		return fmt.Errorf("enqueue execute_run job: %w", err)
	}
	return nil
}
