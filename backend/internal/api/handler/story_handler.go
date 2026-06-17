package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/adapter/markdown"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// StoryHandler implements story-related HTTP handlers.
type StoryHandler struct {
	service *service.StoryService
	runRepo port.RunRepository
}

// NewStoryHandler creates a new StoryHandler. runRepo is used to populate each
// story's latest_run for the live board; it may be nil in tests that don't
// exercise that field.
func NewStoryHandler(svc *service.StoryService, runRepo port.RunRepository) *StoryHandler {
	return &StoryHandler{service: svc, runRepo: runRepo}
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
		writeJSON(w, http.StatusOK, toAPIStory(story, h.latestRunFor(r.Context(), story.ID)))
		return
	}

	page, perPage := paginationDefaults(params.Page, params.PerPage)

	// Status filtering
	if params.Status != nil && *params.Status != "" {
		statuses := parseStatusParam(*params.Status)
		result, err := h.service.ListByStatus(r.Context(), projectID, statuses, page, perPage)
		if err != nil {
			writeErrorResponse(w, err)
			return
		}
		h.writeStoryListResponse(r.Context(), w, result, page, perPage)
		return
	}

	// Default: list all stories for project
	result, err := h.service.ListByProject(r.Context(), projectID, page, perPage)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	h.writeStoryListResponse(r.Context(), w, result, page, perPage)
}

// CreateStory handles POST /projects/{projectId}/stories.
// Only admin users can create stories.
func (h *StoryHandler) CreateStory(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	if !requireAdmin(w, r) {
		return
	}

	var req CreateStoryRequest
	if !decodeJSONBody(w, r, &req) {
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

	// A freshly created story has no run yet, so latest_run is nil.
	writeJSON(w, http.StatusCreated, toAPIStory(story, nil))
}

// GetStory handles GET /projects/{projectId}/stories/{storyId}.
func (h *StoryHandler) GetStory(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, storyID StoryIdPath) {
	story, err := h.service.GetByID(r.Context(), storyID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIStory(story, h.latestRunFor(r.Context(), story.ID)))
}

// UpdateStory handles PUT /projects/{projectId}/stories/{storyId}.
// Only admin users can update stories.
func (h *StoryHandler) UpdateStory(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, storyID StoryIdPath) {
	if !requireAdmin(w, r) {
		return
	}

	var req UpdateStoryRequest
	if !decodeJSONBody(w, r, &req) {
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

	writeJSON(w, http.StatusOK, toAPIStory(story, h.latestRunFor(r.Context(), story.ID)))
}

// DeleteStory handles DELETE /projects/{projectId}/stories/{storyId}.
// Only admin users can delete stories.
func (h *StoryHandler) DeleteStory(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, storyID StoryIdPath) {
	if !requireAdmin(w, r) {
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
	if !requireAdmin(w, r) {
		return
	}

	var req ImportStoriesRequest
	if !decodeJSONBody(w, r, &req) {
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

// toAPIStory converts a domain Story to the API Story type. latest may be nil
// when the story has no run.
func toAPIStory(s *model.Story, latest *model.LatestRun) Story {
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
	story.LatestRun = toAPILatestRun(latest)
	return story
}

// toAPILatestRun maps a domain LatestRun (and its optional current step) to the
// API type. Returns nil when latest is nil.
func toAPILatestRun(latest *model.LatestRun) *LatestRun {
	if latest == nil {
		return nil
	}
	lr := &LatestRun{
		Id:     latest.ID,
		Status: latest.Status,
	}
	if cs := latest.CurrentStep; cs != nil {
		lr.CurrentStep = &LatestRunStep{
			Id:          cs.ID,
			Name:        cs.Name,
			ActionType:  cs.ActionType,
			Status:      cs.Status,
			Index:       cs.Index,
			Total:       cs.Total,
			ContainerId: cs.ContainerID,
		}
	}
	return lr
}

// latestRunFor fetches a single story's latest run, swallowing errors (the field
// is best-effort enrichment and must not fail the request).
func (h *StoryHandler) latestRunFor(ctx context.Context, storyID uuid.UUID) *model.LatestRun {
	if h.runRepo == nil {
		return nil
	}
	latest, err := h.runRepo.GetLatestRunByStory(ctx, storyID)
	if err != nil {
		return nil
	}
	return latest
}

// latestRunsFor batch-fetches latest runs for many stories, avoiding N+1.
// Returns an empty map on error or when no run repo is configured.
func (h *StoryHandler) latestRunsFor(ctx context.Context, stories []*model.Story) map[uuid.UUID]*model.LatestRun {
	if h.runRepo == nil || len(stories) == 0 {
		return map[uuid.UUID]*model.LatestRun{}
	}
	ids := make([]uuid.UUID, len(stories))
	for i, s := range stories {
		ids[i] = s.ID
	}
	runs, err := h.runRepo.GetLatestRunsByStories(ctx, ids)
	if err != nil {
		return map[uuid.UUID]*model.LatestRun{}
	}
	return runs
}

// writeStoryListResponse writes a paginated story list response, populating each
// story's latest_run via a single batch query.
func (h *StoryHandler) writeStoryListResponse(ctx context.Context, w http.ResponseWriter, result *service.StoryListResult, page, perPage int) {
	latestRuns := h.latestRunsFor(ctx, result.Stories)
	resp := StoryList{
		Data: make([]Story, len(result.Stories)),
		Pagination: Pagination{
			Total:   int(result.Total),
			Page:    page,
			PerPage: perPage,
		},
	}
	for i, s := range result.Stories {
		resp.Data[i] = toAPIStory(s, latestRuns[s.ID])
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
