package model

import (
	"time"

	"github.com/google/uuid"
)

// PasswordResetToken represents a one-time token for resetting a user's password.
type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// IsExpired returns true if the token is past its expiry time.
func (t *PasswordResetToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsUsed returns true if the token has already been consumed.
func (t *PasswordResetToken) IsUsed() bool {
	return t.UsedAt != nil
}
