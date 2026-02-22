# Story refactor-3: DB Migration and Final Cleanup

**Status:** ready-for-dev

**Blocked by:** refactor-2 (all consumers must be on `Runner` before renaming the DB column)

## Story

As a backend developer, I want to rename the `container_id` column in `run_steps` to `workload_handle`, update the sqlc queries and generated code, align the domain model field name, and delete all remaining dead code, so that the codebase is fully consistent with the new runtime-agnostic vocabulary.

## Acceptance Criteria

**AC1: Migration `000024` renames the column**
- Given migration `000024_rename_container_id_to_workload_handle.up.sql` is applied
- When the `run_steps` table is inspected
- Then the column `container_id` is renamed to `workload_handle`
- And the rollback migration `000024_rename_container_id_to_workload_handle.down.sql` renames it back to `container_id`
- And the column type (`VARCHAR(255)`) and nullability (`NULL`) are unchanged

**AC2: sqlc queries reference `workload_handle`**
- Given `backend/queries/run_steps.sql` is updated
- When sqlc code generation runs (`cd backend && sqlc generate`)
- Then all references to `container_id` are replaced with `workload_handle`
- And the queries `UpdateRunStepStatus` and `UpdateRunStepContainerInfo` use `workload_handle`
- And the `sqlc.narg('container_id')` expressions are updated to `sqlc.narg('workload_handle')`

**AC3: Generated sqlc code is regenerated**
- Given `backend/internal/adapter/postgres/db/` contains sqlc-generated files
- When `sqlc generate` is run
- Then the generated Go structs no longer have a `ContainerID pgtype.Text` field
- And they have a `WorkloadHandle pgtype.Text` field instead
- And the generated params types (`UpdateRunStepStatusParams`, `UpdateRunStepContainerInfoParams`) reflect the rename

**AC4: `model/run.go` renames `ContainerID` to `WorkloadHandle`**
- Given `RunStep` struct is updated
- When the domain model is used
- Then `RunStep.ContainerID *string` is renamed to `RunStep.WorkloadHandle *string`

**AC5: `adapter/postgres/run_repo.go` is updated**
- Given `run_repo.go` is updated
- When `UpdateRunStepContainerInfo` is called
- Then the params struct field `ContainerID` is referenced as `WorkloadHandle`
- And `toDomainRunStep` maps `row.WorkloadHandle` to `step.WorkloadHandle`
- And the method signature remains `UpdateRunStepContainerInfo(ctx, id, handle *string, logTail *string) (*model.RunStep, error)`

**AC6: `adapter/action/agent_run.go` is updated**
- Given `persistHandle` in `agent_run.go` calls `runRepo.UpdateRunStepContainerInfo`
- When the code compiles
- Then it continues to compile without modification (the method signature is unchanged; only the internal implementation changes)

**AC7: No remaining references to `ContainerID` or `container_id` in domain/adapter code**
- Given the rename is complete
- When the codebase is grep'd for `ContainerID` in `internal/domain` and `internal/adapter`
- Then zero occurrences are found (except in comments and migration rollback file)

**AC8: All tests pass and lint is clean**
- Given the migration and code updates are applied
- When `go test ./... -short` runs
- Then all tests pass
- And `golangci-lint run ./...` reports zero errors

## Tasks / Subtasks

- [ ] Create migration `000024_rename_container_id_to_workload_handle.up.sql` (AC: #1)
- [ ] Create migration `000024_rename_container_id_to_workload_handle.down.sql` (AC: #1)
- [ ] Update `backend/queries/run_steps.sql`: replace `container_id` with `workload_handle` (AC: #2)
- [ ] Run `cd backend && sqlc generate` to regenerate `internal/adapter/postgres/db/` (AC: #3)
- [ ] Update `backend/internal/domain/model/run.go`: rename `ContainerID *string` to `WorkloadHandle *string` in `RunStep` (AC: #4)
- [ ] Update `backend/internal/adapter/postgres/run_repo.go` (AC: #5)
  - [ ] Update `UpdateRunStepContainerInfo` to use `params.WorkloadHandle` instead of `params.ContainerID`
  - [ ] Update `toDomainRunStep` to map `row.WorkloadHandle` to `step.WorkloadHandle`
  - [ ] Update `UpdateRunStepStatus` params if `container_id` appears there too
- [ ] Verify `adapter/action/agent_run.go` still compiles without changes (AC: #6)
- [ ] Grep for residual `ContainerID` and `container_id` references in `internal/` (AC: #7)
- [ ] Run `go build ./...` (AC: #8)
- [ ] Run `go test ./... -short` (AC: #8)
- [ ] Run `golangci-lint run ./...` (AC: #8)

## Dev Notes

### Dependencies

- Story refactor-2 must be merged before this story
- `sqlc` must be installed in the dev environment (`cd backend && sqlc generate`)
- Migration numbering: the last migration is `000023` (two files: `create_password_reset_tokens_table` and `create_revoked_tokens_table` both use `000023`). Use `000024` for this migration.

### File Paths

| Action | Path |
|--------|------|
| CREATE | `backend/migrations/000024_rename_container_id_to_workload_handle.up.sql` |
| CREATE | `backend/migrations/000024_rename_container_id_to_workload_handle.down.sql` |
| MODIFY | `backend/queries/run_steps.sql` |
| REGENERATE | `backend/internal/adapter/postgres/db/` (via `sqlc generate`) |
| MODIFY | `backend/internal/domain/model/run.go` |
| MODIFY | `backend/internal/adapter/postgres/run_repo.go` |

### Technical Specifications

#### Migration files

`000024_rename_container_id_to_workload_handle.up.sql`:
```sql
ALTER TABLE run_steps RENAME COLUMN container_id TO workload_handle;
```

`000024_rename_container_id_to_workload_handle.down.sql`:
```sql
ALTER TABLE run_steps RENAME COLUMN workload_handle TO container_id;
```

Simple `RENAME COLUMN` — no data migration needed, no type change, no constraint change.

#### `queries/run_steps.sql` — updated queries

The two queries that reference `container_id` are `UpdateRunStepStatus` and `UpdateRunStepContainerInfo`. After the rename:

```sql
-- name: UpdateRunStepStatus :one
UPDATE run_steps
SET status = $2,
    started_at = COALESCE(sqlc.narg('started_at'), started_at),
    completed_at = COALESCE(sqlc.narg('completed_at'), completed_at),
    error_message = COALESCE(sqlc.narg('error_message'), error_message),
    workload_handle = COALESCE(sqlc.narg('workload_handle'), workload_handle),
    log_tail = COALESCE(sqlc.narg('log_tail'), log_tail)
WHERE id = $1
RETURNING *;

-- name: UpdateRunStepContainerInfo :one
UPDATE run_steps
SET workload_handle = COALESCE(sqlc.narg('workload_handle'), workload_handle),
    log_tail = COALESCE(sqlc.narg('log_tail'), log_tail)
WHERE id = $1
RETURNING *;
```

All other queries (`CreateRunStep`, `GetRunStep`, `ListRunStepsByRun`, `CreateRetryRunStep`, `ListRetryStepsByParent`) use `SELECT *` or explicit column lists that do not include `container_id` by name — verify with `grep` and update if needed.

#### `model/run.go` — field rename

```go
// RunStep struct
type RunStep struct {
    ID             uuid.UUID
    RunID          uuid.UUID
    StepName       string
    StepOrder      int
    Action         string
    Status         StepStatus
    StartedAt      *time.Time
    CompletedAt    *time.Time
    ErrorMessage   *string
    WorkloadHandle *string   // renamed from ContainerID
    LogTail        *string
    RetryCount     int
    RetryType      *string
    ParentStepID   *uuid.UUID
    CreatedAt      time.Time
}
```

#### `adapter/postgres/run_repo.go` — diff

```go
// UpdateRunStepContainerInfo
func (r *RunRepo) UpdateRunStepContainerInfo(ctx context.Context, id uuid.UUID, handle *string, logTail *string) (*model.RunStep, error) {
    params := UpdateRunStepContainerInfoParams{ID: id}
    if handle != nil {
        params.WorkloadHandle = pgtype.Text{String: *handle, Valid: true}  // was ContainerID
    }
    if logTail != nil {
        params.LogTail = pgtype.Text{String: *logTail, Valid: true}
    }
    row, err := r.queries.UpdateRunStepContainerInfo(ctx, params)
    // ...
    return toDomainRunStep(row), nil
}

// toDomainRunStep
func toDomainRunStep(s RunStep) *model.RunStep {
    step := &model.RunStep{
        // ...
    }
    if s.WorkloadHandle.Valid {                         // was s.ContainerID
        step.WorkloadHandle = &s.WorkloadHandle.String  // was step.ContainerID
    }
    // ...
    return step
}
```

Also update `UpdateRunStepStatus` params if the generated struct includes `WorkloadHandle` (check generated code after `sqlc generate`).

#### sqlc generation

After editing `queries/run_steps.sql`, run:
```bash
cd backend && sqlc generate
```

The generated files in `internal/adapter/postgres/db/` are auto-generated — do NOT manually edit them. If sqlc generates a `ContainerID` field anywhere, it means the query file still contains `container_id` — fix the query and regenerate.

#### Residual reference check

After all edits, run the following to confirm no stale references remain:
```bash
grep -rn "ContainerID\|container_id" \
  backend/internal/domain/ \
  backend/internal/adapter/ \
  --include="*.go"
```

Expected: zero matches. The only legitimate occurrence of `container_id` is in the rollback migration file and possibly in SQL comments — both are acceptable.

#### `UpdateRunStepStatus` in `run_steps.sql`

The existing query includes `container_id = COALESCE(sqlc.narg('container_id'), container_id)`. This line should be removed from `UpdateRunStepStatus` unless there is a concrete use case that still needs to set the handle via this query. At the time of writing (stories refactor-1 through refactor-3), the handle is only set via `UpdateRunStepContainerInfo`. Remove the `workload_handle` line from `UpdateRunStepStatus` to keep the query minimal, unless existing callers rely on it. Verify with `grep -rn "UpdateRunStepStatus"` in the codebase before removing.

### References

- Migration naming convention: `backend/migrations/000001_create_users_table.up.sql` (zero-padded 6 digits, underscore-separated description)
- sqlc config: `backend/sqlc.yaml`
- Generated output directory: `backend/internal/adapter/postgres/db/`
- `run_repo.go` method `toDomainRunStep` at line ~312 maps `ContainerID.Valid` to `step.ContainerID`
- Stories refactor-1 and refactor-2 must be fully merged and CI green before starting this story
