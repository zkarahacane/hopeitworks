package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockNotificationConfigRepo implements port.NotificationConfigRepository for testing.
type mockNotificationConfigRepo struct {
	configs map[uuid.UUID]*model.NotificationConfig
}

func newMockNotificationConfigRepo() *mockNotificationConfigRepo {
	return &mockNotificationConfigRepo{
		configs: make(map[uuid.UUID]*model.NotificationConfig),
	}
}

func (m *mockNotificationConfigRepo) Insert(_ context.Context, cfg *model.NotificationConfig) (*model.NotificationConfig, error) {
	cfg.ID = uuid.New()
	cfg.CreatedAt = time.Now()
	cfg.UpdatedAt = time.Now()
	m.configs[cfg.ID] = cfg
	return cfg, nil
}

func (m *mockNotificationConfigRepo) Get(_ context.Context, id uuid.UUID) (*model.NotificationConfig, error) {
	cfg, ok := m.configs[id]
	if !ok {
		return nil, apperrors.NewNotFound("notification_config", id)
	}
	return cfg, nil
}

func (m *mockNotificationConfigRepo) ListByProject(_ context.Context, projectID uuid.UUID) ([]*model.NotificationConfig, error) {
	var result []*model.NotificationConfig
	for _, cfg := range m.configs {
		if cfg.ProjectID == projectID {
			result = append(result, cfg)
		}
	}
	return result, nil
}

func (m *mockNotificationConfigRepo) Update(_ context.Context, cfg *model.NotificationConfig) (*model.NotificationConfig, error) {
	existing, ok := m.configs[cfg.ID]
	if !ok {
		return nil, apperrors.NewNotFound("notification_config", cfg.ID)
	}
	existing.ChannelType = cfg.ChannelType
	existing.Config = cfg.Config
	existing.EventsFilter = cfg.EventsFilter
	existing.Enabled = cfg.Enabled
	existing.UpdatedAt = time.Now()
	return existing, nil
}

func (m *mockNotificationConfigRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.configs[id]; !ok {
		return apperrors.NewNotFound("notification_config", id)
	}
	delete(m.configs, id)
	return nil
}

func (m *mockNotificationConfigRepo) ListEnabledByProject(_ context.Context, projectID uuid.UUID) ([]*model.NotificationConfig, error) {
	var result []*model.NotificationConfig
	for _, cfg := range m.configs {
		if cfg.ProjectID == projectID && cfg.Enabled {
			result = append(result, cfg)
		}
	}
	return result, nil
}

// mockNotifier implements port.Notifier for testing.
type mockNotifierForTest struct {
	sendCalls int
	sendErr   error
}

func (m *mockNotifierForTest) Send(_ context.Context, _ model.Event, _ map[string]string) error {
	m.sendCalls++
	return m.sendErr
}

func TestNotificationConfigService_Test_Success(t *testing.T) {
	repo := newMockNotificationConfigRepo()
	projectID := uuid.New()
	configID := uuid.New()
	repo.configs[configID] = &model.NotificationConfig{
		ID:          configID,
		ProjectID:   projectID,
		ChannelType: model.ChannelTypeWebhook,
		Config:      map[string]string{"url": "https://example.com/webhook"},
		Enabled:     true,
	}

	notifier := &mockNotifierForTest{}
	notifiers := map[string]port.Notifier{
		model.ChannelTypeWebhook: notifier,
	}

	svc := NewNotificationConfigService(repo, notifiers)

	err := svc.Test(context.Background(), configID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notifier.sendCalls != 1 {
		t.Errorf("expected 1 send call, got %d", notifier.sendCalls)
	}
}

func TestNotificationConfigService_Test_NotFound(t *testing.T) {
	repo := newMockNotificationConfigRepo()
	svc := NewNotificationConfigService(repo, nil)

	err := svc.Test(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*apperrors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Category != apperrors.CategoryNotFound {
		t.Errorf("expected not_found category, got %s", domainErr.Category)
	}
}

func TestNotificationConfigService_Test_UnsupportedChannel(t *testing.T) {
	repo := newMockNotificationConfigRepo()
	configID := uuid.New()
	repo.configs[configID] = &model.NotificationConfig{
		ID:          configID,
		ProjectID:   uuid.New(),
		ChannelType: "slack", // unsupported
		Config:      map[string]string{"url": "https://example.com"},
		Enabled:     true,
	}

	svc := NewNotificationConfigService(repo, map[string]port.Notifier{})

	err := svc.Test(context.Background(), configID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*apperrors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Category != apperrors.CategoryValidation {
		t.Errorf("expected validation category, got %s", domainErr.Category)
	}
}

func TestNotificationConfigService_Test_NotifierError(t *testing.T) {
	repo := newMockNotificationConfigRepo()
	configID := uuid.New()
	repo.configs[configID] = &model.NotificationConfig{
		ID:          configID,
		ProjectID:   uuid.New(),
		ChannelType: model.ChannelTypeDiscord,
		Config:      map[string]string{"url": "https://discord.com/api/webhooks/123"},
		Enabled:     true,
	}

	notifier := &mockNotifierForTest{sendErr: fmt.Errorf("connection refused")}
	notifiers := map[string]port.Notifier{
		model.ChannelTypeDiscord: notifier,
	}

	svc := NewNotificationConfigService(repo, notifiers)

	err := svc.Test(context.Background(), configID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
