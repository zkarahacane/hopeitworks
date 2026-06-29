package service

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/crypto"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Compile-time check: GitConnectionService is THE credential resolution seam.
var _ port.GitCredentialResolver = (*GitConnectionService)(nil)

// Error codes surfaced to the API (mirror api/openapi.yaml). KeyUnset / Invalid /
// InsufficientScope are 422 (CategoryInvalidState); ProbeUnavailable is the
// transient sentinel mapped to 503 by the handler.
const (
	CodeGitConnectionKeyUnset          = "GIT_CONNECTION_KEY_UNSET"
	CodeGitConnectionInvalid           = "GIT_CONNECTION_INVALID"
	CodeGitConnectionInsufficientScope = "GIT_CONNECTION_INSUFFICIENT_SCOPE"
	CodeGitConnectionProbeUnavailable  = "GIT_CONNECTION_PROBE_UNAVAILABLE"

	providerGitHub = "github"
	providerGitea  = "gitea"
)

// ErrGitConnectionProbeUnavailable is the transient-probe sentinel: on PUT/Test the
// provider was unreachable or returned 429/5xx. The handler maps it to HTTP 503 and
// a good existing row is never overwritten.
var ErrGitConnectionProbeUnavailable = stderrors.New(CodeGitConnectionProbeUnavailable)

// knownInsecureKeys are master keys that must NOT be used to encrypt a real PAT
// (B1). An empty key (ENCRYPTION_KEY unset) or a shipped dev default is rejected at
// PUT time only — boot is never blocked.
var knownInsecureKeys = map[string]bool{
	"":                             true,
	"dev-encryption-key-32bytes!!": true,
	"changeme":                     true,
}

// requiredClassicScopes is the minimum classic-PAT scope set for Projects v2 read.
// A token carrying "project" (read+write) also satisfies "read:project".
var requiredClassicScopes = []string{"read:project"}

// SetGitConnectionInput is the validated PUT payload (handler applies defaults).
type SetGitConnectionInput struct {
	Provider string
	Token    string
	Validate bool
}

// GitConnectionView is the advisory status the API returns. It NEVER carries the token.
type GitConnectionView struct {
	Configured      bool
	Source          string // connection | env | none
	Kind            string // pat
	Provider        string
	Status          model.GitConnectionStatus
	SecretLast4     *string
	TokenType       *string
	AccountLogin    *string
	Scopes          []string
	ExpiresAt       *time.Time
	LastValidatedAt *time.Time
	ValidationError *string
}

// GitConnectionTestView is the live "Test connection" result.
type GitConnectionTestView struct {
	Ok            bool
	Status        model.GitConnectionStatus
	AccountLogin  *string
	Scopes        []string
	MissingScopes []string
	TokenType     string
	ExpiresAt     *time.Time
	Message       string
}

// GitConnectionService is both the management service (Status/Set/Test/Clear) and
// the credential resolver the factories consume. PAT plaintext exists only
// transiently in memory; it is encrypted at rest with the same AES-256-GCM key as
// user API keys / credentials.
type GitConnectionService struct {
	repo          port.GitConnectionRepository
	projectRepo   port.ProjectRepository
	validator     port.GitConnectionValidator
	encryptionKey []byte
	keyInsecure   bool // B1: master key empty/dev-default => refuse to store a PAT
	events        port.EventPublisher
	logger        *slog.Logger
}

// NewGitConnectionService wires the service. masterKey is the SAME master key as
// CredentialService/APIKeyService (derived once via crypto.DeriveKey).
func NewGitConnectionService(
	repo port.GitConnectionRepository,
	projectRepo port.ProjectRepository,
	validator port.GitConnectionValidator,
	masterKey string,
	events port.EventPublisher,
	logger *slog.Logger,
) *GitConnectionService {
	if logger == nil {
		logger = slog.Default()
	}
	return &GitConnectionService{
		repo:          repo,
		projectRepo:   projectRepo,
		validator:     validator,
		encryptionKey: crypto.DeriveKey(masterKey),
		keyInsecure:   knownInsecureKeys[masterKey],
		events:        events,
		logger:        logger,
	}
}

// LoadProject fetches the project for authorization (owner-or-admin) checks. Returns
// a not-found DomainError when absent.
func (s *GitConnectionService) LoadProject(ctx context.Context, projectID uuid.UUID) (*model.Project, error) {
	return s.projectRepo.GetByID(ctx, projectID)
}

// ─── Credential resolution seam (port.GitCredentialResolver) ────────────────────

// TokenForProject returns the active token for a project: the stored PAT, else the
// legacy env fallback. On decrypt failure (rotated key, B2) it self-marks the row
// invalid and falls back to env rather than hard-failing.
func (s *GitConnectionService) TokenForProject(ctx context.Context, projectID uuid.UUID) (port.GitToken, error) {
	conn, err := s.repo.GetByProject(ctx, projectID)
	if isNotFound(err) {
		//nolint:nilerr // not-found is expected: fall back to the legacy env token.
		return s.legacyEnvFallback(ctx, projectID), nil
	}
	if err != nil {
		return port.GitToken{}, err
	}

	raw, derr := crypto.Decrypt(conn.EncryptedSecret, s.encryptionKey)
	if derr != nil {
		// B2: rotated/oversized key — never hard-500; mark invalid + env fallback.
		s.logger.Warn("git connection decrypt failed; falling back to env token (rotate ENCRYPTION_KEY => re-enter tokens)",
			"project_id", projectID)
		_ = s.repo.MarkStatus(ctx, projectID, model.GitConnStatusInvalid, strptr(model.ValidationErrDecryptFailed))
		//nolint:nilerr // decrypt failure (rotated key) intentionally degrades to the env token, not a 500.
		return s.legacyEnvFallback(ctx, projectID), nil
	}
	return port.GitToken{Value: string(raw), ExpiresAt: conn.ExpiresAt}, nil
}

// ReconcileFromOperationError self-heals the advisory status from a REAL operation
// error (C1): 401 -> invalid, 403 -> insufficient_scope; transient (429/5xx) is left
// untouched. Best-effort and a no-op when no row exists.
func (s *GitConnectionService) ReconcileFromOperationError(ctx context.Context, projectID uuid.UUID, opErr error) {
	if opErr == nil {
		return
	}
	status, code, flip := classifyOperationError(opErr)
	if !flip {
		return
	}
	if err := s.repo.MarkStatus(ctx, projectID, status, strptr(code)); err != nil {
		s.logger.Warn("git connection status self-heal failed", "project_id", projectID, "error", err)
		return
	}
	s.logger.Warn("git connection self-healed from operation error", "project_id", projectID, "status", string(status))
}

// legacyEnvFallback reproduces the pre-connection behaviour: os.Getenv(git_token_env)
// then GITHUB_TOKEN. Returns a zero token when nothing is set.
func (s *GitConnectionService) legacyEnvFallback(ctx context.Context, projectID uuid.UUID) port.GitToken {
	if p, err := s.projectRepo.GetByID(ctx, projectID); err == nil && p.GitTokenEnv != nil && *p.GitTokenEnv != "" {
		if v := os.Getenv(*p.GitTokenEnv); v != "" {
			return port.GitToken{Value: v}
		}
	}
	return port.GitToken{Value: os.Getenv("GITHUB_TOKEN")}
}

// ─── Management API ─────────────────────────────────────────────────────────────

// Status returns the project's advisory connection status. Absence of a row resolves
// to the env-or-none source. Expiry is computed lazily without calling GitHub.
func (s *GitConnectionService) Status(ctx context.Context, projectID uuid.UUID) (*GitConnectionView, error) {
	conn, err := s.repo.GetByProject(ctx, projectID)
	if isNotFound(err) {
		return s.unconfiguredView(ctx, projectID), nil
	}
	if err != nil {
		return nil, err
	}
	return s.viewFromConn(conn), nil
}

// Set validates (default) then stores an encrypted PAT. Validation outcomes:
//   - 2xx + sufficient scope -> persist status=connected.
//   - 401 -> GIT_CONNECTION_INVALID (422), not persisted.
//   - 403 / classic missing read:project -> GIT_CONNECTION_INSUFFICIENT_SCOPE (422), not persisted.
//   - transient (429/5xx/DNS/TLS) -> 503, never overwrites a good row.
//   - validate=false -> persist unverified (status=unconfigured, last_validated_at NULL).
func (s *GitConnectionService) Set(ctx context.Context, projectID uuid.UUID, actorUserID uuid.UUID, in SetGitConnectionInput) (*GitConnectionView, error) {
	token := strings.TrimSpace(in.Token)
	if token == "" {
		return nil, errors.NewValidation("token", "is required")
	}
	if s.keyInsecure {
		// B1: refuse to encrypt a real PAT under an empty/dev-default key.
		s.logger.Warn("refusing to store git PAT: ENCRYPTION_KEY is unset or a known dev default", "project_id", projectID)
		return nil, errors.NewInvalidState(CodeGitConnectionKeyUnset,
			"server encryption key is unset or insecure; set ENCRYPTION_KEY before storing a token")
	}

	provider := normalizeProvider(in.Provider)
	tokenType := classifyToken(token)

	if !in.Validate {
		view, err := s.persist(ctx, projectID, provider, token, persistFields{
			status:          model.GitConnStatusUnconfigured,
			tokenType:       &tokenType,
			lastValidatedAt: nil,
		})
		if err != nil {
			return nil, err
		}
		s.audit(ctx, projectID, actorUserID, "updated", map[string]any{
			"provider": provider, "token_type": tokenType, "status": string(model.GitConnStatusUnconfigured), "validated": false,
		})
		return view, nil
	}

	baseURL := s.providerBaseURL(ctx, projectID, provider)
	result, perr := s.validator.Probe(ctx, provider, baseURL, token)
	if perr != nil {
		return nil, s.mapProbeError(perr)
	}

	status, missing := s.statusFromProbe(result)
	if status == model.GitConnStatusInsufficient {
		return nil, errors.NewInvalidState(CodeGitConnectionInsufficientScope,
			"token is missing required scope(s): "+strings.Join(missing, ", "))
	}

	now := time.Now().UTC()
	view, err := s.persist(ctx, projectID, provider, token, persistFields{
		status:          status,
		tokenType:       &tokenType,
		scopes:          result.Scopes,
		accountLogin:    strptrIfNonEmpty(result.Login),
		expiresAt:       result.ExpiresAt,
		lastValidatedAt: &now,
	})
	if err != nil {
		return nil, err
	}
	s.audit(ctx, projectID, actorUserID, "updated", map[string]any{
		"provider": provider, "token_type": tokenType, "secret_last4": derefStrOr(view.SecretLast4, ""),
		"scopes": result.Scopes, "status": string(status), "validated": true,
	})
	return view, nil
}

// Test live-probes the stored token (or an unsaved body token) and, when testing the
// stored one, refreshes its advisory status. Returns ErrGitConnectionProbeUnavailable
// on transient failure (handler -> 503) and a 422 DomainError on invalid/insufficient.
func (s *GitConnectionService) Test(ctx context.Context, projectID uuid.UUID, actorUserID uuid.UUID, bodyToken *string) (*GitConnectionTestView, error) {
	testingStored := bodyToken == nil || strings.TrimSpace(*bodyToken) == ""

	provider := providerGitHub
	var token string
	if testingStored {
		conn, err := s.repo.GetByProject(ctx, projectID)
		if isNotFound(err) {
			return nil, errors.NewInvalidState(CodeGitConnectionInvalid, "no stored token to test; provide a token or save one first")
		}
		if err != nil {
			return nil, err
		}
		provider = conn.Provider
		raw, derr := crypto.Decrypt(conn.EncryptedSecret, s.encryptionKey)
		if derr != nil {
			_ = s.repo.MarkStatus(ctx, projectID, model.GitConnStatusInvalid, strptr(model.ValidationErrDecryptFailed))
			return nil, errors.NewInvalidState(CodeGitConnectionInvalid, "stored token could not be decrypted (rotate ENCRYPTION_KEY => re-enter token)")
		}
		token = string(raw)
	} else {
		token = strings.TrimSpace(*bodyToken)
		if p, err := s.projectRepo.GetByID(ctx, projectID); err == nil {
			provider = normalizeProvider(p.GitProvider)
		}
	}

	baseURL := s.providerBaseURL(ctx, projectID, provider)
	result, perr := s.validator.Probe(ctx, provider, baseURL, token)
	if perr != nil {
		if testingStored {
			s.reconcileStoredFromProbe(ctx, projectID, perr)
		}
		return nil, s.mapProbeError(perr)
	}

	status, missing := s.statusFromProbe(result)
	if testingStored {
		if status == model.GitConnStatusConnected {
			_ = s.repo.SetValidation(ctx, port.SetValidationParams{
				ProjectID: projectID, Status: status, AccountLogin: strptrIfNonEmpty(result.Login),
				Scopes: result.Scopes, ExpiresAt: result.ExpiresAt, ValidationError: nil,
			})
		} else {
			_ = s.repo.MarkStatus(ctx, projectID, status, strptr(model.ValidationErrInsufficientScope))
		}
	}

	if status == model.GitConnStatusInsufficient {
		s.audit(ctx, projectID, actorUserID, "tested", map[string]any{"status": string(status), "account_login": result.Login})
		return nil, errors.NewInvalidState(CodeGitConnectionInsufficientScope,
			"token is missing required scope(s): "+strings.Join(missing, ", "))
	}

	s.audit(ctx, projectID, actorUserID, "tested", map[string]any{"status": string(status), "account_login": result.Login})
	return &GitConnectionTestView{
		Ok:            true,
		Status:        status,
		AccountLogin:  strptrIfNonEmpty(result.Login),
		Scopes:        result.Scopes,
		MissingScopes: missing,
		TokenType:     result.TokenType,
		ExpiresAt:     result.ExpiresAt,
		Message:       testMessage(result),
	}, nil
}

// Clear deletes the stored connection (idempotent); resolution reverts to env.
func (s *GitConnectionService) Clear(ctx context.Context, projectID uuid.UUID, actorUserID uuid.UUID) error {
	if err := s.repo.Delete(ctx, projectID); err != nil {
		return err
	}
	s.audit(ctx, projectID, actorUserID, "cleared", map[string]any{})
	return nil
}

// ─── internals ──────────────────────────────────────────────────────────────────

type persistFields struct {
	status          model.GitConnectionStatus
	tokenType       *string
	scopes          []string
	accountLogin    *string
	expiresAt       *time.Time
	lastValidatedAt *time.Time
}

func (s *GitConnectionService) persist(ctx context.Context, projectID uuid.UUID, provider, token string, f persistFields) (*GitConnectionView, error) {
	enc, err := crypto.Encrypt([]byte(token), s.encryptionKey)
	if err != nil {
		return nil, errors.NewInternal("failed to encrypt git token", err)
	}
	conn, err := s.repo.Upsert(ctx, port.UpsertGitConnectionParams{
		ProjectID:       projectID,
		Provider:        provider,
		EncryptedSecret: enc,
		SecretLast4:     strptr(last4(token)),
		TokenType:       f.tokenType,
		Scopes:          f.scopes,
		Status:          f.status,
		AccountLogin:    f.accountLogin,
		ExpiresAt:       f.expiresAt,
		LastValidatedAt: f.lastValidatedAt,
		ValidationError: nil,
	})
	if err != nil {
		return nil, err
	}
	return s.viewFromConn(conn), nil
}

// reconcileStoredFromProbe flips the stored status for a definitive probe failure
// (401/403) and leaves transient failures untouched.
func (s *GitConnectionService) reconcileStoredFromProbe(ctx context.Context, projectID uuid.UUID, perr error) {
	var pe *port.ProbeError
	if !stderrors.As(perr, &pe) || pe.Kind.Transient() {
		return
	}
	switch pe.Kind {
	case port.ProbeUnauthorized:
		_ = s.repo.MarkStatus(ctx, projectID, model.GitConnStatusInvalid, strptr(model.ValidationErrUnauthorized))
	case port.ProbeForbidden:
		_ = s.repo.MarkStatus(ctx, projectID, model.GitConnStatusInsufficient, strptr(model.ValidationErrInsufficientScope))
	}
}

// mapProbeError converts a probe failure to the API-facing error: transient -> the
// 503 sentinel; 401 -> GIT_CONNECTION_INVALID; 403 -> GIT_CONNECTION_INSUFFICIENT_SCOPE.
func (s *GitConnectionService) mapProbeError(perr error) error {
	var pe *port.ProbeError
	if !stderrors.As(perr, &pe) {
		return errors.NewInvalidState(CodeGitConnectionInvalid, "connection probe failed")
	}
	if pe.Kind.Transient() {
		return ErrGitConnectionProbeUnavailable
	}
	switch pe.Kind {
	case port.ProbeUnauthorized:
		return errors.NewInvalidState(CodeGitConnectionInvalid, "the provider rejected this token")
	case port.ProbeForbidden:
		return errors.NewInvalidState(CodeGitConnectionInsufficientScope, "the token lacks a required permission")
	default:
		return errors.NewInvalidState(CodeGitConnectionInvalid, "connection probe failed")
	}
}

// statusFromProbe derives the connection status + missing scopes from a successful
// probe. Fine-grained PATs never hard-fail on scopes (no X-OAuth-Scopes header).
func (s *GitConnectionService) statusFromProbe(r port.ProbeResult) (model.GitConnectionStatus, []string) {
	if r.ExpiresAt != nil && r.ExpiresAt.Before(time.Now()) {
		return model.GitConnStatusExpired, nil
	}
	if r.TokenType == string(model.GitTokenTypeClassic) {
		if missing := missingClassicScopes(r.Scopes); len(missing) > 0 {
			return model.GitConnStatusInsufficient, missing
		}
	}
	return model.GitConnStatusConnected, nil
}

// providerBaseURL resolves the host the probe targets. For github it is "" (the
// validator uses its pinned operator base); for gitea it is derived from the
// project's repo_url. Any lookup/parse failure yields "" (probe-target pinning §5).
func (s *GitConnectionService) providerBaseURL(ctx context.Context, projectID uuid.UUID, provider string) string {
	if provider != providerGitea {
		return ""
	}
	p, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil || p.RepoURL == nil {
		return ""
	}
	parsed, perr := url.Parse(*p.RepoURL)
	if perr != nil || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func (s *GitConnectionService) unconfiguredView(ctx context.Context, projectID uuid.UUID) *GitConnectionView {
	provider := providerGitHub
	if p, err := s.projectRepo.GetByID(ctx, projectID); err == nil {
		provider = normalizeProvider(p.GitProvider)
	}
	source := "none"
	if tok := s.legacyEnvFallback(ctx, projectID); tok.Value != "" {
		source = "env"
	}
	return &GitConnectionView{
		Configured: false,
		Source:     source,
		Kind:       string(model.GitConnectionKindPAT),
		Provider:   provider,
		Status:     model.GitConnStatusUnconfigured,
	}
}

func (s *GitConnectionService) viewFromConn(c *model.GitConnection) *GitConnectionView {
	status := c.Status
	// Lazy expiry: a known past expiry resolves to "expired" without calling GitHub.
	if c.ExpiresAt != nil && c.ExpiresAt.Before(time.Now()) && status != model.GitConnStatusUnconfigured {
		status = model.GitConnStatusExpired
	}
	return &GitConnectionView{
		Configured:      true,
		Source:          "connection",
		Kind:            string(c.Kind),
		Provider:        c.Provider,
		Status:          status,
		SecretLast4:     c.SecretLast4,
		TokenType:       c.TokenType,
		AccountLogin:    c.AccountLogin,
		Scopes:          c.Scopes,
		ExpiresAt:       c.ExpiresAt,
		LastValidatedAt: c.LastValidatedAt,
		ValidationError: c.ValidationError,
	}
}

// audit publishes a redacted git_connection.<action> event. The token is NEVER part
// of the payload. Best-effort: a publish failure is logged, never fatal.
func (s *GitConnectionService) audit(ctx context.Context, projectID, actorUserID uuid.UUID, action string, fields map[string]any) {
	if s.events == nil {
		return
	}
	fields["user_id"] = actorUserID.String()
	fields["project_id"] = projectID.String()
	payload, err := json.Marshal(fields)
	if err != nil {
		s.logger.Warn("failed to marshal git connection audit payload", "project_id", projectID, "error", err)
		return
	}
	if perr := s.events.Publish(ctx, model.Event{
		ID:         uuid.New(),
		ProjectID:  projectID,
		EntityType: "git_connection",
		EntityID:   projectID,
		Action:     action,
		Payload:    payload,
	}); perr != nil {
		s.logger.Warn("failed to publish git connection event", "project_id", projectID, "action", action, "error", perr)
	}
}

// ─── pure helpers ─────────────────────────────────────────────────────────────────

func classifyToken(token string) string {
	switch {
	case strings.HasPrefix(token, "github_pat_"):
		return string(model.GitTokenTypeFineGrained)
	case strings.HasPrefix(token, "ghp_"):
		return string(model.GitTokenTypeClassic)
	default:
		return string(model.GitTokenTypeUnknown)
	}
}

func missingClassicScopes(granted []string) []string {
	have := make(map[string]bool, len(granted))
	for _, g := range granted {
		have[strings.TrimSpace(g)] = true
	}
	var missing []string
	for _, req := range requiredClassicScopes {
		// "project" (read+write) is a superset of "read:project".
		if have[req] || (req == "read:project" && have["project"]) {
			continue
		}
		missing = append(missing, req)
	}
	return missing
}

// classifyOperationError inspects a real git/planning operation error for a
// definitive auth failure. Transient signals (429/5xx) suppress any flip.
func classifyOperationError(err error) (model.GitConnectionStatus, string, bool) {
	msg := strings.ToLower(err.Error())
	if hasHTTPStatus(msg, "429") || strings.Contains(msg, "rate limit") ||
		hasHTTPStatus(msg, "500") || hasHTTPStatus(msg, "502") ||
		hasHTTPStatus(msg, "503") || hasHTTPStatus(msg, "504") {
		return "", "", false // transient: leave status as-is
	}
	if hasHTTPStatus(msg, "401") || strings.Contains(msg, "bad credentials") ||
		strings.Contains(msg, "requires authentication") || strings.Contains(msg, "unauthorized") {
		return model.GitConnStatusInvalid, model.ValidationErrUnauthorized, true
	}
	if hasHTTPStatus(msg, "403") || strings.Contains(msg, "forbidden") {
		return model.GitConnStatusInsufficient, model.ValidationErrInsufficientScope, true
	}
	return "", "", false
}

// hasHTTPStatus reports whether msg contains code as a standalone status token,
// reducing false positives from unrelated numbers (SHAs, PR ids).
func hasHTTPStatus(msg, code string) bool {
	for _, sep := range []string{" " + code + " ", " " + code, ":" + code, "status " + code, "status:" + code, "code " + code} {
		if strings.Contains(msg, sep) {
			return true
		}
	}
	return strings.HasSuffix(msg, " "+code)
}

func testMessage(r port.ProbeResult) string {
	if r.TokenType == string(model.GitTokenTypeFineGrained) {
		return "fine-grained PAT accepted; note that user-owned (personal) Projects v2 boards may be unreadable"
	}
	return "connection verified"
}

func normalizeProvider(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case providerGitea:
		return providerGitea
	default:
		return providerGitHub
	}
}

func last4(s string) string {
	if len(s) <= 4 {
		return s
	}
	return s[len(s)-4:]
}

func strptr(s string) *string { return &s }

func strptrIfNonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefStrOr(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}
