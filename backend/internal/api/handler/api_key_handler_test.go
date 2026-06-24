package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockAPIKeyHandlerRepo is an in-memory port.APIKeyRepository for router-level tests.
type mockAPIKeyHandlerRepo struct {
	mu   sync.Mutex
	keys map[uuid.UUID]*model.UserAPIKey
}

func newMockAPIKeyHandlerRepo() *mockAPIKeyHandlerRepo {
	return &mockAPIKeyHandlerRepo{keys: make(map[uuid.UUID]*model.UserAPIKey)}
}

func (m *mockAPIKeyHandlerRepo) put(k *model.UserAPIKey) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keys[k.ID] = k
}

func (m *mockAPIKeyHandlerRepo) Create(_ context.Context, k *model.UserAPIKey) error {
	m.put(k)
	return nil
}

func (m *mockAPIKeyHandlerRepo) GetByID(_ context.Context, id uuid.UUID) (*model.UserAPIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k, ok := m.keys[id]
	if !ok {
		return nil, apperrors.NewNotFound("api_key", id)
	}
	return k, nil
}

func (m *mockAPIKeyHandlerRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]*model.UserAPIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*model.UserAPIKey
	for _, k := range m.keys {
		if k.UserID == userID {
			out = append(out, k)
		}
	}
	return out, nil
}

func (m *mockAPIKeyHandlerRepo) GetByUserAndProvider(_ context.Context, _ uuid.UUID, provider string) (*model.UserAPIKey, error) {
	return nil, apperrors.NewNotFound("api_key", provider)
}

func (m *mockAPIKeyHandlerRepo) Delete(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.keys, id)
	return nil
}

// newAPIKeyRouter mounts the generated mux for a Server wired only with the API
// key handler. When userID is non-nil, a middleware injects it into the request
// context, simulating the JWT auth middleware. This exercises the real routing
// path (generated wrapper -> *Server.DeleteMyAPIKey -> handler), which is what
// regressed in bug #288 (the route fell through to Unimplemented -> 501).
func newAPIKeyRouter(repo *mockAPIKeyHandlerRepo, userID *uuid.UUID) http.Handler {
	svc := service.NewAPIKeyService(repo, "test-master-key")
	srv := &Server{apiKeys: NewAPIKeyHandler(svc)}
	r := chi.NewRouter()
	if userID != nil {
		uid := *userID
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := middleware.SetUserContext(req.Context(), uid, model.RoleUser)
				next.ServeHTTP(w, req.WithContext(ctx))
			})
		})
	}
	HandlerFromMuxWithBaseURL(srv, r, "/api/v1")
	return r
}

// TestDeleteMyAPIKey_InvalidUUID_400 covers RG5: a non-UUID id is rejected with
// 400 by the generated wrapper, never 501.
func TestDeleteMyAPIKey_InvalidUUID_400(t *testing.T) {
	uid := uuid.New()
	router := newAPIKeyRouter(newMockAPIKeyHandlerRepo(), &uid)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me/api-keys/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("RG5: invalid uuid should be 400, got %d (body %s)", rec.Code, rec.Body.String())
	}
}

// TestDeleteMyAPIKey_Owned_NoContent covers RG6 (route served by the real
// handler, never 501), RG1 (owned delete -> 204, key removed) and RG3 (a second
// delete of the same id is an idempotent 204).
func TestDeleteMyAPIKey_Owned_NoContent(t *testing.T) {
	repo := newMockAPIKeyHandlerRepo()
	uid := uuid.New()
	keyID := uuid.New()
	repo.put(&model.UserAPIKey{ID: keyID, UserID: uid, Provider: "claude", KeyName: "default", KeyHint: "...1234"})
	router := newAPIKeyRouter(repo, &uid)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me/api-keys/"+keyID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotImplemented {
		t.Fatal("RG6: DELETE still served by Unimplemented (501) — the fix regressed")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("RG1: owned delete should be 204, got %d (body %s)", rec.Code, rec.Body.String())
	}
	if _, ok := repo.keys[keyID]; ok {
		t.Fatal("RG1: key should be removed from the store after delete")
	}

	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, httptest.NewRequest(http.MethodDelete, "/api/v1/users/me/api-keys/"+keyID.String(), nil))
	if rec2.Code != http.StatusNoContent {
		t.Fatalf("RG3: second delete of the same id should be idempotent 204, got %d", rec2.Code)
	}
}

// TestDeleteMyAPIKey_Unauthenticated_401 verifies the handler still requires
// authentication once routing is fixed.
func TestDeleteMyAPIKey_Unauthenticated_401(t *testing.T) {
	router := newAPIKeyRouter(newMockAPIKeyHandlerRepo(), nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me/api-keys/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated delete should be 401, got %d (body %s)", rec.Code, rec.Body.String())
	}
}
