package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

// StackHandler implements the read-only stack catalogue HTTP handlers.
type StackHandler struct {
	service *service.StackService
}

// NewStackHandler creates a new StackHandler.
func NewStackHandler(svc *service.StackService) *StackHandler {
	return &StackHandler{service: svc}
}

// ListStacks handles GET /stacks. Returns the full catalogue (small, unpaginated).
func (h *StackHandler) ListStacks(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.List(r.Context())
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := struct {
		Data []Stack `json:"data"`
	}{
		Data: make([]Stack, len(result.Stacks)),
	}
	for i, s := range result.Stacks {
		resp.Data[i] = toAPIStack(s)
	}

	writeJSON(w, http.StatusOK, resp)
}

// toAPIStack converts a domain Stack to the API Stack type. The toolchain is stored
// as raw jsonb; decode it into a map, defaulting to empty on absent/invalid content.
func toAPIStack(s *model.Stack) Stack {
	toolchain := map[string]interface{}{}
	if len(s.Toolchain) > 0 {
		_ = json.Unmarshal(s.Toolchain, &toolchain)
	}
	return Stack{
		Id:        s.ID,
		Key:       StackKey(s.Key),
		ImageRef:  s.ImageRef,
		Toolchain: toolchain,
		CreatedAt: s.CreatedAt,
	}
}
