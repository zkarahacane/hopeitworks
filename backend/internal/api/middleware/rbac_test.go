package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// mockProjectUserRepo is a mock implementation of port.ProjectUserRepository for RBAC tests.
type mockProjectUserRepo struct {
	members map[string]bool // key: "projectID:userID"
}

var _ port.ProjectUserRepository = (*mockProjectUserRepo)(nil)

func newMockProjectUserRepo() *mockProjectUserRepo {
	return &mockProjectUserRepo{members: make(map[string]bool)}
}

func (m *mockProjectUserRepo) key(projectID, userID uuid.UUID) string {
	return projectID.String() + ":" + userID.String()
}

func (m *mockProjectUserRepo) AddUser(_ context.Context, projectID, userID uuid.UUID, role model.ProjectRole) (*model.ProjectUser, error) {
	m.members[m.key(projectID, userID)] = true
	return &model.ProjectUser{ProjectID: projectID, UserID: userID, Role: role}, nil
}

func (m *mockProjectUserRepo) RemoveUser(_ context.Context, projectID, userID uuid.UUID) error {
	delete(m.members, m.key(projectID, userID))
	return nil
}

func (m *mockProjectUserRepo) ListMembers(_ context.Context, _ uuid.UUID) ([]*model.ProjectMember, error) {
	return nil, nil
}

func (m *mockProjectUserRepo) IsUserInProject(_ context.Context, projectID, userID uuid.UUID) (bool, error) {
	return m.members[m.key(projectID, userID)], nil
}

func (m *mockProjectUserRepo) ListProjectsByUser(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Project, error) {
	return nil, nil
}

func (m *mockProjectUserRepo) CountProjectsByUser(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}

func TestRequireProjectAccess(t *testing.T) {
	projectID := uuid.New()
	adminID := uuid.New()
	memberID := uuid.New()
	nonMemberID := uuid.New()

	repo := newMockProjectUserRepo()
	repo.members[repo.key(projectID, memberID)] = true

	mw := RequireProjectAccess(repo)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name       string
		userID     uuid.UUID
		role       model.Role
		projectID  string
		hasAuth    bool
		wantStatus int
		wantNext   bool
	}{
		{
			name:       "admin bypasses check",
			userID:     adminID,
			role:       model.RoleAdmin,
			projectID:  projectID.String(),
			hasAuth:    true,
			wantStatus: http.StatusOK,
			wantNext:   true,
		},
		{
			name:       "assigned user is allowed",
			userID:     memberID,
			role:       model.RoleUser,
			projectID:  projectID.String(),
			hasAuth:    true,
			wantStatus: http.StatusOK,
			wantNext:   true,
		},
		{
			name:       "unassigned user gets 403",
			userID:     nonMemberID,
			role:       model.RoleUser,
			projectID:  projectID.String(),
			hasAuth:    true,
			wantStatus: http.StatusForbidden,
			wantNext:   false,
		},
		{
			name:       "invalid project ID returns 400",
			userID:     memberID,
			role:       model.RoleUser,
			projectID:  "not-a-uuid",
			hasAuth:    true,
			wantStatus: http.StatusBadRequest,
			wantNext:   false,
		},
		{
			name:       "missing auth context returns 403",
			projectID:  projectID.String(),
			hasAuth:    false,
			wantStatus: http.StatusForbidden,
			wantNext:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled = false

			req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+tt.projectID, nil)

			// Set up chi route context with URL param
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.projectID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			if tt.hasAuth {
				ctx := SetUserContext(req.Context(), tt.userID, tt.role)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			mw(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}

			if nextCalled != tt.wantNext {
				t.Errorf("expected next called=%v, got %v", tt.wantNext, nextCalled)
			}

			if tt.wantStatus == http.StatusForbidden {
				var resp map[string]interface{}
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				errObj, ok := resp["error"].(map[string]interface{})
				if !ok {
					t.Fatal("expected error envelope in response")
				}
				if errObj["code"] != "FORBIDDEN" {
					t.Errorf("expected error code FORBIDDEN, got %v", errObj["code"])
				}
			}
		})
	}
}
