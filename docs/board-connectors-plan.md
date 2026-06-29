# Board & Planning Connectors — Plan

> **Scope & product stance.** The board is **in-app** — a generated kanban derived from the internal epic→story model and live run state, *not* a mirror of an external GitHub Project. Connectors are **input-only**: they read external planning sources (GitHub, GitLab, Jira, BMAD, markdown) and write into the internal epic→story format. Markdown is the git-native spec. A rich connector (BMAD/GSD) can skip the scoping/enrichment step because the spec is already agent-ready. The internal story then feeds the agent's execution context (`PROMPT` + `CLAUDE_MD_CONTENT`). Status write-back to the source is a deliberate *later* phase, not v1.

---

## 1. Executive summary

The through-line is one pipeline:

```
external source  →  input adapter  →  internal story (canonical)  →  agent context bundle  →  container execution
 (GH/GL/Jira/        (normalize +       (stories table, keyed         (TemplateContext +        (agent-runtime)
  BMAD/md)            enrich)            by external_id)               CLAUDE_MD_CONTENT)
```

Today we have a working **right half** and a missing **left half**:

- **Right half (built):** a live in-app kanban (`useBoard` + SSE + `boardColumn()`), a canonical `model.Story`, and a working agent-context path (`agent_run.go` → `TemplateContext` + `buildClaudeMD`). The board is already workflow-driven and live (PR #236).
- **Left half (missing):** there is **no connector layer**. The only ingestion paths are manual `POST /stories` and a single markdown blob import (`POST /stories/import`). There is no `external_id`, no `source`, no `url`, no `planning_source` anywhere in the schema — so import is a destructive upsert-by-`key` with zero traceability, and the frontend "Planned in" control is a pure heuristic over `project.git_provider` with an explicit *"Backend gap"* comment in `BoardView.vue:39`.

**The five biggest gaps:**

1. **No external identity on stories.** `model.Story` (`backend/internal/domain/model/story.go`) has no `external_id` / `source` / `source_url`. Re-import overwrites by `key` with no idempotency anchor to the origin and no deep-link back.
2. **No connector port/adapters.** There is no `PlanningSourceAdapter` interface and no per-source adapter. Hexagonal layout exists and is clean (`domain/port` + `adapter/*`) — the slot is empty, not blocked.
3. **Thin issues never get enriched.** A GitHub/Jira issue is a one-liner; an agent needs objective + acceptance criteria + target files + verification. There is no enrichment step between import and execution. Rich BMAD stories are the opposite problem — they already carry everything but our markdown parser **discards** most of it (objective, target_files, even epic linkage — see `story_import.go`).
4. **Agent context is under-fed.** `buildClaudeMD` injects only `project.Description` + role framing; the story objective/AC/target_files live only in the Handlebars `PROMPT`, and `depends_on`, `scope`, epic goal, and any source link never reach the agent.
5. **Board can't show provenance.** No `source` / `external_id` / `url` means no source badge, no "Planned in" backing field, no re-sync affordance, no dependency surfacing.

**The plan:** add a `PlanningSourceAdapter` port + per-source adapters writing to the canonical story format; migrate the story model to carry `external_id` + `source` + `source_url` (idempotent upsert key `(project_id, external_id)`); add an enrichment step for thin issues (River job) and a fast-path for rich BMAD/GSD specs; enrich the agent context bundle; and surface provenance + re-sync on the board. One-way import first, status write-back later.

---

## 2. Current state

### 2.1 Board model (from `map:board-model`)

- **Epic** (`epics` table, `model.Epic`): `id, project_id, name, description, status` — no `owner`/`priority`/`external_id`/`source`/`planning_source`. Each epic response carries a computed `story_counts` (`{Backlog,Running,Done,Failed}` via `CountByEpicGroupedByStatus`).
- **Story** (`stories` table, `model.Story` in `backend/internal/domain/model/story.go`): `id, project_id, epic_id?, key (UNIQUE per project, ^[A-Z0-9]+-\d+$), title, objective?, target_files (jsonb []string), depends_on (jsonb []string of keys), scope?, status, acceptance_criteria?`. No `external_id`, `source`, `url`, `priority`, `labels`, `assignee`, `display_order`.
- **LatestRun** projection (assembled at query time, not a table): carried on every story for the kanban; `current_step` = lowest-`step_order` step in `running`/`waiting_approval`.
- **Live board:** `frontend/src/composables/useBoard.ts` fetches epics+stories via REST (batch `latest_run`, no N+1), opens one `EventSource` on `/api/v1/events/stream`, dispatches each SSE event into `storiesStore` (column placement via pure `boardColumn()`) and `runtimeStream` (timers/cost/gate flags). Five columns: Backlog / Running / In Review / Done / Failed.
- **"Planned in":** `frontend/src/views/BoardView.vue:38-74` derives the planning source from `project.git_provider` — a frontend-only heuristic, explicitly flagged as a backend gap.

### 2.2 Ingestion & internal story format (from `map:ingestion`)

Two paths, nothing else:

1. **Manual create** — `POST /projects/{id}/stories` → `StoryService.Create` (`story_handler.go`). Validates `key` against `^[A-Z0-9]+-\d+$`. Uniqueness `(project_id, key)` → 409 on dup.
2. **Markdown bulk import** — `POST /projects/{id}/stories/import` → `markdown.ParseStoryMarkdown` (`backend/internal/adapter/markdown/parser.go`) → `StoryService.Import` (`backend/internal/domain/service/story_import.go`). Body is one raw markdown string; `---`-delimited blocks with frontmatter (`key, epic, depends_on, scope, status`) + H1 title + body→`acceptance_criteria`. Idempotency = `GetByKey` then Create-or-Update.

Confirmed import gaps: **`epic` frontmatter is parsed then dropped** (`EpicID` never set); **`objective` and `target_files` are not settable via import**; the import path bypasses `StoryService.Create`'s key-regex validation; there is **no structured bulk endpoint** (only the markdown blob).

### 2.3 Agent story-consumption (from `map:agent-story-consumption`)

`AgentRunAction.Execute` (`backend/internal/adapter/action/agent_run.go`):

1. `storyRepo.GetByID` → full `model.Story`.
2. Build `model.TemplateContext` (Handlebars vars): `{{story_key}}`, `{{story_title}}`, `{{story_objective}}`, `{{acceptance_criteria}}`, `{{target_files}}`, plus run/project meta (`{{branch_name}}`, `{{repo_url}}`). `{{diff_content}}` is declared but never set for `agent_run`.
3. Render `agents.template_content` → `PROMPT` / `PROMPT_CONTENT` env.
4. `buildClaudeMD(project, role, story)` (`agent_run.go:194`) → `CLAUDE_MD_CONTENT` (written by agent-runtime to `/workspace/repo/.claude/CLAUDE.md`). Injects only `project.{Name,Description,RepoURL}` + `story.{Key,Title}` + role framing. **Not injected:** objective, AC, target_files, `depends_on`, `scope`, epic goal.
5. Prior-failure context appended to CLAUDE.md on cross-run retries.

**Minimum a story needs to be implementable well:** `objective` (primary "what"), `acceptance_criteria` (definition of done), `target_files` (scope guide), `title`. Everything else is auto-populated by the pipeline. Today these live only in the rendered `PROMPT`; if the agent template is empty, the agent sees nothing.

---

## 3. Connector architecture

### 3.1 Input-adapter pattern (one port, N adapters)

Mirror the existing hexagonal layout (`GitProvider`, `AgentRuntime`, `Notifier` — port in `domain/port`, impls in `adapter/*`). Add **one port** and **one adapter per source**.

**New port** — `backend/internal/domain/port/planning_source.go`:

```go
package port

// SourceRef is the stable external identity of an imported item.
type SourceRef struct {
    System     string // "github" | "gitlab" | "jira" | "bmad" | "markdown" | "manual"
    ExternalID string // node_id (GH) | issue key (Jira) | global id (GL) | file#anchor (md)
    URL        string // deep link back to the origin
}

// ImportedEpic / ImportedStory are the *normalized* shapes an adapter emits.
// They map 1:1 onto model.Epic / model.Story plus the source ref + raw payload.
type ImportedStory struct {
    Ref                SourceRef
    Key                string   // derived/normalized story key (project-unique)
    EpicRef            *SourceRef
    Title              string
    Objective          *string
    AcceptanceCriteria *string
    Scope              *string
    DependsOn          []SourceRef // resolved to keys at upsert time
    TargetFiles        []string
    Status             string
    Labels             []string
    RawBody            string          // original markdown/ADF/issue body — kept for enrichment
    Enriched           bool            // true for BMAD/GSD rich specs → skip enrichment gate
    Extra              map[string]any  // priority, assignee, sprint, estimate (source-specific)
}

type ImportResult struct {
    Created, Updated, Skipped int
    Errors                    []error // partial success; per-item, never aborts the batch
}

// PlanningSourceAdapter normalizes one external source into the canonical model.
// Import is read-only/one-way in v1. WriteBack is optional and added later.
type PlanningSourceAdapter interface {
    System() string
    Fetch(ctx context.Context, cfg SourceConfig) ([]ImportedEpic, []ImportedStory, error)
    Normalize(raw []byte) ([]ImportedEpic, []ImportedStory, error) // for webhook/file payloads
    // WriteBack(ctx, ref SourceRef, status string, runURL string) error  // PHASE 3 (optional)
}
```

**Adapters** (one package each under `backend/internal/adapter/`):

| Adapter | Package | Notes |
|---|---|---|
| GitHub | `adapter/planning/github` | reuse `adapter/github` (gh CLI / App token); GraphQL for ProjectV2, REST for sub-issues |
| GitLab | `adapter/planning/gitlab` | GraphQL Work Items for epic tree; scoped labels → status/scope |
| Jira | `adapter/planning/jira` | REST v3 + Agile API; resolve `customfield_*` IDs at boot |
| BMAD/markdown | `adapter/markdown` (extend) | already exists; promote to full adapter, stop discarding fields |

A thin **`PlanningImportService`** (`backend/internal/domain/service/planning_import.go`) orchestrates: pick adapter by `System()` → `Fetch`/`Normalize` → (optional) enrichment → `StoryRepository.UpsertBySourceRef`. The service depends only on the port + `StoryRepository` + `EpicRepository` (boundary rules respected).

### 3.2 Idempotent import keyed by external id

Today the upsert key is `(project_id, key)` — fine for markdown where the author owns the key, wrong for external trackers where the same item must survive re-import. Add `(project_id, source, external_id)` as the *source-of-record* idempotency key, with `key` derived deterministically:

```sql
INSERT INTO stories (project_id, source, external_id, source_url, key, title, objective, ...)
VALUES (...)
ON CONFLICT (project_id, source, external_id) DO UPDATE
SET title = EXCLUDED.title,
    objective = EXCLUDED.objective,
    acceptance_criteria = EXCLUDED.acceptance_criteria,
    depends_on = EXCLUDED.depends_on,
    source_url = EXCLUDED.source_url,
    -- never clobber an enriched/approved spec with a raw re-pull:
    enriched_spec = COALESCE(stories.enriched_spec, EXCLUDED.enriched_spec),
    synced_at = now(),
    updated_at = now();
```

Per-source external_id choice (must be stable across renames/transfers):

- **GitHub:** issue `node_id` (global; never `number`, which is repo-scoped and reusable).
- **GitLab:** global `id` (not `iid`, which is project-scoped).
- **Jira:** issue `key` (stable per site, e.g. `PROJ-42`).
- **BMAD/markdown:** `relative/path.md#key` (git-native, diffable).

For `source = manual` / legacy markdown without an external id, fall back to the existing `(project_id, key)` constraint. Both indexes coexist.

### 3.3 One-way first, write-back later

- **v1 — one-way inbound.** External tracker is the source of truth for planning; we only read. No conflict resolution, no webhook infra required (poll or on-demand "Import" button). Lowest risk, 90% of the value (visibility on our board, agent-ready stories).
- **v2 — webhook delta.** Add `POST /webhooks/{source}` endpoints, HMAC verify (`X-Hub-Signature-256` / `X-Gitlab-Token` / Jira shared secret), dedup on delivery id (`X-GitHub-Delivery`), call `adapter.Normalize` → upsert. GitHub `projects_v2_item.edited` and GitLab issue hooks carry the field delta inline — no follow-up query. Keep a nightly full resync as a consistency backstop.
- **v3 — status write-back.** ✅ **Livré (GitHub Projects v2)** — connecteur persisté (`GET/PUT /projects/{id}/planning/connector`) avec mapping explicite des 4 statuts internes vers des options GitHub, propagation automatique à chaque transition du pipeline, commentaire optionnel avec lien du run, `writeback_status` sur la story (`disabled|pending|synced|failed`), badge `WritebackStatusBadge` dans le détail de story. Accès owner/admin. Erreurs documentées (`PLANNING_CONNECTOR_NO_GIT_CONNECTION`, `PLANNING_CONNECTOR_INVALID_MAPPING`). Markdown write-back et GitLab/Jira write-back restent futurs (non livrés).

### 3.4 Where it lives (hexagonal placement)

```
domain/port/planning_source.go          ← new port (interface only)
domain/service/planning_import.go       ← orchestration (adapter-agnostic)
adapter/planning/github/                ← new adapter
adapter/planning/gitlab/                ← new adapter
adapter/planning/jira/                  ← new adapter
adapter/markdown/                       ← extend existing parser into a full adapter
api/handler/planning_handler.go         ← POST /planning/import, POST /webhooks/{source}
cmd/api/wire.go                         ← register provider set (one adapter binding per source)
```

Import direction stays `handler → service → port ← adapter`. Adapters never touch business logic; the service never imports an adapter package (wired via go-wire).

---

## 4. Per-source mapping tables

Internal target = `model.Story` fields: `key, title, objective, scope, acceptance_criteria, depends_on, target_files` (+ new `source, external_id, source_url`). "→ agent" = what ultimately reaches the container via `PROMPT` / `CLAUDE_MD_CONTENT`.

### 4.1 GitHub → internal → agent

Sources: [Using the API to manage Projects](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-api-to-manage-projects) · [REST sub-issues](https://docs.github.com/en/rest/issues/sub-issues) · [REST issue types](https://docs.github.com/en/rest/orgs/issue-types) · [Webhook events](https://docs.github.com/en/webhooks/webhook-events-and-payloads) · [projects_v2_item field_value (Jun 2024)](https://github.blog/changelog/2024-06-27-github-issues-projects-graphql-and-webhook-support-for-project-status-updates-and-more/)

| Internal field | GitHub source | Extraction | → agent |
|---|---|---|---|
| `external_id` | `issue.node_id` (global) | GraphQL/REST | write-back anchor; CLAUDE.md "source" line |
| `source_url` | `issue.url` | direct | CLAUDE.md deep link |
| `key` | `S-<n>` derived from `issue.number` | normalize to `^[A-Z0-9]+-\d+$` | `{{story_key}}` |
| `title` | `issue.title` | direct | `{{story_title}}`, storyRef |
| `objective` | `## Context` / `## Background` H2 of `body` | markdown section parse | `{{story_objective}}` |
| `acceptance_criteria` | `## Acceptance Criteria` H2 of `body` (+ `- [ ]` lines) | markdown section parse | `{{acceptance_criteria}}` |
| `scope` | label `backend`/`frontend`/`shared` or issue `type` | label/type map | (board only today; → CLAUDE.md after P1) |
| `depends_on` | `GET /issues/{n}/sub_issues` + `parent` | hierarchy traversal | scheduling; epic goal → CLAUDE.md |
| `target_files` | parsed file paths in body (if present) | regex | `{{target_files}}` |
| EpicRef | issue with type `Epic` (or label) | `issue.type.name == "Epic"` | epic.name → CLAUDE.md |
| `labels`/`Extra` | `labels[]`, `fieldValues[Status/Iteration/StoryPoints]` | ProjectV2 GraphQL (inline `fieldValues`) | board filtering, status |

Hierarchy: Epic-typed issue → epic; its sub-issues → stories; their sub-issues → tasks (cap 2–3 levels). Auth: **GitHub App** (org-scope webhooks for `projects_v2_item`, 1h tokens, per-install rate limits). Keep `first ≤ 20` per GraphQL connection ([resource limits, Sep 2025](https://github.blog/changelog/2025-09-01-graphql-api-resource-limits/)).

### 4.2 GitLab → internal → agent

Sources: [Issues API](https://docs.gitlab.com/api/issues/) · [Epic→WorkItem migration](https://docs.gitlab.com/api/graphql/epic_work_items_api_migration_guide/) · [GraphQL API](https://docs.gitlab.com/api/graphql/) · [Labels (scoped)](https://docs.gitlab.com/user/project/labels/) · [Group webhooks](https://docs.gitlab.com/api/group_webhooks/)

| Internal field | GitLab source | Extraction | → agent |
|---|---|---|---|
| `external_id` | issue global `id` (not `iid`) | GraphQL/REST | write-back anchor |
| `source_url` | `issue.web_url` | direct | CLAUDE.md deep link |
| `key` | `S-<iid>` derived | normalize | `{{story_key}}` |
| `title` | `issue.title` | direct | `{{story_title}}` |
| `objective` | `description` (full markdown) | section parse | `{{story_objective}}` |
| `acceptance_criteria` | `- [ ]`/`- [x]` task list in `description` | regex `- \[(x| )\] (.+)`; `taskCompletionStatus` for progress | `{{acceptance_criteria}}` |
| `scope` | scoped label `scope::backend` etc. | scoped-label parse | (P1 → CLAUDE.md) |
| `status` | scoped label `workflow::*` | scoped-label parse (mutually exclusive per scope) | board column |
| `depends_on` | `WorkItemWidgetHierarchy` / epic→issue | GraphQL Work Items | scheduling |
| EpicRef | group-level Epic (work item) | `workItems(types:[EPIC])` | epic.name → CLAUDE.md |
| `Extra` | `weight`, `health_status`, `milestone`, `assignees[]` | direct | board metadata |

Use **Work Items GraphQL** (epic REST deprecated in 17.0, removal in v5; old epic GraphQL removal 19.0). Auth: **Group Access Token** (`read_api` for import, `api` for write-back). Epics + group webhooks require **Premium/Ultimate** — free-tier fallback: milestones as epics + project-level webhooks.

### 4.3 Jira → internal → agent

Sources: [Issues REST v3](https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issues/) · [Epic Agile API](https://developer.atlassian.com/cloud/jira/software/rest/api-group-epic/) · [Epic Link → parent deprecation](https://community.developer.atlassian.com/t/deprecation-of-the-epic-link-parent-link-and-other-related-fields-in-rest-apis-and-webhooks/54048) · [Webhooks](https://developer.atlassian.com/cloud/jira/platform/webhooks/) · [OAuth 2.0 (3LO)](https://developer.atlassian.com/cloud/jira/platform/oauth-2-3lo-apps/)

| Internal field | Jira source | Extraction | → agent |
|---|---|---|---|
| `external_id` | `issue.key` (e.g. `PROJ-42`) | direct | write-back anchor |
| `source_url` | `…/browse/{key}` | construct | CLAUDE.md deep link |
| `key` | `issue.key` (already `^[A-Z0-9]+-\d+$`) | direct (Jira keys fit our regex!) | `{{story_key}}` |
| `title` | `fields.summary` | direct | `{{story_title}}` |
| `objective` | `fields.description` (ADF) | walk `content[]`, collect `text` nodes → plain text | `{{story_objective}}` |
| `acceptance_criteria` | `customfield_NNNNN` (AC) if present, else from description | resolve field id via `GET /field` at boot; ADF→text | `{{acceptance_criteria}}` |
| `depends_on` | `fields.parent` + issue links | `parent.key`; `hierarchyLevel` (1=epic,0=story,-1=subtask) | scheduling |
| EpicRef | parent with `issuetype.hierarchyLevel == 1` | direct | epic.name → CLAUDE.md |
| `scope`/`labels` | `fields.labels[]` | direct | board metadata |
| `Extra` | `priority`, `assignee.accountId`, Sprint/Story-Points `customfield_*` | resolve ids at boot | board sort/filter |

Jira has **no native AC field** — resolve a per-workspace `customfield_*` id by name match at boot, else fall back to description. ADF must be flattened to plain text/markdown. Auth: **OAuth 2.0 3LO** (read scopes + `manage:jira-webhook`). Webhooks expire after 30 days → cron `PUT /webhook/refresh` every ~25 days.

### 4.4 BMAD / markdown → internal → agent (the rich path)

Sources: [BMAD-METHOD](https://github.com/bmad-code-org/BMAD-METHOD) · [BMAD docs](https://docs.bmad-method.org/) · [GSD file format](https://deepwiki.com/gsd-build/get-shit-done/7-file-format-reference) · local refs: `_bmad-output/implementation-artifacts/2-8-story-import-from-markdown.md`, `_bmad-output/planning-artifacts/epics.md`

| Internal field | BMAD/markdown source | Extraction | → agent |
|---|---|---|---|
| `external_id` | `path.md#key` | construct (git-native) | provenance |
| `source_url` | repo path | construct | CLAUDE.md "spec at" |
| `key` | frontmatter `key` (e.g. `S-2-8`) | YAML | `{{story_key}}` |
| `title` | `# Story {e}.{s}: {Title}` (H1) | heading | `{{story_title}}` |
| `objective` | `## Story` (As a…/I want…/So that…) | section | `{{story_objective}}` |
| `acceptance_criteria` | `## Acceptance Criteria` (BDD Given/When/Then) | section | `{{acceptance_criteria}}` |
| `scope` | frontmatter `scope` | YAML | board + (P1) CLAUDE.md |
| `depends_on` | frontmatter `depends_on: [S-2-3,…]` | YAML | scheduling |
| `target_files` | `### File Paths (exact)` block | fenced/section parse | `{{target_files}}` |
| EpicRef | frontmatter `epic` (+ `docs/prd/epic-list.md`) | YAML + resolve | epic.goal → CLAUDE.md |
| `Enriched = true` | status `ready-for-dev` + Dev Notes present | flag | **skip enrichment gate** |
| (extra context) | `### Architecture Requirements`, `### Technical Specifications`, `### Testing Requirements`, `### References` | sections | feed into `PROMPT`/CLAUDE.md as a rich context block |

This is the highest-fidelity source: a BMAD story at `ready-for-dev` is *the plan, not a pointer to a plan* (~8k tokens, fits one context window). GSD `PLAN.md` parallels it: `<done>`→AC, `<files>`→target_files, `wave`→parallel group, `depends_on`→deps. **Our current markdown parser must stop discarding** `epic` (→ resolve to `EpicID`), `objective`, and `target_files`.

---

## 5. The agent link

### 5.1 How an imported issue becomes agent-ready context

The agent consumes a story through exactly two channels in `agent_run.go`: the Handlebars `TemplateContext` (→ `PROMPT`) and `buildClaudeMD` (→ `CLAUDE_MD_CONTENT`). An imported item is "agent-ready" only when those channels are fully fed. Concretely:

**Thin issue (GitHub/Jira/GitLab one-liner) → enrichment, then execution:**

```
ImportedStory{RawBody, Enriched:false}
        │  upsert into stories (enrichment_status = 'raw')
        ▼
[Enrichment River job]                       ← NEW, runs in adapter/river
   reads: RawBody + repo CLAUDE.md + recent git log + api/openapi.yaml
   produces: objective, acceptance_criteria (Given/When/Then),
             scope, target_files, ambiguity_score
   writes:  stories.enriched_spec (jsonb), enrichment_status
        │
   ambiguity_score > 0.4 ──► enrichment_status='needs_review'
        │                     surfaced on board as HITL gate (reuses hitl_gate)
        ▼ (approved)
[agent_run.go] builds TemplateContext + CLAUDE_MD_CONTENT
   from the enriched fields, NOT the raw issue body
```

The enrichment agent is **not** the coding agent — keep the concerns split (one enriches/plans, another implements). Store the enrichment rationale (why these files) as the audit trail. This matches the SWE-bench finding that the highest-leverage pre-execution intervention is "augment the issue with explicit requirements + affected files".

**Keep a link to the source.** Every story carries `source`, `external_id`, `source_url`. The agent should *see* it: add a line to `buildClaudeMD` — `Source: {source} {external_id} ({source_url})` — so a stuck agent can (when given a read tool) trace back to the original ticket/comments. It is also the write-back anchor.

### 5.2 Story fields the agent actually needs — and the concrete `agent_run.go` changes

The minimal viable package (`objective`, `acceptance_criteria`, `target_files`, `title`) is already wired into `TemplateContext`. The gaps are: these only reach the agent if the *template* references them, and several useful fields never reach the agent at all. Two changes:

1. **Thicken `buildClaudeMD`** (`agent_run.go:194`) so the agent is grounded even with a minimal/empty prompt template. Add, after the project/role block:
   - story `objective` + `acceptance_criteria` (currently CLAUDE.md has *neither*),
   - `target_files` and `scope` (constrains blast radius — `scope` is currently dead in prompts),
   - epic `name`/`goal` (fetch the epic; the agent currently has no sense of the larger goal),
   - a `depends_on` summary line (which sibling stories ran before and what they produced),
   - the `Source:` provenance line.

2. **Pass an `enriched`/rich-context block** into `TemplateContext` for BMAD/GSD stories so `### Architecture Requirements` / `### Technical Specifications` reach the prompt (new optional `{{dev_notes}}` var), without bloating thin-issue prompts.

### 5.3 Rich BMAD docs feed agents directly (skip scoping)

When `ImportedStory.Enriched == true` (BMAD `ready-for-dev` / GSD plan), **bypass the enrichment job entirely**: set `enrichment_status = 'approved'` on import and populate `objective`/`acceptance_criteria`/`target_files` straight from the parsed sections. The story is launch-ready the moment it lands. The only gate is the frontmatter `status` (only `ready-for-dev`/`backlog` need human review before launch). This is the payoff of the "rich connector skips scoping" stance — markdown/BMAD goes import → board → run with no LLM round-trip and no HITL, while GitHub/Jira thin issues go import → enrich → (gate) → run.

---

## 6. Necessary board improvements

1. **Source attribution + external id on stories.** Add `source`, `external_id`, `source_url` to `model.Story`, the `stories` table, the OpenAPI schema, and the story API responses. Unblocks everything below.
2. **"Planned in" provenance — backed for real.** Replace the `git_provider` heuristic in `BoardView.vue:38-74` with a real `planning_source` (per-epic or per-project, derived from the `source` of its stories). Render the design's **source badge** on each card (GitHub/GitLab/Jira/BMAD/Markdown icon) linking to `source_url`.
3. **Re-sync affordance.** An "Import / Re-sync" action per epic (and later a webhook-driven auto-sync indicator) that calls `POST /planning/import`. Because upsert is idempotent on `(project_id, source, external_id)`, re-sync is always safe.
4. **Dependency & context surfacing.** Show `depends_on` on the card (blocked-by chips) and surface the enrichment state: a `raw`/`needs_review`/`approved` badge, with `needs_review` rendered as a HITL gate the user can open (reuse the existing In-Review column + approval view).
5. **Provenance on the epic.** Epic-level source + link, plus optional `priority`/`labels` columns/filters for stories (currently absent), enabling board filtering by source/label.

---

## 7. Prioritized roadmap

References real paths. Each item is a concrete change.

### P0 — make import traceable & rich-path lossless (foundation)

- **Migration** `000032_add_source_to_stories`: add `source VARCHAR(20)`, `external_id VARCHAR(255)`, `source_url TEXT`, `synced_at TIMESTAMPTZ` to `stories`; add partial unique index `(project_id, source, external_id) WHERE external_id IS NOT NULL`. Backfill existing rows to `source='manual'`.
- **Model** `backend/internal/domain/model/story.go`: add `Source`, `ExternalID *string`, `SourceURL *string`, `SyncedAt *time.Time`.
- **Repo** `port/story_repository.go` + `adapter/postgres`: add `UpsertBySourceRef`; sqlc query in `queries/`.
- **Fix the markdown adapter** (`adapter/markdown/parser.go` + `service/story_import.go`): stop discarding `epic` (resolve label → `EpicID`, create-or-find epic by name), parse `objective` from `## Story`, parse `target_files` from `### File Paths (exact)`, set `Enriched=true` for `ready-for-dev`. This is pure value, no new source needed.
- **OpenAPI** `api/openapi.yaml`: add the three fields to the Story schema (single source of truth) → regenerate handlers.
- **Agent context (cheap win)** `agent_run.go:194` `buildClaudeMD`: inject `objective` + `acceptance_criteria` + `target_files` + `scope` + epic name + `Source:` line.

### P1 — connector port + first adapter + enrichment

- **Port** `domain/port/planning_source.go` (`PlanningSourceAdapter`, `SourceRef`, `ImportedStory`).
- **Service** `domain/service/planning_import.go` orchestrating fetch→(enrich)→upsert.
- **First adapter** `adapter/planning/github/` (highest demand; reuse `adapter/github` auth). Map per §4.1, upsert on `node_id`.
- **Handler** `api/handler/planning_handler.go`: `POST /projects/{id}/planning/import` (one-way, on-demand). Wire in `cmd/api/wire.go`.
- **Enrichment River job** `adapter/river/` + `model.Story` fields `enriched_spec JSONB`, `enrichment_status`, `ambiguity_score`; raw GitHub issues enqueue enrichment; `>0.4` → `needs_review` (reuse `hitl_gate`).
- **Frontend** `BoardView.vue`: replace heuristic with real `planning_source`; render source badge + `source_url` link on cards (`features/board/KanbanBoard.vue`).

### P2 — more sources, webhooks, write-back, board polish

- **Adapters** `adapter/planning/gitlab/` (Work Items GraphQL, scoped labels) + `adapter/planning/jira/` (REST v3 + Agile, ADF flattening, boot-time field id resolution).
- **Webhooks** `POST /webhooks/{source}` (HMAC verify, dedup on delivery id, `adapter.Normalize` → upsert) + nightly resync backstop (River cron).
- **Status write-back** `PlanningSourceAdapter.WriteBack`: on run done/blocked, comment+label (GH), transition+comment (Jira), `workflow::done` label (GL). Store write-back events for audit.
- **Board polish**: `depends_on` blocked-by chips, enrichment-state badge, `priority`/`labels` filters, epic-level provenance.
- **Agent context (rich)**: optional `{{dev_notes}}` `TemplateContext` var for BMAD/GSD `### Architecture/Technical Specifications` blocks.

**Through-line check:** P0 makes the canonical store traceable and the rich (BMAD/markdown) path lossless + agent-grounded; P1 lights up the first external source end-to-end (import → enrich → board → agent); P2 generalizes to all sources, adds real-time sync, and closes the loop with optional write-back.
