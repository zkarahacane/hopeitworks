# Board ↔ Pipeline — Build plan v1 (Stage first-class)

> **But.** Décomposition **directement exécutable par Claude Code** (pas des tickets) pour poser le *Stage* comme concept de première classe. Modèle de référence : `board-pipeline-agents-model.md`. Chaque incrément est **indépendamment shippable**, ordonné par dépendance, avec ancres code exactes + critères de vérif. Chaque bloc « ▶ CC » est un prompt prêt à lancer (agent direct ou worktree).
>
> **État vérifié (juin 2026).** Beaucoup existe déjà (runs, steps, executor, HITL gates, board, SSE). Le travail n'est PAS un rebuild — c'est **rendre l'identité de stage durable de bout en bout**, puis **dériver les colonnes des stages**, puis **enforcer les politiques de transition**.

---

## Le constat qui commande tout

| Cible | Existant (ancre) | Manque |
|---|---|---|
| Stage = unité durable | `PipelineGroup{ID,Name,Steps}` (`model/pipeline_config.go:40`) — groups existent, **sans politique** | champ `transition` |
| Identité stage sur l'exécution | `run_service.go:315` fait `parsed.FlatSteps()` → **les steps sont aplatis, le groupe est perdu** ; `RunStep` (`model/run.go:70`) et table `run_steps` (mig `000009`) **n'ont aucun `stage_id`/`group_id`** | colonne `stage_id`/`stage_name` sur `run_steps` |
| Position de la story | `stories.status` (backlog/running/done/failed, `model/story.go:10`) — **pas de `current_stage`** | colonne `current_stage` |
| Events de frontière | `run.started`/`step.started`/`step.completed` (`pipeline_executor.go:113/208/319`) — **aucun `stage.*`** | events `stage.entered`/`stage.exited` |
| Colonnes board | `boardColumn()` **hardcodé** lifecycle (`stores/stories.ts:58`) + `COLUMNS` statique (`KanbanBoard.vue:52`) | dérivation depuis les stages du pipeline |

**Migration libre** : `000036` — develop a déjà `000034`/`000035` (catalogue de stacks runtime, mergés PR #254/#255). Reconfirmer `ls backend/migrations/` au build (le runtime a des branches en cours : `feat/stacks-catalog-config-seed`).

---

## Échelle d'incréments

```
INC 1  Identité de stage end-to-end (backend)         ← keystone, à faire en premier
INC 2  Colonnes board dérivées des stages (frontend)  ← dépend de INC 1
INC 3  Politiques de transition: manual + Go, gate     ← dépend de INC 1 (+2 pour l'UI)
INC 4  Guards/probes                                   ← BLOQUÉ (sauf wallclock) sur gap runtime G1
```

v1 démontrable = **INC 1 + INC 2** (« les colonnes du board reflètent enfin le vrai pipeline, la carte est dans son stage courant »). v1.1 = **INC 3** (interactif). v2 = **INC 4**.

---

## INC 1 — Identité de stage end-to-end (backend)

**Goal.** Chaque `run_step` porte son stage ; la story expose `current_stage` ; l'executor émet `stage.entered`/`stage.exited` et avance `current_stage`. Politique `transition` ajoutée au modèle (défaut `auto` = comportement actuel), **pas encore enforced** (c'est INC 3).

**Fichiers & changements (ancres exactes) :**

1. **Migration `000036_add_stage_identity.up.sql`** (+ `.down.sql`) :
   - `ALTER TABLE run_steps ADD COLUMN stage_id VARCHAR(255); ADD COLUMN stage_name VARCHAR(255);`
   - `ALTER TABLE stories ADD COLUMN current_stage VARCHAR(255);` (nullable)
   - Backfill optionnel : `run_steps.stage_id = 'default'` pour les lignes existantes.

2. **`backend/internal/domain/model/pipeline_config.go`** :
   - `PipelineGroup` (ligne 40) → ajouter `Transition string` (`yaml:"transition,omitempty" json:"transition,omitempty"`), valeurs `auto|manual|gate`, défaut `auto` à la lecture.
   - Ajouter un helper `FlatStepsWithStage()` (ou étendre `FlatSteps`) qui retourne, par step, le `(GroupID, GroupName)` d'origine — **arrêter de jeter l'identité de groupe**.

3. **`backend/internal/domain/model/run.go`** : `RunStep` (ligne 70) → ajouter `StageID string`, `StageName string`.

4. **`backend/internal/domain/service/run_service.go`** (`LaunchRun`, lignes 310–414) : remplacer `flatSteps := parsed.FlatSteps()` par la variante avec stage ; au `CreateRunStep` (ligne 398) stamper `StageID`/`StageName`.

5. **sqlc** (`backend/queries/runs.sql` + regen) : `CreateRunStep` insère stage_id/stage_name ; `ListRunStepsByRun` les sélectionne ; nouvelle query `UpdateStoryCurrentStage`.

6. **`backend/internal/domain/service/pipeline_executor.go`** (`ExecuteRun`, loop lignes 96–170) : détecter le changement de stage entre `steps[i-1]` et `steps[i]` → `publishEvent(... "stage", ..., "entered", {stage_id, stage_name, story_id})` + `UpdateStoryCurrentStage` ; émettre `stage.exited` pour le précédent ; à la complétion du run, `current_stage = "__done__"` (sentinelle) ou null. Réutiliser `publishEvent` (ligne 447).

7. **`api/openapi.yaml`** : `PipelineGroup` (ligne 3170) +`transition` ; `RunStep` (3288) +`stage_id`/`stage_name` ; `Story` (2699) +`current_stage`. Regénérer (`oapi-codegen` backend + `schema.d.ts` front).

**Hors-scope INC 1.** Aucune enforcement de `manual`/`gate` (reste `auto`). Pas de probes. Pas de touche front.

**Verify.** Pipeline à ≥2 groups nommés → lancer un run → `run_steps` portent `stage_id/name` ; `events` contient `stage.entered`/`stage.exited` aux frontières ; `stories.current_stage` avance ; `GET /stories` expose `current_stage`. `go build ./... && go vet ./... && go test -short ./...` verts.

**▶ CC :** *« Implémente INC 1 du build plan `docs/board-pipeline-v1-build.md` : identité de stage end-to-end côté backend Go. Migration 000034 (reconfirme le numéro libre), champs modèle, FlatStepsWithStage, stamping à LaunchRun, events stage.entered/exited + current_stage dans ExecuteRun, OpenAPI + regen. Respecte l'archi hexagonale existante. Branch from develop. build/vet/test -short verts. »*

---

## INC 2 — Colonnes board dérivées des stages (frontend)

**Goal.** Le Kanban affiche les **stages du pipeline projet** comme colonnes (au lieu des 5 hardcodées) ; chaque carte est placée par `current_stage` ; un toggle macro/détail garde la vue lifecycle.

**Fichiers & changements :**

1. **`frontend/src/api/schema.d.ts`** : régénéré depuis l'OpenAPI d'INC 1 (apporte `current_stage`, `transition`).
2. **`frontend/src/stores/stories.ts`** : `boardColumn()` (ligne 58) → nouvelle `stageColumn(story)` basée sur `story.current_stage` ; **garder** `boardColumn()` pour la vue lifecycle (toggle). Ajouter `current_stage` au type `Story` (ligne 30).
3. **`frontend/src/stores/pipelineConfig.ts`** : exposer `stages` (= `groups` avec `id/name/transition`) — déjà à 90% (`groups` computed).
4. **`frontend/src/features/board/KanbanBoard.vue`** : `COLUMNS` statique (ligne 52) → **colonnes dynamiques** = stages du pipeline + colonnes terminales `Done`/`Failed`. Carte sans run → `Backlog`/entrée.
5. **`frontend/src/composables/useBoard.ts`** : injecter les stages du pipeline (fetch `GET /projects/{id}/pipeline`).
6. **SSE** (`composables/useSSE.ts` + `stores/stories.ts handleSSEEvent`) : gérer `stage.entered` → déplacer la carte de colonne.

**Hors-scope.** Pas de drag&drop. Pas de Go button (INC 3).

**Verify.** Board d'un projet à pipeline multi-stages → colonnes = noms de stages réels ; une carte running est dans son stage ; un `stage.entered` SSE la déplace en live ; toggle macro = ancienne vue lifecycle.

**▶ CC :** *« Implémente INC 2 : colonnes Kanban dérivées des stages du pipeline (frontend Vue). Régénère schema.d.ts, stageColumn() + toggle lifecycle, colonnes dynamiques dans KanbanBoard.vue, SSE stage.entered déplace la carte. Branch from develop. »*

---

## INC 3 — Politiques de transition : `manual` + Go, `gate`

**Goal.** Enforcer la politique de transition. `manual` → la carte **entre dans le stage et attend idle** avec un bouton **Go** ; `gate` → HITL après le segment (réutilise l'existant) ; `auto` → inchangé. Un Go enchaîne les stages `auto` jusqu'au prochain `manual`/`gate` (cf. modèle §3).

**Fichiers & changements :**

1. **`pipeline_executor.go`** : à l'entrée d'un stage `transition: manual` non encore déclenché → **suspendre** le run (même mécanique que HITL : `errStepSuspended` + pause), en attente d'un signal « start stage ». `gate` → HITL existant en fin de segment. `auto` → continuer.
2. **HITL / suspension** : réutiliser le pattern de `hitl_gate.go` + `hitl_service.go resumeRun`. Le « Go » d'un stage manuel = un resume ciblé.
3. **Nouvel endpoint** : `POST /projects/{id}/stories/{storyId}/stage/start` (ou réutiliser resume) → déclenche le stage manuel courant. OpenAPI + handler + wire.
4. **Front** : bouton **Go** par les règles du modèle §3 (visible : Backlog, ou carte idle dans stage `manual` ; pas pendant un segment running ni sur un gate). Carte « In Dev, pas démarrée » = état idle visible.

**Décisions §15 à trancher ici (au fil de l'eau, le user a dit « oui ») :** owner/assignee pas requis pour v1 — *skip*. Auto-recovery = INC 4 — *skip*.

**Verify.** Stage `manual` → carte parquée idle + Go ; clic → segment tourne puis auto-advance jusqu'au prochain manual/gate ; `gate` → HITL approve/reject comme aujourd'hui ; tout `auto` → un Go porte jusqu'à Done.

**▶ CC :** *« Implémente INC 3 : enforcement des politiques de transition (manual/gate/auto) + bouton Go + endpoint stage/start. Réutilise la mécanique de suspension HITL. Backend + frontend. Branch from develop. »*

---

## INC 4 — Guards / probes (beaucoup plus buildable qu'annoncé)

**Goal.** Couche de sécurité (§6 du modèle) : `halt-gate` par défaut sur probe-fail, déblocable (§7).

**Réalité vérifiée dans le CODE (pas seulement le doc runtime).** Le code actuel émet déjà via callbacks : **logs incrémentaux** (`agent_callback_handler.go` → `events.log.emitted` + `run_steps.log_tail`, timestampés `events.created_at`), **cost terminal-only** (`cost_records`), **status terminal**.

| Probe | Buildable maintenant ? |
|---|---|
| **heartbeat / liveness** | ✅ **oui** — dérivé du flux `log.emitted` : « pas de log depuis N sec » → halt. Couvre le **hang d'agent** (défaillance #1 de l'overnight). Ping dédié = amélioration future, **pas requis** |
| **wall-clock timeout** | ✅ **oui** — board-side (timer sur `step.started`) |
| **cost ceiling (batch / post-step)** | ✅ **oui** — coût en fin de step → halte le reste de l'epic si budget cumulé dépassé |
| **cost ceiling (mid-run)** | ⚠️ **petit ajout runtime** — émettre `SendCost` périodiquement, pas seulement sur l'event "result" |
| **blast-radius / loop / diff-sanity** | ⛔ phase 2 — besoin de signaux plus riches (files-touched, tool-call stream) |

→ **Le socle de sûreté nocturne (hang + wallclock + cost batch + halt-gate déblocable) est livrable sans toucher au runtime.** Seuls le cost mid-run (petit tweak) et les probes sémantiques (phase 2) restent.

**À porter au plan runtime (don't-regress + extend, cf. `board-pipeline-agents-model.md` §9 G1/G2) :** préserver l'émission existante à la réécriture, ajouter **cost incrémental** + **heartbeat dédié** + **resource-pressure**, pin le contrat **`Stop()` gracieux** (G2). Ne PAS éditer leur doc — le porter en discussion.

**INC 4a — livrable maintenant (le vrai filet de sûreté) :**
- `PipelineStep`/`PipelineGroup` : champ `guards: [{ kind, max/timeout, on_fail }]` (data, OpenAPI). Kinds : `log_silence` (heartbeat-via-logs), `wallclock`, `cost_batch`.
- Job de surveillance (River cron / timer) : pour chaque step running, comparer `now - (dernier log.emitted)` (query `events`) et `now - step.started` aux seuils → action `on_fail`.
- `halt-gate` = variant HITL `gate_type: probe_halt` (mig : étendre le CHECK de `hitl_requests.gate_type` ; `model/hitl.go:19`) + jeu de résolution enrichi (resume/override/take-over/send-back/skip/abort) côté service (`hitl_service.go`) + front.

**▶ CC (4a) :** *« Implémente INC 4a : probes log_silence (heartbeat via gap de logs) + wallclock + cost batch, board-side, + halt-gate (variant probe_halt du HITL) déblocable. Aucune dépendance runtime. Branch from develop. »*

**INC 4b — après extension runtime :** cost ceiling mid-run, probes sémantiques (blast-radius/loop), resource-pressure.

---

## INC 5 — Config UI : transition policy + guards (écart INC 3/4a)

**Goal.** Combler le trou de configuration découvert à l'audit front : le moteur enforce `transition` (INC 3) et les `guards` (INC 4a), mais le pipeline editor ne permet ni de **définir** la policy d'un stage, ni de **configurer** une probe. Sans ça, tout stage reste `auto` et aucune probe n'est activable depuis l'UI → features livrées mais dormantes. **Frontend only — backend + schéma OpenAPI (`transition`, `Guard` avec `on_fail`) supportent déjà tout.**

**Fichiers :** `frontend/src/views/PipelineConfigView.vue`, `frontend/src/features/pipeline/PipelineGroupCard.vue` (+ `PipelineStepCard.vue` si guards au niveau step), `frontend/src/stores/pipelineConfig.ts`.

1. **Éditeur de transition policy** : sur chaque stage (PipelineGroupCard), un Select `auto | manual | gate` (défaut `auto`, hint « auto = avance seule, manual = attend Go, gate = HITL »). Store : `updateGroupTransition(groupId, value)` → `isDirty`. Persisté par le `PUT /pipeline` existant.
2. **Éditeur de guards** : sur le stage (et/ou step), affordance « + Guard » + liste éditable. Par guard : `kind` (log_silence | wallclock | cost_batch), `threshold` (log_silence, secondes) / `max` (wallclock s, cost_batch USD selon kind), `on_fail` (halt-gate | fail | retry, défaut halt-gate). Store : `addGuard/removeGuard/updateGuard`. Types depuis le schéma généré (`Guard`). Persisté par `PUT /pipeline`.
3. Tests unitaires des nouvelles fonctions store + rendu des contrôles.

**Hors scope :** aucun backend ; pas d'affichage des breaches dans le run detail (volontaire). **Verify :** front build + type-check + lint + vitest verts. **▶ CC :** branche `feat/board-config-ui` depuis develop ; commits locaux, pas de push.

## Dette D1 — retry steps stampent le stage (backend)

**Goal.** `CreateRetryRunStep` laisse `stage_id`/`stage_name` à NULL (flaggé par l'agent INC 1) → un step rejoué perd son stage. Stamper le stage du step parent.
**Fichiers :** `backend/internal/domain/service/run_service.go` (+ repo/sqlc si besoin) — là où le retry step est créé ; copier `StageID`/`StageName` depuis le `ParentStepID`. **Verify :** go build/vet/test -short + golangci-lint ; test : un step rejoué porte le stage du parent. **▶ CC :** branche `fix/retry-step-stage-stamp` depuis develop ; commits locaux, pas de push.

## Ordre de lancement recommandé

1. **INC 1** (keystone) → merge develop.
2. **INC 2** (rendu) → v1 démontrable.
3. **INC 3** (interactif) → v1.1.
4. **INC 4a** (heartbeat-via-logs + wallclock + cost batch + halt-gate) → **dès après INC 1**, en parallèle de INC 2/3 ; **aucune dépendance runtime**. C'est le filet de sûreté nocturne.
5. **INC 4b** (cost mid-run + probes sémantiques) quand le runtime étend l'émission (G1) + pin `Stop()` (G2).

> CI develop verte avant chaque enchaînement (cf. règle projet). Sous-agents parallèles → worktrees obligatoires.
