# Story runtime-4: RunContext Metadata Propagation

**Status:** ready-for-dev
**Branch:** `feat/runtime-4-run-context-metadata`
**Commit scope:** `pipeline`

---

## Story

As the pipeline executor, I need `RunContext.Metadata` to be populated with `branch_name`, `template_name`, and `model` before any action executes — so that `AgentRunAction` can launch containers with a valid branch, the correct prompt template, and the per-step model configured in the pipeline YAML.

Currently three critical metadata keys are either never set or not propagated:

1. **`branch_name`** is read by `agent_run.go` at `runCtx.Metadata["branch_name"]` but is never written anywhere. The agent container validates `BRANCH_NAME` at startup and crashes if it is empty.
2. **`template_name`** is injected by `pipeline_executor.go` (lines 214–218) using `actionTypeToTemplateName` — this part is already correctly implemented and must not be changed.
3. **`model`** is present as `PipelineStep.Model` in the YAML config but is never stored on the `Run`, the `RunStep`, or the `RunContext`. `AgentRunAction.createContainer` has no per-step model env var and always falls back to the project-level default.

---

## Acceptance Criteria

**AC #1 — branch_name is generated and available to actions**
- Given a story with key `"runtime-4"` and a run is launched via `LaunchRun`
- When the implement step's `AgentRunAction.Execute` is called
- Then `runCtx.Metadata["branch_name"]` equals `"feat/runtime-4"`
- And the container receives `BRANCH_NAME=feat/runtime-4`

**AC #2 — template_name is set per step based on action_type (verify, no regression)**
- Given a pipeline with steps implement / review / merge
- When each step executes
- Then the implement step has `runCtx.Metadata["template_name"] == "implement"`
- And the review step has `runCtx.Metadata["template_name"] == "review"`
- And the merge step has `runCtx.Metadata["template_name"] == "merge"`
- (This is already implemented in `pipeline_executor.go` — add a test to cover it, do not change the logic)

**AC #3 — model is propagated from pipeline config to the container**
- Given a pipeline config with `model: "claude-opus-4-6"` on the implement step and `model: "claude-sonnet-4-6"` on the review step
- When each step executes
- Then `runCtx.Metadata["model"]` equals `"claude-opus-4-6"` for the implement step
- And `runCtx.Metadata["model"]` equals `"claude-sonnet-4-6"` for the review step
- And if `PipelineStep.Model` is empty, `runCtx.Metadata["model"]` is not set (the container falls back to its own default)

**AC #4 — all tests pass**
- Given the changes are applied
- When `go test ./... -short` is run from `backend/`
- Then all tests pass with zero failures
- And `golangci-lint run ./...` reports zero errors

---

## Tasks / Subtasks

- [ ] **T1.** Generate `branch_name` in `LaunchRun` and store it on `Run.Metadata` (AC: #1)
  - [ ] T1.1 In `run_service.go` `LaunchRun`, after fetching `story`, compute `branchName := "feat/" + story.Key`
  - [ ] T1.2 Populate `run.Metadata` with `map[string]interface{}{"branch_name": branchName}` before calling `runRepo.CreateRun`
  - [ ] T1.3 Confirm `Run.Metadata` is a `map[string]interface{}` JSONB field — it is not currently on the `Run` struct; add it if absent (check `model/run.go` and the DB schema / sqlc queries)
  - [ ] T1.4 Confirm `PipelineExecutor.executeStep` copies `run.Metadata` into `RunContext.Metadata` — it currently initialises `metadata := make(map[string]any)` fresh each time. Merge `run.Metadata` into this map before building `RunContext`

- [ ] **T2.** Propagate `PipelineStep.Model` into `RunContext.Metadata` (AC: #3)
  - [ ] T2.1 In `run_service.go` `LaunchRun`, when creating each `RunStep`, store `stepCfg.Model` in `RunStep.Metadata` (add `Metadata map[string]interface{}` to `RunStep` if absent, backed by JSONB)
  - [ ] T2.2 Alternatively (simpler for MVP): store the step model in `Run.Metadata` keyed by step order or name, so the executor can look it up. Evaluate which approach requires fewer schema changes.
  - [ ] T2.3 In `pipeline_executor.go` `executeStep`, after building `RunContext`, set `runCtx.Metadata["model"] = step.Model` when `step.Model != ""`
  - [ ] T2.4 In `agent_run.go` `createContainer`, read `runCtx.Metadata["model"]` and add `MODEL=<value>` to the container env when present; the entrypoint can use it to override the Claude Code model flag

- [ ] **T3.** Verify template_name injection (no regression) and add test (AC: #2)
  - [ ] T3.1 Confirm the existing logic in `pipeline_executor.go` lines 213–218 is correct and untouched
  - [ ] T3.2 Add a unit test in `pipeline_executor_test.go` (or a new file) that asserts `template_name` is set correctly for each of implement / review / merge action types

- [ ] **T4.** Update `agent_run_test.go` with new metadata tests (AC: #1, #3)
  - [ ] T4.1 Add test `TestAgentRunAction_BranchNameFromMetadata` — sets `branch_name` in `RunContext.Metadata`, verifies `BRANCH_NAME` env var is passed to the container
  - [ ] T4.2 Add test `TestAgentRunAction_ModelFromMetadata` — sets `model` in `RunContext.Metadata`, verifies the container env contains the expected model var
  - [ ] T4.3 Add test `TestAgentRunAction_ModelFallback` — omits `model` from metadata, verifies no model env var is injected (no regression on existing behaviour)

- [ ] **T5.** Run lint and tests (AC: #4)
  - [ ] T5.1 `cd backend && go test ./... -short`
  - [ ] T5.2 `cd backend && golangci-lint run ./...`
  - [ ] T5.3 Fix any lint errors before committing

---

## Dev Notes

### Key observations from code reading

**`branch_name` — root cause:**
`LaunchRun` in `run_service.go` (line 318) creates a `model.Run` with no `Metadata` field. The `Run` struct in `model/run.go` does not currently have a `Metadata` field. The `PipelineExecutor.ExecuteRun` initialises `metadata := make(map[string]any)` at line 113 (a fresh empty map), so nothing from the Run ever flows into `RunContext.Metadata`. The fix requires:
1. Adding `Metadata map[string]interface{}` to `model.Run` (check whether the DB column already exists as JSONB — look at migrations and sqlc-generated code before adding a migration)
2. Setting it in `LaunchRun`
3. Merging it into the `metadata` map in `ExecuteRun` before calling `executeStep`

**`template_name` — already correct:**
`pipeline_executor.go` lines 212–218 already inject `template_name` from `actionTypeToTemplateName`. `agent_run.go` `resolveTemplateName` reads it at line 161. No code change needed here — only a test to prevent regression.

**`model` propagation — two options:**

Option A (preferred, minimal schema change): Add `Metadata map[string]interface{}` to `model.RunStep` backed by JSONB. Store `model` there when creating steps in `LaunchRun`. In `executeStep`, merge step metadata into the shared `metadata` map before building `RunContext`.

Option B (no schema change): Store per-step model in `Run.Metadata` keyed as `"step_<order>_model"`. Simpler but less clean. Only use if the schema change for `RunStep.Metadata` is blocked.

Check `backend/migrations/` and `backend/internal/adapter/postgres/db/models.go` to determine which JSONB columns already exist on `runs` and `run_steps` before deciding.

**`model` in container env:**
The entrypoint (`agent/entrypoint.sh`) does not currently consume a `MODEL` env var. Adding it to the container env is safe — the entrypoint can ignore unknown vars. When the entrypoint is updated (separate story) to honour `MODEL`, it will already be there. For this story, just ensure the env var is passed through.

### File paths

| File | Action |
|------|--------|
| `backend/internal/domain/model/run.go` | ADD `Metadata map[string]interface{}` field to `Run` struct |
| `backend/internal/domain/model/run.go` | ADD `Metadata map[string]interface{}` field to `RunStep` struct (if option A) |
| `backend/internal/domain/service/run_service.go` | MODIFY `LaunchRun` — compute `branch_name`, populate `run.Metadata` and step metadata |
| `backend/internal/domain/service/pipeline_executor.go` | MODIFY `ExecuteRun` — merge `run.Metadata` into the shared `metadata` map |
| `backend/internal/domain/service/pipeline_executor.go` | MODIFY `executeStep` — inject `model` from step metadata into `RunContext.Metadata` |
| `backend/internal/adapter/action/agent_run.go` | MODIFY `createContainer` — add `MODEL` env var when `runCtx.Metadata["model"]` is set |
| `backend/internal/adapter/action/agent_run_test.go` | ADD tests for branch_name and model propagation |
| `backend/internal/domain/service/pipeline_executor_test.go` | ADD test for template_name injection per action_type |
| `backend/migrations/` | ADD migration if `Metadata` JSONB column is missing on `runs` or `run_steps` |
| `backend/queries/` | ADD sqlc queries if new columns require them; regenerate with `make generate` |

### Branch naming convention

The convention `feat/{story-key}` (lowercase) matches what `agent/entrypoint.sh` expects. Story keys are already lowercase in the system (e.g., `"runtime-4"` not `"RUNTIME-4"`). Confirm with the `Story.Key` value from the test fixtures before hardcoding the format.

### No changes to `actionTypeToTemplateName`

The map at `pipeline_executor.go` lines 23–27 is correct and complete. Do not touch it.

### Testing approach

- Use the existing `agentRunFixture` helper in `agent_run_test.go` — it already supports `Metadata` on `RunContext`
- For `pipeline_executor` tests, hand-write a minimal mock `RunRepository` and `ActionRegistry` — follow the patterns in `pipeline_executor_test.go` if it exists, or create one matching the style of `agent_run_test.go`
- All tests must be `-short` compatible (no containers, no network, no filesystem except temp dirs)

---

## Change Log

- Created 2026-02-22
