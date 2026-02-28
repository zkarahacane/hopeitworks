# Diagrammes Mermaid — Adapters & Infrastructure

## 1. Flux SSE Event complet (Publish → Stream → Client)

Chemin critique d'un événement depuis le service métier jusqu'au navigateur. Illustre le rôle du trigger Postgres et la récupération asynchrone du payload complet.

```mermaid
sequenceDiagram
    participant Svc as PipelineExecutor
    participant Repo as EventRepo (postgres)
    participant DB as Postgres
    participant Bus as EventBus (LISTEN/NOTIFY)
    participant SSE as SSEHandler
    participant Cli as Browser (EventSource)

    Svc->>Repo: Publish(event)
    Repo->>DB: INSERT INTO events (id, project_id, entity_type, entity_id, action, payload)
    DB-->>DB: AFTER INSERT trigger: notify_event()
    DB->>DB: pg_notify('events', {id, project_id, entity_type, entity_id, action})
    Note over DB: Payload complet exclu du NOTIFY (limite 8KB)

    DB-->>Bus: WaitForNotification() reçoit notification
    Bus->>DB: eventRepo.GetEventByID(notif.id)
    DB-->>Bus: Event complet avec payload JSON
    Bus->>Bus: Dispatch aux subscribers[notif.ProjectID]
    Bus->>SSE: chan<- model.Event (buffer 100)

    SSE->>SSE: writeSSEEvent(event)
    Note over SSE: event: run.completed\ndata: {...}\nid: <uuid>
    SSE->>Cli: HTTP/1.1 text/event-stream (Flush)
    Cli-->>Cli: EventSource reçoit message
```

---

## 2. Architecture globale des Adapters (dépendances ports/implémentations)

Vue d'ensemble de la séparation ports/adapters. Montre quels services utilisent quels ports, et quelles implémentations fournissent ces ports.

```mermaid
graph TD
    subgraph Services["Domain Services"]
        PS[PipelineExecutor]
        RS[RunService]
        CS[CostService]
    end

    subgraph Ports["Domain Ports"]
        EP[EventPublisher]
        ES[EventSubscriber]
        ER[EventRepository]
        JQ[JobQueue]
        GP[GitProvider]
        GPF[GitProviderFactory]
        TR[TemplateRenderer]
        AR[ActionRegistry]
        NOT[Notifier]
        EMAIL[EmailSender]
        CM[ContainerManager]
    end

    subgraph Adapters["Adapter Implementations"]
        PG_EVT[postgres.EventRepo]
        PG_BUS[postgres.EventBus]
        PG_REPOS[postgres.*Repo x14]
        RIV[river.JobQueue]
        GH[git.GhCliAdapter]
        GITEA[git.GiteaAPIAdapter]
        FACTORY[git.DefaultGitProviderFactory]
        HBS[handlebars.Renderer]
        DISC[discord.Notifier]
        WH[webhook.Notifier]
        SMTP[smtp.EmailSender]
        DOCKER[docker.ContainerManager]
    end

    subgraph Actions["Action Adapter"]
        AGT[agent_run]
        GBR[git_branch]
        GPR[git_pr]
        CI[ci_poll]
        HITL[hitl_gate]
        HUM[human]
        NOTIF[notification]
        RETRY[incremental_retry]
    end

    EP -->|impl| PG_EVT
    ER -->|impl| PG_EVT
    ES -->|impl| PG_BUS
    JQ -->|impl| RIV
    GP -->|impl| GH
    GP -->|impl| GITEA
    GPF -->|impl| FACTORY
    TR -->|impl| HBS
    NOT -->|impl| DISC
    NOT -->|impl| WH
    EMAIL -->|impl| SMTP
    CM -->|impl| DOCKER

    PS -->|uses| EP
    PS -->|uses| AR
    PS -->|uses| JQ
    RS -->|uses| JQ
    AR --> AGT & GBR & GPR & CI & HITL & HUM & NOTIF & RETRY
    AGT -->|uses| CM
    AGT -->|uses| TR
    GBR & GPR & CI & HITL -->|via factory| GPF
    FACTORY -->|reads| PG_REPOS
```

---

## 3. Lifecycle PipelineExecutor + Action Registry

États d'exécution d'un run de pipeline, depuis l'initialisation jusqu'à la complétion ou l'échec.

```mermaid
stateDiagram-v2
    [*] --> Init : ExecuteRun(runID)

    Init --> FetchStep : Charger run + steps depuis DB
    Init --> Failed : Run introuvable

    FetchStep --> Complete : Plus de steps pending
    FetchStep --> ResolveAction : Step suivant trouvé

    ResolveAction --> Execute : ActionRegistry.Get(step.Type)
    ResolveAction --> Failed : Action inconnue

    state Execute {
        [*] --> Running
        Running --> PublishStarted : step.started event
        PublishStarted --> ActionExecute : action.Execute(runCtx)
        ActionExecute --> OnSuccess : exit nil
        ActionExecute --> OnError : exit error
        OnSuccess --> UpdateStepCompleted : UpdateRunStepStatus(completed)
        OnError --> UpdateStepFailed : UpdateRunStepStatus(failed)
        UpdateStepCompleted --> [*]
        UpdateStepFailed --> [*]
    }

    Execute --> FetchStep : Succès → étape suivante
    Execute --> RetryCheck : Erreur + retry_policy définie
    Execute --> Failed : Erreur sans retry

    RetryCheck --> FetchStep : RetryCount < max_retries → IncrementalRetryAction
    RetryCheck --> Failed : RetryCount >= max_retries

    state "Suspended (HITL)" as Suspended
    Execute --> Suspended : hitl_gate / human → waiting_approval
    Suspended --> [*] : Suspension = return nil (non-erreur)

    Complete --> [*]
    Failed --> [*]

    note right of Execute
        Metadata partagé entre steps:
        branch_name, pr_url, agent_image,
        template_content, model
    end note
```

---

## 4. Stratégie Retry incrémental vs full

Arbre de décision de `IncrementalRetryAction` pour déterminer quel type de retry appliquer.

```mermaid
flowchart LR
    A([Step échoué]) --> B{RetryCount\n< max_retries ?}

    B -- Non --> Z([Erreur fatale\nmax retries atteint])

    B -- Oui --> C{RetryCount\n< max_incremental ?}

    C -- Oui --> D[Retry incrémental]
    D --> D1[Injecte error_context\ndans template context]
    D --> D2[Injecte log_tail\n50 dernières lignes]
    D1 & D2 --> D3[Agent reçoit contexte\nd'erreur pour corriger]
    D3 --> E[CreateRetryRunStep\nRetryCount + 1]

    C -- Non --> F[Retry full]
    F --> F1[Supprime error_context\nfrom metadata]
    F --> F2[Pas de log_tail]
    F1 & F2 --> F3[Agent repart\nde zéro]
    F3 --> E

    E --> G[Nouveau RunContext\navec metadata enrichi]
    G --> H([AgentRunAction.Execute\nnouvel appel])
```

---

## 5. Flux Postgres Pool → Queries → DBTX interface

Abstraction `DBTX` qui permet l'utilisation transparente du pool ou d'une transaction dans les repositories.

```mermaid
classDiagram
    class DBTX {
        <<interface>>
        +Exec(ctx, sql, args) CommandTag
        +Query(ctx, sql, args) Rows
        +QueryRow(ctx, sql, args) Row
    }

    class pgxpool_Pool {
        +Exec()
        +Query()
        +QueryRow()
        +Begin(ctx) Tx
    }

    class pgx_Tx {
        +Exec()
        +Query()
        +QueryRow()
        +Commit(ctx)
        +Rollback(ctx)
    }

    class Queries {
        -db DBTX
        +New(db DBTX) Queries
        +WithTx(tx Tx) Queries
    }

    class RunRepo {
        -queries *Queries
        +GetRun(ctx, id) Run
        +CreateRun(ctx, params) Run
        +UpdateRunStepStatus(ctx, id, status)
    }

    class StoryRepo {
        -queries *Queries
        +GetByID(ctx, id) Story
        +UpdateStatus(ctx, id, status)
    }

    class EventRepo {
        -queries *Queries
        +Publish(ctx, event) error
        +GetEventByID(ctx, id) Event
        +GetEventsSince(ctx, projectID, afterID) Events
    }

    DBTX <|.. pgxpool_Pool : implements
    DBTX <|.. pgx_Tx : implements
    Queries --> DBTX : wraps
    RunRepo --> Queries : uses
    StoryRepo --> Queries : uses
    EventRepo --> Queries : uses

    note for Queries "WithTx() permet d'utiliser\nune transaction existante\ntransparemment"
    note for pgxpool_Pool "Pool standard —\ncas non-transactionnel"
    note for pgx_Tx "Transaction explicite —\ncas transactionnel"
```

---

## 6. Flow River Job Queue : EnqueueExecuteRun → Worker → Executor

Visualise l'asynchronisme entre la réponse HTTP synchrone et l'exécution pipeline async par River.

```mermaid
sequenceDiagram
    participant HTTP as HTTP Handler
    participant RS as RunService
    participant RR as RunRepo (postgres)
    participant JQ as JobQueue (river)
    participant PG as Postgres (river_jobs table)
    participant W as ExecuteRunWorker
    participant PE as PipelineExecutor

    HTTP->>RS: LaunchRun(projectID, storyID, configID)
    RS->>RR: CreateRun(params)
    RR-->>RS: run{id, status:pending}
    RS->>JQ: EnqueueExecuteRun(run.id)
    JQ->>PG: INSERT INTO river_jobs {kind:"execute_run", args:{run_id}}
    PG-->>JQ: job inséré
    JQ-->>RS: nil
    RS-->>HTTP: run (202 Accepted)
    HTTP-->>HTTP: Réponse synchrone envoyée

    Note over PG,W: Asynchrone — River poll Postgres

    PG-->>W: River dépile le job (MaxWorkers: 10)
    W->>W: Timeout(job) = 45 minutes
    W->>PE: ExecuteRun(ctx, job.Args.RunID)

    loop Pour chaque step du pipeline
        PE->>PE: ResolveAction(step.Type)
        PE->>PE: action.Execute(runCtx)
        Note over PE: agent_run, git_branch, git_pr,\nci_poll, hitl_gate...
    end

    PE-->>W: nil ou error
    W-->>PG: job completed / failed
```

---

## 7. Git Provider Factory + Multi-Provider dispatch

Stratégie multi-provider : sélection de l'adapter GitHub ou Gitea selon la configuration du projet.

```mermaid
flowchart TD
    A([Service demande\nun GitProvider]) --> B[GitProviderFactory\n.ForProjectID ctx, projectID]

    B --> C[projectRepo.GetByID\nproject.GitProvider\nproject.GitTokenEnv\nproject.RepoURL]

    C --> D{project.GitProvider ?}

    D -- github ou vide --> E[GitHub branch]
    E --> E1[NewGhCliAdapter\nrunner, logger]
    E1 --> E2[GhCliAdapter]
    E2 --> E3[gh CLI\ngh pr create\ngh api repos/...\ngh pr checks]

    D -- gitea --> F[Gitea branch]
    F --> F1[resolveGitToken\nproject.GitTokenEnv → os.Getenv]
    F1 --> F2[extractBaseURL\nfrom project.RepoURL]
    F2 --> F3[NewGiteaAPIAdapter\nbaseURL, token, runner, logger]
    F3 --> F4[GiteaAPIAdapter]
    F4 --> F5[git CLI\ngit clone/branch/push]
    F4 --> F6[HTTP API\nPOST /api/v1/repos/.../pulls\nGET /api/v1/repos/.../statuses]

    D -- autre --> G([Erreur\nunsupported git provider])

    E2 & F4 --> H([port.GitProvider\nCloneRepo, CreateBranch, Push\nCreatePR, MergePR, GetCIStatus\nGetPRDiff, CreateRemoteBranch\nCreateRemotePR, GetRemoteCIStatus])
```

---

## 8. Postgres LISTEN/NOTIFY + Reconnexion (EventBus)

Gestion de la résilience de l'EventBus : reconnexion avec backoff exponentiel, synchronisation goroutine/channel.

```mermaid
sequenceDiagram
    participant APP as Application startup
    participant BUS as EventBus
    participant PG as Postgres (connexion dédiée)
    participant SUB as Subscriber (SSEHandler)

    APP->>BUS: NewEventBus(connString, eventRepo)

    SUB->>BUS: Subscribe(ctx, projectID)
    BUS->>BUS: Crée chan<- model.Event (buffer 100)
    BUS->>BUS: Ajoute subscribers[projectID]

    alt Première subscription
        BUS->>PG: pgx.Connect() connexion dédiée hors pool
        BUS->>PG: LISTEN events
        BUS->>BUS: go listenLoop()
    end

    BUS-->>SUB: <-chan Event, cleanup func

    loop listenLoop goroutine
        BUS->>PG: WaitForNotification(ctx, 5s timeout)
        alt Notification reçue
            PG-->>BUS: notification{channel:"events", payload:{id,project_id,...}}
            BUS->>BUS: JSON decode payload minimal
            BUS->>PG: eventRepo.GetEventByID(id)
            PG-->>BUS: Event complet
            BUS->>BUS: Dispatch aux subscribers[project_id]
            alt Channel plein (non-bloquant)
                BUS->>BUS: Drop + log warning
            else
                BUS->>SUB: chan <- event
            end
        else Timeout 5s
            BUS->>BUS: Continue loop (keepalive implicite)
        else Erreur connexion
            BUS->>BUS: Tentative reconnexion
            loop Backoff exponentiel (1s, 2s, 4s... max 5 tentatives)
                BUS->>PG: pgx.Connect() nouvelle connexion
                BUS->>PG: LISTEN events (re-setup)
                BUS->>BUS: Swap connexion sous mu.Lock()
            end
        end
    end

    SUB->>BUS: cleanup() appelé (déconnexion SSE)
    BUS->>BUS: Retire subscriber du map
    BUS->>BUS: Ferme le channel subscriber

    APP->>BUS: Close()
    BUS->>BUS: Signal stopCh
    BUS->>BUS: Attend doneCh (listenLoop terminé)
    BUS->>PG: conn.Close()
```

---

## 9. Chaîne Template Rendering : Agent → Handlebars → Prompt final

Source de vérité du template, variables disponibles dans le contexte, et timing du rendu dans AgentRunAction.

```mermaid
flowchart LR
    subgraph Source["Source de vérité"]
        A[(Agent.TemplateContent\nHandlebars template\nstocké en DB)]
    end

    subgraph Context["TemplateContext injecté"]
        B[story_key\nstory_title\nstory_objective]
        C[target_files\nacceptance_criteria]
        D[branch_name\nrepo_url]
        E[error_context\nlog_tail\npour les retries]
        F[diff_content\npour les reviews]
    end

    subgraph Render["Rendu Handlebars"]
        G[handlebars.Renderer\n.Render templateContent, ctx]
        H[raymond engine\ngithub.com/aymerick/raymond]
    end

    subgraph Output["Résultat injecté"]
        I[Rendered string\nPrompt final]
        J[Container env var\nPROMPT_CONTENT=...]
        K[Agent Claude Code\nlecture du prompt]
    end

    A --> G
    B & C & D & E & F --> G
    G --> H
    H --> I
    I --> J
    J --> K

    note1([AgentRunAction.Execute\nappelle Renderer juste\navant containerMgr.Create])
    note2([Erreur → DomainError\nTEMPLATE_RENDER_FAILED])
    G -.-> note1
    H -.-> note2
```

---

## 10. Évaluation CI Status : polling + state machine

Machine à états de `CIPollAction` : intervalles de polling, états finaux vs intermédiaires, gestion des erreurs non-bloquantes.

```mermaid
stateDiagram-v2
    [*] --> Initial : CIPollAction.Execute\npr_url, poll_interval=30s, timeout=15min

    Initial --> Polling : Lancer ticker + timer timeout

    state Polling {
        [*] --> WaitTick
        WaitTick --> QueryCI : ticker.C (30s)
        QueryCI --> EvalStatus : gitProvider.GetRemoteCIStatus(pr_url)
    }

    Polling --> Pass : status == "pass"
    Polling --> Fail : status == "fail"
    Polling --> Polling : status == "pending"\npublish ci_poll.checking{pending}
    Polling --> Polling : status == "no_checks"\naucun check configuré encore
    Polling --> Polling : network error\nlog warning, non-bloquant

    Pass --> PublishPass : publish ci_poll.checking{status:pass}
    PublishPass --> [*] : return nil ✓

    Fail --> PublishFail : publish ci_poll.checking{status:fail}
    PublishFail --> [*] : return error CI_FAILED

    Polling --> Timeout : timer.C déclenché
    Timeout --> [*] : return error CI_POLL_TIMEOUT

    note right of Polling
        GitHub: gh api check-runs
        Gitea: GET /commits/{sha}/statuses
        Évaluation locale du résultat agrégé
    end note
```

---

## 11. Hiérarchie Actions + Exécution séquentielle

Vue d'ensemble de toutes les actions disponibles dans l'ActionRegistry, leurs dépendances inter-steps via Metadata, et leur ordre d'exécution.

```mermaid
graph TD
    subgraph Registry["ActionRegistry"]
        AR[ActionRegistry\n.Get step.Type]
    end

    subgraph Actions["Actions disponibles"]
        GB[git_branch\nGitBranchAction]
        GP[git_pr\nGitPRAction]
        AGT[agent_run\nAgentRunAction]
        CI[ci_poll\nCIPollAction]
        HITL[hitl_gate\nHITLGateAction]
        HUM[human\nHumanAction]
        NOT[notification\nNotificationAction]
        RETRY[incremental_retry\nIncrementalRetryAction]
    end

    subgraph Metadata["Metadata partagé RunContext"]
        M1[branch_name]
        M2[pr_url]
        M3[agent_image]
        M4[template_content]
        M5[model]
        M6[error_context]
    end

    subgraph Execution["Ordre d'exécution typique"]
        E1[1 git_branch] --> E2[2 agent_run]
        E2 --> E3[3 git_pr]
        E3 --> E4[4 ci_poll]
        E4 --> E5[5 hitl_gate]
        E5 --> E6[6 notification]
    end

    AR --> GB & GP & AGT & CI & HITL & HUM & NOT & RETRY

    GB -->|écrit| M1
    GP -->|lit| M1
    GP -->|écrit| M2
    CI -->|lit| M2
    HITL -->|lit| M2
    RETRY -->|lit+écrit| M6

    AGT -->|lit| M3
    AGT -->|lit| M4
    AGT -->|lit| M5
    RETRY -->|délègue à| AGT

    style E1 fill:#4a9eff
    style E2 fill:#4a9eff
    style E3 fill:#4a9eff
    style E4 fill:#4a9eff
    style E5 fill:#f0ad4e
    style E6 fill:#5cb85c
```

---

## 12. Container Lifecycle : AgentRunAction (create → start → stream → cleanup)

Gestion complète du lifecycle d'un container agent : injection env, buffering logs, parsing coût, cleanup avec timeout.

```mermaid
sequenceDiagram
    participant AA as AgentRunAction
    participant CM as ContainerManager (docker)
    participant LS as LogStreamer
    participant EP as EventPublisher
    participant CS as CostService
    participant RR as RunRepo

    AA->>AA: Fetch story + project
    AA->>AA: renderer.Render(templateContent, ctx)
    Note over AA: Construit env vars:\nREPO_URL, BRANCH_NAME, STORY_KEY\nPROMPT_CONTENT, GIT_TOKEN\nCLAUDE_CODE_OAUTH_TOKEN, MODEL

    AA->>CM: Create(image, envVars, labels)
    Note over CM: Labels Docker:\nmanaged_by=hopeitworks\nrun_id, step_id, story_key
    CM-->>AA: containerID

    AA->>CM: Start(containerID)
    CM-->>AA: started

    AA->>RR: UpdateRunStep(container_id=containerID)

    AA->>LS: streamAndWait(containerID)
    Note over LS: Ring buffer 50 lignes (log tail)

    loop Pour chaque ligne de log
        LS->>LS: Parse type "cost" ?
        alt Log de coût
            LS->>LS: Accumule costEvents
        else Log normal
            LS->>EP: Publish(log.emitted, {line})
        end
    end

    LS-->>AA: exitCode, logTail, costEvents

    alt exitCode != 0
        AA->>RR: UpdateRunStep(log_tail=logTail)
        AA->>AA: return error (step failed)
    end

    AA->>CS: RecordStepCost(stepID, costEvents)
    Note over CS: Non-fatal si échec

    AA->>CM: cleanupContainer(containerID, timeout=30s)
    Note over CM: Stop container → Remove container\nErreurs loggées en warning\nnon-bloquantes
    CM-->>AA: done
```
