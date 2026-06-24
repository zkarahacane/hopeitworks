package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure RunRepo implements port.RunRepository at compile time.
var _ port.RunRepository = (*RunRepo)(nil)

// RunRepo implements port.RunRepository using sqlc-generated queries.
type RunRepo struct {
	queries *Queries
}

// NewRunRepo creates a new RunRepo.
func NewRunRepo(queries *Queries) *RunRepo {
	return &RunRepo{queries: queries}
}

func (r *RunRepo) CreateRun(ctx context.Context, run *model.Run) (*model.Run, error) {
	metadataJSON := []byte("{}")
	if run.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(run.Metadata)
		if err != nil {
			return nil, apperrors.NewInternal("failed to marshal run metadata", err)
		}
	}

	params := CreateRunParams{
		ProjectID:              run.ProjectID,
		StoryID:                run.StoryID,
		Status:                 string(run.Status),
		PipelineConfigSnapshot: []byte(run.PipelineConfigSnapshot),
		Metadata:               metadataJSON,
	}

	row, err := r.queries.CreateRun(ctx, params)
	if err != nil {
		if isForeignKeyViolation(err) {
			return nil, apperrors.NewNotFound("project", run.ProjectID)
		}
		return nil, apperrors.NewInternal("failed to create run", err)
	}
	return toDomainRun(row), nil
}

func (r *RunRepo) GetRun(ctx context.Context, id uuid.UUID) (*model.Run, error) {
	row, err := r.queries.GetRunWithStoryKey(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("run", id)
		}
		return nil, apperrors.NewInternal("failed to get run", err)
	}
	return toDomainRunWithStoryKey(row.ID, row.ProjectID, row.StoryID, row.Status,
		row.PipelineConfigSnapshot, row.StartedAt, row.CompletedAt, row.ErrorMessage,
		row.CreatedAt, row.UpdatedAt, row.PausedAt, row.Metadata, row.StoryKey), nil
}

func (r *RunRepo) GetActiveRunByStory(ctx context.Context, storyID uuid.UUID) (*model.Run, error) {
	row, err := r.queries.GetActiveRunByStory(ctx, storyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, apperrors.NewInternal("failed to get active run by story", err)
	}
	return toDomainRun(row), nil
}

func (r *RunRepo) GetLatestRunByStory(ctx context.Context, storyID uuid.UUID) (*model.LatestRun, error) {
	row, err := r.queries.GetLatestRunByStory(ctx, storyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, apperrors.NewInternal("failed to get latest run by story", err)
	}
	return toDomainLatestRun(row.RunID, row.RunStatus, row.CurrentStep, row.TotalSteps)
}

func (r *RunRepo) GetLatestRunsByStories(ctx context.Context, storyIDs []uuid.UUID) (map[uuid.UUID]*model.LatestRun, error) {
	result := make(map[uuid.UUID]*model.LatestRun, len(storyIDs))
	if len(storyIDs) == 0 {
		return result, nil
	}
	rows, err := r.queries.GetLatestRunsByStories(ctx, storyIDs)
	if err != nil {
		return nil, apperrors.NewInternal("failed to get latest runs by stories", err)
	}
	for _, row := range rows {
		latest, err := toDomainLatestRun(row.RunID, row.RunStatus, row.CurrentStep, row.TotalSteps)
		if err != nil {
			return nil, err
		}
		result[row.StoryID] = latest
	}
	return result, nil
}

func (r *RunRepo) GetDAGNodeRunInfoByStories(ctx context.Context, storyIDs []uuid.UUID) (map[uuid.UUID]model.DAGNodeRunInfo, error) {
	result := make(map[uuid.UUID]model.DAGNodeRunInfo, len(storyIDs))
	if len(storyIDs) == 0 {
		return result, nil
	}
	rows, err := r.queries.GetDAGNodeRunInfoByStories(ctx, storyIDs)
	if err != nil {
		return nil, apperrors.NewInternal("failed to get dag node run info by stories", err)
	}
	for _, row := range rows {
		info := model.DAGNodeRunInfo{
			RunID:     row.RunID,
			RunStatus: row.RunStatus,
			CostUSD:   numericToFloat64(row.CostUsd),
		}
		if row.ContainerID.Valid {
			containerID := row.ContainerID.String
			info.ContainerID = &containerID
		}
		result[row.StoryID] = info
	}
	return result, nil
}

func (r *RunRepo) ListRunsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Run, error) {
	rows, err := r.queries.ListRunsByProjectWithStoryKey(ctx, ListRunsByProjectWithStoryKeyParams{
		ProjectID: projectID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list runs by project", err)
	}
	runs := make([]*model.Run, len(rows))
	for i, row := range rows {
		run := toDomainRunWithStoryKey(row.ID, row.ProjectID, row.StoryID, row.Status,
			row.PipelineConfigSnapshot, row.StartedAt, row.CompletedAt, row.ErrorMessage,
			row.CreatedAt, row.UpdatedAt, row.PausedAt, row.Metadata, row.StoryKey)
		run.CostUSD = numericToFloat64Ptr(row.CostUsd)
		runs[i] = run
	}
	return runs, nil
}

func (r *RunRepo) ListRunsByStory(ctx context.Context, storyID uuid.UUID, limit, offset int32) ([]*model.Run, error) {
	rows, err := r.queries.ListRunsByStory(ctx, ListRunsByStoryParams{
		StoryID: storyID,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list runs by story", err)
	}
	runs := make([]*model.Run, len(rows))
	for i, row := range rows {
		run := toDomainRunWithStoryKey(row.ID, row.ProjectID, row.StoryID, row.Status,
			row.PipelineConfigSnapshot, row.StartedAt, row.CompletedAt, row.ErrorMessage,
			row.CreatedAt, row.UpdatedAt, row.PausedAt, row.Metadata, "")
		run.CostUSD = numericToFloat64Ptr(row.CostUsd)
		runs[i] = run
	}
	return runs, nil
}

func (r *RunRepo) ListRunsByStatus(ctx context.Context, status model.RunStatus) ([]*model.Run, error) {
	rows, err := r.queries.ListRunsByStatus(ctx, string(status))
	if err != nil {
		return nil, apperrors.NewInternal("failed to list runs by status", err)
	}
	runs := make([]*model.Run, len(rows))
	for i, row := range rows {
		runs[i] = toDomainRun(row)
	}
	return runs, nil
}

func (r *RunRepo) MarkRunOrphanedIfRunning(ctx context.Context, id uuid.UUID, completedAt time.Time, errorMsg string) (bool, error) {
	affected, err := r.queries.MarkRunOrphanedIfRunning(ctx, MarkRunOrphanedIfRunningParams{
		ID:           id,
		CompletedAt:  pgtype.Timestamptz{Time: completedAt, Valid: true},
		ErrorMessage: pgtype.Text{String: errorMsg, Valid: true},
	})
	if err != nil {
		return false, apperrors.NewInternal("failed to mark run orphaned", err)
	}
	return affected > 0, nil
}

func (r *RunRepo) UpdateRunStatus(ctx context.Context, id uuid.UUID, status model.RunStatus, startedAt, completedAt, pausedAt *time.Time, errorMsg *string) (*model.Run, error) {
	params := UpdateRunStatusParams{
		ID:     id,
		Status: string(status),
	}
	if startedAt != nil {
		params.StartedAt = pgtype.Timestamptz{Time: *startedAt, Valid: true}
	}
	if completedAt != nil {
		params.CompletedAt = pgtype.Timestamptz{Time: *completedAt, Valid: true}
	}
	if pausedAt != nil {
		params.PausedAt = pgtype.Timestamptz{Time: *pausedAt, Valid: true}
	}
	if errorMsg != nil {
		params.ErrorMessage = pgtype.Text{String: *errorMsg, Valid: true}
	}

	row, err := r.queries.UpdateRunStatus(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("run", id)
		}
		return nil, apperrors.NewInternal("failed to update run status", err)
	}
	return toDomainRun(row), nil
}

func (r *RunRepo) UpdateRunMetadata(ctx context.Context, runID uuid.UUID, metadata map[string]interface{}) error {
	metadataJSON := []byte("{}")
	if metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return apperrors.NewInternal("failed to marshal run metadata", err)
		}
	}
	if err := r.queries.UpdateRunMetadata(ctx, UpdateRunMetadataParams{
		ID:       runID,
		Metadata: metadataJSON,
	}); err != nil {
		return apperrors.NewInternal("failed to update run metadata", err)
	}
	return nil
}

func (r *RunRepo) CountRunsByProject(ctx context.Context, projectID uuid.UUID) (int64, error) {
	count, err := r.queries.CountRunsByProject(ctx, projectID)
	if err != nil {
		return 0, apperrors.NewInternal("failed to count runs by project", err)
	}
	return count, nil
}

func (r *RunRepo) CountRunsByStory(ctx context.Context, storyID uuid.UUID) (int64, error) {
	count, err := r.queries.CountRunsByStory(ctx, storyID)
	if err != nil {
		return 0, apperrors.NewInternal("failed to count runs by story", err)
	}
	return count, nil
}

func (r *RunRepo) CreateRunStep(ctx context.Context, step *model.RunStep) (*model.RunStep, error) {
	params := CreateRunStepParams{
		RunID:     step.RunID,
		StepName:  step.StepName,
		StepOrder: int32(step.StepOrder),
		Action:    step.Action,
		Status:    string(step.Status),
		StageID:   pgtype.Text{String: step.StageID, Valid: step.StageID != ""},
		StageName: pgtype.Text{String: step.StageName, Valid: step.StageName != ""},
	}

	row, err := r.queries.CreateRunStep(ctx, params)
	if err != nil {
		return nil, apperrors.NewInternal("failed to create run step", err)
	}
	return toDomainRunStep(row), nil
}

func (r *RunRepo) GetRunStep(ctx context.Context, id uuid.UUID) (*model.RunStep, error) {
	row, err := r.queries.GetRunStep(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("run step", id)
		}
		return nil, apperrors.NewInternal("failed to get run step", err)
	}
	return toDomainRunStep(row), nil
}

func (r *RunRepo) ListRunStepsByRun(ctx context.Context, runID uuid.UUID) ([]*model.RunStep, error) {
	rows, err := r.queries.ListRunStepsByRun(ctx, runID)
	if err != nil {
		return nil, apperrors.NewInternal("failed to list run steps", err)
	}
	steps := make([]*model.RunStep, len(rows))
	for i, row := range rows {
		steps[i] = toDomainRunStep(row)
	}
	return steps, nil
}

func (r *RunRepo) UpdateRunStepStatus(ctx context.Context, id uuid.UUID, status model.StepStatus, startedAt, completedAt *time.Time, errorMsg *string) (*model.RunStep, error) {
	params := UpdateRunStepStatusParams{
		ID:     id,
		Status: string(status),
	}
	if startedAt != nil {
		params.StartedAt = pgtype.Timestamptz{Time: *startedAt, Valid: true}
	}
	if completedAt != nil {
		params.CompletedAt = pgtype.Timestamptz{Time: *completedAt, Valid: true}
	}
	if errorMsg != nil {
		params.ErrorMessage = pgtype.Text{String: *errorMsg, Valid: true}
	}

	row, err := r.queries.UpdateRunStepStatus(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("run step", id)
		}
		return nil, apperrors.NewInternal("failed to update run step status", err)
	}
	return toDomainRunStep(row), nil
}

func (r *RunRepo) UpdateRunStepContainerInfo(ctx context.Context, id uuid.UUID, containerID *string, logTail *string) (*model.RunStep, error) {
	params := UpdateRunStepContainerInfoParams{ID: id}
	if containerID != nil {
		params.ContainerID = pgtype.Text{String: *containerID, Valid: true}
	}
	if logTail != nil {
		params.LogTail = pgtype.Text{String: *logTail, Valid: true}
	}

	row, err := r.queries.UpdateRunStepContainerInfo(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("run step", id)
		}
		return nil, apperrors.NewInternal("failed to update run step container info", err)
	}
	return toDomainRunStep(row), nil
}

func (r *RunRepo) CreateRetryRunStep(ctx context.Context, step *model.RunStep) (*model.RunStep, error) {
	params := CreateRetryRunStepParams{
		ID:         step.ID,
		RunID:      step.RunID,
		StepName:   step.StepName,
		StepOrder:  int32(step.StepOrder),
		Action:     step.Action,
		Status:     string(step.Status),
		RetryCount: int32(step.RetryCount),
	}
	if step.RetryType != nil {
		params.RetryType = pgtype.Text{String: *step.RetryType, Valid: true}
	}
	if step.ParentStepID != nil {
		params.ParentStepID = pgtype.UUID{Bytes: *step.ParentStepID, Valid: true}
	}
	if step.StageID != "" {
		params.StageID = pgtype.Text{String: step.StageID, Valid: true}
	}
	if step.StageName != "" {
		params.StageName = pgtype.Text{String: step.StageName, Valid: true}
	}

	row, err := r.queries.CreateRetryRunStep(ctx, params)
	if err != nil {
		return nil, apperrors.NewInternal("failed to create retry run step", err)
	}
	return toDomainRunStep(row), nil
}

func (r *RunRepo) AppendStepLogTail(ctx context.Context, stepID uuid.UUID, text string) error {
	if err := r.queries.AppendRunStepLogTail(ctx, AppendRunStepLogTailParams{
		ID:      stepID,
		LogTail: pgtype.Text{String: text, Valid: true},
	}); err != nil {
		return apperrors.NewInternal("failed to append run step log tail", err)
	}
	return nil
}

func (r *RunRepo) ListRetryStepsByParent(ctx context.Context, parentStepID uuid.UUID) ([]*model.RunStep, error) {
	rows, err := r.queries.ListRetryStepsByParent(ctx, pgtype.UUID{Bytes: parentStepID, Valid: true})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list retry steps by parent", err)
	}
	steps := make([]*model.RunStep, len(rows))
	for i, row := range rows {
		steps[i] = toDomainRunStep(row)
	}
	return steps, nil
}
