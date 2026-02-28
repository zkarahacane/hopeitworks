# Diagrammes Mermaid — Auth, Users & Projects

## 1. Architecture hexagonale — Auth/Users/Projects

Les 4 couches de l'architecture hexagonale appliquées au domaine Auth. Montre l'isolation stricte entre Handler, Service, Port et Adapter.

```mermaid
graph TB
    subgraph HTTP["Couche HTTP"]
        AH[AuthHandler]
        UH[UserHandler]
        PH[ProjectHandler]
        PUH[ProjectUserHandler]
        PR[ProfileHandler]
    end

    subgraph Services["Couche Service (logique métier)"]
        AS[AuthService]
        US[UserService]
        PS[ProjectService]
        PUS[ProjectUserService]
    end

    subgraph Ports["Ports (interfaces)"]
        UR[UserRepository]
        PjR[ProjectRepository]
        PUR[ProjectUserRepository]
        PRT[PasswordResetTokenRepository]
        TBL[TokenBlacklistRepository]
        ES[EmailSender]
    end

    subgraph Adapters["Adapters (implémentations)"]
        PgUser[postgres/UserRepository]
        PgProj[postgres/ProjectRepo]
        PgPU[postgres/ProjectUserRepo]
        PgPRT[postgres/PasswordResetTokenRepository]
        PgTBL[postgres/TokenBlacklistRepo]
        SMTP[smtp/EmailSender]
    end

    subgraph External["Systèmes externes"]
        DB[(PostgreSQL)]
        MAIL[Serveur SMTP]
    end

    AH --> AS
    UH --> US
    PH --> PS
    PH --> PUS
    PUH --> PUS
    PR --> US

    AS --> UR
    AS --> PRT
    AS --> TBL
    AS --> ES
    US --> UR
    PS --> PjR
    PUS --> PUR
    PUS --> PjR
    PUS --> UR

    UR -.implements.- PgUser
    PjR -.implements.- PgProj
    PUR -.implements.- PgPU
    PRT -.implements.- PgPRT
    TBL -.implements.- PgTBL
    ES -.implements.- SMTP

    PgUser --> DB
    PgProj --> DB
    PgPU --> DB
    PgPRT --> DB
    PgTBL --> DB
    SMTP --> MAIL
```

## 2. Flux Login complet

Séquence complète du login : de la requête HTTP jusqu'au cookie JWT retourné, avec les cas d'erreur.

```mermaid
sequenceDiagram
    participant C as Client HTTP
    participant MW as Auth Middleware
    participant AH as AuthHandler
    participant AS as AuthService
    participant UR as UserRepository
    participant PG as PostgreSQL

    C->>MW: POST /api/v1/auth/login {email, password}
    MW->>MW: isPublicPath("/auth/login") → bypass auth
    MW->>AH: next.ServeHTTP

    AH->>AH: decodeJSONBody → {email, password}
    alt Champs vides
        AH-->>C: 400 Bad Request
    end

    AH->>AS: Login(ctx, email, password)
    AS->>UR: GetByEmail(ctx, email)
    UR->>PG: SELECT * FROM users WHERE email = $1
    PG-->>UR: row ou ErrNoRows

    alt Email inconnu
        UR-->>AS: ErrNoRows
        AS-->>AH: ErrInvalidCredentials
        AH-->>C: 401 Unauthorized
    end

    UR-->>AS: *User (avec PasswordHash)
    AS->>AS: bcrypt.CompareHashAndPassword(hash, password)

    alt Mot de passe incorrect
        AS-->>AH: ErrInvalidCredentials
        AH-->>C: 401 Unauthorized
    end

    AS->>AS: generateToken(userID, role)<br/>jwt.NewWithClaims(HS256, Claims{UserID, Role, JTI, ExpiresAt})
    AS->>AS: token.SignedString(jwtSecret)
    AS-->>AH: (*User, token, nil)

    AH->>AH: setTokenCookie(w, token)<br/>HttpOnly, SameSite=Lax, Path=/api
    AH-->>C: 200 OK + Set-Cookie: token=... + userResponse{id, email, name, role}
```

## 3. Flux Forgot Password et Reset Password

Deux séquences distinctes montrant l'anti-énumération sur ForgotPassword et la validation stricte sur ResetPassword.

```mermaid
sequenceDiagram
    participant C as Client HTTP
    participant AH as AuthHandler
    participant AS as AuthService
    participant UR as UserRepository
    participant TR as TokenRepository
    participant EM as EmailSender

    rect rgb(230, 245, 255)
        Note over C,EM: ForgotPassword — POST /api/v1/auth/forgot-password
        C->>AH: {email: "user@example.com"}
        AH->>AS: ForgotPassword(ctx, email)
        AS->>UR: GetByEmail(ctx, email)

        alt Email inconnu
            UR-->>AS: ErrNoRows
            AS-->>AH: nil (no-op, anti-énumération)
        else Email connu
            UR-->>AS: *User
            AS->>AS: generateSecureToken()<br/>32 random bytes → base64url
            AS->>TR: Create(userID, token, now+1h)
            TR-->>AS: *PasswordResetToken
            AS->>EM: Send(ctx, EmailMessage{To, Subject, HTMLBody})
            EM-->>AS: nil
            AS-->>AH: nil
        end
        AH-->>C: 202 Accepted — "If this email exists..."
    end

    rect rgb(255, 245, 230)
        Note over C,EM: ResetPassword — POST /api/v1/auth/reset-password
        C->>AH: {token: "abc...", password: "newpass123"}
        AH->>AS: ResetPassword(ctx, token, newPassword)
        AS->>TR: GetByToken(ctx, token)

        alt Token introuvable
            TR-->>AS: ErrNoRows
            AS-->>AH: ErrResetTokenInvalid
            AH-->>C: 400 RESET_TOKEN_INVALID
        end

        TR-->>AS: *PasswordResetToken
        AS->>AS: prt.IsUsed() → UsedAt != nil

        alt Déjà utilisé
            AS-->>AH: ErrResetTokenInvalid
            AH-->>C: 400 RESET_TOKEN_INVALID
        end

        AS->>AS: prt.IsExpired() → time.Now().After(ExpiresAt)

        alt Expiré
            AS-->>AH: ErrResetTokenExpired
            AH-->>C: 400 RESET_TOKEN_EXPIRED
        end

        AS->>AS: bcrypt.GenerateFromPassword(newPassword)
        AS->>UR: GetByID(prt.UserID)
        AS->>UR: Update(user avec nouveau PasswordHash)
        AS->>TR: MarkUsed(prt.ID)
        AS-->>AH: nil
        AH-->>C: 200 OK — "Password updated successfully"
    end
```

## 4. Modèle User et ses rôles

Classes du domaine : User, Project, ProjectUser, ProjectMember, PasswordResetToken avec leurs attributs et relations.

```mermaid
classDiagram
    class Role {
        <<enumeration>>
        admin
        user
        IsValid() bool
    }

    class ProjectRole {
        <<enumeration>>
        owner
        member
        IsValid() bool
    }

    class User {
        +UUID ID
        +string Email
        +string PasswordHash
        +string Name
        +Role Role
        +time.Time CreatedAt
        +time.Time UpdatedAt
        +*time.Time DeletedAt
    }

    class Project {
        +UUID ID
        +string Name
        +*string Description
        +*UUID OwnerID
        +*string RepoURL
        +string GitProvider
        +*string GitTokenEnv
        +string AgentRuntime
        +*string DefaultModel
        +*float64 MaxBudget
        +int CircuitBreakerCount
        +int CircuitBreakerMax
        +bool CircuitBreakerActive
        +time.Time CreatedAt
        +time.Time UpdatedAt
    }

    class ProjectUser {
        +UUID ProjectID
        +UUID UserID
        +ProjectRole Role
        +time.Time CreatedAt
    }

    class ProjectMember {
        +UUID UserID
        +string Email
        +string Name
        +Role UserRole
        +ProjectRole ProjectRole
        +time.Time AssignedAt
    }

    class PasswordResetToken {
        +UUID ID
        +UUID UserID
        +string Token
        +time.Time ExpiresAt
        +*time.Time UsedAt
        +time.Time CreatedAt
        +IsExpired() bool
        +IsUsed() bool
    }

    User "1" --> "1" Role : a
    ProjectUser "1" --> "1" ProjectRole : a
    User "1" --> "0..*" ProjectUser : est membre via
    Project "1" --> "0..*" ProjectUser : contient
    User "1" --> "0..*" PasswordResetToken : possède
    ProjectMember ..> ProjectUser : vue dénormalisée JOIN users
```

## 5. Cycle de vie du PasswordResetToken

États possibles d'un token de réinitialisation de mot de passe, de sa création à son invalidation.

```mermaid
stateDiagram-v2
    [*] --> Created : ForgotPassword()<br/>token généré, ExpiresAt = now+1h

    Created --> Valid : Requête valide reçue<br/>IsExpired()=false && IsUsed()=false

    Created --> Expired : time.Now() > ExpiresAt<br/>IsExpired() = true

    Valid --> Used : ResetPassword() OK<br/>MarkUsed() → UsedAt = now

    Used --> [*] : ErrResetTokenInvalid<br/>si nouvelle tentative d'utilisation

    Expired --> [*] : ErrResetTokenExpired<br/>si tentative d'utilisation

    note right of Created
        Token URL-safe base64
        32 bytes aléatoires
        Stocké en DB pour audit
    end note

    note right of Used
        UsedAt != nil
        Token non réutilisable
        Reste en DB (pas supprimé)
    end note
```

## 6. Circuit Breaker — Transitions d'états

Cycle de vie du circuit breaker intégré dans le modèle Project, avec les conditions de déclenchement et de remise à zéro.

```mermaid
stateDiagram-v2
    [*] --> ACTIVE : Projet créé<br/>CircuitBreakerCount = 0<br/>CircuitBreakerActive = false

    ACTIVE --> ACTIVE : Failure enregistrée<br/>IncrementCircuitBreakerCount()<br/>Count < Max

    ACTIVE --> BROKEN : Count >= Max<br/>CircuitBreakerActive = true<br/>Pipelines bloqués

    BROKEN --> ACTIVE : Admin → POST /circuit-breaker/reset<br/>ResetCircuitBreaker()<br/>Count = 0, Active = false

    note right of ACTIVE
        CircuitBreakerActive = false
        Nouvelles exécutions autorisées
    end note

    note right of BROKEN
        CircuitBreakerActive = true
        Toute tentative de run bloquée
        Intervention admin requise
    end note
```

## 7. Matrice d'accès aux endpoints Projects

Résumé des permissions par rôle sur tous les endpoints liés aux projets et à leurs membres.

```mermaid
graph LR
    subgraph Endpoints["Endpoints /api/v1/projects"]
        EP1["GET /projects"]
        EP2["POST /projects"]
        EP3["GET /projects/{id}"]
        EP4["PUT /projects/{id}"]
        EP5["DELETE /projects/{id}"]
        EP6["POST /projects/{id}/circuit-breaker/reset"]
        EP7["GET /projects/{id}/users"]
        EP8["POST /projects/{id}/users"]
        EP9["DELETE /projects/{id}/users/{user_id}"]
    end

    subgraph Admin["Role: admin"]
        A1["Tous les projets"]
        A2["Créer projet + seed pipeline"]
        A3["Accès direct"]
        A4["Patch partiel"]
        A5["Hard delete"]
        A6["Reset CB"]
        A7["Voir membres"]
        A8["Ajouter membre"]
        A9["Retirer membre"]
    end

    subgraph User["Role: user"]
        U1["Ses projets uniquement<br/>ListProjectsForUser"]
        U2["403 Forbidden"]
        U3["Si membre du projet<br/>checkProjectAccess"]
        U4["403 Forbidden"]
        U5["403 Forbidden"]
        U6["403 Forbidden"]
        U7["Tout authentifié"]
        U8["403 Forbidden"]
        U9["403 Forbidden"]
    end

    EP1 --> A1
    EP1 --> U1
    EP2 --> A2
    EP2 --> U2
    EP3 --> A3
    EP3 --> U3
    EP4 --> A4
    EP4 --> U4
    EP5 --> A5
    EP5 --> U5
    EP6 --> A6
    EP6 --> U6
    EP7 --> A7
    EP7 --> U7
    EP8 --> A8
    EP8 --> U8
    EP9 --> A9
    EP9 --> U9
```

## 8. Dépendances entre services

Graphe des couplages entre les 4 services du domaine, avec leurs ports injectés et les dépendances optionnelles.

```mermaid
graph TB
    subgraph Services
        AS[AuthService]
        US[UserService]
        PS[ProjectService]
        PUS[ProjectUserService]
        CBS[CircuitBreakerService]
        PCS[PipelineConfigService]
    end

    subgraph Ports
        UR[port.UserRepository]
        PjR[port.ProjectRepository]
        PUR[port.ProjectUserRepository]
        PRT[port.PasswordResetTokenRepository]
        TBL[port.TokenBlacklistRepository]
        ES[port.EmailSender]
    end

    AS -->|required| UR
    AS -->|required| PRT
    AS -->|required| ES
    AS -->|optional SetBlacklistRepo| TBL

    US -->|required| UR

    PS -->|required| PjR
    PS -->|optional SetPipelineConfigService| PCS

    PUS -->|required| PUR
    PUS -->|required| PjR
    PUS -->|required| UR

    CBS -->|required| PjR

    PH[ProjectHandler] -->|injects| PS
    PH -->|injects| PUS
    PH -->|injects| CBS

    style TBL stroke-dasharray: 5 5
    style PCS stroke-dasharray: 5 5
```

## 9. Validation du token JWT — Arbre de décision

Organigramme du middleware Auth montrant chaque point de contrôle et les erreurs retournées.

```mermaid
flowchart TD
    A([Requête HTTP entrante]) --> B{isPublicPath?}

    B -->|Oui| Z([next.ServeHTTP — bypass auth])

    B -->|Non| C{Cookie 'token' présent?}
    C -->|Non| E1([401 — NO_TOKEN])

    C -->|Oui| D{authService.ValidateToken<br/>Parse JWT + HMAC-SHA256}
    D -->|Invalide ou malformé| E2([401 — INVALID_TOKEN])
    D -->|Expiré| E3([401 — TOKEN_EXPIRED])

    D -->|Valide| E{claims.ID non vide<br/>ET blacklistRepo != nil?}
    E -->|Non| G[Injecter userID + role<br/>dans contexte]

    E -->|Oui| F{blacklistRepo.IsRevoked<br/>jti}
    F -->|Révoqué| E4([401 — TOKEN_REVOKED])
    F -->|Non révoqué| G

    G --> H([next.ServeHTTP])

    style E1 fill:#ff6b6b,color:#fff
    style E2 fill:#ff6b6b,color:#fff
    style E3 fill:#ff6b6b,color:#fff
    style E4 fill:#ff6b6b,color:#fff
    style Z fill:#51cf66,color:#fff
    style H fill:#51cf66,color:#fff
```

## 10. Cycle de vie des données utilisateur

Vue globale des transitions d'état d'un utilisateur, depuis la création jusqu'à la suppression, en passant par les sessions et le reset de mot de passe.

```mermaid
graph LR
    subgraph Creation["Création"]
        REG[POST /auth/register<br/>bcrypt hash password<br/>Role = user]
    end

    subgraph Active["Utilisateur actif"]
        USER[User<br/>DeletedAt = nil]
        COOKIE[Cookie JWT<br/>HttpOnly, SameSite=Lax]
    end

    subgraph Session["Gestion de session"]
        LOGIN[POST /auth/login<br/>bcrypt.Compare<br/>JWT signé HS256 + JTI]
        LOGOUT[POST /auth/logout<br/>JTI → TokenBlacklist]
    end

    subgraph PasswordReset["Reset de mot de passe"]
        FORGOT[ForgotPassword<br/>Token 32 bytes / 1h]
        RESET[ResetPassword<br/>Nouveau hash bcrypt<br/>Token → MarkUsed]
    end

    subgraph Deleted["Suppression"]
        SOFT[DELETE /users/{id}<br/>DeletedAt = now<br/>soft-delete]
    end

    REG -->|201 Created| USER
    USER --> LOGIN
    LOGIN -->|Set-Cookie token| COOKIE
    COOKIE -->|Requêtes authentifiées| USER
    COOKIE --> LOGOUT
    LOGOUT -->|JTI blacklisté<br/>Cookie MaxAge=-1| USER

    USER --> FORGOT
    FORGOT -->|Email envoyé| RESET
    RESET -->|Nouveau PasswordHash| USER

    USER --> SOFT
    SOFT -->|Irréversible| DELETED[Deleted<br/>DeletedAt != nil]

    note1["Opérations réversibles:<br/>login/logout (blacklist JTI)"]
    note2["Opération non réversible:<br/>soft-delete (pas de restore API)"]
```
