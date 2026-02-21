# Story feat-3: [BACK] Reset password API with MailHog SMTP

Status: ready-for-dev

## Story

As a platform user who has forgotten their password,
I want to request a password reset link via email and use it to set a new password,
so that I can regain access to my account without contacting an administrator.

## Acceptance Criteria (BDD)

**AC1: Forgot password — valid email**
- **Given** a registered user with email `user@example.com`
- **When** a POST request is made to `/api/v1/auth/forgot-password` with `{"email": "user@example.com"}`
- **Then** the API responds with HTTP 202 and body `{"message": "If this email is registered, a reset link has been sent"}`
- **And** a password reset token is created in the `password_reset_tokens` table with `expires_at = now() + 1h`
- **And** an email is sent via SMTP containing a link of the form `{FRONTEND_URL}/reset-password?token={token}`

**AC2: Forgot password — unknown email (no enumeration)**
- **Given** an email address that does not correspond to any registered user
- **When** a POST request is made to `/api/v1/auth/forgot-password` with that email
- **Then** the API responds with HTTP 202 and the **same body** as AC1 (no enumeration leak)
- **And** no token is created and no email is sent

**AC3: Reset password — valid token**
- **Given** a valid, unexpired, unused reset token in `password_reset_tokens`
- **When** a POST request is made to `/api/v1/auth/reset-password` with `{"token": "...", "password": "newPass123"}`
- **Then** the API responds with HTTP 200 and body `{"message": "Password updated successfully"}`
- **And** the user's `password_hash` is updated in the `users` table
- **And** the token's `used_at` is set to `now()` in `password_reset_tokens`

**AC4: Reset password — expired token**
- **Given** a reset token whose `expires_at` is in the past
- **When** a POST request is made to `/api/v1/auth/reset-password` with that token
- **Then** the API responds with HTTP 400 and error code `RESET_TOKEN_EXPIRED`

**AC5: Reset password — already-used token**
- **Given** a reset token whose `used_at` is not null
- **When** a POST request is made to `/api/v1/auth/reset-password` with that token
- **Then** the API responds with HTTP 400 and error code `RESET_TOKEN_INVALID`

**AC6: Reset password — token not found**
- **Given** a token string that does not exist in `password_reset_tokens`
- **When** a POST request is made to `/api/v1/auth/reset-password` with that token
- **Then** the API responds with HTTP 400 and error code `RESET_TOKEN_INVALID`

**AC7: Reset password — weak new password**
- **Given** a valid reset token
- **When** a POST request is made to `/api/v1/auth/reset-password` with a password shorter than 8 characters
- **Then** the API responds with HTTP 400 and error code `VALIDATION_ERROR`

## Tasks / Subtasks

- [ ] [BACK] Task 1: Add migration `000023_create_password_reset_tokens_table` (AC: #1, #3, #4, #5, #6)
  - [ ] Create `backend/migrations/000023_create_password_reset_tokens_table.up.sql` with the table DDL
  - [ ] Create `backend/migrations/000023_create_password_reset_tokens_table.down.sql` with `DROP TABLE`

- [ ] [BACK] Task 2: Add sqlc queries for `password_reset_tokens` (AC: #1, #3, #4, #5, #6)
  - [ ] Create `backend/queries/password_reset_tokens.sql` with the four queries listed in Dev Notes
  - [ ] Run `cd backend && sqlc generate` to produce `internal/adapter/postgres/password_reset_tokens.sql.go`

- [ ] [BACK] Task 3: Add domain model and port interfaces (AC: #1, #3)
  - [ ] Create `backend/internal/domain/model/password_reset_token.go` — `PasswordResetToken` struct
  - [ ] Create `backend/internal/domain/port/password_reset_token_repository.go` — `PasswordResetTokenRepository` interface
  - [ ] Create `backend/internal/domain/port/email_sender.go` — `EmailSender` interface

- [ ] [BACK] Task 4: Implement `PasswordResetTokenRepository` postgres adapter (AC: #1, #3, #4, #5, #6)
  - [ ] Create `backend/internal/adapter/postgres/password_reset_token_repository.go`
  - [ ] Map sqlc rows to domain model via `toDomainPasswordResetToken()` helper
  - [ ] Add compile-time interface guard `var _ port.PasswordResetTokenRepository = (*PasswordResetTokenRepository)(nil)`

- [ ] [BACK] Task 5: Implement SMTP `EmailSender` adapter with MailHog support (AC: #1)
  - [ ] Create `backend/internal/adapter/smtp/email_sender.go`
  - [ ] Use `net/smtp` stdlib — no third-party email library
  - [ ] Add `SMTPConfig` to `backend/pkg/config/config.go` and env var overrides to `backend/internal/config/loader.go`
  - [ ] Add compile-time interface guard `var _ port.EmailSender = (*EmailSender)(nil)`
  - [ ] Add unit test `backend/internal/adapter/smtp/email_sender_test.go` using a mock SMTP server or recorded dial

- [ ] [BACK] Task 6: Add `ForgotPassword` and `ResetPassword` methods to `AuthService` (AC: #1–#7)
  - [ ] Extend `backend/internal/domain/service/auth_service.go` — inject `PasswordResetTokenRepository` and `EmailSender`
  - [ ] Add sentinel errors: `ErrResetTokenExpired`, `ErrResetTokenInvalid`
  - [ ] Write table-driven unit tests in `backend/internal/domain/service/auth_service_test.go`

- [ ] [BACK] Task 7: Add HTTP handlers and register routes (AC: #1–#7)
  - [ ] Extend `backend/internal/api/handler/auth_handler.go` with `ForgotPassword` and `ResetPassword` handlers
  - [ ] Update `api/openapi.yaml` with the two new paths and schemas
  - [ ] Run `cd backend && make generate` to regenerate `gen_server.go`
  - [ ] Register routes in `backend/internal/api/handler/server.go`
  - [ ] Add MailHog service to `deploy/docker-compose.yml`
  - [ ] Add SMTP config and env var documentation

- [ ] [BACK] Task 8: Wire dependencies (AC: all)
  - [ ] Update `backend/cmd/api/wire.go` provider sets to include `SmtpEmailSender`, `PasswordResetTokenRepository`
  - [ ] Run `cd backend && wire ./cmd/api/` to regenerate `wire_gen.go`
  - [ ] Run `golangci-lint run ./...` — must pass with zero warnings

## Dev Notes

### Dependencies

- No new Go modules needed — use `net/smtp` from stdlib
- MailHog Docker image: `mailhog/mailhog:v1.0.1`

### File Paths (exact)

| File | Purpose |
|------|---------|
| `backend/migrations/000023_create_password_reset_tokens_table.up.sql` | Schema migration |
| `backend/migrations/000023_create_password_reset_tokens_table.down.sql` | Rollback migration |
| `backend/queries/password_reset_tokens.sql` | sqlc query source |
| `backend/internal/adapter/postgres/password_reset_tokens.sql.go` | Generated by sqlc — DO NOT EDIT |
| `backend/internal/adapter/postgres/password_reset_token_repository.go` | Repo adapter |
| `backend/internal/domain/model/password_reset_token.go` | Domain model |
| `backend/internal/domain/port/password_reset_token_repository.go` | Repository port |
| `backend/internal/domain/port/email_sender.go` | EmailSender port |
| `backend/internal/adapter/smtp/email_sender.go` | SMTP adapter |
| `backend/internal/adapter/smtp/email_sender_test.go` | Adapter unit test |
| `backend/internal/domain/service/auth_service.go` | Extended service |
| `backend/internal/api/handler/auth_handler.go` | Extended handler |
| `backend/cmd/api/wire.go` | DI wiring update |
| `backend/pkg/config/config.go` | SMTPConfig struct added |
| `backend/internal/config/loader.go` | SMTP env var overrides |
| `deploy/docker-compose.yml` | MailHog service added |
| `api/openapi.yaml` | New paths + schemas |

### Migration SQL Content (password_reset_tokens table)

**`000023_create_password_reset_tokens_table.up.sql`:**

```sql
CREATE TABLE password_reset_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_password_reset_tokens_token   ON password_reset_tokens (token);
CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens (user_id);
```

**`000023_create_password_reset_tokens_table.down.sql`:**

```sql
DROP TABLE IF EXISTS password_reset_tokens;
```

### sqlc Query Signatures

**`backend/queries/password_reset_tokens.sql`:**

```sql
-- name: CreatePasswordResetToken :one
INSERT INTO password_reset_tokens (user_id, token, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetPasswordResetToken :one
SELECT * FROM password_reset_tokens WHERE token = $1 LIMIT 1;

-- name: MarkPasswordResetTokenUsed :exec
UPDATE password_reset_tokens SET used_at = now() WHERE id = $1;

-- name: DeleteExpiredPasswordResetTokens :exec
DELETE FROM password_reset_tokens WHERE expires_at < now() AND used_at IS NULL;
```

After adding the file, run `cd backend && sqlc generate` to produce `internal/adapter/postgres/password_reset_tokens.sql.go`.

The generated `PasswordResetToken` row struct will look like (do not write this manually — shown for reference):

```go
type PasswordResetToken struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    Token     string
    ExpiresAt time.Time
    UsedAt    pgtype.Timestamptz
    CreatedAt time.Time
}
```

### Domain Model (PasswordResetToken)

**`backend/internal/domain/model/password_reset_token.go`:**

```go
package model

import (
    "time"

    "github.com/google/uuid"
)

// PasswordResetToken represents a one-time token for resetting a user's password.
type PasswordResetToken struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    Token     string
    ExpiresAt time.Time
    UsedAt    *time.Time
    CreatedAt time.Time
}

// IsExpired returns true if the token is past its expiry time.
func (t *PasswordResetToken) IsExpired() bool {
    return time.Now().After(t.ExpiresAt)
}

// IsUsed returns true if the token has already been consumed.
func (t *PasswordResetToken) IsUsed() bool {
    return t.UsedAt != nil
}
```

### New Port: PasswordResetTokenRepository

**`backend/internal/domain/port/password_reset_token_repository.go`:**

```go
package port

import (
    "context"

    "github.com/google/uuid"
    "github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// PasswordResetTokenRepository defines persistence operations for password reset tokens.
type PasswordResetTokenRepository interface {
    Create(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) (*model.PasswordResetToken, error)
    GetByToken(ctx context.Context, token string) (*model.PasswordResetToken, error)
    MarkUsed(ctx context.Context, id uuid.UUID) error
}
```

Note: add `"time"` import to the file above.

### New Port: EmailSender

**`backend/internal/domain/port/email_sender.go`:**

```go
package port

import "context"

// EmailMessage represents a single outbound email.
type EmailMessage struct {
    To      string
    Subject string
    // HTMLBody is the HTML email body. Plain-text is auto-derived by the adapter.
    HTMLBody string
}

// EmailSender delivers transactional emails.
type EmailSender interface {
    Send(ctx context.Context, msg EmailMessage) error
}
```

### SMTP Adapter implementation

**`backend/internal/adapter/smtp/email_sender.go`:**

```go
package smtp

import (
    "bytes"
    "context"
    "fmt"
    "html/template"
    "net/smtp"

    "github.com/zakari/hopeitworks/backend/internal/domain/port"
    apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
    pkgconfig "github.com/zakari/hopeitworks/backend/pkg/config"
)

// EmailSender implements port.EmailSender via stdlib net/smtp.
type EmailSender struct {
    cfg pkgconfig.SMTPConfig
}

var _ port.EmailSender = (*EmailSender)(nil)

// NewEmailSender creates a new SMTP-backed EmailSender.
func NewEmailSender(cfg pkgconfig.SMTPConfig) *EmailSender {
    return &EmailSender{cfg: cfg}
}

// Send delivers msg via the configured SMTP relay.
// MailHog accepts unauthenticated connections — no SMTP auth is used when
// cfg.Username is empty.
func (s *EmailSender) Send(_ context.Context, msg port.EmailMessage) error {
    addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

    headers := fmt.Sprintf(
        "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n",
        s.cfg.From, msg.To, msg.Subject,
    )
    body := headers + msg.HTMLBody

    var auth smtp.Auth
    if s.cfg.Username != "" {
        auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
    }

    if err := smtp.SendMail(addr, auth, s.cfg.From, []string{msg.To}, []byte(body)); err != nil {
        return apperrors.NewInternal("smtp: failed to send email", err)
    }
    return nil
}
```

### Docker Compose changes (MailHog service)

Add the following service to `deploy/docker-compose.yml` inside the `services:` block, before `volumes:`:

```yaml
  mailhog:
    image: mailhog/mailhog:v1.0.1
    container_name: hopeitworks-mailhog
    ports:
      - "${MAILHOG_SMTP_PORT:-1025}:1025"   # SMTP
      - "${MAILHOG_UI_PORT:-8025}:8025"     # Web UI
    networks:
      - hopeitworks
    restart: on-failure
```

Also add the `api` service dependencies on `mailhog`:
```yaml
  api:
    # ... existing config ...
    environment:
      # ... existing env vars ...
      SMTP_HOST: mailhog
      SMTP_PORT: 1025
      SMTP_FROM: noreply@hopeitworks.local
      FRONTEND_URL: ${FRONTEND_URL:-http://localhost:5173}
    depends_on:
      postgres:
        condition: service_healthy
      socket-proxy:
        condition: service_started
      mailhog:
        condition: service_started
```

### Config additions (SMTP settings)

**Addition to `backend/pkg/config/config.go`:**

Add `SMTP SMTPConfig` field to `Config` struct, and add the following type:

```go
// SMTPConfig holds outbound email relay settings.
type SMTPConfig struct {
    Host        string `yaml:"host"`
    Port        int    `yaml:"port"`
    From        string `yaml:"from"`
    Username    string `yaml:"username"`
    Password    string `yaml:"password"`
    FrontendURL string `yaml:"frontend_url"`
}
```

Updated `Config` struct:
```go
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Database DatabaseConfig `yaml:"database"`
    Docker   DockerConfig   `yaml:"docker"`
    Log      LogConfig      `yaml:"logging"`
    SMTP     SMTPConfig     `yaml:"smtp"`
}
```

**Additions to `backend/internal/config/loader.go` inside `applyEnvOverrides`:**

```go
if v := os.Getenv("SMTP_HOST"); v != "" {
    cfg.SMTP.Host = v
}
if v := os.Getenv("SMTP_PORT"); v != "" {
    if port, err := strconv.Atoi(v); err == nil {
        cfg.SMTP.Port = port
    }
}
if v := os.Getenv("SMTP_FROM"); v != "" {
    cfg.SMTP.From = v
}
if v := os.Getenv("SMTP_USERNAME"); v != "" {
    cfg.SMTP.Username = v
}
if v := os.Getenv("SMTP_PASSWORD"); v != "" {
    cfg.SMTP.Password = v
}
if v := os.Getenv("FRONTEND_URL"); v != "" {
    cfg.SMTP.FrontendURL = v
}
```

Default SMTP config in `backend/config.yaml` (add under existing keys):
```yaml
smtp:
  host: localhost
  port: 1025
  from: noreply@hopeitworks.local
  username: ""
  password: ""
  frontend_url: http://localhost:5173
```

### AuthService extension

**Extend `backend/internal/domain/service/auth_service.go`:**

Add the following sentinel errors at the top alongside existing ones:
```go
var (
    ErrInvalidCredentials  = errors.New("invalid credentials")
    ErrEmailAlreadyExists  = errors.New("email already exists")
    ErrValidation          = errors.New("validation error")
    ErrResetTokenExpired   = errors.New("reset token expired")
    ErrResetTokenInvalid   = errors.New("reset token invalid or already used")
)
```

Extend `AuthService` struct:
```go
type AuthService struct {
    repo           port.UserRepository
    tokenRepo      port.PasswordResetTokenRepository
    emailSender    port.EmailSender
    frontendURL    string
    jwtSecret      []byte
    jwtExpiration  time.Duration
}

func NewAuthService(
    repo port.UserRepository,
    tokenRepo port.PasswordResetTokenRepository,
    emailSender port.EmailSender,
    frontendURL string,
    jwtSecret string,
    jwtExpiration time.Duration,
) *AuthService {
    return &AuthService{
        repo:          repo,
        tokenRepo:     tokenRepo,
        emailSender:   emailSender,
        frontendURL:   frontendURL,
        jwtSecret:     []byte(jwtSecret),
        jwtExpiration: jwtExpiration,
    }
}
```

New methods to add to `AuthService`:
```go
// ForgotPassword generates a reset token and sends an email if the address is registered.
// Always returns nil error to prevent email enumeration.
func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
    if email == "" {
        return ErrValidation
    }

    user, err := s.repo.GetByEmail(ctx, email)
    if err != nil {
        // Unknown email — return nil to prevent enumeration.
        return nil
    }

    rawToken, err := generateSecureToken()
    if err != nil {
        return err
    }

    expiresAt := time.Now().Add(1 * time.Hour)
    if _, err := s.tokenRepo.Create(ctx, user.ID, rawToken, expiresAt); err != nil {
        return err
    }

    resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.frontendURL, rawToken)
    return s.emailSender.Send(ctx, port.EmailMessage{
        To:      user.Email,
        Subject: "Reset your HopeItWorks password",
        HTMLBody: buildResetEmailHTML(user.Name, resetLink),
    })
}

// ResetPassword validates the token and updates the user's password.
func (s *AuthService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
    if rawToken == "" || newPassword == "" {
        return ErrValidation
    }
    if len(newPassword) < 8 {
        return ErrValidation
    }

    prt, err := s.tokenRepo.GetByToken(ctx, rawToken)
    if err != nil {
        return ErrResetTokenInvalid
    }
    if prt.IsUsed() {
        return ErrResetTokenInvalid
    }
    if prt.IsExpired() {
        return ErrResetTokenExpired
    }

    hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
    if err != nil {
        return err
    }

    user, err := s.repo.GetByID(ctx, prt.UserID)
    if err != nil {
        return err
    }
    user.PasswordHash = string(hash)
    if _, err := s.repo.Update(ctx, user); err != nil {
        return err
    }

    return s.tokenRepo.MarkUsed(ctx, prt.ID)
}

// generateSecureToken returns a 32-byte URL-safe base64-encoded random token.
func generateSecureToken() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(b), nil
}
```

Required additional imports for `auth_service.go`:
```go
import (
    "crypto/rand"
    "encoding/base64"
    "fmt"
    // ... existing imports ...
)
```

### Email template (HTML reset link)

Add helper function in `auth_service.go` (or a separate `email_templates.go` in the service package):

```go
// buildResetEmailHTML returns a minimal HTML email body with the reset link.
func buildResetEmailHTML(name, resetLink string) string {
    return fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; padding: 24px;">
  <h2>Password Reset Request</h2>
  <p>Hi %s,</p>
  <p>We received a request to reset your HopeItWorks password.
     Click the button below to set a new password. This link expires in <strong>1 hour</strong>.</p>
  <p><a href="%s" style="background:#4F46E5;color:#fff;padding:12px 24px;border-radius:6px;text-decoration:none;">
    Reset my password
  </a></p>
  <p>If you did not request a password reset, you can ignore this email.</p>
</body>
</html>`, name, resetLink)
}
```

### OpenAPI spec additions

Add after the existing `/auth/me` path block in `api/openapi.yaml`:

```yaml
  /auth/forgot-password:
    post:
      operationId: forgotPassword
      summary: Request a password reset link via email
      tags: [auth]
      security: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/ForgotPasswordRequest"
      responses:
        "202":
          description: Reset email dispatched (always returned — prevents email enumeration)
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/MessageResponse"
        "400":
          $ref: "#/components/responses/BadRequest"

  /auth/reset-password:
    post:
      operationId: resetPassword
      summary: Set a new password using a reset token
      tags: [auth]
      security: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/ResetPasswordRequest"
      responses:
        "200":
          description: Password updated successfully
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/MessageResponse"
        "400":
          $ref: "#/components/responses/BadRequest"
```

Add the following schemas under `components/schemas` in `api/openapi.yaml`:

```yaml
    ForgotPasswordRequest:
      type: object
      required: [email]
      properties:
        email:
          type: string
          format: email
          example: user@example.com

    ResetPasswordRequest:
      type: object
      required: [token, password]
      properties:
        token:
          type: string
          description: The reset token received by email
          example: "dGVzdC10b2tlbi1mb3ItcmVzZXQ="
        password:
          type: string
          minLength: 8
          description: The new password (minimum 8 characters)
          example: "newSecurePass123"

    MessageResponse:
      type: object
      required: [message]
      properties:
        message:
          type: string
          example: "If this email is registered, a reset link has been sent"
```

After modifying the spec, run `cd backend && make generate` to regenerate `internal/api/handler/gen_server.go`.

### HTTP Handler additions

Extend `backend/internal/api/handler/auth_handler.go`:

```go
type forgotPasswordRequest struct {
    Email string `json:"email"`
}

type resetPasswordRequest struct {
    Token    string `json:"token"`
    Password string `json:"password"`
}

type messageResponse struct {
    Message string `json:"message"`
}

// ForgotPassword handles POST /auth/forgot-password.
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
    var req forgotPasswordRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
        return
    }
    if req.Email == "" {
        writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "email is required")
        return
    }

    // Ignore error — always return 202 to prevent enumeration.
    _ = h.authService.ForgotPassword(r.Context(), req.Email)

    writeJSON(w, http.StatusAccepted, messageResponse{
        Message: "If this email is registered, a reset link has been sent",
    })
}

// ResetPassword handles POST /auth/reset-password.
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
    var req resetPasswordRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
        return
    }
    if req.Token == "" || req.Password == "" {
        writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "token and password are required")
        return
    }

    if err := h.authService.ResetPassword(r.Context(), req.Token, req.Password); err != nil {
        switch {
        case errors.Is(err, service.ErrResetTokenExpired):
            writeError(w, http.StatusBadRequest, "RESET_TOKEN_EXPIRED", "The reset link has expired. Please request a new one.")
        case errors.Is(err, service.ErrResetTokenInvalid):
            writeError(w, http.StatusBadRequest, "RESET_TOKEN_INVALID", "The reset token is invalid or has already been used.")
        case errors.Is(err, service.ErrValidation):
            writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Password must be at least 8 characters.")
        default:
            writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
        }
        return
    }

    writeJSON(w, http.StatusOK, messageResponse{Message: "Password updated successfully"})
}
```

Register in `backend/internal/api/handler/server.go` under the auth routes block:
```go
r.Post("/auth/forgot-password", h.auth.ForgotPassword)
r.Post("/auth/reset-password",  h.auth.ResetPassword)
```

### Postgres Adapter implementation

**`backend/internal/adapter/postgres/password_reset_token_repository.go`:**

```go
package postgres

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/zakari/hopeitworks/backend/internal/domain/model"
    "github.com/zakari/hopeitworks/backend/internal/domain/port"
    apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// PasswordResetTokenRepository implements port.PasswordResetTokenRepository using sqlc.
type PasswordResetTokenRepository struct {
    q *Queries
}

var _ port.PasswordResetTokenRepository = (*PasswordResetTokenRepository)(nil)

// NewPasswordResetTokenRepository creates a new PasswordResetTokenRepository.
func NewPasswordResetTokenRepository(db DBTX) *PasswordResetTokenRepository {
    return &PasswordResetTokenRepository{q: New(db)}
}

func (r *PasswordResetTokenRepository) Create(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) (*model.PasswordResetToken, error) {
    row, err := r.q.CreatePasswordResetToken(ctx, CreatePasswordResetTokenParams{
        UserID:    userID,
        Token:     token,
        ExpiresAt: expiresAt,
    })
    if err != nil {
        return nil, apperrors.NewInternal("create password reset token", err)
    }
    return toDomainPasswordResetToken(row), nil
}

func (r *PasswordResetTokenRepository) GetByToken(ctx context.Context, token string) (*model.PasswordResetToken, error) {
    row, err := r.q.GetPasswordResetToken(ctx, token)
    if err != nil {
        if err == pgx.ErrNoRows {
            return nil, apperrors.NewNotFound("password_reset_token", token)
        }
        return nil, apperrors.NewInternal("get password reset token", err)
    }
    return toDomainPasswordResetToken(row), nil
}

func (r *PasswordResetTokenRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
    if err := r.q.MarkPasswordResetTokenUsed(ctx, id); err != nil {
        return apperrors.NewInternal("mark password reset token used", err)
    }
    return nil
}

func toDomainPasswordResetToken(row PasswordResetToken) *model.PasswordResetToken {
    var usedAt *time.Time
    if row.UsedAt.Valid {
        usedAt = &row.UsedAt.Time
    }
    return &model.PasswordResetToken{
        ID:        row.ID,
        UserID:    row.UserID,
        Token:     row.Token,
        ExpiresAt: row.ExpiresAt.Time,
        UsedAt:    usedAt,
        CreatedAt: row.CreatedAt.Time,
    }
}
```

Note: the sqlc-generated `PasswordResetToken` struct uses `pgtype.Timestamptz` for `ExpiresAt` and `UsedAt`. Adjust `.Time` field access accordingly once `sqlc generate` has run — inspect the generated struct in `internal/adapter/postgres/password_reset_tokens.sql.go` and correct the field types as needed.

### Error Responses

| Scenario | HTTP | Code |
|----------|------|------|
| Invalid/missing body fields | 400 | `VALIDATION_ERROR` |
| Token not found | 400 | `RESET_TOKEN_INVALID` |
| Token already used | 400 | `RESET_TOKEN_INVALID` |
| Token expired | 400 | `RESET_TOKEN_EXPIRED` |
| Password < 8 chars | 400 | `VALIDATION_ERROR` |
| Internal/SMTP error | 500 | `INTERNAL_ERROR` |

### Security Considerations

- **No email enumeration**: `ForgotPassword` always returns HTTP 202 regardless of whether the email exists. Do not log or surface any difference.
- **Token entropy**: 32 random bytes encoded as URL-safe base64 = 256-bit entropy. Sufficient against brute-force.
- **Token expiry**: 1 hour. Non-configurable for MVP; add to `SMTPConfig` in a future story if needed.
- **Single-use enforcement**: `used_at` is set atomically on successful password change. Re-use of a consumed token returns `RESET_TOKEN_INVALID`.
- **Token storage**: raw token stored in plain text in Postgres (acceptable for MVP). If hardened security is required later, store a SHA-256 hash and compare hash on lookup.
- **Rate limiting**: not enforced at MVP — document the absence explicitly. A future story should add per-IP or per-email rate limiting at the middleware layer before the `ForgotPassword` handler.
- **SMTP credentials**: never log `cfg.SMTP.Password`. The `ScrubHandler` in `pkg/log` already redacts fields containing `password`, but SMTP config must only be passed to the adapter, not exposed in handler context.
- **Token cleanup**: `DeleteExpiredPasswordResetTokens` query is defined but not called automatically at MVP. Wire it to a periodic River job or run it as a manual cron in a future story.

### Testing Requirements

**Unit tests — `backend/internal/domain/service/auth_service_test.go`:**

Table-driven tests covering:
- `ForgotPassword`: email found → token created + email sent
- `ForgotPassword`: email not found → returns nil (no error)
- `ForgotPassword`: empty email → returns `ErrValidation`
- `ResetPassword`: valid token → password updated + token marked used
- `ResetPassword`: expired token → returns `ErrResetTokenExpired`
- `ResetPassword`: used token → returns `ErrResetTokenInvalid`
- `ResetPassword`: token not found → returns `ErrResetTokenInvalid`
- `ResetPassword`: password < 8 chars → returns `ErrValidation`

Use hand-written mocks for `port.PasswordResetTokenRepository` and `port.EmailSender`. Pattern:
```go
type mockPasswordResetTokenRepo struct {
    createFn     func(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) (*model.PasswordResetToken, error)
    getByTokenFn func(ctx context.Context, token string) (*model.PasswordResetToken, error)
    markUsedFn   func(ctx context.Context, id uuid.UUID) error
}

type mockEmailSender struct {
    sendFn func(ctx context.Context, msg port.EmailMessage) error
}
```

**Unit tests — `backend/internal/adapter/smtp/email_sender_test.go`:**

- Test that `Send` constructs correct MIME headers (From, To, Subject, Content-Type)
- Can use `net/smtp/smtptest` if available, otherwise skip integration and mock `smtp.SendMail` via dependency injection of a `sendMailFn` field

**Handler tests — `backend/internal/api/handler/auth_handler_test.go`:**

Add cases for:
- `POST /auth/forgot-password`: valid email → 202
- `POST /auth/forgot-password`: missing email → 400 `VALIDATION_ERROR`
- `POST /auth/reset-password`: valid token → 200
- `POST /auth/reset-password`: expired token → 400 `RESET_TOKEN_EXPIRED`
- `POST /auth/reset-password`: invalid token → 400 `RESET_TOKEN_INVALID`
- `POST /auth/reset-password`: weak password → 400 `VALIDATION_ERROR`

**Integration test (optional, tagged):**

`backend/internal/adapter/postgres/password_reset_token_repository_integration_test.go`:
- Create token, retrieve by token string, mark used, verify `used_at` is set
- Use `testutil.NewTestDB(t)` for ephemeral Postgres container

## Dev Agent Record

_To be filled by the implementing agent._

## Change Log

| Date | Change | Author |
|------|--------|--------|
| 2026-02-21 | Initial story created | architect |
