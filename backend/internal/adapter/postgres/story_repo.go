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
	targetFiles, err := marshalJSONB(story.TargetFiles)
	if err != nil {
		return nil, err
	}
	dependsOn, err := marshalJSONB(story.DependsOn)
	if err != nil {
		return nil, err
	}
	params := CreateStoryParams{
		ProjectID:          story.ProjectID,
		EpicID:             uuidFromPtr(story.EpicID),
		Key:                story.Key,
		Title:              story.Title,
		Objective:          textFromStringPtr(story.Objective),
		TargetFiles:        targetFiles,
		DependsOn:          dependsOn,
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
	return toDomainStory(row)
}

func (r *StoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Story, error) {
	row, err := r.queries.GetStory(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("story", id)
		}
		return nil, apperrors.NewInternal("failed to get story", err)
	}
	return toDomainStory(row)
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
	return toDomainStory(row)
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
		s, err := toDomainStory(row)
		if err != nil {
			return nil, err
		}
		stories[i] = s
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
		s, err := toDomainStory(row)
		if err != nil {
			return nil, err
		}
		stories[i] = s
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
		s, err := toDomainStory(row)
		if err != nil {
			return nil, err
		}
		stories[i] = s
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

func (r *StoryRepo) CountByEpicGroupedByStatus(ctx context.Context, epicID uuid.UUID) (model.StoryCounts, error) {
	rows, err := r.queries.CountStoriesByEpicGroupedByStatus(ctx, pgtype.UUID{Bytes: epicID, Valid: true})
	if err != nil {
		return model.StoryCounts{}, apperrors.NewInternal("failed to count stories by epic grouped by status", err)
	}
	var counts model.StoryCounts
	for _, row := range rows {
		switch row.Status {
		case model.StoryStatusBacklog:
			counts.Backlog = int(row.Count)
		case model.StoryStatusRunning:
			counts.Running = int(row.Count)
		case model.StoryStatusDone:
			counts.Done = int(row.Count)
		case model.StoryStatusFailed:
			counts.Failed = int(row.Count)
		}
	}
	return counts, nil
}

func (r *StoryRepo) Update(ctx context.Context, story *model.Story) (*model.Story, error) {
	targetFiles, err := marshalJSONB(story.TargetFiles)
	if err != nil {
		return nil, err
	}
	dependsOn, err := marshalJSONB(story.DependsOn)
	if err != nil {
		return nil, err
	}
	params := UpdateStoryParams{
		ID:                 story.ID,
		Title:              textFromStringPtr(&story.Title),
		Objective:          textFromStringPtr(story.Objective),
		TargetFiles:        targetFiles,
		DependsOn:          dependsOn,
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
	return toDomainStory(row)
}

// UpdateStoryCurrentStage sets a story's current_stage to the given value.
// A nil currentStage clears the stage (NULL) — used at run completion.
func (r *StoryRepo) UpdateStoryCurrentStage(ctx context.Context, id uuid.UUID, currentStage *string) (*model.Story, error) {
	row, err := r.queries.UpdateStoryCurrentStage(ctx, UpdateStoryCurrentStageParams{
		ID:           id,
		CurrentStage: textFromStringPtr(currentStage),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("story", id)
		}
		return nil, apperrors.NewInternal("failed to update story current stage", err)
	}
	return toDomainStory(row)
}

func (r *StoryRepo) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.queries.DeleteStory(ctx, id)
	if err != nil {
		return apperrors.NewInternal("failed to delete story", err)
	}
	return nil
}

// GetBySourceRef resolves a remote-sourced story by (project, source, external_id).
func (r *StoryRepo) GetBySourceRef(ctx context.Context, projectID uuid.UUID, source, externalID string) (*model.Story, error) {
	row, err := r.queries.GetStoryBySourceRef(ctx, GetStoryBySourceRefParams{
		ProjectID:  projectID,
		Source:     source,
		ExternalID: pgtype.Text{String: externalID, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("story", externalID)
		}
		return nil, apperrors.NewInternal("failed to get story by source ref", err)
	}
	return toDomainStory(row)
}

// CreateFromImport inserts a story with the import-managed columns the service
// already computed. target_files / current_stage are intentionally not written.
func (r *StoryRepo) CreateFromImport(ctx context.Context, story *model.Story) (*model.Story, error) {
	dependsOn, err := marshalJSONB(story.DependsOn)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.CreateStoryFromImport(ctx, CreateStoryFromImportParams{
		ProjectID:          story.ProjectID,
		EpicID:             uuidFromPtr(story.EpicID),
		Key:                story.Key,
		Title:              story.Title,
		Objective:          textFromStringPtr(story.Objective),
		AcceptanceCriteria: textFromStringPtr(story.AcceptanceCriteria),
		Scope:              textFromStringPtr(story.Scope),
		DependsOn:          dependsOn,
		Status:             story.Status,
		Source:             story.Source,
		ExternalID:         textFromStringPtr(story.ExternalID),
		SourceUrl:          textFromStringPtr(story.SourceURL),
		LastImportHash:     textFromStringPtr(story.LastImportHash),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflict("story", story.Key)
		}
		return nil, apperrors.NewInternal("failed to create story from import", err)
	}
	return toDomainStory(row)
}

// UpdateFromImport overwrites the import-managed columns of an unlocked story.
func (r *StoryRepo) UpdateFromImport(ctx context.Context, story *model.Story) (*model.Story, error) {
	dependsOn, err := marshalJSONB(story.DependsOn)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.UpdateStoryFromImport(ctx, UpdateStoryFromImportParams{
		ID:                 story.ID,
		Title:              story.Title,
		Objective:          textFromStringPtr(story.Objective),
		AcceptanceCriteria: textFromStringPtr(story.AcceptanceCriteria),
		Scope:              textFromStringPtr(story.Scope),
		DependsOn:          dependsOn,
		Status:             story.Status,
		EpicID:             uuidFromPtr(story.EpicID),
		Source:             story.Source,
		ExternalID:         textFromStringPtr(story.ExternalID),
		SourceUrl:          textFromStringPtr(story.SourceURL),
		LastImportHash:     textFromStringPtr(story.LastImportHash),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("story", story.ID)
		}
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflict("story", story.Key)
		}
		return nil, apperrors.NewInternal("failed to update story from import", err)
	}
	return toDomainStory(row)
}

// UpdateProvenanceOnly refreshes a locked story's title + provenance only.
func (r *StoryRepo) UpdateProvenanceOnly(ctx context.Context, story *model.Story) (*model.Story, error) {
	row, err := r.queries.UpdateStoryProvenanceOnly(ctx, UpdateStoryProvenanceOnlyParams{
		ID:         story.ID,
		Title:      story.Title,
		Source:     story.Source,
		ExternalID: textFromStringPtr(story.ExternalID),
		SourceUrl:  textFromStringPtr(story.SourceURL),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("story", story.ID)
		}
		return nil, apperrors.NewInternal("failed to update story provenance", err)
	}
	return toDomainStory(row)
}

// toDomainStory maps a sqlc-generated Story to a domain Story.
func toDomainStory(s Story) (*model.Story, error) {
	targetFiles, err := unmarshalJSONBStringSlice(s.TargetFiles)
	if err != nil {
		return nil, err
	}
	dependsOn, err := unmarshalJSONBStringSlice(s.DependsOn)
	if err != nil {
		return nil, err
	}
	story := &model.Story{
		ID:          s.ID,
		ProjectID:   s.ProjectID,
		Key:         s.Key,
		Title:       s.Title,
		Status:      s.Status,
		TargetFiles: targetFiles,
		DependsOn:   dependsOn,
		Source:      s.Source,
		SyncedAt:    timeFromTimestamptz(s.SyncedAt),
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
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
	if s.CurrentStage.Valid {
		story.CurrentStage = &s.CurrentStage.String
	}
	if s.ExternalID.Valid {
		story.ExternalID = &s.ExternalID.String
	}
	if s.SourceUrl.Valid {
		story.SourceURL = &s.SourceUrl.String
	}
	if s.LastImportHash.Valid {
		story.LastImportHash = &s.LastImportHash.String
	}
	return story, nil
}

// marshalJSONB marshals a string slice to JSONB bytes. Returns nil for nil slices.
func marshalJSONB(vals []string) ([]byte, error) {
	if vals == nil {
		return nil, nil
	}
	b, err := json.Marshal(vals)
	if err != nil {
		return nil, apperrors.NewInternal("failed to marshal JSONB field", err)
	}
	return b, nil
}

// unmarshalJSONBStringSlice unmarshals JSONB bytes to a string slice. Returns nil for nil/empty input.
func unmarshalJSONBStringSlice(data []byte) ([]string, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var result []string
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, apperrors.NewInternal("failed to unmarshal JSONB field", err)
	}
	return result, nil
}
