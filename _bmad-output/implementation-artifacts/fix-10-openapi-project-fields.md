# Story fix-10: [SHARED] Expose project infrastructure fields in OpenAPI spec

Status: ready-for-dev

## Story

As a frontend developer or API consumer,
I want `CreateProjectRequest`, `UpdateProjectRequest`, and the `Project` response schema to include all infrastructure fields (`repo_url`, `git_provider`, `git_token_env`, `agent_runtime`, `default_model`),
so that projects can be fully configured through the API without having to resort to direct database manipulation.

## Context

The `projects` table (migration `000002`) and the `model.Project` struct both carry five infrastructure fields that are never exposed through the OpenAPI contract:

| Field | DB column | Model field |
|-------|-----------|-------------|
| Repository URL | `repo_url TEXT` | `RepoURL *string` |
| Git provider | `git_provider VARCHAR(50) NOT NULL DEFAULT 'github'` | `GitProvider string` |
| Git token env var name | `git_token_env VARCHAR(255)` | `GitTokenEnv *string` |
| Agent runtime | `agent_runtime VARCHAR(50) NOT NULL DEFAULT 'docker'` | `AgentRuntime string` |
| Default Claude model | `default_model VARCHAR(100)` | `DefaultModel *string` |

The sqlc queries (`backend/queries/projects.sql`) already include all five columns in both `CreateProject` and `UpdateProject` — no DB or sqlc changes are required. The gap is entirely in:

1. `api/openapi.yaml` — schemas missing the fields
2. `backend/internal/api/handler/project_handler.go` — handler not reading/passing the fields
3. `backend/internal/domain/service/project_service.go` — `CreateProjectParams` / `UpdateProjectParams` missing the fields
4. `backend/internal/api/handler/helpers.go` — `toAPIProject()` not mapping the fields to the response

## Acceptance Criteria (BDD)

**AC1: OpenAPI spec includes all infrastructure fields in `Project` response**
- **Given** the `api/openapi.yaml` `Project` schema
- **When** I inspect the `properties` block
- **Then** `repo_url`, `git_provider`, `git_token_env`, `agent_runtime`, and `default_model` are present with correct types and descriptions

**AC2: `CreateProjectRequest` accepts infrastructure fields**
- **Given** the `api/openapi.yaml` `CreateProjectRequest` schema
- **When** I POST a project with `repo_url`, `git_provider`, `git_token_env`, `agent_runtime`, `default_model` in the body
- **Then** the request is accepted (not rejected with 400) and the values are persisted

**AC3: `UpdateProjectRequest` accepts infrastructure fields**
- **Given** an existing project
- **When** I PUT `repo_url`, `git_provider`, `git_token_env`, `agent_runtime`, `default_model` fields
- **Then** the project is updated with the new values and the response reflects them

**AC4: GET `/projects/{id}` response includes all infrastructure fields**
- **Given** a project with `repo_url = "https://github.com/org/repo"`, `git_provider = "github"`, `agent_runtime = "docker"`
- **When** I call `GET /api/v1/projects/{id}`
- **Then** the JSON response contains `repo_url`, `git_provider`, `git_token_env`, `agent_runtime`, `default_model`

**AC5: `git_provider` defaults to `github` and `agent_runtime` defaults to `docker` when not supplied on create**
- **Given** a `CreateProjectRequest` with only `name` set
- **When** the project is created
- **Then** `git_provider` is `"github"` and `agent_runtime` is `"docker"` in the response (matching the DB defaults already enforced in `project_service.go`)

**AC6: Backend code generation and lint pass**
- **Given** the updated `api/openapi.yaml`
- **When** I run `cd backend && make generate` followed by `golangci-lint run ./...`
- **Then** both commands exit with code 0 and the generated types include the new fields

## Tasks / Subtasks

- [ ] **Task 1: Update `api/openapi.yaml`**
  - [ ] Add `repo_url`, `git_provider`, `git_token_env`, `agent_runtime`, `default_model` to the `Project` response schema
  - [ ] Add the same fields to `CreateProjectRequest`
  - [ ] Add the same fields to `UpdateProjectRequest`
  - [ ] Ensure `git_provider` and `agent_runtime` use `enum` values to constrain to known options

- [ ] **Task 2: Regenerate backend Go types**
  - [ ] Run `cd backend && make generate` — oapi-codegen must produce updated `CreateProjectRequest`, `UpdateProjectRequest`, `Project` structs

- [ ] **Task 3: Update `project_service.go` — `CreateProjectParams` and `UpdateProjectParams`**
  - [ ] Add `RepoURL *string`, `GitProvider *string`, `GitTokenEnv *string`, `AgentRuntime *string`, `DefaultModel *string` to `CreateProjectParams`
  - [ ] Apply the new params in `Create()` (override defaults only when non-nil)
  - [ ] Add same fields to `UpdateProjectParams` and apply them in `Update()`

- [ ] **Task 4: Update `project_handler.go` — read new fields from request**
  - [ ] In `CreateProject`, map `req.RepoUrl`, `req.GitProvider`, `req.GitTokenEnv`, `req.AgentRuntime`, `req.DefaultModel` to the service params
  - [ ] In `UpdateProject`, map the same fields to `UpdateProjectParams`

- [ ] **Task 5: Update `helpers.go` — `toAPIProject()` response mapping**
  - [ ] Map `p.RepoURL`, `p.GitProvider`, `p.GitTokenEnv`, `p.AgentRuntime`, `p.DefaultModel` onto the generated `Project` API type
  - [ ] Handle nullable pointer fields with nil checks

- [ ] **Task 6: Verify sqlc query params**
  - [ ] Confirm `backend/internal/adapter/postgres/` `CreateProject` and `UpdateProject` repository methods already pass all five columns (they should — sqlc-generated from the existing SQL)
  - [ ] If the repository `Create()` or `Update()` methods need updating to accept the new model fields, fix them

- [ ] **Task 7: Run lint and unit tests**
  - [ ] `cd backend && golangci-lint run ./...` — must be clean
  - [ ] `cd backend && go test ./... -short` — must pass

## Dev Notes

### Files to Modify

| File | Change |
|------|--------|
| `api/openapi.yaml` | Add fields to `Project`, `CreateProjectRequest`, `UpdateProjectRequest` schemas |
| `backend/internal/api/handler/helpers.go` | Update `toAPIProject()` to map new fields |
| `backend/internal/api/handler/project_handler.go` | Read new fields from request in `CreateProject` and `UpdateProject` |
| `backend/internal/domain/service/project_service.go` | Add fields to `CreateProjectParams`, `UpdateProjectParams`, apply in `Create()` and `Update()` |

### Files to Regenerate (not manually edited)

| File | Command |
|------|---------|
| `backend/internal/api/handler/` oapi-codegen output | `cd backend && make generate` |

### No DB or sqlc changes needed

The `projects.sql` queries already include all five fields:

```sql
-- CreateProject already has: repo_url, git_provider, git_token_env, agent_runtime, default_model
INSERT INTO projects (name, description, owner_id, repo_url, git_provider, git_token_env, agent_runtime, default_model, max_budget)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- UpdateProject already has COALESCE for all five fields
UPDATE projects
SET name = COALESCE(sqlc.narg('name'), name),
    repo_url = COALESCE(sqlc.narg('repo_url'), repo_url),
    git_provider = COALESCE(sqlc.narg('git_provider'), git_provider),
    git_token_env = COALESCE(sqlc.narg('git_token_env'), git_token_env),
    agent_runtime = COALESCE(sqlc.narg('agent_runtime'), agent_runtime),
    default_model = COALESCE(sqlc.narg('default_model'), default_model),
    ...
WHERE id = @id
RETURNING *;
```

### Exact OpenAPI YAML changes

**In `Project` schema** (after `owner_id`, before `max_budget` — around line 1896):

```yaml
        repo_url:
          type: string
          nullable: true
          description: URL of the project's git repository
          example: "https://github.com/org/my-project"
        git_provider:
          type: string
          enum: [github, gitlab, bitbucket]
          description: Git provider for this project
          example: github
        git_token_env:
          type: string
          nullable: true
          description: Name of the environment variable holding the git token (not the token itself)
          example: GITHUB_TOKEN
        agent_runtime:
          type: string
          enum: [docker, kubernetes]
          description: Container runtime used to execute agent steps
          example: docker
        default_model:
          type: string
          nullable: true
          description: Default Claude model override for agent steps in this project
          example: claude-opus-4-6
```

**In `CreateProjectRequest` schema** (after `description` property — around line 1938):

```yaml
        repo_url:
          type: string
          nullable: true
          description: URL of the project's git repository
          example: "https://github.com/org/my-project"
        git_provider:
          type: string
          enum: [github, gitlab, bitbucket]
          default: github
          description: Git provider for this project
          example: github
        git_token_env:
          type: string
          nullable: true
          description: Name of the environment variable holding the git token (not the token itself)
          example: GITHUB_TOKEN
        agent_runtime:
          type: string
          enum: [docker, kubernetes]
          default: docker
          description: Container runtime used to execute agent steps
          example: docker
        default_model:
          type: string
          nullable: true
          description: Default Claude model override for agent steps in this project
          example: claude-opus-4-6
```

**In `UpdateProjectRequest` schema** (after `max_budget` property — around line 1957):

```yaml
        repo_url:
          type: string
          nullable: true
          description: URL of the project's git repository
          example: "https://github.com/org/my-project"
        git_provider:
          type: string
          enum: [github, gitlab, bitbucket]
          description: Git provider for this project
          example: github
        git_token_env:
          type: string
          nullable: true
          description: Name of the environment variable holding the git token (not the token itself)
          example: GITHUB_TOKEN
        agent_runtime:
          type: string
          enum: [docker, kubernetes]
          description: Container runtime used to execute agent steps
          example: docker
        default_model:
          type: string
          nullable: true
          description: Default Claude model override for agent steps in this project
          example: claude-opus-4-6
```

### Go code changes

**`backend/internal/domain/service/project_service.go`** — extend params structs:

```go
// CreateProjectParams holds parameters for creating a project.
type CreateProjectParams struct {
    Name         string
    Description  *string
    OwnerID      *uuid.UUID
    RepoURL      *string
    GitProvider  *string  // defaults to "github" if nil
    GitTokenEnv  *string
    AgentRuntime *string  // defaults to "docker" if nil
    DefaultModel *string
}

// In Create():
project := &model.Project{
    Name:         params.Name,
    Description:  params.Description,
    OwnerID:      params.OwnerID,
    RepoURL:      params.RepoURL,
    GitProvider:  "github", // default
    AgentRuntime: "docker", // default
    GitTokenEnv:  params.GitTokenEnv,
    DefaultModel: params.DefaultModel,
}
if params.GitProvider != nil && *params.GitProvider != "" {
    project.GitProvider = *params.GitProvider
}
if params.AgentRuntime != nil && *params.AgentRuntime != "" {
    project.AgentRuntime = *params.AgentRuntime
}
```

```go
// UpdateProjectParams holds parameters for updating a project.
type UpdateProjectParams struct {
    ID           uuid.UUID
    Name         *string
    Description  *string
    MaxBudget    *float64
    SetBudget    bool
    RepoURL      *string
    SetRepoURL   bool  // allows explicit nil (clearing the field)
    GitProvider  *string
    GitTokenEnv  *string
    SetTokenEnv  bool  // allows explicit nil (clearing the field)
    AgentRuntime *string
    DefaultModel *string
    SetModel     bool  // allows explicit nil (clearing the field)
}

// In Update(), after existing MaxBudget block:
if params.GitProvider != nil {
    existing.GitProvider = *params.GitProvider
}
if params.AgentRuntime != nil {
    existing.AgentRuntime = *params.AgentRuntime
}
if params.SetRepoURL {
    existing.RepoURL = params.RepoURL
}
if params.SetTokenEnv {
    existing.GitTokenEnv = params.GitTokenEnv
}
if params.SetModel {
    existing.DefaultModel = params.DefaultModel
}
```

**`backend/internal/api/handler/project_handler.go`** — map fields in handlers:

```go
// In CreateProject():
params := service.CreateProjectParams{
    Name:         req.Name,
    Description:  req.Description,
    RepoURL:      req.RepoUrl,       // oapi-codegen generates RepoUrl from repo_url
    GitProvider:  req.GitProvider,
    GitTokenEnv:  req.GitTokenEnv,
    AgentRuntime: req.AgentRuntime,
    DefaultModel: req.DefaultModel,
}

// In UpdateProject():
params := service.UpdateProjectParams{
    ID:           id,
    Name:         req.Name,
    Description:  req.Description,
    MaxBudget:    req.MaxBudget,
    SetBudget:    req.MaxBudget != nil,
    GitProvider:  req.GitProvider,
    AgentRuntime: req.AgentRuntime,
    RepoURL:      req.RepoUrl,
    SetRepoURL:   req.RepoUrl != nil,
    GitTokenEnv:  req.GitTokenEnv,
    SetTokenEnv:  req.GitTokenEnv != nil,
    DefaultModel: req.DefaultModel,
    SetModel:     req.DefaultModel != nil,
}
```

> **Note on oapi-codegen field naming:** oapi-codegen converts `snake_case` YAML fields to `CamelCase` Go struct fields. `repo_url` becomes `RepoUrl`, `git_provider` becomes `GitProvider`, etc. Verify against the generated output after running `make generate`.

**`backend/internal/api/handler/helpers.go`** — extend `toAPIProject()`:

```go
func toAPIProject(p *model.Project) Project {
    proj := Project{
        Id:                   p.ID,
        Name:                 p.Name,
        OwnerId:              uuid.Nil,
        GitProvider:          p.GitProvider,
        AgentRuntime:         p.AgentRuntime,
        MaxBudget:            p.MaxBudget,
        CircuitBreakerCount:  p.CircuitBreakerCount,
        CircuitBreakerActive: p.CircuitBreakerActive,
        CircuitBreakerMax:    p.CircuitBreakerMax,
        CreatedAt:            p.CreatedAt,
        UpdatedAt:            p.UpdatedAt,
    }
    if p.Description != nil {
        proj.Description = p.Description
    }
    if p.OwnerID != nil {
        proj.OwnerId = *p.OwnerID
    }
    if p.RepoURL != nil {
        proj.RepoUrl = p.RepoURL
    }
    if p.GitTokenEnv != nil {
        proj.GitTokenEnv = p.GitTokenEnv
    }
    if p.DefaultModel != nil {
        proj.DefaultModel = p.DefaultModel
    }
    return proj
}
```

### Downstream impact

- **fix-12-frontend-api-regen** depends on this story: once this spec change is merged, the frontend agent regenerates its API client and picks up the new types
- **fix-13-project-repo-form** depends on both fix-10 and fix-12: the frontend form for setting repo URL and git provider cannot be built until the API types exist

### Validation hint

After `make generate`, search the generated handler file for `RepoUrl` (or the exact field name oapi-codegen produces) to confirm the struct was regenerated correctly before touching the handler mapping code.

## Dev Agent Record

_To be filled by the implementing agent._

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-22 | story-writer | Initial story created |
