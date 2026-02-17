package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// PromptTemplateHandler implements prompt-template-related HTTP handlers.
type PromptTemplateHandler struct {
	service *service.PromptTemplateService
}

// NewPromptTemplateHandler creates a new PromptTemplateHandler.
func NewPromptTemplateHandler(svc *service.PromptTemplateService) *PromptTemplateHandler {
	return &PromptTemplateHandler{service: svc}
}

// ListPromptTemplates handles GET /projects/{projectId}/templates.
func (h *PromptTemplateHandler) ListPromptTemplates(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params ListPromptTemplatesParams) {
	page := 1
	perPage := 20
	if params.Page != nil && *params.Page > 0 {
		page = *params.Page
	}
	if params.PerPage != nil && *params.PerPage > 0 {
		perPage = *params.PerPage
	}

	result, err := h.service.ListByProject(r.Context(), projectID, page, perPage)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := PromptTemplateList{
		Data: make([]PromptTemplate, len(result.Templates)),
		Pagination: Pagination{
			Total:   int(result.Total),
			Page:    page,
			PerPage: perPage,
		},
	}
	for i, t := range result.Templates {
		resp.Data[i] = toAPIPromptTemplate(t)
	}

	writeJSON(w, http.StatusOK, resp)
}

// CreatePromptTemplate handles POST /projects/{projectId}/templates.
// Only admin users can create prompt templates.
func (h *PromptTemplateHandler) CreatePromptTemplate(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	if !middleware.IsAdmin(r.Context()) {
		writeErrorResponse(w, errors.NewForbidden("Admin access required"))
		return
	}

	var req CreatePromptTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, errors.NewValidation("body", "invalid JSON"))
		return
	}

	params := service.CreatePromptTemplateParams{
		ProjectID:       projectID,
		Name:            req.Name,
		TemplateContent: req.TemplateContent,
		Type:            string(req.Type),
	}

	tmpl, err := h.service.Create(r.Context(), params)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toAPIPromptTemplate(tmpl))
}

// GetPromptTemplate handles GET /projects/{projectId}/templates/{templateId}.
func (h *PromptTemplateHandler) GetPromptTemplate(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, templateID TemplateIdPath) {
	tmpl, err := h.service.GetByID(r.Context(), templateID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIPromptTemplate(tmpl))
}

// UpdatePromptTemplate handles PUT /projects/{projectId}/templates/{templateId}.
// Only admin users can update prompt templates.
func (h *PromptTemplateHandler) UpdatePromptTemplate(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, templateID TemplateIdPath) {
	if !middleware.IsAdmin(r.Context()) {
		writeErrorResponse(w, errors.NewForbidden("Admin access required"))
		return
	}

	var req UpdatePromptTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, errors.NewValidation("body", "invalid JSON"))
		return
	}

	params := service.UpdatePromptTemplateParams{
		ID:              templateID,
		Name:            req.Name,
		TemplateContent: req.TemplateContent,
	}
	if req.Type != nil {
		s := string(*req.Type)
		params.Type = &s
	}

	tmpl, err := h.service.Update(r.Context(), params)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIPromptTemplate(tmpl))
}

// DeletePromptTemplate handles DELETE /projects/{projectId}/templates/{templateId}.
// Only admin users can delete prompt templates.
func (h *PromptTemplateHandler) DeletePromptTemplate(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, templateID TemplateIdPath) {
	if !middleware.IsAdmin(r.Context()) {
		writeErrorResponse(w, errors.NewForbidden("Admin access required"))
		return
	}

	if err := h.service.Delete(r.Context(), templateID); err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// toAPIPromptTemplate converts a domain PromptTemplate to the API PromptTemplate type.
func toAPIPromptTemplate(t *model.PromptTemplate) PromptTemplate {
	return PromptTemplate{
		Id:              t.ID,
		ProjectId:       t.ProjectID,
		Name:            t.Name,
		TemplateContent: t.TemplateContent,
		Type:            PromptTemplateType(t.Type),
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
	}
}
