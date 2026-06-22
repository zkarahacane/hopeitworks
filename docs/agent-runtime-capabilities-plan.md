# Plan — Runtime pluggable & couche de capacités des agents

> Refonte de la couche d'exécution (runtime) et des capacités (skills / MCP / tool-policy / stacks / environment) des agents de hopeitworks : adapters de runtime derrière le port hexagonal, capacités comme données plateforme agnostiques, injection fetch-at-startup, catalogue de stacks/environments, et substrats d'isolation par contexte.
> Date : 2026-06-22. Ce plan **supersede** les docs existants sur le substrat/runtime et vient **par-dessus** la boucle review→fix et la mémoire pgvector (voir § Relation aux docs existants).

---

## Résumé exécutif

**État actuel.** Les agents tournent dans des containers Docker créés via le SDK Docker derrière le port `ContainerManager` (`backend/internal/domain/port/container_manager.go`, impl. `docker.NewDockerContainerManager` câblée sous l'alias `NewAgentRuntime`). Le mode d'exécution est détecté par une string magique — `strings.Contains(agentImage, "hopeitworks/agent-")` (`backend/internal/adapter/action/agent_run.go:533`) — sur un champ `agents.image` **string libre non validée** (`backend/internal/domain/model/agent.go:27`). Le container exécute `claude -p` ou `opencode` via `agent-runtime` (provider choisi par l'env `PROVIDER`, `agent-runtime/internal/provider/provider.go:29`). Aucune notion de skill, serveur MCP, LSP, tool-policy, stack pinnée ni environment de projet ; tout passe par l'env (qui fuite dans `docker inspect` et plafonne en taille).

**Cible.**
- Un **port runtime stable** avec des **adapters pluggables** (Claude Code, CMA, OpenCode, …) ; un agent déclare `runtime + model + provider + capabilities`. Pas d'Anthropic-only.
- Les **capacités** (skills, serveurs MCP, tool-policy) deviennent des **données plateforme** versionnées, scope `global`/`project`, composées sur l'agent par jointure ; chaque adapter expose `SupportedCapabilities()` et traduit via `Provision()` (non supporté → warn+skip).
- **Injection fetch-at-startup** : l'`agent-runtime` récupère son bundle (skills + `.mcp.json` + system prompt + secrets) depuis l'API au démarrage, authentifié par le container-token. Fin de l'injection par env.
- **MCP = services HTTP réseau** par défaut (catalogue curé, réseau interne, RBAC tool-level, audit, egress default-deny) ; **LSP in-image** ; **images = catalogue de stacks** pinnées multi-arch ; **environment par projet** (stacks + sidecars + source devcontainer/compose/Makefile du repo) avec **golden images pré-seedées**.
- **Substrat & isolation par contexte** : microsandbox (microVM, défaut si KVM), adapter K8s + gVisor (fallback durci sans KVM), CMA-cloud en option ; invariant : tout reste **K8s-Pod-exprimable** (jamais DinD ni microVM comme hypothèse de domaine).
- **Aucun secret baké** : plan donnée injectée au runtime, plan binaire = image ou service réseau.

---

## Décisions actées (le contrat — 14 points)

1. **Runtime = adapters pluggables** derrière le port hexagonal existant (`ContainerManager` aujourd'hui, à généraliser en `AgentRuntime`). Adapters : `ClaudeCodeRuntime`, `CMARuntime`, `OpenCodeRuntime`, extensible. Un agent déclare runtime + modèle + provider + capabilities. Pas d'Anthropic-only ; CMA = option, jamais dépendance.
2. **Ne pas reconstruire le harness coding en API brute** (ça réinvente Claude Code ; pas d'Agent SDK Go). Les adapters **wrappent** des harness existants : CLI `claude`, CLI `opencode`, service CMA.
3. **Capacités = données plateforme** agnostiques au runtime, scope `global`/`project`, versionnées, composées via `agent_capabilities`. Chaque adapter expose `SupportedCapabilities()` et traduit la spec agnostique via `Provision()`. Non supporté → **warn+skip**, jamais block.
4. **Deux plans.** DONNÉE (texte des skills, `.mcp.json`, system prompt, credentials) = injectée au runtime. BINAIRE (toolchains, CLI, LSP, binaires MCP) = image OU service réseau. **Jamais de secret baké.**
5. **Injection = fetch-at-startup** : l'`agent-runtime` récupère son bundle de capacités (skills + mcp + prompt + secrets) depuis l'API au démarrage, authentifié par le **container-token** (extension du canal callback). Pas l'env (taille + fuite `docker inspect`).
6. **MCP = services HTTP réseau** par défaut (URL + auth enregistrées, 0 rebuild d'image) ; stdio-in-image seulement pour l'ultra-sensible. Sécurité : catalogue curé admin, réseau interne, RBAC tool-level (un agent review ne **voit** pas les outils d'écriture), audit de chaque appel, egress default-deny par serveur.
7. **LSP reste in-image** (besoin des fichiers locaux du repo), par stack.
8. **Image = catalogue de stacks** : entité `stack` (`go`/`node`/`python`/`go-node`) → image pinnée, multi-arch, portant toolchain + CLI runtime + LSP. L'agent référence runtime + stack ; **abandon de la string libre `agents.image`**. Tue : string non validée, couplage `strings.Contains(image,"hopeitworks/agent-")`, absence de pull.
9. **Environment (par projet, ≠ image de stack)** : stack(s) + `services[]` (sidecars postgres/redis/mailhog…) + source = `devcontainer.json` / `docker-compose.yml` / `Makefile` **du repo** si présent (sinon stack + services déclarés en UI). Au run : composer l'image + lever les sidecars sur un réseau isolé par run + injecter les conn-strings + exécuter les commandes du projet (make test, migrations, seed).
10. **Coût du seed = golden images pré-seedées** (seed au build/CI, clone copy-on-write par run) ; pas de re-seed par run. Ce que les vendeurs appellent « snapshots » (Daytona/E2B/CMA).
11. **Invariant de portabilité : Environment K8s-Pod-exprimable.** Jamais DinD ni microVM comme hypothèse de domaine. Services = sidecars-in-Pod. testcontainers (que le backend hopeitworks utilise déjà via testcontainers-go) = **KubeDock** sur K8s, pas DinD/sysbox. Isolation = capability d'adapter.
12. **Substrat & isolation par contexte** (pas un choix global) :
    - Hôte/pool **avec KVM** → **microsandbox** (libkrun microVM) : **le défaut** (vrai kernel → testcontainers/DinD marchent dedans, perf quasi-native).
    - K8s/OpenShift **durci sans KVM** → adapter K8s `runtimeClassName: gvisor` (kernel userspace, OpenShift-viable), au prix d'un coût perf/compat syscall (testcontainers sous gVisor → KubeDock + sidecar Postgres).
    - CMA-cloud contourne l'infra locale, en option.
13. **gVisor vs microsandbox = couches différentes** : gVisor = RuntimeClass d'isolation (sans KVM) ; microsandbox = isolation microVM + SDK de gestion. Comparables seulement sur l'axe isolation. microVM = isolation plus forte + meilleure fidélité coding mais exige KVM ; gVisor = fallback sans-KVM.
14. **Build first = microsandbox** (décidé 2026-06-22 : on est host Docker, KVM dispo, meilleure fidélité coding). L'adapter K8s/gVisor viendra quand un client OpenShift sans KVM arrivera ; le port + l'invariant Pod-exprimable rendent l'ajout non-bloquant.

---

## 1. Architecture runtime — adapters derrière le port

### 1.1 Le port aujourd'hui

Le port actuel est `ContainerManager` (`backend/internal/domain/port/container_manager.go`) : un CRUD de container Docker bas niveau (`Create/Start/Stop/Remove/Wait/ListContainers`) prenant un `model.ContainerOpts`. L'impl. unique est `docker.ContainerManager` (`NewDockerContainerManager`, câblée sous l'alias wire `NewAgentRuntime`). Le port décrit **comment manipuler un container Docker**, pas **comment exécuter un agent** — il fuit le substrat dans le domaine.

`AgentRunAction` (`backend/internal/adapter/action/agent_run.go`) orchestre : `createContainer` (env + image), détection de mode par `isCallbackMode` (`:532`, `strings.Contains(agentImage, "hopeitworks/agent-")`), puis stream/wait. Le couplage substrat ↔ logique d'agent y est total.

### 1.2 Le port cible : `AgentRuntime`

Généraliser le port en une abstraction **orientée exécution d'agent**, pas container Docker :

```go
// backend/internal/domain/port/agent_runtime.go
type AgentRuntime interface {
    // Provision applique la spec de capacités agnostique au mécanisme natif de l'adapter.
    // Capacité non supportée -> warn + skip (jamais d'erreur bloquante).
    Provision(ctx context.Context, spec model.CapabilitySpec) (model.ProvisionResult, error)
    // Launch démarre une exécution d'agent (clone + harness + capacités) et renvoie un handle.
    Launch(ctx context.Context, spec model.RunSpec) (model.RunHandle, error)
    Wait(ctx context.Context, h model.RunHandle) (model.RunResult, error)
    Stop(ctx context.Context, h model.RunHandle) error
    // SupportedCapabilities déclare ce que l'adapter sait traduire (skills/mcp-http/mcp-stdio/tool-policy/lsp...).
    SupportedCapabilities() model.CapabilitySet
}
```

`ContainerManager` reste une **dépendance interne** de l'adapter substrat (Docker/microsandbox/K8s), pas un port de domaine.

### 1.3 Les adapters

| Adapter | Harness wrappé | Modèles/providers | Notes |
|---|---|---|---|
| `ClaudeCodeRuntime` | CLI `claude -p` | Anthropic | Reprend l'`agent-runtime` actuel (`provider/claude.go`) ; ajoute fetch-at-startup + provision. |
| `OpenCodeRuntime` | CLI `opencode` | gpt / gemini / deepseek / … (multi-modèle) | Déjà amorcé (`provider/opencode.go`, `provider.New` `:33`). |
| `CMARuntime` | service Anthropic Managed Agents | Anthropic (cloud) | **Option**, jamais dépendance. Contourne l'infra locale. |
| (extensible) | … | … | Nouveau harness = nouvel adapter, sans toucher au domaine. |

**Principe (décision #2)** : aucun adapter ne réimplémente la boucle agentique en appels API bruts ; ils pilotent un harness existant (CLI ou service). L'Agent SDK Anthropic n'existe pas en Go → on wrappe les CLI.

### 1.4 Modèle d'agent cible

`model.Agent` (`backend/internal/domain/model/agent.go`) porte aujourd'hui `Model`, `Image` (string libre), `Provider` (`claude`/`opencode`), `Type`, `Scope`. Cible :

- **remplacer `Image`** (string libre) par `RuntimeKind` + `StackRef` (FK catalogue) ;
- garder `Model` + `Provider` (sémantique runtime) ;
- ajouter la **jointure `agent_capabilities`** (skills / mcp / tool-policy composées).

Le couplage `strings.Contains(image,"hopeitworks/agent-")` disparaît : le mode (callback/fetch-at-startup) découle du `RuntimeKind`, pas d'une heuristique de string.

---

## 2. Capacités — modèle de données & authoring

### 2.1 Capacités = données plateforme (décision #3)

Trois familles de capacités, toutes **versionnées**, scope `global` (admin) ou `project`, et **agnostiques au runtime** :

| Famille | Contenu (plan DONNÉE) | Plan BINAIRE associé |
|---|---|---|
| `skill` | `SKILL.md` + ressources (scripts, rubrics) | aucun (texte pur) ou outils déjà dans l'image |
| `mcp_server` | URL + auth (HTTP) **ou** réf. binaire stdio (ultra-sensible) | service réseau (défaut) **ou** binaire in-image |
| `tool_policy` | allow/deny par outil, par rôle | aucun |

Schéma (esquisse) :

```sql
-- capabilities : entité versionnée, scope global/project
CREATE TABLE capabilities (
  id          uuid PRIMARY KEY,
  kind        text NOT NULL,         -- 'skill' | 'mcp_server' | 'tool_policy'
  name        text NOT NULL,
  version     int  NOT NULL,
  scope       text NOT NULL,         -- 'global' | 'project'
  project_id  uuid NULL REFERENCES projects(id),
  spec        jsonb NOT NULL,        -- spec agnostique (texte skill / url+auth mcp / allow-deny)
  created_at  timestamptz NOT NULL DEFAULT now()
);
-- composition sur l'agent
CREATE TABLE agent_capabilities (
  agent_id      uuid NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
  capability_id uuid NOT NULL REFERENCES capabilities(id),
  PRIMARY KEY (agent_id, capability_id)
);
```

Les **credentials** (token MCP, clé provider) sont référencés par la spec mais stockés chiffrés à part (jamais dans `spec` en clair, jamais bakés).

### 2.2 Provision : agnostique → natif (décision #3)

Au lancement, le runtime appelle `Provision(spec)` :

1. la plateforme assemble la `CapabilitySpec` de l'agent (jointure `agent_capabilities`) ;
2. l'adapter compare avec `SupportedCapabilities()` ;
3. supporté → traduit vers le mécanisme natif (ex. skill → `.claude/skills/<name>/SKILL.md` ; mcp_server HTTP → entrée `.mcp.json` ; tool_policy → `--allowedTools/--disallowedTools`) ;
4. **non supporté → warn + skip** (l'agent tourne dégradé, jamais bloqué).

### 2.3 Authoring (admin / projet)

- UI admin : CRUD capacités globales (catalogue curé), édition de `SKILL.md`, enregistrement de serveurs MCP (URL + auth + liste d'outils exposés + RBAC rôle), tool-policies par rôle.
- UI projet : capacités scope `project` (skills maison, MCP interne), sans toucher le catalogue global.
- Versionnement : une capacité éditée crée une nouvelle `version` ; l'agent épingle (ou suit `latest` selon politique).

Ceci **remplace** le `buildClaudeMD` en string switch (rôle hardcodé dans `agent_run.go`) par des skills versionnés et composables (cf. agent-images-tools-skills-plan.md, qui identifiait déjà ce problème).

---

## 3. Injection & sécurité MCP

### 3.1 Fetch-at-startup (décision #5)

**Aujourd'hui** : tout passe par l'env du container (`PROVIDER`, `CLAUDE_MD_CONTENT`, `PROMPT`… via `createContainer`/`buildEnv` dans `agent_run.go`, consommés par `config.Load` de l'`agent-runtime`). Limites : taille d'env, et **fuite dans `docker inspect`** (secrets visibles).

**Cible** : au démarrage, l'`agent-runtime` appelle un endpoint `GET /agent-runtime/bundle` de l'API hopeitworks, **authentifié par le container-token** (déjà émis pour le canal callback, cf. `container_token_store`). Réponse = bundle :

```json
{
  "system_prompt": "...",
  "skills": [{ "name": "...", "files": { "SKILL.md": "...", "scripts/x.sh": "..." } }],
  "mcp": { "mcpServers": { "kanban": { "url": "http://mcp-kanban.internal/...", "headers": {"Authorization":"..."} } } },
  "tool_policy": { "allow": ["Read","Grep"], "deny": ["Bash(rm:*)"] },
  "credentials": { "ANTHROPIC_API_KEY": "..." }
}
```

L'`agent-runtime` matérialise skills/`.mcp.json`/system prompt sur disque puis lance le harness. **Aucun secret en env, aucun secret baké** (décision #4). Le container-token est court-vécu et lié au run.

### 3.2 MCP = services HTTP réseau (décisions #4, #6)

- **Défaut : URL + auth enregistrées** dans le catalogue → MCP custom = 0 rebuild d'image. Le binaire MCP tourne en **service réseau** (sidecar ou service interne), pas baké.
- **stdio-in-image** : réservé à l'ultra-sensible (pas d'exposition réseau acceptable).

**Sécurité MCP (surface d'attaque #1 du plan)** :

| Contrôle | Mise en œuvre |
|---|---|
| Catalogue curé | seuls les serveurs enregistrés par l'admin sont injectables |
| Réseau interne uniquement | MCP non exposés publiquement ; réseau isolé par run |
| RBAC tool-level | un agent `review` ne **voit** pas les outils d'écriture (filtrage à la génération du `.mcp.json` + tool_policy) |
| Audit | chaque appel d'outil MCP loggé (qui/quoi/quand) |
| Egress default-deny | par serveur MCP ; allow-list explicite |

### 3.3 LSP in-image (décision #7)

Le LSP a besoin des **fichiers locaux du repo cloné** → il reste dans l'image, **par stack** (`gopls`, `typescript-language-server`, `pyright`/`ruff`…). Ce n'est pas une capacité injectée mais une propriété de la stack (§4).

---

## 4. Images = catalogue de stacks & Environment

### 4.1 Catalogue de stacks (décision #8)

Aujourd'hui les images existent (`agent-images/stacks/{go,node,go-node,python}` + `base`, build via `agent-images/Makefile` et `.github/workflows/agent-images.yml`) mais sont référencées par la **string libre `agents.image`**. Cible : une **entité `stack`** :

```sql
CREATE TABLE stacks (
  id           uuid PRIMARY KEY,
  key          text UNIQUE NOT NULL,   -- 'go' | 'node' | 'python' | 'go-node'
  image_ref    text NOT NULL,          -- digest pinné, multi-arch
  toolchain    jsonb NOT NULL,         -- versions toolchain + CLI runtime + LSP
  created_at   timestamptz NOT NULL DEFAULT now()
);
```

Chaque stack = image **pinnée par digest**, **multi-arch** (amd64 + arm64), portant **toolchain + CLI runtime (`claude`/`opencode`) + LSP**. L'agent référence `runtime + stack` (FK), plus de string libre. La plateforme **possède les refs** → pull déterministe (corrige l'« absence de pull » de l'état actuel).

> Reprend les correctifs d'images de agent-images-tools-skills-plan.md (pinning, dédup `go-node`, multi-arch, stack python enrichie) — qui restent valides et s'intègrent ici comme propriétés de l'entité `stack`.

### 4.2 Environment par projet (décision #9)

`environment` ≠ image de stack : c'est la **composition d'exécution d'un projet**.

```sql
CREATE TABLE environments (
  id          uuid PRIMARY KEY,
  project_id  uuid NOT NULL REFERENCES projects(id),
  stacks      jsonb NOT NULL,   -- une ou plusieurs stack keys
  services    jsonb NOT NULL,   -- sidecars: [{name:'postgres',image:'...',env:{...}}, ...]
  source      text NOT NULL,    -- 'devcontainer' | 'compose' | 'makefile' | 'declared'
  commands    jsonb NOT NULL,   -- {test:'make test', migrate:'...', seed:'...'}
  created_at  timestamptz NOT NULL DEFAULT now()
);
```

**Source de vérité** : si le repo contient `devcontainer.json` / `docker-compose.yml` / `Makefile`, l'environment en **dérive** ; sinon stack + services déclarés en UI.

**Au run** :
1. composer l'image (stack + couches projet) ;
2. lever les `services[]` (sidecars) sur un **réseau isolé par run** ;
3. injecter les conn-strings (`DATABASE_URL`, `REDIS_URL`…) ;
4. exécuter les commandes du projet (`make test`, migrations, seed).

### 4.3 Golden images pré-seedées (décision #10)

Le coût du seed (migrations + données de fixtures) est résolu **au build/CI** : on construit des **golden images pré-seedées** (DB sidecar déjà migrée+seedée figée en image), puis **clone copy-on-write par run**. Pas de re-seed par run. C'est le « snapshot » des vendeurs (Daytona/E2B/CMA).

### 4.4 Invariant de portabilité (décision #11)

L'`environment` doit rester **K8s-Pod-exprimable** : `services[]` = **sidecars-in-Pod** (pas DinD). Les testcontainers (utilisés par le backend hopeitworks lui-même via testcontainers-go) deviennent, sur K8s, du **KubeDock** (API Docker → vrais Pods), jamais DinD/sysbox. L'isolation est une **capability d'adapter** de substrat, pas une hypothèse de domaine.

---

## 5. Substrat & isolation

### 5.1 Par contexte, pas global (décisions #12, #13)

| Contexte de déploiement | Substrat / isolation | Pourquoi |
|---|---|---|
| Hôte/pool **avec KVM** (VM, bare-metal, OpenShift Virt + KVM device-plugin) | **microsandbox** (libkrun microVM) — **défaut** | vrai kernel → testcontainers/DinD marchent dedans, perf quasi-native, meilleure fidélité coding |
| K8s/OpenShift **durci sans KVM** | adapter K8s `runtimeClassName: gvisor` | kernel userspace, pas de KVM, OpenShift-viable ; coût perf/compat syscall ; testcontainers → KubeDock + sidecar Postgres |
| Cloud Anthropic | **CMA-cloud** (option) | contourne l'infra locale si Anthropic acceptable |

**gVisor ≠ microsandbox** : gVisor = RuntimeClass d'isolation (sans KVM) ; microsandbox = isolation microVM + SDK de gestion. Comparables uniquement sur l'axe isolation. microVM = isolation + fidélité supérieures mais exige KVM ; gVisor = fallback sans-KVM.

### 5.2 L'isolation comme capability d'adapter

Chaque adapter de substrat déclare son niveau d'isolation et ce qu'il sait faire tourner (DinD/testcontainers natifs vs KubeDock requis). Le domaine ne décide jamais « microVM » ou « DinD » : il déclare un `environment` portable, l'adapter le réalise selon son substrat.

---

## Matrice capacité × runtime

| Capacité | ClaudeCodeRuntime | OpenCodeRuntime | CMARuntime |
|---|---|---|---|
| `skill` (SKILL.md) | natif (`.claude/skills/`) | mappé (instructions/prompt) | mappé (managed config) |
| `mcp_server` HTTP | natif (`.mcp.json`) | natif (config MCP) | natif (MCP managed) |
| `mcp_server` stdio | natif (in-image) | natif (in-image) | warn+skip (selon support) |
| `tool_policy` (allow/deny) | natif (`--allowedTools`) | partiel → warn+skip si non mappable | managed policy |
| `lsp` (in-image) | via stack | via stack | selon image managed |
| system prompt | natif | natif | natif |

> Règle invariante (décision #3) : toute case « non supporté » = **warn + skip**, jamais block.

## Matrice environment × substrat

| Besoin environment | microsandbox (KVM) | K8s + gVisor (sans KVM) | CMA-cloud |
|---|---|---|---|
| sidecars (postgres/redis…) | sidecars dans la VM | sidecars-in-Pod | managed (selon offre) |
| testcontainers (testcontainers-go) | natif (vrai kernel) | **KubeDock** + sidecar Postgres | selon offre |
| DinD / build d'images | natif | non (KubeDock) | selon offre |
| golden image pré-seedée (CoW) | snapshot VM | image + initContainer/PVC | snapshot managed |
| egress default-deny | netfilter VM | NetworkPolicy | policy managed |
| K8s-Pod-exprimable | oui (adaptable) | oui (natif) | n/a (externe) |

---

## Roadmap par phases

> Ordre = du port stable vers les substrats, pour pouvoir livrer sans casser l'existant. Effort : S < ½j, M ~1–3j, L > 3j.

### P0 — Refactor du port + abstractions de domaine — **L**
**Objectif** : généraliser `ContainerManager` en port `AgentRuntime` orienté exécution d'agent ; introduire les abstractions `capability` / `credential` / `stack` / `environment` (modèles + interfaces, sans impl. complète). Tuer le couplage `strings.Contains(image,"hopeitworks/agent-")`.
**Fichiers** :
- `backend/internal/domain/port/agent_runtime.go` (nouveau), `backend/internal/domain/port/container_manager.go` (déclasser en interne adapter)
- `backend/internal/adapter/action/agent_run.go` (`isCallbackMode:532`, `createContainer`, `buildEnv` → dispatch par `RuntimeKind`)
- `backend/internal/domain/model/agent.go` (remplacer `Image` par `RuntimeKind` + `StackRef`), nouveaux `model/capability.go`, `model/stack.go`, `model/environment.go`
- `backend/internal/adapter/docker/container_manager.go` (devient impl. de l'adapter substrat Docker)
- `backend/cmd/api/wire.go` (réviser l'alias `NewAgentRuntime`)

### P1 — Modèle de capacités + injection fetch-at-startup — **L**
**Objectif** : tables `capabilities` / `agent_capabilities` + credentials chiffrés ; endpoint bundle authentifié container-token ; `agent-runtime` bascule de l'env vers le fetch.
**Fichiers** :
- `backend/migrations/0000XX_capabilities.up.sql` / `.down.sql`, `backend/queries/capabilities.sql`
- `backend/internal/api/handler/` (endpoint `GET /agent-runtime/bundle`), `backend/internal/domain/port/container_token_store.go` (réutilisé)
- `agent-runtime/internal/config/config.go` (lire bundle au lieu de l'env), `agent-runtime/internal/callback/client.go` (ajout fetch bundle), `agent-runtime/internal/runner/runner.go` (matérialiser skills/.mcp.json/prompt)
- `api/openapi.yaml` (contrat bundle)
- `agent-runtime/internal/provider/{claude.go,opencode.go}` (consommer tool_policy/.mcp.json générés)

### P2 — Catalogue de stacks + Environment/sidecars + golden images — **L**
**Objectif** : entité `stack` pinnée multi-arch ; `environment` par projet (dérivé devcontainer/compose/Makefile) ; sidecars sur réseau isolé + conn-strings ; golden images pré-seedées en CI.
**Fichiers** :
- `backend/migrations/` (`stacks`, `environments`), `backend/queries/`
- `agent-images/Makefile`, `agent-images/stacks/*`, `agent-images/base/Dockerfile` (pinning, multi-arch, dédup `go-node`, python enrichie, LSP par stack)
- `.github/workflows/agent-images.yml` (build multi-arch + golden images seedées)
- `backend/internal/adapter/action/agent_run.go` (composition image + lever sidecars + injecter `DATABASE_URL` + exécuter `make test`/migrate/seed)

### P3 — Premier adapter de substrat : microsandbox — **L** (décision #14, build first)
**Objectif** : adapter substrat microsandbox (libkrun) derrière `AgentRuntime` ; isolation microVM ; testcontainers/DinD natifs ; clone CoW des golden images.
**Fichiers** :
- `backend/internal/adapter/microsandbox/` (nouveau, impl. `AgentRuntime`)
- `backend/cmd/api/wire.go` (sélection d'adapter par config), `backend/internal/config/`

### P4 — Adapter K8s/gVisor + KubeDock — **L**
**Objectif** : adapter substrat K8s (`runtimeClassName: gvisor`), sidecars-in-Pod, KubeDock pour testcontainers, NetworkPolicy egress default-deny. Déclenché par premier client OpenShift.
**Fichiers** :
- `backend/internal/adapter/k8s/` (nouveau, impl. `AgentRuntime`), manifests/Helm dans `deploy/`
- `backend/internal/config/` (substrat = K8s)

### Px — CMA optionnel — **M**
**Objectif** : `CMARuntime` wrappant Anthropic Managed Agents pour les déploiements où l'infra locale n'est pas souhaitable. Strictement optionnel.
**Fichiers** : `backend/internal/adapter/cma/` (nouveau), `backend/internal/config/`.

---

## Décisions ouvertes & risques

**Décision prise (2026-06-22)** : 1er adapter de substrat = **microsandbox** (host Docker actuel, KVM disponible, meilleure fidélité coding). P3 cible donc microsandbox ; P4 (K8s/gVisor) sera priorisé si un client OpenShift sans KVM arrive d'abord. Le port + l'invariant Pod-exprimable rendent le choix réversible. Plus de décision ouverte bloquante.

**Risques** :

| Risque | Impact | Mitigation |
|---|---|---|
| Surface d'attaque MCP (exfiltration, prompt injection via outils) | élevé | catalogue curé, réseau interne, RBAC tool-level, egress default-deny, audit de chaque appel (§3.2) |
| Dépendance à la beta CMA (si utilisé) | moyen | CMA = adapter optionnel ; jamais une dépendance de domaine (décisions #1, #12) ; bascule d'adapter si Anthropic coupe l'accès |
| Coût de maintenance multi-adapters (Claude/OpenCode/CMA × Docker/microsandbox/K8s) | moyen | matrice capacité×runtime + warn+skip pour borner le test ; construire un adapter à la fois (P3 puis P4) |
| KVM indisponible sur clusters managés (EKS/GKE/AKS, OpenShift sans Virt) | moyen | fallback gVisor (P4) ; KVM = condition de microsandbox, pas du produit |
| Golden images : dérive seed ↔ schéma réel | faible | rebuild golden au CI sur changement de migration ; clone CoW garantit l'idempotence par run |

---

## Relation aux docs existants

Ce plan **supersede** les trois docs **sur le substrat et le runtime** (comment/où s'exécute un agent, comment les capacités sont injectées et sécurisées). Il vient **par-dessus** la couche capacités/comportement, qui reste valide :

| Doc | Ce que ce plan supersede | Ce que le doc garde (par-dessus cette couche) |
|---|---|---|
| `docs/agent-engineering-research.md` | « security posture = container permissif » et l'injection par env → remplacés par substrat isolé + fetch-at-startup + RBAC MCP | la **boucle de vérification fermée**, la **mémoire pgvector** cross-run, le constat « pas de LSP/MCP/skills » (que ce plan **réalise**) |
| `docs/agent-images-tools-skills-plan.md` | string libre `agents.image`, `buildClaudeMD` en switch, MCP non spécifié → remplacés par catalogue de stacks, capacités versionnées, MCP-services | les **correctifs d'images** (pinning, dédup `go-node`, multi-arch, python enrichie), les **Agent Skills** et la **tool-policy par rôle** (intégrés ici comme capacités/stacks), le **LSP par stack** |
| `docs/agent-orchestration-plan.md` | rien sur le substrat (hors scope du doc) | **intégralement valide** : boucle **review→fix**, **handoff typé** (`review_findings`), **payload feedback HITL**, router déterministe en Go. Cette orchestration tourne **au-dessus** des runtimes/capacités définis ici |

**Through-line** : ce plan fournit le **runtime + les capacités sécurisées + les environments portables** ; la boucle review→fix, la mémoire pgvector et l'orchestration de pipeline restent la couche **comportementale** par-dessus.
