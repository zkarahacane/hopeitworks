# Story F-1.3: [BACK] Secure /api/v1/users endpoint to admin-only access

Status: ready-for-dev

## Story

As an admin,
I want user management endpoints to be restricted to admin users only,
so that regular users cannot view, edit, or delete other user accounts.

## Context

The user management handlers (`ListUsers`, `GetUser`, `UpdateUser`, `DeleteUser`) in
`backend/internal/api/handler/user_handler.go` already call the `requireAdmin(w, r)` helper
as their first action. This helper calls `middleware.IsAdmin(r.Context())`, which checks the
role injected by the global JWT auth middleware.

The vulnerability is one of **missing defense-in-depth**: the admin enforcement relies solely
on per-handler checks with no route-group-level middleware enforcing it. There is no
`RequireAdmin` chi middleware in `backend/internal/api/middleware/rbac.go`. If a handler
inadvertently omits the `requireAdmin` call (e.g. a future handler added to the same domain),
there is no safety net at the router level.

Additionally, the routes are registered through `handler.HandlerFromMuxWithBaseURL(server, r, "/api/v1")`
in `backend/cmd/api/main.go` (line 358), which mounts all oapi-codegen routes flat on the same
router with no sub-group applying admin-specific middleware to `/api/v1/users/*`.

The fix is to:
1. Add a `RequireAdmin` chi middleware to `backend/internal/api/middleware/rbac.go`
2. Register the `/api/v1/users` route group manually with that middleware applied, instead of
   relying on oapi-codegen's flat `HandlerFromMuxWithBaseURL` registration for those routes

## Acceptance Criteria (BDD)

**AC1: Non-admin GET /api/v1/users returns 403**
- **Given** a user with role "user" is authenticated (valid JWT cookie, role=user)
- **When** GET /api/v1/users is called
- **Then** response is 403 Forbidden with error code `FORBIDDEN`

**AC2: Admin GET /api/v1/users returns 200**
- **Given** a user with role "admin" is authenticated (valid JWT cookie, role=admin)
- **When** GET /api/v1/users is called
- **Then** response is 200 with a `UserList` payload

**AC3: Non-admin cannot modify users**
- **Given** a user with role "user" is authenticated
- **When** PUT /api/v1/users/{id} or DELETE /api/v1/users/{id} is called
- **Then** response is 403 Forbidden with error code `FORBIDDEN`

**AC4: Admin can modify users**
- **Given** a user with role "admin" is authenticated
- **When** PUT /api/v1/users/{id} or DELETE /api/v1/users/{id} is called
- **Then** operation succeeds (200 or 204)

**AC5: GET /api/v1/users/{id} for a non-admin returns 403**
- **Given** a user with role "user" is authenticated
- **When** GET /api/v1/users/{id} is called
- **Then** response is 403 Forbidden with error code `FORBIDDEN`

## Tasks / Subtasks

### Task 1 — Add `RequireAdmin` middleware to `backend/internal/api/middleware/rbac.go`

Add a new exported middleware function after `RequireProjectAccess`:

```go
// RequireAdmin returns chi middleware that allows only users with role "admin".
// Returns 403 Forbidden for any authenticated user without the admin role.
func RequireAdmin() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !IsAdmin(r.Context()) {
                writeForbidden(w, "Admin access required")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

No new imports are needed — `IsAdmin`, `writeForbidden`, and `http` are already available in the package.

### Task 2 — Apply `RequireAdmin` middleware at the route group level in `backend/cmd/api/main.go`

The oapi-codegen call `handler.HandlerFromMuxWithBaseURL(server, r, "/api/v1")` registers all routes
on the flat router `r`. The `/api/v1/users` routes must be registered on a sub-router that has
`authmw.RequireAdmin()` applied.

The oapi-codegen generated router (`HandlerFromMuxWithBaseURL`) does not support per-group
middleware injection. The solution is to mount the user management routes manually on a dedicated
chi sub-router **before** the `HandlerFromMuxWithBaseURL` call, and to ensure the `Server` methods
still delegate to `UserHandler` as they do today.

In `main.go`, after building `server` (line 338) and `r := chi.NewRouter()` with global middleware,
add a manually registered sub-router for user routes:

```go
// Admin-only: user management routes
r.Route("/api/v1/users", func(r chi.Router) {
    r.Use(authmw.RequireAdmin())
    r.Get("/", func(w http.ResponseWriter, req *http.Request) {
        server.ListUsers(w, req, handler.ListUsersParams{
            Page:    pageIntPtr(req.URL.Query().Get("page")),
            PerPage: pageIntPtr(req.URL.Query().Get("per_page")),
        })
    })
    r.Route("/{id}", func(r chi.Router) {
        r.Get("/", func(w http.ResponseWriter, req *http.Request) {
            server.GetUser(w, req, handler.IdPath(chi.URLParam(req, "id")))
        })
        r.Put("/", func(w http.ResponseWriter, req *http.Request) {
            server.UpdateUser(w, req, handler.IdPath(chi.URLParam(req, "id")))
        })
        r.Delete("/", func(w http.ResponseWriter, req *http.Request) {
            server.DeleteUser(w, req, handler.IdPath(chi.URLParam(req, "id")))
        })
    })
})
```

**Important:** Chi resolves routes in registration order. The manual `/api/v1/users` sub-router
must be registered **before** `handler.HandlerFromMuxWithBaseURL(server, r, "/api/v1")` so that
chi matches the more-specific admin-guarded route first. Verify the `IdPath` type — in the
generated code it is `type IdPath = uuid.UUID`, so the correct conversion is
`uuid.MustParse(chi.URLParam(req, "id"))` (with proper error handling for invalid UUIDs, returning
400 before calling the handler).

Alternatively, if the generated `HandlerFromMuxWithBaseURL` allows per-route override via chi's
route conflict resolution, confirm that behavior in `gen_server.go` before choosing the approach.

### Task 3 — Add `RequireAdmin` middleware unit tests to `backend/internal/api/middleware/rbac_test.go`

Add a `TestRequireAdmin` test following the same pattern as `TestRequireProjectAccess`:

```go
func TestRequireAdmin(t *testing.T) {
    mw := RequireAdmin()
    nextCalled := false
    next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
        nextCalled = true
        w.WriteHeader(http.StatusOK)
    })

    tests := []struct {
        name       string
        role       model.Role
        hasAuth    bool
        wantStatus int
        wantNext   bool
    }{
        {name: "admin is allowed", role: model.RoleAdmin, hasAuth: true, wantStatus: http.StatusOK, wantNext: true},
        {name: "user role gets 403", role: model.RoleUser, hasAuth: true, wantStatus: http.StatusForbidden, wantNext: false},
        {name: "no context gets 403", hasAuth: false, wantStatus: http.StatusForbidden, wantNext: false},
    }
    // table-driven test body — follow TestRequireProjectAccess pattern
}
```

### Task 4 — Remove redundant `requireAdmin` calls from user handler methods (optional cleanup)

Once the route-group middleware enforces admin access at the router level, the `requireAdmin(w, r)`
calls at the top of each handler method in `backend/internal/api/handler/user_handler.go` are
redundant. They can be removed to avoid duplicated logic.

**This task is optional for the P0 fix** — keeping the in-handler checks is a valid defense-in-depth
posture. Remove them only if the team prefers a single enforcement point. If removed, verify that
existing unit tests in `user_handler_test.go` still pass (they inject context directly without
going through the middleware stack, so the 403 assertions would need to be updated to expect
200/204 from the handler in isolation — the middleware test covers the 403 path).

### Task 5 — Verify with `golangci-lint`

```bash
cd backend && golangci-lint run ./...
```

The `errcheck` linter requires `_ = json.NewEncoder(w).Encode(...)` pattern for ignored return
values (already used consistently in the middleware package). Ensure `RequireAdmin` follows the
same pattern if it uses `json.NewEncoder`.

## Dev Notes

- Priority: P0 — security vulnerability (defense-in-depth gap, in-handler check exists but has no router-level backstop)
- `RequireAdmin` middleware lives in `backend/internal/api/middleware/rbac.go` alongside `RequireProjectAccess`
- `IsAdmin` helper is already in `backend/internal/api/middleware/auth.go` — no new role-check logic needed
- `writeForbidden` helper is already in `backend/internal/api/middleware/rbac.go` — reuse it
- The `profile` endpoints (`GET/PUT /api/v1/profile`, `POST /api/v1/profile/password`) must NOT
  be included in the admin-only group — they are for any authenticated user
- The `GET /api/v1/auth/me` endpoint is also not admin-only
- Chi route registration order matters: manually registered routes take precedence over
  `HandlerFromMuxWithBaseURL` when registered first on the same router
- `IdPath` in the generated types resolves to `uuid.UUID` — parse with proper error handling
- No database migrations required
- No OpenAPI spec changes required
- No frontend changes required
