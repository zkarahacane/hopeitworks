# Story refactor-1: Create Runner Interface and AgentSpec Model

**Status:** ready-for-dev

## Story

As a backend developer, I want to replace the `ContainerManager` port (6 methods, Docker-coupled) with a new `Runner` port (4 methods, runtime-agnostic), and introduce `AgentSpec` as the canonical input model, so that the domain layer no longer leaks Docker terminology and a future Kubernetes adapter can be plugged in without touching any consumer code.

This story covers only the creation of the new abstractions and the Docker implementation. Consumers (`agent_run.go`, `orphan_cleaner.go`, `timeout_enforcer.go`) are **not** modified in this story — they continue to use `ContainerManager` until story refactor-2.

## Acceptance Criteria

**AC1: `port/runner.go` exists with the correct interface**
- Given the file `backend/internal/domain/port/runner.go` exists
- When another package imports it
- Then it exports a `Runner` interface with exactly 4 methods: `Submit`, `Wait`, `Terminate`, `List`
- And it exports a `RunnerInfo` struct with fields `Handle string`, `Labels map[string]string`, `CreatedAt time.Time`

**AC2: `model/agent_spec.go` exists with the correct types**
- Given the file `backend/internal/domain/model/agent_spec.go` exists
- When another package imports it
- Then it exports `AgentSpec` with fields `Image string`, `Env []string`, `Labels map[string]string`, `Resources AgentResources`
- And it exports `AgentResources` with fields `Memory int64`, `CPUs float64`
- And `AgentSpec` does NOT contain a `NetworkName` field (network is constructor-level config)

**AC3: `adapter/docker/runner.go` implements `port.Runner`**
- Given `DockerRunner` is created via `NewDockerRunner(host, network, logger)`
- When `Submit(ctx, spec)` is called with a valid `AgentSpec`
- Then it creates AND starts the container in a single atomic operation
- And it enforces `managed_by=hopeitworks` label unconditionally
- And it sets `Privileged=false` and `Binds=nil` (security constraints preserved)
- And it returns an opaque handle (the container ID)

**AC4: `Submit` and `Terminate` are atomic**
- Given a `DockerRunner`
- When `Submit` is called, the container is both created and started before the handle is returned
- When `Terminate` is called with a valid handle, it stops (SIGTERM, 10s) then removes the container (force + volumes) in one call
- And `Terminate` is idempotent: calling it twice on the same handle does not return an error

**AC5: `port/log_streamer.go` uses `handle` instead of `containerID`**
- Given `port/log_streamer.go` is updated
- When `StreamLogs` is called
- Then the first parameter name is `handle string` (not `containerID string`)
- And the semantic contract is identical

**AC6: `adapter/docker/log_streamer.go` is updated to match**
- Given `adapter/docker/log_streamer.go` is updated
- When `StreamLogs` is called with a `handle`
- Then it passes the handle directly to `ContainerLogs` and `ContainerWait` (Docker container ID = handle for the Docker adapter)
- And all internal log fields use `handle` instead of `container_id`

**AC7: `port.ContainerManager` and `model.ContainerOpts` are NOT removed**
- Given this story is only additive
- When the code compiles
- Then `port/container_manager.go` still exists and is unchanged
- And `model/container.go` still exists and is unchanged
- And all existing tests still pass

**AC8: Unit tests for `DockerRunner` cover the new interface**
- Given `adapter/docker/runner_test.go` exists
- When the test suite runs with `go test ./... -short`
- Then it covers: `Submit` success (labels, resources, network, security), `Submit` error propagation, `Wait` success (exit 0 and non-zero), `Wait` error, `Wait` context cancellation, `Terminate` success, `Terminate` idempotent (not found is not an error), `List` success, `List` error

## Tasks / Subtasks

- [ ] Create `backend/internal/domain/port/runner.go` with `Runner` interface and `RunnerInfo` struct (AC: #1)
- [ ] Create `backend/internal/domain/model/agent_spec.go` with `AgentSpec` and `AgentResources` (AC: #2)
- [ ] Create `backend/internal/adapter/docker/runner.go` implementing `port.Runner` (AC: #3, #4)
  - [ ] `NewDockerRunner(host string, network string, logger *slog.Logger) (*DockerRunner, error)`
  - [ ] `Submit`: ContainerCreate + ContainerStart, enforce `managed_by` label, security constraints
  - [ ] `Wait`: wrap `ContainerWait` with context cancellation handling
  - [ ] `Terminate`: stop (10s grace) then remove (force + volumes); treat "container not found" as success
  - [ ] `List`: wrap `ContainerList` with label filters, map to `[]port.RunnerInfo`
- [ ] Update `backend/internal/domain/port/log_streamer.go`: rename `containerID` param to `handle` (AC: #5)
- [ ] Update `backend/internal/adapter/docker/log_streamer.go`: rename `containerID` to `handle` in all method signatures and internal log fields (AC: #6)
- [ ] Create `backend/internal/adapter/docker/runner_test.go` with unit tests using `mockDockerClient` pattern (AC: #8)
- [ ] Verify `go test ./... -short` passes and `golangci-lint run ./...` is clean

## Dev Notes

### Dependencies

- No new Go dependencies — uses the existing `github.com/docker/docker` SDK already in `go.mod`
- `port/container_manager.go` and `model/container.go` must NOT be touched in this story

### File Paths

| Action | Path |
|--------|------|
| CREATE | `backend/internal/domain/port/runner.go` |
| CREATE | `backend/internal/domain/model/agent_spec.go` |
| CREATE | `backend/internal/adapter/docker/runner.go` |
| CREATE | `backend/internal/adapter/docker/runner_test.go` |
| MODIFY | `backend/internal/domain/port/log_streamer.go` |
| MODIFY | `backend/internal/adapter/docker/log_streamer.go` |

### Technical Specifications

#### `port/runner.go`

```go
package port

import (
    "context"
    "time"

    "github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// RunnerInfo represents metadata about a submitted workload.
type RunnerInfo struct {
    Handle    string
    Labels    map[string]string
    CreatedAt time.Time
}

// Runner abstracts workload lifecycle operations, agnostic of the underlying runtime.
// For Docker, a handle is the container ID. For Kubernetes, it would be namespace/job-name.
type Runner interface {
    // Submit creates and starts a workload from the given spec.
    // Returns an opaque handle identifying the workload.
    Submit(ctx context.Context, spec model.AgentSpec) (string, error)

    // Wait blocks until the workload identified by handle exits.
    // Returns the exit code.
    Wait(ctx context.Context, handle string) (int, error)

    // Terminate stops and removes the workload identified by handle.
    // Terminate is idempotent: if the workload does not exist, no error is returned.
    Terminate(ctx context.Context, handle string) error

    // List returns all workloads matching the specified labels.
    List(ctx context.Context, labels map[string]string) ([]RunnerInfo, error)
}
```

#### `model/agent_spec.go`

```go
package model

// AgentSpec specifies the configuration for a workload submitted to a Runner.
type AgentSpec struct {
    // Image is the container image (e.g., "hopeitworks/agent:latest").
    Image string

    // Env is a list of environment variables in KEY=VALUE format.
    Env []string

    // Labels are key-value pairs for workload metadata.
    // Standard labels: managed_by, run_id, step_id, story_key.
    Labels map[string]string

    // Resources defines resource limits for the workload.
    Resources AgentResources
}

// AgentResources defines resource constraints for a workload.
type AgentResources struct {
    // Memory is the memory limit in bytes (0 = unlimited).
    Memory int64

    // CPUs is the CPU limit as a float (0 = unlimited, 1.0 = 1 CPU).
    CPUs float64
}
```

#### `adapter/docker/runner.go` — key implementation notes

- The `dockerClient` interface in `runner.go` must mirror the one in `container_manager.go` (same Docker SDK methods) — do not share the interface type to keep the files independent
- `Submit` must enforce `managed_by=hopeitworks` even if `spec.Labels` is nil (initialize map first)
- `Terminate` must use `errdefs.IsNotFound(err)` from `github.com/docker/docker/errdefs` to detect "container not found" and return nil in that case
- `Wait` selects on `statusCh`, `errCh`, and `ctx.Done()` — same pattern as the existing `ContainerManager.Wait`
- Constructor: `NewDockerRunner(host string, network string, logger *slog.Logger) (*DockerRunner, error)`
- The network is stored in the struct field `network string` and applied in `Submit` only (not in the interface)

#### `log_streamer.go` — rename only

The change in `port/log_streamer.go` is purely a parameter rename: `containerID string` → `handle string`. The semantic contract is identical — for the Docker adapter, handle == container ID. This is a source-compatible change within the package since the interface method is called via the port interface.

The change in `adapter/docker/log_streamer.go`:
- Rename all occurrences of `containerID` parameter to `handle` in `StreamLogs`, `streamLoop`, and `handleContainerExit`
- Update all `slog.String("container_id", ...)` log fields to `slog.String("handle", ...)`
- The value passed to `client.ContainerLogs` and `client.ContainerWait` remains the same (`handle`)

#### `Terminate` idempotency

```go
import "github.com/docker/docker/errdefs"

func (r *DockerRunner) Terminate(ctx context.Context, handle string) error {
    timeout := stopTimeoutSeconds
    if err := r.client.ContainerStop(ctx, handle, dockercontainer.StopOptions{Timeout: &timeout}); err != nil {
        if !errdefs.IsNotFound(err) {
            return apperrors.NewContainerError(...)
        }
        // Already gone — skip remove, return nil
        return nil
    }
    if err := r.client.ContainerRemove(ctx, handle, dockercontainer.RemoveOptions{Force: true, RemoveVolumes: true}); err != nil {
        if !errdefs.IsNotFound(err) {
            return apperrors.NewContainerError(...)
        }
    }
    return nil
}
```

#### Compile-time interface check

Add at the top of `runner.go`:
```go
var _ port.Runner = (*DockerRunner)(nil)
```

### Testing Requirements

- Test file: `backend/internal/adapter/docker/runner_test.go`
- Use the same `mockDockerClient` pattern as `container_manager_test.go` (the mock must implement the `dockerClient` interface defined in `runner.go`)
- The mock for `runner_test.go` is a separate type in the same package — do NOT reuse the one from `container_manager_test.go` to avoid coupling
- Required test cases:
  - `TestSubmit_Success`: verifies container created+started, `managed_by` label set, image/env passed correctly, security constraints (Privileged=false, Binds=nil)
  - `TestSubmit_LabelsNil`: verifies `managed_by` is still set when `spec.Labels` is nil
  - `TestSubmit_MemoryAndCPULimits`: verifies NanoCPUs and Memory passed to HostConfig
  - `TestSubmit_Network`: verifies NetworkingConfig has the runner's network
  - `TestSubmit_CreateError`: Docker create error → `DomainError` with code `CONTAINER_OPERATION_FAILED`
  - `TestSubmit_StartError`: Docker start error → `DomainError`
  - `TestWait_SuccessExitZero`: returns 0
  - `TestWait_SuccessNonZeroExit`: returns non-zero exit code without error
  - `TestWait_Error`: Docker wait error → `DomainError`
  - `TestWait_ContextCancelled`: context cancelled → `DomainError`
  - `TestTerminate_Success`: stop then remove called in order
  - `TestTerminate_NotFound`: container not found on stop → returns nil (idempotent)
  - `TestTerminate_StopError`: non-notfound stop error → returns `DomainError`
  - `TestList_Success`: returns correct `RunnerInfo` slice with handles and labels
  - `TestList_Empty`: returns empty slice, no error
  - `TestList_Error`: Docker list error → `DomainError`

### References

- Existing implementation to mirror: `backend/internal/adapter/docker/container_manager.go`
- Existing test pattern to follow: `backend/internal/adapter/docker/container_manager_test.go`
- `errdefs` package: `github.com/docker/docker/errdefs` (already in `go.mod` transitively via Docker SDK)
- Stories consuming this interface: refactor-2 (migration of consumers)
