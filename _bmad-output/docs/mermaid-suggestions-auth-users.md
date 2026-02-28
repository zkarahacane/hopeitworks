# Suggestions de diagrammes Mermaid — Auth, Users, Projects

## Vue d'ensemble

Ce document propose des diagrammes Mermaid pertinents pour enrichir la documentation backend `backend-auth-users-projects.md`. Chaque suggestion inclut le type Mermaid, ce qu'il illustre et son utilité.

---

## Suggestions

### 1. Architecture hexagonale — Auth/Users/Projects (Graphique)

**Type** : `graph TB`

**Ce qu'il montrerait** : Les 4 couches (Handler → Service → Port → Adapter) pour **un** domaine (ex. Auth). Boîtes chi, sqlc, SMTP en bas. Montre l'isola­tion entre couches.

**Utilité** : Visualise rapidement le pattern hexagonal appliqué concrètement. Aide les nouveaux contributeurs à comprendre où implémenter une nouvelle fonctionnalité.

---

### 2. Flux Login complet (Diagramme de séquence)

**Type** : `sequenceDiagram`

**Ce qu'il montrerait** : Acteurs (Client HTTP, AuthHandler, AuthService, UserRepository, PostgreSQL), messages (POST /login → bcrypt verify → JWT generate → cookie set). 5-6 flèches.

**Utilité** : Remplace le flux textuel 8.1 par une timeline visuelle. Montre l'ordre des appels, les erreurs possibles (401, 400), et où intervient chaque couche.

---

### 3. Flux Forgot / Reset Password (Diagramme de séquence)

**Type** : `sequenceDiagram`

**Ce qu'il montrerait** : 2 séquences parallèles côte à côte : ForgotPassword (email inconnu → anti-énumération) vs ResetPassword (token valide → update password). Inclut EmailSender et TokenRepository.

**Utilité** : Clarifie l'asymétrie entre les deux endpoints (202 vs 200/400) et l'anti-énumération. Montre où EmailSender intervient.

---

### 4. Modèle User et ses rôles (Diagramme de classe)

**Type** : `classDiagram`

**Ce qu'il montrerait** : Classe User avec champs (ID, Email, PasswordHash, Name, Role, CreatedAt, DeletedAt). Énumération Role (admin, user). ProjectUser avec champs et ProjectRole (owner, member). Relation de liaison.

**Utilité** : Dénormalise les structs Go éparpillées. Montre les énumérations et les nullable (DeletedAt, UsedAt). Utile pour les développeurs frontend qui interrogent l'API.

---

### 5. Statut du token reset password (Diagramme d'état)

**Type** : `stateDiagram-v2`

**Ce qu'il montrerait** : 4 états (Created → Expired | Used | Valid). Transitions : temps → Expired, API reset → Used, requête valide → Valid. Chaque état a des conditions (IsExpired(), IsUsed()).

**Utilité** : Visualise le cycle de vie d'un token de réinitialisation. Clarifie pourquoi `ErrResetTokenInvalid` s'applique aux cas "absent" et "already used".

---

### 6. Circuit Breaker — Transition d'états (Diagramme d'état)

**Type** : `stateDiagram-v2`

**Ce qu'il montrerait** : 2 états (ACTIVE, BROKEN). Condition : `Count >= Max` → BROKEN. Action : `Reset()` ou timeout → ACTIVE. Incrémentation du counter sur chaque échec.

**Utilité** : Montre où les champs `CircuitBreakerCount`, `CircuitBreakerMax`, `CircuitBreakerActive` interviennent. Utile pour comprendre la stratégie de résilience.

---

### 7. Accès aux projets — Matrice Admin vs Non-Admin (Tableau Mermaid)

**Type** : `graph LR` ou `table` (via markdown)

**Ce qu'il montrerait** : 2 colonnes (Admin, Non-Admin). Lignes pour chaque endpoint (GET /projects, POST, PUT, DELETE). Cases remplies avec ✓ ou détail du contrôle (ex. "isUserInProject").

**Utilité** : Résume section 6.3 (ProjectHandler) en une vue d'ensemble. Clarifie rapidement quels endpoints chaque rôle peut appeler.

---

### 8. Dépendances entre services (Graphique orienté)

**Type** : `graph TB`

**Ce qu'il montrerait** : 4 services (AuthService, UserService, ProjectService, ProjectUserService) comme boîtes. Flèches pour les dépendances : ProjectUserService → ProjectService, ProjectService → PipelineConfigService (injection optionnelle). Imports de ports.

**Utilité** : Montre les couplages et les injections optionnelles. Utile pour refactoring ou ajout de nouvelles fonctionnalités.

---

### 9. Validation du token JWT — Arbre des contrôles (Organigramme)

**Type** : `flowchart TD`

**Ce qu'il montrerait** : Cookie absent? → 401. Parse token → 401 si invalide. Signature HMAC valide? → 401 sinon. Expiré? → 401. Dans blacklist? → 401. OK → UserID + Role en contexte. 5-6 points de décision.

**Utilité** : Remplace section 7.1 (Auth middleware) par un organigramme clair. Montre l'ordre des vérifications et où chaque erreur est levée.

---

### 10. Cycle de vie des données utilisateur (Graphique)

**Type** : `graph LR`

**Ce qu'il montrerait** : Création (Register) → Active + Token JWT → Logout (token blacklisté) → DeletedAt nullable. Alternatif : ForgotPassword → Token de reset → ResetPassword. Montre les transitions de statut.

**Utilité** : Synthétise les sections 8.1-8.3 (flux complets) en une vue globale du cycle de vie. Clarifie les opérations non-réversibles (soft-delete) vs réversibles (logout = blacklist).

---

## Recommandations d'intégration

- **Flux complets** (Login, ForgotPassword, Reset) : placer après section 8 (Flux complets).
- **Diagrammes de classe** : placer après section 2 (Modèles de domaine).
- **Diagramme d'état circuit breaker** : placer après section 2.2 (Project model).
- **Matrice d'accès** : placer après section 6 (API Handlers).
- **Dépendances services** : placer après section 4 (Services).
- **Arbre validation token** : placer après section 7.1 (Auth middleware).

---

**Total suggéré** : 10 diagrammes, mix de 2-3 `sequenceDiagram`, 2-3 `stateDiagram-v2`, 3-4 `graph TB/LR/TD`, 1 `classDiagram`.
