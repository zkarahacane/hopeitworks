# Suggestions de diagrammes Mermaid — Backend Agent/Container Docs

## Diagrammes recommandés

### 1. **Container Lifecycle State Machine**
- **Type :** `stateDiagram`
- **Contenu :** États du container (inexistant → created → running → stopped → supprimé) avec transitions (Create, Start, Stop, Remove, exit naturel)
- **Utilité :** Montre clairement les transitions d'état et évite les confusions sur l'ordre des opérations

### 2. **AgentRunAction Execution Flow**
- **Type :** `flowchart` (LR ou TD)
- **Contenu :** Pipeline d'exécution : fetch story → fetch project → render prompt → resolve image → create container → start → stream logs → cost tracking → cleanup, avec points de sortie d'erreur
- **Utilité :** Remplace le pseudo-code actuel, plus lisible et traçable pour déboguer les étapes manquantes

### 3. **Dependency Injection Wiring**
- **Type :** `graph` (TB)
- **Contenu :** Montre comment AgentRunAction reçoit ses dépendances (ContainerManager, LogStreamer, EventPublisher, services, repos) et comment elles s'interconnectent
- **Utilité :** Aide à comprendre la structure hexagonale et les interfaces mockables pour les tests

### 4. **Log Streaming Architecture (Goroutines & Channels)**
- **Type :** `flowchart`
- **Contenu :** Les 3 goroutines (stdcopy demux, scanner.Scan, streamLoop) avec leurs channels (pw, lineCh, logCh, doneCh, ctx.Done, idleTimer)
- **Utilité :** Visualise la concurrence complexe et les points de synchronisation dans LogStreamer

### 5. **NDJSON Parsing Decision Tree**
- **Type :** `flowchart` (TD)
- **Contenu :** Logique de parsing (ligne vide → skip, JSON invalide → plain text, JSON valide → extraction fields, type=result → normalize cost event)
- **Utilité :** Documente les chemins de parsing et les transformations de cost events de manière graphique

### 6. **Container Environment Variables & Metadata Flow**
- **Type :** `graph` ou `sequenceDiagram`
- **Contenu :** Source → build env vars map → ContainerOpts.Env → container startup → entrypoint.sh validation
- **Utilité :** Trace chaque variable d'où elle provient (runCtx, metadata, os.Getenv) jusqu'au container

### 7. **Docker API Call Sequence (Per Operation)**
- **Type :** `sequenceDiagram`
- **Contenu :** ContainerManager.Create() → Docker API calls (ContainerCreate, ContainerStart, etc.) avec erreur handling
- **Utilité :** Montre l'interaction exacte backend ↔ Docker API pour chaque opération

### 8. **Cost Event Extraction & Accumulation**
- **Type :** `flowchart` (TD)
- **Contenu :** Log stream → cost event parsing → accumulation en slice → CostService.RecordStepCost → DB, avec les deux formats supportés (custom + result event)
- **Utilité :** Clarifie comment les costs sont extraits des logs et persisted

### 9. **Container Cleanup Error Recovery**
- **Type :** `flowchart` (TD)
- **Contenu :** Cleanup sequence (Stop avec SIGTERM/timeout/SIGKILL → Remove) avec chemins d'erreur, timeouts indépendants (30s), et logging en warn
- **Utilité :** Montre que le cleanup est "best effort" et explique la stratégie de timeout

### 10. **Handler → Service → Action → Adapter Layering**
- **Type :** `graph` (TB) ou `classDiagram`
- **Contenu :** Montre les couches (handler → AgentService/AgentRunAction → port/adapter) et comment les flux CRUD vs execution se splitent
- **Utilité :** Visualise la séparation des responsabilités et évite de mélanger CRUD API handlers avec l'action d'exécution

---

## Notes d'implémentation

- Les diagrammes **flowchart** et **stateDiagram** sont les plus utiles pour capturer les séquences et transitions complexes de ce domaine
- Ajouter les diagrammes dans une section "## Architecture Diagrams" après la section "Vue d'ensemble", avant "Modèles de domaine"
- Les **sequenceDiagram** sont pertinents pour les interactions multi-composants (Docker API, handler → services)
- Les **graph/classDiagram** aident à montrer l'architecture en couches et la DI sans encombrer les séquences
