# Story 10.1: Todo app reference project structure

Status: ready-for-dev

## Story

As a developer,
I want a reference project with a simple app,
so that I can validate the pipeline with a known baseline.

## Acceptance Criteria (BDD)

**AC1: test-project directory structure exists**
- **Given** the test-project/ directory is initialized
- **When** I examine the directory layout
- **Then** I see backend/, frontend/, Dockerfile, docker-compose.yml, and README.md

**AC2: Backend Express server starts successfully**
- **Given** I have the test-project cloned locally
- **When** I run `cd test-project/backend && npm install && node index.js`
- **Then** the server starts on port 3000 and exposes CRUD endpoints (GET /todos, POST /todos, PUT /todos/:id, DELETE /todos/:id)

**AC3: Frontend HTML page loads and connects to backend**
- **Given** the Express backend is running on port 3000
- **When** I open test-project/frontend/index.html in a browser
- **Then** the page loads, connects to the backend via fetch(), and can create, read, update, and delete todos

**AC4: Docker compose stack runs end-to-end**
- **Given** docker-compose.yml is configured in test-project/
- **When** I run `docker compose -f test-project/docker-compose.yml up`
- **Then** both app and postgres containers start successfully, and the app is accessible on http://localhost:3000

**AC5: README documents the project structure and setup**
- **Given** test-project/README.md exists
- **When** I read the README
- **Then** it explains the project purpose, lists local setup commands, and documents the directory structure

## Tasks / Subtasks

- [ ] Task 1: Create test-project/ directory structure (AC: #1)
  - [ ] Create test-project/ at repository root
  - [ ] Create test-project/backend/ directory
  - [ ] Create test-project/frontend/ directory
  - [ ] Verify directory structure is in place

- [ ] Task 2: Create Node.js Express backend (AC: #2, #4)
  - [ ] Create test-project/backend/package.json with dependencies: express, pg, body-parser, cors
  - [ ] Create test-project/backend/index.js with Express server
  - [ ] Implement GET /todos endpoint (fetch all todos from Postgres)
  - [ ] Implement POST /todos endpoint (insert new todo, return created record)
  - [ ] Implement PUT /todos/:id endpoint (update todo by id)
  - [ ] Implement DELETE /todos/:id endpoint (delete todo by id)
  - [ ] Configure Postgres connection via pg module (use env vars: DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD)
  - [ ] Add error handling for database connections and queries
  - [ ] Add CORS middleware to allow requests from frontend
  - [ ] Create todos table on startup (CREATE TABLE IF NOT EXISTS)

- [ ] Task 3: Create frontend HTML page (AC: #3, #4)
  - [ ] Create test-project/frontend/index.html as single-file app (no build step)
  - [ ] Add basic HTML structure with form to create todos
  - [ ] Add button to fetch all todos from backend
  - [ ] Add buttons to update and delete todos inline
  - [ ] Implement fetch() calls to GET /todos, POST /todos, PUT /todos/:id, DELETE /todos/:id
  - [ ] Add basic CSS styling (inline <style> tag)
  - [ ] Add error handling for failed requests
  - [ ] Display todos in a simple list

- [ ] Task 4: Create Dockerfile (AC: #4)
  - [ ] Create test-project/Dockerfile with multi-stage build
  - [ ] Stage 1: Builder - use node:20-alpine
  - [ ] Copy package*.json and run npm install
  - [ ] Stage 2: Runtime - use node:20-alpine
  - [ ] Copy node_modules and backend/ code from builder
  - [ ] Expose port 3000
  - [ ] Set ENTRYPOINT to run `node /app/index.js`

- [ ] Task 5: Create docker-compose.yml (AC: #4)
  - [ ] Create test-project/docker-compose.yml
  - [ ] Define postgres service: postgres:16-alpine
  - [ ] Configure postgres environment: POSTGRES_DB=todos_db, POSTGRES_USER=todos_user, POSTGRES_PASSWORD=todos_password
  - [ ] Add postgres volume for data persistence
  - [ ] Add postgres healthcheck (pg_isready)
  - [ ] Define app service: build from test-project/ Dockerfile
  - [ ] Configure app environment: DB_HOST=postgres, DB_PORT=5432, DB_NAME=todos_db, DB_USER=todos_user, DB_PASSWORD=todos_password
  - [ ] Map app port 3000:3000
  - [ ] Add app depends_on postgres with condition: service_healthy
  - [ ] Create shared network for services
  - [ ] Add restart policies (on-failure)

- [ ] Task 6: Create README.md (AC: #5)
  - [ ] Create test-project/README.md
  - [ ] Document project purpose: reference app for validating hopeitworks pipeline
  - [ ] Document local setup: backend Node.js requirements, npm install, node index.js
  - [ ] Document docker compose setup: docker compose up
  - [ ] Document API endpoints (GET, POST, PUT, DELETE /todos)
  - [ ] Document directory structure and file purposes
  - [ ] Include note: "This project is intentionally simple and minimal — no TypeScript, no bundler. Used for agent pipeline validation."

## Dev Notes

This story creates a **reference project** at the repo root (`test-project/`), independent of backend/ and frontend/ domains. It serves as a validation baseline for the hopeitworks pipeline.

### Key Design Principles

1. **Minimal scope**: No TypeScript, no build tools, no frameworks beyond Express. Keeps testing focused.
2. **Isolated from main project**: Lives at repo root, not inside backend/ or frontend/. Clean separation.
3. **Full pipeline test**: Backend (Node), frontend (vanilla JS/HTML), database (Postgres), Docker, docker-compose.
4. **Agent testing baseline**: Pipeline will clone this repo, create feature branches, and assign Claude agents to implement "stories" targeting test-project files. This validates the entire workflow end-to-end.

### Architecture Requirements

**Exact Project Structure:**
```
test-project/
├── backend/
│   ├── index.js              # Express server with CRUD endpoints
│   └── package.json          # Dependencies: express, pg, body-parser, cors
├── frontend/
│   └── index.html            # Single-file app with inline CSS/JS
├── Dockerfile                # Multi-stage: node:20-alpine builder + runtime
├── docker-compose.yml        # app (Node) + postgres:16-alpine
└── README.md                 # Project documentation
```

### Technical Specifications

**Node.js Backend:**
- Version: 20-alpine (matches Dockerfile)
- Framework: Express.js (minimal)
- Database: Postgres via pg module
- Port: 3000 (default Node.js dev port)
- Endpoints:
  - `GET /todos` — fetch all todos (SELECT * FROM todos)
  - `POST /todos` — create todo, return created record (INSERT INTO todos ...)
  - `PUT /todos/:id` — update todo (UPDATE todos SET ... WHERE id=?)
  - `DELETE /todos/:id` — delete todo (DELETE FROM todos WHERE id=?)
- Connection: Use environment variables for DB connection (DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD)
- Schema: Auto-create on startup:
  ```sql
  CREATE TABLE IF NOT EXISTS todos (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    completed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
  );
  ```

**Frontend:**
- Type: Single HTML file, no build step
- Technology: Vanilla JavaScript (fetch API) + inline CSS
- Server: Served via browser (file:// or simple HTTP server)
- Interactions:
  - Display list of todos fetched from backend
  - Form to add new todo (POST /todos)
  - Checkbox to toggle completed status (PUT /todos/:id)
  - Delete button (DELETE /todos/:id)
  - Auto-refresh after each action
- Error handling: Log errors to console and display user-friendly messages

**PostgreSQL Configuration:**
- Version: 16 (official Docker image: postgres:16-alpine)
- Database name: todos_db
- Default user: todos_user
- Default password: todos_password
- Port exposed: 5432 (host:5432 → container:5432)
- Healthcheck: `pg_isready -U todos_user`
- Volume: postgres_data (named volume for persistence)

**Docker Compose:**
- Compose version: 3.8
- Services: postgres, app
- Network: shared network for inter-service communication
- Restart policy: on-failure for both services
- Depends_on: app depends on postgres (service_healthy condition)

### File Content Guidelines

**backend/package.json:**
```json
{
  "name": "todo-app-backend",
  "version": "1.0.0",
  "description": "Minimal Node.js Express backend for todo app",
  "main": "index.js",
  "scripts": {
    "start": "node index.js"
  },
  "dependencies": {
    "express": "^4.18.2",
    "pg": "^8.11.3",
    "body-parser": "^1.20.2",
    "cors": "^2.8.5"
  }
}
```

**backend/index.js:**
- Initialize Express app
- Configure CORS middleware
- Configure body-parser for JSON
- Setup Postgres connection pool from env vars
- Create todos table on startup (IF NOT EXISTS)
- Implement 4 CRUD endpoints
- Add error handling and logging
- Start server on port 3000

**frontend/index.html:**
- Single HTML file with inline CSS and JavaScript
- Display list of todos (fetched via GET /todos)
- Form to add new todo (POST /todos)
- Inline edit/delete buttons for each todo
- Fetch calls to http://localhost:3000/todos (or backend service name in docker-compose)
- Basic CSS: simple layout, readable fonts
- Error handling: catch and log fetch errors

**Dockerfile:**
```dockerfile
# Stage 1: Builder
FROM node:20-alpine AS builder
WORKDIR /app
COPY backend/package*.json ./
RUN npm install

# Stage 2: Runtime
FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/node_modules ./node_modules
COPY backend/ ./
EXPOSE 3000
CMD ["node", "index.js"]
```

**docker-compose.yml:**
- postgres service: image postgres:16-alpine, environment variables, volume, healthcheck
- app service: build from ./Dockerfile, environment variables, port mapping, depends_on with service_healthy
- shared network: bridge network for service communication

**README.md:**
- Heading: "Todo App Reference Project"
- Purpose section: Explain this is a validation baseline for hopeitworks pipeline
- Local setup section: npm install, node index.js, verify on http://localhost:3000
- Docker setup section: docker compose up, verify on http://localhost:3000
- API documentation section: list all endpoints with example requests/responses
- Directory structure section: brief description of each file/folder
- Note: "This app is intentionally minimal (no TypeScript, no bundler). Used for testing the hopeitworks pipeline end-to-end."

### Testing Requirements

**Manual verification checklist:**
1. `cd test-project/backend && npm install` completes without errors
2. `cd test-project/backend && node index.js` starts server and logs "Server running on port 3000"
3. `curl http://localhost:3000/todos` returns empty array `[]`
4. `curl -X POST http://localhost:3000/todos -H "Content-Type: application/json" -d '{"title":"Test"}' ` returns created todo with id, title, completed=false
5. `curl http://localhost:3000/todos` returns array with the created todo
6. `docker compose -f test-project/docker-compose.yml up` starts both services
7. `curl http://localhost:3000/todos` returns todos from docker environment
8. Open test-project/frontend/index.html in browser and verify:
   - Page loads without errors
   - Fetch request to backend succeeds (check browser console)
   - Can create, update, delete todos via UI

### Notes on Pipeline Integration

The hopeitworks pipeline will:
1. Clone this test-project repo as a separate project
2. Create feature branches: `feat/TEST-1-something`, `feat/TEST-2-something`, etc.
3. Assign Claude agents to implement "stories" targeting test-project files
4. Run CI (lint, test, docker build) against the test-project
5. Validate the pipeline output by comparing against this known baseline

This ensures the pipeline itself works correctly before being applied to real customer projects.

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Epic 10: Reference Test Project & Validation]
- [Source: _bmad-output/planning-artifacts/epics.md#Story 10.1]
- [Dependency: Story 3-8 (agent-run action)]
- [Dependency: Story 6-1 (pipeline configs)]

## Dev Agent Record

### Agent Model Used

To be filled by implementing agent.

### Debug Log References

To be filled by implementing agent.

### Completion Notes List

To be filled by implementing agent.

### File List

To be filled by implementing agent. Expected files:
- `/workspace/test-project/backend/index.js`
- `/workspace/test-project/backend/package.json`
- `/workspace/test-project/frontend/index.html`
- `/workspace/test-project/Dockerfile`
- `/workspace/test-project/docker-compose.yml`
- `/workspace/test-project/README.md`

## Change Log

- **2026-02-18**: Story file created for wave 13
