# Story R-1-3: [BACK] PipelineGroup in backend model + YAML parsing + validation

Status: ready-for-dev

## Story

As a **platform developer**,
I want the backend domain model, YAML parser, and validation logic to support `PipelineGroup`,
so that pipeline configurations can organize steps into named groups with full backward compatibility for existing flat-steps YAML files.

## Acceptance Criteria (BDD)

### Scenario 1: PipelineGroup struct exists in the domain model

```gherkin
Given the backend compiles successfully
When I inspect backend/internal/domain/model/pipeline_config.go
Then a PipelineGroup struct exists with fields:
  | field | type          |
  | ID    | string        |
  | Name  | string        |
  | Steps | []PipelineStep|
  And PipelineConfig has a "Groups []PipelineGroup" field instead of a flat "Steps" field
```

### Scenario 2: PipelineStep has a Config field

```gherkin
Given the backend compiles successfully
When I inspect the PipelineStep struct
Then it has a "Config map[string]string" field
```

### Scenario 3: New action types are accepted

```gherkin
Given a PipelineConfig YAML with action_type "git_branch"
When the YAML is parsed and validated
Then no validation error is returned for the action_type
  And the same is true for: git_pr, notification, human, ci_poll, hitl_gate
```

### Scenario 4: Backward compatibility — flat steps YAML is auto-wrapped

```gherkin
Given a PipelineConfig YAML that has a top-level "steps:" array (old format)
When the YAML is parsed
Then the steps are automatically wrapped in a single PipelineGroup named "Default"
  And the parsed PipelineConfig has one group containing all the original steps
  And no error is returned
```

### Scenario 5: New default pipeline config uses groups

```gherkin
Given a new project is created
When the default pipeline config YAML is generated
Then it has a "groups:" top-level key (not "steps:")
  And it contains at least the following groups in order:
    | group name  | action types in group           |
    | Setup       | git_branch                      |
    | Development | agent_run                       |
    | Review      | agent_run                       |
    | Merge       | git_pr                          |
    | Delivery    | ci_poll, notification           |
```

### Scenario 6: Validation rejects unknown action types

```gherkin
Given a PipelineConfig YAML with action_type "unknown_action"
When the YAML is validated
Then a validation error is returned with code "INVALID_ACTION_TYPE"
```

### Scenario 7: Validation rejects empty group name

```gherkin
Given a PipelineConfig YAML with a group that has an empty "name" field
When the YAML is validated
Then a validation error is returned indicating the group name is required
```

### Scenario 8: Validation rejects groups with no steps

```gherkin
Given a PipelineConfig YAML with a group that has an empty "steps" array
When the YAML is validated
Then a validation error is returned indicating each group must have at least one step
```

## Technical Notes

### Struct Changes in `pipeline_config.go`

```go
// New struct
type PipelineGroup struct {
    ID    string         `yaml:"id"    json:"id"`
    Name  string         `yaml:"name"  json:"name"`
    Steps []PipelineStep `yaml:"steps" json:"steps"`
}

// Updated PipelineConfig
type PipelineConfig struct {
    Version  string          `yaml:"version"  json:"version"`
    Groups   []PipelineGroup `yaml:"groups"   json:"groups"`
    // Remove: Steps []PipelineStep
}

// Updated PipelineStep
type PipelineStep struct {
    Name        string            `yaml:"name"         json:"name"`
    Action      ActionType        `yaml:"action"       json:"action"`
    Description string            `yaml:"description"  json:"description"`
    RetryPolicy RetryPolicy       `yaml:"retry_policy" json:"retry_policy"`
    Config      map[string]string `yaml:"config"       json:"config"`
    // ... existing fields unchanged
}
```

### Backward-Compatible YAML Parsing

Introduce an intermediate `PipelineConfigYAML` struct for unmarshalling that handles both old and new formats:

```go
type pipelineConfigYAML struct {
    Version string          `yaml:"version"`
    Groups  []PipelineGroup `yaml:"groups"`
    Steps   []PipelineStep  `yaml:"steps"` // legacy flat format
}

func ParsePipelineConfigYAML(data []byte) (*PipelineConfig, error) {
    var raw pipelineConfigYAML
    if err := yaml.Unmarshal(data, &raw); err != nil {
        return nil, err
    }

    cfg := &PipelineConfig{Version: raw.Version}

    if len(raw.Groups) > 0 {
        cfg.Groups = raw.Groups
    } else if len(raw.Steps) > 0 {
        // Legacy: wrap flat steps in a single default group
        cfg.Groups = []PipelineGroup{
            {ID: "default", Name: "Default", Steps: raw.Steps},
        }
    }

    return cfg, nil
}
```

### New Valid Action Types

```go
var ValidActionTypes = map[ActionType]struct{}{
    "agent_run":    {},
    "git_branch":   {},
    "git_pr":       {},
    "notification": {},
    "human":        {},
    "ci_poll":      {},
    "hitl_gate":    {},
}
```

### Updated `DefaultPipelineConfigYAML`

```yaml
version: "1"
groups:
  - id: setup
    name: Setup
    steps:
      - name: create-branch
        action: git_branch
        description: Create feature branch from base
        config:
          base_branch: main

  - id: development
    name: Development
    steps:
      - name: dev-agent
        action: agent_run
        description: Run development agent
        retry_policy:
          max_retries: 2

  - id: review
    name: Review
    steps:
      - name: review-agent
        action: agent_run
        description: Run code review agent
        retry_policy:
          max_retries: 1

  - id: merge
    name: Merge
    steps:
      - name: create-pr
        action: git_pr
        description: Create and merge pull request

  - id: delivery
    name: Delivery
    steps:
      - name: poll-ci
        action: ci_poll
        description: Wait for CI to pass
        config:
          timeout_minutes: "30"
      - name: notify
        action: notification
        description: Send completion notification
```

### Validation Updates in `pipeline_config_service.go`

Replace validation that iterates over `cfg.Steps` with validation that iterates over `cfg.Groups` and each group's `Steps`:

```go
func (s *PipelineConfigService) validate(cfg *model.PipelineConfig) error {
    if len(cfg.Groups) == 0 {
        return errors.NewValidation("groups", "pipeline config must have at least one group")
    }
    for i, group := range cfg.Groups {
        if group.Name == "" {
            return errors.NewValidation(fmt.Sprintf("groups[%d].name", i), "group name is required")
        }
        if len(group.Steps) == 0 {
            return errors.NewValidation(fmt.Sprintf("groups[%d].steps", i), "group must have at least one step")
        }
        for j, step := range group.Steps {
            if _, ok := model.ValidActionTypes[step.Action]; !ok {
                return errors.NewValidation(
                    fmt.Sprintf("groups[%d].steps[%d].action", i, j),
                    fmt.Sprintf("unknown action type: %s", step.Action),
                )
            }
        }
    }
    return nil
}
```

### Pipeline Executor Adaptation

The `PipelineExecutor` currently iterates over a flat `config.Steps` slice. After this change it must iterate over `config.Groups` and each group's `Steps`. This is a breaking internal change. Update `backend/internal/adapter/pipeline/executor.go` (or equivalent) to flatten groups into an ordered step list at execution time:

```go
func flattenSteps(cfg *model.PipelineConfig) []model.PipelineStep {
    var steps []model.PipelineStep
    for _, g := range cfg.Groups {
        steps = append(steps, g.Steps...)
    }
    return steps
}
```

Use `flattenSteps(cfg)` wherever the executor previously accessed `cfg.Steps` directly.

## Tasks / Subtasks

### 1. Domain Model

- [ ] **1.1** Add `PipelineGroup` struct to `backend/internal/domain/model/pipeline_config.go` (AC: #1)
- [ ] **1.2** Replace `PipelineConfig.Steps []PipelineStep` with `PipelineConfig.Groups []PipelineGroup` (AC: #1)
- [ ] **1.3** Add `Config map[string]string` field to `PipelineStep` struct (AC: #2)
- [ ] **1.4** Add new entries to `ValidActionTypes`: `git_branch`, `git_pr`, `notification`, `human`, `ci_poll`, `hitl_gate` (AC: #3)

### 2. YAML Parsing

- [ ] **2.1** Introduce intermediate `pipelineConfigYAML` struct with both `Groups` and legacy `Steps` fields (AC: #4)
- [ ] **2.2** Implement `ParsePipelineConfigYAML()` with auto-wrap logic: if `steps:` is present and `groups:` is absent, wrap in a single "Default" group (AC: #4)
- [ ] **2.3** Update all callers of the old YAML unmarshal path to use `ParsePipelineConfigYAML()` (AC: #4)

### 3. Default Config

- [ ] **3.1** Update `DefaultPipelineConfigYAML` constant to use the new groups-based structure with Setup / Development / Review / Merge / Delivery groups (AC: #5)

### 4. Validation

- [ ] **4.1** Update `pipeline_config_service.go` validation to iterate over groups and their steps (AC: #6, #7, #8)
- [ ] **4.2** Add validation: each group must have a non-empty `name` (AC: #7)
- [ ] **4.3** Add validation: each group must have at least one step (AC: #8)
- [ ] **4.4** Validation error code for invalid action type: `INVALID_ACTION_TYPE` (AC: #6)

### 5. Pipeline Executor Adaptation

- [ ] **5.1** Add `flattenSteps(cfg *model.PipelineConfig) []model.PipelineStep` helper (or inline) in the executor (AC: #1)
- [ ] **5.2** Replace all direct accesses to `cfg.Steps` in `executor.go` with the flattened step list (AC: #1)

### 6. Tests

- [ ] **6.1** Unit tests for `ParsePipelineConfigYAML()`: new groups format parsed correctly, legacy flat steps auto-wrapped in "Default" group, empty input returns error
- [ ] **6.2** Unit tests for validation: empty groups, empty group name, empty group steps, unknown action type, valid new action types
- [ ] **6.3** Unit test for `flattenSteps()`: correct flattening order across multiple groups
- [ ] **6.4** Update any existing tests that reference `PipelineConfig.Steps` directly to use `PipelineConfig.Groups`

### 7. Lint & Verify

- [ ] **7.1** `cd backend && golangci-lint run ./...`
- [ ] **7.2** `cd backend && go test ./... -short`

## Dev Notes

### Dependencies

**Depends on:** R-1-1 (OpenAPI spec must be updated and code regenerated first so the generated Go types are consistent with the domain model changes here)

### Architecture Requirements

- Domain model changes live in `backend/internal/domain/model/pipeline_config.go`
- Service validation lives in `backend/internal/domain/service/pipeline_config_service.go`
- The executor adaptation follows hexagonal boundaries: the executor is an adapter; the domain model is the source of truth
- No direct DB schema changes in this story — pipeline configs are stored as YAML text; the struct change only affects in-memory parsing

### References

- `backend/internal/domain/model/pipeline_config.go` — primary file to modify
- `backend/internal/domain/service/pipeline_config_service.go` — validation to update
- `backend/internal/adapter/pipeline/executor.go` (or equivalent executor file) — step iteration to update
- Story R-1-1 — OpenAPI spec changes (prerequisite)

## Dev Agent Record

## Change Log
