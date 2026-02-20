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
	params := CreateRunParams{
		ProjectID:              run.ProjectID,
		StoryID:                run.StoryID,
		Status:                 string(run.Status),
		PipelineConfigSnapshot: []byte(run.PipelineConfigSnapshot),
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
	row, err := r.queries.GetRun(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("run", id)
		}
		return nil, apperrors.NewInternal("failed to get run", err)
	}
	return toDomainRun(row), nil
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

func (r *RunRepo) ListRunsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Run, error) {
	rows, err := r.queries.ListRunsByProject(ctx, ListRunsByProjectParams{
		ProjectID: projectID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list runs by project", err)
	}
	runs := make([]*model.Run, len(rows))
	for i, row := range rows {
		runs[i] = toDomainRun(row)
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
		runs[i] = toDomainRun(row)
	}
	return runs, nil
}

func (r *RunRepo) UpdateRunStatus(ctx context.Context, id uuid.UUID, status model.RunStatus, startedAt, completedAt *time.Time, errorMsg *string) (*model.Run, error) {
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

// toDomainRun maps a sqlc-generated Run to a domain Run.
func toDomainRun(r Run) *model.Run {
	run := &model.Run{
		ID:                     r.ID,
		ProjectID:              r.ProjectID,
		StoryID:                r.StoryID,
		Status:                 model.RunStatus(r.Status),
		PipelineConfigSnapshot: json.RawMessage(r.PipelineConfigSnapshot),
		CreatedAt:              r.CreatedAt,
		UpdatedAt:              r.UpdatedAt,
	}
	if r.StartedAt.Valid {
		run.StartedAt = &r.StartedAt.Time
	}
	if r.CompletedAt.Valid {
		run.CompletedAt = &r.CompletedAt.Time
	}
	if r.ErrorMessage.Valid {
		run.ErrorMessage = &r.ErrorMessage.String
	}
	return run
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

	row, err := r.queries.CreateRetryRunStep(ctx, params)
	if err != nil {
		return nil, apperrors.NewInternal("failed to create retry run step", err)
	}
	return toDomainRunStep(row), nil
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

// toDomainRunStep maps a sqlc-generated RunStep to a domain RunStep.
func toDomainRunStep(s RunStep) *model.RunStep {
	step := &model.RunStep{
		ID:         s.ID,
		RunID:      s.RunID,
		StepName:   s.StepName,
		StepOrder:  int(s.StepOrder),
		Action:     s.Action,
		Status:     model.StepStatus(s.Status),
		RetryCount: int(s.RetryCount),
		CreatedAt:  s.CreatedAt,
	}
	if s.StartedAt.Valid {
		step.StartedAt = &s.StartedAt.Time
	}
	if s.CompletedAt.Valid {
		step.CompletedAt = &s.CompletedAt.Time
	}
	if s.ErrorMessage.Valid {
		step.ErrorMessage = &s.ErrorMessage.String
	}
	if s.ContainerID.Valid {
		step.ContainerID = &s.ContainerID.String
	}
	if s.LogTail.Valid {
		step.LogTail = &s.LogTail.String
	}
	if s.RetryType.Valid {
		step.RetryType = &s.RetryType.String
	}
	if s.ParentStepID.Valid {
		uid := uuid.UUID(s.ParentStepID.Bytes)
		step.ParentStepID = &uid
	}
	return step
}
