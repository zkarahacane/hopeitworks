# Todo App — Pipeline Validation Baseline

A minimal todo application used as a reference project for validating the hopeitworks pipeline. This is an **evergreen baseline** — the CI pipeline must always pass.

## Purpose

This project serves as the ultimate smoke test for the hopeitworks platform. The pipeline runs stories against this project to verify the complete flow: agent execution, PR creation, CI polling, review, and merge.

## Architecture

- **Backend:** Node.js + Express API with Postgres
- **Frontend:** Static HTML/CSS/JS served by nginx
- **Database:** PostgreSQL 16

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/api/todos` | List all todos |
| GET | `/api/todos/:id` | Get a single todo |
| POST | `/api/todos` | Create a todo |
| PUT | `/api/todos/:id` | Update a todo |
| DELETE | `/api/todos/:id` | Delete a todo |

## Local Development

### With Docker Compose (recommended)

```bash
cd test-project
docker compose up
```

This starts:
- PostgreSQL on port 5433 (host) / 5432 (container)
- Backend API on port 3000
- Frontend on port 8080

Open http://localhost:8080 in your browser.

### Without Docker

1. Start PostgreSQL and create the `todo` database
2. Run the schema and seed data:
   ```bash
   psql -U todo -d todo -f init.sql
   psql -U todo -d todo -f seed.sql
   ```
3. Start the backend:
   ```bash
   cd backend
   npm install
   npm start
   ```
4. Serve the frontend (any static file server):
   ```bash
   npx serve frontend -l 8080
   ```

## Testing

```bash
cd backend
npm install
npm test
```

## Linting

```bash
cd backend
npm run lint
```

## Standalone Docker Build

```bash
docker build -t todo-app .
docker run -p 80:80 -e DATABASE_URL=postgres://todo:todo@host:5432/todo todo-app
```

## Project Structure

```
test-project/
├── README.md                   # This file
├── CLAUDE.md                   # Agent scoping document
├── Dockerfile                  # Combined app image (backend + frontend)
├── Dockerfile.backend          # Backend-only image (used by docker-compose)
├── docker-compose.yml          # Local dev stack
├── nginx.conf                  # Nginx config for standalone Dockerfile
├── nginx-compose.conf          # Nginx config for docker-compose
├── entrypoint.sh               # Standalone container entrypoint
├── init.sql                    # Database schema
├── seed.sql                    # Sample data (8 todos)
├── backend/
│   ├── package.json
│   ├── server.js               # Express API
│   ├── eslint.config.js
│   └── test/
│       └── todos.test.js       # Unit tests (Node.js test runner)
├── frontend/
│   ├── index.html
│   ├── styles.css
│   └── app.js                  # Vanilla JS todo UI
└── stories/                    # Reference stories for pipeline validation
    ├── S-TODO-1-add-todo.md
    ├── S-TODO-2-list-todos.md
    ├── S-TODO-3-complete-todo.md
    ├── S-TODO-4-delete-todo.md
    └── S-TODO-5-todo-ui.md
```

## Reference Stories

The `stories/` directory contains 5 markdown stories with frontmatter fields (`key`, `epic`, `scope`, `depends_on`). These stories form a dependency DAG used to validate the platform's DAG scheduler and story import functionality.

Dependency graph:
```
S-TODO-1 (Add Todo)
├── S-TODO-2 (List Todos) ──┐
├── S-TODO-3 (Complete Todo) ├── S-TODO-5 (Todo UI)
└── S-TODO-4 (Delete Todo) ──┘
```
