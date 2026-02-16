package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

func setupProjectUserHandler() (*ProjectUserHandler, *mockProjectUserRepoForHandler, *mockProjectRepo, *mockUserRepoForHandler) {
	puRepo := newMockProjectUserRepoForHandler()
	projectRepo := newMockProjectRepo()
	userRepo := newMockUserRepoForHandler()
	puSvc := service.NewProjectUserService(puRepo, projectRepo, userRepo)
	h := NewProjectUserHandler(puSvc)
	return h, puRepo, projectRepo, userRepo
}

// addChiURLParams adds chi route context URL params to the request context.
func addChiURLParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	return r.WithContext(ctx)
}

func TestAddUser_AdminOnly(t *testing.T) {
	h, _, projectRepo, userRepo := setupProjectUserHandler()

	projectID := uuid.New()
	projectRepo.projects[projectID] = &model.Project{ID: projectID, Name: "test"}
	userID := uuid.New()
	userRepo.users[userID] = &model.User{ID: userID, Email: "user@test.com", Name: "Test", Role: model.RoleUser}

	tests := []struct {
		name       string
		role       model.Role
		wantStatus int
	}{
		{
			name:       "admin can add user",
			role:       model.RoleAdmin,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "non-admin gets 403",
			role:       model.RoleUser,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"user_id":"` + userID.String() + `","role":"member"}`
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/users", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")

			ctx := middleware.SetUserContext(req.Context(), uuid.New(), tt.role)
			req = req.WithContext(ctx)
			req = addChiURLParams(req, map[string]string{"id": projectID.String()})

			rec := httptest.NewRecorder()
			h.AddUser(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRemoveUser_AdminOnly(t *testing.T) {
	h, puRepo, projectRepo, _ := setupProjectUserHandler()

	projectID := uuid.New()
	projectRepo.projects[projectID] = &model.Project{ID: projectID, Name: "test"}
	userID := uuid.New()
	puRepo.assignments[puRepo.key(projectID, userID)] = model.ProjectRoleMember

	tests := []struct {
		name       string
		role       model.Role
		wantStatus int
	}{
		{
			name:       "non-admin gets 403",
			role:       model.RoleUser,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "admin can remove user",
			role:       model.RoleAdmin,
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String()+"/users/"+userID.String(), nil)

			ctx := middleware.SetUserContext(req.Context(), uuid.New(), tt.role)
			req = req.WithContext(ctx)
			req = addChiURLParams(req, map[string]string{
				"id":      projectID.String(),
				"user_id": userID.String(),
			})

			rec := httptest.NewRecorder()
			h.RemoveUser(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestListMembers_AsAssignedUser(t *testing.T) {
	h, _, projectRepo, _ := setupProjectUserHandler()

	projectID := uuid.New()
	projectRepo.projects[projectID] = &model.Project{ID: projectID, Name: "test"}

	userID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/users", nil)

	ctx := middleware.SetUserContext(req.Context(), userID, model.RoleUser)
	req = req.WithContext(ctx)
	req = addChiURLParams(req, map[string]string{"id": projectID.String()})

	rec := httptest.NewRecorder()
	h.ListMembers(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp []ProjectMemberResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

func TestAddUser_InvalidBody(t *testing.T) {
	h, _, projectRepo, _ := setupProjectUserHandler()

	projectID := uuid.New()
	projectRepo.projects[projectID] = &model.Project{ID: projectID, Name: "test"}

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{invalid}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing user_id",
			body:       `{"role":"member"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/users", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
			req = req.WithContext(ctx)
			req = addChiURLParams(req, map[string]string{"id": projectID.String()})

			rec := httptest.NewRecorder()
			h.AddUser(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}
