# Story R-6-1: [BACK] Update default pipeline config to use groups

Status: ready-for-dev

## Story

As a **platform developer**,
I want the default pipeline configuration YAML to be organized into named groups,
so that newly created projects receive a structured, multi-group default config that reflects the real agent pipeline workflow.

## Acceptance Criteria (BDD)

### Scenario 1: Default pipeline config YAML uses groups structure

```gherkin
Given the DefaultPipelineConfigYAML constant in pipeline_config_service.go
When I parse the YAML into a PipelineConfig struct
Then the parsed config has a "groups" field (not a flat "steps" field)
  And there are 5 groups: Setup, Development, Review, Merge, Delivery
```

### Scenario 2: Setup group contains git_branch step

```gherkin
Given the parsed DefaultPipelineConfigYAML
When I inspect the "Setup" group
Then it contains exactly one step of action_type "git_branch"
```

### Scenario 3: Development group contains agent_run (implement) step

```gherkin
Given the parsed DefaultPipelineConfigYAML
When I inspect the "Development" group
Then it contains exactly one step of action_type "agent_run"
  And the step has a config key indicating it is the implementation step (e.g. name: "implement" or role: "dev")
```

### Scenario 4: Review group contains agent_run (review) step

```gherkin
Given the parsed DefaultPipelineConfigYAML
When I inspect the "Review" group
Then it contains exactly one step of action_type "agent_run"
  And the step config indicates it is the review step (e.g. name: "review" or role: "review")
```

### Scenario 5: Merge group contains git_pr step

```gherkin
Given the parsed DefaultPipelineConfigYAML
When I inspect the "Merge" group
Then it contains exactly one step of action_type "git_pr"
```

### Scenario 6: Delivery group contains ci_poll and notification steps

```gherkin
Given the parsed DefaultPipelineConfigYAML
When I inspect the "Delivery" group
Then it contains two steps
  And the first step has action_type "ci_poll"
  And the second step has action_type "notification"
```

### Scenario 7: Backward compat — flat steps YAML is still parsed correctly

```gherkin
Given an old-format PipelineConfig YAML with a flat "steps" field (no groups)
When the backend parses this YAML
Then the steps are wrapped into a single default group without error
  And no data is lost
```

### Scenario 8: Existing tests pass

```gherkin
Given all code changes
When I run "cd backend && go test ./... -short"
Then all tests pass
  And any tests asserting on DefaultPipelineConfigYAML are updated to match the new structure
```

## Tasks / Subtasks

- [ ] **1.1** [BACK] Update `DefaultPipelineConfigYAML` constant in `backend/internal/domain/service/pipeline_config_service.go` (AC: #1–#6)
  - [ ] Replace flat `steps:` structure with `groups:` structure
  - [ ] Define 5 groups: Setup (git_branch), Development (agent_run implement), Review (agent_run review), Merge (git_pr), Delivery (ci_poll + notification)
  - [ ] Ensure each group has an `id` (kebab-case slug) and `name` (human-readable)
  - [ ] Ensure each step has `id`, `name`, `action_type`, and relevant `config` keys

- [ ] **1.2** [BACK] Implement backward-compat YAML parsing for flat steps (AC: #7)
  - [ ] In the YAML unmarshal logic (or a custom UnmarshalYAML), detect if the top-level has `steps` instead of `groups`
  - [ ] If flat steps detected, wrap them in a single group named "Default" with id "default"
  - [ ] This ensures stored configs from before R-1-1 remain readable

- [ ] **1.3** [BACK] Update seed data if `DefaultPipelineConfigYAML` is used in migration seeds (AC: #1)
  - [ ] Check if `000026_seed_merge_template.up.sql` or any other migration embeds the YAML
  - [ ] If so, update the seeded YAML to match the new groups structure

- [ ] **1.4** [BACK] Update any tests that assert on `DefaultPipelineConfigYAML` structure (AC: #8)
  - [ ] Search for test files that reference `DefaultPipelineConfigYAML` or assert on `Steps` field
  - [ ] Update assertions to expect `Groups` with the 5 named groups

- [ ] **1.5** [BACK] Lint and test (AC: #8)
  - [ ] `cd backend && golangci-lint run ./...`
  - [ ] `cd backend && go test ./... -short`

## Dev Notes

### Dependencies

- **R-1-3** — the `PipelineGroup` Go model must exist (generated from R-1-1 spec and implemented in R-1-3) so the YAML can be parsed into it. This story depends on that model being available.

### Architecture Requirements

- `DefaultPipelineConfigYAML` is a Go string constant — update it in place, do not generate it dynamically
- Backward compat parsing must not break existing stored configs — wrap flat steps in a "Default" group rather than failing
- Do not modify the database migration files that have already been applied (migrations 000001–000026 are immutable); only update the constant and any not-yet-applied seed logic

### Technical Specifications

**Updated DefaultPipelineConfigYAML:**

```yaml
groups:
  - id: setup
    name: Setup
    steps:
      - id: git-branch
        name: Create Branch
        action_type: git_branch
        config:
          base_branch: main

  - id: development
    name: Development
    steps:
      - id: agent-implement
        name: Implement Story
        action_type: agent_run
        config:
          role: dev
          phase: dev-story

  - id: review
    name: Review
    steps:
      - id: agent-review
        name: Code Review
        action_type: agent_run
        config:
          role: review
          phase: code-review

  - id: merge
    name: Merge
    steps:
      - id: git-pr
        name: Create & Merge PR
        action_type: git_pr
        config:
          strategy: squash

  - id: delivery
    name: Delivery
    steps:
      - id: ci-poll
        name: Wait for CI
        action_type: ci_poll
        config:
          timeout_minutes: "30"
      - id: notify
        name: Notify Completion
        action_type: notification
        config:
          channel: default
```

**Backward compat parsing (pseudo-code):**

```go
type rawPipelineConfig struct {
    Groups []PipelineGroup `yaml:"groups"`
    Steps  []PipelineStep  `yaml:"steps"` // legacy flat format
}

func parsePipelineConfig(yamlStr string) (*PipelineConfig, error) {
    var raw rawPipelineConfig
    if err := yaml.Unmarshal([]byte(yamlStr), &raw); err != nil {
        return nil, err
    }
    if len(raw.Groups) == 0 && len(raw.Steps) > 0 {
        // Wrap legacy flat steps in a single default group
        raw.Groups = []PipelineGroup{{
            ID:    "default",
            Name:  "Default",
            Steps: raw.Steps,
        }}
    }
    return &PipelineConfig{Groups: raw.Groups}, nil
}
```

### Testing Requirements

- Add a test `TestParsePipelineConfig_BackwardCompat` that parses an old flat-steps YAML and verifies it produces one group named "Default"
- Add a test `TestDefaultPipelineConfigYAML_Groups` that parses `DefaultPipelineConfigYAML` and verifies 5 groups with correct action types
- Update any existing test that asserts `len(config.Steps) == N` to assert on groups instead

### References

- `backend/internal/domain/service/pipeline_config_service.go` — DefaultPipelineConfigYAML constant and parsing logic
- `backend/internal/domain/model/` — PipelineGroup, PipelineStep, PipelineConfig models (from R-1-3)
- `backend/migrations/000026_seed_merge_template.up.sql` — check if it embeds pipeline YAML

## Dev Agent Record

## Change Log
