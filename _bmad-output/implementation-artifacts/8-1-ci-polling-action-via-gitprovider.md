# Story 8.1: [BACK] CI Polling Action via GitProvider

Status: ready-for-dev

## Story

As a pipeline executor, I want a `ci_poll` action that polls the CI status of a merged PR, so that downstream steps only run after the CI pipeline passes.

## Acceptance Criteria (BDD)

**AC1: PR URL extraction from RunContext metadata**
- **Given** a `ci_poll` action is executed with a RunContext
- **When** `runCtx.Metadata["pr_url"]` contains a valid PR URL string
- **Then** the action uses that URL to drive CI polling
- **And** if `pr_url` is missing or empty, the action returns an error with code `CI_POLL_MISSING_PR_URL`

**AC2: Polling loop with configurable interval and timeout**
- **Given** a valid PR URL is present in metadata
- **When** the action polls `GitProvider.GetCIStatus(ctx, workDir)`
- **Then** it retries every 30 seconds by default (overridable via `Metadata["poll_interval_seconds"]`)
- **And** times out after 15 minutes by default (overridable via `Metadata["timeout_seconds"]`)
- **And** on timeout it returns an error with code `CI_POLL_TIMEOUT`

**AC3: CI pass → action succeeds**
- **Given** `GitProvider.GetCIStatus` returns `"pass"`
- **When** the polling loop receives this status
- **Then** the action returns nil (success)

**AC4: CI failure → action fails with details**
- **Given** `GitProvider.GetCIStatus` returns `"fail"`
- **When** the polling loop receives this status
- **Then** the action returns an error containing the CI failure status and the PR URL

**AC5: Progress events during polling**
- **Given** the action is polling
- **When** each polling tick fires
- **Then** an event is published via `EventPublisher` with entity_type `"ci_poll"`, action `"checking"`, and payload `{"pr_url": "...", "status": "pending"}`

**AC6: Context cancellation is respected**
- **Given** the parent context is cancelled during polling
- **When** the polling loop checks `ctx.Err()`
- **Then** the action exits immediately and returns the context error

**AC7: ActionRegistry registration**
- **Given** the `CIPollAction` is implemented
- **When** the application starts
- **Then** the action is registered in the `ActionRegistry` with name `"ci_poll"`

**AC8: Unit tests cover all branches**
- **Given** unit tests in `backend/internal/adapter/action/__tests__/ci_poll_test.go`
- **When** tests run
- **Then** happy path, timeout, CI failure, missing PR URL, and context cancellation are all tested

## Tasks / Subtasks

- [ ] [BACK] Task 1: Define `CIPollAction` struct and constructor (AC: #7)
  - [ ] Create `backend/internal/adapter/action/ci_poll.go`
  - [ ] Define `CIPollAction` struct with deps: `gitProvider port.GitProvider`, `eventPub port.EventPublisher`, `logger *slog.Logger`
  - [ ] Define `CIPollConfig` struct: `DefaultPollInterval time.Duration`, `DefaultTimeout time.Duration`
  - [ ] Implement `NewCIPollAction(gitProvider, eventPub, config, logger) *CIPollAction`
  - [ ] Implement `Name() string` returning `"ci_poll"`

- [ ] [BACK] Task 2: Implement `Execute` — PR URL extraction and config resolution (AC: #1, #2)
  - [ ] Extract `pr_url` from `runCtx.Metadata["pr_url"].(string)` — return `errors.NewValidation("pr_url", "CI_POLL_MISSING_PR_URL")` if absent
  - [ ] Parse optional `poll_interval_seconds` from metadata (default 30s)
  - [ ] Parse optional `timeout_seconds` from metadata (default 900s / 15min)
  - [ ] Create a `context.WithTimeout` derived from the parent context

- [ ] [BACK] Task 3: Implement polling loop (AC: #2, #3, #4, #5, #6)
  - [ ] Use `time.NewTicker(pollInterval)` inside a `for` loop with `select` on ticker, timeout context, and parent context
  - [ ] On each tick call `a.gitProvider.GetCIStatus(ctx, workDir)` where `workDir` is read from `runCtx.Metadata["work_dir"].(string)` with fallback `""`
  - [ ] On `"pass"` → return nil
  - [ ] On `"fail"` → return `fmt.Errorf("CI_POLL_FAILED: CI checks failed for PR %s", prURL)`
  - [ ] On `"pending"` or `"no_checks"` → publish progress event and continue
  - [ ] On ticker context done → return `errors.NewInternal("CI_POLL_TIMEOUT", fmt.Errorf("CI polling timed out after %v for PR %s", timeout, prURL))`
  - [ ] On parent context done → return `ctx.Err()`

- [ ] [BACK] Task 4: Implement progress event publishing (AC: #5)
  - [ ] Create `publishPollingEvent(ctx, runCtx, prURL, status string)` helper method
  - [ ] Build `model.Event` with `EntityType: "ci_poll"`, `Action: "checking"`, `Payload: {"pr_url": prURL, "status": status}`
  - [ ] Call `eventPub.Publish(ctx, event)` — log warn on error, do not fail polling

- [ ] [BACK] Task 5: Wire `CIPollAction` into `ActionRegistry` (AC: #7)
  - [ ] In `backend/cmd/api/wire.go`: add `NewCIPollAction` to the adapter provider set
  - [ ] Register with `actionRegistry.Register(ciPollAction)` in app initialization (follow same pattern as `AgentRunAction`)
  - [ ] Ensure `PipelineExecutor` can look it up by name `"ci_poll"`

- [ ] [BACK] Task 6: Write unit tests (AC: #8)
  - [ ] Create `backend/internal/adapter/action/__tests__/ci_poll_test.go`
  - [ ] **Test: happy path** — `GetCIStatus` returns `"pass"` on first tick, action returns nil, event published
  - [ ] **Test: pending then pass** — `GetCIStatus` returns `"pending"` twice then `"pass"`, 3 events published
  - [ ] **Test: CI failure** — `GetCIStatus` returns `"fail"`, error contains `CI_POLL_FAILED`
  - [ ] **Test: timeout** — use short timeout config (1ms), action returns timeout error
  - [ ] **Test: missing PR URL** — empty metadata, error contains `CI_POLL_MISSING_PR_URL`
  - [ ] **Test: context cancellation** — cancel context during poll, error is `context.Canceled`
  - [ ] Hand-written mocks for `GitProvider` and `EventPublisher`
  - [ ] Run `golangci-lint run ./...` — must pass

## Dev Notes

### Dependencies

**Story 3-2 (GitProvider port — DONE):** `GetCIStatus(ctx context.Context, workDir string) (string, error)` is already defined on the `GitProvider` interface. The `ci_poll` action calls this method passing `workDir` from metadata (an empty string is acceptable — the gh CLI implementation derives status from the current branch's PR).

**Story 3-7 (Pipeline executor — DONE):** `model.Action` interface with `Name() string` and `Execute(ctx, *model.RunContext) error`. `RunContext` carries `Metadata map[string]any`.

**Story 3-6 (EventPublisher — DONE):** `port.EventPublisher` with `Publish(ctx, model.Event) error`.

### Architecture Requirements

- `CIPollAction` is an adapter in `backend/internal/adapter/action/` — implements `model.Action`
- Depends only on ports: `GitProvider`, `EventPublisher`
- No imports from `api/` layer
- Use `context.WithTimeout` for the overall polling deadline — do not block indefinitely
- `time.NewTicker` for periodic polling — always call `ticker.Stop()` in a defer

### File Paths (exact)

```
backend/internal/adapter/action/ci_poll.go
backend/internal/adapter/action/__tests__/ci_poll_test.go
backend/cmd/api/wire.go                                       # Add CIPollAction provider
```

### Technical Specifications

**CIPollConfig struct:**
```go
// CIPollConfig holds tuneable parameters for CI polling.
type CIPollConfig struct {
    // DefaultPollInterval is how often to check CI status (default: 30s).
    DefaultPollInterval time.Duration
    // DefaultTimeout is the maximum time to wait for CI to pass (default: 15min).
    DefaultTimeout time.Duration
}
```

**CIPollAction struct:**
```go
// CIPollAction implements model.Action for polling CI status via GitProvider.
type CIPollAction struct {
    gitProvider port.GitProvider
    eventPub    port.EventPublisher
    config      CIPollConfig
    logger      *slog.Logger
}

func NewCIPollAction(
    gitProvider port.GitProvider,
    eventPub port.EventPublisher,
    config CIPollConfig,
    logger *slog.Logger,
) *CIPollAction {
    return &CIPollAction{
        gitProvider: gitProvider,
        eventPub:    eventPub,
        config:      config,
        logger:      logger,
    }
}

func (a *CIPollAction) Name() string { return "ci_poll" }
```

**Execute skeleton:**
```go
func (a *CIPollAction) Execute(ctx context.Context, runCtx *model.RunContext) error {
    prURL, _ := runCtx.Metadata["pr_url"].(string)
    if prURL == "" {
        return errors.NewValidation("pr_url", "missing required metadata key pr_url")
    }

    workDir, _ := runCtx.Metadata["work_dir"].(string)

    pollInterval := a.config.DefaultPollInterval
    if secs, ok := runCtx.Metadata["poll_interval_seconds"].(float64); ok && secs > 0 {
        pollInterval = time.Duration(secs) * time.Second
    }
    timeout := a.config.DefaultTimeout
    if secs, ok := runCtx.Metadata["timeout_seconds"].(float64); ok && secs > 0 {
        timeout = time.Duration(secs) * time.Second
    }

    pollCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    ticker := time.NewTicker(pollInterval)
    defer ticker.Stop()

    for {
        select {
        case <-pollCtx.Done():
            if ctx.Err() != nil {
                return ctx.Err() // parent cancelled
            }
            return fmt.Errorf("CI_POLL_TIMEOUT: timed out after %v waiting for CI on %s", timeout, prURL)
        case <-ticker.C:
            status, err := a.gitProvider.GetCIStatus(pollCtx, workDir)
            if err != nil {
                a.logger.Warn("ci_poll: GetCIStatus error", "error", err, "pr_url", prURL)
                a.publishPollingEvent(ctx, runCtx, prURL, "error")
                continue
            }
            a.publishPollingEvent(ctx, runCtx, prURL, status)
            switch status {
            case "pass":
                return nil
            case "fail":
                return fmt.Errorf("CI_POLL_FAILED: CI checks failed for PR %s", prURL)
            default:
                // "pending", "no_checks" → keep polling
            }
        }
    }
}
```

**Metadata keys (read):**
- `pr_url` (required) — the PR URL to poll; set by a preceding `git_pr` step
- `work_dir` (optional) — working directory for gh CLI; defaults to `""`
- `poll_interval_seconds` (optional) — override default 30s interval
- `timeout_seconds` (optional) — override default 900s timeout

**Error codes:**
- `CI_POLL_MISSING_PR_URL` — required metadata key absent
- `CI_POLL_TIMEOUT` — deadline exceeded before CI passed
- `CI_POLL_FAILED` — CI checks returned failure status

**Default config values:**
```go
CIPollConfig{
    DefaultPollInterval: 30 * time.Second,
    DefaultTimeout:      15 * time.Minute,
}
```

### Testing Requirements

**Mock GitProvider:**
```go
type MockGitProvider struct {
    GetCIStatusFn func(ctx context.Context, workDir string) (string, error)
    calls         int
}
func (m *MockGitProvider) GetCIStatus(ctx context.Context, workDir string) (string, error) {
    m.calls++
    return m.GetCIStatusFn(ctx, workDir)
}
// Implement remaining GitProvider methods as no-ops returning nil
```

**Mock EventPublisher:**
```go
type MockEventPublisher struct {
    Published []model.Event
}
func (m *MockEventPublisher) Publish(_ context.Context, e model.Event) error {
    m.Published = append(m.Published, e)
    return nil
}
```

Use a very short `CIPollConfig{DefaultPollInterval: 1*time.Millisecond, DefaultTimeout: 1*time.Second}` in tests to avoid slow tests.

### References

- `backend/internal/domain/port/git_provider.go` — `GetCIStatus` signature
- `backend/internal/domain/port/event_subscriber.go` — `EventPublisher` (publish side)
- `backend/internal/adapter/action/agent_run.go` — structural pattern to follow
- `backend/internal/domain/service/action_registry.go` — `Register` call pattern
- `backend/cmd/api/wire.go` — DI wiring location for new providers
- `backend/.golangci.yml` — lint configuration

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
