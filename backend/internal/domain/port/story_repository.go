package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// StoryRepository defines the interface for story persistence operations.
type StoryRepository interface {
	Create(ctx context.Context, story *model.Story) (*model.Story, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Story, error)
	GetByKey(ctx context.Context, projectID uuid.UUID, key string) (*model.Story, error)
	ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Story, error)
	ListByStatus(ctx context.Context, projectID uuid.UUID, statuses []string, limit, offset int32) ([]*model.Story, error)
	ListByEpic(ctx context.Context, epicID uuid.UUID, limit, offset int32) ([]*model.Story, error)
	CountByProject(ctx context.Context, projectID uuid.UUID) (int64, error)
	CountByStatus(ctx context.Context, projectID uuid.UUID, statuses []string) (int64, error)
	// CountByEpicGroupedByStatus returns story counts per lifecycle status for an epic
	// in a single GROUP BY query.
	CountByEpicGroupedByStatus(ctx context.Context, epicID uuid.UUID) (model.StoryCounts, error)
	Update(ctx context.Context, story *model.Story) (*model.Story, error)
	// UpdateStoryCurrentStage sets the story's current_stage. A nil currentStage
	// clears the stage (NULL). Advanced by the executor at stage boundaries.
	UpdateStoryCurrentStage(ctx context.Context, id uuid.UUID, currentStage *string) (*model.Story, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
