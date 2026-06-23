# Capacités de télémétrie des runtimes — recherche (juin 2026)

> Recherche web (workflow `runtime-telemetry-capabilities`, 5 agents, sourcée) sur ce que les **versions récentes** de Claude Code, OpenCode et concurrents intégrables savent émettre, pour cadrer le **contrat de télémétrie sortante** du port `AgentRuntime` (débloque INC 4b / gap G1 — voir `board-pipeline-agents-model.md` §9 et `board-pipeline-v1-build.md` §INC 4).
>
> 5 signaux visés (= probes INC 4b) : **incremental_cost** (cost mid-run), **tool_call_events** (loop detection), **files_touched** (blast-radius), **resource_metrics** (resource-pressure), **heartbeat_progress** (liveness).

## 1. Matrice de capacités

✅ natif & streamé live · 🟡 partiel / per-step-pas-per-timer / synthétisable · ❌ pas fourni par l'outil

| Runtime (version mi-2026) | incremental_cost | tool_call_events | files_touched | resource_metrics | heartbeat |
|---|:---:|:---:|:---:|:---:|:---:|
| **Claude Code** CLI 2.1.186 + Agent SDK | ✅ | ✅ | ✅ | ❌ | 🟡 |
| **opencode** v1.17.9 (serve/run) | ✅¹ | ✅ | ✅ | ❌ | 🟡 |
| **Codex CLI** v0.142 | ✅² | ✅ | ✅ | ❌ | 🟡 |
| **Goose** v1.38 | 🟡 (fin de run) | ✅ | 🟡 | ❌ | 🟡 |
| **Aider** v0.86 | 🟡 (texte) | ❌ | 🟡 (git diff) | ❌ | ❌ |
| **Cursor CLI** (rolling) | 🟡 (final) | ✅ | ✅ | ❌ | 🟡 |
| **Amp Neo** | ❌ | ✅ | 🟡 | ❌ | 🟡 |
| **Cline CLI** 2.0 | 🟡 (résumé) | ✅ | 🟡 | ❌ | 🟡 |
| **Crush** | ✅ (step_finish) | 🟡 | 🟡 | ❌ | 🟡 |
| **Continue CLI** | ? | ✅ | ✅ | ❌ | 🟡 |

¹ per-step `step_finish`, mais bug #26855 (flush final) → lire le cost final via session API, pas l'EOF stdout. ² per-turn `turn.completed` + métrique OTel `codex.turn.token_usage`.

> **Invariant dur : `resource_metrics` = ❌ partout.** Aucun agent de code n'émet CPU/mem/IO. **C'est structurellement une affaire de substrat** (cgroups v2 / cAdvisor / Docker stats / metrics-server), pas de l'outil ni de l'adapter.

## 2. LCD (commun) vs adapter-specific

- **`tool_call_events`** — le signal le plus fort, natif et structuré partout sauf Aider. → **port commun.**
- **`files_touched`** — **dérivé des tool-calls** (un edit = un tool-call). Événements dédiés là où natif (Claude `PostToolUse` file_path, opencode `file.edited`, Codex `file changes`, Cursor `writeToolCall`), projeté depuis les tool-calls ailleurs. → **vue dérivée, pas un champ de contrat séparé** (maximise la couverture).
- **`incremental_cost`** — **scinder** : un `UsageDelta` per-step que le tier fort remplit live (Claude `api_request`, opencode/Crush `step_finish`, Codex `turn.completed`) + un `UsageTotal` final que **tout le monde** peut émettre. Jamais exiger un cost **périodique sur timer** : personne ne l'émet.
- **`heartbeat`** — personne n'a de keepalive content-free sur timer fixe. → **synthétisé par la couche runtime** (watchdog sur le timestamp du dernier event). C'est déjà ce que fait INC 4a (`log_silence`).
- **`resource_metrics`** — **substrat**, jamais l'adapter (cgroups v2 / cAdvisor / metrics-server).
- **Cost en devise** = estimation client-side (Claude `total_cost_usd`, opencode) → **les tokens sont la monnaie portable**, le $ reste best-effort/flaggé estimate.

## 3. Contrat de télémétrie `AgentRuntime` recommandé

Un petit jeu d'**événements normalisés** (l'adapter traduit le flux natif de chaque outil) + 2 signaux **runtime-owned** :

```
Adapter-normalized (poussés live) :
  ToolCallEvent  { ts, tool_use_id, tool_name, phase:started|completed, args?, result_summary?, success?, duration_ms? }
        → loop-detection. Universel sauf Aider.
  FileTouchEvent { ts, path, op:read|write|edit, tool_use_id }
        → blast-radius. DÉRIVÉ de ToolCallEvent (edit/write/read).
  UsageDelta     { ts, tokens:{input,output,cache_read,cache_creation,reasoning?}, cost_usd?, model, step_id }
        → cost-mid-run. Per-step où supporté. Tokens obligatoires, cost optionnel+flag estimate.
  UsageTotal     { tokens{...}, cost_usd?, model_breakdown }
        → réconciliation finale. EVERY adapter peut l'émettre.

Runtime-owned (substrat / synthèse, PAS l'outil) :
  ResourceSample { ts, cpu_pct, mem_bytes, io_read/write_bytes }   → resource-pressure. cgroups v2 / cAdvisor.
  Heartbeat      { ts, last_event_ts, state:alive|stalled }        → liveness. Watchdog sur gap d'events (= INC 4a).
```

Règles : contrat **event-normalized**, transport au choix de l'adapter (Claude stream-json + OTLP ; opencode SSE `/event` ; Codex/Goose `--json` ; Cursor/Cline/Amp NDJSON). `files_touched` et `heartbeat` ne sont **pas** des champs d'adapter (l'un dérivé, l'autre synthétisé) — c'est ça qui garde le contrat agnostique. **Aider = le plancher** (seulement `UsageTotal` scrappé + git-diff) → adapter dégradé assumé, on ne baisse pas le contrat à son niveau.

## 4. Implications pour INC 4b (le déblocage réel)

La recherche **réduit** le périmètre « bloqué » que j'avais annoncé :

- **resource-pressure** : **pas bloqué sur l'agent du tout** — c'est du **cgroups au niveau substrat**. Buildable indépendamment de la refonte runtime. (Aligné sur l'invariant projet : runtime = isolation/ressources.)
- **Pour nos 2 adapters réels (claude_code, opencode)** : tool-calls, files (dérivés) et **cost incrémental sont déjà dans les flux émis aujourd'hui** — Claude via `stream-json` / OTEL `api_request` (per-request) ; opencode via SSE `step_finish` (per-step). Le binaire lit déjà ces flux → c'est **« forwarder plus de l'existant »**, pas une nouvelle capacité.
- **heartbeat** : déjà couvert par INC 4a (`log_silence`).

> **Donc le vrai bloqueur d'INC 4b n'est pas « le runtime ne peut pas émettre »** — les outils émettent déjà. C'est : (a) **normaliser** ces 4 events sur le port `AgentRuntime`, (b) faire **forwarder** au binaire agent les tool-calls + UsageDelta (petit), (c) **brancher cgroups** pour le resource. Aucune dépendance à une nouvelle capacité des outils. La seule raison d'attendre = le binaire runtime est en cours de refonte → **définir le contrat maintenant** et implémenter le forwarding dans cette refonte, pas après.

## 5. Capacités 2026 à exploiter (nos 2 adapters)

- **Claude Code** : `stream-json` + `--include-partial-messages` (deltas), OTEL `claude_code.api_request` (cost/tokens **par requête**), hook `PostToolUse` (file_path), event OTEL `tool_result` (corrélé `tool_use_id`), `--bare` (runs déterministes CI), hint stream-stall (2.1.185). **Stratégie d'émission** : stream-json stdout **+** OTLP vers sidecar **+** hook PostToolUse pour pousser files_touched + heartbeat synthétique.
- **opencode** : `opencode serve` (HTTP + SSE, **le meilleur modèle d'embedding** : lancer une fois, piloter via OpenAPI typé), 80+ events bus, hook plugin `event` (catch-all) = **point de forwarding unique**, SDKs TS/Python/Go. **Lire le cost final via session API** (évite le bug #26855).
- **Transverse** : **MCP universel** (sauf Aider) = injection de capacités (pas un canal de télémétrie). Propagation **W3C `traceparent`** (Claude, opencode, Codex) = les spans agent s'imbriquent sous le span orchestrateur gratuitement.

## 6. À vérifier / faible confiance (les versions bougent vite)

- **Lot « Concurrents B » (Cline/Crush/Continue/Cursor/Amp) = confiance moyenne** → provisoire.
- **opencode** : pinner une version connue-bonne (bug #26855 flush cost final ; #27966 régression SSE `/event` ~1.14.42+). **OTEL natif NON confirmé** → plugins communautaires uniquement.
- **Claude** : `total_cost_usd` = estimation client-side (vraie facturation via Usage/Cost API) ; **usage SDK sur plan abonnement = crédit séparé depuis 2026-06-15** → préférer auth API-key/Bedrock/Vertex en container ; export de traces encore **beta**.
- **Continue CLI = EOL** (racheté par Cursor, repo read-only) → pas un choix d'avenir.
- **Cursor CLI** : non-versionné/rolling ; hooks file-edit **IDE-only** en CLI (utiliser l'event write tool-call). **Aider** : API Python non-officielle/instable → ne pas builder d'adapter dessus.

## Sources principales
Claude Code : code.claude.com/docs (headless, agent-sdk/observability, monitoring-usage), anthropics/claude-code CHANGELOG. opencode : opencode.ai/docs (server, plugins, sdk), sst/opencode issues #26855/#14246/#27966. Codex : developers.openai.com/codex. Goose : github.com/block/goose (cf. caveat org `aaif-goose` vs `block`). Cursor : cursor.com/docs/cli. Cline : docs.cline.bot. (Liste complète des URLs dans le résultat workflow.)
