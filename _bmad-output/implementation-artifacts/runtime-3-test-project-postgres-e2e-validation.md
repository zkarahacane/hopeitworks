# Story runtime-3: Fix test-project and End-to-End Validation

**Status:** ready-for-dev
**Branch:** `feat/runtime-3-test-project-e2e-validation`
**Commit scope:** `test-project`

---

## Story

As a developer validating the hopeitworks pipeline, I need the todo app in `test-project/` to work as a single coherent stack — with PostgreSQL as its database, a correct `docker-compose.yml`, a clean Dockerfile, and accurate README — so that I can launch a real agent run against it and see a pass/fail end-to-end without any infrastructure confusion.

---

## Context: Current State of test-project

The `test-project/` directory contains two coexisting implementations created at different times:

**`src/` (old — SQLite, better-sqlite3):**
- `src/app.js`, `src/db.js`, `src/routes/todos.js` — SQLite implementation
- Root `package.json` points to `src/app.js` as `main`, uses `better-sqlite3`
- Root `Dockerfile` copies `src/` and runs `src/app.js`
- Root `docker-compose.yml` has `DB_PATH` env var (SQLite) and no Postgres service

**`backend/` (new — PostgreSQL, pg driver):**
- `backend/app.js`, `backend/server.js` — PostgreSQL implementation using `pg`
- `backend/package.json` has `pg`, `cors`, `express` as deps
- `init.sql` has the PostgreSQL schema (correct)
- `CLAUDE.md` says tech stack is: Node.js + Express + pg (PostgreSQL)

**The mismatch:** `docker-compose.yml` and the root `Dockerfile` still point to the SQLite implementation. `init.sql` is PostgreSQL DDL. The `CLAUDE.md` says PostgreSQL. The root `README.md` says SQLite (wrong).

**Decision:** Consolidate to the **PostgreSQL implementation** (`backend/` directory). The `src/` directory is the legacy SQLite version and will be removed or left as-is if tests depend on it. The canonical entry point becomes `backend/server.js`.

---

## Acceptance Criteria

**AC #1 — `docker-compose.yml` includes a Postgres service**
- Given `test-project/docker-compose.yml`
- When `docker compose up` is run from `test-project/`
- Then a `postgres` service starts alongside `todo-app`, the app connects to it, and `GET /health` returns `{"status":"ok"}`

**AC #2 — The app connects to PostgreSQL on startup**
- Given the Postgres service is running with `DATABASE_URL` configured
- When the todo-app container starts
- Then `backend/server.js` connects without error and the `todos` table is initialized via `init.sql`

**AC #3 — Schema initialization is automatic**
- Given the postgres service starts fresh
- When the todo-app connects
- Then the `todos` table is created (via `init.sql` mounted or run in entrypoint) without manual intervention

**AC #4 — CRUD endpoints work against Postgres**
- Given the stack is running
- When `POST /api/todos`, `GET /api/todos`, `PUT /api/todos/:id`, `DELETE /api/todos/:id` are called
- Then they return correct HTTP responses using the PostgreSQL backend

**AC #5 — `docker compose up` is the single command to start the stack**
- Given `test-project/docker-compose.yml`
- When `docker compose up -d` is run
- Then both services start, the app is available on port 3000, and no manual db setup is required

**AC #6 — Root `Dockerfile` builds the PostgreSQL app**
- Given `test-project/Dockerfile`
- When `docker build .` is run from `test-project/`
- Then the resulting image runs `backend/server.js` (not `src/app.js`)

**AC #7 — `backend/` tests pass**
- Given `test-project/backend/`
- When `cd test-project/backend && npm test` is run against a running Postgres
- Then all tests in `backend/test/todos.test.js` pass

**AC #8 — README reflects actual tech stack**
- Given `test-project/README.md`
- When a developer reads it
- Then it accurately describes PostgreSQL (not SQLite), the correct `docker compose` commands, and the correct test commands

**AC #9 — `CLAUDE.md` is consistent**
- Given `test-project/CLAUDE.md`
- When an agent reads it
- Then it correctly describes `backend/server.js` as the entry point and PostgreSQL as the database (already accurate — verify no changes needed)

---

## Tasks / Subtasks

- [ ] **T1.** Update `test-project/docker-compose.yml` to add Postgres and fix app config (AC: #1, #2, #5)
  - [ ] T1.1 Add `postgres` service: `image: postgres:16-alpine`, env `POSTGRES_DB=todo`, `POSTGRES_USER=todo`, `POSTGRES_PASSWORD=todo`
  - [ ] T1.2 Add healthcheck to postgres service: `pg_isready -U todo`
  - [ ] T1.3 Update `todo-app` service: remove `DB_PATH` env var, add `DATABASE_URL=postgres://todo:todo@postgres:5432/todo`
  - [ ] T1.4 Add `depends_on: postgres: condition: service_healthy` to `todo-app`
  - [ ] T1.5 Add `ports: - "5432:5432"` to postgres for local debugging
  - [ ] T1.6 Add named volume `postgres_data` and mount it in postgres service
  - [ ] T1.7 Add `networks` section with a single `todo-net` bridge network; attach both services

- [ ] **T2.** Fix schema initialization — mount `init.sql` into Postgres init dir (AC: #3)
  - [ ] T2.1 Mount `init.sql` into `/docker-entrypoint-initdb.d/init.sql` in the postgres service via volumes
  - [ ] T2.2 Verify `init.sql` uses PostgreSQL DDL (currently correct: `SERIAL PRIMARY KEY`, `TIMESTAMPTZ`, `NOW()`)

- [ ] **T3.** Update `test-project/Dockerfile` to point to `backend/server.js` (AC: #6)
  - [ ] T3.1 Change `COPY src/ ./src/` to `COPY backend/ ./backend/`
  - [ ] T3.2 Change `RUN npm install --omit=dev` to use `backend/package.json`
  - [ ] T3.3 Change `CMD ["node", "src/app.js"]` to `CMD ["node", "backend/server.js"]`
  - [ ] T3.4 Add `HEALTHCHECK` pointing to `http://localhost:3000/health`

  **Note:** The root `package.json` uses `better-sqlite3`. After this change, the root `package.json` is no longer the app's entry point. The `docker-compose.yml` `build: .` context uses the root `Dockerfile`, which will install `backend/package.json` deps. Adjust accordingly.

- [ ] **T4.** Verify `backend/server.js` uses `DATABASE_URL` env var (AC: #2, #4)
  - [ ] T4.1 Confirm `backend/server.js` reads `process.env.DATABASE_URL` (already does: line 5)
  - [ ] T4.2 Confirm `backend/app.js` handles all 5 CRUD routes + health check (already does)
  - [ ] T4.3 No code changes needed in `backend/` — it is already correct

- [ ] **T5.** Update `test-project/README.md` (AC: #8)
  - [ ] T5.1 Replace "SQLite (via better-sqlite3)" with "PostgreSQL 16"
  - [ ] T5.2 Remove references to `npm run seed` / `seed.sql` for SQLite
  - [ ] T5.3 Update "Getting Started" to use `docker compose up` as the primary path
  - [ ] T5.4 Update test commands to reflect `cd backend && npm test`
  - [ ] T5.5 Correct CI pipeline description (remove Jest/supertest references if not used, verify current test runner is `node --test`)
  - [ ] T5.6 Add "Prerequisites: Docker + Docker Compose" section
  - [ ] T5.7 Add "End-to-End with hopeitworks" section explaining how to register this project in the platform and launch a run

- [ ] **T6.** Smoke test the full stack (AC: #1, #4, #5)
  - [ ] T6.1 Run `docker compose up -d` from `test-project/`
  - [ ] T6.2 Wait for health checks to pass
  - [ ] T6.3 Run: `curl -s http://localhost:3000/health` → `{"status":"ok"}`
  - [ ] T6.4 Run: `curl -s -X POST http://localhost:3000/api/todos -H "Content-Type: application/json" -d '{"title":"test"}'` → 201
  - [ ] T6.5 Run: `curl -s http://localhost:3000/api/todos` → list with the created todo
  - [ ] T6.6 Run: `docker compose down -v` to clean up

- [ ] **T7.** Document end-to-end pipeline validation steps (AC: #8)
  - [ ] T7.1 Add section to README: how to register `test-project/` as a hopeitworks project (repo URL, pipeline config)
  - [ ] T7.2 Document which stories in `stories/todo-stories.md` to use for validation
  - [ ] T7.3 Document the expected pipeline run flow: implement → review → merge

---

## Dev Notes

### Dependencies

- Docker and Docker Compose v2 must be available
- No backend (Go) changes required for this story
- No frontend (Vue) changes required for this story

### File Paths

| File | Change |
|------|--------|
| `test-project/docker-compose.yml` | Add postgres service, fix todo-app env, add network |
| `test-project/Dockerfile` | Point to `backend/server.js` instead of `src/app.js` |
| `test-project/README.md` | Correct tech stack, commands, add hopeitworks integration docs |
| `test-project/backend/server.js` | No change (already uses `DATABASE_URL`) |
| `test-project/backend/app.js` | No change (already correct pg implementation) |
| `test-project/init.sql` | No change (already PostgreSQL DDL) |
| `test-project/src/` | Leave as-is (legacy SQLite, not used by Docker) |

### Technical Specifications

**Updated `test-project/docker-compose.yml`:**

```yaml
services:
  postgres:
    image: postgres:16-alpine
    container_name: todo-postgres
    environment:
      POSTGRES_DB: todo
      POSTGRES_USER: todo
      POSTGRES_PASSWORD: todo
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U todo -d todo"]
      interval: 5s
      timeout: 5s
      retries: 5
    networks:
      - todo-net

  todo-app:
    build: .
    container_name: todo-app
    ports:
      - "3000:3000"
    environment:
      PORT: 3000
      DATABASE_URL: postgres://todo:todo@postgres:5432/todo
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped
    networks:
      - todo-net

volumes:
  postgres_data:

networks:
  todo-net:
    driver: bridge
```

**Updated `test-project/Dockerfile`:**

```dockerfile
FROM node:20-alpine

WORKDIR /app

# Install backend deps
COPY backend/package.json backend/package-lock.json ./backend/
RUN cd backend && npm install --omit=dev

# Copy backend source
COPY backend/ ./backend/

EXPOSE 3000

HEALTHCHECK --interval=10s --timeout=3s --start-period=10s \
  CMD wget -qO- http://localhost:3000/health || exit 1

CMD ["node", "backend/server.js"]
```

**Postgres service healthcheck pattern (from `deploy/docker-compose.yml` reference):**

```yaml
healthcheck:
  test: ["CMD-SHELL", "pg_isready -U todo -d todo"]
  interval: 5s
  timeout: 5s
  retries: 5
```

**Schema auto-initialization:**

PostgreSQL's official Docker image runs any `.sql` files found in `/docker-entrypoint-initdb.d/` on first start (when the data directory is empty). Mounting `init.sql` there is the standard pattern — no entrypoint script change needed.

**`seed.sql` note:**

The current `seed.sql` uses SQLite syntax (`INSERT OR REPLACE`, integer boolean `0/1`). It is NOT compatible with PostgreSQL. Do NOT mount it in `/docker-entrypoint-initdb.d/`. If seed data is needed for tests, create a separate `seed.postgres.sql` with standard SQL. This is out of scope for this story — leave the existing `seed.sql` as-is.

**`backend/test/todos.test.js` — requires running Postgres:**

The backend tests require a Postgres instance. They can be run:
1. Against the compose stack: `docker compose up -d && cd backend && DATABASE_URL=postgres://todo:todo@localhost:5432/todo npm test`
2. Against a test container (future improvement)

For this story, document option 1 in the README. Full test isolation via testcontainers is deferred.

### End-to-End Validation with hopeitworks

To run a real pipeline on the todo app after runtime-1 and runtime-2 are complete:

1. Ensure `hopeitworks/agent:latest` is built (runtime-1)
2. Ensure actions are wired (runtime-2)
3. Start the hopeitworks dev stack: `cd deploy && docker compose up -d`
4. Register `test-project/` as a project in hopeitworks (via UI or API):
   - `repo_url`: the GitHub URL of the hopeitworks repo (since `test-project/` is a subdirectory)
   - The pipeline will implement stories that modify files in `test-project/`
5. Import a story from `test-project/stories/todo-stories.md` (e.g., `TODO-1`)
6. Launch a run via UI → `POST /api/v1/projects/{id}/stories/{story_id}/runs`
7. Observe the pipeline: implement step → container spawns `hopeitworks/agent:latest` → agent clones repo → runs claude → pushes branch → CI polls → HITL gate → merge

### Known Issue: `src/` vs `backend/` coexistence

The `src/` directory (SQLite version) remains in the repo for now. It does not affect the Docker build since the Dockerfile explicitly copies `backend/`. However, the root `package.json` still references `src/app.js` — this is fine for local SQLite development but confusing. A future cleanup story (not in scope here) should:
- Remove `src/` entirely
- Update root `package.json` to point to `backend/server.js`
- Consolidate to a single implementation

For now: the Docker path is authoritative. `src/` is legacy.

### Testing Requirements

- `docker compose up -d` from `test-project/` succeeds
- `GET http://localhost:3000/health` returns 200
- `POST /api/todos` creates a record in Postgres (verify via `docker exec todo-postgres psql -U todo -d todo -c "SELECT * FROM todos"`)
- `GET /api/todos` returns the created record
- `docker compose down -v` cleans up without error
- `cd test-project/backend && DATABASE_URL=postgres://todo:todo@localhost:5432/todo npm test` passes (requires running stack)

### References

- `test-project/backend/server.js` — PostgreSQL server (already correct)
- `test-project/backend/app.js` — Express routes (already correct)
- `test-project/init.sql` — PostgreSQL schema (already correct)
- `test-project/CLAUDE.md` — agent scoping (already says PostgreSQL)
- `test-project/stories/todo-stories.md` — pipeline validation stories
- `deploy/docker-compose.yml` — reference for postgres service pattern in this project
- `backend/internal/integration/` — pipeline integration tests that reference test-project stories
