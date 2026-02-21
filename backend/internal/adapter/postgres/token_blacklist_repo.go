package postgres

import (
	"context"
	"time"

	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

var _ port.TokenBlacklistRepository = (*TokenBlacklistRepo)(nil)

// TokenBlacklistRepo implements port.TokenBlacklistRepository using sqlc.
type TokenBlacklistRepo struct {
	q *Queries
}

// NewTokenBlacklistRepo creates a new TokenBlacklistRepo.
func NewTokenBlacklistRepo(db DBTX) *TokenBlacklistRepo {
	return &TokenBlacklistRepo{q: New(db)}
}

// Revoke adds a token's JTI to the blacklist until expiresAt.
func (r *TokenBlacklistRepo) Revoke(ctx context.Context, jti string, expiresAt time.Time) error {
	err := r.q.InsertRevokedToken(ctx, InsertRevokedTokenParams{
		Jti:       jti,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return apperrors.NewInternal("failed to revoke token", err)
	}
	return nil
}

// IsRevoked returns true if the JTI is in the blacklist.
func (r *TokenBlacklistRepo) IsRevoked(ctx context.Context, jti string) (bool, error) {
	revoked, err := r.q.IsTokenRevoked(ctx, jti)
	if err != nil {
		return false, apperrors.NewInternal("failed to check token revocation", err)
	}
	return revoked, nil
}

// DeleteExpired removes all entries whose expiresAt has passed.
func (r *TokenBlacklistRepo) DeleteExpired(ctx context.Context) error {
	if err := r.q.DeleteExpiredRevokedTokens(ctx); err != nil {
		return apperrors.NewInternal("failed to delete expired revoked tokens", err)
	}
	return nil
}
