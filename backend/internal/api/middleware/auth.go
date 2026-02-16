package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

type contextKey string

const (
	ContextKeyUserID contextKey = "user_id"
	ContextKeyRole   contextKey = "user_role"
)

// UserIDFromContext extracts the user ID from the request context.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ContextKeyUserID).(uuid.UUID)
	return id, ok
}

// RoleFromContext extracts the user role from the request context.
func RoleFromContext(ctx context.Context) (model.Role, bool) {
	role, ok := ctx.Value(ContextKeyRole).(model.Role)
	return role, ok
}

// IsAdmin checks if the user in the context has admin role.
func IsAdmin(ctx context.Context) bool {
	role, ok := RoleFromContext(ctx)
	return ok && role == model.RoleAdmin
}

// SetUserContext sets user information in the request context.
// Used for testing and by auth middleware after successful JWT validation.
func SetUserContext(ctx context.Context, userID uuid.UUID, role model.Role) context.Context {
	ctx = context.WithValue(ctx, ContextKeyUserID, userID)
	ctx = context.WithValue(ctx, ContextKeyRole, role)
	return ctx
}

// Auth returns middleware that validates JWT tokens and injects user context.
func Auth(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("token")
			if err != nil {
				writeUnauthorized(w)
				return
			}

			claims, err := authService.ValidateToken(cookie.Value)
			if err != nil {
				writeUnauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextKeyRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"Authentication required"}}`))
}
