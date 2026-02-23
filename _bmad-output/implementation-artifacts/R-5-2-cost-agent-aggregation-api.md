# Story R-5-2: [BACK] Cost service + handler for aggregation by agent

Status: ready-for-dev

## Story

As a **platform developer**,
I want a dedicated API endpoint that returns cost aggregations grouped by agent,
so that the frontend can display per-agent cost breakdowns and the API contract exposes token counts on all cost schemas.

## Acceptance Criteria (BDD)

### Scenario 1: GET /api/v1/projects/{projectId}/costs/agents returns agent breakdown

```gherkin
Given a project with cost records associated to multiple agents
When I call GET /api/v1/projects/{projectId}/costs/agents with a valid auth token
Then I receive HTTP 200
  And the response body is a JSON array of AgentCostBreakdown objects
  And each object contains: agent_id, agent_name, tokens_input, tokens_output, cost_usd, runs_count
  And the list is ordered by cost_usd descending
```

### Scenario 2: Endpoint returns empty array when no agent-linked costs exist

```gherkin
Given a project with cost records that have no agent_id set
When I call GET /api/v1/projects/{projectId}/costs/agents
Then I receive HTTP 200
  And the response body is an empty array []
```

### Scenario 3: Unauthenticated request is rejected

```gherkin
Given no auth token is provided
When I call GET /api/v1/projects/{projectId}/costs/agents
Then I receive HTTP 401
```

### Scenario 4: AgentCostBreakdown schema is defined in OpenAPI spec

```gherkin
Given the updated api/openapi.yaml
When I inspect components/schemas
Then "AgentCostBreakdown" schema is present with fields:
  | field         | type    | format | required |
  | agent_id      | string  | uuid   | yes      |
  | agent_name    | string  |        | yes      |
  | tokens_input  | integer | int64  | yes      |
  | tokens_output | integer | int64  | yes      |
  | cost_usd      | number  | double | yes      |
  | runs_count    | integer | int32  | yes      |
```

### Scenario 5: Existing cost schemas expose tokens_input and tokens_output

```gherkin
Given the updated api/openapi.yaml
When I inspect the "CostRecord" schema
Then tokens_input and tokens_output fields are present with type integer and format int64
```

### Scenario 6: Backend and frontend code generation succeeds

```gherkin
Given the updated api/openapi.yaml
When I run "cd backend && make generate" and "cd frontend && npm run generate-api"
Then both commands exit with code 0
  And the generated Go server interface includes GetProjectCostsByAgent handler method
  And the generated TypeScript types include AgentCostBreakdown
```

### Scenario 7: Unit tests pass for service and handler

```gherkin
Given the new CostService.GetProjectCostsByAgent method and its HTTP handler
When I run "cd backend && go test ./... -short"
Then all tests pass including tests for the new service method and handler
```

## Tasks / Subtasks

- [ ] **1.1** [SHARED] Update `api/openapi.yaml` — add `AgentCostBreakdown` schema (AC: #4)
  - [ ] Fields: agent_id (uuid), agent_name (string), tokens_input (int64), tokens_output (int64), cost_usd (number), runs_count (int32)

- [ ] **1.2** [SHARED] Update `api/openapi.yaml` — add endpoint `GET /api/v1/projects/{projectId}/costs/agents` (AC: #1, #2, #3)
  - [ ] operationId: `getProjectCostsByAgent`
  - [ ] Response 200: array of `AgentCostBreakdown`
  - [ ] Response 401: `$ref: "#/components/responses/Unauthorized"`
  - [ ] Response 404: `$ref: "#/components/responses/NotFound"`

- [ ] **1.3** [SHARED] Update `api/openapi.yaml` — ensure `CostRecord` schema includes `tokens_input` and `tokens_output` fields (AC: #5)
  - [ ] Add `tokens_input: {type: integer, format: int64}` if missing
  - [ ] Add `tokens_output: {type: integer, format: int64}` if missing

- [ ] **1.4** [BACK] Run `cd backend && make generate` and fix any compilation errors (AC: #6)

- [ ] **1.5** [FRONT] Run `cd frontend && npm run generate-api` (AC: #6)

- [ ] **1.6** [BACK] Add `GetProjectCostsByAgent(ctx context.Context, projectID uuid.UUID) ([]model.AgentCostBreakdown, error)` method to `CostService` in `backend/internal/domain/service/cost_service.go` (AC: #1)
  - [ ] Delegates to `CostRepository.ListByProjectByAgent`

- [ ] **1.7** [BACK] Implement the `getProjectCostsByAgent` handler in `backend/internal/api/handler/cost_handler.go` (AC: #1, #2, #3)
  - [ ] Parse projectId from path
  - [ ] Call CostService.GetProjectCostsByAgent
  - [ ] Return JSON array (empty array if no results)

- [ ] **1.8** [BACK] Wire the new handler in the router/server (AC: #6)

- [ ] **1.9** [BACK] Write unit tests for CostService.GetProjectCostsByAgent (AC: #7)
  - [ ] Mock repository returning sample breakdowns
  - [ ] Mock repository returning empty slice

- [ ] **1.10** [BACK] Write unit tests for the handler (AC: #7)
  - [ ] 200 with results
  - [ ] 200 with empty results
  - [ ] 401 unauthenticated

- [ ] **1.11** [BACK] Lint and full test run (AC: #7)
  - [ ] `cd backend && golangci-lint run ./...`
  - [ ] `cd backend && go test ./... -short`

## Dev Notes

### Dependencies

- **R-5-1** — requires `agent_id` column on cost_records, `AgentCostBreakdown` model, and `CostRepository.ListByProjectByAgent` port method to be in place before this story can implement the service and handler.

### Architecture Requirements

- OpenAPI spec is updated first, code generation runs, then implementation follows — never the reverse
- `CostService` remains in `backend/internal/domain/service/` — no direct DB access from handler
- Handler only does HTTP parsing/serialization; all business logic in service
- The endpoint is protected by the existing JWT auth middleware

### Technical Specifications

**OpenAPI endpoint definition:**

```yaml
/api/v1/projects/{projectId}/costs/agents:
  get:
    operationId: getProjectCostsByAgent
    summary: Get cost breakdown aggregated by agent for a project
    tags: [costs]
    parameters:
      - $ref: "#/components/parameters/ProjectIdPath"
    responses:
      "200":
        description: Agent cost breakdown
        content:
          application/json:
            schema:
              type: array
              items:
                $ref: "#/components/schemas/AgentCostBreakdown"
      "401":
        $ref: "#/components/responses/Unauthorized"
      "404":
        $ref: "#/components/responses/NotFound"
```

**OpenAPI AgentCostBreakdown schema:**

```yaml
AgentCostBreakdown:
  type: object
  required:
    - agent_id
    - agent_name
    - tokens_input
    - tokens_output
    - cost_usd
    - runs_count
  properties:
    agent_id:
      type: string
      format: uuid
    agent_name:
      type: string
    tokens_input:
      type: integer
      format: int64
    tokens_output:
      type: integer
      format: int64
    cost_usd:
      type: number
      format: double
    runs_count:
      type: integer
      format: int32
```

**CostService method:**

```go
// GetProjectCostsByAgent returns cost aggregations grouped by agent for a project.
func (s *CostService) GetProjectCostsByAgent(ctx context.Context, projectID uuid.UUID) ([]model.AgentCostBreakdown, error)
```

### Testing Requirements

- Use table-driven tests for the handler (httptest.NewRecorder)
- Mock `CostRepository` interface for service unit tests
- Do not use integration tests (testcontainers) for this story — unit tests only

### References

- `api/openapi.yaml` — add AgentCostBreakdown schema and endpoint
- `backend/internal/domain/service/cost_service.go` — add GetProjectCostsByAgent
- `backend/internal/api/handler/cost_handler.go` — add handler implementation
- `backend/internal/domain/port/cost_repository.go` — ListByProjectByAgent (defined in R-5-1)
- `backend/internal/domain/model/cost_record.go` — AgentCostBreakdown struct (defined in R-5-1)

## Dev Agent Record

## Change Log
