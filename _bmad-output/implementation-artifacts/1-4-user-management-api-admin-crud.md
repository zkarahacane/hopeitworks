# Story 1.4: [BACK] User management API (admin CRUD)

Status: ready-for-dev

## Story

As an admin,
I want to manage user accounts,
So that I can control platform access.

## Acceptance Criteria (BDD)

**AC1: Admin can list users with pagination**
- **Given** I am authenticated as admin
- **When** I GET /api/v1/users
- **Then** I receive HTTP 200 with a paginated list of users (data[] + pagination)

**AC2: Non-admin cannot list users**
- **Given** I am authenticated as a non-admin user
- **When** I GET /api/v1/users
- **Then** I receive HTTP 403

**AC3: Admin can get a single user**
- **Given** I am authenticated as admin
- **When** I GET /api/v1/users/{id}
- **Then** I receive HTTP 200 with the user object (including role)

**AC4: Admin can update a user (including role change)**
- **Given** I am authenticated as admin
- **When** I PUT /api/v1/users/{id} with role change payload
- **Then** the user role is updated and I receive HTTP 200 with the updated user

**AC5: Non-admin cannot update other users**
- **Given** I am authenticated as a non-admin user
- **When** I PUT /api/v1/users/{id} targeting another user
- **Then** I receive HTTP 403

**AC6: Admin can deactivate a user**
- **Given** I am authenticated as admin
- **When** I DELETE /api/v1/users/{id}
- **Then** the user is soft-deleted (deleted_at set) and I receive HTTP 204

**AC7: OpenAPI spec includes role field**
- **Given** the OpenAPI spec is updated
- **When** code is regenerated
- **Then** the `User` schema includes `role` and `UpdateUserRequest` includes `role`

## Tasks / Subtasks

- [ ] [BACK] Task 1: Update OpenAPI spec to add `role` field + `Forbidden` response + soft-delete semantics (AC: #7, #2, #5, #6)
  - [ ] Add `role` field (type: string, enum: [admin, user]) to `User` schema in `api/openapi.yaml`
  - [ ] Add `role` field (type: string, enum: [admin, user]) to `UpdateUserRequest` schema
  - [ ] Add `Forbidden` response component (`403` with Error schema) to `components/responses`
  - [ ] Add `403` response reference to `GET /users`, `PUT /users/{id}`, `DELETE /users/{id}`
  - [ ] Regenerate backend types: `cd backend && make generate`
  - [ ] Regenerate frontend types: `cd frontend && npm run generate-api`
  - [ ] Verify the generated `User` struct in `gen_server.go` now includes `Role` field
  - [ ] Verify the generated `UpdateUserRequest` struct includes `Role` field

- [ ] [BACK] Task 2: Add `CountUsers` sqlc query + migration for soft-delete (AC: #1, #6)
  - [ ] Create migration `backend/migrations/000003_add_users_deleted_at.up.sql`:
    ```sql
    ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ;
    CREATE INDEX idx_users_deleted_at ON users (deleted_at) WHERE deleted_at IS NULL;
    ```
  - [ ] Create `backend/migrations/000003_add_users_deleted_at.down.sql`:
    ```sql
    DROP INDEX IF EXISTS idx_users_deleted_at;
    ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;
    ```
  - [ ] Add `CountUsers` query to `backend/queries/users.sql`:
    ```sql
    -- name: CountUsers :one
    SELECT COUNT(*) FROM users WHERE deleted_at IS NULL;
    ```
  - [ ] Update `ListUsers` query to filter soft-deleted:
    ```sql
    -- name: ListUsers :many
    SELECT * FROM users WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2;
    ```
  - [ ] Update `GetUserByID` query to filter soft-deleted:
    ```sql
    -- name: GetUserByID :one
    SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL;
    ```
  - [ ] Update `GetUserByEmail` query to filter soft-deleted:
    ```sql
    -- name: GetUserByEmail :one
    SELECT * FROM users WHERE email = $1 AND deleted_at IS NULL;
    ```
  - [ ] Replace `DeleteUser` query with soft-delete:
    ```sql
    -- name: DeleteUser :exec
    UPDATE users SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;
    ```
  - [ ] Run `cd backend && sqlc generate` to verify generation succeeds

- [ ] [BACK] Task 3: Update domain model and port interface (AC: #1, #6)
  - [ ] Add `DeletedAt *time.Time` field to `User` struct in `backend/internal/domain/model/user.go`
  - [ ] Add `Count(ctx context.Context) (int64, error)` method to `UserRepository` interface in `backend/internal/domain/port/user_repository.go`
  - [ ] Verify existing methods signatures remain compatible

- [ ] [BACK] Task 4: Update UserRepository postgres adapter (AC: #1, #6)
  - [ ] Add `Count` method to `backend/internal/adapter/postgres/user_repository.go`:
    ```go
    func (r *UserRepository) Count(ctx context.Context) (int64, error) {
        return r.q.CountUsers(ctx)
    }
    ```
  - [ ] Update `toDomainUser` mapping to handle `deleted_at` field (nullable timestamp)
  - [ ] Verify adapter still satisfies `port.UserRepository` interface (compile check)

- [ ] [BACK] Task 5: Create UserService domain service (AC: #1, #3, #4, #6)
  - [ ] Create `backend/internal/domain/service/user_service.go`
  - [ ] Constructor: `func NewUserService(repo port.UserRepository) *UserService`
  - [ ] `GetByID(ctx, id uuid.UUID) (*model.User, error)` -- delegates to repo, returns not-found domain error
  - [ ] `List(ctx, page, perPage int) (*UserListResult, error)` -- validates pagination params, calls repo.List + repo.Count
    ```go
    type UserListResult struct {
        Users []*model.User
        Total int64
    }
    ```
  - [ ] `Update(ctx, params UpdateUserParams) (*model.User, error)` -- validates inputs, fetches existing, applies partial updates, calls repo.Update
    ```go
    type UpdateUserParams struct {
        ID    uuid.UUID
        Name  *string
        Email *string
        Role  *model.Role
    }
    ```
  - [ ] Validation rules: name not empty if provided, name <= 255 chars, email not empty if provided, role must be valid (`admin` or `user`) if provided
  - [ ] `Delete(ctx, id uuid.UUID) error` -- verifies user exists, delegates to repo.Delete (now soft-delete)
  - [ ] Service depends only on `port.UserRepository` (no adapter imports)

- [ ] [BACK] Task 6: Create UserHandler with admin-only RBAC (AC: #1, #2, #3, #4, #5, #6)
  - [ ] Create `backend/internal/api/handler/user_handler.go`
  - [ ] Constructor: `func NewUserHandler(svc *service.UserService) *UserHandler`
  - [ ] `ListUsers(w, r, params ListUsersParams)` -- admin-only check, parse page/per_page, call service.List, return UserList JSON
  - [ ] `GetUser(w, r, id IdPath)` -- admin-only check, call service.GetByID, return User JSON (include `role` field)
  - [ ] `UpdateUser(w, r, id IdPath)` -- admin-only check, decode UpdateUserRequest, call service.Update, return User JSON
  - [ ] `DeleteUser(w, r, id IdPath)` -- admin-only check, call service.Delete, return 204
  - [ ] All endpoints use `middleware.IsAdmin(r.Context())` for RBAC, return 403 via `writeErrorResponse(w, errors.NewForbidden("Admin access required"))` if not admin
  - [ ] Add `toAPIUser(u *model.User) User` helper in `backend/internal/api/handler/helpers.go` that maps domain User to generated API User type (including `role`)

- [ ] [BACK] Task 7: Wire UserHandler into Server and main.go (AC: #1-6)
  - [ ] Update `backend/internal/api/handler/server.go`:
    - Add `users *UserHandler` field to `Server` struct
    - Update `NewServer` to accept `*UserHandler` parameter
    - Add delegation methods: `ListUsers`, `GetUser`, `UpdateUser`, `DeleteUser`
  - [ ] Update `backend/cmd/api/main.go`:
    - Instantiate `UserService` with existing `userRepo`
    - Instantiate `UserHandler` with `userService`
    - Pass `userHandler` to `NewServer`
  - [ ] Verify build: `cd backend && go build ./...`

- [ ] [BACK] Task 8: Unit tests for UserService (AC: #1, #3, #4, #6)
  - [ ] Create `backend/internal/domain/service/user_service_test.go`
  - [ ] Create `MockUserRepository` implementing `port.UserRepository` (with `CountFn` and all existing methods)
  - [ ] Test `List` happy path -- returns users + total, respects pagination bounds
  - [ ] Test `List` with invalid page/perPage -- defaults applied
  - [ ] Test `GetByID` happy path -- returns user
  - [ ] Test `GetByID` not found -- returns domain not-found error
  - [ ] Test `Update` happy path -- partial updates applied (name, email, role)
  - [ ] Test `Update` with invalid role -- returns validation error
  - [ ] Test `Update` with empty name -- returns validation error
  - [ ] Test `Delete` happy path -- calls repo.Delete
  - [ ] Test `Delete` not found -- returns domain not-found error

- [ ] [BACK] Task 9: Unit tests for UserHandler (AC: #1, #2, #3, #4, #5, #6)
  - [ ] Create `backend/internal/api/handler/user_handler_test.go`
  - [ ] Create mock `UserService` or use the handler with a mock repository
  - [ ] Test `ListUsers` as admin -- returns 200 with paginated user list
  - [ ] Test `ListUsers` as non-admin -- returns 403
  - [ ] Test `GetUser` as admin -- returns 200 with user (including role field)
  - [ ] Test `GetUser` not found -- returns 404
  - [ ] Test `UpdateUser` as admin with role change -- returns 200 with updated user
  - [ ] Test `UpdateUser` as non-admin -- returns 403
  - [ ] Test `UpdateUser` with invalid JSON -- returns 400
  - [ ] Test `DeleteUser` as admin -- returns 204
  - [ ] Test `DeleteUser` as non-admin -- returns 403
  - [ ] Test `DeleteUser` not found -- returns 404
  - [ ] Use `middleware.SetUserContext` to inject admin/non-admin context in tests

- [ ] [BACK] Task 10: Verify build, lint, and full test suite (AC: #1-7)
  - [ ] Run `cd backend && go build ./...` -- must compile
  - [ ] Run `cd backend && go vet ./...` -- must pass
  - [ ] Run `cd backend && go test ./... -short` -- all unit tests pass
  - [ ] Verify no `console.log`, `fmt.Println`, or commented-out code in committed files
  - [ ] Verify all exported functions have doc comments

## Dev Notes

This story builds the user management CRUD vertical on top of the existing auth infrastructure from Story 1-3. The key changes are: (1) a new `UserService` separate from `AuthService` for CRUD operations, (2) a new `UserHandler` for the `/users` endpoints, (3) OpenAPI spec updates to expose the `role` field, and (4) soft-delete semantics for user deactivation.

### Dependencies

**Story 1-3 (done):** Provides `users` table, `UserRepository` port+adapter, `AuthService`, JWT middleware, `model.User` with `Role` type, and all sqlc queries. This story extends that foundation.

**Story 1-5 (done):** Provides reference patterns for handler RBAC, service structure, pagination, and server wiring. `ProjectHandler` is the template for `UserHandler`.

**Story 1-15 (done):** Provides the chi router, `main.go` wiring, and middleware stack.

### Architecture Requirements

**Hexagonal boundaries:**
- `domain/model/user.go` -- add `DeletedAt` field (no new external imports)
- `domain/port/user_repository.go` -- add `Count` method
- `domain/service/user_service.go` -- NEW file, depends only on `port.UserRepository`
- `adapter/postgres/user_repository.go` -- add `Count` method, update mappings
- `api/handler/user_handler.go` -- NEW file, depends on `service.UserService`

**Import direction:** `handler -> service -> port <- adapter`

**Key distinction:** `AuthService` handles register/login/token operations. `UserService` handles admin CRUD (list, get, update, delete). They share the same `UserRepository` port but serve different purposes.

### File Paths (exact)

```
api/
└── openapi.yaml                              # MODIFIED: add role to User + UpdateUserRequest, add Forbidden response

backend/
├── migrations/
│   ├── 000003_add_users_deleted_at.up.sql    # NEW
│   └── 000003_add_users_deleted_at.down.sql  # NEW
├── queries/
│   └── users.sql                             # MODIFIED: add CountUsers, update ListUsers/GetUserByID/GetUserByEmail/DeleteUser
├── internal/
│   ├── domain/
│   │   ├── model/
│   │   │   └── user.go                       # MODIFIED: add DeletedAt field
│   │   ├── port/
│   │   │   └── user_repository.go            # MODIFIED: add Count method
│   │   └── service/
│   │       └── user_service.go               # NEW
│   │       └── user_service_test.go          # NEW
│   ├── adapter/
│   │   └── postgres/
│   │       └── user_repository.go            # MODIFIED: add Count, update toDomainUser
│   └── api/
│       └── handler/
│           ├── user_handler.go               # NEW
│           ├── user_handler_test.go          # NEW
│           ├── helpers.go                    # MODIFIED: add toAPIUser
│           └── server.go                     # MODIFIED: add users field + delegation
└── cmd/
    └── api/
        └── main.go                           # MODIFIED: wire UserService + UserHandler
```

### Technical Specifications

**OpenAPI spec changes (`api/openapi.yaml`):**

Add to `User` schema properties:
```yaml
        role:
          type: string
          enum: [admin, user]
          example: user
```

Add `role` to `User` required fields.

Add to `UpdateUserRequest` schema properties:
```yaml
        role:
          type: string
          enum: [admin, user]
```

Add `Forbidden` response component:
```yaml
    Forbidden:
      description: Forbidden - admin access required
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"
```

Add `403` to user endpoints:
```yaml
        "403":
          $ref: "#/components/responses/Forbidden"
```

**After regeneration, expected generated types:**

```go
// User (in gen_server.go)
type User struct {
    CreatedAt time.Time           `json:"created_at"`
    Email     openapi_types.Email `json:"email"`
    Id        openapi_types.UUID  `json:"id"`
    Name      string              `json:"name"`
    Role      string              `json:"role"`        // NEW
    UpdatedAt time.Time           `json:"updated_at"`
}

// UpdateUserRequest (in gen_server.go)
type UpdateUserRequest struct {
    Email *openapi_types.Email `json:"email,omitempty"`
    Name  *string              `json:"name,omitempty"`
    Role  *string              `json:"role,omitempty"` // NEW
}
```

**UserService key signatures:**

```go
package service

type UserService struct {
    repo port.UserRepository
}

func NewUserService(repo port.UserRepository) *UserService

type UserListResult struct {
    Users []*model.User
    Total int64
}

type UpdateUserParams struct {
    ID    uuid.UUID
    Name  *string
    Email *string
    Role  *model.Role
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
func (s *UserService) List(ctx context.Context, page, perPage int) (*UserListResult, error)
func (s *UserService) Update(ctx context.Context, params UpdateUserParams) (*model.User, error)
func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error
```

**UserHandler key signatures:**

```go
package handler

type UserHandler struct {
    service *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request, params ListUsersParams)
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request, id IdPath)
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request, id IdPath)
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request, id IdPath)
```

**toAPIUser helper (in helpers.go):**

```go
func toAPIUser(u *model.User) User {
    return User{
        Id:        u.ID,
        Email:     openapi_types.Email(u.Email),
        Name:      u.Name,
        Role:      string(u.Role),
        CreatedAt: u.CreatedAt,
        UpdatedAt: u.UpdatedAt,
    }
}
```

**Server wiring update:**

```go
type Server struct {
    Unimplemented
    projects *ProjectHandler
    users    *UserHandler
}

func NewServer(projects *ProjectHandler, users *UserHandler) *Server {
    return &Server{projects: projects, users: users}
}

func (s *Server) ListUsers(w http.ResponseWriter, r *http.Request, params ListUsersParams) {
    s.users.ListUsers(w, r, params)
}
func (s *Server) GetUser(w http.ResponseWriter, r *http.Request, id IdPath) {
    s.users.GetUser(w, r, id)
}
func (s *Server) UpdateUser(w http.ResponseWriter, r *http.Request, id IdPath) {
    s.users.UpdateUser(w, r, id)
}
func (s *Server) DeleteUser(w http.ResponseWriter, r *http.Request, id IdPath) {
    s.users.DeleteUser(w, r, id)
}
```

**main.go wiring additions:**

```go
// User service (add after projectHandler)
userService := service.NewUserService(userRepo)
userHandler := handler.NewUserHandler(userService)
server := handler.NewServer(projectHandler, userHandler)
```

**Error codes used in this story:**
- `FORBIDDEN` -- non-admin accessing admin-only endpoints (403)
- `USER_NOT_FOUND` -- user ID does not exist (404)
- `VALIDATION_ERROR` -- invalid input fields (400)
- `USER_ALREADY_EXISTS` -- duplicate email on update (409)

### Testing Requirements

**Unit tests (fast, no containers):**
- `backend/internal/domain/service/user_service_test.go` -- table-driven tests with MockUserRepository
- `backend/internal/api/handler/user_handler_test.go` -- HTTP handler tests with mock service, using `httptest.NewRecorder`

**Mock pattern:** Hand-written mocks implementing `port.UserRepository` with function fields (same pattern as existing `auth_service_test.go`).

**Test context setup:** Use `middleware.SetUserContext(ctx, adminID, model.RoleAdmin)` for admin tests and `middleware.SetUserContext(ctx, userID, model.RoleUser)` for non-admin tests.

**Verification checklist:**
1. `cd backend && make generate` -- regeneration succeeds with updated spec
2. `cd backend && go build ./...` -- compiles cleanly
3. `cd backend && go vet ./...` -- no issues
4. `cd backend && go test ./... -short` -- all tests pass
5. Admin GET `/api/v1/users` -> 200 with data[] and pagination
6. Non-admin GET `/api/v1/users` -> 403
7. Admin GET `/api/v1/users/{id}` -> 200 with user including role
8. Admin PUT `/api/v1/users/{id}` with `{"role": "admin"}` -> 200 with updated role
9. Non-admin PUT `/api/v1/users/{id}` -> 403
10. Admin DELETE `/api/v1/users/{id}` -> 204 (soft-delete: deleted_at set, user no longer appears in list)

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 1.4]
- [Source: _bmad-output/planning-artifacts/architecture.md#Backend Architecture -- Hexagonal Structure]
- [Source: _bmad-output/planning-artifacts/architecture.md#API Design]
- [Source: api/openapi.yaml -- /users endpoints, User, UpdateUserRequest, UserList schemas]
- [Source: backend/internal/api/handler/project_handler.go -- reference RBAC pattern]
- [Source: backend/internal/domain/service/project_service.go -- reference service pattern]
- [Source: backend/internal/api/handler/server.go -- delegation pattern]
