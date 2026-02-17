---
key: S-TODO-4
epic: todo-app
scope: backend
depends_on:
  - S-TODO-1
---

# Delete Todo

As a user,
I want to delete a todo,
So that I can remove tasks I no longer need.

## Acceptance Criteria

**Given** a todo exists
**When** I delete it by id
**Then** the todo is removed and a 204 response is returned

**Given** a todo does not exist
**When** I try to delete it
**Then** a 404 error is returned
