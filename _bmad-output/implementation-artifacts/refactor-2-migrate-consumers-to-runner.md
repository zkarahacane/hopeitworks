# Story refactor-2: Migrate Consumers to Runner

**Status:** ready-for-dev

**Blocked by:** refactor-1 (Runner interface and DockerRunner must be merged first)

## Story

As a backend developer, I want to migrate all consumers of `ContainerManager` to use the new `Runner` interface, so that the domain layer no longer references Docker-specific types, and the old `ContainerManager` port, `ContainerOpts` model, and their Docker adapter can be deleted.

This story assumes that `port/runner.go`, `model/agent_spec.go`, and `adapter/docker/runner.go` are already in place (story refactor-1 merged). The DB column rename is handled in story refactor-3.

## Acceptance Criteria

**AC1: `adapter/action/agent_run.go` uses `port.Runner` instead of `port.ContainerManager`**
- Given `AgentRunAction` is updated
- When an agent run is executed
- Then `AgentRunAction` holds a `runner port.Runner` field (not `containerMgr port.ContainerManager`)
- And `createContainer + Start` is replaced by a single `runner.Submit(ctx, spec)` call returning a handle
- And `cleanupContainer` is replaced by `runner.Terminate(ctx, handle)` (Stop+Remove collapsed into one call)
- And `AgentConfig.NetworkName` is removed (network is now in the `DockerRunner` constructor)
- And the handle returned by `Submit` is persisted to `run_steps.container_id` via `runRepo.UpdateRunStepContainerInfo` (column rename happens in refactor-3; field name in domain model `RunStep.ContainerID` remains for now)

**AC2: `service/orphan_cleaner.go` uses `port.Runner`**
- Given `OrphanCleaner` is updated
- When `CleanupOrphans` is called
- Then it calls `runner.List(ctx, {"managed_by": "hopeitworks"})` instead of `containerMgr.ListContainers`
- And it calls `runner.Terminate(ctx, info.Handle)` instead of `containerMgr.Remove`
- And the orphan detection logic (run_id label, run status check) is unchanged

**AC3: `service/timeout_enforcer.go` uses `port.Runner`**
- Given `TimeoutEnforcer` is updated
- When `CheckTimeouts` is called
- Then it calls `runner.List(ctx, {"managed_by": "hopeitworks"})` instead of `containerMgr.ListContainers`
- And it calls `runner.Terminate(ctx, info.Handle)` instead of `containerMgr.Stop` (Stop+Remove collapsed)
- And the timeout detection logic (started_at, project timeout override) is unchanged

**AC4: `cmd/api/main.go` wiring uses `DockerRunner`**
- Given `main.go` is updated
- When the application starts
- Then `NewDockerRunner(cfg.Docker.Host, cfg.Docker.AgentNetwork, logger)` is called once
- And the resulting `runner` is passed to `NewAgentRunAction`, `NewOrphanCleaner`, and `NewTimeoutEnforcer`
- And the separate `NewDockerLogStreamerFromHost` call remains unchanged (log streamer is independent)
- And `AgentConfig.NetworkName` field and its usage are removed from `main.go`

**AC5: Old files are deleted**
- Given the migration is complete
- When the repository is inspected
- Then `backend/internal/domain/port/container_manager.go` does not exist
- And `backend/internal/domain/model/container.go` does not exist
- And `backend/internal/adapter/docker/container_manager.go` does not exist
- And `backend/internal/adapter/docker/container_manager_test.go` does not exist

**AC6: Existing tests are updated and pass**
- Given the consumer test files reference `mockContainerManager` and `port.ContainerInfo`
- When those files are updated
- Then `adapter/action/agent_run_test.go` uses `mockRunner` implementing `port.Runner`
- And `service/timeout_enforcer_test.go` (and its shared mock) uses `mockRunner`
- And `service/orphan_cleaner_test.go` uses `mockRunner`
- And `go test ./... -short` passes with zero failures

**AC7: `golangci-lint run ./...` passes**
- Given all files are updated
- When the linter runs
- Then there are zero lint errors

## Tasks / Subtasks

- [ ] Update `backend/internal/adapter/action/agent_run.go` (AC: #1)
  - [ ] Replace `containerMgr port.ContainerManager` field with `runner port.Runner`
  - [ ] Replace `NewAgentRunAction` signature (remove `containerMgr`, add `runner port.Runner`; remove `AgentConfig.NetworkName`)
  - [ ] Replace `createContainer + containerMgr.Start` with `runner.Submit(ctx, spec)` building `model.AgentSpec`
  - [ ] Replace `cleanupContainer` (`Stop` + `Remove`) with `runner.Terminate(ctx, handle)`
  - [ ] Persist handle via `runRepo.UpdateRunStepContainerInfo(ctx, stepID, &handle, nil)` (field name unchanged for now)
  - [ ] Remove `AgentConfig.NetworkName` from the struct
- [ ] Update `backend/internal/domain/service/orphan_cleaner.go` (AC: #2)
  - [ ] Replace `containerMgr port.ContainerManager` with `runner port.Runner`
  - [ ] Replace `ListContainers` with `runner.List`, update loop variable from `.ID` to `.Handle`
  - [ ] Replace `containerMgr.Remove(ctx, container.ID)` with `runner.Terminate(ctx, info.Handle)`
  - [ ] Update `NewOrphanCleaner` constructor signature
- [ ] Update `backend/internal/domain/service/timeout_enforcer.go` (AC: #3)
  - [ ] Replace `containerMgr port.ContainerManager` with `runner port.Runner`
  - [ ] Replace `ListContainers` with `runner.List`, update loop variable from `.ID` to `.Handle`
  - [ ] Replace `containerMgr.Stop(ctx, container.ID)` with `runner.Terminate(ctx, info.Handle)`
  - [ ] Update `NewTimeoutEnforcer` constructor signature
- [ ] Update `backend/cmd/api/main.go` (AC: #4)
  - [ ] Replace `NewDockerContainerManager` call with `NewDockerRunner(cfg.Docker.Host, cfg.Docker.AgentNetwork, logger)`
  - [ ] Pass `runner` to `NewAgentRunAction`, `NewOrphanCleaner`, `NewTimeoutEnforcer`
  - [ ] Remove `AgentConfig.NetworkName` from the `AgentConfig` literal
- [ ] Update `backend/internal/adapter/action/agent_run_test.go` (AC: #6)
  - [ ] Replace `mockContainerManager` with `mockRunner` implementing `port.Runner`
  - [ ] Update all test cases: `Submit` replaces `Create`+`Start`, `Terminate` replaces `Stop`+`Remove`
- [ ] Update `backend/internal/domain/service/timeout_enforcer_test.go` (AC: #6)
  - [ ] Replace `mockContainerManager` with `mockRunner`; update `listFn`, `stopCalls` → `terminateCalls`
- [ ] Update `backend/internal/domain/service/orphan_cleaner_test.go` (AC: #6)
  - [ ] Replace `mockContainerManager` with `mockRunner`; update `removeCalls` → `terminateCalls`
- [ ] Delete `backend/internal/domain/port/container_manager.go` (AC: #5)
- [ ] Delete `backend/internal/domain/model/container.go` (AC: #5)
- [ ] Delete `backend/internal/adapter/docker/container_manager.go` (AC: #5)
- [ ] Delete `backend/internal/adapter/docker/container_manager_test.go` (AC: #5)
- [ ] Run `go build ./...` to confirm zero compilation errors (AC: #6)
- [ ] Run `go test ./... -short` and confirm all tests pass (AC: #6)
- [ ] Run `golangci-lint run ./...` and confirm zero errors (AC: #7)

## Dev Notes

### Dependencies

- Story refactor-1 must be merged before starting this story
- No new Go dependencies
- `model/run.go` field `RunStep.ContainerID *string` is NOT renamed in this story (DB rename is story refactor-3)

### File Paths

| Action | Path |
|--------|------|
| MODIFY | `backend/internal/adapter/action/agent_run.go` |
| MODIFY | `backend/internal/adapter/action/agent_run_test.go` |
| MODIFY | `backend/internal/domain/service/orphan_cleaner.go` |
| MODIFY | `backend/internal/domain/service/orphan_cleaner_test.go` |
| MODIFY | `backend/internal/domain/service/timeout_enforcer.go` |
| MODIFY | `backend/internal/domain/service/timeout_enforcer_test.go` |
| MODIFY | `backend/cmd/api/main.go` |
| DELETE | `backend/internal/domain/port/container_manager.go` |
| DELETE | `backend/internal/domain/model/container.go` |
| DELETE | `backend/internal/adapter/docker/container_manager.go` |
| DELETE | `backend/internal/adapter/docker/container_manager_test.go` |

### Technical Specifications

#### `agent_run.go` — `createContainer` becomes `buildSpec`

The `createContainer` method currently builds `model.ContainerOpts` and calls `containerMgr.Create`. Replace it with a `buildSpec` method that builds `model.AgentSpec`:

```go
func (a *AgentRunAction) buildSpec(
    runCtx *model.RunContext,
    project *model.Project,
    story *model.Story,
    claudeMD, prompt, branchName string,
) model.AgentSpec {
    repoURL := ""
    if project.RepoURL != nil {
        repoURL = *project.RepoURL
    }
    return model.AgentSpec{
        Image: a.config.DefaultImage,
        Resources: model.AgentResources{
            Memory: a.config.DefaultMemory,
            CPUs:   a.config.DefaultCPUs,
        },
        Env: []string{
            "CLAUDE_MD_CONTENT=" + claudeMD,
            "REPO_URL=" + repoURL,
            "BRANCH_NAME=" + branchName,
            "STORY_KEY=" + story.Key,
            "PROMPT_CONTENT=" + prompt,
            "GITHUB_TOKEN=" + os.Getenv("GITHUB_TOKEN"),
            "CLAUDE_CODE_OAUTH_TOKEN=" + os.Getenv("CLAUDE_CODE_OAUTH_TOKEN"),
        },
        Labels: map[string]string{
            "managed_by": "hopeitworks",
            "run_id":     runCtx.Run.ID.String(),
            "step_id":    runCtx.RunStep.ID.String(),
            "story_key":  story.Key,
        },
    }
}
```

The `Execute` method changes from:
```go
containerID, err := a.createContainer(...)
defer a.cleanupContainer(containerID)
a.containerMgr.Start(ctx, containerID)
a.persistContainerID(ctx, runCtx.RunStep.ID, containerID)
```
To:
```go
spec := a.buildSpec(runCtx, project, story, claudeMD, prompt, branchName)
handle, err := a.runner.Submit(ctx, spec)
if err != nil {
    return fmt.Errorf("submit agent workload: %w", err)
}
defer a.cleanupWorkload(handle)
a.persistHandle(ctx, runCtx.RunStep.ID, handle)
```

#### `AgentConfig` struct cleanup

Remove `NetworkName string` from `AgentConfig` since the network is now in the `DockerRunner` constructor. The field was previously used in `createContainer` → `ContainerOpts.NetworkName`.

#### `cleanupContainer` → `cleanupWorkload`

```go
func (a *AgentRunAction) cleanupWorkload(handle string) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    if err := a.runner.Terminate(ctx, handle); err != nil {
        a.logger.Warn("failed to terminate agent workload during cleanup",
            "handle", handle, "error", err)
    }
    a.logger.Debug("agent workload cleaned up", "handle", handle)
}
```

#### `persistContainerID` → `persistHandle`

```go
func (a *AgentRunAction) persistHandle(ctx context.Context, stepID uuid.UUID, handle string) {
    if _, err := a.runRepo.UpdateRunStepContainerInfo(ctx, stepID, &handle, nil); err != nil {
        a.logger.Warn("failed to persist workload handle to run step",
            "step_id", stepID, "handle", handle, "error", err)
    }
}
```

#### `orphan_cleaner.go` — List + Terminate

```go
func (o *OrphanCleaner) CleanupOrphans(ctx context.Context) error {
    workloads, err := o.runner.List(ctx, map[string]string{"managed_by": "hopeitworks"})
    if err != nil {
        return err
    }
    orphanCount := 0
    for _, w := range workloads {
        runIDStr := w.Labels["run_id"]
        // ... same orphan detection logic, but use w.Handle instead of container.ID ...
        o.terminateOrphan(ctx, w.Handle, reason)
    }
    ...
}

func (o *OrphanCleaner) terminateOrphan(ctx context.Context, handle, reason string) {
    if err := o.runner.Terminate(ctx, handle); err != nil {
        o.logger.Error("failed to terminate orphan workload", "handle", handle, "reason", reason, "error", err)
    } else {
        o.logger.Info("terminated orphan workload", "handle", handle, "reason", reason)
    }
}
```

#### `timeout_enforcer.go` — List + Terminate

```go
func (t *TimeoutEnforcer) CheckTimeouts(ctx context.Context) error {
    workloads, err := t.runner.List(ctx, map[string]string{"managed_by": "hopeitworks"})
    if err != nil {
        return err
    }
    for _, w := range workloads {
        runIDStr := w.Labels["run_id"]
        stepIDStr := w.Labels["step_id"]
        // ... same timeout logic, use w.Handle ...
        if err := t.runner.Terminate(ctx, w.Handle); err != nil {
            t.logger.Error("failed to terminate timed-out workload", "handle", w.Handle, "error", err)
            continue
        }
        // ... update run step and run status ...
    }
    return nil
}
```

Note: the old `TimeoutEnforcer` called `Stop` only (not `Remove`). With `Terminate`, the container is also removed. This is intentional — a timed-out container should be cleaned up completely.

#### Test mocks

The `mockRunner` in service tests follows the same pattern as `mockContainerManager`:

```go
type mockRunner struct {
    mu              sync.Mutex
    listFn          func(ctx context.Context, labels map[string]string) ([]port.RunnerInfo, error)
    terminateCalls  []string
    terminateFn     func(ctx context.Context, handle string) error
    submitFn        func(ctx context.Context, spec model.AgentSpec) (string, error)
    waitFn          func(ctx context.Context, handle string) (int, error)
}
```

Methods `getTerminateCalls()` and `getSubmitCalls()` follow the same mutex-protected copy pattern.

In `agent_run_test.go`, the `mockContainerManager` type is replaced by `mockRunner`. All tests that set `createFn`/`startFn` are updated to set `submitFn`. All tests that check `stopCalls`/`removeCalls` are updated to check `terminateCalls`.

#### `main.go` wiring diff (key changes)

```go
// Before:
containerMgr, err := dockeradapter.NewDockerContainerManager(cfg.Docker.Host, logger)
// ...
agentCfg := actionadapter.AgentConfig{
    NetworkName: cfg.Docker.AgentNetwork,
    // ...
}
agentRunAction := actionadapter.NewAgentRunAction(containerMgr, logStreamer, ...)
// ...
orphanCleaner := service.NewOrphanCleaner(containerMgr, runRepo, logger)
timeoutEnforcer := service.NewTimeoutEnforcer(containerMgr, ...)

// After:
runner, err := dockeradapter.NewDockerRunner(cfg.Docker.Host, cfg.Docker.AgentNetwork, logger)
// ...
agentCfg := actionadapter.AgentConfig{
    // NetworkName removed
    // ...
}
agentRunAction := actionadapter.NewAgentRunAction(runner, logStreamer, ...)
// ...
orphanCleaner := service.NewOrphanCleaner(runner, runRepo, logger)
timeoutEnforcer := service.NewTimeoutEnforcer(runner, ...)
```

### References

- `port/runner.go` created in story refactor-1
- `model/agent_spec.go` created in story refactor-1
- `adapter/docker/runner.go` created in story refactor-1
- DB column rename deferred to story refactor-3
- `model/run.go` `RunStep.ContainerID` field stays as-is until story refactor-3
