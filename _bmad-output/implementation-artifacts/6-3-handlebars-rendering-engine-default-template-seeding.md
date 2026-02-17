# Story 6.3: [BACK] Handlebars rendering engine + default template seeding

Status: ready-for-dev

## Story

As a backend developer,
I want a Handlebars rendering engine for prompt templates and default template seeding,
so that prompts are rendered with story context variables and projects start with sensible defaults.

## Acceptance Criteria (BDD)

**AC1: TemplateRenderer renders templates with story context variables**
- **Given** a Handlebars template with variables: story_key, story_title, story_objective, target_files, acceptance_criteria, error_context, diff_content
- **When** I call RenderTemplate with a TemplateContext
- **Then** the template is rendered with all variables substituted correctly

**AC2: TemplateRenderer handles invalid Handlebars syntax**
- **Given** a template with invalid Handlebars syntax
- **When** I call RenderTemplate
- **Then** I receive a DomainError with code TEMPLATE_RENDER_FAILED and syntax details

**AC3: Migration seeds default prompt templates**
- **Given** migration 000011 exists and projects exist in database
- **When** migrations are applied
- **Then** default templates (implement.hbs, implement-retry.hbs, review.hbs, merge-conflict.hbs) are seeded for all existing projects

**AC4: TemplateService resolves templates from DB**
- **Given** a template exists in the database for a project
- **When** I call TemplateService.RenderForStory with the template name
- **Then** the template is loaded from DB and rendered with story context

**AC5: TemplateService falls back to default templates**
- **Given** a template does not exist in the database for a project
- **When** I call TemplateService.RenderForStory with a known default template name
- **Then** the default template content is used and rendered with story context

**AC6: TemplateService returns error for unknown templates**
- **Given** a template does not exist in DB and no default exists
- **When** I call TemplateService.RenderForStory
- **Then** I receive a DomainError with code TEMPLATE_NOT_FOUND

## Tasks / Subtasks

- [ ] [BACK] Task 1: Create TemplateContext domain model (AC: #1)
  - [ ] Create `backend/internal/domain/model/template_context.go`
  - [ ] Define TemplateContext struct with all context variables: StoryKey, StoryTitle, StoryObjective, TargetFiles, AcceptanceCriteria, ErrorContext, DiffContent, BranchName, RepoURL
  - [ ] Add JSON tags for all fields (snake_case for consistency with API)
  - [ ] Document each field with clear purpose comments

- [ ] [BACK] Task 2: Create TemplateRenderer port interface (AC: #1-2)
  - [ ] Create `backend/internal/domain/port/template_renderer.go`
  - [ ] Define TemplateRenderer interface with Render(templateContent string, ctx *model.TemplateContext) (string, error)
  - [ ] Document expected behavior: parse Handlebars template, substitute variables, return rendered string or error

- [ ] [BACK] Task 3: Implement Handlebars adapter using raymond library (AC: #1-2)
  - [ ] Create `backend/internal/adapter/handlebars/renderer.go`
  - [ ] Add dependency: `go get github.com/aymerick/raymond`
  - [ ] Implement TemplateRenderer interface using raymond.Render
  - [ ] Convert TemplateContext to map[string]interface{} for raymond
  - [ ] Handle parse errors and return DomainError with code TEMPLATE_RENDER_FAILED
  - [ ] Include syntax error details in error message

- [ ] [BACK] Task 4: Write unit tests for Handlebars renderer (AC: #1-2)
  - [ ] Create `backend/internal/adapter/handlebars/renderer_test.go`
  - [ ] Test valid template rendering with all context variables
  - [ ] Test template with loops (target_files array)
  - [ ] Test invalid Handlebars syntax returns TEMPLATE_RENDER_FAILED error
  - [ ] Test missing variables in context (should render empty string or default)
  - [ ] Test special characters and escaping

- [ ] [BACK] Task 5: Create TemplateService in domain layer (AC: #4-6)
  - [ ] Create `backend/internal/domain/service/template_service.go`
  - [ ] Implement TemplateService struct with dependencies: PromptTemplateRepository, TemplateRenderer, logger
  - [ ] Implement RenderForStory(ctx, projectID, templateName, tmplCtx) method
  - [ ] Resolve template: try DB first via PromptTemplateRepository.GetByProjectAndName
  - [ ] Fallback to default template content if not found in DB
  - [ ] Call TemplateRenderer.Render with resolved content
  - [ ] Return TEMPLATE_NOT_FOUND if template not in DB and no default exists
  - [ ] Add helper method for default template content (hardcoded fallbacks for implement, implement-retry, review, merge-conflict)

- [ ] [BACK] Task 6: Write unit tests for TemplateService (AC: #4-6)
  - [ ] Create `backend/internal/domain/service/template_service_test.go`
  - [ ] Test template found in DB: mock repository returns template, verify rendered output
  - [ ] Test fallback to default: mock repository returns not found, verify default template used
  - [ ] Test unknown template: mock repository returns not found, no default exists, verify TEMPLATE_NOT_FOUND error
  - [ ] Test render error propagation: mock renderer returns error, verify error bubbles up
  - [ ] Use mock PromptTemplateRepository and mock TemplateRenderer

- [ ] [BACK] Task 7: Create migration 000011 to seed default prompt templates (AC: #3)
  - [ ] Create `backend/migrations/000011_seed_default_prompt_templates.up.sql`
  - [ ] Create `backend/migrations/000011_seed_default_prompt_templates.down.sql`
  - [ ] Write INSERT statements for each default template: implement, implement-retry, review, merge-conflict
  - [ ] Use INSERT ... SELECT pattern to seed for all existing projects
  - [ ] Add WHERE NOT EXISTS clause to avoid duplicates on re-run
  - [ ] Down migration: DELETE default templates (WHERE name IN ('implement', 'implement-retry', 'review', 'merge-conflict'))

- [ ] [BACK] Task 8: Write default template content (AC: #3)
  - [ ] Define implement.hbs template content: story header, objective, target files (loop), acceptance criteria
  - [ ] Define implement-retry.hbs template content: retry header, previous error context, existing changes (diff), objective
  - [ ] Define review.hbs template content: review header, story context, diff content to review, review criteria
  - [ ] Define merge-conflict.hbs template content: merge conflict header, story context, conflict details, resolution guidance
  - [ ] Embed templates as Go string constants in migration SQL (escape single quotes)
  - [ ] Verify templates compile with raymond locally before embedding

- [ ] [BACK] Task 9: Wire TemplateService into main.go and verify (AC: #1-6)
  - [ ] Instantiate HandlebarsRenderer in main.go
  - [ ] Instantiate TemplateService with PromptTemplateRepository, HandlebarsRenderer, logger
  - [ ] Add TemplateService to DI wiring (update wire providers if using go-wire)
  - [ ] Run migration 000011 against dev database
  - [ ] Manual test: verify default templates exist in prompt_templates table for all projects
  - [ ] Manual test: call TemplateService.RenderForStory with DB template, verify output
  - [ ] Manual test: delete a template from DB, call RenderForStory, verify fallback to default
  - [ ] Manual test: call with unknown template name, verify TEMPLATE_NOT_FOUND error

## Dev Notes

This story adds the rendering engine for prompt templates: a Handlebars adapter (using raymond), a TemplateService to resolve and render templates, and a migration to seed default templates. It follows hexagonal architecture with clear port/adapter separation.

### Dependencies

**Story 6-2 (Prompt templates table + CRUD API, wave 5):** The prompt_templates table must exist for migration 000011 to seed default templates. The PromptTemplateRepository port must exist for TemplateService to resolve templates from DB.

**Story 3-8 (Agent run action, wave 7):** Will consume TemplateService to render prompts for agent runs. For now, this story only provides the rendering infrastructure.

**External dependency:** `github.com/aymerick/raymond` — Go Handlebars implementation. This is the most popular and actively maintained Handlebars library for Go.

### Architecture Requirements

**Hexagonal Architecture - Exact file paths:**

```
backend/
├── migrations/
│   ├── 000011_seed_default_prompt_templates.up.sql
│   └── 000011_seed_default_prompt_templates.down.sql
├── internal/
│   ├── domain/
│   │   ├── model/
│   │   │   └── template_context.go              # TemplateContext struct (domain model)
│   │   ├── port/
│   │   │   └── template_renderer.go             # TemplateRenderer interface
│   │   └── service/
│   │       ├── template_service.go              # TemplateService (resolve + render)
│   │       └── template_service_test.go         # Unit tests
│   └── adapter/
│       └── handlebars/
│           ├── renderer.go                      # TemplateRenderer impl (uses raymond)
│           └── renderer_test.go                 # Unit tests
└── cmd/
    └── api/
        └── main.go                              # Updated wiring
```

**Strict boundaries:**
- `domain/model/` and `domain/port/` import NOTHING from adapter/
- `domain/service/` depends only on `domain/port/` interfaces (TemplateRenderer, PromptTemplateRepository)
- `adapter/handlebars/` implements `domain/port/TemplateRenderer`, imports raymond library
- No direct imports of adapter code in domain layer

**Note:** Default templates are referenced but not stored as `.hbs` files in this story. They are embedded directly in the migration SQL. Future stories may refactor to load from `agent/prompts/*.hbs` files, but for MVP, SQL embedding is simpler.

### File Paths (exact)

- Migration: `backend/migrations/000011_seed_default_prompt_templates.{up,down}.sql`
- Domain model: `backend/internal/domain/model/template_context.go`
- Port interface: `backend/internal/domain/port/template_renderer.go`
- Service: `backend/internal/domain/service/template_service.go`
- Service tests: `backend/internal/domain/service/template_service_test.go`
- Handlebars adapter: `backend/internal/adapter/handlebars/renderer.go`
- Adapter tests: `backend/internal/adapter/handlebars/renderer_test.go`

### Technical Specifications

**TemplateContext model (`backend/internal/domain/model/template_context.go`):**
```go
package model

// TemplateContext provides variables available to Handlebars templates.
// All fields are exported and JSON-tagged for serialization and template rendering.
type TemplateContext struct {
    StoryKey           string   `json:"story_key"`            // Story identifier (e.g., "S-42")
    StoryTitle         string   `json:"story_title"`          // Story summary/title
    StoryObjective     string   `json:"story_objective"`      // Story description/objective
    TargetFiles        []string `json:"target_files"`         // Files to modify (for implement templates)
    AcceptanceCriteria string   `json:"acceptance_criteria"`  // Acceptance criteria (BDD format)
    ErrorContext       string   `json:"error_context"`        // Error details (for retry templates)
    DiffContent        string   `json:"diff_content"`         // Git diff or changes (for review/merge templates)
    BranchName         string   `json:"branch_name"`          // Git branch name
    RepoURL            string   `json:"repo_url"`             // Repository URL
}
```

**TemplateRenderer port (`backend/internal/domain/port/template_renderer.go`):**
```go
package port

import "github.com/zakari/hopeitworks/backend/internal/domain/model"

// TemplateRenderer renders Handlebars templates with story context.
type TemplateRenderer interface {
    // Render renders a Handlebars template string with the given context.
    // Returns the rendered string or an error if the template syntax is invalid.
    Render(templateContent string, ctx *model.TemplateContext) (string, error)
}
```

**TemplateService (`backend/internal/domain/service/template_service.go`):**
```go
package service

import (
    "context"
    "log/slog"
    "github.com/google/uuid"
    "github.com/zakari/hopeitworks/backend/internal/domain/model"
    "github.com/zakari/hopeitworks/backend/internal/domain/port"
)

type TemplateService struct {
    templateRepo port.PromptTemplateRepository
    renderer     port.TemplateRenderer
    logger       *slog.Logger
}

func NewTemplateService(
    templateRepo port.PromptTemplateRepository,
    renderer port.TemplateRenderer,
    logger *slog.Logger,
) *TemplateService {
    return &TemplateService{
        templateRepo: templateRepo,
        renderer:     renderer,
        logger:       logger,
    }
}

// RenderForStory resolves a template by name for a project, falls back to defaults, and renders it.
// Returns the rendered prompt string or an error.
func (s *TemplateService) RenderForStory(
    ctx context.Context,
    projectID uuid.UUID,
    templateName string,
    tmplCtx *model.TemplateContext,
) (string, error) {
    // Try to load from DB
    template, err := s.templateRepo.GetByProjectAndName(ctx, projectID, templateName)
    if err != nil {
        // If not found in DB, try default templates
        defaultContent := s.getDefaultTemplate(templateName)
        if defaultContent == "" {
            return "", &DomainError{Code: "TEMPLATE_NOT_FOUND", Message: "Template not found"}
        }
        return s.renderer.Render(defaultContent, tmplCtx)
    }

    // Render template from DB
    return s.renderer.Render(template.TemplateContent, tmplCtx)
}

// getDefaultTemplate returns hardcoded default template content for known template names.
// Returns empty string if no default exists.
func (s *TemplateService) getDefaultTemplate(name string) string {
    // See Task 8 for full template content
    defaults := map[string]string{
        "implement":       "...", // Full template in implementation
        "implement-retry": "...", // Full template in implementation
        "review":          "...", // Full template in implementation
        "merge-conflict":  "...", // Full template in implementation
    }
    return defaults[name]
}
```

**Note:** PromptTemplateRepository needs a new method: `GetByProjectAndName(ctx, projectID, name)`. This will be added to the existing PromptTemplateRepository port interface from Story 6-2. Add this query to `backend/queries/prompt_templates.sql`:

```sql
-- name: GetPromptTemplateByProjectAndName :one
SELECT * FROM prompt_templates
WHERE project_id = $1 AND name = $2;
```

**Migration 000011 SQL (`backend/migrations/000011_seed_default_prompt_templates.up.sql`):**
```sql
-- Seed default prompt templates for all existing projects
-- Each project gets: implement, implement-retry, review, merge-conflict

-- implement template
INSERT INTO prompt_templates (project_id, name, template_content, type)
SELECT p.id, 'implement', '
Implement story {{story_key}}: {{story_title}}

## Objective
{{story_objective}}

## Target Files
{{#each target_files}}
- {{this}}
{{/each}}

## Acceptance Criteria
{{acceptance_criteria}}

## Branch
{{branch_name}}
', 'implement'
FROM projects p
WHERE NOT EXISTS (
    SELECT 1 FROM prompt_templates pt
    WHERE pt.project_id = p.id AND pt.name = 'implement'
);

-- implement-retry template
INSERT INTO prompt_templates (project_id, name, template_content, type)
SELECT p.id, 'implement-retry', '
Retry implementation for {{story_key}}: {{story_title}}

## Previous Error
{{error_context}}

## Existing Changes
{{diff_content}}

## Objective
{{story_objective}}

Fix the issues described above while preserving the existing changes.
', 'retry'
FROM projects p
WHERE NOT EXISTS (
    SELECT 1 FROM prompt_templates pt
    WHERE pt.project_id = p.id AND pt.name = 'implement-retry'
);

-- review template
INSERT INTO prompt_templates (project_id, name, template_content, type)
SELECT p.id, 'review', '
Review changes for {{story_key}}: {{story_title}}

## Story Context
**Objective:** {{story_objective}}

**Acceptance Criteria:**
{{acceptance_criteria}}

## Changes to Review
{{diff_content}}

## Review Instructions
- Verify all acceptance criteria are met
- Check code quality and adherence to project conventions
- Flag any issues or suggest improvements
', 'review'
FROM projects p
WHERE NOT EXISTS (
    SELECT 1 FROM prompt_templates pt
    WHERE pt.project_id = p.id AND pt.name = 'review'
);

-- merge-conflict template
INSERT INTO prompt_templates (project_id, name, template_content, type)
SELECT p.id, 'merge-conflict', '
Resolve merge conflict for {{story_key}}: {{story_title}}

## Story Context
**Objective:** {{story_objective}}

## Conflict Details
{{error_context}}

## Current Changes
{{diff_content}}

## Resolution Instructions
- Review the conflict markers in the diff
- Resolve conflicts while preserving the story objective
- Ensure all acceptance criteria remain satisfied after resolution
', 'merge'
FROM projects p
WHERE NOT EXISTS (
    SELECT 1 FROM prompt_templates pt
    WHERE pt.project_id = p.id AND pt.name = 'merge-conflict'
);
```

**Migration 000011 down SQL (`backend/migrations/000011_seed_default_prompt_templates.down.sql`):**
```sql
-- Remove default templates seeded by this migration
DELETE FROM prompt_templates
WHERE name IN ('implement', 'implement-retry', 'review', 'merge-conflict');
```

**Handlebars adapter implementation (`backend/internal/adapter/handlebars/renderer.go`):**
```go
package handlebars

import (
    "fmt"
    "github.com/aymerick/raymond"
    "github.com/zakari/hopeitworks/backend/internal/domain/model"
)

type Renderer struct{}

func NewRenderer() *Renderer {
    return &Renderer{}
}

// Render implements port.TemplateRenderer
func (r *Renderer) Render(templateContent string, ctx *model.TemplateContext) (string, error) {
    // Convert TemplateContext to map for raymond
    data := map[string]interface{}{
        "story_key":           ctx.StoryKey,
        "story_title":         ctx.StoryTitle,
        "story_objective":     ctx.StoryObjective,
        "target_files":        ctx.TargetFiles,
        "acceptance_criteria": ctx.AcceptanceCriteria,
        "error_context":       ctx.ErrorContext,
        "diff_content":        ctx.DiffContent,
        "branch_name":         ctx.BranchName,
        "repo_url":            ctx.RepoURL,
    }

    result, err := raymond.Render(templateContent, data)
    if err != nil {
        return "", &DomainError{
            Code:    "TEMPLATE_RENDER_FAILED",
            Message: fmt.Sprintf("Failed to render template: %v", err),
        }
    }

    return result, nil
}
```

**Error codes used:**
- `TEMPLATE_RENDER_FAILED` — Handlebars parse/render error with syntax details (400/500)
- `TEMPLATE_NOT_FOUND` — template not in DB and no default exists (404)

**Error responses (match OpenAPI error envelope):**
```json
{
  "error": {
    "code": "TEMPLATE_RENDER_FAILED",
    "message": "Failed to render template: parse error at line 5"
  }
}
```

### Testing Requirements

**Unit test coverage (renderer_test.go):**
1. Valid template with all variables → renders correctly
2. Template with loops ({{#each target_files}}) → iterates correctly
3. Template with missing variables → renders empty string (Handlebars default)
4. Invalid Handlebars syntax (unclosed {{#if}}) → returns TEMPLATE_RENDER_FAILED
5. Template with special characters → escapes correctly

**Unit test coverage (template_service_test.go):**
1. Template found in DB → loads and renders DB template
2. Template not in DB, default exists → falls back to default template
3. Template not in DB, no default → returns TEMPLATE_NOT_FOUND error
4. Renderer returns error → error propagates to caller
5. All default template names resolve correctly (implement, implement-retry, review, merge-conflict)

**Manual verification checklist:**
1. Add dependency: `go get github.com/aymerick/raymond`
2. Run `go mod tidy` and verify raymond is in go.mod
3. `go build ./...` compiles successfully
4. `golangci-lint run ./...` passes with no errors
5. Run migration 000011: `migrate -path migrations/ -database $DB_URL up`
6. Query DB: `SELECT project_id, name FROM prompt_templates WHERE name IN ('implement', 'implement-retry', 'review', 'merge-conflict');`
7. Verify 4 templates per project in DB
8. Run unit tests: `go test ./internal/adapter/handlebars/... -v`
9. Run unit tests: `go test ./internal/domain/service/... -v`
10. Manual integration test: instantiate TemplateService, call RenderForStory with test context, verify output

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 6.3]
- [Source: _bmad-output/planning-artifacts/architecture.md#Backend Architecture -- Hexagonal Structure]
- [Source: _bmad-output/planning-artifacts/architecture.md#Agent Runtime & Prompts]
- [Source: Story 6-2 (prompt templates table) — provides PromptTemplateRepository port]
- [raymond library docs: https://github.com/aymerick/raymond]

## Dev Agent Record

## Change Log
