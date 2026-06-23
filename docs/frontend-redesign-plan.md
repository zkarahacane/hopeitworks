# Frontend Redesign Plan — "The execution layer, made visible"

Basé sur : la passe design (Hopeitworks Redesign.pdf), l'audit UI (16 écrans + bugs U1–U4),
et les capacités backend (cost/logs/DAG/HITL). Stack cible : Vue 3 · PrimeVue 4 · Tailwind v4 · Pinia.

Parti pris : **console d'observabilité runtime, pas un CRUD**. Mono = vérité machine
(IDs, durées, coûts, conteneurs, logs), Grotesk = voix humaine. Couleur = sens, pas déco.

---

## Phase 0 — Fondation (design system)

### 0.1 Design tokens (Tailwind v4 `@theme`)
- **Surfaces dark** : `#0B0F0D` `#131A16` `#1A221D` `#28332C`
- **Surfaces light** : `#F6F8F6` `#FFFFFF` `#EFF2EF` `#D7DCD8`
- **Signaux (un seul système de statut)** : running = vert phosphore · gate = ambre · failed = rouge · info = bleu · queued = gris
- **Typo** : Space Grotesk (600/500/400/300 — headings, KPIs, UI) · JetBrains Mono (IDs, durées, coûts, branches, logs)
- Dark/Light en parité via sélecteur `.app-dark` (déjà la convention PrimeVue 4).

### 0.2 Système de statut unique (corrige "blue means everything")
Une seule source de vérité `statusToken(state)` mappant les **5 états produit** → token couleur + icône + label.
Aligner les enums backend (`run.status`, `run_step.status`, `story.status`) sur ces 5 familles.
→ supprime les usages bleu fourre-tout et l'incohérence badge/spinner.

### 0.3 Preset PrimeVue 4
Custom preset (nouvelle API de theming à tokens) calé sur la palette ci-dessus, dark/light.

### 0.4 Primitives "live signals" (CSS + SSE)
- **Pulse dot** — `@keyframes` opacity/scale, classe `.live-pulse`
- **Marching edges** — SVG `stroke-dasharray` + `stroke-dashoffset` animé (arêtes DAG actives)
- **Blinking cursor** — caret clignotant en pied de flux de logs
- **Count-up** — coût & durée qui montent (JS branché sur le flux SSE, pas de lib)
- **Amber breathe** — respiration ambre quand un gate attend un humain
Alimentées par un store Pinia `useRuntimeStream` au-dessus de `useSSE` (déjà présent).

### 0.5 Composants noyau (à créer/refactor)
`StatusBadge` · `AgentChip` · `ContainerChip` (`ctr·a3f9 · isolated`) · `CostTicker` ·
`LiveProgress` (determinate/indeterminate) · `StepTimeline` (phases) · `DagNode` + `DagEdge` ·
`LogStreamPanel` (remplace `RunStepLogPanel`, corrige le bug U1) · `HitlGateCard` (ambre) · `PhaseGroup`.

---

## Phase 1 — Les 4 écrans héros

| Héros | Route existante | Refonte |
|---|---|---|
| **Execution Graph** (dark, flagship) | `/projects/:id/epics/:eid/dag` (VueFlow) | nœuds story riches (statut, conteneur, timer, coût), arêtes "marching" entre stories parallèles, inspecteur latéral avec logs en flux + mini-map |
| **Run Detail + gate HITL** (dark) | `/runs/:id` | timeline par phase (Setup/Dev/Review/Delivery), **carte gate ambre** qui stoppe le pipeline (Approve/Request changes/Reject), flux de logs, **coût par rôle** |
| **Story Board** (light) | `/projects/:id/board` + `/epics/:eid` | kanban généré, bandeau **"Planned in" (GitHub/BMAD/Jira/markdown)**, colonne **In Review** qui réclame l'humain ("Needs you · review →") |
| **Pipeline Editor** (light) | `/projects/:id/pipeline` | restyle de l'éditeur existant, framing **"opinionated on runtime · free on process"**, palette de types de steps draggables, gate human visuellement distinct |

---

## Phase 2 — Écrans secondaires (~12, dans le système)

login · dashboard (dedupe runs) · projects · project overview · agents · costs ·
approvals · profile/API keys · admin users · notifications · settings (fix route) ·
run list · story detail · import stories.

---

## Contrat données ↔ live signals (là où design et backend convergent)

| Signal design | Donnée backend | État |
|---|---|---|
| Cost ticker / cost-by-role | `cost_records` (tokens + cost_usd) | ✅ peuplé (fix 6A) — manque l'agrégation par rôle/agent |
| Log stream panel | `run_steps.log_tail` + SSE log events | ✅ persisté sur succès (fix logs) |
| DAG live (arêtes marching, timers) | epic-run parallèle + SSE `run.*`/`step.*` | existe, à enrichir (deps, conteneur, coût par nœud) |
| Amber gate | `hitl_requests` + run `paused` | ✅ pause+resume validés (fix #3) |
| Statut unifié | enums `run/step/story.status` | à normaliser sur les 5 familles |
| Conteneur chip `ctr·a3f9` | `run_steps.container_id` | présent |

---

## 8 incohérences UX (design) → fixes, fusionnées avec l'audit

1. **`/settings` → 404** alors que le nav y pointe → unifier route Settings/Profile (U2)
2. **Badge "backlog" + spinner "Running"** simultanés → un seul statut dérivé (U3)
3. **Run failed $0.00 vs Costs $0.81** → remonter le coût au niveau run (cost rollup)
4. **Log panel bloqué "Connecting…/No output"** → `LogStreamPanel` + lifecycle correct (U1) + persistance log (fait)
5. **Dashboard = 6 runs S-01 identiques** → grouper/dédupliquer les runs par story
6. **"Blue means everything"** → système de statut unique (0.2)
7. Logo header tronqué "Hope" → marque complète
8. Langage couleur incohérent → tokens signaux (0.1)

---

## Ordre de build recommandé
1. **Phase 0** (tokens + statut unique + primitives live) — fondation, débloque tout le reste
2. **Phase 1** héros, dans l'ordre d'impact pitch : Execution Graph → Run Detail/HITL → Board → Pipeline Editor
3. **Phase 2** secondaires + les 8 fixes UX (la plupart tombent "gratuitement" une fois le système en place)

## À demander à la passe design ensuite
- DAG héros **interactif** (cliquable) — le flagship pour le pitch
- Les 12 écrans secondaires déclinés dans le système
- Parité dark/light complète sur chaque héros
