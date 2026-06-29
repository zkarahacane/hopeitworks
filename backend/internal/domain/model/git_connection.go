package model

import (
	"time"

	"github.com/google/uuid"
)

// GitConnectionKind is the credential mechanism a connection uses. v1 ships PAT
// only; the column is a forward hedge (DB CHECK pins it to 'pat').
type GitConnectionKind string

const (
	// GitConnectionKindPAT stores an encrypted Personal Access Token at rest.
	GitConnectionKindPAT GitConnectionKind = "pat"
)

// GitConnectionStatus is the advisory, last-known connection state. GitHub is the
// source of truth; this value is refreshed by "Test connection" and self-heals on
// real operation errors (401 -> invalid, 403 -> insufficient_scope).
type GitConnectionStatus string

const (
	// GitConnStatusUnconfigured means no usable connection is stored.
	GitConnStatusUnconfigured GitConnectionStatus = "unconfigured"
	// GitConnStatusConnected means the last probe succeeded with sufficient scope.
	GitConnStatusConnected GitConnectionStatus = "connected"
	// GitConnStatusInvalid means the provider rejected the token (HTTP 401).
	GitConnStatusInvalid GitConnectionStatus = "invalid"
	// GitConnStatusExpired means expires_at is known and in the past (lazy).
	GitConnStatusExpired GitConnectionStatus = "expired"
	// GitConnStatusInsufficient means the token is missing a required scope (HTTP 403 / missing read:project).
	GitConnStatusInsufficient GitConnectionStatus = "insufficient_scope"
)

// GitConnectionTokenType classifies a PAT from its prefix.
type GitConnectionTokenType string

const (
	// GitTokenTypeClassic is a classic PAT (ghp_...). Returns X-OAuth-Scopes.
	GitTokenTypeClassic GitConnectionTokenType = "classic"
	// GitTokenTypeFineGrained is a fine-grained PAT (github_pat_...). No scopes header.
	GitTokenTypeFineGrained GitConnectionTokenType = "fine_grained"
	// GitTokenTypeUnknown is any other token shape.
	GitTokenTypeUnknown GitConnectionTokenType = "unknown"
)

// Validation error codes. These are FIXED codes persisted in validation_error;
// raw upstream/probe text is NEVER persisted (security hardening A3).
const (
	ValidationErrUnauthorized      = "unauthorized"
	ValidationErrInsufficientScope = "insufficient_scope"
	ValidationErrProbe5xx          = "probe_5xx"
	ValidationErrDNS               = "dns_error"
	ValidationErrTLS               = "tls_error"
	ValidationErrDecryptFailed     = "decrypt_failed"
	ValidationErrExpired           = "expired"
)

// GitConnection is a project's connection to its git host. EncryptedSecret holds
// nonce+ciphertext+tag and is NEVER serialised (json:"-"), logged, or returned by
// the API.
type GitConnection struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	Provider  string
	Kind      GitConnectionKind

	EncryptedSecret []byte `json:"-"` // kind=pat
	SecretLast4     *string
	TokenType       *string
	Scopes          []string

	Status          GitConnectionStatus
	AccountLogin    *string
	ExpiresAt       *time.Time
	LastValidatedAt *time.Time
	ValidationError *string

	CreatedAt time.Time
	UpdatedAt time.Time
}
