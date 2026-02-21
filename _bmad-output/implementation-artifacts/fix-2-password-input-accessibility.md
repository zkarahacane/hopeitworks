# Story fix-2: Fix PrimeVue Password component accessibility in LoginView

Status: ready-for-dev

## Story

As a user,
I want the password field to be properly labeled for screen readers and test automation,
so that the login form is accessible.

## Bug

In `frontend/src/views/LoginView.vue`, the `<Password>` PrimeVue component uses `id="password"` which sets the id on the wrapper `<div>`, not on the actual `<input>`. The `<label for="password">` points to the div, not the input. The actual input gets a PrimeVue-generated id like `pv_id_9` with no `aria-label` or `aria-labelledby`. Playwright's `getByLabel(/password/i)` finds 0 results. `page.locator('input[type="password"]')` works.

**Fix:** Change `id="password"` to `inputId="password"` on the `<Password>` component. PrimeVue forwards `inputId` to the inner `<input>`.

## Acceptance Criteria (BDD)

**AC1: Password input has correct id**
- **Given** the login page is rendered
- **When** inspecting the DOM
- **Then** the `<input type="password">` element has `id="password"` (not the wrapper div)

**AC2: Label association is correct**
- **Given** the login page is rendered
- **When** inspecting the DOM
- **Then** `<label for="password">` correctly resolves to the `<input>` element

**AC3: Playwright getByLabel works**
- **Given** the login page is loaded in a browser
- **When** using `page.getByLabel(/password/i)`
- **Then** exactly 1 element is returned

**AC4: Screen reader announces label**
- **Given** the password input is focused
- **When** a screen reader reads the focused element
- **Then** it announces "Password"

## Tasks / Subtasks

- [ ] Task 1: Fix the PrimeVue Password component prop
  - [ ] Change `id="password"` to `inputId="password"` in `frontend/src/views/LoginView.vue`
- [ ] Task 2: Verify rendered HTML
  - [ ] Confirm the inner `<input>` has `id="password"` and the `<label>` association is correct
- [ ] Task 3: Run existing E2E login tests
  - [ ] Confirm all login-related Playwright tests still pass after the change
