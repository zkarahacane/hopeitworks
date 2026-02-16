# Story 1.17: GitHub Actions CI Pipeline

Status: ready-for-dev

## Story

As a developer,
I want a GitHub Actions CI pipeline that validates backend and frontend code on every push and PR,
so that regressions are caught early and the main branch stays green.

## Acceptance Criteria (BDD)

**AC1: Backend build and tests pass**
- **Given** a push to main or wave-* branch, or a PR targeting main or wave-*
- **When** the CI pipeline runs the backend job
- **Then** `go build ./cmd/api` succeeds, `go test ./...` passes, and `go vet ./...` reports no issues

**AC2: Backend codegen is up-to-date**
- **Given** the backend job is running
- **When** the pipeline executes `make generate`
- **Then** there are no uncommitted changes (i.e., generated code is already committed and matches the spec)

**AC3: OpenAPI spec validation passes**
- **Given** the backend job is running
- **When** the pipeline executes `make lint-api` (which runs @redocly/cli lint on the OpenAPI spec)
- **Then** the spec validation succeeds with no errors

**AC4: Frontend lint, type-check, test, and build pass**
- **Given** a push to main or wave-* branch, or a PR targeting main or wave-*
- **When** the CI pipeline runs the frontend job
- **Then** `npm ci`, `npm run lint`, `npm run type-check`, `npm run test:unit`, and `npm run build` all succeed

**AC5: Matrix strategy covers required versions**
- **Given** the CI workflow definition
- **When** I examine the matrix configuration
- **Then** Go 1.23 and Node 22 are used as the matrix versions

**AC6: PostgreSQL service container is available for backend tests**
- **Given** the backend job is running and requires a database
- **When** tests that need PostgreSQL execute
- **Then** a PostgreSQL 16 service container is running and accessible via environment variables

**AC7: CI triggers on correct events**
- **Given** the workflow file at `.github/workflows/ci.yml`
- **When** I examine the trigger configuration
- **Then** it triggers on push to `main` and `wave-*` branches, and on pull_request targeting `main` and `wave-*` branches

## Tasks / Subtasks

- [ ] Task 1: Create `.github/workflows/ci.yml` workflow file (AC: #7)
  - [ ] Define workflow name: `CI`
  - [ ] Configure trigger on push to `main` and `wave-*` branches
  - [ ] Configure trigger on pull_request targeting `main` and `wave-*` branches
  - [ ] Set concurrency group to cancel in-progress runs for the same branch/PR

- [ ] Task 2: Define backend job with Go matrix and PostgreSQL service (AC: #1, #5, #6)
  - [ ] Create `backend` job running on `ubuntu-latest`
  - [ ] Configure Go version matrix: `[1.23]`
  - [ ] Add PostgreSQL 16 service container with healthcheck
  - [ ] Set DB environment variables (`DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_SSLMODE`)
  - [ ] Checkout code
  - [ ] Setup Go with `actions/setup-go@v5` and enable caching
  - [ ] Run `go mod download` in `backend/` directory
  - [ ] Run `go build ./cmd/api` in `backend/` directory
  - [ ] Run `go vet ./...` in `backend/` directory
  - [ ] Run `go test ./... -race -coverprofile=coverage.out` in `backend/` directory

- [ ] Task 3: Add codegen verification step to backend job (AC: #2)
  - [ ] Install codegen dependencies (oapi-codegen, sqlc, or whatever `make generate` requires)
  - [ ] Run `make generate` in `backend/` directory
  - [ ] Run `git diff --exit-code` to verify no uncommitted changes after generation

- [ ] Task 4: Add OpenAPI spec lint step to backend job (AC: #3)
  - [ ] Setup Node.js (for @redocly/cli) using `actions/setup-node@v4`
  - [ ] Install @redocly/cli via npx or npm
  - [ ] Run `make lint-api` in `backend/` directory

- [ ] Task 5: Define frontend job with Node matrix (AC: #4, #5)
  - [ ] Create `frontend` job running on `ubuntu-latest`
  - [ ] Configure Node version matrix: `[22]`
  - [ ] Checkout code
  - [ ] Setup Node.js with `actions/setup-node@v4` and enable npm caching
  - [ ] Run `npm ci` in `frontend/` directory
  - [ ] Run `npm run lint` in `frontend/` directory
  - [ ] Run `npm run type-check` in `frontend/` directory
  - [ ] Run `npm run test:unit` in `frontend/` directory
  - [ ] Run `npm run build` in `frontend/` directory

- [ ] Task 6: Verify CI workflow runs successfully (AC: #1, #2, #3, #4, #7)
  - [ ] Push workflow to a wave-* branch
  - [ ] Verify backend job passes (build, vet, test, codegen check, API lint)
  - [ ] Verify frontend job passes (lint, type-check, test, build)
  - [ ] Verify PostgreSQL service container is accessible during backend tests
  - [ ] Verify pipeline triggers correctly on PR creation

## Dev Notes

### Architecture Requirements

**Workflow File Location:**
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
    branches: [main, 'wave-*']
  pull_request:
    branches: [main, 'wave-*']
```

**Concurrency:**
```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true
```

**Backend Job Key Details:**
- Working directory: `backend/`
- Go version: 1.23 (matrix)
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
- Steps order: checkout -> setup-go -> mod download -> build -> vet -> test -> generate check -> lint-api

**Frontend Job Key Details:**
- Working directory: `frontend/`
- Node version: 22 (matrix)
- Use npm cache via `actions/setup-node@v4` with `cache: 'npm'` and `cache-dependency-path: 'frontend/package-lock.json'`
- Steps order: checkout -> setup-node -> npm ci -> lint -> type-check -> test:unit -> build

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
- name: Lint OpenAPI spec
  run: make lint-api
  working-directory: backend/
```
The `make lint-api` target should already be defined in the backend Makefile from Story 1-2 (OpenAPI spec codegen pipeline). It uses `npx @redocly/cli lint` under the hood.

### Dependencies

- **Story 1-1** (Go project scaffolding): provides `backend/` directory structure, Makefile, Go module
- **Story 1-7** (Vue scaffolding): provides `frontend/` directory structure, package.json with lint/type-check/test:unit/build scripts

### References

- [GitHub Actions: Using a matrix for your jobs](https://docs.github.com/en/actions/using-jobs/using-a-matrix-for-your-jobs)
- [GitHub Actions: Service containers](https://docs.github.com/en/actions/using-containerized-services/about-service-containers)
- [actions/setup-go](https://github.com/actions/setup-go)
- [actions/setup-node](https://github.com/actions/setup-node)

## Dev Agent Record

## Change Log
