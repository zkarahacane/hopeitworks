# Agent Orchestration & Capabilities — Implementation Plan

> Scope: the next tier of agent capabilities for hopeitworks — the execution layer that runs role-based AI coding agents in Docker containers, driven by a configurable pipeline (groups → steps) with HITL gates.
> Grounded in the current code: `backend/internal/domain/service/pipeline_executor.go`, `backend/internal/adapter/action/agent_run.go`, `backend/internal/adapter/action/incremental_retry.go`, `backend/internal/adapter/action/hitl_gate.go`, `backend/internal/domain/service/hitl_service.go`, `backend/internal/domain/model/pipeline_config.go`, `backend/internal/domain/model/run_context.go`, `backend/internal/domain/model/template_context.go`, `agent-runtime/internal/{config,runner,callback}/`, `agent-images/`.

---

## 1. Executive summary

The runtime works end-to-end: agents clone, implement, build/test, commit, and open a PR. Agents already receive role context (`buildClaudeMD`), a verification mandate (the "Definition of done" block), and passive prior-failure memory (`priorFailureContext`). What's missing is the loop that makes a multi-agent pipeline actually better than a single agent: **the reviewer's findings are never consumed.**

The single biggest unlock is the **review→fix feedback loop**. Today a `role: review` step's output lands only in `run_steps.log_tail` and the SSE stream, then the executor moves to the next step or marks the run complete (`pipeline_executor.go:144-184`). There is no structured verdict, no path to re-invoke the implementer with the findings, and no representation of "soft block" — the reviewer must exit non-zero to fail the run, which is a blunt, terminal signal. Closing this loop is where the research consensus is strongest: actor-critic loops with an **isolated** critic eliminate 90%+ of issues in 3–5 rounds, and Multi-Agent Reflexion reports +6.2 points Pass@1 over single-agent Reflexion on HumanEval ([understandingdata.com](https://understandingdata.com/posts/actor-critic-adversarial-coding/), [MAR — arxiv.org/html/2512.20845](https://arxiv.org/html/2512.20845)). Our reviewer is already session-isolated (a fresh container that reads the diff via git) — we have the hard half; we're missing the cheap half (wiring).

The unlocks, in priority order:

1. **Review→fix loop (P0, headline).** A structured reviewer verdict + a `loop`/`fix` pipeline primitive that re-runs the implementer with the findings injected, bounded by max-iterations, with oscillation detection, escalating to the **existing** HITL gate. This is mostly backend plumbing over primitives we already have (the metadata bus, `RegisterAlias`, the `errStepSuspended` suspend mechanism, `incremental_retry`'s child-step pattern).

2. **Typed handoff between agents (P0/P1).** The metadata bus carries only infrastructure facts (`branch_name`, `pr_url`). Add a `review_findings` artifact so an upstream agent's structured output flows to a downstream step — the precondition for both the loop and any future supervisor routing ([Anthropic multi-agent](https://www.anthropic.com/engineering/multi-agent-research-system)).

3. **HITL feedback payload (P0).** Today `Approve`/`Reject` carry only a `user_id` and an unused reason string (`hitl_service.go:174-204`). A human reject should inject its reason into metadata as `error_context` so the next implementer run sees it. Cheap, high-leverage.

4. **P1 capabilities** that compound the loop: **LSP diagnostics** as a mandatory post-edit step (machine-checked facts instead of token-expensive grep), **Agent Skills per role** baked into images (progressive disclosure of the review checklist and the fix protocol), **MCP** exposing our kanban + GitHub to agents, and **cross-sprint memory** (pgvector) to replace the single-failure `priorFailureContext` with durable, multi-signal project memory.

The through-line from Anthropic's guidance: **start simple, code decides when to stop (not the LLM), instill heuristics with explicit guardrails, and scale effort to task stakes** ([Building Effective Agents](https://www.anthropic.com/research/building-effective-agents)). Our loop design follows that — a deterministic policy/router in Go, not an LLM supervisor.

---

## 2. The review→fix loop (centerpiece)

### 2.1 The gap, precisely

- `agent_run` (role=review) writes **nothing** to `RunContext.Metadata`. Its only persisted artifact is `run_steps.log_tail` (`agent_run.go:432-433`).
- The executor is a fixed linear walk over `step_order` (`pipeline_executor.go:119-159`). The only non-linear signal is `errStepSuspended`, which pauses the *whole* run for HITL — there is no "re-run step N".
- `TemplateContext.DiffContent` exists (`template_context.go:21`) but `agent_run.go` never populates it; the reviewer fetches its own diff via `git diff origin/main...HEAD` (`buildClaudeMD` reviewer branch).
- HITL approval/reject injects no feedback into metadata (`hitl_service.go`).

So we need four things: (a) a structured reviewer output, (b) a channel to carry it (metadata), (c) a pipeline primitive that loops back to a fix agent, and (d) bounded escalation into the existing HITL gate.

### 2.2 Structured review output

The reviewer must emit a machine-actionable verdict, not prose. Add a **findings callback** to the agent-runtime so the reviewer reports structured JSON the same way it reports logs/cost/status today.

**Schema** (`backend/internal/domain/model/review_findings.go`, new):

```json
{
  "verdict": "approved | needs_fix | escalate",
  "iteration": 2,
  "open_blocking_count": 1,
  "findings": [
    {
      "id": "F-001",
      "severity": "critical | major | minor",
      "category": "security | correctness | performance | style | test_coverage",
      "file": "backend/internal/adapter/action/agent_run.go",
      "line_range": [311, 314],
      "description": "API key appended to env without redaction guard.",
      "suggested_fix": "Resolve via apiKeySvc only; never log env.",
      "blocking": true,
      "regression_from_prev_iter": false
    }
  ]
}
```

Design decisions, each tied to a source:

- **Stable `id` across iterations** so the fixer (and the router) can track which findings were addressed and detect re-raises — hash-based dedup prevents re-raising identical findings as "new" ([dev.to — Stop the Loop](https://dev.to/alessandro_pignati/stop-the-loop-how-to-prevent-infinite-conversations-in-your-ai-agents-ekj)).
- **`verdict: "escalate"` is first-class**, not an error path ([Anthropic — Measuring Agent Autonomy](https://www.anthropic.com/research/measuring-agent-autonomy)).
- **`regression_from_prev_iter`** flags oscillation (reviewer flags something the fixer just introduced) ([dev.to — Stop the Loop](https://dev.to/alessandro_pignati/stop-the-loop-how-to-prevent-infinite-conversations-in-your-ai-agents-ekj)).
- **Structure the critic's reasoning before the JSON** (premises → trace → conclusion) — Meta's semi-formal prompting materially improves review precision ([VentureBeat](https://venturebeat.com/orchestration/metas-new-structured-prompting-technique-makes-llms-significantly-better-at)). This lives in the review Skill (§4), not in code.

**How it's emitted.** Add `SendFindings` to `agent-runtime/internal/callback/client.go` (mirrors `SendStatus`), posting to a new `POST /internal/agent/callback/runs/{run}/steps/{step}/findings` endpoint. The reviewer writes the JSON to a known path (`/workspace/review-findings.json`); `runner.go` reads it after the provider loop and calls `SendFindings`. The API handler validates and stores it on the run step (new nullable `run_steps.review_findings JSONB` column).

### 2.3 How findings flow via run metadata

After the review step completes, the executor must surface the findings on the metadata bus so the next construct can act on them. In `pipeline_executor.go:executeStep`, after `action.Execute` and before persisting metadata, re-read the step and, if `review_findings` is set, write:

```go
runCtx.Metadata["review_findings"]    = findingsJSON   // full structured object
runCtx.Metadata["review_verdict"]     = verdict        // "needs_fix" | "approved" | "escalate"
runCtx.Metadata["open_blocking_count"] = blockingCount
```

This is exactly the persistence path already used for `branch_name` — `UpdateRunMetadata` runs after each step (`pipeline_executor.go:277`) and survives a HITL suspend/resume, so the findings outlive a pause. This is the **typed handoff** the codebase lacks today: `agent_run` finally writes to the bus.

### 2.4 The new pipeline construct: a `loop` group + a `fix` action

Two complementary additions. Both are config-level and reuse existing machinery.

**(a) `fix` role for `agent_run`.** No new action type needed — `agent_run` with `config.role: "fix"`. Extend `buildClaudeMD` (`agent_run.go:194`) with a `case "fix":` that frames the agent as a fixer and instructs it to address findings **by ID**. In `agent_run.Execute`, when `review_findings` is present in metadata, render it into the prompt: populate `TemplateContext.DiffContent` (currently always empty) and a new `TemplateContext.ReviewFindings` field so the fix template can do `{{#each review_findings}}`. Per the research, the fixer's default is a **fresh session** receiving only the diff + requirements + findings — the implementer's reasoning is noise ([MindStudio](https://www.mindstudio.ai/blog/automated-code-review-multiple-ai-agents)). Our containers are already fresh sessions, so this falls out naturally.

**(b) A `loop` construct in the pipeline config.** Add an optional `loop` block to `PipelineGroup` in `pipeline_config.go`:

```yaml
groups:
  - id: implement-review-fix
    name: Implement / Review / Fix
    loop:
      max_iterations: 3            # soft cap
      hard_max: 5                  # hard ceiling
      continue_while: "review_verdict == needs_fix"
      on_exhausted: hitl_gate      # escalate to the existing human gate
    steps:
      - id: implement
        action_type: agent_run
        agent_id: <impl-agent>
        config: { role: dev }
      - id: review
        action_type: agent_run
        agent_id: <review-agent>
        config: { role: review }
      - id: fix
        action_type: agent_run
        agent_id: <fix-agent>
        config: { role: fix }      # only runs when verdict == needs_fix
```

**Executor support.** Add loop awareness to `ExecuteRun` (`pipeline_executor.go`). The cleanest fit with the current model — which materializes steps as `run_steps` rows and walks them by `step_order` — is to **materialize iterations as sibling steps**, reusing the `incremental_retry` pattern that already creates child `RunStep` rows with `ParentStepID` and `retry_count+1` (`incremental_retry.go:82-96`):

1. After the `review` step, the executor checks `review_verdict`.
2. `approved` → break the loop, jump past the loop group's remaining steps to the next group.
3. `needs_fix` and `iteration < max_iterations` → run the `fix` step (already in the group), then re-materialize a fresh `review` step (new `run_steps` row, `iteration = n+1`, `parent_step_id` = the loop anchor) and continue. The fix→review pair repeats.
4. `escalate`, or `iteration >= max_iterations` with `open_blocking_count > 0`, or oscillation → trigger `on_exhausted` (the HITL gate).

Loop state (`iteration`, the per-iteration findings hashes) lives in run metadata so it survives suspend/resume. This keeps the executor's single-pass-over-steps invariant intact while expressing a bounded loop — "code decides when to stop, not the LLM" ([LangGraph supervisor](https://dev.to/focused_dot_io/multi-agent-orchestration-in-langgraph-supervisor-vs-swarm-tradeoffs-and-architecture-1b7e)), and it matches Google ADK's `LoopAgent` shape (a sub-agent sequence with an escalation/`max_iterations` exit) ([ADK Loop Agent](https://google.github.io/adk-docs/agents/workflow-agents/loop-agents/)).

**The router is deterministic Go, not an LLM.** Add a small `loopRouter` helper in `pipeline_executor.go` that reads metadata and returns one of `{continue_fix, approved_break, escalate}`. Severity routing and cap enforcement are code, eliminating the largest source of routing hallucination ([handoff research §7.3 — policy engine is code](https://www.anthropic.com/research/building-effective-agents)).

### 2.5 Max-iterations & termination (defense in depth)

Three layers, defaults to ship:

1. **Success gate (clean exit):** `review_verdict == approved` AND `open_blocking_count == 0`. Binary and observable — vague "looks good" criteria invite runaway loops ([MindStudio agentic loop](https://www.mindstudio.ai/blog/how-to-build-agentic-loop-claude-code)).
2. **Hard iteration cap:** `max_iterations: 3` soft, `hard_max: 5`. 3–5 rounds eliminate 90%+ of issues; >5 signals scope decomposition failure, not insufficient iteration ([understandingdata.com](https://understandingdata.com/posts/actor-critic-adversarial-coding/), [dev.to — Stop the Loop](https://dev.to/alessandro_pignati/stop-the-loop-how-to-prevent-infinite-conversations-in-your-ai-agents-ekj)).
3. **Budget gate:** reuse the existing `CostService` already wired into `agent_run` (`agent_run.go:439`). Add a per-loop token/USD ceiling; exceeding it fails **open to the HITL gate**, never silently skips ([self-improving agent — arxiv.org/html/2504.15228v1](https://arxiv.org/html/2504.15228v1)).

**Oscillation detection** (in `loopRouter`):
- Track finding `id`s per iteration in metadata. If `F-001` appears → disappears → reappears, hard-escalate ([dev.to — Stop the Loop](https://dev.to/alessandro_pignati/stop-the-loop-how-to-prevent-infinite-conversations-in-your-ai-agents-ekj)).
- Anti-recursion: a finding may be raised against the same `file`+`line_range` at most twice; the third raise forces `escalate`.
- Optional P2: cosine similarity on reviewer output embeddings (>0.92 between iter N and N-2 = semantic loop). Defer until pgvector memory (§4) lands, since it reuses the same embedding infra.

### 2.6 When it escalates to the existing HITL gate

`on_exhausted: hitl_gate` reuses `HITLGateAction` unchanged. The triggers, in priority order:

| Trigger | Source |
|---|---|
| `review_verdict == escalate` emitted by the reviewer | [Anthropic — autonomy](https://www.anthropic.com/research/measuring-agent-autonomy) |
| `open_blocking_count > 0` after `hard_max` iterations | [understandingdata.com](https://understandingdata.com/posts/actor-critic-adversarial-coding/) |
| Oscillation / anti-recursion tripped | [dev.to — Stop the Loop](https://dev.to/alessandro_pignati/stop-the-loop-how-to-prevent-infinite-conversations-in-your-ai-agents-ekj) |
| A `critical` finding persists ≥ 2 iterations | [handoff research §5](https://www.anthropic.com/research/building-effective-agents) |
| Budget exceeded | [self-improving agent](https://arxiv.org/html/2504.15228v1) |

When escalating, the executor sets `runCtx.Metadata["pr_url"]` (already done by `git_pr`) so `HITLGateAction` fetches the PR diff (`hitl_gate.go:62-82`), and additionally writes the **findings summary** into the HITL request so the human sees unresolved blockers + iteration history — extend the `HITLRequest` model with an optional `FindingsContext *string`.

**Close the human-feedback gap on the way back.** Today `Reject`'s reason string dies in the `hitl_requests` row (`hitl_service.go:191-198`) and `Approve` carries nothing. Change `HITLService.Reject` to, instead of (or in addition to) failing the step, **inject the reason into run metadata as `error_context`** and re-enqueue the loop so the next fix iteration consumes the human's instruction — exactly the channel `agent_run` already reads (`agent_run.go:131-136`). This turns the human into another critic in the same loop rather than a dead end.

### 2.7 End-to-end shape

```
implement (role=dev) ──commit/push──▶ review (role=review, fresh container)
                                            │ SendFindings → run_steps.review_findings
                                            ▼
                            executor.loopRouter (deterministic Go)
              ┌─────────────────────────────┼─────────────────────────────┐
        approved                        needs_fix                      escalate / cap / oscillation
        (break loop)        fix (role=fix) ← findings via metadata        HITL gate (existing)
                                            │                          findings summary + PR diff
                                            └──re-materialize review──┘   reject → error_context → re-loop
```

This is the actor-critic-with-isolated-critic pattern, with a deterministic router and the human gate as the escalation sink — matching Anthropic's "heuristics + explicit guardrails, effort scaled to stakes" guidance ([multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system)).

---

## 3. Handoff & escalation model for our role-based pipeline

The review→fix loop is the first instance of a general need: **typed handoff + deterministic routing + escalation**. Generalize it.

### 3.1 Typed handoff via a metadata artifact

Today the only handoff between agents is the git branch + two infra keys (`branch_name`, `pr_url`). Introduce a **structured briefing** on the metadata bus — not a transcript dump. Per the handoff research, the briefing carries decisions+rationale, artifact *references*, and findings with evidence; it **excludes** raw message history, and prior assistant output is re-cast as narrative context to avoid attribution hallucination (accuracy drops ~30%→11% over 3 hops with raw history) ([XTrace](https://xtrace.ai/blog/ai-agent-context-handoff), [AgentHallu — arXiv](https://arxiv.org/html/2601.06818v1)).

Concretely: a `handoff` key in `RunContext.Metadata` (`{decisions[], artifacts{diff_ref, test_results_ref}, findings[], open_questions[]}`). The diff and test results are *references* (PR URL, step IDs), not inline — keeping the bus small as chains grow ([Anthropic orchestrator-worker](https://www.anthropic.com/engineering/multi-agent-research-system)). `review_findings` (§2.2) is the first concrete instance.

### 3.2 Deterministic routing, not an LLM supervisor

Adopt the **supervisor (centralized)** model conceptually but implement the supervisor as **code in the executor**, not an LLM node. For coding pipelines, centralized routing wins on accuracy (~94% vs ~91%) and auditability, at the cost of latency we don't care about ([LangGraph supervisor vs swarm](https://dev.to/focused_dot_io/multi-agent-orchestration-in-langgraph-supervisor-vs-swarm-tradeoffs-and-architecture-1b7e)). Our `loopRouter` (§2.4) is exactly this. Keep `resolution_notes[]` in run metadata so the router never re-routes the same finding twice.

### 3.3 Escalation triggers (general)

An agent step escalates to the HITL gate rather than looping when:

| Trigger | Type | Action |
|---|---|---|
| N consecutive iterations without progress (N=3) | Stuck | HITL gate |
| `critical` finding persists ≥ 2 iterations | Policy | HITL gate |
| Irreversible action required (merge to main, deploy) | Guardrail | HITL gate (already how `git_pr` → `hitl_gate` is sequenced) |
| Oscillation / anti-recursion | Loop guard | HITL gate |
| Budget exceeded | Economic | HITL gate (fail-open) |
| Ambiguous scope (`verdict: escalate`) | Scope | HITL gate |

Every escalation carries *what was attempted, why it failed, what's needed, and artifact refs* — never a silent fail-forward ([handoff research §5](https://www.anthropic.com/research/building-effective-agents)).

### 3.4 Guardrails as code, scoped tools per role

Guardrails belong outside the LLM. Scope container capabilities per role:
- **dev/fix:** write access to the workspace, may commit/push.
- **review:** read-only — the reviewer should not be able to mutate the branch. Today `runner.go:84-94` commits whenever the working tree is dirty and exit code is 0; for a review role this should be **disabled** (a review container that edits files is a guardrail violation). Add a `READONLY=true` env for review/merge roles that skips `CommitAndPush`.
- **Irreversible actions** (merge to main) always pass through the HITL gate.

These map to the OpenAI Agents SDK guardrail placement model (input at entry, tool guardrails mid-chain, approval for irreversible ops) ([Guardrails — OpenAI Agents SDK](https://openai.github.io/openai-agents-python/guardrails/)) and the policy-as-code principle ([Agentic AI guardrails 2026](https://medium.com/@dewasheesh.rana/agentic-ai-in-production-designing-autonomous-multi-agent-systems-with-guardrails-2026-guide-a5a1c8461772)).

---

## 4. P1 capabilities

### 4.1 LSP / diagnostics in agent containers — effort **M**

**What:** Run a language server inside each stack image and expose `diagnostics`, `references`, `rename_symbol` to the agent. Wire a **mandatory post-edit diagnostic loop**: baseline → write → differential diagnostics → self-correct, with silent fallback to syntax-only if the LSP process fails ([Hermes Agent LSP](https://hermes-agent.nousresearch.com/docs/user-guide/features/lsp), [Codex CLI LSP](https://codex.danielvaughan.com/2026/04/25/codex-cli-lsp-integration-language-server-semantic-code-intelligence/)).

**Why:** LSP gives machine-checked semantic facts at a fraction of the token cost — find-references ~500 tokens + zero false positives vs ~2000 + noise for grep; differential type errors in <200ms vs running the full suite. The review role also benefits: a diff can be syntax-clean yet break types in files outside the diff — LSP catches those, grep can't.

**How (fits `agent-images/`):** ship `agent-lsp` (or `mcp-language-server`) per stack. For `agent-images/stacks/go/Dockerfile` add `gopls`; node → `typescript-language-server`; python → `pyright`. Mount the workspace at the fixed `/workspace/repo` path the runner already uses (`runner.go:40`), initialize the LSP index once at container start. Pre-warm caches at build time (`GOMODCACHE`, `npm_config_cache`) to avoid runtime network calls. The agent-runtime connects to the LSP over stdio/MCP. Tie to: [agent-lsp](https://www.agent-lsp.com/), [mcp-language-server pattern](https://amirteymoori.com/lsp-language-server-protocol-ai-coding-tools/).

### 4.2 Agent Skills per role — effort **S–M**

**What:** Bake `SKILL.md` + reference files + scripts into each image under `/skills/<role>/`, loaded by progressive disclosure (frontmatter always; body on trigger; reference files on demand) ([Anthropic Agent Skills](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview), [best practices](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices)).

**Why:** Today the entire role framing is a hardcoded Go string in `buildClaudeMD`. Skills move that into versioned, testable content with near-zero idle token cost, and let us encode the review checklist and the **fix protocol** (address findings by ID, run LSP after every edit) as mandatory `MUST`/`NEVER` steps rather than soft prose. Heavy rubrics defer to reference files (zero cost on a 2-line diff).

**How:** `agent-images/skills/{dev,review,fix}/`. `review/SKILL.md` encodes the structured-findings output format (§2.2) as a checklist so the reviewer doesn't short-circuit on trivial diffs; `fix/SKILL.md` encodes "address each finding by ID, acknowledge in the patch message." `agent-runtime` scans `AGENT_SKILLS_DIR` at startup and loads frontmatters — replicating Claude Code's `~/.claude/skills/` behavior inside the container. The `review/SKILL.md` checklist also runs LSP diagnostics on changed files as a final net (§4.1).

### 4.3 MCP for our kanban + GitHub — effort **M–L**

**What:** Expose hopeitworks's own data (stories, runs, comments) to agents via an internal MCP server, and give agents scoped GitHub access via MCP ([mark3labs/mcp-go](https://github.com/mark3labs/mcp-go), [GitLab/GitHub MCP](https://docs.gitlab.com/user/gitlab_duo/model_context_protocol/mcp_server/)).

**Why:** Lets a fix agent read the story's acceptance criteria, post a comment back to the kanban, or inspect related PRs without bespoke API glue — and turns the kanban into a live coordination surface for agents, matching the product's "kanban in-app" positioning.

**How:** A Go MCP server (`mark3labs/mcp-go`) wrapping the existing internal API: `get_stories`, `update_story_status`, `get_run`, `list_runs`, `create_comment`. **Transport: stdio with env-injected credentials** — agents run in Docker on the same daemon, so stdio is the right, lower-surface choice; the MCP spec itself says stdio implementations should use env credentials, not OAuth ([MCP authorization spec](https://modelcontextprotocol.io/specification/draft/basic/authorization), [stdio vs HTTP](https://www.truefoundry.com/blog/mcp-stdio-vs-streamable-http-enterprise)). Enforce **tool-level RBAC from day one** — `list_tools` filters by the role's token so a review agent never even *sees* `update_story_status` (visibility blocks prompt-injection paths) ([Maxim RBAC](https://www.getmaxim.ai/articles/mcp-rbac-tool-level-permissions-for-production-ai-agents/), [DEV.to tool scoping](https://dev.to/supertrained/tool-level-permission-scoping-in-mcp-why-server-authentication-isnt-enough-58ni)). GitHub access uses a project-scoped token, never account-wide; audit-log every tool call (`caller_id`, `tool_name`, `decision`).

### 4.4 Cross-sprint memory (pgvector) — effort **L**

**What:** Replace the single-failure `priorFailureContext` (`agent_run.go:589`) with durable, multi-signal project memory on the existing Postgres via pgvector ([mem0 — State of Memory 2026](https://mem0.ai/blog/state-of-ai-agent-memory-2026), [pgvector DBA guide](https://www.dbi-services.com/blog/pgvector-a-guide-for-dba-part-2-indexes-update-march-2026/)).

**Why:** `priorFailureContext` only surfaces the *last failed run's* error/log tail. Durable memory captures decisions, conventions, bug fixes, and patterns across sprints — the procedural/semantic memory that most helps a coding agent avoid repeating mistakes and respect project conventions.

**How:** New `agent_memories` table (project/agent/run scoped, `memory_type`, `content`, `embedding vector(1536)`, `confidence`, `expires_at`) with an **HNSW** index (better recall, no rebuild on insert, fine for <500K rows). Retrieval: multi-signal (semantic + BM25 + entity), filter by `project_id` **before** the vector search, top-K reranked by recency × confidence, synthesized into a concise session-start briefing injected into `CLAUDE_MD_CONTENT`. Consolidation runs **async as a River job** post-run (queue already in place) — extract facts, dedup, decay confidence by type (conventions slow, bug_fixes fast), prune low-confidence — i.e. a DIY of Anthropic's "Dreaming" ([VentureBeat — Dreaming](https://venturebeat.com/technology/anthropic-introduces-dreaming-a-system-that-lets-ai-agents-learn-from-their-own-mistakes)). Never store file contents, PII, or secrets — only synthesized patterns. Also unblocks the semantic oscillation detection deferred in §2.5.

---

## 5. Prioritized roadmap

### P0 — the review→fix loop and its preconditions (lead item)

| # | Change | Where | Effort |
|---|---|---|---|
| P0-1 | **Structured findings channel**: `SendFindings` in callback client; `POST .../findings` endpoint; `run_steps.review_findings JSONB` migration; `ReviewFindings` model + validation | `agent-runtime/internal/callback/client.go`, `agent-runtime/internal/runner/runner.go`, `backend/internal/api/handler/`, new migration, `backend/internal/domain/model/review_findings.go` | M |
| P0-2 | **Findings on the metadata bus**: after a review step, write `review_findings`/`review_verdict`/`open_blocking_count` into `RunContext.Metadata` (reuse the `UpdateRunMetadata` persistence already at `pipeline_executor.go:277`) | `backend/internal/domain/service/pipeline_executor.go`, `backend/internal/adapter/action/agent_run.go` | S |
| P0-3 | **`fix` role**: `case "fix"` in `buildClaudeMD`; populate `TemplateContext.DiffContent` + new `ReviewFindings` field; render findings into the fix prompt | `backend/internal/adapter/action/agent_run.go`, `backend/internal/domain/model/template_context.go` | S |
| P0-4 | **`loop` config + executor router**: `loop` block on `PipelineGroup`; `loopRouter` (deterministic Go) with max-iterations, oscillation/anti-recursion, budget gate; materialize fix→review iterations as sibling steps (reuse `incremental_retry` child-step pattern) | `backend/internal/domain/model/pipeline_config.go`, `backend/internal/domain/service/pipeline_executor.go` | L |
| P0-5 | **Escalate to existing HITL gate** on exhaustion/escalate; attach findings summary to the `HITLRequest` (`FindingsContext`) | `backend/internal/domain/service/pipeline_executor.go`, `backend/internal/adapter/action/hitl_gate.go`, `backend/internal/domain/model/hitl.go` | M |
| P0-6 | **HITL feedback payload**: `Reject` injects its reason into metadata as `error_context` and re-loops instead of dead-ending; surface it to the next fix iteration | `backend/internal/domain/service/hitl_service.go` | S |
| P0-7 | **Read-only review containers**: `READONLY` env skips `CommitAndPush` for review/merge roles (guardrail) | `agent-runtime/internal/runner/runner.go`, `agent-runtime/internal/config/config.go`, `backend/internal/adapter/action/agent_run.go` | S |

**Defaults to ship:** `max_iterations: 3`, `hard_max: 5`, isolated reviewer (already true), structured JSON verdict with stable IDs, hash-based oscillation tracking, escalate as a first-class verdict, budget ceiling fails open to HITL.

### P1 — capabilities that compound the loop

| # | Change | Where | Effort |
|---|---|---|---|
| P1-1 | **Agent Skills per role** baked into images; `agent-runtime` loads `AGENT_SKILLS_DIR`; move review checklist + fix protocol out of `buildClaudeMD` into `SKILL.md` | `agent-images/skills/{dev,review,fix}/`, `agent-runtime/` | S–M |
| P1-2 | **LSP diagnostics** per stack image; mandatory post-edit diagnostic loop wired into the dev/fix Skills; review Skill runs diagnostics on changed files | `agent-images/stacks/{go,node,python,go-node}/Dockerfile`, `agent-runtime/` | M |
| P1-3 | **Typed handoff briefing** generalized beyond findings (`handoff` metadata key with artifact refs, re-cast prior output) | `backend/internal/domain/model/run_context.go`, `backend/internal/domain/service/pipeline_executor.go` | M |
| P1-4 | **MCP server** (kanban + runs) over stdio with tool-level RBAC + audit log; scoped GitHub MCP | new `backend/internal/adapter/mcp/` (mark3labs/mcp-go), `agent-images/` | M–L |

### P2 — durable intelligence

| # | Change | Where | Effort |
|---|---|---|---|
| P2-1 | **Cross-sprint memory** (pgvector HNSW): `agent_memories` table, multi-signal retrieval, session-start briefing into `CLAUDE_MD_CONTENT`, async River consolidation job (DIY "Dreaming"); supersedes `priorFailureContext` | `backend/` (new migration + memory service + River job), `backend/internal/adapter/action/agent_run.go` | L |
| P2-2 | **Semantic oscillation detection** (cosine similarity on reviewer-output embeddings) once the embedding infra from P2-1 exists | `backend/internal/domain/service/pipeline_executor.go` | S |
| P2-3 | **Multi-critic review** (MAR): N persona-critics + judge synthesizing one reflection, for high-stakes stories — opt-in per pipeline given ~3× cost | config + `agent-images/skills/review/` | M |

---

### Sources

Review→fix loop: [Actor-Critic Adversarial Coding](https://understandingdata.com/posts/actor-critic-adversarial-coding/) · [Reflection (DeepLearning.ai)](https://www.deeplearning.ai/the-batch/agentic-design-patterns-part-2-reflection) · [MAR — arxiv.org/html/2512.20845](https://arxiv.org/html/2512.20845) · [Self-Improving Coding Agent — arxiv.org/html/2504.15228v1](https://arxiv.org/html/2504.15228v1) · [Stop the Loop](https://dev.to/alessandro_pignati/stop-the-loop-how-to-prevent-infinite-conversations-in-your-ai-agents-ekj) · [Agentic Loop with Claude Code](https://www.mindstudio.ai/blog/how-to-build-agentic-loop-claude-code) · [Automated Code Review with Multiple AI Agents](https://www.mindstudio.ai/blog/automated-code-review-multiple-ai-agents) · [Meta structured prompting](https://venturebeat.com/orchestration/metas-new-structured-prompting-technique-makes-llms-significantly-better-at) · [Google ADK Loop Agent](https://google.github.io/adk-docs/agents/workflow-agents/loop-agents/)

Handoff & escalation: [Building Effective Agents](https://www.anthropic.com/research/building-effective-agents) · [Multi-Agent Research System](https://www.anthropic.com/engineering/multi-agent-research-system) · [Measuring Agent Autonomy](https://www.anthropic.com/research/measuring-agent-autonomy) · [Handoffs — OpenAI Agents SDK](https://openai.github.io/openai-agents-python/handoffs/) · [Guardrails — OpenAI Agents SDK](https://openai.github.io/openai-agents-python/guardrails/) · [Supervisor vs Swarm](https://dev.to/focused_dot_io/multi-agent-orchestration-in-langgraph-supervisor-vs-swarm-tradeoffs-and-architecture-1b7e) · [AgentHallu — arXiv](https://arxiv.org/html/2601.06818v1) · [AI Agent Context Handoff — XTrace](https://xtrace.ai/blog/ai-agent-context-handoff) · [Guardrails 2026](https://medium.com/@dewasheesh.rana/agentic-ai-in-production-designing-autonomous-multi-agent-systems-with-guardrails-2026-guide-a5a1c8461772)

LSP & Skills: [Codex CLI LSP](https://codex.danielvaughan.com/2026/04/25/codex-cli-lsp-integration-language-server-semantic-code-intelligence/) · [Hermes Agent LSP](https://hermes-agent.nousresearch.com/docs/user-guide/features/lsp) · [agent-lsp](https://www.agent-lsp.com/) · [LSP for AI coding tools](https://amirteymoori.com/lsp-language-server-protocol-ai-coding-tools/) · [Agent Skills overview](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview) · [Skill best practices](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices) · [Equipping agents with Agent Skills](https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills)

MCP & memory: [MCP Authorization spec](https://modelcontextprotocol.io/specification/draft/basic/authorization) · [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) · [MCP stdio vs HTTP](https://www.truefoundry.com/blog/mcp-stdio-vs-streamable-http-enterprise) · [MCP RBAC](https://www.getmaxim.ai/articles/mcp-rbac-tool-level-permissions-for-production-ai-agents/) · [Tool-level permission scoping](https://dev.to/supertrained/tool-level-permission-scoping-in-mcp-why-server-authentication-isnt-enough-58ni) · [GitLab MCP Server](https://docs.gitlab.com/user/gitlab_duo/model_context_protocol/mcp_server/) · [State of AI Agent Memory 2026](https://mem0.ai/blog/state-of-ai-agent-memory-2026) · [pgvector DBA guide](https://www.dbi-services.com/blog/pgvector-a-guide-for-dba-part-2-indexes-update-march-2026/) · [Anthropic Dreaming](https://venturebeat.com/technology/anthropic-introduces-dreaming-a-system-that-lets-ai-agents-learn-from-their-own-mistakes)
