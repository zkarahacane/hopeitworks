# Story 1-21: [SHARED] Sprint status reconciliation — mark merged stories as done

Status: done

## Story

As a project maintainer,
I want the sprint-status.yaml to accurately reflect the actual state of all merged stories,
so that project tracking is reliable and future pipeline decisions use correct dependency data.

## Context

The sprint-status.yaml file has drifted from reality. Multiple stories (fix-9 through fix-14, refactor-1 through refactor-3, and the post-mvp-e2e-usability epic) are shown as "backlog", "review", or "dev-complete" despite being fully implemented, merged via PRs, and present on the develop branch.

This was discovered during a codebase health audit: all 536 frontend tests pass, all backend tests pass with race detection, lint and type-check are clean, but the tracking file is stale.

### Affected stories (all confirmed merged to develop)

| Story | Was | Now | PR |
|-------|-----|-----|----|
| fix-9-backend-wiring | review | done | #119 |
| fix-10-openapi-project-fields | review | done | #118 |
| fix-11-missing-service-methods | dev-complete | done | #120 |
| fix-12-frontend-api-regen | backlog | done | #121 |
| fix-13-project-repo-form | backlog | done | #123 |
| fix-14-runs-dashboard-routing | backlog | done | #122 |
| post-mvp-e2e-usability (epic) | backlog | done | — |

## Acceptance Criteria (BDD)

**AC1: All merged stories are marked as done**
- **Given** stories fix-9 through fix-14 have been merged to develop via PRs
- **When** the sprint-status.yaml is updated
- **Then** all six stories show `status: done` and the `post-mvp-e2e-usability` epic shows `done`

**AC2: Wave statuses reflect completion**
- **Given** all stories in wave-22 and wave-23 are done
- **When** the sprint-status.yaml is updated
- **Then** wave-22 and wave-23 story entries show `status: done`

**AC3: No false positives**
- **Given** the update is based on git log evidence (merged PRs)
- **When** compared against the develop branch
- **Then** no story is marked done unless its corresponding PR commit exists on develop

## Tasks / Subtasks

- [x] [SHARED] Task 1: Audit develop branch git log to confirm merged PRs (AC: #3)
- [x] [SHARED] Task 2: Update sprint-status.yaml — mark fix-9 through fix-14 and epic as done (AC: #1, #2)

## Dev Agent Record

**Agent:** Claude Opus 4.6
**Branch:** feat/1-21
**Date:** 2026-02-22

### Files Changed

| File | Change |
|------|--------|
| `_bmad-output/implementation-artifacts/sprint-status.yaml` | Mark fix-9 through fix-14 as done, mark post-mvp-e2e-usability epic as done |
| `_bmad-output/implementation-artifacts/1-21-sprint-status-reconciliation.md` | New — story definition |

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-22 | dev-agent | Story created and implemented |
