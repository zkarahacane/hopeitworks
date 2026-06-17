# Agent Engineering — Current State vs Best Practices

> Research synthesis for the hopeitworks execution layer (AI coding agents run per-role in Docker containers: dev, review).
> Maps the **current state** of our agents against 2025–2026 **best practices** for autonomous coding agents, then turns the gap into a prioritized roadmap.
> Scope of the codebase touched: `agent-images/`, `agent-runtime/`, `backend/internal/adapter/action/agent_run.go`, prompt templates, and platform DB.

---

## 1. Executive summary

The biggest gaps between our current agents and a capable autonomous coding agent:

- **No closed verification loop.** Agents run single-shot (`claude --print`), commit, and exit — they never run tests, check acceptance criteria against their own output, or self-correct. Anthropic calls a runnable check "the difference between a session you watch and one you walk away from." Today we walk away from a session that never checked itself. (`agent-runtime/internal/provider/claude.go`)
- **No system prompt / persona, and `CLAUDE.md` injection is dead code.** The agent gets the rendered task prompt and a full repo clone — nothing about its role, conventions, or definition of "done." `CLAUDE_MD_CONTENT` is wired in `agent-runtime` but the backend **never populates it** (`createContainer` in `agent_run.go`), so even static project context can't be injected.
- **Zero memory across runs.** Each container is ephemeral and clones fresh. A retry on a previously-failed story sees only static story fields — never the prior run's `error_message`/`log_tail`, never accumulated lessons, never what other agents in the same epic did. The product vision's "autonomie progressive (track record)" has no substrate today.
- **No tool intelligence layer (LSP / MCP).** Agents get raw text tools only (`Read`/`Grep`/`Bash`). No language server (safe rename, type-error feedback after edits, find-references), no MCP servers (no structured GitHub/Jira data, no access to our own kanban API, no DB introspection).
- **No Agent Skills.** Role behavior (how to implement a feature, how to review a PR) lives — if anywhere — in an ad-hoc prompt, not in versioned, progressively-disclosed `SKILL.md` packages. This is the natural home for our per-role specialization and it is entirely absent from the images.
- **Security posture is "permissive container," not "sandbox."** `--dangerously-skip-permissions` with no `--allowedTools`, no seccomp/AppArmor profile, no `--cap-drop`, no read-only rootfs, **unrestricted egress** (bridge network, not default-deny), and no hard resource ceiling baked in. Plain runc is no longer considered safe for agent code in 2026 (multiple 2025–2026 host-escape CVEs). For untrusted agent images this is the highest-risk gap.
- **Git identity unset → commit can fail or produce authorless commits.** The runtime calls `git commit` without ever running `git config user.*`, and no `~/.gitconfig` is baked into the base image.
- **Prompt template has latent bugs.** `diff_content` is declared on `TemplateContext` but never populated from `runCtx` — any review/merge template referencing `{{diff_content}}` renders empty, silently degrading the review role.

---

## 2. Current state

*Synthesised from the three map sources (Docker env, invocation/context, memory).*

### 2.1 Docker environment (`agent-images/`)

- **Base** (`agent-images/base/Dockerfile`): `debian:bookworm-slim`, non-root `agent` user, WORKDIR `/workspace`, Node 22, `@anthropic-ai/claude-code` + `opencode-ai` global, `git`/`curl`/`jq`/`openssh-client`, and the Go `agent-runtime` binary as ENTRYPOINT.
- **Stacks** (`agent-images/stacks/{go,node,python,go-node}/Dockerfile`): each extends base. `go` adds Go 1.23 + `sqlc`/`oapi-codegen`/`wire`/`golangci-lint`/`gh`; `node` adds `typescript`/`prettier`/`eslint`/`gh`; `python` adds only `python3`/`pip`/`gh`; `go-node` is Go-first and **omits** Node tooling.
- **Isolation**: `tecnativa/docker-socket-proxy` exposes only container/network APIs to the backend; agents get **no Docker socket**, **no volume mounts** (`Binds = nil`, ephemeral FS), attached to `agent-network` (bridge → **internet reachable**), isolated from the internal `hopeitworks` network. `Privileged: false`.
- **Resource limits**: `DefaultMemory`/`DefaultCPUs` from `AgentConfig` flow to `dockercontainer.Resources` at startup but **no ceiling** is hardcoded in any Dockerfile/compose. Container token TTL 2h, stop timeout 10s.
- **Auth routing** (`claude.go`): `sk-ant-oat*` → `CLAUDE_CODE_OAUTH_TOKEN`; else → `ANTHROPIC_API_KEY`.

### 2.2 Invocation & context (`agent-runtime`, `agent_run.go`)

- **Exact command** (`agent-runtime/internal/provider/claude.go:34-42`):
  `claude --print --output-format stream-json --model <MODEL> --dangerously-skip-permissions --verbose "<prompt>"`.
- **Single-shot** (`--print`): runs once, exits. No outer agentic loop, no self-eval. Retry = a brand-new container with `error_context` + `log_tail` injected into the prompt.
- **Full default tool set, no restriction**: Bash, Read, Write, Edit, Glob, Grep, LS, WebSearch, WebFetch, Task. **No `--allowedTools`, no `--mcp-config`, no `--append-system-prompt`.** Zero MCP servers, zero Agent Skills.
- **Context the agent receives**: Handlebars-rendered prompt (`story_key`, `story_title`, `story_objective`, `target_files`, `acceptance_criteria`, `branch_name`, `repo_url`; retry adds `error_context`, `log_tail`), the full repo clone on `BRANCH_NAME`, and optionally `.claude/CLAUDE.md` written from `CLAUDE_MD_CONTENT`. **`diff_content` exists on `TemplateContext` but is never populated.**
- **Post-run**: on exit 0 the runtime does `git commit` (hardcoded `feat(<story_key>): agent implementation`) + push. Git identity is never configured.
- **OpenCode path**: `opencode run --format json --model <MODEL> "<prompt>"`, fully buffered (`cmd.Output()`), no streaming.

### 2.3 Memory & cross-run state

- The "agent" is a per-step container running `agent-runtime`; **cleaned up (stop + remove) after every step** (`cleanupContainer`). No persistent container, FS, or process survives.
- **DB persists** stories, runs, run_steps (`log_tail`, `error_message`, `retry_*`), cost_records, append-only events, epic grouping. `runs.metadata` (JSONB) carries **intra-run** cross-step handoffs (`branch_name`, `pr_url`, …), re-read from DB on resume — but **mutations are not written back mid-run** (lost on crash).
- **The agent's entire "memory"** = the env vars at startup. It has **no access** to the `events` table, prior runs' logs, other stories' history, or any cross-run store. `CLAUDE_MD_CONTENT` injection is dead from the backend.

---

## 3. Recommendations by theme

Each item: **what** to change → **why** (cited best practice) → **effort** (S ≤ ½ day, M ≈ 1–3 days, L > 3 days).

### 3.1 Docker environment & sandbox

**R-D1 — Configure git identity in the base image.** Bake a `~/.gitconfig` (or have `agent-runtime` run `git config user.email/user.name` before committing) using a bot identity, e.g. `agent@hopeitworks.dev`.
*Why:* `git commit` currently runs with no identity set, risking commit failure or authorless commits. (map:docker-env gap 2, map:agent-invocation gap 8)
*Effort:* **S**

**R-D2 — Enforce a hard resource ceiling and deny-by-default egress.** Set `Memory`/`MemorySwap` (no swap), `NanoCPUs`, `PidsLimit` (100–256), and a wall-clock timeout in `createContainer`; never allow `0` (unlimited). Move agents off the open bridge to a **default-deny egress** posture with an explicit domain allowlist (LLM endpoint, package registry, our callback API, git host) via an egress proxy or iptables; block IMDS `169.254.169.254`.
*Why:* "Default-deny egress is non-negotiable — a perfectly isolated filesystem is useless if the agent can phone-home with secrets." Hard cgroup limits (memory hard-kill, PID limit to stop fork bombs, wall-clock auto-destroy) are baseline. ([Northflank](https://northflank.com/blog/how-to-sandbox-ai-agents), [Bunnyshell](https://www.bunnyshell.com/guides/coding-agent-sandbox/), [Anthropic sandbox-runtime](https://github.com/anthropic-experimental/sandbox-runtime)) (map:docker-env gaps 1, 10)
*Effort:* **M**

**R-D3 — Harden the container runtime.** Add `--cap-drop=ALL`, `--security-opt no-new-privileges`, a custom **seccomp** profile (block `io_uring_*`, `userfaultfd`, BPF, namespace-creation, fs-admin ioctls), an AppArmor/SELinux profile, **read-only root FS** with only `/workspace` + a size-capped `tmpfs /tmp` (noexec,nosuid,nodev) writable, and userns remapping.
*Why:* Plain runc is no longer safe for agent code — 2025–2026 host-escape CVEs (CVE-2025-31133/52565/52881, CVE-2026-34040). Layer defense-in-depth; a VM/seccomp boundary is the meaningful one. ([Blaxel](https://blaxel.ai/blog/container-escape), [clampdown](https://github.com/89luca89/clampdown), [Anthropic sandbox-runtime](https://github.com/anthropic-experimental/sandbox-runtime)) (map:docker-env gap 4)
*Effort:* **L**

**R-D4 — MicroVM isolation for untrusted/third-party agent images.** Run non-first-party images under **Firecracker** or **Kata** instead of runc.
*Why:* For multi-tenant / untrusted code, a kernel boundary (microVM) is the recommended primitive; container escape vs VM escape is a $/exploit difference of orders of magnitude. ([Northflank](https://northflank.com/blog/how-to-sandbox-ai-agents), [Shipyard Docker Sandboxes](https://shipyard.build/blog/docker-sandboxes-claude-code/)) (map:docker-env gap 4)
*Effort:* **L**

**R-D5 — Pin base images by digest and never install at runtime.** `FROM debian:bookworm-slim@sha256:…`; pin tool versions; scan with Trivy/Docker Scout on build, gate CRITICAL/HIGH. Pre-install Python tooling (`uv`/`poetry`, `pytest`, `ruff`, `black`) in `stacks/python`, and Node tooling in `stacks/go-node`.
*Why:* Runtime `apt`/`pip` bypasses image scanning, is non-reproducible, and is a supply-chain attack surface (Shai-Hulud npm campaigns). Pinning + build-time install is the reproducibility/security baseline. ([Bunnyshell](https://www.bunnyshell.com/guides/sandboxed-environments-ai-coding/), [MintMCP](https://www.mintmcp.com/blog/sandbox-claude-code)) (map:docker-env gaps 7, 8)
*Effort:* **M**

**R-D6 — Scoped, short-lived credentials only; never the home dir.** Keep mounting only `/workspace` (already true). Add explicit `denyRead` for `~/.ssh`, `~/.aws`, `~/.npmrc`, `.env*`. Keep credentials as already-scoped short-lived tokens (the 2h container token is good); ensure secrets are never echoed into logged env/prompt.
*Why:* The Cursor Nov-2025 leak happened via `cat ~/.npmrc` with a readable home; mount only the project dir, deny-read dotfiles, use short-lived scoped tokens. ([Luca Becker](https://luca-becker.me/blog/cursor-sandboxing-leaks-secrets/), [MintMCP](https://www.mintmcp.com/blog/sandbox-claude-code)) (map:docker-env gap 10)
*Effort:* **S**

### 3.2 Context engineering

**R-C1 — Add a system prompt / role persona via `--append-system-prompt`.** Inject a concise role contract (dev vs review), output expectations, and "definition of done" that does not depend on `CLAUDE.md` existing.
*Why:* Beyond an optional, caller-supplied `CLAUDE.md`, the agent has no instructions about its role or what "done" looks like. ([Claude Code best practices](https://code.claude.com/docs/en/best-practices)) (map:agent-invocation gap 1)
*Effort:* **S**

**R-C2 — Activate `CLAUDE_MD_CONTENT` and keep it lean.** Populate `CLAUDE_MD_CONTENT` in `createContainer` from a project-scoped context file; cap it (< 150 lines, hard limit 300): stack, key commands, directory map, conventions, critical gotchas — no code style (delegate to linter), no file-by-file descriptions, nothing LLM-generated.
*Why:* The injection path exists but is dead. Comprehensive/LLM-generated context files **reduce** task success (arxiv 2602.11988) by encouraging over-exploration; minimal human-written files outperform. ([HumanLayer](https://www.humanlayer.dev/blog/writing-a-good-claude-md), [arxiv 2602.11988](https://arxiv.org/abs/2602.11988), [Claude Code docs](https://code.claude.com/docs/en/best-practices)) (map:memory gap 6, map:agent-invocation gap 1)
*Effort:* **S**

**R-C3 — Restructure prompt templates into the canonical order with verifiable acceptance criteria.** `TASK → SCOPE (in/out, "never touch X") → RELEVANT FILES (@refs/codemaps) → ACCEPTANCE CRITERIA (numbered, each verifiable) → VERIFICATION COMMAND → COMMIT FORMAT → CONSTRAINTS`. Make at least one acceptance criterion a runnable command returning pass/fail, and require the agent to **show test output, not assert "tests pass."**
*Why:* The single most impactful addition is a check the agent can run; explicit scope boundaries and "show evidence" are non-negotiable rules. ([Claude Code best practices](https://code.claude.com/docs/en/best-practices), [Repo Prompt](https://repoprompt.com/blog/context-over-convenience)) (map:agent-invocation gap 1)
*Effort:* **M**

**R-C4 — Fix the `diff_content` bug for the review role.** Populate `TemplateContext.DiffContent` in `agent_run.go` from `runCtx`/story (e.g., `git diff` against base) before rendering.
*Why:* Review/merge templates referencing `{{diff_content}}` currently render empty, silently degrading review quality. A fresh-context reviewer should see exactly the diff + acceptance criteria. ([Anthropic best practices](https://code.claude.com/docs/en/best-practices)) (map:agent-invocation gap 2)
*Effort:* **S**

**R-C5 — Add a closed-loop execution pattern (explore → plan → implement → verify → commit).** Replace pure single-shot with a verification gate: after implementation the agent runs the test command, fixes on failure (max N retries), and only commits when green. Implement via a **Stop hook** that re-runs the suite and blocks turn-end until it passes, or an outer loop in `agent-runtime`.
*Why:* Single-shot has no self-evaluation; the closed-loop pattern + stop hook turns "looks done" into "is done." ([Claude Code best practices](https://code.claude.com/docs/en/best-practices)) (map:agent-invocation gap 3)
*Effort:* **L**

**R-C6 — Use subagents for exploration to protect the implementing context.** For exploratory stories, delegate codebase investigation to a read-only Explore subagent that returns a minimal file list + summary, not raw file contents.
*Why:* Context rot starts well before the token limit (Chroma, 18 models); subagent isolation avoids the "130k-token exploration tax" and keeps the implementer's context clean. ([Zylos](https://zylos.ai/research/2026-03-31-context-window-management-session-lifecycle-long-running-agents/), [LogRocket](https://blog.logrocket.com/context-engineering-for-ides-agents-md-agent-skills/), [Anthropic multi-agent](https://www.anthropic.com/engineering/multi-agent-research-system)) (map:agent-invocation gap 3)
*Effort:* **M**

### 3.3 Tools & MCP

**R-T1 — Wire LSP / `agent-lsp` per stack.** Install language servers matched to each stack image (`gopls`, `pyright`, `typescript-language-server`) and expose them via the built-in `LSP` tool or the `agent-lsp` MCP server (stdio).
*Why:* Raw `Grep`/`Read` cannot safely rename symbols, find references, or preview an edit's diagnostic blast radius. LSP fires after every edit and reports type errors with no build step — the non-obvious essential for a coding agent. ([Claude Code tools reference](https://code.claude.com/docs/en/tools-reference), [agent-lsp](https://github.com/blackwell-systems/agent-lsp)) (map:agent-invocation gap 4)
*Effort:* **M**

**R-T2 — Constrain tools instead of `--dangerously-skip-permissions`.** Replace blanket skip with an explicit `permissions.allow`/`deny` policy (`Bash(git *)`, `Bash(go *)`, `Edit(/workspace/**)`, deny `Bash(rm -rf *)`, deny `WebSearch` for the review role) so headless mode never blocks while still scoping capability per role.
*Why:* Skipping all permission checks with no allow/deny list is a security/isolation gap; per-role tool scoping (allowlist/denylist, `disallowedTools` wins) is the documented model. ([Claude Code tools reference](https://code.claude.com/docs/en/tools-reference), [sub-agents docs](https://code.claude.com/docs/en/sub-agents)) (map:agent-invocation gap 5)
*Effort:* **M**

**R-T3 — Add an MCP server for the hopeitworks API (kanban/story state).** A scoped MCP server (Streamable HTTP, sidecar) letting agents read kanban state, query project config, and update story status — instead of being blind to the platform that orchestrates them.
*Why:* MCP adds value precisely for authenticated structured access Bash can't hold; agents currently have no access to our own API, events table, or board. ([Claude Code MCP docs](https://code.claude.com/docs/en/mcp), [MCP transports](https://www.speakeasy.com/mcp/core-concepts/transports)) (map:agent-invocation gap 4, map:memory gap 1)
*Effort:* **L**

**R-T4 — Keep git/CI on Bash; add `github`/`gitlab` MCP only for structured data.** Don't wrap `git`/`gh` in MCP. Add a `github` MCP server (HTTP, `Authorization: Bearer ${GITHUB_TOKEN}`) only where structured PR/issue/pipeline JSON beats parsing shell output (notably the review role). Enable `ToolSearch`/deferred loading so tool schemas don't burn context at scale.
*Why:* Git/`gh` via Bash already covers most needs; MCP earns its keep only for schema-typed external data, and 150+ eager tool schemas cost real context. ([Claude Code tools reference](https://code.claude.com/docs/en/tools-reference), [MCP docs](https://code.claude.com/docs/en/mcp)) (map:agent-invocation gap 4)
*Effort:* **M**

**R-T5 — Secure any MCP we add.** stdio for same-container servers; Streamable HTTP + OAuth 2.1/PKCE for sidecar/remote (never SSE — deprecated); `.mcp.json` checked into VCS and reviewed like source; default-deny egress per server (pairs with R-D2).
*Why:* Tool definitions are executable code (tool-poisoning/prompt-injection risk); HTTP servers require OAuth 2.1+PKCE per the June-2025 spec; SSE is deprecated. ([Practical DevSecOps MCP guide](https://www.practical-devsecops.com/mcp-security-architecture-guide/), [MCP transport spec](https://modelcontextprotocol.info/specification/draft/basic/transports/)) (map:docker-env gap 4)
*Effort:* **M**

### 3.4 Agent Skills

**R-S1 — Ship per-role Agent Skills in the images.** Bake `.claude/skills/` into the role images: a dev skill (`implementing-feature`) and a review skill (`reviewing-pr`), each a `SKILL.md` (frontmatter `name` + third-person `description` with trigger keywords) plus on-demand reference files (`TESTING.md`, `SECURITY-CHECKLIST.md`) and deterministic `scripts/` wrappers.
*Why:* Skills turn a general agent into a domain specialist via **progressive disclosure** — only ~100 tokens of metadata are resident until triggered; script source never enters context. This is the right home for role specialization and is entirely absent today. ([Agent Skills overview](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview), [Authoring best practices](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices)) (map:agent-invocation: "No Agent Skills")
*Effort:* **M**

**R-S2 — One skill = one workflow; keep `SKILL.md` < 500 lines.** Split `implementing-feature` / `writing-tests` / `writing-docs`; put deterministic steps (lint, migrate, build) in bundled scripts; never chain reference → reference → reference. Write for the least-capable model in the pool.
*Why:* God-skills bloat context and misroute; the description is the routing key; deterministic steps belong in scripts (output enters context, code doesn't). ([Authoring best practices](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices)) (map:agent-invocation: "No Agent Skills")
*Effort:* **S**

**R-S3 — Version-control project skills; revisit our existing GSD skills.** Keep `.claude/skills/` in-repo so skills evolve via PR. Note: a large GSD skill library is already installed at the harness level — decide which (if any) belong **inside** agent containers vs. the orchestrator only.
*Why:* Project skills committed to the repo are reviewable and evolve with the code they support. ([Claude Code skills docs](https://code.claude.com/docs/en/skills)) 
*Effort:* **S**

### 3.5 Memory

**R-M1 — Inject prior-run failure context on retry and surface it to the agent.** On a retry run, fetch the failed run's `error_message` + `log_tail` (already in `run_steps`) and inject them as a distinct memory block. Today this is only wired for same-run incremental retries.
*Why:* An agent retrying a previously-failed story currently has no awareness of what was tried or what errored — the cheapest, highest-value memory we already store but don't surface. ([Anthropic context engineering](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents)) (map:memory gap 1)
*Effort:* **M**

**R-M2 — Layer 1: project-wide static memory file, mounted read-only.** A `project.memory.md` (stack, conventions, module map, key decisions, rejected patterns) committed to the repo, human-updated at sprint boundaries, injected into every container (via the now-activated `CLAUDE_MD_CONTENT`, R-C2).
*Why:* "Context window is RAM, not disk" — everything that survives a session boundary lives outside it; semantic/procedural memory belongs in a versioned file. ([Anthropic context engineering](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents), [Augment Code](https://www.augmentcode.com/guides/agent-memory-vs-context-engineering)) (map:memory gaps 2, 6)
*Effort:* **M**

**R-M3 — Layer 2: per-run/per-phase writable memory + consolidation.** A writable memory scoped by `(project_id, phase_id, agent_role)` where agents append decisions/lessons/blockers **immediately** (not batched). At phase end, a lightweight consolidation pass dedups, promotes high-confidence episodic → semantic, archives low-confidence. Use an entry schema with `type/content/code_refs/confidence/decay_rate`. Reuse Postgres (add `pgvector`); avoid a graph DB at this scale.
*Why:* ~22% of tool-result tokens go to re-establishing already-learned context; selective memory + lifecycle (decay, consolidation, contradiction-archival) is what makes cross-run learning durable and auditable — critical for a HITL platform. ([Medium blueprint](https://medium.com/@sourabh.node/persistent-memory-for-ai-coding-agents-an-engineering-blueprint-for-cross-session-continuity-999136960877), [mem0.ai](https://mem0.ai/blog/state-of-ai-agent-memory-2026), [Anthropic memory tool](https://platform.claude.com/docs/en/agents-and-tools/tool-use/memory-tool)) (map:memory gaps 2, 4)
*Effort:* **L**

**R-M4 — Layer 3: cross-sprint retrieval injected at session start.** At phase/sprint start, retrieve Layer-1 context + top-K Layer-2 entries (ranked recency × confidence × role-relevance), synthesize a 500–1000 token briefing, inject as a structured block ahead of the task. Target < 2s retrieve→inject.
*Why:* Just-in-time retrieval beats preload-everything; a coherent retrieved briefing re-anchors context across the session boundary and is the substrate for the product's "autonomie progressive (track record)." ([Anthropic context engineering](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents), [Medium blueprint](https://medium.com/@sourabh.node/persistent-memory-for-ai-coding-agents-an-engineering-blueprint-for-cross-session-continuity-999136960877)) (map:memory gaps 3, 4)
*Effort:* **L**

**R-M5 — Persist `runs.metadata` mutations mid-run.** Write back action-produced values (`pr_url`, etc.) to `runs.metadata` as they happen, not only at launch.
*Why:* In-flight metadata is in-memory only during execution; a mid-run crash loses it, and resume restores only launch-time values. ([Anthropic memory tool](https://platform.claude.com/docs/en/agents-and-tools/tool-use/memory-tool)) (map:memory gap 5)
*Effort:* **S**

---

## 4. Prioritized roadmap

Effort in parentheses. Targets: `AI` = `agent-images/`, `AR` = `agent-runtime/`, `PT` = prompt templates, `BE` = backend/platform (`agent_run.go`, DB).

### P0 — correctness, safety, quick wins (do first)

| ID | Change | Target | Effort |
|---|---|---|---|
| R-D1 | Bake git identity (`~/.gitconfig` or `git config` in runtime before commit) | AI, AR | S |
| R-C4 | Populate `TemplateContext.DiffContent` from `runCtx`/`git diff` (fixes empty review diff) | BE, PT | S |
| R-C2 | Activate `CLAUDE_MD_CONTENT` in `createContainer`; ship a lean (< 150-line) project context file | BE | S |
| R-C1 | Inject role persona via `--append-system-prompt` (dev vs review) | AR, PT | S |
| R-D6 | `denyRead` dotfiles (`~/.ssh`, `~/.aws`, `~/.npmrc`, `.env*`); confirm only `/workspace` mounted | AI, BE | S |
| R-M1 | On retry, surface prior run `error_message` + `log_tail` as a memory block | BE, PT | M |
| R-D2 | Hard resource ceiling (mem/CPU/PIDs/timeout, never `0`) + default-deny egress allowlist; block IMDS | AI, BE | M |
| R-M5 | Persist `runs.metadata` mutations mid-run | BE | S |

### P1 — capability uplift (autonomy + tools)

| ID | Change | Target | Effort |
|---|---|---|---|
| R-C3 | Restructure prompt templates to canonical order; verifiable acceptance criteria; "show test output" | PT | M |
| R-C5 | Closed-loop execute→verify→commit (Stop hook re-runs tests / outer loop in runtime) | AR, PT | L |
| R-T1 | Wire LSP / `agent-lsp` per stack (`gopls`, `pyright`, `tsserver`) | AI | M |
| R-T2 | Replace blanket `--dangerously-skip-permissions` with per-role `permissions.allow`/`deny` | AR, BE | M |
| R-S1 | Ship per-role Agent Skills (`implementing-feature`, `reviewing-pr`) in images | AI | M |
| R-S2 | One-skill-per-workflow, `SKILL.md` < 500 lines, deterministic steps in scripts | AI | S |
| R-D5 | Pin base images by digest; build-time-only tooling; fix `python`/`go-node` tool gaps; Trivy gate | AI | M |
| R-C6 | Explore subagent (read-only) for investigation-heavy stories | AR, PT | M |
| R-M2 | Layer 1 project-wide static memory file (`project.memory.md`), read-only into every container | BE, AI | M |

### P2 — platform memory, MCP, hardened isolation

| ID | Change | Target | Effort |
|---|---|---|---|
| R-M3 | Layer 2 per-run writable memory `(project, phase, role)` + phase-end consolidation (pgvector) | BE | L |
| R-M4 | Layer 3 cross-sprint retrieval → synthesized briefing injected at session start | BE | L |
| R-T3 | hopeitworks API MCP server (read kanban, update story status) | BE, AR | L |
| R-T4 | `github`/`gitlab` MCP for structured PR/issue data (review role); enable `ToolSearch` | AI, BE | M |
| R-T5 | Secure MCP: stdio local / HTTP+OAuth2.1 remote, no SSE, `.mcp.json` in VCS, per-server egress | AI, BE | M |
| R-D3 | Runtime hardening: `--cap-drop=ALL`, `no-new-privileges`, seccomp, AppArmor, read-only rootfs, tmpfs, userns | AI, BE | L |
| R-D4 | Firecracker/Kata microVM isolation for untrusted/third-party agent images | BE | L |
| R-S3 | Version-control project skills; decide GSD-skill placement (container vs orchestrator) | AI | S |

**Sequencing note.** P0 is largely independent and ships safety+correctness fixes within days. R-C2 (activate `CLAUDE_MD_CONTENT`) is a prerequisite for R-M2. R-C5 (closed loop) and R-T1 (LSP) compound — verification is far more reliable with type-error feedback. The memory ladder must go R-M1 → R-M2 → R-M3 → R-M4 in order. R-T5 (MCP security) must land **with** any MCP work (R-T3/R-T4), not after.
