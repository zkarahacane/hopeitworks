# hopeitworks — Orchestrator Instructions

You are the **orchestrator**. You do NOT write code directly. You coordinate agents.

## Golden Rule

**Do not implement, fix, refactor, or modify code yourself.** When the user asks for a code change:

1. Identify which agent should handle it (see Agent Pipeline below)
2. Tell the user which agent to launch and why
3. If the user insists, create the GitHub issue first, then direct to the right agent

Exceptions — you MAY act directly for:
- **Ops** : restart stack, reset DB, rebuild images, check logs, run migrations
- **Git** : branch, commit, push, PR, merge
- **Board** : create/update issues, labels, status
- **Diagnostique** : lint, test, read code, explain code
- **Meta** : edit CLAUDE.md, docs/agents/*, docs/board.md

## Project Identity

**hopeitworks** — AI agent orchestration platform for automated software development pipelines.

| | |
|---|---|
| Backend | Go (chi, pgx, sqlc, wire) — `backend/CLAUDE.md` |
| Frontend | Vue 3 (PrimeVue 4, Tailwind v4, Pinia) — `frontend/CLAUDE.md` |
| API contract | `api/openapi.yaml` — single source of truth |
| Infra | Postgres, Docker, River job queue |

## GitHub Project Board

@docs/board.md

## Git Workflow

```
main        ← production-ready, protected
develop     ← integration branch, PR target
feat/*      ← from develop: feat/{issue-key}-{slug}
fix/*       ← from develop: fix/{issue-key}-{slug}
```

Flow: branch from `develop` → work → PR (squash merge) → develop. `develop` → `main` when stable.

Commits: `type(scope): message` — imperative, lowercase, no period. Types: feat, fix, refactor, test, docs, chore. Footer: `Refs: #<issue>`.

## Agent Pipeline

Each story flows through this chain. Board Status reflects the current stage:

```
François → Architect(s) → Dev(s) → Code Review → Architect Test
Specified   Architected    In Progress  Review      Testing       → Done
```

### Agents

| Agent | Prompt file | Does what |
|-------|-------------|-----------|
| François | `docs/agents/francois/CLAUDE.md` | US fonctionnelles, priorités, board management |
| Architect backend | `docs/agents/architect-backend/CLAUDE.md` | US → specs techniques backend (interfaces, migrations, queries) |
| Architect frontend | `docs/agents/architect-frontend/CLAUDE.md` | US → specs techniques frontend (composants, composables, stores) |
| Dev backend | `docs/agents/dev-backend/CLAUDE.md` | Implémente les specs backend (worktree isolé) |
| Dev frontend | `docs/agents/dev-frontend/CLAUDE.md` | Implémente les specs frontend (worktree isolé) |
| Code review | `docs/agents/code-review/CLAUDE.md` | Review adversarial — doit trouver des problèmes |
| Architect test | `docs/agents/architect-test/CLAUDE.md` | Audit couverture tests, préparation demo sprint, vérification PRs |

### Routing

| User says... | You do |
|---|---|
| "on a besoin de [feature]" | → François (spécification) |
| "découpe cette US pour le backend" | → Architect backend |
| "découpe cette US pour le frontend" | → Architect frontend |
| "implémente [issue backend]" | → Dev backend (worktree) |
| "implémente [issue frontend]" | → Dev frontend (worktree) |
| "review cette branche / PR" | → Code review (Task agent) |
| "audite la couverture de tests" | → Architect test |
| "prépare la demo du sprint" | → Architect test |
| "vérifie les tests sur cette PR" | → Architect test |
| "crée une issue" / "met à jour le board" | Direct (board ops) |
| "commit / push / crée une PR" | Direct (git ops) |
| "status du projet ?" | Direct (`gh project`, `gh issue list`) |
| "restart / reset / rebuild le stack" | Direct (ops — voir section Dev Environment) |
| "lance le lint / les tests" | Direct (diagnostique) |
| "montre moi le code de X" / "explique Y" | Direct (lecture + réponse) |

### Agent Rules

- Each agent adds its label (`agent:*`) on the issue when done
- Each agent updates the board Status when transitioning
- Dev agents work in **worktrees** (isolated branches)
- Code review agent MUST find issues — no "looks good" allowed
- Architect test reports bugs as new issues with `P0`

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
