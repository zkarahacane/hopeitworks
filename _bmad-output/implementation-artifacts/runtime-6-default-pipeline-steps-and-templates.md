# Story runtime-6: Default Pipeline Steps and Prompt Templates

**Status:** ready-for-dev
**Branch:** `feat/runtime-6-default-steps-templates`
**Commit scope:** `pipeline`

---

## Story

As the hopeitworks platform maintainer, I need the default pipeline configuration to use current model names and the prompt template DB seeds to be complete and consistent with the hardcoded fallbacks — so that new projects are seeded with correct, up-to-date defaults and existing projects are not missing the `merge` template in their DB.

---

## Acceptance Criteria

**AC #1 — Model names are current**
- Given `DefaultPipelineConfigYAML` is read
- When the YAML is parsed
- Then all model references use current model IDs (`claude-opus-4-6`, `claude-sonnet-4-6`)
- And no reference to `claude-sonnet-4-5` remains in the codebase

**AC #2 — All default templates are seeded in DB**
- Given migration 000024 is applied
- When all templates are queried for any project
- Then `implement`, `implement-retry`, `review`, `merge`, and `merge-conflict` templates all exist in DB
- And the migration is idempotent (re-running does not create duplicates)

**AC #3 — Template content is consistent**
- Given `getDefaultTemplate` returns hardcoded fallbacks
- When compared with DB-seeded templates (migration 000012 + migration 000024)
- Then the content is identical for each template name
- Specifically: `implement-retry` DB seed must include `{{log_tail}}` (currently missing from 000012)

**AC #4 — Templates provide quality prompts**
- Given the `implement` template is rendered with story data
- When an agent reads the prompt
- Then the prompt includes: story key, title, objective, acceptance criteria, file paths to modify, branch name, and clear instructions
- And the `review` template includes: what to check (ACs, lint, tests), how to report findings
- And the `merge` template includes: CI check, rebase strategy, squash merge instructions

**AC #5 — All tests pass**
- Given the changes are applied
- When `go test ./... -short` runs
- Then all tests pass
- And `golangci-lint run ./...` reports zero errors

---

## Tasks / Subtasks

- [ ] **T1.** Fix model names in `DefaultPipelineConfigYAML` (AC: #1)
  - [ ] T1.1 In `pipeline_config_service.go`, change `review` step model from `claude-sonnet-4-5` to `claude-sonnet-4-6`
  - [ ] T1.2 In `pipeline_config_service.go`, change `merge` step model from `claude-sonnet-4-5` to `claude-sonnet-4-6`

- [ ] **T2.** Create migration 000024 to seed the `merge` template (AC: #2)
  - [ ] T2.1 Check `ls backend/migrations/` to confirm 000024 is the correct next number
  - [ ] T2.2 Create `backend/migrations/000024_seed_merge_template.up.sql` — INSERT merge template for all projects where it does not already exist
  - [ ] T2.3 Create `backend/migrations/000024_seed_merge_template.down.sql` — DELETE the merge template for all projects

- [ ] **T3.** Sync `implement-retry` template content (AC: #3)
  - [ ] T3.1 Update migration 000012 `implement-retry` seed to add `## Log Tail\n{{log_tail}}` section between `## Previous Error` and `## Existing Changes` (to match the hardcoded fallback in `template_service.go`)
  - [ ] T3.2 Verify the hardcoded fallback in `getDefaultTemplate` exactly matches the 000012 seed for all other templates (implement, review, merge-conflict)

- [ ] **T4.** Review and improve template content (AC: #4)
  - [ ] T4.1 Improve `implement` hardcoded fallback and 000012 seed: add explicit instruction line ("Implement the story according to the acceptance criteria") and a `## Dev Notes` section using `{{dev_notes}}` if the variable exists in `TemplateContext`
  - [ ] T4.2 Improve `review` hardcoded fallback and 000012 seed: add explicit check items (ACs met, `golangci-lint` passes, tests added, no secrets, no `console.log`)
  - [ ] T4.3 Improve `merge` hardcoded fallback and 000024 seed: add explicit steps (check CI on feature branch, rebase on `develop`, create PR with `gh pr create`, squash merge with `gh pr merge --squash`, verify post-merge CI on `develop`)

- [ ] **T5.** Run lint and tests (AC: #5)
  - [ ] T5.1 `cd backend && golangci-lint run ./...` — must report zero errors
  - [ ] T5.2 `cd backend && go test ./... -short` — must pass

---

## Dev Notes

### Migration number

Verify with `ls backend/migrations/` before creating files. At time of story writing, the last migration was 000023 (two separate 000023 files exist — `create_password_reset_tokens` and `create_revoked_tokens`). Use 000024 unless a higher number already exists.

### Template variables available in `TemplateContext`

Check `backend/internal/domain/model/` for the `TemplateContext` struct to know which Handlebars variables are valid. Do not reference variables that do not exist on the struct.

### Down migration for 000024

The down migration must only delete templates that were seeded by the up migration — do not delete templates that were manually created by users. Use a `WHERE template_content = '...'` guard or, safer, just DELETE by name since these are the platform defaults:

```sql
DELETE FROM prompt_templates WHERE name = 'merge';
```

### Hardcoded fallback vs DB seed — when each is used

- DB seeds (000012, 000024): used for existing projects after migration runs. New projects are seeded by `SeedDefaultTemplates` service call at project creation time.
- Hardcoded fallbacks in `getDefaultTemplate`: safety net when DB lookup returns not-found (e.g., template deleted by user, new template name added before next migration).
- Both must stay in sync — the DB is the primary source, the hardcoded fallback is the emergency backup.

### Do NOT remove hardcoded fallbacks

The `getDefaultTemplate` function in `template_service.go` must retain all fallbacks even after adding the DB seed migration.

### Template syntax

Templates use Handlebars syntax. Variables: `{{variable_name}}`. Loops: `{{#each list}}- {{this}}{{/each}}`. The renderer is registered in `TemplateRenderer` port.

### Merge template content guidance

The merge step agent must:
1. Check that the feature branch CI is green (use `gh pr checks` or `gh run list`)
2. Rebase the feature branch on `develop` (or base branch)
3. Open a PR with `gh pr create --title "..." --body "..."` following conventional commit format
4. Squash merge with `gh pr merge --squash --auto` or after CI green
5. Confirm the merge and verify `develop` CI passes

---

## File Paths

| Action | Path |
|--------|------|
| MODIFY | `backend/internal/domain/service/pipeline_config_service.go` |
| MODIFY | `backend/internal/domain/service/template_service.go` |
| MODIFY | `backend/migrations/000012_seed_default_prompt_templates.up.sql` |
| CREATE | `backend/migrations/000024_seed_merge_template.up.sql` |
| CREATE | `backend/migrations/000024_seed_merge_template.down.sql` |

---

## Change Log

- Story created (2026-02-22)
