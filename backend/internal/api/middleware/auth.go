package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const (
	userIDKey contextKey = "user_id"
	roleKey   contextKey = "user_role"
)

// AuthMiddleware validates JWT cookies and injects user info into context.
// This is a placeholder that will be fully implemented in Story 1-3.
// For now, it extracts the user context if already set (e.g., by tests).
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Full JWT validation will be implemented in Story 1-3.
		// This middleware is a pass-through to allow the handler to check
		// for authentication context that may be set by upstream middleware.
		next.ServeHTTP(w, r)
	})
}

// SetUserContext sets user information in the request context.
// Used by auth middleware after successful JWT validation.
func SetUserContext(ctx context.Context, userID uuid.UUID, role string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, roleKey, role)
	return ctx
}

// UserIDFromContext extracts the user ID from the context.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}

// RoleFromContext extracts the user role from the context.
func RoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(roleKey).(string)
	return role, ok
}

// IsAdmin checks if the user in the context has admin role.
func IsAdmin(ctx context.Context) bool {
	role, ok := RoleFromContext(ctx)
	return ok && role == "admin"
}
