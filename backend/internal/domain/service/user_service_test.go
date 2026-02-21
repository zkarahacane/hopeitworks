package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockUserRepo is a mock implementation of port.UserRepository for testing.
type mockUserRepo struct {
	users            map[uuid.UUID]*model.User
	createFn         func(ctx context.Context, u *model.User) (*model.User, error)
	updateFn         func(ctx context.Context, u *model.User) (*model.User, error)
	deleteFn         func(ctx context.Context, id uuid.UUID) error
	updatePasswordFn func(ctx context.Context, id uuid.UUID, hash string) error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users: make(map[uuid.UUID]*model.User),
	}
}

func (m *mockUserRepo) Create(ctx context.Context, user *model.User) (*model.User, error) {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	return user, nil
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, errors.NewNotFound("user", email)
}

func (m *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, errors.NewNotFound("user", id)
	}
	return u, nil
}

func (m *mockUserRepo) List(_ context.Context, limit, offset int32) ([]*model.User, error) {
	result := make([]*model.User, 0)
	i := int32(0)
	for _, u := range m.users {
		if i >= offset && i < offset+limit {
			result = append(result, u)
		}
		i++
	}
	return result, nil
}

func (m *mockUserRepo) Count(_ context.Context) (int64, error) {
	return int64(len(m.users)), nil
}

func (m *mockUserRepo) Update(ctx context.Context, user *model.User) (*model.User, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, user)
	}
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	return user, nil
}

func (m *mockUserRepo) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	if m.updatePasswordFn != nil {
		return m.updatePasswordFn(ctx, id, hash)
	}
	u, ok := m.users[id]
	if !ok {
		return errors.NewNotFound("user", id)
	}
	u.PasswordHash = hash
	u.UpdatedAt = time.Now()
	return nil
}

func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	delete(m.users, id)
	return nil
}

func seedUser(repo *mockUserRepo, name, email string, role model.Role) *model.User {
	id := uuid.New()
	u := &model.User{
		ID:        id,
		Name:      name,
		Email:     email,
		Role:      role,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.users[id] = u
	return u
}

func TestUserService_GetByID(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewUserService(repo)

	user := seedUser(repo, "Alice", "alice@example.com", model.RoleUser)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
		errCode string
	}{
		{
			name:    "existing user",
			id:      user.ID,
			wantErr: false,
		},
		{
			name:    "non-existent user",
			id:      uuid.New(),
			wantErr: true,
			errCode: "USER_NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.GetByID(context.Background(), tt.id)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				domainErr, ok := err.(*errors.DomainError)
				if !ok {
					t.Fatalf("expected DomainError, got %T", err)
				}
				if domainErr.Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, domainErr.Code)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ID != tt.id {
				t.Errorf("expected ID %v, got %v", tt.id, result.ID)
			}
		})
	}
}

func TestUserService_List(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewUserService(repo)

	for i := 0; i < 5; i++ {
		seedUser(repo, "User", "user"+string(rune('0'+i))+"@example.com", model.RoleUser)
	}

	tests := []struct {
		name      string
		page      int
		perPage   int
		wantTotal int64
	}{
		{name: "default pagination", page: 1, perPage: 20, wantTotal: 5},
		{name: "clamp page to 1", page: 0, perPage: 20, wantTotal: 5},
		{name: "clamp perPage to 20", page: 1, perPage: 0, wantTotal: 5},
		{name: "clamp perPage max to 100", page: 1, perPage: 200, wantTotal: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.List(context.Background(), tt.page, tt.perPage)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Total != tt.wantTotal {
				t.Errorf("expected total %d, got %d", tt.wantTotal, result.Total)
			}
		})
	}
}

func TestUserService_Update(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewUserService(repo)

	user := seedUser(repo, "Alice", "alice@example.com", model.RoleUser)

	newName := "Alice Updated"
	newEmail := "alice-new@example.com"
	adminRole := model.RoleAdmin
	invalidRole := model.Role("superadmin")
	emptyName := ""
	longName := string(make([]byte, 256))
	emptyEmail := ""

	tests := []struct {
		name    string
		params  UpdateUserParams
		wantErr bool
		errCode string
	}{
		{
			name:    "update name",
			params:  UpdateUserParams{ID: user.ID, Name: &newName},
			wantErr: false,
		},
		{
			name:    "update email",
			params:  UpdateUserParams{ID: user.ID, Email: &newEmail},
			wantErr: false,
		},
		{
			name:    "update role to admin",
			params:  UpdateUserParams{ID: user.ID, Role: &adminRole},
			wantErr: false,
		},
		{
			name:    "not found",
			params:  UpdateUserParams{ID: uuid.New(), Name: &newName},
			wantErr: true,
			errCode: "USER_NOT_FOUND",
		},
		{
			name:    "invalid role",
			params:  UpdateUserParams{ID: user.ID, Role: &invalidRole},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "empty name",
			params:  UpdateUserParams{ID: user.ID, Name: &emptyName},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "name too long",
			params:  UpdateUserParams{ID: user.ID, Name: &longName},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "empty email",
			params:  UpdateUserParams{ID: user.ID, Email: &emptyEmail},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.Update(context.Background(), tt.params)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				domainErr, ok := err.(*errors.DomainError)
				if !ok {
					t.Fatalf("expected DomainError, got %T", err)
				}
				if domainErr.Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, domainErr.Code)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

func TestUserService_Delete(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewUserService(repo)

	user := seedUser(repo, "ToDelete", "delete@example.com", model.RoleUser)

	// Delete existing user
	err := svc.Delete(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Delete non-existent user
	err = svc.Delete(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent user, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != "USER_NOT_FOUND" {
		t.Errorf("expected USER_NOT_FOUND, got %q", domainErr.Code)
	}
}

func TestUserService_UpdateProfile(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewUserService(repo)

	user := seedUser(repo, "Alice", "alice@example.com", model.RoleUser)

	newName := "Alice Updated"
	newEmail := "alice-new@example.com"
	emptyName := ""
	longName := string(make([]byte, 256))

	tests := []struct {
		name    string
		params  UpdateProfileParams
		wantErr bool
		errCode string
	}{
		{
			name:    "update name only",
			params:  UpdateProfileParams{ID: user.ID, Name: &newName},
			wantErr: false,
		},
		{
			name:    "update email only",
			params:  UpdateProfileParams{ID: user.ID, Email: &newEmail},
			wantErr: false,
		},
		{
			name:    "update both name and email",
			params:  UpdateProfileParams{ID: user.ID, Name: &newName, Email: &newEmail},
			wantErr: false,
		},
		{
			name:    "empty name",
			params:  UpdateProfileParams{ID: user.ID, Name: &emptyName},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "name too long",
			params:  UpdateProfileParams{ID: user.ID, Name: &longName},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "user not found",
			params:  UpdateProfileParams{ID: uuid.New(), Name: &newName},
			wantErr: true,
			errCode: "UNAUTHORIZED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.UpdateProfile(context.Background(), tt.params)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				domainErr, ok := err.(*errors.DomainError)
				if !ok {
					t.Fatalf("expected DomainError, got %T", err)
				}
				if domainErr.Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, domainErr.Code)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

func TestUserService_UpdateProfile_DuplicateEmail(t *testing.T) {
	repo := newMockUserRepo()
	// Override update to simulate duplicate key error
	repo.updateFn = func(_ context.Context, _ *model.User) (*model.User, error) {
		return nil, &pgDuplicateKeyError{}
	}
	svc := NewUserService(repo)

	user := seedUser(repo, "Alice", "alice@example.com", model.RoleUser)
	email := "taken@example.com"

	_, err := svc.UpdateProfile(context.Background(), UpdateProfileParams{
		ID:    user.ID,
		Email: &email,
	})
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != "EMAIL_ALREADY_EXISTS" {
		t.Errorf("expected EMAIL_ALREADY_EXISTS, got %q", domainErr.Code)
	}
}

func TestUserService_ChangePassword(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewUserService(repo)

	// Seed user with a known bcrypt hash for "oldpassword"
	hash, _ := bcrypt.GenerateFromPassword([]byte("oldpassword"), bcrypt.DefaultCost)
	user := seedUser(repo, "Alice", "alice@example.com", model.RoleUser)
	user.PasswordHash = string(hash)

	tests := []struct {
		name            string
		userID          uuid.UUID
		currentPassword string
		newPassword     string
		wantErr         bool
		errCode         string
	}{
		{
			name:            "successful password change",
			userID:          user.ID,
			currentPassword: "oldpassword",
			newPassword:     "newpassword123",
			wantErr:         false,
		},
		{
			name:            "wrong current password",
			userID:          user.ID,
			currentPassword: "wrongpassword",
			newPassword:     "newpassword123",
			wantErr:         true,
			errCode:         "UNAUTHORIZED",
		},
		{
			name:            "new password too short",
			userID:          user.ID,
			currentPassword: "oldpassword",
			newPassword:     "short",
			wantErr:         true,
			errCode:         "VALIDATION_ERROR",
		},
		{
			name:            "user not found",
			userID:          uuid.New(),
			currentPassword: "oldpassword",
			newPassword:     "newpassword123",
			wantErr:         true,
			errCode:         "UNAUTHORIZED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset password hash for repeated tests
			if tt.name == "successful password change" || tt.name == "new password too short" {
				h, _ := bcrypt.GenerateFromPassword([]byte("oldpassword"), bcrypt.DefaultCost)
				user.PasswordHash = string(h)
			}

			err := svc.ChangePassword(context.Background(), tt.userID, tt.currentPassword, tt.newPassword)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				domainErr, ok := err.(*errors.DomainError)
				if !ok {
					t.Fatalf("expected DomainError, got %T", err)
				}
				if domainErr.Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, domainErr.Code)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify the password hash was updated
			updatedUser := repo.users[tt.userID]
			if bcrypt.CompareHashAndPassword([]byte(updatedUser.PasswordHash), []byte(tt.newPassword)) != nil {
				t.Error("password hash was not updated correctly")
			}
		})
	}
}
