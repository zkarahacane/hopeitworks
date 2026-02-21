package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure EpicRunRepo implements port.EpicRunRepository at compile time.
var _ port.EpicRunRepository = (*EpicRunRepo)(nil)

// EpicRunRepo implements port.EpicRunRepository using sqlc-generated queries.
type EpicRunRepo struct {
	queries *Queries
}

// NewEpicRunRepo creates a new EpicRunRepo.
func NewEpicRunRepo(queries *Queries) *EpicRunRepo {
	return &EpicRunRepo{queries: queries}
}

// CreateEpicRun creates a new epic run record.
func (r *EpicRunRepo) CreateEpicRun(ctx context.Context, run *model.EpicRun) (*model.EpicRun, error) {
	params := CreateEpicRunParams{
		ProjectID: run.ProjectID,
		EpicID:    run.EpicID,
		Status:    EpicRunStatus(run.Status),
	}

	row, err := r.queries.CreateEpicRun(ctx, params)
	if err != nil {
		if isForeignKeyViolation(err) {
			return nil, apperrors.NewNotFound("project or epic", run.ProjectID)
		}
		return nil, apperrors.NewInternal("failed to create epic run", err)
	}
	return toDomainEpicRun(row), nil
}

// GetEpicRun retrieves an epic run by ID.
func (r *EpicRunRepo) GetEpicRun(ctx context.Context, id uuid.UUID) (*model.EpicRun, error) {
	row, err := r.queries.GetEpicRun(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("epic_run", id)
		}
		return nil, apperrors.NewInternal("failed to get epic run", err)
	}
	return toDomainEpicRun(row), nil
}

// UpdateEpicRunStatus updates the status and optional completion time.
func (r *EpicRunRepo) UpdateEpicRunStatus(ctx context.Context, id uuid.UUID, status model.EpicRunStatus, completedAt *time.Time) (*model.EpicRun, error) {
	params := UpdateEpicRunStatusParams{
		ID:     id,
		Status: EpicRunStatus(status),
	}
	if completedAt != nil {
		params.CompletedAt = pgtype.Timestamptz{Time: *completedAt, Valid: true}
	}

	row, err := r.queries.UpdateEpicRunStatus(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("epic_run", id)
		}
		return nil, apperrors.NewInternal("failed to update epic run status", err)
	}
	return toDomainEpicRun(row), nil
}

// InsertEpicRunStory inserts a story association for an epic run.
func (r *EpicRunRepo) InsertEpicRunStory(ctx context.Context, story model.EpicRunStory) error {
	params := InsertEpicRunStoryParams{
		EpicRunID:  story.EpicRunID,
		StoryID:    story.StoryID,
		GroupIndex: int32(story.GroupIndex),
		Status:     story.Status,
	}
	if story.RunID != nil {
		params.RunID = pgtype.UUID{Bytes: *story.RunID, Valid: true}
	}

	err := r.queries.InsertEpicRunStory(ctx, params)
	if err != nil {
		return apperrors.NewInternal("failed to insert epic run story", err)
	}
	return nil
}

// UpdateEpicRunStoryStatus updates the status and run ID for a story in an epic run.
func (r *EpicRunRepo) UpdateEpicRunStoryStatus(ctx context.Context, epicRunID, storyID uuid.UUID, status string, runID *uuid.UUID) error {
	params := UpdateEpicRunStoryStatusParams{
		EpicRunID: epicRunID,
		StoryID:   storyID,
		Status:    status,
	}
	if runID != nil {
		params.RunID = pgtype.UUID{Bytes: *runID, Valid: true}
	}

	err := r.queries.UpdateEpicRunStoryStatus(ctx, params)
	if err != nil {
		return apperrors.NewInternal("failed to update epic run story status", err)
	}
	return nil
}

// ListEpicRunStories returns all stories for an epic run ordered by group index.
func (r *EpicRunRepo) ListEpicRunStories(ctx context.Context, epicRunID uuid.UUID) ([]model.EpicRunStory, error) {
	rows, err := r.queries.ListEpicRunStories(ctx, epicRunID)
	if err != nil {
		return nil, apperrors.NewInternal("failed to list epic run stories", err)
	}

	stories := make([]model.EpicRunStory, len(rows))
	for i, row := range rows {
		stories[i] = toDomainEpicRunStory(row)
	}
	return stories, nil
}

// toDomainEpicRun maps a sqlc-generated EpicRun to a domain EpicRun.
func toDomainEpicRun(r EpicRun) *model.EpicRun {
	epicRun := &model.EpicRun{
		ID:        r.ID,
		ProjectID: r.ProjectID,
		EpicID:    r.EpicID,
		Status:    model.EpicRunStatus(r.Status),
		CreatedAt: r.CreatedAt,
	}
	if r.CompletedAt.Valid {
		epicRun.CompletedAt = &r.CompletedAt.Time
	}
	return epicRun
}

// toDomainEpicRunStory maps a sqlc-generated EpicRunStory to a domain EpicRunStory.
func toDomainEpicRunStory(s EpicRunStory) model.EpicRunStory {
	story := model.EpicRunStory{
		EpicRunID:  s.EpicRunID,
		StoryID:    s.StoryID,
		GroupIndex: int(s.GroupIndex),
		Status:     s.Status,
	}
	if s.RunID.Valid {
		runID := uuid.UUID(s.RunID.Bytes)
		story.RunID = &runID
	}
	return story
}
