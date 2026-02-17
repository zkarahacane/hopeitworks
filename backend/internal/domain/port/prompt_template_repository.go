package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// PromptTemplateRepository defines the interface for prompt template persistence operations.
type PromptTemplateRepository interface {
	Create(ctx context.Context, template *model.PromptTemplate) (*model.PromptTemplate, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.PromptTemplate, error)
	GetByProjectAndName(ctx context.Context, projectID uuid.UUID, name string) (*model.PromptTemplate, error)
	ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.PromptTemplate, error)
	CountByProject(ctx context.Context, projectID uuid.UUID) (int64, error)
	Update(ctx context.Context, template *model.PromptTemplate) (*model.PromptTemplate, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
