package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// ─── minimal port mocks for the handler authz tests ─────────────────────────────

type gcHandlerProjectRepo struct{ project *model.Project }

func (m *gcHandlerProjectRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Project, error) {
	if m.project != nil && m.project.ID == id {
		return m.project, nil
	}
	return nil, apperrors.NewNotFound("project", id)
}
func (m *gcHandlerProjectRepo) Create(_ context.Context, p *model.Project) (*model.Project, error) {
	return p, nil
}
func (m *gcHandlerProjectRepo) List(_ context.Context, _, _ int32) ([]*model.Project, error) {
	return nil, nil
}
func (m *gcHandlerProjectRepo) Count(_ context.Context) (int64, error) { return 0, nil }
func (m *gcHandlerProjectRepo) Update(_ context.Context, p *model.Project) (*model.Project, error) {
	return p, nil
}
func (m *gcHandlerProjectRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (m *gcHandlerProjectRepo) IncrementCircuitBreakerCount(_ context.Context, _ uuid.UUID) (*model.Project, error) {
	return nil, nil
}
func (m *gcHandlerProjectRepo) ResetCircuitBreaker(_ context.Context, _ uuid.UUID) (*model.Project, error) {
	return nil, nil
}

type gcHandlerConnRepo struct{}

func (gcHandlerConnRepo) GetByProject(_ context.Context, projectID uuid.UUID) (*model.GitConnection, error) {
	return nil, apperrors.NewNotFound("git_connection", projectID)
}
func (gcHandlerConnRepo) Upsert(_ context.Context, _ port.UpsertGitConnectionParams) (*model.GitConnection, error) {
	return nil, nil
}
func (gcHandlerConnRepo) SetValidation(_ context.Context, _ port.SetValidationParams) error {
	return nil
}
func (gcHandlerConnRepo) MarkStatus(_ context.Context, _ uuid.UUID, _ model.GitConnectionStatus, _ *string) error {
	return nil
}
func (gcHandlerConnRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

func newGitConnHandlerForTest(owner uuid.UUID) (*GitConnectionHandler, uuid.UUID) {
	projectID := uuid.New()
	projRepo := &gcHandlerProjectRepo{project: &model.Project{ID: projectID, OwnerID: &owner, GitProvider: "github"}}
	svc := service.NewGitConnectionService(gcHandlerConnRepo{}, projRepo, nil, "test-master-key-please-rotate-32!", nil, nil)
	return NewGitConnectionHandler(svc), projectID
}

func TestGetProjectGitConnection_Authorization(t *testing.T) {
	owner := uuid.New()

	tests := []struct {
		name     string
		actor    uuid.UUID
		role     model.Role
		wantCode int
	}{
		{"owner allowed", owner, model.RoleUser, http.StatusOK},
		{"admin allowed (non-owner)", uuid.New(), model.RoleAdmin, http.StatusOK},
		{"non-owner non-admin forbidden", uuid.New(), model.RoleUser, http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, projectID := newGitConnHandlerForTest(owner)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/git-connection", nil)
			req = req.WithContext(middleware.SetUserContext(req.Context(), tt.actor, tt.role))
			rec := httptest.NewRecorder()

			h.GetProjectGitConnection(rec, req, projectID)

			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d (body=%s)", rec.Code, tt.wantCode, rec.Body.String())
			}
		})
	}
}
