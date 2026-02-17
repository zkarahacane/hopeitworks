package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/adapter/markdown"
	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// StoryHandler implements story-related HTTP handlers.
type StoryHandler struct {
	service *service.StoryService
}

// NewStoryHandler creates a new StoryHandler.
func NewStoryHandler(svc *service.StoryService) *StoryHandler {
	return &StoryHandler{service: svc}
}

// ListStories handles GET /projects/{projectId}/stories.
// Supports status filtering (?status=backlog,running) and key lookup (?key=S-14).
func (h *StoryHandler) ListStories(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params ListStoriesParams) {
	// Key lookup: return single story
	if params.Key != nil && *params.Key != "" {
		story, err := h.service.GetByKey(r.Context(), projectID, *params.Key)
		if err != nil {
			writeErrorResponse(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toAPIStory(story))
		return
	}

	page := 1
	perPage := 20
	if params.Page != nil && *params.Page > 0 {
		page = *params.Page
	}
	if params.PerPage != nil && *params.PerPage > 0 {
		perPage = *params.PerPage
	}

	// Status filtering
	if params.Status != nil && *params.Status != "" {
		statuses := parseStatusParam(*params.Status)
		result, err := h.service.ListByStatus(r.Context(), projectID, statuses, page, perPage)
		if err != nil {
			writeErrorResponse(w, err)
			return
		}
		writeStoryListResponse(w, result, page, perPage)
		return
	}

	// Default: list all stories for project
	result, err := h.service.ListByProject(r.Context(), projectID, page, perPage)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeStoryListResponse(w, result, page, perPage)
}

// CreateStory handles POST /projects/{projectId}/stories.
// Only admin users can create stories.
func (h *StoryHandler) CreateStory(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	if !middleware.IsAdmin(r.Context()) {
		writeErrorResponse(w, errors.NewForbidden("Admin access required"))
		return
	}

	var req CreateStoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, errors.NewValidation("body", "invalid JSON"))
		return
	}

	params := service.CreateStoryParams{
		ProjectID:          projectID,
		Key:                req.Key,
		Title:              req.Title,
		Objective:          req.Objective,
		AcceptanceCriteria: req.AcceptanceCriteria,
	}
	if req.EpicId != nil {
		id := uuid.UUID(*req.EpicId)
		params.EpicID = &id
	}
	if req.TargetFiles != nil {
		params.TargetFiles = *req.TargetFiles
	}
	if req.DependsOn != nil {
		params.DependsOn = *req.DependsOn
	}
	if req.Scope != nil {
		s := string(*req.Scope)
		params.Scope = &s
	}
	if req.Status != nil {
		params.Status = string(*req.Status)
	}

	story, err := h.service.Create(r.Context(), params)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toAPIStory(story))
}

// GetStory handles GET /projects/{projectId}/stories/{storyId}.
func (h *StoryHandler) GetStory(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, storyID StoryIdPath) {
	story, err := h.service.GetByID(r.Context(), storyID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIStory(story))
}

// UpdateStory handles PUT /projects/{projectId}/stories/{storyId}.
// Only admin users can update stories.
func (h *StoryHandler) UpdateStory(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, storyID StoryIdPath) {
	if !middleware.IsAdmin(r.Context()) {
		writeErrorResponse(w, errors.NewForbidden("Admin access required"))
		return
	}

	var req UpdateStoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, errors.NewValidation("body", "invalid JSON"))
		return
	}

	params := service.UpdateStoryParams{
		ID:                 storyID,
		Title:              req.Title,
		Objective:          req.Objective,
		AcceptanceCriteria: req.AcceptanceCriteria,
	}
	if req.EpicId != nil {
		id := uuid.UUID(*req.EpicId)
		params.EpicID = &id
	}
	if req.TargetFiles != nil {
		params.TargetFiles = req.TargetFiles
	}
	if req.DependsOn != nil {
		params.DependsOn = req.DependsOn
	}
	if req.Scope != nil {
		s := string(*req.Scope)
		params.Scope = &s
	}
	if req.Status != nil {
		s := string(*req.Status)
		params.Status = &s
	}

	story, err := h.service.Update(r.Context(), params)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIStory(story))
}

// DeleteStory handles DELETE /projects/{projectId}/stories/{storyId}.
// Only admin users can delete stories.
func (h *StoryHandler) DeleteStory(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, storyID StoryIdPath) {
	if !middleware.IsAdmin(r.Context()) {
		writeErrorResponse(w, errors.NewForbidden("Admin access required"))
		return
	}

	if err := h.service.Delete(r.Context(), storyID); err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ImportStories handles POST /projects/{projectId}/stories/import.
// Only admin users can import stories.
func (h *StoryHandler) ImportStories(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	if !middleware.IsAdmin(r.Context()) {
		writeErrorResponse(w, errors.NewForbidden("Admin access required"))
		return
	}

	var req ImportStoriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, errors.NewValidation("body", "invalid JSON"))
		return
	}

	if strings.TrimSpace(req.Content) == "" {
		writeErrorResponse(w, errors.NewValidation("content", "must not be empty"))
		return
	}

	parsed := markdown.ParseStoryMarkdown(req.Content)
	inputs := make([]service.ImportStoryInput, len(parsed))
	for i, p := range parsed {
		inputs[i] = service.ImportStoryInput{
			Key:                p.Key,
			Title:              p.Title,
			Epic:               p.Epic,
			DependsOn:          p.DependsOn,
			Scope:              p.Scope,
			Status:             p.Status,
			AcceptanceCriteria: p.AcceptanceCriteria,
			ParseError:         p.ParseError,
		}
	}

	result, err := h.service.Import(r.Context(), projectID, inputs)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	apiErrors := make([]ImportStoryError, len(result.Errors))
	for i, e := range result.Errors {
		apiErrors[i] = ImportStoryError{
			Key:     e.Key,
			Message: e.Message,
			Code:    e.Code,
		}
	}

	resp := ImportStoriesResult{
		Imported: result.Imported,
		Updated:  result.Updated,
		Failed:   result.Failed,
		Errors:   apiErrors,
	}

	writeJSON(w, http.StatusOK, resp)
}

// toAPIStory converts a domain Story to the API Story type.
func toAPIStory(s *model.Story) Story {
	story := Story{
		Id:        s.ID,
		ProjectId: s.ProjectID,
		Key:       s.Key,
		Title:     s.Title,
		Status:    StoryStatus(s.Status),
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
	if s.EpicID != nil {
		story.EpicId = s.EpicID
	}
	if s.Objective != nil {
		story.Objective = s.Objective
	}
	if s.Scope != nil {
		scope := StoryScope(*s.Scope)
		story.Scope = &scope
	}
	if s.AcceptanceCriteria != nil {
		story.AcceptanceCriteria = s.AcceptanceCriteria
	}
	if s.TargetFiles != nil {
		story.TargetFiles = &s.TargetFiles
	}
	if s.DependsOn != nil {
		story.DependsOn = &s.DependsOn
	}
	return story
}

// writeStoryListResponse writes a paginated story list response.
func writeStoryListResponse(w http.ResponseWriter, result *service.StoryListResult, page, perPage int) {
	resp := StoryList{
		Data: make([]Story, len(result.Stories)),
		Pagination: Pagination{
			Total:   int(result.Total),
			Page:    page,
			PerPage: perPage,
		},
	}
	for i, s := range result.Stories {
		resp.Data[i] = toAPIStory(s)
	}
	writeJSON(w, http.StatusOK, resp)
}

// parseStatusParam parses a comma-separated status string into a slice.
func parseStatusParam(s string) []string {
	parts := strings.Split(s, ",")
	statuses := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			statuses = append(statuses, trimmed)
		}
	}
	return statuses
}
