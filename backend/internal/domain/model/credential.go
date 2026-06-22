package model

import (
	"time"

	"github.com/google/uuid"
)

// Credential is a named secret encrypted at rest with AES-256-GCM (pkg/crypto, the
// same scheme as user API keys). EncryptedValue holds nonce+ciphertext+tag. The
// plaintext is produced only when assembling a RuntimeBundle for an authenticated
// container fetch — never persisted in clear, never logged, never placed in the
// container env (so it cannot leak via `docker inspect`).
type Credential struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Scope          string     `json:"scope"` // CapabilityScope* (global | project)
	ProjectID      *uuid.UUID `json:"project_id"`
	EncryptedValue []byte     `json:"-"` // never serialised over the API
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
