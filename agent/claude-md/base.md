# Base Agent Instructions — Project-Wide Conventions

You are an AI agent working on the **hopeitworks** platform. Follow these conventions strictly for all code changes.

## Git Workflow

### Branch Naming

- Feature branches: `feat/{story-key}-{slug}` (e.g., `feat/1-14-claude-md-files`)
- Fix branches: `fix/{story-key}-{slug}` (e.g., `fix/1-3-ci-poller`)
- The branch name is provided to you — always work on the assigned branch

### Conventional Commits

Format: `type(scope): message`

Types:
- `feat` — new feature
- `fix` — bug fix
- `refactor` — code restructuring without behavior change
- `test` — adding or updating tests
- `docs` — documentation changes
- `chore` — build, CI, tooling changes

Scope matches the domain area:
- Backend: `pipeline`, `auth`, `api`, `dag`, `git`, `agent`, `event`, `cost`, `config`
- Frontend: `ui`, `stories`, `runs`, `approvals`, `dag`, `editor`, `auth`
- Shared: `api-spec`, `deploy`, `ci`

Rules:
- Message in imperative mood, lowercase, no period at end
- Body is optional — explains WHY, not WHAT
- One logical change per commit

Examples:
```
feat(pipeline): add retry logic for failed steps
fix(auth): handle token expiry on page refresh
refactor(dag): extract cycle detection into helper
test(git): add integration tests for PR creation
chore(deploy): update docker-compose health checks
```

### Merge Strategy

- Squash merge by default
- Delete branch after merge
- PR title follows conventional commit format

## Commit Standards

- Scope matches domain (see list above)
- Message: imperative mood, lowercase, no period
- Body: optional, explains WHY not WHAT
- Footer: reference story key (e.g., `Refs: S-14`)

## Code Quality Standards

### Mandatory

- No commented-out code in commits
- No `console.log` or `fmt.Println` in production code (use structured logging)
- All exported functions and types must be documented
- Error messages must be actionable (include context: what failed, what was expected)
- No hardcoded secrets, tokens, or credentials — use environment variables
- No `TODO` or `FIXME` without a linked story key

### Formatting

- Backend: `gofmt` / `goimports` (enforced by golangci-lint)
- Frontend: Prettier + ESLint (enforced by lint scripts)
- Commit only formatted code

### Linting

- Backend: `golangci-lint run ./...`
- Frontend: `npm run lint`

## Testing Principles

- Every new feature has tests
- Tests must be deterministic — no flaky tests, no time-dependent assertions
- Use factories over static fixtures for test data
- Integration tests tagged or named separately from unit tests
- Test the behavior, not the implementation
- Aim for high coverage on business logic; skip trivial getters/setters

### Test Organization

- Tests co-located with source in `__tests__/` directories (both Go and Vue)
- Go: use `-short` flag to separate unit from integration tests
- Frontend: unit tests via Vitest, E2E tests via Playwright

## Documentation

- README updated for public API changes
- Code comments explain WHY, not WHAT
- Document non-obvious architectural decisions inline
- Generated code should never be manually edited — regenerate from source

## API Contract

- `api/openapi.yaml` is the single source of truth for the REST API
- All API changes start with updating the OpenAPI spec
- Both backend and frontend generate code from this spec
- Never manually write types that should be generated from the spec

## Naming Conventions

### Database

- Tables: `snake_case`, plural (`stories`, `run_steps`, `pipeline_configs`)
- Columns: `snake_case` (`created_at`, `project_id`, `retry_count`)
- Foreign keys: `{referenced_table_singular}_id` (`project_id`, `story_id`)
- Indexes: `idx_{table}_{columns}` (`idx_stories_project_id`, `idx_runs_status`)
- Constraints: `{table}_{type}_{columns}` (`runs_fk_story_id`, `stories_uq_key_project`)

### API

- Endpoints: plural nouns, kebab-case for multi-word (`/pipeline-configs`, `/run-steps`)
- Route params: `{id}` format (OpenAPI standard)
- Query params: `snake_case` (`project_id`, `per_page`, `sort_by`)
- JSON fields: `snake_case` (matches Go JSON tags and Postgres columns)
- Dates: ISO 8601 strings (`"2026-02-15T10:30:00Z"`)

### Events (SSE / Postgres NOTIFY)

- Format: `{entity}.{action}` dot-notation (`run.started`, `step.completed`, `hitl.pending`)
- Payload: JSON with `snake_case` fields

## Error Handling Philosophy

- Errors are values — handle them explicitly, never ignore
- Wrap errors with context as they propagate up the call stack
- API errors follow a consistent envelope format:
  ```json
  {
    "error": {
      "code": "STORY_NOT_FOUND",
      "message": "Story S-03 not found in project X",
      "details": {}
    }
  }
  ```
- Error codes are `UPPER_SNAKE_CASE`
- Error messages are human-readable and actionable

## Security

- Never log secrets, tokens, or API keys
- Use environment variables for all sensitive configuration
- Validate all external input at system boundaries
- Agent containers have no host filesystem access
