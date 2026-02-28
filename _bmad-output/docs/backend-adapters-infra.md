# Backend Infrastructure — Adapters & Infrastructure Techniques

Documentation technique de l'infrastructure backend hopeitworks.
Source code: `backend/internal/adapter/`, `backend/internal/domain/`, `backend/queries/`, `backend/migrations/`.

---

## Table des matières

1. [Postgres Adapter](#1-postgres-adapter)
   - 1.1 [Pool & connexion](#11-pool--connexion)
   - 1.2 [Migrations](#12-migrations)
   - 1.3 [Pattern sqlc & Queries](#13-pattern-sqlc--queries)
   - 1.4 [Repository implementations](#14-repository-implementations)
   - 1.5 [Schema des tables clés](#15-schema-des-tables-clés)
2. [River Adapter — Job Queue](#2-river-adapter--job-queue)
   - 2.1 [JobQueue](#21-jobqueue)
   - 2.2 [ExecuteRunWorker](#22-executerunworker)
3. [Git Adapter](#3-git-adapter)
   - 3.1 [Port GitProvider](#31-port-gitprovider)
   - 3.2 [GhCliAdapter (GitHub)](#32-ghcliadapter-github)
   - 3.3 [GiteaAPIAdapter](#33-giteaapiadapter)
   - 3.4 [GitProviderFactory](#34-gitproviderfactory)
4. [Events System](#4-events-system)
   - 4.1 [Domain model Event](#41-domain-model-event)
   - 4.2 [Ports](#42-ports)
   - 4.3 [EventRepo — Publisher & Repository](#43-eventrepo--publisher--repository)
   - 4.4 [EventBus — LISTEN/NOTIFY](#44-eventbus--listennotify)
   - 4.5 [SSE Handler](#45-sse-handler)
   - 4.6 [Trigger SQL Postgres](#46-trigger-sql-postgres)
   - 4.7 [Types d'événements](#47-types-dévénements)
5. [Handlebars Adapter — Template Renderer](#5-handlebars-adapter--template-renderer)
6. [Markdown Adapter](#6-markdown-adapter)
7. [Action Adapter](#7-action-adapter)
   - 7.1 [Interface Action](#71-interface-action)
   - 7.2 [AgentRunAction](#72-agentrunaction)
   - 7.3 [GitBranchAction](#73-gitbranchaction)
   - 7.4 [GitPRAction](#74-gitpraction)
   - 7.5 [CIPollAction](#75-cipollaction)
   - 7.6 [HITLGateAction](#76-hitlgateaction)
   - 7.7 [HumanAction](#77-humanaction)
   - 7.8 [NotificationAction](#78-notificationaction)
   - 7.9 [IncrementalRetryAction](#79-incrementalretryaction)
8. [Discord & Webhook Adapters](#8-discord--webhook-adapters)
   - 8.1 [Discord Notifier](#81-discord-notifier)
   - 8.2 [Webhook Notifier](#82-webhook-notifier)
   - 8.3 [SMTP EmailSender](#83-smtp-emailsender)
   - 8.4 [Port Notifier](#84-port-notifier)
9. [Configuration](#9-configuration)
   - 9.1 [Struct Config](#91-struct-config)
   - 9.2 [Loader](#92-loader)
   - 9.3 [Variables d'environnement](#93-variables-denvironnement)

---

## 1. Postgres Adapter

**Package:** `internal/adapter/postgres`

Le postgres adapter regroupe toutes les implémentations de repositories, l'EventBus LISTEN/NOTIFY, et les fichiers générés par sqlc. C'est l'adapteur de persistance central du système.

### 1.1 Pool & connexion

**Fichier:** `pool.go`

```go
func NewPool(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error)
```

- Construit le DSN : `postgres://user:password@host:port/name?sslmode=...`
- Configure `MaxConns`, `MinConns`, `MaxConnLifetime` depuis la config
- Ping avec timeout 5s pour valider la connectivité
- Retourne une erreur wrappée si le ping échoue

**Driver utilisé:** `github.com/jackc/pgx/v5/pgxpool` (pgx natif, pas `database/sql`)

### 1.2 Migrations

**Fichier:** `migrator.go`

```go
func RunMigrations(migrationsFS fs.FS, dsn string, logger *slog.Logger) error
```

- Utilise `golang-migrate/migrate` avec le driver `pgx/v5`
- Source : filesystem embarqué (`io/fs.FS`) pointant vers `backend/migrations/`
- Lit la version courante avant migration pour logger `from_version` → `to_version`
- `migrate.ErrNoChange` est géré silencieusement (log "up to date")
- Auto-migrate configurable via `database.auto_migrate` (défaut: `true`)

**Numérotation des migrations:** `000001_create_users_table.up.sql` / `000001_create_users_table.down.sql`

Liste des migrations (29 au total, jusqu'à `000029_add_agent_id_to_cost_records`).

### 1.3 Pattern sqlc & Queries

**Fichier généré:** `db.go` (généré par sqlc v1.30.0)

```go
type DBTX interface {
    Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
    Query(context.Context, string, ...interface{}) (pgx.Rows, error)
    QueryRow(context.Context, string, ...interface{}) pgx.Row
}

type Queries struct { db DBTX }

func (q *Queries) WithTx(tx pgx.Tx) *Queries
```

Le type `DBTX` accepte indifféremment un `*pgxpool.Pool` ou une `pgx.Tx`, ce qui permet l'utilisation transactionnelle transparente.

**Fichiers de queries SQL:** `backend/queries/*.sql`

| Fichier | Domaine |
|---------|---------|
| `agents.sql` | Agents CRUD |
| `cost_records.sql` | Enregistrements de coût |
| `epic_runs.sql` | Runs d'épic |
| `epics.sql` | Epics CRUD |
| `events.sql` | Événements (créer, requêtes) |
| `hitl_requests.sql` | Requêtes HITL |
| `notification_configs.sql` | Configs de notifications |
| `password_reset_tokens.sql` | Tokens de reset de mot de passe |
| `pipeline_configs.sql` | Configurations de pipeline |
| `project_users.sql` | Membership projet/user |
| `projects.sql` | Projets CRUD |
| `revoked_tokens.sql` | Tokens JWT révoqués |
| `run_steps.sql` | Steps de run |
| `runs.sql` | Runs de pipeline |
| `stories.sql` | Stories CRUD |
| `users.sql` | Utilisateurs CRUD |

**Exemple de query sqlc (`events.sql`):**

```sql
-- name: CreateEvent :one
INSERT INTO events (id, project_id, entity_type, entity_id, action, payload, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetEventsSince :many
SELECT e.*
FROM events e
WHERE e.project_id = $1
  AND e.created_at > (
      SELECT anchor.created_at FROM events anchor WHERE anchor.id = $2
  )
ORDER BY e.created_at ASC;
```

**Régénération:** `cd backend && sqlc generate`

### 1.4 Repository implementations

Chaque repository suit le même pattern :

1. Déclare une struct wrappant `*Queries`
2. Vérifie l'implémentation du port à la compilation via `var _ port.XxxRepository = (*XxxRepo)(nil)`
3. Délègue aux méthodes sqlc générées
4. Mappe les erreurs pgx → `DomainError` (NotFound pour `pgx.ErrNoRows`, Internal pour les autres, Conflict pour les violations d'unicité)
5. Mappe les lignes sqlc → modèles du domaine via des fonctions `toDomainXxx`

**Exemple — RunRepo:**

```go
var _ port.RunRepository = (*RunRepo)(nil)

type RunRepo struct { queries *Queries }

func (r *RunRepo) GetRun(ctx context.Context, id uuid.UUID) (*model.Run, error) {
    row, err := r.queries.GetRunWithStoryKey(ctx, id)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, apperrors.NewNotFound("run", id)
        }
        return nil, apperrors.NewInternal("failed to get run", err)
    }
    return toDomainRunWithStoryKey(row.ID, row.ProjectID, ..., row.StoryKey), nil
}
```

**Repositories implémentés dans le package postgres:**

| Fichier | Port implémenté |
|---------|-----------------|
| `agent_repo.go` | `port.AgentRepository` |
| `cost_repo.go` | `port.CostRepository` |
| `epic_repo.go` | `port.EpicRepository` |
| `epic_run_repository.go` | `port.EpicRunRepository` |
| `event_repo.go` | `port.EventPublisher` + `port.EventRepository` |
| `hitl_repo.go` | `port.HITLRepository` |
| `notification_config_repository.go` | `port.NotificationConfigRepository` |
| `password_reset_token_repository.go` | `port.PasswordResetTokenRepository` |
| `pipeline_config_repo.go` | `port.PipelineConfigRepository` |
| `project_repo.go` | `port.ProjectRepository` |
| `project_user_repo.go` | `port.ProjectUserRepository` |
| `run_repo.go` | `port.RunRepository` |
| `story_repo.go` | `port.StoryRepository` |
| `token_blacklist_repo.go` | `port.TokenBlacklistRepository` |
| `user_repository.go` | `port.UserRepository` |
| `event_bus.go` | `port.EventSubscriber` |

**Mapping d'erreurs Postgres:**

```go
// Exemples de helpers internes au package
func isForeignKeyViolation(err error) bool { ... }  // code pgx 23503
func isUniqueViolation(err error) bool { ... }       // code pgx 23505
```

### 1.5 Schema des tables clés

**Table `events` (migration 000006):**

```sql
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_events_project_id_created_at ON events(project_id, created_at);
CREATE INDEX idx_events_entity_type_entity_id ON events(entity_type, entity_id);
```

La table est **append-only** — des triggers bloquent tout `UPDATE` et `DELETE`. Un trigger `AFTER INSERT` lance `pg_notify('events', ...)`.

---

## 2. River Adapter — Job Queue

**Package:** `internal/adapter/river`

River est une job queue Postgres-native. Les jobs sont persistés dans des tables Postgres dédiées (auto-migrées au démarrage) et consommés de manière transactionnelle.

### 2.1 JobQueue

**Fichier:** `job_queue.go`

```go
var _ port.JobQueue = (*JobQueue)(nil)

type JobQueue struct {
    client *river.Client[pgx.Tx]
}
```

**Port implémenté:**

```go
// internal/domain/port/job_queue.go
type JobQueue interface {
    EnqueueExecuteRun(ctx context.Context, runID uuid.UUID) error
}
```

**Construction:**

```go
func NewJobQueue(pool *pgxpool.Pool, workers *river.Workers) (*JobQueue, error)
```

1. Crée un `rivermigrate.Migrator` pour auto-migrer les tables River
2. Crée un `river.Client` avec `MaxWorkers: 10` sur la queue par défaut
3. Tous les types de workers doivent être enregistrés dans `workers` avant l'appel

**Enqueue:**

```go
func (q *JobQueue) EnqueueExecuteRun(ctx context.Context, runID uuid.UUID) error {
    _, err := q.client.Insert(ctx, ExecuteRunArgs{RunID: runID}, nil)
    return err
}
```

**Client exposé:** `(q *JobQueue) Client() *river.Client[pgx.Tx]` — utilisé pour `Start`/`Stop` du lifecycle.

### 2.2 ExecuteRunWorker

**Fichier:** `execute_run_job.go`

```go
type ExecuteRunArgs struct {
    RunID uuid.UUID `json:"run_id"`
}

func (ExecuteRunArgs) Kind() string { return "execute_run" }
```

Le `Kind()` est l'identifiant unique du job dans River — il doit correspondre lors de l'enregistrement du worker.

```go
type ExecuteRunWorker struct {
    river.WorkerDefaults[ExecuteRunArgs]
    executor *service.PipelineExecutor
}

func (w *ExecuteRunWorker) Timeout(_ *river.Job[ExecuteRunArgs]) time.Duration {
    return 45 * time.Minute  // Les containers Claude peuvent prendre 30+ min
}

func (w *ExecuteRunWorker) Work(ctx context.Context, job *river.Job[ExecuteRunArgs]) error {
    return w.executor.ExecuteRun(ctx, job.Args.RunID)
}
```

**Flux complet:**

```
Handler HTTP (POST /runs)
  → RunService.LaunchRun()
    → RunRepo.CreateRun()           # Persiste le run
    → JobQueue.EnqueueExecuteRun()  # Insert job dans River
      ← River consomme le job asynchroniquement
        → ExecuteRunWorker.Work()
          → PipelineExecutor.ExecuteRun()
            → Actions séquentielles (agent_run, git_pr, hitl_gate...)
```

---

## 3. Git Adapter

**Package:** `internal/adapter/git`

### 3.1 Port GitProvider

**Fichier:** `internal/domain/port/git_provider.go`

```go
type GitProvider interface {
    CloneRepo(ctx context.Context, repoURL string, targetDir string) error
    CreateBranch(ctx context.Context, workDir string, branchName string) error
    Push(ctx context.Context, workDir string, commitMsg string) error
    CreatePR(ctx context.Context, workDir string, title string, body string, baseBranch string) (prURL string, err error)
    MergePR(ctx context.Context, workDir string, prIdentifier string) error
    GetCIStatus(ctx context.Context, workDir string) (status string, err error)
    GetPRDiff(ctx context.Context, prURL string) (string, error)
    CreateRemoteBranch(ctx context.Context, repoURL string, branchName string, baseBranch string) error
    CreateRemotePR(ctx context.Context, repoURL string, title string, body string, headBranch string, baseBranch string) (prURL string, err error)
    GetRemoteCIStatus(ctx context.Context, prURL string) (status string, err error)
}
```

**Constantes CI:**

```go
const (
    CIStatusPass     = "pass"
    CIStatusFail     = "fail"
    CIStatusPending  = "pending"
    CIStatusNoChecks = "no_checks"
)
```

**Validation de nommage de branche:**

```go
var branchNamePattern = regexp.MustCompile(`^(feat|fix)/[a-zA-Z0-9]+-[a-zA-Z0-9-]+$`)
// Valide: feat/1-14-claude-md-files, fix/3-ci-poller
```

### 3.2 GhCliAdapter (GitHub)

**Fichier:** `gh_cli_adapter.go`

```go
type GhCliAdapter struct {
    runner port.CommandRunner
    logger *slog.Logger
}
```

Toutes les opérations passent par le `CommandRunner` (abstraction testable autour des commandes shell).

| Méthode | Commandes utilisées |
|---------|---------------------|
| `CloneRepo` | `gh repo clone <url> <dir>` |
| `CreateBranch` | `git checkout -b <branch>` |
| `Push` | `git add .` → `git commit -m <msg>` → `git push -u origin HEAD` |
| `CreatePR` | `gh pr create --title <t> --body <b> --base <base>` |
| `MergePR` | `gh pr merge <pr> --squash --delete-branch` |
| `GetCIStatus` | `gh pr checks --json name,state,conclusion` |
| `GetPRDiff` | `gh pr diff <pr_url>` |
| `CreateRemoteBranch` | `gh api repos/{owner}/{repo}/git/refs` (via GitHub API) |
| `CreateRemotePR` | `gh api repos/{owner}/{repo}/pulls` |
| `GetRemoteCIStatus` | `gh api repos/{owner}/{repo}/commits/{sha}/check-runs` |

**Parsing des URLs GitHub:**

```go
var githubRepoPattern = regexp.MustCompile(`^https?://[^/]+/([^/]+)/([^/]+?)(?:\.git)?$`)
var githubPRPattern   = regexp.MustCompile(`^https?://[^/]+/([^/]+)/([^/]+)/pull/(\d+)$`)
```

**Logique CI:**

- `fail` si un check a `conclusion=failure|timed_out|action_required`
- `pending` si un check est en `queued|in_progress`
- `pass` si tous les checks sont terminés sans erreur
- `no_checks` si la liste est vide

**Gestion des erreurs:** les erreurs CLI sont inspectées par string matching sur stdout (`"authentication"`, `"merge conflict"`, `"no pull requests found"`) pour produire des `DomainError` typés (`ErrCodeGitAuthFailed`, `ErrCodeMergeConflict`, `ErrCodePRNotFound`).

### 3.3 GiteaAPIAdapter

**Fichier:** `gitea_api_adapter.go`

```go
type GiteaAPIAdapter struct {
    baseURL    string
    token      string
    runner     port.CommandRunner
    httpClient *http.Client
    logger     *slog.Logger
}
```

**Stratégie hybride :**
- Opérations locales (clone, branch, push) : `git` CLI via `CommandRunner`
- Opérations distantes (PR, CI, merge) : appels HTTP directs à l'API Gitea v1

| Méthode | Transport |
|---------|-----------|
| `CloneRepo` | `git clone <token@url> <dir>` (token injecté dans l'URL) |
| `CreateBranch` | `git checkout -b <branch>` |
| `Push` | `git add/commit/push` via runner |
| `CreatePR` | `POST /api/v1/repos/{owner}/{repo}/pulls` |
| `MergePR` | `POST /api/v1/repos/{owner}/{repo}/pulls/{index}/merge` (squash, delete branch) |
| `GetCIStatus` | `GET /api/v1/repos/{owner}/{repo}/commits/{sha}/statuses` |
| `GetPRDiff` | `GET /api/v1/repos/{owner}/{repo}/pulls/{index}.diff` |
| `CreateRemoteBranch` | `POST /api/v1/repos/{owner}/{repo}/branches` |
| `CreateRemotePR` | `POST /api/v1/repos/{owner}/{repo}/pulls` |
| `GetRemoteCIStatus` | PR details → `GET /api/v1/repos/{owner}/{repo}/commits/{sha}/statuses` |

**Authentification:** header `Authorization: token <token>`

**Helpers internes:**

```go
func (a *GiteaAPIAdapter) doJSON(ctx, method, endpoint, body) ([]byte, error)
func (a *GiteaAPIAdapter) doGet(ctx, endpoint, accept) ([]byte, error)
func injectTokenInURL(repoURL, token string) (string, error)  // https://token@host/path
func stripCredentials(rawURL string) string                    // retire les credentials pour le matching
```

**Parsing URL Gitea:**

```go
var giteaRepoPattern = regexp.MustCompile(`^https?://[^/]+/([^/]+)/([^/]+?)(?:\.git)?$`)
var giteaPRPattern   = regexp.MustCompile(`^https?://[^/]+/([^/]+)/([^/]+)/pulls/(\d+)`)
```

### 3.4 GitProviderFactory

**Fichier:** `provider_factory.go`

```go
type DefaultGitProviderFactory struct {
    projectRepo port.ProjectRepository
    runner      port.CommandRunner
    logger      *slog.Logger
}

func (f *DefaultGitProviderFactory) ForProjectID(ctx context.Context, projectID uuid.UUID) (port.GitProvider, error) {
    project, err := f.projectRepo.GetByID(ctx, projectID)
    switch project.GitProvider {
    case "github", "":
        return NewGhCliAdapter(f.runner, f.logger), nil
    case "gitea":
        token := resolveGitToken(project.GitTokenEnv)
        baseURL := extractBaseURL(safeDeref(project.RepoURL))
        return NewGiteaAPIAdapter(baseURL, token, f.runner, f.logger), nil
    default:
        return nil, fmt.Errorf("unsupported git provider: %s", project.GitProvider)
    }
}
```

**Résolution du token:**

```go
func resolveGitToken(gitTokenEnv *string) string {
    if gitTokenEnv != nil && *gitTokenEnv != "" {
        if v := os.Getenv(*gitTokenEnv); v != "" {
            return v
        }
    }
    return os.Getenv("GITHUB_TOKEN") // fallback backward compat
}
```

Le champ `project.GitTokenEnv` contient le **nom** de la variable d'environnement (ex: `GITEA_TOKEN`), pas le token lui-même. Chaque projet peut avoir son propre token d'accès.

---

## 4. Events System

Le système d'événements est entièrement basé sur Postgres. Il n'y a pas de broker externe (pas de Redis, Kafka, etc.).

### 4.1 Domain model Event

**Fichier:** `internal/domain/model/event.go`

```go
type Event struct {
    ID         uuid.UUID       `json:"id"`
    ProjectID  uuid.UUID       `json:"project_id"`
    EntityType string          `json:"entity_type"` // ex: "run", "step", "hitl", "log"
    EntityID   uuid.UUID       `json:"entity_id"`
    Action     string          `json:"action"`      // ex: "started", "completed", "pending"
    Payload    json.RawMessage `json:"payload"`
    CreatedAt  time.Time       `json:"created_at"`
}

func (e Event) EventName() string {
    return e.EntityType + "." + e.Action  // ex: "run.completed", "hitl_gate.pending"
}
```

### 4.2 Ports

**`EventPublisher`** (`internal/domain/port/event_publisher.go`):
```go
type EventPublisher interface {
    Publish(ctx context.Context, event model.Event) error
}
```

**`EventSubscriber`** (`internal/domain/port/event_subscriber.go`):
```go
type EventSubscriber interface {
    Subscribe(ctx context.Context, projectID uuid.UUID) (<-chan model.Event, func(), error)
    Close() error
}
```

**`EventRepository`** (`internal/domain/port/event_repository.go`):
```go
type EventRepository interface {
    GetEventByID(ctx context.Context, id uuid.UUID) (*model.Event, error)
    GetEventsSince(ctx context.Context, projectID uuid.UUID, afterEventID uuid.UUID) ([]*model.Event, error)
}
```

### 4.3 EventRepo — Publisher & Repository

**Fichier:** `internal/adapter/postgres/event_repo.go`

`EventRepo` implémente à la fois `EventPublisher` et `EventRepository`.

```go
var _ port.EventPublisher = (*EventRepo)(nil)
var _ port.EventRepository = (*EventRepo)(nil)

func (r *EventRepo) Publish(ctx context.Context, event model.Event) error {
    // Auto-génère ID et CreatedAt si absents
    // INSERT INTO events (...) RETURNING *
    // Le trigger Postgres NOTIFY est déclenché automatiquement après INSERT
}

func (r *EventRepo) GetEventByID(ctx context.Context, id uuid.UUID) (*model.Event, error) { ... }

func (r *EventRepo) GetEventsSince(ctx context.Context, projectID uuid.UUID, afterEventID uuid.UUID) ([]*model.Event, error) {
    // SELECT e.* WHERE project_id = $1 AND created_at > (SELECT created_at FROM events WHERE id = $2)
}
```

**Point clé:** `Publish` ne fait qu'un `INSERT`. C'est le trigger SQL Postgres `events_notify_trigger` qui envoie le `pg_notify` automatiquement.

### 4.4 EventBus — LISTEN/NOTIFY

**Fichier:** `internal/adapter/postgres/event_bus.go`

```go
var _ port.EventSubscriber = (*EventBus)(nil)

type EventBus struct {
    connString  string
    conn        *pgx.Conn          // connexion dédiée, hors pool
    eventRepo   port.EventRepository
    subscribers map[uuid.UUID][]chan<- model.Event  // par projectID
    closedChans map[chan<- model.Event]struct{}      // tracking des channels fermés
    listening   bool
    stopCh      chan struct{}
    doneCh      chan struct{}
    closed      bool
    mu          sync.Mutex
}
```

**Conception:**
- Connexion pgx **dédiée** (hors pool) pour `LISTEN events` — le pool ne supporte pas LISTEN
- Buffer de 100 événements par channel subscriber
- Un seul `LISTEN` sur le channel `"events"`, tous les subscribers partagent la même connexion

**Flux Subscribe:**

```
SSEHandler.ServeHTTP()
  → EventBus.Subscribe(projectID)
    → Si !listening: LISTEN events + goroutine listenLoop()
    → Crée chan<- model.Event (buffer 100)
    → Ajoute au map subscribers[projectID]
    → Retourne (<-chan, cleanup func)
```

**Flux listenLoop:**

```
listenLoop() [goroutine]
  → conn.WaitForNotification(ctx 5s timeout)
  → handleNotification(notification)
    → JSON decode du payload minimal (id, project_id, entity_type, entity_id, action)
    → eventRepo.GetEventByID() pour enrichir avec le payload complet
      (pg_notify limite à 8KB — le payload JSON complet est récupéré depuis la DB)
    → Dispatch aux subscribers[notif.ProjectID]
    → Drop avec warning si channel plein (non-bloquant)
```

**Reconnexion:**
- En cas d'erreur de connexion : tentative de reconnexion avec backoff exponentiel (1s, 2s, 4s... jusqu'à 5 tentatives)
- Re-émet `LISTEN events` sur la nouvelle connexion
- Swap atomique de la connexion sous lock

**Fermeture propre:**
```
EventBus.Close()
  → Signale stopCh
  → Attend doneCh (listenLoop terminé)
  → Ferme tous les channels subscribers
  → Ferme la connexion Postgres
```

### 4.5 SSE Handler

**Fichier:** `internal/api/handler/sse_handler.go`

Route: `GET /api/v1/events/stream?project_id={uuid}`

```go
type SSEHandler struct {
    eventSub        port.EventSubscriber
    eventRepo       port.EventRepository
    projectUserRepo port.ProjectUserRepository
    logger          *slog.Logger
}
```

**Comportement:**

1. Valide `project_id` (UUID requis)
2. Authentification JWT via contexte middleware
3. Vérification membership projet (admins exemptés)
4. Assert `http.Flusher`
5. **Désactive le WriteTimeout HTTP** via `http.NewResponseController.SetWriteDeadline(time.Time{})` — sinon le timeout serveur (15s) tuerait la connexion SSE avant le premier keepalive (30s)
6. Headers SSE : `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `X-Accel-Buffering: no`
7. Flush immédiat des headers (permet à `EventSource` de passer CONNECTING → OPEN)
8. **Replay des événements manqués** si `Last-Event-ID` présent dans la requête → `eventRepo.GetEventsSince()`
9. Subscribe à l'EventBus
10. Boucle select : événements | keepalive (30s) | context done

**Format SSE:**

```
event: run.completed
data: {"id":"...","project_id":"...","entity_type":"run","action":"completed","payload":{...}}
id: <event-uuid>

: keepalive
```

**Fonction writeSSEEvent:**

```go
func writeSSEEvent(w io.Writer, f http.Flusher, event model.Event) error {
    payload, _ := json.Marshal(event)
    fmt.Fprintf(w, "event: %s\ndata: %s\nid: %s\n\n",
        event.EventName(), payload, event.ID)
    f.Flush()
}
```

### 4.6 Trigger SQL Postgres

```sql
-- Trigger "notify" — déclenché AFTER INSERT sur events
CREATE OR REPLACE FUNCTION notify_event() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('events', json_build_object(
        'id', NEW.id,
        'project_id', NEW.project_id,
        'entity_type', NEW.entity_type,
        'entity_id', NEW.entity_id,
        'action', NEW.action
        -- SANS payload : pg_notify limite à 8KB
    )::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER events_notify_trigger
    AFTER INSERT ON events
    FOR EACH ROW EXECUTE FUNCTION notify_event();
```

Pourquoi pas le payload complet dans le NOTIFY ? Postgres limite la taille des notifications à **8KB**. Le payload JSON des événements (notamment les logs d'agent) peut dépasser cette limite. L'EventBus reçoit uniquement les métadonnées (id, project_id, entity_type, entity_id, action) et fait un `SELECT` supplémentaire pour récupérer l'événement complet.

### 4.7 Types d'événements

Format : `{entity_type}.{action}`

| Event name | Émis par | Description |
|-----------|----------|-------------|
| `run.started` | PipelineExecutor | Run démarré |
| `run.completed` | PipelineExecutor | Run terminé avec succès |
| `run.failed` | PipelineExecutor | Run en erreur |
| `step.started` | PipelineExecutor | Step démarré |
| `step.completed` | PipelineExecutor | Step terminé |
| `step.failed` | PipelineExecutor | Step en erreur |
| `log.emitted` | AgentRunAction | Ligne de log agent |
| `hitl_gate.pending` | HITLGateAction | Approbation PR requise |
| `human.pending` | HumanAction | Validation humaine requise |
| `ci_poll.checking` | CIPollAction | Polling CI en cours |
| `notification.sent` | NotificationAction | Notification envoyée |

---

## 5. Handlebars Adapter — Template Renderer

**Package:** `internal/adapter/handlebars`

**Fichier:** `renderer.go`

```go
var _ port.TemplateRenderer = (*Renderer)(nil)

type Renderer struct{}

func (r *Renderer) Render(templateContent string, ctx *model.TemplateContext) (string, error) {
    data := map[string]interface{}{
        "story_key":           ctx.StoryKey,
        "story_title":         ctx.StoryTitle,
        "story_objective":     ctx.StoryObjective,
        "target_files":        ctx.TargetFiles,
        "acceptance_criteria": ctx.AcceptanceCriteria,
        "error_context":       ctx.ErrorContext,
        "diff_content":        ctx.DiffContent,
        "branch_name":         ctx.BranchName,
        "repo_url":            ctx.RepoURL,
    }
    result, err := raymond.Render(templateContent, data)
    // ...
}
```

**Librairie:** `github.com/aymerick/raymond` — implémentation Go de Handlebars.js

**Port:**

```go
// internal/domain/port/template_renderer.go
type TemplateRenderer interface {
    Render(templateContent string, ctx *model.TemplateContext) (string, error)
}
```

**TemplateContext** (`internal/domain/model/template_context.go`):

```go
type TemplateContext struct {
    StoryKey           string   // ex: "S-14"
    StoryTitle         string
    StoryObjective     string
    TargetFiles        []string
    AcceptanceCriteria string
    ErrorContext       string   // pour les templates de retry
    LogTail            string   // derniers logs du step échoué
    DiffContent        string   // diff git pour les templates de review
    BranchName         string
    RepoURL            string
}
```

**Usage:** le `Renderer` est appelé dans `AgentRunAction.Execute()` pour transformer le `template_content` de l'agent (stocké dans `Agent.TemplateContent`) en prompt final injecté dans le container via la variable d'environnement `PROMPT_CONTENT`.

**Exemple de template Handlebars:**

```handlebars
You are implementing story {{story_key}}: {{story_title}}

Repository: {{repo_url}}
Branch: {{branch_name}}

## Acceptance Criteria
{{acceptance_criteria}}

{{#if error_context}}
## Previous Error Context
{{error_context}}
{{/if}}
```

**Erreur:** une erreur de rendu produit un `DomainError` avec code `TEMPLATE_RENDER_FAILED` et catégorie `validation`.

---

## 6. Markdown Adapter

**Package:** `internal/adapter/markdown`

**Fichier:** `parser.go`

Ce parser est utilisé pour ingérer des fichiers markdown de stories en lot (import depuis des fichiers `.md` structurés).

**Structure d'un fichier story markdown:**

```markdown
---
key: S-01
epic: E-01
depends_on: [S-02]
scope: backend
status: backlog
---
# Story title here

## Acceptance Criteria
- As a user, I want...
```

**Types:**

```go
type FrontmatterFields struct {
    Key       string   `yaml:"key"`
    Epic      string   `yaml:"epic"`
    DependsOn []string `yaml:"depends_on"`
    Scope     string   `yaml:"scope"`
    Status    string   `yaml:"status"`
}

type ParsedStory struct {
    Key                string
    Title              string
    Epic               string
    DependsOn          []string
    Scope              string
    Status             string
    AcceptanceCriteria string
    ParseError         error
}
```

**API:**

```go
func ParseStoryMarkdown(content string) []ParsedStory
```

**Algorithme:**
1. Divise le contenu sur les délimiteurs `---` (frontmatter open/close)
2. Pour chaque bloc : parse le YAML frontmatter avec `gopkg.in/yaml.v3`
3. Extrait le titre depuis le premier `# H1` avec regex `(?m)^# (.+)$`
4. Le reste du body devient `AcceptanceCriteria`
5. Si le YAML est invalide : retourne un `ParsedStory{ParseError: err}` sans interrompre le parsing des autres blocs

**Particularité:** un document peut contenir plusieurs stories (délimitées par `---`). Chaque story est retournée même si son YAML est invalide (avec `ParseError` non-nil).

---

## 7. Action Adapter

**Package:** `internal/adapter/action`

Le système d'actions est le coeur de l'exécution pipeline. Chaque step d'un pipeline config correspond à un `Action` enregistré dans l'`ActionRegistry`.

### 7.1 Interface Action

```go
// internal/domain/model/action.go
type Action interface {
    Name() string
    Execute(ctx context.Context, runCtx *model.RunContext) error
}
```

Le `RunContext` contient tout le contexte nécessaire à l'exécution d'un step :

```go
type RunContext struct {
    Run      *Run
    RunStep  *RunStep
    StoryID  uuid.UUID
    ProjectID uuid.UUID
    Metadata map[string]any  // accumulateur partagé entre steps du même run
}
```

**Le `Metadata` est partagé entre steps**: les actions écrivent des clés (`branch_name`, `pr_url`) que les actions suivantes lisent. C'est le mécanisme de passage de données inter-step.

**Actions implémentées:**

| Nom | Fichier | Description |
|-----|---------|-------------|
| `agent_run` | `agent_run.go` | Exécute un container Claude Code |
| `git_branch` | `git_branch.go` | Crée une branche remote |
| `git_pr` | `git_pr.go` | Crée une pull request |
| `ci_poll` | `ci_poll.go` | Attend que la CI passe |
| `hitl_gate` | `hitl_gate.go` | Pause pour approbation PR |
| `human` | `human.go` | Pause pour validation humaine |
| `notification` | `notification.go` | Publie un event de notification |
| `incremental_retry` | `incremental_retry.go` | Retry avec contexte d'erreur |

### 7.2 AgentRunAction

**Fichier:** `agent_run.go`

L'action principale : exécute un agent Claude Code dans un container Docker.

```go
type AgentRunAction struct {
    containerMgr port.ContainerManager
    logStreamer   port.LogStreamer
    eventPub     port.EventPublisher
    storyRepo    port.StoryRepository
    projectRepo  port.ProjectRepository
    runRepo      port.RunRepository
    renderer     port.TemplateRenderer
    costSvc      *service.CostService
    config       AgentConfig
    logger       *slog.Logger
}
```

**Séquence d'exécution:**

```
1. Fetch story (storyRepo.GetByID)
2. Fetch project (projectRepo.GetByID)
3. Résoudre template_content depuis runCtx.Metadata["template_content"]
4. Rendre le prompt via renderer.Render(templateContent, tmplCtx)
5. Résoudre agent_image depuis runCtx.Metadata["agent_image"]
6. Créer le container (containerMgr.Create)
7. Démarrer le container (containerMgr.Start)
8. Persister container_id dans run_step
9. Streamer les logs + attendre la sortie (streamAndWait)
10. Vérifier exit code (0 = succès)
```

**Variables d'environnement injectées dans le container:**

```
REPO_URL=<project.RepoURL>
BRANCH_NAME=<runCtx.Metadata["branch_name"]>
STORY_KEY=<story.Key>
PROMPT_CONTENT=<rendered template>
GIT_TOKEN=<token résolu depuis project.GitTokenEnv>
GIT_PROVIDER=<project.GitProvider>
GITHUB_TOKEN=<même token>
CLAUDE_CODE_OAUTH_TOKEN=<env CLAUDE_CODE_OAUTH_TOKEN>
MODEL=<runCtx.Metadata["model"]>  (optionnel)
```

**Labels Docker:**

```
managed_by=hopeitworks
run_id=<uuid>
step_id=<uuid>
story_key=<key>
```

**streamAndWait:**
- Ring buffer de `config.LogTailLines` (défaut 50) pour les derniers logs
- Parsing des log events de type `"cost"` → accumulation dans `costEvents`
- Les events de log non-cost sont publiés dans le système d'événements via `publishLogEvent`
- À la sortie du container : si exit code != 0, persiste le log tail dans run_step
- Enregistrement des coûts via `costSvc.RecordStepCost()` (non-fatal si échec)

**Cleanup:** `defer cleanupContainer(containerID)` avec timeout 30s — stop + remove, erreurs loggées en warning.

### 7.3 GitBranchAction

**Fichier:** `git_branch.go`

Crée une branche feature via l'API distante (sans clone local).

```go
func (a *GitBranchAction) Name() string { return "git_branch" }
```

**Config lue depuis `runCtx.RunStep.Config`:**

| Clé | Défaut | Description |
|-----|--------|-------------|
| `branch_pattern` | `feat/{story_key}-{slug}` | Pattern de nommage |
| `base_branch` | `main` | Branche base |

**Génération du nom de branche:**

```go
slug := slugify(story.Title)  // lowercase, caractères non-alphanumériques → "-"
branchName := strings.ReplaceAll(pattern, "{story_key}", story.Key)
branchName = strings.ReplaceAll(branchName, "{slug}", slug)
```

**Résultat:** `runCtx.Metadata["branch_name"] = branchName`

### 7.4 GitPRAction

**Fichier:** `git_pr.go`

Crée une PR via l'API distante. Lit `branch_name` depuis `Metadata` (posé par `git_branch`).

**Config lue depuis `runCtx.Metadata`:**

| Clé | Défaut | Description |
|-----|--------|-------------|
| `title_template` | `{story_key}: {story_title}` | Template du titre PR |
| `target_branch` | `main` | Branche cible |
| `draft` | `false` | Créer en mode draft |
| `branch_name` | *requis* | Source branch (from git_branch) |

**Corps de la PR (`buildPRBody`):**
```
## S-14: Story title

### Objective

<objective tronqué à 500 chars>

---
> Generated by hopeitworks pipeline
```

**Résultat:** `runCtx.Metadata["pr_url"] = prURL`

### 7.5 CIPollAction

**Fichier:** `ci_poll.go`

Attend que la CI passe sur une PR, avec timeout configurable.

**Config lue depuis `runCtx.Metadata`:**

| Clé | Type | Défaut | Description |
|-----|------|--------|-------------|
| `pr_url` | string | *requis* | URL de la PR à surveiller |
| `poll_interval_seconds` | float64 | 30s | Fréquence de polling |
| `timeout_seconds` | float64 | 15min | Timeout maximum |

**Boucle de polling:**

```
ticker.C → gitProvider.GetRemoteCIStatus(prURL)
  "pass"    → publish ci_poll.checking{status:pass} → return nil
  "fail"    → publish ci_poll.checking{status:fail} → return error
  "pending" → publish ci_poll.checking{status:pending} → continue
  "no_checks" → continue
  erreur    → log warning → continue
timeout     → return error CI_POLL_TIMEOUT
```

### 7.6 HITLGateAction

**Fichier:** `hitl_gate.go`

Suspend le pipeline pour approbation d'une PR. Contrairement à `human`, inclut le diff de la PR.

**Séquence:**

```
1. Fetch story
2. (Optionnel) Fetch PR diff via gitProvider.GetPRDiff(runCtx.Metadata["pr_url"])
   — non-fatal si échec
3. hitlRepo.Create({GateType:"approval", DiffContent:&diff})
4. runRepo.UpdateRunStepStatus(waiting_approval)
5. eventPub.Publish(hitl_gate.pending)
6. return nil  ← suspension n'est pas une erreur
```

### 7.7 HumanAction

**Fichier:** `human.go`

Similaire à `HITLGateAction` mais sans fetch du diff. Affiche un message et des instructions configurables.

**Config lue depuis `runCtx.RunStep.Config`:**

| Clé | Défaut | Description |
|-----|--------|-------------|
| `message` | `Human approval required for step {step_name}` | Message affiché |
| `instructions` | `` | Instructions optionnelles |

**Substitution simple:** `{story_key}`, `{step_name}`, `{branch_name}`, `{pr_url}`

Crée un `HITLRequest` avec `GateType:"human"` et publie un event `human.pending`.

### 7.8 NotificationAction

**Fichier:** `notification.go`

Publie un event `notification.sent`. Ne fait **pas** d'appel réseau — c'est l'EventBus (et éventuellement un worker de notification séparé) qui gère la livraison.

**Config lue depuis `runCtx.Metadata`:**

| Clé | Défaut |
|-----|--------|
| `message` | `Pipeline step {step_name} completed` |

**Substitution:** `{story_key}`, `{step_name}`, `{run_id}`, `{branch_name}`, `{pr_url}`

Toujours `return nil` — les erreurs de publication sont loggées en warning, non-fatales.

### 7.9 IncrementalRetryAction

**Fichier:** `incremental_retry.go`

Coordonne la logique de retry d'un step agent échoué.

**Config lue depuis `runCtx.Metadata`:**

| Clé | Défaut | Description |
|-----|--------|-------------|
| `parent_step_id` | *requis* | ID du step parent échoué |
| `retry_policy.max_retries` | 3 | Nombre max de retries total |
| `retry_policy.max_incremental` | 2 | Nombre max de retries incrémentaux |

**Logique:**

```
1. Fetch parent step
2. Check parent.RetryCount < max_retries
3. Déterminer retry type:
   - "incremental" si RetryCount < max_incremental → injecte error_context + log_tail
   - "full" si RetryCount >= max_incremental → supprime error_context (fresh start)
4. runRepo.CreateRetryRunStep() avec RetryCount+1, ParentStepID
5. Construit nouveau RunContext avec metadata enrichi
6. Délègue à AgentRunAction.Execute()
```

**Retry incrémental vs full:**
- **Incrémental**: le template Handlebars reçoit `error_context` et `log_tail` pour que l'agent puisse corriger son erreur en contexte
- **Full**: pas de contexte d'erreur, l'agent repart de zéro

---

## 8. Discord & Webhook Adapters

### 8.1 Discord Notifier

**Package:** `internal/adapter/discord`
**Fichier:** `notifier.go`

```go
var _ port.Notifier = (*Notifier)(nil)

func (n *Notifier) Send(ctx context.Context, event model.Event, config map[string]string) error
```

**Config requise:** `config["url"]` — URL du webhook Discord.

**Payload Discord (embed):**

```json
{
  "embeds": [{
    "title": "run.completed",
    "description": "Project: <uuid> | Entity: run (<uuid>)",
    "color": 3066993  // vert pour completed, rouge pour failed, jaune pour pending
  }]
}
```

**Mapping couleurs:**

| Event | Couleur | Hex |
|-------|---------|-----|
| `run.completed` | Vert | `#2ECC71` |
| `run.failed` | Rouge | `#E74C3C` |
| `hitl_gate.pending` | Jaune | `#F1C40F` |
| autres | Gris | `#95A5A6` |

### 8.2 Webhook Notifier

**Package:** `internal/adapter/webhook`
**Fichier:** `notifier.go`

```go
var _ port.Notifier = (*Notifier)(nil)

func (n *Notifier) Send(ctx context.Context, event model.Event, config map[string]string) error
```

**Config requise:** `config["url"]` — URL du webhook cible.

**Payload:** le `model.Event` complet sérialisé en JSON, `Content-Type: application/json`.

Contrairement au Discord notifier, le webhook générique envoie l'intégralité de l'event sans transformation.

### 8.3 SMTP EmailSender

**Package:** `internal/adapter/smtp`
**Fichier:** `email_sender.go`

```go
var _ port.EmailSender = (*EmailSender)(nil)

type EmailSender struct {
    cfg pkgconfig.SMTPConfig
}
```

Utilise `net/smtp` stdlib (pas de librairie externe).

```go
func (s *EmailSender) Send(_ context.Context, msg port.EmailMessage) error {
    addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
    // Construit les headers MIME avec Content-Type: text/html; charset=UTF-8
    // auth est nil si cfg.Username == "" (compatible MailHog en dev)
    smtp.SendMail(addr, auth, s.cfg.From, []string{msg.To}, []byte(body))
}
```

Utilisé pour les emails de reset de mot de passe et autres notifications email.

### 8.4 Port Notifier

```go
// internal/domain/port/notifier.go
type Notifier interface {
    Send(ctx context.Context, event model.Event, config map[string]string) error
}
```

Le `config map[string]string` permet à chaque implémentation de lire sa configuration spécifique (URL du webhook, token Discord, etc.) depuis les `NotificationConfig` stockées en base.

---

## 9. Configuration

### 9.1 Struct Config

**Fichier:** `backend/pkg/config/config.go`

```go
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Database DatabaseConfig `yaml:"database"`
    Docker   DockerConfig   `yaml:"docker"`
    Log      LogConfig      `yaml:"logging"`
    SMTP     SMTPConfig     `yaml:"smtp"`
}

type DatabaseConfig struct {
    Host            string `yaml:"host"`
    Port            int    `yaml:"port"`
    Name            string `yaml:"name"`
    User            string `yaml:"user"`
    Password        string `yaml:"password"`
    SSLMode         string `yaml:"sslmode"`
    MaxConns        int32  `yaml:"max_conns"`
    MinConns        int32  `yaml:"min_conns"`
    MaxConnLifetime string `yaml:"max_conn_lifetime"`
    AutoMigrate     *bool  `yaml:"auto_migrate"`  // défaut: true
}

type DockerConfig struct {
    Host         string `yaml:"host"`          // ex: "tcp://socket-proxy:2375"
    AgentNetwork string `yaml:"agent_network"` // réseau Docker pour les agents
}

type ServerConfig struct {
    Port         int           `yaml:"port"`
    ReadTimeout  time.Duration `yaml:"read_timeout"`
    WriteTimeout time.Duration `yaml:"write_timeout"`
}

type SMTPConfig struct {
    Host        string `yaml:"host"`
    Port        int    `yaml:"port"`
    From        string `yaml:"from"`
    Username    string `yaml:"username"`
    Password    string `yaml:"password"`
    FrontendURL string `yaml:"frontend_url"` // pour les liens dans les emails
}

type LogConfig struct {
    Level string `yaml:"level"`
}
```

### 9.2 Loader

**Fichier:** `internal/config/loader.go`

```go
func Load(path string) (*pkgconfig.Config, error) {
    // 1. os.ReadFile(path) → YAML
    // 2. yaml.Unmarshal → Config struct
    // 3. setDefaults(&cfg)     — AutoMigrate = true si nil
    // 4. applyEnvOverrides(&cfg)
    // 5. validate(&cfg)        — host, name, user, password requis
}
```

**Validation:** seuls `database.host`, `database.name`, `database.user`, `database.password` sont requis. Tout le reste est optionnel (avec valeurs par défaut).

### 9.3 Variables d'environnement

Les variables d'environnement surchargent les valeurs YAML au runtime.

| Variable | Config | Description |
|----------|--------|-------------|
| `DB_HOST` | `database.host` | Hôte PostgreSQL |
| `DB_PORT` | `database.port` | Port PostgreSQL |
| `DB_NAME` | `database.name` | Nom de la base |
| `DB_USER` | `database.user` | Utilisateur |
| `DB_PASSWORD` | `database.password` | Mot de passe |
| `DB_SSLMODE` | `database.sslmode` | Mode SSL (`disable`, `require`) |
| `DB_AUTO_MIGRATE` | `database.auto_migrate` | `true`/`1` pour activer |
| `SERVER_PORT` | `server.port` | Port HTTP |
| `LOG_LEVEL` | `logging.level` | Niveau de log |
| `DOCKER_HOST` | `docker.host` | Socket Docker (ex: `tcp://socket-proxy:2375`) |
| `DOCKER_AGENT_NETWORK` | `docker.agent_network` | Réseau Docker agents |
| `SMTP_HOST` | `smtp.host` | Hôte SMTP |
| `SMTP_PORT` | `smtp.port` | Port SMTP |
| `SMTP_FROM` | `smtp.from` | Adresse expéditeur |
| `SMTP_USERNAME` | `smtp.username` | Auth SMTP (optionnel) |
| `SMTP_PASSWORD` | `smtp.password` | Auth SMTP (optionnel) |
| `FRONTEND_URL` | `smtp.frontend_url` | URL frontend (liens emails) |
| `CLAUDE_CODE_OAUTH_TOKEN` | runtime | Token OAuth Claude Code (agents) |
| `GITHUB_TOKEN` | runtime | Token GitHub (git operations) |
| `{project.GitTokenEnv}` | runtime | Token Git par projet (configurable) |

**Important:** `CLAUDE_CODE_OAUTH_TOKEN` et `GITHUB_TOKEN` ne font pas partie de la struct Config — ils sont lus directement via `os.Getenv()` dans `AgentRunAction` au moment de la création du container.

---

## Schéma d'architecture des adapters

```
Domain Ports (interfaces)
         │
         ├── port.EventPublisher ←── postgres.EventRepo
         ├── port.EventSubscriber ←── postgres.EventBus (LISTEN/NOTIFY)
         ├── port.EventRepository ←── postgres.EventRepo
         │
         ├── port.JobQueue ←── river.JobQueue (River + Postgres)
         │       └── ExecuteRunWorker → service.PipelineExecutor
         │
         ├── port.GitProvider ←── git.GhCliAdapter (gh CLI)
         │                   └── git.GiteaAPIAdapter (HTTP API)
         ├── port.GitProviderFactory ←── git.DefaultGitProviderFactory
         │
         ├── port.TemplateRenderer ←── handlebars.Renderer (raymond)
         │
         ├── port.Notifier ←── discord.Notifier
         │               └── webhook.Notifier
         ├── port.EmailSender ←── smtp.EmailSender
         │
         └── port.Action (ActionRegistry)
                 ├── action.AgentRunAction   (agent_run)
                 ├── action.GitBranchAction  (git_branch)
                 ├── action.GitPRAction      (git_pr)
                 ├── action.CIPollAction     (ci_poll)
                 ├── action.HITLGateAction   (hitl_gate)
                 ├── action.HumanAction      (human)
                 ├── action.NotificationAction (notification)
                 └── action.IncrementalRetryAction (incremental_retry)

Flux événement SSE:
  INSERT INTO events
    → Trigger pg_notify('events', {metadata})
      → EventBus.listenLoop() reçoit notification
        → eventRepo.GetEventByID() enrichissement payload
          → Dispatch aux channels subscribers[projectID]
            → SSEHandler écrit frame SSE au client
```
