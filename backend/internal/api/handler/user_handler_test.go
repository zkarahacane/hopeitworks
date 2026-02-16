package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockUserRepo is a mock implementation of port.UserRepository for handler tests.
type mockUserRepo struct {
	users map[uuid.UUID]*model.User
}

// Compile-time check that mockUserRepo implements port.UserRepository.
var _ port.UserRepository = (*mockUserRepo)(nil)

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users: make(map[uuid.UUID]*model.User),
	}
}

func (m *mockUserRepo) Create(_ context.Context, user *model.User) (*model.User, error) {
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

func (m *mockUserRepo) Update(_ context.Context, user *model.User) (*model.User, error) {
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	return user, nil
}

func (m *mockUserRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.users, id)
	return nil
}

func setupUserHandler() (*UserHandler, *mockUserRepo) {
	repo := newMockUserRepo()
	svc := service.NewUserService(repo)
	h := NewUserHandler(svc)
	return h, repo
}

func seedTestUser(repo *mockUserRepo, name, email string, role model.Role) *model.User {
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

func TestListUsers_AdminOnly(t *testing.T) {
	h, repo := setupUserHandler()

	for i := 0; i < 3; i++ {
		seedTestUser(repo, "User", "user"+string(rune('0'+i))+"@example.com", model.RoleUser)
	}

	tests := []struct {
		name       string
		role       model.Role
		wantStatus int
	}{
		{
			name:       "admin can list",
			role:       model.RoleAdmin,
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-admin gets 403",
			role:       model.RoleUser,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "no role gets 403",
			role:       "",
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
			if tt.role != "" {
				ctx := middleware.SetUserContext(req.Context(), uuid.New(), tt.role)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			h.ListUsers(rec, req, ListUsersParams{})

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var resp UserList
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if len(resp.Data) != 3 {
					t.Errorf("expected 3 users, got %d", len(resp.Data))
				}
				if resp.Pagination.Total != 3 {
					t.Errorf("expected total 3, got %d", resp.Pagination.Total)
				}
			}
		})
	}
}

func TestGetUser_AdminOnly(t *testing.T) {
	h, repo := setupUserHandler()
	user := seedTestUser(repo, "Alice", "alice@example.com", model.RoleUser)

	tests := []struct {
		name       string
		role       model.Role
		id         uuid.UUID
		wantStatus int
	}{
		{
			name:       "admin can get user",
			role:       model.RoleAdmin,
			id:         user.ID,
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-admin gets 403",
			role:       model.RoleUser,
			id:         user.ID,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "admin gets 404 for non-existent",
			role:       model.RoleAdmin,
			id:         uuid.New(),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+tt.id.String(), nil)
			if tt.role != "" {
				ctx := middleware.SetUserContext(req.Context(), uuid.New(), tt.role)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			h.GetUser(rec, req, tt.id)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var resp User
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if string(resp.Role) != string(model.RoleUser) {
					t.Errorf("expected role %q, got %q", model.RoleUser, resp.Role)
				}
				if resp.Name != "Alice" {
					t.Errorf("expected name 'Alice', got %q", resp.Name)
				}
			}
		})
	}
}

func TestUpdateUser_AdminOnly(t *testing.T) {
	h, repo := setupUserHandler()
	user := seedTestUser(repo, "Alice", "alice@example.com", model.RoleUser)

	tests := []struct {
		name       string
		role       model.Role
		id         uuid.UUID
		body       string
		wantStatus int
	}{
		{
			name:       "admin can update role",
			role:       model.RoleAdmin,
			id:         user.ID,
			body:       `{"role":"admin"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "admin can update name",
			role:       model.RoleAdmin,
			id:         user.ID,
			body:       `{"name":"Alice Updated"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-admin gets 403",
			role:       model.RoleUser,
			id:         user.ID,
			body:       `{"name":"Hacked"}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "invalid JSON returns 400",
			role:       model.RoleAdmin,
			id:         user.ID,
			body:       `{invalid}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found returns 404",
			role:       model.RoleAdmin,
			id:         uuid.New(),
			body:       `{"name":"Test"}`,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+tt.id.String(),
				bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			if tt.role != "" {
				ctx := middleware.SetUserContext(req.Context(), uuid.New(), tt.role)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			h.UpdateUser(rec, req, tt.id)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestUpdateUser_RoleChange(t *testing.T) {
	h, repo := setupUserHandler()
	user := seedTestUser(repo, "Alice", "alice@example.com", model.RoleUser)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+user.ID.String(),
		bytes.NewBufferString(`{"role":"admin"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.UpdateUser(rec, req, user.ID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp User
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if string(resp.Role) != "admin" {
		t.Errorf("expected role 'admin', got %q", resp.Role)
	}
}

func TestDeleteUser_AdminOnly(t *testing.T) {
	h, repo := setupUserHandler()

	tests := []struct {
		name       string
		role       model.Role
		seedUser   bool
		wantStatus int
	}{
		{
			name:       "admin can delete",
			role:       model.RoleAdmin,
			seedUser:   true,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "non-admin gets 403",
			role:       model.RoleUser,
			seedUser:   true,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "admin gets 404 for non-existent",
			role:       model.RoleAdmin,
			seedUser:   false,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var targetID uuid.UUID
			if tt.seedUser {
				u := seedTestUser(repo, "ToDelete", "delete@example.com", model.RoleUser)
				targetID = u.ID
			} else {
				targetID = uuid.New()
			}

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+targetID.String(), nil)
			if tt.role != "" {
				ctx := middleware.SetUserContext(req.Context(), uuid.New(), tt.role)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			h.DeleteUser(rec, req, targetID)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}
