# Mermaid Diagram Suggestions — backend-stories-epics-config.md

## Résumé
Ce document propose 8 diagrammes Mermaid qui enrichissent la documentation en visualisant les flux, modèles et interactions décrites en sections 1-9.

---

## 1. Hexagonal Architecture Overview
**Type:** `graph TD` (architecture layering)

**Montre:** Les 4 couches hexagonales (handler → service → port ← adapter) appliquées à chaque domaine (Stories, Epics, Pipeline Config, Cost Tracking, HITL, Notifications). Clarifierait les dépendances entre couches et l'injection des adapters Postgres.

**Utilité:** Nouveau lecteur comprend immédiatement pourquoi le code est structuré ainsi et où chercher pour implémenter une feature.

---

## 2. Story Lifecycle & Status Transitions
**Type:** `stateDiagram-v2`

**Montre:** Les états possibles d'une Story (`backlog` → `running` → `done` ou `failed`) et les conditions/événements qui déclenchent chaque transition. Inclure les rôles (developer, reviewer) et les hooks (Run launcher, DAG scheduler).

**Utilité:** Visualise la sémantique complète du statut Story, évite les ambiguïtés sur les transitions valides vs invalides.

---

## 3. Story DAG Construction & Execution Flow
**Type:** `flowchart LR` (data flow)

**Montre:** Comment le `DependsOn` field (liste de clés strings) est utilisé par le DAG scheduler pour construire le graphe de dépendances d'un epic, puis comment les stories sont ordonnancées en parallèle/séquence. Inclure: Epic.Stories → DAG graph → scheduler queue → agent execution.

**Utilité:** Explique le chaînage des stories dans une epic sans lister les algos — utile pour product & dev.

---

## 4. Epic Lifecycle with DAG State
**Type:** `sequenceDiagram` (actors = Frontend, API, Service, DAGScheduler, Postgres)

**Montre:** Flux complet d'une epic : création → ajout de stories → lancement du DAG via `/launch` → polling/polling SSE du statut du epic_run → completion. Inclure les appels service+repo par acteur.

**Utilité:** Débutant visualise les interactions entre frontend, service, DAG et la DB sans traverser 200 lignes de code.

---

## 5. Pipeline Config Hierarchy
**Type:** `graph TD` (tree structure)

**Montre:** Structure d'imbrication : Project → PipelineConfig → Groups → Steps. Chaque niveau avec ses responsabilités (validation, mutation, DI). Inclure l'arborescence JSON et les contraintes (max steps, agent_id required, etc.).

**Utilité:** Clarifie les limites de ce qu'on peut configurer à chaque niveau et simplifie les merges/overrides de configs.

---

## 6. Cost Tracking Event Flow
**Type:** `flowchart TD` (data pipeline)

**Montre:** LogEvent stream (NDJSON) → LogStreamer parser → CostEvent extraction (si type="cost") → CostService.RecordStepCost → CostRepository.InsertCostRecord → Postgres aggregation. Inclure les transformations de type et les erreurs (malformed JSON, invalid fields).

**Utilité:** Explique comment les costs du container arrivent dans la DB sans ambiguïté sur les pertes potentielles de données.

---

## 7. HITL Gate Approval Workflow
**Type:** `sequenceDiagram` (actors = Executor, HITL Service, Reviewer, Postgres, EventPublisher)

**Montre:** Moment exact où executor crée la gate → passe à "waiting_approval" → reviewer approuve/rejette → service update du step status → event published → executor reprend. Clarifier l'invariant: step ne progresse que si gate approuvée.

**Utilité:** Élimine la confusion sur qui bloque le pipeline et comment un reviewer reprend manuellement l'exécution.

---

## 8. Notification Dispatch Pipeline
**Type:** `flowchart LR` (streaming + filtering)

**Montre:** EventPublisher.Publish → Postgres NOTIFY → NotificationDispatcher.Start → subscribe par project → merge channels → dispatch (filter EventsFilter) → router par channel_type → notifier.Send (Discord/Webhook). Inclure silencing des erreurs.

**Utilité:** Débutant comprend comment une story peut déclencher N notifications et pourquoi une erreur Discord n'arrête pas Webhook.

---

## 9. Cross-Domain Event Flow (Story → Run → Cost/HITL → Notification)
**Type:** `graph TD` (cross-cutting concern)

**Montre:** Chemin complet : Story.Launch → RunService.CreateRun → PipelineExecutor.Execute → step types branching (agent_run branch: cost stream; hitl_gate branch: approval). Puis Event.Publish → Notifications. C'est la "happy path" + "sad path" (failure).

**Utilité:** Visualise l'interconnexion de tous les domaines en une seule image — montre pourquoi chaque domaine existe.

---

## 10. API Handler → Service → Repository Call Chain
**Type:** `sequenceDiagram` (actors = Client, Handler, Service, Port, PostgresAdapter)

**Montre:** Une requête HTTP simple (ex: POST /stories) : valider JSON → call service.Create → service appelle repo.Create → adapter.Create → Postgres → return. Inclure error handling et HTTP codes (200/201/400/409/500).

**Utilité:** Clarifier la sémantique de validation (handler vs service) et où chaque erreur est attrapée.

---

## Priorité d'implémentation

**Haute:** 1, 4, 9 (architecture + workflows core)
**Moyenne:** 2, 3, 6, 7, 8 (domaines métier)
**Basse:** 5, 10 (détails d'implémentation)
