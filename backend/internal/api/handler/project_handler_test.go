package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockProjectRepo is a mock implementation of port.ProjectRepository for handler tests.
type mockProjectRepo struct {
	projects map[uuid.UUID]*model.Project
}

// Compile-time check that mockProjectRepo implements port.ProjectRepository.
var _ port.ProjectRepository = (*mockProjectRepo)(nil)

func newMockProjectRepo() *mockProjectRepo {
	return &mockProjectRepo{
		projects: make(map[uuid.UUID]*model.Project),
	}
}

func (m *mockProjectRepo) Create(_ context.Context, project *model.Project) (*model.Project, error) {
	project.ID = uuid.New()
	m.projects[project.ID] = project
	return project, nil
}

func (m *mockProjectRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Project, error) {
	p, ok := m.projects[id]
	if !ok {
		return nil, errors.NewNotFound("project", id)
	}
	return p, nil
}

func (m *mockProjectRepo) List(_ context.Context, limit, offset int32) ([]*model.Project, error) {
	result := make([]*model.Project, 0)
	i := int32(0)
	for _, p := range m.projects {
		if i >= offset && i < offset+limit {
			result = append(result, p)
		}
		i++
	}
	return result, nil
}

func (m *mockProjectRepo) Count(_ context.Context) (int64, error) {
	return int64(len(m.projects)), nil
}

func (m *mockProjectRepo) Update(_ context.Context, project *model.Project) (*model.Project, error) {
	m.projects[project.ID] = project
	return project, nil
}

func (m *mockProjectRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.projects, id)
	return nil
}

func (m *mockProjectRepo) IncrementCircuitBreakerCount(_ context.Context, _ uuid.UUID) (*model.Project, error) {
	return &model.Project{}, nil
}

func (m *mockProjectRepo) ResetCircuitBreaker(_ context.Context, _ uuid.UUID) (*model.Project, error) {
	return &model.Project{}, nil
}

// mockProjectUserRepoForHandler is a mock implementation of port.ProjectUserRepository for handler tests.
type mockProjectUserRepoForHandler struct {
	assignments map[string]model.ProjectRole // key: "projectID:userID"
	projects    map[uuid.UUID]*model.Project // user-project list
}

var _ port.ProjectUserRepository = (*mockProjectUserRepoForHandler)(nil)

func newMockProjectUserRepoForHandler() *mockProjectUserRepoForHandler {
	return &mockProjectUserRepoForHandler{
		assignments: make(map[string]model.ProjectRole),
		projects:    make(map[uuid.UUID]*model.Project),
	}
}

func (m *mockProjectUserRepoForHandler) key(projectID, userID uuid.UUID) string {
	return projectID.String() + ":" + userID.String()
}

func (m *mockProjectUserRepoForHandler) AddUser(_ context.Context, projectID, userID uuid.UUID, role model.ProjectRole) (*model.ProjectUser, error) {
	m.assignments[m.key(projectID, userID)] = role
	return &model.ProjectUser{ProjectID: projectID, UserID: userID, Role: role}, nil
}

func (m *mockProjectUserRepoForHandler) RemoveUser(_ context.Context, projectID, userID uuid.UUID) error {
	delete(m.assignments, m.key(projectID, userID))
	return nil
}

func (m *mockProjectUserRepoForHandler) ListMembers(_ context.Context, _ uuid.UUID) ([]*model.ProjectMember, error) {
	return nil, nil
}

func (m *mockProjectUserRepoForHandler) IsUserInProject(_ context.Context, projectID, userID uuid.UUID) (bool, error) {
	_, ok := m.assignments[m.key(projectID, userID)]
	return ok, nil
}

func (m *mockProjectUserRepoForHandler) ListProjectsByUser(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Project, error) {
	result := make([]*model.Project, 0, len(m.projects))
	for _, p := range m.projects {
		result = append(result, p)
	}
	return result, nil
}

func (m *mockProjectUserRepoForHandler) CountProjectsByUser(_ context.Context, _ uuid.UUID) (int64, error) {
	return int64(len(m.projects)), nil
}

// mockUserRepoForHandler is a minimal mock for port.UserRepository in handler tests.
type mockUserRepoForHandler struct {
	users map[uuid.UUID]*model.User
}

var _ port.UserRepository = (*mockUserRepoForHandler)(nil)

func newMockUserRepoForHandler() *mockUserRepoForHandler {
	return &mockUserRepoForHandler{users: make(map[uuid.UUID]*model.User)}
}

func (m *mockUserRepoForHandler) Create(_ context.Context, user *model.User) (*model.User, error) {
	user.ID = uuid.New()
	m.users[user.ID] = user
	return user, nil
}

func (m *mockUserRepoForHandler) GetByEmail(_ context.Context, _ string) (*model.User, error) {
	return nil, errors.NewNotFound("user", "email")
}

func (m *mockUserRepoForHandler) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, errors.NewNotFound("user", id)
	}
	return u, nil
}

func (m *mockUserRepoForHandler) List(_ context.Context, _, _ int32) ([]*model.User, error) {
	return nil, nil
}

func (m *mockUserRepoForHandler) Count(_ context.Context) (int64, error) { return 0, nil }

func (m *mockUserRepoForHandler) Update(_ context.Context, user *model.User) (*model.User, error) {
	return user, nil
}

func (m *mockUserRepoForHandler) UpdatePasswordHash(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (m *mockUserRepoForHandler) Delete(_ context.Context, _ uuid.UUID) error { return nil }

func setupHandler() (*ProjectHandler, *mockProjectRepo) {
	h, repo, _ := setupHandlerWithStories()
	return h, repo
}

// setupHandlerWithStories builds a ProjectHandler wired to a story repo so tests
// can seed stories and assert the enriched story_count (#289).
func setupHandlerWithStories() (*ProjectHandler, *mockProjectRepo, *mockStoryRepo) {
	repo := newMockProjectRepo()
	svc := service.NewProjectService(repo)
	puRepo := newMockProjectUserRepoForHandler()
	userRepo := newMockUserRepoForHandler()
	puSvc := service.NewProjectUserService(puRepo, repo, userRepo)
	storyRepo := newMockStoryRepo()
	handler := NewProjectHandler(svc, puSvc, nil, storyRepo)
	return handler, repo, storyRepo
}

func TestCreateProject_AdminOnly(t *testing.T) {
	h, _ := setupHandler()

	tests := []struct {
		name       string
		role       model.Role
		body       string
		wantStatus int
	}{
		{
			name:       "admin can create",
			role:       model.RoleAdmin,
			body:       `{"name":"test-project"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "non-admin gets 403",
			role:       model.RoleUser,
			body:       `{"name":"test-project"}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "no role gets 403",
			role:       "",
			body:       `{"name":"test-project"}`,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			if tt.role != "" {
				ctx := middleware.SetUserContext(req.Context(), uuid.New(), tt.role)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			h.CreateProject(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCreateProject_Validation(t *testing.T) {
	h, _ := setupHandler()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid",
			body:       `{"name":"test-project"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "empty name",
			body:       `{"name":""}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			body:       `{invalid}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			h.CreateProject(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestListProjects_Admin(t *testing.T) {
	h, repo := setupHandler()

	// Seed data
	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.projects[id] = &model.Project{
			ID:           id,
			Name:         "project-" + id.String()[:8],
			GitProvider:  "github",
			AgentRuntime: "docker",
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.ListProjects(rec, req, ListProjectsParams{})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp ProjectList
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 3 {
		t.Errorf("expected 3 projects, got %d", len(resp.Data))
	}
	if resp.Pagination.Total != 3 {
		t.Errorf("expected total 3, got %d", resp.Pagination.Total)
	}
}

func TestListProjects_NonAdmin_ReturnsAssigned(t *testing.T) {
	repo := newMockProjectRepo()
	svc := service.NewProjectService(repo)
	puRepo := newMockProjectUserRepoForHandler()
	userRepo := newMockUserRepoForHandler()
	puSvc := service.NewProjectUserService(puRepo, repo, userRepo)
	h := NewProjectHandler(svc, puSvc, nil, newMockStoryRepo())

	userID := uuid.New()

	// Create 3 projects, assign user to 2
	for i := 0; i < 3; i++ {
		id := uuid.New()
		p := &model.Project{
			ID:           id,
			Name:         "project-" + id.String()[:8],
			GitProvider:  "github",
			AgentRuntime: "docker",
		}
		repo.projects[id] = p
		if i < 2 {
			puRepo.assignments[puRepo.key(id, userID)] = model.ProjectRoleMember
			puRepo.projects[id] = p
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	ctx := middleware.SetUserContext(req.Context(), userID, model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.ListProjects(rec, req, ListProjectsParams{})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp ProjectList
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 2 {
		t.Errorf("expected 2 projects for non-admin, got %d", len(resp.Data))
	}
	if resp.Pagination.Total != 2 {
		t.Errorf("expected total 2, got %d", resp.Pagination.Total)
	}
}

func TestGetProject_NotFound(t *testing.T) {
	h, _ := setupHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+uuid.New().String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetProject(rec, req, uuid.New())

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestGetProject_NonAdminNotAssigned(t *testing.T) {
	h, repo := setupHandler()

	id := uuid.New()
	repo.projects[id] = &model.Project{
		ID:           id,
		Name:         "test",
		GitProvider:  "github",
		AgentRuntime: "docker",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+id.String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetProject(rec, req, id)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for unassigned non-admin, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateProject_AdminOnly(t *testing.T) {
	h, repo := setupHandler()

	id := uuid.New()
	repo.projects[id] = &model.Project{
		ID:           id,
		Name:         "original",
		GitProvider:  "github",
		AgentRuntime: "docker",
	}

	// Non-admin should get 403
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+id.String(),
		bytes.NewBufferString(`{"name":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.UpdateProject(rec, req, id)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", rec.Code)
	}

	// Admin should succeed
	req = httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+id.String(),
		bytes.NewBufferString(`{"name":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx = middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	h.UpdateProject(rec, req, id)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteProject_AdminOnly(t *testing.T) {
	h, repo := setupHandler()

	id := uuid.New()
	repo.projects[id] = &model.Project{
		ID:           id,
		Name:         "to-delete",
		GitProvider:  "github",
		AgentRuntime: "docker",
	}

	// Non-admin should get 403
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+id.String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.DeleteProject(rec, req, id)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", rec.Code)
	}

	// Admin should succeed
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+id.String(), nil)
	ctx = middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	h.DeleteProject(rec, req, id)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 for admin, got %d", rec.Code)
	}
}

// seedStories adds n backlog stories for the given project to the mock story repo.
func seedStories(repo *mockStoryRepo, projectID uuid.UUID, n int) {
	for i := 0; i < n; i++ {
		id := uuid.New()
		repo.stories[id] = &model.Story{
			ID:        id,
			ProjectID: projectID,
			Key:       "S-" + id.String()[:8],
			Title:     "story",
			Status:    model.StoryStatusBacklog,
		}
	}
}

func listProjectsAsAdmin(t *testing.T, h *ProjectHandler) ProjectList {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.ListProjects(rec, req, ListProjectsParams{})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	var resp ProjectList
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return resp
}

// TestListProjects_StoryCount covers RG1 (5 stories -> 5), RG2 (no stories -> 0),
// and RG3 (exactly 1 story -> 1) for the project list endpoint (#289).
func TestListProjects_StoryCount(t *testing.T) {
	tests := []struct {
		name       string
		numStories int
		want       int
	}{
		{name: "RG1 five stories", numStories: 5, want: 5},
		{name: "RG2 no stories", numStories: 0, want: 0},
		{name: "RG3 exactly one story", numStories: 1, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, projectRepo, storyRepo := setupHandlerWithStories()

			projectID := uuid.New()
			projectRepo.projects[projectID] = &model.Project{
				ID:           projectID,
				Name:         "proj",
				GitProvider:  "github",
				AgentRuntime: "docker",
			}
			seedStories(storyRepo, projectID, tt.numStories)

			resp := listProjectsAsAdmin(t, h)
			if len(resp.Data) != 1 {
				t.Fatalf("expected 1 project, got %d", len(resp.Data))
			}
			got := resp.Data[0].StoryCount
			if got == nil {
				t.Fatalf("expected story_count to be set, got nil")
			}
			if *got != tt.want {
				t.Errorf("expected story_count %d, got %d", tt.want, *got)
			}
		})
	}
}

// TestListProjects_StoryCount_PerProjectIsolation ensures counts are not leaked
// across projects: each project reports only its own stories (#289).
func TestListProjects_StoryCount_PerProjectIsolation(t *testing.T) {
	h, projectRepo, storyRepo := setupHandlerWithStories()

	withStories := uuid.New()
	without := uuid.New()
	projectRepo.projects[withStories] = &model.Project{ID: withStories, Name: "with", GitProvider: "github", AgentRuntime: "docker"}
	projectRepo.projects[without] = &model.Project{ID: without, Name: "without", GitProvider: "github", AgentRuntime: "docker"}
	seedStories(storyRepo, withStories, 3)

	resp := listProjectsAsAdmin(t, h)

	counts := make(map[uuid.UUID]int)
	for _, p := range resp.Data {
		if p.StoryCount == nil {
			t.Fatalf("project %s missing story_count", p.Id)
		}
		counts[p.Id] = *p.StoryCount
	}
	if counts[withStories] != 3 {
		t.Errorf("expected 3 stories for seeded project, got %d", counts[withStories])
	}
	if counts[without] != 0 {
		t.Errorf("expected 0 stories for empty project, got %d", counts[without])
	}
}

// TestListProjects_StoryCount_MatchesDetailCount proves RG4: the list's
// story_count equals the count that backs the project detail's stories
// pagination.total. Both go through StoryRepository.CountByProject (the same
// query the StoryService.ListByProject uses for its total), so they agree (#289).
func TestListProjects_StoryCount_MatchesDetailCount(t *testing.T) {
	h, projectRepo, storyRepo := setupHandlerWithStories()

	projectID := uuid.New()
	projectRepo.projects[projectID] = &model.Project{ID: projectID, Name: "proj", GitProvider: "github", AgentRuntime: "docker"}
	seedStories(storyRepo, projectID, 7)

	// List story_count.
	resp := listProjectsAsAdmin(t, h)
	if len(resp.Data) != 1 || resp.Data[0].StoryCount == nil {
		t.Fatalf("unexpected list response: %+v", resp.Data)
	}
	listCount := *resp.Data[0].StoryCount

	// Detail count: the StoryService backs the detail view's stories
	// pagination.total via the same repo. perPage=1 mirrors the front's call.
	detail, err := service.NewStoryService(storyRepo).ListByProject(context.Background(), projectID, 1, 1)
	if err != nil {
		t.Fatalf("ListByProject failed: %v", err)
	}
	if listCount != int(detail.Total) {
		t.Errorf("RG4 mismatch: list story_count=%d, detail total=%d", listCount, detail.Total)
	}
}

// TestListProjects_StoryCount_DegradesOnError proves RG5: when the story count
// fails, the project list still returns 200 (never 500 from the count alone)
// and the failing project's story_count degrades to 0 (#289 regression).
func TestListProjects_StoryCount_DegradesOnError(t *testing.T) {
	h, projectRepo, storyRepo := setupHandlerWithStories()
	storyRepo.countErr = errors.NewInternal("count stories", context.DeadlineExceeded)

	projectID := uuid.New()
	projectRepo.projects[projectID] = &model.Project{ID: projectID, Name: "proj", GitProvider: "github", AgentRuntime: "docker"}
	seedStories(storyRepo, projectID, 4)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.ListProjects(rec, req, ListProjectsParams{})

	if rec.Code != http.StatusOK {
		t.Fatalf("RG5: expected 200 despite count failure, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	var resp ProjectList
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].StoryCount == nil {
		t.Fatalf("unexpected list response: %+v", resp.Data)
	}
	if *resp.Data[0].StoryCount != 0 {
		t.Errorf("RG5: expected story_count to degrade to 0, got %d", *resp.Data[0].StoryCount)
	}
}
