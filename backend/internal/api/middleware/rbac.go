package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// RequireProjectAccess returns chi middleware that checks if the authenticated
// user has access to the project identified by the {id} URL parameter.
// Admins bypass the check entirely.
func RequireProjectAccess(repo port.ProjectUserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserIDFromContext(r.Context())
			if !ok {
				writeForbidden(w, "Authentication required")
				return
			}

			if IsAdmin(r.Context()) {
				next.ServeHTTP(w, r)
				return
			}

			idStr := chi.URLParam(r, "id")
			projectID, err := uuid.Parse(idStr)
			if err != nil {
				writeBadRequest(w, "Invalid project ID format")
				return
			}

			isMember, err := repo.IsUserInProject(r.Context(), projectID, userID)
			if err != nil {
				writeInternalError(w)
				return
			}
			if !isMember {
				writeForbidden(w, "You are not a member of this project")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeForbidden(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    "FORBIDDEN",
			"message": msg,
		},
	})
}

func writeBadRequest(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    "VALIDATION_ERROR",
			"message": msg,
		},
	})
}

func writeInternalError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    "INTERNAL_ERROR",
			"message": "An internal error occurred",
		},
	})
}
