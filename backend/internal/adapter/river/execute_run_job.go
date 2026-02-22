package river

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

// ExecuteRunArgs is the River job payload for pipeline execution.
type ExecuteRunArgs struct {
	RunID uuid.UUID `json:"run_id"`
}

// Kind returns the unique job kind identifier used by River.
func (ExecuteRunArgs) Kind() string { return "execute_run" }

// ExecuteRunWorker processes execute_run jobs by calling PipelineExecutor.
type ExecuteRunWorker struct {
	river.WorkerDefaults[ExecuteRunArgs]
	executor *service.PipelineExecutor
}

// NewExecuteRunWorker creates a new ExecuteRunWorker.
func NewExecuteRunWorker(executor *service.PipelineExecutor) *ExecuteRunWorker {
	return &ExecuteRunWorker{executor: executor}
}

// Timeout returns the maximum duration for a pipeline run job.
// Agent containers can take 30+ minutes for complex stories.
func (w *ExecuteRunWorker) Timeout(_ *river.Job[ExecuteRunArgs]) time.Duration {
	return 45 * time.Minute
}

// Work executes the pipeline run identified by the job payload.
func (w *ExecuteRunWorker) Work(ctx context.Context, job *river.Job[ExecuteRunArgs]) error {
	return w.executor.ExecuteRun(ctx, job.Args.RunID)
}
