package river

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

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
func NewJobQueue(pool *pgxpool.Pool, workers *river.Workers) (*JobQueue, error) {
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
func (q *JobQueue) EnqueueExecuteRun(ctx context.Context, runID uuid.UUID) error {
	_, err := q.client.Insert(ctx, ExecuteRunArgs{RunID: runID}, nil)
	if err != nil {
		return fmt.Errorf("enqueue execute_run job: %w", err)
	}
	return nil
}
