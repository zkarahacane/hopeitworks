package port

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// UpsertGitConnectionParams carries the full state of a PAT connection to persist.
// On conflict (one row per project) it replaces the existing row.
type UpsertGitConnectionParams struct {
	ProjectID       uuid.UUID
	Provider        string
	EncryptedSecret []byte
	SecretLast4     *string
	TokenType       *string
	Scopes          []string
	Status          model.GitConnectionStatus
	AccountLogin    *string
	ExpiresAt       *time.Time
	LastValidatedAt *time.Time
	ValidationError *string
}

// SetValidationParams refreshes the advisory validation metadata of an existing
// connection (used by "Test connection" on a stored token).
type SetValidationParams struct {
	ProjectID       uuid.UUID
	Status          model.GitConnectionStatus
	AccountLogin    *string
	Scopes          []string
	ExpiresAt       *time.Time
	ValidationError *string
}

// GitConnectionRepository persists one PAT connection per project. It only ever
// stores/returns ciphertext; encryption lives in the GitConnectionService.
type GitConnectionRepository interface {
	// GetByProject returns the project's connection, or a not-found DomainError.
	GetByProject(ctx context.Context, projectID uuid.UUID) (*model.GitConnection, error)
	// Upsert inserts or replaces the project's connection, returning the stored row.
	Upsert(ctx context.Context, p UpsertGitConnectionParams) (*model.GitConnection, error)
	// SetValidation refreshes validation metadata in place (no-op if absent).
	SetValidation(ctx context.Context, p SetValidationParams) error
	// MarkStatus flips status + validation_error in place, preserving other
	// last-known fields (no-op if absent). Used by lazy expiry and the C1 self-heal.
	MarkStatus(ctx context.Context, projectID uuid.UUID, status model.GitConnectionStatus, validationError *string) error
	// Delete removes the project's connection (idempotent).
	Delete(ctx context.Context, projectID uuid.UUID) error
}
