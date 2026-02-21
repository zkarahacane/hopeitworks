---
key: S-TODO-3
epic: todo-app
scope: backend
depends_on:
  - S-TODO-1
---

# Complete Todo

As a user,
I want to mark a todo as completed,
So that I can track my progress.

## Acceptance Criteria

**Given** a todo exists
**When** I update it with completed=true
**Then** the todo is marked as completed and the updated record is returned

**Given** a todo does not exist
**When** I try to update it
**Then** a 404 error is returned
