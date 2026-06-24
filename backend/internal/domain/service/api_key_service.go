package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/crypto"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// APIKeyService provides business logic for user API key operations.
type APIKeyService struct {
	repo          port.APIKeyRepository
	encryptionKey []byte
}

// NewAPIKeyService creates a new APIKeyService.
func NewAPIKeyService(repo port.APIKeyRepository, masterKey string) *APIKeyService {
	return &APIKeyService{
		repo:          repo,
		encryptionKey: crypto.DeriveKey(masterKey),
	}
}

// CreateKey encrypts and stores a new API key for a user.
func (s *APIKeyService) CreateKey(ctx context.Context, userID uuid.UUID, provider, keyName, rawKey string) (*model.UserAPIKey, error) {
	if provider == "" {
		return nil, errors.NewValidation("provider", "is required")
	}
	if keyName == "" {
		return nil, errors.NewValidation("key_name", "is required")
	}
	if rawKey == "" {
		return nil, errors.NewValidation("api_key", "is required")
	}

	// Generate hint: last 4 chars prefixed with "..."
	hint := generateHint(rawKey)

	// Encrypt the raw key
	encrypted, err := crypto.Encrypt([]byte(rawKey), s.encryptionKey)
	if err != nil {
		return nil, errors.NewInternal("failed to encrypt API key", err)
	}

	key := &model.UserAPIKey{
		ID:           uuid.New(),
		UserID:       userID,
		Provider:     provider,
		KeyName:      keyName,
		EncryptedKey: encrypted,
		KeyHint:      hint,
	}

	if err := s.repo.Create(ctx, key); err != nil {
		return nil, err
	}

	return key, nil
}

// DecryptKey retrieves and decrypts an API key by ID.
func (s *APIKeyService) DecryptKey(ctx context.Context, id uuid.UUID) (string, error) {
	key, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	plaintext, err := crypto.Decrypt(key.EncryptedKey, s.encryptionKey)
	if err != nil {
		return "", errors.NewInternal("failed to decrypt API key", err)
	}

	return string(plaintext), nil
}

// DecryptKeyForUserProvider retrieves and decrypts the API key for a specific user and provider.
func (s *APIKeyService) DecryptKeyForUserProvider(ctx context.Context, userID uuid.UUID, provider string) (string, error) {
	key, err := s.repo.GetByUserAndProvider(ctx, userID, provider)
	if err != nil {
		return "", err
	}

	plaintext, err := crypto.Decrypt(key.EncryptedKey, s.encryptionKey)
	if err != nil {
		return "", errors.NewInternal("failed to decrypt API key", err)
	}

	return string(plaintext), nil
}

// ListKeys returns all API keys for a user (hint only, never decrypted).
func (s *APIKeyService) ListKeys(ctx context.Context, userID uuid.UUID) ([]*model.UserAPIKey, error) {
	return s.repo.ListByUser(ctx, userID)
}

// DeleteKey removes an API key owned by the given user. The deletion is scoped
// to userID, so a user can never delete another user's key by guessing its UUID.
// A key that is absent or owned by someone else is a no-op: the call is
// idempotent (a second delete of the same id never errors), and it leaks no
// information about keys the caller does not own.
func (s *APIKeyService) DeleteKey(ctx context.Context, userID, id uuid.UUID) error {
	key, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if domErr, ok := err.(*errors.DomainError); ok && domErr.Category == errors.CategoryNotFound {
			return nil
		}
		return err
	}
	if key.UserID != userID {
		return nil
	}
	return s.repo.Delete(ctx, id)
}

// generateHint returns the last 4 characters of the key prefixed with "...".
// If the key is shorter than 4 characters, it returns "..."+key.
func generateHint(rawKey string) string {
	if len(rawKey) <= 4 {
		return fmt.Sprintf("...%s", rawKey)
	}
	return fmt.Sprintf("...%s", rawKey[len(rawKey)-4:])
}
