package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// ContextKeyContainerToken is the context key for container token data.
const ContextKeyContainerToken contextKey = "container_token"

// InternalAuth returns middleware that validates container bearer tokens.
// This is separate from the JWT auth middleware and is used for internal
// agent callback endpoints.
func InternalAuth(tokenStore port.ContainerTokenStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"Bearer token required"}}`))
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			ct, err := tokenStore.Validate(r.Context(), token)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":{"code":"INVALID_TOKEN","message":"Invalid or expired token"}}`))
				return
			}

			// Inject the container token info into context for handlers
			ctx := SetContainerTokenContext(r.Context(), ct)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SetContainerTokenContext stores a ContainerToken in the request context.
func SetContainerTokenContext(ctx context.Context, ct *model.ContainerToken) context.Context {
	return context.WithValue(ctx, ContextKeyContainerToken, ct)
}

// ContainerTokenFromContext extracts the ContainerToken from the request context.
func ContainerTokenFromContext(ctx context.Context) (*model.ContainerToken, bool) {
	ct, ok := ctx.Value(ContextKeyContainerToken).(*model.ContainerToken)
	return ct, ok
}
