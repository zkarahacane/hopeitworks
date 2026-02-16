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

func (m *mockRepo) Create(ctx context.Context, user *model.User) (*model.User, error) {
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

func (m *mockRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, errors.New("no rows")
	}
	return u, nil
}

func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	u, ok := m.users[id.String()]
	if !ok {
		return nil, errors.New("no rows")
	}
	return u, nil
}

func (m *mockRepo) List(ctx context.Context, limit, offset int32) ([]*model.User, error) {
	return nil, nil
}

func (m *mockRepo) Update(ctx context.Context, user *model.User) (*model.User, error) {
	return nil, nil
}

func (m *mockRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

type pgDupError struct{}

func (e *pgDupError) Error() string    { return "duplicate key" }
func (e *pgDupError) SQLState() string { return "23505" }

func newTestHandler() (*AuthHandler, *mockRepo) {
	repo := newMockRepo()
	authSvc := service.NewAuthService(repo, "test-secret-key", 24*time.Hour)
	handler := NewAuthHandler(authSvc, repo, false)
	return handler, repo
}

func TestRegisterHandler_Success(t *testing.T) {
	h, _ := newTestHandler()

	body := `{"email":"test@example.com","password":"secureP@ss1","name":"Test User"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
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
		if c.Name == "token" && c.Value != "" {
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

	body := `{"email":"test@example.com","password":"secureP@ss1","name":"Test User"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Register(rec, req)

	// Register same email again
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
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
	body := `{"email":"test@example.com","password":"secureP@ss1","name":"Test User"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
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
		if c.Name == "token" && c.Value != "" {
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
	body := `{"email":"test@example.com","password":"secureP@ss1","name":"Test User"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
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
		if c.Name == "token" && c.MaxAge < 0 {
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
	body := `{"email":"test@example.com","password":"secureP@ss1","name":"Test User"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Register(rec, req)

	var regResp userResponse
	json.NewDecoder(rec.Body).Decode(&regResp)
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
	json.NewDecoder(rec.Body).Decode(&meResp)
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
