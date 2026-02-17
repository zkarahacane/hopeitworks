---
key: S-TODO-2
epic: todo-app
scope: backend
depends_on:
  - S-TODO-1
---

# List Todos

As a user,
I want to see all my todos,
So that I can review what I need to do.

## Acceptance Criteria

**Given** todos exist in the database
**When** I request the list of todos
**Then** all todos are returned sorted by creation date (newest first)

**Given** no todos exist
**When** I request the list
**Then** an empty array is returned
