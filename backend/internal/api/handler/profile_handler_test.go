package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

func setupProfileHandler() (*ProfileHandler, *mockUserRepo) {
	repo := newMockUserRepo()
	svc := service.NewUserService(repo)
	h := NewProfileHandler(svc)
	return h, repo
}

func seedTestUserWithPassword(repo *mockUserRepo, name, email, password string, role model.Role) *model.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	u := seedTestUser(repo, name, email, role)
	u.PasswordHash = string(hash)
	return u
}

func TestGetMyProfile(t *testing.T) {
	h, repo := setupProfileHandler()
	user := seedTestUser(repo, "Alice", "alice@example.com", model.RoleUser)

	tests := []struct {
		name       string
		userID     uuid.UUID
		hasAuth    bool
		wantStatus int
	}{
		{
			name:       "authenticated user gets profile",
			userID:     user.ID,
			hasAuth:    true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "no auth context returns 401",
			hasAuth:    false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "user not found returns 401",
			userID:     uuid.New(),
			hasAuth:    true,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
			if tt.hasAuth {
				ctx := middleware.SetUserContext(req.Context(), tt.userID, model.RoleUser)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			h.GetMyProfile(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var resp User
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.Name != "Alice" {
					t.Errorf("expected name 'Alice', got %q", resp.Name)
				}
				if string(resp.Email) != "alice@example.com" {
					t.Errorf("expected email 'alice@example.com', got %q", resp.Email)
				}
			}
		})
	}
}

func TestUpdateMyProfile(t *testing.T) {
	h, repo := setupProfileHandler()
	user := seedTestUser(repo, "Alice", "alice@example.com", model.RoleUser)

	tests := []struct {
		name       string
		userID     uuid.UUID
		hasAuth    bool
		body       string
		wantStatus int
	}{
		{
			name:       "update name only",
			userID:     user.ID,
			hasAuth:    true,
			body:       `{"name":"Alice Updated"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "update email only",
			userID:     user.ID,
			hasAuth:    true,
			body:       `{"email":"new@example.com"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "no auth returns 401",
			hasAuth:    false,
			body:       `{"name":"Alice Updated"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "empty body returns 400",
			userID:     user.ID,
			hasAuth:    true,
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON returns 400",
			userID:     user.ID,
			hasAuth:    true,
			body:       `{invalid}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty name returns 400",
			userID:     user.ID,
			hasAuth:    true,
			body:       `{"name":""}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me",
				bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			if tt.hasAuth {
				ctx := middleware.SetUserContext(req.Context(), tt.userID, model.RoleUser)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			h.UpdateMyProfile(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var resp User
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.Id != user.ID {
					t.Errorf("expected user ID %v, got %v", user.ID, resp.Id)
				}
			}
		})
	}
}

func TestChangeMyPassword(t *testing.T) {
	h, repo := setupProfileHandler()
	user := seedTestUserWithPassword(repo, "Alice", "alice@example.com", "oldpassword", model.RoleUser)

	tests := []struct {
		name       string
		userID     uuid.UUID
		hasAuth    bool
		body       string
		wantStatus int
		resetPw    bool
	}{
		{
			name:       "successful password change",
			userID:     user.ID,
			hasAuth:    true,
			body:       `{"current_password":"oldpassword","new_password":"newpassword123"}`,
			wantStatus: http.StatusNoContent,
			resetPw:    true,
		},
		{
			name:       "wrong current password",
			userID:     user.ID,
			hasAuth:    true,
			body:       `{"current_password":"wrongpassword","new_password":"newpassword123"}`,
			wantStatus: http.StatusUnauthorized,
			resetPw:    true,
		},
		{
			name:       "new password too short",
			userID:     user.ID,
			hasAuth:    true,
			body:       `{"current_password":"oldpassword","new_password":"short"}`,
			wantStatus: http.StatusBadRequest,
			resetPw:    true,
		},
		{
			name:       "no auth returns 401",
			hasAuth:    false,
			body:       `{"current_password":"oldpassword","new_password":"newpassword123"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "empty current password returns 400",
			userID:     user.ID,
			hasAuth:    true,
			body:       `{"current_password":"","new_password":"newpassword123"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty new password returns 400",
			userID:     user.ID,
			hasAuth:    true,
			body:       `{"current_password":"oldpassword","new_password":""}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON returns 400",
			userID:     user.ID,
			hasAuth:    true,
			body:       `{invalid}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset password hash before tests that need the original password
			if tt.resetPw {
				hash, _ := bcrypt.GenerateFromPassword([]byte("oldpassword"), bcrypt.DefaultCost)
				user.PasswordHash = string(hash)
			}

			req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/password",
				bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			if tt.hasAuth {
				ctx := middleware.SetUserContext(req.Context(), tt.userID, model.RoleUser)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			h.ChangeMyPassword(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestChangeMyPassword_VerifyErrorCode(t *testing.T) {
	h, repo := setupProfileHandler()
	user := seedTestUserWithPassword(repo, "Alice", "alice@example.com", "oldpassword", model.RoleUser)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/password",
		bytes.NewBufferString(`{"current_password":"wrongpassword","new_password":"newpassword123"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), user.ID, model.RoleUser)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.ChangeMyPassword(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	var errResp Error
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if errResp.Error.Code != "INVALID_CREDENTIALS" {
		t.Errorf("expected error code INVALID_CREDENTIALS, got %q", errResp.Error.Code)
	}
}
