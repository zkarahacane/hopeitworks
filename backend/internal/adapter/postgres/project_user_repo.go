package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure ProjectUserRepo implements port.ProjectUserRepository at compile time.
var _ port.ProjectUserRepository = (*ProjectUserRepo)(nil)

// ProjectUserRepo implements port.ProjectUserRepository using sqlc-generated queries.
type ProjectUserRepo struct {
	queries *Queries
}

// NewProjectUserRepo creates a new ProjectUserRepo.
func NewProjectUserRepo(queries *Queries) *ProjectUserRepo {
	return &ProjectUserRepo{queries: queries}
}

func (r *ProjectUserRepo) AddUser(ctx context.Context, projectID, userID uuid.UUID, role model.ProjectRole) (*model.ProjectUser, error) {
	row, err := r.queries.AddUserToProject(ctx, AddUserToProjectParams{
		ProjectID: projectID,
		UserID:    userID,
		Role:      string(role),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflict("project_user", "user already assigned")
		}
		if isForeignKeyViolation(err) {
			return nil, apperrors.NewNotFound("project or user", projectID)
		}
		return nil, apperrors.NewInternal("failed to add user to project", err)
	}
	return toDomainProjectUser(row), nil
}

func (r *ProjectUserRepo) RemoveUser(ctx context.Context, projectID, userID uuid.UUID) error {
	err := r.queries.RemoveUserFromProject(ctx, RemoveUserFromProjectParams{
		ProjectID: projectID,
		UserID:    userID,
	})
	if err != nil {
		return apperrors.NewInternal("failed to remove user from project", err)
	}
	return nil
}

func (r *ProjectUserRepo) ListMembers(ctx context.Context, projectID uuid.UUID) ([]*model.ProjectMember, error) {
	rows, err := r.queries.ListProjectUsers(ctx, projectID)
	if err != nil {
		return nil, apperrors.NewInternal("failed to list project members", err)
	}
	members := make([]*model.ProjectMember, len(rows))
	for i, row := range rows {
		members[i] = &model.ProjectMember{
			UserID:      row.ID,
			Email:       row.Email,
			Name:        row.Name,
			UserRole:    model.Role(row.UserRole),
			ProjectRole: model.ProjectRole(row.ProjectRole),
			AssignedAt:  row.AssignedAt,
		}
	}
	return members, nil
}

func (r *ProjectUserRepo) IsUserInProject(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	isMember, err := r.queries.IsUserInProject(ctx, IsUserInProjectParams{
		ProjectID: projectID,
		UserID:    userID,
	})
	if err != nil {
		return false, apperrors.NewInternal("failed to check project membership", err)
	}
	return isMember, nil
}

func (r *ProjectUserRepo) ListProjectsByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*model.Project, error) {
	rows, err := r.queries.ListProjectsByUser(ctx, ListProjectsByUserParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list projects by user", err)
	}
	projects := make([]*model.Project, len(rows))
	for i, row := range rows {
		projects[i] = toDomainProject(row)
	}
	return projects, nil
}

func (r *ProjectUserRepo) CountProjectsByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := r.queries.CountProjectsByUser(ctx, userID)
	if err != nil {
		return 0, apperrors.NewInternal("failed to count projects by user", err)
	}
	return count, nil
}

func toDomainProjectUser(pu ProjectUser) *model.ProjectUser {
	return &model.ProjectUser{
		ProjectID: pu.ProjectID,
		UserID:    pu.UserID,
		Role:      model.ProjectRole(pu.Role),
		CreatedAt: pu.CreatedAt,
	}
}

// isForeignKeyViolation checks if a pgx error is a foreign key constraint violation.
func isForeignKeyViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23503"
	}
	return false
}
