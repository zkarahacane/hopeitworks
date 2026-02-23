# Story R-1-1: [SHARED] Add PipelineGroup + new action_types to OpenAPI spec

Status: ready-for-dev

## Story

As a **platform developer**,
I want the OpenAPI spec to model `PipelineGroup` and the full set of supported action types,
so that both the backend and frontend generate correct, strongly-typed code that reflects the real pipeline configuration structure.

## Acceptance Criteria (BDD)

### Scenario 1: PipelineGroup schema exists in the spec

```gherkin
Given the file api/openapi.yaml is loaded
When I inspect the components/schemas section
Then a "PipelineGroup" schema is present with:
  | field | type            | required |
  | id    | string          | yes      |
  | name  | string          | yes      |
  | steps | array(PipelineStep) | yes  |
```

### Scenario 2: PipelineConfig uses groups instead of flat steps

```gherkin
Given the "PipelineConfig" schema in api/openapi.yaml
When I inspect its properties
Then it has a "groups" field of type array(PipelineGroup)
  And it no longer has a top-level "steps" field
```

### Scenario 3: New action_type enum values are present

```gherkin
Given the "action_type" enum in api/openapi.yaml
When I inspect the allowed values
Then the enum includes at minimum:
  | agent_run   |
  | git_branch  |
  | git_pr      |
  | notification|
  | human       |
  | ci_poll     |
  | hitl_gate   |
```

### Scenario 4: PipelineStep has an optional config field

```gherkin
Given the "PipelineStep" schema in api/openapi.yaml
When I inspect its properties
Then a "config" field is present that is:
  | type                | object              |
  | additionalProperties| string              |
  | required            | no (optional)       |
```

### Scenario 5: Backend code generation succeeds

```gherkin
Given the updated api/openapi.yaml
When I run "cd backend && make generate"
Then the command exits with code 0
  And the generated Go types include PipelineGroup and the updated PipelineConfig
  And the generated action_type constants include the new values
```

### Scenario 6: Frontend code generation succeeds

```gherkin
Given the updated api/openapi.yaml
When I run "cd frontend && npm run generate-api"
Then the command exits with code 0
  And the generated TypeScript types include PipelineGroup and the updated PipelineConfig
  And the action_type union type includes the new string literals
```

## Technical Notes

### OpenAPI Schema Changes

**New schema — `PipelineGroup`:**

```yaml
PipelineGroup:
  type: object
  required:
    - id
    - name
    - steps
  properties:
    id:
      type: string
      description: Unique identifier for this group within the pipeline config
    name:
      type: string
      description: Human-readable name for the group (e.g. "Setup", "Development")
    steps:
      type: array
      items:
        $ref: "#/components/schemas/PipelineStep"
```

**Modified schema — `PipelineConfig`:**

Replace the flat `steps` field with `groups`:

```yaml
# Before
steps:
  type: array
  items:
    $ref: "#/components/schemas/PipelineStep"

# After
groups:
  type: array
  items:
    $ref: "#/components/schemas/PipelineGroup"
  description: Ordered list of step groups. Groups are executed sequentially; steps within a group are executed sequentially.
```

**Modified schema — `PipelineStep` (add `config`):**

```yaml
config:
  type: object
  additionalProperties:
    type: string
  description: Optional per-action-type configuration key-value pairs
```

**Modified enum — `action_type`:**

```yaml
action_type:
  type: string
  enum:
    - agent_run
    - git_branch
    - git_pr
    - notification
    - human
    - ci_poll
    - hitl_gate
```

### Impact on Existing Code

- Any backend handler or service that reads `PipelineConfig.Steps` will fail to compile after regen — those callers are updated in story **R-1-3**.
- Any frontend component that reads `pipelineConfig.steps` will get a TypeScript error — those components are updated in the pipeline config editor story.
- This story is intentionally a **breaking spec change**; downstream stories fix the callers.

### Backward Compatibility Note

The OpenAPI spec does not carry backward compatibility for internal data models at MVP. Migration of stored YAML is handled in story **R-1-3** (backward-compatible YAML parsing with auto-wrap).

## Tasks / Subtasks

### 1. OpenAPI Spec — Schema Changes

- [ ] **1.1** Add `PipelineGroup` schema under `components/schemas` in `api/openapi.yaml` (AC: #1)
- [ ] **1.2** Replace `PipelineConfig.steps` field with `PipelineConfig.groups: array(PipelineGroup)` (AC: #2)
- [ ] **1.3** Add new values to `action_type` enum: `git_branch`, `git_pr`, `notification`, `human`, `ci_poll`, `hitl_gate` (AC: #3)
- [ ] **1.4** Add optional `config: object (additionalProperties: string)` field to `PipelineStep` schema (AC: #4)

### 2. Backend — Code Regeneration

- [ ] **2.1** Run `cd backend && make generate` and confirm it exits 0 (AC: #5)
- [ ] **2.2** Fix any compilation errors introduced by the schema change in existing backend code that references the old `Steps` field (note: deep fixes are in R-1-3; this task only fixes compilation blockers to keep `make generate` green)

### 3. Frontend — Code Regeneration

- [ ] **3.1** Run `cd frontend && npm run generate-api` and confirm it exits 0 (AC: #6)
- [ ] **3.2** Fix any TypeScript type errors in frontend code that referenced the old `pipelineConfig.steps` field (note: deep fixes are in a follow-up story; this task only fixes type-check blockers)

### 4. Lint & Verify

- [ ] **4.1** `cd backend && golangci-lint run ./...`
- [ ] **4.2** `cd frontend && npm run lint && npm run type-check`
- [ ] **4.3** `cd backend && go test ./... -short`

## Dev Notes

### Dependencies

None. This story modifies only `api/openapi.yaml` and runs code generation. No migration, no new DB schema.

Downstream stories that depend on this:
- **R-1-3** — backend model update (depends on generated Go types from this story)
- Pipeline config editor UI stories — depend on generated TypeScript types

### Architecture Requirements

`api/openapi.yaml` is the single source of truth. All type changes flow from spec → generated code → implementation. Never manually edit generated files (`backend/internal/api/generated/`, `frontend/src/api/generated/`).

### References

- `api/openapi.yaml` — file to modify
- `backend/Makefile` — `make generate` target (runs oapi-codegen)
- `frontend/package.json` — `generate-api` script (runs openapi-typescript + openapi-fetch)
- Story R-1-3 — backend model update (depends on this story)

## Dev Agent Record

## Change Log
