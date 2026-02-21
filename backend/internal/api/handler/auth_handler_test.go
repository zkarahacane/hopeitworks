package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

// mockRepo is a test double for port.UserRepository.
type mockRepo struct {
	users   map[string]*model.User
	byEmail map[string]*model.User
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		users:   make(map[string]*model.User),
		byEmail: make(map[string]*model.User),
	}
}

func (m *mockRepo) Create(_ context.Context, user *model.User) (*model.User, error) {
	if _, exists := m.byEmail[user.Email]; exists {
		return nil, &pgDupError{}
	}
	user.ID = uuid.New()
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	m.users[user.ID.String()] = user
	m.byEmail[user.Email] = user
	return user, nil
}

func (m *mockRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, errors.New("no rows")
	}
	return u, nil
}

func (m *mockRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	u, ok := m.users[id.String()]
	if !ok {
		return nil, errors.New("no rows")
	}
	return u, nil
}

func (m *mockRepo) List(_ context.Context, _, _ int32) ([]*model.User, error) {
	return nil, nil
}

func (m *mockRepo) Count(_ context.Context) (int64, error) {
	return int64(len(m.users)), nil
}

func (m *mockRepo) Update(_ context.Context, user *model.User) (*model.User, error) {
	existing, ok := m.users[user.ID.String()]
	if !ok {
		return nil, errors.New("no rows")
	}
	if user.PasswordHash != "" {
		existing.PasswordHash = user.PasswordHash
	}
	existing.UpdatedAt = time.Now()
	return existing, nil
}

func (m *mockRepo) UpdatePasswordHash(_ context.Context, _ uuid.UUID, _ string) error { return nil }

func (m *mockRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

type pgDupError struct{}

func (e *pgDupError) Error() string    { return "duplicate key" }
func (e *pgDupError) SQLState() string { return "23505" }

// mockTokenRepo is a test double for port.PasswordResetTokenRepository.
type mockTokenRepo struct {
	tokens map[string]*model.PasswordResetToken
}

func newMockTokenRepo() *mockTokenRepo {
	return &mockTokenRepo{tokens: make(map[string]*model.PasswordResetToken)}
}

func (m *mockTokenRepo) Create(_ context.Context, userID uuid.UUID, token string, expiresAt time.Time) (*model.PasswordResetToken, error) {
	prt := &model.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	m.tokens[token] = prt
	return prt, nil
}

func (m *mockTokenRepo) GetByToken(_ context.Context, token string) (*model.PasswordResetToken, error) {
	prt, ok := m.tokens[token]
	if !ok {
		return nil, errors.New("not found")
	}
	return prt, nil
}

func (m *mockTokenRepo) MarkUsed(_ context.Context, id uuid.UUID) error {
	for _, prt := range m.tokens {
		if prt.ID == id {
			now := time.Now()
			prt.UsedAt = &now
			return nil
		}
	}
	return errors.New("not found")
}

// mockEmailSender is a test double for port.EmailSender.
type mockEmailSender struct {
	sendFn func(ctx context.Context, msg port.EmailMessage) error
}

func newMockEmailSender() *mockEmailSender {
	return &mockEmailSender{}
}

func (m *mockEmailSender) Send(ctx context.Context, msg port.EmailMessage) error {
	if m.sendFn != nil {
		return m.sendFn(ctx, msg)
	}
	return nil
}

func newTestHandler() (*AuthHandler, *mockRepo) {
	repo := newMockRepo()
	tokenRepo := newMockTokenRepo()
	emailSender := newMockEmailSender()
	authSvc := service.NewAuthService(repo, tokenRepo, emailSender, "http://localhost:5173", "test-secret-key", 24*time.Hour)
	h := NewAuthHandler(authSvc, repo, false)
	return h, repo
}

func newTestHandlerWithDeps() (*AuthHandler, *mockRepo, *mockTokenRepo, *mockEmailSender) {
	repo := newMockRepo()
	tokenRepo := newMockTokenRepo()
	emailSender := newMockEmailSender()
	authSvc := service.NewAuthService(repo, tokenRepo, emailSender, "http://localhost:5173", "test-secret-key", 24*time.Hour)
	h := NewAuthHandler(authSvc, repo, false)
	return h, repo, tokenRepo, emailSender
}

const (
	testRegisterBody   = `{"email":"test@example.com","password":"secureP@ss1","name":"Test User"}`
	testTokenCookieKey = "token"
)

func TestRegisterHandler_Success(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(testRegisterBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}

	var resp userResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", resp.Email)
	}
	if resp.Name != "Test User" {
		t.Errorf("expected name Test User, got %s", resp.Name)
	}

	// Check cookie is set
	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == testTokenCookieKey && c.Value != "" {
			found = true
			if !c.HttpOnly {
				t.Error("cookie should be httpOnly")
			}
			if c.Path != "/api" {
				t.Errorf("expected cookie path /api, got %s", c.Path)
			}
		}
	}
	if !found {
		t.Error("expected token cookie to be set")
	}
}

func TestRegisterHandler_DuplicateEmail(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(testRegisterBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Register(rec, req)

	// Register same email again
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(testRegisterBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.Register(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rec.Code)
	}
}

func TestRegisterHandler_ValidationError(t *testing.T) {
	h, _ := newTestHandler()

	tests := []struct {
		name string
		body string
		code int
	}{
		{"missing fields", `{}`, http.StatusBadRequest},
		{"short password", `{"email":"a@b.com","password":"short","name":"T"}`, http.StatusBadRequest},
		{"invalid json", `{invalid`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.Register(rec, req)

			if rec.Code != tt.code {
				t.Errorf("expected %d, got %d", tt.code, rec.Code)
			}
		})
	}
}

func TestLoginHandler_Success(t *testing.T) {
	h, _ := newTestHandler()

	// Register first
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(testRegisterBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Register(rec, req)

	// Login
	loginBody := `{"email":"test@example.com","password":"secureP@ss1"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == testTokenCookieKey && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected token cookie on login")
	}
}

func TestLoginHandler_WrongPassword(t *testing.T) {
	h, _ := newTestHandler()

	// Register
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(testRegisterBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Register(rec, req)

	// Login with wrong password
	loginBody := `{"email":"test@example.com","password":"wrongpassword"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestLogoutHandler(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rec := httptest.NewRecorder()
	h.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}

	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == testTokenCookieKey && c.MaxAge < 0 {
			found = true
		}
	}
	if !found {
		t.Error("expected token cookie to be cleared")
	}
}

func TestMeHandler_Authenticated(t *testing.T) {
	h, _ := newTestHandler()

	// Register to create a user
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(testRegisterBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Register(rec, req)

	var regResp userResponse
	if err := json.NewDecoder(rec.Body).Decode(&regResp); err != nil {
		t.Fatalf("failed to decode register response: %v", err)
	}
	userID, _ := uuid.Parse(regResp.ID)

	// Call Me with context set (simulating middleware)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, userID)
	ctx = context.WithValue(ctx, middleware.ContextKeyRole, model.RoleUser)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	h.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var meResp userResponse
	if err := json.NewDecoder(rec.Body).Decode(&meResp); err != nil {
		t.Fatalf("failed to decode me response: %v", err)
	}
	if meResp.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", meResp.Email)
	}
}

func TestMeHandler_Unauthenticated(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()
	h.Me(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// ── ForgotPassword handler tests ────────────────────────────────────────

func TestForgotPasswordHandler_ValidEmail(t *testing.T) {
	h, _, _, _ := newTestHandlerWithDeps()

	// Register a user first
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(testRegisterBody))
	regReq.Header.Set("Content-Type", "application/json")
	regRec := httptest.NewRecorder()
	h.Register(regRec, regReq)

	// Request forgot password
	body := `{"email":"test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ForgotPassword(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rec.Code)
	}

	var resp messageResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Message != "If this email is registered, a reset link has been sent" {
		t.Errorf("unexpected message: %s", resp.Message)
	}
}

func TestForgotPasswordHandler_UnknownEmail(t *testing.T) {
	h, _ := newTestHandler()

	body := `{"email":"unknown@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ForgotPassword(rec, req)

	// Must still return 202 (no enumeration)
	if rec.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rec.Code)
	}
}

func TestForgotPasswordHandler_MissingEmail(t *testing.T) {
	h, _ := newTestHandler()

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ForgotPassword(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// ── ResetPassword handler tests ─────────────────────────────────────────

func TestResetPasswordHandler_ValidToken(t *testing.T) {
	h, _, tokenRepo, _ := newTestHandlerWithDeps()

	// Register a user
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(testRegisterBody))
	regReq.Header.Set("Content-Type", "application/json")
	regRec := httptest.NewRecorder()
	h.Register(regRec, regReq)

	var regResp userResponse
	if err := json.NewDecoder(regRec.Body).Decode(&regResp); err != nil {
		t.Fatalf("failed to decode register response: %v", err)
	}
	userID, _ := uuid.Parse(regResp.ID)

	// Create a token directly in the mock
	prt := &model.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     "valid-reset-token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}
	tokenRepo.tokens["valid-reset-token"] = prt

	// Reset password
	body := `{"token":"valid-reset-token","password":"newSecureP@ss1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ResetPassword(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", rec.Code, rec.Body.String())
	}
}

func TestResetPasswordHandler_ExpiredToken(t *testing.T) {
	h, _, tokenRepo, _ := newTestHandlerWithDeps()

	prt := &model.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Token:     "expired-token",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
	tokenRepo.tokens["expired-token"] = prt

	body := `{"token":"expired-token","password":"newSecureP@ss1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ResetPassword(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}

	var resp errorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if resp.Error.Code != "RESET_TOKEN_EXPIRED" {
		t.Errorf("expected error code RESET_TOKEN_EXPIRED, got %s", resp.Error.Code)
	}
}

func TestResetPasswordHandler_InvalidToken(t *testing.T) {
	h, _ := newTestHandler()

	body := `{"token":"nonexistent-token","password":"newSecureP@ss1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ResetPassword(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}

	var resp errorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if resp.Error.Code != "RESET_TOKEN_INVALID" {
		t.Errorf("expected error code RESET_TOKEN_INVALID, got %s", resp.Error.Code)
	}
}

func TestResetPasswordHandler_WeakPassword(t *testing.T) {
	h, _ := newTestHandler()

	body := `{"token":"some-token","password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ResetPassword(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestResetPasswordHandler_MissingFields(t *testing.T) {
	h, _ := newTestHandler()

	tests := []struct {
		name string
		body string
	}{
		{"missing token", `{"password":"newSecureP@ss1"}`},
		{"missing password", `{"token":"some-token"}`},
		{"empty body", `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.ResetPassword(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", rec.Code)
			}
		})
	}
}
