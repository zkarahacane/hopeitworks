package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/crypto"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// CredentialService encrypts, stores, and resolves named secrets using AES-256-GCM
// (the same crypto as user API keys). Plaintext only ever exists transiently while a
// bundle is assembled for an authenticated container fetch.
type CredentialService struct {
	repo          port.CredentialRepository
	encryptionKey []byte
}

// NewCredentialService creates a new CredentialService.
func NewCredentialService(repo port.CredentialRepository, masterKey string) *CredentialService {
	return &CredentialService{
		repo:          repo,
		encryptionKey: crypto.DeriveKey(masterKey),
	}
}

// CreateGlobal encrypts and stores a global-scoped credential.
func (s *CredentialService) CreateGlobal(ctx context.Context, name, rawValue string) (*model.Credential, error) {
	return s.create(ctx, name, model.CapabilityScopeGlobal, nil, rawValue)
}

// CreateProject encrypts and stores a project-scoped credential.
func (s *CredentialService) CreateProject(ctx context.Context, name string, projectID uuid.UUID, rawValue string) (*model.Credential, error) {
	return s.create(ctx, name, model.CapabilityScopeProject, &projectID, rawValue)
}

func (s *CredentialService) create(ctx context.Context, name, scope string, projectID *uuid.UUID, rawValue string) (*model.Credential, error) {
	if name == "" {
		return nil, errors.NewValidation("name", "is required")
	}
	if rawValue == "" {
		return nil, errors.NewValidation("value", "is required")
	}

	encrypted, err := crypto.Encrypt([]byte(rawValue), s.encryptionKey)
	if err != nil {
		return nil, errors.NewInternal("failed to encrypt credential", err)
	}

	cred := &model.Credential{
		ID:             uuid.New(),
		Name:           name,
		Scope:          scope,
		ProjectID:      projectID,
		EncryptedValue: encrypted,
	}
	return s.repo.Create(ctx, cred)
}

// Resolve returns the decrypted value of the credential named `name`. When projectID is
// non-nil it prefers the project-scoped credential and falls back to the global one;
// otherwise it resolves only the global credential. Returns a not-found DomainError when
// no credential matches — callers (e.g. bundle assembly) warn+skip on that.
func (s *CredentialService) Resolve(ctx context.Context, name string, projectID *uuid.UUID) (string, error) {
	var cred *model.Credential

	if projectID != nil {
		pc, err := s.repo.GetProjectByName(ctx, name, *projectID)
		switch {
		case err == nil:
			cred = pc
		case isNotFound(err):
			// fall through to the global lookup
		default:
			return "", err
		}
	}

	if cred == nil {
		gc, err := s.repo.GetGlobalByName(ctx, name)
		if err != nil {
			return "", err
		}
		cred = gc
	}

	plaintext, err := crypto.Decrypt(cred.EncryptedValue, s.encryptionKey)
	if err != nil {
		return "", errors.NewInternal("failed to decrypt credential", err)
	}
	return string(plaintext), nil
}
