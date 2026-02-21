# Story fix-3: Add catch-all 404 route to Vue Router

Status: ready-for-dev

## Story

As a user,
I want to see a meaningful error page when I navigate to an invalid URL,
so that I'm not confused by a blank page.

## Bug

The Vue Router in `frontend/src/router/index.ts` has no catch-all route. Navigating to `/this-does-not-exist` renders nothing useful — no redirect, no 404 message.

## Acceptance Criteria (BDD)

**AC1: Catch-all route exists**
- **Given** the Vue Router configuration in `frontend/src/router/index.ts`
- **When** inspecting the route definitions
- **Then** a catch-all route `/:pathMatch(.*)*` is present

**AC2: Invalid URL shows a 404 message**
- **Given** the application is loaded
- **When** navigating to a URL that matches no defined route (e.g., `/this-does-not-exist`)
- **Then** a "Page not found" message (or equivalent) is displayed

**AC3: 404 page has navigation back to dashboard**
- **Given** the 404 page is displayed
- **When** the user reads the page
- **Then** a link or button is present to navigate back to the dashboard

**AC4: 404 page renders inside AppShell**
- **Given** the 404 page is displayed
- **When** inspecting the layout
- **Then** the sidebar and header from AppShell are visible

## Tasks / Subtasks

- [ ] Task 1: Create the NotFoundView component
  - [ ] Create `frontend/src/views/NotFoundView.vue`
  - [ ] Use PrimeVue components (EmptyState pattern or similar) for styling
  - [ ] Include a link/button to navigate back to the dashboard
- [ ] Task 2: Register the catch-all route
  - [ ] Add `{ path: '/:pathMatch(.*)*', component: NotFoundView }` to `frontend/src/router/index.ts`
  - [ ] Ensure the catch-all is the last route in the array
- [ ] Task 3: Verify routing behaviour
  - [ ] Confirm the catch-all does not interfere with existing routes
  - [ ] Confirm the 404 page renders within AppShell (sidebar and header visible)
