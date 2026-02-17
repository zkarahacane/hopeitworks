---
key: S-TODO-5
epic: todo-app
scope: frontend
depends_on:
  - S-TODO-2
  - S-TODO-3
  - S-TODO-4
---

# Todo UI

As a user,
I want a web interface for managing todos,
So that I can interact with the app through a browser.

## Acceptance Criteria

**Given** the frontend is loaded
**When** the page renders
**Then** existing todos are fetched and displayed

**Given** I type a title and click Add
**When** the form is submitted
**Then** a new todo appears in the list

**Given** I check the checkbox on a todo
**When** the checkbox state changes
**Then** the todo is marked as completed via the API

**Given** I click the delete button
**When** the button is clicked
**Then** the todo is removed from the list and the API
