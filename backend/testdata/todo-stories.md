---
key: TODO-1
epic: E-1
scope: backend
status: backlog
---
# Add create todo endpoint

## Acceptance Criteria
- POST /todos creates a new todo item
- Returns 201 with the created todo
- Validates required fields

---
key: TODO-2
epic: E-1
scope: backend
status: backlog
---
# Add list todos endpoint

## Acceptance Criteria
- GET /todos returns all todo items
- Supports pagination
- Returns 200 with list

---
key: TODO-3
epic: E-1
depends_on:
  - TODO-1
scope: backend
status: backlog
---
# Add update todo endpoint

## Acceptance Criteria
- PUT /todos/{id} updates an existing todo
- Returns 200 with updated todo
- Returns 404 if todo not found

---
key: TODO-4
epic: E-1
scope: backend
status: backlog
---
# Add delete todo endpoint

## Acceptance Criteria
- DELETE /todos/{id} deletes a todo
- Returns 204 on success
- Returns 404 if todo not found

---
key: TODO-5
epic: E-1
depends_on:
  - TODO-2
  - TODO-3
scope: frontend
status: backlog
---
# Add todo list UI

## Acceptance Criteria
- Displays list of todos fetched from API
- Supports creating, updating, and deleting todos
- Shows loading and error states
