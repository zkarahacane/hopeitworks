<div align="center">

# hopeitworks

### AI agent orchestration platform for software development

**Plan your work anywhere. Let role-based AI agents implement it in parallel — inside isolated Docker containers, with human-in-the-loop gates, an in-app kanban, and a live execution tree.**

[Architecture](#-architecture) · [Features](#-features) · [Tech stack](#-tech-stack) · [Engineering highlights](#-engineering-highlights) · [Run it locally](#-run-it-locally)

</div>

---

## What it is

**hopeitworks** turns a backlog into merged pull requests. You launch an *epic*; the platform schedules its stories on a dependency graph and runs each one through a fully configurable pipeline — an isolated AI agent writes the code, opens a branch, a reviewer agent checks it, a human approves at a gate, a PR is opened and CI is polled. You watch the whole thing build in real time. Agents run on a **pluggable execution substrate** — Docker by default for dev/CI, a hardened microVM (microsandbox) as the production isolation policy.

> **Positioning** — hopeitworks is an **execution layer that is agnostic to both your planning tool and your process.** Plan in markdown, Jira, GitHub Issues, BMAD, GSD — whatever. The team composes its *own* agents (image / model / provider / prompt) and its *own* pipeline (no workflow is imposed). The platform is opinionated only about the **runtime**: containers, parallelism, isolation, and human-in-the-loop.

> **North Star** — *"hopeitworks builds itself"*: the platform is used to develop its own codebase.

<div align="center">
<img src="docs/screenshots/09-dag-view.png" alt="Live epic DAG — stories scheduled on a dependency graph with per-node status" width="850">
<br><em>Epic DAG — stories scheduled on their dependency graph, status streamed live as agents run.</em>
</div>

---

## The core loop

```
Project (Git repo)
   └─ Epic ──────────── launched as a whole, stories run in parallel
        └─ Story ─────── one user story with testable acceptance criteria
             └─ Run ──── one full pipeline execution for that story
                  └─ Step ── git_branch → agent_run → review → HITL gate → git_pr → ci_poll
                       └─ Agent ── execs a harness on a pluggable substrate (Docker / microVM / …)
```

1. **Plan** anywhere and import stories (markdown, in-app editor, …).
2. **Launch** a story or an entire epic. The scheduler builds a DAG and runs independent stories concurrently.
3. **Execute** — each story flows through its project's pipeline; agents run on the configured execution substrate (Docker container, microVM, …) and talk back to the API over HTTP callbacks.
4. **Supervise** — follow agent logs over SSE, approve changes at human gates, retry failed steps.
5. **Ship** — branches and PRs land on your Git provider, ready to merge.

---

## ✨ Features

| Area | Capabilities |
|------|--------------|
| **Orchestration & execution** | Launch a story or a whole epic; DAG-based parallel scheduling; per-project configurable pipeline (step groups, agents, models, prompts); pause / resume / cancel a run; retry a failed step from the UI |
| **Agents & runtime** | Configurable `Agent` entity (image · model · provider · prompt); multi-provider (Claude, opencode); Go agent runtime with HTTP callbacks; **pluggable execution substrate** behind one port (Docker default, microsandbox microVM for hardened isolation); stack catalogue of digest-pinned agent images (go-node, node, go, python); **Environment** feature for sidecar services (db, redis, …) with injected connection strings |
| **Resilience** | Incremental retry (the agent receives the diff + CI error and fixes the existing code); fallback to a full retry after repeated failures; project-scoped circuit breaker; native CI polling via the Git provider (no agent wasted on waiting) |
| **Human-in-the-loop** | Automatic pause at configured checkpoints; diff viewer; approve / reject with a reason |
| **Real-time monitoring** | SSE streaming of agent logs in the browser; step-by-step progress; live execution tree; Discord & webhook notifications |
| **Cost tracking** | Tokens & cost per step / run / story / agent; cost dashboard with aggregations; per-project budget limits |
| **Stories & epics** | Kanban board with status filtering; markdown story import; in-app story editor; epics with DAG computation and visualization |
| **Auth & projects** | JWT auth, admin/user roles; per-user API keys encrypted with AES-256; projects connected to a Git repo with per-project members & permissions |

**Built-in pipeline actions:** `agent_run` · `ci_poll` · `git_branch` · `git_pr` · `hitl_gate` · `notification` · `incremental_retry`.

---

## 🏗 Architecture

Hexagonal (ports & adapters) backend, an OpenAPI contract as the single source of truth, and a typed Vue 3 SPA.

```mermaid
flowchart LR
    subgraph FE["Frontend — Vue 3 SPA (nginx)"]
        UI["Kanban · DAG · Run timeline<br/>Pipeline & Agent editors · Costs · Approvals"]
    end

    subgraph API["Backend — Go (hexagonal)"]
        H["chi handlers<br/>(generated from OpenAPI)"]
        S["Domain services<br/>Scheduler · Pipeline · CircuitBreaker"]
        Q["River job queue<br/>(durable, Postgres)"]
        EV["LISTEN/NOTIFY → SSE"]
    end

    subgraph RT["Agent runtime — port.AgentRuntime"]
        C1["Substrate adapter<br/>(Docker / microVM / exec)<br/>agent harness + LLM"]
    end

    PG[("PostgreSQL")]
    GIT["Git provider<br/>(GitHub API · go-github)"]

    UI -->|"openapi-fetch (typed)"| H
    UI <-->|"SSE live updates"| EV
    H --> S --> Q --> C1
    C1 -->|"HTTP callbacks"| H
    S -->|"branch · PR · CI (GitHub API)"| GIT
    C1 -->|"git commit / push"| GIT
    S --- PG
    EV --- PG
```

**Domain model:** `Project → Epic → Story → Run → Step → Agent`. A *Project* is a Git repo with its config; an *Epic* is a set of stories with a dependency DAG; a *Run* is one pipeline execution; a *Step* is a pipeline stage; an *Agent* executes a step in an isolated container.

Detailed design docs live under [`_bmad-output/docs/`](_bmad-output/docs/) (architecture + Mermaid diagrams) and [`docs/product.md`](docs/product.md) (product vision).

---

## 🧰 Tech stack

| Layer | Technologies |
|-------|--------------|
| **Backend** | Go · [chi](https://github.com/go-chi/chi) router · [pgx](https://github.com/jackc/pgx) + [sqlc](https://sqlc.dev) (type-safe SQL) · [google/wire](https://github.com/google/wire) (compile-time DI) · [River](https://riverqueue.com) (Postgres-backed durable job queue) · golang-migrate · pgxlisten (LISTEN/NOTIFY) · JWT · slog |
| **Frontend** | Vue 3 (Composition API) · [PrimeVue 4](https://primevue.org) (unstyled + design tokens) · Tailwind CSS v4 · Pinia · Vue Router · [openapi-fetch](https://openapi-ts.dev) (typed client) · [Vue Flow](https://vueflow.dev) (DAG) · Monaco (editors) · Vitest · Playwright |
| **Contract** | A single [`api/openapi.yaml`](api/openapi.yaml) is the source of truth — the Go server interfaces (oapi-codegen) **and** the TypeScript client are generated from it |
| **Agent substrate** | Pluggable behind `port.AgentRuntime` — **Docker** (default, via a filtered docker-socket-proxy) or **microsandbox** (libkrun microVM, KVM); `exec` and K8s/OpenShift slot behind the same port |
| **Git** | Backend git actions (branch · PR · CI) talk to the **GitHub API** directly via [go-github](https://github.com/google/go-github) (token auth) — no `gh` CLI dependency on the backend path; the agent commits/pushes with `git` |
| **Runtime / infra** | PostgreSQL · nginx (serves the SPA + reverse-proxies the API) · sidecar services via the **Environment** feature (isolated run-network + injected conn-strings) · MailHog (dev mail) · SSE |

---

## 💡 Engineering highlights

- **OpenAPI as the single source of truth** — both the Go handler interfaces and the typed TS client are code-generated from one spec, so the API contract cannot silently drift between front and back.
- **Hexagonal architecture with compile-time DI** — strict `domain → port ← adapter` boundaries, compile-time interface checks, and dependency wiring resolved at build time with `wire`.
- **DAG topological scheduling** — stories are ordered with a Kahn topological sort; implicit *file-conflict edges* are added so two agents never edit the same file concurrently.
- **Durable, recoverable execution** — a Postgres-backed job queue, a guarded run/step state machine, and HITL **suspend/resume** that replays a paused run to exactly the right step.
- **Live execution tree** — Postgres `LISTEN/NOTIFY` fan-out to **SSE** with `Last-Event-ID` replay on reconnect, so the UI rebuilds live without polling.
- **Pluggable execution substrate** — an agent is just *"exec a harness"*; *where* it execs lives behind a single `port.AgentRuntime`. Docker (default, scoped socket-proxy — no raw daemon access) and a hardened **microVM** (microsandbox / libkrun, KVM-class kernel isolation for untrusted agent code) are equal adapters selected by `SUBSTRATE`; `exec` and a future K8s/OpenShift adapter slot in without touching the domain.
- **Isolation & security by design** — per-user API keys encrypted with **AES-256**, JWT auth with role-based access, and substrate-chosen isolation (container or microVM) for the untrusted code the agent generates.
- **Resilience built in** — a project-scoped **circuit breaker** and **incremental retry** that feeds the agent the previous diff + the CI error instead of starting from scratch.

---

## 🖼 Screenshots

| Story board (kanban) | Run detail & step timeline |
|---|---|
| <img src="docs/screenshots/07-board.png" width="420"> | <img src="docs/screenshots/03-run-detail.png" width="420"> |

| Configurable pipeline | Agent composition |
|---|---|
| <img src="docs/screenshots/10-pipeline.png" width="420"> | <img src="docs/screenshots/11-agents.png" width="420"> |

| Cost tracking | Human-in-the-loop approvals |
|---|---|
| <img src="docs/screenshots/12-costs.png" width="420"> | <img src="docs/screenshots/13-approvals.png" width="420"> |

---

## 🚀 Run it locally

The whole stack runs in Docker (frontend, API, Postgres, mail, and a filtered Docker socket proxy). Agents use the **Docker substrate by default** (`SUBSTRATE=docker`) — no KVM required.

```bash
# Build images, start the stack, and seed dev data
./scripts/update-stack.sh --reset
```

| Service | URL |
|---------|-----|
| App (Vue SPA via nginx) | http://localhost:5173 |
| API | http://localhost:8080 |
| Mail UI (MailHog) | http://localhost:8025 |
| PostgreSQL | localhost:5432 |

**Seed login:** `admin@hopeitworks.dev` / `admin1234`

> Pipeline execution needs a Git provider with push access (`GITHUB_TOKEN`) and an LLM token for the agents. See [`docs/local-setup.md`](docs/local-setup.md) for the full setup.

### Runtime — the execution substrate

An agent is fundamentally **"exec a harness"**: clone the repo, run the Go `agent-runtime`,
which drives the chosen CLI (`claude` / `opencode`) and reports its outcome back over plain
HTTP. *Where* that exec happens is a separate, **pluggable** concern behind a single
`port.AgentRuntime`. Every substrate is an equal adapter — **Docker is not special** — selected
by the `SUBSTRATE` config:

| Substrate | Role | Needs |
|-----------|------|-------|
| **Docker** (`SUBSTRATE=docker`) | **Default for dev / CI** — runs anywhere a Docker daemon exists (Linux, macOS, Windows) | a Docker daemon |
| **microsandbox** (`SUBSTRATE=microsandbox`) | **Production isolation policy** — libkrun **microVM**, KVM-class kernel isolation for the untrusted code the agent generates; native nested-container fidelity (testcontainers / DinD inside the agent) | Linux + **KVM** (`/dev/kvm`) |
| **exec** (`SUBSTRATE=exec`) | Local inner-loop — runs the harness as a child process, no container, fastest/in-process debuggable; dev-only | — |
| **K8s / OpenShift** | Future (P4) — addable behind the same port (RuntimeClass gVisor / Kata), no domain change | — |

microsandbox is **Linux/KVM-only** and cannot run inside Docker Desktop on macOS (no nested-virt
passthrough). On macOS, run the agent stack inside a Linux VM with nested virtualization (Apple
**M3+ / macOS 15+**) — see [`deploy/lima/README.md`](deploy/lima/README.md) for a one-command,
reproducible setup. (Selecting microsandbox requires the `microsandbox` build tag; otherwise the
adapter is built as a clear no-op stub.)

### Image vs services — the agent image is *where it runs*, sidecars are *what it talks to*

These are two distinct concerns:

- **Agent image (the microVM/container rootfs)** = the environment the agent *runs in*: the
  `agent-runtime` harness + CLI (`claude` / `opencode`) **and the toolchain** (go / node / python)
  it needs to build and test. These come from a **catalogue of stacks** — digest-pinned ghcr
  images (go-node, node, go, python). A microVM, like a container, **always needs an image** (its
  rootfs).
- **Services** (db, redis, keycloak, …) = network **dependencies**, not the agent's runtime. They
  are run as **sidecars** via the **Environment** feature: an isolated per-run network plus
  connection strings injected into the agent's env. They are reachable services, never part of the
  image.
- **Caveat:** sidecars-under-microVM aren't supported by microsandbox yet — it degrades to
  injected connection strings only. The **Docker substrate** gives full sidecars.

### Repository layout

```
backend/        Go API — hexagonal (domain / port / adapter), chi, sqlc, wire, River
frontend/       Vue 3 SPA — PrimeVue 4, Tailwind v4, Pinia, typed OpenAPI client
api/            openapi.yaml — single source of truth for the API contract
agent-runtime/  Go binary executed inside agent containers (HTTP callbacks)
agent-images/   Multi-stack agent Docker images (go-node, node, go, python)
deploy/         docker-compose stack + nginx config
scripts/        Stack lifecycle, dev reset, e2e helpers
docs/           Product vision, setup, design notes
```

---

<div align="center">
<sub>Built with Go, Vue 3, Docker, and PostgreSQL · hexagonal architecture · OpenAPI-first · pluggable agent execution substrate (Docker / microVM)</sub>
</div>
