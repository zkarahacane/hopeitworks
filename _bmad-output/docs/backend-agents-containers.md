# Backend — Agents, Containers & Docker Runtime

Documentation technique du domaine Agent/Container/Docker du backend hopeitworks.

---

## Table des matieres

1. [Vue d'ensemble](#1-vue-densemble)
2. [Modeles de domaine](#2-modeles-de-domaine)
   - 2.1 [Agent](#21-agent)
   - 2.2 [ContainerOpts](#22-containeropts)
   - 2.3 [LogEvent](#23-logevent)
3. [Ports (interfaces)](#3-ports-interfaces)
   - 3.1 [AgentRepository](#31-agentrepository)
   - 3.2 [ContainerManager](#32-containermanager)
   - 3.3 [LogStreamer](#33-logstreamer)
   - 3.4 [CommandRunner](#34-commandrunner)
4. [Service — AgentService](#4-service--agentservice)
5. [Adapter Docker](#5-adapter-docker)
   - 5.1 [ContainerManager (impl)](#51-containermanager-impl)
   - 5.2 [LogStreamer (impl)](#52-logstreamer-impl)
   - 5.3 [NDJSON Parser](#53-ndjson-parser)
6. [Action — AgentRunAction](#6-action--agentrunaction)
7. [Agent Runtime (container)](#7-agent-runtime-container)
   - 7.1 [entrypoint.sh](#71-entrypointsh)
   - 7.2 [Variables d'environnement](#72-variables-denvironnement)
   - 7.3 [Sequence d'execution](#73-sequence-dexecution)
8. [API Handler — AgentHandler](#8-api-handler--agenthandler)
9. [Lifecycle complet d'un container agent](#9-lifecycle-complet-dun-container-agent)
10. [Tests](#10-tests)
11. [Securite et contraintes](#11-securite-et-contraintes)

---

## 1. Vue d'ensemble

Le domaine Agent/Container est responsable de l'execution des agents IA (Claude Code) dans des containers Docker isoles. Chaque step de pipeline de type `agent_run` lance un container, y injecte le contexte d'execution (prompt rendu, tokens, URL du repo), streame ses logs en NDJSON, capture les couts en tokens, puis nettoie le container.

**Flux general :**

```
Pipeline executor
    -> AgentRunAction.Execute()
        -> Render prompt (TemplateRenderer)
        -> ContainerManager.Create()
        -> ContainerManager.Start()
        -> LogStreamer.StreamLogs()  [goroutine]
        -> container exit (doneCh)
        -> CostService.RecordStepCost()
        -> ContainerManager.Stop() + Remove()  [defer cleanup]
```

**Architecture hexagonale :**

```
handler/AgentHandler
    -> service/AgentService
        -> port/AgentRepository  <- adapter/postgres
action/AgentRunAction
    -> port/ContainerManager  <- adapter/docker/ContainerManager
    -> port/LogStreamer       <- adapter/docker/LogStreamer
    -> port/TemplateRenderer  <- adapter/handlebars
    -> port/EventPublisher    <- adapter/postgres
    -> service/CostService
```

---

## 2. Modeles de domaine

### 2.1 Agent

**Fichier :** `backend/internal/domain/model/agent.go`

```go
type Agent struct {
    ID              uuid.UUID  `json:"id"`
    Name            string     `json:"name"`
    Model           string     `json:"model"`           // ex: "claude-opus-4-6"
    Image           string     `json:"image"`           // ex: "hopeitworks/agent:latest"
    TemplateContent string     `json:"template_content"` // template Handlebars du prompt
    Type            string     `json:"type"`            // "implement", "review", "merge", "retry", "custom"
    Scope           string     `json:"scope"`           // "global" ou "project"
    ProjectID       *uuid.UUID `json:"project_id"`      // nil si scope=global
    CreatedAt       time.Time  `json:"created_at"`
    UpdatedAt       time.Time  `json:"updated_at"`
}

const AgentScopeGlobal  = "global"
const AgentScopeProject = "project"
```

**Semantique :** L'entite Agent est la source de verite pour la configuration d'un agent. Elle porte l'image Docker (`Image`), le modele Claude (`Model`), et le template Handlebars du prompt (`TemplateContent`). Il n'y a pas d'image embarquee dans le backend : tout est configurable par l'administrateur via l'API.

**Scope :**
- `global` : disponible pour tous les projets (ex: agents d'implementation standards). `ProjectID` est nil.
- `project` : scoped a un projet specifique. `ProjectID` est obligatoire.

### 2.2 ContainerOpts

**Fichier :** `backend/internal/domain/model/container.go`

```go
type ContainerOpts struct {
    Image       string            // Image Docker (ex: "hopeitworks/agent:latest")
    Env         []string          // Variables d'env en format KEY=VALUE
    NetworkName string            // Nom du reseau Docker a attacher
    Labels      map[string]string // Metadonnees du container (managed_by, run_id, step_id)
    Memory      int64             // Limite memoire en bytes (0 = illimite)
    CPUs        float64           // Limite CPU (0 = illimite, 1.0 = 1 vCPU)
}
```

**Usage :** `ContainerOpts` est la structure de configuration passee a `ContainerManager.Create()`. Elle est construite par `AgentRunAction.createContainer()` a partir du `RunContext` et des donnees du projet/story.

### 2.3 LogEvent

**Fichier :** `backend/internal/domain/model/log_event.go`

```go
type LogEvent struct {
    RunID        string         `json:"run_id"`
    StepID       string         `json:"step_id"`
    Timestamp    time.Time      `json:"timestamp"`
    Level        string         `json:"level"`        // "info", "warn", "error", "debug"
    Message      string         `json:"message"`
    RawLine      string         `json:"raw_line"`     // ligne brute avant parsing
    IsJSON       bool           `json:"is_json"`
    Data         map[string]any `json:"data,omitempty"`
    Type         string         `json:"type,omitempty"`         // ex: "cost"
    InputTokens  int64          `json:"input_tokens,omitempty"`
    OutputTokens int64          `json:"output_tokens,omitempty"`
    Model        string         `json:"model,omitempty"`
}
```

**Semantique :** Chaque ligne de stdout/stderr du container est parsee en `LogEvent`. Si la ligne est du NDJSON valide, `IsJSON=true` et `Data` contient les champs. Les evenements de type `"cost"` sont intercepts par `AgentRunAction` pour l'accumulation des couts — ils ne sont pas publies dans le bus d'evenements.

---

## 3. Ports (interfaces)

### 3.1 AgentRepository

**Fichier :** `backend/internal/domain/port/agent_repository.go`

```go
type AgentRepository interface {
    CreateAgent(ctx context.Context, agent *model.Agent) (*model.Agent, error)
    GetAgent(ctx context.Context, id uuid.UUID) (*model.Agent, error)
    ListAgentsByProject(ctx context.Context, projectID uuid.UUID) ([]*model.Agent, error)
    ListGlobalAgents(ctx context.Context) ([]*model.Agent, error)
    // Retourne agents du projet + tous les agents globaux
    ListAgentsByProjectMerged(ctx context.Context, projectID uuid.UUID) ([]*model.Agent, error)
    UpdateAgent(ctx context.Context, agent *model.Agent) (*model.Agent, error)
    DeleteAgent(ctx context.Context, id uuid.UUID) error
}
```

**Implementation :** L'implementation Postgres est dans `backend/internal/adapter/postgres/` (generee via sqlc).

### 3.2 ContainerManager

**Fichier :** `backend/internal/domain/port/container_manager.go`

```go
type ContainerManager interface {
    Create(ctx context.Context, opts model.ContainerOpts) (string, error)
    Start(ctx context.Context, containerID string) error
    // SIGTERM + 10s timeout + SIGKILL
    Stop(ctx context.Context, containerID string) error
    // Force remove + suppression volumes
    Remove(ctx context.Context, containerID string) error
    // Bloquant jusqu'a la sortie du container, retourne l'exit code
    Wait(ctx context.Context, containerID string) (int, error)
    // Filtre par labels (ex: managed_by=hopeitworks)
    ListContainers(ctx context.Context, labels map[string]string) ([]ContainerInfo, error)
}

type ContainerInfo struct {
    ID        string
    Labels    map[string]string
    CreatedAt time.Time
}
```

**Note :** `Wait` et le streaming de logs sont deux mecanismes distincts. `AgentRunAction` n'utilise pas `Wait` directement — il utilise `LogStreamer.StreamLogs()` qui retourne un `doneCh` avec l'exit code une fois le container termine.

### 3.3 LogStreamer

**Fichier :** `backend/internal/domain/port/log_streamer.go`

```go
type LogStreamer interface {
    // Retourne deux channels :
    // - logCh : recoit les LogEvent au fil de l'execution
    // - doneCh : recoit l'exit code quand le container se termine
    // Les deux channels sont fermes quand le container sort ou le contexte est annule.
    StreamLogs(ctx context.Context, containerID string, runID string, stepID string) (<-chan model.LogEvent, <-chan int, error)
}
```

### 3.4 CommandRunner

**Fichier :** `backend/internal/domain/port/command_runner.go`

```go
type CommandRunner interface {
    Run(ctx context.Context, workDir string, name string, args ...string) (stdout string, err error)
}
```

**Usage :** Abstraction pour l'execution de commandes shell (ex: `gh` CLI pour les operations Git). Rend les appels CLI testables via mock.

---

## 4. Service — AgentService

**Fichier :** `backend/internal/domain/service/agent_service.go`

`AgentService` contient la logique metier CRUD des agents. Il s'appuie uniquement sur `port.AgentRepository`.

### Methodes

| Methode | Description |
|---------|-------------|
| `Create(ctx, CreateAgentParams)` | Valide et cree un nouvel agent |
| `GetByID(ctx, id)` | Recupere un agent par ID |
| `ListByProject(ctx, projectID)` | Liste les agents scopes au projet |
| `ListGlobal(ctx)` | Liste les agents globaux |
| `ListMerged(ctx, projectID)` | Liste projet + globaux (vue unifiee) |
| `Update(ctx, UpdateAgentParams)` | Met a jour name/model/image/template_content |
| `Delete(ctx, id)` | Supprime un agent (verifie l'existence avant) |

### Validation a la creation

```go
type CreateAgentParams struct {
    ProjectID       *uuid.UUID
    Name            string
    Model           string
    Image           string
    TemplateContent string
    Scope           string // defaut: "project"
}
```

Regles de validation :
- `name` : obligatoire, max 255 caracteres
- `template_content` : obligatoire (ne peut pas etre vide)
- `scope` : doit etre `"global"` ou `"project"` (defaut: `"project"`)
- `project_id` : obligatoire si `scope == "project"`

### Validation a la mise a jour

```go
type UpdateAgentParams struct {
    ID              uuid.UUID
    Name            *string   // pointeur = champ optionnel (nil = pas de changement)
    Model           *string
    Image           *string
    TemplateContent *string
}
```

Seuls les champs non-nil sont mis a jour (patch semantique). Le scope et le project_id ne sont pas modifiables apres creation.

---

## 5. Adapter Docker

### 5.1 ContainerManager (impl)

**Fichier :** `backend/internal/adapter/docker/container_manager.go`

#### Construction

```go
func NewDockerContainerManager(dockerHost string, logger *slog.Logger) (*ContainerManager, error)
```

Se connecte a Docker via l'URL `dockerHost` (ex: `"tcp://socket-proxy:2375"` en dev). Utilise le SDK Docker avec negociation automatique de version d'API.

#### Interface mockable

```go
type dockerClient interface {
    ContainerCreate(...)
    ContainerStart(...)
    ContainerStop(...)
    ContainerRemove(...)
    ContainerWait(...)
    ContainerList(...)
}
```

Le type `dockerClient` est une interface interne qui expose uniquement le sous-ensemble du SDK Docker utilise. Cela permet l'injection d'un mock dans les tests unitaires sans Docker reel.

#### Create

```go
func (m *ContainerManager) Create(ctx context.Context, opts model.ContainerOpts) (string, error)
```

**Comportement :**
1. Ajoute automatiquement le label `managed_by=hopeitworks` (meme si `Labels` est nil)
2. Configure `ContainerConfig` : image, env vars, labels
3. Configure `HostConfig` : pas de mode privileged, pas de binds/mounts, ressources CPU/memoire
4. Configure le reseau si `NetworkName` est non-vide
5. Retourne le container ID (string Docker)

**Contraintes de securite hardcodees :**
- `Privileged: false` (jamais de mode privileged)
- `Binds: nil` (aucun montage du filesystem hote)

**Resources :**
- Memoire : `Memory = opts.Memory` (0 = Docker unlimited)
- CPU : `NanoCPUs = int64(opts.CPUs * 1e9)` (0 = Docker unlimited)

#### Start / Stop / Remove / Wait

```go
// Stop : SIGTERM + 10 secondes + SIGKILL
Stop(ctx, containerID) // timeout hardcode a 10s

// Remove : force=true, volumes=true
Remove(ctx, containerID)

// Wait : bloquant sur WaitConditionNotRunning, retourne exit code
Wait(ctx, containerID) // gere ctx.Done()
```

**Gestion d'erreurs :** Toutes les erreurs Docker sont encapsulees dans `apperrors.NewContainerError()` avec le code `CONTAINER_OPERATION_FAILED`.

#### ListContainers

```go
ListContainers(ctx, labels map[string]string) ([]ContainerInfo, error)
```

Filtre via les labels Docker (`All: true` pour inclure les containers arretes). Utile pour inventorier les containers hopeitworks orphelins.

### 5.2 LogStreamer (impl)

**Fichier :** `backend/internal/adapter/docker/log_streamer.go`

#### Construction

```go
func NewDockerLogStreamer(client logStreamClient, logger *slog.Logger) *LogStreamer
func NewDockerLogStreamerFromHost(dockerHost string, logger *slog.Logger) (*LogStreamer, error)
```

`DefaultIdleTimeout = 60 * time.Second` — temps d'attente sans output avant d'emettre un warning.

#### StreamLogs

```go
func (s *LogStreamer) StreamLogs(ctx, containerID, runID, stepID) (<-chan model.LogEvent, <-chan int, error)
```

**Fonctionnement interne :**

1. `ContainerLogs()` avec `Follow: true`, `ShowStdout: true`, `ShowStderr: true`
2. Demultiplexage du stream Docker via `stdcopy.StdCopy()` (les logs Docker sont encapsules dans des frames de 8 bytes header stdout/stderr)
3. Pipe vers un `bufio.Scanner` pour lecture ligne par ligne
4. Chaque ligne est parsee via `parseNDJSONLine()`
5. Idle timer reinitialisee a chaque ligne recue ; si expire, un `LogEvent{Level: "warn"}` est emis
6. A la fin du stream (EOF ou erreur scanner), `handleContainerExit()` appelle `ContainerWait()` avec un timeout de 30s independant pour recuperer l'exit code
7. Les deux channels `logCh` et `doneCh` sont fermes en `defer`

**Gestion de l'annulation du contexte :**
- Si le contexte est annule, la goroutine de streaming s'arrete (`case <-ctx.Done()`)
- `handleContainerExit` utilise un `context.Background()` avec timeout 30s pour recuperer l'exit code meme si le contexte parent est annule

**Goroutines lancees :**
```
StreamLogs()
  |- goroutine: stdcopy.StdCopy(pw, pw, reader)  // demux Docker frames
  |- goroutine: scanner.Scan() -> lineCh          // lecture lignes
  \- goroutine: streamLoop()                      // select sur lineCh/ctx/timer
```

### 5.3 NDJSON Parser

**Fichier :** `backend/internal/adapter/docker/ndjson_parser.go`

```go
func parseNDJSONLine(line string, runID string, stepID string) *model.LogEvent
```

**Comportement :**
- Ligne vide ou espaces uniquement -> `nil` (skip)
- JSON invalide -> `LogEvent{IsJSON: false, Level: "info", Message: line}`
- JSON valide -> extraction de `level`, `message`, `timestamp`, `type`

**Evenements de cout — deux formats supportes :**

```json
// Format 1 : evenement custom emis par l'entrypoint
{"type":"cost","input_tokens":1000,"output_tokens":200,"model":"claude-opus-4-6"}

// Format 2 : evenement "result" du stream-json de Claude Code (autoritatif)
{
  "type": "result",
  "usage": {"input_tokens": 12450, "output_tokens": 2310},
  "modelUsage": {
    "claude-opus-4-6-20251101": {"inputTokens": 10200, "outputTokens": 1980, "costUSD": 0.0842}
  }
}
```

Les evenements `"result"` sont normalises en `type="cost"`. Pour `modelUsage` avec plusieurs modeles, `pickPrimaryModel()` selectionne le modele avec le plus grand `inputTokens`.

---

## 6. Action — AgentRunAction

**Fichier :** `backend/internal/adapter/action/agent_run.go`

`AgentRunAction` est l'implementation de `model.Action` pour le type de step `agent_run`. C'est le point d'orchestration central du cycle de vie d'un container.

### Configuration

```go
type AgentConfig struct {
    DefaultMemory int64   // memoire par defaut (ex: 4294967296 = 4GB)
    DefaultCPUs   float64 // CPU par defaut (ex: 2.0)
    NetworkName   string  // reseau Docker des containers agents
    LogTailLines  int     // nb de lignes conservees pour le log tail (defaut: 50)
}
```

### Dependencies injectees

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

### Execute — etapes detaillees

```go
func (a *AgentRunAction) Execute(ctx context.Context, runCtx *model.RunContext) error
```

**Etape 1 — Fetch story**
```go
story, err := a.storyRepo.GetByID(ctx, runCtx.StoryID)
```

**Etape 2 — Fetch project**
```go
project, err := a.projectRepo.GetByID(ctx, runCtx.ProjectID)
```

**Etape 3 — Rendu du prompt**
```go
// Recupere template_content depuis les metadonnees du step
templateContent, _ := runCtx.Metadata["template_content"].(string)
branchName, _ := runCtx.Metadata["branch_name"].(string)

tmplCtx := &model.TemplateContext{
    StoryKey:           story.Key,
    StoryTitle:         story.Title,
    StoryObjective:     derefString(story.Objective),
    TargetFiles:        story.TargetFiles,
    AcceptanceCriteria: derefString(story.AcceptanceCriteria),
    BranchName:         branchName,
    RepoURL:            repoURL,
    // Contexte de retry si present :
    ErrorContext: runCtx.Metadata["error_context"].(string),
    LogTail:      runCtx.Metadata["log_tail"].(string),
}
prompt, err = a.renderer.Render(templateContent, tmplCtx)
```

**Etape 4 — Resolution de l'image**
```go
agentImage, _ := runCtx.Metadata["agent_image"].(string)
// obligatoire — erreur si absent
```

**Etape 5 — Creation du container**

Variables d'environnement injectees :

| Variable | Source |
|----------|--------|
| `REPO_URL` | `project.RepoURL` |
| `BRANCH_NAME` | `runCtx.Metadata["branch_name"]` |
| `STORY_KEY` | `story.Key` |
| `PROMPT_CONTENT` | prompt rendu par le template |
| `GIT_TOKEN` | `os.Getenv(project.GitTokenEnv)` (defaut: `GITHUB_TOKEN`) |
| `GIT_PROVIDER` | `project.GitProvider` (ex: "github" ou "gitea") |
| `GITHUB_TOKEN` | meme valeur que `GIT_TOKEN` (compat backward) |
| `CLAUDE_CODE_OAUTH_TOKEN` | `os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")` |
| `MODEL` | `runCtx.Metadata["model"]` (optionnel) |

Labels Docker poses :

| Label | Valeur |
|-------|--------|
| `managed_by` | `"hopeitworks"` |
| `run_id` | `runCtx.Run.ID.String()` |
| `step_id` | `runCtx.RunStep.ID.String()` |
| `story_key` | `story.Key` |

**Etape 6 — Start**
```go
a.containerMgr.Start(ctx, containerID)
```

**Etape 7 — Persistance du container ID**
```go
a.runRepo.UpdateRunStepContainerInfo(ctx, runCtx.RunStep.ID, &containerID, nil)
```
Non-fatal si l'appel echoue (log warn uniquement).

**Etape 8 — Stream logs + attente sortie**

```go
logCh, doneCh, err := a.logStreamer.StreamLogs(ctx, containerID, runID, stepID)

// Ring buffer de LogTailLines entrees pour le log tail
logTail := make([]string, 0, tailSize)
var costEvents []model.CostEvent

go func() {
    for logEvent := range logCh {
        // Maintenance du ring buffer
        logTail = append(logTail, logEvent.Message)

        if logEvent.Type == "cost" {
            costEvents = append(costEvents, ...)
            continue // les evenements cost ne sont pas publies
        }
        a.publishLogEvent(ctx, runCtx, logEvent) // -> EventPublisher
    }
}()

exitCode := <-doneCh
```

**Etape 9 — Gestion exit code**
- `exitCode != 0` -> persistance du log tail + retour d'erreur
- `exitCode == 0` -> succes

**Etape 10 — Enregistrement des couts**
```go
a.costSvc.RecordStepCost(ctx, runCtx.RunStep.ID, runCtx.ProjectID, costEvents, agentID)
// Non-fatal (log warn si echec)
```

**Cleanup (defer)**
```go
defer a.cleanupContainer(containerID)
// Timeout independant de 30s
// Stop() + Remove() — erreurs loggees en warn, non-fatales
```

### Publication des log events

Chaque `LogEvent` (hors `type=cost`) est publie via `EventPublisher` sous la forme :

```go
model.Event{
    EntityType: "log",
    EntityID:   runCtx.RunStep.ID,
    Action:     "emitted",
    Payload:    json.Marshal(logEvent),
}
```

Ces evenements sont consommes par le SSE handler pour le streaming temps-reel vers le frontend.

---

## 7. Agent Runtime (container)

### 7.1 entrypoint.sh

**Fichier :** `agent/entrypoint.sh`

Script Bash qui s'execute a l'interieur du container agent. Il est le seul point d'entree du container.

### 7.2 Variables d'environnement

**Obligatoires :**

| Variable | Description |
|----------|-------------|
| `REPO_URL` | URL HTTPS du depot git |
| `BRANCH_NAME` | Branche a checkout (creee si inexistante) |
| `PROMPT_CONTENT` | Prompt rendu par le backend (contenu complet) |
| `GIT_TOKEN` | Token git pour authentification clone/push |
| `CLAUDE_CODE_OAUTH_TOKEN` | Token OAuth pour Claude Code |

**Optionnelles :**

| Variable | Description | Defaut |
|----------|-------------|--------|
| `CLAUDE_MD_CONTENT` | Contenu CLAUDE.md a injecter | (non injecte si vide) |
| `GIT_PROVIDER` | Fournisseur git | `"github"` |
| `STORY_KEY` | Cle de la story pour le contexte git | - |
| `GIT_AUTHOR_NAME` | Nom de l'auteur git | `"hopeitworks-agent"` |
| `GIT_AUTHOR_EMAIL` | Email de l'auteur git | `"agent@hopeitworks.local"` |

### 7.3 Sequence d'execution

```
1. Validation des variables obligatoires
   -> exit 1 si manquante + emit_log("error")

2. Configuration git identity
   git config --global user.name "$GIT_AUTHOR_NAME"
   git config --global user.email "$GIT_AUTHOR_EMAIL"

3. Configuration GitHub CLI (si GIT_PROVIDER == "github")
   export GH_TOKEN="$GIT_TOKEN"

4. Clone du depot
   CLONE_URL="https://${GIT_TOKEN}@${host}/owner/repo"
   git clone "$CLONE_URL" /workspace

5. Checkout de la branche cible
   - Si la branche existe en remote : fetch + checkout
   - Sinon : git checkout -b "$BRANCH_NAME"

6. Configuration du remote pour le push
   git remote set-url origin "https://${GIT_TOKEN}@..."

7. Injection CLAUDE.md (optionnel)
   if [[ -n "$CLAUDE_MD_CONTENT" ]]; then
       echo "$CLAUDE_MD_CONTENT" > /workspace/CLAUDE.md
   fi

8. Ecriture du prompt
   echo "$PROMPT_CONTENT" > /tmp/prompt.md

9. Execution Claude Code
   claude --dangerously-skip-permissions \
          --print \
          --verbose \
          --output-format stream-json \
          < /tmp/prompt.md

10. Propagation de l'exit code
    exit "$EXIT_CODE"
```

**Format de log :** Toutes les lignes emises par `emit_log()` sont du NDJSON :
```json
{"type":"log","level":"info","message":"Cloning repository...","timestamp":"2026-02-26T10:00:00Z"}
```

**Sortie de Claude Code (`--output-format stream-json`) :** Claude Code emet son propre NDJSON, incluant les evenements `"result"` avec `usage` et `modelUsage`. Ces evenements sont interceptes par le `ndjson_parser.go` pour extraire les couts.

**Securite :**
- Le token git est injecte directement dans l'URL de clone (jamais stocke en clair sur disk)
- `CLAUDE_CODE_OAUTH_TOKEN` est reconnu nativement par Claude Code (authMethod: oauth_token) — il ne faut PAS le remapper vers `ANTHROPIC_API_KEY`
- Le container travaille dans `/workspace` (isolation complete, pas d'acces au filesystem hote)

---

## 8. API Handler — AgentHandler

**Fichier :** `backend/internal/api/handler/agent_handler.go`

`AgentHandler` implemente les routes REST pour la gestion des agents. Les types (`Agent`, `CreateAgentRequest`, `UpdateAgentRequest`, etc.) sont generes par oapi-codegen depuis `api/openapi.yaml`.

### Routes implementees

| Methode | Route | Handler | Auth requise |
|---------|-------|---------|--------------|
| `GET` | `/api/v1/agents` | `ListGlobalAgents` | User |
| `GET` | `/api/v1/projects/{projectId}/agents` | `ListProjectAgents` | User |
| `POST` | `/api/v1/projects/{projectId}/agents` | `CreateAgent` | **Admin** |
| `GET` | `/api/v1/projects/{projectId}/agents/{agentId}` | `GetAgent` | User |
| `PUT` | `/api/v1/projects/{projectId}/agents/{agentId}` | `UpdateAgent` | **Admin** |
| `DELETE` | `/api/v1/projects/{projectId}/agents/{agentId}` | `DeleteAgent` | **Admin** |

### Detail des handlers

**ListGlobalAgents** — `GET /agents`
- Retourne uniquement les agents `scope=global`
- Reponse paginee : `{ data: [...], pagination: { total, page, per_page } }`

**ListProjectAgents** — `GET /projects/{projectId}/agents`
- Retourne les agents du projet PLUS les agents globaux (vue merged)
- Utilise `AgentService.ListMerged()`

**CreateAgent** — `POST /projects/{projectId}/agents`
- Requiert role `admin` (403 sinon)
- Scope par defaut : `"project"` si non specifie
- `project_id` pris de l'URL path, pas du body

**UpdateAgent** — `PUT /projects/{projectId}/agents/{agentId}`
- Requiert role `admin`
- Mise a jour partielle : seuls les champs presents dans le body sont modifies

**DeleteAgent** — `DELETE /projects/{projectId}/agents/{agentId}`
- Requiert role `admin`
- Retourne `204 No Content` en cas de succes

### Conversion domaine -> API

```go
func toAPIAgent(a *model.Agent) Agent {
    return Agent{
        Id:              a.ID,
        Name:            a.Name,
        Model:           a.Model,
        Image:           a.Image,
        TemplateContent: a.TemplateContent,
        Scope:           AgentScope(a.Scope),
        ProjectId:       a.ProjectID,
        CreatedAt:       a.CreatedAt,
        UpdatedAt:       a.UpdatedAt,
    }
}
```

---

## 9. Lifecycle complet d'un container agent

Sequence complete de la creation au nettoyage :

```
[Pipeline Executor]
    |
    | 1. Execute action "agent_run"
    v
[AgentRunAction.Execute(ctx, runCtx)]
    |
    | 2. Fetch story + project (DB)
    | 3. Render prompt via TemplateRenderer (Handlebars)
    | 4. Resolve agentImage from metadata
    |
    | 5. createContainer()
    |    -> ContainerManager.Create(ContainerOpts{
    |         Image:       agentImage,
    |         Env:         [REPO_URL, BRANCH_NAME, ...],
    |         NetworkName: "hopeitworks-net",
    |         Memory:      4GB,
    |         CPUs:        2.0,
    |         Labels:      {managed_by, run_id, step_id, story_key},
    |       })
    |    -> Docker API: ContainerCreate()
    |    <- containerID
    |
    | 6. ContainerManager.Start(containerID)
    |    -> Docker API: ContainerStart()
    |
    | 7. runRepo.UpdateRunStepContainerInfo(containerID)  // persist
    |
    | 8. streamAndWait()
    |    -> LogStreamer.StreamLogs(containerID, runID, stepID)
    |       -> Docker API: ContainerLogs(Follow: true)
    |       -> goroutine: stdcopy demux -> scanner -> lineCh
    |       <- logCh chan LogEvent
    |       <- doneCh chan int  (exit code)
    |
    |    LOOP: goroutine consomme logCh
    |      |
    |      |-- LogEvent.Type == "cost" -> accumulate costEvents
    |      |-- LogEvent sinon          -> EventPublisher.Publish("log.emitted")
    |                                     -> SSE vers frontend
    |
    |    <- doneCh : exitCode
    |
    | 9. exitCode != 0 ?
    |      -> persistLogTail (ring buffer des dernieres lignes)
    |      -> retour erreur
    |
    | 10. CostService.RecordStepCost(costEvents)
    |
    | 11. defer cleanupContainer(containerID)
    |     -> ContainerManager.Stop()   (SIGTERM + 10s + SIGKILL)
    |     -> ContainerManager.Remove() (force + volumes)
    v
[Done]


[DANS LE CONTAINER - entrypoint.sh]
    |
    | 1. Validation env vars
    | 2. git config (identity)
    | 3. git clone https://token@repo /workspace
    | 4. git checkout branch (existante ou nouvelle)
    | 5. git remote set-url (token dans URL pour push)
    | 6. echo "$CLAUDE_MD_CONTENT" > /workspace/CLAUDE.md  (optionnel)
    | 7. echo "$PROMPT_CONTENT" > /tmp/prompt.md
    | 8. claude --dangerously-skip-permissions \
    |           --print --verbose \
    |           --output-format stream-json \
    |           < /tmp/prompt.md
    |    -> emit NDJSON sur stdout (log events + result event avec usage)
    | 9. exit $EXIT_CODE
```

### Etats et transitions du container

```
[inexistant]
    -> Create()  -> [created]
    -> Start()   -> [running]
                    [running] : logs streames, exit code attendu
    -> Stop()    -> [stopped]  (ou directement si exit naturel)
    -> Remove()  -> [supprime]
```

---

## 10. Tests

### Tests unitaires — docker/container_manager_test.go

**Strategie :** mock de l'interface `dockerClient` (sous-ensemble du SDK Docker).

Cas couverts :
- `TestCreate_Success` : verification des opts transmises au SDK
- `TestCreate_SecurityConstraints` : verifie `Privileged=false` et `Binds=nil`
- `TestCreate_ManagedByLabelAddedWhenLabelsNil` : label auto-ajoute meme si Labels=nil
- `TestCreate_MemoryAndCPULimits` : conversion correcte en bytes et nanoCPUs
- `TestCreate_ZeroLimitsNotApplied` : 0 = pas de limite (Docker unlimited)
- `TestCreate_NetworkConfig` : EndpointsConfig correctement construit
- `TestCreate_NoNetworkWhenEmpty` : NetworkName="" -> nil networkingConfig
- `TestCreate_Error` : Docker erreur wrappee en DomainError (code CONTAINER_OPERATION_FAILED)
- `TestStart_*`, `TestStop_*`, `TestRemove_*` : success + error paths
- `TestStop_Success` : verifie timeout=10s dans StopOptions
- `TestRemove_Success` : verifie Force=true et RemoveVolumes=true
- `TestWait_SuccessExitZero/NonZeroExit` : exit codes propages correctement
- `TestWait_ContextCancelled` : annulation contexte -> DomainError
- `TestListContainers_*` : filtrage par labels, All=true

### Tests unitaires — docker/log_streamer_test.go

**Strategie :** mock de `logStreamClient` + fabrication de streams Docker multiplexes via `stdcopy.NewStdWriter`.

Cas couverts :
- `TestStreamLogs_ValidNDJSON` : 3 lignes NDJSON -> 3 LogEvent avec RunID/StepID corrects
- `TestStreamLogs_MixedValidInvalidJSON` : mix NDJSON/plain text -> IsJSON correct
- `TestStreamLogs_ContainerExit` : exit code 42 recu sur doneCh
- `TestStreamLogs_ContextCancellation` : channels fermes proprement sur annulation
- `TestStreamLogs_ContainerLogsError` : erreur Docker -> DomainError (ErrCodeLogStreamFailed)
- `TestStreamLogs_EmptyLinesSkipped` : lignes vides ignorees (count=2 sur 4 lignes)
- `TestStreamLogs_ChannelsClosed` : logCh et doneCh fermes apres EOF
- `TestStreamLogs_IdleTimeout` : warning "warn" emis apres 100ms sans output
- `TestStreamLogs_IdleTimeoutResets` : timer reinitialisee a chaque ligne recue
- `TestStreamLogs_WaitError` : echec ContainerWait -> doneCh ferme sans valeur

### Tests unitaires — docker/ndjson_parser_test.go

Cas couverts (table-driven) :
- JSON valide avec tous les champs
- JSON valide sans `level` (defaut: "info")
- JSON valide sans `timestamp` (defaut: time.Now())
- JSON valide sans `message` (defaut: "")
- JSON invalide -> wrape comme texte brut (IsJSON=false)
- Ligne vide -> nil
- Ligne d'espaces -> nil
- Timestamp invalide -> fallback time.Now()
- Tableau JSON `[...]` -> traite comme texte brut

Cas specifiques `result` events :
- `result` avec usage simple -> normalise en `type=cost`
- `result` avec plusieurs modeles -> `pickPrimaryModel` selectionne celui avec max inputTokens
- `result` sans `modelUsage` -> cost event sans Model

### Tests unitaires — action/agent_run_test.go

**Strategie :** fixture complete avec mocks de toutes les dependances.

```go
type agentRunFixture struct {
    containerMgr *mockContainerManager  // capture createCalls, startCalls, etc.
    logStreamer   *mockLogStreamer       // configurable via streamLogsFn
    eventPub     *mockEventPublisher    // capture events publies
    storyRepo    *mockStoryRepo
    projectRepo  *mockProjectRepo
    runRepo      *mockRunRepo           // capture containerInfoCalls
    costSvc      *service.CostService  // reel avec mockCostRepo
    action       *action.AgentRunAction
}
```

Cas couverts :
- `TestAgentRunAction_HappyPath` : verification complete de opts, labels, env vars, persist, events, cleanup
- `TestAgentRunAction_MissingAgentImage` : erreur avant creation container
- `TestAgentRunAction_EmptyTemplateContent` : PROMPT_CONTENT vide (ok)
- `TestAgentRunAction_AgentFailure` : exit code 1 -> log tail persiste + cleanup
- `TestAgentRunAction_ContainerCreateFailure` : Start non appele apres echec Create
- `TestAgentRunAction_ContextCancellation` : ctx.Err() propage, cleanup effectue
- `TestAgentRunAction_StoryNotFound` : erreur avant creation container
- `TestAgentRunAction_ProjectNotFound` : erreur avant creation container
- `TestAgentRunAction_CleanupOnAllPaths` : cleanup effectue meme si Start echoue
- `TestAgentRunAction_BranchNameFromMetadata` : BRANCH_NAME correct dans env
- `TestAgentRunAction_ModelFromMetadata` : MODEL injecte si present dans metadata
- `TestAgentRunAction_ModelFallback` : pas de MODEL env var si absent

### Tests unitaires — handler/agent_handler_test.go

**Strategie :** mock de `port.AgentRepository` (in-memory map), service reel `AgentService`.

Cas couverts :
- `TestCreateAgent_AdminOnly` : admin=201, user=403, pas de role=403
- `TestCreateAgent_Validation` : name vide=400, template vide=400, JSON invalide=400, valide=201
- `TestGetAgent_Found` : 200 + verification du body JSON
- `TestGetAgent_NotFound` : 404
- `TestUpdateAgent_AdminOnly` : user=403, admin=200
- `TestDeleteAgent_AdminOnly` : user=403, admin=204
- `TestDeleteAgent_NotFound` : admin + UUID inconnu=404
- `TestListGlobalAgents` : 3 globaux + 1 projet -> 3 retournes
- `TestListProjectAgents` : 2 projet + 1 global -> 3 retournes (merged)

---

## 11. Securite et contraintes

### Contraintes Docker hardcodees

| Contrainte | Valeur | Justification |
|------------|--------|---------------|
| `Privileged` | `false` | jamais de mode privileged pour les containers agent |
| `Binds` | `nil` | pas d'acces au filesystem hote |
| `managed_by` label | `"hopeitworks"` | toujours present pour inventaire/cleanup |

### Acces Docker

Le backend ne se connecte jamais directement au socket Docker. En developpement local, il passe par `socket-proxy` (TCP `tcp://socket-proxy:2375`) qui expose un sous-ensemble securise de l'API Docker.

### Secrets

- Tokens git injectes via variables d'environnement (jamais loggues — `ScrubHandler` redacte les champs sensibles dans les logs slog)
- `CLAUDE_CODE_OAUTH_TOKEN` : passe directement tel quel, reconnu nativement par Claude Code
- Pas de secrets en dur dans le code — tout vient de `os.Getenv()`

### Isolation des agents

- Chaque step de pipeline obtient son propre container isole
- Le container clone une copie independante du repo dans `/workspace`
- Pas de partage de volume entre containers
- Timeout de cleanup : 30s maximum pour Stop + Remove (contexte independant)

### Labels obligatoires

Tout container cree par hopeitworks porte :
```
managed_by=hopeitworks
run_id=<uuid>
step_id=<uuid>
story_key=<string>
```

Ces labels permettent l'inventaire et le cleanup des containers orphelins via `ListContainers()`.
