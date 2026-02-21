package port

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// PasswordResetTokenRepository defines persistence operations for password reset tokens.
type PasswordResetTokenRepository interface {
	Create(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) (*model.PasswordResetToken, error)
	GetByToken(ctx context.Context, token string) (*model.PasswordResetToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
}
