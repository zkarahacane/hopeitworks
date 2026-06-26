package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure GitConnectionRepo implements port.GitConnectionRepository at compile time.
var _ port.GitConnectionRepository = (*GitConnectionRepo)(nil)

// GitConnectionRepo persists one PAT connection per project using sqlc queries.
// It only ever stores/returns ciphertext; encryption lives in GitConnectionService.
type GitConnectionRepo struct {
	queries *Queries
}

// NewGitConnectionRepository creates a new GitConnectionRepo.
func NewGitConnectionRepository(queries *Queries) *GitConnectionRepo {
	return &GitConnectionRepo{queries: queries}
}

// GetByProject returns the project's connection, or a not-found DomainError.
func (r *GitConnectionRepo) GetByProject(ctx context.Context, projectID uuid.UUID) (*model.GitConnection, error) {
	row, err := r.queries.GetGitConnectionByProject(ctx, projectID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("git_connection", projectID)
		}
		return nil, apperrors.NewInternal("failed to get git connection", err)
	}
	return toDomainGitConnection(row), nil
}

// Upsert inserts or replaces the project's connection, returning the stored row.
func (r *GitConnectionRepo) Upsert(ctx context.Context, p port.UpsertGitConnectionParams) (*model.GitConnection, error) {
	row, err := r.queries.UpsertGitConnectionPAT(ctx, UpsertGitConnectionPATParams{
		ProjectID:       p.ProjectID,
		Provider:        p.Provider,
		EncryptedSecret: p.EncryptedSecret,
		SecretLast4:     textFromStringPtr(p.SecretLast4),
		TokenType:       textFromStringPtr(p.TokenType),
		Scopes:          nonNilStrings(p.Scopes),
		Status:          string(p.Status),
		AccountLogin:    textFromStringPtr(p.AccountLogin),
		ExpiresAt:       timestamptzFromTimePtr(p.ExpiresAt),
		LastValidatedAt: timestamptzFromTimePtr(p.LastValidatedAt),
		ValidationError: textFromStringPtr(p.ValidationError),
	})
	if err != nil {
		if isForeignKeyViolation(err) {
			return nil, apperrors.NewNotFound("project", p.ProjectID)
		}
		return nil, apperrors.NewInternal("failed to upsert git connection", err)
	}
	return toDomainGitConnection(row), nil
}

// SetValidation refreshes validation metadata in place (no-op if absent).
func (r *GitConnectionRepo) SetValidation(ctx context.Context, p port.SetValidationParams) error {
	if err := r.queries.SetGitConnectionValidation(ctx, SetGitConnectionValidationParams{
		ProjectID:       p.ProjectID,
		Status:          string(p.Status),
		AccountLogin:    textFromStringPtr(p.AccountLogin),
		Scopes:          nonNilStrings(p.Scopes),
		ExpiresAt:       timestamptzFromTimePtr(p.ExpiresAt),
		ValidationError: textFromStringPtr(p.ValidationError),
	}); err != nil {
		return apperrors.NewInternal("failed to set git connection validation", err)
	}
	return nil
}

// MarkStatus flips status + validation_error in place, preserving other fields.
func (r *GitConnectionRepo) MarkStatus(ctx context.Context, projectID uuid.UUID, status model.GitConnectionStatus, validationError *string) error {
	if err := r.queries.MarkGitConnectionStatus(ctx, MarkGitConnectionStatusParams{
		ProjectID:       projectID,
		Status:          string(status),
		ValidationError: textFromStringPtr(validationError),
	}); err != nil {
		return apperrors.NewInternal("failed to mark git connection status", err)
	}
	return nil
}

// Delete removes the project's connection (idempotent).
func (r *GitConnectionRepo) Delete(ctx context.Context, projectID uuid.UUID) error {
	if err := r.queries.DeleteGitConnectionByProject(ctx, projectID); err != nil {
		return apperrors.NewInternal("failed to delete git connection", err)
	}
	return nil
}

// toDomainGitConnection maps a sqlc-generated row to the domain model.
func toDomainGitConnection(c GitConnection) *model.GitConnection {
	return &model.GitConnection{
		ID:              c.ID,
		ProjectID:       c.ProjectID,
		Provider:        c.Provider,
		Kind:            model.GitConnectionKind(c.Kind),
		EncryptedSecret: c.EncryptedSecret,
		SecretLast4:     stringPtrFromText(c.SecretLast4),
		TokenType:       stringPtrFromText(c.TokenType),
		Scopes:          c.Scopes,
		Status:          model.GitConnectionStatus(c.Status),
		AccountLogin:    stringPtrFromText(c.AccountLogin),
		ExpiresAt:       timeFromTimestamptz(c.ExpiresAt),
		LastValidatedAt: timeFromTimestamptz(c.LastValidatedAt),
		ValidationError: stringPtrFromText(c.ValidationError),
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
}

// stringPtrFromText converts a nullable Postgres text to *string (nil for NULL).
func stringPtrFromText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	v := t.String
	return &v
}

// timestamptzFromTimePtr converts *time.Time to a nullable Postgres timestamptz.
func timestamptzFromTimePtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// nonNilStrings ensures a nil slice is stored as an empty array (column is NOT NULL).
func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
