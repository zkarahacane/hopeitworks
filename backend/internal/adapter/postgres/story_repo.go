package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure StoryRepo implements port.StoryRepository at compile time.
var _ port.StoryRepository = (*StoryRepo)(nil)

// StoryRepo implements port.StoryRepository using sqlc-generated queries.
type StoryRepo struct {
	queries *Queries
}

// NewStoryRepo creates a new StoryRepo.
func NewStoryRepo(queries *Queries) *StoryRepo {
	return &StoryRepo{queries: queries}
}

func (r *StoryRepo) Create(ctx context.Context, story *model.Story) (*model.Story, error) {
	params := CreateStoryParams{
		ProjectID:          story.ProjectID,
		EpicID:             uuidFromPtr(story.EpicID),
		Key:                story.Key,
		Title:              story.Title,
		Objective:          textFromStringPtr(story.Objective),
		TargetFiles:        marshalJSONB(story.TargetFiles),
		DependsOn:          marshalJSONB(story.DependsOn),
		Scope:              textFromStringPtr(story.Scope),
		Status:             story.Status,
		AcceptanceCriteria: textFromStringPtr(story.AcceptanceCriteria),
	}

	row, err := r.queries.CreateStory(ctx, params)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflict("story", story.Key)
		}
		return nil, apperrors.NewInternal("failed to create story", err)
	}
	return toDomainStory(row), nil
}

func (r *StoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Story, error) {
	row, err := r.queries.GetStory(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("story", id)
		}
		return nil, apperrors.NewInternal("failed to get story", err)
	}
	return toDomainStory(row), nil
}

func (r *StoryRepo) GetByKey(ctx context.Context, projectID uuid.UUID, key string) (*model.Story, error) {
	row, err := r.queries.GetStoryByKey(ctx, GetStoryByKeyParams{
		ProjectID: projectID,
		Key:       key,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("story", key)
		}
		return nil, apperrors.NewInternal("failed to get story by key", err)
	}
	return toDomainStory(row), nil
}

func (r *StoryRepo) ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Story, error) {
	rows, err := r.queries.ListStoriesByProject(ctx, ListStoriesByProjectParams{
		ProjectID: projectID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list stories", err)
	}
	stories := make([]*model.Story, len(rows))
	for i, row := range rows {
		stories[i] = toDomainStory(row)
	}
	return stories, nil
}

func (r *StoryRepo) ListByStatus(ctx context.Context, projectID uuid.UUID, statuses []string, limit, offset int32) ([]*model.Story, error) {
	rows, err := r.queries.ListStoriesByStatus(ctx, ListStoriesByStatusParams{
		ProjectID: projectID,
		Column2:   statuses,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list stories by status", err)
	}
	stories := make([]*model.Story, len(rows))
	for i, row := range rows {
		stories[i] = toDomainStory(row)
	}
	return stories, nil
}

func (r *StoryRepo) ListByEpic(ctx context.Context, epicID uuid.UUID, limit, offset int32) ([]*model.Story, error) {
	rows, err := r.queries.ListStoriesByEpic(ctx, ListStoriesByEpicParams{
		EpicID: pgtype.UUID{Bytes: epicID, Valid: true},
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list stories by epic", err)
	}
	stories := make([]*model.Story, len(rows))
	for i, row := range rows {
		stories[i] = toDomainStory(row)
	}
	return stories, nil
}

func (r *StoryRepo) CountByProject(ctx context.Context, projectID uuid.UUID) (int64, error) {
	count, err := r.queries.CountStoriesByProject(ctx, projectID)
	if err != nil {
		return 0, apperrors.NewInternal("failed to count stories", err)
	}
	return count, nil
}

func (r *StoryRepo) CountByStatus(ctx context.Context, projectID uuid.UUID, statuses []string) (int64, error) {
	count, err := r.queries.CountStoriesByStatus(ctx, CountStoriesByStatusParams{
		ProjectID: projectID,
		Column2:   statuses,
	})
	if err != nil {
		return 0, apperrors.NewInternal("failed to count stories by status", err)
	}
	return count, nil
}

func (r *StoryRepo) Update(ctx context.Context, story *model.Story) (*model.Story, error) {
	params := UpdateStoryParams{
		ID:                 story.ID,
		Title:              textFromStringPtr(&story.Title),
		Objective:          textFromStringPtr(story.Objective),
		TargetFiles:        marshalJSONB(story.TargetFiles),
		DependsOn:          marshalJSONB(story.DependsOn),
		Scope:              textFromStringPtr(story.Scope),
		Status:             pgtype.Text{String: story.Status, Valid: true},
		AcceptanceCriteria: textFromStringPtr(story.AcceptanceCriteria),
		EpicID:             uuidFromPtr(story.EpicID),
	}

	row, err := r.queries.UpdateStory(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("story", story.ID)
		}
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflict("story", story.Key)
		}
		return nil, apperrors.NewInternal("failed to update story", err)
	}
	return toDomainStory(row), nil
}

func (r *StoryRepo) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.queries.DeleteStory(ctx, id)
	if err != nil {
		return apperrors.NewInternal("failed to delete story", err)
	}
	return nil
}

// toDomainStory maps a sqlc-generated Story to a domain Story.
func toDomainStory(s Story) *model.Story {
	story := &model.Story{
		ID:        s.ID,
		ProjectID: s.ProjectID,
		Key:       s.Key,
		Title:     s.Title,
		Status:    s.Status,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
	if s.EpicID.Valid {
		epicID := uuid.UUID(s.EpicID.Bytes)
		story.EpicID = &epicID
	}
	if s.Objective.Valid {
		story.Objective = &s.Objective.String
	}
	if s.Scope.Valid {
		story.Scope = &s.Scope.String
	}
	if s.AcceptanceCriteria.Valid {
		story.AcceptanceCriteria = &s.AcceptanceCriteria.String
	}
	story.TargetFiles = unmarshalJSONBStringSlice(s.TargetFiles)
	story.DependsOn = unmarshalJSONBStringSlice(s.DependsOn)
	return story
}

// marshalJSONB marshals a string slice to JSONB bytes. Returns nil for nil/empty slices.
func marshalJSONB(vals []string) []byte {
	if vals == nil {
		return nil
	}
	b, err := json.Marshal(vals)
	if err != nil {
		return nil
	}
	return b
}

// unmarshalJSONBStringSlice unmarshals JSONB bytes to a string slice. Returns nil for nil/empty input.
func unmarshalJSONBStringSlice(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	var result []string
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}
