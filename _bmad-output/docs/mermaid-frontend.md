# Diagrammes Mermaid — Frontend

## 1. Architecture globale des Stores et Composables

Flux de dépendances depuis les composants jusqu'au backend : composants Vue → composables → stores Pinia → client API → backend.

```mermaid
graph TD
    subgraph Components["Composants Vue"]
        C1[Feature Components]
        C2[View Components]
        C3[UI Shared Components]
    end

    subgraph Composables["Composables src/composables/"]
        UC1[useAsyncAction]
        UC2[useSSE]
        UC3[useAuth]
        UC4[useStories / useEpics / useProjects]
        UC5[useRunLauncher / useAgentEditor]
        UC6[usePipelineConfig / useCosts]
    end

    subgraph Stores["Stores Pinia src/stores/"]
        S1[auth]
        S2[projects]
        S3[stories / epics]
        S4[runs / epicRun]
        S5[agents / pipelineConfig]
        S6[hitl / approvals]
        S7[users / layout]
    end

    subgraph API["API Client src/api/"]
        API1[apiClient\nopenapi-fetch]
        API2[schema.d.ts\ngénéré depuis openapi.yaml]
    end

    subgraph Backend["Backend Go"]
        BE1[REST API :8080]
        BE2[SSE /events/stream]
    end

    C2 --> UC3
    C2 --> UC4
    C2 --> UC5
    C2 --> UC6
    C1 --> UC1
    C1 --> UC4

    UC3 --> S1
    UC4 --> S3
    UC5 --> S4
    UC6 --> S5
    UC2 --> S4
    UC2 --> S6

    UC1 --> API1
    S1 --> API1
    S2 --> API1
    S3 --> API1
    S4 --> API1
    S5 --> API1
    S6 --> API1
    S7 --> API1

    API1 --> API2
    API1 --> BE1
    UC2 --> BE2
```

---

## 2. Flux d'authentification et guards

Parcours complet depuis la navigation jusqu'à l'accès à une route : restauration de session, vérification d'authentification, guard admin, et redirections.

```mermaid
flowchart TD
    NAV([Navigation vers une route]) --> FIRST{Premier\nchargement ?}

    FIRST -->|Oui| CHECK[auth.checkAuth\nGET /auth/me]
    FIRST -->|Non| REQAUTH

    CHECK --> CHECK_RESULT{Session\nrestaured ?}
    CHECK_RESULT -->|Oui| REQAUTH
    CHECK_RESULT -->|Non| REQAUTH

    REQAUTH{requiresAuth\n!== false ?}

    REQAUTH -->|Non| ADMIN_CHECK
    REQAUTH -->|Oui| IS_AUTH{isAuthenticated ?}

    IS_AUTH -->|Non| REDIRECT_LOGIN[Redirect /login\n?redirect=fullPath]
    IS_AUTH -->|Oui| LOGIN_CHECK{Route = /login ?}

    LOGIN_CHECK -->|Oui| REDIRECT_DASH[Redirect /\nou ?redirect param]
    LOGIN_CHECK -->|Non| ADMIN_CHECK

    ADMIN_CHECK{requiresAdmin\n=== true ?}

    ADMIN_CHECK -->|Non| ACCESS([Accès accordé])
    ADMIN_CHECK -->|Oui| IS_ADMIN{user.role\n=== admin ?}

    IS_ADMIN -->|Oui| ACCESS
    IS_ADMIN -->|Non| REDIRECT_HOME[Redirect /]
```

---

## 3. Cycle de vie SSE et synchronisation temps réel

Séquence complète depuis le montage d'un composant jusqu'à la mise à jour réactive de l'UI via les événements SSE.

```mermaid
sequenceDiagram
    participant Component as Composant Vue
    participant useSSE as useSSE composable
    participant ES as EventSource
    participant Backend as Backend SSE<br/>/events/stream
    participant Store as Store Pinia<br/>(runs / hitl / epicRun)
    participant UI as UI Réactive

    Component->>useSSE: useSSE(projectId, onEvent)
    useSSE->>ES: new EventSource(/api/v1/events/stream?project_id=...)
    ES-->>useSSE: onopen → status = 'open'

    Note over ES,Backend: Connexion HTTP longue durée (Keep-Alive)

    Backend-->>ES: event: run.started\ndata: {...}
    ES->>useSSE: addEventListener callback
    useSSE->>useSSE: JSON.parse(e.data)
    useSSE->>Component: onEvent('run.started', data)
    Component->>Store: store.handleSSEEvent(data)
    Store->>Store: Mise à jour état réactif\n(items, current, pendingCount...)
    Store-->>UI: Vue réagit au changement\n(computed + template re-render)

    Backend-->>ES: event: log.emitted\ndata: {...}
    ES->>useSSE: addEventListener callback
    useSSE->>Component: onEvent('log.emitted', data)
    Component->>Store: stepLogsStore.addLine(data)
    Store-->>UI: LogViewer auto-scroll

    Backend-->>ES: event: hitl.pending\ndata: {...}
    ES->>useSSE: addEventListener callback
    useSSE->>Component: onEvent('hitl.pending', data)
    Component->>Store: hitlStore.handlePendingEvent(data)
    Store-->>UI: Badge AppSidebar\n(pendingCount++)

    Component->>useSSE: onBeforeUnmount → close()
    useSSE->>ES: es.close()
    useSSE->>useSSE: status = 'closed'
```

---

## 4. État et transitions d'une exécution de run

Machines à états valides pour un run : transitions possibles, actions déclenchantes, et états terminaux.

```mermaid
stateDiagram-v2
    [*] --> pending : POST /runs (launch)

    pending --> running : Exécution démarrée\n[run.started SSE]
    pending --> cancelled : cancel()

    running --> paused : pause()
    running --> completed : Tous les steps OK\n[run.completed SSE]
    running --> failed : Step échoué\n[run.failed SSE]
    running --> cancelled : cancel()

    paused --> running : resume()
    paused --> cancelled : cancel()

    failed --> running : retryStep()\n[step retry]

    completed --> [*]
    cancelled --> [*]
    failed --> [*] : Sans retry

    note right of running
        SSE actif
        logs en temps réel
        pause/cancel disponibles
    end note

    note right of paused
        Container suspendu
        resume ou cancel seulement
    end note

    note right of failed
        retryStep() disponible
        sur le step échoué
    end note
```

---

## 5. Anatomie d'une Feature (board comme exemple)

Structure interne de la feature `board` : composants, composables locaux, stores et endpoints API utilisés.

```mermaid
graph TD
    subgraph Views["Views (routes)"]
        V1[BoardView\nproject-board]
        V2[EpicDetailView\nepic-detail]
        V3[StoryDetailView\nstory-detail]
    end

    subgraph Composants["features/board/ — Composants"]
        EC[EpicCardGrid\nEpicCard\nBoardEmptyState]
        EP[EpicDetailLayout\nStoryListPanel\nStoryFilterBar]
        SD[StoryStatusCard\nStoryDetailPanel\nStoryEditorForm]
        DL[CreateStoryDialog\nStoryImportDialog]
        RS[RunStatusIndicator]
    end

    subgraph Composables["Composables utilisés"]
        UE[useEpics]
        US[useStories]
        USE[useStoryEditor]
        USI[useStoryImport]
        URL[useRunLauncher]
    end

    subgraph Stores["Stores Pinia"]
        SE[epics store]
        SS[stories store]
    end

    subgraph API["API Endpoints"]
        A1[GET /projects/:id/epics]
        A2[GET /projects/:id/stories]
        A3[GET /projects/:id/stories/:id]
        A4[PUT /projects/:id/stories/:id]
        A5[POST /projects/:id/stories]
        A6[POST /projects/:id/stories/import]
        A7[POST /projects/:id/stories/:id/runs]
    end

    V1 --> EC
    V2 --> EP
    V2 --> SD
    V2 --> DL
    V3 --> SD

    EC --> UE
    EP --> US
    SD --> USE
    DL --> USI
    SD --> URL
    RS --> SS

    UE --> SE
    US --> SS

    SE --> A1
    SS --> A2
    SS --> A3
    SS --> A4
    SS --> A5
    USI --> A6
    URL --> A7
```

---

## 6. Pipeline de génération de code OpenAPI → Types

Flux complet depuis la spec OpenAPI jusqu'aux types TypeScript et interfaces Go utilisés par le frontend et le backend.

```mermaid
flowchart LR
    SPEC([api/openapi.yaml\nSource unique de vérité])

    SPEC -->|openapi-typescript\nnpm run generate-api| SCHEMA[frontend/src/api/schema.d.ts\nTypes TypeScript générés]
    SPEC -->|oapi-codegen\nmake generate| GOSERVER[backend/internal/api/\nInterfaces Go générées]

    SCHEMA -->|importé par| CLIENT[frontend/src/api/client.ts\nopenapi-fetch typé]

    CLIENT -->|types paths| COMPOSABLES[Composables\napiClient.GET / POST...]
    GOSERVER -->|implémenté par| HANDLERS[backend Handlers\ntype-safe]

    COMPOSABLES -->|requêtes HTTP typées| HANDLERS

    style SPEC fill:#f5a623,color:#000
    style SCHEMA fill:#7ed321,color:#000
    style GOSERVER fill:#7ed321,color:#000
```

---

## 7. Hiérarchie des routes et lazy-loading

Organisation des routes par domaine, distinction authentifiées / publiques, et stratégie de lazy-loading.

```mermaid
graph TD
    ROOT(["/"])

    subgraph Public["Routes publiques (no auth)"]
        LOGIN[/login]
        FORGOT[/forgot-password]
        RESET[/reset-password]
        NF["/:pathMatch(.*)*\n404 Not Found"]
    end

    subgraph Auth["Routes authentifiées"]
        DASH[/\ndashboard]
        PROFILE[/profile\n🔄 lazy]
        RUNS[/runs]
        APPROVALS[/approvals]

        subgraph Projects["Routes projets /projects/"]
            PLIST[/projects]
            PDET["/projects/:id"]
            POVERVIEW["/projects/:id/\noverview"]
            PBOARD["/projects/:id/board"]
            PRUNS["/projects/:id/runs\n🔄 lazy"]
            PPIPE["/projects/:id/pipeline"]
            PAGENTS["/projects/:id/agents"]
            PAGENT_NEW["/projects/:id/agents/new\n🔄 lazy + Admin"]
            PAGENT_EDIT["/projects/:id/agents/:agentId\n🔄 lazy"]
            PSET["/projects/:id/settings\n🔄 lazy"]
            PNOTIF["/projects/:id/settings/notifications\n🔄 lazy"]
            PCOSTS["/projects/:id/costs\n🔄 lazy"]
            STORY["/projects/:id/stories/:storyId"]
            RUN["/runs/:id"]
            HITL["/projects/:id/runs/:runId/approve/:stepId\n🔄 lazy"]

            subgraph Epics["Routes epics"]
                EPIC_DET["/projects/:id/epics/:epicId\n🔄 lazy"]
                EPIC_DAG["/projects/:id/epics/:epicId/dag\n🔄 lazy"]
                EPIC_RUN["/projects/:id/epic-runs/:epicRunId\n🔄 lazy"]
            end
        end

        subgraph Admin["Routes admin (requiresAdmin)"]
            ADMIN_USERS[/admin/users\n🔄 lazy]
        end
    end

    ROOT --> Public
    ROOT --> Auth
    PDET --> POVERVIEW
    PDET --> PBOARD
    PDET --> PRUNS
    PDET --> Epics
    PDET --> PPIPE
    PDET --> PAGENTS
    PAGENTS --> PAGENT_NEW
    PAGENTS --> PAGENT_EDIT
    PDET --> PSET
    PSET --> PNOTIF
    PDET --> PCOSTS
```

---

## 8. Flux de validation et soumission de formulaire

Cycle complet d'un formulaire : saisie utilisateur, validation vee-validate + zod, appel API, gestion de réponse et notification.

```mermaid
flowchart TD
    USER([Utilisateur remplit le formulaire]) --> INPUT[Saisie dans InputText\nSelect / Textarea]

    INPUT --> TOUCHED{Champ\ntouché/blur ?}
    TOUCHED -->|Oui| VALIDATE[vee-validate + zod\nvalidation des règles]
    TOUCHED -->|Non| INPUT

    VALIDATE --> VALID{isValid ?}
    VALID -->|Non| SHOW_ERR[Affiche erreurs inline\nsous chaque champ]
    SHOW_ERR --> INPUT

    VALID -->|Oui| SUBMIT[Clic bouton Submit\nou Enter]
    SUBMIT --> LOADING[isLoading = true\nBouton disabled\nProgressSpinner visible]

    LOADING --> CALL[useAsyncAction.execute\napiClient.POST / PUT]
    CALL --> RESP{Réponse API}

    RESP -->|200 / 201 Success| SUCCESS[data retourné\nisLoading = false]
    SUCCESS --> TOAST_OK[Toast 'success'\nPrimeVue ToastService]
    SUCCESS --> UPDATE[Mise à jour store\n+ emit updated]
    UPDATE --> CLOSE[Fermeture dialog\nou redirect]

    RESP -->|400 Validation| VERR[error = apiError\nisLoading = false]
    VERR --> INLINE_ERR[Message inline\n400 = erreur utilisateur]

    RESP -->|409 Conflict| CERR[error = apiError\nexemple: ALREADY_RUNNING_ERROR]
    CERR --> TOAST_WARN[Toast 'warn'\nmessage spécifique]

    RESP -->|500 Server| SERR[error = Error\nisLoading = false]
    SERR --> TOAST_ERR[Toast 'error'\nerreur transiente]
```

---

## 9. Composition CSS (layers + Tailwind + PrimeVue)

Ordre de cascade CSS des trois layers et leurs zones de responsabilité respectives.

```mermaid
graph BT
    subgraph L1["Layer 1 — tailwind-base"]
        TB1[tailwindcss/preflight.css\nCSS resets navigateurs]
        TB2[tailwindcss/theme.css\nVariables CSS Tailwind]
    end

    subgraph L2["Layer 2 — primevue"]
        PV1[PrimeVue Aura preset\nHopeTheme definePreset]
        PV2[Composants UI\nButton Tag DataTable Dialog...]
        PV3[Design tokens\nblue.50 → blue.950\nsurface light/dark]
    end

    subgraph L3["Layer 3 — tailwind-utilities"]
        TU1[tailwindcss/utilities.css\nflex grid gap p- m- w- h-]
        TU2[Utilitaires layout uniquement\npas de couleurs ni typo]
    end

    subgraph CUSTOM["Animations custom (hors layers)"]
        AN1[pulse-run keyframe\n.running-indicator]
    end

    L1 -->|"surchargé par"| L2
    L2 -->|"surchargé par"| L3
    L3 -.->|"cohabite avec"| CUSTOM

    note1["Les utilitaires Tailwind\nsurpassent PrimeVue\ngrâce à l'ordre des layers"]

    style L1 fill:#e8f4f8
    style L2 fill:#fff3e0
    style L3 fill:#e8f5e9
    style CUSTOM fill:#f3e5f5
```

---

## 10. Modèle de données SSE et événements temps réel

Hiérarchie des 18 types d'événements SSE, leurs familles et les stores Pinia qui les consomment.

```mermaid
classDiagram
    class SSEBaseEvent {
        +String eventName
        +Unknown data
        +parse() void
    }

    class RunEvent {
        +String run_id
        +String project_id
        +String story_key
        +String status
    }

    class StepEvent {
        +String run_id
        +String step_id
        +String status
        +Number cost_usd
    }

    class LogEvent {
        +String run_id
        +String step_id
        +String line
        +String timestamp
    }

    class HITLEvent {
        +String hitl_request_id
        +String run_id
        +String step_id
        +String project_id
        +String story_key
        +String pr_url
    }

    class StoryEvent {
        +String story_id
        +String project_id
        +String new_status
    }

    class EpicRunEvent {
        +String epic_run_id
        +String project_id
        +String story_key
        +String status
    }

    SSEBaseEvent <|-- RunEvent : run.started\nrun.completed\nrun.failed\nrun.cancelled
    SSEBaseEvent <|-- StepEvent : step.started\nstep.completed\nstep.failed\nstep.cancelled
    SSEBaseEvent <|-- LogEvent : log.emitted
    SSEBaseEvent <|-- HITLEvent : hitl.pending\nhitl.approved\nhitl.rejected
    SSEBaseEvent <|-- StoryEvent : story.status_updated
    SSEBaseEvent <|-- EpicRunEvent : epic_run.started\nepic_run.group.started\nepic_run.story.completed\nepic_run.completed\nepic_run.failed

    class RunsStore {
        +handleSSEEvent()
        +updateRunStatus()
    }

    class HITLStore {
        +handlePendingEvent()
        +handleResolvedEvent()
    }

    class EpicRunStore {
        +handleSSEEvent()
    }

    RunEvent --> RunsStore : consommé par
    StepEvent --> RunsStore : consommé par
    HITLEvent --> HITLStore : consommé par
    EpicRunEvent --> EpicRunStore : consommé par
```
