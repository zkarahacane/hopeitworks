---
stepsCompleted: [1, 2, 3, 4]
inputDocuments: []
session_topic: 'Architecture Hopeworks v2 - orchestrateur collaboratif pour usine logicielle IA'
session_goals: 'Résoudre problèmes UX v1, passer à solution collaborative multi-users, architecture robuste state+events, déployable K8s/Docker'
selected_approach: 'AI-Recommended Techniques'
techniques_used: ['First Principles Thinking', 'Morphological Analysis', 'Cross-Pollination']
ideas_generated: [51]
context_file: ''
session_active: false
workflow_completed: true
---

# Brainstorming Session Results

**Facilitator:** Zakari
**Date:** 2026-02-15

## Session Overview

**Topic:** Architecture Hopeworks v2 - orchestrateur collaboratif pour usine logicielle IA

**Goals:**
- Résoudre les problèmes UX de v1 (CLI difficile pour Epic/Stories management)
- Passer d'un outil solo à une solution collaborative multi-users/teams
- Architecture robuste avec state management + event system
- Déployable K8s/Docker avec jobs ou agents sur postes dev
- Maintenir les avantages de v1 (isolation, cost tracking, dogfooding)

### Session Setup

Session initialisée avec approche AI-Recommended - techniques personnalisées basées sur le contexte de migration v1→v2 et architecture distribuée collaborative.

## Technique Selection

**Approach:** AI-Recommended Techniques
**Analysis Context:** Architecture système distribué avec focus sur migration v1→v2, collaboration multi-user, et décisions d'architecture technique

**Recommended Techniques:**

1. **First Principles Thinking** (Creative) - Déconstruction des vérités fondamentales de v1 pour identifier les problèmes racines et les principes immuables de v2. Permet de distinguer symptômes vs causes réelles.

2. **Morphological Analysis** (Deep) - Exploration systématique de toutes les combinaisons d'architecture possibles (state backend, runtime, UI, auth) pour identifier les patterns optimaux de façon exhaustive.

3. **Cross-Pollination** (Creative) - Inspiration depuis orchestrateurs et platforms existants (Temporal, Argo Workflows, Backstage, GitLab CI) pour adopter les patterns qui marchent et éviter de réinventer.

**AI Rationale:** Séquence calibrée pour profil DevOps/SRE avec besoin d'architecture technique robuste. Progression logique : déconstruire → explorer systématiquement → valider avec patterns éprouvés.

## Technique Execution Results

### First Principles Thinking (30 ideas)

**Fundamental Truths Identified:**
1. Code quality is non-negotiable (output constraint, not just a goal)
2. Automation + human intervention spectrum (not binary)
3. Simple to use (UX is the missing piece from v1)
4. Complementary to existing planning tools (BMAD/GSD/Jira), not a replacement

**Key Ideas Generated:**

**Architecture & Infrastructure:**
- #1: Quality Gates as First-Class Citizens - architectural barrier, not optional
- #8: Workflow Customizable per Team - CI remains central, custom steps around it
- #20: CI Remains Sacred - Hopeworks adapts around existing CI, never replaces
- #21: CI Result Parser Agents - one agent observes CI results, another acts on them
- #22/31: Simple Agent Sandboxing - 3 levels (restricted/standard/privileged), not RBAC jungle
- #26/35: Docker Compose + CLI - transparent setup, no magic commands
- #27/36: OSS-First, Enterprise Later - not a blocker for MVP
- #38: Postgres as State Backend - not SQLite (too many files, not multi-user)
- #39: Hybrid Runtime - Docker local (dev solo) + K8s Jobs (team collab)
- #42: Argo-Style State Visualization - workflow DAG in web UI
- #43: Simple Setup - `docker-compose up` and running in 2 min

**Agent Orchestration & Workflow:**
- #2: Human-in-the-Loop Spectrum - continuum from Full Auto to Manual
- #3: Progressive Autonomy Based on Track Record - system learns what to auto-approve
- #9: Agent separation of concerns - parser agent + fixer agent, not hybrid code
- #10: Default Workflows + Marketplace - CI remains central, agents cover what CI doesn't
- #11/34: Functional Testing via Sandboxed Agents - E2E validation in ephemeral containers
- #12: MVP-First Strategy - MVP mode (fast, auto) vs V1 mode (supervised, quality)
- #13: Metrics-Driven Recommendations - proactive advice based on SonarQube/CI results
- #25/32/33: Pluggable Analyzers from CI Results - reuse CI artifacts, don't re-run tools
- #30: Configurable Human Gates - granular per project/epic/story, human-only "done" option
- #37: Pause-and-Resume in Auto Mode - emergency brake even in full auto

**UX & Interface:**
- #4: Git-Native Interface - repo as interface for IaC repeatability
- #5/40: Web-First + CLI Day-1 - both first-class citizens
- #14: Tri-Modal Interface (Web + CLI + Git) - all three interoperable
- #15: ArgoCD-Style CLI Auth - `hopeworks login <url>`, always connected to instance
- #44: Accessible but Not Dumbed-Down - Lovable-inspired, but not simplistic

**Story Management & Integration:**
- #6: BMAD-to-Hopeworks Pipeline - BMAD = planning, Hopeworks = execution
- #7: Universal Story Format - multi-source support
- #16: Pluggable Story Sources + Internal Fallback - external (Jira/BMAD) or built-in
- #17: Lightweight Stories (< 2KB) - stories are pointers, agents enrich context dynamically
- #23: Project Import from Existing Tools - try Hopeworks on 1-2 epics, no big bang
- #24: Brownfield-Friendly - respect existing projects, don't bulldoze
- #28: MCP-Exposed Story API - Claude can pilot Hopeworks natively
- #41: Story Converter/Adapter - MVP-friendly manual import, native integrations post-MVP

**Observability & Cost:**
- #18: Token Budget Management - tracking mandatory, limits optional
- #19: Auto-Issue Management - create/close/label issues, but human gates configurable
- #29: Token Tracking Mandatory - observability feature, not business constraint

### Morphological Analysis (Express - 6 ideas)

**Architecture Decisions:**
- State Backend: **Postgres** (not SQLite - multi-user, no file bloat)
- Agent Runtime: **Hybrid** (Docker local for dev solo, K8s for team collab)
- UI: **Web-first + CLI day-1** (both first-class)
- Story Source: **Internal + converter** (pluggable sources post-MVP)
- Deployment: **Docker Compose** (local), K8s (team) - same codebase
- Auth: **Keycloak** (don't reinvent auth, config as code)

### Cross-Pollination (Express - 3 ideas)

**Inspirations:**
- **Argo Workflows**: UI + state visualization (workflow DAG, status, logs) - LOVED
- **ArgoCD**: CLI auth model (`argocd login`), declarative config
- **Lovable**: Accessible UX for beginners, rapid iteration
- **Backstage**: REJECTED - too complex to setup, anti-pattern for Hopeworks

### Additional Ideas from Technical Discussion (7 ideas)

- #45: Release Management Agent - dedicated Opus agent for semantic versioning, changelogs, GitHub Releases
- #46: Reference Test Project (todo app) - real project for agent validation with build + seed SQL + E2E
- #47: Bootstrap Dogfooding Scripts - bash scripts to run Claude Code agents in Docker during development
- #48: Context-Aware CLAUDE.md - prevents context pollution, handoff system when context saturates
- #49: Event-Driven Architecture (Redis/RabbitMQ) - everything is an event, queue-based decoupling
- #50: Cross-Repo Version Sync - umbrella versioning across 3 repos, compatibility matrix
- #51: Claude Native Sub-Agents in Docker - let Claude Code orchestrate, agents run in containers

## Idea Organization and Prioritization

### Thematic Organization

**Theme 1: Architecture & Infrastructure** (11 ideas)
Core: #38 Postgres, #39 Hybrid Runtime, #20 CI Sacred, #21 CI Parser, #35 Docker Compose, #43 Simple Setup
Post-MVP: #31 RBAC, #42 Argo Viz, #1 Quality Gates config, #8 Custom Workflows, #36 OSS licensing

**Theme 2: Agent Orchestration & Workflow** (12 ideas)
Core: #2 HITL Spectrum, #30 Human Gates, #37 Pause/Resume, #12 MVP-First, #21 CI Parser+Fixer
Post-MVP: #3 Progressive Autonomy, #10 Marketplace, #11 E2E Testing, #25 Tech Debt, #32/#33 Analyzers

**Theme 3: UX & Interface** (6 ideas) - PRIORITY #1
Core: #40 Web-First+CLI, #5 Web Dashboard, #15 CLI Auth, #44 Accessible UX
Post-MVP: #14 Tri-Modal, #42 DAG Viz, #4 Git-Native

**Theme 4: Story Management & Integration** (8 ideas)
Core: #17 Lightweight Stories, #16 Internal System, #28 MCP API (basic)
Post-MVP: #41 Story Converter, #23 Project Import, #24 Brownfield, #6/#7 BMAD Pipeline

**Theme 5: Observability & Cost** (4 ideas)
Core: #29 Token Tracking Mandatory
Post-MVP: #13 Auto-Tuning, #18 Budget Limits, #19 Auto-Issue Management

**Theme 6: DevOps & Release** (7 ideas)
Core: #45 Release Agent, #47 Bootstrap Scripts, #48 CLAUDE.md, #49 Event-Driven, #50 Version Sync
Post-MVP: #46 Test Project (todo app), #51 Claude Sub-Agents refinement

### User Learnings (Design Constraints)
1. Too much context pollutes agent output - be precise in context and instructions
2. Automate and validate tests end-to-end (code to final usage) is primordial
3. Main conversation must not be polluted - agents run in containerized environments
4. Parallelize development tasks whenever possible
5. Build isolated staging environment (local first, cloud if budget)
6. No monolith - separate back/front for context segmentation in agent Docker environments
7. Maintain OpenAPI contract between repos, stop frontend if API dependency is major
8. Mock data is critical for testing (todo app + build + SQL seed)

## Action Plan

### Technical Decisions

| Element | Choice |
|---------|--------|
| Backend | Go + Hexagonal Architecture |
| Frontend | Vue 3 + TypeScript strict + Vite |
| CLI | Go (same as backend) |
| Auth | Keycloak (Docker + config as code) |
| DB | Postgres (state) |
| Queue | Redis Streams (events) |
| Repos | 3 separate repos (api, ui, cli) + test-todoapp |
| Contract | OpenAPI generated from Go backend |
| Agent Runtime | Docker containers with Claude Code |
| Versioning | Semantic versioning, umbrella versions, conventional commits |

### Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  hopeworks-ui│     │ hopeworks-cli│     │  Claude Main  │
│   (Vue 3)    │     │    (Go)      │     │ (orchestrate) │
└──────┬───────┘     └──────┬───────┘     └──────┬───────┘
       │ WebSocket          │ REST                │ Sub-agents
       │                    │                     │ in Docker
┌──────▼────────────────────▼─────────────────────▼───────┐
│                    hopeworks-api (Go)                     │
│                  Hexagonal Architecture                   │
│  Domain: Stories, Runs, Steps, Agents, Events            │
│  Ports: REST API, WebSocket, MCP, Queue Consumer         │
└──────┬──────────────┬──────────────┬────────────────────┘
       │              │              │
┌──────▼──────┐ ┌─────▼─────┐ ┌─────▼──────┐
│  Postgres   │ │   Redis    │ │  Docker    │
│  (state)    │ │ (events/   │ │  (agent    │
│             │ │  queues)   │ │  runtime)  │
└─────────────┘ └───────────┘ └────────────┘
```

### Docker Compose Stack
- api (Go backend)
- ui (Vue 3 frontend)
- postgres (state DB)
- redis (events/queues)
- keycloak (auth)

### Epics

| # | Epic | Repos | Version |
|---|------|-------|---------|
| 0 | Bootstrap (repos, CI, docker-compose, Keycloak) | all | v0.0.1 |
| 1 | API Foundations (domain, hexagonal, OpenAPI, story CRUD) | api | v0.1.0 |
| 2 | Web UI MVP (stories, logs, run controls, prompt editor) | ui | v0.2.0 |
| 3 | CLI MVP (login, run, status, logs) | cli | v0.3.0 |
| 4 | Agent Orchestration (pipeline, implement, CI fix, merge) | api | v0.4.0 |
| 5 | Stories + MCP (MCP server, import adapter) | api | v0.5.0 |
| 6 | Test Project (todo app, seed, E2E) | test-todoapp | v0.6.0 |
| 7 | Dogfooding (Hopeworks builds itself) | all | v1.0.0 |

### Dogfooding Strategy
- Phase 1: Bootstrap scripts (bash) to run Claude Code agents in Docker
- Phase 2: CLAUDE.md per repo preventing context pollution
- Phase 3: Handoff system when context saturates
- Phase 4: Hopeworks v2 orchestrates its own development

## Session Summary and Insights

**Key Achievements:**
- 51 breakthrough ideas generated across 3 techniques
- Clear architecture vision: event-driven, hexagonal, 3 repos, Docker+K8s
- MVP scope defined: 8 epics, ~33 stories, estimated ~$210-310
- Dogfooding strategy from day 1

**Breakthrough Moments:**
- CI remains sacred (don't replace, adapt around) - fundamental design principle
- Event-driven with Redis queues - transforms from synchronous v1 to async v2
- Separate repos for context isolation in agent containers - key insight from v1 learnings
- Keycloak for auth - don't reinvent, configure as code
- Claude main as orchestrator, sub-agents in Docker - natural usage pattern

**User Creative Strengths:**
- Zakari brings deep operational experience (DevOps/SRE) to architectural decisions
- Strong bias toward pragmatism: "don't over-engineer, but don't cut corners"
- Excellent pattern recognition from v1 learnings and real-world tool usage (ArgoCD, K8s)
- Clear vision of what NOT to do (anti-Backstage complexity, anti-RBAC jungle)
