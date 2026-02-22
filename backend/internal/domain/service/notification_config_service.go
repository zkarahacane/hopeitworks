package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// NotificationConfigService provides business logic for managing notification configs.
type NotificationConfigService struct {
	repo      port.NotificationConfigRepository
	notifiers map[string]port.Notifier
}

// NewNotificationConfigService creates a new NotificationConfigService.
func NewNotificationConfigService(repo port.NotificationConfigRepository, notifiers map[string]port.Notifier) *NotificationConfigService {
	return &NotificationConfigService{repo: repo, notifiers: notifiers}
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

// Test sends a test notification using the given notification config.
func (s *NotificationConfigService) Test(ctx context.Context, id uuid.UUID) error {
	cfg, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	notifier, ok := s.notifiers[cfg.ChannelType]
	if !ok {
		return errors.NewValidation("channel_type",
			fmt.Sprintf("unsupported notification channel type: %s", cfg.ChannelType))
	}

	payload, _ := json.Marshal(map[string]string{
		"message": "This is a test notification from hopeitworks",
	})

	testEvent := model.Event{
		ID:         uuid.New(),
		ProjectID:  cfg.ProjectID,
		EntityType: "notification",
		EntityID:   id,
		Action:     "test",
		Payload:    payload,
		CreatedAt:  time.Now(),
	}

	return notifier.Send(ctx, testEvent, cfg.Config)
}
