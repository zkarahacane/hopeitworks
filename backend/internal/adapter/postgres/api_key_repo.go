package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure APIKeyRepo implements port.APIKeyRepository at compile time.
var _ port.APIKeyRepository = (*APIKeyRepo)(nil)

// APIKeyRepo implements port.APIKeyRepository using sqlc-generated queries.
type APIKeyRepo struct {
	queries *Queries
}

// NewAPIKeyRepository creates a new APIKeyRepo.
func NewAPIKeyRepository(queries *Queries) *APIKeyRepo {
	return &APIKeyRepo{queries: queries}
}

// Create inserts a new user API key.
func (r *APIKeyRepo) Create(ctx context.Context, key *model.UserAPIKey) error {
	params := CreateUserAPIKeyParams{
		ID:           key.ID,
		UserID:       key.UserID,
		Provider:     key.Provider,
		KeyName:      key.KeyName,
		EncryptedKey: key.EncryptedKey,
		KeyHint:      key.KeyHint,
	}

	row, err := r.queries.CreateUserAPIKey(ctx, params)
	if err != nil {
		if isUniqueViolation(err) {
			return apperrors.NewConflict("api_key", key.Provider+"/"+key.KeyName)
		}
		if isForeignKeyViolation(err) {
			return apperrors.NewNotFound("user", key.UserID)
		}
		return apperrors.NewInternal("failed to create API key", err)
	}

	key.CreatedAt = row.CreatedAt
	key.UpdatedAt = row.UpdatedAt
	return nil
}

// GetByID retrieves an API key by ID.
func (r *APIKeyRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.UserAPIKey, error) {
	row, err := r.queries.GetUserAPIKey(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("api_key", id)
		}
		return nil, apperrors.NewInternal("failed to get API key", err)
	}
	return toDomainAPIKey(row), nil
}

// ListByUser retrieves all API keys for a user.
func (r *APIKeyRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*model.UserAPIKey, error) {
	rows, err := r.queries.ListUserAPIKeys(ctx, userID)
	if err != nil {
		return nil, apperrors.NewInternal("failed to list API keys", err)
	}
	keys := make([]*model.UserAPIKey, len(rows))
	for i, row := range rows {
		keys[i] = toDomainAPIKey(row)
	}
	return keys, nil
}

// GetByUserAndProvider retrieves an API key by user ID and provider.
func (r *APIKeyRepo) GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*model.UserAPIKey, error) {
	row, err := r.queries.GetUserAPIKeyByProvider(ctx, GetUserAPIKeyByProviderParams{
		UserID:   userID,
		Provider: provider,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("api_key", provider)
		}
		return nil, apperrors.NewInternal("failed to get API key by provider", err)
	}
	return toDomainAPIKey(row), nil
}

// Delete removes an API key by ID.
func (r *APIKeyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.queries.DeleteUserAPIKey(ctx, id)
	if err != nil {
		return apperrors.NewInternal("failed to delete API key", err)
	}
	return nil
}

// toDomainAPIKey maps a sqlc-generated UserApiKey to a domain UserAPIKey.
func toDomainAPIKey(k UserApiKey) *model.UserAPIKey {
	return &model.UserAPIKey{
		ID:           k.ID,
		UserID:       k.UserID,
		Provider:     k.Provider,
		KeyName:      k.KeyName,
		EncryptedKey: k.EncryptedKey,
		KeyHint:      k.KeyHint,
		CreatedAt:    k.CreatedAt,
		UpdatedAt:    k.UpdatedAt,
	}
}
