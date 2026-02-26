# Story F-2.1: [BACK] Fix seed runs to reference correct project UUID

Status: ready-for-dev

## Story

As a developer,
I want seed run data to be correctly linked to the seeded project,
so that the Runs page and Cost Dashboard display sample data after a fresh setup.

## Acceptance Criteria (BDD)

**AC1: Seeded runs appear in project runs list**
- **Given** a fresh database seeded with seed.sql
- **When** navigating to the Todo App project's Runs tab
- **Then** seeded runs are visible (3 runs: 1 completed, 1 running, 1 pending)

**AC2: Seeded costs appear in cost dashboard**
- **Given** a fresh database seeded with seed.sql
- **When** navigating to the Todo App project's Costs tab
- **Then** cost data from seeded runs is displayed (2 cost records totalling ~$0.81)

**AC3: All FK references are consistent**
- **Given** seed.sql has been executed
- **When** querying runs, run_steps, step_costs
- **Then** all foreign key references (project_id, story_id, run_id, run_step_id) point to existing records

**AC4: Seed is idempotent**
- **Given** seed.sql has already been executed once
- **When** seed.sql is executed a second time
- **Then** no errors occur and no duplicate rows are created

## Tasks / Subtasks

### Task 1: Fix project INSERT to use id-based conflict resolution

The current `ON CONFLICT (name)` clause on the `projects` insert is the root cause.
If a project named "Todo App" already exists with a *different* UUID, the `id` column is
never updated, causing all subsequent inserts that reference
`00000000-0000-0000-0000-000000000101` to violate FK constraints silently
(thanks to `ON CONFLICT DO NOTHING` on the dependent tables).

- Change the conflict target on the `projects` INSERT from `ON CONFLICT (name)` to
  `ON CONFLICT (id)`, so that a re-run always upserts the row at the known UUID:

```sql
INSERT INTO projects (id, name, description, owner_id, repo_url, git_provider, git_token_env, default_model)
VALUES (
    '00000000-0000-0000-0000-000000000101',
    ...
) ON CONFLICT (id) DO UPDATE SET
    name          = EXCLUDED.name,
    description   = EXCLUDED.description,
    owner_id      = EXCLUDED.owner_id,
    repo_url      = EXCLUDED.repo_url,
    git_provider  = EXCLUDED.git_provider,
    git_token_env = EXCLUDED.git_token_env,
    default_model = EXCLUDED.default_model;
```

  Verify that `projects` has a unique constraint or primary key on `id` (it does — uuid PK).

### Task 2: Apply the same id-based conflict resolution to epics and stories

The epic INSERT uses `ON CONFLICT (project_id, name)` and story INSERTs use
`ON CONFLICT (project_id, key)`. These are semantically correct for uniqueness but
they do NOT guarantee the well-known UUIDs are preserved on re-seed.
Change all three to `ON CONFLICT (id) DO UPDATE SET ...` to ensure downstream FK
references always resolve to the fixed UUIDs:

- Epic `00000000-0000-0000-0000-000000000201` — change conflict target to `(id)`
- Story `00000000-0000-0000-0000-000000000301` — change conflict target to `(id)`
- Story `00000000-0000-0000-0000-000000000302` — change conflict target to `(id)`
- Story `00000000-0000-0000-0000-000000000303` — change conflict target to `(id)`

### Task 3: Verify run_steps and cost_records insert correctly after the fix

The run INSERTs already use `ON CONFLICT DO NOTHING` keyed on the primary key UUID,
so they are structurally correct once the upstream project/story UUIDs are stable.
Trace and confirm:

- `runs` (`00000000-...-000000000501/502/503`) → `project_id` and `story_id` resolve
- `run_steps` (`00000000-...-000000000601..612`) → `run_id` resolves
- `hitl_requests` (`00000000-...-000000000701`) → `run_step_id` resolves
- `cost_records` (`00000000-...-000000001001/1002`) → `run_step_id` and `project_id` resolve
- `events` (`00000000-...-000000000801..804`) → `project_id` and `entity_id` resolve

No structural changes should be needed for these tables once Tasks 1 and 2 are applied.

### Task 4: Manual verification after fix

Run the seed on a clean DB and confirm:

```bash
cd backend && make reset-db && make seed
```

Then execute these spot-check queries:

```sql
-- Confirm project UUID is stable
SELECT id, name FROM projects WHERE name = 'Todo App';
-- Expected: 00000000-0000-0000-0000-000000000101

-- Confirm runs are visible
SELECT id, status FROM runs WHERE project_id = '00000000-0000-0000-0000-000000000101';
-- Expected: 3 rows

-- Confirm cost records are linked
SELECT cr.id, cr.cost_usd FROM cost_records cr
JOIN run_steps rs ON rs.id = cr.run_step_id
JOIN runs r ON r.id = rs.run_id
WHERE r.project_id = '00000000-0000-0000-0000-000000000101';
-- Expected: 2 rows (0.6315, 0.183)
```

## Dev Notes

- **Priority:** P1
- **File to modify:** `backend/testdata/seed.sql` only — no Go code changes required
- **Depends on:** F-1-2 (seed bcrypt fix) — both touch `seed.sql`; merge F-1-2 first to avoid conflicts
- **Root cause summary:** `ON CONFLICT (name)` on the `projects` table does not guarantee
  the row's `id` matches the hardcoded UUID used in all downstream inserts. On a DB where
  "Todo App" already exists under a different UUID, the `id` field is never reconciled, and
  `ON CONFLICT DO NOTHING` on the runs/steps silently swallows the FK violations.
- **Preferred fix:** `ON CONFLICT (id)` for all seed entities that have downstream FK references,
  ensuring the well-known UUIDs are always authoritative.
- **Alternative (not preferred):** Replace hardcoded UUIDs with `INSERT ... SELECT` subqueries
  that look up by name. This is more resilient but significantly more verbose and harder to read.
