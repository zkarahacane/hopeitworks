# Frontend Redesign — Screen Reference

Source : passe design « Hopeitworks Redesign 2.pdf » (8 pages, art direction validée).
Référence canonique pour les agents Phase 1 (héros) et Phase 2 (secondaires).
À lire AVEC le contrat de fondation (tokens + `statusToken` + `useRuntimeStream` + composants noyau).

Parti pris design : **console d'observabilité runtime, pas un CRUD**. Mono (JetBrains Mono) =
vérité machine (IDs, durées, coûts, branches, conteneurs, logs). Space Grotesk = voix humaine.
Couleur = sens (5 signaux), jamais déco. Le bleu est banni des statuts.

---

## Système (page 1 — confirme le contrat P0)

- **Palette dark** : `#0B0F0D` `#131A16` `#1A221D` `#28332C`
- **Palette light** : `#F6F8F6` `#FFFFFF` `#EFF2EF` `#D7DCD8`
- **Signaux** : running (vert phosphore, pulsé) · gate (ambre) · failed (rouge) · queued (gris) · info (bleu, réservé non-statut)
- **Badges de statut** : `queued` · `● running` (pulse) · `◑ awaiting you` (ambre) · `✓ done` · `× failed`
- **Agent · container** : chip `Dev Agent` + `ctr·a3f9 · isolated` (mono)
- **Cost ticker** : `$0.8152` accruing live, per run (count-up, mono)
- **Live progress** : determinate / indeterminate
- **Light mode parity** : chaque écran décliné dark + light

---

## Héros 1 — Execution Graph (DARK · flagship) · `features/dag/`

Route : `/projects/:id/epics/:eid/dag`. Le flagship du pitch, interactif.

- **Header** : breadcrumb `hopeitworks / Todo App / epic·MVP` · indicateur `● 2 running` à droite.
- **Rail gauche** : icônes nav (board/graph/runs…), item graph actif (vert).
- **Titre** : « Execution Graph » · sous-titre `6 stories · 2 running in parallel · isolated containers` · légende dots running/done/failed.
- **Nœuds story** (riches, statut-colorés) :
  - `S-01` ✓ done — « Scaffold Go backend + Vue frontend » · `03:42` · `$0.18`
  - `S-02` ● running — « Setup CI pipeline with GitHub Actions » · `ctr·a3f9` · `03:18`
  - `S-03` ● running — « Configure linting and code formatting » · `ctr·7c1d` · `01:27`
  - `S-04` queued — « Auth API + session middleware » · `waiting on S-02`
  - `S-05` × failed — « Implement Todo CRUD endpoints » · `exit 1` · `create PR` · `↻ retry`
  - `S-06` queued — « Wire frontend to API » · `waiting on S-03`
- **Arêtes marching** : dashed vert animé entre stories dépendantes/parallèles (S-01→S-02/S-03, etc.).
- **Inspecteur latéral droit** (nœud sélectionné = S-02) : titre, chip `ctr·a3f…`, bloc **PIPELINE** (Setup `08:04` ✓ · Develop running · Review queued · Deliver queued), **LIVE LOG** (flux mono horodaté + blinking cursor).
- **Contrôles** : zoom +/- bas-gauche, toggle dark/light, **mini-map** bas-droite.
- **DagNode / DagEdge** : créés ICI (spécifiques feature), consomment la primitive marching-edge + `statusToken` + chips de la fondation.
- **Live** : `useRuntimeStream` → active-node set, timers/coûts par nœud, marching edges sur arêtes actives.

---

## Héros 2 — Run Detail · The Human Gate (DARK) · `features/runs/`

Route : `/runs/:id`. Fixes #2 (statut unique) #3 (cost rollup) #4 (log lifecycle U1).

- **Header** : breadcrumb `… / Runs / run·0a807b61` · `S-02 · run·0a807b61` + badge `◑ awaiting approval` ambre · titre « Setup CI pipeline… » · **ELAPSED `06:53`** (mono, count-up).
- **Timeline de phase** (horizontale) : `Setup ✓ ── Development ✓ ── Review & Merge ◑` (phase courante ambre) — composant `StepTimeline` / `PhaseGroup`.
- **Carte gate ambre** (`HitlGateCard`) :
  - « Human approval required » · « Sonnet Review Agent approved the change. Sign-off needed before merge to `main`. »
  - chip branche `feat/s02-ci-pipeline → main` · `+128` `-14` · `6 files · PR #42`
  - boutons : **Approve & merge** · **Request changes** · **Reject** · `waiting 04:43` · `assigned 9C`
- **COST BY ROLE** (droite) : barres Dev Agent / Review Agent / Merge Agent + « Total this run » — vient de l'agrégation backend (#6).
- **STEPS** : `Create branch · git_branch` ✓ `08:04` · `Implement story · Dev Agent` ✓ `03:17` · `Code review · Review Agent` ✓ `08:26` · `Approval gate · human` ◑ `04:43` · `Create PR · git_pr` — · `Notify completion · notify` —.
- **STREAM** (droite) : `● streaming · ctr·a3f…` log live mono + blinking cursor (`LogStreamPanel`).

---

## Héros 3 — Story Board (LIGHT · plan anywhere) · `features/stories/`

Route : `/projects/:id/board`.

- **Header** : breadcrumb `… / Board` · titre « Story Board » · sous-titre « Epic·MVP — board generated from your stories, kept live by the runtime ».
- **Bandeau « PLANNED IN »** (segmented) : `✓ GitHub Issues` · `BMAD` · `Jira` · … — source de planning.
- **Colonnes kanban** (compteurs en tête) :
  - **Backlog (2)** : S-04 Auth API `waiting on S-02` · S-06 Wire frontend `waiting on S-03`
  - **Running (2)** : S-02 Setup CI `● running` `ctr·a3f9` `03:18` · S-03 Configure linting `ctr·7c1d` `01:27`
  - **In Review (1)** : S-07 Add dark mode toggle `◑ gate` + **bouton ambre `Needs you · review →`**
  - **Done (3)** : S-01 Scaffold `03:42 $0.18` · S-08 Health check endpoint `01:08` · S-09 Dockerfile + compose `02:15 $0.09`
  - **Failed** : S-05 Implement… `exit 1`
- Cartes = `StatusBadge` + `ContainerChip` + cost/timer mono. Colonne In Review réclame l'humain.

---

## Héros 4 — Pipeline Editor (LIGHT · free on process) · `features/pipeline-editor/`

Route : `/projects/:id/pipeline`.

- **Header** : titre « Pipeline » + pill `opinionated on runtime · free on process` · sous-titre « Compose roles, steps and gates. The runtime handles containers, isolation & parallelism. » · bouton `+ Add group`.
- **Groupes → steps** (drag handle `⠿`, toggle `Auto`/`Manual` par step, type chip) :
  - **Setup (1)** : `1 Create branch` — `git_branch` — Auto
  - **Development (1)** : `1 Implement story` — `Dev Agent ▾` — `agent_run` — Manual
  - **Review & Merge (3)** : `1 Code review` — `Review Agent ▾` — `agent_run` — Auto · `2 Approval gate` « human stops the pipeline here » — `human` — Manual (**surligné ambre**) · `3 Create PR` — `git_pr` — Auto
  - **Delivery (2)** : `1 Wait for CI` — `ci_wait` — Auto · `2 Notify completion` — `notify` — Auto
  - `+ Add step` par groupe.
- **Rail droit** : palette **STEP types draggables** + liste **AGENTS** (chips D/R/M).
- Gate `human` visuellement distinct (ambre). Types de steps : `git_branch · agent_run · human · git_pr · ci_wait · notify`.

---

## Écrans secondaires (pages 6-8)

Tous déclinés dans le système, dark/light selon écran, incohérences UX corrigées.

| Écran | Mode | Notes clés |
|---|---|---|
| **Login** | dark | split : hero gauche « Plan anywhere. Watch it run. » + mini-graph animé + `© 2026 · runtime online ●` ; droite Sign in (email/password) + `Continue with GitHub` + « No account? Ask your workspace admin. » |
| **Dashboard** (control room) | dark | « Welcome back, Admin ». KPIs : `Active runs 3` · `◑ Gates waiting 2` (ambre) · `Stories done today 7` · `Spend today $2.419k` (vert). **Live runs** dédupliqués (fix #5) · panneau **« Needs you »** (Approval S-07 / S-11 + bouton Approve). |
| **Epic Stories · list** | dark | gauche liste stories (statut+timer) ; droite détail : description, **ACCEPTANCE CRITERIA** (checklist), `DEPENDS ON S-01 ✓`, `AGENT Dev Agent`, `RUN COST $0.7607`, bouton `Open run →`. Fix #8 (badge+spinner). |
| **Step Detail · Log Drawer** | dark | drawer droit `Implement story · running`, `started/elapsed/ctr`, tabs **Logs / Output / Diff**, `● streaming · ctr-a3f9` + blinking cursor, log mono. Fix #4 (états stream réels). |
| **Costs** (real rollups) | dark | « Rolled up per run, per role — no more $0.00 on failed runs » (fix #3). KPIs `This week $8.4205` · `This month $31.78` · `Avg/story $0.53`. Chart spend-over-time (ligne verte) · **By role** (barres Dev/Review/Merge) · table story/run/status. |
| **Approvals · HITL queue** | dark | « 2 waiting · Pipelines paused on a human gate. The runtime holds the container until you decide. » Cartes S-07 (`feat/s07-dark-mode→main · PR #44 +96 -8`, `waiting 04:43`, Approve) · S-11 (`PR #45 +212 -3`, `waiting 12:25`). |
| **404** (branded) | dark | mini-graph avec nœud rouge · « 404 · This node isn't in the graph » · « /settings now lives under your profile. » · boutons `Go to dashboard` / `Profile & settings`. **Fix #1**. |
| **Projects** | light | cards projet : Todo App (`● 2 running`, chips `docker/github/sonnet-4.6`, `◑ 1 gate`, `updated 8m ago`) · Billing Service (`idle`, `docker/gitlab/opus-4.6`, `18 stories`) · `+ Connect a repo`. |
| **Project Overview** (+ runtime config) | light | tabs `Overview/Board/Runs/Pipeline/Agents/Costs/Settings`. Info projet (repo, git provider, default model `claude-sonnet-4-6`, created `Jun 17 2026`, description) + panneau **Runtime** (image/max parallel/isolation/base branch) + stats `9 stories · 23 runs`. |
| **Project Agents** | light | « Roles available to assign in your pipeline ». Table : Todo Dev Agent (project, `claude-sonnet-4-6`, `agent-go-node:latest`) · Todo Merge Agent (project, `opus-4.6`, `ghcr.io/…/agent:latest`) · Sonnet Review Agent (global) · Opus Dev Agent (global). |
| **Profile · API keys** (button states fixed) | light | « My Profile » : Profile information (Name, Email, `Role admin`, `member since Jun 2026`, Save changes) · Change password (current/new/confirm, Update password) · **API keys** (« Stored encrypted — only last 4 chars », provider `claude`, key `sub-prod`, hint `····8gAA`). |
| **Administration · Users** | light | « User management » table email/name/role/created (badges `admin`/`user`), pagination `1 2`. Admin-only. |
| **Settings** | — | **Fix #1** : `/settings` top-level n'existe plus → vit sous le profil (« Profile & settings »). Unifier route Settings/Profile. |

---

## Rappel data ↔ live signals (où backend et design convergent)

| Signal | Donnée backend | État |
|---|---|---|
| Cost ticker / cost-by-role | `cost_records` (tokens + cost_usd) | peuplé ; **agrégation par rôle/agent + rollup run-level = lot backend #6** |
| Log stream / step drawer | `run_steps.log_tail` + SSE `log.emitted` | persisté ; `LogStreamPanel` fixe le lifecycle U1 |
| DAG live (marching, timers) | epic-run parallèle + SSE `run.*`/`step.*`/`epic_run.*` | à enrichir (deps, conteneur, coût par nœud) via `useRuntimeStream` |
| Amber gate | `hitl_requests` + run `paused` | pause/resume OK ; `HitlGateCard` |
| Statut unifié | enums `run/step/story/epic.status` | normalisé sur 5 familles via `statusToken` (P0) |
| Container chip `ctr·a3f9` | `run_steps.container_id` | présent |
