# Story 7.1: [BACK] DAG Builder + Topological Sort

Status: ready-for-dev

## Story

As a platform engineer, I want a DAG builder service that computes execution layers for an epic's stories using topological sort, So that the scheduler can run independent stories in parallel and respect declared and inferred dependencies.

## Acceptance Criteria (BDD)

**AC1: Linear dependency chain produces sequential layers**
- **Given** stories S-01, S-02, S-03 where S-02 depends on S-01 and S-03 depends on S-02
- **When** BuildDAG is called with these stories
- **Then** the result contains 3 layers: [[S-01], [S-02], [S-03]]

**AC2: Independent stories are grouped in the same layer**
- **Given** stories S-01, S-02, S-03 where S-02 and S-03 both depend on S-01 but not on each other
- **When** BuildDAG is called
- **Then** the result contains 2 layers: [[S-01], [S-02, S-03]]

**AC3: Cycle detection returns DAG_CYCLE_DETECTED error**
- **Given** stories S-01 and S-02 where S-01 depends on S-02 and S-02 depends on S-01
- **When** BuildDAG is called
- **Then** a DomainError with code DAG_CYCLE_DETECTED is returned

**AC4: File conflict creates implicit sequential ordering**
- **Given** stories S-01 and S-02 with no explicit DependsOn but sharing a common target file
- **When** BuildDAG is called
- **Then** the result places them in separate layers (S-01 before S-02, stable sort by key)

**AC5: Empty input returns empty result**
- **Given** an empty slice of stories
- **When** BuildDAG is called
- **Then** an empty DAGResult (zero groups) is returned without error

**AC6: Unknown dependency key is silently ignored**
- **Given** story S-02 has DependsOn containing key "S-99" which is not in the input slice
- **When** BuildDAG is called
- **Then** S-99 is ignored and S-02 is treated as having no dependency on it (no error)

**AC7: GET /api/v1/projects/{projectId}/epics/{epicId}/dag returns node/edge graph**
- **Given** an authenticated user with access to the project
- **When** GET /api/v1/projects/{projectId}/epics/{epicId}/dag is called
- **Then** the response is 200 with `{ nodes: [...], edges: [...] }` where each node has id, key, title, status, layer; each edge has source and target story keys

## Tasks / Subtasks

- [ ] [BACK] Task 1: Add DAGResult model type (AC: #1, #2, #4, #5)
  - [ ] Create `backend/internal/domain/model/dag.go`
  - [ ] Define `DAGResult` struct: `Groups [][]Story`
  - [ ] Define `DAGNode` struct: `Story Story; Layer int`
  - [ ] Export constants: no new constants needed — error code is string `"DAG_CYCLE_DETECTED"`

- [ ] [BACK] Task 2: Implement SchedulerService with BuildDAG (AC: #1, #2, #3, #5, #6)
  - [ ] Create `backend/internal/domain/service/scheduler_service.go`
  - [ ] Implement Kahn's algorithm: build in-degree map + adjacency list from explicit DependsOn fields
  - [ ] Resolve DependsOn keys to story keys from the input slice only — skip unknown keys (AC6)
  - [ ] Return `(DAGResult, error)` — error is `*errors.DomainError` with code `DAG_CYCLE_DETECTED` when cycle found
  - [ ] Empty input returns `DAGResult{Groups: [][]model.Story{}}`, nil

- [ ] [BACK] Task 3: Add file conflict detection as implicit dependency edges (AC: #4)
  - [ ] Before running Kahn's, compute overlap: for each pair of stories, if their TargetFiles sets intersect, add an implicit directed edge (lower-key story → higher-key story, sorted lexicographically)
  - [ ] Merge implicit edges with explicit DependsOn edges before topological sort
  - [ ] Implicit edges must not introduce phantom nodes — only add edge if both stories are in the input

- [ ] [BACK] Task 4: Write unit tests for SchedulerService (AC: #1–#6)
  - [ ] File: `backend/internal/domain/service/scheduler_service_test.go`
  - [ ] Table-driven tests covering: linear chain, diamond (fan-out/fan-in), all independent, cycle of 2, cycle of 3, empty input, unknown dep key ignored, file conflict implicit edge, combined explicit + implicit deps
  - [ ] Use `testutil.NewStory(testutil.WithKey("S-01"), testutil.WithDeps("S-02"))` factory pattern

- [ ] [BACK] Task 5: Add GET /epics/{epicId}/dag endpoint to OpenAPI spec (AC: #7)
  - [ ] Update `api/openapi.yaml`: add path `/projects/{projectId}/epics/{epicId}/dag` with operationId `getEpicDAG`
  - [ ] Add response schema `EpicDAGResponse`: `{ nodes: EpicDAGNode[], edges: EpicDAGEdge[] }`
  - [ ] `EpicDAGNode`: `{ id: string, key: string, title: string, status: string, layer: integer }`
  - [ ] `EpicDAGEdge`: `{ source: string, target: string }` (source/target are story keys)
  - [ ] Also add stub for POST `/projects/{projectId}/epics/{epicId}/runs` with operationId `launchEpicRun` (202 response, body `{ epic_run_id: string, status: string, stories_count: integer }`) — implementation deferred to wave 11

- [ ] [BACK] Task 6: Regenerate backend types from OpenAPI spec (AC: #7)
  - [ ] Run `cd backend && make generate` to regenerate `internal/api/handler/gen_server.go`
  - [ ] Verify the new `GetEpicDAG` and `LaunchEpicRun` interface methods appear in `gen_server.go`
  - [ ] No manual edits to generated files

- [ ] [BACK] Task 7: Implement GetEpicDAG handler (AC: #7)
  - [ ] Add `GetEpicDAG` method to `EpicHandler` in `backend/internal/api/handler/epic_handler.go`
  - [ ] Handler calls `storyRepo.ListByEpic(ctx, epicID, limit=500, offset=0)` to get all stories
  - [ ] Calls `schedulerSvc.BuildDAG(stories)` — if cycle error, returns 422 with DAG_CYCLE_DETECTED code
  - [ ] Transforms DAGResult into flat `EpicDAGResponse`: iterate groups, assign layer index, collect edges from DependsOn + file conflicts
  - [ ] Wire `EpicHandler` to accept `*service.SchedulerService` as second constructor param; update `NewEpicHandler`
  - [ ] Register `GetEpicDAG` delegation in `server.go`

- [ ] [BACK] Task 8: Update DI wiring for SchedulerService (AC: #7)
  - [ ] Add `NewSchedulerService` to `wire.go` provider set in `backend/cmd/api/`
  - [ ] Pass `SchedulerService` to `NewEpicHandler`
  - [ ] Run `cd backend && wire ./cmd/api/` to regenerate `wire_gen.go`
  - [ ] Verify the app compiles: `cd backend && go build ./...`

- [ ] [BACK] Task 9: Write handler test for GetEpicDAG (AC: #7)
  - [ ] File: `backend/internal/api/handler/epic_handler_test.go` (extend existing)
  - [ ] Test: stories with no deps → all in layer 0, zero edges
  - [ ] Test: stories with deps → correct layers and edges returned
  - [ ] Test: cycle in stories → 422 with DAG_CYCLE_DETECTED error code
  - [ ] Test: epic not found → 404 forwarded from storyRepo mock
  - [ ] Use hand-written mock for `port.StoryRepository` and a real `SchedulerService`

- [ ] [BACK] Task 10: Run lint and unit tests (AC: all)
  - [ ] `cd backend && golangci-lint run ./...` — must pass with zero errors
  - [ ] `cd backend && go test ./... -short` — all unit tests green
  - [ ] Fix any errcheck, revive, or goimports violations before committing

## Dev Notes

### Dependencies

- Story 2-2: Stories table with `depends_on` and `target_files` columns — DONE (model.Story already has DependsOn []string and TargetFiles []string)
- Story 2-3: ListByEpic query already available in `port.StoryRepository`
- No new DB migrations required — pure computation service

### Architecture Requirements

The `SchedulerService` is a pure domain service with no port dependencies. It takes `[]model.Story` and returns `(model.DAGResult, error)`. It has no DB access — the handler fetches stories and passes them in.

```
EpicHandler
  ├── EpicService (existing — unchanged)
  ├── SchedulerService (new — pure computation)
  └── StoryRepository (added to handler for GetEpicDAG)
```

`SchedulerService` signature:

```go
type SchedulerService struct{}

func NewSchedulerService() *SchedulerService {
    return &SchedulerService{}
}

// BuildDAG computes topological execution layers for the given stories.
// Returns DAGResult with Groups where each group can run in parallel.
// Returns DAG_CYCLE_DETECTED DomainError if a cycle is found.
func (s *SchedulerService) BuildDAG(stories []model.Story) (model.DAGResult, error)
```

### File Paths (exact)

```
api/openapi.yaml                                                        (add dag + epic run endpoints)
backend/internal/domain/model/dag.go                                    (new)
backend/internal/domain/service/scheduler_service.go                    (new)
backend/internal/domain/service/scheduler_service_test.go               (new)
backend/internal/api/handler/epic_handler.go                            (add GetEpicDAG method, update constructor)
backend/internal/api/handler/server.go                                  (add GetEpicDAG + LaunchEpicRun delegation)
backend/internal/api/handler/epic_handler_test.go                       (extend with GetEpicDAG tests)
backend/internal/api/handler/gen_server.go                              (regenerated — do not edit manually)
backend/cmd/api/wire.go                                                 (add SchedulerService provider)
backend/cmd/api/wire_gen.go                                             (regenerated — do not edit manually)
```

### Technical Specifications

**model/dag.go:**
```go
package model

// DAGResult holds the result of a topological sort on stories.
// Groups are execution layers: all stories in Groups[i] can run concurrently,
// and all must complete before any story in Groups[i+1] starts.
type DAGResult struct {
    Groups [][]Story
}
```

**Kahn's algorithm implementation sketch:**
```go
func (s *SchedulerService) BuildDAG(stories []model.Story) (model.DAGResult, error) {
    if len(stories) == 0 {
        return model.DAGResult{Groups: [][]model.Story{}}, nil
    }

    // Index stories by key
    byKey := make(map[string]*model.Story, len(stories))
    for i := range stories {
        byKey[stories[i].Key] = &stories[i]
    }

    // Build edges: explicit (DependsOn) + implicit (file overlap)
    adj := make(map[string][]string)      // key → keys that depend on it
    inDegree := make(map[string]int)
    for _, s := range stories {
        inDegree[s.Key] = inDegree[s.Key] // ensure presence
        for _, dep := range s.DependsOn {
            if _, ok := byKey[dep]; !ok {
                continue // skip unknown keys (AC6)
            }
            adj[dep] = append(adj[dep], s.Key)
            inDegree[s.Key]++
        }
    }

    // Implicit file-conflict edges: for each pair sharing target files,
    // add edge from lexicographically smaller key to larger key
    // (compute overlap via set intersection on TargetFiles)
    // ... see implementation below ...

    // Kahn's: process zero-in-degree nodes layer by layer
    // Return DAG_CYCLE_DETECTED if remaining nodes > 0 after queue exhausted
}
```

**File conflict edge computation:**
```go
// Build file → stories index
fileIndex := make(map[string][]string) // filename → []storyKey
for _, s := range stories {
    for _, f := range s.TargetFiles {
        fileIndex[f] = append(fileIndex[f], s.Key)
    }
}
// For each file with >1 story, sort keys and add edges: keys[0]→keys[1], keys[1]→keys[2], etc.
// Skip if explicit edge already exists to avoid duplicate in-degree counting
```

**OpenAPI additions to api/openapi.yaml (append under /projects/{projectId}/epics/{epicId}):**
```yaml
  /projects/{projectId}/epics/{epicId}/dag:
    get:
      operationId: getEpicDAG
      summary: Get the DAG visualization for an epic's stories
      tags: [epics]
      parameters:
        - $ref: "#/components/parameters/ProjectIdPath"
        - $ref: "#/components/parameters/EpicIdPath"
      responses:
        "200":
          description: DAG nodes and edges
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/EpicDAGResponse"
        "404":
          $ref: "#/components/responses/NotFound"
        "401":
          $ref: "#/components/responses/Unauthorized"

  /projects/{projectId}/epics/{epicId}/runs:
    post:
      operationId: launchEpicRun
      summary: Launch a batch run for all stories in an epic (async)
      tags: [epics]
      parameters:
        - $ref: "#/components/parameters/ProjectIdPath"
        - $ref: "#/components/parameters/EpicIdPath"
      responses:
        "202":
          description: Epic run scheduling accepted
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/EpicRunAccepted"
        "404":
          $ref: "#/components/responses/NotFound"
        "401":
          $ref: "#/components/responses/Unauthorized"
```

**New schemas to add to openapi.yaml components/schemas:**
```yaml
    EpicDAGNode:
      type: object
      required: [id, key, title, status, layer]
      properties:
        id:
          type: string
          format: uuid
        key:
          type: string
        title:
          type: string
        status:
          type: string
        layer:
          type: integer
          description: Zero-based execution layer index

    EpicDAGEdge:
      type: object
      required: [source, target]
      properties:
        source:
          type: string
          description: Source story key (dependency)
        target:
          type: string
          description: Target story key (dependent)

    EpicDAGResponse:
      type: object
      required: [nodes, edges]
      properties:
        nodes:
          type: array
          items:
            $ref: "#/components/schemas/EpicDAGNode"
        edges:
          type: array
          items:
            $ref: "#/components/schemas/EpicDAGEdge"

    EpicRunAccepted:
      type: object
      required: [epic_run_id, status, stories_count]
      properties:
        epic_run_id:
          type: string
          format: uuid
        status:
          type: string
          enum: [scheduling]
        stories_count:
          type: integer
```

**GetEpicDAG handler structure:**
```go
func (h *EpicHandler) GetEpicDAG(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, epicID EpicIdPath) {
    stories, err := h.storyRepo.ListByEpic(r.Context(), epicID, 500, 0)
    if err != nil {
        writeErrorResponse(w, err)
        return
    }

    storyValues := make([]model.Story, len(stories))
    for i, s := range stories {
        storyValues[i] = *s
    }

    dag, err := h.scheduler.BuildDAG(storyValues)
    if err != nil {
        writeErrorResponse(w, err)
        return
    }

    resp := toEpicDAGResponse(dag, storyValues)
    writeJSON(w, http.StatusOK, resp)
}
```

**toEpicDAGResponse transform:**
```go
func toEpicDAGResponse(dag model.DAGResult, stories []model.Story) EpicDAGResponse {
    nodes := make([]EpicDAGNode, 0)
    edges := make([]EpicDAGEdge, 0)

    for layer, group := range dag.Groups {
        for _, s := range group {
            nodes = append(nodes, EpicDAGNode{
                Id:     s.ID,
                Key:    s.Key,
                Title:  s.Title,
                Status: s.Status,
                Layer:  layer,
            })
            for _, dep := range s.DependsOn {
                edges = append(edges, EpicDAGEdge{Source: dep, Target: s.Key})
            }
        }
    }

    return EpicDAGResponse{Nodes: nodes, Edges: edges}
}
```

Note: implicit file-conflict edges are not included in the response edges — only explicit DependsOn edges surface to the frontend. Implicit edges affect only layer placement.

### Testing Requirements

**scheduler_service_test.go table cases:**
- `"empty input"` → Groups length 0, no error
- `"single story no deps"` → 1 group with 1 story
- `"two independent stories"` → 1 group with 2 stories
- `"linear chain A→B→C"` → 3 groups [[A],[B],[C]]
- `"diamond A→B, A→C, B→D, C→D"` → 3 groups [[A],[B,C],[D]]
- `"cycle of two A↔B"` → error DAG_CYCLE_DETECTED
- `"cycle of three A→B→C→A"` → error DAG_CYCLE_DETECTED
- `"unknown dep key ignored"` → story with DependsOn=["GHOST-99"] treated as no-dep
- `"file conflict implicit edge"` → S-01 and S-02 share file, S-01 before S-02 (lex order)
- `"combined explicit + file conflict"` → explicit takes precedence, no duplicate edges

**epic_handler_test.go additions:**
- Mock `StoryRepository` returning 3 stories with no deps → response has 3 nodes all layer 0, 0 edges
- Mock returning stories with cycle → handler returns 422 body with `"code": "DAG_CYCLE_DETECTED"`
- Mock returning error on ListByEpic → handler propagates error

### References

- Kahn's algorithm: https://en.wikipedia.org/wiki/Topological_sorting#Kahn's_algorithm
- Existing error pattern: `backend/pkg/errors/` — use `errors.NewValidation` or add a new `NewInvalidState` call with code `"DAG_CYCLE_DETECTED"`
- Existing handler pattern: `backend/internal/api/handler/epic_handler.go`
- Existing service pattern: `backend/internal/domain/service/epic_service.go`
- Story model: `backend/internal/domain/model/story.go` — DependsOn []string, TargetFiles []string
- StoryRepository port: `backend/internal/domain/port/story_repository.go` — `ListByEpic` already defined

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
