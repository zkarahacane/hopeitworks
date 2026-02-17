package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// PipelineConfigHandler implements pipeline config HTTP handlers.
type PipelineConfigHandler struct {
	service *service.PipelineConfigService
}

// NewPipelineConfigHandler creates a new PipelineConfigHandler.
func NewPipelineConfigHandler(svc *service.PipelineConfigService) *PipelineConfigHandler {
	return &PipelineConfigHandler{service: svc}
}

// GetPipelineConfig handles GET /projects/{projectId}/pipeline.
func (h *PipelineConfigHandler) GetPipelineConfig(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	config, err := h.service.GetByProjectID(r.Context(), projectID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIPipelineConfig(config))
}

// UpdatePipelineConfig handles PUT /projects/{projectId}/pipeline.
// Only admin users can update pipeline configs.
func (h *PipelineConfigHandler) UpdatePipelineConfig(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	if !middleware.IsAdmin(r.Context()) {
		writeErrorResponse(w, errors.NewForbidden("Admin access required"))
		return
	}

	var req UpdatePipelineConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, errors.NewValidation("body", "invalid JSON"))
		return
	}

	config, err := h.service.Upsert(r.Context(), projectID, req.ConfigYaml)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIPipelineConfig(config))
}

// toAPIPipelineConfig converts a domain PipelineConfig to the API PipelineConfig type.
func toAPIPipelineConfig(c *model.PipelineConfig) PipelineConfig {
	return PipelineConfig{
		Id:         c.ID,
		ProjectId:  c.ProjectID,
		ConfigYaml: c.ConfigYAML,
		Version:    c.Version,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}
