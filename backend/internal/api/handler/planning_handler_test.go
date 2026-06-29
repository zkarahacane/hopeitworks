package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	planningadapter "github.com/zakari/hopeitworks/backend/internal/adapter/planning"
	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

func setupPlanningHandler() (*PlanningHandler, *mockStoryRepo, *mockEpicRepo) {
	storyRepo := newMockStoryRepo()
	epicRepo := newMockEpicRepo()
	// Real markdown factory so the connector actually parses the body end-to-end.
	factory := planningadapter.NewFactory(nil, nil, nil)
	svc := service.NewPlanningImportService(storyRepo, epicRepo, factory)
	return NewPlanningHandler(svc), storyRepo, epicRepo
}

func planningRequest(t *testing.T, projectID uuid.UUID, body string, role model.Role) (*httptest.ResponseRecorder, *http.Request) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/planning/import",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if role != "" {
		req = req.WithContext(middleware.SetUserContext(req.Context(), uuid.New(), role))
	}
	return httptest.NewRecorder(), req
}

const sampleMarkdown = "---\\nkey: S-1\\nepic: Auth\\nscope: backend\\nstatus: done\\n---\\n# Login flow\\n\\nThe user can log in."

func TestImportPlanning_AdminOnly(t *testing.T) {
	h, _, _ := setupPlanningHandler()
	projectID := uuid.New()
	body := `{"source":"markdown","markdown":{"content":"` + sampleMarkdown + `"}}`

	rec, req := planningRequest(t, projectID, body, model.RoleUser)
	h.ImportPlanning(rec, req, projectID)
	if rec.Code != http.StatusForbidden {
		t.Errorf("non-admin expected 403, got %d", rec.Code)
	}
}

func TestImportPlanning_Validation(t *testing.T) {
	h, _, _ := setupPlanningHandler()
	projectID := uuid.New()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"empty markdown content", `{"source":"markdown","markdown":{"content":""}}`, http.StatusBadRequest},
		{"markdown missing config", `{"source":"markdown"}`, http.StatusBadRequest},
		{"github missing project_url", `{"source":"github_projects","github_projects":{"project_url":""}}`, http.StatusBadRequest},
		{"unknown source", `{"source":"jira"}`, http.StatusBadRequest},
		{"invalid json", `{nope}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, req := planningRequest(t, projectID, tt.body, model.RoleAdmin)
			h.ImportPlanning(rec, req, projectID)
			if rec.Code != tt.wantStatus {
				t.Errorf("expected %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestImportPlanning_MarkdownHappyPath(t *testing.T) {
	h, storyRepo, epicRepo := setupPlanningHandler()
	projectID := uuid.New()
	body := `{"source":"markdown","markdown":{"content":"` + sampleMarkdown + `"}}`

	rec, req := planningRequest(t, projectID, body, model.RoleAdmin)
	h.ImportPlanning(rec, req, projectID)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp PlanningImportResult
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Source != PlanningImportResultSourceMarkdown {
		t.Errorf("expected source markdown, got %q", resp.Source)
	}
	if resp.StoriesCreated != 1 {
		t.Errorf("expected 1 story created, got %d", resp.StoriesCreated)
	}
	if resp.EpicsCreated != 1 {
		t.Errorf("expected 1 epic created (Auth), got %d", resp.EpicsCreated)
	}
	if len(resp.Items) == 0 {
		t.Errorf("expected per-item decisions in the result")
	}
	if len(storyRepo.stories) != 1 {
		t.Errorf("expected 1 story persisted, got %d", len(storyRepo.stories))
	}
	if len(epicRepo.epics) != 1 {
		t.Errorf("expected 1 epic persisted, got %d", len(epicRepo.epics))
	}
	// status: done is honored as an explicit promotion on a brand-new backlog row.
	for _, s := range storyRepo.stories {
		if s.Status != model.StoryStatusDone {
			t.Errorf("story with markdown status:done should map to done, got %q", s.Status)
		}
		if s.Source != string("markdown") {
			t.Errorf("story should carry source=markdown, got %q", s.Source)
		}
	}
}

func TestImportPlanning_DryRunWritesNothing(t *testing.T) {
	h, storyRepo, epicRepo := setupPlanningHandler()
	projectID := uuid.New()
	body := `{"source":"markdown","dry_run":true,"markdown":{"content":"` + sampleMarkdown + `"}}`

	rec, req := planningRequest(t, projectID, body, model.RoleAdmin)
	h.ImportPlanning(rec, req, projectID)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp PlanningImportResult
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.DryRun {
		t.Errorf("expected dry_run=true in result")
	}
	if resp.StoriesCreated != 1 {
		t.Errorf("dry-run should still PLAN 1 story create, got %d", resp.StoriesCreated)
	}
	if len(storyRepo.stories) != 0 || len(epicRepo.epics) != 0 {
		t.Errorf("dry-run must NOT write: stories=%d epics=%d", len(storyRepo.stories), len(epicRepo.epics))
	}
}

func TestImportPlanning_GithubNotImplemented422(t *testing.T) {
	h, _, _ := setupPlanningHandler()
	projectID := uuid.New()
	body := `{"source":"github_projects","github_projects":{"project_url":"https://github.com/orgs/x/projects/1"}}`

	rec, req := planningRequest(t, projectID, body, model.RoleAdmin)
	h.ImportPlanning(rec, req, projectID)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("github_projects (Phase 3, not implemented) should surface SOURCE_ERROR/422, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}
