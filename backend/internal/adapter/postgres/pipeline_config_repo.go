package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure PipelineConfigRepo implements port.PipelineConfigRepository at compile time.
var _ port.PipelineConfigRepository = (*PipelineConfigRepo)(nil)

// PipelineConfigRepo implements port.PipelineConfigRepository using sqlc-generated queries.
type PipelineConfigRepo struct {
	queries *Queries
}

// NewPipelineConfigRepo creates a new PipelineConfigRepo.
func NewPipelineConfigRepo(queries *Queries) *PipelineConfigRepo {
	return &PipelineConfigRepo{queries: queries}
}

func (r *PipelineConfigRepo) GetByProjectID(ctx context.Context, projectID uuid.UUID) (*model.PipelineConfig, error) {
	row, err := r.queries.GetPipelineConfig(ctx, projectID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("pipeline_config", projectID)
		}
		return nil, apperrors.NewInternal("failed to get pipeline config", err)
	}
	return toDomainPipelineConfig(row), nil
}

func (r *PipelineConfigRepo) Upsert(ctx context.Context, config *model.PipelineConfig) (*model.PipelineConfig, error) {
	params := UpsertPipelineConfigParams{
		ProjectID:  config.ProjectID,
		ConfigYaml: config.ConfigYAML,
	}

	row, err := r.queries.UpsertPipelineConfig(ctx, params)
	if err != nil {
		return nil, apperrors.NewInternal("failed to upsert pipeline config", err)
	}
	return toDomainPipelineConfig(row), nil
}

// toDomainPipelineConfig maps a sqlc-generated PipelineConfig to a domain PipelineConfig.
func toDomainPipelineConfig(p PipelineConfig) *model.PipelineConfig {
	return &model.PipelineConfig{
		ID:         p.ID,
		ProjectID:  p.ProjectID,
		ConfigYAML: p.ConfigYaml,
		Version:    int(p.Version),
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}
