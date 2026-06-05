# hopeitworks

**AI agent orchestration platform.** Tu planifies où tu veux (BMAD, GSD, Jira, GitHub, markdown) → la plateforme exécute avec des agents par rôle dans des containers Docker, gates HITL, kanban in-app, et un arbre live de l'exécution parallèle.

Positionnement : **couche d'exécution agnostique au planning ET au process**. L'équipe compose ses propres agents (image/modèle/provider/prompt) et son pipeline (aucun workflow imposé, 100% configurable). La plateforme est opiniâtre uniquement sur le runtime — containers, parallélisme, isolation, HITL.

## Stack

| | |
|---|---|
| Backend | Go (chi, pgx, sqlc, wire) — `backend/CLAUDE.md` |
| Frontend | Vue 3 (PrimeVue 4, Tailwind v4, Pinia) — `frontend/CLAUDE.md` |
| API contract | `api/openapi.yaml` — single source of truth |
| Infra | Postgres, Docker, River (job queue), SSE |
| Agent runtime | `agent-runtime/` (binaire Go exécuté dans les containers), `agent-images/` (images Docker) |

Vision produit détaillée : `docs/product.md`.

## Git workflow

```
main        ← production-ready, protected
develop     ← integration branch, PR target
feat/* fix/* chore/*   ← from develop
```

Flow : branch from `develop` → PR → `develop`. `develop` → `main` quand stable.

Commits : `type(scope): message` — impératif, minuscule, sans point final. Types : feat, fix, refactor, test, docs, chore.

## Working style

Construis **directement** — Claude Code + Opus gèrent l'implémentation sans cérémonie multi-rôles. Déléguer à des sous-agents (`Agent` tool, `isolation: "worktree"`) uniquement quand ça aide : tâches de code parallèles, exploration large de la codebase, ou préserver le contexte principal. Pas de role-play PM/architecte/review.

## Development Environment

Devcontainer = code-only. Deux Docker stacks sur le même daemon :

### Stack stable (host, branch develop)
```bash
./scripts/update-stack.sh              # rebuild + restart
./scripts/update-stack.sh --reset      # rebuild + reseed
# Ports : API 8080, Postgres 5432, MailHog 8025
```

### Stack de test agents (safe depuis devcontainer)
```bash
./scripts/agent-stack.sh up            # start stack test isolé
./scripts/agent-stack.sh down          # stop
./scripts/agent-stack.sh reset         # reset DB test uniquement
./scripts/agent-stack.sh status        # health check
# Ports : API 8081, Postgres 5433, MailHog 8026
```

### Bloqué en devcontainer (protège le stack stable)
```bash
./scripts/reset-dev.sh                 # → utiliser le host
./scripts/e2e-stack.sh up              # → utiliser le host
```

Seed credentials: `admin@hopeitworks.dev` / `admin1234`
