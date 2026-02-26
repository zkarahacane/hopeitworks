# Story F-2.2: [FRONT] Fix board epic story counts display

Status: ready-for-dev

## Story

As a user viewing the story board,
I want to see accurate story counts per status on each epic card,
so that I can quickly understand the progress of each epic.

## Acceptance Criteria (BDD)

**AC1: Backlog count matches**
- **Given** an epic has 3 stories in "backlog" status
- **When** viewing the Board page
- **Then** the epic card shows "3 Backlog"

**AC2: All status counts display**
- **Given** an epic has stories in various statuses
- **When** viewing the Board page
- **Then** each status count (Backlog, Running, Done, Failed) reflects actual story counts

**AC3: Counts update after story status change**
- **Given** a story changes status
- **When** the Board page is refreshed or updated via SSE
- **Then** the epic card counts reflect the new status

## Root Cause Analysis

The bug is a full-stack gap. The frontend (`EpicCard.vue`) correctly reads `epic.story_counts` from the
API response and the generated TypeScript types (`schema.d.ts`) correctly define `StoryCounts` as a
required field on `Epic`. However, the backend never populates this field:

1. `backend/internal/domain/model/epic.go` — `model.Epic` has no `StoryCounts` field.
2. `backend/queries/epics.sql` — `ListEpicsByProject` does a plain `SELECT *` with no story count
   aggregation join.
3. `backend/internal/api/handler/epic_handler.go` — `toAPIEpic()` constructs the API `Epic` struct
   but never sets `StoryCounts`, so it is serialized as `{"backlog":0,"running":0,"done":0,"failed":0}`.
4. The frontend receives zero counts and displays them faithfully — the frontend code is correct.

## Tasks / Subtasks

### Task 1 — Add `StoryCounts` to domain model
File: `backend/internal/domain/model/epic.go`

Add the `StoryCounts` struct and embed it in `Epic`:

```go
// StoryCounts holds the count of stories per status for an epic.
type StoryCounts struct {
    Backlog int
    Running int
    Done    int
    Failed  int
}

type Epic struct {
    // ... existing fields ...
    StoryCounts StoryCounts
}
```

### Task 2 — Add a SQL query that returns story counts per epic
File: `backend/queries/epics.sql`

Add a new query `ListEpicsByProjectWithCounts` that LEFT JOINs stories and aggregates by status. It must
return all existing epic columns plus four count columns (`backlog_count`, `running_count`, `done_count`,
`failed_count`).

```sql
-- name: ListEpicsByProjectWithCounts :many
SELECT
    e.*,
    COUNT(s.id) FILTER (WHERE s.status = 'backlog')  AS backlog_count,
    COUNT(s.id) FILTER (WHERE s.status = 'running')  AS running_count,
    COUNT(s.id) FILTER (WHERE s.status = 'done')     AS done_count,
    COUNT(s.id) FILTER (WHERE s.status = 'failed')   AS failed_count
FROM epics e
LEFT JOIN stories s ON s.epic_id = e.id
WHERE e.project_id = $1
GROUP BY e.id
ORDER BY e.created_at DESC
LIMIT $2 OFFSET $3;
```

Regenerate sqlc after adding this query:
```bash
cd backend && sqlc generate
```

### Task 3 — Update `EpicRepo.ListByProject` to use the new query
File: `backend/internal/adapter/postgres/epic_repo.go`

Replace the call to `r.queries.ListEpicsByProject` with the new
`r.queries.ListEpicsByProjectWithCounts`. Update `toDomainEpic` (or add a separate mapper) to
map the four count columns onto `model.StoryCounts`.

The new mapper should read the count columns from the sqlc-generated row struct and set them on
`model.Epic.StoryCounts`.

### Task 4 — Populate `StoryCounts` in `toAPIEpic`
File: `backend/internal/api/handler/epic_handler.go`

Update `toAPIEpic` to set the `StoryCounts` field on the API `Epic` response type:

```go
func toAPIEpic(e *model.Epic) Epic {
    epic := Epic{
        // ... existing fields ...
        StoryCounts: StoryCounts{
            Backlog: e.StoryCounts.Backlog,
            Running: e.StoryCounts.Running,
            Done:    e.StoryCounts.Done,
            Failed:  e.StoryCounts.Failed,
        },
    }
    // ...
    return epic
}
```

### Task 5 — Add/update unit test for `toAPIEpic`
File: `backend/internal/api/handler/epic_handler_test.go`

Add a test case asserting that `toAPIEpic` propagates `StoryCounts` correctly from domain to API type.

### Task 6 — Add integration test for `ListByProject` story counts
File: `backend/internal/adapter/postgres/epic_repo.go` (test file alongside)

Add an integration test that:
1. Creates an epic
2. Creates stories in different statuses under that epic
3. Calls `ListByProject`
4. Asserts that the returned `model.Epic.StoryCounts` matches the expected values

### Task 7 — Verify frontend renders correctly (no frontend code change expected)
Files: `frontend/src/features/board/EpicCard.vue`, `frontend/src/stores/epics.ts`

Confirm no frontend changes are needed:
- `EpicCard.vue` already reads `epic.story_counts[status.key]` for all four statuses.
- `useEpicsStore.fetchEpics` already stores the full `Epic` object from the API response.
- The generated `schema.d.ts` already types `story_counts` as a required `StoryCounts` object.

If a visual regression test exists in `frontend/e2e/tests/`, add a scenario asserting non-zero counts
appear on epic cards when stories exist.

## Dev Notes

- Priority: P2
- Scope: primarily backend (Tasks 1-6); frontend requires no code changes (Task 7 is verification only)
- The `CountEpicsByProject` query in `epics.sql` is unrelated — it counts epics, not stories
- The existing `GetEpic` (single epic by ID) handler has the same missing `StoryCounts` issue and
  should be fixed as part of this story for consistency
- `golangci-lint run ./...` must pass after changes — run before committing
- No OpenAPI spec change is needed — `story_counts` is already defined as required in `api/openapi.yaml`
  and the generated `schema.d.ts` is already correct
