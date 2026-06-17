package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// RunRepository defines the interface for run and run step persistence operations.
type RunRepository interface {
	CreateRun(ctx context.Context, run *model.Run) (*model.Run, error)
	GetRun(ctx context.Context, id uuid.UUID) (*model.Run, error)
	// GetActiveRunByStory returns the most recent pending or running run for a story, or nil if none.
	GetActiveRunByStory(ctx context.Context, storyID uuid.UUID) (*model.Run, error)
	// GetLatestRunByStory returns the most recent run for a story (any status) as a
	// lightweight projection with its current in-progress step, or nil if the story
	// has no runs.
	GetLatestRunByStory(ctx context.Context, storyID uuid.UUID) (*model.LatestRun, error)
	// GetLatestRunsByStories batch-fetches the most recent run per story for the given
	// story IDs. Returns a map keyed by story ID; stories with no run are absent.
	GetLatestRunsByStories(ctx context.Context, storyIDs []uuid.UUID) (map[uuid.UUID]*model.LatestRun, error)
	// GetDAGNodeRunInfoByStories batch-fetches latest-run enrichment (run id/status,
	// container id, total cost) per story for the given story IDs. Returns a map keyed
	// by story ID; stories with no run are absent.
	GetDAGNodeRunInfoByStories(ctx context.Context, storyIDs []uuid.UUID) (map[uuid.UUID]model.DAGNodeRunInfo, error)
	ListRunsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Run, error)
	ListRunsByStory(ctx context.Context, storyID uuid.UUID, limit, offset int32) ([]*model.Run, error)
	UpdateRunStatus(ctx context.Context, id uuid.UUID, status model.RunStatus, startedAt, completedAt, pausedAt *time.Time, errorMsg *string) (*model.Run, error)
	// UpdateRunMetadata persists the run's metadata map to the DB.
	// Used to survive HITL suspend/resume by flushing in-memory mutations after each step.
	UpdateRunMetadata(ctx context.Context, runID uuid.UUID, metadata map[string]interface{}) error
	CountRunsByProject(ctx context.Context, projectID uuid.UUID) (int64, error)
	CountRunsByStory(ctx context.Context, storyID uuid.UUID) (int64, error)

	CreateRunStep(ctx context.Context, step *model.RunStep) (*model.RunStep, error)
	GetRunStep(ctx context.Context, id uuid.UUID) (*model.RunStep, error)
	ListRunStepsByRun(ctx context.Context, runID uuid.UUID) ([]*model.RunStep, error)
	UpdateRunStepStatus(ctx context.Context, id uuid.UUID, status model.StepStatus, startedAt, completedAt *time.Time, errorMsg *string) (*model.RunStep, error)

	// UpdateRunStepContainerInfo updates container_id and/or log_tail on a run step
	// without changing its status. Nil values are ignored (existing values preserved).
	UpdateRunStepContainerInfo(ctx context.Context, id uuid.UUID, containerID *string, logTail *string) (*model.RunStep, error)

	// CreateRetryRunStep persists a new retry run step with retry metadata.
	CreateRetryRunStep(ctx context.Context, step *model.RunStep) (*model.RunStep, error)

	// ListRetryStepsByParent returns all retry steps for a given parent step, ordered by retry_count asc.
	ListRetryStepsByParent(ctx context.Context, parentStepID uuid.UUID) ([]*model.RunStep, error)

	// AppendStepLogTail atomically appends text to a run step's log_tail, keeping
	// at most 16 000 characters (right-truncated at the DB level).
	AppendStepLogTail(ctx context.Context, stepID uuid.UUID, text string) error
}
