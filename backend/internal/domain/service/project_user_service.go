package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// ProjectUserService provides business logic for project-user associations.
type ProjectUserService struct {
	repo        port.ProjectUserRepository
	projectRepo port.ProjectRepository
	userRepo    port.UserRepository
}

// NewProjectUserService creates a new ProjectUserService.
func NewProjectUserService(repo port.ProjectUserRepository, projectRepo port.ProjectRepository, userRepo port.UserRepository) *ProjectUserService {
	return &ProjectUserService{
		repo:        repo,
		projectRepo: projectRepo,
		userRepo:    userRepo,
	}
}

// AddUser assigns a user to a project with the given role.
func (s *ProjectUserService) AddUser(ctx context.Context, projectID, userID uuid.UUID, role model.ProjectRole) (*model.ProjectUser, error) {
	if !role.IsValid() {
		return nil, errors.NewValidation("role", "must be 'owner' or 'member'")
	}

	// Validate project exists
	_, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Validate user exists
	_, err = s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.repo.AddUser(ctx, projectID, userID, role)
}

// RemoveUser removes a user from a project.
func (s *ProjectUserService) RemoveUser(ctx context.Context, projectID, userID uuid.UUID) error {
	isMember, err := s.repo.IsUserInProject(ctx, projectID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.NewNotFound("project_user", userID)
	}

	return s.repo.RemoveUser(ctx, projectID, userID)
}

// ListMembers lists all members of a project.
func (s *ProjectUserService) ListMembers(ctx context.Context, projectID uuid.UUID) ([]*model.ProjectMember, error) {
	// Validate project exists
	_, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return s.repo.ListMembers(ctx, projectID)
}

// IsUserInProject checks if a user is a member of a project.
func (s *ProjectUserService) IsUserInProject(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	return s.repo.IsUserInProject(ctx, projectID, userID)
}

// ListProjectsForUser retrieves a paginated list of projects assigned to a user.
func (s *ProjectUserService) ListProjectsForUser(ctx context.Context, userID uuid.UUID, page, perPage int) (*ListResult, error) {
	limit, offset := paginationToLimitOffset(page, perPage)

	projects, err := s.repo.ListProjectsByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.CountProjectsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &ListResult{
		Projects: projects,
		Total:    total,
	}, nil
}
