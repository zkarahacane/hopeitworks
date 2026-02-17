# Story 3.8: [BACK] Agent run action — compose CLAUDE.md + launch container + stream logs

Status: ready-for-dev

## Story

As a backend developer, I want an agent_run action that composes CLAUDE.md, launches a container, and streams output, so that Claude Code agents execute with proper context and real-time log capture.

## Acceptance Criteria (BDD)

**AC1: CLAUDE.md composition based on story scope**
- **Given** an agent_run action is triggered for a story with scope "backend"
- **When** the CLAUDEMDComposer composes the CLAUDE.md content
- **Then** the content is `base.md + backend.md + project.md` concatenated with newline separators
- **And** each file is read from the `agent/claude-md/` directory (path configurable)

- **Given** a story with scope "frontend"
- **When** the CLAUDEMDComposer composes the CLAUDE.md content
- **Then** the content is `base.md + frontend.md + project.md`

- **Given** a story with scope nil, empty, or "shared"
- **When** the CLAUDEMDComposer composes the CLAUDE.md content
- **Then** the content is `base.md + project.md` (no scope-specific file)

**AC2: Prompt template resolution and rendering**
- **Given** a pipeline step with action_type "implement"
- **When** the agent_run action resolves the prompt template
- **Then** it uses the TemplateService.RenderForStory with template name derived from RunStep metadata or action_type
- **And** the rendered prompt includes story context (key, title, objective, branch, repo URL, acceptance criteria, target files)

**AC3: Container creation with proper environment**
- **Given** CLAUDE.md content and rendered prompt are ready
- **When** the container is created via ContainerManager.Create
- **Then** ContainerOpts includes:
  - Image from project config or default
  - Env: `CLAUDE_MD_CONTENT`, `REPO_URL`, `BRANCH_NAME`, `STORY_KEY`, `PROMPT_CONTENT`, `GITHUB_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`
  - Labels: `managed_by=hopeitworks`, `run_id={run_id}`, `step_id={step_id}`, `story_key={story_key}`
  - NetworkName from Docker config
  - Memory and CPU limits from config defaults

**AC4: Container lifecycle — start, stream logs, wait**
- **Given** the container is created successfully
- **When** the action starts the container
- **Then** ContainerManager.Start is called with the container ID
- **And** the run step's container_id is persisted via a new UpdateRunStepContainerID method on RunRepository
- **And** LogStreamer.StreamLogs is called to begin streaming
- **And** log events are forwarded to EventPublisher in a goroutine
- **And** ContainerManager.Wait blocks until the container exits

**AC5: Successful agent completion (exit code 0)**
- **Given** the container exits with code 0
- **When** the action processes the result
- **Then** the action returns nil (success)
- **And** the container is removed via ContainerManager.Remove
- **And** log streaming goroutine has completed

**AC6: Agent failure (non-zero exit code)**
- **Given** the container exits with non-zero exit code (e.g., 1)
- **When** the action processes the result
- **Then** the action returns an error with message including exit code and log tail
- **And** the run step's log_tail is persisted (last N lines from log events)
- **And** the container is removed via ContainerManager.Remove

**AC7: Container cleanup on all paths**
- **Given** the container was created
- **When** the action completes (success, failure, or context cancellation)
- **Then** the container is always removed (deferred cleanup)
- **And** cleanup errors are logged but do not override the primary error

**AC8: ActionRegistry wiring**
- **Given** the AgentRunAction is implemented
- **When** the application starts
- **Then** the action is registered in the ActionRegistry with name "agent_run"
- **And** the PipelineExecutor can look it up by name

**AC9: Unit tests verify all behaviors**
- **Given** unit tests in `backend/internal/adapter/action/__tests__/agent_run_test.go`
- **When** tests are executed
- **Then** happy path test verifies: story fetch, CLAUDE.md compose, prompt render, container create/start/wait/remove, log streaming
- **And** failure test verifies: non-zero exit code returns error with log tail
- **And** scope test verifies: backend/frontend/shared scopes compose correct CLAUDE.md
- **And** cancellation test verifies: context cancellation stops container and cleans up
- **And** all tests use mock ports (ContainerManager, LogStreamer, EventPublisher, StoryRepository, TemplateService)

## Tasks / Subtasks

- [ ] [BACK] Task 1: Extend RunRepository port with container_id and log_tail update support (AC: #4, #6)
  - [ ] Add `UpdateRunStepContainerInfo(ctx, id, containerID *string, logTail *string) (*model.RunStep, error)` to `port.RunRepository`
  - [ ] Implement in `backend/internal/adapter/postgres/run_repo.go` using existing sqlc `UpdateRunStepStatus` query (pass container_id and log_tail via the existing COALESCE columns, keeping status unchanged — or add a dedicated sqlc query `UpdateRunStepContainerInfo`)
  - [ ] Add unit test for the new method

- [ ] [BACK] Task 2: Implement CLAUDEMDComposer helper (AC: #1)
  - [ ] Create `backend/internal/adapter/action/claude_md_composer.go`
  - [ ] Define `CLAUDEMDComposer` struct with `basePath string` (path to `agent/claude-md/` directory)
  - [ ] Implement `NewCLAUDEMDComposer(basePath string) *CLAUDEMDComposer`
  - [ ] Implement `Compose(scope string) (string, error)` method
  - [ ] Read `base.md`, scope-specific file (`backend.md` or `frontend.md`), and `project.md`
  - [ ] Concatenate with `\n\n` separator
  - [ ] Handle missing scope (nil/empty/"shared") by skipping scope-specific file
  - [ ] Return actionable errors with file path context

- [ ] [BACK] Task 3: Implement AgentRunAction struct and constructor (AC: #3, #8)
  - [ ] Create `backend/internal/adapter/action/agent_run.go`
  - [ ] Define `AgentRunAction` struct with dependencies: `ContainerManager`, `LogStreamer`, `EventPublisher`, `StoryRepository`, `ProjectRepository`, `TemplateService`, `CLAUDEMDComposer`, `logger`, `agentConfig`
  - [ ] Define `AgentConfig` struct: `DefaultImage string`, `DefaultMemory int64`, `DefaultCPUs float64`, `NetworkName string`, `LogTailLines int`
  - [ ] Implement `NewAgentRunAction(...)` constructor
  - [ ] Implement `Name() string` returning `"agent_run"`

- [ ] [BACK] Task 4: Implement Execute — story fetch, CLAUDE.md composition, prompt rendering (AC: #1, #2)
  - [ ] In `Execute(ctx, runCtx)`: fetch story from StoryRepository by `runCtx.StoryID`
  - [ ] Fetch project from ProjectRepository by `runCtx.ProjectID`
  - [ ] Determine story scope from `story.Scope` (default to empty if nil)
  - [ ] Call `CLAUDEMDComposer.Compose(scope)` to build CLAUDE.md content
  - [ ] Determine template name from RunStep.Action or RunContext.Metadata `"template_name"` key (default: `"implement"`)
  - [ ] Build `model.TemplateContext` from story fields and project fields
  - [ ] Determine branch name from `runCtx.Metadata["branch_name"]` (set by a previous git_create_branch step)
  - [ ] Call `TemplateService.RenderForStory(ctx, projectID, templateName, tmplCtx)` to get rendered prompt

- [ ] [BACK] Task 5: Implement Execute — container creation and start (AC: #3, #4)
  - [ ] Build `model.ContainerOpts` with:
    - `Image`: from project config `DefaultModel` or `agentConfig.DefaultImage`
    - `Env`: `CLAUDE_MD_CONTENT=<composed>`, `REPO_URL=<project.RepoURL>`, `BRANCH_NAME=<branch>`, `STORY_KEY=<story.Key>`, `PROMPT_CONTENT=<rendered>`, `GITHUB_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` from `os.Getenv`
    - `Labels`: `managed_by=hopeitworks`, `run_id=<runID>`, `step_id=<stepID>`, `story_key=<storyKey>`
    - `NetworkName`: from Docker config
    - `Memory` and `CPUs`: from agentConfig defaults
  - [ ] Call `ContainerManager.Create(ctx, opts)` to get container ID
  - [ ] Defer `cleanupContainer(ctx, containerID)` for guaranteed cleanup
  - [ ] Call `ContainerManager.Start(ctx, containerID)`
  - [ ] Persist container ID to run step via `RunRepository.UpdateRunStepContainerInfo`

- [ ] [BACK] Task 6: Implement Execute — log streaming and event forwarding (AC: #4, #5, #6)
  - [ ] Call `LogStreamer.StreamLogs(ctx, containerID, runID, stepID)` to get log and done channels
  - [ ] Start goroutine to consume log channel:
    - Collect last N log lines in a ring buffer (for log tail on failure)
    - For each `LogEvent`, publish as event via `EventPublisher` with entity_type `"log"`, entity_id as step_id, action `"emitted"`, payload containing the log event data
  - [ ] Call `ContainerManager.Wait(ctx, containerID)` to get exit code
  - [ ] Wait for log goroutine to finish (use sync.WaitGroup or channel)
  - [ ] If exit code == 0: return nil
  - [ ] If exit code != 0: persist log tail to run step, return error with exit code and log tail summary

- [ ] [BACK] Task 7: Implement container cleanup helper (AC: #7)
  - [ ] Create `cleanupContainer(containerID string)` method on AgentRunAction
  - [ ] Call `ContainerManager.Stop(ctx, containerID)` (ignore "already stopped" errors)
  - [ ] Call `ContainerManager.Remove(ctx, containerID)`
  - [ ] Log errors at warn level without failing

- [ ] [BACK] Task 8: Wire AgentRunAction into ActionRegistry (AC: #8)
  - [ ] In `backend/cmd/api/wire.go` or a dedicated provider file:
    - Add `AgentRunAction` to the adapter provider set
    - Register it in the `ActionRegistry` during app initialization
  - [ ] Alternatively, create `backend/internal/adapter/action/registry.go` with a `RegisterActions(reg port.ActionRegistry, actions ...model.Action)` helper
  - [ ] Ensure the action is available when `PipelineExecutor` calls `actionReg.Get("agent_run")`

- [ ] [BACK] Task 9: Write unit tests (AC: #9)
  - [ ] Create `backend/internal/adapter/action/__tests__/agent_run_test.go`
  - [ ] **Test: happy path** — story fetched, CLAUDE.md composed (backend scope), prompt rendered, container created/started, logs streamed, exit code 0, container removed
  - [ ] **Test: frontend scope** — CLAUDE.md composed with frontend.md
  - [ ] **Test: shared/nil scope** — CLAUDE.md composed with base.md + project.md only
  - [ ] **Test: agent failure** — exit code 1, error returned with log tail, container removed
  - [ ] **Test: container create failure** — returns error, no start/wait called
  - [ ] **Test: context cancellation** — container stopped and removed
  - [ ] **Test: template not found** — returns TEMPLATE_NOT_FOUND error
  - [ ] **Test: story not found** — returns STORY_NOT_FOUND error
  - [ ] **Test: CLAUDEMDComposer** — separate tests for scope composition logic (can be in `claude_md_composer_test.go`)
  - [ ] All mocks hand-written, following project patterns
  - [ ] Run `golangci-lint run ./...` — must pass

## Dev Notes

### Dependencies

**Story 3-4 (Docker container lifecycle manager) - DONE:** Provides `ContainerManager` port with Create, Start, Stop, Remove, Wait, ListContainers. The AgentRunAction uses Create, Start, Wait, Stop, Remove.

**Story 3-5 (NDJSON log streaming) - DONE:** Provides `LogStreamer` port with StreamLogs returning log channel and done channel. The AgentRunAction consumes log events and forwards them.

**Story 3-6 (Events table + pgxlisten event bus) - DONE:** Provides `EventPublisher` port. Log events are published as system events for SSE forwarding.

**Story 3-7 (Pipeline executor sequential step runner) - DONE:** Provides `Action` interface, `RunContext`, `PipelineExecutor`, and `ActionRegistry` port. The AgentRunAction implements the Action interface.

**Story 6-3 (Handlebars rendering engine + default template seeding) - DONE:** Provides `TemplateRenderer` port and `TemplateService` for resolving and rendering prompt templates from DB with fallback to defaults.

### Architecture Requirements

**Hexagonal architecture:**
- `AgentRunAction` is an adapter in `backend/internal/adapter/action/` — it implements `model.Action` (domain interface)
- It depends on ports: `ContainerManager`, `LogStreamer`, `EventPublisher`, `StoryRepository`, `ProjectRepository`, `RunRepository`, and the `TemplateService` (domain service)
- NO imports from `api/` layer
- Import direction: `action (adapter) -> port (domain) <- concrete adapters`

**CLAUDE.md composition:**
- Files read from filesystem at action execution time (not cached)
- Composition rule: `base.md + (backend.md | frontend.md | nothing) + project.md`
- The `basePath` for CLAUDE.md templates is configurable (defaults to `agent/claude-md/`)

**Container environment:**
- `GITHUB_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` are read from the backend process's environment via `os.Getenv`
- These are NOT stored in the database — they are runtime secrets
- The ScrubHandler in slog will redact these from logs automatically

**Log tail ring buffer:**
- Keep last N lines (configurable, default 50) for error context
- On agent failure, persist the log tail to `run_steps.log_tail` column
- The log tail is included in the error message returned to PipelineExecutor

### File Paths (exact)

```
backend/internal/adapter/action/agent_run.go              # AgentRunAction implementation
backend/internal/adapter/action/claude_md_composer.go      # CLAUDEMDComposer helper
backend/internal/adapter/action/__tests__/agent_run_test.go          # Unit tests for AgentRunAction
backend/internal/adapter/action/__tests__/claude_md_composer_test.go # Unit tests for CLAUDEMDComposer
backend/internal/domain/port/run_repository.go             # Extended with UpdateRunStepContainerInfo
backend/internal/adapter/postgres/run_repo.go              # Implementation of new method
backend/queries/run_steps.sql                              # New sqlc query (if needed)
agent/claude-md/base.md                                    # READ ONLY — base instructions
agent/claude-md/backend.md                                 # READ ONLY — backend instructions
agent/claude-md/frontend.md                                # READ ONLY — frontend instructions
agent/claude-md/project.md                                 # READ ONLY — project context
```

### Technical Specifications

**AgentConfig struct:**
```go
// backend/internal/adapter/action/agent_run.go
package action

// AgentConfig holds configuration for agent container execution.
type AgentConfig struct {
    // DefaultImage is the Docker image for agent containers (e.g., "hopeitworks/agent:latest").
    DefaultImage string
    // DefaultMemory is the memory limit in bytes (e.g., 4GB = 4294967296).
    DefaultMemory int64
    // DefaultCPUs is the CPU limit (e.g., 2.0).
    DefaultCPUs float64
    // NetworkName is the Docker network for agent containers.
    NetworkName string
    // LogTailLines is the number of log lines to keep for error context.
    LogTailLines int
    // ClaudeMDPath is the path to the agent/claude-md/ directory.
    ClaudeMDPath string
}
```

**AgentRunAction struct:**
```go
// AgentRunAction implements model.Action for running Claude Code agents in containers.
type AgentRunAction struct {
    containerMgr port.ContainerManager
    logStreamer   port.LogStreamer
    eventPub     port.EventPublisher
    storyRepo    port.StoryRepository
    projectRepo  port.ProjectRepository
    runRepo      port.RunRepository
    templateSvc  *service.TemplateService
    composer     *CLAUDEMDComposer
    config       AgentConfig
    logger       *slog.Logger
}

// NewAgentRunAction creates a new agent run action.
func NewAgentRunAction(
    containerMgr port.ContainerManager,
    logStreamer port.LogStreamer,
    eventPub port.EventPublisher,
    storyRepo port.StoryRepository,
    projectRepo port.ProjectRepository,
    runRepo port.RunRepository,
    templateSvc *service.TemplateService,
    config AgentConfig,
    logger *slog.Logger,
) *AgentRunAction {
    return &AgentRunAction{
        containerMgr: containerMgr,
        logStreamer:   logStreamer,
        eventPub:     eventPub,
        storyRepo:    storyRepo,
        projectRepo:  projectRepo,
        runRepo:      runRepo,
        templateSvc:  templateSvc,
        composer:     NewCLAUDEMDComposer(config.ClaudeMDPath),
        config:       config,
        logger:       logger,
    }
}

// Name returns the action identifier.
func (a *AgentRunAction) Name() string {
    return "agent_run"
}
```

**Execute method (high-level flow):**
```go
func (a *AgentRunAction) Execute(ctx context.Context, runCtx *model.RunContext) error {
    // 1. Fetch story
    story, err := a.storyRepo.GetByID(ctx, runCtx.StoryID)
    if err != nil {
        return fmt.Errorf("fetch story: %w", err)
    }

    // 2. Fetch project
    project, err := a.projectRepo.GetByID(ctx, runCtx.ProjectID)
    if err != nil {
        return fmt.Errorf("fetch project: %w", err)
    }

    // 3. Compose CLAUDE.md
    scope := ""
    if story.Scope != nil {
        scope = *story.Scope
    }
    claudeMD, err := a.composer.Compose(scope)
    if err != nil {
        return fmt.Errorf("compose CLAUDE.md: %w", err)
    }

    // 4. Resolve and render prompt template
    templateName := a.resolveTemplateName(runCtx)
    branchName, _ := runCtx.Metadata["branch_name"].(string)
    repoURL := ""
    if project.RepoURL != nil {
        repoURL = *project.RepoURL
    }

    tmplCtx := &model.TemplateContext{
        StoryKey:           story.Key,
        StoryTitle:         story.Title,
        StoryObjective:     derefString(story.Objective),
        TargetFiles:        story.TargetFiles,
        AcceptanceCriteria: derefString(story.AcceptanceCriteria),
        BranchName:         branchName,
        RepoURL:            repoURL,
    }
    prompt, err := a.templateSvc.RenderForStory(ctx, runCtx.ProjectID, templateName, tmplCtx)
    if err != nil {
        return fmt.Errorf("render prompt template %q: %w", templateName, err)
    }

    // 5. Create container
    containerID, err := a.createContainer(ctx, runCtx, project, story, claudeMD, prompt, branchName)
    if err != nil {
        return fmt.Errorf("create container: %w", err)
    }
    defer a.cleanupContainer(containerID)

    // 6. Start container
    if err := a.containerMgr.Start(ctx, containerID); err != nil {
        return fmt.Errorf("start container: %w", err)
    }

    // 7. Persist container ID to run step
    a.persistContainerID(ctx, runCtx.RunStep.ID, containerID)

    // 8. Stream logs + wait for exit
    exitCode, err := a.streamAndWait(ctx, containerID, runCtx)
    if err != nil {
        return fmt.Errorf("stream/wait: %w", err)
    }

    // 9. Check exit code
    if exitCode != 0 {
        return fmt.Errorf("agent exited with code %d", exitCode)
    }

    return nil
}
```

**CLAUDEMDComposer:**
```go
// backend/internal/adapter/action/claude_md_composer.go
package action

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

// CLAUDEMDComposer reads and concatenates CLAUDE.md template files
// based on the story scope.
type CLAUDEMDComposer struct {
    basePath string
}

// NewCLAUDEMDComposer creates a new composer with the given base path.
func NewCLAUDEMDComposer(basePath string) *CLAUDEMDComposer {
    return &CLAUDEMDComposer{basePath: basePath}
}

// Compose builds the CLAUDE.md content for the given scope.
// Composition rule: base.md + (backend.md | frontend.md | nothing) + project.md
func (c *CLAUDEMDComposer) Compose(scope string) (string, error) {
    files := []string{"base.md"}

    switch strings.ToLower(scope) {
    case "backend":
        files = append(files, "backend.md")
    case "frontend":
        files = append(files, "frontend.md")
    // "shared", "", or any other value: no scope-specific file
    }

    files = append(files, "project.md")

    var parts []string
    for _, f := range files {
        path := filepath.Join(c.basePath, f)
        content, err := os.ReadFile(path)
        if err != nil {
            return "", fmt.Errorf("read CLAUDE.md template %q: %w", path, err)
        }
        parts = append(parts, strings.TrimSpace(string(content)))
    }

    return strings.Join(parts, "\n\n"), nil
}
```

**Container creation:**
```go
func (a *AgentRunAction) createContainer(
    ctx context.Context,
    runCtx *model.RunContext,
    project *model.Project,
    story *model.Story,
    claudeMD, prompt, branchName string,
) (string, error) {
    repoURL := ""
    if project.RepoURL != nil {
        repoURL = *project.RepoURL
    }

    opts := model.ContainerOpts{
        Image:       a.config.DefaultImage,
        NetworkName: a.config.NetworkName,
        Memory:      a.config.DefaultMemory,
        CPUs:        a.config.DefaultCPUs,
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

    return a.containerMgr.Create(ctx, opts)
}
```

**Log streaming and wait:**
```go
func (a *AgentRunAction) streamAndWait(
    ctx context.Context,
    containerID string,
    runCtx *model.RunContext,
) (int, error) {
    runID := runCtx.Run.ID.String()
    stepID := runCtx.RunStep.ID.String()

    logCh, doneCh, err := a.logStreamer.StreamLogs(ctx, containerID, runID, stepID)
    if err != nil {
        return -1, fmt.Errorf("start log streaming: %w", err)
    }

    // Ring buffer for log tail
    tailSize := a.config.LogTailLines
    if tailSize <= 0 {
        tailSize = 50
    }
    logTail := make([]string, 0, tailSize)

    // Consume logs in goroutine
    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
        for logEvent := range logCh {
            // Maintain ring buffer
            if len(logTail) >= tailSize {
                logTail = logTail[1:]
            }
            logTail = append(logTail, logEvent.Message)

            // Forward to event system
            a.publishLogEvent(ctx, runCtx, logEvent)
        }
    }()

    // Wait for container exit
    exitCode := <-doneCh

    // Wait for log goroutine to finish
    wg.Wait()

    // On failure, persist log tail
    if exitCode != 0 {
        tail := strings.Join(logTail, "\n")
        a.persistLogTail(ctx, runCtx.RunStep.ID, tail)
    }

    return exitCode, nil
}
```

**Cleanup helper:**
```go
func (a *AgentRunAction) cleanupContainer(containerID string) {
    ctx := context.Background()

    if err := a.containerMgr.Stop(ctx, containerID); err != nil {
        a.logger.Warn("failed to stop container during cleanup",
            "container_id", containerID, "error", err)
    }

    if err := a.containerMgr.Remove(ctx, containerID); err != nil {
        a.logger.Warn("failed to remove container during cleanup",
            "container_id", containerID, "error", err)
    }

    a.logger.Debug("container cleaned up", "container_id", containerID)
}
```

**RunRepository extension (port):**
```go
// Add to port.RunRepository interface:
UpdateRunStepContainerInfo(ctx context.Context, id uuid.UUID, containerID *string, logTail *string) (*model.RunStep, error)
```

**RunRepository extension (adapter implementation):**
```go
// In backend/internal/adapter/postgres/run_repo.go:
func (r *RunRepo) UpdateRunStepContainerInfo(ctx context.Context, id uuid.UUID, containerID *string, logTail *string) (*model.RunStep, error) {
    // Reuse the UpdateRunStepStatus query with status unchanged
    // We need a dedicated query that only updates container_id and log_tail
    // without changing the status field.
    step, err := r.queries.GetRunStep(ctx, id)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, apperrors.NewNotFound("run step", id)
        }
        return nil, apperrors.NewInternal("failed to get run step", err)
    }

    params := UpdateRunStepStatusParams{
        ID:     id,
        Status: step.Status, // Keep existing status
    }
    if containerID != nil {
        params.ContainerID = pgtype.Text{String: *containerID, Valid: true}
    }
    if logTail != nil {
        params.LogTail = pgtype.Text{String: *logTail, Valid: true}
    }

    row, err := r.queries.UpdateRunStepStatus(ctx, params)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, apperrors.NewNotFound("run step", id)
        }
        return nil, apperrors.NewInternal("failed to update run step container info", err)
    }
    return toDomainRunStep(row), nil
}
```

**Template name resolution:**
```go
func (a *AgentRunAction) resolveTemplateName(runCtx *model.RunContext) string {
    // Check metadata for explicit template name override
    if name, ok := runCtx.Metadata["template_name"].(string); ok && name != "" {
        return name
    }
    // Default to "implement" for agent_run action
    return service.TemplateNameImplement
}
```

**Event publishing for log events:**
```go
func (a *AgentRunAction) publishLogEvent(ctx context.Context, runCtx *model.RunContext, logEvent model.LogEvent) {
    payload, err := json.Marshal(logEvent)
    if err != nil {
        a.logger.Error("failed to marshal log event", "error", err)
        return
    }

    event := model.Event{
        ID:         uuid.New(),
        ProjectID:  runCtx.ProjectID,
        EntityType: "log",
        EntityID:   runCtx.RunStep.ID,
        Action:     "emitted",
        Payload:    payload,
    }

    if err := a.eventPub.Publish(ctx, event); err != nil {
        a.logger.Warn("failed to publish log event", "error", err)
    }
}
```

**Error codes:**
- `STORY_NOT_FOUND` — story does not exist (from StoryRepository)
- `PROJECT_NOT_FOUND` — project does not exist (from ProjectRepository)
- `TEMPLATE_NOT_FOUND` — prompt template not found in DB and no default exists
- `CONTAINER_CREATE_FAILED` — Docker container creation failed
- `CONTAINER_START_FAILED` — Docker container start failed
- `AGENT_FAILED` — agent exited with non-zero exit code (includes log tail)

**Metadata keys used:**
- `branch_name` (read) — set by a previous `git_create_branch` action step
- `template_name` (read, optional) — override the default template name
- `container_id` (write) — set by this action for downstream steps

### Testing Requirements

**Unit tests (backend/internal/adapter/action/__tests__/agent_run_test.go):**

1. **Happy path test:**
   - Mock StoryRepository returns a story with scope "backend"
   - Mock ProjectRepository returns a project with RepoURL
   - Mock TemplateService renders prompt successfully
   - Mock ContainerManager: Create returns ID, Start succeeds, Wait returns exit code 0
   - Mock LogStreamer: StreamLogs returns channels, sends 3 log events then closes
   - Mock EventPublisher: captures published events
   - Mock RunRepository: UpdateRunStepContainerInfo succeeds
   - Verify: container removed after execution, no error returned
   - Verify: container opts include correct env vars and labels

2. **Frontend scope test:**
   - Story scope = "frontend"
   - Verify CLAUDEMDComposer reads base.md + frontend.md + project.md

3. **Shared/nil scope test:**
   - Story scope = nil or "shared"
   - Verify CLAUDEMDComposer reads base.md + project.md only

4. **Agent failure test (exit code 1):**
   - ContainerManager.Wait returns exit code 1
   - LogStreamer sends 10 log events
   - Verify error returned contains "exited with code 1"
   - Verify RunRepository.UpdateRunStepContainerInfo called with log tail
   - Verify container removed

5. **Container create failure test:**
   - ContainerManager.Create returns error
   - Verify: Start and Wait NOT called
   - Verify: error wraps "create container"

6. **Context cancellation test:**
   - Cancel context during Wait
   - Verify: container cleanup called
   - Verify: error is context.Canceled

7. **Template not found test:**
   - TemplateService.RenderForStory returns TEMPLATE_NOT_FOUND
   - Verify: no container created
   - Verify: error wraps "render prompt template"

8. **Story not found test:**
   - StoryRepository.GetByID returns not found
   - Verify: no container created
   - Verify: error wraps "fetch story"

**CLAUDEMDComposer tests (backend/internal/adapter/action/__tests__/claude_md_composer_test.go):**

1. Test backend scope reads base.md + backend.md + project.md
2. Test frontend scope reads base.md + frontend.md + project.md
3. Test shared scope reads base.md + project.md
4. Test empty scope reads base.md + project.md
5. Test missing base.md returns error with file path
6. Use `t.TempDir()` to create temporary CLAUDE.md files for testing

**Mock patterns:**
```go
type MockContainerManager struct {
    CreateFn func(ctx context.Context, opts model.ContainerOpts) (string, error)
    StartFn  func(ctx context.Context, containerID string) error
    StopFn   func(ctx context.Context, containerID string) error
    RemoveFn func(ctx context.Context, containerID string) error
    WaitFn   func(ctx context.Context, containerID string) (int, error)
    ListContainersFn func(ctx context.Context, labels map[string]string) ([]port.ContainerInfo, error)

    CreateCalls []model.ContainerOpts // Track calls for assertions
}
```

**Linting:**
- Run `golangci-lint run ./...` from `backend/` — must pass before commit
- Rename unused mock parameters to `_`
- Use structured slog logging, not fmt.Println

### References

- Story 3-4: Docker container lifecycle manager (ContainerManager port + adapter)
- Story 3-5: NDJSON log streaming from container (LogStreamer port + adapter)
- Story 3-6: Events table + pgxlisten event bus (EventPublisher port + adapter)
- Story 3-7: Pipeline executor sequential step runner (Action interface, RunContext, PipelineExecutor)
- Story 6-3: Handlebars rendering engine + default template seeding (TemplateRenderer, TemplateService)
- `agent/claude-md/README.md`: Composition rules for CLAUDE.md files
- `backend/.golangci.yml`: Linting configuration
- `backend/pkg/errors/errors.go`: DomainError implementation
- `backend/internal/domain/service/template_service.go`: TemplateService (RenderForStory)
- `backend/internal/domain/service/pipeline_executor.go`: PipelineExecutor (consumer of Action interface)
- `backend/internal/domain/port/run_repository.go`: RunRepository port (to be extended)
- `backend/queries/run_steps.sql`: sqlc queries (container_id and log_tail columns exist)

## Dev Agent Record

(To be filled during implementation)

## Change Log

- 2026-02-17: Story created for Wave 7 agent runtime execution
