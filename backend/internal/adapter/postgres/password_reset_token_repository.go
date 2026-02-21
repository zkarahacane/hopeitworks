package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// PasswordResetTokenRepository implements port.PasswordResetTokenRepository using sqlc.
type PasswordResetTokenRepository struct {
	q *Queries
}

var _ port.PasswordResetTokenRepository = (*PasswordResetTokenRepository)(nil)

// NewPasswordResetTokenRepository creates a new PasswordResetTokenRepository.
func NewPasswordResetTokenRepository(db DBTX) *PasswordResetTokenRepository {
	return &PasswordResetTokenRepository{q: New(db)}
}

// Create persists a new password reset token.
func (r *PasswordResetTokenRepository) Create(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) (*model.PasswordResetToken, error) {
	row, err := r.q.CreatePasswordResetToken(ctx, CreatePasswordResetTokenParams{
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, apperrors.NewInternal("create password reset token", err)
	}
	return toDomainPasswordResetToken(row), nil
}

// GetByToken retrieves a password reset token by its token string.
func (r *PasswordResetTokenRepository) GetByToken(ctx context.Context, token string) (*model.PasswordResetToken, error) {
	row, err := r.q.GetPasswordResetToken(ctx, token)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewNotFound("password_reset_token", token)
		}
		return nil, apperrors.NewInternal("get password reset token", err)
	}
	return toDomainPasswordResetToken(row), nil
}

// MarkUsed sets the used_at timestamp for the given token ID.
func (r *PasswordResetTokenRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	if err := r.q.MarkPasswordResetTokenUsed(ctx, id); err != nil {
		return apperrors.NewInternal("mark password reset token used", err)
	}
	return nil
}

func toDomainPasswordResetToken(row PasswordResetToken) *model.PasswordResetToken {
	var usedAt *time.Time
	if row.UsedAt.Valid {
		usedAt = &row.UsedAt.Time
	}
	return &model.PasswordResetToken{
		ID:        row.ID,
		UserID:    row.UserID,
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt,
		UsedAt:    usedAt,
		CreatedAt: row.CreatedAt,
	}
}
