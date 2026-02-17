package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// PipelineConfigRepository defines the interface for pipeline config persistence operations.
type PipelineConfigRepository interface {
	GetByProjectID(ctx context.Context, projectID uuid.UUID) (*model.PipelineConfig, error)
	Upsert(ctx context.Context, config *model.PipelineConfig) (*model.PipelineConfig, error)
}
