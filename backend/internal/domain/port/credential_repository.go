package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// CredentialRepository persists encrypted, named secrets. The stored EncryptedValue is
// opaque ciphertext; encryption/decryption lives in the CredentialService, never here.
type CredentialRepository interface {
	Create(ctx context.Context, c *model.Credential) (*model.Credential, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Credential, error)
	GetGlobalByName(ctx context.Context, name string) (*model.Credential, error)
	GetProjectByName(ctx context.Context, name string, projectID uuid.UUID) (*model.Credential, error)
	// ListByScope returns all global credentials plus, when projectID is non-nil, the
	// credentials owned by that project.
	ListByScope(ctx context.Context, projectID *uuid.UUID) ([]*model.Credential, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
