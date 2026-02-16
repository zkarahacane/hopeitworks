# Story 1.14: [SHARED] CLAUDE.md files for agent scoping

Status: ready-for-dev

## Story

As a platform architect,
I want composed CLAUDE.md files from modular templates,
So that agents enforce project boundaries and follow best practices.

## Acceptance Criteria (BDD)

**Given** agent/claude-md/ directory exists
**When** I examine base.md
**Then** it documents git workflow, branch naming, conventional commits, quality standards

**Given** agent/claude-md/backend.md exists
**When** I examine it
**Then** it documents hexagonal architecture, chi, sqlc, DomainError, slog, testing patterns

**Given** agent/claude-md/frontend.md exists
**When** I examine it
**Then** it documents Composition API, PrimeVue first, Tailwind layout, useAsyncAction, Pinia patterns

**Given** agent/claude-md/project.md exists
**When** I examine it
**Then** it documents current state, key file paths, shared contract (api/openapi.yaml)

**Given** agent/claude-md/README.md exists
**When** I examine it
**Then** it specifies composition: base + (backend OR frontend) + project

## Tasks / Subtasks

- [ ] Create agent/claude-md/ directory structure (AC: all)
  - [ ] Create agent/claude-md/README.md with composition rules
  - [ ] Create agent/claude-md/base.md skeleton
  - [ ] Create agent/claude-md/backend.md skeleton
  - [ ] Create agent/claude-md/frontend.md skeleton
  - [ ] Create agent/claude-md/project.md skeleton

- [ ] Populate base.md with project-wide conventions (AC: base.md)
  - [ ] Git workflow and branch naming patterns
  - [ ] Conventional commits specification
  - [ ] Code quality standards
  - [ ] General testing principles
  - [ ] Documentation expectations

- [ ] Populate backend.md with Go-specific patterns (AC: backend.md)
  - [ ] Hexagonal architecture principles and package layout
  - [ ] Chi router patterns and middleware
  - [ ] sqlc query conventions and database patterns
  - [ ] DomainError pattern and error handling
  - [ ] slog structured logging conventions
  - [ ] Testing patterns (unit, integration, testcontainers)
  - [ ] go-wire dependency injection patterns
  - [ ] pgx/v5 transaction management

- [ ] Populate frontend.md with Vue-specific patterns (AC: frontend.md)
  - [ ] Vue 3 Composition API conventions
  - [ ] PrimeVue component usage rules
  - [ ] Tailwind CSS layout-only usage
  - [ ] useAsyncAction pattern for API calls
  - [ ] Pinia store patterns
  - [ ] openapi-fetch API client usage
  - [ ] Component organization (ui/ vs features/)
  - [ ] Testing patterns (Vitest, Playwright)

- [ ] Populate project.md with current state information (AC: project.md)
  - [ ] Project overview and current phase
  - [ ] Key file path references
  - [ ] Shared API contract location (api/openapi.yaml)
  - [ ] Current implementation status
  - [ ] Known constraints and decisions

- [ ] Validate CLAUDE.md files against architecture document (AC: all)
  - [ ] Cross-reference all patterns with architecture.md
  - [ ] Ensure no contradictions or missing patterns
  - [ ] Verify completeness for MVP scope

## Dev Notes

This story creates the foundation for all future agent-driven development by establishing modular CLAUDE.md templates that will be composed at runtime into agent containers. These files are CRITICAL for ensuring agents respect architectural boundaries and follow established conventions.

### Architecture Requirements

**File Locations and Composition:**
```
agent/claude-md/
├── README.md          # Composition rules: base + (backend OR frontend) + project
├── base.md            # Common: git, commits, quality, testing principles
├── backend.md         # Go: hexagonal, chi, sqlc, DomainError, slog, testing
├── frontend.md        # Vue: Composition API, PrimeVue, Tailwind, Pinia, testing
└── project.md         # Current state: phase, key paths, openapi.yaml, status
```

**Composition Logic (for later implementation):**
- Backend agent: `base.md` + `backend.md` + `project.md` → injected as single CLAUDE.md
- Frontend agent: `base.md` + `frontend.md` + `project.md` → injected as single CLAUDE.md
- This composition happens in the agent runtime (FR27-FR28), not in this story

**Scoping Rules:**
- Backend agents MUST NEVER touch `frontend/` directory
- Frontend agents MUST NEVER touch `backend/` directory
- Both can reference `api/openapi.yaml` (read-only, coordinated changes)
- Each CLAUDE.md section must be self-contained (agent may not have access to architecture doc at runtime)

### Technical Specifications

**Root Project CLAUDE.md (out of scope for this story):**
This story creates templates in `agent/claude-md/`. The root `/Users/zakari.karahacane/projects/hopeitworks/CLAUDE.md` is separate and contains BMAD workflow references. Backend and frontend directories may get their own CLAUDE.md files in later stories.

**base.md Content Guidelines:**
```markdown
# Git Workflow
- Branch naming: feat/S-{key}-{slug}, fix/S-{key}-{slug}
- Conventional commits: feat(scope): message, fix(scope): message
- Example: feat(pipeline): add retry logic
- Example: fix(auth): token expiry handling

# Commit Standards
- Scope matches domain: pipeline, auth, api, ui, dag, etc.
- Message: imperative mood, lowercase, no period
- Body: optional, explains WHY not WHAT

# Quality Standards
- No commented-out code in commits
- No console.log or fmt.Println in production code
- All exported functions/types documented
- Error messages must be actionable

# Testing Principles
- Every new feature has tests
- Tests must be deterministic
- Use factories over fixtures
- Integration tests tagged separately from unit tests

# Documentation
- README updated for public API changes
- CHANGELOG.md follows Keep a Changelog format
- Code comments explain WHY, not WHAT
```

**backend.md Content Guidelines (extract from architecture.md):**
```markdown
# Hexagonal Architecture

Package layout:
- `internal/domain/model/` — Entities: Story, Run, RunStep, Project, Epic, Event, User
- `internal/domain/port/` — Interfaces: GitProvider, AgentRuntime, Repository, Transactor
- `internal/domain/service/` — Business logic: PipelineService, SchedulerService
- `internal/adapter/` — Implementations: postgres, github, docker, river
- `internal/api/handler/` — oapi-codegen generated handlers
- `pkg/` — Shared utilities: log, errors, exec, config

Boundary rules:
- Services depend on ports (interfaces), never on adapters
- Adapters implement ports
- No business logic in handlers or adapters

# Chi Router Patterns

Route registration:
```go
r := chi.NewRouter()
r.Use(middleware.Auth)
r.Route("/api/v1/projects", func(r chi.Router) {
    r.Get("/", handler.ListProjects)
    r.Post("/", handler.CreateProject)
    r.Route("/{id}", func(r chi.Router) {
        r.Get("/", handler.GetProject)
        r.Put("/", handler.UpdateProject)
    })
})
```

Middleware order: RequestID → Logger → CORS → Auth

# sqlc Conventions

Queries in `backend/queries/*.sql`:
```sql
-- name: GetStoryByKey :one
SELECT * FROM stories WHERE project_id = $1 AND key = $2 LIMIT 1;

-- name: ListStoriesByStatus :many
SELECT * FROM stories WHERE project_id = $1 AND status = ANY($2::text[]);
```

Generated code usage:
```go
story, err := q.GetStoryByKey(ctx, db.GetStoryByKeyParams{
    ProjectID: projectID,
    Key:       storyKey,
})
```

# DomainError Pattern

Error construction:
```go
import "github.com/zakari/hopeitworks/backend/pkg/errors"

// In service layer
if story == nil {
    return nil, errors.NewNotFound("story", storyKey)
}
if !valid {
    return nil, errors.NewValidation("field", "reason")
}
```

Error categories: NotFound, Validation, Conflict, Unauthorized, Forbidden, Internal

API middleware maps category → HTTP status

# slog Structured Logging

Context enrichment:
```go
import "github.com/zakari/hopeitworks/backend/pkg/log"

logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
ctx = log.WithLogger(ctx, logger)

// In service
log.LoggerFrom(ctx).Info("processing run",
    "run_id", runID,
    "story_key", story.Key,
)
```

Sensitive values automatically scrubbed by ScrubHandler

# Testing Patterns

Unit tests (table-driven):
```go
func TestBuildDAG(t *testing.T) {
    tests := []struct {
        name    string
        stories []model.Story
        wantDAG model.DAG
        wantErr bool
    }{
        // cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test
        })
    }
}
```

Integration tests (testcontainers):
```go
func TestRepositoryWithDB(t *testing.T) {
    ctx := context.Background()
    db := testutil.NewTestDB(t) // spins up Postgres container
    defer db.Close()

    repo := postgres.NewStoryRepository(db.Pool)
    // test against real DB
}
```

Factories:
```go
story := testutil.NewStory(
    testutil.WithKey("S-01"),
    testutil.WithDeps("S-02", "S-03"),
)
```

# go-wire Dependency Injection

Provider sets in `wire.go`:
```go
var ServiceSet = wire.NewSet(
    service.NewPipelineService,
    service.NewSchedulerService,
)

var AdapterSet = wire.NewSet(
    postgres.NewStoryRepository,
    github.NewGitProvider,
)
```

Generate: `wire ./cmd/api/`

# pgx/v5 Transaction Management

Transactor pattern:
```go
type Transactor interface {
    WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// Usage in service
err := transactor.WithinTransaction(ctx, func(ctx context.Context) error {
    // Repo extracts tx from context, falls back to pool if no tx
    return repo.Save(ctx, story)
})
```
```

**frontend.md Content Guidelines (extract from architecture.md):**
```markdown
# Vue 3 Composition API Conventions

Components use `<script setup>`:
```vue
<script setup lang="ts">
import { ref, computed } from 'vue'
import { useAsyncAction } from '@/composables/useAsyncAction'

const props = defineProps<{
  storyId: string
}>()

const emit = defineEmits<{
  updated: [story: Story]
}>()

// Logic here
</script>
```

No Options API. Pure Composition API only.

# PrimeVue Component Usage Rules

1. Use PrimeVue components for everything they provide
2. Never reinvent: Button, DataTable, Dialog, Toast, Menu, Tag, Badge, InputText, etc.
3. Override via design tokens, NOT via custom CSS
4. Severity props for status: `severity="success"` not `:style="{ color: 'green' }"`

Example:
```vue
<Button label="Run" severity="success" @click="handleRun" />
<Tag :value="status" :severity="statusSeverity(status)" />
```

# Tailwind CSS Layout-Only Usage

Use Tailwind ONLY for layout:
- flex, grid, gap, padding, margin
- NOT for colors, typography (use PrimeVue tokens)

Example:
```vue
<div class="flex flex-col gap-4 p-6">
  <DataTable :value="stories" />
</div>
```

No `<style scoped>` blocks except for complex animations

# useAsyncAction Pattern

Every async operation wraps in useAsyncAction:
```ts
import { useAsyncAction } from '@/composables/useAsyncAction'

const { execute, isLoading, error, data } = useAsyncAction(async (storyId: string) => {
  const response = await apiClient.GET('/api/v1/stories/{id}', {
    params: { path: { id: storyId } }
  })
  return response.data
})
```

Components render based on `isLoading`, `error`, `data`

# Pinia Store Patterns

Store structure:
```ts
import { defineStore } from 'pinia'

export const useStoriesStore = defineStore('stories', () => {
  const stories = ref<Story[]>([])
  const isLoading = ref(false)

  async function fetchStories(projectId: string) {
    isLoading.value = true
    try {
      const response = await apiClient.GET('/api/v1/projects/{id}/stories', {
        params: { path: { id: projectId } }
      })
      stories.value = response.data || []
    } finally {
      isLoading.value = false
    }
  }

  return { stories, isLoading, fetchStories }
})
```

SSE events update stores reactively

# openapi-fetch API Client Usage

Generated client:
```ts
import { createClient } from '@/api/client'

const apiClient = createClient<paths>({
  baseUrl: '/api/v1',
  credentials: 'include', // JWT in httpOnly cookie
})

// Usage
const { data, error } = await apiClient.GET('/api/v1/stories/{id}', {
  params: { path: { id: storyId } }
})
```

All types generated from `api/openapi.yaml`

# Component Organization

**ui/** — Shared components used by 2+ features
- `ui/primitives/` — PrimeVue wrappers, base components
- `ui/composed/` — Reusable combinations (DataTable, LogViewer)
- `ui/layout/` — Page structure (AppShell, PageHeader)

**features/** — Domain-specific components
- `features/projects/` — ProjectList, ProjectSettings
- `features/stories/` — StoryBoard, StoryDetail
- `features/runs/` — RunTimeline, RunDetail

Rule: If used by 2+ features → ui/. Otherwise → stays in feature.

# Testing Patterns

Vitest unit tests:
```ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import StoryCard from '../StoryCard.vue'

describe('StoryCard', () => {
  it('displays story title', () => {
    const wrapper = mount(StoryCard, {
      props: { story: { title: 'Test Story', status: 'backlog' } }
    })
    expect(wrapper.text()).toContain('Test Story')
  })
})
```

Playwright E2E tests:
```ts
import { test, expect } from '@playwright/test'

test('launch story run', async ({ page }) => {
  await page.goto('/projects/1/stories')
  await page.click('text=Run Story')
  await expect(page.locator('.run-timeline')).toBeVisible()
})
```
```

**project.md Content Guidelines:**
```markdown
# Project Overview

hopeitworks v2 — AI agent orchestration platform
- Current phase: MVP implementation (Epic 1-4)
- Tech stack: Go backend, Vue 3 frontend, Postgres, Docker
- Architecture: Hexagonal (backend), feature-based (frontend)

# Key File Paths

- **API contract**: `api/openapi.yaml` — single source of truth for REST API
- **Backend migrations**: `backend/migrations/*.sql`
- **Backend queries**: `backend/queries/*.sql` — sqlc source
- **Frontend API client**: `frontend/src/api/client.ts` — generated from openapi.yaml
- **Shared types**: Generated from openapi.yaml (Go: oapi-codegen, TS: openapi-typescript)

# Shared API Contract

All API changes start with `api/openapi.yaml`:
1. Update OpenAPI spec
2. Regenerate backend: `cd backend && make generate`
3. Regenerate frontend: `cd frontend && npm run generate-api`
4. Implement handlers (backend) and views (frontend) in parallel

# Current Implementation Status

- Epic 1: Project scaffolding — IN PROGRESS
- Epic 2: Story board — NOT STARTED
- Epic 3: Pipeline execution — NOT STARTED
- Epic 4: Agent runtime — NOT STARTED

# Known Constraints

- Backend agents work ONLY in `backend/` directory
- Frontend agents work ONLY in `frontend/` directory
- API contract changes require coordination between both sides
- MVP = measurement, not enforcement (cost tracking, no halting)
- Docker mode for MVP (K8s deferred to Phase 2)
```

### Content Guidelines

**Each CLAUDE.md section must be self-contained:**
- Agent containers may not have access to the full architecture document
- Include exact library versions where critical (e.g., "pgx/v5", "PrimeVue 4", "Vue 3")
- Include exact commands (e.g., `make generate`, `npm run generate-api`)
- Include folder structure conventions explicitly
- Include testing commands and patterns explicitly

**Reference the architecture document for accuracy:**
All patterns, naming conventions, and architectural decisions MUST be pulled verbatim from:
`/Users/zakari.karahacane/projects/hopeitworks/_bmad-output/planning-artifacts/architecture.md`

Cross-reference sections:
- base.md → Architecture sections: "Git Flow", "Naming Patterns", "Process Patterns"
- backend.md → Architecture sections: "Backend Architecture — Foundations", "Backend Architecture — Hexagonal Structure", "Backend Architecture — Domain Services", "Testing Strategy — Backend Testing"
- frontend.md → Architecture sections: "Frontend Architecture", "Hybrid Structure", "PrimeVue Setup", "Style Conventions", "Functional Patterns", "Testing Strategy — Frontend Testing"
- project.md → Architecture sections: "Project Context Analysis", "Project Structure Decision", "Implementation Priority"

**Exact standards to include (from architecture.md):**

**Git conventions:**
- Branch naming: `feat/S-{key}-{slug}`, `fix/S-{key}-{slug}`
- Commits: conventional commits (`feat(pipeline): add retry logic`)
- Squash merge by default, delete branch after merge

**Database conventions:**
- Tables: `snake_case`, plural
- Columns: `snake_case`
- Foreign keys: `{referenced_table_singular}_id`
- Indexes: `idx_{table}_{columns}`

**API conventions:**
- Endpoints: plural nouns, kebab-case
- Route params: `{id}` format
- Query params: `snake_case`
- JSON fields: `snake_case`
- Dates: ISO 8601 strings

**Go conventions:**
- Files: `snake_case.go`
- Packages: single lowercase word
- Types: `PascalCase`
- Interfaces: descriptive noun (not `IInterface`)
- Variables: `camelCase`

**Vue/TypeScript conventions:**
- Components: `PascalCase.vue`
- Composables: `use` prefix, `camelCase`
- Stores: domain noun
- Utils: `camelCase.ts`
- Types/interfaces: `PascalCase`

### File Structure

Create the following files with exact content based on architecture document:

```
agent/
└── claude-md/
    ├── README.md          # Composition rules
    ├── base.md            # 200-300 lines: git, commits, quality, testing
    ├── backend.md         # 400-600 lines: hexagonal, chi, sqlc, errors, logging, testing
    ├── frontend.md        # 400-600 lines: Vue, PrimeVue, Tailwind, Pinia, testing
    └── project.md         # 100-150 lines: current state, paths, contract, constraints
```

### Testing Requirements

**Validation:**
1. All CLAUDE.md files are valid Markdown (lint with markdownlint)
2. All code examples in CLAUDE.md files are syntactically correct
3. All references to architecture.md patterns are accurate
4. No contradictions between base, backend, frontend sections
5. Composition rule is clear in README.md

**Linting:**
```bash
# Markdown lint
npx markdownlint agent/claude-md/*.md

# Link validation (all internal references valid)
# Manual review against architecture.md
```

**Acceptance validation:**
Each AC maps to a specific file:
- AC1 (base.md) → `agent/claude-md/base.md` exists and documents git workflow, conventional commits, quality
- AC2 (backend.md) → `agent/claude-md/backend.md` exists and documents hexagonal, chi, sqlc, DomainError, slog, testing
- AC3 (frontend.md) → `agent/claude-md/frontend.md` exists and documents Composition API, PrimeVue, Tailwind, Pinia, testing
- AC4 (project.md) → `agent/claude-md/project.md` exists and documents current state, key paths, openapi.yaml
- AC5 (README.md) → `agent/claude-md/README.md` exists and specifies composition rule

### Project Structure Notes

**Alignment with project structure:**
This story creates files in the `agent/` directory as specified in the architecture document (section "Project Structure Decision: Monorepo with Strict Boundaries").

The `agent/` directory contains:
```
agent/
├── Dockerfile              # Project-specific agent image (extends base)
├── Dockerfile.base         # Base image: Claude Code + git + gh CLI
├── entrypoint.sh           # Container entry script
├── scripts/                # Runtime scripts
├── claude-md/              # ← THIS STORY: CLAUDE.md templates
└── prompts/                # Handlebars prompt templates
```

**No conflicts with existing structure:**
The `agent/claude-md/` directory does not yet exist. This story creates it for the first time.

**Note on composition implementation:**
The actual composition logic (base + backend/frontend + project → single CLAUDE.md) will be implemented in the agent runtime story (Epic 4, Story 4.x: Agent Runtime). This story ONLY creates the modular template files.

### References

- [Architecture: Project Structure Decision] `/Users/zakari.karahacane/projects/hopeitworks/_bmad-output/planning-artifacts/architecture.md` lines 60-165
- [Architecture: Backend Architecture — Foundations] lines 465-525
- [Architecture: Backend Architecture — Hexagonal Structure] lines 527-590
- [Architecture: Frontend Architecture] lines 872-1025
- [Architecture: Implementation Patterns & Consistency Rules] lines 1116-1182
- [Architecture: Git Flow] lines 862-868
- [Architecture: Testing Strategy] lines 1028-1112
- [PRD: Technical Success Criteria] `/Users/zakari.karahacane/projects/hopeitworks/_bmad-output/planning-artifacts/prd.md` lines 54-60
- [PRD: FR28: CLAUDE.md injection] lines 269-270

## Dev Agent Record

### Agent Model Used

_To be filled by dev agent_

### Debug Log References

_To be filled by dev agent_

### Completion Notes List

_To be filled by dev agent_

### File List

_To be filled by dev agent_
