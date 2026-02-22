# Story runtime-2: Action Wiring and action_type Mapping

**Status:** ready-for-dev
**Branch:** `feat/runtime-2-action-wiring-action-type-mapping`
**Commit scope:** `pipeline`

---

## Story

As the pipeline executor, I need all action types defined in the pipeline config (`implement`, `review`, `merge`, `ci_poll`, `hitl_gate`) to resolve correctly from the action registry — so that launching a run on the todo app does not fail with `ACTION_NOT_FOUND` for any step.

Currently:
- `ci_poll` and `hitl_gate` adapters exist with tests but are never registered in `main.go`
- The default pipeline config uses `action_type: implement/review/merge` but the registry only contains `agent_run` and `incremental_retry`
- `pipeline_executor.go` resolves actions via `actionReg.Get(step.Action)` where `step.Action = stepCfg.ActionType` (set in `run_service.go` line 336)

---

## Acceptance Criteria

**AC #1 — `ci_poll` is registered at startup**
- Given the backend starts with a valid Docker connection
- When `actionReg.Get("ci_poll")` is called
- Then it returns the `CIPollAction` instance (not an error)

**AC #2 — `hitl_gate` is registered at startup**
- Given the backend starts
- When `actionReg.Get("hitl_gate")` is called
- Then it returns the `HITLGateAction` instance (not an error)

**AC #3 — `implement` resolves to `AgentRunAction`**
- Given the action registry is populated
- When the pipeline executor processes a step with `action = "implement"`
- Then `actionReg.Get("implement")` returns the `AgentRunAction` (same instance as `agent_run`)

**AC #4 — `review` resolves to `AgentRunAction`**
- Given the action registry is populated
- When the pipeline executor processes a step with `action = "review"`
- Then `actionReg.Get("review")` returns the `AgentRunAction`

**AC #5 — `merge` resolves to `AgentRunAction`**
- Given the action registry is populated
- When the pipeline executor processes a step with `action = "merge"`
- Then `actionReg.Get("merge")` returns the `AgentRunAction`

**AC #6 — Template name is derived from action_type**
- Given an `AgentRunAction` executing a step with `action = "review"`
- When `resolveTemplateName` is called on the run context
- Then it returns `service.TemplateNameReview` (not `service.TemplateNameImplement`)

**AC #7 — `agent_run` still resolves (no regression)**
- Given the registry after all changes
- When `actionReg.Get("agent_run")` is called
- Then it still returns the `AgentRunAction`

**AC #8 — No panics or nil pointer dereferences at startup**
- Given all actions are registered
- When the backend starts and the action registry is fully wired
- Then `golangci-lint run ./...` passes and `go test ./... -short` passes

**AC #9 — `validatePipelineConfigYAML` does not reject `ci_poll` or `hitl_gate`**
- Given `action_type: ci_poll` or `action_type: hitl_gate` in a pipeline config
- When `validatePipelineConfigYAML` is called
- Then validation passes (these are now valid action types)

---

## Tasks / Subtasks

- [ ] **T1.** Register `ci_poll` in `backend/cmd/api/main.go` (AC: #1)
  - [ ] T1.1 After the `hitlRepo` line (~line 255 in main.go), instantiate `CIPollConfig` with defaults (`DefaultPollInterval: 30*time.Second`, `DefaultTimeout: 15*time.Minute`)
  - [ ] T1.2 Instantiate `NewCIPollAction(gitProvider, eventRepo, ciPollCfg, logger)` — need to wire a `GitProvider` first (see T1.3)
  - [ ] T1.3 Instantiate the GitHub adapter `githubadapter.NewGitProvider(runner, logger)` in main.go (currently missing — needed for `CIPollAction`)
  - [ ] T1.4 Register: `actionReg.Register(ciPollAction)` with log line `logger.Info("ci_poll action registered")`

- [ ] **T2.** Register `hitl_gate` in `backend/cmd/api/main.go` (AC: #2)
  - [ ] T2.1 Instantiate `NewHITLGateAction(hitlRepo, runRepo, gitProvider, eventRepo, storyRepo, logger)`
  - [ ] T2.2 Register: `actionReg.Register(hitlGateAction)` with log line `logger.Info("hitl_gate action registered")`
  - [ ] T2.3 Place registration OUTSIDE the `if containerMgr != nil` block — `hitl_gate` does not require Docker

- [ ] **T3.** Add `RegisterAlias` to `InMemoryActionRegistry` (AC: #3, #4, #5, #7)
  - [ ] T3.1 Add method `RegisterAlias(alias string, action model.Action)` to `InMemoryActionRegistry` in `backend/internal/domain/service/action_registry.go`
  - [ ] T3.2 Method stores the action under the alias key (same `actions` map)
  - [ ] T3.3 Update `port.ActionRegistry` interface in `backend/internal/domain/port/` to include `RegisterAlias` if it exists, OR just use `Register` with a wrapper action (see T3-alt below)

  **Alternative T3-alt (simpler, no interface change):** In `main.go`, after registering `agentRunAction`, call `actionReg.Register` with a thin wrapper type for each alias:
  ```go
  // Register action_type aliases for the default pipeline config
  for _, alias := range []string{"implement", "review", "merge"} {
      actionReg.RegisterAlias(alias, agentRunAction)
  }
  ```
  Prefer the `RegisterAlias` approach on `InMemoryActionRegistry` directly (no interface change needed since `InMemoryActionRegistry` is used concretely in main.go).

- [ ] **T4.** Propagate `action_type` as `template_name` metadata in `run_service.go` (AC: #6)
  - [ ] T4.1 In `RunService.LaunchRun()`, when creating `RunStep`, set metadata field to carry the template name derived from `action_type`
  - [ ] T4.2 Map: `implement` → `service.TemplateNameImplement`, `review` → `service.TemplateNameReview`, `merge` → `service.TemplateNameMerge`
  - [ ] T4.3 The `RunStep` model may not have a `Metadata` field — if so, store the mapping as `step.Action` stays as-is but inject a `template_name` key into the `metadata` map passed to `RunContext` in `pipeline_executor.go`

  **Investigation note:** Check whether `RunStep.Metadata` or `RunContext.Metadata` is the right place. Looking at `agent_run.go` line 110: `templateName` is read from `runCtx.Metadata["template_name"]`. The `RunContext.Metadata` is a `map[string]any` initialized per-run in `pipeline_executor.go` line 106. The cleanest fix: in `pipeline_executor.go`'s `executeStep()`, before calling `action.Execute()`, set `runCtx.Metadata["template_name"]` based on `step.Action` for the known aliases.

- [ ] **T5.** Update `model.ValidActionTypes` to include `ci_poll` and `hitl_gate` (AC: #9)
  - [ ] T5.1 In `backend/internal/domain/model/pipeline_config.go`, add `"ci_poll": true` and `"hitl_gate": true` to `ValidActionTypes`

- [ ] **T6.** Lint and test (AC: #8)
  - [ ] T6.1 Run `cd backend && golangci-lint run ./...` — must pass
  - [ ] T6.2 Run `cd backend && go test ./... -short` — must pass
  - [ ] T6.3 Run `cd backend && go test ./internal/integration/ -v -run TestIntegration_PipelineValidation` — must pass

---

## Dev Notes

### Dependencies

- `GitProvider` adapter (`github.com/zakari/hopeitworks/backend/internal/adapter/github`) must be instantiated in `main.go` — it is currently NOT wired (only `AgentRunAction` uses it implicitly via `AgentConfig`, but `CIPollAction` needs it directly)
- `CommandRunner` from `pkg/exec` is needed by the GitHub adapter

### File Paths

| File | Change |
|------|--------|
| `backend/cmd/api/main.go` | Register `ci_poll`, `hitl_gate`; add GitProvider wiring; register aliases `implement`, `review`, `merge` |
| `backend/internal/domain/service/action_registry.go` | Add `RegisterAlias` method |
| `backend/internal/domain/service/pipeline_executor.go` | Inject `template_name` into metadata based on `step.Action` aliases |
| `backend/internal/domain/model/pipeline_config.go` | Add `ci_poll`, `hitl_gate` to `ValidActionTypes` |
| `backend/internal/domain/port/action.go` (if exists) | No change needed if `RegisterAlias` is not on the interface |

### Technical Specifications

**Wiring `CIPollAction` in `main.go`:**

```go
// GitHub provider (needed by ci_poll and hitl_gate)
cmdRunner := exec.NewCommandRunner()
githubProvider := githubadapter.NewGitProvider(cmdRunner, logger)

// CI Poll action
ciPollCfg := actionadapter.CIPollConfig{
    DefaultPollInterval: 30 * time.Second,
    DefaultTimeout:      15 * time.Minute,
}
ciPollAction := actionadapter.NewCIPollAction(githubProvider, eventRepo, ciPollCfg, logger)
actionReg.Register(ciPollAction)
logger.Info("ci_poll action registered")

// HITL Gate action (outside containerMgr block — no Docker needed)
hitlGateAction := actionadapter.NewHITLGateAction(hitlRepo, runRepo, githubProvider, eventRepo, storyRepo, logger)
actionReg.Register(hitlGateAction)
logger.Info("hitl_gate action registered")
```

**`RegisterAlias` in `action_registry.go`:**

```go
// RegisterAlias registers an action under an alias name.
// The action is stored under the alias key; its own Name() is not affected.
func (r *InMemoryActionRegistry) RegisterAlias(alias string, action model.Action) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.actions[alias] = action
}
```

**Template name injection in `pipeline_executor.go` `executeStep()`:**

```go
// actionTypeToTemplateName maps pipeline config action_type aliases to prompt template names.
var actionTypeToTemplateName = map[string]string{
    "implement": service.TemplateNameImplement,
    "review":    service.TemplateNameReview,
    "merge":     service.TemplateNameMerge,
}

// In executeStep(), before action.Execute():
if tmplName, ok := actionTypeToTemplateName[step.Action]; ok {
    if _, exists := runCtx.Metadata["template_name"]; !exists {
        runCtx.Metadata["template_name"] = tmplName
    }
}
```

This ensures `implement`, `review`, `merge` steps pick the correct prompt template, while an explicit `template_name` in metadata (from incremental retry) still takes precedence.

**Import paths to add in `main.go`:**

```go
githubadapter "github.com/zakari/hopeitworks/backend/internal/adapter/github"
"github.com/zakari/hopeitworks/backend/pkg/exec"
```

**Verify `TemplateNameReview` and `TemplateNameMerge` exist:**

Check `backend/internal/domain/service/template_service.go` for the `TemplateNameXxx` constants. If `TemplateNameReview` or `TemplateNameMerge` are missing, add them with values `"review"` and `"merge"`.

**Check `CIPollAction` signature — `eventPub` vs `eventRepo`:**

`NewCIPollAction` takes `port.EventPublisher`. In `main.go`, `eventRepo` is `pgadapter.NewEventRepo(queries)` which implements `port.EventPublisher`. Pass `eventRepo` directly.

### Testing Requirements

- `golangci-lint run ./...` must pass (errcheck, revive, staticcheck)
- `go test ./... -short` must pass
- `go test ./internal/integration/ -run TestIntegration_PipelineValidation -v` must pass — these tests use noop actions via the registry
- Manual smoke: start the backend and check logs for all action registration lines

### References

- `backend/internal/adapter/action/ci_poll.go` — `NewCIPollAction` signature
- `backend/internal/adapter/action/hitl_gate.go` — `NewHITLGateAction` signature
- `backend/internal/adapter/action/agent_run.go` — `resolveTemplateName` reads `runCtx.Metadata["template_name"]`
- `backend/internal/domain/service/action_registry.go` — current registry implementation
- `backend/internal/domain/service/pipeline_executor.go` — where `actionReg.Get(step.Action)` is called (line 191)
- `backend/internal/domain/service/run_service.go` — where `step.Action = stepCfg.ActionType` is set (line 336)
- `backend/internal/domain/model/pipeline_config.go` — `ValidActionTypes` map
- `backend/internal/domain/service/pipeline_config_service.go` — `DefaultPipelineConfigYAML` uses `implement`, `review`, `merge`
- `backend/internal/adapter/github/` — GitProvider adapter
