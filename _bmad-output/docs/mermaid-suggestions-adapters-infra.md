# Suggestions de Diagrammes Mermaid — Backend Adapters & Infrastructure

Diagrammes recommandés pour enrichir la documentation technique `backend-adapters-infra.md`.

---

## 1. **Flux SSE Event complet (Publish → Stream → Client)**
**Type:** `sequenceDiagram`
**Contenu:** Interaction séquentielle : INSERT event → trigger NOTIFY → EventBus listen → enrichissement BD → dispatch subscribers → SSE write.
**Utilité:** Visualise le chemin critique (8-9 étapes) d'un événement du service au navigateur, clarifie le rôle du trigger Postgres et la récupération asynchrone du payload.

---

## 2. **Architecture globale des Adapters (dépendances)**
**Type:** `graph TD` (flowchart top-down)
**Contenu:** Ports au centre (EventPublisher, GitProvider, TemplateRenderer, etc.) → implémentations (postgres.EventRepo, git.GhCliAdapter, handlebars.Renderer...) → services utilisant ces ports.
**Utilité:** Vue d'ensemble de la séparation ports/adapters, montre qui dépend de qui, aide à localiser une adaptation en cas de modification.

---

## 3. **Lifecycle PipelineExecutor + Action Registry**
**Type:** `stateDiagram-v2`
**Contenu:** États : Init → FetchStep → ResolveAction → Execute → OnSuccess/OnError → Next/Retry → Complete.
**Utilité:** Illustre comment les steps sont exécutées séquentiellement, où les actions sont résolues, quand le metadata partagé entre steps est alimenté/consommé.

---

## 4. **Stratégie Retry incrémental vs full**
**Type:** `flowchart LR`
**Contenu:** Decision tree : step failed → check RetryCount < max_retries → si < max_incremental → inject error_context + log_tail (retry incrémental) vs si >= → no context (retry full) → new AgentRunAction.
**Utilité:** Clarifie la logique complexe du IncrementalRetryAction, aide à comprendre quand l'agent reçoit du contexte d'erreur vs repart de zéro.

---

## 5. **Flux Postgres Pool → Queries → DBTX interface**
**Type:** `classDiagram`
**Contenu:** pgxpool.Pool implements DBTX | Tx implements DBTX | Queries wraps DBTX | Repositories wrap Queries.
**Utilité:** Montre l'abstraction `DBTX` qui autorise transactions transparentes, pattern clé du projet pour la testabilité et la composition transactionnelle.

---

## 6. **Flow River Job Queue : EnqueueExecuteRun → Worker → Executor**
**Type:** `sequenceDiagram`
**Contenu:** HTTP Handler → RunService.LaunchRun() → RunRepo.CreateRun() → JobQueue.EnqueueExecuteRun() → [async] River pulls job → ExecuteRunWorker.Work() → PipelineExecutor.ExecuteRun().
**Utilité:** Visualise l'asynchronisme, montre le timeout River (45min), clarifie que le HTTP response est synchrone mais l'exécution est async.

---

## 7. **Git Provider Factory + Multi-Provider dispatch**
**Type:** `flowchart TD`
**Contenu:** GitProviderFactory.ForProjectID() → switch(project.GitProvider) → GitHub branch (GhCliAdapter + gh CLI) vs Gitea branch (GiteaAPIAdapter + HTTP API + git CLI).
**Utilité:** Aide à comprendre la stratégie multi-provider (GitHub vs Gitea), qui utilise CLI vs HTTP API, résolution du token par projet.

---

## 8. **Postgres LISTEN/NOTIFY + Reconnexion (EventBus)**
**Type:** `sequenceDiagram`
**Contenu:** EventBus.Subscribe() → LISTEN setup → listenLoop() blocking → WaitForNotification(5s timeout) → notification reçue → {reconnect logic + exponential backoff si erreur} → Dispatch → cleanup.
**Utilité:** Illustre la gestion de la résilience (reconnexion, backoff), le timeout 5s, la synchronisation goroutine/channel, critère pour comprendre la latence des SSE.

---

## 9. **Chaîne Template Rendering : Agent → Handlebars → Prompt final**
**Type:** `flowchart LR`
**Contenu:** Agent.TemplateContent (Handlebars) → Renderer.Render(templateContent, TemplateContext) → raymond engine → rendered string → injecté dans env PROMPT_CONTENT du container.
**Utilité:** Illustre la source de vérité (Agent.TemplateContent), le context disponible (story_key, target_files, error_context, diff_content...), le timing du rendu (AgentRunAction).

---

## 10. **Évaluation CI Status : polling + state machine**
**Type:** `stateDiagram-v2`
**Contenu:** Initial → Polling (30s interval) → [gitProvider.GetRemoteCIStatus()] → pass (success) | fail (error) | pending (continue) | no_checks (continue) | network error (warn + continue) → timeout (error).
**Utilité:** Clarifie le polling boucle du CIPollAction, états finaux vs intermédiaires, gestion des erreurs non-bloquantes.

---

## 11. **Hiérarchie Actions + Exécution parallèle/séquentielle**
**Type:** `graph TD`
**Contenu:** ActionRegistry → Actions (git_branch, git_pr, agent_run, ci_poll, hitl_gate, human, notification, incremental_retry) → ExecutionModel (séquence strict, sauf branches parallèles si supported future).
**Utilité:** Vue d'ensemble de toutes les actions disponibles, ordre d'exécution prévisible, évite les suppositions sur la composition (ex: git_pr attend git_branch pour branch_name).

---

## 12. **Container Lifecycle : AgentRunAction (create → start → stream → cleanup)**
**Type:** `sequenceDiagram`
**Contenu:** containerMgr.Create(image, env) → containerMgr.Start() → streamAndWait(logs + cost events) → parse logs → ring buffer last 50 lines → exit code check → costSvc.RecordStepCost() → cleanupContainer(timeout 30s).
**Utilité:** Montre la gestion du lifecycle non-triviale (labels Docker, env injection, log buffering, cost parsing, cleanup avec timeout).

