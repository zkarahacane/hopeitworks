package model

import (
	"time"

	"github.com/google/uuid"
)

// UserAPIKey represents an encrypted API key stored for a user.
type UserAPIKey struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Provider     string
	KeyName      string
	EncryptedKey []byte
	KeyHint      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
