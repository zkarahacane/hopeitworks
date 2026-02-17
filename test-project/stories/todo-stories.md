---
key: TODO-1
epic: Todo App
depends_on: []
scope: backend
status: backlog
---
# Add create todo endpoint

**Given** the API is running
**When** I send a POST to /api/todos with a valid title
**Then** a new todo is created with status "pending"

**Given** the API is running
**When** I send a POST to /api/todos without a title
**Then** I receive a 400 validation error

---
key: TODO-2
epic: Todo App
depends_on: []
scope: backend
status: backlog
---
# Add list todos endpoint

**Given** todos exist in the database
**When** I send a GET to /api/todos
**Then** I receive a paginated list of todos

**Given** no todos exist
**When** I send a GET to /api/todos
**Then** I receive an empty list with total count 0

---
key: TODO-3
epic: Todo App
depends_on:
  - TODO-1
scope: backend
status: backlog
---
# Add update todo endpoint

**Given** a todo exists
**When** I send a PUT to /api/todos/:id with updated fields
**Then** the todo is updated and returned

**Given** I want to mark a todo as complete
**When** I send a PUT with completed=true
**Then** the todo status changes to "done"

---
key: TODO-4
epic: Todo App
depends_on:
  - TODO-1
scope: backend
status: backlog
---
# Add delete todo endpoint

**Given** a todo exists
**When** I send a DELETE to /api/todos/:id
**Then** the todo is removed from the database

**Given** a todo does not exist
**When** I send a DELETE to /api/todos/:id
**Then** I receive a 404 error

---
key: TODO-5
epic: Todo App
depends_on:
  - TODO-2
  - TODO-3
scope: frontend
status: backlog
---
# Add todo list UI with completion toggle

**Given** I am on the todo list page
**When** the page loads
**Then** I see all todos with their status

**Given** a todo is displayed
**When** I click the completion toggle
**Then** the todo status is updated via API
