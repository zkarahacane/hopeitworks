# Story 1.6: [BACK] RBAC middleware + project_users table

Status: ready-for-dev

## Story

As a platform administrator,
I want role-based access control on all project-scoped endpoints,
So that users only access their assigned projects.

## Acceptance Criteria (BDD)

**AC1: Migration creates project_users table**
- **Given** migration 000003 exists
- **When** migrations are applied
- **Then** a `project_users` table is created with composite PK (project_id UUID FK -> projects, user_id UUID FK -> users), role (TEXT NOT NULL DEFAULT 'member'), created_at (TIMESTAMPTZ)

**AC2: Admin bypasses project assignment check**
- **Given** a request to a project-scoped endpoint by an admin
- **When** RBAC middleware runs
- **Then** access is granted without assignment check

**AC3: Assigned user can access project**
- **Given** a request by a user assigned to the project
- **When** RBAC middleware runs
- **Then** access is granted

**AC4: Unassigned user is denied**
- **Given** a request by a user NOT assigned to the project
- **When** RBAC middleware runs
- **Then** HTTP 403 is returned with error envelope `{"error":{"code":"FORBIDDEN","message":"..."}}`

**AC5: Admin can assign user to project**
- **Given** I am admin
- **When** I POST `/api/v1/projects/{id}/users` with `{"user_id":"..."}`
- **Then** the user is assigned to the project and HTTP 201 is returned

**AC6: Admin can remove user from project**
- **Given** I am admin
- **When** I DELETE `/api/v1/projects/{id}/users/{user_id}`
- **Then** the user is removed from the project and HTTP 204 is returned

**AC7: Admin can list project members**
- **Given** I am admin or an assigned user
- **When** I GET `/api/v1/projects/{id}/users`
- **Then** I receive the list of users assigned to the project

**AC8: Non-admin list projects returns only assigned projects**
- **Given** I am a non-admin user assigned to projects A and B but not C
- **When** I GET `/api/v1/projects`
- **Then** I receive only projects A and B

## Tasks / Subtasks

- [ ] [BACK] Task 1: Create migration 000003 for project_users table (AC: #1)
  - [ ] Create `backend/migrations/000003_create_project_users_table.up.sql`
  - [ ] Create `backend/migrations/000003_create_project_users_table.down.sql`
  - [ ] Define project_users table:
    ```sql
    CREATE TABLE project_users (
        project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
        user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
        role       TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'member')),
        created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
        PRIMARY KEY (project_id, user_id)
    );

    CREATE INDEX idx_project_users_user_id ON project_users(user_id);
    CREATE INDEX idx_project_users_project_id ON project_users(project_id);
    ```
  - [ ] Down migration drops the table and indexes

- [ ] [BACK] Task 2: Create sqlc queries for project_users (AC: #1, #5, #6, #7)
  - [ ] Create `backend/queries/project_users.sql` with the following queries:
    ```sql
    -- name: AddUserToProject :one
    INSERT INTO project_users (project_id, user_id, role)
    VALUES ($1, $2, $3)
    RETURNING *;

    -- name: RemoveUserFromProject :exec
    DELETE FROM project_users WHERE project_id = $1 AND user_id = $2;

    -- name: ListProjectUsers :many
    SELECT u.id, u.email, u.name, u.role AS user_role, pu.role AS project_role, pu.created_at AS assigned_at
    FROM project_users pu
    JOIN users u ON u.id = pu.user_id
    WHERE pu.project_id = $1
    ORDER BY pu.created_at ASC;

    -- name: IsUserInProject :one
    SELECT EXISTS(
        SELECT 1 FROM project_users WHERE project_id = $1 AND user_id = $2
    ) AS is_member;

    -- name: ListUserProjectIDs :many
    SELECT project_id FROM project_users WHERE user_id = $1;

    -- name: ListProjectsByUser :many
    SELECT p.* FROM projects p
    INNER JOIN project_users pu ON pu.project_id = p.id
    WHERE pu.user_id = $1
    ORDER BY p.created_at DESC
    LIMIT $2 OFFSET $3;

    -- name: CountProjectsByUser :one
    SELECT COUNT(*) FROM projects p
    INNER JOIN project_users pu ON pu.project_id = p.id
    WHERE pu.user_id = $1;
    ```
  - [ ] Run `cd backend && make generate` and verify sqlc output compiles

- [ ] [BACK] Task 3: Create ProjectUser domain model and ProjectUserRepository port (AC: #1, #5, #6, #7)
  - [ ] Create `backend/internal/domain/model/project_user.go`:
    ```go
    package model

    import (
        "time"
        "github.com/google/uuid"
    )

    type ProjectRole string

    const (
        ProjectRoleOwner  ProjectRole = "owner"
        ProjectRoleMember ProjectRole = "member"
    )

    type ProjectUser struct {
        ProjectID uuid.UUID
        UserID    uuid.UUID
        Role      ProjectRole
        CreatedAt time.Time
    }

    // ProjectMember is the joined view returned by ListProjectUsers.
    type ProjectMember struct {
        UserID      uuid.UUID
        Email       string
        Name        string
        UserRole    Role        // global role (admin/user)
        ProjectRole ProjectRole // role within project (owner/member)
        AssignedAt  time.Time
    }
    ```
  - [ ] Create `backend/internal/domain/port/project_user_repository.go`:
    ```go
    package port

    import (
        "context"
        "github.com/google/uuid"
        "github.com/zakari/hopeitworks/backend/internal/domain/model"
    )

    type ProjectUserRepository interface {
        AddUser(ctx context.Context, projectID, userID uuid.UUID, role model.ProjectRole) (*model.ProjectUser, error)
        RemoveUser(ctx context.Context, projectID, userID uuid.UUID) error
        ListMembers(ctx context.Context, projectID uuid.UUID) ([]*model.ProjectMember, error)
        IsUserInProject(ctx context.Context, projectID, userID uuid.UUID) (bool, error)
        ListProjectsByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*model.Project, error)
        CountProjectsByUser(ctx context.Context, userID uuid.UUID) (int64, error)
    }
    ```

- [ ] [BACK] Task 4: Implement Postgres adapter for ProjectUserRepository (AC: #5, #6, #7)
  - [ ] Create `backend/internal/adapter/postgres/project_user_repo.go`
  - [ ] Implement `ProjectUserRepository` interface using sqlc-generated queries
  - [ ] Map between sqlc types and domain model types
  - [ ] Handle pgx errors:
    - Unique violation (23505) on AddUser -> `errors.NewConflict("project_user", "user already assigned")`
    - Foreign key violation (23503) on AddUser -> `errors.NewNotFound("project or user", id)`
  - [ ] Compile-time interface check: `var _ port.ProjectUserRepository = (*ProjectUserRepo)(nil)`

- [ ] [BACK] Task 5: Create ProjectUserService domain service (AC: #5, #6, #7, #8)
  - [ ] Create `backend/internal/domain/service/project_user_service.go`
  - [ ] Constructor: `NewProjectUserService(repo port.ProjectUserRepository, projectRepo port.ProjectRepository, userRepo port.UserRepository) *ProjectUserService`
  - [ ] Implement `AddUser(ctx, projectID, userID uuid.UUID, role ProjectRole) (*model.ProjectUser, error)`:
    - Validate project exists (via projectRepo.GetByID)
    - Validate user exists (via userRepo.GetByID)
    - Delegate to repo.AddUser
  - [ ] Implement `RemoveUser(ctx, projectID, userID uuid.UUID) error`:
    - Check membership exists (via repo.IsUserInProject), return NotFound if not
    - Delegate to repo.RemoveUser
  - [ ] Implement `ListMembers(ctx, projectID uuid.UUID) ([]*model.ProjectMember, error)`:
    - Validate project exists
    - Delegate to repo.ListMembers
  - [ ] Implement `IsUserInProject(ctx, projectID, userID uuid.UUID) (bool, error)`:
    - Delegate to repo.IsUserInProject
  - [ ] Implement `ListProjectsForUser(ctx, userID uuid.UUID, page, perPage int) (*ListResult, error)`:
    - Paginate using repo.ListProjectsByUser + repo.CountProjectsByUser
    - Return same `ListResult` struct used by ProjectService

- [ ] [BACK] Task 6: Create RBAC middleware for project-scoped routes (AC: #2, #3, #4)
  - [ ] Create `backend/internal/api/middleware/rbac.go`
  - [ ] Implement `RequireProjectAccess(projectUserRepo port.ProjectUserRepository) func(http.Handler) http.Handler`:
    - Extract `user_id` and `role` from context (set by Auth middleware)
    - If role == admin -> call `next.ServeHTTP` (bypass)
    - Extract `{id}` from chi URL params via `chi.URLParam(r, "id")`
    - Parse project_id as UUID; return 400 if invalid
    - Query `projectUserRepo.IsUserInProject(ctx, projectID, userID)`
    - If true -> call `next.ServeHTTP`
    - If false -> return 403 with error envelope:
      ```json
      {"error":{"code":"FORBIDDEN","message":"You are not a member of this project"}}
      ```
  - [ ] Write `writeForbidden(w http.ResponseWriter, msg string)` helper

- [ ] [BACK] Task 7: Create ProjectUserHandler and register routes (AC: #5, #6, #7)
  - [ ] Create `backend/internal/api/handler/project_user_handler.go`
  - [ ] Implement `POST /api/v1/projects/{id}/users` (admin only):
    - Parse request body: `{"user_id":"<uuid>", "role":"member"}` (role optional, default "member")
    - Call `projectUserService.AddUser`
    - Return HTTP 201 with created ProjectUser
  - [ ] Implement `DELETE /api/v1/projects/{id}/users/{user_id}` (admin only):
    - Parse project_id and user_id from URL params
    - Call `projectUserService.RemoveUser`
    - Return HTTP 204
  - [ ] Implement `GET /api/v1/projects/{id}/users` (admin or assigned user):
    - Call `projectUserService.ListMembers`
    - Return HTTP 200 with array of ProjectMember
  - [ ] These routes are NOT in the OpenAPI spec yet -- register them manually on the chi router (not via oapi-codegen). See Dev Notes for OpenAPI update guidance.
  - [ ] Request/response types defined locally in the handler file (not generated):
    ```go
    type AddProjectUserRequest struct {
        UserID uuid.UUID          `json:"user_id"`
        Role   model.ProjectRole  `json:"role,omitempty"`
    }

    type ProjectMemberResponse struct {
        UserID      uuid.UUID          `json:"user_id"`
        Email       string             `json:"email"`
        Name        string             `json:"name"`
        UserRole    string             `json:"user_role"`
        ProjectRole string             `json:"project_role"`
        AssignedAt  time.Time          `json:"assigned_at"`
    }
    ```

- [ ] [BACK] Task 8: Update ProjectHandler.ListProjects to filter by user assignment (AC: #8)
  - [ ] Modify `ProjectHandler` to accept `ProjectUserService` as an additional dependency:
    ```go
    type ProjectHandler struct {
        service     *service.ProjectService
        userService *service.ProjectUserService
    }
    func NewProjectHandler(svc *service.ProjectService, userSvc *service.ProjectUserService) *ProjectHandler
    ```
  - [ ] In `ListProjects`: check `middleware.IsAdmin(ctx)`:
    - If admin -> call `service.List(ctx, page, perPage)` (unchanged, returns all)
    - If not admin -> extract userID from context, call `userService.ListProjectsForUser(ctx, userID, page, perPage)`
  - [ ] Update `NewServer` and `main.go` accordingly (NewProjectHandler now takes 2 args)

- [ ] [BACK] Task 9: Wire everything into main.go and mount routes (AC: #2, #3, #4, #5, #6, #7, #8)
  - [ ] In `main.go`:
    - Instantiate `ProjectUserRepo` from sqlc queries
    - Instantiate `ProjectUserService` with repos
    - Update `NewProjectHandler` call to pass `ProjectUserService`
    - Instantiate `ProjectUserHandler`
    - Mount project_users routes manually on chi router:
      ```go
      r.Route("/api/v1/projects/{id}/users", func(r chi.Router) {
          r.Use(authmw.Auth(authService))
          r.Use(authmw.RequireProjectAccess(projectUserRepo))
          r.Get("/", projectUserHandler.ListMembers)
          r.Post("/", projectUserHandler.AddUser)       // admin check inside handler
          r.Delete("/{user_id}", projectUserHandler.RemoveUser) // admin check inside handler
      })
      ```
    - Apply `RequireProjectAccess` middleware to project-scoped routes (`/projects/{id}`, `/projects/{id}/users`) but NOT to `GET /projects` or `POST /projects` (those are list/create, not project-scoped)
  - [ ] Ensure existing oapi-codegen-generated routes for GET/PUT/DELETE `/projects/{id}` also pass through the RBAC middleware. This requires restructuring route mounting:
    - Move project-specific routes (GET/PUT/DELETE `/{id}`) to a subrouter with RBAC middleware
    - Keep `GET /projects` and `POST /projects` outside the RBAC subrouter

- [ ] [BACK] Task 10: Unit tests for RBAC middleware, ProjectUserService, and ProjectUserHandler (AC: #2, #3, #4, #5, #6, #7, #8)
  - [ ] Create `backend/internal/api/middleware/rbac_test.go`:
    - Test admin bypasses check (no DB call)
    - Test assigned user is allowed
    - Test unassigned user gets 403
    - Test invalid project_id returns 400
    - Test missing auth context returns 401/403
  - [ ] Create `backend/internal/domain/service/project_user_service_test.go`:
    - Test AddUser with valid project + user
    - Test AddUser with non-existent project -> NotFound
    - Test AddUser with non-existent user -> NotFound
    - Test AddUser duplicate -> Conflict
    - Test RemoveUser happy path
    - Test RemoveUser when not a member -> NotFound
    - Test ListMembers
    - Test ListProjectsForUser pagination
  - [ ] Create `backend/internal/api/handler/project_user_handler_test.go`:
    - Test POST /projects/{id}/users as admin -> 201
    - Test POST /projects/{id}/users as non-admin -> 403
    - Test DELETE /projects/{id}/users/{user_id} as admin -> 204
    - Test GET /projects/{id}/users as assigned user -> 200
    - Test GET /projects/{id}/users as unassigned user -> 403 (handled by middleware)
  - [ ] Use hand-written mocks for `ProjectUserRepository` (same pattern as existing mock repos)
  - [ ] All tests must be deterministic and pass with `go test ./... -short`

## Dev Notes

This story adds project-level RBAC to the backend. It introduces a join table (`project_users`), a new hexagonal slice (model/port/adapter/service/handler), and a chi middleware that gates access to project-scoped routes. It also modifies the existing `ProjectHandler.ListProjects` to filter by assignment for non-admin users.

### Dependencies

**Story 1-3 (done):** Auth middleware (`backend/internal/api/middleware/auth.go`) provides `UserIDFromContext()`, `RoleFromContext()`, `IsAdmin()`, `SetUserContext()`. The RBAC middleware chains after Auth.

**Story 1-5 (done):** Projects table (migration 000002), `ProjectRepository` port + postgres adapter, `ProjectService`, `ProjectHandler`, `Server` struct, response helpers in `helpers.go`, `DomainError` in `pkg/errors/`.

**Story 1-15 (done):** Services wiring in `main.go`, chi router setup.

### Architecture Requirements

**Hexagonal Architecture -- new files:**

```
backend/
├── migrations/
│   ├── 000003_create_project_users_table.up.sql
│   └── 000003_create_project_users_table.down.sql
├── queries/
│   └── project_users.sql
├── internal/
│   ├── domain/
│   │   ├── model/
│   │   │   └── project_user.go
│   │   ├── port/
│   │   │   └── project_user_repository.go
│   │   └── service/
│   │       └── project_user_service.go
│   │       └── project_user_service_test.go
│   ├── adapter/
│   │   └── postgres/
│   │       └── project_user_repo.go
│   └── api/
│       ├── handler/
│       │   └── project_user_handler.go
│       │   └── project_user_handler_test.go
│       └── middleware/
│           └── rbac.go
│           └── rbac_test.go
```

**Modified files:**
- `backend/internal/api/handler/project_handler.go` -- add ProjectUserService dependency, filter ListProjects
- `backend/internal/api/handler/server.go` -- update NewServer/NewProjectHandler signature
- `backend/cmd/api/main.go` -- wire new repos/services/handlers, mount routes

**Strict boundaries (unchanged):**
- `domain/model/` and `domain/port/` import NOTHING from adapter/ or api/
- `domain/service/` depends only on `domain/port/` interfaces
- `adapter/postgres/` implements `domain/port/` interfaces, imports sqlc-generated code
- `api/handler/` depends on `domain/service/`, never directly on adapter/
- `api/middleware/rbac.go` depends on `domain/port/ProjectUserRepository` for the `IsUserInProject` check (middleware needs DB access; this follows the same pattern as `auth.go` depending on `service.AuthService`)

### File Paths (exact)

| Purpose | Path |
|---------|------|
| Migration up | `backend/migrations/000003_create_project_users_table.up.sql` |
| Migration down | `backend/migrations/000003_create_project_users_table.down.sql` |
| sqlc queries | `backend/queries/project_users.sql` |
| Domain model | `backend/internal/domain/model/project_user.go` |
| Port interface | `backend/internal/domain/port/project_user_repository.go` |
| Domain service | `backend/internal/domain/service/project_user_service.go` |
| Postgres adapter | `backend/internal/adapter/postgres/project_user_repo.go` |
| RBAC middleware | `backend/internal/api/middleware/rbac.go` |
| Handler | `backend/internal/api/handler/project_user_handler.go` |
| Modified: project handler | `backend/internal/api/handler/project_handler.go` |
| Modified: server | `backend/internal/api/handler/server.go` |
| Modified: main.go | `backend/cmd/api/main.go` |

### Technical Specifications

**RBAC middleware design:**

The middleware is a standard chi middleware. It is applied ONLY to routes that contain a `{id}` URL param referring to a project ID. It does NOT apply to `GET /projects` (list) or `POST /projects` (create).

```go
// RequireProjectAccess returns chi middleware that checks if the authenticated
// user has access to the project identified by the {id} URL parameter.
// Admins bypass the check entirely.
func RequireProjectAccess(repo port.ProjectUserRepository) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. Extract user from context (set by Auth middleware)
            userID, ok := UserIDFromContext(r.Context())
            if !ok {
                writeForbidden(w, "Authentication required")
                return
            }

            // 2. Admin bypasses
            if IsAdmin(r.Context()) {
                next.ServeHTTP(w, r)
                return
            }

            // 3. Extract project_id from URL
            idStr := chi.URLParam(r, "id")
            projectID, err := uuid.Parse(idStr)
            if err != nil {
                // Bad UUID -> 400
                writeBadRequest(w, "Invalid project ID format")
                return
            }

            // 4. Check membership
            isMember, err := repo.IsUserInProject(r.Context(), projectID, userID)
            if err != nil {
                writeInternalError(w)
                return
            }
            if !isMember {
                writeForbidden(w, "You are not a member of this project")
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

**Route mounting strategy:**

The existing oapi-codegen-generated routes use `HandlerFromMuxWithBaseURL(server, r, "/api/v1")` which registers all routes on the router. To apply RBAC middleware only to project-scoped routes while keeping the generated server, the recommended approach is:

1. Keep using `HandlerFromMuxWithBaseURL` for list/create endpoints
2. Add a chi `Group` or `Route` with RBAC middleware for the manual project_users sub-routes
3. For the existing GET/PUT/DELETE `/projects/{id}` routes: since they go through the generated server, add the RBAC check INSIDE the `ProjectHandler` methods (GetProject, UpdateProject, DeleteProject) by injecting `ProjectUserRepository` or `ProjectUserService` -- OR restructure the router. The simplest approach for MVP is to check inside the handler methods, similar to how admin checks are done today.

**Recommended approach (handler-level RBAC for existing routes):**

Rather than restructuring the oapi-codegen router mounting, add an RBAC helper method to `ProjectHandler`:

```go
func (h *ProjectHandler) checkProjectAccess(ctx context.Context, projectID uuid.UUID) error {
    if middleware.IsAdmin(ctx) {
        return nil
    }
    userID, _ := middleware.UserIDFromContext(ctx)
    isMember, err := h.userService.IsUserInProject(ctx, projectID, userID)
    if err != nil {
        return err
    }
    if !isMember {
        return errors.NewForbidden("You are not a member of this project")
    }
    return nil
}
```

Then call `checkProjectAccess` at the top of `GetProject`, `UpdateProject`, `DeleteProject`.

For the NEW `/projects/{id}/users` routes (not in OpenAPI), use the chi middleware approach with `RequireProjectAccess`.

**OpenAPI spec update:**

The OpenAPI spec (`api/openapi.yaml`) does NOT yet have project_users endpoints. The dev agent MUST add the following to the spec:

1. New tag: `project-users`
2. New paths:
   - `GET /projects/{id}/users` -- listProjectUsers
   - `POST /projects/{id}/users` -- addProjectUser
   - `DELETE /projects/{id}/users/{user_id}` -- removeProjectUser
3. New schemas:
   - `AddProjectUserRequest` (user_id required, role optional)
   - `ProjectMember` (user_id, email, name, user_role, project_role, assigned_at)
   - `ProjectMemberList` (array of ProjectMember)
4. New response: `Forbidden` (403)

After updating the spec, run `cd backend && make generate` and `cd frontend && npm run generate-api`. If the frontend generation fails (Story 1-16 may not be fully wired), that is acceptable -- the important thing is the spec is updated. The handler can still be hand-written for now; it will be aligned with the generated interface when the spec is regenerated.

**If the dev agent prefers NOT to update the OpenAPI spec in this story** (to avoid cross-domain changes), the handler routes should be registered manually on the chi router and a TODO comment should reference this story for the spec update.

### Testing Requirements

**Unit test strategy:**
- All tests use hand-written mocks (no mockgen), same pattern as existing `project_handler_test.go` and `auth_test.go`
- Use `middleware.SetUserContext()` to inject auth context in handler tests
- Use table-driven tests for RBAC middleware scenarios
- Use `httptest.NewRecorder` + `httptest.NewRequest` for HTTP tests

**Manual verification checklist:**
1. Apply migration: `migrate -path backend/migrations -database $DB_URL up`
2. Verify table: `\d project_users` shows composite PK, FKs, indexes
3. Run `make generate` -- sqlc generates project_users query functions
4. `go build ./...` compiles successfully
5. Admin POST `/api/v1/projects/{id}/users` with `{"user_id":"..."}` -> 201
6. Admin GET `/api/v1/projects/{id}/users` -> 200 with members list
7. Admin DELETE `/api/v1/projects/{id}/users/{user_id}` -> 204
8. Non-admin (assigned) GET `/api/v1/projects/{id}` -> 200
9. Non-admin (NOT assigned) GET `/api/v1/projects/{id}` -> 403
10. Non-admin GET `/api/v1/projects` -> returns only assigned projects
11. Admin GET `/api/v1/projects` -> returns all projects (unchanged)
12. `go test ./... -short` passes

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 1.6]
- [Source: _bmad-output/planning-artifacts/architecture.md#Backend Architecture -- Hexagonal Structure]
- [Source: _bmad-output/planning-artifacts/architecture.md#Authentication & Security]
- [Source: api/openapi.yaml -- /projects endpoints]
- [Source: backend/internal/api/middleware/auth.go -- UserIDFromContext, RoleFromContext, IsAdmin]
- [Source: backend/internal/api/handler/project_handler.go -- existing RBAC pattern with IsAdmin]
- [Source: backend/internal/api/handler/helpers.go -- writeJSON, writeErrorResponse, toAPIProject]
- [Source: backend/pkg/errors/errors.go -- NewForbidden, NewNotFound, NewConflict]
