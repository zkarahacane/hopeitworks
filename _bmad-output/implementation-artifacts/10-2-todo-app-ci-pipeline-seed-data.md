# Story 10.2: Todo App CI Pipeline + Seed Data [SHARED]

Status: ready-for-dev

## Story

As a developer,
I want the todo app to have a functioning CI pipeline,
So that I can validate CI polling in the main system.

## Acceptance Criteria (BDD)

**AC1: CI workflow triggers and runs all stages**
- **Given** a PR is created against the test-project repository
- **When** the workflow runs
- **Then** it executes build, lint, test, and e2e stages in sequence, all passing

**AC2: Linting validates code quality**
- **Given** the CI pipeline runs
- **When** the lint stage executes
- **Then** `npm run lint` in `test-project/backend/` passes with eslint configuration from `.eslintrc.json`

**AC3: Unit tests cover API handlers**
- **Given** the CI pipeline runs
- **When** the test stage executes
- **Then** `npm test` runs unit tests in `test-project/backend/index.test.js` using Node.js built-in test runner with 100% pass rate

**AC4: E2E tests validate CRUD operations**
- **Given** the CI pipeline runs
- **When** the e2e stage executes
- **Then** `test-project/e2e/test.sh` runs bash curl tests that validate all CRUD endpoints (GET, POST, PATCH, DELETE) with correct HTTP status codes

**AC5: Database seeding provides baseline data**
- **Given** the Postgres database is initialized
- **When** `test-project/seed.sql` runs
- **Then** the `todos` table is populated with 10 sample todos (id uuid, title text, completed boolean, created_at timestamptz)

**AC6: Main branch CI is permanently green**
- **Given** all tasks are complete
- **When** CI runs on main branch
- **Then** all stages pass consistently, providing a stable baseline for pipeline validation in the main system

## Tasks / Subtasks

- [ ] Task 1: Create `.github/workflows/ci.yml` for test-project (AC: #1, #2, #3, #4, #6)
  - [ ] Define workflow name: `CI`
  - [ ] Configure trigger on push to `main` and `develop` branches
  - [ ] Configure trigger on pull_request targeting `main` and `develop` branches
  - [ ] Set concurrency group to cancel in-progress runs for the same branch/PR
  - [ ] Create job matrix or sequential jobs: build → lint → test → e2e
  - [ ] Setup Node.js 20 with caching
  - [ ] Add Postgres service container (postgres:16-alpine) with healthcheck
  - [ ] Run `npm ci` in `test-project/backend/`
  - [ ] Run `npm run lint` in `test-project/backend/`
  - [ ] Run `npm test` in `test-project/backend/`
  - [ ] Run `test-project/e2e/test.sh` with app started and database seeded

- [ ] Task 2: Create ESLint configuration (AC: #2)
  - [ ] Create `test-project/backend/.eslintrc.json` with reasonable defaults for Node.js + Express
  - [ ] Include rules: no-unused-vars, no-console, semi, quotes, eqeqeq
  - [ ] Add package.json lint script: `"lint": "eslint *.js"`

- [ ] Task 3: Create unit tests for API handlers (AC: #3)
  - [ ] Create `test-project/backend/index.test.js` using Node.js built-in test runner (`node:test`)
  - [ ] Import `test()` and `assert()` from `node:test`
  - [ ] Write test suites for each endpoint: GET /todos, POST /todos, GET /todos/:id, PATCH /todos/:id, DELETE /todos/:id
  - [ ] Test happy paths: correct status codes, response structure validation
  - [ ] Test error cases: 404 on missing todo, 400 on invalid input
  - [ ] Add package.json test script: `"test": "node --test"`

- [ ] Task 4: Create seed.sql with sample data (AC: #5)
  - [ ] Create `test-project/seed.sql`
  - [ ] Insert 10 sample todos into the todos table
  - [ ] Ensure each todo has: id (uuid v4 or generated), title (text), completed (boolean, mix of true/false), created_at (timestamptz, varied dates)
  - [ ] Example data should be realistic (e.g., "Set up testing", "Write documentation", "Deploy to production")

- [ ] Task 5: Create E2E test script (AC: #4)
  - [ ] Create `test-project/e2e/test.sh` (bash executable)
  - [ ] Script starts the todo app in background (`npm start`)
  - [ ] Wait for app to be ready (retry loop with `curl -f http://localhost:3000/health`)
  - [ ] Run curl tests for all CRUD operations:
    - [ ] GET /todos → expect 200, verify response is JSON array
    - [ ] POST /todos (create new) → expect 201, verify id and title in response
    - [ ] GET /todos/:id → expect 200, verify single todo object
    - [ ] PATCH /todos/:id (toggle completed) → expect 200, verify completed status changed
    - [ ] DELETE /todos/:id → expect 204, verify todo is gone (GET returns 404)
  - [ ] Stop the app gracefully on exit
  - [ ] Exit with non-zero code if any test fails

- [ ] Task 6: Verify CI runs end-to-end (AC: #1, #2, #3, #4, #6)
  - [ ] Push all changes to develop branch
  - [ ] Verify workflow file is valid YAML
  - [ ] Verify GitHub Actions detects and runs the workflow
  - [ ] Verify all stages pass: build, lint, test, e2e
  - [ ] Merge to main branch when CI passes
  - [ ] Verify main branch CI is green and stays green

## Dev Notes

This story creates a complete CI/CD pipeline for the test-project reference app. The CI pipeline validates that the todo app is functional and can be used as a baseline for testing the main system's CI polling features.

### Architecture Requirements

**Exact File Structure:**
```
test-project/
├── .github/
│   └── workflows/
│       └── ci.yml                    # Main CI workflow
├── backend/
│   ├── .eslintrc.json               # ESLint configuration
│   ├── index.test.js                # Unit tests (node:test)
│   ├── package.json                 # Updated with lint and test scripts
│   └── index.js                     # Existing Express app
├── e2e/
│   └── test.sh                       # Bash curl E2E tests
└── seed.sql                          # Sample data
```

### Technical Specifications

**Workflow Trigger Configuration:**
```yaml
on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]
```

**Concurrency:**
```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true
```

**Node.js Version:** Node 20

**PostgreSQL Service Container:**
- Image: `postgres:16-alpine`
- `POSTGRES_USER: hopeitworks`
- `POSTGRES_PASSWORD: hopeitworks_ci_password`
- `POSTGRES_DB: hopeitworks_test`
- Healthcheck: `pg_isready -U hopeitworks`
- Ports: `5432:5432`

**Environment Variables for Test DB:**
- `DB_HOST: localhost`
- `DB_PORT: 5432`
- `DB_NAME: hopeitworks_test`
- `DB_USER: hopeitworks`
- `DB_PASSWORD: hopeitworks_ci_password`
- `DB_SSLMODE: disable`

**ESLint Configuration (.eslintrc.json):**
```json
{
  "env": {
    "node": true,
    "es2021": true
  },
  "extends": "eslint:recommended",
  "parserOptions": {
    "ecmaVersion": "latest"
  },
  "rules": {
    "no-unused-vars": ["error", { "argsIgnorePattern": "^_" }],
    "no-console": "warn",
    "semi": ["error", "always"],
    "quotes": ["error", "single"],
    "eqeqeq": ["error", "always"]
  }
}
```

**Unit Test Pattern (node:test):**
```javascript
import test from 'node:test';
import assert from 'node:assert';
import app from './index.js';

test('GET /todos returns 200', async () => {
  const response = await fetch('http://localhost:3000/todos');
  assert.strictEqual(response.status, 200);
});
```

**E2E Test Script Pattern:**
```bash
#!/bin/bash
set -e

# Start app in background
npm start &
APP_PID=$!

# Wait for app to be ready
for i in {1..30}; do
  if curl -f http://localhost:3000/health > /dev/null 2>&1; then
    break
  fi
  sleep 1
done

# Run tests
curl -f http://localhost:3000/todos
# ... more curl tests ...

# Stop app
kill $APP_PID
```

**Seed SQL Pattern:**
```sql
INSERT INTO todos (id, title, completed, created_at) VALUES
  (gen_random_uuid(), 'Set up testing', false, now()),
  (gen_random_uuid(), 'Write documentation', false, now() - interval '2 days'),
  -- ... 8 more todos ...
;
```

### Dependencies

- **Story 10-1** (Todo app structure): provides `test-project/` directory with Express app, Postgres schema, package.json
- **Story 1-17** (GitHub Actions CI Pipeline): reference for workflow structure and GitHub Actions patterns

### Workflow Job Order

1. **Setup**: Checkout code, setup Node 20, setup Postgres 16
2. **Install**: `npm ci` (clean install of dependencies)
3. **Lint**: `npm run lint` in `backend/`
4. **Test**: `npm test` in `backend/`
5. **E2E**: `bash test-project/e2e/test.sh`

All stages must pass for PR to be mergeable.

### Success Criteria for Dev

- [ ] Workflow file is valid YAML and accepted by GitHub Actions
- [ ] All stages run automatically on push and PR events
- [ ] Lint stage catches ES violations (test by intentionally breaking a rule)
- [ ] Test stage validates unit test pass/fail logic (test by intentionally breaking a test)
- [ ] E2E stage validates CRUD operations end-to-end
- [ ] Seed data loads successfully before E2E tests
- [ ] Main branch CI is green after merge
- [ ] CI runs complete in under 5 minutes

### References

- [Node.js test module](https://nodejs.org/api/test.html)
- [GitHub Actions: Service containers](https://docs.github.com/en/actions/using-containerized-services/about-service-containers)
- [actions/setup-node](https://github.com/actions/setup-node)
- [ESLint configuration](https://eslint.org/docs/latest/use/configure/configuration-files)

## Dev Agent Record

## Change Log
