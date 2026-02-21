# Story fix-4: Fix E2E test helpers and selectors for real backend tests

Status: ready-for-dev

## Story

As a developer,
I want the E2E smoke tests to use correct selectors and API patterns,
so that tests pass when the app works correctly.

## Bugs found

1. `loginViaUI` in `helpers/auth.ts` uses `getByLabel(/password/i)` which doesn't work with PrimeVue Password — use `input[type="password"]` as fallback.
2. `loginViaAPI` sets cookies on `context` but some tests use the separate `request` fixture — use `context.request` instead.
3. `smoke-navigation.spec.ts` uses `getByRole('link', ...)` for sidebar items but the sidebar uses `<button>` elements, not `<a>` links.

## Acceptance Criteria (BDD)

**AC1: loginViaUI fills the password field**
- **Given** the login page is loaded
- **When** `loginViaUI` is called
- **Then** the password field is successfully filled (fallback to `input[type="password"]` if `getByLabel` fails)

**AC2: API calls use context.request**
- **Given** a test using `loginViaAPI` followed by API calls
- **When** the test runs
- **Then** `context.request` is used for all API calls (not the separate `request` fixture) so cookies are correctly applied

**AC3: Navigation tests use correct role**
- **Given** the sidebar renders `<button>` elements for navigation
- **When** navigation tests query sidebar items
- **Then** `getByRole('button', ...)` is used instead of `getByRole('link', ...)`

**AC4: All 28 smoke tests have correct selectors**
- **Given** the app is running and healthy
- **When** the full smoke suite is executed
- **Then** no test fails due to selector issues (failures may still occur for app-level bugs, not selector bugs)

## Tasks / Subtasks

- [ ] Task 1: Fix loginViaUI password selector
  - [ ] In `frontend/e2e/real-tests/helpers/auth.ts`, try `getByLabel(/password/i)` first, fall back to `page.locator('input[type="password"]')`
- [ ] Task 2: Fix loginViaAPI to use context.request
  - [ ] Audit all test files that call `loginViaAPI` then make API calls via the `request` fixture
  - [ ] Replace `request` fixture usage with `context.request` in those tests
  - [ ] Fix `smoke-login.spec.ts` AUDIT test specifically
- [ ] Task 3: Fix navigation test selectors
  - [ ] In `frontend/e2e/real-tests/smoke-navigation.spec.ts`, replace `getByRole('link', ...)` with `getByRole('button', ...)` for sidebar items
- [ ] Task 4: Audit remaining spec files
  - [ ] Review all spec files under `frontend/e2e/real-tests/` for similar selector mismatches
  - [ ] Fix any additional issues found
