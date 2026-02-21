# Story feat-1: [BACK] Logout API with token blacklisting

Status: ready-for-dev

## Story

As an authenticated user,
I want a server-side logout endpoint that invalidates my JWT token,
so that logging out prevents the token from being reused even if it was intercepted or copied before expiry.

## Acceptance Criteria (BDD)

**AC1: Token blacklisted on logout**
- **Given** a user is authenticated with a valid JWT in the `token` httpOnly cookie
- **When** they call `POST /api/v1/auth/logout`
- **Then** the server adds the token's JTI (JWT ID) and its expiry to `revoked_tokens`, clears the cookie, and returns `204 No Content`

**AC2: Blacklisted token rejected by auth middleware**
- **Given** a user has logged out (their token's JTI is in `revoked_tokens`)
- **When** they make any authenticated request with that same token cookie
- **Then** the auth middleware returns `401 Unauthorized` with code `TOKEN_REVOKED`

**AC3: No cookie still returns 401**
- **Given** a request with no `token` cookie is sent to `POST /api/v1/auth/logout`
- **Then** the server returns `401 Unauthorized` with code `UNAUTHORIZED` (the route is protected by the existing auth middleware)

**AC4: JWT claims must include a JTI**
- **Given** a user logs in or registers
- **When** a JWT is generated
- **Then** the token's `Claims` struct includes a unique `jti` (JWT ID) field in `RegisteredClaims`, enabling per-token revocation

**AC5: Expired blacklist entries are cleaned up periodically**
- **Given** revoked tokens exist in the `revoked_tokens` table
- **When** their `expires_at` timestamp has passed
- **Then** a periodic background cleanup (running on startup via goroutine) removes them, preventing unbounded table growth

**AC6: Double logout is idempotent**
- **Given** a user has already logged out
- **When** they call `POST /api/v1/auth/logout` again (with the same now-blacklisted token)
- **Then** the middleware rejects the request with `401 TOKEN_REVOKED` (AC2 applies, logout itself is never reached a second time)

## Tasks / Subtasks

- [ ] [BACK] Task 1: Add JTI to JWT generation (AC: #4)
  - [ ] Import `github.com/google/uuid` in `auth_service.go` (already in go.mod)
  - [ ] In `generateToken()`, set `RegisteredClaims.ID` to `uuid.New().String()` before signing
  - [ ] Verify `ValidateToken()` returns claims with a non-empty `ID` field (no change needed, `RegisteredClaims` already carries it)

- [ ] [BACK] Task 2: Create migration `000023_create_revoked_tokens_table` (AC: #1, #2, #5)
  - [ ] Write `000023_create_revoked_tokens_table.up.sql` with the table and index (see Migration SQL Content below)
  - [ ] Write `000023_create_revoked_tokens_table.down.sql`

- [ ] [BACK] Task 3: Add sqlc queries for revoked tokens (AC: #1, #2, #5)
  - [ ] Create `backend/queries/revoked_tokens.sql` with `InsertRevokedToken`, `IsTokenRevoked`, `DeleteExpiredRevokedTokens` (see sqlc Query Signatures below)
  - [ ] Run `cd backend && sqlc generate` to produce the generated Go code

- [ ] [BACK] Task 4: Implement `TokenBlacklistRepository` port and postgres adapter (AC: #1, #2, #5)
  - [ ] Create `backend/internal/domain/port/token_blacklist_repository.go` with the `TokenBlacklistRepository` interface
  - [ ] Create `backend/internal/adapter/postgres/token_blacklist_repo.go` implementing the port using sqlc-generated queries

- [ ] [BACK] Task 5: Extend `AuthService` with `Logout` and `PurgeExpiredTokens` methods (AC: #1, #5)
  - [ ] Add `blacklistRepo port.TokenBlacklistRepository` field to `AuthService`
  - [ ] Add `Logout(ctx context.Context, tokenString string) error` — parses claims, extracts JTI + expiry, calls `blacklistRepo.Revoke`
  - [ ] Add `PurgeExpiredTokens(ctx context.Context) error` — delegates to `blacklistRepo.DeleteExpired`
  - [ ] Add sentinel error `ErrTokenRevoked` for use by middleware

- [ ] [BACK] Task 6: Update auth middleware to check the blacklist (AC: #2, #3)
  - [ ] Add `blacklistRepo port.TokenBlacklistRepository` parameter to `Auth()` function signature
  - [ ] After `authService.ValidateToken(cookie.Value)` succeeds, call `blacklistRepo.IsRevoked(ctx, claims.ID)`
  - [ ] If revoked, call `writeUnauthorized(w)` with body `{"error":{"code":"TOKEN_REVOKED","message":"Token has been revoked"}}` and return

- [ ] [BACK] Task 7: Update `AuthHandler.Logout` to call `authService.Logout` (AC: #1)
  - [ ] Extract token string from the cookie before clearing it
  - [ ] Call `h.authService.Logout(r.Context(), cookie.Value)`
  - [ ] On error, log with slog (warn level) but still clear the cookie and return 204 — logout must always succeed from the client's perspective
  - [ ] Clear the cookie as today (MaxAge: -1)

- [ ] [BACK] Task 8: Start background cleanup goroutine (AC: #5)
  - [ ] In `internal/api/router.go` or `cmd/api/main.go`, after wiring, launch a goroutine that calls `authService.PurgeExpiredTokens` every hour via a `time.Ticker`
  - [ ] Goroutine must respect context cancellation for clean shutdown
  - [ ] Log purge results at debug/info level with `slog`

- [ ] [BACK] Task 9: Wire `TokenBlacklistRepository` into the DI graph (AC: all)
  - [ ] Add `postgres.NewTokenBlacklistRepo` to `wire.go` provider set (or directly pass to `NewAuthService` and `Auth` middleware)
  - [ ] Update `NewAuthService` signature to accept `port.TokenBlacklistRepository`
  - [ ] Update `Auth()` call in router to pass the blacklist repo
  - [ ] Regenerate wire: `cd backend && wire ./cmd/api/`

- [ ] [BACK] Task 10: Tests (AC: #1, #2, #4, #5, #6)
  - [ ] Unit test `AuthService.Logout` with mock `TokenBlacklistRepository`
  - [ ] Unit test auth middleware: token in blacklist → 401 TOKEN_REVOKED
  - [ ] Integration test `TokenBlacklistRepository` against a real Postgres container (testcontainers)
  - [ ] Integration test full flow: login → logout → re-use token → 401

## Dev Notes

### Dependencies

No new Go modules required. All dependencies already in `backend/go.mod`:
- `github.com/golang-jwt/jwt/v5` — JWT parsing (JTI extraction)
- `github.com/google/uuid` — JTI generation
- `github.com/jackc/pgx/v5` — Postgres driver
- sqlc for query generation

### File Paths (exact)

| File | Action |
|------|--------|
| `backend/migrations/000023_create_revoked_tokens_table.up.sql` | Create |
| `backend/migrations/000023_create_revoked_tokens_table.down.sql` | Create |
| `backend/queries/revoked_tokens.sql` | Create |
| `backend/internal/domain/port/token_blacklist_repository.go` | Create |
| `backend/internal/adapter/postgres/token_blacklist_repo.go` | Create |
| `backend/internal/domain/service/auth_service.go` | Modify |
| `backend/internal/api/middleware/auth.go` | Modify |
| `backend/internal/api/handler/auth_handler.go` | Modify |
| `backend/internal/api/router.go` | Modify |
| `backend/cmd/api/wire.go` | Modify |
| `backend/cmd/api/wire_gen.go` | Regenerate (never edit manually) |
| `backend/internal/adapter/postgres/revoked_tokens.sql.go` | Regenerate via sqlc |

### Migration SQL Content

**`000023_create_revoked_tokens_table.up.sql`:**

```sql
-- Stores revoked JWT IDs (JTI) to prevent reuse after logout.
-- Entries are cleaned up once expires_at has passed.
CREATE TABLE revoked_tokens (
    jti        TEXT        NOT NULL PRIMARY KEY,  -- JWT ID claim (uuid string)
    expires_at TIMESTAMPTZ NOT NULL               -- copied from JWT exp claim
);

-- Index for fast lookup in auth middleware hot path
CREATE INDEX idx_revoked_tokens_expires_at ON revoked_tokens (expires_at);
```

**`000023_create_revoked_tokens_table.down.sql`:**

```sql
DROP TABLE IF EXISTS revoked_tokens;
```

### sqlc Query Signatures

**`backend/queries/revoked_tokens.sql`:**

```sql
-- name: InsertRevokedToken :exec
INSERT INTO revoked_tokens (jti, expires_at)
VALUES ($1, $2)
ON CONFLICT (jti) DO NOTHING;

-- name: IsTokenRevoked :one
SELECT EXISTS (
    SELECT 1 FROM revoked_tokens WHERE jti = $1
) AS revoked;

-- name: DeleteExpiredRevokedTokens :exec
DELETE FROM revoked_tokens WHERE expires_at < now();
```

After writing this file, run: `cd backend && sqlc generate`

This produces `backend/internal/adapter/postgres/revoked_tokens.sql.go` with:
- `InsertRevokedToken(ctx, InsertRevokedTokenParams{Jti string, ExpiresAt pgtype.Timestamptz})`
- `IsTokenRevoked(ctx, jti string) (bool, error)`
- `DeleteExpiredRevokedTokens(ctx) error`

### Domain Model / Port changes

**`backend/internal/domain/port/token_blacklist_repository.go`:**

```go
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
```

**`backend/internal/adapter/postgres/token_blacklist_repo.go`** (skeleton — implement from sqlc output):

```go
package postgres

import (
    "context"
    "time"

    "github.com/jackc/pgx/v5/pgtype"
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

func (r *TokenBlacklistRepo) Revoke(ctx context.Context, jti string, expiresAt time.Time) error {
    err := r.q.InsertRevokedToken(ctx, InsertRevokedTokenParams{
        Jti:       jti,
        ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
    })
    if err != nil {
        return apperrors.NewInternal("failed to revoke token", err)
    }
    return nil
}

func (r *TokenBlacklistRepo) IsRevoked(ctx context.Context, jti string) (bool, error) {
    revoked, err := r.q.IsTokenRevoked(ctx, jti)
    if err != nil {
        return false, apperrors.NewInternal("failed to check token revocation", err)
    }
    return revoked, nil
}

func (r *TokenBlacklistRepo) DeleteExpired(ctx context.Context) error {
    if err := r.q.DeleteExpiredRevokedTokens(ctx); err != nil {
        return apperrors.NewInternal("failed to delete expired revoked tokens", err)
    }
    return nil
}
```

### AuthService changes

**Add to `service/auth_service.go`:**

New sentinel error:
```go
var ErrTokenRevoked = errors.New("token has been revoked")
```

Updated struct (add blacklistRepo field):
```go
type AuthService struct {
    repo          port.UserRepository
    blacklistRepo port.TokenBlacklistRepository
    jwtSecret     []byte
    jwtExpiration time.Duration
}

func NewAuthService(
    repo port.UserRepository,
    blacklistRepo port.TokenBlacklistRepository,
    jwtSecret string,
    jwtExpiration time.Duration,
) *AuthService {
    return &AuthService{
        repo:          repo,
        blacklistRepo: blacklistRepo,
        jwtSecret:     []byte(jwtSecret),
        jwtExpiration: jwtExpiration,
    }
}
```

Updated `generateToken` — add JTI:
```go
func (s *AuthService) generateToken(userID uuid.UUID, role model.Role) (string, error) {
    now := time.Now()
    claims := &Claims{
        UserID: userID,
        Role:   role,
        RegisteredClaims: jwt.RegisteredClaims{
            ID:        uuid.New().String(), // JTI — unique per token
            ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtExpiration)),
            IssuedAt:  jwt.NewNumericDate(now),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.jwtSecret)
}
```

New `Logout` method:
```go
// Logout invalidates the given JWT token string by adding its JTI to the blacklist.
func (s *AuthService) Logout(ctx context.Context, tokenString string) error {
    claims, err := s.ValidateToken(tokenString)
    if err != nil {
        // Token already expired or invalid — nothing to revoke.
        return nil
    }
    jti := claims.ID
    if jti == "" {
        return nil // legacy token without JTI — skip
    }
    expiresAt := claims.ExpiresAt.Time
    return s.blacklistRepo.Revoke(ctx, jti, expiresAt)
}

// PurgeExpiredTokens removes expired entries from the token blacklist.
func (s *AuthService) PurgeExpiredTokens(ctx context.Context) error {
    return s.blacklistRepo.DeleteExpired(ctx)
}
```

### Auth Middleware changes

Updated `Auth()` signature in `backend/internal/api/middleware/auth.go`:

```go
// Auth returns middleware that validates JWT tokens, checks the blacklist, and injects user context.
func Auth(authService *service.AuthService, blacklistRepo port.TokenBlacklistRepository) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if isPublicPath(r.URL.Path) {
                next.ServeHTTP(w, r)
                return
            }

            cookie, err := r.Cookie("token")
            if err != nil {
                writeUnauthorized(w)
                return
            }

            claims, err := authService.ValidateToken(cookie.Value)
            if err != nil {
                writeUnauthorized(w)
                return
            }

            // Check blacklist (token revoked via logout)
            if claims.ID != "" {
                revoked, err := blacklistRepo.IsRevoked(r.Context(), claims.ID)
                if err != nil {
                    writeUnauthorized(w)
                    return
                }
                if revoked {
                    writeRevokedToken(w)
                    return
                }
            }

            ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
            ctx = context.WithValue(ctx, ContextKeyRole, claims.Role)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func writeRevokedToken(w http.ResponseWriter) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusUnauthorized)
    _, _ = w.Write([]byte(`{"error":{"code":"TOKEN_REVOKED","message":"Token has been revoked"}}`))
}
```

Import to add: `"github.com/zakari/hopeitworks/backend/internal/domain/port"`

### AuthHandler.Logout changes

Updated `Logout` in `backend/internal/api/handler/auth_handler.go`:

```go
// Logout handles POST /auth/logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
    cookie, err := r.Cookie("token")
    if err == nil && cookie.Value != "" {
        // Best-effort blacklist — always clear cookie regardless of outcome
        if logoutErr := h.authService.Logout(r.Context(), cookie.Value); logoutErr != nil {
            // Log but do not fail the request
            slog.WarnContext(r.Context(), "failed to blacklist token on logout",
                "error", logoutErr,
            )
        }
    }
    http.SetCookie(w, &http.Cookie{
        Name:     "token",
        Value:    "",
        Path:     "/api",
        HttpOnly: true,
        Secure:   h.cookieSecure,
        SameSite: http.SameSiteLaxMode,
        MaxAge:   -1,
    })
    w.WriteHeader(http.StatusNoContent)
}
```

Import to add: `"log/slog"`

### Background cleanup goroutine

In `backend/internal/api/router.go` or wherever the app lifecycle is managed (add after wiring, before `ListenAndServe`). Preferred location is a new `StartBackgroundJobs(ctx context.Context, authService *service.AuthService, logger *slog.Logger)` function called from `main.go`:

```go
// StartBackgroundJobs launches recurring maintenance tasks.
// Call this once from main after the server starts. The goroutine stops when ctx is cancelled.
func StartBackgroundJobs(ctx context.Context, authService *service.AuthService, logger *slog.Logger) {
    go func() {
        ticker := time.NewTicker(1 * time.Hour)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                if err := authService.PurgeExpiredTokens(ctx); err != nil {
                    logger.Warn("failed to purge expired revoked tokens", "error", err)
                } else {
                    logger.Info("purged expired revoked tokens")
                }
            }
        }
    }()
}
```

### OpenAPI spec changes

The `POST /auth/logout` endpoint already exists in `api/openapi.yaml` (lines 103-112). No schema changes are needed — the endpoint has no request body and returns 204 / 401. The blacklisting is a server-side implementation detail invisible to the spec.

No regeneration of backend types (`make generate`) is needed for this story.

### Error Responses

| Situation | HTTP | Code | Message |
|-----------|------|------|---------|
| No cookie on any protected route | 401 | `UNAUTHORIZED` | `Authentication required` |
| Token signature invalid / expired | 401 | `UNAUTHORIZED` | `Authentication required` |
| Token JTI present in `revoked_tokens` | 401 | `TOKEN_REVOKED` | `Token has been revoked` |
| Blacklist DB error during middleware check | 401 | `UNAUTHORIZED` | `Authentication required` |
| Blacklist DB error during logout | 204 | — | Cookie cleared, error logged only |

### Testing Requirements

**Unit tests** (`-short` compatible, no containers):

1. `TestAuthService_Logout_RevokesToken` — mock `TokenBlacklistRepository.Revoke` called with correct JTI and expiry
2. `TestAuthService_Logout_InvalidToken_Noop` — expired/invalid token string causes no call to `Revoke`
3. `TestAuthService_Logout_EmptyJTI_Noop` — legacy token with empty JTI causes no call to `Revoke`
4. `TestAuthMiddleware_RevokedToken_Returns401` — middleware with mock blacklist returning `true` → 401 `TOKEN_REVOKED`
5. `TestAuthMiddleware_ValidToken_NotRevoked_Passes` — mock returns `false` → handler called
6. `TestAuthService_GenerateToken_HasJTI` — generated token always has non-empty `claims.ID`

**Integration tests** (`run Integration`, testcontainers):

7. `TestTokenBlacklistRepo_Integration` — `Revoke` then `IsRevoked` returns true; `DeleteExpired` removes it after advancing time
8. `TestLogoutFlow_Integration` — full HTTP cycle: login → get cookie → logout → replay cookie → assert 401 `TOKEN_REVOKED`

Test file locations:
- `backend/internal/domain/service/auth_service_test.go` (extend existing or create)
- `backend/internal/api/middleware/auth_test.go` (extend existing or create)
- `backend/internal/adapter/postgres/token_blacklist_repo_integration_test.go` (new)

## Dev Agent Record

_To be filled by the implementing agent._

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-21 | zakari | Initial story draft |
