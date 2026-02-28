# Backend Core Pipeline Documentation

Documentation technique du coeur du backend hopeitworks : Pipeline, Run, DAG, et Epic Runs.

**Derniere mise a jour** : 2026-02-26

---

## Table des matieres

1. [Vue d'ensemble architecture backend](#1-vue-densemble-architecture-backend)
   - [Structure hexagonale](#structure-hexagonale)
   - [Entry point](#entry-point-cmdapimain-go)
   - [Dependency injection (Wire)](#dependency-injection-wire)
   - [Middleware stack](#middleware-stack)
   - [Convention de routage](#convention-de-routage)
2. [Modeles de domaine](#2-modeles-de-domaine)
   - [Run et RunStep](#run-et-runstep)
   - [PipelineConfig](#pipelineconfig)
   - [DAG](#dag)
   - [EpicRun](#epicrun)
   - [RunContext](#runcontext)
   - [TemplateContext](#templatecontext)
   - [Action](#action)
   - [Container](#container)
   - [Modeles de support](#modeles-de-support)
3. [Services](#3-services)
   - [PipelineExecutor](#pipelineexecutor)
   - [ParallelGroupExecutor](#parallelgroupexecutor)
   - [RunService](#runservice)
   - [EpicRunService](#epicrunservice)
   - [SchedulerService](#schedulerservice)
   - [PipelineConfigService](#pipelineconfigservice)
   - [ActionRegistry](#actionregistry)
   - [CircuitBreakerService](#circuitbreakerservice)
   - [TimeoutEnforcer](#timeoutenforcer)
   - [OrphanCleaner](#orphancleaner)
4. [Ports (Interfaces)](#4-ports-interfaces)
   - [RunRepository](#runrepository)
   - [PipelineConfigRepository](#pipelineconfigrepository)
   - [EpicRunRepository](#epicrunrepository)
   - [ContainerManager](#containermanager)
   - [JobQueue](#jobqueue)
   - [CommandRunner](#commandrunner)
   - [ActionRegistry (port)](#actionregistry-port)
   - [LogStreamer](#logstreamer)
   - [TemplateRenderer](#templaterenderer)
   - [EventPublisher](#eventpublisher)
   - [GitProvider et GitProviderFactory](#gitprovider-et-gitproviderfactory)
5. [Actions (Adapter layer)](#5-actions-adapter-layer)
   - [AgentRunAction](#agentrunaction)
   - [GitBranchAction](#gitbranchaction)
   - [GitPRAction](#gitpraction)
   - [HITLGateAction](#hitlgateaction)
   - [HumanAction](#humanaction)
   - [CIPollAction](#cipollaction)
   - [NotificationAction](#notificationaction)
   - [IncrementalRetryAction](#incrementalretryaction)
6. [Tests](#6-tests)
   - [Patterns de test](#patterns-de-test)
   - [Mocks et fixtures](#mocks-et-fixtures)
   - [Couverture des cas edge](#couverture-des-cas-edge)
7. [Flux de donnees complets](#7-flux-de-donnees-complets)
   - [Single Story Run](#single-story-run)
   - [Epic Run (multi-story DAG)](#epic-run-multi-story-dag)
   - [Retry flow](#retry-flow)
   - [HITL Suspension flow](#hitl-suspension-flow)

---

## 1. Vue d'ensemble architecture backend

### Structure hexagonale

Le backend suit une architecture hexagonale stricte avec une separation nette entre domaine, ports et adapteurs :

```
backend/
├── cmd/api/
│   ├── main.go              # Entry point, wiring manuel
│   ├── wire.go              # DI wiring (go-wire, build tag wireinject)
│   └── providers.go         # Wire provider sets
├── internal/
│   ├── domain/
│   │   ├── model/           # Entites pures (zero import externe)
│   │   ├── port/            # Interfaces (contrats)
│   │   └── service/         # Logique metier
│   ├── adapter/
│   │   ├── action/          # Implementations Action (agent_run, git_branch, etc.)
│   │   ├── docker/          # ContainerManager implementation
│   │   ├── git/             # GitProvider + GitProviderFactory
│   │   ├── postgres/        # Tous les Repository impls (sqlc)
│   │   ├── river/           # JobQueue implementation (River)
│   │   ├── handlebars/      # TemplateRenderer implementation
│   │   ├── discord/         # Notifier (webhook Discord)
│   │   ├── webhook/         # Notifier generique
│   │   └── smtp/            # EmailSender
│   ├── api/
│   │   ├── handler/         # HTTP handlers (oapi-codegen generated interface)
│   │   └── middleware/      # Auth JWT, RBAC
│   └── config/              # Chargement config YAML + env
└── pkg/
    ├── errors/              # DomainError + categories
    ├── log/                 # slog helpers
    ├── exec/                # CommandRunner
    └── config/              # Structs de config
```

**Regle d'import stricte** :

```
handler -> service -> port <- adapter
```

Les services dependent uniquement des ports (interfaces). Les adapteurs implementent les ports. Aucune logique metier dans les handlers ou adapteurs. Le modele de domaine n'a zero dependance externe.

### Entry point (`cmd/api/main.go`)

Le `main.go` assemble manuellement toutes les dependances (le wiring Wire genere n'est pas utilise en pratique, le main fait tout a la main). L'ordre d'initialisation est :

1. **Config** : `internalconfig.Load("config.yaml")` + env overrides
2. **Logger** : `slog` structure JSON
3. **Database** : pool `pgx/v5` + auto-migration via `golang-migrate`
4. **Event Bus** : connexion LISTEN/NOTIFY Postgres dediee
5. **Repositories** : tous instancies depuis les queries sqlc
6. **Services** : AuthService, ProjectService, StoryService, etc.
7. **Actions** : enregistrement dans `ActionRegistry` (agent_run, git_branch, git_pr, ci_poll, hitl_gate, human, notification)
8. **Pipeline Executor** : wired avec ActionRegistry, EventPublisher, CircuitBreaker
9. **River Job Queue** : workers enregistres, client demarre en goroutine
10. **RunService** : cree avec tous les repos + job queue
11. **Background services** : OrphanCleaner (startup), TimeoutEnforcer (goroutine), token cleanup
12. **Epic Run orchestration** : ParallelGroupExecutor + EpicRunService
13. **HTTP Router** : chi avec middlewares + handler registration
14. **Graceful shutdown** : signal handling SIGTERM/SIGINT

### Dependency Injection (Wire)

Le fichier `wire.go` definit les provider sets Wire mais en pratique le wiring est fait manuellement dans `main.go` pour plus de controle sur l'ordre d'initialisation et la gestion conditionnelle (ex: Docker peut etre absent).

Provider sets definis dans `providers.go` :

| Set | Packages |
|-----|----------|
| `ConfigSet` | `internalconfig.Load` |
| `LogSet` | `pkglog.New` |
| `PostgresSet` | `postgres.NewPool` |
| `RouterSet` | `api.NewRouter` |

### Middleware stack

L'ordre des middlewares chi est :

1. `chimiddleware.Logger` -- log des requetes HTTP
2. `chimiddleware.Recoverer` -- recovery des panics
3. `chimiddleware.RequestID` -- injection request ID unique
4. `authmw.Auth(authService, blacklistRepo)` -- validation JWT

Le middleware Auth :
- **Paths publics** : `/healthz`, `/api/v1/auth/register`, `/api/v1/auth/login`, `/api/v1/auth/forgot-password`, `/api/v1/auth/reset-password`
- Lit le token depuis le cookie `token`
- Valide le JWT via `AuthService.ValidateToken()`
- Verifie le blacklist (tokens revoques par logout)
- Injecte `user_id` et `user_role` dans le context Go

### Convention de routage

Les routes sont generees par oapi-codegen depuis `api/openapi.yaml` :

```go
handler.HandlerFromMuxWithBaseURL(server, r, "/api/v1")
```

Le `Server` struct implemente l'interface generee `ServerInterface` en deleguant a des handlers specifiques :

| Endpoint pattern | Handler | Method |
|-----------------|---------|--------|
| `POST /projects/{projectId}/stories/{storyId}/runs` | `RunHandler.LaunchRun` | Lancement d'un run |
| `GET /runs/{runId}` | `RunHandler.GetRun` | Detail run + steps |
| `POST /projects/{projectId}/runs/{runId}/pause` | `RunHandler.PauseRun` | Pause run |
| `POST /projects/{projectId}/runs/{runId}/resume` | `RunHandler.ResumeRun` | Resume run |
| `POST /projects/{projectId}/runs/{runId}/cancel` | `RunHandler.CancelRun` | Cancel run |
| `POST /runs/{runId}/steps/{stepId}/retry` | `RunHandler.RetryStep` | Retry step echoue |
| `GET /projects/{projectId}/pipeline-config` | `PipelineConfigHandler.GetPipelineConfig` | Config pipeline |
| `PUT /projects/{projectId}/pipeline-config` | `PipelineConfigHandler.UpdatePipelineConfig` | Mise a jour config |
| `POST /projects/{projectId}/epics/{epicId}/runs` | `EpicRunHandler.LaunchEpicRun` | Lancement epic run |
| `GET /projects/{projectId}/epic-runs/{epicRunId}` | `EpicRunHandler.GetEpicRun` | Detail epic run |

Routes additionnelles montees manuellement (non generees) :
- `GET /api/v1/events/stream` -- SSE pour events temps reel
- `GET/POST/DELETE /api/v1/projects/{id}/users` -- gestion membres projet

---

## 2. Modeles de domaine

Fichier source : `backend/internal/domain/model/`

### Run et RunStep

**Fichier** : `run.go`

#### Run

```go
type Run struct {
    ID                     uuid.UUID
    ProjectID              uuid.UUID
    StoryID                uuid.UUID
    StoryKey               string              // optionnel, peuple par JOIN
    Status                 RunStatus
    PipelineConfigSnapshot json.RawMessage     // snapshot JSON de la config au moment du lancement
    Metadata               map[string]interface{} // donnees inter-steps (branch_name, model, template_content...)
    StartedAt              *time.Time
    CompletedAt            *time.Time
    PausedAt               *time.Time
    ErrorMessage           *string
    CreatedAt              time.Time
    UpdatedAt              time.Time
    Steps                  []RunStep
    Progress               int                 // calcule, non persiste (0-100%)
}
```

**Statuts Run** (`RunStatus`) :

| Statut | Description |
|--------|-------------|
| `pending` | Cree, en attente d'execution |
| `running` | En cours d'execution |
| `paused` | Mis en pause par l'utilisateur |
| `completed` | Tous les steps ont reussi |
| `failed` | Un step a echoue |
| `cancelled` | Annule par l'utilisateur |

**Transitions valides** :

```
pending   -> running, cancelled
running   -> paused, completed, failed, cancelled
paused    -> running, cancelled
failed    -> running  (retry)
completed -> (terminal)
cancelled -> (terminal)
```

**Methode** `ComputeProgress(steps []RunStep) int` : Calcule le pourcentage de progression (nombre de steps `completed` / total steps * 100).

#### RunStep

```go
type RunStep struct {
    ID           uuid.UUID
    RunID        uuid.UUID
    StepName     string
    StepOrder    int           // ordre sequentiel d'execution
    Action       string        // type d'action (agent_run, git_branch, etc.)
    Status       StepStatus
    StartedAt    *time.Time
    CompletedAt  *time.Time
    ErrorMessage *string
    ContainerID  *string       // ID container Docker (pour agent_run)
    LogTail      *string       // dernieres lignes de log en cas d'erreur
    RetryCount   int           // nombre de retries effectues
    RetryType    *string       // "incremental" ou "full"
    ParentStepID *uuid.UUID    // ID du step parent (pour retries)
    Config       map[string]string // config step-specific, transient (non persiste)
    CreatedAt    time.Time
}
```

**Statuts Step** (`StepStatus`) :

| Statut | Description |
|--------|-------------|
| `pending` | En attente |
| `running` | En cours |
| `completed` | Termine avec succes |
| `failed` | Echoue |
| `cancelled` | Annule |
| `waiting_approval` | Suspendu, en attente d'approbation humaine (HITL) |

**Transitions valides** :

```
pending           -> running, cancelled
running           -> completed, failed, cancelled, waiting_approval
waiting_approval  -> running, completed, failed, cancelled
completed         -> (terminal)
failed            -> (terminal)
cancelled         -> (terminal)
```

**Validation** : `ValidateRunTransition()` et `ValidateStepTransition()` verifient les transitions contre les maps de transitions valides et retournent un `DomainError` avec code `INVALID_STATE_TRANSITION`.

### PipelineConfig

**Fichier** : `pipeline_config.go`

```go
type PipelineConfig struct {
    ID         uuid.UUID
    ProjectID  uuid.UUID
    ConfigYAML string    // contenu YAML brut
    Version    int
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

#### Structure YAML parsee

```go
type PipelineConfigYAML struct {
    Groups []PipelineGroup
}

type PipelineGroup struct {
    ID    string
    Name  string
    Steps []PipelineStep
}

type PipelineStep struct {
    ID          string
    Name        string
    ActionType  string            // agent_run, git_branch, git_pr, ci_poll, hitl_gate, human, notification
    Description string
    AgentID     string            // UUID de l'agent (requis pour agent_run)
    Model       string            // modele AI (ex: claude-opus-4-6)
    AutoApprove bool
    RetryPolicy RetryPolicy
    Config      map[string]string // config specifique au step
}

type RetryPolicy struct {
    MaxRetries int
    RetryType  string // none, on-failure, always
}
```

**Backward compatibility** : `ParsePipelineConfigYAML()` supporte deux formats :
- **Nouveau** : `groups:` avec structure hierarchique
- **Legacy** : `steps:` plat, auto-wrappe dans un groupe "Default"

**Methode** `FlatSteps()` : retourne tous les steps de tous les groupes, aplatis dans l'ordre.

**Types d'action valides** :

```go
var ValidActionTypes = map[string]bool{
    "agent_run":    true,
    "git_branch":   true,
    "git_pr":       true,
    "notification": true,
    "human":        true,
    "ci_poll":      true,
    "hitl_gate":    true,
    // Legacy (backward compat)
    "implement": true,
    "review":    true,
    "merge":     true,
    "test":      true,
    "custom":    true,
}
```

### DAG

**Fichier** : `dag.go`

```go
type DAGResult struct {
    Groups [][]Story
}
```

`DAGResult` represente le resultat d'un tri topologique sur les stories. Chaque `Groups[i]` est une couche d'execution : toutes les stories du groupe `i` peuvent s'executer en parallele, et toutes doivent completer avant que le groupe `i+1` ne demarre.

### EpicRun

**Fichier** : `epic_run.go`

```go
type EpicRun struct {
    ID          uuid.UUID
    ProjectID   uuid.UUID
    EpicID      uuid.UUID
    Status      EpicRunStatus
    CreatedAt   time.Time
    CompletedAt *time.Time
    Stories     []EpicRunStory
}

type EpicRunStory struct {
    EpicRunID  uuid.UUID
    StoryID    uuid.UUID
    RunID      *uuid.UUID    // ID du Run cree pour cette story
    GroupIndex int           // couche DAG
    Status     string        // pending, running, completed, failed
}
```

**Statuts EpicRun** :

| Statut | Description |
|--------|-------------|
| `pending` | Cree mais pas encore demarre |
| `running` | Execution active |
| `completed` | Toutes les stories ont reussi |
| `failed` | Une story a echoue |
| `paused` | Mis en pause |

**Transitions** : `pending -> running`, `running -> completed, failed, paused`.

### RunContext

**Fichier** : `run_context.go`

```go
type RunContext struct {
    Run       *Run
    RunStep   *RunStep
    ProjectID uuid.UUID
    StoryID   uuid.UUID
    Metadata  map[string]any    // donnees partagees entre steps
}
```

Le `RunContext` est le vehicule principal de donnees entre les actions d'un pipeline. Le champ `Metadata` est un dictionnaire mutable qui permet aux steps de communiquer :

| Cle Metadata | Producteur | Consommateur | Description |
|-------------|-----------|-------------|-------------|
| `branch_name` | `git_branch` / `LaunchRun` | `agent_run`, `git_pr`, `notification` | Nom de la branche |
| `pr_url` | `git_pr` | `ci_poll`, `hitl_gate`, `notification` | URL de la PR |
| `model` | Executor (per-step) | `agent_run` | Modele AI a utiliser |
| `template_content` | Executor (per-step) | `agent_run` | Template Handlebars du prompt |
| `agent_id` | Executor (per-step) | `agent_run` | UUID de l'agent |
| `agent_image` | Executor (per-step) | `agent_run` | Image Docker de l'agent |
| `error_context` | `incremental_retry` | `agent_run` | Erreur du step precedent (retry) |
| `log_tail` | `incremental_retry` | `agent_run` | Derniers logs du step echoue |

### TemplateContext

**Fichier** : `template_context.go`

```go
type TemplateContext struct {
    StoryKey           string
    StoryTitle         string
    StoryObjective     string
    TargetFiles        []string
    AcceptanceCriteria string
    ErrorContext       string    // pour retry
    LogTail            string    // pour retry
    DiffContent        string    // pour review/merge
    BranchName         string
    RepoURL            string
}
```

Variables disponibles dans les templates Handlebars pour les prompts agents. Le `TemplateRenderer` (port) recoit ce contexte pour rendre les templates.

### Action

**Fichier** : `action.go`

```go
type Action interface {
    Name() string
    Execute(ctx context.Context, runCtx *RunContext) error
}
```

Interface centrale du systeme de pipeline. Chaque type de step (agent_run, git_branch, etc.) est une implementation concrete de `Action`. Le `Name()` correspond au champ `action_type` dans la config pipeline YAML.

### Container

**Fichier** : `container.go`

```go
type ContainerOpts struct {
    Image       string
    Env         []string           // KEY=VALUE
    NetworkName string
    Labels      map[string]string  // managed_by, run_id, step_id
    Memory      int64              // bytes (0 = illimite)
    CPUs        float64            // (0 = illimite)
}
```

Configuration pour creer un container agent Docker. Les labels standards (`managed_by`, `run_id`, `step_id`) sont utilises par le TimeoutEnforcer et l'OrphanCleaner pour identifier et gerer les containers.

### Modeles de support

**Event** (`event.go`) :
```go
type Event struct {
    ID, ProjectID, EntityID uuid.UUID
    EntityType, Action      string    // ex: "run"."started", "step"."completed"
    Payload                 json.RawMessage
    CreatedAt               time.Time
}
```
Methode `EventName()` retourne la notation pointee `entity_type.action`.

**Story** (`story.go`) :
Champs cles pour le pipeline : `DependsOn []string` (cles de dependance pour le DAG), `TargetFiles []string` (fichiers cibles pour les conflits implicites), `Status string` (backlog/running/done/failed).

**HITLRequest** (`hitl.go`) :
Enregistrement d'une gate HITL. `GateType` peut etre "approval" (hitl_gate) ou "human" (human action). `DiffContent` contient le diff PR (optionnel).

**LogEvent** (`log_event.go`) :
Evenement de log d'un container agent. Supporte le parsing NDJSON. Le champ `Type` = "cost" declenche le tracking des couts (InputTokens, OutputTokens, Model).

**Validation** (`validation.go`) :
Constantes de validation partagees : `MaxNameLength=255`, `MaxStoryKeyLength=50`, etc.

---

## 3. Services

Fichier source : `backend/internal/domain/service/`

### PipelineExecutor

**Fichier** : `pipeline_executor.go`

**But** : Orchestrer l'execution sequentielle des steps d'un pipeline run.

**Structure** :
```go
type PipelineExecutor struct {
    runRepo        port.RunRepository
    storyRepo      port.StoryRepository
    actionReg      port.ActionRegistry
    eventPub       port.EventPublisher
    circuitBreaker *CircuitBreakerService
    logger         *slog.Logger
}
```

**Methode principale** : `ExecuteRun(ctx context.Context, runID uuid.UUID) error`

**Flux d'execution** :

1. **Verification** : Recupere le run depuis le repository
2. **Circuit breaker** : Verifie que le circuit breaker n'est pas ouvert pour le projet. Si ouvert, marque le run `failed` et retourne
3. **Liste des steps** : Recupere et trie les steps par `step_order`
4. **Transition run** : `pending -> running`, publie `run.started`
5. **Transition story** : Met la story a `running` (best-effort)
6. **Initialisation metadata** : Merge les metadata persistees du run dans le dictionnaire partage
7. **Boucle d'execution** : Pour chaque step non-complete :
   - Verifie l'annulation (context cancelled)
   - Verifie la pause (re-lecture du statut depuis la DB)
   - Execute le step via `executeStep()`
   - Si `errStepSuspended` (HITL gate), arrete proprement (return nil)
   - Si erreur, appelle `handleStepFailure()` et record dans circuit breaker
8. **Completion** : Tous les steps OK -> `run.completed`, story -> `done`, reset circuit breaker

**Methode** `executeStep()` :
1. Transition step `pending -> running`, publie `step.started`
2. Lookup de l'action dans `ActionRegistry`
3. Extraction de la config step-specific depuis le snapshot pipeline
4. Construction du `RunContext` avec injection per-step de : `template_content`, `model`, `agent_id`, `agent_image` (cles prefixees `step_<order>_*` dans metadata)
5. Execution de l'action
6. Re-fetch du step pour detecter suspension (`waiting_approval`)
7. Transition step `running -> completed`, publie `step.completed`

**Methode** `handleStepFailure()` :
- Marque step et run comme `failed`
- Publie `step.failed` et `run.failed`
- Met la story a `failed`

**Methode** `handleCancellation()` :
- Utilise `context.Background()` (context original annule)
- Marque step et run comme `cancelled`
- Publie les events correspondants

**Sentinel errors** :
- `ErrRunPaused` : retourne quand le run est detecte en pause
- `errStepSuspended` : erreur interne quand un step passe a `waiting_approval`

### ParallelGroupExecutor

**Fichier** : `parallel_group_executor.go`

**But** : Executer les couches DAG d'un epic run. Couches sequentielles, stories paralleles au sein de chaque couche.

**Structure** :
```go
type ParallelGroupExecutor struct {
    epicRunRepo port.EpicRunRepository
    runSvc      *RunService
    executor    *PipelineExecutor
    eventPub    port.EventPublisher
    logger      *slog.Logger
}
```

**Methode principale** : `Execute(ctx, epicRun, dag DAGResult) error`

**Flux** :

1. Transition epic run `pending -> running`
2. Pour chaque couche DAG (`dag.Groups[i]`) :
   - Publie `epic_run_group.started`
   - Lance toutes les stories de la couche en parallele via `errgroup`
   - Chaque story : `runStory()` -> `LaunchRun()` + `ExecuteRun()` direct
   - Si une story echoue : **fail-fast**, marque epic run `failed`, arrete
3. Toutes les couches OK : marque epic run `completed`

**Methode** `runStory()` :
1. Cree un Run via `RunService.LaunchRun()`
2. Met a jour `EpicRunStory.status = running`
3. Execute le run directement via `PipelineExecutor.ExecuteRun()` (pas de job queue, deja async)
4. Met a jour le statut epic_run_story selon le resultat

### RunService

**Fichier** : `run_service.go`

**But** : Logique metier pour toutes les operations sur les Runs. Point d'entree principal pour lancer, pauser, reprendre, annuler les runs.

**Structure** :
```go
type RunService struct {
    runRepo            port.RunRepository
    projectRepo        port.ProjectRepository
    storyRepo          port.StoryRepository
    pipelineConfigRepo port.PipelineConfigRepository
    jobQueue           port.JobQueue
    eventPub           port.EventPublisher
    containerMgr       port.ContainerManager
    agentRepo          port.AgentRepository
}
```

**Methodes principales** :

#### `LaunchRun(ctx, projectID, storyID) (*Run, error)`

C'est LA methode centrale pour demarrer un pipeline. Flux complet :

1. **Validation story** : Existe, appartient au projet, pas `done`, pas de run actif
2. **Fetch pipeline config** : Recupere le YAML du projet
3. **Parse YAML** : Backward-compatible (groups ou steps legacy)
4. **Validation agents** : Pour les steps `agent_run`, `agent_id` est requis. L'agent est resolve et ses attributs (model, image, template_content) sont snapshots dans les metadata du run
5. **Snapshot** : Serialise la config parsee en JSON
6. **Creation Run** : Avec metadata (branch_name, per-step model/agent_id/agent_image/template_content)
7. **Creation RunSteps** : Un par step dans l'ordre
8. **Enqueue** : Job River `execute_run` pour execution asynchrone
9. **Retour** : Run avec steps, status `pending`

#### `PauseRun(ctx, projectID, runID)`

Transition `running -> paused`. Le step en cours continue jusqu'a completion mais aucun nouveau step n'est lance (le PipelineExecutor detecte la pause avant chaque step).

#### `ResumeRun(ctx, projectID, runID)`

Transition `paused -> running`, re-enqueue un job `execute_run`. Le PipelineExecutor skip les steps deja `completed`.

#### `CancelRun(ctx, projectID, runID)`

1. Stop les containers running via `ContainerManager.Stop()`
2. Marque tous les steps pending/running comme `cancelled`
3. Transition run `-> cancelled`

#### `RetryStep(ctx, runID, stepID)`

1. Valide que le step est `failed`
2. Verifie les limites de retry (defaut 3, configurable dans RetryPolicy)
3. Determine le type : `incremental` (retries 1-2) ou `full` (retry 3+)
4. Cree un nouveau RunStep avec `ParentStepID` et `RetryCount`
5. Transition run `failed -> running`
6. Enqueue job `execute_run` pour reprendre

#### `CreateRun(ctx, params)` (legacy)

Creation manuelle d'un run avec config JSON fournie. Utilise principalement pour les tests.

#### Methodes de lecture

- `GetRun()` : Run + steps + progress calcule
- `ListRunsByProject()` / `ListRunsByStory()` : Pagination, enrichissement avec steps

### EpicRunService

**Fichier** : `epic_run_service.go`

**But** : Orchestrer l'execution de toutes les stories d'un epic via le DAG.

**Structure** :
```go
type EpicRunService struct {
    epicRunRepo port.EpicRunRepository
    storyRepo   port.StoryRepository
    epicRepo    port.EpicRepository
    scheduler   *SchedulerService
    executor    *ParallelGroupExecutor
    eventPub    port.EventPublisher
    logger      *slog.Logger
}
```

**Methode** `LaunchEpicRun(ctx, projectID, epicID)` :

1. Valide l'epic (existe, appartient au projet)
2. Fetch toutes les stories de l'epic
3. Calcule le DAG via `SchedulerService.BuildDAG()`
4. Cree l'enregistrement `EpicRun` (status `pending`)
5. Insere les `EpicRunStory` avec les `group_index` corrects
6. **Lance le ParallelGroupExecutor dans une goroutine detachee** (`context.WithoutCancel()`)
7. Retourne immediatement (202 Accepted)

**Methode** `GetEpicRun()` : Recupere l'epic run avec toutes ses stories.

### SchedulerService

**Fichier** : `scheduler_service.go`

**But** : Calculer l'ordonnancement DAG des stories. Service pur, stateless, sans dependance I/O.

**Structure** : `type SchedulerService struct{}` (vide, aucune dependance)

**Methode** `BuildDAG(stories []Story) (DAGResult, error)` :

**Algorithme** : Kahn's algorithm (tri topologique couche par couche)

1. **Indexation** : Stories par cle
2. **Construction du graphe** :
   - **Aretes explicites** : A partir de `Story.DependsOn[]` (cles inconnues ignorees)
   - **Aretes implicites** : Conflit de fichiers (`Story.TargetFiles`). Si plusieurs stories modifient le meme fichier, des aretes sont ajoutees dans l'ordre lexicographique des cles pour serialiser l'execution
3. **Tri topologique** :
   - Collecte les noeuds avec in-degree 0 (aucune dependance non satisfaite)
   - Les groupe dans une couche (tri lexicographique pour determinisme)
   - Decremente l'in-degree des dependants
   - Repete jusqu'a ce que tous les noeuds soient traites
4. **Detection de cycle** : Si aucun noeud a in-degree 0 et des noeuds restent -> `DAG_CYCLE_DETECTED`

**Exemples de resultats** :

| Scenario | Input | Output |
|----------|-------|--------|
| 2 stories independantes | S-01, S-02 (no deps) | `[[S-01, S-02]]` |
| Chaine lineaire A->B->C | S-01, S-02(dep S-01), S-03(dep S-02) | `[[S-01], [S-02], [S-03]]` |
| Diamant | S-01, S-02(dep S-01), S-03(dep S-01), S-04(dep S-02,S-03) | `[[S-01], [S-02, S-03], [S-04]]` |
| Conflit fichier | S-01(shared.go), S-02(shared.go) | `[[S-01], [S-02]]` (ordre lexicographique) |
| Cycle | S-01(dep S-02), S-02(dep S-01) | `DAG_CYCLE_DETECTED` error |

### PipelineConfigService

**Fichier** : `pipeline_config_service.go`

**But** : CRUD sur la configuration pipeline d'un projet.

**Methodes** :
- `GetByProjectID()` : Recupere la config
- `Upsert()` : Valide et sauvegarde la config YAML
- `SeedDefault()` : Cree la config par defaut pour un nouveau projet

**Validation** (`validatePipelineConfigYAML()`) :
1. Parse YAML (backward-compatible groups/steps)
2. Au moins un groupe
3. Chaque groupe : nom non-vide, au moins un step
4. Chaque step : nom non-vide, `action_type` non-vide et valide

**Config par defaut** (`DefaultPipelineConfigYAML`) :

```yaml
groups:
  - id: setup       # git_branch (Create Branch)
  - id: development # agent_run (Implement Story)
  - id: review      # agent_run (Code Review)
  - id: merge       # git_pr (Create & Merge PR)
  - id: delivery    # ci_poll + notification
```

### ActionRegistry

**Fichier** : `action_registry.go`

**But** : Registre thread-safe des implementations `Action`, lookup par nom.

```go
type InMemoryActionRegistry struct {
    mu      sync.RWMutex
    actions map[string]model.Action
}
```

**Methodes** :
- `Register(action)` : Enregistre (ecrase si meme nom)
- `RegisterAlias(alias, action)` : Enregistre sous un alias (ex: "implement" -> AgentRunAction)
- `Get(name) (Action, error)` : Lookup, retourne `ACTION_NOT_FOUND` si absent

Verifie a la compilation que `InMemoryActionRegistry` implemente `port.ActionRegistry`.

### CircuitBreakerService

**Fichier** : `circuit_breaker_service.go`

**But** : Proteger contre les echecs en cascade. Quand les echecs consecutifs depassent un seuil, le circuit breaker s'ouvre et bloque tous les runs du projet.

**Structure** :
```go
type CircuitBreakerService struct {
    projectRepo port.ProjectRepository
    eventPub    port.EventPublisher
    logger      *slog.Logger
}
```

L'etat est stocke sur le `Project` (champs `CircuitBreakerCount`, `CircuitBreakerActive`, `CircuitBreakerMax`).

**Methodes** :
- `CheckCircuitBreaker(ctx, projectID)` : Retourne `CIRCUIT_BREAKER_OPEN` si actif
- `RecordFailure(ctx, projectID)` : Incremente le compteur, trip si seuil atteint, publie event
- `RecordSuccess(ctx, projectID)` : Reset le compteur a 0 (evite les faux positifs)
- `Reset(ctx, projectID)` : Reset admin, publie `circuit_breaker.reset`

**Integration avec PipelineExecutor** :
- Verifie avant chaque execution de run
- Record failure apres un echec de step
- Record success apres une completion reussie

### TimeoutEnforcer

**Fichier** : `timeout_enforcer.go`

**But** : Surveiller les containers agent et forcer l'arret si le timeout est depasse.

**Structure** :
```go
type TimeoutEnforcer struct {
    containerMgr   port.ContainerManager
    runRepo        port.RunRepository
    projectRepo    port.ProjectRepository
    logger         *slog.Logger
    defaultTimeout time.Duration  // 30 minutes par defaut
    checkInterval  time.Duration  // 30 secondes par defaut
}
```

**Fonctionnement** :
1. `Start()` : Boucle infinie avec ticker, bloque jusqu'a annulation du context
2. `CheckTimeouts()` : A chaque tick :
   - Liste tous les containers avec label `managed_by=hopeitworks`
   - Pour chaque container : extrait `run_id` et `step_id` des labels
   - Verifie le temps ecoule depuis `startedAt` du step
   - Si depasse (timeout projet-specifique ou defaut) : stop container, marque step et run `failed` avec raison `container_timeout`
3. `getTimeout()` : Timeout specifique au projet (`project.MaxContainerTimeout`) ou defaut

### OrphanCleaner

**Fichier** : `orphan_cleaner.go`

**But** : Nettoyer les containers laisses par un crash ou un arret inattendu. Execute une seule fois au demarrage de l'API.

**Criteres d'orphelin** :
- Pas de label `run_id`
- `run_id` invalide ou run inexistant en DB
- Run associe n'est pas actif (ni `running` ni `pending`)

**Comportement** : Continue meme si des suppressions individuelles echouent (resilient).

---

## 4. Ports (Interfaces)

Fichier source : `backend/internal/domain/port/`

### RunRepository

**Fichier** : `run_repository.go`

```go
type RunRepository interface {
    // Run CRUD
    CreateRun(ctx, run) (*Run, error)
    GetRun(ctx, id) (*Run, error)
    GetActiveRunByStory(ctx, storyID) (*Run, error)  // pending ou running
    ListRunsByProject(ctx, projectID, limit, offset) ([]*Run, error)
    ListRunsByStory(ctx, storyID, limit, offset) ([]*Run, error)
    UpdateRunStatus(ctx, id, status, startedAt, completedAt, pausedAt, errorMsg) (*Run, error)
    CountRunsByProject(ctx, projectID) (int64, error)
    CountRunsByStory(ctx, storyID) (int64, error)

    // RunStep CRUD
    CreateRunStep(ctx, step) (*RunStep, error)
    GetRunStep(ctx, id) (*RunStep, error)
    ListRunStepsByRun(ctx, runID) ([]*RunStep, error)
    UpdateRunStepStatus(ctx, id, status, startedAt, completedAt, errorMsg) (*RunStep, error)
    UpdateRunStepContainerInfo(ctx, id, containerID, logTail) (*RunStep, error)

    // Retry support
    CreateRetryRunStep(ctx, step) (*RunStep, error)
    ListRetryStepsByParent(ctx, parentStepID) ([]*RunStep, error)
}
```

Interface la plus riche du systeme. Gere a la fois les Runs et leurs RunSteps. Utilisee par quasiment tous les services core.

### PipelineConfigRepository

**Fichier** : `pipeline_config_repository.go`

```go
type PipelineConfigRepository interface {
    GetByProjectID(ctx, projectID) (*PipelineConfig, error)
    Upsert(ctx, config) (*PipelineConfig, error)
}
```

Simple CRUD. Upsert (insert or update) car chaque projet a une seule config.

### EpicRunRepository

**Fichier** : `epic_run_repository.go`

```go
type EpicRunRepository interface {
    CreateEpicRun(ctx, run) (*EpicRun, error)
    GetEpicRun(ctx, id) (*EpicRun, error)
    UpdateEpicRunStatus(ctx, id, status, completedAt) (*EpicRun, error)
    InsertEpicRunStory(ctx, story EpicRunStory) error
    UpdateEpicRunStoryStatus(ctx, epicRunID, storyID, status, runID) error
    ListEpicRunStories(ctx, epicRunID) ([]EpicRunStory, error)
}
```

Gere l'entite EpicRun et la table de jointure `epic_run_stories`.

### ContainerManager

**Fichier** : `container_manager.go`

```go
type ContainerManager interface {
    Create(ctx, opts ContainerOpts) (containerID string, error)
    Start(ctx, containerID) error
    Stop(ctx, containerID) error     // SIGTERM, 10s, SIGKILL
    Remove(ctx, containerID) error
    Wait(ctx, containerID) (exitCode int, error)
    ListContainers(ctx, labels map[string]string) ([]ContainerInfo, error)
}
```

Abstraction du lifecycle Docker. `ContainerInfo` contient ID, labels, et created_at.

### JobQueue

**Fichier** : `job_queue.go`

```go
type JobQueue interface {
    EnqueueExecuteRun(ctx, runID uuid.UUID) error
}
```

Interface minimale. Implementation via River (job queue Postgres-based). Le worker River appelle `PipelineExecutor.ExecuteRun()`.

### CommandRunner

**Fichier** : `command_runner.go`

```go
type CommandRunner interface {
    Run(ctx, workDir, name string, args ...string) (stdout string, error)
}
```

Abstraction des commandes shell pour testabilite. Utilise principalement par les GitProvider impls.

### ActionRegistry (port)

**Fichier** : `action_registry.go`

```go
type ActionRegistry interface {
    Register(action model.Action)
    Get(name string) (model.Action, error)
}
```

Contrat pour le registre d'actions. Implementation in-memory thread-safe.

### LogStreamer

**Fichier** : `log_streamer.go`

```go
type LogStreamer interface {
    StreamLogs(ctx, containerID, runID, stepID string) (
        <-chan model.LogEvent,  // events de log parses
        <-chan int,             // exit code quand le container s'arrete
        error,
    )
}
```

Stream les logs d'un container Docker en temps reel. Parse le NDJSON pour extraire les events de cout et les logs structures.

### TemplateRenderer

**Fichier** : `template_renderer.go`

```go
type TemplateRenderer interface {
    Render(templateContent string, ctx *model.TemplateContext) (string, error)
}
```

Rend les templates Handlebars avec le contexte story. Implementation via la librairie `handlebars`.

### EventPublisher

**Fichier** : `event_publisher.go`

```go
type EventPublisher interface {
    Publish(ctx, event model.Event) error
}
```

Persiste un event en DB. Le trigger Postgres NOTIFY propage automatiquement l'event au bus SSE.

### GitProvider et GitProviderFactory

**Fichier** : `git_provider.go`, `git_provider_factory.go`

```go
type GitProvider interface {
    CloneRepo(ctx, repoURL, targetDir) error
    CreateBranch(ctx, workDir, branchName) error
    Push(ctx, workDir, commitMsg) error
    CreatePR(ctx, workDir, title, body, baseBranch) (prURL, error)
    MergePR(ctx, workDir, prIdentifier) error
    GetCIStatus(ctx, workDir) (status, error)
    GetPRDiff(ctx, prURL) (string, error)
    CreateRemoteBranch(ctx, repoURL, branchName, baseBranch) error
    CreateRemotePR(ctx, repoURL, title, body, headBranch, baseBranch) (prURL, error)
    GetRemoteCIStatus(ctx, prURL) (status, error)
}

type GitProviderFactory interface {
    ForProjectID(ctx, projectID uuid.UUID) (GitProvider, error)
}
```

Deux modes : local (workDir-based) et remote (API-based). Le factory resolve le bon provider (GitHub ou Gitea) en fonction de la config du projet.

---

## 5. Actions (Adapter layer)

Fichier source : `backend/internal/adapter/action/`

Toutes les actions implementent `model.Action` (interface `Name() string` + `Execute(ctx, *RunContext) error`).

### AgentRunAction

**Fichier** : `agent_run.go` | **Nom** : `agent_run`

**But** : Executer un agent Claude Code dans un container Docker.

**Flux d'execution** :

1. Fetch story et project
2. Resolve et render le prompt Handlebars depuis `metadata["template_content"]`
3. Resolve l'image Docker depuis `metadata["agent_image"]` (requis)
4. Cree le container avec les env vars : `REPO_URL`, `BRANCH_NAME`, `STORY_KEY`, `PROMPT_CONTENT`, `GIT_TOKEN`, `MODEL`, `CLAUDE_CODE_OAUTH_TOKEN`
5. Demarre le container
6. Persiste le `container_id` sur le RunStep
7. Stream les logs via `LogStreamer` :
   - Maintient un ring buffer des N derniers messages (pour log_tail en cas d'erreur)
   - Accumule les events de cout (type "cost") pour recording ulterieur
   - Forward les logs normaux vers le systeme d'events
8. Attend la sortie du container
9. Si exit code != 0 : persiste le log tail, retourne erreur
10. Record les couts accumules via `CostService.RecordStepCost()`
11. Cleanup : stop + remove container (toujours, via defer)

**Labels container** : `managed_by=hopeitworks`, `run_id=<UUID>`, `step_id=<UUID>`, `story_key=<key>`

**Aliases registres** : `implement`, `review`, `merge` -> tous resolvent vers `AgentRunAction`

### GitBranchAction

**Fichier** : `git_branch.go` | **Nom** : `git_branch`

**But** : Creer une branche feature sur le remote via API Git.

**Config step** :
- `branch_pattern` : Pattern de nommage (defaut `feat/{story_key}-{slug}`)
- `base_branch` : Branche de base (defaut `main`)

**Flux** :
1. Resolve GitProvider via factory
2. Fetch project (pour repo_url) et story (pour key et title)
3. Render le nom de branche : remplace `{story_key}` et `{slug}` (slugify du titre)
4. Appelle `GitProvider.CreateRemoteBranch()` (API, pas de clone)
5. Stocke `branch_name` dans `runCtx.Metadata`

### GitPRAction

**Fichier** : `git_pr.go` | **Nom** : `git_pr`

**But** : Creer une Pull Request sur le remote via API Git.

**Metadata lues** :
- `title_template` : Template de titre (defaut `{story_key}: {story_title}`)
- `target_branch` : Branche cible (defaut `main`)
- `draft` : `"true"` pour draft PR
- `branch_name` : Branche source (requis, set par git_branch)

**Flux** :
1. Resolve GitProvider, project, story
2. Render le titre avec variables (story_key, story_title, scope, branch_name)
3. Construit le body PR (objective tronque a 500 chars + footer "Generated by hopeitworks")
4. Appelle `GitProvider.CreateRemotePR()` (API, pas de clone)
5. Stocke `pr_url` dans `runCtx.Metadata`

### HITLGateAction

**Fichier** : `hitl_gate.go` | **Nom** : `hitl_gate`

**But** : Suspendre le pipeline en attendant une approbation humaine, avec diff PR optionnel.

**Flux** :
1. Fetch story (pour story_key)
2. Si `pr_url` dans metadata : fetch le diff PR via `GitProvider.GetPRDiff()` (non-fatal)
3. Cree un `HITLRequest` avec `GateType="approval"` et `DiffContent`
4. Transition step -> `waiting_approval`
5. Publie `hitl_gate.pending`
6. Retourne `nil` (la suspension n'est PAS une erreur)

Le `PipelineExecutor` re-fetch le step apres `Execute()` et detecte `waiting_approval` -> retourne `errStepSuspended` -> arrete proprement le pipeline.

### HumanAction

**Fichier** : `human.go` | **Nom** : `human`

**But** : Suspendre le pipeline pour une approbation humaine avec message configurable (sans diff PR).

**Config step** :
- `message` : Template de message (defaut `"Human approval required for step {step_name}"`)
- `instructions` : Instructions optionnelles

**Flux** similaire a HITLGateAction mais avec `GateType="human"` et pas de fetch de diff.

### CIPollAction

**Fichier** : `ci_poll.go` | **Nom** : `ci_poll`

**But** : Poller le statut CI d'une PR jusqu'a pass, fail, ou timeout.

**Config** :
- `DefaultPollInterval` : 30 secondes
- `DefaultTimeout` : 15 minutes

**Metadata lues** : `pr_url` (requis), `poll_interval_seconds`, `timeout_seconds`

**Flux** :
1. Cree un context avec timeout
2. Boucle sur un ticker :
   - Appelle `GitProvider.GetRemoteCIStatus(pr_url)`
   - `pass` -> retourne nil
   - `fail` -> retourne erreur `CI_POLL_FAILED`
   - `pending`/`no_checks` -> continue
   - Publie `ci_poll.checking` a chaque iteration
3. Timeout -> erreur `CI_POLL_TIMEOUT`

### NotificationAction

**Fichier** : `notification.go` | **Nom** : `notification`

**But** : Publier un event de notification avec message template.

**Non-fatal** : Les erreurs de publication sont loggees mais ne font pas echouer le step.

**Flux** :
1. Lit le template de message depuis metadata (defaut `"Pipeline step {step_name} completed"`)
2. Fetch story pour story_key
3. Render le template avec variables (story_key, step_name, run_id, branch_name, pr_url)
4. Publie `notification.sent`

### IncrementalRetryAction

**Fichier** : `incremental_retry.go` | **Nom** : `incremental_retry`

**But** : Coordonner la logique de retry pour les steps agent echoues.

**Metadata lues** : `parent_step_id`, `retry_policy.max_retries`, `retry_policy.max_incremental`

**Flux** :
1. Fetch le step parent
2. Verifie les limites de retry
3. Determine le type : `incremental` (retries 1-2, injecte error_context/log_tail) ou `full` (retry 3+, nettoie le contexte d'erreur)
4. Cree un nouveau RunStep avec metadata de retry
5. Delegue l'execution a `AgentRunAction`

---

## 6. Tests

### Patterns de test

**Table-driven tests** : Pattern standard Go pour tous les tests unitaires.

```go
func TestBuildDAG(t *testing.T) {
    tests := []struct {
        name       string
        stories    []model.Story
        wantGroups [][]string
        wantErr    string
    }{...}
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) { ... })
    }
}
```

**Fixture pattern** : Les tests du PipelineExecutor utilisent `executorTestFixture` qui encapsule le setup complet (run, steps, mocks, executor).

### Mocks et fixtures

Tous les mocks sont **ecrits a la main** (pas de mockgen). Ils implementent directement les interfaces port.

**Exemples** :

| Mock | Interface implementee | Localisation |
|------|----------------------|--------------|
| `mockRunRepo` | `port.RunRepository` | `pipeline_executor_test.go` |
| `mockStoryRepoForExecutor` | `port.StoryRepository` | `pipeline_executor_test.go` |
| `mockActionRegistry` | `port.ActionRegistry` | `pipeline_executor_test.go` |
| `mockEventPublisher` | `port.EventPublisher` | `pipeline_executor_test.go` |
| `mockAction` | `model.Action` | `pipeline_executor_test.go` |

**Pattern mock** : Chaque methode a un champ callback (`xxFn`) qui peut etre override par le test :

```go
type mockRunRepo struct {
    getRunFn            func(ctx context.Context, id uuid.UUID) (*model.Run, error)
    updateRunStatusFn   func(ctx context.Context, ...) (*model.Run, error)
    // ...
}
```

**Factory helpers** :

```go
func newStory(key string, dependsOn []string, targetFiles []string) model.Story
func newExecutorTestFixture(stepCount int) *executorTestFixture
```

### Couverture des cas edge

#### Pipeline Executor (`pipeline_executor_test.go`)

| Test | Scenario |
|------|----------|
| `TestExecuteRun_HappyPath` | 3 steps en sequence, tous OK, verification des transitions et de l'ordre |
| `TestExecuteRun_EventsPublishedInOrder` | Verification de l'ordre exact des events : run.started, story.status_updated, step.started/completed..., run.completed, story.status_updated |
| `TestExecuteRun_StepFailure` | Step 1 echoue, step 2 ne s'execute pas, run marque failed |
| `TestExecuteRun_Cancellation` | Context annule apres step 0, steps et run marques cancelled |
| `TestExecuteRun_StepTimestamps` | Verification que started_at et completed_at sont dans la fenetre d'execution |
| `TestExecuteRun_MetadataSharedBetweenSteps` | Step 0 ecrit branch_name, step 1 le lit |
| `TestExecuteRun_RunNotFound` | Run inexistant -> DomainError not_found |
| `TestExecuteRun_ActionNotFound` | Action non enregistree -> run failed |
| `TestExecuteRun_StepOrderRespected` | Steps retournes en ordre inverse -> toujours executes dans l'ordre step_order |
| `TestExecuteRun_FailureEventOrder` | Verification que step.failed precede run.failed dans les events |
| `TestExecuteRun_StepSuspendedForApproval` | Step passe a waiting_approval -> pipeline arrete sans erreur |
| `TestExecuteRun_PauseStopsExecution` | Run detecte comme paused -> ErrRunPaused, steps suivants non executes |
| `TestExecuteRun_ResumeSkipsCompletedSteps` | Step 0 deja completed -> steps 1 et 2 seulement executes |
| `TestExecuteRun_CircuitBreakerBlocksExecution` | Circuit breaker ouvert -> run immediatement failed |
| `TestExecuteRun_CircuitBreakerRecordsFailure` | Echec -> compteur CB incremente |
| `TestExecuteRun_CircuitBreakerRecordsSuccess` | Succes -> compteur CB reset a 0 |
| `TestExecuteRun_TemplateContentInjectedPerStep` | template_content injecte correctement par step |
| `TestExecuteRun_RunMetadataMergedIntoContext` | Metadata du run mergees dans le context d'execution |
| `TestExecuteRun_ModelInjectedPerStep` | model injecte correctement par step, absent pour step sans model |

#### Scheduler Service (`scheduler_service_test.go`)

| Test | Scenario |
|------|----------|
| `empty input` | 0 stories -> 0 groupes |
| `single story no deps` | 1 story -> 1 groupe |
| `two independent stories` | 2 stories sans deps -> meme groupe |
| `linear chain A->B->C` | Chaine -> 3 groupes sequentiels |
| `diamond A->B, A->C, B->D, C->D` | Diamant -> 3 groupes |
| `cycle of two A<->B` | Cycle -> DAG_CYCLE_DETECTED |
| `cycle of three A->B->C->A` | Cycle 3 -> DAG_CYCLE_DETECTED |
| `unknown dep key ignored` | Dependance vers cle inconnue -> ignoree |
| `file conflict implicit edge` | 2 stories meme fichier -> serialisees par ordre lexicographique |
| `combined explicit and file conflict` | Deps explicites + conflit fichier combines |
| `file conflict with three stories` | 3 stories meme fichier -> 3 couches sequentielles |

#### Run Model (`run_test.go`)

| Test | Scenario |
|------|----------|
| `ValidateRunTransition` | Matrice complete de transitions valides/invalides pour tous les statuts |
| `ComputeProgress` | 0 steps, nil, aucun complete, 2/3, 3/3, mixed statuses, cancelled |
| `ValidateStepTransition` | Matrice complete pour tous les statuts step, incluant waiting_approval |

---

## 7. Flux de donnees complets

### Single Story Run

```
Client POST /projects/{pid}/stories/{sid}/runs
    |
    v
RunHandler.LaunchRun()
    |
    v
RunService.LaunchRun()
    |-- Validate story (exists, not done, no active run)
    |-- Fetch pipeline config YAML
    |-- Parse YAML -> PipelineConfigYAML
    |-- Validate agent_id for agent_run steps
    |-- Resolve agents -> snapshot model, image, template_content in metadata
    |-- Create Run (status: pending, with metadata)
    |-- Create RunSteps (one per flat step)
    |-- Enqueue River job (execute_run)
    |
    v
River Worker (async)
    |
    v
PipelineExecutor.ExecuteRun()
    |-- Check circuit breaker
    |-- Transition run: pending -> running
    |-- Transition story: backlog -> running
    |-- For each step (sorted by step_order):
    |     |-- Check cancellation
    |     |-- Check pause
    |     |-- executeStep():
    |     |     |-- Transition step: pending -> running
    |     |     |-- ActionRegistry.Get(step.action)
    |     |     |-- Extract step config from snapshot
    |     |     |-- Inject per-step metadata (template_content, model, agent_id, agent_image)
    |     |     |-- Action.Execute(ctx, runCtx)
    |     |     |     |-- [agent_run]: container create -> start -> stream logs -> wait -> record cost
    |     |     |     |-- [git_branch]: CreateRemoteBranch -> store branch_name in metadata
    |     |     |     |-- [git_pr]: CreateRemotePR -> store pr_url in metadata
    |     |     |     |-- [ci_poll]: poll GetRemoteCIStatus until pass/fail/timeout
    |     |     |     |-- [hitl_gate]: create HITL request -> step waiting_approval
    |     |     |     |-- [notification]: publish notification.sent event
    |     |     |-- If step suspended (waiting_approval): return nil (stop cleanly)
    |     |     |-- Transition step: running -> completed
    |-- All steps OK:
    |     |-- Transition run: running -> completed
    |     |-- Transition story: running -> done
    |     |-- Reset circuit breaker
    |-- Step failure:
    |     |-- Transition step: running -> failed
    |     |-- Transition run: running -> failed
    |     |-- Transition story: running -> failed
    |     |-- Record circuit breaker failure
```

### Epic Run (multi-story DAG)

```
Client POST /projects/{pid}/epics/{eid}/runs
    |
    v
EpicRunHandler.LaunchEpicRun()
    |
    v
EpicRunService.LaunchEpicRun()
    |-- Validate epic (exists, belongs to project)
    |-- Fetch all stories for epic
    |-- SchedulerService.BuildDAG(stories) -> DAGResult
    |     |-- Build adjacency list (explicit deps + file conflicts)
    |     |-- Kahn's algorithm -> topological layers
    |-- Create EpicRun (status: pending)
    |-- Insert EpicRunStory rows with group_index
    |-- Launch ParallelGroupExecutor in detached goroutine
    |-- Return 202 Accepted
    |
    v
ParallelGroupExecutor.Execute() (background goroutine)
    |-- Transition epic run: pending -> running
    |-- For each DAG layer (sequentially):
    |     |-- Launch stories in parallel (errgroup)
    |     |     |-- runStory():
    |     |     |     |-- RunService.LaunchRun() -> create run + steps
    |     |     |     |-- PipelineExecutor.ExecuteRun() (direct, not via job queue)
    |     |     |     |-- Update EpicRunStory status
    |     |-- If any story fails: fail-fast, mark epic run failed
    |-- All layers completed:
    |     |-- Transition epic run: running -> completed
```

### Retry flow

```
Client POST /runs/{runId}/steps/{stepId}/retry
    |
    v
RunHandler.RetryStep()
    |
    v
RunService.RetryStep()
    |-- Validate step is failed
    |-- Check retry limits (from pipeline config RetryPolicy)
    |-- Determine retry type: incremental (1-2) or full (3+)
    |-- Create new RunStep (with parent_step_id, retry_count, retry_type)
    |-- Transition run: failed -> running
    |-- Enqueue execute_run job
    |
    v
PipelineExecutor.ExecuteRun() (resumes from last completed step)
    |-- Skips completed steps
    |-- Executes retry step (and any remaining steps)
```

### HITL Suspension flow

```
Pipeline execution reaches hitl_gate step:
    |
    v
HITLGateAction.Execute()
    |-- Create HITLRequest (status: pending)
    |-- Fetch PR diff (optional)
    |-- Transition step: running -> waiting_approval
    |-- Publish hitl_gate.pending event
    |-- Return nil (not an error)
    |
    v
PipelineExecutor (back in executeStep)
    |-- Re-fetch step -> detects waiting_approval
    |-- Return errStepSuspended (internal sentinel)
    |-- ExecuteRun catches errStepSuspended -> return nil
    |-- Pipeline is now suspended (run stays "running", step stays "waiting_approval")

...later...

Client POST /hitl-requests/{id}/approve
    |
    v
HITLHandler -> HITLService.Approve()
    |-- Transition HITLRequest: pending -> approved
    |-- Transition step: waiting_approval -> completed
    |-- Re-enqueue execute_run job
    |
    v
PipelineExecutor.ExecuteRun() (resumes)
    |-- Skips completed steps (including the approved HITL step)
    |-- Continues with remaining steps
```

---

*Document genere depuis l'analyse du code source du backend hopeitworks, branche develop, commit 319a721.*
