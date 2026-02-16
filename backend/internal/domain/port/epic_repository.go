package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// EpicRepository defines the interface for epic persistence operations.
type EpicRepository interface {
	Create(ctx context.Context, epic *model.Epic) (*model.Epic, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Epic, error)
	ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Epic, error)
	CountByProject(ctx context.Context, projectID uuid.UUID) (int64, error)
	Update(ctx context.Context, epic *model.Epic) (*model.Epic, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
