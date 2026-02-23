# Story R-1-2: [SHARED] Add Agent entity to OpenAPI spec (replaces PromptTemplate)

Status: ready-for-dev

## Story

As a **platform developer**,
I want an `Agent` schema and CRUD endpoints defined in the OpenAPI spec,
so that both the backend and frontend can generate strongly-typed code for managing agents (the replacement for prompt templates) with support for global and project-scoped agents.

## Acceptance Criteria (BDD)

### Scenario 1: Agent schema exists in the spec

```gherkin
Given the file api/openapi.yaml is loaded
When I inspect the components/schemas section
Then an "Agent" schema is present with:
  | field            | type      | required | notes                      |
  | id               | string    | yes      | uuid format                |
  | name             | string    | yes      |                            |
  | model            | string    | yes      | e.g. "claude-opus-4-6"     |
  | image            | string    | yes      | Docker image reference     |
  | template_content | string    | yes      | Handlebars prompt template |
  | scope            | string    | yes      | enum: global / project     |
  | project_id       | string    | no       | uuid, nullable             |
  | created_at       | string    | yes      | ISO 8601 datetime          |
  | updated_at       | string    | yes      | ISO 8601 datetime          |
```

### Scenario 2: Project-scoped Agent CRUD endpoints exist

```gherkin
Given the updated api/openapi.yaml
When I inspect the paths section
Then the following endpoints are defined:
  | method | path                                           | operationId           |
  | GET    | /api/v1/projects/{projectId}/agents            | listProjectAgents     |
  | POST   | /api/v1/projects/{projectId}/agents            | createAgent           |
  | GET    | /api/v1/projects/{projectId}/agents/{agentId}  | getAgent              |
  | PUT    | /api/v1/projects/{projectId}/agents/{agentId}  | updateAgent           |
  | DELETE | /api/v1/projects/{projectId}/agents/{agentId}  | deleteAgent           |
```

### Scenario 3: Global Agent list endpoint exists

```gherkin
Given the updated api/openapi.yaml
When I inspect the paths section
Then a "GET /api/v1/agents" endpoint is defined with operationId "listGlobalAgents"
  And it returns a paginated list of Agent objects where scope is "global"
```

### Scenario 4: Existing template endpoints are preserved

```gherkin
Given the updated api/openapi.yaml
When I inspect the paths section
Then the existing endpoints under /api/v1/projects/{projectId}/templates still exist
  And they are marked as deprecated via the "deprecated: true" OpenAPI flag
  And they are not removed
```

### Scenario 5: Backend code generation succeeds

```gherkin
Given the updated api/openapi.yaml
When I run "cd backend && make generate"
Then the command exits with code 0
  And the generated Go server interface includes methods for all new Agent endpoints
  And the generated types include the Agent struct
```

### Scenario 6: Frontend code generation succeeds

```gherkin
Given the updated api/openapi.yaml
When I run "cd frontend && npm run generate-api"
Then the command exits with code 0
  And the generated TypeScript types include the Agent interface
  And the generated path types include all new Agent endpoint paths
```

## Technical Notes

### New Schema — `Agent`

```yaml
Agent:
  type: object
  required:
    - id
    - name
    - model
    - image
    - template_content
    - scope
    - created_at
    - updated_at
  properties:
    id:
      type: string
      format: uuid
    name:
      type: string
      minLength: 1
      maxLength: 255
    model:
      type: string
      description: LLM model identifier (e.g. "claude-opus-4-6", "claude-sonnet-4-6")
    image:
      type: string
      description: Docker image reference for the agent runtime container
    template_content:
      type: string
      description: Handlebars prompt template content
    scope:
      type: string
      enum:
        - global
        - project
      description: >
        "global" agents are available across all projects;
        "project" agents are scoped to a single project.
    project_id:
      type: string
      format: uuid
      nullable: true
      description: Required when scope is "project"; null for global agents
    created_at:
      type: string
      format: date-time
    updated_at:
      type: string
      format: date-time
```

### New Schema — `CreateAgentRequest`

```yaml
CreateAgentRequest:
  type: object
  required:
    - name
    - model
    - image
    - template_content
  properties:
    name:
      type: string
    model:
      type: string
    image:
      type: string
    template_content:
      type: string
    scope:
      type: string
      enum: [global, project]
      default: project
```

### New Schema — `UpdateAgentRequest`

```yaml
UpdateAgentRequest:
  type: object
  properties:
    name:
      type: string
    model:
      type: string
    image:
      type: string
    template_content:
      type: string
```

### New Parameter — `AgentIdPath`

```yaml
AgentIdPath:
  name: agentId
  in: path
  required: true
  schema:
    type: string
    format: uuid
```

### New Endpoints

**Project-scoped agents:**

```yaml
/api/v1/projects/{projectId}/agents:
  get:
    operationId: listProjectAgents
    summary: List agents available for a project (project-scoped + global)
    tags: [agents]
    parameters:
      - $ref: "#/components/parameters/ProjectIdPath"
    responses:
      "200":
        description: List of agents
        content:
          application/json:
            schema:
              type: object
              properties:
                data:
                  type: array
                  items:
                    $ref: "#/components/schemas/Agent"
                pagination:
                  $ref: "#/components/schemas/Pagination"
      "401":
        $ref: "#/components/responses/Unauthorized"
  post:
    operationId: createAgent
    summary: Create a new project-scoped agent
    tags: [agents]
    parameters:
      - $ref: "#/components/parameters/ProjectIdPath"
    requestBody:
      required: true
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/CreateAgentRequest"
    responses:
      "201":
        description: Agent created
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Agent"
      "401":
        $ref: "#/components/responses/Unauthorized"
      "422":
        $ref: "#/components/responses/UnprocessableEntity"

/api/v1/projects/{projectId}/agents/{agentId}:
  get:
    operationId: getAgent
    summary: Get a single agent by ID
    tags: [agents]
    parameters:
      - $ref: "#/components/parameters/ProjectIdPath"
      - $ref: "#/components/parameters/AgentIdPath"
    responses:
      "200":
        description: Agent found
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Agent"
      "401":
        $ref: "#/components/responses/Unauthorized"
      "404":
        $ref: "#/components/responses/NotFound"
  put:
    operationId: updateAgent
    summary: Update an agent
    tags: [agents]
    parameters:
      - $ref: "#/components/parameters/ProjectIdPath"
      - $ref: "#/components/parameters/AgentIdPath"
    requestBody:
      required: true
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/UpdateAgentRequest"
    responses:
      "200":
        description: Agent updated
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Agent"
      "401":
        $ref: "#/components/responses/Unauthorized"
      "404":
        $ref: "#/components/responses/NotFound"
  delete:
    operationId: deleteAgent
    summary: Delete an agent
    tags: [agents]
    parameters:
      - $ref: "#/components/parameters/ProjectIdPath"
      - $ref: "#/components/parameters/AgentIdPath"
    responses:
      "204":
        description: Agent deleted
      "401":
        $ref: "#/components/responses/Unauthorized"
      "404":
        $ref: "#/components/responses/NotFound"
```

**Global agents:**

```yaml
/api/v1/agents:
  get:
    operationId: listGlobalAgents
    summary: List all global agents (scope = global)
    tags: [agents]
    responses:
      "200":
        description: List of global agents
        content:
          application/json:
            schema:
              type: object
              properties:
                data:
                  type: array
                  items:
                    $ref: "#/components/schemas/Agent"
                pagination:
                  $ref: "#/components/schemas/Pagination"
      "401":
        $ref: "#/components/responses/Unauthorized"
```

**Deprecation of existing template endpoints:**

Add `deprecated: true` to each operation under `/api/v1/projects/{projectId}/templates` and its sub-paths. Do not remove them.

## Tasks / Subtasks

### 1. OpenAPI Spec — Schemas

- [ ] **1.1** Add `Agent` schema under `components/schemas` (AC: #1)
- [ ] **1.2** Add `CreateAgentRequest` schema under `components/schemas`
- [ ] **1.3** Add `UpdateAgentRequest` schema under `components/schemas`
- [ ] **1.4** Add `AgentIdPath` parameter under `components/parameters`

### 2. OpenAPI Spec — Endpoints

- [ ] **2.1** Add `GET /api/v1/projects/{projectId}/agents` (listProjectAgents) (AC: #2)
- [ ] **2.2** Add `POST /api/v1/projects/{projectId}/agents` (createAgent) (AC: #2)
- [ ] **2.3** Add `GET /api/v1/projects/{projectId}/agents/{agentId}` (getAgent) (AC: #2)
- [ ] **2.4** Add `PUT /api/v1/projects/{projectId}/agents/{agentId}` (updateAgent) (AC: #2)
- [ ] **2.5** Add `DELETE /api/v1/projects/{projectId}/agents/{agentId}` (deleteAgent) (AC: #2)
- [ ] **2.6** Add `GET /api/v1/agents` (listGlobalAgents) (AC: #3)
- [ ] **2.7** Mark all existing `/api/v1/projects/{projectId}/templates` operations as `deprecated: true` (AC: #4)

### 3. Backend — Code Regeneration

- [ ] **3.1** Run `cd backend && make generate` and confirm it exits 0 (AC: #5)
- [ ] **3.2** Add stub `NotImplemented` handler implementations for any new interface methods required by oapi-codegen to keep compilation green (full implementation is in R-1-4)

### 4. Frontend — Code Regeneration

- [ ] **4.1** Run `cd frontend && npm run generate-api` and confirm it exits 0 (AC: #6)

### 5. Lint & Verify

- [ ] **5.1** `cd backend && golangci-lint run ./...`
- [ ] **5.2** `cd frontend && npm run lint && npm run type-check`
- [ ] **5.3** `cd backend && go test ./... -short`

## Dev Notes

### Dependencies

None. This story modifies only `api/openapi.yaml` and runs code generation.

Downstream stories that depend on this:
- **R-1-4** — backend Agent model and DB migration (depends on generated Go types)
- Agent management UI stories — depend on generated TypeScript types

### Architecture Requirements

`api/openapi.yaml` is the single source of truth. Generated files must never be manually edited. New handler stub methods (if required by oapi-codegen) should return `http.StatusNotImplemented` until R-1-4 implements them.

### References

- `api/openapi.yaml` — file to modify
- `backend/Makefile` — `make generate` target
- `frontend/package.json` — `generate-api` script
- Story R-1-4 — backend model and migration (depends on this story)

## Dev Agent Record

## Change Log
