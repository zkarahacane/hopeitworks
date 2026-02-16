# Story 1.17: GitHub Actions CI Pipeline [INFRA]

Status: ready-for-dev

## Story

As a developer,
I want a GitHub Actions CI pipeline that validates backend and frontend code on every push and PR,
so that regressions are caught early and the main branch stays green.

## Acceptance Criteria (BDD)

**AC1: CI triggers on correct events**
- **Given** the workflow file at `.github/workflows/ci.yml`
- **When** I examine the trigger configuration
- **Then** it triggers on push to `main`, `develop`, and `wave-*` branches, and on pull_request targeting `main`, `develop`, and `wave-*` branches

**AC2: Backend build and tests pass**
- **Given** a push or PR triggers the CI pipeline
- **When** the backend job runs
- **Then** `go build ./cmd/api` succeeds, `go test ./... -race` passes, `go vet ./...` reports no issues, `make generate` produces no uncommitted changes, and `make lint-api` validates the OpenAPI spec

**AC3: Frontend lint, type-check, test, and build pass**
- **Given** a push or PR triggers the CI pipeline
- **When** the frontend job runs
- **Then** `npm ci`, `npm run lint`, `npm run type-check`, `npm run test:unit`, and `npm run build` all succeed

**AC4: Concurrency cancels in-progress runs**
- **Given** a new push to the same branch while CI is already running
- **When** the new workflow run starts
- **Then** the previous in-progress run for the same branch/PR is cancelled

## Tasks / Subtasks

- [ ] Task 1: Create `.github/workflows/ci.yml` with triggers and concurrency (AC: #1, #4)
  - [ ] Define workflow name: `CI`
  - [ ] Configure trigger on push to `main`, `develop`, and `wave-*` branches
  - [ ] Configure trigger on pull_request targeting `main`, `develop`, and `wave-*` branches
  - [ ] Set concurrency group to cancel in-progress runs for the same branch/PR

- [ ] Task 2: Define backend job with PostgreSQL service container (AC: #2)
  - [ ] Create `backend` job running on `ubuntu-latest`
  - [ ] Add PostgreSQL 16 service container with healthcheck
  - [ ] Set DB environment variables for test connection
  - [ ] Checkout code, setup Go 1.23 with caching
  - [ ] Run `go mod download`, `go build ./cmd/api`, `go vet ./...`, `go test ./... -race -coverprofile=coverage.out` in `backend/`
  - [ ] Run `make generate` then `git diff --exit-code` to verify codegen is committed
  - [ ] Setup Node.js (for @redocly/cli), run `make lint-api` in `backend/`

- [ ] Task 3: Define frontend job (AC: #3)
  - [ ] Create `frontend` job running on `ubuntu-latest`
  - [ ] Checkout code, setup Node 22 with npm caching (`cache-dependency-path: 'frontend/package-lock.json'`)
  - [ ] Run `npm ci`, `npm run lint`, `npm run type-check`, `npm run test:unit`, `npm run build` in `frontend/`

- [ ] Task 4: Verify CI workflow runs on a wave-* branch (AC: #1, #2, #3, #4)
  - [ ] Push workflow to a wave-* branch
  - [ ] Verify backend job passes (build, vet, test, codegen check, API lint)
  - [ ] Verify frontend job passes (lint, type-check, test, build)
  - [ ] Verify concurrency cancellation works on duplicate push

## Dev Notes

This story creates a single CI workflow file. No application code is modified. The `make generate` and `make lint-api` targets already exist from Story 1-2. The frontend npm scripts already exist from Story 1-7.

### Architecture Requirements

**Exact File Structure:**
```
hopeitworks/
└── .github/
    └── workflows/
        └── ci.yml          # Main CI workflow
```

### Technical Specifications

**Trigger Configuration:**
```yaml
on:
  push:
    branches: [main, develop, 'wave-*']
  pull_request:
    branches: [main, develop, 'wave-*']
```

**Concurrency:**
```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true
```

**Backend Job — Key Details:**
- Working directory: `backend/`
- Go version: 1.23
- Runner: `ubuntu-latest`
- PostgreSQL 16 service container (postgres:16-alpine)
  - `POSTGRES_USER: hopeitworks`
  - `POSTGRES_PASSWORD: hopeitworks_ci_password`
  - `POSTGRES_DB: hopeitworks_test`
  - Healthcheck: `pg_isready -U hopeitworks`
  - Ports: `5432:5432`
- Environment variables for test DB connection:
  - `DB_HOST: localhost`
  - `DB_PORT: 5432`
  - `DB_NAME: hopeitworks_test`
  - `DB_USER: hopeitworks`
  - `DB_PASSWORD: hopeitworks_ci_password`
  - `DB_SSLMODE: disable`
- Steps order: checkout -> setup-go -> mod download -> build -> vet -> test -> generate + git diff check -> setup-node -> lint-api

**Codegen Verification Pattern:**
```yaml
- name: Run codegen
  run: make generate
  working-directory: backend/

- name: Verify codegen is committed
  run: |
    if [ -n "$(git status --porcelain)" ]; then
      echo "::error::Generated code is not up-to-date. Run 'make generate' and commit the changes."
      git diff
      exit 1
    fi
```

**OpenAPI Lint Step:**
```yaml
- name: Setup Node.js (for redocly)
  uses: actions/setup-node@v4
  with:
    node-version: '22'

- name: Lint OpenAPI spec
  run: make lint-api
  working-directory: backend/
```

**Frontend Job — Key Details:**
- Working directory: `frontend/`
- Node version: 22
- Runner: `ubuntu-latest`
- Use npm cache via `actions/setup-node@v4` with `cache: 'npm'` and `cache-dependency-path: 'frontend/package-lock.json'`
- Steps order: checkout -> setup-node -> npm ci -> lint -> type-check -> test:unit -> build

### Dependencies

- **Story 1-1** (Go project scaffolding): provides `backend/` directory structure, Makefile, Go module
- **Story 1-7** (Vue scaffolding): provides `frontend/` directory structure, package.json with lint/type-check/test:unit/build scripts

### References

- [GitHub Actions: Service containers](https://docs.github.com/en/actions/using-containerized-services/about-service-containers)
- [actions/setup-go](https://github.com/actions/setup-go)
- [actions/setup-node](https://github.com/actions/setup-node)

## Dev Agent Record

## Change Log
