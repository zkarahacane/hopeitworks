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

## Délégation à des sous-agents (worktrees)

### Worktree obligatoire — anti-collision

Tout sous-agent qui **écrit du code** tourne dans son **propre worktree** (`isolation: "worktree"`). Jamais deux agents qui écrivent dans le repo principal, ni sur la même branche, en parallèle — sinon ils se marchent dessus.

Règles non négociables :

1. **1 agent = 1 worktree = 1 branche** (`feat/*|fix/*|chore/*` depuis `develop`).
2. **Périmètres disjoints** : découpe le travail parallèle par zone qui ne se chevauche pas — `backend/`, `frontend/`, `agent-runtime/`, `docs/`. Si deux tâches touchent les mêmes fichiers, **ne les parallélise pas** : séquence-les.
3. **Chaque prompt de délégation écrivant du code commence par le préambule garde-fou** ci-dessous.
4. Dis explicitement à l'agent **quels chemins il possède** et lesquels lui sont interdits.
5. Un agent lecture seule (exploration, audit, review) n'a PAS besoin de worktree.

### Préambule garde-fou (copier en tête de CHAQUE prompt d'agent qui écrit)

> **Avant toute écriture, vérifie ton isolation :**
> 1. `git rev-parse --show-toplevel` → tu dois être dans un worktree `.claude/worktrees/...`, **jamais** dans le repo racine `/Users/.../hopeitworks`.
> 2. `git branch --show-current` → tu dois être sur **ta** branche dédiée (`feat/...|fix/...|chore/...`), **jamais** sur `develop` ni `main`.
> 3. Si l'une des deux échoue : **arrête-toi, n'écris rien**, signale le problème.
> 4. Reste **strictement** dans ton périmètre de fichiers : `<CHEMINS AUTORISÉS>`. Ne touche à rien d'autre.

### Nettoyage des worktrees

Après merge de la branche d'un agent, supprime son worktree. Un worktree resté **locké** (process agent mal terminé) se force :

```bash
git worktree list                       # voir les worktrees actifs (dont .claude/worktrees/)
git worktree remove <path>              # suppression propre
git worktree remove -f -f <path>        # forcer si locké
git worktree prune                      # nettoyer les références mortes
```

Vérifie `git -C <path> status` **avant** de supprimer : ne jette jamais un worktree avec du travail non commité.

## Definition of Done

Une US / feature n'est **DONE** que si **tous** ces points sont couverts — ou explicitement déclarés _non applicables_ (et tu dis pourquoi) :

- [ ] **Backend** — code + tests (unit, + integration testcontainers si DB) + `golangci-lint` vert.
- [ ] **Frontend** — le pendant UI existe (view / feature / store / composable). **Aucune US backend ne se livre sans son volet front.** Si vraiment pas de front, écris-le noir sur blanc.
- [ ] **API contract** — `api/openapi.yaml` à jour + types régénérés des **deux** côtés (`make generate` back, `npm run generate-api` front).
- [ ] **E2E Playwright** — spec ajoutée ou mise à jour dans `frontend/e2e/` couvrant le parcours user touché.
- [ ] **Doc fonctionnelle** — `docs/product.md` reflète tout comportement user-visible nouveau ou modifié.
- [ ] **CI verte** sur `develop`.

### Règle anti-oubli front

Par **défaut, toute US a un volet front.** Quand tu découpes une US, liste d'abord ce qui change côté UI ; back-only est l'exception qui se justifie, pas la norme. Si tu délègues le back et le front à deux agents parallèles, ils partagent le **même** contrat `api/openapi.yaml` — fige-le d'abord, puis fan-out.

### DoD reviewer (fin de tâche)

Avant d'annoncer « terminé » sur une tâche qui touche au produit, lance un sous-agent reviewer (Sonnet, ou Haiku pour un petit diff) qui audite le diff contre la DoD et retourne les manques. Prompt type :

> Audite le diff `git diff develop...HEAD` (lecture seule) contre cette Definition of Done :
> back+tests+lint / **front présent** / openapi.yaml + types régénérés / **spec Playwright** ajoutée ou mise à jour / **docs/product.md** mis à jour si l'UX change / CI.
> Pour chaque point : ✅ couvert, ⚠️ partiel, ❌ manquant — avec le fichier qui le prouve (ou son absence). Ne corrige rien, retourne juste le rapport.

Traite chaque ❌ avant de clore — ou justifie le _non applicable_.

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

### Substrat microsandbox (microVM) — Mac → VM Lima

Le **défaut code/CI est Docker** (`SUBSTRATE=docker`, tourne partout). Depuis la migration
substrate-abstraction (ADR `docs/agent-substrate-abstraction-adr.md`), `agent_run` exécute **via
`port.AgentRuntime`** : Docker est un adapter derrière le port, égal à microsandbox/exec. **En prod
le substrat-policy est microsandbox** (`deploy/docker-compose.microsandbox.yml`, `SUBSTRATE=microsandbox`)
— l'agent tourne dans un microVM durci ; `SUBSTRATE` est désormais **live-wired** (sélectionner
microsandbox injecte vraiment l'adapter microVM ; sans `-tags microsandbox`, `Launch` renvoie
`ErrNotBuilt` et le run échoue clairement). Le substrat durci **microsandbox** (microVM libkrun) est
**Linux/KVM-only** : il a besoin de `/dev/kvm` et **Docker Desktop sur macOS ne l'expose pas aux
conteneurs** (pas de nested-virt passthrough). Sur Mac, on le fait donc tourner dans une **VM Linux
Lima** avec virtu imbriquée (Apple **M3+ / macOS 15+**) :

```bash
brew install lima
limactl start ./deploy/lima/microsandbox-vm.yaml --name microsandbox --tty=false
# → VM Linux : /dev/kvm fonctionnel + Docker + microsandbox installés
limactl shell microsandbox          # entrer dans la VM (y déployer la stack agent-test, SUBSTRATE=microsandbox)
limactl stop microsandbox           # libérer la RAM   |   limactl delete microsandbox
```

- Le code SDK microsandbox est derrière `//go:build microsandbox` → build par défaut/CI inchangé ;
  l'image se build via `backend/Dockerfile.microsandbox` (CGO, glibc 2.39 / ubuntu:24.04).
- Lima **auto-forwarde** les ports de la VM vers `localhost` du Mac (0 conf réseau).
- Détails + runbook + pièges : [`deploy/lima/README.md`](deploy/lima/README.md).
- Validé sur Apple M4 Pro / macOS 26 : `KVM_CREATE_VM` ✓, `msb run` boote un vrai microVM.

Seed credentials: `admin@hopeitworks.dev` / `admin1234`
