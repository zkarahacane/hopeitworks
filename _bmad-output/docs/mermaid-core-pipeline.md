# Diagrammes Mermaid — Backend Core Pipeline

## 1. Architecture Hexagonale

Vue d'ensemble des couches hexagonales : domaine au centre, ports comme contrats, adapteurs en périphérie. Flux d'import unidirectionnel strict.

```mermaid
graph TB
    subgraph Externe["Monde extérieur"]
        HTTP["Client HTTP"]
        Docker["Docker API"]
        PG["PostgreSQL"]
        GH["GitHub / Gitea"]
        River["River (Job Queue)"]
    end

    subgraph API["api/ (Handlers + Middleware)"]
        H["Handlers\n(oapi-codegen)"]
        MW["Middleware\n(JWT, RBAC)"]
    end

    subgraph Domain["domain/ (Logique métier)"]
        SVC["Services\n(PipelineExecutor, RunService,\nEpicRunService, SchedulerService...)"]
        MODEL["Models\n(Run, RunStep, DAGResult,\nEpicRun, Action...)"]
        PORT["Ports\n(RunRepository, ContainerManager,\nJobQueue, GitProvider...)"]
    end

    subgraph Adapter["adapter/ (Implémentations)"]
        PGA["postgres/\n(Repositories sqlc)"]
        DOCKER["docker/\n(ContainerManager)"]
        RIVERA["river/\n(JobQueue)"]
        GITA["git/\n(GitProvider, GitProviderFactory)"]
        ACTIONA["action/\n(AgentRun, GitBranch, GitPR,\nCIPoll, HITLGate, Notification...)"]
        HBS["handlebars/\n(TemplateRenderer)"]
    end

    HTTP --> H
    MW --> H
    H --> SVC
    SVC --> PORT
    SVC --> MODEL
    PORT -.->|implémente| PGA
    PORT -.->|implémente| DOCKER
    PORT -.->|implémente| RIVERA
    PORT -.->|implémente| GITA
    PORT -.->|implémente| ACTIONA
    PORT -.->|implémente| HBS

    PGA --> PG
    DOCKER --> Docker
    RIVERA --> River
    GITA --> GH
    River --> SVC

    style Domain fill:#e8f4f8,stroke:#2980b9
    style Adapter fill:#fef9e7,stroke:#f39c12
    style API fill:#f9ebea,stroke:#e74c3c
    style Externe fill:#f4f4f4,stroke:#7f8c8d
```

---

## 2. Machine d'états — Run

Tous les états d'un Run avec transitions valides. Les états `completed` et `cancelled` sont terminaux.

```mermaid
stateDiagram-v2
    [*] --> pending : CreateRun

    pending --> running : LaunchRun / River job
    pending --> cancelled : CancelRun

    running --> paused : PauseRun
    running --> completed : tous les steps OK
    running --> failed : un step échoue
    running --> cancelled : CancelRun

    paused --> running : ResumeRun (re-enqueue)
    paused --> cancelled : CancelRun

    failed --> running : RetryStep (re-enqueue)

    completed --> [*]
    cancelled --> [*]

    note right of running
        Story transite aussi :
        backlog → running → done/failed
    end note

    note right of failed
        Circuit Breaker
        enregistre l'échec
    end note
```

---

## 3. Machine d'états — RunStep

États d'un RunStep avec transitions. Le passage par `waiting_approval` représente la suspension HITL.

```mermaid
stateDiagram-v2
    [*] --> pending : CreateRunStep

    pending --> running : executeStep() début
    pending --> cancelled : CancelRun

    running --> completed : Action.Execute() OK
    running --> failed : Action.Execute() erreur
    running --> cancelled : contexte annulé
    running --> waiting_approval : HITLGateAction / HumanAction

    waiting_approval --> completed : HITLService.Approve() → step déjà approuvé
    waiting_approval --> running : Resume après approbation (re-enqueue)
    waiting_approval --> failed : HITLService.Reject()
    waiting_approval --> cancelled : CancelRun

    completed --> [*]
    failed --> [*]
    cancelled --> [*]

    note right of waiting_approval
        PipelineExecutor détecte
        waiting_approval via re-fetch
        → retourne errStepSuspended
        → pipeline s'arrête proprement
    end note
```

---

## 4. Flux d'exécution — PipelineExecutor.ExecuteRun()

Algorithme complet de l'exécution d'un run, avec points de décision : circuit breaker, pause, annulation, suspension HITL.

```mermaid
flowchart TD
    START([ExecuteRun appelé par River worker]) --> FETCH[Récupère le Run depuis RunRepository]
    FETCH --> CB{Circuit Breaker\nouvert ?}
    CB -- oui --> CBFAIL[Marque run failed\nRetourne erreur CB]
    CBFAIL --> END_ERR([Fin — Échec CB])

    CB -- non --> SORT[Trie les steps par step_order]
    SORT --> TRANSITION[Run: pending → running\nPublie run.started]
    TRANSITION --> STORY[Story: backlog → running\nbest-effort]
    STORY --> MERGE_META[Merge metadata persistées\ndans le dictionnaire partagé]
    MERGE_META --> LOOP_START{Prochain step\nnon-complété ?}

    LOOP_START -- non, tous OK --> COMPLETE[Run: running → completed\nPublie run.completed\nStory: → done\nReset circuit breaker]
    COMPLETE --> END_OK([Fin — Succès])

    LOOP_START -- oui --> CTX_CANCEL{Contexte\nannulé ?}
    CTX_CANCEL -- oui --> CANCEL[handleCancellation()\nStep + run → cancelled\nPublie events cancelled]
    CANCEL --> END_CANCEL([Fin — Annulé])

    CTX_CANCEL -- non --> PAUSE_CHK{Run en\npause en DB ?}
    PAUSE_CHK -- oui --> RETURN_PAUSED([Retourne ErrRunPaused])

    PAUSE_CHK -- non --> EXEC[executeStep()]
    EXEC --> SUSPENDED{Step est\nwaiting_approval ?}
    SUSPENDED -- oui --> RETURN_NIL([Retourne nil — pipeline suspendu])

    SUSPENDED -- non --> STEP_OK{Step OK ?}
    STEP_OK -- oui --> LOOP_START

    STEP_OK -- non --> FAIL[handleStepFailure()\nStep → failed, run → failed\nStory → failed\nRecord CB failure]
    FAIL --> END_FAIL([Fin — Échec step])
```

---

## 5. Algorithme DAG — Kahn's Algorithm (SchedulerService.BuildDAG)

Construction du graphe de dépendances et tri topologique couche par couche.

```mermaid
flowchart TD
    INPUT([stories: []Story]) --> INDEX[Indexation par clé\nmap storyKey → Story]
    INDEX --> GRAPH[Construction du graphe\nadjacency list + in-degree]

    GRAPH --> EXPLICIT[Arêtes explicites\nStory.DependsOn → arêtes directes\ncles inconnues ignorées]
    GRAPH --> IMPLICIT[Arêtes implicites\nConflits fichiers TargetFiles\nSérialisation par ordre lexicographique des clés]

    EXPLICIT --> TOPO[Tri topologique — Kahn]
    IMPLICIT --> TOPO

    TOPO --> ZEROIN[Collecte noeuds\nin-degree = 0\ntri lexicographique → déterminisme]
    ZEROIN --> EMPTY{Aucun noeud\nà in-degree 0 ?}
    EMPTY -- noeud restants --> CYCLE([Erreur: DAG_CYCLE_DETECTED])
    EMPTY -- liste non vide --> LAYER[Groupe les noeuds → couche i]

    LAYER --> DECR[Décrémente in-degree\ndes dépendants]
    DECR --> DONE{Tous les noeuds\ntraités ?}
    DONE -- non --> ZEROIN
    DONE -- oui --> RESULT([DAGResult.Groups\ncouches d'exécution parallèle])

    note1["Exemples:\n• S-01, S-02 indépendants → [[S-01, S-02]]\n• Chaîne A→B→C → [[A],[B],[C]]\n• Diamant → [[A],[B,C],[D]]"]
    RESULT --- note1
```

---

## 6. Orchestration Epic Run — ParallelGroupExecutor

Séquence d'interactions entre services pour l'exécution parallèle par couche DAG avec sémantique fail-fast.

```mermaid
sequenceDiagram
    participant Client as Client REST
    participant ERS as EpicRunService
    participant SCH as SchedulerService
    participant PGE as ParallelGroupExecutor
    participant RS as RunService
    participant PE as PipelineExecutor

    Client->>ERS: LaunchEpicRun(projectID, epicID)
    ERS->>SCH: BuildDAG(stories)
    SCH-->>ERS: DAGResult [[layer0], [layer1], ...]
    ERS->>ERS: CreateEpicRun (status: pending)
    ERS->>ERS: InsertEpicRunStory × N (group_index)
    ERS->>PGE: Execute(epicRun, dag) [goroutine détachée]
    ERS-->>Client: 202 Accepted {epic_run_id, status: scheduling}

    Note over PGE: context.WithoutCancel() — découplé du request
    PGE->>PGE: EpicRun: pending → running

    loop Pour chaque couche DAG (séquentielle)
        PGE->>PGE: Publie epic_run_group.started

        par Stories en parallèle (errgroup)
            PGE->>RS: LaunchRun(storyID) → Run pending
            RS-->>PGE: Run créé avec steps
            PGE->>PE: ExecuteRun(runID) [direct, pas job queue]
            PE-->>PGE: OK / Erreur
            PGE->>PGE: UpdateEpicRunStory status
        end

        alt Une story échoue
            PGE->>PGE: EpicRun: running → failed [fail-fast]
            PGE->>PGE: Publie epic_run.failed
        else Toutes réussies
            Note over PGE: Couche suivante...
        end
    end

    PGE->>PGE: EpicRun: running → completed
    PGE->>PGE: Publie epic_run.completed
```

---

## 7. Flux des Metadata entre Actions

Producteurs et consommateurs des clés du dictionnaire `RunContext.Metadata`. Les steps communiquent exclusivement via ce mécanisme.

```mermaid
graph LR
    subgraph Producteurs
        LAUNCH["RunService.LaunchRun()\n+ Executor (per-step)"]
        GITBRANCH["GitBranchAction"]
        GITPR["GitPRAction"]
        INCRETRY["IncrementalRetryAction"]
    end

    subgraph Metadata["RunContext.Metadata"]
        BN["branch_name"]
        PU["pr_url"]
        MODEL["model"]
        TC["template_content"]
        AID["agent_id"]
        AIMG["agent_image"]
        EC["error_context"]
        LT["log_tail"]
    end

    subgraph Consommateurs
        AGENTRUN["AgentRunAction"]
        GITPR2["GitPRAction"]
        CIPOLL["CIPollAction"]
        HITL["HITLGateAction"]
        NOTIF["NotificationAction"]
    end

    LAUNCH -->|"snapshot agent attrs"| MODEL
    LAUNCH -->|"snapshot agent attrs"| TC
    LAUNCH -->|"snapshot agent attrs"| AID
    LAUNCH -->|"snapshot agent attrs"| AIMG
    LAUNCH -->|"branch_name initial"| BN

    GITBRANCH -->|"écrit"| BN
    GITPR -->|"écrit"| PU
    INCRETRY -->|"écrit"| EC
    INCRETRY -->|"écrit"| LT

    BN -->|"lit"| AGENTRUN
    BN -->|"lit"| GITPR2
    BN -->|"lit"| NOTIF
    PU -->|"lit"| CIPOLL
    PU -->|"lit"| HITL
    PU -->|"lit"| NOTIF
    MODEL -->|"lit"| AGENTRUN
    TC -->|"lit"| AGENTRUN
    AID -->|"lit"| AGENTRUN
    AIMG -->|"lit"| AGENTRUN
    EC -->|"lit"| AGENTRUN
    LT -->|"lit"| AGENTRUN

    style Metadata fill:#f0f7fb,stroke:#3498db
```

---

## 8. Action Registry — Pattern de dispatch

Peuplement du registre au démarrage et résolution des actions à l'exécution des steps.

```mermaid
flowchart TD
    subgraph INIT["Démarrage main.go (étape 7)"]
        REG[InMemoryActionRegistry]
        A1[AgentRunAction] -->|Register name=agent_run| REG
        A1 -->|RegisterAlias implement| REG
        A1 -->|RegisterAlias review| REG
        A1 -->|RegisterAlias merge| REG
        A2[GitBranchAction] -->|Register name=git_branch| REG
        A3[GitPRAction] -->|Register name=git_pr| REG
        A4[CIPollAction] -->|Register name=ci_poll| REG
        A5[HITLGateAction] -->|Register name=hitl_gate| REG
        A6[HumanAction] -->|Register name=human| REG
        A7[NotificationAction] -->|Register name=notification| REG
        A8[IncrementalRetryAction] -->|Register name=incremental_retry| REG
    end

    subgraph EXEC["executeStep() — runtime"]
        STEP[RunStep.Action\n ex: 'agent_run'] --> LOOKUP["ActionRegistry.Get(step.action)"]
        LOOKUP --> FOUND{Action\ntrouvée ?}
        FOUND -- non --> NOTFOUND[Erreur ACTION_NOT_FOUND\nStep → failed]
        FOUND -- oui --> EXECUTE["Action.Execute(ctx, runCtx)"]
        EXECUTE --> RESULT{Résultat}
        RESULT -- nil --> SUCCESS[Step → completed]
        RESULT -- error --> FAILURE[handleStepFailure()]
        RESULT -- waiting_approval --> SUSPEND[errStepSuspended\nPipeline suspendu]
    end

    REG -.->|injecté dans PipelineExecutor| LOOKUP
```

---

## 9. Circuit Breaker — Logique de décision

Protection contre les cascades d'échecs, état stocké sur l'entité Project en base.

```mermaid
flowchart TD
    START([ExecuteRun — avant exécution]) --> CHECK["CheckCircuitBreaker(projectID)"]
    CHECK --> ACTIVE{Project.\nCircuitBreakerActive ?}

    ACTIVE -- true --> BLOCKED[Retourne CIRCUIT_BREAKER_OPEN\nRun → failed immédiatement]
    BLOCKED --> END_BLOCK([Fin — bloqué])

    ACTIVE -- false --> RUN[Exécution du pipeline...]

    RUN --> RESULT{Résultat}

    RESULT -- Succès --> SUCCESS["RecordSuccess()\nCircuitBreakerCount = 0\nPas de trip possible"]
    SUCCESS --> END_OK([Fin — OK])

    RESULT -- Échec step --> FAILURE["RecordFailure()\nCircuitBreakerCount++"]
    FAILURE --> THRESHOLD{Count >=\nCircuitBreakerMax ?}
    THRESHOLD -- non --> END_FAIL([Fin — Échec enregistré])
    THRESHOLD -- oui --> TRIP["CircuitBreakerActive = true\nPublie circuit_breaker.tripped\nBloque tous les prochains runs du projet"]
    TRIP --> END_TRIP([Fin — CB ouvert])

    RESET([Admin: Reset]) --> RESET_OP["Reset()\nCircuitBreakerCount = 0\nCircuitBreakerActive = false\nPublie circuit_breaker.reset"]
    RESET_OP --> ACTIVE
```

---

## 10. Retry Flow — Incremental vs Full

Décision de type de retry selon le compteur, et données injectées dans chaque stratégie.

```mermaid
flowchart TD
    START([Client POST /runs/runId/steps/stepId/retry]) --> VALIDATE{Step en\nétat failed ?}
    VALIDATE -- non --> ERR_STATE([Erreur: état invalide])
    VALIDATE -- oui --> POLICY[Vérifie RetryPolicy\ndepuis PipelineConfig snapshot]

    POLICY --> LIMIT{RetryCount >=\nMaxRetries ?}
    LIMIT -- oui --> ERR_LIMIT([Erreur: limite atteinte])

    LIMIT -- non --> TYPE{RetryCount\n< MaxIncremental\n(1 ou 2) ?}

    TYPE -- oui, retry incrémental --> INCR["Type: incremental\nInjecte dans Metadata:\n• error_context = step.ErrorMessage\n• log_tail = step.LogTail\nL'agent reçoit le contexte d'erreur\npour cibler la correction"]

    TYPE -- non, retry complet --> FULL["Type: full\nNettoie Metadata:\n• error_context = ''\n• log_tail = ''\nRelance l'agent depuis zéro"]

    INCR --> CREATE["CreateRetryRunStep()\nparent_step_id = stepID\nretry_count++\nretry_type = 'incremental'"]
    FULL --> CREATE2["CreateRetryRunStep()\nparent_step_id = stepID\nretry_count++\nretry_type = 'full'"]

    CREATE --> RESUME["Run: failed → running\nEnqueue execute_run job"]
    CREATE2 --> RESUME

    RESUME --> PE["PipelineExecutor.ExecuteRun()\nSkip steps completed\nExécute le retry step"]
```

---

## 11. Flux Complet — Single Story Run

Séquence complète depuis l'appel REST jusqu'à la completion, passant par le job queue River.

```mermaid
sequenceDiagram
    participant C as Client REST
    participant H as RunHandler
    participant RS as RunService
    participant DB as PostgreSQL
    participant RQ as River JobQueue
    participant PE as PipelineExecutor
    participant AR as ActionRegistry
    participant ACT as Action (ex: AgentRun)
    participant EP as EventPublisher

    C->>H: POST /projects/{pid}/stories/{sid}/runs
    H->>RS: LaunchRun(projectID, storyID)

    RS->>DB: Valide story (exists, not done, no active run)
    RS->>DB: GetByProjectID → PipelineConfig YAML
    RS->>RS: Parse YAML → valide action_types, agents
    RS->>DB: Resolve agents → snapshot model/image/template_content
    RS->>DB: CreateRun (status: pending, metadata)
    RS->>DB: CreateRunStep × N (step_order 0..N)
    RS->>RQ: EnqueueExecuteRun(runID)
    RS-->>H: Run{status: pending, steps: [...]}
    H-->>C: 201 Created {run}

    Note over RQ,PE: Exécution asynchrone

    RQ->>PE: ExecuteRun(runID)
    PE->>DB: GetRun + ListRunStepsByRun
    PE->>PE: CheckCircuitBreaker
    PE->>DB: UpdateRunStatus → running
    PE->>EP: Publie run.started
    PE->>DB: UpdateStoryStatus → running
    PE->>EP: Publie story.status_updated

    loop Pour chaque step
        PE->>DB: UpdateRunStepStatus → running
        PE->>EP: Publie step.started
        PE->>AR: Get(step.action)
        AR-->>PE: Action impl
        PE->>ACT: Execute(ctx, runCtx)
        ACT-->>PE: nil (succès)
        PE->>DB: UpdateRunStepStatus → completed
        PE->>EP: Publie step.completed
    end

    PE->>DB: UpdateRunStatus → completed
    PE->>EP: Publie run.completed
    PE->>DB: UpdateStoryStatus → done
    PE->>EP: Publie story.status_updated
    PE->>PE: RecordSuccess (circuit breaker)
```

---

## 12. Lifecycle Container Agent et Labels

Création, démarrage, streaming de logs, cleanup. TimeoutEnforcer et OrphanCleaner utilisent les labels pour la supervision.

```mermaid
flowchart TD
    subgraph AGENT_RUN["AgentRunAction.Execute()"]
        CREATE["ContainerManager.Create(ContainerOpts)\nLabels: managed_by=hopeitworks\n        run_id=UUID\n        step_id=UUID\n        story_key=S-XX\nEnv: REPO_URL, BRANCH_NAME, STORY_KEY\n     PROMPT_CONTENT, GIT_TOKEN, MODEL\n     CLAUDE_CODE_OAUTH_TOKEN"]
        START["ContainerManager.Start(containerID)"]
        PERSIST["UpdateRunStepContainerInfo(containerID)"]
        STREAM["LogStreamer.StreamLogs()\nParse NDJSON en temps réel\n→ ring buffer log_tail\n→ accumule events 'cost'"]
        WAIT["ContainerManager.Wait()\nexit code"]
        CHECK{exit code\n== 0 ?}
        COST["CostService.RecordStepCost()\ntokens accumulés"]
        CLEANUP["Stop() + Remove()\n[defer — toujours exécuté]"]
        FAIL_LOG["Persiste log_tail sur step\nRetourne erreur"]

        CREATE --> START --> PERSIST --> STREAM --> WAIT --> CHECK
        CHECK -- oui --> COST --> CLEANUP
        CHECK -- non --> FAIL_LOG --> CLEANUP
    end

    subgraph SUPERVISORS["Services de supervision (background)"]
        TIMEOUT["TimeoutEnforcer\n(ticker toutes les 30s)"]
        ORPHAN["OrphanCleaner\n(une fois au démarrage)"]

        TIMEOUT --> LIST_T["ListContainers\n(label: managed_by=hopeitworks)"]
        LIST_T --> CHECK_T{temps écoulé\n> timeout projet ?}
        CHECK_T -- oui --> FORCE["Stop container\nStep + Run → failed\n(raison: container_timeout)"]

        ORPHAN --> LIST_O["ListContainers\n(label: managed_by=hopeitworks)"]
        LIST_O --> CHECK_O{run_id absent\nou run inactif ?}
        CHECK_O -- oui --> REMOVE["Remove container orphelin"]
    end

    CLEANUP -.->|libère le container| SUPERVISORS
```

---

## 13. Cost Tracking — Du container au stockage

Flux de parsing des logs NDJSON d'un container agent pour extraire et enregistrer les coûts.

```mermaid
flowchart TD
    CONTAINER["Container agent\n(Claude Code)"]
    NDJSON["STDOUT — flux NDJSON\n{type: 'log', message: '...'}\n{type: 'cost', input_tokens: 1200, output_tokens: 450, model: 'claude-opus-4-6'}\n{type: 'log', message: '...'}"]

    CONTAINER --> NDJSON
    NDJSON --> STREAMER["LogStreamer.StreamLogs()\nLit ligne par ligne"]

    STREAMER --> PARSE{Parse JSON\ntype ?}

    PARSE -- type='log' --> LOG_EVENT["Crée LogEvent{Type: log}\nForward channel <-chan LogEvent"]
    PARSE -- type='cost' --> COST_EVENT["Crée LogEvent{Type: cost\nInputTokens, OutputTokens, Model}"]
    PARSE -- invalide --> SKIP["Log warning\nIgnore la ligne"]

    LOG_EVENT --> FORWARD["Events forwarded\nvers EventPublisher\n(run_log.received)"]

    COST_EVENT --> ACCUM["Accumulation en mémoire\ndans AgentRunAction\ncumulateur par step"]

    ACCUM --> EXIT{Container\nterminé avec succès ?}
    EXIT -- oui --> RECORD["CostService.RecordStepCost()\nInputTokens, OutputTokens, Model\nStepID, RunID"]
    EXIT -- non --> DISCARD["Coûts partiels discardés\n(step failed)"]

    RECORD --> DB["Persist en DB\npour reporting"]

    style COST_EVENT fill:#fef9e7,stroke:#f39c12
    style RECORD fill:#eafaf1,stroke:#27ae60
```

---

## 14. HITL Gate — Suspension et Reprise

Séquence complète de suspension du pipeline par une gate HITL et sa reprise après approbation humaine.

```mermaid
sequenceDiagram
    participant PE as PipelineExecutor
    participant HITL as HITLGateAction
    participant DB as PostgreSQL
    participant EP as EventPublisher
    participant U as Utilisateur
    participant API as HITLHandler
    participant HS as HITLService

    Note over PE,HITL: Phase 1 — Suspension
    PE->>PE: executeStep() — step hitl_gate
    PE->>DB: UpdateRunStepStatus → running
    PE->>EP: Publie step.started
    PE->>HITL: Execute(ctx, runCtx)

    HITL->>DB: Fetch story (story_key)
    opt pr_url dans metadata
        HITL->>HITL: GitProvider.GetPRDiff(pr_url) [non-fatal]
    end
    HITL->>DB: CreateHITLRequest{gate_type: approval, diff_content}
    HITL->>DB: UpdateRunStepStatus → waiting_approval
    HITL->>EP: Publie hitl_gate.pending
    HITL-->>PE: return nil (pas une erreur)

    PE->>DB: Re-fetch step → détecte waiting_approval
    PE->>PE: return errStepSuspended (sentinel interne)
    PE->>PE: ExecuteRun() catch errStepSuspended → return nil
    Note over PE: Pipeline suspendu proprement\nRun reste "running"\nStep reste "waiting_approval"

    Note over U,HS: Phase 2 — Reprise (plus tard)
    U->>API: POST /hitl-requests/{id}/approve
    API->>HS: Approve(requestID)
    HS->>DB: HITLRequest: pending → approved
    HS->>DB: UpdateRunStepStatus: waiting_approval → completed
    HS->>EP: Publie hitl_gate.approved
    HS->>PE: EnqueueExecuteRun(runID) [via JobQueue]

    Note over PE: Nouvelle exécution River
    PE->>DB: ListRunStepsByRun
    PE->>PE: Skip steps completed (incl. step HITL)
    PE->>PE: Continue avec les steps restants...
```

---

## 15. Ordre d'initialisation — Dependency Injection (main.go)

Les 14 étapes d'initialisation dans `main.go`, avec les dépendances d'ordre entre composants.

```mermaid
flowchart TD
    S1["1. Config\ninternalconfig.Load()"]
    S2["2. Logger\nslog structuré JSON"]
    S3["3. Database\npgx pool + golang-migrate\n(auto-migrations)"]
    S4["4. Event Bus\nconnexion LISTEN/NOTIFY\nPostgres dédiée"]
    S5["5. Repositories\nsqlc — RunRepo, StoryRepo,\nProjectRepo, EpicRunRepo..."]
    S6["6. Services métier\nAuthService, ProjectService,\nStoryService, etc."]
    S7["7. Actions\nEnregistrement ActionRegistry\n8 actions + aliases legacy"]
    S8["8. PipelineExecutor\nActionRegistry + EventPublisher\n+ CircuitBreakerService"]
    S9["9. River JobQueue\nWorkers enregistrés\nClient démarré en goroutine"]
    S10["10. RunService\nRunRepo + PipelineConfigRepo\n+ JobQueue + ContainerMgr"]
    S11["11. Background services\nOrphanCleaner (startup)\nTimeoutEnforcer (goroutine)\nToken cleanup"]
    S12["12. Epic Run orchestration\nParallelGroupExecutor\n+ EpicRunService"]
    S13["13. HTTP Router\nchi + middlewares\n+ handler registration"]
    S14["14. Graceful Shutdown\nSIGTERM/SIGINT handling"]

    S1 --> S2 --> S3 --> S4 --> S5
    S5 --> S6
    S5 --> S7
    S7 --> S8
    S4 --> S8
    S3 --> S9
    S8 --> S9
    S8 --> S10
    S9 --> S10
    S5 --> S10
    S10 --> S11
    S8 --> S12
    S10 --> S12
    S12 --> S13
    S6 --> S13
    S13 --> S14

    style S1 fill:#fadbd8,stroke:#e74c3c
    style S3 fill:#d5f5e3,stroke:#27ae60
    style S8 fill:#d6eaf8,stroke:#2980b9
    style S9 fill:#fef9e7,stroke:#f39c12
    style S14 fill:#f4ecf7,stroke:#8e44ad
```
