package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// EnvironmentRepository persists a project's execution composition: the stacks,
// sidecar services, config source and commands that describe how a project runs.
//
// Product decision: exactly one Environment per project (UNIQUE project_id). Create
// returns a Conflict if the project already has one; GetByProjectID returns the single
// row or NotFound when absent. This is the P2c1 persistence layer only — no API or
// run-path wiring yet (P2c2).
type EnvironmentRepository interface {
	// Create persists a new environment. Returns Conflict if the project already has one.
	Create(ctx context.Context, e *model.Environment) (*model.Environment, error)
	// GetByID returns an environment by its id, or NotFound.
	GetByID(ctx context.Context, id uuid.UUID) (*model.Environment, error)
	// GetByProjectID returns the project's single environment, or NotFound when absent.
	GetByProjectID(ctx context.Context, projectID uuid.UUID) (*model.Environment, error)
	// Update mutates an existing environment's stacks/services/source/commands.
	Update(ctx context.Context, e *model.Environment) (*model.Environment, error)
	// Delete removes an environment by its id.
	Delete(ctx context.Context, id uuid.UUID) error
}
