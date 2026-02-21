package port

import (
	"context"
	"time"
)

// TokenBlacklistRepository manages revoked JWT IDs.
type TokenBlacklistRepository interface {
	// Revoke adds a token's JTI to the blacklist until expiresAt.
	Revoke(ctx context.Context, jti string, expiresAt time.Time) error
	// IsRevoked returns true if the JTI is in the blacklist.
	IsRevoked(ctx context.Context, jti string) (bool, error)
	// DeleteExpired removes all entries whose expiresAt has passed.
	DeleteExpired(ctx context.Context) error
}
