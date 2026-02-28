# Diagrammes Mermaid — Stories, Epics & Config

## 1. Architecture Hexagonale — Vue d'ensemble

Montre les 4 couches (handler → service → port ← adapter) appliquées aux domaines Stories, Epics, Pipeline Config, Cost, HITL et Notifications. La flèche indique la direction des dépendances : les services ne dépendent que des ports (interfaces), jamais des adapters.

```mermaid
graph TD
    subgraph HTTP["Couche HTTP (chi)"]
        H1[StoryHandler]
        H2[EpicHandler]
        H3[PipelineConfigHandler]
        H4[CostHandler]
        H5[HITLHandler]
        H6[NotificationConfigHandler]
    end

    subgraph Services["Couche Service (domaine)"]
        S1[StoryService]
        S2[EpicService]
        S3[PipelineConfigService]
        S4[CostService]
        S5[HITLService]
        S6[NotificationConfigService]
        S7[NotificationDispatcher]
        S8[RunService]
        S9[PipelineExecutor]
        S10[SchedulerService]
    end

    subgraph Ports["Ports (interfaces)"]
        P1[StoryRepository]
        P2[EpicRepository]
        P3[PipelineConfigRepository]
        P4[CostRepository]
        P5[HITLRepository]
        P6[NotificationConfigRepository]
        P7[EventPublisher]
        P8[EventSubscriber]
        P9[RunRepository]
        P10[JobQueue]
        P11[Notifier]
    end

    subgraph Adapters["Adapters (Postgres / River / Discord / Webhook)"]
        A1[postgres.StoryRepo]
        A2[postgres.EpicRepo]
        A3[postgres.PipelineConfigRepo]
        A4[postgres.CostRepo]
        A5[postgres.HITLRepo]
        A6[postgres.NotifConfigRepo]
        A7[postgres.EventPublisher]
        A8[postgres.EventSubscriber]
        A9[postgres.RunRepo]
        A10[river.JobQueue]
        A11[discord.Notifier]
        A12[webhook.Notifier]
    end

    H1 --> S1
    H2 --> S2
    H3 --> S3
    H4 --> S4
    H5 --> S5
    H6 --> S6

    S1 --> P1
    S2 --> P2
    S3 --> P3
    S4 --> P4 & P9
    S5 --> P5 & P9 & P7
    S6 --> P6
    S7 --> P8 & P6 & P11
    S8 --> P9 & P10 & P7
    S9 --> P9 & P7

    P1 -.->|implements| A1
    P2 -.->|implements| A2
    P3 -.->|implements| A3
    P4 -.->|implements| A4
    P5 -.->|implements| A5
    P6 -.->|implements| A6
    P7 -.->|implements| A7
    P8 -.->|implements| A8
    P9 -.->|implements| A9
    P10 -.->|implements| A10
    P11 -.->|implements| A11
    P11 -.->|implements| A12
```

---

## 2. Story Lifecycle — Transitions de statut

Montre tous les états possibles d'une Story et les événements qui déclenchent chaque transition. Les transitions sont encodées dans `isValidStoryStatus()` + la logique de `PipelineExecutor` et `RunService`.

```mermaid
stateDiagram-v2
    [*] --> backlog : Create (default)

    backlog --> running : RunService.LaunchRun()\nPipelineExecutor → story.status = running

    running --> done : PipelineExecutor\nAll steps completed\nstory.status = done

    running --> failed : PipelineExecutor\nStep failure\nstory.status = failed

    running --> running : RetryStep\n(run re-enqueued)

    failed --> running : RetryStep\nRun → running\nStory ← running

    done --> [*] : Terminal state\n(guard: STORY_ALREADY_COMPLETED)

    backlog --> backlog : StoryService.Update()\nstatus patch (manuel)

    note right of backlog
        Guard LaunchRun:
        - status != done
        - no active run
        - pipeline config exists
    end note

    note right of running
        Publié: story.status_updated
        (EntityType=story, Action=status_updated)
    end note
```

---

## 3. DAG Construction & Exécution — Stories dans un Epic

Montre comment le champ `DependsOn []string` (clés de stories) est utilisé par `SchedulerService.BuildDAG()` pour construire les couches d'exécution parallèle via l'algorithme de Kahn. Les file conflicts (TargetFiles) génèrent des edges implicites.

```mermaid
flowchart LR
    subgraph Input["Epic.Stories (input)"]
        ST1["S-01\nDependsOn: []"]
        ST2["S-02\nDependsOn: [S-01]"]
        ST3["S-03\nDependsOn: [S-01]"]
        ST4["S-04\nDependsOn: [S-02, S-03]"]
    end

    subgraph BuildDAG["SchedulerService.BuildDAG()"]
        IDX["Index par Key\nbyKey map[string]*Story"]
        ADJ["Adjacency list\nadj[dep] → []dependents"]
        INDEG["In-degree map\ninDegree[key] = 0"]
        FILE["addFileConflictEdges()\nTargetFiles → edges implicites"]
        KAHN["Kahn's algorithm\ncouche par couche"]
    end

    subgraph DAGResult["DAGResult.Groups"]
        G0["Groups[0]\nS-01 (seul, in-degree=0)"]
        G1["Groups[1]\nS-02 + S-03 (parallèle)"]
        G2["Groups[2]\nS-04 (attend S-02 + S-03)"]
    end

    subgraph Exec["Exécution (EpicRunService)"]
        E0["Lancer S-01\n(attendre completion)"]
        E1["Lancer S-02 ET S-03\nen parallèle"]
        E2["Lancer S-04\n(après S-02 + S-03 done)"]
    end

    Input --> IDX
    IDX --> ADJ
    ADJ --> INDEG
    INDEG --> FILE
    FILE --> KAHN
    KAHN --> DAGResult
    DAGResult --> Exec

    note1["Cycle détecté → DAG_CYCLE_DETECTED\nDomainError (CategoryInvalidState)"]
    KAHN -. "zeroKeys = [] avant fin" .-> note1
```

---

## 4. Epic Lifecycle — Séquence complète avec DAG

Montre le flux complet : création d'un epic → ajout de stories → lancement via `/launch` → DAG scheduling → completion. Les acteurs correspondent aux couches réelles du code.

```mermaid
sequenceDiagram
    actor FE as Frontend
    participant API as API Handler
    participant EpicSvc as EpicService
    participant RunSvc as RunService
    participant SchedSvc as SchedulerService
    participant EpicRunSvc as EpicRunService
    participant JobQ as River JobQueue
    participant DB as Postgres

    FE->>API: POST /projects/{id}/epics
    API->>EpicSvc: Create(name, description)
    EpicSvc->>DB: INSERT epics (status=backlog)
    DB-->>EpicSvc: Epic{id, status=backlog}
    EpicSvc-->>API: Epic
    API-->>FE: 201 Epic

    FE->>API: POST /projects/{id}/stories (epic_id=...)
    API->>StoryService: Create(key, title, depends_on, ...)
    StoryService->>DB: INSERT stories
    DB-->>StoryService: Story
    StoryService-->>FE: 201 Story

    FE->>API: POST /projects/{id}/epics/{epicId}/launch
    API->>EpicRunSvc: LaunchEpicRun(projectID, epicID)
    EpicRunSvc->>DB: SELECT stories WHERE epic_id=...
    DB-->>EpicRunSvc: []Story
    EpicRunSvc->>SchedSvc: BuildDAG(stories)
    SchedSvc-->>EpicRunSvc: DAGResult{Groups}
    EpicRunSvc->>DB: INSERT epic_runs (status=running)
    loop Pour chaque groupe DAG
        EpicRunSvc->>RunSvc: LaunchRun(projectID, storyID)
        RunSvc->>DB: INSERT run + run_steps
        RunSvc->>JobQ: EnqueueExecuteRun(runID)
        JobQ-->>RunSvc: ok
    end
    EpicRunSvc-->>API: EpicRun{id, status=running}
    API-->>FE: 202 Accepted {epic_run_id, status=running}

    Note over JobQ,DB: River worker exécute chaque run async
    JobQ->>PipelineExecutor: ExecuteRun(runID)
    PipelineExecutor->>DB: UPDATE story.status = done/failed

    FE->>API: GET /projects/{id}/epic-runs/{epicRunId}
    API->>DB: SELECT epic_runs + runs
    DB-->>API: EpicRun{status, progress}
    API-->>FE: EpicRun (SSE ou polling)
```

---

## 5. Pipeline Config — Hiérarchie des modèles

Montre la structure d'imbrication : `PipelineConfig` → `PipelineConfigYAML` → `[]PipelineGroup` → `[]PipelineStep` + `RetryPolicy`. Inclut les contraintes de validation et les action_types valides.

```mermaid
graph TD
    PC["PipelineConfig\n───────────────\nID: uuid\nProjectID: uuid\nConfigYAML: string\nVersion: int\nCreatedAt / UpdatedAt"]

    PCYAML["PipelineConfigYAML\n───────────────\nGroups: []PipelineGroup\n\nParsé via ParsePipelineConfigYAML()\nCompatibilité legacy: steps[] → Group{Default}"]

    PG["PipelineGroup\n───────────────\nID: string\nName: string\nSteps: []PipelineStep"]

    PS["PipelineStep\n───────────────\nID: string\nName: string\nActionType: string ← ValidActionTypes\nDescription: string (opt)\nAgentID: string (required si agent_run)\nModel: string (opt)\nAutoApprove: bool\nRetryPolicy: RetryPolicy\nConfig: map[string]string (opt)"]

    RP["RetryPolicy\n───────────────\nMaxRetries: int\nRetryType: string\n  none | on-failure | always"]

    AT["ValidActionTypes\n───────────────\nagent_run\ngit_branch\ngit_pr\nnotification\nhuman\nci_poll\nhitl_gate\n(legacy: implement, review,\n merge, test, custom)"]

    FLAT["FlatSteps()\n───────────────\nRetourne tous les steps\nen ordre, groupes aplatis"]

    PC -->|"ConfigYAML parsé"| PCYAML
    PCYAML -->|"1..N"| PG
    PG -->|"1..N"| PS
    PS -->|"1"| RP
    PS -->|"ActionType ∈"| AT
    PCYAML -->|"helper"| FLAT

    note1["Contraintes:\n• agent_id requis si action_type = agent_run\n• Validé dans RunService.LaunchRun()\n• Snapshot JSON stocké dans Run.PipelineConfigSnapshot"]
    PC -.->|"validation"| note1
```

---

## 6. Cost Tracking — Flux de données

Montre le chemin complet depuis le stream de logs NDJSON du container agent jusqu'à l'enregistrement en base. Le `LogEvent.Type == "cost"` est le déclencheur d'extraction.

```mermaid
flowchart TD
    CONTAINER["Agent Container\n(stdout NDJSON stream)"]

    STREAMER["port.LogStreamer\nStream(ctx, containerID, runID, stepID)"]

    PARSE["ParseLogEvent()\n───────────────\n• JSON valide → IsJSON=true\n• Data[type] = LogEvent.Type\n• Data[input_tokens] → InputTokens\n• Data[output_tokens] → OutputTokens\n• Data[model] → Model"]

    FILTER{{"LogEvent.Type == 'cost' ?"}}

    SKIP["Ignorer\n(log normal, warn, error...)"]

    COSTEVENT["model.CostEvent\n───────────────\nInputTokens: int64\nOutputTokens: int64\nModel: string"]

    ACCUMULATE["Accumuler []CostEvent\njusqu'à fin du step"]

    COSTSVC["CostService.RecordStepCost()\n───────────────\n• Agréger input/output tokens\n• ComputeCostUSD(model, in, out)\n  → pricing map (Opus/Sonnet/Haiku)\n• Fallback: prefix match si versioned model ID"]

    COSTRECORD["model.CostRecord\n───────────────\nRunStepID / ProjectID / AgentID\nTokensInput / TokensOutput\nCostUSD / Model\nCreatedAt"]

    REPO["CostRepository.InsertCostRecord()\n→ Postgres: cost_records table"]

    UNKNOWN["Model inconnu:\ncoût = 0.0\nslog.Warn (non bloquant)"]

    CONTAINER --> STREAMER
    STREAMER --> PARSE
    PARSE --> FILTER
    FILTER -- "non" --> SKIP
    FILTER -- "oui" --> COSTEVENT
    COSTEVENT --> ACCUMULATE
    ACCUMULATE --> COSTSVC
    COSTSVC --> COSTRECORD
    COSTSVC -->|"model inconnu"| UNKNOWN
    COSTRECORD --> REPO
```

---

## 7. HITL Gate — Workflow d'approbation

Montre le moment précis où l'executor crée la gate, la suspension du pipeline, l'action du reviewer, et la reprise. L'invariant clé : le step ne progresse que si la gate est approuvée.

```mermaid
sequenceDiagram
    participant Exec as PipelineExecutor
    participant Action as hitl_gate Action
    participant HITLSvc as HITLService
    participant RunRepo as RunRepository
    participant EvtPub as EventPublisher
    participant Reviewer as Reviewer (API)
    participant DB as Postgres

    Exec->>Action: Execute(ctx, runCtx)
    Action->>DB: INSERT hitl_requests\n(status=pending, gate_type=approval)
    Action->>RunRepo: UpdateRunStepStatus(stepID, waiting_approval)
    Action-->>Exec: nil (no error)

    Exec->>RunRepo: GetRunStep(stepID)
    RunRepo-->>Exec: step.status = waiting_approval
    Exec-->>Exec: return errStepSuspended
    Note over Exec: Pipeline suspendu\naucun step suivant lancé

    Note over Reviewer: Reviewer consulte les pending HITL
    Reviewer->>HITLSvc: GET /hitl/pending?project_id=...
    HITLSvc->>DB: ListPendingByProject()
    DB-->>Reviewer: []PendingHITLRequest

    alt Approved
        Reviewer->>HITLSvc: POST /hitl/{id}/approve
        HITLSvc->>DB: UPDATE hitl_requests SET status=approved
        HITLSvc->>RunRepo: UpdateRunStepStatus(stepID, running)
        HITLSvc->>EvtPub: Publish(hitl_gate.approved)
        HITLSvc-->>Reviewer: 200 HITLRequest{status=approved}
        Note over Exec: River re-détecte le step running\net reprend l'exécution
    else Rejected
        Reviewer->>HITLSvc: POST /hitl/{id}/reject (reason)
        HITLSvc->>DB: UPDATE hitl_requests SET status=rejected
        HITLSvc->>RunRepo: UpdateRunStepStatus(stepID, failed, reason)
        HITLSvc->>EvtPub: Publish(hitl_gate.rejected)
        HITLSvc-->>Reviewer: 200 HITLRequest{status=rejected}
        Note over Exec: Step failed → Run failed\nStory.status = failed
    end
```

---

## 8. Notification Dispatch — Pipeline de routage

Montre le chemin complet d'un event depuis `EventPublisher.Publish()` jusqu'aux notifiers (Discord/Webhook). Le fan-in merge les channels par projet ; les erreurs d'envoi sont silencées (logged, non fatales).

```mermaid
flowchart LR
    PUB["EventPublisher.Publish(event)\n(postgres NOTIFY)"]

    subgraph Dispatcher["NotificationDispatcher.Start()"]
        PROJECTS["projectRepo.List()\n→ tous les projets"]
        SUBSCRIBE["eventSub.Subscribe(ctx, projectID)\n→ chan model.Event + cleanup"]
        FANIN["fanIn goroutine\n(par projet)\n→ merged chan (buf 256)"]
        MERGED["merged chan model.Event"]
        DISPATCH["dispatch(ctx, event)"]
    end

    CONFIGS["repo.ListEnabledByProject()\n→ []NotificationConfig\n(enabled=true, project_id=event.ProjectID)"]

    FILTER{{"cfg.EventsFilter\ncontains event.EventName() ?<br>(format: entity_type.action)"}}

    SKIP2["Ignorer cette config"]

    ROUTER{{"cfg.ChannelType ?"}}

    DISCORD["discord.Notifier.Send()\n→ Discord Webhook POST"]

    WEBHOOK["webhook.Notifier.Send()\n→ Generic HTTP POST"]

    ERR["slog.Warn(send failed)\nNon fatal — pipeline continue"]

    PUB -->|"Postgres LISTEN/NOTIFY"| SUBSCRIBE
    PROJECTS --> SUBSCRIBE
    SUBSCRIBE --> FANIN
    FANIN --> MERGED
    MERGED --> DISPATCH
    DISPATCH --> CONFIGS
    CONFIGS --> FILTER
    FILTER -- "non" --> SKIP2
    FILTER -- "oui" --> ROUTER
    ROUTER -- "discord" --> DISCORD
    ROUTER -- "webhook" --> WEBHOOK
    DISCORD -->|"erreur"| ERR
    WEBHOOK -->|"erreur"| ERR
```

---

## 9. Cross-Domain Event Flow — Happy Path & Sad Path

Montre l'interconnexion de tous les domaines sur le chemin complet d'une story : du lancement au coût, avec branches `agent_run` (cost stream) et `hitl_gate` (approval gate), et propagation des événements vers les notifications.

```mermaid
graph TD
    START["POST /projects/{id}/stories/{storyId}/runs\n(RunService.LaunchRun)"]

    GUARD["Guards:\n• story.status != done\n• no active run\n• pipeline config exists\n• agent_id requis par step agent_run"]

    CREATERUN["RunRepo.CreateRun() + CreateRunSteps()\nRun.status = pending"]

    ENQUEUE["JobQueue.EnqueueExecuteRun(runID)\n→ River job async"]

    EXECUTOR["PipelineExecutor.ExecuteRun()\nRun.status = running\nPublish: run.started\nStory.status = running"]

    STEP_LOOP["Pour chaque RunStep (step_order)"]

    BRANCH{{"step.action ?"}}

    subgraph AgentRun["Branche: agent_run"]
        CONTAINER["docker.AgentRuntime\nStartContainer(image, env)"]
        LOGSTREAM["LogStreamer.Stream()\nNDJSON → LogEvent"]
        COSTEXTRACT["LogEvent.Type=cost\n→ []CostEvent accumulés"]
        COSTSVC2["CostService.RecordStepCost()\n→ cost_records INSERT"]
    end

    subgraph HITLBranch["Branche: hitl_gate"]
        HITLCREATE["INSERT hitl_requests (pending)\nstep → waiting_approval"]
        SUSPEND["errStepSuspended\nPipeline suspendu"]
        APPROVE["Reviewer: POST /hitl/{id}/approve\nstep → running → completed"]
        REJECT["Reviewer: POST /hitl/{id}/reject\nstep → failed"]
    end

    STEPDONE["Step completed\nPublish: step.completed"]
    STEPFAIL["Step failed\nPublish: step.failed\nRun.status = failed\nStory.status = failed\nCircuitBreaker.RecordFailure()"]

    RUNDONE["All steps done\nRun.status = completed\nStory.status = done\nPublish: run.completed\nCircuitBreaker.RecordSuccess()"]

    NOTIF["NotificationDispatcher\nfiltre EventsFilter\n→ Discord / Webhook"]

    START --> GUARD
    GUARD -->|"valid"| CREATERUN
    CREATERUN --> ENQUEUE
    ENQUEUE --> EXECUTOR
    EXECUTOR --> STEP_LOOP
    STEP_LOOP --> BRANCH
    BRANCH -- "agent_run" --> CONTAINER
    CONTAINER --> LOGSTREAM
    LOGSTREAM --> COSTEXTRACT
    COSTEXTRACT --> COSTSVC2
    COSTSVC2 --> STEPDONE
    BRANCH -- "hitl_gate" --> HITLCREATE
    HITLCREATE --> SUSPEND
    SUSPEND -->|"approved"| APPROVE
    SUSPEND -->|"rejected"| REJECT
    APPROVE --> STEPDONE
    REJECT --> STEPFAIL
    STEPDONE -->|"next step"| STEP_LOOP
    STEPDONE -->|"last step"| RUNDONE
    STEP_LOOP -->|"error"| STEPFAIL
    RUNDONE --> NOTIF
    STEPFAIL --> NOTIF
```

---

## 10. API Handler → Service → Repository — Chaîne d'appel

Montre une requête HTTP complète (POST /stories) : parsing → validation handler → service → repo → Postgres, avec la gestion des erreurs et le mapping HTTP. Le pattern est identique pour tous les domaines.

```mermaid
sequenceDiagram
    actor Client as Client (HTTP)
    participant Handler as StoryHandler\n(chi + oapi-codegen)
    participant Middleware as ErrorMiddleware\n(DomainError → HTTP)
    participant Service as StoryService
    participant Port as StoryRepository\n(port interface)
    participant Adapter as postgres.StoryRepo\n(sqlc generated)
    participant DB as Postgres

    Client->>Handler: POST /projects/{id}/stories\n{key, title, status, depends_on, ...}

    Handler->>Handler: Parse JSON body\nBind projectID from URL param
    alt JSON invalide
        Handler-->>Client: 400 Bad Request\n{error: {code: VALIDATION, message: ...}}
    end

    Handler->>Service: Create(CreateStoryParams{...})

    Service->>Service: Validate:\n• key != "" et pattern [A-Z0-9]+-N\n• title != "" et <= 255 chars\n• status ∈ {backlog,running,done,failed}\n• scope ∈ {backend,frontend,shared}
    alt Validation échoue
        Service-->>Handler: DomainError{Category=validation}
        Handler->>Middleware: renderError(w, err)
        Middleware-->>Client: 400 {error: {code: STORY_KEY_INVALID, message: ...}}
    end

    Service->>Port: Create(ctx, &Story{...})
    Port->>Adapter: Create(ctx, story)
    Adapter->>DB: INSERT INTO stories\n(project_id, epic_id, key, title,\nstatus, depends_on, ...) RETURNING *

    alt Conflict (key déjà existant)
        DB-->>Adapter: pgx.ErrUniqueViolation
        Adapter-->>Service: DomainError{Category=conflict, Code=STORY_KEY_CONFLICT}
        Service-->>Handler: DomainError
        Middleware-->>Client: 409 {error: {code: STORY_KEY_CONFLICT, message: ...}}
    end

    DB-->>Adapter: Story row
    Adapter-->>Port: *model.Story
    Port-->>Service: *model.Story
    Service-->>Handler: *model.Story

    Handler->>Handler: renderJSON(w, 201, story)
    Handler-->>Client: 201 Created\n{id, key, title, status, depends_on, ...}
```
