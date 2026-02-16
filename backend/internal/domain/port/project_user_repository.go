package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// ProjectUserRepository defines the interface for project-user association persistence.
type ProjectUserRepository interface {
	AddUser(ctx context.Context, projectID, userID uuid.UUID, role model.ProjectRole) (*model.ProjectUser, error)
	RemoveUser(ctx context.Context, projectID, userID uuid.UUID) error
	ListMembers(ctx context.Context, projectID uuid.UUID) ([]*model.ProjectMember, error)
	IsUserInProject(ctx context.Context, projectID, userID uuid.UUID) (bool, error)
	ListProjectsByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*model.Project, error)
	CountProjectsByUser(ctx context.Context, userID uuid.UUID) (int64, error)
}
