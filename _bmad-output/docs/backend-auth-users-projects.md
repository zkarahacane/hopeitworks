# Documentation Backend — Auth, Users, Projects, Project Users

> Généré le 2026-02-26. Source : code dans `backend/`.

---

## Table des matières

1. [Vue d'ensemble](#1-vue-densemble)
2. [Modèles de domaine](#2-modèles-de-domaine)
   - [User](#21-user)
   - [Project](#22-project)
   - [ProjectUser / ProjectMember](#23-projectuser--projectmember)
   - [PasswordResetToken](#24-passwordresettoken)
3. [Ports (interfaces)](#3-ports-interfaces)
   - [UserRepository](#31-userrepository)
   - [ProjectRepository](#32-projectrepository)
   - [ProjectUserRepository](#33-projectuserrepository)
   - [PasswordResetTokenRepository](#34-passwordresettokenrepository)
   - [TokenBlacklistRepository](#35-tokenblacklistrepository)
   - [EmailSender](#36-emailsender)
4. [Services](#4-services)
   - [AuthService](#41-authservice)
   - [UserService](#42-userservice)
   - [ProjectService](#43-projectservice)
   - [ProjectUserService](#44-projectuserservice)
5. [Adapters](#5-adapters)
   - [postgres/UserRepository](#51-postgresuserrepository)
   - [postgres/ProjectRepo](#52-postgresprojectrepo)
   - [postgres/ProjectUserRepo](#53-postgresprojectuserrepo)
   - [postgres/PasswordResetTokenRepository](#54-postgrespasswordresettokenrepository)
   - [postgres/TokenBlacklistRepo](#55-postgrestokenblacklistrepo)
   - [smtp/EmailSender](#56-smtpemailsender)
6. [API Handlers](#6-api-handlers)
   - [AuthHandler](#61-authhandler)
   - [UserHandler](#62-userhandler)
   - [ProjectHandler](#63-projecthandler)
   - [ProjectUserHandler](#64-projectuserhandler)
   - [ProfileHandler](#65-profilehandler)
7. [Middleware](#7-middleware)
   - [Auth middleware](#71-auth-middleware)
   - [RBAC middleware](#72-rbac-middleware)
8. [Flux complets](#8-flux-complets)
   - [Login](#81-login)
   - [Forgot / Reset Password](#82-forgot--reset-password)
   - [Ajout d'un membre à un projet](#83-ajout-dun-membre-à-un-projet)
9. [Tests](#9-tests)

---

## 1. Vue d'ensemble

Les domaines Auth, Users, Projects et Project Users constituent la fondation du backend hopeitworks. Ils suivent l'architecture hexagonale strictement :

```
Handler (HTTP) → Service (logique métier) → Port (interface) ← Adapter (implémentation Postgres/SMTP)
```

**Fichiers clés** :

| Couche | Chemin |
|--------|--------|
| Modèles | `backend/internal/domain/model/` |
| Ports | `backend/internal/domain/port/` |
| Services | `backend/internal/domain/service/` |
| Adapters Postgres | `backend/internal/adapter/postgres/` |
| Adapter SMTP | `backend/internal/adapter/smtp/` |
| Handlers HTTP | `backend/internal/api/handler/` |
| Middleware | `backend/internal/api/middleware/` |

---

## 2. Modèles de domaine

### 2.1 User

**Fichier** : `backend/internal/domain/model/user.go`

```go
type Role string

const (
    RoleAdmin Role = "admin"
    RoleUser  Role = "user"
)

type User struct {
    ID           uuid.UUID
    Email        string
    PasswordHash string
    Name         string
    Role         Role
    CreatedAt    time.Time
    UpdatedAt    time.Time
    DeletedAt    *time.Time  // nil = actif, non-nil = soft-deleted
}
```

**Rôles globaux** : `admin` ou `user`. Validé via `Role.IsValid()`.

- `admin` : accès complet à la gestion des utilisateurs, projets, configuration
- `user` : accès restreint aux projets dont il est membre

Le mot de passe n'est jamais stocké en clair. Seul le hash bcrypt est persisté dans `PasswordHash`.

---

### 2.2 Project

**Fichier** : `backend/internal/domain/model/project.go`

```go
type Project struct {
    ID                   uuid.UUID
    Name                 string
    Description          *string
    OwnerID              *uuid.UUID
    RepoURL              *string
    GitProvider          string       // "github" | "gitea" (défaut: "github")
    GitTokenEnv          *string      // nom de la variable d'env contenant le token Git
    AgentRuntime         string       // "docker" (défaut: "docker")
    DefaultModel         *string      // modèle Claude par défaut pour les agents
    MaxBudget            *float64     // budget max USD (nullable)
    MaxContainerTimeout  *time.Duration
    CircuitBreakerCount  int
    CircuitBreakerActive bool
    CircuitBreakerMax    int
    CreatedAt            time.Time
    UpdatedAt            time.Time
}
```

**Points notables** :
- `GitProvider` et `AgentRuntime` ont des valeurs par défaut (`"github"` et `"docker"`)
- `GitTokenEnv` est le **nom** de la variable d'environnement (pas la valeur du token), pour éviter de persister des secrets
- Le circuit breaker est intégré directement dans le modèle Project : il compte les failures consécutives et se déclenche quand `CircuitBreakerCount >= CircuitBreakerMax`

---

### 2.3 ProjectUser / ProjectMember

**Fichier** : `backend/internal/domain/model/project_user.go`

```go
type ProjectRole string

const (
    ProjectRoleOwner  ProjectRole = "owner"
    ProjectRoleMember ProjectRole = "member"
)

// Table de liaison user <-> project
type ProjectUser struct {
    ProjectID uuid.UUID
    UserID    uuid.UUID
    Role      ProjectRole
    CreatedAt time.Time
}

// Vue dénormalisée retournée par ListProjectUsers (JOIN users)
type ProjectMember struct {
    UserID      uuid.UUID
    Email       string
    Name        string
    UserRole    Role        // rôle global (admin/user)
    ProjectRole ProjectRole // rôle dans le projet (owner/member)
    AssignedAt  time.Time
}
```

`ProjectMember` est une vue dénormalisée utilisée pour l'affichage. Elle n'est pas persistée — elle résulte d'un JOIN SQL entre `project_users` et `users`.

---

### 2.4 PasswordResetToken

**Fichier** : `backend/internal/domain/model/password_reset_token.go`

```go
type PasswordResetToken struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    Token     string      // token URL-safe base64 (32 bytes aléatoires)
    ExpiresAt time.Time   // +1h à la création
    UsedAt    *time.Time  // nil = non utilisé
    CreatedAt time.Time
}

func (t *PasswordResetToken) IsExpired() bool { return time.Now().After(t.ExpiresAt) }
func (t *PasswordResetToken) IsUsed() bool    { return t.UsedAt != nil }
```

Token à usage unique, expire après 1 heure. Les méthodes `IsExpired()` et `IsUsed()` encapsulent la logique de validation dans le modèle.

---

## 3. Ports (interfaces)

### 3.1 UserRepository

**Fichier** : `backend/internal/domain/port/user_repository.go`

```go
type UserRepository interface {
    Create(ctx context.Context, user *model.User) (*model.User, error)
    GetByEmail(ctx context.Context, email string) (*model.User, error)
    GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
    List(ctx context.Context, limit, offset int32) ([]*model.User, error)
    Count(ctx context.Context) (int64, error)
    Update(ctx context.Context, user *model.User) (*model.User, error)
    UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error
    Delete(ctx context.Context, id uuid.UUID) error
}
```

`UpdatePasswordHash` est séparé d'`Update` pour éviter de passer l'objet entier lors d'un changement de mot de passe — seul le hash est mis à jour.

---

### 3.2 ProjectRepository

**Fichier** : `backend/internal/domain/port/project_repository.go`

```go
type ProjectRepository interface {
    Create(ctx context.Context, project *model.Project) (*model.Project, error)
    GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error)
    List(ctx context.Context, limit, offset int32) ([]*model.Project, error)
    Count(ctx context.Context) (int64, error)
    Update(ctx context.Context, project *model.Project) (*model.Project, error)
    Delete(ctx context.Context, id uuid.UUID) error

    // Gestion du circuit breaker
    IncrementCircuitBreakerCount(ctx context.Context, id uuid.UUID) (*model.Project, error)
    ResetCircuitBreaker(ctx context.Context, id uuid.UUID) (*model.Project, error)
}
```

Les deux méthodes de circuit breaker sont dans ce port car elles modifient l'état du `Project` directement en base.

---

### 3.3 ProjectUserRepository

**Fichier** : `backend/internal/domain/port/project_user_repository.go`

```go
type ProjectUserRepository interface {
    AddUser(ctx context.Context, projectID, userID uuid.UUID, role model.ProjectRole) (*model.ProjectUser, error)
    RemoveUser(ctx context.Context, projectID, userID uuid.UUID) error
    ListMembers(ctx context.Context, projectID uuid.UUID) ([]*model.ProjectMember, error)
    IsUserInProject(ctx context.Context, projectID, userID uuid.UUID) (bool, error)
    ListProjectsByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*model.Project, error)
    CountProjectsByUser(ctx context.Context, userID uuid.UUID) (int64, error)
}
```

`ListProjectsByUser` et `CountProjectsByUser` permettent la vue inverse : donner les projets accessibles à un utilisateur non-admin.

---

### 3.4 PasswordResetTokenRepository

**Fichier** : `backend/internal/domain/port/password_reset_token_repository.go`

```go
type PasswordResetTokenRepository interface {
    Create(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) (*model.PasswordResetToken, error)
    GetByToken(ctx context.Context, token string) (*model.PasswordResetToken, error)
    MarkUsed(ctx context.Context, id uuid.UUID) error
}
```

Interface minimaliste : on ne peut que créer, récupérer par token et marquer comme utilisé. Pas de suppression — les tokens restent en base pour audit.

---

### 3.5 TokenBlacklistRepository

**Fichier** : `backend/internal/domain/port/token_blacklist_repository.go`

```go
type TokenBlacklistRepository interface {
    Revoke(ctx context.Context, jti string, expiresAt time.Time) error
    IsRevoked(ctx context.Context, jti string) (bool, error)
    DeleteExpired(ctx context.Context) error
}
```

Gère la liste noire des JTI (JWT ID) révoqués. `DeleteExpired` est appelé périodiquement via `AuthService.PurgeExpiredTokens` pour éviter la croissance infinie de la table.

---

### 3.6 EmailSender

**Fichier** : `backend/internal/domain/port/email_sender.go`

```go
type EmailMessage struct {
    To       string
    Subject  string
    HTMLBody string
}

type EmailSender interface {
    Send(ctx context.Context, msg EmailMessage) error
}
```

Interface à une seule méthode. Seuls les emails HTML sont supportés (reset password). L'implémentation courante est SMTP via `net/smtp`.

---

## 4. Services

### 4.1 AuthService

**Fichier** : `backend/internal/domain/service/auth_service.go`

**But** : authentification, émission et validation des JWT, gestion des tokens de réinitialisation de mot de passe, révocation de sessions.

#### Structure

```go
type AuthService struct {
    repo          port.UserRepository
    blacklistRepo port.TokenBlacklistRepository  // optionnel, injecté via SetBlacklistRepo
    tokenRepo     port.PasswordResetTokenRepository
    emailSender   port.EmailSender
    frontendURL   string
    jwtSecret     []byte
    jwtExpiration time.Duration
}
```

#### Sentinelles d'erreur

```go
var (
    ErrInvalidCredentials = errors.New("invalid credentials")
    ErrEmailAlreadyExists = errors.New("email already exists")
    ErrValidation         = errors.New("validation error")
    ErrTokenRevoked       = errors.New("token has been revoked")
    ErrResetTokenExpired  = errors.New("reset token expired")
    ErrResetTokenInvalid  = errors.New("reset token invalid or already used")
)
```

#### Claims JWT

```go
type Claims struct {
    UserID uuid.UUID  `json:"user_id"`
    Role   model.Role `json:"role"`
    jwt.RegisteredClaims  // inclut ID (JTI), ExpiresAt, IssuedAt
}
```

Chaque token a un JTI (`uuid.New().String()`) qui permet la révocation individuelle.

#### Méthodes clés

| Méthode | Comportement |
|---------|-------------|
| `Register(ctx, email, password, name)` | Valide les inputs (non vide, password >= 8 chars), hash bcrypt, crée le user avec `RoleUser`, génère un JWT. Retourne `ErrEmailAlreadyExists` sur violation unique. |
| `Login(ctx, email, password)` | Récupère le user par email, compare le hash bcrypt. Toujours `ErrInvalidCredentials` en cas d'échec (pas d'énumération). |
| `ValidateToken(tokenString)` | Parse le JWT avec HMAC-SHA256, vérifie la signature et l'expiration. Retourne les `*Claims`. |
| `Logout(ctx, tokenString)` | Valide le token, extrait le JTI, l'ajoute au blacklist. Si le token est invalide ou sans JTI : no-op (intentionnel). |
| `ForgotPassword(ctx, email)` | Génère un token aléatoire 32 bytes (base64 URL-safe), expire dans 1h, envoie l'email. **Retourne toujours nil** même si l'email n'existe pas (anti-énumération). |
| `ResetPassword(ctx, token, newPassword)` | Récupère le token, vérifie non-utilisé et non-expiré, hash le nouveau mot de passe, met à jour le user, marque le token comme utilisé. |
| `PurgeExpiredTokens(ctx)` | Délègue à `blacklistRepo.DeleteExpired()`. |

#### Détail du flux `isDuplicateKeyError`

Pour éviter d'importer `pgx/pgconn` dans la couche domaine, le service utilise une interface locale :

```go
type sqlStateError interface {
    SQLState() string
}
```

Si l'erreur implémente cette interface et retourne `"23505"` (unique violation PostgreSQL), c'est un doublon d'email.

---

### 4.2 UserService

**Fichier** : `backend/internal/domain/service/user_service.go`

**But** : CRUD admin sur les utilisateurs, mise à jour de profil en self-service, changement de mot de passe.

#### Structure

```go
type UserService struct {
    repo port.UserRepository
}
```

#### Types de paramètres

```go
// Admin CRUD
type UpdateUserParams struct {
    ID    uuid.UUID
    Name  *string
    Email *string
    Role  *model.Role  // uniquement via admin
}

// Self-service (rôle exclu intentionnellement)
type UpdateProfileParams struct {
    ID    uuid.UUID
    Name  *string
    Email *string
}

type UserListResult struct {
    Users []*model.User
    Total int64
}
```

#### Méthodes clés

| Méthode | Comportement |
|---------|-------------|
| `GetByID(ctx, id)` | Retourne `DomainError{NotFound, "USER_NOT_FOUND"}` si absent. |
| `List(ctx, page, perPage)` | Pagination normalisée via `paginationToLimitOffset`. Retourne users + total. |
| `Update(ctx, params)` | Mise à jour admin : valide name (non vide, max 255 chars), email (non vide), role (`IsValid()`). Patch partiel via pointeurs. |
| `UpdateProfile(ctx, params)` | Mise à jour self-service : même validations sur name/email, mais **sans le champ role**. En cas de doublon d'email : `DomainError{Conflict, "EMAIL_ALREADY_EXISTS"}`. |
| `ChangePassword(ctx, userID, currentPassword, newPassword)` | Vérifie le hash bcrypt actuel, valide new >= 8 chars, génère nouveau hash, appelle `repo.UpdatePasswordHash`. |
| `Delete(ctx, id)` | Vérifie existence, délègue à `repo.Delete` (soft-delete en base). |

**Erreur sentinelle** :
```go
var ErrInvalidCurrentPassword = errors.NewUnauthorized("current password is incorrect")
```

---

### 4.3 ProjectService

**Fichier** : `backend/internal/domain/service/project_service.go`

**But** : CRUD des projets, seed automatique du pipeline config par défaut à la création.

#### Structure

```go
type ProjectService struct {
    repo                  port.ProjectRepository
    pipelineConfigService *PipelineConfigService  // optionnel, injecté via SetPipelineConfigService
}
```

#### Types de paramètres

```go
type CreateProjectParams struct {
    Name         string
    Description  *string
    OwnerID      *uuid.UUID
    RepoURL      *string
    GitProvider  *string  // défaut "github" si nil ou vide
    GitTokenEnv  *string
    AgentRuntime *string  // défaut "docker" si nil ou vide
    DefaultModel *string
}

type UpdateProjectParams struct {
    ID           uuid.UUID
    Name         *string
    Description  *string
    MaxBudget    *float64
    SetBudget    bool  // true pour mettre à nil explicitement
    RepoURL      *string
    SetRepoURL   bool
    GitProvider  *string
    GitTokenEnv  *string
    SetTokenEnv  bool
    AgentRuntime *string
    DefaultModel *string
    SetModel     bool
}

type ListResult struct {
    Projects []*model.Project
    Total    int64
}
```

**Pattern `SetXxx bool`** : Les champs nullable (`MaxBudget`, `RepoURL`, `GitTokenEnv`, `DefaultModel`) utilisent un booléen compagnon pour distinguer "non fourni" de "mis à nil explicitement". Cela évite d'effacer des valeurs par accident.

#### Méthodes clés

| Méthode | Comportement |
|---------|-------------|
| `Create(ctx, params)` | Valide name (obligatoire, max 255), description (max 1000). Defaults : `GitProvider="github"`, `AgentRuntime="docker"`. Après création, appelle `pipelineConfigService.SeedDefault` si le service est injecté. |
| `GetByID(ctx, id)` | Délègue au repo (erreur NotFound propagée depuis l'adapter). |
| `List(ctx, page, perPage)` | Pagination + count total. |
| `Update(ctx, params)` | Read-modify-write : fetch l'existant, applique les champs fournis avec validations, sauvegarde. |
| `Delete(ctx, id)` | Vérifie existence avant suppression (hard delete en base). |

---

### 4.4 ProjectUserService

**Fichier** : `backend/internal/domain/service/project_user_service.go`

**But** : gestion des membres de projet (ajout, suppression, listing), accès croisé user<->projet.

#### Structure

```go
type ProjectUserService struct {
    repo        port.ProjectUserRepository
    projectRepo port.ProjectRepository
    userRepo    port.UserRepository
}
```

#### Méthodes clés

| Méthode | Comportement |
|---------|-------------|
| `AddUser(ctx, projectID, userID, role)` | Valide `role.IsValid()`, vérifie existence du projet et de l'user, délègue au repo. Retourne `DomainError{Validation}` si rôle invalide. |
| `RemoveUser(ctx, projectID, userID)` | Vérifie que l'user est bien dans le projet via `IsUserInProject`, sinon `DomainError{NotFound}`. |
| `ListMembers(ctx, projectID)` | Vérifie existence du projet, retourne `[]*model.ProjectMember` (vue dénormalisée). |
| `IsUserInProject(ctx, projectID, userID)` | Délégation directe au repo. Utilisé par les handlers pour le contrôle d'accès. |
| `ListProjectsForUser(ctx, userID, page, perPage)` | Retourne les projets assignés à un user avec pagination. Utilisé par `ProjectHandler.ListProjects` pour les non-admins. |

---

## 5. Adapters

### 5.1 postgres/UserRepository

**Fichier** : `backend/internal/adapter/postgres/user_repository.go`

Implémente `port.UserRepository` via les queries sqlc générées.

```go
type UserRepository struct {
    q *Queries  // queries sqlc
}
var _ port.UserRepository = (*UserRepository)(nil)

func NewUserRepository(db DBTX) *UserRepository
```

**Mapping** via `toDomainUser(User) *model.User` : convertit la struct sqlc en modèle domaine en gérant le champ nullable `DeletedAt` (`pgtype.Timestamptz` → `*time.Time`).

**Méthode Update** : utilise `pgtype.Text` pour les champs optionnels — un champ est mis à jour uniquement si sa valeur `Valid=true`.

---

### 5.2 postgres/ProjectRepo

**Fichier** : `backend/internal/adapter/postgres/project_repo.go`

Implémente `port.ProjectRepository`.

```go
type ProjectRepo struct {
    queries *Queries
}
var _ port.ProjectRepository = (*ProjectRepo)(nil)

func NewProjectRepo(queries *Queries) *ProjectRepo
```

**Gestion des erreurs** : toutes les erreurs Postgres sont wrappées en `DomainError` :
- `pgx.ErrNoRows` → `NewNotFound`
- `SQLSTATE 23505` → `NewConflict`
- Autres → `NewInternal`

**Helpers de conversion** (dans ce fichier, réutilisés dans tout le package) :

```go
// string pointer → pgtype.Text (null si nil)
func textFromStringPtr(s *string) pgtype.Text

// uuid pointer → pgtype.UUID (null si nil)
func uuidFromPtr(u *uuid.UUID) pgtype.UUID

// float64 pointer → pgtype.Numeric (stocké en centimes avec exp=-2)
func numericFromFloat64Ptr(f *float64) pgtype.Numeric

// pgtype.Numeric → float64 (inverse)
func numericToFloat64(n pgtype.Numeric) float64

// vérifie SQLSTATE 23505
func isUniqueViolation(err error) bool
```

**Mapping `toDomainProject`** : convertit tous les champs nullable (`pgtype.Text`, `pgtype.UUID`, `pgtype.Numeric`) en pointeurs Go.

**Circuit breaker** : `IncrementCircuitBreakerCount` et `ResetCircuitBreaker` appellent des queries SQL dédiées qui modifient atomiquement les champs `circuit_breaker_count` et `circuit_breaker_active`.

---

### 5.3 postgres/ProjectUserRepo

**Fichier** : `backend/internal/adapter/postgres/project_user_repo.go`

Implémente `port.ProjectUserRepository`.

```go
type ProjectUserRepo struct {
    queries *Queries
}
var _ port.ProjectUserRepository = (*ProjectUserRepo)(nil)

func NewProjectUserRepo(queries *Queries) *ProjectUserRepo
```

**Gestion des erreurs dans `AddUser`** :
- `SQLSTATE 23505` → `NewConflict("project_user", "user already assigned")`
- `SQLSTATE 23503` → `NewNotFound("project or user", ...)` (FK violation)

**`isForeignKeyViolation`** : helper local qui vérifie `SQLSTATE == "23503"`.

**`ListMembers`** : la query SQL fait un JOIN entre `project_users` et `users` pour remplir `ProjectMember` directement.

**`ListProjectsByUser`** : réutilise `toDomainProject` défini dans `project_repo.go`.

---

### 5.4 postgres/PasswordResetTokenRepository

**Fichier** : `backend/internal/adapter/postgres/password_reset_token_repository.go`

Implémente `port.PasswordResetTokenRepository`.

```go
type PasswordResetTokenRepository struct {
    q *Queries
}
var _ port.PasswordResetTokenRepository = (*PasswordResetTokenRepository)(nil)

func NewPasswordResetTokenRepository(db DBTX) *PasswordResetTokenRepository
```

**`GetByToken`** : retourne `NewNotFound` si `pgx.ErrNoRows`.

**`toDomainPasswordResetToken`** : gère le `UsedAt` nullable (`pgtype.Timestamptz` → `*time.Time`).

---

### 5.5 postgres/TokenBlacklistRepo

**Fichier** : `backend/internal/adapter/postgres/token_blacklist_repo.go`

Implémente `port.TokenBlacklistRepository`.

```go
type TokenBlacklistRepo struct {
    q *Queries
}
var _ port.TokenBlacklistRepository = (*TokenBlacklistRepo)(nil)

func NewTokenBlacklistRepo(db DBTX) *TokenBlacklistRepo
```

| Méthode | Query sqlc |
|---------|-----------|
| `Revoke` | `InsertRevokedToken` — insère `(jti, expires_at)` |
| `IsRevoked` | `IsTokenRevoked` — retourne bool |
| `DeleteExpired` | `DeleteExpiredRevokedTokens` — purge les entrées expirées |

---

### 5.6 smtp/EmailSender

**Fichier** : `backend/internal/adapter/smtp/email_sender.go`

Implémente `port.EmailSender` via `net/smtp` de la stdlib Go.

```go
type EmailSender struct {
    cfg pkgconfig.SMTPConfig  // Host, Port, From, Username, Password
}
var _ port.EmailSender = (*EmailSender)(nil)

func NewEmailSender(cfg pkgconfig.SMTPConfig) *EmailSender
```

**Comportement** :
- Construit les headers MIME (From, To, Subject, Content-Type: text/html)
- Si `cfg.Username` est vide : connexion sans authentification (compatible MailHog en dev)
- Si `cfg.Username` est défini : auth SMTP PlainAuth
- En cas d'erreur : `apperrors.NewInternal("smtp: failed to send email", err)`

---

## 6. API Handlers

### 6.1 AuthHandler

**Fichier** : `backend/internal/api/handler/auth_handler.go`

```go
type AuthHandler struct {
    authService  *service.AuthService
    userRepo     port.UserRepository
    cookieSecure bool  // true en production (HTTPS)
}
```

Le handler gère directement la mécanique des cookies HTTP (pas de token dans les headers).

#### Endpoints

| Méthode | Route | Handler | Auth requise |
|---------|-------|---------|--------------|
| POST | `/api/v1/auth/register` | `Register` | Non (public) |
| POST | `/api/v1/auth/login` | `Login` | Non (public) |
| POST | `/api/v1/auth/logout` | `Logout` | Non (best-effort) |
| GET | `/api/v1/auth/me` | `Me` | Oui |
| POST | `/api/v1/auth/forgot-password` | `ForgotPassword` | Non (public) |
| POST | `/api/v1/auth/reset-password` | `ResetPassword` | Non (public) |

#### Cookie JWT

Tous les endpoints qui émettent un token utilisent `setTokenCookie` :

```go
http.Cookie{
    Name:     "token",
    Value:    token,
    Path:     "/api",
    HttpOnly: true,
    Secure:   cookieSecure,
    SameSite: http.SameSiteLaxMode,
    MaxAge:   int(authService.JWTExpiration().Seconds()),
}
```

Le logout efface le cookie en settant `MaxAge: -1` et `Value: ""`.

#### Réponses par endpoint

**Register** (POST `/auth/register`) :
- Body : `{"email":"...", "password":"...", "name":"..."}`
- 201 Created : `userResponse{id, email, name, role, created_at, updated_at}` + cookie
- 400 : validation (champs vides, password < 8 chars, JSON invalide)
- 409 : email déjà utilisé

**Login** (POST `/auth/login`) :
- Body : `{"email":"...", "password":"..."}`
- 200 OK : `userResponse` + cookie
- 400 : champs manquants
- 401 : mauvaises credentials

**Logout** (POST `/auth/logout`) :
- Blackliste le token si présent (best-effort, échec non bloquant)
- 204 No Content + cookie effacé

**Me** (GET `/auth/me`) :
- Lit `userID` depuis le contexte (injecté par middleware Auth)
- 200 OK : `userResponse`
- 401 : pas de contexte auth

**ForgotPassword** (POST `/auth/forgot-password`) :
- Body : `{"email":"..."}`
- Toujours 202 Accepted (même email inconnu — anti-énumération)
- 400 : email vide

**ResetPassword** (POST `/auth/reset-password`) :
- Body : `{"token":"...", "password":"..."}`
- 200 OK : `{"message": "Password updated successfully"}`
- 400 + `RESET_TOKEN_EXPIRED` : token expiré
- 400 + `RESET_TOKEN_INVALID` : token invalide ou déjà utilisé
- 400 + `VALIDATION_ERROR` : password < 8 chars ou champs vides

---

### 6.2 UserHandler

**Fichier** : `backend/internal/api/handler/user_handler.go`

```go
type UserHandler struct {
    service *service.UserService
}
```

**Tous les endpoints nécessitent le rôle admin** (`requireAdmin` retourne 403 sinon).

#### Endpoints

| Méthode | Route | Handler | Statut succès |
|---------|-------|---------|---------------|
| GET | `/api/v1/users` | `ListUsers` | 200 + `UserList` paginé |
| GET | `/api/v1/users/{id}` | `GetUser` | 200 + `User` |
| PUT | `/api/v1/users/{id}` | `UpdateUser` | 200 + `User` mis à jour |
| DELETE | `/api/v1/users/{id}` | `DeleteUser` | 204 No Content |

**`ListUsers`** : paramètres de query `page` et `per_page` (optionnels, défaut 1 et 20). Retourne :

```json
{
  "data": [...],
  "pagination": {"total": 42, "page": 1, "per_page": 20}
}
```

**`UpdateUser`** : body `UpdateUserRequest` — tous les champs sont optionnels (patch). Permet de changer name, email et/ou role.

---

### 6.3 ProjectHandler

**Fichier** : `backend/internal/api/handler/project_handler.go`

```go
type ProjectHandler struct {
    service        *service.ProjectService
    userService    *service.ProjectUserService
    circuitBreaker *service.CircuitBreakerService
}
```

#### Contrôle d'accès différencié

```go
// checkProjectAccess : admins passent toujours, sinon vérification de membership
func (h *ProjectHandler) checkProjectAccess(r *http.Request, projectID uuid.UUID) error {
    if middleware.IsAdmin(r.Context()) {
        return nil
    }
    // IsUserInProject via ProjectUserService
}
```

#### Endpoints

| Méthode | Route | Accès | Comportement |
|---------|-------|-------|-------------|
| GET | `/api/v1/projects` | Tous authentifiés | Admins : tous les projets. Non-admins : leurs projets via `ListProjectsForUser`. |
| POST | `/api/v1/projects` | Admin seulement | Crée le projet + seed pipeline config par défaut. Owner = user courant. |
| GET | `/api/v1/projects/{id}` | Admin ou membre | `checkProjectAccess` |
| PUT | `/api/v1/projects/{id}` | Admin seulement | Patch partiel avec les `SetXxx bool` |
| DELETE | `/api/v1/projects/{id}` | Admin seulement | Hard delete |
| POST | `/api/v1/projects/{id}/circuit-breaker/reset` | Admin seulement | Délègue à `CircuitBreakerService.Reset` |

---

### 6.4 ProjectUserHandler

**Fichier** : `backend/internal/api/handler/project_user_handler.go`

```go
type ProjectUserHandler struct {
    service *service.ProjectUserService
}
```

#### Endpoints

| Méthode | Route | Accès | Comportement |
|---------|-------|-------|-------------|
| POST | `/api/v1/projects/{id}/users` | Admin seulement | Ajoute un user au projet. Role défaut = `member` si omis. Body : `{"user_id": "...", "role": "owner|member"}` |
| DELETE | `/api/v1/projects/{id}/users/{user_id}` | Admin seulement | Retire un user du projet. 404 si non-membre. |
| GET | `/api/v1/projects/{id}/users` | Tous authentifiés | Liste les membres. Pas de pagination (retourne tous). |

**Response membre** :
```json
{
  "user_id": "...",
  "email": "...",
  "name": "...",
  "user_role": "admin|user",
  "project_role": "owner|member",
  "assigned_at": "2026-02-15T10:30:00Z"
}
```

---

### 6.5 ProfileHandler

**Fichier** : `backend/internal/api/handler/profile_handler.go`

```go
type ProfileHandler struct {
    userService *service.UserService
}
```

Endpoints self-service pour l'utilisateur authentifié.

| Méthode | Route | Comportement |
|---------|-------|-------------|
| GET | `/api/v1/users/me` | Retourne le profil de l'user courant (lu depuis contexte) |
| PUT | `/api/v1/users/me` | Met à jour name et/ou email (au moins un requis). Role non modifiable. |
| PUT | `/api/v1/users/me/password` | Change le mot de passe : vérifie `current_password`, valide et hash `new_password`. |

**ChangeMyPassword** response :
- 204 No Content : succès
- 401 + `INVALID_CREDENTIALS` : mauvais mot de passe courant
- 400 : champ vide ou nouveau mot de passe trop court

---

## 7. Middleware

### 7.1 Auth middleware

**Fichier** : `backend/internal/api/middleware/auth.go`

```go
func Auth(authService *service.AuthService, blacklistRepo port.TokenBlacklistRepository) func(http.Handler) http.Handler
```

**Flux pour chaque requête** :

```
1. isPublicPath(r.URL.Path) ? → next.ServeHTTP (bypass)
2. Lire le cookie "token" → 401 si absent
3. authService.ValidateToken(cookie.Value) → 401 si invalide/expiré
4. claims.ID != "" && blacklistRepo != nil → blacklistRepo.IsRevoked(jti) → 401 + TOKEN_REVOKED si révoqué
5. Injecter (userID, role) dans le contexte
6. next.ServeHTTP
```

**Chemins publics** (bypass auth) :
- `/healthz`
- `/api/v1/auth/register`
- `/api/v1/auth/login`
- `/api/v1/auth/forgot-password`
- `/api/v1/auth/reset-password`

**Context keys** :

```go
const (
    ContextKeyUserID contextKey = "user_id"
    ContextKeyRole   contextKey = "user_role"
)
```

**Helpers context** :
```go
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool)
func RoleFromContext(ctx context.Context) (model.Role, bool)
func IsAdmin(ctx context.Context) bool
func SetUserContext(ctx context.Context, userID uuid.UUID, role model.Role) context.Context
```

---

### 7.2 RBAC middleware

**Fichier** : `backend/internal/api/middleware/rbac.go`

```go
func RequireProjectAccess(repo port.ProjectUserRepository) func(http.Handler) http.Handler
```

Middleware chi qui vérifie l'accès au projet identifié par le paramètre URL `{id}` :
1. Vérifie authentification (UserID dans contexte)
2. Si admin → bypass
3. Sinon : `repo.IsUserInProject(projectID, userID)` → 403 si non-membre

**Helpers HTTP** internes (pour les réponses d'erreur JSON) :
- `writeForbidden(w, msg)` → 403 + `{"error":{"code":"FORBIDDEN","message":"..."}}`
- `writeBadRequest(w, msg)` → 400 + `{"error":{"code":"VALIDATION_ERROR","message":"..."}}`
- `writeInternalError(w)` → 500

**`requireAdmin` dans `handler/helpers.go`** : vérifie `middleware.IsAdmin(r.Context())`, retourne 403 si non-admin. Utilisé directement dans les handlers (pas un middleware chi).

---

## 8. Flux complets

### 8.1 Login

```
POST /api/v1/auth/login
  Body: {"email": "...", "password": "..."}

Auth middleware → isPublicPath → bypass

AuthHandler.Login
  → authService.Login(ctx, email, password)
      → userRepo.GetByEmail → pgx → users table
      → bcrypt.CompareHashAndPassword
      → generateToken(userID, role)
          → jwt.NewWithClaims(HS256, Claims{UserID, Role, JTI, ExpiresAt, IssuedAt})
          → token.SignedString(jwtSecret)
      → return user, token
  → setTokenCookie(w, token) → HttpOnly cookie "token"
  → writeJSON(w, 200, userResponse)
```

### 8.2 Forgot / Reset Password

```
POST /api/v1/auth/forgot-password
  Body: {"email": "user@example.com"}

AuthHandler.ForgotPassword
  → authService.ForgotPassword(ctx, email)
      → userRepo.GetByEmail → nil si inconnu → return nil (no-op)
      → generateSecureToken() → 32 random bytes → base64url
      → tokenRepo.Create(userID, token, now+1h)
      → emailSender.Send(ctx, EmailMessage{To: email, HTMLBody: reset_link_html})
  → writeJSON(w, 202, {"message": "If this email..."})

POST /api/v1/auth/reset-password
  Body: {"token": "...", "password": "newpass"}

AuthHandler.ResetPassword
  → authService.ResetPassword(ctx, token, newPassword)
      → tokenRepo.GetByToken(token) → ErrResetTokenInvalid si not found
      → prt.IsUsed() → ErrResetTokenInvalid si déjà utilisé
      → prt.IsExpired() → ErrResetTokenExpired si expiré
      → bcrypt.GenerateFromPassword(newPassword)
      → userRepo.GetByID(prt.UserID)
      → userRepo.Update(user avec nouveau hash)
      → tokenRepo.MarkUsed(prt.ID)
  → writeJSON(w, 200, {"message": "Password updated successfully"})
```

### 8.3 Ajout d'un membre à un projet

```
POST /api/v1/projects/{id}/users
  Body: {"user_id": "uuid", "role": "member"}

Auth middleware
  → cookie "token" → ValidateToken → injecte (userID, role=admin) dans ctx

ProjectUserHandler.AddUser
  → requireAdmin → IsAdmin(ctx) → ok (admin requis)
  → uuid.Parse(chi.URLParam("id")) → projectID
  → decodeJSONBody → AddProjectUserRequest{userID, role}
  → role vide → default "member"
  → projectUserService.AddUser(ctx, projectID, userID, role)
      → role.IsValid() → ok
      → projectRepo.GetByID(projectID) → vérifie existence
      → userRepo.GetByID(userID) → vérifie existence
      → repo.AddUser(projectID, userID, role)
          → INSERT INTO project_users → SQLSTATE 23505 → NewConflict
  → writeJSON(w, 201, ProjectUser)
```

---

## 9. Tests

### Stratégie générale

Tous les tests des services et handlers sont des **tests unitaires** utilisant des mocks hand-written (pas de mockgen). Chaque test suit le pattern **table-driven**.

Les tests d'intégration (avec Postgres réel via testcontainers) existent pour d'autres domaines mais pas pour auth/users/projects au moment de la rédaction.

### Mocks disponibles

#### Dans `service/auth_service_test.go`

| Mock | Port implémenté | Notes |
|------|----------------|-------|
| `mockUserRepository` | `port.UserRepository` | In-memory map, supporte `createFn` override |
| `mockPasswordResetTokenRepo` | `port.PasswordResetTokenRepository` | In-memory, supporte `createFn`/`getByTokenFn`/`markUsedFn` overrides |
| `mockEmailSender` | `port.EmailSender` | Capture `lastMsg` et compteur `sendCall` |
| `mockBlacklistRepo` | `port.TokenBlacklistRepository` | In-memory map, supporte `revokeFn`/`revokedFn` overrides |
| `pgDuplicateKeyError` | Simulation SQLSTATE 23505 | Implémente l'interface `SQLState() string` |

#### Dans `service/user_service_test.go`

| Mock | Port implémenté |
|------|----------------|
| `mockUserRepo` | `port.UserRepository` |

Supporte `createFn`, `updateFn`, `deleteFn`, `updatePasswordFn` pour injecter des comportements d'erreur.

#### Dans `service/project_service_test.go`

| Mock | Port implémenté |
|------|----------------|
| `mockProjectRepo` | `port.ProjectRepository` |

#### Dans `service/project_user_service_test.go`

| Mock | Port implémenté |
|------|----------------|
| `mockProjectUserRepo` | `port.ProjectUserRepository` |
| `mockProjectUserServiceUserRepo` | `port.UserRepository` |

Réutilise `mockProjectRepo` de `project_service_test.go`.

#### Dans `handler/auth_handler_test.go`

| Mock | Port implémenté |
|------|----------------|
| `mockRepo` | `port.UserRepository` |
| `mockTokenRepo` | `port.PasswordResetTokenRepository` |
| `mockEmailSender` | `port.EmailSender` |
| `pgDupError` | Simulation SQLSTATE 23505 |

#### Dans `middleware/auth_test.go`

| Mock | Interface |
|------|-----------|
| `noopRepo` | `port.UserRepository` |
| `noopTokenRepo` | `port.PasswordResetTokenRepository` |
| `noopEmailSender` | `port.EmailSender` |
| `mockBlacklistRepo` | `port.TokenBlacklistRepository` |

### Cas couverts

#### AuthService

| Test | Cas |
|------|-----|
| `TestRegister_Success` | Création user, hash bcrypt, token JWT |
| `TestRegister_DuplicateEmail` | → `ErrEmailAlreadyExists` |
| `TestRegister_ValidationError` | Email/password/name vides, password < 8 chars |
| `TestLogin_Success` | Credentials corrects → token |
| `TestLogin_WrongPassword` | → `ErrInvalidCredentials` |
| `TestLogin_NonexistentUser` | → `ErrInvalidCredentials` (même erreur, anti-énumération) |
| `TestValidateToken_Success` | Claims correctes (role, userID) |
| `TestValidateToken_InvalidToken` | Token malformé |
| `TestValidateToken_WrongSecret` | Token d'un autre service |
| `TestValidateToken_ExpiredToken` | Token avec expiration négative |
| `TestGenerateToken_HasJTI` | JTI non vide |
| `TestAuthService_Logout_RevokesToken` | JTI ajouté au blacklist |
| `TestAuthService_Logout_InvalidToken_Noop` | Token invalide → no-op sans erreur |
| `TestAuthService_Logout_EmptyJTI_Noop` | Token legacy sans JTI → no-op |
| `TestForgotPassword_ValidEmail` | Token créé, email envoyé |
| `TestForgotPassword_UnknownEmail` | Retourne nil (anti-énumération) |
| `TestForgotPassword_EmptyEmail` | → `ErrValidation` |
| `TestResetPassword_ValidToken` | Nouveau password, vieux password invalide, token marqué utilisé |
| `TestResetPassword_ExpiredToken` | → `ErrResetTokenExpired` |
| `TestResetPassword_UsedToken` | → `ErrResetTokenInvalid` |
| `TestResetPassword_TokenNotFound` | → `ErrResetTokenInvalid` |
| `TestResetPassword_WeakPassword` | → `ErrValidation` |
| `TestResetPassword_EmptyFields` | → `ErrValidation` (token vide, password vide, les deux) |

#### UserService

| Test | Cas |
|------|-----|
| `TestUserService_GetByID` | Existant et non-existant (`USER_NOT_FOUND`) |
| `TestUserService_List` | Pagination, clamp page/perPage |
| `TestUserService_Update` | name, email, role, not found, invalid role, empty fields |
| `TestUserService_Delete` | Existant et non-existant |
| `TestUserService_UpdateProfile` | name, email, combinés, empty fields, user not found |
| `TestUserService_UpdateProfile_DuplicateEmail` | → `EMAIL_ALREADY_EXISTS` |
| `TestUserService_ChangePassword` | Succès, mauvais password, nouveau trop court, user not found |

#### ProjectService

| Test | Cas |
|------|-----|
| `TestProjectService_Create` | Valide, name vide, name trop long, description trop longue |
| `TestProjectService_List` | Pagination, clamp |
| `TestProjectService_Update` | Valide, not found, empty name |
| `TestProjectService_Delete` | Existant et non-existant |

#### ProjectUserService

| Test | Cas |
|------|-----|
| `TestProjectUserService_AddUser` | Valide, doublon, projet inexistant, user inexistant, rôle invalide |
| `TestProjectUserService_RemoveUser` | Existant et non-membre |
| `TestProjectUserService_ListMembers` | 2 membres, projet inexistant |
| `TestProjectUserService_ListProjectsForUser` | Total, pagination page 2 |

#### AuthHandler (HTTP)

| Test | Cas |
|------|-----|
| `TestRegisterHandler_Success` | 201, cookie HttpOnly, rôle dans réponse |
| `TestRegisterHandler_DuplicateEmail` | 409 |
| `TestRegisterHandler_ValidationError` | 400 (missing, short password, invalid JSON) |
| `TestLoginHandler_Success` | 200, cookie |
| `TestLoginHandler_WrongPassword` | 401 |
| `TestLogoutHandler` | 204, cookie effacé (MaxAge=-1) |
| `TestMeHandler_Authenticated` | 200, email correct |
| `TestMeHandler_Unauthenticated` | 401 |
| `TestForgotPasswordHandler_*` | 202 email connu, 202 email inconnu, 400 email vide |
| `TestResetPasswordHandler_*` | 200 token valide, 400 expiré (`RESET_TOKEN_EXPIRED`), 400 invalide (`RESET_TOKEN_INVALID`), 400 password faible, 400 champs manquants |

#### Auth Middleware

| Test | Cas |
|------|-----|
| `TestAuthMiddleware_ValidToken` | UserID et Role injectés dans contexte |
| `TestAuthMiddleware_NoCookie` | 401 |
| `TestAuthMiddleware_InvalidToken` | 401 |
| `TestAuthMiddleware_ExpiredToken` | 401 |
| `TestAuthMiddleware_RevokedToken_Returns401` | 401 + body `TOKEN_REVOKED` |
| `TestAuthMiddleware_ValidToken_NotRevoked_Passes` | 200, UserID correct |
| `TestContextHelpers` | UserIDFromContext, RoleFromContext sur contexte vide et rempli |
