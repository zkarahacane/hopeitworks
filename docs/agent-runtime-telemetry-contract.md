# Contrat de télémétrie sortante `AgentRuntime` — proposition

> **Statut : PROPOSITION, à coordonner avec le chantier runtime** (`agent-runtime-capabilities-plan.md`, édité en parallèle — **ne pas modifier leur fichier**). Cet artefact lève le **gap G1** identifié dans `board-pipeline-agents-model.md` §9 et débloque **INC 4b** (`board-pipeline-v1-build.md`). Il est cadré par la recherche `runtime-telemetry-capabilities.md` (ce que les outils savent émettre, juin 2026).
>
> **Principe.** Le runtime/adapter **émet** un petit jeu d'événements **normalisés** ; un ingesteur board les traduit en `events` (Postgres, append-only) ; l'évaluateur de policy (probes/guards) **réagit**. Le domaine ne contient que de la *policy* (seuils + actions) ; l'émission et le substrat sont *runtime/adapter*. Jamais d'infra dans le domaine.

## 1. Frontière (qui produit quoi)

| Émis par | Événements | Nature |
|---|---|---|
| **Adapter** (depuis le flux natif de l'outil) | `ToolCallEvent`, `UsageDelta`, `UsageTotal` | normalisés, poussés live |
| **Board (ingesteur)** | `FileTouchEvent` | **dérivé** de `ToolCallEvent` |
| **Substrat / orchestrateur** | `ResourceSample` | cgroups v2 / cAdvisor / metrics-server |
| **Board (watchdog)** | `Heartbeat` | **synthétisé** sur le gap d'events |

> Ce qui rend le contrat *agnostique* : `FileTouchEvent` et `Heartbeat` ne sont **pas** des champs que l'adapter doit fournir (dérivé / synthétisé). Et `resource_metrics` n'est demandé à **aucun** outil — c'est du substrat (confirmé : ❌ chez les 10 runtimes étudiés).

## 2. Les événements (wire shape)

Champs communs à tout event : `run_id`, `step_id`, `ts` (RFC3339, source-side).

```go
// — Adapter-normalized (poussés depuis le container pendant le run) —

ToolCallEvent {           // → probe loop-detection (INC 4b)
  tool_use_id  string     // corrèle started/completed, dédup les appels parallèles
  tool_name    string
  phase        string     // "started" | "completed"
  args         any?       // optionnel (peut être tronqué/omis par policy de contenu)
  result_summary string?  // optionnel
  success      bool?      // sur "completed"
  duration_ms  int?       // sur "completed"
}

UsageDelta {              // → probe cost-mid-run (INC 4b). Émis per-step où dispo.
  tokens { input, output, cache_read, cache_creation, reasoning? }  // OBLIGATOIRE
  cost_usd     float?     // optionnel + flag estimate (estimation client-side)
  model        string
  step_id      string
}

UsageTotal {              // réconciliation finale. TOUT adapter peut l'émettre.
  tokens { ... }
  cost_usd     float?     // estimate
  model_breakdown map<string, {tokens, cost_usd?}>
}

// — Runtime-owned (PAS depuis l'outil) —

ResourceSample {          // → probe resource-pressure (INC 4b). Échantillonné par le substrat.
  cpu_pct      float
  mem_bytes    int
  io_read_bytes, io_write_bytes int
}

Heartbeat {               // → probe liveness (INC 4b). Synthétisé par le watchdog.
  last_event_ts string
  state         string    // "alive" | "stalled"
}

// — Dérivé board-side, PAS un champ d'adapter —

FileTouchEvent {          // → probe blast-radius (INC 4b). Projeté depuis ToolCallEvent (edit/write/read).
  path         string
  op           string     // "read" | "write" | "edit"
  tool_use_id  string
}
```

Règles : **tokens = monnaie portable**, `cost_usd` toujours optionnel + flaggé estimate (Claude `total_cost_usd` et opencode sont des estimations client-side). Le contrat est *event-normalized* ; le **transport est au choix de l'adapter**.

## 3. Transport — extension du protocole callback existant

Le mécanisme existe déjà : `agent_callback_handler.go` reçoit aujourd'hui `…/logs`, `…/cost`, `…/status` (POST depuis le container) et les écrit dans `events` (`log.emitted`) / `cost_records`. On **étend** ce canal, on n'en invente pas un autre :

| Endpoint inbound (`/internal/agent/callback/runs/{runId}/steps/{stepId}/…`) | Porte | Statut |
|---|---|---|
| `logs` | (existant) liveness de secours | ✅ existe |
| `cost` | → `UsageDelta` (per-step) + un POST final `UsageTotal` | ⚙️ existe, à appeler **plus souvent** + flag final |
| `tools` | → `ToolCallEvent` | 🆕 nouvel endpoint |
| `status` | terminal | ✅ existe |

- `FileTouchEvent` : **dérivé dans l'ingesteur** quand un `ToolCallEvent` porte un outil edit/write/read — pas d'endpoint.
- `ResourceSample` : échantillonné par l'orchestrateur/watchdog (lecture cgroups du container), pas un callback agent.
- `Heartbeat` : synthétisé board-side (watchdog sur `max(last_event_ts)` tous events confondus) — généralise le `log_silence` déjà livré par INC 4a.

Chaque callback inbound → écrit dans `events` (le pont « callback → events » nommé en §9) → SSE + évaluateur de policy.

## 4. Mapping par adapter (nos 2 réels + plancher)

| Event | **Claude Code** (2.1.186) | **opencode** (1.17.9) | Plancher / autres |
|---|---|---|---|
| `ToolCallEvent` | `stream-json` blocs `tool_use`/`tool_result` ; hook `PostToolUse` ; OTEL `claude_code.tool_result` (`tool_use_id`) | SSE `message.part.updated` (parts tool) ; hooks plugin `tool.execute.before/after` | Codex `exec --json` item.* ; Cursor `tool_call` ; **Aider ❌** |
| `UsageDelta` | OTEL `claude_code.api_request` (cost/tokens **par requête**) ou usage per-message (SDK/stream-json) | part `step_finish` (tokens+cost) via SSE/`run --format json` | Codex `turn.completed` ; Crush `step_finish` ; sinon fin de run |
| `UsageTotal` | message `result` final (`total_cost_usd` + `modelUsage`) | **session API** (évite le flush-race #26855) | tous |
| `FileTouchEvent` *(dérivé)* | depuis tool `Edit`/`Write`/`Read` (+ hook `file_path`, `lines_of_code.count`) | events natifs `file.edited`/`file.watcher.updated` ou tool parts | depuis tool-calls partout |
| `ResourceSample` *(substrat)* | cgroups v2 / cAdvisor / metrics-server | idem | idem |
| `Heartbeat` *(synthèse)* | watchdog + hint stream-stall (2.1.185) | watchdog + `session.status`/`idle` + `GET /global/health` | watchdog |

**Stratégie d'émission recommandée par adapter :**
- **Claude Code** : `--output-format stream-json` (tool/text/cost live sur stdout) **+** hook `PostToolUse` (POST `…/tools` + file_path) **+** appel `…/cost` par `api_request`. Optionnel : OTLP vers sidecar collector en dual-path.
- **opencode** : `opencode serve` (HTTP+SSE) ; un **hook plugin `event` catch-all** = point de forwarding unique vers les callbacks ; lire le cost final via session API.
- **Aider = plancher assumé** : seulement `UsageTotal` (scrappé) + `FileTouchEvent` via git-diff → adapter dégradé, on ne baisse pas le contrat à son niveau.

## 5. Câblage event → probe INC 4b

| Probe INC 4b | Consomme | Bloqué ? |
|---|---|---|
| loop-detection | `ToolCallEvent` (répétition même tool/args) | 🟢 dispo pour nos 2 adapters (stream-json / SSE) |
| blast-radius | `FileTouchEvent` (hors `target_files`/`scope`) | 🟢 dérivé de ToolCallEvent |
| cost mid-run | `UsageDelta` (cumul > ceiling) | 🟢 per-request (Claude) / per-step (opencode) |
| resource-pressure | `ResourceSample` (> seuils) | 🟢 **indépendant de l'agent** (cgroups) |
| liveness | `Heartbeat` (`state=stalled`) | ✅ déjà couvert (INC 4a `log_silence`, à généraliser) |

→ **Aucune probe n'attend une capacité manquante des outils.** Le travail est : (a) le nouvel endpoint `…/tools` + dérivation `FileTouchEvent` + `ResourceSample` (cgroups) côté board ; (b) faire **forwarder** au binaire les tool-calls + cost per-step (petit, il parse déjà le stream) ; (c) formaliser le contrat sur le port.

## 6. Surface du port `AgentRuntime` (à acter avec le chantier runtime)

Le port reste `Launch/Wait/Stop/Provision/SupportedCapabilities`. **Proposition** : garder le **protocole callback HTTP comme contrat de télémétrie** (mécanisme prouvé), et que le port **garantisse** qu'un adapter, run vivant, POST les events normalisés ci-dessus. Deux variantes à trancher ensemble :
- **(A, recommandé)** le contrat = les endpoints callback étendus (`…/tools`, `…/cost` enrichi) ; le port documente la garantie d'émission. Changement minimal.
- **(B)** ajouter une méthode `Emit`/stream typée au port. Plus pur, plus de surface.

`SupportedCapabilities` devrait déclarer **quels events un adapter sait émettre** (ex. Aider = `{UsageTotal, FileTouch(git)}` seulement) → les probes activables se dérivent des capacités de l'adapter.

## 7. Points de coordination / questions ouvertes

- **G2 — contrat `Stop()`** : doit être **gracieux/non-corrompant** (pas d'écriture partielle) pour que le halt-gate laisse un run reprenable. À pin avec le runtime.
- **Version pinning** : opencode bugs #26855 (flush cost final) / #27966 (régression SSE) → pinner une version connue-bonne ; OTEL natif opencode **non confirmé** (plugins communautaires).
- **Auth en container** : Claude `total_cost_usd` = estimation (vraie facturation via Usage/Cost API) ; usage SDK plan abonnement = **crédit séparé depuis 2026-06-15** → auth API-key/Bedrock/Vertex, pas le login claude.ai.
- **Dual-path OTLP optionnel** : Claude/Codex/Goose ont un OTLP natif (W3C `traceparent` → spans imbriqués sous l'orchestrateur). Utile en *complément* d'observabilité, mais **le callback reste le contrat canonique** (tous les outils n'ont pas d'OTEL → ne pas en dépendre pour les probes).

## 8. Découpage d'implémentation (INC 4b)

| Côté | Travail |
|---|---|
| **Binaire `agent-runtime/`** (à faire **pendant** la refonte runtime) | forwarder `ToolCallEvent` (déjà dans le stream-json parsé) + `UsageDelta` per-step (appeler `…/cost` plus souvent) + `UsageTotal` final |
| **Substrat / orchestrateur** (board, indépendant du binaire) | `ResourceSample` depuis cgroups ; généraliser le watchdog `Heartbeat` à tous les events |
| **Ingesteur board** | endpoint `…/tools` → `events` ; dériver `FileTouchEvent` ; alimenter les probes/guards (INC 4a en place) |
| **Port** | acter variante A/B + `SupportedCapabilities` par event |

> **Ordonnancement** : (b) substrat + ingesteur = livrables board-side **maintenant** (INC 4a les a amorcés). (a) forwarding binaire = à intégrer dans la refonte runtime pour ne pas builder deux fois. La spec ci-dessus est l'artefact à poser dans la discussion runtime.
