package service

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// ErrInvalidCurrentPassword is returned when the current password does not match.
var ErrInvalidCurrentPassword = errors.NewUnauthorized("current password is incorrect")

// UserService provides business logic for user management (admin CRUD).
type UserService struct {
	repo port.UserRepository
}

// NewUserService creates a new UserService.
func NewUserService(repo port.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// UserListResult holds the result of a paginated user list operation.
type UserListResult struct {
	Users []*model.User
	Total int64
}

// UpdateUserParams holds parameters for updating a user.
type UpdateUserParams struct {
	ID    uuid.UUID
	Name  *string
	Email *string
	Role  *model.Role
}

// GetByID retrieves a user by ID.
func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.NewNotFound("user", id)
	}
	return user, nil
}

// List retrieves a paginated list of users.
func (s *UserService) List(ctx context.Context, page, perPage int) (*UserListResult, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	offset := int32((page - 1) * perPage)
	limit := int32(perPage)

	users, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, err
	}

	return &UserListResult{
		Users: users,
		Total: total,
	}, nil
}

// Update validates inputs and updates an existing user.
func (s *UserService) Update(ctx context.Context, params UpdateUserParams) (*model.User, error) {
	existing, err := s.repo.GetByID(ctx, params.ID)
	if err != nil {
		return nil, errors.NewNotFound("user", params.ID)
	}

	if params.Name != nil {
		if *params.Name == "" {
			return nil, errors.NewValidation("name", "must not be empty")
		}
		if len(*params.Name) > 255 {
			return nil, errors.NewValidation("name", "must be 255 characters or less")
		}
		existing.Name = *params.Name
	}

	if params.Email != nil {
		if *params.Email == "" {
			return nil, errors.NewValidation("email", "must not be empty")
		}
		existing.Email = *params.Email
	}

	if params.Role != nil {
		if !params.Role.IsValid() {
			return nil, errors.NewValidation("role", "must be 'admin' or 'user'")
		}
		existing.Role = *params.Role
	}

	return s.repo.Update(ctx, existing)
}

// UpdateProfileParams holds parameters for self-service profile updates.
// Role is intentionally excluded — users cannot change their own role.
type UpdateProfileParams struct {
	ID    uuid.UUID
	Name  *string
	Email *string
}

// UpdateProfile validates and applies a self-service profile update.
func (s *UserService) UpdateProfile(ctx context.Context, params UpdateProfileParams) (*model.User, error) {
	existing, err := s.repo.GetByID(ctx, params.ID)
	if err != nil {
		return nil, errors.NewUnauthorized("user not found")
	}

	if params.Name != nil {
		if *params.Name == "" {
			return nil, errors.NewValidation("name", "must not be empty")
		}
		if len(*params.Name) > 255 {
			return nil, errors.NewValidation("name", "must be 255 characters or less")
		}
		existing.Name = *params.Name
	}

	if params.Email != nil {
		if *params.Email == "" {
			return nil, errors.NewValidation("email", "must not be empty")
		}
		existing.Email = *params.Email
	}

	updated, err := s.repo.Update(ctx, existing)
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, errors.NewConflict("email", *params.Email)
		}
		return nil, err
	}
	return updated, nil
}

// ChangePassword verifies the current password and sets a new bcrypt hash.
func (s *UserService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return errors.NewUnauthorized("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrInvalidCurrentPassword
	}

	if len(newPassword) < 8 {
		return errors.NewValidation("new_password", "must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.NewInternal("failed to hash password", err)
	}

	return s.repo.UpdatePasswordHash(ctx, userID, string(hash))
}

// Delete soft-deletes a user by ID.
func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return errors.NewNotFound("user", id)
	}
	return s.repo.Delete(ctx, id)
}
