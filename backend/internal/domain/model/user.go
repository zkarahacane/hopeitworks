package model

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

func (r Role) IsValid() bool {
	return r == RoleAdmin || r == RoleUser
}

// User represents a platform user.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Name         string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}
