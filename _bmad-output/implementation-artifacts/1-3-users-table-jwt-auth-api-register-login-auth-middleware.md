# Story 1.3: Users table + JWT auth API (register, login, auth middleware)

Status: ready-for-dev

## Story

As a user,
I want a users table and JWT authentication,
so that user data is persisted and I can securely access protected endpoints.

## Acceptance Criteria (BDD)

**AC1: Users table migration**
- **Given** migration 000001 exists
- **When** migrations are applied
- **Then** a users table is created with: id (UUID PK), email (unique), password_hash, name, role (admin/user default 'user'), created_at, updated_at

**AC2: sqlc query generation**
- **Given** sqlc queries are defined in `backend/queries/users.sql`
- **When** I run `make generate`
- **Then** Go functions for CreateUser, GetUserByEmail, GetUserByID, ListUsers, UpdateUser, DeleteUser are generated

**AC3: Register endpoint**
- **Given** the API is running
- **When** I POST `/api/v1/auth/register` with valid email, password, and name
- **Then** I receive HTTP 201 with user object and password is bcrypt-hashed in the database

**AC4: Login success**
- **Given** a registered user exists
- **When** I POST `/api/v1/auth/login` with correct credentials
- **Then** I receive HTTP 200 with user object and JWT in httpOnly secure cookie containing user_id, role, exp

**AC5: Login failure**
- **Given** a registered user exists
- **When** I POST `/api/v1/auth/login` with wrong password
- **Then** I receive HTTP 401

**AC6: Auth middleware injects context**
- **Given** a request includes a valid JWT cookie
- **When** auth middleware runs
- **Then** user context (id, role) is injected into request context

**AC7: Auth middleware rejects invalid token**
- **Given** a request has no JWT or invalid JWT
- **When** auth middleware runs
- **Then** HTTP 401 is returned

## Tasks / Subtasks

- [ ] [BACK] Task 1: Create users table migration (AC: #1)
  - [ ] Create `backend/migrations/000001_create_users_table.up.sql`
  - [ ] Create `backend/migrations/000001_create_users_table.down.sql`
  - [ ] Define users table: id (UUID PK default gen_random_uuid()), email (unique not null), password_hash (text not null), name (text not null), role (text not null default 'user' check in ('admin','user')), created_at (timestamptz default now()), updated_at (timestamptz default now())
  - [ ] Add index on email for fast lookup
  - [ ] Add trigger for auto-updating updated_at on row change

- [ ] [BACK] Task 2: Create sqlc query file for users (AC: #2)
  - [ ] Create `backend/queries/users.sql` with all 6 named queries
  - [ ] CreateUser: INSERT returning full row
  - [ ] GetUserByEmail: SELECT by email (single row)
  - [ ] GetUserByID: SELECT by id (single row)
  - [ ] ListUsers: SELECT all with LIMIT/OFFSET
  - [ ] UpdateUser: UPDATE name, email, role by id returning full row
  - [ ] DeleteUser: DELETE by id
  - [ ] Run `sqlc generate` to verify generation succeeds

- [ ] [BACK] Task 3: Create User domain model and UserRepository port (AC: #1, #2)
  - [ ] Create `backend/internal/domain/model/user.go` with User struct and Role constants
  - [ ] Create `backend/internal/domain/port/user_repository.go` with UserRepository interface
  - [ ] Define interface methods matching sqlc query signatures (Create, GetByEmail, GetByID, List, Update, Delete)

- [ ] [BACK] Task 4: Implement postgres UserRepository adapter (AC: #2)
  - [ ] Create `backend/internal/adapter/postgres/user_repository.go`
  - [ ] Implement UserRepository interface using sqlc-generated Queries
  - [ ] Map between sqlc-generated types and domain model types

- [ ] [BACK] Task 5: Implement AuthService domain service (AC: #3, #4, #5)
  - [ ] Create `backend/internal/domain/service/auth_service.go`
  - [ ] Implement Register(ctx, email, password, name) → (User, token, error): bcrypt hash, create user, generate JWT
  - [ ] Implement Login(ctx, email, password) → (User, token, error): lookup user, bcrypt compare, generate JWT
  - [ ] Implement generateToken(userID, role) → (string, error): JWT with user_id, role, exp claims using golang-jwt/jwt/v5
  - [ ] Implement ValidateToken(tokenString) → (Claims, error): parse and validate JWT
  - [ ] JWT secret and expiration duration injected via config (constructor params)

- [ ] [BACK] Task 6: Implement auth HTTP handlers (AC: #3, #4, #5)
  - [ ] Create `backend/internal/api/handler/auth_handler.go`
  - [ ] Implement POST /auth/register: decode body, call AuthService.Register, set httpOnly secure cookie, return 201 + User JSON
  - [ ] Implement POST /auth/login: decode body, call AuthService.Login, set httpOnly secure cookie, return 200 + User JSON
  - [ ] Implement POST /auth/logout: clear cookie, return 204
  - [ ] Implement GET /auth/me: read user from context (set by middleware), return 200 + User JSON
  - [ ] Cookie config: httpOnly=true, secure=true (false in dev), sameSite=Lax, path=/api

- [ ] [BACK] Task 7: Implement JWT auth middleware (AC: #6, #7)
  - [ ] Create `backend/internal/api/middleware/auth.go`
  - [ ] Extract JWT from cookie named `token`
  - [ ] Validate token via AuthService.ValidateToken
  - [ ] Inject user_id (UUID) and role (string) into request context
  - [ ] Return HTTP 401 JSON error if no cookie, invalid token, or expired token
  - [ ] Expose context helper functions: UserIDFromContext(ctx), RoleFromContext(ctx)

## Dev Notes

This story implements the full auth vertical slice: database schema, data access, domain logic, HTTP layer, and middleware. It depends on Story 1-2 (OpenAPI spec provides the contract) and Story 1-15 (services wiring provides the running server). If 1-15 is not yet merged, the handlers and middleware can still be implemented and unit-tested but won't be wirable into main.go until 1-15 lands. Wire them together in main.go if 1-15 is already done, otherwise leave a TODO comment.

### Split Consideration

This story has exactly 7 tasks. If implementation proves too large, split into:
- **1-3a:** Tasks 1-5 (migration + sqlc + domain model + port + adapter + auth service) — pure domain layer, no HTTP
- **1-3b:** Tasks 6-7 (auth handler + middleware) — HTTP layer only

### Dependencies

- **Story 1-1 (done):** Project scaffold, directory structure, go.mod
- **Story 1-2 (dev-complete):** OpenAPI spec defines auth endpoints contract, sqlc.yaml config exists
- **Story 1-15 (ready-for-dev):** Services wiring (chi router, config, DB pool). Handlers/middleware integrate here.

### Architecture Requirements

**Hexagonal Architecture Boundaries:**
- `internal/domain/model/` — User struct, Role type — NO external imports
- `internal/domain/port/` — UserRepository interface — depends only on domain/model
- `internal/domain/service/` — AuthService — depends on domain/port + domain/model + golang-jwt + bcrypt
- `internal/adapter/postgres/` — UserRepository impl — depends on domain/port + sqlc-generated code + pgx
- `internal/api/handler/` — AuthHandler — depends on domain/service
- `internal/api/middleware/` — AuthMiddleware — depends on domain/service (ValidateToken)

### File Paths (exact)

```
backend/
├── migrations/
│   ├── 000001_create_users_table.up.sql
│   └── 000001_create_users_table.down.sql
├── queries/
│   └── users.sql
├── internal/
│   ├── domain/
│   │   ├── model/
│   │   │   └── user.go
│   │   ├── port/
│   │   │   └── user_repository.go
│   │   └── service/
│   │       └── auth_service.go
│   ├── adapter/
│   │   └── postgres/
│   │       └── user_repository.go      # Hand-written adapter (wraps sqlc-generated code)
│   │       # plus sqlc-generated files: db.go, models.go, users.sql.go (gitignored)
│   ├── api/
│   │   ├── handler/
│   │   │   └── auth_handler.go
│   │   └── middleware/
│   │       └── auth.go
```

### Migration SQL Content

**`backend/migrations/000001_create_users_table.up.sql`:**
```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name        TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_email ON users (email);

-- Auto-update updated_at on row change
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

**`backend/migrations/000001_create_users_table.down.sql`:**
```sql
DROP TRIGGER IF EXISTS set_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS users;
```

### sqlc Query Signatures

**`backend/queries/users.sql`:**
```sql
-- name: CreateUser :one
INSERT INTO users (email, password_hash, name, role)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: UpdateUser :one
UPDATE users
SET name = COALESCE(sqlc.narg('name'), name),
    email = COALESCE(sqlc.narg('email'), email),
    role = COALESCE(sqlc.narg('role'), role),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
```

### Domain Model

**`backend/internal/domain/model/user.go`:**
```go
package model

import (
    "time"
    "github.com/google/uuid"
)

type Role string

const (
    RoleAdmin Role = "admin"
    RoleUser  Role = "user"
)

type User struct {
    ID           uuid.UUID
    Email        string
    PasswordHash string
    Name         string
    Role         Role
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### Domain Port

**`backend/internal/domain/port/user_repository.go`:**
```go
package port

import (
    "context"
    "github.com/google/uuid"
    "github.com/zakari/hopeitworks/backend/internal/domain/model"
)

type UserRepository interface {
    Create(ctx context.Context, user *model.User) (*model.User, error)
    GetByEmail(ctx context.Context, email string) (*model.User, error)
    GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
    List(ctx context.Context, limit, offset int32) ([]*model.User, error)
    Update(ctx context.Context, user *model.User) (*model.User, error)
    Delete(ctx context.Context, id uuid.UUID) error
}
```

### Auth Service Key Signatures

```go
type AuthService struct { ... }

func NewAuthService(repo port.UserRepository, jwtSecret string, jwtExpiration time.Duration) *AuthService
func (s *AuthService) Register(ctx context.Context, email, password, name string) (*model.User, string, error)
func (s *AuthService) Login(ctx context.Context, email, password string) (*model.User, string, error)
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error)
```

**Claims struct:**
```go
type Claims struct {
    UserID uuid.UUID
    Role   model.Role
    jwt.RegisteredClaims
}
```

### JWT Cookie Configuration

```go
http.Cookie{
    Name:     "token",
    Value:    tokenString,
    Path:     "/api",
    HttpOnly: true,
    Secure:   true, // false in development (controlled by config)
    SameSite: http.SameSiteLaxMode,
    MaxAge:   int(jwtExpiration.Seconds()),
}
```

### Context Keys for Middleware

```go
type contextKey string

const (
    ContextKeyUserID contextKey = "user_id"
    ContextKeyRole   contextKey = "user_role"
)

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool)
func RoleFromContext(ctx context.Context) (model.Role, bool)
```

### Go Dependencies to Add

```
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto/bcrypt
go get github.com/google/uuid
go get github.com/golang-migrate/migrate/v4
```

Note: `pgx/v5` and `chi/v5` should already be available from Story 1-2 / 1-15 dependencies.

### Config Additions (config.yaml)

Add under existing config structure (from Story 1-15):
```yaml
auth:
  jwt_secret: dev-secret-change-me-in-production
  jwt_expiration: 24h
  cookie_secure: false  # true in production
```

Environment variable overrides: `AUTH_JWT_SECRET`, `AUTH_JWT_EXPIRATION`, `AUTH_COOKIE_SECURE`.

### Error Responses

Follow the error envelope pattern from the OpenAPI spec:
```json
{
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "Invalid email or password"
  }
}
```

Error codes used:
- `INVALID_CREDENTIALS` — wrong email or password (401)
- `EMAIL_ALREADY_EXISTS` — duplicate email on register (409)
- `VALIDATION_ERROR` — missing/invalid fields (400)
- `UNAUTHORIZED` — no token or invalid token (401)

### Testing Requirements

**Manual verification checklist:**
1. Apply migration: `migrate -path backend/migrations -database "postgres://..." up`
2. Verify table: `psql -c "\d users"` shows all columns and constraints
3. Generate sqlc: `cd backend && sqlc generate` succeeds
4. Register: `curl -X POST http://localhost:8080/api/v1/auth/register -H 'Content-Type: application/json' -d '{"email":"test@example.com","password":"secureP@ss1","name":"Test User"}'` returns 201
5. Login: `curl -c cookies.txt -X POST http://localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"test@example.com","password":"secureP@ss1"}'` returns 200 + Set-Cookie header
6. Auth me: `curl -b cookies.txt http://localhost:8080/api/v1/auth/me` returns 200 + user
7. Bad login: `curl -X POST http://localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"test@example.com","password":"wrong"}'` returns 401
8. No token: `curl http://localhost:8080/api/v1/auth/me` returns 401
9. Logout: `curl -b cookies.txt -X POST http://localhost:8080/api/v1/auth/logout` returns 204

### References

- [Source: _bmad-output/planning-artifacts/architecture.md#Authentication & Security]
- [Source: _bmad-output/planning-artifacts/architecture.md#Auth (api/middleware/)]
- [Source: _bmad-output/planning-artifacts/architecture.md#REST Endpoints — Auth]
- [Source: _bmad-output/planning-artifacts/architecture.md#Backend Architecture — Hexagonal Structure]
- [Source: _bmad-output/planning-artifacts/epics.md#Story 1.3]
- [Source: api/openapi.yaml — RegisterRequest, LoginRequest, User schemas]
- [Source: backend/sqlc.yaml — sqlc configuration with pgx/v5 and UUID overrides]

## Dev Agent Record

### Agent Model Used

_To be filled by the dev agent after implementation_

### Debug Log References

_To be filled by the dev agent after implementation_

### Completion Notes List

_To be filled by the dev agent after implementation. Include:_
- Any deviations from the spec and rationale
- Issues encountered and solutions
- Additional files created beyond the spec
- Recommendations for future stories

### File List

_To be filled by the dev agent after implementation. List all files created or modified with absolute paths._

## Change Log
