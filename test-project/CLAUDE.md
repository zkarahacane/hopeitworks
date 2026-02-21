# Todo App — Agent Scoping

## Scope

This is a **SHARED** reference project (not backend-only or frontend-only).
Agents may modify both backend and frontend code.

## Purpose

Evergreen validation baseline for the hopeitworks pipeline.
This project must remain stable — CI must always pass.

## Tech Stack

- Backend: Node.js 20 + Express + pg (PostgreSQL driver)
- Frontend: Vanilla HTML/CSS/JS (no build step)
- Database: PostgreSQL 16
- Tests: Node.js built-in test runner (`node --test`)
- Linting: ESLint 9 (flat config)

## Conventions

- Backend entry point: `backend/server.js`
- Frontend entry point: `frontend/index.html`
- API prefix: `/api/todos`
- Error format: `{ error: { code: "UPPER_SNAKE_CASE", message: "..." } }`
- Database table: `todos` (see `init.sql` for schema)

## Commands

```bash
# Run tests
cd backend && npm test

# Lint
cd backend && npm run lint

# Start locally (requires Postgres)
cd backend && npm start

# Docker compose
docker compose up
```

## Constraints

- Keep the app minimal — do not add frameworks, ORMs, or build tools
- Do not break CI — this is an evergreen baseline
- Stories in `stories/` are pipeline test fixtures, not functional requirements
