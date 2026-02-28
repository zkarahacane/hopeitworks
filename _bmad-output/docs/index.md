# Documentation technique — hopeitworks

> Generee le 2026-02-26 par 6 agents Claude en parallele (2 Opus + 4 Sonnet).

## Frontend

| Document | Lignes | Contenu |
|----------|--------|---------|
| [frontend-architecture.md](frontend-architecture.md) | 780 | Architecture Vue 3, 12 features, 20 composables, 12 stores, routing, UI partagee |

## Backend

| Document | Lignes | Contenu |
|----------|--------|---------|
| [backend-core-pipeline.md](backend-core-pipeline.md) | 1461 | Pipeline executor, Runs, DAG, Epic Runs, Actions, Circuit Breaker, Scheduler |
| [backend-auth-users-projects.md](backend-auth-users-projects.md) | 1199 | Auth JWT, Users, Projects, RBAC, Middleware, Password Reset |
| [backend-agents-containers.md](backend-agents-containers.md) | 968 | Agent CRUD, Docker adapter, Container lifecycle, Log streaming, Agent runtime |
| [backend-adapters-infra.md](backend-adapters-infra.md) | 1385 | Postgres, River jobs, Git (GitHub+Gitea), Events/SSE, Handlebars, Config |
| [backend-stories-epics-config.md](backend-stories-epics-config.md) | 1291 | Stories, Epics, Pipeline Config, Cost tracking, HITL, Notifications |

## Diagrammes Mermaid

| Document | Diagrammes | Contenu |
|----------|------------|---------|
| [mermaid-frontend.md](mermaid-frontend.md) | 10 | Architecture stores/API, auth guards, SSE, routing, CSS layers |
| [mermaid-core-pipeline.md](mermaid-core-pipeline.md) | 15 | Run/Step state machines, DAG Kahn, Epic orchestration, HITL, DI init |
| [mermaid-auth-users.md](mermaid-auth-users.md) | 10 | Login/reset flows, JWT validation, circuit breaker, RBAC matrice |
| [mermaid-agents-containers.md](mermaid-agents-containers.md) | 10 | Container lifecycle, AgentRunAction, log streaming, NDJSON parsing |
| [mermaid-adapters-infra.md](mermaid-adapters-infra.md) | 12 | SSE full path, LISTEN/NOTIFY, River jobs, Git dispatch, template chain |
| [mermaid-stories-epics.md](mermaid-stories-epics.md) | 10 | Story lifecycle, cost tracking, HITL approval, notification dispatch |

## Total : ~7000 lignes de doc technique + 67 diagrammes Mermaid (~3900 lignes)
