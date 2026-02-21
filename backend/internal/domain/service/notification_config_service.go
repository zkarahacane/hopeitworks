package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// NotificationConfigService provides business logic for managing notification configs.
type NotificationConfigService struct {
	repo port.NotificationConfigRepository
}

// NewNotificationConfigService creates a new NotificationConfigService.
func NewNotificationConfigService(repo port.NotificationConfigRepository) *NotificationConfigService {
	return &NotificationConfigService{repo: repo}
}

// Create inserts a new notification config for the given project.
func (s *NotificationConfigService) Create(ctx context.Context, projectID uuid.UUID, channelType string, config map[string]string, eventsFilter []string, enabled bool) (*model.NotificationConfig, error) {
	cfg := &model.NotificationConfig{
		ProjectID:    projectID,
		ChannelType:  channelType,
		Config:       config,
		EventsFilter: eventsFilter,
		Enabled:      enabled,
	}
	return s.repo.Insert(ctx, cfg)
}

// Get retrieves a notification config by ID.
func (s *NotificationConfigService) Get(ctx context.Context, id uuid.UUID) (*model.NotificationConfig, error) {
	return s.repo.Get(ctx, id)
}

// ListByProject returns all notification configs for a project.
func (s *NotificationConfigService) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*model.NotificationConfig, error) {
	return s.repo.ListByProject(ctx, projectID)
}

// Update updates an existing notification config.
func (s *NotificationConfigService) Update(ctx context.Context, id uuid.UUID, channelType string, config map[string]string, eventsFilter []string, enabled bool) (*model.NotificationConfig, error) {
	cfg := &model.NotificationConfig{
		ID:           id,
		ChannelType:  channelType,
		Config:       config,
		EventsFilter: eventsFilter,
		Enabled:      enabled,
	}
	return s.repo.Update(ctx, cfg)
}

// Delete removes a notification config by ID.
func (s *NotificationConfigService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
