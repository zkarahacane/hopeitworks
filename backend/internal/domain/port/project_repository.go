package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// ProjectRepository defines the interface for project persistence operations.
type ProjectRepository interface {
	Create(ctx context.Context, project *model.Project) (*model.Project, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error)
	List(ctx context.Context, limit, offset int32) ([]*model.Project, error)
	Count(ctx context.Context) (int64, error)
	Update(ctx context.Context, project *model.Project) (*model.Project, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
