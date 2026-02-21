package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

func newTestAuthService() *service.AuthService {
	return service.NewAuthService(&noopRepo{}, &noopTokenRepo{}, &noopEmailSender{}, "http://localhost:5173", "test-secret-key", 24*time.Hour)
}

// noopRepo is a minimal mock repo just for token generation.
type noopRepo struct{}

func (r *noopRepo) Create(_ context.Context, user *model.User) (*model.User, error) {
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	return user, nil
}
func (r *noopRepo) GetByEmail(_ context.Context, _ string) (*model.User, error) {
	return nil, nil
}
func (r *noopRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	return nil, nil
}
func (r *noopRepo) List(_ context.Context, _, _ int32) ([]*model.User, error) {
	return nil, nil
}
func (r *noopRepo) Count(_ context.Context) (int64, error) { return 0, nil }
func (r *noopRepo) Update(_ context.Context, _ *model.User) (*model.User, error) {
	return nil, nil
}
func (r *noopRepo) UpdatePasswordHash(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (r *noopRepo) Delete(_ context.Context, _ uuid.UUID) error                       { return nil }

type noopTokenRepo struct{}

func (r *noopTokenRepo) Create(_ context.Context, _ uuid.UUID, _ string, _ time.Time) (*model.PasswordResetToken, error) {
	return nil, nil
}
func (r *noopTokenRepo) GetByToken(_ context.Context, _ string) (*model.PasswordResetToken, error) {
	return nil, nil
}
func (r *noopTokenRepo) MarkUsed(_ context.Context, _ uuid.UUID) error { return nil }

type noopEmailSender struct{}

func (s *noopEmailSender) Send(_ context.Context, _ port.EmailMessage) error { return nil }

func TestAuthMiddleware_ValidToken(t *testing.T) {
	authSvc := newTestAuthService()

	user, token, err := authSvc.Register(context.Background(), "test@example.com", "secureP@ss1", "Test User")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	var capturedUserID uuid.UUID
	var capturedRole model.Role
	var userIDFound, roleFound bool

	handler := Auth(authSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID, userIDFound = UserIDFromContext(r.Context())
		capturedRole, roleFound = RoleFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !userIDFound {
		t.Error("user ID not found in context")
	}
	if capturedUserID != user.ID {
		t.Errorf("expected user ID %s, got %s", user.ID, capturedUserID)
	}
	if !roleFound {
		t.Error("role not found in context")
	}
	if capturedRole != model.RoleUser {
		t.Errorf("expected role user, got %s", capturedRole)
	}
}

func TestAuthMiddleware_NoCookie(t *testing.T) {
	authSvc := newTestAuthService()

	handler := Auth(authSvc)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("handler should not be called when no cookie is present")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	authSvc := newTestAuthService()

	handler := Auth(authSvc)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("handler should not be called with invalid token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "invalid-token-value"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	// Create a service with negative expiration for immediately expired tokens
	expiredSvc := service.NewAuthService(&noopRepo{}, &noopTokenRepo{}, &noopEmailSender{}, "http://localhost:5173", "test-secret-key", -1*time.Hour)

	_, token, err := expiredSvc.Register(context.Background(), "test@example.com", "secureP@ss1", "Test")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Use the same secret for validation
	authSvc := newTestAuthService()

	handler := Auth(authSvc)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("handler should not be called with expired token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()

	// Empty context should return false
	_, ok := UserIDFromContext(ctx)
	if ok {
		t.Error("expected false for empty context user ID")
	}

	_, ok = RoleFromContext(ctx)
	if ok {
		t.Error("expected false for empty context role")
	}

	// Context with values should return true
	id := uuid.New()
	ctx = context.WithValue(ctx, ContextKeyUserID, id)
	ctx = context.WithValue(ctx, ContextKeyRole, model.RoleAdmin)

	gotID, ok := UserIDFromContext(ctx)
	if !ok || gotID != id {
		t.Errorf("expected user ID %s, got %s (ok=%v)", id, gotID, ok)
	}

	gotRole, ok := RoleFromContext(ctx)
	if !ok || gotRole != model.RoleAdmin {
		t.Errorf("expected role admin, got %s (ok=%v)", gotRole, ok)
	}
}
