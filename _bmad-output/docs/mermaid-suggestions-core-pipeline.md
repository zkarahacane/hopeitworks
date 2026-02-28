# Suggestions Diagrammes Mermaid — Documentation Pipeline Core

Suggestions de diagrammes Mermaid pour enrichir la documentation backend pipeline.

---

## 1. Hexagonal Architecture Overview
**Type**: `graph TB` (architecture diagram)

**Ce qu'il montre** : Couches hexagonales (domain au centre, ports, adapteurs périphériques) avec flux d'import unidirectionnel `handler -> service -> port <- adapter`.

**Pourquoi** : Visualise clairement la séparation des responsabilités et les dépendances, utile pour les nouveaux contributeurs.

---

## 2. Run State Machine
**Type**: `stateDiagram-v2`

**Ce qu'il montre** : Tous les états Run (pending, running, paused, completed, failed, cancelled) avec transitions valides et conditions de changement (ex: `running --pause--> paused`, `failed --retry--> running`).

**Pourquoi** : Les transitions sont complexes (notamment le retry qui passe par failed -> running). Un diagram clarifie les chemins autorisés.

---

## 3. RunStep State Machine
**Type**: `stateDiagram-v2`

**Ce qu'il montre** : États RunStep (pending, running, waiting_approval, completed, failed, cancelled) et transitions, notamment le passage par `waiting_approval` (HITL suspension).

**Pourquoi** : Les steps ont plus d'états que les runs. Visualiser `running -> waiting_approval -> running/completed/failed` est critique pour HITL.

---

## 4. PipelineExecutor Execution Flow
**Type**: `flowchart TD`

**Ce qu'il montre** : Flux complet de `ExecuteRun()` : vérification circuit breaker, boucle steps, executeStep(), détection pause/annulation, gestion erreurs, transition finale.

**Pourquoi** : L'algorithme ExecuteRun est le cœur du système. Un flowchart détaillé clarifie les points de décision et les chemins d'erreur.

---

## 5. DAG Computation (Kahn's Algorithm)
**Type**: `flowchart TD`

**Ce qu'il montre** : Étapes de l'algorithme SchedulerService.BuildDAG() : indexation, construction graphe (dépendances explicites + implicites conflit fichiers), tri topologique itérative, détection cycle.

**Pourquoi** : L'algorithme DAG est stateless mais complexe. Un flowchart montre comment les arêtes implicites sont détectées et les couches construites.

---

## 6. EpicRun Orchestration (ParallelGroupExecutor)
**Type**: `sequenceDiagram`

**Ce qu'il montre** : Interactions EpicRunService -> ParallelGroupExecutor -> [pour chaque couche DAG] : runStory() -> LaunchRun() + ExecuteRun() en parallèle via errgroup. Transitions d'état epic run, fail-fast si une story échoue.

**Pourquoi** : Montre la séquence parallèle par couche, l'interaction avec le DAG, et la sémantique fail-fast du epic run.

---

## 7. Metadata Passing Through RunContext
**Type**: `graph LR`

**Ce qu'il montre** : Comment les clés Metadata (`branch_name`, `pr_url`, `model`, `template_content`, `agent_id`, `agent_image`, `error_context`, `log_tail`) circulent entre actions : producteurs et consommateurs.

**Pourquoi** : Les steps communiquent via Metadata dict. Visualiser le flux (ex: git_branch produit branch_name -> agent_run/git_pr/notification la consomment) clarifie les dépendances implicites.

---

## 8. Action Registry & Dispatch Pattern
**Type**: `flowchart TD`

**Ce qu'il montre** : Comment l'ActionRegistry (registre in-memory) est peuplé au démarrage (6 actions : agent_run, git_branch, git_pr, ci_poll, hitl_gate, human, notification, incremental_retry), lookup dans executeStep(), et dispatch à l'implémentation.

**Pourquoi** : Pattern de dispatch par nom + registre est central. Montre les 8 actions concrètes et comment elles sont résolues.

---

## 9. CircuitBreaker Decision Logic
**Type**: `flowchart TD`

**Ce qu'il montre** : Vérification avant exécution run, enregistrement succès/échec, conditions de tripping (seuil échecs consécutifs), reset après succès.

**Pourquoi** : Protection contre cascades d'échecs. Le flowchart montre quand le breaker bloque et comment il se réinitialise.

---

## 10. Retry Flow (Incremental vs Full)
**Type**: `sequenceDiagram` ou `flowchart TD`

**Ce qu'il montre** : RetryStep() détermine le type retry (incremental retries 1-2, full retry 3+). Incremental passe `error_context` + `log_tail` à l'agent. Full relance complet. ParentStepID lie l'historique.

**Pourquoi** : Deux stratégies retry différentes. Un diagram montre la décision (retry count) et les données passées à chaque type.

---

## 11. Complete Single Story Run Flow
**Type**: `sequenceDiagram`

**Ce qu'il montre** : Interaction client REST -> RunService.LaunchRun() -> créer Run + steps, snapshot config, enqueue River job -> River worker -> PipelineExecutor.ExecuteRun() -> [loop steps] executeStep() -> [lookup Action, execute, pub events] -> completion/failure -> Update story status.

**Pourquoi** : Flux complet utilisateur visible. Montre les 4 phases : création, enqueuement, exécution, completion.

---

## 12. Container Lifecycle & Labels Management
**Type**: `flowchart TD` ou `sequenceDiagram`

**Ce qu'il montre** : Création container agent (ContainerOpts avec labels managed_by, run_id, step_id), démarrage, logs streaming NDJSON, parsing cost events, arrêt gracieux, cleanup orphelins. TimeoutEnforcer et OrphanCleaner s'appuient sur les labels.

**Pourquoi** : Les labels sont la clé pour tracking containers. Montre le cycle complet et comment TimeoutEnforcer/OrphanCleaner les utilisent.

---

## 13. Cost Tracking Flow
**Type**: `flowchart TD`

**Ce qu'il montre** : Agent produit NDJSON logs avec Type="cost" (InputTokens, OutputTokens, Model), LogStreamer/ContainerManager parsent, enregistrent dans LogEvent, archivé pour reporting.

**Pourquoi** : Cost tracking complexe avec parsing NDJSON et validation. Un flowchart clarifie comment les logs deviennent des événements de coût.

---

## 14. HITL Gate Suspension & Resume
**Type**: `sequenceDiagram` ou `flowchart TD`

**Ce qu'il montre** : Step exécute HITLGateAction, crée HITLRequest, marque step `waiting_approval`, PipelineExecutor reçoit `errStepSuspended`, retourne proprement. User approuve/rejette via API, step transition `waiting_approval -> running/completed/failed`, run reprend.

**Pourquoi** : Le flow suspension-reprise est critique. Montre comment l'API pauserait l'exécution et la reprend après approbation.

---

## 15. Dependency Injection Initialization Order
**Type**: `flowchart TD`

**Ce qu'il montre** : 14 étapes d'initialisation main.go : Config -> Logger -> DB -> EventBus -> Repositories -> Services -> Actions registry -> PipelineExecutor -> River -> RunService -> Background services -> Epic orchestration -> Router -> Graceful shutdown.

**Pourquoi** : L'ordre d'init est strict et criticalement documenté dans les commentaires main.go. Un diagram montre les dépendances d'ordre.

---

## Notes Complémentaires

- **Haute priorité** : 1, 2, 3, 4, 5, 6 (architecture core + flows centraux)
- **Moyenne priorité** : 7, 8, 9, 11, 14 (patterns opérationnels importants)
- **Détail utile** : 10, 12, 13, 15 (edge cases + operational details)

Tous les diagrammes doivent maintenir la nomenclature française du document (états, services, actions) pour cohérence.
