package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockProjectUserRepo is a mock implementation of port.ProjectUserRepository for service tests.
type mockProjectUserRepo struct {
	assignments map[string]*model.ProjectUser // key: "projectID:userID"
	members     map[uuid.UUID][]*model.ProjectMember
	projects    map[uuid.UUID][]*model.Project // userID -> projects
}

var _ port.ProjectUserRepository = (*mockProjectUserRepo)(nil)

func newMockProjectUserRepo() *mockProjectUserRepo {
	return &mockProjectUserRepo{
		assignments: make(map[string]*model.ProjectUser),
		members:     make(map[uuid.UUID][]*model.ProjectMember),
		projects:    make(map[uuid.UUID][]*model.Project),
	}
}

func (m *mockProjectUserRepo) key(projectID, userID uuid.UUID) string {
	return projectID.String() + ":" + userID.String()
}

func (m *mockProjectUserRepo) AddUser(_ context.Context, projectID, userID uuid.UUID, role model.ProjectRole) (*model.ProjectUser, error) {
	k := m.key(projectID, userID)
	if _, exists := m.assignments[k]; exists {
		return nil, errors.NewConflict("project_user", "user already assigned")
	}
	pu := &model.ProjectUser{ProjectID: projectID, UserID: userID, Role: role}
	m.assignments[k] = pu
	return pu, nil
}

func (m *mockProjectUserRepo) RemoveUser(_ context.Context, projectID, userID uuid.UUID) error {
	delete(m.assignments, m.key(projectID, userID))
	return nil
}

func (m *mockProjectUserRepo) ListMembers(_ context.Context, projectID uuid.UUID) ([]*model.ProjectMember, error) {
	return m.members[projectID], nil
}

func (m *mockProjectUserRepo) IsUserInProject(_ context.Context, projectID, userID uuid.UUID) (bool, error) {
	_, ok := m.assignments[m.key(projectID, userID)]
	return ok, nil
}

func (m *mockProjectUserRepo) ListProjectsByUser(_ context.Context, userID uuid.UUID, limit, offset int32) ([]*model.Project, error) {
	all := m.projects[userID]
	start := int(offset)
	if start > len(all) {
		return []*model.Project{}, nil
	}
	end := start + int(limit)
	if end > len(all) {
		end = len(all)
	}
	return all[start:end], nil
}

func (m *mockProjectUserRepo) CountProjectsByUser(_ context.Context, userID uuid.UUID) (int64, error) {
	return int64(len(m.projects[userID])), nil
}

// mockProjectUserServiceUserRepo is a mock implementation of port.UserRepository for service tests.
type mockProjectUserServiceUserRepo struct {
	users map[uuid.UUID]*model.User
}

var _ port.UserRepository = (*mockProjectUserServiceUserRepo)(nil)

func newMockProjectUserServiceUserRepo() *mockProjectUserServiceUserRepo {
	return &mockProjectUserServiceUserRepo{users: make(map[uuid.UUID]*model.User)}
}

func (m *mockProjectUserServiceUserRepo) Create(_ context.Context, user *model.User) (*model.User, error) {
	user.ID = uuid.New()
	m.users[user.ID] = user
	return user, nil
}

func (m *mockProjectUserServiceUserRepo) GetByEmail(_ context.Context, _ string) (*model.User, error) {
	return nil, errors.NewNotFound("user", "email")
}

func (m *mockProjectUserServiceUserRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, errors.NewNotFound("user", id)
	}
	return u, nil
}

func (m *mockProjectUserServiceUserRepo) List(_ context.Context, _, _ int32) ([]*model.User, error) {
	return nil, nil
}
func (m *mockProjectUserServiceUserRepo) Count(_ context.Context) (int64, error) { return 0, nil }
func (m *mockProjectUserServiceUserRepo) Update(_ context.Context, user *model.User) (*model.User, error) {
	return user, nil
}
func (m *mockProjectUserServiceUserRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

func setupProjectUserService() (*ProjectUserService, *mockProjectUserRepo, *mockProjectRepo, *mockProjectUserServiceUserRepo) {
	puRepo := newMockProjectUserRepo()
	// Reuse mockProjectRepo from project_service_test.go
	projectRepo := newMockProjectRepoForService()
	userRepo := newMockProjectUserServiceUserRepo()
	svc := NewProjectUserService(puRepo, projectRepo, userRepo)
	return svc, puRepo, projectRepo, userRepo
}

func TestProjectUserService_AddUser(t *testing.T) {
	svc, _, projectRepo, userRepo := setupProjectUserService()

	projectID := uuid.New()
	projectRepo.projects[projectID] = &model.Project{ID: projectID, Name: "test"}

	userID := uuid.New()
	userRepo.users[userID] = &model.User{ID: userID, Email: "user@example.com", Name: "Test User", Role: model.RoleUser}

	tests := []struct {
		name    string
		pID     uuid.UUID
		uID     uuid.UUID
		role    model.ProjectRole
		wantErr bool
		errCode string
	}{
		{
			name: "valid assignment",
			pID:  projectID,
			uID:  userID,
			role: model.ProjectRoleMember,
		},
		{
			name:    "duplicate assignment",
			pID:     projectID,
			uID:     userID,
			role:    model.ProjectRoleMember,
			wantErr: true,
			errCode: "PROJECT_USER_ALREADY_EXISTS",
		},
		{
			name:    "non-existent project",
			pID:     uuid.New(),
			uID:     userID,
			role:    model.ProjectRoleMember,
			wantErr: true,
			errCode: "PROJECT_NOT_FOUND",
		},
		{
			name:    "non-existent user",
			pID:     projectID,
			uID:     uuid.New(),
			role:    model.ProjectRoleMember,
			wantErr: true,
			errCode: "USER_NOT_FOUND",
		},
		{
			name:    "invalid role",
			pID:     projectID,
			uID:     userID,
			role:    "invalid",
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.AddUser(context.Background(), tt.pID, tt.uID, tt.role)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				domainErr, ok := err.(*errors.DomainError)
				if !ok {
					t.Fatalf("expected DomainError, got %T: %v", err, err)
				}
				if domainErr.Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, domainErr.Code)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ProjectID != tt.pID || result.UserID != tt.uID {
				t.Errorf("unexpected result: %+v", result)
			}
		})
	}
}

func TestProjectUserService_RemoveUser(t *testing.T) {
	svc, puRepo, projectRepo, userRepo := setupProjectUserService()

	projectID := uuid.New()
	projectRepo.projects[projectID] = &model.Project{ID: projectID, Name: "test"}
	userID := uuid.New()
	userRepo.users[userID] = &model.User{ID: userID, Email: "user@example.com"}

	// Assign first
	puRepo.assignments[puRepo.key(projectID, userID)] = &model.ProjectUser{
		ProjectID: projectID, UserID: userID, Role: model.ProjectRoleMember,
	}

	// Remove existing member
	err := svc.RemoveUser(context.Background(), projectID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Remove non-member
	err = svc.RemoveUser(context.Background(), projectID, uuid.New())
	if err == nil {
		t.Fatal("expected error for non-member, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %q", domainErr.Category)
	}
}

func TestProjectUserService_ListMembers(t *testing.T) {
	svc, puRepo, projectRepo, _ := setupProjectUserService()

	projectID := uuid.New()
	projectRepo.projects[projectID] = &model.Project{ID: projectID, Name: "test"}

	puRepo.members[projectID] = []*model.ProjectMember{
		{UserID: uuid.New(), Email: "a@example.com", Name: "User A", UserRole: model.RoleUser, ProjectRole: model.ProjectRoleMember},
		{UserID: uuid.New(), Email: "b@example.com", Name: "User B", UserRole: model.RoleAdmin, ProjectRole: model.ProjectRoleOwner},
	}

	members, err := svc.ListMembers(context.Background(), projectID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}

	// Non-existent project
	_, err = svc.ListMembers(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent project, got nil")
	}
}

func TestProjectUserService_ListProjectsForUser(t *testing.T) {
	svc, puRepo, _, _ := setupProjectUserService()

	userID := uuid.New()
	for i := 0; i < 5; i++ {
		id := uuid.New()
		puRepo.projects[userID] = append(puRepo.projects[userID], &model.Project{
			ID:   id,
			Name: "project-" + id.String()[:8],
		})
	}

	result, err := svc.ListProjectsForUser(context.Background(), userID, 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 5 {
		t.Errorf("expected total 5, got %d", result.Total)
	}
	if len(result.Projects) != 5 {
		t.Errorf("expected 5 projects, got %d", len(result.Projects))
	}

	// Pagination: page 2, perPage 3
	result, err = svc.ListProjectsForUser(context.Background(), userID, 2, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Projects) != 2 {
		t.Errorf("expected 2 projects on page 2 with perPage 3, got %d", len(result.Projects))
	}
	if result.Total != 5 {
		t.Errorf("expected total 5, got %d", result.Total)
	}
}
