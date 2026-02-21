package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

// EpicRunHandler implements epic run-related HTTP handlers.
type EpicRunHandler struct {
	service *service.EpicRunService
}

// NewEpicRunHandler creates a new EpicRunHandler.
func NewEpicRunHandler(svc *service.EpicRunService) *EpicRunHandler {
	return &EpicRunHandler{service: svc}
}

// LaunchEpicRun handles POST /projects/{projectId}/epics/{epicId}/runs.
func (h *EpicRunHandler) LaunchEpicRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, epicID EpicIdPath) {
	epicRun, err := h.service.LaunchEpicRun(r.Context(), projectID, epicID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := EpicRunAccepted{
		EpicRunId:    epicRun.ID,
		Status:       Scheduling,
		StoriesCount: len(epicRun.Stories),
	}

	writeJSON(w, http.StatusAccepted, resp)
}

// GetEpicRun handles GET /projects/{projectId}/epic-runs/{epicRunId}.
func (h *EpicRunHandler) GetEpicRun(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, epicRunID EpicRunIdPath) {
	epicRun, err := h.service.GetEpicRun(r.Context(), epicRunID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	stories := make([]EpicRunStory, len(epicRun.Stories))
	for i, s := range epicRun.Stories {
		stories[i] = EpicRunStory{
			StoryId:    s.StoryID,
			GroupIndex: s.GroupIndex,
			Status:     s.Status,
		}
		if s.RunID != nil && *s.RunID != uuid.Nil {
			stories[i].RunId = s.RunID
		}
	}

	resp := EpicRunDetail{
		Id:        epicRun.ID,
		ProjectId: epicRun.ProjectID,
		EpicId:    epicRun.EpicID,
		Status:    EpicRunDetailStatus(epicRun.Status),
		CreatedAt: epicRun.CreatedAt,
		Stories:   stories,
	}
	if epicRun.CompletedAt != nil {
		resp.CompletedAt = epicRun.CompletedAt
	}

	writeJSON(w, http.StatusOK, resp)
}
