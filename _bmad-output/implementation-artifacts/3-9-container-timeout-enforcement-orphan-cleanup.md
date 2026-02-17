# Story 3.9: [BACK] Container timeout enforcement + orphan cleanup

Status: ready-for-dev

## Story

As a platform operator, I want container timeouts and orphan cleanup, so that runaway containers don't consume resources indefinitely.

## Acceptance Criteria (BDD)

**AC1: TimeoutEnforcer domain service monitors running containers**
- **Given** a TimeoutEnforcer service in `backend/internal/domain/service/timeout_enforcer.go`
- **When** the service is reviewed
- **Then** it has a Start(ctx) method that launches a background monitoring goroutine
- **And** it depends on ContainerManager port and RunRepository port
- **And** it uses slog for structured logging
- **And** it has a configurable default timeout (30 minutes)

**AC2: TimeoutEnforcer detects containers exceeding timeout**
- **Given** a running container managed by TimeoutEnforcer
- **When** CheckTimeouts is called periodically (every 30 seconds)
- **Then** it lists all containers with managed_by=hopeitworks label
- **And** it checks each container's started_at time from run_step record
- **And** it compares time.Since(started_at) against configured timeout
- **And** it identifies containers exceeding timeout

**AC3: TimeoutEnforcer enforces timeout by stopping container and marking step failed**
- **Given** a container has exceeded its timeout
- **When** timeout enforcement is triggered
- **Then** it stops the container via ContainerManager.Stop
- **And** it marks the run_step as 'failed' with error "container_timeout"
- **And** it marks the run as 'failed'
- **And** it logs the timeout event via slog with container ID and run ID context

**AC4: TimeoutEnforcer uses project-specific timeout if configured**
- **Given** a project has max_container_timeout configured
- **When** TimeoutEnforcer checks timeout for a container belonging to that project
- **Then** it uses project.max_container_timeout instead of default timeout
- **And** it logs the applied timeout value via slog

**AC5: OrphanCleaner domain service removes orphaned containers on startup**
- **Given** an OrphanCleaner service in `backend/internal/domain/service/orphan_cleaner.go`
- **When** the service is reviewed
- **Then** it has a CleanupOrphans(ctx) method
- **And** it depends on ContainerManager port and RunRepository port
- **And** it uses slog for structured logging

**AC6: OrphanCleaner detects containers not associated with active runs**
- **Given** containers exist with managed_by=hopeitworks label
- **When** CleanupOrphans is called
- **Then** it lists all containers with managed_by=hopeitworks label
- **And** for each container, it extracts run_id from container labels
- **And** it checks if run_id exists and is in active state (running, pending)
- **And** it identifies orphan containers (no active run or run not found)

**AC7: OrphanCleaner removes all orphan containers and logs summary**
- **Given** multiple orphan containers are detected
- **When** cleanup executes
- **Then** it removes each orphan container via ContainerManager.Remove
- **And** it logs each removal via slog with container ID context
- **And** it logs a summary via slog with total orphans removed count
- **And** it continues cleanup even if individual removals fail (logs error, does not halt)

**AC8: ContainerManager port includes ListContainers method**
- **Given** the ContainerManager port interface in `backend/internal/domain/port/container_manager.go`
- **When** the interface is reviewed
- **Then** it declares ListContainers(ctx, labels map[string]string) ([]ContainerInfo, error)
- **And** ContainerInfo struct includes ID, Labels, CreatedAt fields
- **And** the method filters containers by label key-value pairs

**AC9: Unit tests verify TimeoutEnforcer behavior**
- **Given** unit tests in `backend/internal/domain/service/timeout_enforcer_test.go`
- **When** tests are executed
- **Then** timeout detection test verifies container exceeding timeout is identified
- **And** timeout enforcement test verifies Stop is called and step marked failed
- **And** configurable timeout test verifies project timeout overrides default
- **And** no timeout test verifies containers within timeout are not stopped
- **And** all tests use mock ContainerManager and mock RunRepository

**AC10: Unit tests verify OrphanCleaner behavior**
- **Given** unit tests in `backend/internal/domain/service/orphan_cleaner_test.go`
- **When** tests are executed
- **Then** no orphans test verifies no containers removed when all have active runs
- **And** multiple orphans test verifies all orphan containers are removed
- **And** cleanup failure test verifies cleanup continues on individual removal error
- **And** summary logging test verifies total count is logged
- **And** all tests use mock ContainerManager and mock RunRepository

**AC11: TimeoutEnforcer and OrphanCleaner wired into API startup**
- **Given** API startup in `backend/cmd/api/main.go`
- **When** the application starts
- **Then** OrphanCleaner.CleanupOrphans is called once during startup
- **And** TimeoutEnforcer.Start is launched as a background goroutine
- **And** both services are gracefully stopped on application shutdown

## Tasks / Subtasks

- [ ] [BACK] Task 1: Extend ContainerManager port with ListContainers method (AC: #8)
  - [ ] Edit `backend/internal/domain/port/container_manager.go`
  - [ ] Add ContainerInfo struct with ID, Labels, CreatedAt fields
  - [ ] Add ListContainers(ctx, labels) method to ContainerManager interface
  - [ ] Document method with godoc comments

- [ ] [BACK] Task 2: Create TimeoutEnforcer domain service (AC: #1)
  - [ ] Create `backend/internal/domain/service/timeout_enforcer.go`
  - [ ] Define TimeoutEnforcer struct with ContainerManager, RunRepository, logger, defaultTimeout dependencies
  - [ ] Implement Start(ctx) method launching background goroutine with ticker (30s interval)
  - [ ] Document service and methods with godoc comments

- [ ] [BACK] Task 3: Implement timeout detection and enforcement (AC: #2, #3, #4)
  - [ ] Implement CheckTimeouts(ctx) method in TimeoutEnforcer
  - [ ] List all containers via ContainerManager.ListContainers(managed_by=hopeitworks)
  - [ ] For each container: extract run_id, step_id from labels, fetch run_step from RunRepository
  - [ ] Compare time.Since(started_at) against timeout (project timeout or default)
  - [ ] If exceeded: Stop container, mark step failed with error "container_timeout", mark run failed
  - [ ] Log timeout event via slog with structured fields (container_id, run_id, step_id, timeout)

- [ ] [BACK] Task 4: Create OrphanCleaner domain service (AC: #5)
  - [ ] Create `backend/internal/domain/service/orphan_cleaner.go`
  - [ ] Define OrphanCleaner struct with ContainerManager, RunRepository, logger dependencies
  - [ ] Implement CleanupOrphans(ctx) method
  - [ ] Document service and methods with godoc comments

- [ ] [BACK] Task 5: Implement orphan detection and cleanup (AC: #6, #7)
  - [ ] List all containers via ContainerManager.ListContainers(managed_by=hopeitworks)
  - [ ] For each container: extract run_id from labels
  - [ ] Check if run_id exists in RunRepository and run status is active (running, pending)
  - [ ] If run not found or not active: mark container as orphan
  - [ ] Remove each orphan via ContainerManager.Remove
  - [ ] Log each removal and final summary (total orphans removed) via slog
  - [ ] Continue cleanup on error (log error, don't halt)

- [ ] [BACK] Task 6: Write unit tests for TimeoutEnforcer (AC: #9)
  - [ ] Create `backend/internal/domain/service/timeout_enforcer_test.go`
  - [ ] Create mock ContainerManager and mock RunRepository
  - [ ] Test timeout hit: container exceeds timeout → Stop called, step/run marked failed
  - [ ] Test timeout not hit: container within timeout → no Stop, no state change
  - [ ] Test configurable timeout: project timeout overrides default
  - [ ] Test error handling: Stop failure logged, does not panic

- [ ] [BACK] Task 7: Write unit tests for OrphanCleaner (AC: #10)
  - [ ] Create `backend/internal/domain/service/orphan_cleaner_test.go`
  - [ ] Create mock ContainerManager and mock RunRepository
  - [ ] Test no orphans: all containers have active runs → no removals
  - [ ] Test multiple orphans: containers without active runs → all removed
  - [ ] Test cleanup failure: Remove error logged, cleanup continues
  - [ ] Test summary logging: total orphans count logged via slog

- [ ] [BACK] Task 8: Wire TimeoutEnforcer and OrphanCleaner into API startup (AC: #11)
  - [ ] Edit `backend/cmd/api/main.go` or app initialization
  - [ ] Create OrphanCleaner instance, call CleanupOrphans(ctx) during startup
  - [ ] Create TimeoutEnforcer instance, launch Start(ctx) as goroutine
  - [ ] Ensure graceful shutdown: cancel context on SIGTERM/SIGINT to stop TimeoutEnforcer
  - [ ] Update wire.go provider sets if needed for DI

- [ ] [BACK] Task 9: Implement ListContainers in Docker adapter (AC: #8)
  - [ ] Edit `backend/internal/adapter/docker/container_manager.go`
  - [ ] Implement ListContainers(ctx, labels) using Docker SDK ContainerList with filters
  - [ ] Map Docker container response to ContainerInfo struct
  - [ ] Wrap errors in DomainError with context

## Dev Notes

### Dependencies
- Story 3-4: Docker container lifecycle (provides ContainerManager port with Stop, Remove, existing)
- Story 3-1: Runs/RunSteps tables (provides RunRepository for checking active runs)
- Story 3-7: Pipeline executor (handles step failure cascading, wave 6)
- Docker SDK: `github.com/docker/docker/client`
- Docker API types: `github.com/docker/docker/api/types/container`, `github.com/docker/docker/api/types/filters`

### Architecture Requirements
- **Hexagonal architecture:** TimeoutEnforcer and OrphanCleaner are domain services, depend on ports (ContainerManager, RunRepository)
- **No direct Docker SDK imports in domain services:** all Docker operations via ContainerManager port
- **Testability:** Both services use injected ports, allowing mock implementations in unit tests
- **Error handling:** All errors wrapped in DomainError via pkg/errors
- **Structured logging:** Use slog for all events (timeout detected, container stopped, orphan removed, summary)
- **Graceful shutdown:** TimeoutEnforcer stops when context is cancelled

### File Paths (exact)
```
backend/internal/domain/service/timeout_enforcer.go       # TimeoutEnforcer service
backend/internal/domain/service/timeout_enforcer_test.go  # TimeoutEnforcer unit tests
backend/internal/domain/service/orphan_cleaner.go         # OrphanCleaner service
backend/internal/domain/service/orphan_cleaner_test.go    # OrphanCleaner unit tests
backend/internal/domain/port/container_manager.go         # ADD ListContainers method, ContainerInfo struct
backend/internal/adapter/docker/container_manager.go      # IMPLEMENT ListContainers method
backend/cmd/api/main.go                                   # Wire services into startup
```

### Technical Specifications

**Extended ContainerManager port interface:**
```go
package port

import (
    "context"
    "time"
)

// ContainerInfo represents metadata about a container.
type ContainerInfo struct {
    ID        string
    Labels    map[string]string
    CreatedAt time.Time
}

// ContainerManager abstracts Docker container lifecycle operations.
type ContainerManager interface {
    // ... existing methods (Create, Start, Stop, Remove, Wait) ...

    // ListContainers lists all containers matching the specified labels.
    // labels is a map of key-value pairs for filtering (e.g., managed_by=hopeitworks).
    ListContainers(ctx context.Context, labels map[string]string) ([]ContainerInfo, error)
}
```

**TimeoutEnforcer service:**
```go
package service

import (
    "context"
    "log/slog"
    "time"

    "hopeitworks/backend/internal/domain/port"
)

type TimeoutEnforcer struct {
    containerMgr   port.ContainerManager
    runRepo        port.RunRepository
    logger         *slog.Logger
    defaultTimeout time.Duration // 30 minutes
}

func NewTimeoutEnforcer(
    containerMgr port.ContainerManager,
    runRepo port.RunRepository,
    logger *slog.Logger,
    defaultTimeout time.Duration,
) *TimeoutEnforcer {
    return &TimeoutEnforcer{
        containerMgr:   containerMgr,
        runRepo:        runRepo,
        logger:         logger,
        defaultTimeout: defaultTimeout,
    }
}

// Start begins monitoring all active containers in a background goroutine.
// It checks every 30 seconds for containers exceeding their timeout.
func (t *TimeoutEnforcer) Start(ctx context.Context) error {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            t.logger.Info("timeout enforcer stopped")
            return ctx.Err()
        case <-ticker.C:
            if err := t.CheckTimeouts(ctx); err != nil {
                t.logger.Error("timeout check failed", "error", err)
            }
        }
    }
}

// CheckTimeouts iterates active containers and enforces timeouts.
func (t *TimeoutEnforcer) CheckTimeouts(ctx context.Context) error {
    // List all containers with managed_by=hopeitworks label
    containers, err := t.containerMgr.ListContainers(ctx, map[string]string{
        "managed_by": "hopeitworks",
    })
    if err != nil {
        return err
    }

    for _, container := range containers {
        // Extract run_id and step_id from labels
        runID := container.Labels["run_id"]
        stepID := container.Labels["step_id"]

        if runID == "" || stepID == "" {
            continue
        }

        // Fetch run_step to get started_at and project timeout
        runStep, err := t.runRepo.GetRunStep(ctx, stepID)
        if err != nil {
            t.logger.Warn("failed to fetch run step", "step_id", stepID, "error", err)
            continue
        }

        // Determine timeout (project-specific or default)
        timeout := t.defaultTimeout
        project, err := t.runRepo.GetProjectForRun(ctx, runID)
        if err == nil && project.MaxContainerTimeout > 0 {
            timeout = project.MaxContainerTimeout
        }

        // Check if timeout exceeded
        elapsed := time.Since(runStep.StartedAt)
        if elapsed > timeout {
            t.logger.Warn("container timeout exceeded",
                "container_id", container.ID,
                "run_id", runID,
                "step_id", stepID,
                "elapsed", elapsed,
                "timeout", timeout,
            )

            // Stop container
            if err := t.containerMgr.Stop(ctx, container.ID); err != nil {
                t.logger.Error("failed to stop container", "container_id", container.ID, "error", err)
                continue
            }

            // Mark step and run as failed
            if err := t.runRepo.UpdateRunStepStatus(ctx, stepID, "failed", "container_timeout"); err != nil {
                t.logger.Error("failed to update run step status", "step_id", stepID, "error", err)
            }

            if err := t.runRepo.UpdateRunStatus(ctx, runID, "failed"); err != nil {
                t.logger.Error("failed to update run status", "run_id", runID, "error", err)
            }

            t.logger.Info("container stopped due to timeout", "container_id", container.ID, "run_id", runID)
        }
    }

    return nil
}
```

**OrphanCleaner service:**
```go
package service

import (
    "context"
    "log/slog"

    "hopeitworks/backend/internal/domain/port"
)

type OrphanCleaner struct {
    containerMgr port.ContainerManager
    runRepo      port.RunRepository
    logger       *slog.Logger
}

func NewOrphanCleaner(
    containerMgr port.ContainerManager,
    runRepo port.RunRepository,
    logger *slog.Logger,
) *OrphanCleaner {
    return &OrphanCleaner{
        containerMgr: containerMgr,
        runRepo:      runRepo,
        logger:       logger,
    }
}

// CleanupOrphans removes containers not associated with active runs.
// Called once during API startup.
func (o *OrphanCleaner) CleanupOrphans(ctx context.Context) error {
    // List all containers with managed_by=hopeitworks label
    containers, err := o.containerMgr.ListContainers(ctx, map[string]string{
        "managed_by": "hopeitworks",
    })
    if err != nil {
        return err
    }

    orphanCount := 0

    for _, container := range containers {
        // Extract run_id from labels
        runID := container.Labels["run_id"]

        if runID == "" {
            // No run_id label → orphan
            o.removeOrphan(ctx, container.ID, "no_run_id_label")
            orphanCount++
            continue
        }

        // Check if run exists and is active
        run, err := o.runRepo.GetRun(ctx, runID)
        if err != nil {
            // Run not found → orphan
            o.removeOrphan(ctx, container.ID, "run_not_found")
            orphanCount++
            continue
        }

        // Check if run is active (running, pending)
        if run.Status != "running" && run.Status != "pending" {
            // Run is completed/failed → orphan
            o.removeOrphan(ctx, container.ID, "run_not_active")
            orphanCount++
            continue
        }
    }

    o.logger.Info("orphan cleanup completed", "orphans_removed", orphanCount)
    return nil
}

func (o *OrphanCleaner) removeOrphan(ctx context.Context, containerID, reason string) {
    if err := o.containerMgr.Remove(ctx, containerID); err != nil {
        o.logger.Error("failed to remove orphan container",
            "container_id", containerID,
            "reason", reason,
            "error", err,
        )
    } else {
        o.logger.Info("removed orphan container",
            "container_id", containerID,
            "reason", reason,
        )
    }
}
```

**Container label convention:**
- All agent containers have label `managed_by=hopeitworks`
- Containers also have labels: `run_id=<uuid>`, `step_id=<uuid>`
- OrphanCleaner: list by managed_by label → for each, check if run_id is in active state → if not, remove
- TimeoutEnforcer: list by managed_by label → for each, check started_at from run_step → if timeout exceeded, stop and mark failed

**Error codes to add to pkg/errors:**
```go
const (
    ErrCodeContainerTimeout       = "CONTAINER_TIMEOUT"
    ErrCodeOrphanCleanupFailed    = "ORPHAN_CLEANUP_FAILED"
)
```

**API startup integration (main.go or app initialization):**
```go
// Create services
orphanCleaner := service.NewOrphanCleaner(containerMgr, runRepo, logger)
timeoutEnforcer := service.NewTimeoutEnforcer(
    containerMgr,
    runRepo,
    logger,
    30*time.Minute, // default timeout
)

// Run orphan cleanup once on startup
if err := orphanCleaner.CleanupOrphans(ctx); err != nil {
    logger.Error("orphan cleanup failed", "error", err)
}

// Start timeout enforcer in background
go func() {
    if err := timeoutEnforcer.Start(ctx); err != nil && err != context.Canceled {
        logger.Error("timeout enforcer failed", "error", err)
    }
}()

// Graceful shutdown: cancel context on SIGTERM/SIGINT to stop TimeoutEnforcer
```

**Docker adapter ListContainers implementation:**
```go
package docker

import (
    "context"

    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/filters"

    "hopeitworks/backend/internal/domain/port"
    "hopeitworks/backend/pkg/errors"
)

func (m *DockerContainerManager) ListContainers(ctx context.Context, labels map[string]string) ([]port.ContainerInfo, error) {
    // Build filters
    filterArgs := filters.NewArgs()
    for key, value := range labels {
        filterArgs.Add("label", key+"="+value)
    }

    // List containers
    containers, err := m.client.ContainerList(ctx, container.ListOptions{
        All:     true, // Include stopped containers
        Filters: filterArgs,
    })
    if err != nil {
        return nil, errors.NewDomainError(
            errors.ErrCodeContainerOperationFailed,
            fmt.Sprintf("failed to list containers: %v", err),
            map[string]any{"labels": labels},
        )
    }

    // Map to ContainerInfo
    result := make([]port.ContainerInfo, 0, len(containers))
    for _, c := range containers {
        result = append(result, port.ContainerInfo{
            ID:        c.ID,
            Labels:    c.Labels,
            CreatedAt: time.Unix(c.Created, 0),
        })
    }

    return result, nil
}
```

### Testing Requirements

**Unit tests (timeout_enforcer_test.go):**
- Mock ContainerManager returns list of containers with labels
- Mock RunRepository returns run_step with started_at timestamp
- Test timeout hit: container exceeds timeout → Stop called, step/run marked failed
- Test timeout not hit: container within timeout → no Stop, no state change
- Test configurable timeout: project.max_container_timeout overrides default
- Test multiple containers: only timeout-exceeded containers are stopped
- Test error handling: Stop failure logged, does not panic, continues to next container
- No actual Docker daemon required

**Unit tests (orphan_cleaner_test.go):**
- Mock ContainerManager returns list of containers with labels
- Mock RunRepository returns run status for each run_id
- Test no orphans: all containers have active runs → no Remove calls
- Test multiple orphans: containers without active runs → all removed
- Test orphan reasons: no run_id label, run not found, run not active
- Test cleanup failure: Remove error logged, cleanup continues to next container
- Test summary logging: verify total orphans count logged via slog
- No actual Docker daemon required

**Integration testing considerations:**
- TimeoutEnforcer and OrphanCleaner can be integration-tested with testcontainers-go
- Create real containers with managed_by=hopeitworks label
- Verify timeout enforcement stops container after timeout
- Verify orphan cleanup removes containers without active runs
- Tag with `//go:build integration` if implemented

### References
- Story 3-4: Docker container lifecycle manager (ContainerManager port)
- Story 3-1: Runs/RunSteps tables (RunRepository port)
- Story 3-7: Pipeline executor (handles step failure cascading, wave 6)
- Architecture doc: `_bmad-output/planning-artifacts/architecture.md`
- Docker SDK docs: https://pkg.go.dev/github.com/docker/docker/client
- NFR12: Orphan container cleanup on API startup
- NFR14: Hard timeout per container enforced (default 30min)

## Dev Agent Record

(To be filled during implementation)

## Change Log

- 2026-02-17: Story created for Wave 6 backend infrastructure (container lifecycle management)
