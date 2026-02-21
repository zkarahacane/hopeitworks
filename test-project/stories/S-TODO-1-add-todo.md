---
key: S-TODO-1
epic: todo-app
scope: backend
depends_on: []
---

# Add Todo

As a user,
I want to add a new todo item,
So that I can track tasks I need to complete.

## Acceptance Criteria

**Given** the todo app is running
**When** I submit a new todo with a title
**Then** the todo is saved to the database and returned with an id

**Given** I submit an empty title
**When** the API processes the request
**Then** a 400 error is returned with a validation message
