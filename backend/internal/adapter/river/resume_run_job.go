package river

import (
	"context"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

// ResumeRunArgs is the River job payload for resuming a suspended pipeline run.
type ResumeRunArgs struct {
	RunID  uuid.UUID `json:"run_id"`
	StepID uuid.UUID `json:"step_id"`
}

// Kind returns the unique job kind identifier used by River.
func (ResumeRunArgs) Kind() string { return "resume_run" }

// ResumeRunWorker processes resume_run jobs by calling PipelineExecutor.
type ResumeRunWorker struct {
	river.WorkerDefaults[ResumeRunArgs]
	executor *service.PipelineExecutor
}

// NewResumeRunWorker creates a new ResumeRunWorker.
func NewResumeRunWorker(executor *service.PipelineExecutor) *ResumeRunWorker {
	return &ResumeRunWorker{executor: executor}
}

// Work resumes the pipeline run from the specified step.
func (w *ResumeRunWorker) Work(ctx context.Context, job *river.Job[ResumeRunArgs]) error {
	return w.executor.ResumeRun(ctx, job.Args.RunID, job.Args.StepID)
}
