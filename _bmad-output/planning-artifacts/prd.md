---
stepsCompleted: ['step-01-init', 'step-02-discovery', 'step-03-success', 'step-04-journeys', 'step-05-domain-skipped', 'step-06-innovation-skipped', 'step-07-project-type', 'step-08-scoping', 'step-09-functional', 'step-10-nonfunctional', 'step-11-polish', 'step-12-complete']
inputDocuments:
  - '_bmad-output/brainstorming/brainstorming-session-2026-02-15.md'
  - '_bmad-output/brainstorming/handoff-2026-02-15.md'
workflowType: 'prd'
documentCounts:
  briefs: 0
  research: 0
  brainstorming: 2
  projectDocs: 0
classification:
  projectType: 'saas_b2b_developer_platform'
  domain: 'developer_tools_devops'
  complexity: 'medium'
  projectContext: 'greenfield'
---

# Product Requirements Document — Hopeitworks v2

**Author:** Zakari
**Date:** 2026-02-15

## Executive Summary

Hopeitworks is an AI agent orchestration platform that automates software development pipelines. It launches AI coding agents (Claude Code) inside isolated Docker containers, orchestrates them through configurable pipelines (implement → CI → review → merge), and parallelizes work across stories using DAG-based scheduling.

**Problem:** Orchestrating multiple AI coding agents is manual, fragile, and expensive. V1 proved the concept but suffered from broken CI polling (agents instead of wait), non-incremental retries ($12+ wasted), no real-time visibility, and CLI-only UX.

**Solution:** Rebuild from scratch with v1 learnings. Event-driven architecture (Postgres LISTEN/NOTIFY), real-time SSE streaming, configurable pipelines, incremental retry, and a web UI + CLI for monitoring and control.

**Differentiator:** Hopeitworks builds itself. The North Star is dogfooding — the platform develops its own code with parallel agent execution, automatic error correction, and human-in-the-loop gates.

**Target users:** Power developers orchestrating AI agents, colleagues onboarding their repos, functional users tracking project progress.

**Tech stack:** Go backend (chi + sqlx + oapi-codegen), Vue 3 + TypeScript frontend, Postgres, Docker containers, JWT auth.

## Success Criteria

### User Success
- **Dogfooding operational:** Hopeitworks v2 develops its own code (stories → implement → CI → merge) with functional incremental retry
- **Visible parallelism:** Launch an epic and see agents working in parallel with real-time status (web UI + CLI)
- **Zero CI waste:** CI step polls results instead of launching an agent ($0 vs $0.5-1/run in v1)
- **Intelligent retry:** On CI failure, agent receives error context and fixes existing code instead of starting from scratch
- **2-minute setup:** `docker-compose up` → running instance, frictionless for self and colleagues
- **Git provider agnostic:** Abstract `GitProvider` interface — switch provider via config

### Business Success
- **Daily usage:** Used on personal and professional projects
- **Peer adoption:** At least 1 colleague tries it and completes a full run
- **Cost per story < $3:** Implement + CI + merge without wasted retries (vs $5+ in v1)

### Technical Success
- **Native CI wait:** Polls CI results (GitHub Actions, GitLab CI) instead of launching an agent
- **Event-driven:** Postgres LISTEN/NOTIFY for real-time state (replaces SQLite polling)
- **SSE streaming:** Agent logs visible in real-time in the browser
- **Incremental retry:** Agent receives diff + CI error, works on existing branch
- **CLAUDE.md injected:** Project conventions respected by agents
- **Abstract interfaces:** GitProvider, AgentRuntime, Authorizer — each with single MVP impl, extensible later

### Measurable Outcomes
- Full epic (8+ stories) executed in auto mode with ≥80% success rate without human intervention
- Average story time (implement → merge) < 10 minutes
- Zero zombie containers after auto cleanup

## User Journeys

### Journey 1 — Zakari, Power User: "The Friday Night Epic"

**Opening:** Friday 5pm, Zakari has finalized 12 stories for Epic 3. Each story has dependencies, target files, and acceptance criteria. He wants to launch everything and follow along.

**Rising Action:** He opens the web UI, selects Epic 3, clicks "Run Epic". The orchestrator builds the DAG, displays parallel groups: "Group 1: 4 stories (zero dependencies), Group 2: 5 stories (depend on G1), Group 3: 3 stories (depend on G2)". He launches in auto mode. 4 containers start. The dashboard shows 4 lines moving: implement → CI wait → review. His phone gets a Discord notification: "3-01 merged, 3-02 merged, 3-03 CI fail (lint error L42)".

**Climax:** 3-03 failed. The retry agent starts automatically — receives the CI log, the current diff, fixes line 42, pushes. CI passes. Meanwhile Group 2 stories have already started since their G1 dependencies are resolved. Zakari watches the pipeline advance without touching anything.

**Resolution:** 2 hours later, 11/12 stories merged. 1 story blocked on an architecture choice — HITL gate, notification "3-09 needs approval". Zakari opens the diff in the web UI, approves. Merge. Epic 3 done. Total cost: $28.

**Capabilities revealed:** Epic runner, DAG scheduling, SSE streaming, notifications (Discord/webhook), incremental retry, HITL approve via web UI, cost tracking.

### Journey 2 — Karim, Dev Colleague: "First Run on His Repo"

**Opening:** Karim, a Java backend dev, sees Zakari launching epics in auto mode. He wants to try on his Spring Boot API. He doesn't know Claude Code.

**Rising Action:** Zakari gives him the instance URL. Karim logs in (JWT), creates a new project, connects his GitHub repo. He writes his first story in markdown: refactor the `/users` endpoint. He clicks "Run". A container starts, he sees logs in real-time — Claude analyzes the code, proposes a refactor, pushes a PR.

**Climax:** Karim's GitLab CI runs (he's on GitLab, not GitHub). Hopeitworks polls results via the GitLab provider. CI passes. The review agent validates. Karim sees the MR appear in GitLab.

**Resolution:** Karim merges. He saved 2 hours of mechanical refactoring. He asks Zakari how to write better stories for parallelization.

**Capabilities revealed:** Simple onboarding, multi-project, abstract GitProvider (GitLab), story editor, log streaming, MR creation via provider.

### Journey 3 — Sophie, Functional User: "Tracking Project Progress"

**Opening:** Sophie is a PM/PO, Zakari's friend, somewhat technical (can configure a docker-compose). She wants to follow the progress of a project she spec'd.

**Rising Action:** She logs into the web UI. Dashboard: 3 epics, 24 stories. At a glance: Epic 1 done, Epic 2 in progress (6/8 stories), Epic 3 backlog. She clicks Epic 2, sees which stories are running, which are waiting. One run is "waiting approval".

**Climax:** She reads the diff of the pending story. It's a wording change in the UI. She understands the change, clicks approve. The agent merges.

**Resolution:** Sophie checks the metrics: total project cost, average time per story, success rate. She writes the next stories for Epic 3 in markdown. No coding needed, just writing specs.

**Capabilities revealed:** Progress dashboard, HITL approve (non-tech friendly), markdown story editor, project metrics.

### Journey 4 — Zakari Admin: "Setting Up a New Instance"

**Opening:** Zakari deploys Hopeitworks for a side project with a friend. He clones the repo, adjusts `docker-compose.yml`.

**Rising Action:** He configures: Git provider (GitHub), default model (Sonnet), pipeline steps, Discord notifications, budget limit per story ($5). He runs `docker-compose up`. API + UI + Postgres start. He logs in, creates the first project, connects the repo.

**Climax:** He invites Sophie and Karim (JWT tokens). Each sees their projects. Zakari sees everything + global metrics (costs, runs, active containers).

**Resolution:** The instance runs. Maintenance near zero. Logs in stdout (slog JSON), pluggable into LGTM stack later.

**Capabilities revealed:** docker-compose setup, YAML config (provider, models, pipeline, notifications, budgets), multi-user JWT, admin view, slog observability.

### Journey Requirements Summary

| Capability | J1 Power User | J2 Dev Colleague | J3 Functional | J4 Admin |
|------------|:------------:|:----------------:|:-------------:|:--------:|
| Epic runner + DAG | ✅ | | ✅ (view) | |
| SSE log streaming | ✅ | ✅ | | |
| Notifications | ✅ | | | ✅ (config) |
| Incremental retry | ✅ | | | |
| HITL approve/reject | ✅ | | ✅ | |
| Abstract GitProvider | | ✅ (GitLab) | | ✅ (config) |
| Story tracking | ✅ | ✅ | ✅ | |
| Multi-project | | ✅ | | ✅ |
| Progress dashboard | ✅ | | ✅ | ✅ |
| Cost tracking | ✅ | | ✅ (view) | ✅ |
| Pipeline config | | | | ✅ |
| Prompt editor | ✅ | | | ✅ |

## Technical Architecture

### Abstraction-First Design

Three core interfaces, each with single MVP implementation:

| Interface | MVP Implementation | Future |
|-----------|-------------------|--------|
| `GitProvider` | GitHub (gh CLI/API) | GitLab, Gitea, Bitbucket, Azure DevOps |
| `AgentRuntime` | Claude Code in Docker | opencode, other AI coding tools |
| `CIProvider` | Poll via GitProvider API | Direct CI integrations if needed |

Additional extensible interfaces: `Authorizer` (permissions), `Notifier` (notifications).

### Multi-Tenancy Readiness
- All database tables include `project_id` foreign key
- No global state or hard-coded single-user assumptions
- Architecture allows adding tenant isolation without schema migration

### Permission Model
- MVP: 2 roles — `admin` (full access) and `user` (own projects)
- Extensible via `Authorizer` interface for RBAC, per-project roles, or team-based permissions

### Integration Points
- **Git forges:** Clone, branch, PR/MR, merge, CI status (via `GitProvider`)
- **AI coding agents:** Prompt injection, event streaming, cost tracking (via `AgentRuntime`)
- **Notifications:** Discord, webhooks (via `Notifier`)
- **Observability:** slog JSON stdout, LGTM-compatible

### Implementation Principles
- All abstractions = Go interfaces with single MVP implementation
- Interface + 1 impl; second impl adds value later
- Config-driven provider selection: `git_provider: github`, `agent_runtime: claude-code`

## Product Scope & Phased Development

### MVP Strategy

**Approach:** Problem-solving MVP — rebuild from scratch with v1 learnings as design constraints. Abstractions defined from day 1, only MVP providers implemented.

**Resource:** Solo developer (Zakari) + Claude agents. Target: < 30 stories for MVP.

**Core Journeys Supported:** J1 (Power User), J4 (Admin), partial J2/J3 (login, follow runs, approve)

### MVP Feature Set (Phase 1)

| Category | Feature |
|----------|---------|
| **Backend** | Go API + Postgres (stories, runs, steps, events) |
| **Pipeline** | Configurable state machine — add/remove/reorder steps per project, each step has agent, model, prompt template, auto/hitl, retry policy |
| **Scheduler** | DAG builder + parallel executor (dependency + file conflict detection) |
| **Retry** | Incremental retry (receives CI error + diff, fixes on existing branch). Fallback to full retry after 2 failed incremental attempts |
| **GitProvider** | Interface + GitHub implementation |
| **AgentRuntime** | Interface + Claude Code / Docker implementation |
| **Containers** | Create, env injection (CLAUDE.md + secrets), log streaming, cleanup |
| **SSE** | Real-time log streaming to web UI |
| **Web UI** | Story list + status tracker, run controls, log viewer, approve/reject, prompt template editor, pipeline editor |
| **Auth** | JWT simple (admin/user) |
| **Cost** | Token/cost tracking, viewing, and limit setting per run and story |
| **Notifications** | Discord + webhook |
| **Prompts** | Handlebars templates (implement, retry, review, merge), editable via web UI |
| **Pipeline Config** | YAML per project, editable via web UI — define steps, agents, models, auto/hitl gates |
| **Test Project** | Reference todo app with build, seed SQL, CI pipeline, E2E tests — stable baseline for pipeline validation |

### Phase 2 — Growth
- GitLab provider
- Full story editor web (create/edit stories from UI)
- DAG visualization (Argo-style)
- Metrics/FinOps dashboard
- Progressive autonomy (auto-approve based on track record)
- opencode AgentRuntime support
- CLI interface (run, status, logs, approve/reject)
- Budget enforcement (halt execution on limit exceeded)

### Phase 3 — Expansion
- Gitea, Bitbucket, Azure DevOps providers
- MCP server (Claude pilots Hopeitworks natively)
- K8s Jobs runtime
- Custom workflow marketplace
- Plugin analyzers (SonarQube, etc.)

### Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| **Incremental retry fails** | Solid prompt engineering + fallback to full retry after 2 failed incremental attempts |
| **Solo resource constraint** | MVP < 30 stories. Dogfooding bootstrapped with bash scripts |
| **Chicken-egg (pipeline needed for dogfooding)** | First stories hand-coded or via bash scripts, then Hopeitworks takes over |

## Functional Requirements

### Project Management

- **FR1:** Admin can create a new project and connect it to a Git repository
- **FR2:** Admin can configure a project's Git provider, agent runtime, default model, and budget limits
- **FR3:** Users can view the list of projects they have access to
- **FR4:** Users can view stories grouped by epic within a project
- **FR5:** Users can view the status of each story (backlog, running, done, failed)

### Story Management

- **FR6:** Users can view story details (objectives, target files, dependencies, acceptance criteria)
- **FR7:** Users can import stories from markdown files in the repository
  *Note: Users can trigger a sync that reads markdown files with frontmatter from a designated folder in the repository (e.g., `.hopeitworks/stories/*.md`), parses them, and upserts stories in the database.*
- **FR8:** System can parse story frontmatter (key, epic, depends_on, scope, status)

### Pipeline Execution

- **FR9:** Users can launch a single story run
- **FR10:** Users can launch all stories of an epic as a batch run
- **FR11:** System can build a DAG from story dependencies and file conflicts
- **FR12:** System can execute stories in parallel groups based on the DAG
- **FR13:** System can execute pipeline steps sequentially for each story (configurable step chain)
- **FR14:** System can poll CI results via the GitProvider instead of launching an agent
- **FR15:** System can perform incremental retry when CI fails (agent receives error context + existing diff)
- **FR16:** System can fallback to full retry after 2 failed incremental attempts
- **FR17:** Users can pause a running story or epic
- **FR18:** Users can resume a paused story or epic

### Pipeline Configuration

- **FR19:** Admin can define pipeline steps per project (add, remove, reorder)
- **FR20:** Admin can configure each step's agent, model, prompt template, auto/hitl gate, and retry policy
- **FR21:** Users can view the current pipeline configuration for a project
- **FR22:** System stores pipeline configuration as YAML per project

### Human-in-the-Loop (HITL)

- **FR23:** System can pause execution at configured HITL gates and notify the user
- **FR24:** Users can approve a pending HITL request via web UI
- **FR25:** Users can reject a pending HITL request with a reason via web UI
- **FR26:** Users can approve or reject HITL requests via CLI *(Phase 2)*

### Agent & Container Management

- **FR27:** System can create an isolated Docker container per agent run
- **FR28:** System can inject CLAUDE.md, secrets, and configuration as environment variables into containers
- **FR29:** System can stream agent output (NDJSON) from containers in real-time
- **FR30:** System can clean up containers after run completion or failure
- **FR31:** System can enforce a hard timeout per container (configurable, default 30min)
- **FR32:** System can apply circuit breaker (stop after N consecutive failures)

### Prompt Management

- **FR33:** Admin can view all prompt templates (implement, retry, review, merge, custom)
- **FR34:** Admin can edit prompt templates via the web UI
- **FR35:** System can render prompt templates with story context variables (Handlebars)
- **FR36:** Admin can create custom prompt templates for custom pipeline steps

### Real-Time Monitoring

- **FR37:** Users can view live agent logs via SSE streaming in the web UI
- **FR38:** Users can view live agent logs via CLI *(Phase 2)*
- **FR39:** Users can view run progress (current step, substep) in real-time
- **FR40:** System can send notifications (Discord, webhook) on run events (success, failure, HITL pending)

### Cost & Observability

- **FR41:** System can track token usage and cost per step, run, and story
- **FR42:** Users can view cost breakdown per run and per story
- **FR43:** Admin can set budget limits per story and per project
- **FR44:** System can halt execution when budget limit is exceeded *(Phase 2)*

### Authentication & Authorization

- **FR45:** Users can authenticate via JWT
- **FR46:** Admin can create and manage user accounts
- **FR47:** Admin has full access to all projects and configurations
- **FR48:** Users can only access projects they are assigned to

### Git Operations

- **FR49:** System can clone a repository, create branches, and push commits via the GitProvider interface
- **FR50:** System can create pull/merge requests via the GitProvider interface
- **FR51:** System can poll CI status via the GitProvider interface
- **FR52:** System can merge pull/merge requests via the GitProvider interface

### Test Environment

- **FR53:** System includes a reference test project (todo app) with build, seed SQL, CI pipeline, and E2E tests for pipeline validation

## Non-Functional Requirements

### Performance

- **NFR1:** SSE event latency from container to browser < 1 second
- **NFR2:** REST API response time < 500ms for CRUD operations
- **NFR3:** DAG computation for 50+ stories < 2 seconds
- **NFR4:** Container startup time budget < 30 seconds

### Security

- **NFR5:** Agent containers fully isolated via Docker — no host filesystem access
- **NFR6:** Docker socket access restricted via docker-socket-proxy (allowlisted operations only)
- **NFR7:** API keys and Git tokens injected as environment variables, never persisted in database or logs
- **NFR8:** slog output scrubs sensitive values (tokens, keys) from structured logs
- **NFR9:** JWT tokens signed with configurable secret and expiration

### Reliability

- **NFR10:** API crash must not terminate running agent containers
- **NFR11:** All run/step state persisted in Postgres — resumable after API restart
- **NFR12:** Orphan container cleanup on API startup (garbage collector)
- **NFR13:** Circuit breaker halts execution after N consecutive failures (configurable, default 3)
- **NFR14:** Hard timeout per container enforced (configurable, default 30min)
- **NFR15:** Reference test project maintains permanently green CI baseline for pipeline validation
