package model

import (
	"time"

	"github.com/google/uuid"
)

// ProjectRole represents a user's role within a project.
type ProjectRole string

const (
	ProjectRoleOwner  ProjectRole = "owner"
	ProjectRoleMember ProjectRole = "member"
)

// IsValid checks if the project role is valid.
func (r ProjectRole) IsValid() bool {
	return r == ProjectRoleOwner || r == ProjectRoleMember
}

// ProjectUser represents the association between a user and a project.
type ProjectUser struct {
	ProjectID uuid.UUID
	UserID    uuid.UUID
	Role      ProjectRole
	CreatedAt time.Time
}

// ProjectMember is the joined view returned by ListProjectUsers.
type ProjectMember struct {
	UserID      uuid.UUID
	Email       string
	Name        string
	UserRole    Role        // global role (admin/user)
	ProjectRole ProjectRole // role within project (owner/member)
	AssignedAt  time.Time
}
