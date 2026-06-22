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

// Ensure CredentialRepo implements port.CredentialRepository at compile time.
var _ port.CredentialRepository = (*CredentialRepo)(nil)

// CredentialRepo implements port.CredentialRepository using sqlc-generated queries.
// It only ever stores/returns ciphertext; encryption lives in the CredentialService.
type CredentialRepo struct {
	queries *Queries
}

// NewCredentialRepository creates a new CredentialRepo.
func NewCredentialRepository(queries *Queries) *CredentialRepo {
	return &CredentialRepo{queries: queries}
}

// Create inserts a new encrypted credential.
func (r *CredentialRepo) Create(ctx context.Context, c *model.Credential) (*model.Credential, error) {
	row, err := r.queries.CreateCredential(ctx, CreateCredentialParams{
		Name:           c.Name,
		Scope:          c.Scope,
		ProjectID:      uuidFromPtr(c.ProjectID),
		EncryptedValue: c.EncryptedValue,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflict("credential", c.Name)
		}
		if isForeignKeyViolation(err) {
			return nil, apperrors.NewNotFound("project", c.ProjectID)
		}
		return nil, apperrors.NewInternal("failed to create credential", err)
	}
	return toDomainCredential(row), nil
}

// GetByID retrieves a credential by ID.
func (r *CredentialRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Credential, error) {
	row, err := r.queries.GetCredential(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("credential", id)
		}
		return nil, apperrors.NewInternal("failed to get credential", err)
	}
	return toDomainCredential(row), nil
}

// GetGlobalByName retrieves a global-scoped credential by name.
func (r *CredentialRepo) GetGlobalByName(ctx context.Context, name string) (*model.Credential, error) {
	row, err := r.queries.GetGlobalCredentialByName(ctx, name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("credential", name)
		}
		return nil, apperrors.NewInternal("failed to get global credential", err)
	}
	return toDomainCredential(row), nil
}

// GetProjectByName retrieves a project-scoped credential by name.
func (r *CredentialRepo) GetProjectByName(ctx context.Context, name string, projectID uuid.UUID) (*model.Credential, error) {
	row, err := r.queries.GetProjectCredentialByName(ctx, GetProjectCredentialByNameParams{
		Name:      name,
		ProjectID: uuidFromPtr(&projectID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("credential", name)
		}
		return nil, apperrors.NewInternal("failed to get project credential", err)
	}
	return toDomainCredential(row), nil
}

// ListByScope returns all global credentials plus those owned by projectID (if non-nil).
func (r *CredentialRepo) ListByScope(ctx context.Context, projectID *uuid.UUID) ([]*model.Credential, error) {
	scope := uuid.Nil
	if projectID != nil {
		scope = *projectID
	}
	rows, err := r.queries.ListCredentialsByScope(ctx, scope)
	if err != nil {
		return nil, apperrors.NewInternal("failed to list credentials", err)
	}
	out := make([]*model.Credential, len(rows))
	for i, row := range rows {
		out[i] = toDomainCredential(row)
	}
	return out, nil
}

// Delete removes a credential by ID.
func (r *CredentialRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteCredential(ctx, id); err != nil {
		return apperrors.NewInternal("failed to delete credential", err)
	}
	return nil
}

// toDomainCredential maps a sqlc-generated Credential to the domain model.
func toDomainCredential(c Credential) *model.Credential {
	return &model.Credential{
		ID:             c.ID,
		Name:           c.Name,
		Scope:          c.Scope,
		ProjectID:      pgtypeToUUIDPtr(c.ProjectID),
		EncryptedValue: c.EncryptedValue,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}
