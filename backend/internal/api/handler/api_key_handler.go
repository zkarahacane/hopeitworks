package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// APIKeyHandler implements HTTP handlers for user API key management.
type APIKeyHandler struct {
	service *service.APIKeyService
}

// NewAPIKeyHandler creates a new APIKeyHandler.
func NewAPIKeyHandler(svc *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{service: svc}
}

// apiKeyResponse is the JSON response for an API key (never includes encrypted_key).
type apiKeyResponse struct {
	ID        uuid.UUID `json:"id"`
	Provider  string    `json:"provider"`
	KeyName   string    `json:"key_name"`
	KeyHint   string    `json:"key_hint"`
	CreatedAt time.Time `json:"created_at"`
}

// createAPIKeyRequest is the JSON request body for creating an API key.
type createAPIKeyRequest struct {
	Provider string `json:"provider"`
	KeyName  string `json:"key_name"`
	APIKey   string `json:"api_key"`
}

// ListMyAPIKeys handles GET /api/v1/users/me/api-keys.
func (h *APIKeyHandler) ListMyAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, errors.NewUnauthorized("authentication required"))
		return
	}

	keys, err := h.service.ListKeys(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := make([]apiKeyResponse, len(keys))
	for i, k := range keys {
		resp[i] = toAPIKeyResponse(k)
	}

	writeJSON(w, http.StatusOK, resp)
}

// CreateMyAPIKey handles POST /api/v1/users/me/api-keys.
func (h *APIKeyHandler) CreateMyAPIKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, errors.NewUnauthorized("authentication required"))
		return
	}

	var req createAPIKeyRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	key, err := h.service.CreateKey(r.Context(), userID, req.Provider, req.KeyName, req.APIKey)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toAPIKeyResponse(key))
}

// DeleteMyAPIKey handles DELETE /api/v1/users/me/api-keys/{keyId}.
func (h *APIKeyHandler) DeleteMyAPIKey(w http.ResponseWriter, r *http.Request) {
	_, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, errors.NewUnauthorized("authentication required"))
		return
	}

	keyIDStr := chi.URLParam(r, "keyId")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		writeErrorResponse(w, errors.NewValidation("keyId", "must be a valid UUID"))
		return
	}

	if err := h.service.DeleteKey(r.Context(), keyID); err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// toAPIKeyResponse converts a domain UserAPIKey to the API response (never exposes encrypted_key).
func toAPIKeyResponse(k *model.UserAPIKey) apiKeyResponse {
	return apiKeyResponse{
		ID:        k.ID,
		Provider:  k.Provider,
		KeyName:   k.KeyName,
		KeyHint:   k.KeyHint,
		CreatedAt: k.CreatedAt,
	}
}
