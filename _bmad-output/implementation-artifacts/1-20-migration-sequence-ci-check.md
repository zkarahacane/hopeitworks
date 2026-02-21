# Story 1.20: Migration Sequence CI Check [INFRA]

Status: ready-for-dev

## Story

As a developer,
I want CI to reject PRs that introduce duplicate or out-of-sequence migration numbers,
so that `golang-migrate` never fails silently and migration conflicts are caught before merge.

## Context

Migration files must have unique, monotonically increasing sequence numbers (e.g., `000001`, `000002`).
When multiple agents implement stories in parallel, they may independently pick the same number,
causing `golang-migrate` to error with "duplicate migration file" and block all migrations.
This happened twice during Epic 1 development (duplicates on 000006 and 000014).

## Acceptance Criteria (BDD)

**AC1: CI fails on duplicate migration numbers**
- **Given** a PR that introduces two or more migration files with the same sequence number
- **When** the CI pipeline runs the migration check
- **Then** the job fails with a clear error listing the conflicting files

**AC2: CI passes on valid sequential migrations**
- **Given** a PR with properly numbered migration files (no duplicates, no gaps above the current max)
- **When** the CI pipeline runs the migration check
- **Then** the job passes without error

**AC3: Check runs as a dedicated CI step in the backend job**
- **Given** the `.github/workflows/ci.yml` backend job
- **When** I examine the workflow
- **Then** a step named "Check migration sequence" runs before the Go build step and executes `scripts/check-migrations.sh`

**AC4: Script is self-contained and locally runnable**
- **Given** the script at `scripts/check-migrations.sh`
- **When** I run it locally from the repo root
- **Then** it exits 0 on a clean migration directory and exits non-zero with a human-readable error on duplicates

## Tasks / Subtasks

- [ ] Task 1: Write `scripts/check-migrations.sh` (AC: #1, #2, #4)
  - [ ] Extract sequence numbers from all `*.up.sql` filenames in `backend/migrations/`
  - [ ] Detect and report any duplicate sequence numbers with the conflicting filenames
  - [ ] Exit 0 if no duplicates, exit 1 with error output if duplicates found
  - [ ] Make the script executable (`chmod +x`)

- [ ] Task 2: Add check step to CI workflow (AC: #3)
  - [ ] Add step "Check migration sequence" in `.github/workflows/ci.yml` backend job, before the build step
  - [ ] Step runs `bash scripts/check-migrations.sh`

- [ ] Task 3: Verify locally and in CI (AC: #1, #2, #3, #4)
  - [ ] Run `bash scripts/check-migrations.sh` on current clean repo → exits 0
  - [ ] Temporarily introduce a duplicate and verify script exits 1 with clear output
  - [ ] Push to PR and verify CI step appears and passes

## Dev Notes

### Script Logic

```bash
#!/usr/bin/env bash
# check-migrations.sh — fail if any two migration files share the same sequence number

set -euo pipefail

MIGRATIONS_DIR="backend/migrations"

# Extract sequence numbers from *.up.sql filenames
numbers=$(ls "$MIGRATIONS_DIR"/*.up.sql 2>/dev/null | xargs -I{} basename {} | grep -oE '^[0-9]+')

duplicates=$(echo "$numbers" | sort | uniq -d)

if [ -n "$duplicates" ]; then
  echo "ERROR: duplicate migration sequence numbers found:"
  for num in $duplicates; do
    echo "  $num:"
    ls "$MIGRATIONS_DIR"/${num}_*.up.sql | sed 's/^/    /'
  done
  exit 1
fi

echo "Migration sequence check passed."
```

### CI Step Placement

Insert before the `go build` step in the backend job:

```yaml
- name: Check migration sequence
  run: bash scripts/check-migrations.sh
```

### Architecture Requirements

**Files to create or modify:**

```
hopeitworks/
├── scripts/
│   └── check-migrations.sh   # new — migration duplicate checker
└── .github/
    └── workflows/
        └── ci.yml             # add one step to backend job
```

### Dependencies

- **Story 1-17** (GitHub Actions CI pipeline): the `ci.yml` workflow must exist before adding a step to it

### References

- Incident: PR #79 fixed two separate migration number conflicts (000006 × 3, 000014 × 2)

## Dev Agent Record

## Change Log
