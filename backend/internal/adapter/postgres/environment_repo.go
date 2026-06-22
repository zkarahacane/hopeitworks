package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure EnvironmentRepo implements port.EnvironmentRepository at compile time.
var _ port.EnvironmentRepository = (*EnvironmentRepo)(nil)

// EnvironmentRepo implements port.EnvironmentRepository using sqlc-generated queries.
type EnvironmentRepo struct {
	queries *Queries
}

// NewEnvironmentRepo creates a new EnvironmentRepo.
func NewEnvironmentRepo(queries *Queries) *EnvironmentRepo {
	return &EnvironmentRepo{queries: queries}
}

// Create persists a new environment. The UNIQUE constraint on project_id enforces
// one environment per project: a second insert maps to a Conflict.
func (r *EnvironmentRepo) Create(ctx context.Context, e *model.Environment) (*model.Environment, error) {
	services, err := marshalServices(e.Services)
	if err != nil {
		return nil, err
	}
	commands, err := marshalStringMap(e.Commands)
	if err != nil {
		return nil, apperrors.NewInternal("failed to marshal environment commands", err)
	}

	row, err := r.queries.CreateEnvironment(ctx, CreateEnvironmentParams{
		ProjectID: e.ProjectID,
		Stacks:    e.Stacks,
		Services:  services,
		Source:    e.Source,
		Commands:  commands,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflict("environment", e.ProjectID.String())
		}
		if isForeignKeyViolation(err) {
			return nil, apperrors.NewNotFound("project", e.ProjectID)
		}
		return nil, apperrors.NewInternal("failed to create environment", err)
	}
	return toDomainEnvironment(row)
}

// GetByID returns an environment by its id.
func (r *EnvironmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Environment, error) {
	row, err := r.queries.GetEnvironment(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("environment", id)
		}
		return nil, apperrors.NewInternal("failed to get environment", err)
	}
	return toDomainEnvironment(row)
}

// GetByProjectID returns the project's single environment, or NotFound when absent.
func (r *EnvironmentRepo) GetByProjectID(ctx context.Context, projectID uuid.UUID) (*model.Environment, error) {
	row, err := r.queries.GetEnvironmentByProjectID(ctx, projectID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("environment", projectID)
		}
		return nil, apperrors.NewInternal("failed to get environment by project id", err)
	}
	return toDomainEnvironment(row)
}

// Update mutates an existing environment's stacks/services/source/commands. The
// updated_at column is maintained by the set_environments_updated_at trigger.
func (r *EnvironmentRepo) Update(ctx context.Context, e *model.Environment) (*model.Environment, error) {
	services, err := marshalServices(e.Services)
	if err != nil {
		return nil, err
	}
	commands, err := marshalStringMap(e.Commands)
	if err != nil {
		return nil, apperrors.NewInternal("failed to marshal environment commands", err)
	}

	row, err := r.queries.UpdateEnvironment(ctx, UpdateEnvironmentParams{
		ID:       e.ID,
		Stacks:   e.Stacks,
		Services: services,
		Source:   e.Source,
		Commands: commands,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("environment", e.ID)
		}
		return nil, apperrors.NewInternal("failed to update environment", err)
	}
	return toDomainEnvironment(row)
}

// Delete removes an environment by its id.
func (r *EnvironmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteEnvironment(ctx, id); err != nil {
		return apperrors.NewInternal("failed to delete environment", err)
	}
	return nil
}

// toDomainEnvironment maps a sqlc-generated Environment to the domain model,
// unmarshalling the JSONB services and commands columns.
func toDomainEnvironment(e Environment) (*model.Environment, error) {
	services, err := unmarshalServices(e.Services)
	if err != nil {
		return nil, err
	}
	commands, err := unmarshalStringMap(e.Commands)
	if err != nil {
		return nil, apperrors.NewInternal("failed to unmarshal environment commands", err)
	}
	stacks := e.Stacks
	if stacks == nil {
		stacks = []string{}
	}
	return &model.Environment{
		ID:        e.ID,
		ProjectID: e.ProjectID,
		Stacks:    stacks,
		Services:  services,
		Source:    e.Source,
		Commands:  commands,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}, nil
}

// marshalServices serialises []EnvironmentService to JSON bytes for JSONB storage.
func marshalServices(s []model.EnvironmentService) ([]byte, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, apperrors.NewInternal("failed to marshal environment services", err)
	}
	return b, nil
}

// unmarshalServices deserialises JSONB bytes into []EnvironmentService.
func unmarshalServices(data []byte) ([]model.EnvironmentService, error) {
	if len(data) == 0 {
		return []model.EnvironmentService{}, nil
	}
	var s []model.EnvironmentService
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, apperrors.NewInternal("failed to unmarshal environment services", err)
	}
	if s == nil {
		s = []model.EnvironmentService{}
	}
	return s, nil
}
