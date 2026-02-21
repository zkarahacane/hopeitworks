package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// NotificationHandler implements notification config HTTP handlers.
type NotificationHandler struct {
	service *service.NotificationConfigService
}

// NewNotificationHandler creates a new NotificationHandler.
func NewNotificationHandler(svc *service.NotificationConfigService) *NotificationHandler {
	return &NotificationHandler{service: svc}
}

// ListNotificationConfigs handles GET /projects/{projectId}/notifications.
func (h *NotificationHandler) ListNotificationConfigs(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	configs, err := h.service.ListByProject(r.Context(), projectID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	data := make([]NotificationConfig, len(configs))
	for i, c := range configs {
		data[i] = toAPINotificationConfig(c)
	}

	writeJSON(w, http.StatusOK, NotificationConfigList{Data: data})
}

// CreateNotificationConfig handles POST /projects/{projectId}/notifications.
func (h *NotificationHandler) CreateNotificationConfig(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	var req CreateNotificationConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, errors.NewValidation("body", "invalid JSON"))
		return
	}

	if string(req.ChannelType) != model.ChannelTypeDiscord && string(req.ChannelType) != model.ChannelTypeWebhook {
		writeErrorResponse(w, errors.NewValidation("channel_type", "must be 'discord' or 'webhook'"))
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	eventsFilter := req.EventsFilter
	if eventsFilter == nil {
		eventsFilter = []string{}
	}

	configMap := req.Config
	if configMap == nil {
		configMap = map[string]string{}
	}

	cfg, err := h.service.Create(r.Context(), projectID, string(req.ChannelType), configMap, eventsFilter, enabled)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toAPINotificationConfig(cfg))
}

// UpdateNotificationConfig handles PUT /projects/{projectId}/notifications/{notificationId}.
func (h *NotificationHandler) UpdateNotificationConfig(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, notificationID NotificationIdPath) {
	var req UpdateNotificationConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, errors.NewValidation("body", "invalid JSON"))
		return
	}

	configMap := req.Config
	if configMap == nil {
		configMap = map[string]string{}
	}

	eventsFilter := req.EventsFilter
	if eventsFilter == nil {
		eventsFilter = []string{}
	}

	cfg, err := h.service.Update(r.Context(), uuid.UUID(notificationID), string(req.ChannelType), configMap, eventsFilter, req.Enabled)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPINotificationConfig(cfg))
}

// DeleteNotificationConfig handles DELETE /projects/{projectId}/notifications/{notificationId}.
func (h *NotificationHandler) DeleteNotificationConfig(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, notificationID NotificationIdPath) {
	if err := h.service.Delete(r.Context(), uuid.UUID(notificationID)); err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// toAPINotificationConfig converts a domain NotificationConfig to the API type.
func toAPINotificationConfig(c *model.NotificationConfig) NotificationConfig {
	configMap := make(map[string]string, len(c.Config))
	for k, v := range c.Config {
		configMap[k] = v
	}

	eventsFilter := c.EventsFilter
	if eventsFilter == nil {
		eventsFilter = []string{}
	}

	return NotificationConfig{
		Id:           c.ID,
		ProjectId:    c.ProjectID,
		ChannelType:  NotificationConfigChannelType(c.ChannelType),
		Config:       configMap,
		EventsFilter: eventsFilter,
		Enabled:      c.Enabled,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}
