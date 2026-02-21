# Story feat-2: [BACK] User profile API (GET/PUT /users/me)

Status: ready-for-dev

## Story

As an authenticated user,
I want to view and update my own profile (name, email) and change my password via dedicated self-service endpoints,
so that I can manage my account without requiring admin access.

## Acceptance Criteria (BDD)

**AC1: Get own profile**
- **Given** I am authenticated (valid JWT cookie)
- **When** I send `GET /api/v1/users/me`
- **Then** I receive HTTP 200 with my user object (`id`, `email`, `name`, `role`, `created_at`, `updated_at`)

**AC2: Update own profile (name and/or email)**
- **Given** I am authenticated
- **When** I send `PUT /api/v1/users/me` with a JSON body containing `name` and/or `email`
- **Then** I receive HTTP 200 with the updated user object
- **And** the changes are persisted to the database

**AC3: Change own password**
- **Given** I am authenticated
- **When** I send `PUT /api/v1/users/me/password` with `current_password` and `new_password`
- **Then** I receive HTTP 204 No Content on success

**AC4: Password change fails on wrong current password**
- **Given** I am authenticated
- **When** I send `PUT /api/v1/users/me/password` with an incorrect `current_password`
- **Then** I receive HTTP 401 with error code `INVALID_CREDENTIALS`

**AC5: Update profile fails on empty name or malformed email**
- **Given** I am authenticated
- **When** I send `PUT /api/v1/users/me` with an empty `name` or a non-email string for `email`
- **Then** I receive HTTP 400 with error code `VALIDATION_ERROR`

**AC6: Unauthenticated requests are rejected**
- **Given** I have no valid JWT cookie
- **When** I send `GET /api/v1/users/me` or `PUT /api/v1/users/me` or `PUT /api/v1/users/me/password`
- **Then** I receive HTTP 401 with error code `UNAUTHORIZED`

## Tasks / Subtasks

- [ ] [BACK] Task 1: Update `api/openapi.yaml` with new endpoints and schemas (AC: #1, #2, #3, #5, #6)
  - [ ] Add `GET /users/me` endpoint (operationId: `getMyProfile`, tag: `users`)
  - [ ] Add `PUT /users/me` endpoint (operationId: `updateMyProfile`, tag: `users`) with `UpdateMyProfileRequest` schema
  - [ ] Add `PUT /users/me/password` endpoint (operationId: `changeMyPassword`, tag: `users`) with `ChangePasswordRequest` schema
  - [ ] Reference the existing `User` response schema for both GET and PUT responses
  - [ ] Regenerate backend types: `cd backend && make generate`

- [ ] [BACK] Task 2: Add `UpdateProfile` and `ChangePassword` methods to `UserService` (AC: #1, #2, #3, #4, #5)
  - [ ] Add `UpdateProfileParams` struct (fields: `ID uuid.UUID`, `Name *string`, `Email *string`)
  - [ ] Add `UpdateProfile(ctx, params UpdateProfileParams) (*model.User, error)` — validates name non-empty (max 255), delegates to `repo.Update`; no role field (users cannot self-promote)
  - [ ] Add `ChangePassword(ctx, userID uuid.UUID, currentPassword, newPassword string) error` — fetches user, verifies current password with `bcrypt.CompareHashAndPassword`, hashes new password, calls `repo.UpdatePasswordHash`
  - [ ] Add `ErrInvalidCurrentPassword` sentinel to `UserService` (or reuse `AuthService.ErrInvalidCredentials`; prefer a new sentinel in service package to avoid cross-service import)
  - [ ] Add sqlc query `UpdateUserPasswordHash` in `backend/queries/users.sql`
  - [ ] Run `cd backend && sqlc generate` to regenerate DB layer
  - [ ] Add `UpdatePasswordHash(ctx, id uuid.UUID, hash string) error` to `port.UserRepository` interface
  - [ ] Implement `UpdatePasswordHash` on `postgres.UserRepository`

- [ ] [BACK] Task 3: Implement `ProfileHandler` in `backend/internal/api/handler/profile_handler.go` (AC: #1, #2, #3, #4, #5, #6)
  - [ ] `GetMyProfile(w, r)` — extract `userID` from `middleware.UserIDFromContext`, call `userService.GetByID`, respond 200 with `toAPIUser`
  - [ ] `UpdateMyProfile(w, r)` — decode body, validate, call `userService.UpdateProfile`, respond 200 with `toAPIUser`
  - [ ] `ChangeMyPassword(w, r)` — decode body, validate non-empty fields, call `userService.ChangePassword`, respond 204; map `ErrInvalidCurrentPassword` → 401 `INVALID_CREDENTIALS`
  - [ ] Use `writeErrorResponse` / `writeJSON` helpers from `handler` package

- [ ] [BACK] Task 4: Wire `ProfileHandler` into `Server` and register routes (AC: #1, #2, #3, #6)
  - [ ] Add `profile *ProfileHandler` field to `Server` struct in `server.go`
  - [ ] Update `NewServer(...)` to accept `*ProfileHandler`
  - [ ] Add delegate methods: `GetMyProfile`, `UpdateMyProfile`, `ChangeMyPassword` on `Server`
  - [ ] Instantiate `ProfileHandler` in `cmd/api/main.go` and pass to `NewServer`
  - [ ] Verify routes are registered via `HandlerFromMuxWithBaseURL` (generated from the updated OpenAPI spec)

- [ ] [BACK] Task 5: Tests (AC: #1, #2, #3, #4, #5, #6)
  - [ ] Unit tests for `UserService.UpdateProfile` — table-driven: valid update, empty name, email conflict (duplicate)
  - [ ] Unit tests for `UserService.ChangePassword` — correct current password, wrong current password, new password too short
  - [ ] Handler tests for `ProfileHandler` using `httptest` with mock `UserService` — cover all 3 endpoints and error paths

## Dev Notes

### Dependencies

- No new Go modules required — `bcrypt` already used in `auth_service.go` (via `golang.org/x/crypto`)
- `UserRepository.Update` already exists; only `UpdatePasswordHash` is new

### File Paths (exact)

| File | Action |
|------|--------|
| `api/openapi.yaml` | Add 3 new paths and 2 new schemas |
| `backend/queries/users.sql` | Add `UpdateUserPasswordHash` query |
| `backend/internal/adapter/postgres/db/` | Regenerated by sqlc — do not edit manually |
| `backend/internal/domain/port/user_repository.go` | Add `UpdatePasswordHash` method |
| `backend/internal/adapter/postgres/user_repository.go` | Implement `UpdatePasswordHash` |
| `backend/internal/domain/service/user_service.go` | Add `UpdateProfile`, `ChangePassword`, params structs, sentinel error |
| `backend/internal/api/handler/profile_handler.go` | New file — `ProfileHandler` |
| `backend/internal/api/handler/server.go` | Add `profile` field + delegate methods |
| `backend/cmd/api/main.go` | Instantiate `ProfileHandler`, pass to `NewServer` |

### New/Modified sqlc Queries

Add to `backend/queries/users.sql`:

```sql
-- name: UpdateUserPasswordHash :exec
UPDATE users
SET password_hash = $2,
    updated_at    = now()
WHERE id = $1 AND deleted_at IS NULL;
```

Regenerate: `cd backend && sqlc generate`

This generates `db.UpdateUserPasswordHash(ctx, db.UpdateUserPasswordHashParams{ID: id, PasswordHash: hash})`.

### OpenAPI spec additions (new endpoints + schemas)

Add to `api/openapi.yaml` under `paths:`, after the existing `/users/{id}` block:

```yaml
  /users/me:
    get:
      operationId: getMyProfile
      summary: Get the current user's profile
      tags: [users]
      responses:
        "200":
          description: Current user profile
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
        "401":
          $ref: "#/components/responses/Unauthorized"

    put:
      operationId: updateMyProfile
      summary: Update the current user's profile (name and/or email)
      tags: [users]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/UpdateMyProfileRequest"
      responses:
        "200":
          description: Updated user profile
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
        "400":
          $ref: "#/components/responses/BadRequest"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "409":
          $ref: "#/components/responses/Conflict"

  /users/me/password:
    put:
      operationId: changeMyPassword
      summary: Change the current user's password
      tags: [users]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/ChangePasswordRequest"
      responses:
        "204":
          description: Password changed successfully
        "400":
          $ref: "#/components/responses/BadRequest"
        "401":
          $ref: "#/components/responses/Unauthorized"
```

Add to `components/schemas:`:

```yaml
    UpdateMyProfileRequest:
      type: object
      properties:
        name:
          type: string
          minLength: 1
          maxLength: 255
        email:
          type: string
          format: email
      minProperties: 1

    ChangePasswordRequest:
      type: object
      required: [current_password, new_password]
      properties:
        current_password:
          type: string
          minLength: 1
        new_password:
          type: string
          minLength: 8
```

After updating the spec, regenerate: `cd backend && make generate`

### Handler Signatures

```go
// backend/internal/api/handler/profile_handler.go

package handler

import (
    "encoding/json"
    "net/http"

    "golang.org/x/crypto/bcrypt"

    "github.com/zakari/hopeitworks/backend/internal/api/middleware"
    "github.com/zakari/hopeitworks/backend/internal/domain/service"
    "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// ProfileHandler handles self-service profile endpoints for the authenticated user.
type ProfileHandler struct {
    userService *service.UserService
}

// NewProfileHandler creates a new ProfileHandler.
func NewProfileHandler(svc *service.UserService) *ProfileHandler {
    return &ProfileHandler{userService: svc}
}

// GetMyProfile handles GET /users/me.
func (h *ProfileHandler) GetMyProfile(w http.ResponseWriter, r *http.Request)

// UpdateMyProfile handles PUT /users/me.
func (h *ProfileHandler) UpdateMyProfile(w http.ResponseWriter, r *http.Request)

// ChangeMyPassword handles PUT /users/me/password.
func (h *ProfileHandler) ChangeMyPassword(w http.ResponseWriter, r *http.Request)
```

Service method signatures to add to `backend/internal/domain/service/user_service.go`:

```go
// ErrInvalidCurrentPassword is returned when the current password does not match.
var ErrInvalidCurrentPassword = errors.NewUnauthorized("current password is incorrect")

// UpdateProfileParams holds parameters for self-service profile updates.
// Role is intentionally excluded — users cannot change their own role.
type UpdateProfileParams struct {
    ID    uuid.UUID
    Name  *string
    Email *string
}

// UpdateProfile validates and applies a self-service profile update.
func (s *UserService) UpdateProfile(ctx context.Context, params UpdateProfileParams) (*model.User, error)

// ChangePassword verifies the current password and sets a new bcrypt hash.
func (s *UserService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error
```

Port addition in `backend/internal/domain/port/user_repository.go`:

```go
// UpdatePasswordHash replaces the bcrypt password hash for a user.
UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error
```

Postgres adapter addition in `backend/internal/adapter/postgres/user_repository.go`:

```go
// UpdatePasswordHash implements port.UserRepository.
func (r *UserRepository) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
    return r.queries.UpdateUserPasswordHash(ctx, db.UpdateUserPasswordHashParams{
        ID:           id,
        PasswordHash: hash,
    })
}
```

Server delegates to add in `backend/internal/api/handler/server.go`:

```go
// GetMyProfile delegates to ProfileHandler.
func (s *Server) GetMyProfile(w http.ResponseWriter, r *http.Request) {
    s.profile.GetMyProfile(w, r)
}

// UpdateMyProfile delegates to ProfileHandler.
func (s *Server) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
    s.profile.UpdateMyProfile(w, r)
}

// ChangeMyPassword delegates to ProfileHandler.
func (s *Server) ChangeMyPassword(w http.ResponseWriter, r *http.Request) {
    s.profile.ChangeMyPassword(w, r)
}
```

### Error Responses

| Scenario | HTTP | Code |
|----------|------|------|
| No valid JWT cookie | 401 | `UNAUTHORIZED` |
| Empty name in profile update | 400 | `VALIDATION_ERROR` |
| Name > 255 chars | 400 | `VALIDATION_ERROR` |
| Malformed email | 400 | `VALIDATION_ERROR` |
| Email already taken by another user | 409 | `CONFLICT` |
| Wrong current password | 401 | `INVALID_CREDENTIALS` |
| New password < 8 chars | 400 | `VALIDATION_ERROR` |
| User not found (deleted between auth and handler) | 401 | `UNAUTHORIZED` (treat as auth failure, not 404) |

Note: The email conflict on `PUT /users/me` flows through `repo.Update`, which will return a Postgres unique constraint error (code 23505). Use the existing `isDuplicateKeyError` pattern (already in `auth_service.go`) or surface it via the postgres adapter wrapping it as `errors.NewConflict("email", email)`.

### Testing Requirements

- Use `httptest.NewRecorder()` and `httptest.NewRequest()` for handler tests
- Inject `middleware.SetUserContext(ctx, userID, role)` to simulate authenticated requests in handler tests
- Mock `UserService` with a hand-written mock implementing the methods under test
- Service unit tests use a mock `UserRepository` (hand-written)
- All tests must pass `go test ./... -short`
- Lint must pass: `cd backend && golangci-lint run ./...`

Key test cases:
1. `GET /users/me` — 200 returns correct user fields; 401 when no user in context
2. `PUT /users/me` — 200 with name-only update; 200 with email-only update; 400 on empty name; 400 on no fields provided; 409 on duplicate email
3. `PUT /users/me/password` — 204 on success; 401 on wrong current password; 400 on new password < 8 chars
4. `UserService.ChangePassword` — verifies bcrypt check is applied; verifies new hash is stored

## Dev Agent Record

_To be filled in by the implementing agent._

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-21 | Claude | Initial draft |
