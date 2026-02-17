# Story 2.3: [BACK] Story markdown import + frontmatter parsing

Status: ready-for-dev

## Story

As a project maintainer,
I want to bulk import stories from markdown with YAML frontmatter,
So that I can manage stories as code.

## Acceptance Criteria (BDD)

**AC1: Import endpoint parses YAML frontmatter and extracts story fields**
- **Given** POST /api/v1/projects/{projectId}/stories/import is called with markdown content
- **When** the system parses YAML frontmatter from each story block
- **Then** key, epic, depends_on, scope, status fields are extracted
- **And** remaining markdown body is used as acceptance_criteria
- **And** the first H1 heading (`# Title`) is used as the story title

**AC2: Existing story key triggers update instead of create**
- **Given** a story with key S-03 already exists in the project
- **When** import includes a story with key S-03
- **Then** the existing story is updated with the new frontmatter values
- **And** no duplicate key error is returned for that story
- **And** the response reflects the story was updated (not created)

**AC3: Partial success — valid stories saved, per-story errors reported**
- **Given** import contains a mix of valid and invalid stories
- **When** the endpoint processes the request
- **Then** valid stories are created or updated successfully
- **And** errors are reported per-story with key and error message
- **And** HTTP 200 is returned with a partial-success response body
- **And** the response includes counts: imported, updated, failed

**AC4: Invalid YAML frontmatter is reported per-story without failing import**
- **Given** a story block with malformed YAML in frontmatter
- **When** parsing fails
- **Then** that story is added to the errors list with a YAML_PARSE_ERROR
- **And** other valid stories in the same import are still processed

**AC5: Admin-only endpoint**
- **Given** a non-admin user calls POST /api/v1/projects/{projectId}/stories/import
- **When** the request is processed
- **Then** HTTP 403 is returned with FORBIDDEN error code

**AC6: Missing required fields (key or title) reported per-story**
- **Given** a story block with no key in frontmatter, or no H1 title in body
- **When** import processes it
- **Then** that story is added to errors with VALIDATION_ERROR
- **And** other stories continue to be processed

## Tasks / Subtasks

- [ ] [BACK] Task 1: Update OpenAPI spec with import endpoint and schemas (AC: #1, #3, #5)
  - [ ] Add `POST /api/v1/projects/{projectId}/stories/import` endpoint to `api/openapi.yaml`
  - [ ] Request body schema: `ImportStoriesRequest` with field `content: string` (raw markdown, required)
  - [ ] Response schema: `ImportStoriesResult` with fields:
    - `imported: integer` — count of newly created stories
    - `updated: integer` — count of updated stories
    - `failed: integer` — count of stories that failed
    - `errors: array of ImportStoryError` (key, message, code)
  - [ ] Add `ImportStoryError` schema with fields: `key: string`, `message: string`, `code: string`
  - [ ] Response HTTP 200 for partial/full success, HTTP 400 for empty content, HTTP 403 for non-admin
  - [ ] Regenerate backend types: `cd backend && make generate`

- [ ] [BACK] Task 2: Implement markdown parser for frontmatter extraction (AC: #1, #4, #6)
  - [ ] Create `backend/internal/adapter/markdown/parser.go`
  - [ ] Add dependency `gopkg.in/yaml.v3` if not already in `go.mod` (check first)
  - [ ] Implement `ParseStoryMarkdown(content string) ([]ParsedStory, error)` that splits multi-document markdown
  - [ ] Split input on `---` separator to identify individual story blocks (each block starts with `---\n...\n---`)
  - [ ] For each block: extract YAML frontmatter between first `---` and second `---`
  - [ ] Parse frontmatter YAML into `FrontmatterFields` struct: `Key`, `Epic`, `DependsOn []string`, `Scope`, `Status`
  - [ ] Extract first H1 heading from markdown body as `Title` (regex `^# (.+)$` on first matching line)
  - [ ] Remaining body (after H1 line stripped) becomes `AcceptanceCriteria`
  - [ ] Return `ParsedStory` slice: each entry has parsed fields + raw parse error if frontmatter was invalid
  - [ ] Blocks with no frontmatter delimiters are skipped (not counted as errors)
  - [ ] Document `ParsedStory` and `FrontmatterFields` structs with godoc comments

- [ ] [BACK] Task 3: Add ImportStory method to StoryService (AC: #1, #2, #3, #4, #6)
  - [ ] Add `Import(ctx context.Context, projectID uuid.UUID, stories []ParsedStory) (*ImportResult, error)` to `backend/internal/domain/service/story_service.go`
  - [ ] Define `ImportResult` struct in service package: `Imported int`, `Updated int`, `Failed int`, `Errors []ImportStoryError`
  - [ ] Define `ImportStoryError` struct: `Key string`, `Message string`, `Code string`
  - [ ] For each parsed story:
    - If parse error present: append to errors with code `YAML_PARSE_ERROR`, increment Failed, continue
    - If key is empty: append to errors with code `VALIDATION_ERROR`, increment Failed, continue
    - If title is empty: append to errors with code `VALIDATION_ERROR`, increment Failed, continue
    - Call `repo.GetByKey(ctx, projectID, key)`:
      - If not found: call `repo.Create(ctx, story)`, increment Imported on success
      - If found: call `repo.Update(ctx, existing)` with new fields, increment Updated on success
      - On create/update error: append to errors with code `IMPORT_ERROR`, increment Failed
  - [ ] Service depends only on `port.StoryRepository` — no imports from adapter/markdown
  - [ ] `Import` does NOT use a transaction (partial success is intentional — each story is independent)

- [ ] [BACK] Task 4: Create ImportStories HTTP handler (AC: #1, #3, #5)
  - [ ] Add `ImportStories` method to `backend/internal/api/handler/story_handler.go`
  - [ ] Route: `POST /api/v1/projects/{projectId}/stories/import`
  - [ ] RBAC: admin only — return 403 if not admin
  - [ ] Parse `projectId` from URL param (UUID), return 400 if invalid
  - [ ] Decode request body as `ImportStoriesRequest` JSON: `{ "content": "<markdown string>" }`
  - [ ] Validate content is non-empty string, return 400 with VALIDATION_ERROR if empty
  - [ ] Call `markdown.ParseStoryMarkdown(req.Content)` to get `[]ParsedStory`
  - [ ] Call `storyService.Import(ctx, projectID, parsedStories)` to get `*ImportResult`
  - [ ] Map `ImportResult` to `ImportStoriesResult` response schema and render HTTP 200 JSON
  - [ ] Register route in chi router under `/api/v1/projects/{projectId}/stories/import` (must be declared before `/{storyId}` to avoid chi route conflict)

- [ ] [BACK] Task 5: Unit tests for markdown parser (AC: #1, #4, #6)
  - [ ] Create `backend/internal/adapter/markdown/parser_test.go`
  - [ ] Test single story block: valid frontmatter + H1 title → correct ParsedStory fields
  - [ ] Test multi-story block: two story blocks separated by `---` → two ParsedStory entries
  - [ ] Test invalid YAML frontmatter: ParsedStory has non-nil ParseError, other fields empty
  - [ ] Test missing H1 title: ParsedStory has empty Title (handled by service as VALIDATION_ERROR)
  - [ ] Test frontmatter with all optional fields missing (only key): partial ParsedStory
  - [ ] Test content with no frontmatter delimiters: returns empty slice (no stories, no error)
  - [ ] Test depends_on as YAML list: correctly parsed as `[]string`

- [ ] [BACK] Task 6: Unit tests for StoryService.Import (AC: #2, #3, #4, #6)
  - [ ] Add import test cases to `backend/internal/domain/service/story_service_test.go`
  - [ ] Test all-new stories: Import returns Imported=N, Updated=0, Failed=0, Errors=[]
  - [ ] Test all-existing stories (by key): Import returns Imported=0, Updated=N, Failed=0, Errors=[]
  - [ ] Test mix of new and existing: correct Imported/Updated counts
  - [ ] Test parse error in ParsedStory: story added to Errors with YAML_PARSE_ERROR, others processed
  - [ ] Test empty key in ParsedStory: story added to Errors with VALIDATION_ERROR
  - [ ] Test empty title in ParsedStory: story added to Errors with VALIDATION_ERROR
  - [ ] Test repo.Create failure: story added to Errors with IMPORT_ERROR
  - [ ] Use mock StoryRepository

- [ ] [BACK] Task 7: Wire import handler and verify (AC: #1-6)
  - [ ] Ensure `markdown` package is importable from handler (no circular dependency)
  - [ ] Inject markdown parser call in handler (handler imports `adapter/markdown`, calls `ParseStoryMarkdown`)
  - [ ] Run `go build ./...` — must compile successfully
  - [ ] Run `golangci-lint run ./...` — must pass with no errors
  - [ ] Manual test: admin POST `/api/v1/projects/{projectId}/stories/import` with single story markdown → 200 with imported=1
  - [ ] Manual test: second POST with same key → 200 with updated=1, imported=0
  - [ ] Manual test: two valid + one invalid YAML → 200 with imported=2, failed=1, errors=[{key, message}]
  - [ ] Manual test: non-admin POST → 403

## Dev Notes

### Dependencies

**Story 2-2 (Stories table + Story CRUD API — DONE):** Provides `StoryRepository` port, `StoryService`, `StoryHandler`, and the stories table. This story extends `StoryService` with an `Import` method and adds one new handler method to the existing handler. Both the port interface and service must already exist.

### Architecture Requirements

**Hexagonal boundaries:**
- Markdown parser lives in `backend/internal/adapter/markdown/` — it is an input adapter (converts raw text to domain inputs), NOT in domain layer
- `StoryService.Import` receives `[]ParsedStory` (defined in the markdown package or a shared input DTO) — the service must not import `adapter/markdown`
- Define `ParsedStory` as an input DTO in the service package or as a shared type that both handler and service can import without creating circular dependencies
- Recommended: define `ParsedStory` in `backend/internal/domain/service/story_service.go` (or a sibling file `story_import.go`) and have the markdown adapter return this type from the `service` package
- Alternative (cleaner): handler calls `markdown.ParseStoryMarkdown` → gets `[]markdown.ParsedStory` → maps to `[]service.ImportStoryInput` → calls `service.Import`
- Handler depends on service and on adapter/markdown — this is acceptable (handler is at the boundary)

**File structure:**

```
backend/
├── internal/
│   ├── adapter/
│   │   └── markdown/
│   │       ├── parser.go          # ParseStoryMarkdown function + ParsedStory, FrontmatterFields
│   │       └── parser_test.go     # Unit tests
│   ├── domain/
│   │   └── service/
│   │       ├── story_service.go   # Add Import method + ImportResult, ImportStoryError types
│   │       └── story_service_test.go  # Add Import unit tests
│   └── api/
│       └── handler/
│           └── story_handler.go   # Add ImportStories handler + route registration
└── go.mod                         # Ensure gopkg.in/yaml.v3 is present
```

### File Paths (exact)

- Markdown parser: `backend/internal/adapter/markdown/parser.go`
- Markdown parser tests: `backend/internal/adapter/markdown/parser_test.go`
- Story service (extend): `backend/internal/domain/service/story_service.go`
- Story service tests (extend): `backend/internal/domain/service/story_service_test.go`
- Story handler (extend): `backend/internal/api/handler/story_handler.go`
- OpenAPI spec: `api/openapi.yaml`

### Technical Specifications

**Request body (`ImportStoriesRequest`):**
```json
{
  "content": "---\nkey: S-01\nscope: backend\n---\n# Story Title\n\nBody content..."
}
```

**Response body (`ImportStoriesResult` — HTTP 200):**
```json
{
  "imported": 2,
  "updated": 1,
  "failed": 1,
  "errors": [
    {
      "key": "S-05",
      "message": "invalid YAML frontmatter: yaml: unmarshal error",
      "code": "YAML_PARSE_ERROR"
    }
  ]
}
```

**Markdown format accepted (single story block):**
```markdown
---
key: S-03
epic: E-01
depends_on:
  - S-01
  - S-02
scope: backend
status: backlog
---
# Story Title Here

Acceptance criteria and other body text here.
```

**Multi-story import (multiple blocks separated by `---`):**
```markdown
---
key: S-03
scope: backend
---
# First Story

Body of first story.

---
key: S-04
scope: frontend
depends_on:
  - S-03
---
# Second Story

Body of second story.
```

**`FrontmatterFields` struct (in `adapter/markdown`):**
```go
// FrontmatterFields represents the parsed YAML frontmatter of a story block.
type FrontmatterFields struct {
    Key       string   `yaml:"key"`
    Epic      string   `yaml:"epic"`
    DependsOn []string `yaml:"depends_on"`
    Scope     string   `yaml:"scope"`
    Status    string   `yaml:"status"`
}
```

**`ParsedStory` struct (in `adapter/markdown`):**
```go
// ParsedStory holds the result of parsing one story block from markdown.
// ParseError is non-nil if YAML frontmatter failed to parse.
type ParsedStory struct {
    Key                string
    Title              string
    Epic               string
    DependsOn          []string
    Scope              string
    Status             string
    AcceptanceCriteria string
    ParseError         error // non-nil if YAML parsing failed for this block
}
```

**`ParseStoryMarkdown` function:**
```go
// ParseStoryMarkdown splits a markdown document into individual story blocks
// and parses the YAML frontmatter and title from each.
// Blocks are delimited by lines consisting solely of "---".
// Returns one ParsedStory per detected block (with ParseError set for invalid YAML).
func ParseStoryMarkdown(content string) ([]ParsedStory, error) {
    // Split on "---" delimiter lines
    // For each block: extract frontmatter (between first --- and second ---)
    // Parse YAML into FrontmatterFields
    // Extract first "# Title" line from body
    // Return []ParsedStory
}
```

**`ImportResult` and `ImportStoryError` types (add to `story_service.go`):**
```go
// ImportResult holds the aggregated result of a markdown import operation.
type ImportResult struct {
    Imported int
    Updated  int
    Failed   int
    Errors   []ImportStoryError
}

// ImportStoryError describes a per-story import failure.
type ImportStoryError struct {
    Key     string
    Message string
    Code    string
}
```

**Error codes used:**
- `YAML_PARSE_ERROR` — frontmatter YAML is invalid for this story block
- `VALIDATION_ERROR` — missing required field (key or title) in a story block
- `IMPORT_ERROR` — database-level error during create or update for this story
- `FORBIDDEN` — non-admin attempting import (403)
- `VALIDATION_ERROR` — empty content body (400)

**OpenAPI additions to `api/openapi.yaml`:**
```yaml
# Under paths:
  /projects/{projectId}/stories/import:
    post:
      operationId: importStories
      summary: Bulk import stories from markdown with YAML frontmatter
      tags: [stories]
      parameters:
        - $ref: "#/components/parameters/ProjectIdPath"
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/ImportStoriesRequest"
      responses:
        "200":
          description: Import result with per-story success and error counts
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ImportStoriesResult"
        "400":
          $ref: "#/components/responses/BadRequest"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "403":
          $ref: "#/components/responses/Forbidden"

# Under components/schemas:
  ImportStoriesRequest:
    type: object
    required: [content]
    properties:
      content:
        type: string
        description: Raw markdown content with YAML frontmatter story blocks

  ImportStoriesResult:
    type: object
    required: [imported, updated, failed, errors]
    properties:
      imported:
        type: integer
        description: Number of newly created stories
      updated:
        type: integer
        description: Number of updated stories (key already existed)
      failed:
        type: integer
        description: Number of stories that failed to import
      errors:
        type: array
        items:
          $ref: "#/components/schemas/ImportStoryError"

  ImportStoryError:
    type: object
    required: [key, message, code]
    properties:
      key:
        type: string
        description: Story key (may be empty if key could not be extracted)
      message:
        type: string
        description: Human-readable error description
      code:
        type: string
        description: Machine-readable error code
```

**Chi route registration note:** The import route `/projects/{projectId}/stories/import` must be registered BEFORE the parameterized route `/projects/{projectId}/stories/{storyId}` to avoid chi treating `import` as a storyId value. In the chi router:
```go
r.Route("/api/v1/projects/{projectId}/stories", func(r chi.Router) {
    r.Get("/", h.ListStories)
    r.Post("/", h.CreateStory)
    r.Post("/import", h.ImportStories)  // must come before /{storyId}
    r.Route("/{storyId}", func(r chi.Router) {
        r.Get("/", h.GetStory)
        r.Put("/", h.UpdateStory)
        r.Delete("/", h.DeleteStory)
    })
})
```

**`gopkg.in/yaml.v3` dependency:** Check if already present in `backend/go.mod`. If not, add with `go get gopkg.in/yaml.v3` inside the `backend/` directory.

### Testing Requirements

**Manual verification checklist:**
1. `go build ./...` — compiles successfully
2. `golangci-lint run ./...` — no errors
3. Admin POST `/api/v1/projects/{id}/stories/import` with single story block → 200 `{"imported":1,"updated":0,"failed":0,"errors":[]}`
4. Second POST with same key → 200 `{"imported":0,"updated":1,"failed":0,"errors":[]}`
5. POST with two valid stories + one invalid YAML → 200 `{"imported":2,"updated":0,"failed":1,"errors":[{"key":"","message":"...","code":"YAML_PARSE_ERROR"}]}`
6. POST with story missing H1 title → 200 `{"imported":0,"updated":0,"failed":1,"errors":[{"key":"S-01","message":"title is required","code":"VALIDATION_ERROR"}]}`
7. POST with empty `content` → 400 with VALIDATION_ERROR
8. Non-admin POST → 403 with FORBIDDEN

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 2.3]
- [Source: Story 2-2 — existing StoryService, StoryRepository port, StoryHandler patterns]
- [Source: backend/CLAUDE.md — hexagonal architecture, chi router patterns, DomainError, slog]
- [Source: api/openapi.yaml — existing stories endpoints for pattern consistency]

## Dev Agent Record

## Change Log

- 2026-02-17: Story created for Wave 6 story board management
