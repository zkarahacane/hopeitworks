# Backend Documentation — Stories, Epics, Pipeline Config, Cost Tracking, HITL, Notifications

## Table des matières

1. [Vue d'ensemble de l'architecture](#1-vue-densemble-de-larchitecture)
2. [Domaine Stories](#2-domaine-stories)
   - [Modèle](#21-modèle)
   - [Port (interface repository)](#22-port-interface-repository)
   - [Service](#23-service)
   - [Import de stories (markdown)](#24-import-de-stories-markdown)
   - [API Handler](#25-api-handler)
   - [Tests](#26-tests)
3. [Domaine Epics](#3-domaine-epics)
   - [Modèle](#31-modèle)
   - [Port (interface repository)](#32-port-interface-repository)
   - [Service](#33-service)
   - [API Handler](#34-api-handler)
   - [Tests](#35-tests)
4. [Domaine Pipeline Config](#4-domaine-pipeline-config)
   - [Modèles](#41-modèles)
   - [Port (interface repository)](#42-port-interface-repository)
   - [Service](#43-service)
   - [API Handler](#44-api-handler)
   - [Tests](#45-tests)
5. [Domaine Cost Tracking](#5-domaine-cost-tracking)
   - [Modèles](#51-modèles)
   - [Port (interface repository)](#52-port-interface-repository)
   - [Service](#53-service)
   - [API Handler](#54-api-handler)
   - [Tests](#55-tests)
6. [Domaine HITL (Human-in-the-Loop)](#6-domaine-hitl-human-in-the-loop)
   - [Modèle](#61-modèle)
   - [Port (interface repository)](#62-port-interface-repository)
   - [Service](#63-service)
   - [API Handler](#64-api-handler)
   - [Tests](#65-tests)
7. [Domaine Notifications](#7-domaine-notifications)
   - [Modèle](#71-modèle)
   - [Ports](#72-ports)
   - [Service NotificationConfigService](#73-service-notificationconfigservice)
   - [Service NotificationDispatcher](#74-service-notificationdispatcher)
   - [API Handler](#75-api-handler)
   - [Tests](#76-tests)
8. [Modèle LogEvent](#8-modèle-logevent)
9. [Flux de données transversaux](#9-flux-de-données-transversaux)

---

## 1. Vue d'ensemble de l'architecture

Ces domaines respectent strictement l'architecture hexagonale du backend hopeitworks :

```
handler → service → port ← adapter (postgres)
```

- Les **handlers** (`internal/api/handler/`) valident la requête HTTP et délèguent au service.
- Les **services** (`internal/domain/service/`) contiennent la logique métier et dépendent d'interfaces (ports).
- Les **ports** (`internal/domain/port/`) sont les interfaces que les adapters doivent implémenter.
- Les **modèles** (`internal/domain/model/`) sont des structs Go purs, sans dépendances externes.

Constantes de validation partagées (fichier `internal/domain/model/validation.go`) :

```go
const (
    MaxNameLength               = 255
    MaxStoryKeyLength           = 50
    MaxProjectDescriptionLength = 1000
    MaxEpicDescriptionLength    = 2000
)
```

---

## 2. Domaine Stories

### 2.1 Modèle

Fichier : `internal/domain/model/story.go`

```go
// Statuts possibles d'une story
const (
    StoryStatusBacklog = "backlog"
    StoryStatusRunning = "running"
    StoryStatusDone    = "done"
    StoryStatusFailed  = "failed"
)

// Scopes possibles d'une story
const (
    StoryScopeBackend  = "backend"
    StoryScopeFrontend = "frontend"
    StoryScopeShared   = "shared"
)

type Story struct {
    ID                 uuid.UUID
    ProjectID          uuid.UUID
    EpicID             *uuid.UUID   // nullable — story peut exister sans epic
    Key                string       // format validé : [A-Z0-9]+-\d+ (ex: S-14, STORY-123)
    Title              string
    Objective          *string
    TargetFiles        []string     // fichiers ciblés par l'agent
    DependsOn          []string     // clés des stories dont celle-ci dépend (pour le DAG)
    Scope              *string      // "backend", "frontend", "shared"
    Status             string
    AcceptanceCriteria *string
    CreatedAt          time.Time
    UpdatedAt          time.Time
}
```

Le champ `DependsOn` contient des clés de stories (strings, pas des UUIDs) et est utilisé par le DAG scheduler pour construire le graphe de dépendances des stories d'un epic.

### 2.2 Port (interface repository)

Fichier : `internal/domain/port/story_repository.go`

```go
type StoryRepository interface {
    Create(ctx context.Context, story *model.Story) (*model.Story, error)
    GetByID(ctx context.Context, id uuid.UUID) (*model.Story, error)
    GetByKey(ctx context.Context, projectID uuid.UUID, key string) (*model.Story, error)
    ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Story, error)
    ListByStatus(ctx context.Context, projectID uuid.UUID, statuses []string, limit, offset int32) ([]*model.Story, error)
    ListByEpic(ctx context.Context, epicID uuid.UUID, limit, offset int32) ([]*model.Story, error)
    CountByProject(ctx context.Context, projectID uuid.UUID) (int64, error)
    CountByStatus(ctx context.Context, projectID uuid.UUID, statuses []string) (int64, error)
    Update(ctx context.Context, story *model.Story) (*model.Story, error)
    Delete(ctx context.Context, id uuid.UUID) error
}
```

Notes :
- `GetByKey` est utilisé pour le lookup par clé métier (ex: "S-14") dans le contexte d'un projet.
- `ListByEpic` est utilisé par le handler epic pour construire le DAG.
- `ListByStatus` supporte le filtrage multi-statuts (slice), permettant par exemple de récupérer `backlog` + `failed` en une requête.

### 2.3 Service

Fichier : `internal/domain/service/story_service.go`

```go
type StoryService struct {
    repo port.StoryRepository
}
```

#### Création

```go
type CreateStoryParams struct {
    ProjectID          uuid.UUID
    EpicID             *uuid.UUID
    Key                string
    Title              string
    Objective          *string
    TargetFiles        []string
    DependsOn          []string
    Scope              *string
    Status             string       // défaut : "backlog" si vide
    AcceptanceCriteria *string
}
```

Règles de validation appliquées dans `Create` :
- `Key` : obligatoire, max 50 caractères, format regex `^[A-Z0-9]+-\d+$`
- `Title` : obligatoire, max 255 caractères
- `ProjectID` : obligatoire (uuid.Nil rejeté)
- `Status` : doit être parmi `backlog`, `running`, `done`, `failed` (défaut : `backlog`)
- `Scope` : si présent, doit être parmi `backend`, `frontend`, `shared`

#### Mise à jour

```go
type UpdateStoryParams struct {
    ID                 uuid.UUID
    Title              *string      // pointer = champ optionnel (patch sémantique)
    Objective          *string
    TargetFiles        *[]string
    DependsOn          *[]string
    Scope              *string
    Status             *string
    AcceptanceCriteria *string
    EpicID             *uuid.UUID
}
```

Le service récupère la story existante (`GetByID`), applique les champs non-nil, puis appelle `Update`. Cela implémente une sémantique de PATCH partiel tout en passant par l'endpoint PUT.

#### Autres méthodes

| Méthode | Description |
|---------|-------------|
| `GetByID(ctx, id)` | Délègue directement au repo |
| `GetByKey(ctx, projectID, key)` | Lookup par clé métier |
| `ListByProject(ctx, projectID, page, perPage)` | Liste paginée — retourne `StoryListResult{Stories, Total}` |
| `ListByStatus(ctx, projectID, statuses, page, perPage)` | Liste filtrée par statuts multiples |
| `Delete(ctx, id)` | Vérifie l'existence avant de supprimer |

### 2.4 Import de stories (markdown)

Fichier : `internal/domain/service/story_import.go`

L'import est une méthode du `StoryService` qui traite un batch de stories parsées depuis un fichier markdown. Chaque story est traitée indépendamment (pas de transaction globale) pour permettre le succès partiel.

```go
type ImportStoryInput struct {
    Key                string
    Title              string
    Epic               string
    DependsOn          []string
    Scope              string
    Status             string
    AcceptanceCriteria string
    ParseError         error    // erreur de parsing YAML frontmatter
}

type ImportResult struct {
    Imported int
    Updated  int
    Failed   int
    Errors   []ImportStoryError
}

type ImportStoryError struct {
    Key     string
    Message string
    Code    string  // "YAML_PARSE_ERROR", "VALIDATION_ERROR", "IMPORT_ERROR"
}
```

Logique de l'import :

1. Si `ParseError != nil` → enregistre comme erreur `YAML_PARSE_ERROR`, continue.
2. Si `Key` vide → erreur `VALIDATION_ERROR`.
3. Si `Title` vide → erreur `VALIDATION_ERROR`.
4. Lookup `GetByKey(projectID, key)` :
   - `not_found` → crée une nouvelle story via `repo.Create`.
   - Trouvée → met à jour via `repo.Update`.
5. En cas d'erreur de création/mise à jour → erreur `IMPORT_ERROR`.

Le handler délègue le parsing du markdown à l'adapter `internal/adapter/markdown`, puis mappe les `ParsedStory` en `ImportStoryInput` pour le service.

### 2.5 API Handler

Fichier : `internal/api/handler/story_handler.go`

```go
type StoryHandler struct {
    service *service.StoryService
}
```

| Endpoint | Handler | Auth |
|----------|---------|------|
| `GET /projects/{projectId}/stories` | `ListStories` | Tous |
| `POST /projects/{projectId}/stories` | `CreateStory` | Admin |
| `GET /projects/{projectId}/stories/{storyId}` | `GetStory` | Tous |
| `PUT /projects/{projectId}/stories/{storyId}` | `UpdateStory` | Admin |
| `DELETE /projects/{projectId}/stories/{storyId}` | `DeleteStory` | Admin |
| `POST /projects/{projectId}/stories/import` | `ImportStories` | Admin |

**Particularités de `ListStories`** :

Supporte deux modes de requête via query params :
1. `?key=S-14` → lookup exact, retourne un objet unique (pas de liste paginée).
2. `?status=backlog,running` → filtre multi-statuts (parsing CSV).
3. Aucun param → liste paginée complète.

**Conversion domaine → API** (`toAPIStory`) :

```go
func toAPIStory(s *model.Story) Story {
    // Mappe les champs, convertit les types null-safe
    // Scope → StoryScope (type généré OpenAPI)
    // Status → StoryStatus (type généré OpenAPI)
}
```

### 2.6 Tests

Fichiers : `internal/domain/service/story_service_test.go`, `internal/api/handler/story_handler_test.go`

Patterns utilisés :
- Tests table-driven avec `t.Run` pour chaque cas.
- `mockStoryRepo` écrit à la main, implémente `port.StoryRepository`.
- Pas de testcontainers dans les tests unitaires.

Couverture des cas clés :
- Validation des formats de clé : `S-01` (valide), `s-01` (invalide), `S-` (invalide), `STORY-123` (valide).
- Import : cas `AllNew`, `AllExisting`, `MixNewAndExisting`, `ParseError`, `EmptyKey`, `EmptyTitle`, `CreateFailure`.
- Update : test de chaque champ optionnel individuellement.

---

## 3. Domaine Epics

### 3.1 Modèle

Fichier : `internal/domain/model/epic.go`

```go
const (
    EpicStatusBacklog    = "backlog"
    EpicStatusInProgress = "in_progress"
    EpicStatusDone       = "done"
)

type Epic struct {
    ID          uuid.UUID
    ProjectID   uuid.UUID
    Name        string
    Description *string   // nullable, max 2000 chars
    Status      string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### 3.2 Port (interface repository)

Fichier : `internal/domain/port/epic_repository.go`

```go
type EpicRepository interface {
    Create(ctx context.Context, epic *model.Epic) (*model.Epic, error)
    GetByID(ctx context.Context, id uuid.UUID) (*model.Epic, error)
    ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Epic, error)
    CountByProject(ctx context.Context, projectID uuid.UUID) (int64, error)
    Update(ctx context.Context, epic *model.Epic) (*model.Epic, error)
    Delete(ctx context.Context, id uuid.UUID) error
}
```

Note : il n'y a pas de `GetByName` — les epics sont identifiés par UUID. Pas de lookup par clé métier contrairement aux stories.

### 3.3 Service

Fichier : `internal/domain/service/epic_service.go`

```go
type EpicService struct {
    repo port.EpicRepository
}
```

#### Création

```go
type CreateEpicParams struct {
    ProjectID   uuid.UUID
    Name        string
    Description *string
    Status      string   // défaut : "backlog"
}
```

Validations :
- `Name` : obligatoire, max 255 caractères.
- `Description` : si présente, max 2000 caractères.
- `ProjectID` : obligatoire.
- `Status` : doit être parmi `backlog`, `in_progress`, `done` (défaut : `backlog`).

#### Mise à jour

```go
type UpdateEpicParams struct {
    ID          uuid.UUID
    Name        *string
    Description *string
    Status      *string
}
```

Même sémantique patch que pour les stories : récupère l'existant, applique les champs non-nil, sauvegarde.

#### Autres méthodes

| Méthode | Description |
|---------|-------------|
| `GetByID(ctx, id)` | Délègue au repo |
| `ListByProject(ctx, projectID, page, perPage)` | Liste paginée — retourne `EpicListResult{Epics, Total}` |
| `Delete(ctx, id)` | Vérifie l'existence avant suppression |

### 3.4 API Handler

Fichier : `internal/api/handler/epic_handler.go`

```go
type EpicHandler struct {
    service   *service.EpicService
    scheduler *service.SchedulerService  // pour le DAG
    storyRepo port.StoryRepository       // accès direct pour le DAG
}
```

| Endpoint | Handler | Auth |
|----------|---------|------|
| `GET /projects/{projectId}/epics` | `ListEpics` | Tous |
| `POST /projects/{projectId}/epics` | `CreateEpic` | Admin |
| `GET /projects/{projectId}/epics/{epicId}` | `GetEpic` | Tous |
| `PUT /projects/{projectId}/epics/{epicId}` | `UpdateEpic` | Admin |
| `DELETE /projects/{projectId}/epics/{epicId}` | `DeleteEpic` | Admin |
| `GET /projects/{projectId}/epics/{epicId}/dag` | `GetEpicDAG` | Tous |

**Endpoint DAG** (`GetEpicDAG`) :

1. Récupère toutes les stories de l'epic via `storyRepo.ListByEpic` (max 500).
2. Construit le DAG via `scheduler.BuildDAG`.
3. Retourne `EpicDAGResponse` avec `nodes` et `edges`.

Structure de la réponse DAG :
```json
{
  "nodes": [
    {"id": "uuid", "key": "S-01", "title": "...", "status": "backlog", "layer": 0}
  ],
  "edges": [
    {"source": "S-01", "target": "S-02"}
  ]
}
```

Le champ `layer` indique la profondeur dans le DAG (0 = pas de dépendances). Les `edges` utilisent les **clés** de stories (pas les UUIDs) car c'est le format du champ `DependsOn`.

### 3.5 Tests

Fichier : `internal/domain/service/epic_service_test.go`

Couverture :
- Création avec statut par défaut (`backlog`) et statut explicite.
- Validation : nom vide, nom trop long, description trop longue, statut invalide, project_id manquant.
- GetByID : trouvé et not_found.
- ListByProject : pagination avec clamp automatique (page 0 → 1, perPage 0 → 20, perPage > 100 → 100).
- Update : chaque champ optionnel, validation des limites.
- Delete : existant et not_found.

---

## 4. Domaine Pipeline Config

### 4.1 Modèles

Fichier : `internal/domain/model/pipeline_config.go`

#### Entité de persistance

```go
type PipelineConfig struct {
    ID         uuid.UUID
    ProjectID  uuid.UUID
    ConfigYAML string    // stocké tel quel en base
    Version    int       // auto-incrémenté à chaque upsert
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

#### Structure YAML

Le format actuel utilise des **groupes** (plusieurs étapes par groupe, exécutées séquentiellement dans l'ordre). L'ancien format plat (`steps:` à la racine) est rétrocompatible via `ParsePipelineConfigYAML`.

```go
type PipelineConfigYAML struct {
    Groups []PipelineGroup `yaml:"groups" json:"groups"`
}

type PipelineGroup struct {
    ID    string         `yaml:"id"    json:"id"`
    Name  string         `yaml:"name"  json:"name"`
    Steps []PipelineStep `yaml:"steps" json:"steps"`
}

type PipelineStep struct {
    ID          string            `yaml:"id"          json:"id"`
    Name        string            `yaml:"name"        json:"name"`
    ActionType  string            `yaml:"action_type" json:"action_type"`
    Description string            `yaml:"description,omitempty"`
    AgentID     string            `yaml:"agent_id,omitempty"`    // UUID de l'agent à utiliser
    Model       string            `yaml:"model,omitempty"`       // override du modèle Claude
    AutoApprove bool              `yaml:"auto_approve"`
    RetryPolicy RetryPolicy       `yaml:"retry_policy"`
    Config      map[string]string `yaml:"config,omitempty"`
}

type RetryPolicy struct {
    MaxRetries int    `yaml:"max_retries" json:"max_retries"`
    RetryType  string `yaml:"retry_type"  json:"retry_type"` // none, on-failure, always
}
```

#### Action types valides

```go
var ValidActionTypes = map[string]bool{
    "agent_run":    true,  // lance un container agent Claude
    "git_branch":   true,  // crée une branche Git
    "git_pr":       true,  // crée/merge une PR
    "notification": true,  // envoie une notification
    "human":        true,  // point d'intervention humaine
    "ci_poll":      true,  // attend le CI
    "hitl_gate":    true,  // gate HITL (approve/reject)
    // Legacy (conservés pour rétrocompat)
    "implement": true,
    "review":    true,
    "merge":     true,
    "test":      true,
    "custom":    true,
}
```

#### Parsing avec rétrocompatibilité

```go
func ParsePipelineConfigYAML(data []byte) (*PipelineConfigYAML, error)
```

- Si le YAML contient `groups:` → utilise le format groupes.
- Si le YAML contient `steps:` (format plat legacy) → les enveloppe dans un seul groupe `"Default"`.
- Toujours retourne une `PipelineConfigYAML` avec des groupes.

Méthode utilitaire :
```go
func (c *PipelineConfigYAML) FlatSteps() []PipelineStep
// Retourne toutes les steps de tous les groupes dans l'ordre.
```

#### Config par défaut

La config par défaut est définie dans le service comme une constante YAML :

```yaml
groups:
  - id: setup
    name: Setup
    steps:
      - id: git-branch
        name: Create Branch
        action_type: git_branch
        config:
          base_branch: main

  - id: development
    name: Development
    steps:
      - id: agent-implement
        name: Implement Story
        action_type: agent_run
        config:
          role: dev
          phase: dev-story

  - id: review
    name: Review
    steps:
      - id: agent-review
        name: Code Review
        action_type: agent_run
        config:
          role: review
          phase: code-review

  - id: merge
    name: Merge
    steps:
      - id: git-pr
        name: Create & Merge PR
        action_type: git_pr
        config:
          strategy: squash

  - id: delivery
    name: Delivery
    steps:
      - id: ci-poll
        name: Wait for CI
        action_type: ci_poll
        config:
          timeout_minutes: "30"
      - id: notify
        name: Notify Completion
        action_type: notification
        config:
          channel: default
```

### 4.2 Port (interface repository)

Fichier : `internal/domain/port/pipeline_config_repository.go`

```go
type PipelineConfigRepository interface {
    GetByProjectID(ctx context.Context, projectID uuid.UUID) (*model.PipelineConfig, error)
    Upsert(ctx context.Context, config *model.PipelineConfig) (*model.PipelineConfig, error)
}
```

Interface minimale : pas de Create/Update séparés — l'upsert gère les deux cas, avec incrémentation automatique de version.

### 4.3 Service

Fichier : `internal/domain/service/pipeline_config_service.go`

```go
type PipelineConfigService struct {
    repo port.PipelineConfigRepository
}
```

#### Méthodes

| Méthode | Description |
|---------|-------------|
| `GetByProjectID(ctx, projectID)` | Retourne la config ou `not_found` |
| `Upsert(ctx, projectID, configYAML)` | Valide le YAML puis sauvegarde |
| `SeedDefault(ctx, projectID)` | Crée la config par défaut sans validation (YAML connu) |

#### Validation du YAML (dans `Upsert`)

La fonction `validatePipelineConfigYAML` applique ces règles :
1. YAML non vide.
2. Parse YAML valide (syntaxe).
3. Au moins un groupe.
4. Pour chaque groupe : `name` non vide, au moins une step.
5. Pour chaque step : `name` non vide, `action_type` non vide et présent dans `ValidActionTypes`.

Codes d'erreur retournés :
- `VALIDATION_ERROR` — YAML vide.
- `INVALID_PIPELINE_CONFIG` — structure invalide (pas de groupes, step sans nom...).
- `INVALID_ACTION_TYPE` — `action_type` inconnu.

### 4.4 API Handler

Fichier : `internal/api/handler/pipeline_config_handler.go`

```go
type PipelineConfigHandler struct {
    service *service.PipelineConfigService
}
```

| Endpoint | Handler | Auth |
|----------|---------|------|
| `GET /projects/{projectId}/pipeline` | `GetPipelineConfig` | Tous |
| `PUT /projects/{projectId}/pipeline` | `UpdatePipelineConfig` | Admin |

**Flux `UpdatePipelineConfig`** :

1. Parse le corps JSON contenant `groups` (format API).
2. Convertit les groupes API → YAML via `groupsToYAML`.
3. Appelle `service.Upsert` qui valide et persiste.
4. Re-parse le YAML stocké pour retourner la réponse API normalisée.

Le handler utilise des structs intermédiaires YAML locaux (`pipelineStepYAML`, `pipelineGroupYAML`) pour la sérialisation, distincts des types du domaine.

### 4.5 Tests

Fichier : `internal/domain/service/pipeline_config_service_test.go`

Couverture :
- Upsert valide → version 1.
- Second upsert → version 2 (incrémentation).
- YAML invalide : vide, syntaxe incorrecte, pas de steps, step sans nom, step sans action_type, action_type invalide.
- Tous les nouveaux action types (7 types canoniques).
- Format groupes et format legacy (flat steps).
- `SeedDefault` : vérifie les 5 groupes, leurs IDs, noms et config des steps.
- `GetByProjectID` : not_found puis trouvé après seed.

---

## 5. Domaine Cost Tracking

### 5.1 Modèles

Fichier : `internal/domain/model/cost_record.go`

#### Entité principale

```go
type CostRecord struct {
    ID           uuid.UUID
    RunStepID    uuid.UUID   // step qui a généré ce coût
    ProjectID    uuid.UUID
    AgentID      *uuid.UUID  // nullable — agent optionnel
    TokensInput  int64
    TokensOutput int64
    CostUSD      float64     // calculé à partir des tokens et du modèle
    Model        string      // nom du modèle Claude utilisé
    CreatedAt    time.Time
}
```

Un `CostRecord` est créé par step de pipeline exécutant un agent. Tous les événements de coût d'une même step sont agrégés en un seul record.

#### Type intermédiaire : CostEvent

```go
// Parsé depuis le flux NDJSON du container agent
type CostEvent struct {
    InputTokens  int64
    OutputTokens int64
    Model        string
}
```

Les `CostEvent` sont émis dans les logs du container sous forme de lignes JSON avec `"type": "cost"`.

#### Calcul du coût

```go
var modelPricingMap = map[string]Pricing{
    "claude-opus-4-6":   {InputPerMTok: 15.0, OutputPerMTok: 75.0},
    "claude-sonnet-4-6": {InputPerMTok: 3.0,  OutputPerMTok: 15.0},
    "claude-haiku-4-5":  {InputPerMTok: 0.25, OutputPerMTok: 1.25},
}

func ComputeCostUSD(model string, inputTokens, outputTokens int64) (float64, bool)
```

La fonction fait d'abord un match exact, puis un match par préfixe pour les IDs de modèles versionnés (ex: `claude-opus-4-6-20251101` → `claude-opus-4-6`). Retourne `(0, false)` pour les modèles inconnus.

#### Vues agrégées

| Type | Description |
|------|-------------|
| `CostSummary` | Résumé simplifié : totaux période/semaine/mois, moyenne par story, budget |
| `ProjectCostSummary` | Détail complet : totaux + breakdown par story/run/modèle |
| `StoryCostSummary` | Coût total d'une story avec comptes de tokens et runs |
| `RunCostDetail` | Coût d'un run avec breakdown par step |
| `AgentCostBreakdown` | Coût agrégé par agent pour un projet |
| `StoryCostBreakdown` | Ligne de breakdown par story |
| `RunCostBreakdown` | Ligne de breakdown par run |
| `CostByModel` | Coût agrégé par modèle |
| `StepCostBreakdown` | Coût d'une step individuelle |
| `CostDataPoint` | Point de données quotidien pour graphes (`Date: "YYYY-MM-DD"`) |
| `RunCostRow` | Ligne pour la liste paginée des runs avec coûts |

### 5.2 Port (interface repository)

Fichier : `internal/domain/port/cost_repository.go`

```go
type CostRepository interface {
    InsertCostRecord(ctx, record) (*model.CostRecord, error)
    GetCostByRunStep(ctx, runStepID) (*model.CostRecord, error)
    SumCostByProject(ctx, projectID, since) (totalCost float64, totalInput, totalOutput int64, err error)
    SumCostByRun(ctx, runID) (float64, error)
    SumCostByStory(ctx, storyID) (totalCost float64, totalInput, totalOutput int64, runCount int, err error)
    ListCostsByProjectByStory(ctx, projectID, since) ([]model.StoryCostBreakdown, error)
    ListCostsByProjectByRun(ctx, projectID, since) ([]model.RunCostBreakdown, error)
    ListCostsByProjectByModel(ctx, projectID, since) ([]model.CostByModel, error)
    ListStepCostsByRun(ctx, runID) ([]model.StepCostBreakdown, error)
    ListDailyCostsByProject(ctx, projectID, since) ([]model.CostDataPoint, error)
    ListCostsByProjectByRunPaginated(ctx, projectID, since, limit, offset) ([]model.RunCostRow, error)
    CountCostsByProjectByRun(ctx, projectID, since) (int64, error)
    ListByProjectByAgent(ctx, projectID) ([]model.AgentCostBreakdown, error)
}
```

### 5.3 Service

Fichier : `internal/domain/service/cost_service.go`

```go
type CostService struct {
    costRepo    port.CostRepository
    projectRepo port.ProjectRepository
    storyRepo   port.StoryRepository
    runRepo     port.RunRepository
    logger      *slog.Logger
}
```

#### Enregistrement d'un coût de step

```go
func (s *CostService) RecordStepCost(
    ctx context.Context,
    stepID, projectID uuid.UUID,
    events []model.CostEvent,
    agentID *uuid.UUID,
) error
```

Logique :
1. Si `events` vide → no-op (retourne nil).
2. Agrège tous les events : somme des tokens, dernière valeur de `Model`.
3. Calcule le coût via `model.ComputeCostUSD`.
4. Si modèle inconnu → log warning, coût = 0 (pas d'erreur).
5. Insère un seul `CostRecord` via `costRepo.InsertCostRecord`.

#### Périodes de reporting

```go
func parsePeriod(period string) (time.Time, error)
// Accepte: "7d", "30d", "90d"
// Retourne la date de début (UTC) ou VALIDATION_ERROR
```

#### Méthodes de consultation

| Méthode | Description |
|---------|-------------|
| `GetProjectCosts(ctx, projectID, period)` | Détail complet : totaux + 3 breakdowns. Vérifie l'existence du projet. |
| `GetProjectCostSummary(ctx, projectID, period)` | Résumé avec totaux semaine/mois/période + moyenne par story |
| `GetProjectCostChart(ctx, projectID, period)` | Points quotidiens pour graphe |
| `GetProjectCostRuns(ctx, projectID, period, page, perPage)` | Liste paginée de runs avec coûts |
| `GetProjectCostsByAgent(ctx, projectID)` | Breakdown par agent (sans filtre de période) |
| `GetStoryCosts(ctx, projectID, storyID)` | Vérifie l'appartenance au projet avant d'agréger |
| `GetRunCosts(ctx, projectID, runID)` | Vérifie l'appartenance au projet avant d'agréger |

### 5.4 API Handler

Fichier : `internal/api/handler/cost_handler.go`

```go
type CostHandler struct {
    service *service.CostService
}
```

| Endpoint | Handler | Query params |
|----------|---------|--------------|
| `GET /projects/{projectId}/costs` | `GetProjectCosts` | `period` (7d/30d/90d) |
| `GET /projects/{projectId}/costs/summary` | `GetProjectCostSummary` | `period` |
| `GET /projects/{projectId}/costs/chart` | `GetProjectCostChart` | `period` |
| `GET /projects/{projectId}/costs/runs` | `GetProjectCostRuns` | `period`, `page`, `per_page` |
| `GET /projects/{projectId}/costs/agents` | `GetProjectCostsByAgent` | — |
| `GET /projects/{projectId}/stories/{storyId}/costs` | `GetStoryCosts` | — |
| `GET /projects/{projectId}/runs/{runId}/costs` | `GetRunCosts` | — |

La période par défaut est `"7d"` si le query param est absent.

### 5.5 Tests

Fichier : `internal/domain/service/cost_service_test.go`

Patterns notables :
- Mock entièrement fonctionnel avec des champs `Fn` pour chaque méthode (pattern function-per-method).
- Utilise `testify/assert` et `testify/require` (contrairement aux autres domaines qui utilisent le stdlib `testing`).
- Test des prix exacts pour chaque modèle (`InDelta` avec tolérance 0.001).

Couverture :
- `RecordStepCost` : events vides, modèle connu (opus/sonnet/haiku), modèle inconnu (coût = 0), plusieurs events agrégés en 1 insert, erreur repo propagée, avec/sans agentID.
- `ComputeCostUSD` : tous les modèles connus, inconnu, zéro tokens.
- `GetProjectCosts` : projet not_found, période invalide, zéro coûts, période vide (défaut 7d).
- `GetStoryCosts` : story not_found, mauvais projet (not_found), zéro coûts.
- `GetRunCosts` : run not_found, mauvais projet, zéro coûts.
- `GetProjectCostChart` : succès, projet not_found, période vide.
- `GetProjectCostRuns` : succès avec données, projet not_found, pagination.
- `GetProjectCostsByAgent` : succès, liste vide, projet not_found, erreur repo.
- `parsePeriod` : toutes les valeurs valides et invalides.

---

## 6. Domaine HITL (Human-in-the-Loop)

### 6.1 Modèle

Fichier : `internal/domain/model/hitl.go`

```go
type HITLStatus string

const (
    HITLStatusPending  HITLStatus = "pending"
    HITLStatusApproved HITLStatus = "approved"
    HITLStatusRejected HITLStatus = "rejected"
)

// HITLRequest est créé par l'action hitl_gate du pipeline executor
// quand une step nécessite une validation humaine.
type HITLRequest struct {
    ID              uuid.UUID
    RunStepID       uuid.UUID   // step qui a déclenché ce gate
    GateType        string      // "approval" ou "human"
    DiffContent     *string     // diff PR (optionnel, fetché depuis GitProvider)
    Message         *string     // message optionnel pour le reviewer
    Status          HITLStatus
    ResolvedAt      *time.Time
    ResolvedBy      *uuid.UUID  // ID de l'utilisateur qui a résolu
    RejectionReason *string
    CreatedAt       time.Time
}

// Vue dénormalisée pour le listing des pending HITL
type PendingHITLRequest struct {
    ID        uuid.UUID
    RunID     uuid.UUID
    StepID    uuid.UUID
    StoryKey  string
    DiffURL   *string
    CreatedAt time.Time
}
```

La machine à états est simple : `pending` → `approved` ou `rejected`. Les transitions inverses sont interdites.

### 6.2 Port (interface repository)

Fichier : `internal/domain/port/hitl_repository.go`

```go
type HITLRepository interface {
    Create(ctx, req) (*model.HITLRequest, error)
    GetByID(ctx, id) (*model.HITLRequest, error)
    GetByRunStepID(ctx, runStepID) (*model.HITLRequest, error)
    UpdateStatus(ctx, id, status, resolvedBy, rejectionReason, resolvedAt) (*model.HITLRequest, error)
    ListPendingByProject(ctx, projectID) ([]*model.PendingHITLRequest, error)
    CountPendingByProject(ctx, projectID) (int64, error)
    ListFiltered(ctx, status *string, limit, offset) ([]*model.HITLRequest, error)
    CountFiltered(ctx, status *string) (int64, error)
}
```

### 6.3 Service

Fichier : `internal/domain/service/hitl_service.go`

```go
type HITLService struct {
    hitlRepo port.HITLRepository
    runRepo  port.RunRepository
    eventPub port.EventPublisher
    logger   *slog.Logger
}
```

#### Approve

```go
func (s *HITLService) Approve(ctx, hitlRequestID, userID) (*model.HITLRequest, error)
```

1. `GetByID` → retourne `not_found` si inexistant.
2. Vérifie `status == pending` → sinon `VALIDATION_ERROR`.
3. `hitlRepo.UpdateStatus(approved, userID, nil, now)`.
4. `runRepo.UpdateRunStepStatus(stepID, running, ...)` — reprend la step (log warning si échec).
5. Publie l'événement `hitl_gate.approved` via `eventPub`.

#### Reject

```go
func (s *HITLService) Reject(ctx, hitlRequestID, userID, reason *string) (*model.HITLRequest, error)
```

1. `GetByID` → `not_found` si inexistant.
2. Vérifie `status == pending` → sinon `VALIDATION_ERROR`.
3. `hitlRepo.UpdateStatus(rejected, userID, reason, now)`.
4. `runRepo.UpdateRunStepStatus(stepID, failed, ..., "rejected by reviewer")` — log warning si échec.
5. Publie l'événement `hitl_gate.rejected`.

#### Autres méthodes

| Méthode | Description |
|---------|-------------|
| `GetByID(ctx, id)` | Lookup direct |
| `GetProjectIDForHITLRequest(ctx, hitlRequestID)` | Remonte la chaîne hitl → step → run pour le RBAC |
| `ListPendingByProject(ctx, projectID)` | Retourne `(pending, count, error)` |
| `ListAll(ctx, status *string, page, perPage)` | Liste globale avec filtre optionnel par statut |
| `GetByStepID(ctx, stepID)` | Lookup par step (pour le pipeline executor) |

#### Publication d'événement

La méthode privée `publishEvent` construit un `model.Event` :
- `EntityType` : `"hitl_gate"`
- `Action` : `"approved"` ou `"rejected"`
- `Payload` : JSON avec `hitl_request_id`, `run_id`, `step_id`, `user_id`

Si `eventPub` est nil ou si la publication échoue → log warning, pas de propagation d'erreur.

### 6.4 API Handler

Fichier : `internal/api/handler/hitl_handler.go`

```go
type HITLHandler struct {
    service *service.HITLService
}
```

| Endpoint | Handler | Auth |
|----------|---------|------|
| `GET /projects/{projectId}/hitl/pending` | `ListPendingHITLRequests` | Tous |
| `GET /hitl-requests/{hitlRequestId}` | `GetHITLRequest` | Tous |
| `POST /hitl-requests/{hitlRequestId}/approve` | `ApproveHITLRequest` | Authentifié |
| `POST /hitl-requests/{hitlRequestId}/reject` | `RejectHITLRequest` | Authentifié |
| `GET /hitl-requests` | `ListHITLRequests` | Tous |
| `GET /hitl-requests/by-step/{stepId}` | `GetHITLRequestByStep` | Tous |

**Approve/Reject** : extraient le `userID` du contexte JWT (`middleware.UserIDFromContext`). Retournent 401 si non authentifié.

**Reject** : le corps JSON est optionnel. Si `Content-Length > 0`, décode `{"reason": "..."}`.

### 6.5 Tests

Fichier : `internal/domain/service/hitl_service_test.go`

Couverture :
- `Approve` : pending→approved (succès), déjà approved (VALIDATION_ERROR), déjà rejected (VALIDATION_ERROR).
- `Approve` : vérifie que la step est passée à `running` et qu'un événement `approved` est publié.
- `Reject` : pending→rejected avec raison, avec raison nil, déjà approved (VALIDATION_ERROR).
- `Reject` : vérifie que la step est passée à `failed`.
- `Approve` not_found.
- `Reject` avec erreur repo sur UpdateStatus.
- `ListPendingByProject` : 0 et plusieurs pending.
- `ListAll` : sans filtre, filtré par `pending`, filtré par `approved`, pagination.
- `GetByStepID` : trouvé, not_found.

---

## 7. Domaine Notifications

### 7.1 Modèle

Fichier : `internal/domain/model/notification_config.go`

```go
const ChannelTypeDiscord = "discord"
const ChannelTypeWebhook = "webhook"

type NotificationConfig struct {
    ID           uuid.UUID
    ProjectID    uuid.UUID
    ChannelType  string              // "discord" ou "webhook"
    Config       map[string]string   // ex: {"url": "https://discord.com/api/webhooks/..."}
    EventsFilter []string            // ex: ["run.completed", "run.failed"]
    Enabled      bool
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

`EventsFilter` contient des noms d'événements au format `entity_type.action` (ex: `run.completed`, `hitl_gate.approved`). Seuls les événements présents dans ce filtre déclenchent une notification via ce channel.

### 7.2 Ports

Fichier : `internal/domain/port/notification_config_repository.go`

```go
type NotificationConfigRepository interface {
    Insert(ctx, cfg) (*model.NotificationConfig, error)
    Get(ctx, id) (*model.NotificationConfig, error)
    ListByProject(ctx, projectID) ([]*model.NotificationConfig, error)
    Update(ctx, cfg) (*model.NotificationConfig, error)
    Delete(ctx, id) error
    ListEnabledByProject(ctx, projectID) ([]*model.NotificationConfig, error)  // filtré sur enabled=true
}
```

Fichier : `internal/domain/port/notifier.go`

```go
type Notifier interface {
    Send(ctx context.Context, event model.Event, config map[string]string) error
}
```

L'interface `Notifier` est implémentée par :
- `internal/adapter/discord/` → webhook Discord
- `internal/adapter/webhook/` → webhook générique

Les notifiers sont injectés sous forme de `map[string]port.Notifier` avec la clé = `channel_type`.

### 7.3 Service NotificationConfigService

Fichier : `internal/domain/service/notification_config_service.go`

```go
type NotificationConfigService struct {
    repo      port.NotificationConfigRepository
    notifiers map[string]port.Notifier
}
```

#### Méthodes CRUD

| Méthode | Description |
|---------|-------------|
| `Create(ctx, projectID, channelType, config, eventsFilter, enabled)` | Crée une config |
| `Get(ctx, id)` | Récupère par ID |
| `ListByProject(ctx, projectID)` | Liste toutes les configs d'un projet |
| `Update(ctx, id, channelType, config, eventsFilter, enabled)` | Met à jour |
| `Delete(ctx, id)` | Supprime |

#### Test de notification

```go
func (s *NotificationConfigService) Test(ctx, id) error
```

1. Récupère la config par ID.
2. Cherche le notifier correspondant à `cfg.ChannelType` dans la map.
3. Si `ChannelType` inconnu → `VALIDATION_ERROR` avec message descriptif.
4. Construit un `model.Event` de test (`entity_type: "notification"`, `action: "test"`).
5. Appelle `notifier.Send` avec la config du channel.

### 7.4 Service NotificationDispatcher

Fichier : `internal/domain/service/notification_dispatcher.go`

Le dispatcher est un composant long-running qui s'abonne aux events de tous les projets et route vers les notifiers appropriés.

```go
type NotificationDispatcher struct {
    eventSub    port.EventSubscriber
    repo        port.NotificationConfigRepository
    projectRepo port.ProjectRepository
    notifiers   map[string]port.Notifier
    mu          sync.Mutex
    cleanups    []func()
}
```

#### Démarrage

```go
func (d *NotificationDispatcher) Start(ctx context.Context)
```

Lance `run` en goroutine. S'arrête proprement quand `ctx` est annulé.

#### Logique interne (`run`)

1. Récupère tous les projets existants via `projectRepo.List`.
2. Pour chaque projet : `eventSub.Subscribe(ctx, projectID)` → obtient un channel d'events.
3. Lance une goroutine `fanIn` par channel pour les merger dans un channel central `merged`.
4. Boucle principale : lit `merged`, appelle `dispatch` pour chaque event.
5. Sur `ctx.Done()` : appelle tous les cleanups (désinscription).

#### Dispatch d'un event

```go
func (d *NotificationDispatcher) dispatch(ctx, event)
```

1. Récupère toutes les configs **activées** du projet via `repo.ListEnabledByProject`.
2. Pour chaque config :
   - Vérifie si `event.EventName()` est dans `cfg.EventsFilter`.
   - Si oui, appelle le notifier correspondant.
   - Erreur de send → log warning, continue (ne bloque pas les autres).

Les erreurs de dispatch sont toujours silencieuses (log warning uniquement) pour éviter qu'une notification ratée n'interrompe le pipeline.

### 7.5 API Handler

Fichier : `internal/api/handler/notification_handler.go`

```go
type NotificationHandler struct {
    service *service.NotificationConfigService
}
```

| Endpoint | Handler | Méthode |
|----------|---------|---------|
| `GET /projects/{projectId}/notifications` | `ListNotificationConfigs` | GET |
| `POST /projects/{projectId}/notifications` | `CreateNotificationConfig` | POST |
| `PUT /projects/{projectId}/notifications/{notificationId}` | `UpdateNotificationConfig` | PUT |
| `DELETE /projects/{projectId}/notifications/{notificationId}` | `DeleteNotificationConfig` | DELETE |
| `POST /projects/{projectId}/notifications/{notificationId}/test` | `TestNotificationConfig` | POST → 204 |

**Validation du channel_type dans `CreateNotificationConfig`** :

Le handler valide que `channel_type` est `"discord"` ou `"webhook"` avant d'appeler le service. Les valeurs non supportées retournent `VALIDATION_ERROR`.

**Conversion domaine → API** (`toAPINotificationConfig`) :

Protège contre les nil avec des valeurs par défaut (slice vide pour `EventsFilter`, map vide pour `Config`).

### 7.6 Tests

Fichier : `internal/domain/service/notification_config_service_test.go`

Couverture de `NotificationConfigService.Test` :
- Succès : discord/webhook → notifier appelé 1 fois.
- Config not_found → `CategoryNotFound`.
- Channel type inconnu (`"slack"`) → `CategoryValidation`.
- Erreur du notifier → propagée.

Fichier : `internal/domain/service/notification_dispatcher_test.go`

Couverture du dispatcher (package `service_test`) :
- Event correspondant au filtre → notifier appelé.
- Event hors filtre → notifier non appelé.
- Erreur d'un notifier → ne bloque pas les autres notifiers pour le même event.

---

## 8. Modèle LogEvent

Fichier : `internal/domain/model/log_event.go`

```go
type LogEvent struct {
    RunID     string         `json:"run_id"`
    StepID    string         `json:"step_id"`
    Timestamp time.Time      `json:"timestamp"`
    Level     string         `json:"level"`   // info, warn, error, debug
    Message   string         `json:"message"`
    RawLine   string         `json:"raw_line"`
    IsJSON    bool           `json:"is_json"`
    Data      map[string]any `json:"data,omitempty"`    // champs JSON parsés
    Type      string         `json:"type,omitempty"`    // ex: "cost" pour les events de coût
    // Champs peuplés uniquement si Type == "cost"
    InputTokens  int64       `json:"input_tokens,omitempty"`
    OutputTokens int64       `json:"output_tokens,omitempty"`
    Model        string      `json:"model,omitempty"`
}
```

Ce modèle est utilisé par le log streamer pour parser les lignes NDJSON émises par les containers agents. Quand une ligne est un JSON valide avec `"type": "cost"`, les champs de coût sont extraits et utilisés pour créer des `CostEvent` qui alimentent ensuite le `CostService.RecordStepCost`.

---

## 9. Flux de données transversaux

### 9.1 Pipeline d'exécution → Cost Tracking

```
PipelineExecutor
  └─ exécute une step de type "agent_run"
       └─ lance un container via AgentRuntime
            └─ stream les logs NDJSON via LogStreamer
                 └─ parse chaque ligne en LogEvent
                      └─ si LogEvent.Type == "cost"
                           └─ collecte dans []CostEvent
  └─ à la fin de la step :
       └─ CostService.RecordStepCost(stepID, projectID, events, agentID)
            └─ agrège les events → 1 CostRecord
                 └─ CostRepository.InsertCostRecord
```

### 9.2 Pipeline d'exécution → HITL Gate

```
PipelineExecutor
  └─ exécute une step de type "hitl_gate"
       └─ HITLRepository.Create(request)
            └─ step passe à status "waiting_approval"
  └─ pipeline en attente

Reviewer
  └─ GET /hitl-requests/{id} (voit le diff)
  └─ POST /hitl-requests/{id}/approve (ou /reject)
       └─ HITLService.Approve
            └─ HITLRepository.UpdateStatus(approved)
            └─ RunRepository.UpdateRunStepStatus(running)
            └─ EventPublisher.Publish(hitl_gate.approved)

PipelineExecutor
  └─ polling ou event SSE
       └─ détecte step en status "running"
            └─ reprend l'exécution du pipeline
```

### 9.3 Event Bus → Notifications

```
Système (n'importe quel service)
  └─ EventPublisher.Publish(event)
       └─ Postgres NOTIFY via pgxlisten

NotificationDispatcher
  └─ subscribé via EventSubscriber
       └─ reçoit l'event sur le channel merged
            └─ NotificationConfigRepository.ListEnabledByProject(event.ProjectID)
                 └─ filtre sur EventsFilter
                      └─ Notifier.Send(event, cfg.Config)
                           → Discord webhook ou Generic webhook
```

### 9.4 Format des noms d'événements

Les événements suivent la convention `entity_type.action` (méthode `EventName()` sur `model.Event`) :

| Événement | Déclencheur |
|-----------|-------------|
| `run.started` | Lancement d'un run |
| `run.completed` | Fin de run avec succès |
| `run.failed` | Fin de run avec erreur |
| `step.started` | Début d'une step |
| `step.completed` | Fin d'une step |
| `step.failed` | Échec d'une step |
| `hitl_gate.approved` | Approbation HITL |
| `hitl_gate.rejected` | Rejet HITL |
| `notification.test` | Test de notification |

Ces noms sont utilisés dans `EventsFilter` des `NotificationConfig` pour router les notifications.
