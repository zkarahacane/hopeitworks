# Story F-1.4: [SHARED] Fix forgot/reset password end-to-end flow

Status: ready-for-dev

## Story

As a user who forgot their password,
I want to reset it via email and then log in with the new password,
so that I can regain access to my account.

## Acceptance Criteria (BDD)

**AC1: Forgot password page loads without redirect**
- **Given** user is not authenticated
- **When** navigating to `/forgot-password`
- **Then** the page renders with email input form, no redirect to `/login`

**AC2: Reset password page loads with token**
- **Given** user has a valid reset token
- **When** navigating to `/reset-password?token=xxx`
- **Then** the page renders with new password form, no redirect to `/login`

**AC3: Password reset stores correct hash**
- **Given** user submits a new password via the reset form
- **When** the backend processes the reset
- **Then** the `password_hash` column in the `users` table is updated with a valid bcrypt hash of the new password

**AC4: Login works after reset**
- **Given** user has successfully reset their password (API returned 200)
- **When** they attempt to log in with the new password
- **Then** login succeeds with HTTP 200 and a valid session cookie is set

**AC5: Reset token expires**
- **Given** a reset token older than 1 hour (or a manually expired token)
- **When** user submits the reset form with that token
- **Then** the API returns 400 with error code `RESET_TOKEN_EXPIRED`

## Root Cause Analysis

### Backend — critical bug (causes AC4 failure)

`auth_service.go` `ResetPassword()` (line 240) calls `s.repo.Update(ctx, user)` after computing the new bcrypt hash. The underlying `UpdateUser` SQL query uses `COALESCE(sqlc.narg(...))` only for `name`, `email`, and `role` columns — **`password_hash` is not in that query**. The new hash is silently discarded, leaving the old password in the database.

The fix is one line: replace the `GetByID` + `Update` pattern with a direct call to `s.repo.UpdatePasswordHash(ctx, prt.UserID, string(hash))`.

The `UpdatePasswordHash` port method (`port/user_repository.go` line 18), the SQL query (`queries/users.sql` lines 27-31: `UpdateUserPasswordHash`), and the adapter implementation (`adapter/postgres/user_repository.go` lines 85-89) all exist and are correct. Only the service call is wrong.

Note: the existing unit test `TestResetPassword_ValidToken` passes because the mock's `Update()` simulates `password_hash` propagation (mock lines 80-81), masking the real-DB bug.

### Frontend — routes are already correct, guard logic is correct

`router/index.ts` lines 29-38: both `/forgot-password` and `/reset-password` already carry `meta: { requiresAuth: false }`.

`router/guards.ts` line 22: `const requiresAuth = to.meta.requiresAuth !== false` — routes explicitly set to `false` are treated as public.

`api/client.ts` lines 9-13: the `authMiddleware` skips the redirect-to-login for any path starting with `/api/v1/auth/`. The `checkAuth()` call (`GET /auth/me` → `/api/v1/auth/me`) on first navigation returns 401 for unauthenticated users but does not trigger a redirect, so the guard correctly reaches the unauthenticated state and allows the public route.

**Frontend task**: add unit tests confirming the guard allows unauthenticated access to these routes, and add an E2E smoke test covering the full reset flow.

## Tasks / Subtasks

### Task 1 — Backend: fix `ResetPassword` to call `UpdatePasswordHash` (CRITICAL)

File: `backend/internal/domain/service/auth_service.go`

Current code (lines 234-244):
```go
hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
if err != nil {
    return err
}

user, err := s.repo.GetByID(ctx, prt.UserID)
if err != nil {
    return err
}
user.PasswordHash = string(hash)
if _, err := s.repo.Update(ctx, user); err != nil {
    return err
}

return s.tokenRepo.MarkUsed(ctx, prt.ID)
```

Replace with:
```go
hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
if err != nil {
    return err
}

if err := s.repo.UpdatePasswordHash(ctx, prt.UserID, string(hash)); err != nil {
    return err
}

return s.tokenRepo.MarkUsed(ctx, prt.ID)
```

This removes the unnecessary `GetByID` call and uses the dedicated `UpdateUserPasswordHash` SQL query which correctly updates the `password_hash` column.

### Task 2 — Backend: fix `TestResetPassword_ValidToken` to catch the real-DB path

File: `backend/internal/domain/service/auth_service_test.go`

The mock `Update()` currently simulates password propagation (lines 80-81), masking the bug. After the fix in Task 1, the test still passes because `UpdatePasswordHash` is now called instead. However, add a new test that verifies `Update()` is NOT called during reset (i.e., ensure the mock's `Update` is never invoked in the reset flow), so any future regression where someone switches back to `Update` is caught.

Add a test case:
```go
func TestResetPassword_UsesUpdatePasswordHash_NotUpdate(t *testing.T) {
    repo := newMockRepo()
    // Track whether Update() is called
    updateCalled := false
    // Wrap mock to detect Update() calls
    // ...register user, trigger forgot password, call ResetPassword...
    // Assert updateCalled == false
}
```

### Task 3 — Backend: add integration test for full reset flow against real DB

File: `backend/internal/domain/service/auth_service_test.go` or a new `backend/internal/adapter/postgres/auth_integration_test.go`

Tag with `Integration` in test name. Use `testutil.NewTestDB(t)` (testcontainers). Steps:
1. Create user via `UserRepository.Create`
2. Create a valid reset token via `PasswordResetTokenRepository.Create`
3. Call `AuthService.ResetPassword` with the token and new password
4. Query the `users` table directly and verify `password_hash` is a valid bcrypt hash of the new password
5. Call `AuthService.Login` with the new password — assert no error
6. Call `AuthService.Login` with the old password — assert `ErrInvalidCredentials`

### Task 4 — Frontend: add router guard unit test for public auth routes

File: `frontend/src/router/__tests__/guards.spec.ts` (create if it does not exist)

Using Vitest + `@vue/test-utils`, test that:
- Navigating to `/forgot-password` when `isAuthenticated === false` does NOT redirect
- Navigating to `/reset-password?token=abc` when `isAuthenticated === false` does NOT redirect
- Navigating to `/` when `isAuthenticated === false` redirects to `/login`

Mock `useAuthStore` to control `isAuthenticated` state.

### Task 5 — Frontend: E2E smoke test for reset password flow

File: `frontend/e2e/tests/auth.spec.ts` (add to existing file or create)

Using Playwright against the local dev stack (MailHog at `http://localhost:8025`):
1. Navigate to `/forgot-password`, enter a known test user email, submit
2. Poll MailHog API (`GET http://localhost:8025/api/v2/messages`) to retrieve the reset email
3. Extract the `?token=` value from the reset link in the email body
4. Navigate to `/reset-password?token=<extracted_token>`
5. Enter a new password (min 8 chars), submit
6. Assert redirect to `/login?reset=success`
7. Log in with the new password, assert redirect to `/` (dashboard)
8. Log out, attempt login with old password, assert error message is shown

## Dev Notes

- Priority: P1 — users are completely blocked from recovering accounts
- The backend fix (Task 1) is a one-line change; Tasks 2-3 add regression coverage
- Depends on F-1-1 (auth redirect fix in `client.ts`) if that story changes the 401 middleware behaviour — verify no conflict before merging
- Frontend routes `/forgot-password` and `/reset-password` already have `meta: { requiresAuth: false }` — no route config change needed
- Backend `UpdatePasswordHash` port, query, and adapter are all correctly implemented — only the service call is wrong
- Test with MailHog at `http://localhost:8025` for the reset email in dev/E2E environments
- The `base64.URLEncoding` tokens generated by `generateSecureToken()` contain `=` padding characters; confirm the URL is not double-encoded in the frontend when building the reset link
- After fix: run `go test ./internal/domain/service/... -run TestResetPassword` and `golangci-lint run ./...` before committing
