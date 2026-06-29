package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

const testMasterKey = "test-master-key-please-rotate-32!"

// ─── mocks ──────────────────────────────────────────────────────────────────────

type gcRepo struct {
	conns      map[uuid.UUID]*model.GitConnection
	upsertErr  error
	getErr     error
	markCalls  int
	lastStatus model.GitConnectionStatus
}

func newGCRepo() *gcRepo { return &gcRepo{conns: map[uuid.UUID]*model.GitConnection{}} }

func (r *gcRepo) GetByProject(_ context.Context, projectID uuid.UUID) (*model.GitConnection, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	c, ok := r.conns[projectID]
	if !ok {
		return nil, apperrors.NewNotFound("git_connection", projectID)
	}
	return c, nil
}

func (r *gcRepo) Upsert(_ context.Context, p port.UpsertGitConnectionParams) (*model.GitConnection, error) {
	if r.upsertErr != nil {
		return nil, r.upsertErr
	}
	c := &model.GitConnection{
		ID:              uuid.New(),
		ProjectID:       p.ProjectID,
		Provider:        p.Provider,
		Kind:            model.GitConnectionKindPAT,
		EncryptedSecret: p.EncryptedSecret,
		SecretLast4:     p.SecretLast4,
		TokenType:       p.TokenType,
		Scopes:          p.Scopes,
		Status:          p.Status,
		AccountLogin:    p.AccountLogin,
		ExpiresAt:       p.ExpiresAt,
		LastValidatedAt: p.LastValidatedAt,
		ValidationError: p.ValidationError,
	}
	r.conns[p.ProjectID] = c
	return c, nil
}

func (r *gcRepo) SetValidation(_ context.Context, p port.SetValidationParams) error {
	if c, ok := r.conns[p.ProjectID]; ok {
		c.Status = p.Status
		c.AccountLogin = p.AccountLogin
		c.Scopes = p.Scopes
		c.ExpiresAt = p.ExpiresAt
		c.ValidationError = p.ValidationError
	}
	return nil
}

func (r *gcRepo) MarkStatus(_ context.Context, projectID uuid.UUID, status model.GitConnectionStatus, ve *string) error {
	r.markCalls++
	r.lastStatus = status
	if c, ok := r.conns[projectID]; ok {
		c.Status = status
		c.ValidationError = ve
	}
	return nil
}

func (r *gcRepo) Delete(_ context.Context, projectID uuid.UUID) error {
	delete(r.conns, projectID)
	return nil
}

// gcValidator is a programmable port.GitConnectionValidator.
type gcValidator struct {
	result port.ProbeResult
	err    error
	calls  int
}

func (v *gcValidator) Probe(_ context.Context, _, _, _ string) (port.ProbeResult, error) {
	v.calls++
	return v.result, v.err
}

func newGCService(t *testing.T, repo *gcRepo, projRepo *mockProjectRepo, val port.GitConnectionValidator, masterKey string) *GitConnectionService {
	t.Helper()
	return NewGitConnectionService(repo, projRepo, val, masterKey, newCBMockEventPublisher(), testLogger())
}

func projectWithOwner(t *testing.T, repo *mockProjectRepo, provider string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	owner := uuid.New()
	repo.projects[id] = &model.Project{ID: id, OwnerID: &owner, GitProvider: provider}
	return id
}

// ─── resolver / dispatch ──────────────────────────────────────────────────────────

func TestTokenForProject_PATDispatch_RoundTrip(t *testing.T) {
	repo := newGCRepo()
	proj := newMockProjectRepoForService()
	val := &gcValidator{result: port.ProbeResult{Login: "octocat", TokenType: "classic", Scopes: []string{"read:project"}}}
	svc := newGCService(t, repo, proj, val, testMasterKey)
	pid := projectWithOwner(t, proj, "github")

	// Store via Set (validate path) then resolve the decrypted value back.
	if _, err := svc.Set(context.Background(), pid, uuid.New(), SetGitConnectionInput{Provider: "github", Token: "ghp_classicTOKENvalue1234567890", Validate: true}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	tok, err := svc.TokenForProject(context.Background(), pid)
	if err != nil {
		t.Fatalf("TokenForProject: %v", err)
	}
	if tok.Value != "ghp_classicTOKENvalue1234567890" {
		t.Fatalf("decrypt round-trip mismatch: got %q", tok.Value)
	}
}

func TestTokenForProject_LegacyEnvFallback_NoRow(t *testing.T) {
	repo := newGCRepo()
	proj := newMockProjectRepoForService()
	svc := newGCService(t, repo, proj, &gcValidator{}, testMasterKey)
	pid := projectWithOwner(t, proj, "github")

	t.Setenv("GITHUB_TOKEN", "env-fallback-token")
	tok, err := svc.TokenForProject(context.Background(), pid)
	if err != nil {
		t.Fatalf("TokenForProject: %v", err)
	}
	if tok.Value != "env-fallback-token" {
		t.Fatalf("expected env fallback, got %q", tok.Value)
	}
}

func TestTokenForProject_DecryptFailFallsBackToEnv_B2(t *testing.T) {
	repo := newGCRepo()
	proj := newMockProjectRepoForService()
	svc := newGCService(t, repo, proj, &gcValidator{}, testMasterKey)
	pid := projectWithOwner(t, proj, "github")

	// Stored ciphertext was encrypted under a DIFFERENT key (rotation simulation).
	repo.conns[pid] = &model.GitConnection{
		ProjectID:       pid,
		Kind:            model.GitConnectionKindPAT,
		EncryptedSecret: []byte("not-decryptable-under-this-key"),
		Status:          model.GitConnStatusConnected,
	}
	t.Setenv("GITHUB_TOKEN", "env-after-rotation")

	tok, err := svc.TokenForProject(context.Background(), pid)
	if err != nil {
		t.Fatalf("expected no hard error on decrypt failure, got %v", err)
	}
	if tok.Value != "env-after-rotation" {
		t.Fatalf("expected env fallback after decrypt failure, got %q", tok.Value)
	}
	if repo.lastStatus != model.GitConnStatusInvalid {
		t.Fatalf("expected status flipped to invalid, got %q", repo.lastStatus)
	}
}

// ─── validate-before-store + transient + scope ─────────────────────────────────────

func TestSet_ValidateBeforeStore_401NotPersisted(t *testing.T) {
	repo := newGCRepo()
	proj := newMockProjectRepoForService()
	val := &gcValidator{err: &port.ProbeError{Kind: port.ProbeUnauthorized}}
	svc := newGCService(t, repo, proj, val, testMasterKey)
	pid := projectWithOwner(t, proj, "github")

	_, err := svc.Set(context.Background(), pid, uuid.New(), SetGitConnectionInput{Token: "ghp_xxxxxxxxxxxxxxxxxxxxxxxx", Validate: true})
	var de *apperrors.DomainError
	if !errors.As(err, &de) || de.Code != CodeGitConnectionInvalid {
		t.Fatalf("expected GIT_CONNECTION_INVALID, got %v", err)
	}
	if _, ok := repo.conns[pid]; ok {
		t.Fatal("invalid token must NOT be persisted")
	}
}

func TestSet_403InsufficientScope(t *testing.T) {
	repo := newGCRepo()
	proj := newMockProjectRepoForService()
	val := &gcValidator{err: &port.ProbeError{Kind: port.ProbeForbidden}}
	svc := newGCService(t, repo, proj, val, testMasterKey)
	pid := projectWithOwner(t, proj, "github")

	_, err := svc.Set(context.Background(), pid, uuid.New(), SetGitConnectionInput{Token: "ghp_xxxxxxxxxxxxxxxxxxxxxxxx", Validate: true})
	var de *apperrors.DomainError
	if !errors.As(err, &de) || de.Code != CodeGitConnectionInsufficientScope {
		t.Fatalf("expected GIT_CONNECTION_INSUFFICIENT_SCOPE, got %v", err)
	}
}

func TestSet_ClassicMissingReadProject_Insufficient(t *testing.T) {
	repo := newGCRepo()
	proj := newMockProjectRepoForService()
	val := &gcValidator{result: port.ProbeResult{Login: "octocat", TokenType: "classic", Scopes: []string{"repo"}}}
	svc := newGCService(t, repo, proj, val, testMasterKey)
	pid := projectWithOwner(t, proj, "github")

	_, err := svc.Set(context.Background(), pid, uuid.New(), SetGitConnectionInput{Token: "ghp_xxxxxxxxxxxxxxxxxxxxxxxx", Validate: true})
	var de *apperrors.DomainError
	if !errors.As(err, &de) || de.Code != CodeGitConnectionInsufficientScope {
		t.Fatalf("expected insufficient scope for classic missing read:project, got %v", err)
	}
}

func TestSet_FineGrainedNeverHardFailsScope(t *testing.T) {
	repo := newGCRepo()
	proj := newMockProjectRepoForService()
	// fine-grained: no X-OAuth-Scopes header -> empty scopes, must NOT fail.
	val := &gcValidator{result: port.ProbeResult{Login: "octocat", TokenType: "fine_grained"}}
	svc := newGCService(t, repo, proj, val, testMasterKey)
	pid := projectWithOwner(t, proj, "github")

	view, err := svc.Set(context.Background(), pid, uuid.New(), SetGitConnectionInput{Token: "github_pat_abc123", Validate: true})
	if err != nil {
		t.Fatalf("fine-grained PAT must not hard-fail on scopes: %v", err)
	}
	if view.Status != model.GitConnStatusConnected {
		t.Fatalf("expected connected, got %q", view.Status)
	}
}

func TestSet_TransientProbe_ReturnsSentinel_DoesNotOverwrite(t *testing.T) {
	repo := newGCRepo()
	proj := newMockProjectRepoForService()
	pid := projectWithOwner(t, proj, "github")
	// pre-existing good row
	repo.conns[pid] = &model.GitConnection{ProjectID: pid, Kind: model.GitConnectionKindPAT, Status: model.GitConnStatusConnected, EncryptedSecret: []byte("x")}

	val := &gcValidator{err: &port.ProbeError{Kind: port.Probe5xx}}
	svc := newGCService(t, repo, proj, val, testMasterKey)

	_, err := svc.Set(context.Background(), pid, uuid.New(), SetGitConnectionInput{Token: "ghp_xxxxxxxxxxxxxxxxxxxxxxxx", Validate: true})
	if !errors.Is(err, ErrGitConnectionProbeUnavailable) {
		t.Fatalf("expected transient sentinel, got %v", err)
	}
	if repo.conns[pid].Status != model.GitConnStatusConnected {
		t.Fatal("transient probe must not overwrite a good row")
	}
	if !bytesEqual(repo.conns[pid].EncryptedSecret, []byte("x")) {
		t.Fatal("transient probe must not overwrite the stored secret")
	}
}

func TestSet_ValidateFalse_PersistsUnverified(t *testing.T) {
	repo := newGCRepo()
	proj := newMockProjectRepoForService()
	val := &gcValidator{}
	svc := newGCService(t, repo, proj, val, testMasterKey)
	pid := projectWithOwner(t, proj, "github")

	view, err := svc.Set(context.Background(), pid, uuid.New(), SetGitConnectionInput{Token: "ghp_xxxxxxxxxxxxxxxxxxxxxxxx", Validate: false})
	if err != nil {
		t.Fatalf("Set(validate=false): %v", err)
	}
	if val.calls != 0 {
		t.Fatal("validate=false must NOT probe")
	}
	if view.Status != model.GitConnStatusUnconfigured {
		t.Fatalf("expected unconfigured (unverified), got %q", view.Status)
	}
	if view.LastValidatedAt != nil {
		t.Fatal("validate=false must leave last_validated_at NULL")
	}
}

// ─── B1 key-unset guard ────────────────────────────────────────────────────────────

func TestSet_KeyUnset_RejectsPUT_B1(t *testing.T) {
	for _, key := range []string{"", "dev-encryption-key-32bytes!!"} {
		repo := newGCRepo()
		proj := newMockProjectRepoForService()
		svc := newGCService(t, repo, proj, &gcValidator{}, key)
		pid := projectWithOwner(t, proj, "github")

		_, err := svc.Set(context.Background(), pid, uuid.New(), SetGitConnectionInput{Token: "ghp_xxxxxxxxxxxxxxxxxxxxxxxx", Validate: true})
		var de *apperrors.DomainError
		if !errors.As(err, &de) || de.Code != CodeGitConnectionKeyUnset {
			t.Fatalf("key %q: expected GIT_CONNECTION_KEY_UNSET, got %v", key, err)
		}
		if _, ok := repo.conns[pid]; ok {
			t.Fatalf("key %q: must not persist under an insecure key", key)
		}
	}
}

// ─── C1 live self-heal ─────────────────────────────────────────────────────────────

func TestReconcileFromOperationError_Flips(t *testing.T) {
	cases := []struct {
		name   string
		errMsg string
		want   model.GitConnectionStatus
		flip   bool
	}{
		{"401 -> invalid", "GET https://api.github.com/repos/x: 401 Bad credentials", model.GitConnStatusInvalid, true},
		{"403 -> insufficient", "POST .../merge: 403 Forbidden (resource not accessible)", model.GitConnStatusInsufficient, true},
		{"429 -> no flip", "GET ...: 429 rate limit exceeded", "", false},
		{"503 -> no flip", "GET ...: 503 service unavailable", "", false},
		{"merge conflict -> no flip", "merge conflict detected: 409", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newGCRepo()
			proj := newMockProjectRepoForService()
			svc := newGCService(t, repo, proj, &gcValidator{}, testMasterKey)
			pid := uuid.New()
			repo.conns[pid] = &model.GitConnection{ProjectID: pid, Status: model.GitConnStatusConnected}

			svc.ReconcileFromOperationError(context.Background(), pid, errors.New(tc.errMsg))

			if tc.flip {
				if repo.conns[pid].Status != tc.want {
					t.Fatalf("expected flip to %q, got %q", tc.want, repo.conns[pid].Status)
				}
			} else if repo.conns[pid].Status != model.GitConnStatusConnected {
				t.Fatalf("expected no flip, status became %q", repo.conns[pid].Status)
			}
		})
	}
}

// ─── lazy expiry ───────────────────────────────────────────────────────────────────

func TestStatus_LazyExpiry(t *testing.T) {
	repo := newGCRepo()
	proj := newMockProjectRepoForService()
	svc := newGCService(t, repo, proj, &gcValidator{}, testMasterKey)
	pid := projectWithOwner(t, proj, "github")

	past := time.Now().Add(-time.Hour)
	repo.conns[pid] = &model.GitConnection{
		ProjectID: pid, Kind: model.GitConnectionKindPAT, Provider: "github",
		Status: model.GitConnStatusConnected, ExpiresAt: &past, EncryptedSecret: []byte("x"),
	}

	view, err := svc.Status(context.Background(), pid)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if view.Status != model.GitConnStatusExpired {
		t.Fatalf("expected lazy expiry -> expired, got %q", view.Status)
	}
}

func TestStatus_NoRow_EnvSource(t *testing.T) {
	repo := newGCRepo()
	proj := newMockProjectRepoForService()
	svc := newGCService(t, repo, proj, &gcValidator{}, testMasterKey)
	pid := projectWithOwner(t, proj, "github")

	t.Setenv("GITHUB_TOKEN", "env-token")
	view, err := svc.Status(context.Background(), pid)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if view.Configured {
		t.Fatal("expected configured=false with no row")
	}
	if view.Source != "env" {
		t.Fatalf("expected source=env, got %q", view.Source)
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────────

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// last4 sanity (display hint never exposes more than 4 chars).
func TestLast4(t *testing.T) {
	if got := last4("ghp_abcd1234"); got != "1234" {
		t.Fatalf("last4 mismatch: %q", got)
	}
	if got := last4("ab"); got != "ab" {
		t.Fatalf("last4 short mismatch: %q", got)
	}
	if strings.Contains(last4("ghp_secretlongtoken9999"), "secret") {
		t.Fatal("last4 leaked more than the suffix")
	}
}
