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

// Ensure CapabilityRepo implements port.CapabilityRepository at compile time.
var _ port.CapabilityRepository = (*CapabilityRepo)(nil)

// CapabilityRepo implements port.CapabilityRepository using sqlc-generated queries.
type CapabilityRepo struct {
	queries *Queries
}

// NewCapabilityRepository creates a new CapabilityRepo.
func NewCapabilityRepository(queries *Queries) *CapabilityRepo {
	return &CapabilityRepo{queries: queries}
}

// Create inserts a new capability.
func (r *CapabilityRepo) Create(ctx context.Context, c *model.Capability) (*model.Capability, error) {
	version := c.Version
	if version == 0 {
		version = 1
	}
	row, err := r.queries.CreateCapability(ctx, CreateCapabilityParams{
		Kind:      c.Kind,
		Name:      c.Name,
		Version:   int32(version),
		Scope:     c.Scope,
		ProjectID: uuidFromPtr(c.ProjectID),
		Spec:      c.Spec,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflict("capability", c.Name)
		}
		if isForeignKeyViolation(err) {
			return nil, apperrors.NewNotFound("project", c.ProjectID)
		}
		return nil, apperrors.NewInternal("failed to create capability", err)
	}
	return toDomainCapability(row), nil
}

// GetByID retrieves a capability by ID.
func (r *CapabilityRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
	row, err := r.queries.GetCapability(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("capability", id)
		}
		return nil, apperrors.NewInternal("failed to get capability", err)
	}
	return toDomainCapability(row), nil
}

// ListByScope returns all global capabilities plus those owned by projectID (if non-nil).
func (r *CapabilityRepo) ListByScope(ctx context.Context, projectID *uuid.UUID) ([]*model.Capability, error) {
	scope := uuid.Nil
	if projectID != nil {
		scope = *projectID
	}
	rows, err := r.queries.ListCapabilitiesByScope(ctx, scope)
	if err != nil {
		return nil, apperrors.NewInternal("failed to list capabilities", err)
	}
	return toDomainCapabilities(rows), nil
}

// Delete removes a capability by ID.
func (r *CapabilityRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteCapability(ctx, id); err != nil {
		return apperrors.NewInternal("failed to delete capability", err)
	}
	return nil
}

// AttachToAgent composes a capability onto an agent (idempotent).
func (r *CapabilityRepo) AttachToAgent(ctx context.Context, agentID, capabilityID uuid.UUID) error {
	err := r.queries.AttachCapabilityToAgent(ctx, AttachCapabilityToAgentParams{
		AgentID:      agentID,
		CapabilityID: capabilityID,
	})
	if err != nil {
		if isForeignKeyViolation(err) {
			return apperrors.NewNotFound("agent_or_capability", agentID)
		}
		return apperrors.NewInternal("failed to attach capability to agent", err)
	}
	return nil
}

// DetachFromAgent removes a capability composition from an agent.
func (r *CapabilityRepo) DetachFromAgent(ctx context.Context, agentID, capabilityID uuid.UUID) error {
	err := r.queries.DetachCapabilityFromAgent(ctx, DetachCapabilityFromAgentParams{
		AgentID:      agentID,
		CapabilityID: capabilityID,
	})
	if err != nil {
		return apperrors.NewInternal("failed to detach capability from agent", err)
	}
	return nil
}

// ListForAgent returns the capabilities composed onto an agent.
func (r *CapabilityRepo) ListForAgent(ctx context.Context, agentID uuid.UUID) ([]*model.Capability, error) {
	rows, err := r.queries.ListCapabilitiesForAgent(ctx, agentID)
	if err != nil {
		return nil, apperrors.NewInternal("failed to list capabilities for agent", err)
	}
	return toDomainCapabilities(rows), nil
}

// toDomainCapability maps a sqlc-generated Capability to the domain model.
func toDomainCapability(c Capability) *model.Capability {
	return &model.Capability{
		ID:        c.ID,
		Kind:      c.Kind,
		Name:      c.Name,
		Version:   int(c.Version),
		Scope:     c.Scope,
		ProjectID: pgtypeToUUIDPtr(c.ProjectID),
		Spec:      c.Spec,
		CreatedAt: c.CreatedAt,
	}
}

func toDomainCapabilities(rows []Capability) []*model.Capability {
	out := make([]*model.Capability, len(rows))
	for i, row := range rows {
		out[i] = toDomainCapability(row)
	}
	return out
}
