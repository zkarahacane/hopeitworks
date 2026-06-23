package handler

import (
	"net/http"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

// EnvironmentHandler implements the project environment HTTP handlers.
type EnvironmentHandler struct {
	service *service.EnvService
}

// NewEnvironmentHandler creates a new EnvironmentHandler.
func NewEnvironmentHandler(svc *service.EnvService) *EnvironmentHandler {
	return &EnvironmentHandler{service: svc}
}

// GetProjectEnvironment handles GET /projects/{projectId}/environment.
func (h *EnvironmentHandler) GetProjectEnvironment(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	env, err := h.service.GetByProject(r.Context(), projectID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIEnvironment(env))
}

// PutProjectEnvironment handles PUT /projects/{projectId}/environment (upsert).
func (h *EnvironmentHandler) PutProjectEnvironment(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	var req EnvironmentInput
	if !decodeJSONBody(w, r, &req) {
		return
	}

	input := service.UpsertEnvironmentInput{
		Stacks:   req.Stacks,
		Services: inputServicesToModel(req.Services),
		Source:   string(req.Source),
		Commands: req.Commands,
	}

	env, err := h.service.Upsert(r.Context(), projectID, input)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIEnvironment(env))
}

// DeleteProjectEnvironment handles DELETE /projects/{projectId}/environment.
func (h *EnvironmentHandler) DeleteProjectEnvironment(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	if err := h.service.Delete(r.Context(), projectID); err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// toAPIEnvironment converts a domain Environment to the generated API Environment type.
func toAPIEnvironment(e *model.Environment) Environment {
	services := make([]EnvironmentService, len(e.Services))
	for i, svc := range e.Services {
		env := make(map[string]string, len(svc.Env))
		for k, v := range svc.Env {
			env[k] = v
		}
		services[i] = EnvironmentService{
			Name:  svc.Name,
			Image: svc.Image,
			Env:   env,
		}
	}
	commands := make(map[string]string, len(e.Commands))
	for k, v := range e.Commands {
		commands[k] = v
	}
	return Environment{
		Id:        e.ID,
		ProjectId: e.ProjectID,
		Stacks:    e.Stacks,
		Services:  services,
		Source:    EnvironmentSource(e.Source),
		Commands:  commands,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

// inputServicesToModel converts API EnvironmentService slice to domain model slice.
func inputServicesToModel(apiServices []EnvironmentService) []model.EnvironmentService {
	result := make([]model.EnvironmentService, len(apiServices))
	for i, svc := range apiServices {
		env := make(map[string]string, len(svc.Env))
		for k, v := range svc.Env {
			env[k] = v
		}
		result[i] = model.EnvironmentService{
			Name:  svc.Name,
			Image: svc.Image,
			Env:   env,
		}
	}
	return result
}
