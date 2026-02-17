# Todo App - Reference Project

A minimal todo application used as a reference project for validating the hopeitworks pipeline end-to-end.

## Purpose

This project serves as a baseline for the hopeitworks CI polling and pipeline validation features. It provides:

- A simple REST API with CRUD operations for todos
- A static HTML frontend for managing todos
- A CI pipeline with build, lint, and test stages
- Seed data for consistent testing

## Tech Stack

- **Runtime:** Node.js 20+
- **Framework:** Express
- **Database:** SQLite (via better-sqlite3)
- **Testing:** Jest + supertest (unit), curl-based E2E
- **Linting:** ESLint 9 (flat config)
- **CI:** GitHub Actions

## Getting Started

### Local Development

```bash
# Install dependencies
npm install

# Seed the database
npm run seed

# Start the app
npm start
# App runs on http://localhost:3000
```

### Docker

```bash
# Build and run
docker compose up -d

# Or build manually
docker build -t todo-app .
docker run -p 3000:3000 todo-app
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/api/todos` | List all todos |
| GET | `/api/todos/:id` | Get a todo by ID |
| POST | `/api/todos` | Create a new todo |
| PUT | `/api/todos/:id` | Update a todo |
| DELETE | `/api/todos/:id` | Delete a todo |

### Request/Response Examples

**Create a todo:**
```bash
curl -X POST http://localhost:3000/api/todos \
  -H "Content-Type: application/json" \
  -d '{"title": "Buy groceries"}'
```

**List todos:**
```bash
curl http://localhost:3000/api/todos
```

## Testing

```bash
# Unit tests
npm test

# E2E tests (requires running app on localhost:3000)
npm run test:e2e

# Lint
npm run lint
```

## CI Pipeline

The GitHub Actions CI pipeline (`.github/workflows/ci.yml`) runs the following stages:

1. **Install** - Install npm dependencies
2. **Lint** - Run ESLint on source and test files
3. **Unit Tests** - Run Jest test suite
4. **Build** - Build Docker image
5. **E2E Tests** - Start the app in Docker and run curl-based E2E tests

The pipeline triggers on:
- Push to `main`
- Pull requests targeting `main`
- Manual dispatch

## Seed Data

The `seed.sql` file contains 8 sample todos for testing. To seed the database:

```bash
npm run seed
```

This creates the `todos` table (if not exists) and inserts sample todos with `INSERT OR REPLACE` for idempotency.
