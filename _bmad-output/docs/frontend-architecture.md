# hopeitworks Frontend Architecture

## Table of Contents

- [1. Architecture globale](#1-architecture-globale)
  - [1.1 Stack technique](#11-stack-technique)
  - [1.2 Structure des dossiers](#12-structure-des-dossiers)
  - [1.3 Point d'entree et bootstrap](#13-point-dentree-et-bootstrap)
  - [1.4 Routing et guards](#14-routing-et-guards)
  - [1.5 Gestion d'etat (Pinia)](#15-gestion-detat-pinia)
  - [1.6 API Client](#16-api-client)
  - [1.7 Theming PrimeVue](#17-theming-primevue)
  - [1.8 CSS et Tailwind](#18-css-et-tailwind)
  - [1.9 Configuration Vite](#19-configuration-vite)
- [2. Features](#2-features)
  - [2.1 board - Story Board et gestion des epics](#21-board---story-board-et-gestion-des-epics)
  - [2.2 runs - Execution et suivi des pipelines](#22-runs---execution-et-suivi-des-pipelines)
  - [2.3 pipeline - Configuration du pipeline](#23-pipeline---configuration-du-pipeline)
  - [2.4 agents - Gestion des agents IA](#24-agents---gestion-des-agents-ia)
  - [2.5 approvals - Workflow HITL (Human-in-the-Loop)](#25-approvals---workflow-hitl-human-in-the-loop)
  - [2.6 dag - Visualisation du DAG](#26-dag---visualisation-du-dag)
  - [2.7 epics - Monitoring des epic runs](#27-epics---monitoring-des-epic-runs)
  - [2.8 costs - Suivi des couts](#28-costs---suivi-des-couts)
  - [2.9 projects - Gestion des projets](#29-projects---gestion-des-projets)
  - [2.10 admin - Administration utilisateurs](#210-admin---administration-utilisateurs)
  - [2.11 profile - Profil utilisateur](#211-profile---profil-utilisateur)
  - [2.12 notifications - Canaux de notification](#212-notifications---canaux-de-notification)
- [3. UI partagee](#3-ui-partagee)
  - [3.1 Layout](#31-layout)
  - [3.2 Composed](#32-composed)
- [4. Composables](#4-composables)
- [5. Stores](#5-stores)
- [6. Utilitaires](#6-utilitaires)
- [7. Configuration et build](#7-configuration-et-build)

---

## 1. Architecture globale

### 1.1 Stack technique

| Technologie | Version | Role |
|---|---|---|
| Vue 3 | ^3.5.28 | Framework UI, Composition API exclusivement |
| TypeScript | ~5.9.3 | Typage strict |
| PrimeVue 4 | ^4.5.4 | Bibliotheque de composants UI (preset Aura, unstyled) |
| Tailwind CSS v4 | ^4.1.18 | Utilitaires de layout uniquement |
| Pinia 3 | ^3.0.4 | Gestion d'etat |
| Vue Router 5 | ^5.0.2 | Routing SPA avec guards d'auth |
| openapi-fetch | ^0.17.0 | Client API type depuis OpenAPI spec |
| vee-validate + zod | ^4.15.1 / ^3.25.76 | Validation de formulaires |
| @vueuse/core | ^14.2.1 | Composables utilitaires |
| @vue-flow/core | ^1.48.2 | Visualisation de graphes DAG |
| Monaco Editor | ^0.55.1 | Editeur de templates agents |
| Chart.js | ^4.5.1 | Graphiques de couts |
| ansi-to-html | ^0.7.2 | Rendu des logs agents (ANSI) |
| diff2html | ^3.4.56 | Visualisation de diffs PR |
| marked + DOMPurify | ^17.0.2 / ^3.3.1 | Rendu Markdown securise |
| Handlebars | ^4.7.8 | Preview de templates agents cote client |
| date-fns | ^4.1.0 | Formatage de dates (tree-shakeable) |
| Vite 7 | ^7.3.1 | Build tool et dev server |
| Vitest 4 | ^4.0.18 | Tests unitaires |
| Playwright | ^1.58.2 | Tests E2E |

### 1.2 Structure des dossiers

```
frontend/src/
├── api/                    # Client API genere depuis OpenAPI
│   ├── client.ts           # Instance openapi-fetch avec middleware auth
│   └── schema.d.ts         # Types generes depuis api/openapi.yaml
├── assets/
│   └── main.css            # CSS layers (tailwind-base, primevue, tailwind-utilities)
├── composables/            # Composables partages (logique pure reutilisable)
├── features/               # Domaines metier (composants + composables locaux)
│   ├── admin/
│   ├── agents/
│   ├── approvals/
│   ├── board/
│   ├── costs/
│   ├── dag/
│   ├── epics/
│   ├── notifications/
│   ├── pipeline/
│   ├── profile/
│   ├── projects/
│   └── runs/
├── router/                 # Definitions de routes et guards
│   ├── index.ts
│   └── guards.ts
├── stores/                 # Stores Pinia (un par domaine)
├── theme/                  # Configuration PrimeVue (tokens et preset)
│   ├── index.ts            # HopeTheme (definePreset Aura)
│   └── tokens.ts           # Tokens primitifs et semantiques
├── types/                  # Types partages
│   └── pagination.ts
├── ui/                     # Composants partages (layout, primitives, composed)
│   ├── layout/             # AppShell, AppHeader, AppSidebar, AppStatusBar
│   ├── primitives/         # (reserve, vide pour le moment)
│   └── composed/           # LogViewer
├── utils/                  # Fonctions pures (formateurs, parseurs)
├── views/                  # Composants de vue (1:1 avec les routes)
│   └── admin/
├── App.vue                 # Composant racine (rend AppShell)
└── main.ts                 # Bootstrap de l'application
```

### 1.3 Point d'entree et bootstrap

**`src/main.ts`** initialise l'application dans cet ordre :

1. Import du CSS global (`assets/main.css`)
2. Creation de l'app Vue avec `createApp(App)`
3. Installation de Pinia (gestion d'etat)
4. Installation de Vue Router
5. Installation de PrimeVue ConfirmationService et ToastService
6. Enregistrement de la directive `v-tooltip`
7. Configuration PrimeVue avec le preset `HopeTheme` (Aura), dark mode via classe `.dark`, CSS layers ordonnes `tailwind-base, primevue, tailwind-utilities`
8. Montage sur `#app`

**`src/App.vue`** est minimal : il rend uniquement `<AppShell />`.

### 1.4 Routing et guards

**Fichier** : `src/router/index.ts`

Le routeur utilise `createWebHistory` et definit les routes suivantes :

| Route | Nom | Composant | Auth | Notes |
|---|---|---|---|---|
| `/login` | `login` | LoginView | Non | |
| `/forgot-password` | `forgot-password` | ForgotPasswordView | Non | |
| `/reset-password` | `reset-password` | ResetPasswordView | Non | Token en query param |
| `/` | `dashboard` | DashboardView | Oui | |
| `/projects` | `projects` | ProjectsView | Oui | |
| `/projects/:id` | - | ProjectDetailView | Oui | Layout parent avec tabs |
| `/projects/:id/` | `project-overview` | ProjectOverview | Oui | Enfant |
| `/projects/:id/board` | `project-board` | BoardView | Oui | Enfant |
| `/projects/:id/runs` | `project-runs` | ProjectRunsView | Oui | Lazy-loaded |
| `/projects/:id/epics/:epicId` | `epic-detail` | EpicDetailView | Oui | Lazy-loaded |
| `/projects/:id/epics/:epicId/dag` | `epic-dag` | EpicDagView | Oui | Lazy-loaded |
| `/projects/:id/epic-runs/:epicRunId` | `epic-run-monitor` | EpicRunView | Oui | Lazy-loaded |
| `/projects/:id/pipeline` | `project-pipeline` | PipelineConfigView | Oui | |
| `/projects/:id/agents` | `project-agents` | AgentListView | Oui | |
| `/projects/:id/agents/new` | `agent-create` | AgentEditorView | Oui + Admin | Lazy-loaded |
| `/projects/:id/agents/:agentId` | `agent-editor` | AgentEditorView | Oui | Lazy-loaded |
| `/projects/:id/runs/:runId/approve/:stepId` | `hitl-approve` | HITLApprovalView | Oui | Lazy-loaded |
| `/projects/:id/settings` | `project-settings` | ProjectSettingsView | Oui | Lazy-loaded |
| `/projects/:id/settings/notifications` | `project-notifications` | NotificationSettingsView | Oui | Lazy-loaded |
| `/projects/:id/costs` | `project-costs` | CostDashboardView | Oui | Lazy-loaded |
| `/projects/:projectId/stories/:storyId` | `story-detail` | StoryDetailView | Oui | |
| `/runs/:id` | `run-detail` | RunDetailView | Oui | |
| `/approvals` | `approvals` | ApprovalsView | Oui | |
| `/runs` | `runs` | RunsView | Oui | |
| `/admin/users` | `admin-users` | UserManagementView | Oui + Admin | Lazy-loaded |
| `/profile` | `profile` | ProfileView | Oui | Lazy-loaded |
| `/:pathMatch(.*)*` | `not-found` | NotFoundView | Non | Catch-all 404 |

**Guards** (`src/router/guards.ts`) :

1. **`setupAuthGuard`** : Executee avant chaque navigation. Restaure la session une seule fois au premier chargement via `auth.checkAuth()`. Redirige vers `/login?redirect=...` si la route necessite l'authentification et que l'utilisateur n'est pas connecte. Redirige les utilisateurs connectes qui accedent a `/login` vers le dashboard.

2. **`setupAdminGuard`** : Redirige les utilisateurs non-admin vers `/` lorsqu'une route a `meta.requiresAdmin: true`.

### 1.5 Gestion d'etat (Pinia)

Tous les stores suivent le pattern **setup store** (Composition API) sauf `auth` qui utilise le pattern Options API. Un store par domaine metier. Details en [section 5](#5-stores).

### 1.6 API Client

**Fichier** : `src/api/client.ts`

Client cree avec `openapi-fetch` et type via `paths` depuis `src/api/schema.d.ts` (genere par `openapi-typescript` depuis `api/openapi.yaml`).

Configuration :
- `baseUrl: '/api/v1'`
- `credentials: 'include'` (cookies httpOnly pour JWT)
- Middleware `authMiddleware` : intercepte les reponses 401 sur les endpoints non-auth et redirige vers la page de login

Generation des types :
```bash
npm run generate-api
# Equivalent a : openapi-typescript ../api/openapi.yaml -o src/api/schema.d.ts
```

### 1.7 Theming PrimeVue

**Fichiers** : `src/theme/index.ts`, `src/theme/tokens.ts`

Le theme `HopeTheme` est un preset `definePreset(Aura, ...)` qui surcharge la palette primaire avec les tokens bleu de PrimeVue (`{blue.50}` a `{blue.950}`).

`tokens.ts` definit des tokens primitifs (borderRadius, spacing) et semantiques (surface colors pour light/dark mode) a titre documentaire. Le dark mode est active via la classe `.dark` sur `<html>`.

### 1.8 CSS et Tailwind

**Fichier** : `src/assets/main.css`

CSS organise en layers :
```css
@layer tailwind-base, primevue, tailwind-utilities;
```

Cet ordre garantit que les utilitaires Tailwind peuvent surcharger les styles PrimeVue. Tailwind est utilise **uniquement pour le layout** (flex, grid, gap, padding, margin, width, height). Les couleurs, typographie et styles de composants viennent de PrimeVue.

Une seule animation custom est definie : `pulse-run` pour l'indicateur de step en cours d'execution.

### 1.9 Configuration Vite

**Fichier** : `vite.config.ts`

- **Plugins** : `vue()`, `tailwindcss()`, `vueDevTools()`
- **Alias** : `@` pointe vers `./src`
- **Build** : Monaco Editor est isole dans un chunk separe (`manualChunks`)
- **Dev server** : port 5173, proxy `/api/v1` vers `http://localhost:8080`

---

## 2. Features

### 2.1 board - Story Board et gestion des epics

**But** : Afficher la liste des epics d'un projet sous forme de grille de cartes, permettre la navigation vers le detail d'un epic, et gerer les stories au sein d'un epic (liste, filtre, detail, creation, edition, import).

**Composants** :

| Fichier | Description |
|---|---|
| `EpicCardGrid.vue` | Grille responsive de cartes d'epics |
| `EpicCard.vue` | Carte d'un epic affichant titre, description et compteurs de stories par statut |
| `EpicDetailLayout.vue` | Layout split-panel : liste de stories a gauche, detail de la story selectionnee a droite |
| `StoryListPanel.vue` | Panneau de gauche avec barre de filtre et liste scrollable de stories |
| `StoryFilterBar.vue` | Barre de filtres (statut, recherche textuelle) |
| `StoryStatusCard.vue` | Carte d'une story dans la liste avec key, titre et badge de statut |
| `StoryDetailPanel.vue` | Panneau de detail d'une story : metadonnees, objectif, criteres d'acceptation, fichiers cibles, dependances, bouton de lancement |
| `StoryEditorForm.vue` | Formulaire d'edition inline d'une story (title, objective, acceptance_criteria, target_files, depends_on, scope) |
| `CreateStoryDialog.vue` | Dialog modale pour creer une nouvelle story |
| `StoryImportDialog.vue` | Dialog d'import de stories depuis un fichier Markdown |
| `RunStatusIndicator.vue` | Badge de statut du dernier run d'une story |
| `BoardEmptyState.vue` | Etat vide quand aucun epic n'existe |

**Composables utilises** : `useEpics`, `useStories`, `useStoryEditor`, `useStoryImport`, `useRunLauncher`

**Stores** : `epics`, `stories`

**Routes** : `project-board` (BoardView), `epic-detail` (EpicDetailView), `story-detail` (StoryDetailView)

**API** :
- `GET /projects/{projectId}/epics` - liste des epics
- `GET /projects/{projectId}/stories` - liste des stories (filtre par epic_id)
- `GET /projects/{projectId}/stories/{storyId}` - detail d'une story
- `PUT /projects/{projectId}/stories/{storyId}` - mise a jour d'une story
- `POST /projects/{projectId}/stories` - creation d'une story
- `POST /projects/{projectId}/stories/import` - import de stories depuis markdown

### 2.2 runs - Execution et suivi des pipelines

**But** : Lancer des runs sur des stories, visualiser le pipeline d'execution avec ses steps, consulter les logs en temps reel, controler le run (pause/resume/cancel/retry), et afficher les couts par step.

**Composants** :

| Fichier | Description |
|---|---|
| `RunPipelineView.vue` | Vue horizontale du pipeline montrant les stages et steps avec leur statut |
| `RunStageColumn.vue` | Colonne d'un stage dans le pipeline (groupe de steps) |
| `RunJobRow.vue` | Ligne d'un step dans un stage avec nom, statut, duree et boutons d'action |
| `RunLaunchButton.vue` | Bouton de lancement d'un run |
| `RunLaunchConfirmDialog.vue` | Dialog de confirmation avant lancement |
| `RunCancelConfirmDialog.vue` | Dialog de confirmation avant annulation |
| `RunStepLogPanel.vue` | Panneau lateral (drawer) affichant les logs d'un step en temps reel via SSE |

**Composables locaux** :

| Fichier | Description |
|---|---|
| `useRunDetail.ts` | Fetche un run avec ses steps, s'abonne aux SSE pour rafraichir automatiquement sur les evenements run/step |
| `useRecentRuns.ts` | Fetche les runs recents (scope projet ou global cross-projet via fan-out) |
| `useStepLogs.ts` | S'abonne aux SSE `log.emitted` et collecte les lignes de log filtrees par runId/stepId |
| `useRunCosts.ts` | Fetche le detail des couts d'un run (total + breakdown par step), refetch sur changement de statut |

**Stores** : `runs`

**Routes** : `run-detail` (RunDetailView), `runs` (RunsView), `project-runs` (ProjectRunsView)

**API** :
- `GET /runs/{runId}` - detail d'un run avec steps
- `GET /projects/{projectId}/runs` - liste des runs d'un projet
- `POST /projects/{projectId}/stories/{storyId}/runs` - lancement d'un run
- `POST /projects/{projectId}/runs/{runId}/pause` - pause d'un run
- `POST /projects/{projectId}/runs/{runId}/resume` - reprise d'un run
- `POST /projects/{projectId}/runs/{runId}/cancel` - annulation d'un run
- `POST /runs/{runId}/steps/{stepId}/retry` - retry d'un step
- `GET /projects/{projectId}/runs/{runId}/costs` - couts d'un run
- SSE `GET /api/v1/events/stream?project_id=...` - evenements temps reel

### 2.3 pipeline - Configuration du pipeline

**But** : Configurer les groupes et steps du pipeline d'un projet (structure, noms, types d'action, agents assignes, retry policies).

**Composants** :

| Fichier | Description |
|---|---|
| `PipelineStepList.vue` | Liste des groupes de pipeline avec drag-and-drop pour reordonner |
| `PipelineGroupCard.vue` | Carte d'un groupe avec son nom editable, liste de steps, boutons add/remove |
| `PipelineStepCard.vue` | Carte d'un step avec nom, action, agent assigne, retry policy |
| `AddStepDialog.vue` | Dialog pour ajouter un nouveau step a un groupe (nom, action, agent_id, retry) |

**Composables utilises** : `usePipelineConfig`

**Stores** : `pipelineConfig`

**Routes** : `project-pipeline` (PipelineConfigView)

**API** :
- `GET /projects/{projectId}/pipeline` - lecture de la config pipeline
- `PUT /projects/{projectId}/pipeline` - sauvegarde de la config pipeline

### 2.4 agents - Gestion des agents IA

**But** : Lister, creer, editer et supprimer les agents IA d'un projet. Chaque agent a un nom, modele LLM, image Docker et un template Handlebars de prompt.

**Composants** :

| Fichier | Description |
|---|---|
| `AgentTable.vue` | Table des agents avec colonnes (nom, modele, image, scope) et actions (edit, delete) |
| `AgentEmptyState.vue` | Etat vide quand aucun agent n'existe |
| `AgentEditorLayout.vue` | Layout de l'editeur : toolbar + editeur Monaco + sidebar de variables |
| `AgentEditorToolbar.vue` | Toolbar avec nom, modele, image, scope, boutons save/preview/cancel |
| `MonacoEditorWrapper.vue` | Wrapper pour l'editeur Monaco (Handlebars templates) |
| `AgentVariableSidebar.vue` | Sidebar listant les variables Handlebars disponibles pour le template |
| `AgentPreviewDialog.vue` | Dialog affichant le rendu du template avec des donnees d'exemple |

**Composables utilises** : `useAgents`, `useAgentEditor`

**Stores** : `agents`

**Routes** : `project-agents` (AgentListView), `agent-create` (AgentEditorView), `agent-editor` (AgentEditorView)

**API** :
- `GET /projects/{projectId}/agents` - liste des agents
- `GET /projects/{projectId}/agents/{agentId}` - detail d'un agent
- `POST /projects/{projectId}/agents` - creation d'un agent
- `PUT /projects/{projectId}/agents/{agentId}` - mise a jour d'un agent
- `DELETE /projects/{projectId}/agents/{agentId}` - suppression d'un agent

### 2.5 approvals - Workflow HITL (Human-in-the-Loop)

**But** : Lister les approbations en attente et permettre a un humain d'approuver ou rejeter un changement genere par un agent, en visualisant le diff du code.

**Composants** :

| Fichier | Description |
|---|---|
| `HITLPendingTable.vue` | Table des requetes HITL en attente avec story key, titre, date et bouton "Review" |
| `DiffViewer.vue` | Visualisation de diff (side-by-side ou line-by-line) utilisant diff2html |

**Composables locaux** :

| Fichier | Description |
|---|---|
| `useApprovalActions.ts` | Actions approve/reject wrappees dans `useAsyncAction`, appellent les endpoints HITL |

**Stores** : `hitl`, `approvals`

**Routes** : `approvals` (ApprovalsView), `hitl-approve` (HITLApprovalView)

**API** :
- `GET /hitl-requests` - liste des requetes HITL (filtre par status)
- `GET /hitl-requests/by-step/{stepId}` - requete HITL pour un step specifique
- `POST /hitl-requests/{hitlRequestId}/approve` - approuver
- `POST /hitl-requests/{hitlRequestId}/reject` - rejeter (avec raison)

### 2.6 dag - Visualisation du DAG

**But** : Afficher le graphe de dependances (DAG) des stories d'un epic, et lancer un epic run (execution de toutes les stories).

**Composants** :

| Fichier | Description |
|---|---|
| `DagGraph.vue` | Graphe interactif utilisant VueFlow avec nodes de stories et edges de dependances |
| `DagStoryNode.vue` | Noeud custom pour une story dans le DAG (key, titre, badge de statut) |

**Composables locaux** :

| Fichier | Description |
|---|---|
| `useDagLayout.ts` | Fetche les donnees DAG depuis l'API et les transforme en nodes/edges VueFlow avec positionnement par layer |
| `useEpicLauncher.ts` | Lance un epic run via POST et retourne l'epic_run_id |

**Routes** : `epic-dag` (EpicDagView)

**API** :
- `GET /projects/{projectId}/epics/{epicId}/dag` - donnees du DAG
- `POST /projects/{projectId}/epics/{epicId}/runs` - lancement d'un epic run

### 2.7 epics - Monitoring des epic runs

**But** : Monitorer en temps reel l'execution d'un epic run avec progression, statut par story, et graphe VueFlow.

**Composants** :

| Fichier | Description |
|---|---|
| `EpicRunStatusNode.vue` | Noeud custom VueFlow pour le monitoring d'un epic run (statut par story) |
| `EpicRunGroupList.vue` | Liste des groupes d'execution (layers) avec les stories et leur statut |

**Composables locaux** :

| Fichier | Description |
|---|---|
| `useEpicRunMonitor.ts` | Wire le store epicRun avec SSE pour mise a jour temps reel, derive les nodes/edges VueFlow |

**Stores** : `epicRun`

**Routes** : `epic-run-monitor` (EpicRunView)

**API** :
- `GET /projects/{projectId}/epic-runs/{epicRunId}` - detail d'un epic run
- SSE evenements `epic_run.*`

### 2.8 costs - Suivi des couts

**But** : Dashboard de suivi des couts d'un projet : resume, graphique temporel, couts par run et couts par agent.

**Composants** :

| Fichier | Description |
|---|---|
| `CostSummaryCard.vue` | Carte de resume (cout total semaine, mois, cout moyen par story) |
| `CostChart.vue` | Graphique Chart.js des couts dans le temps |
| `RunCostTable.vue` | Table des runs avec leur cout |
| `AgentCostTable.vue` | Table des couts agreges par agent |

**Composables utilises** : `useCosts`

**Routes** : `project-costs` (CostDashboardView)

**API** :
- `GET /projects/{projectId}/costs/summary` - resume des couts
- `GET /projects/{projectId}/costs/chart` - donnees du graphique
- `GET /projects/{projectId}/costs/runs` - couts par run
- `GET /projects/{projectId}/costs/agents` - couts par agent

### 2.9 projects - Gestion des projets

**But** : CRUD de projets avec liste paginee, creation, settings, overview.

**Composants** :

| Fichier | Description |
|---|---|
| `ProjectListTable.vue` | Table de projets avec pagination server-side |
| `ProjectEmptyState.vue` | Etat vide quand aucun projet n'existe |
| `CreateProjectDialog.vue` | Dialog de creation de projet (nom, description, repo_url, git_provider, etc.) |
| `ProjectOverview.vue` | Vue d'ensemble d'un projet (epics, stats) |
| `ProjectSettingsForm.vue` | Formulaire des settings projet (nom, description, repo, provider, model, runtime) |
| `CircuitBreakerBanner.vue` | Banniere d'alerte quand le circuit breaker est actif sur un projet |

**Composables utilises** : `useProjects`, `useProject`

**Stores** : `projects`

**Routes** : `projects` (ProjectsView), `project-overview` (ProjectDetailView > ProjectOverview), `project-settings` (ProjectSettingsView)

**API** :
- `GET /projects` - liste paginee des projets
- `GET /projects/{id}` - detail d'un projet
- `POST /projects` - creation
- `PUT /projects/{id}` - mise a jour

### 2.10 admin - Administration utilisateurs

**But** : CRUD utilisateurs (admin only) : liste paginee, creation, edition, suppression.

**Composants** :

| Fichier | Description |
|---|---|
| `UserTable.vue` | Table des utilisateurs avec actions edit/delete et pagination |
| `CreateUserDialog.vue` | Dialog de creation d'un utilisateur (email, password, name) |
| `EditUserDialog.vue` | Dialog d'edition d'un utilisateur (name, email) |

**Composables utilises** : `useUsers`

**Stores** : `users`

**Routes** : `admin-users` (UserManagementView)

**API** :
- `GET /users` - liste paginee
- `POST /auth/register` - creation
- `PUT /users/{id}` - mise a jour
- `DELETE /users/{id}` - suppression

### 2.11 profile - Profil utilisateur

**But** : Self-service pour le profil utilisateur : modifier nom/email, changer le mot de passe.

**Composants** :

| Fichier | Description |
|---|---|
| `ProfileInfoForm.vue` | Formulaire de modification du profil (nom, email) |
| `ChangePasswordForm.vue` | Formulaire de changement de mot de passe (current + new + confirm) |

**Composables utilises** : `useProfile`

**Stores** : `auth` (via useProfile)

**Routes** : `profile` (ProfileView)

**API** :
- `GET /users/me` - profil courant
- `PUT /users/me` - mise a jour du profil
- `PUT /users/me/password` - changement de mot de passe

### 2.12 notifications - Canaux de notification

**But** : Configurer des canaux de notification (Discord, webhook) pour recevoir des alertes sur les evenements pipeline.

**Composants** :

| Fichier | Description |
|---|---|
| `NotificationChannelRow.vue` | Ligne d'un canal : type, config masquee, toggle on/off, boutons test/delete |
| `AddChannelDialog.vue` | Dialog de creation d'un canal (type, URL/webhook, filtres d'evenements) |

**Composables utilises** : `useNotifications`

**Routes** : `project-notifications` (NotificationSettingsView)

**API** :
- `GET /projects/{projectId}/notifications` - liste des configs
- `POST /projects/{projectId}/notifications` - creation
- `PUT /projects/{projectId}/notifications/{notificationId}` - mise a jour
- `DELETE /projects/{projectId}/notifications/{notificationId}` - suppression
- `POST /projects/{projectId}/notifications/{notificationId}/test` - envoi d'un test

---

## 3. UI partagee

### 3.1 Layout

Composants de structure applicative dans `src/ui/layout/` :

| Fichier | Description |
|---|---|
| `AppShell.vue` | Shell principal de l'application. Gere deux modes : authentifie (header + sidebar + content + status bar + mobile bottom nav) et non-authentifie (router-view seul). Integre le systeme de notification toast pour les HITL, les raccourcis clavier (`[` pour toggle sidebar), et la gestion responsive (mobile overlay sidebar). |
| `AppHeader.vue` | Barre de header fixe avec logo, hamburger menu (mobile), et menu utilisateur (profil, logout) via PrimeVue `Menu` popup. |
| `AppSidebar.vue` | Navigation laterale collapsible. Items : Dashboard, Projects, Runs, Approvals, Settings. Section admin conditionnelle. Badge de notification sur Approvals (pendingCount du HITL store). Mode mobile : overlay drawer avec backdrop. |
| `AppStatusBar.vue` | Barre de statut en pied de page (desktop uniquement). Affiche l'indicateur de connexion et le numero de version. |

### 3.2 Composed

Composants composes reutilisables dans `src/ui/composed/` :

| Fichier | Description |
|---|---|
| `LogViewer.vue` | Visualiseur de logs en temps reel. Affiche des lignes avec timestamps et coloration ANSI (via `formatLogLine`). Auto-scroll intelligent (se met en pause quand l'utilisateur scrolle vers le haut). Toolbar avec statut SSE (Tag avec severite) et bouton Clear. Export du type `LogLine` (text + timestamp) et accepte un prop `SSEStatus`. |

---

## 4. Composables

Tous les composables partages sont dans `src/composables/`.

| Composable | Description | Utilise par |
|---|---|---|
| **`useAsyncAction`** | Pattern generique pour les operations async. Retourne `{ data, error, isLoading, execute }`. Gere automatiquement le loading state et la capture d'erreurs. Accepte une fonction async generiquement typee. | Quasi-totalite des composables et views |
| **`useSSE`** | Gere une connexion EventSource vers `/api/v1/events/stream?project_id=...`. Ecoute 17 types d'evenements connus (run.*, step.*, log.*, hitl.*, story.*, epic_run.*). Dispatche les evenements parses via un callback `onEvent(eventName, data)`. Auto-cleanup via `onBeforeUnmount`. Retourne `{ status, close }`. | RunDetailView, useRunDetail, useRecentRuns, useStepLogs, useEpicRunMonitor |
| **`useAuth`** | Wrapper readonly du store auth. Expose `user`, `isAuthenticated`, `loading`, `error`, `login`, `logout`, `checkAuth`, `forgotPassword`, `resetPassword` comme computed/bound methods. | LoginView, ForgotPasswordView, ResetPasswordView, AppShell, guards |
| **`useKeyboard`** | Enregistre des raccourcis clavier globaux. Ignore les evenements provenant d'inputs/textareas/contenteditable. Bind/unbind sur mount/unmount. | AppShell (`[` pour toggle sidebar) |
| **`useBreakpoint`** | Detecte le breakpoint mobile (`max-width: 1023px`) via `matchMedia`. Retourne `{ isMobile }` reactif. | AppShell |
| **`useRelativeTime`** | Retourne un temps relatif reactif ("just now", "5m ago", "3d ago") qui se met a jour toutes les 60 secondes via `useIntervalFn`. Accepte un `MaybeRef<string \| Date \| null>`. | Composants de detail |
| **`useNotifications`** | CRUD complet pour les configs de notification d'un projet. Optimistic update pour le toggle enabled/disabled. Auto-fetch on mount. | NotificationSettingsView |
| **`useProject`** | Fetche un projet unique par ID. Auto-fetch on mount. Retourne `{ project, isLoading, error, fetchProject, retry }`. | ProjectDetailView |
| **`useProjects`** | Wrapper du store projects avec retry et `useAsyncAction` pour create/update. | ProjectsView, ProjectSettingsView |
| **`useStories`** | Gere la liste de stories d'un epic : fetch, filtres (statut, recherche), selection. Auto-fetch on mount. Watch sur le filtre statut pour re-fetch. | EpicDetailView |
| **`useStoryDetail`** | Fetche une story unique avec `useAsyncAction`. Auto-fetch on mount. | StoryDetailView |
| **`useStoryEditor`** | Gestion d'un mode d'edition inline pour une story. Possede tout le state d'edition : `isEditing`, `draftFields`, `validationErrors`, `apiError`, `isSaving`. Methods : `startEdit`, `cancelEdit`, `saveEdit`. | StoryDetailPanel, StoryEditorForm |
| **`useStoryImport`** | Import de stories depuis un fichier Markdown. Parse cote client pour preview (extraction key/title/scope depuis frontmatter), puis appel API pour l'import. | StoryImportDialog |
| **`useEpics`** | Wrapper du store epics. Auto-fetch on mount. | BoardView |
| **`useRunLauncher`** | Lance un run sur une story via POST. Gere le cas 409 Conflict (ALREADY_RUNNING_ERROR). | EpicDetailView, StoryDetailView |
| **`useAgents`** | Wrapper du store agents. Fetch avec pagination et retry. | AgentListView, PipelineConfigView |
| **`useAgentEditor`** | Etat complet de l'editeur d'agent : fetch/save/preview, dirty tracking, gestion new vs existing. Preview des templates Handlebars cote client avec un contexte d'exemple. | AgentEditorView |
| **`usePipelineConfig`** | Wrapper du store pipelineConfig. Auto-fetch on mount et watch sur projectId. Expose toutes les operations de groupe et step. | PipelineConfigView |
| **`useCosts`** | Fetche les donnees de cout d'un projet (summary, chart, runs, agents). Gestion du period (7d/30d). Auto-fetch on mount. | CostDashboardView |
| **`useUsers`** | Wrapper du store users avec `useAsyncAction` pour chaque operation CRUD. | UserManagementView |
| **`useProfile`** | Operations self-service : fetchMe, updateMe, changePassword. Wrappees dans `useAsyncAction`. | ProfileView |

---

## 5. Stores

Tous les stores Pinia sont dans `src/stores/`.

### `auth` (Options API)

**State** : `user: User | null`, `loading: boolean`, `error: string | null`

**Getters** : `isAuthenticated` (user !== null)

**Actions** :
- `login(email, password)` - authentification, retourne boolean
- `logout()` - deconnexion (appel API + reset state)
- `forgotPassword(email)` - demande de reset (toujours retourne true pour eviter la divulgation)
- `resetPassword(token, password)` - reset du mot de passe
- `checkAuth()` - restauration de session via `GET /auth/me`
- `fetchMe()` - charge le profil via `GET /users/me`
- `updateMe(payload)` - met a jour le profil via `PUT /users/me`

**Interface User** : `{ id, email, name, role: 'admin' | 'user', created_at?, updated_at? }`

### `projects` (Setup store)

**State** : `items: Project[]`, `pagination: Pagination | null`, `isLoading`, `error`

**Actions** : `fetchProjects(params?)`, `createProject(payload)`, `updateProject(id, payload)`, `reset()`

**Interface Project** : `{ id, name, description?, repo_url?, git_provider?, git_token_env?, agent_runtime?, default_model?, owner_id, circuit_breaker_active?, created_at, updated_at }`

### `stories` (Setup store)

**State** : `items: Story[]`, `selectedStoryId`, `filters: { status, search }`, `isLoading`, `error`

**Getters** : `filteredStories` (filtre par statut et recherche textuelle), `selectedStory`

**Actions** : `fetchStoriesByEpic(projectId, epicId)`, `updateStory(projectId, storyId, fields)`, `createStory(projectId, fields)`, `setSelectedStory(id)`, `setFilters(filters)`, `clearError()`, `reset()`

**Interface Story** : `{ id, epic_id, project_id, key, title, status, objective?, acceptance_criteria?, target_files?, depends_on?, scope?, latest_run?, created_at, updated_at }`

### `runs` (Setup store)

**State** : `items`, `current`, `isLoading`, `isPausing`, `isResuming`, `isRetrying`, `isCancelling`, `circuitBreakerActive`

**Actions** :
- `pauseRun(projectId, runId)` - pause d'un run en cours
- `resumeRun(projectId, runId)` - reprise d'un run en pause
- `retryStep(runId, stepId)` - retry d'un step echoue
- `cancelRun(projectId, runId)` - annulation d'un run
- `updateRunStatus(runId, status)` - mise a jour locale du statut
- `handleSSEEvent(event)` - dispatche les evenements SSE vers le HITL store et gere les evenements circuit breaker

### `agents` (Setup store)

**State** : `items: Agent[]`, `pagination`, `isLoading`, `error`

**Actions** : `fetchAgents(projectId, params?)`, `createAgent(projectId, params)`, `updateAgent(projectId, agentId, params)`, `deleteAgent(projectId, agentId)`, `clearError()`, `reset()`

**Interface Agent** : `{ id, name, model, image, template_content, scope: 'global' | 'project', project_id?, created_at, updated_at }`

### `epics` (Setup store)

**State** : `items: Epic[]`, `isLoading`, `error`

**Actions** : `fetchEpics(projectId)`, `clearError()`, `reset()`

Le type `Epic` est genere depuis le schema OpenAPI (`components['schemas']['Epic']`).

### `epicRun` (Setup store)

**State** : `epicRun: EpicRun | null`, `isLoading`, `error`

**Getters** : `completedCount`, `totalCount`, `progressPercent`, `failedStories`

**Actions** : `fetchEpicRun(projectId, epicRunId)`, `handleSSEEvent(eventName, data)`, `reset()`

Types `EpicRun` et `EpicRunStory` generes depuis le schema OpenAPI.

### `pipelineConfig` (Setup store)

**State** : `config: PipelineConfig | null`, `isLoading`, `error`, `isDirty`, `isSaving`

**Getters** : `groups` (depuis config.groups), `steps` (flat across all groups)

**Actions** (gestion de groupes) : `fetchConfig(projectId)`, `updateGroups(newGroups)`, `addGroup(name?)`, `removeGroup(groupId)`, `renameGroup(groupId, name)`, `reorderGroups(from, to)`, `saveConfig(projectId)`

**Actions** (gestion de steps) : `addStep(step)`, `removeStep(index)`, `reorderSteps(from, to)`, `updateStep(index, step)`, `addStepToGroup(groupId, step)`, `removeStepFromGroup(groupId, stepId)`, `updateStepInGroup(groupId, stepId, step)`, `reorderStepsInGroup(groupId, from, to)`

Types generes depuis le schema OpenAPI : `PipelineConfig`, `PipelineStep`, `PipelineGroup`, `RetryPolicy`.

### `hitl` (Setup store)

**State** : `pendingItems: HITLPendingItem[]`, `isLoading`, `error`

**Getters** : `pendingCount`

**Actions** : `fetchPending()` (GET /hitl-requests?status=pending), `handlePendingEvent(payload)` (dedup par hitlRequestId), `handleResolvedEvent(hitlRequestId)` (retire de la liste)

**Interface HITLPendingItem** : `{ hitlRequestId, runId, stepId, projectId, projectName, storyKey, storyTitle, prUrl, pendingSince }`

### `approvals` (Setup store)

**State** : `pendingApprovals: PendingApproval[]`

**Actions** : `addPendingApproval(approval)`, `removePendingApproval(hitlRequestId)`, `handleHITLPendingEvent(payload)`

Store complementaire a `hitl`, utilise pour le dispatch SSE depuis le runs store.

### `users` (Setup store)

**State** : `users: User[]`, `pagination: Pagination`, `isLoading`

**Actions** : `fetchUsers(params?)`, `createUser(payload)`, `updateUser(id, payload)`, `deleteUser(id)`

### `layout` (Setup store)

**State** : `sidebarCollapsed: boolean` (persiste dans localStorage)

**Actions** : `toggleSidebar()`

---

## 6. Utilitaires

Fonctions pures dans `src/utils/` :

| Fichier | Fonction(s) | Description |
|---|---|---|
| `apiError.ts` | `getApiErrorMessage(error, fallback)` | Extrait le message d'erreur de la reponse OpenAPI, fallback sur un message par defaut |
| `runStatus.ts` | `runStatusSeverity` (map), `statusSeverity(status)` | Mapping statut -> severite PrimeVue Tag (pending=secondary, running=info, paused=warn, completed=success, failed=danger, cancelled=warn) |
| `formatCost.ts` | `formatCostUSD(value)`, `formatTokenCount(count)` | Formatage monnaie USD (jusqu'a 5 decimales) et nombre de tokens avec separateurs de milliers |
| `formatDate.ts` | `formatRelativeDate(dateStr)`, `formatDate(dateStr)` | Dates relatives ("3 days ago") et formatees ("Feb 15, 2026") via date-fns |
| `formatDuration.ts` | `formatDuration(startedAt?, completedAt?)` | Duree en mm:ss entre deux timestamps ISO. Utilise Date.now() si completedAt absent |
| `formatLogLine.ts` | `formatLogLine(raw, timestamp)` | Convertit une ligne de log avec codes ANSI en HTML avec prefix HH:MM:SS |
| `renderMarkdown.ts` | `renderMarkdown(input)` | Rend du markdown en HTML securise (marked + DOMPurify) |
| `pipelineStageUtils.ts` | `groupStepsByStage(groups, steps)` | Regroupe les steps d'un run dans les stages du pipeline config via l'offset cumulatif |
| `maskUrl.ts` | `maskUrl(url)` | Masque une URL en montrant uniquement les 6 derniers caracteres |
| `models.ts` | `LLM_MODEL_OPTIONS` | Liste des modeles LLM disponibles (Opus 4, Sonnet 4, Haiku 3.5) pour la configuration des agents |

---

## 7. Configuration et build

### Variables d'environnement

Le frontend utilise les variables d'environnement Vite :
- `BASE_URL` - Chemin de base pour le routeur (defaut: `/`)
- Le proxy dev renvoie `/api/v1` vers `http://localhost:8080` (configure dans `vite.config.ts`)

### Scripts npm

| Script | Description |
|---|---|
| `npm run dev` | Serveur de dev Vite (port 5173) |
| `npm run build` | Build production (type-check + vite build en parallele) |
| `npm run preview` | Preview du build de production |
| `npm run test:unit` | Tests unitaires Vitest |
| `npm run test:e2e` | Tests E2E Playwright |
| `npm run test:e2e:real` | Tests E2E contre la stack reelle |
| `npm run type-check` | Verification TypeScript (`vue-tsc --build`) |
| `npm run lint` | ESLint avec auto-fix et cache |
| `npm run lint:oxlint` | Oxlint avec auto-fix (linter Rust rapide) |
| `npm run format` | Prettier |
| `npm run generate-api` / `generate:api` | Generation des types TypeScript depuis `api/openapi.yaml` |

### TypeScript

- `tsconfig.json` : references vers `tsconfig.app.json`, `tsconfig.node.json`, `tsconfig.vitest.json`
- `tsconfig.app.json` : etend `@vue/tsconfig/tsconfig.dom.json`, alias `@/*` -> `./src/*`
- Strict mode active

### Linting

- ESLint avec config Vue/TypeScript + Vitest plugin + oxlint + Prettier
- Oxlint comme linter rapide complementaire (config `.oxlintrc.json`)
- Prettier pour le formatage (`.prettierrc.json`)

### Build optimise

- Monaco Editor isole dans un chunk separe pour eviter de bloater le bundle principal
- Lazy-loading des routes secondaires (agent-create, epic-dag, epic-run, project-settings, etc.)

### Tests

- **Unitaires** (Vitest) : co-localises dans `__tests__/` a cote des fichiers source. Cibles : composables, stores, utils, schemas zod. Pattern : `describe/it/expect` avec `@vue/test-utils` pour les composants.
- **E2E** (Playwright) : dans `frontend/e2e/tests/` et `frontend/e2e/real-tests/`. Config standard (`playwright.config.ts`) et config pour la stack reelle (`playwright.e2e-real.config.ts`).
