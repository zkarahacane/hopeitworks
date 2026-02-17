package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

// --- Mocks ---

type mockEventSubscriber struct {
	ch chan model.Event
}

func (m *mockEventSubscriber) Subscribe(_ context.Context, _ uuid.UUID) (<-chan model.Event, func(), error) {
	return m.ch, func() {}, nil
}

func (m *mockEventSubscriber) Close() error { return nil }

type mockNotificationConfigRepo struct {
	configs []*model.NotificationConfig
	err     error
}

func (m *mockNotificationConfigRepo) Insert(_ context.Context, cfg *model.NotificationConfig) (*model.NotificationConfig, error) {
	return cfg, nil
}

func (m *mockNotificationConfigRepo) Get(_ context.Context, _ uuid.UUID) (*model.NotificationConfig, error) {
	return nil, nil
}

func (m *mockNotificationConfigRepo) ListByProject(_ context.Context, _ uuid.UUID) ([]*model.NotificationConfig, error) {
	return m.configs, m.err
}

func (m *mockNotificationConfigRepo) Update(_ context.Context, cfg *model.NotificationConfig) (*model.NotificationConfig, error) {
	return cfg, nil
}

func (m *mockNotificationConfigRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

func (m *mockNotificationConfigRepo) ListEnabledByProject(_ context.Context, _ uuid.UUID) ([]*model.NotificationConfig, error) {
	return m.configs, m.err
}

type mockProjectRepo struct{}

func (m *mockProjectRepo) Create(_ context.Context, p *model.Project) (*model.Project, error) {
	return p, nil
}

func (m *mockProjectRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.Project, error) {
	return nil, nil
}

func (m *mockProjectRepo) List(_ context.Context, _, _ int32) ([]*model.Project, error) {
	return []*model.Project{
		{ID: uuid.New()},
	}, nil
}

func (m *mockProjectRepo) Count(_ context.Context) (int64, error) { return 0, nil }

func (m *mockProjectRepo) Update(_ context.Context, p *model.Project) (*model.Project, error) {
	return p, nil
}

func (m *mockProjectRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

type mockNotifier struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (m *mockNotifier) Send(_ context.Context, _ model.Event, _ map[string]string) error {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	return m.err
}

func (m *mockNotifier) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

// --- Tests ---

func TestDispatcher_MatchingEvent_CallsNotifier(t *testing.T) {
	projectID := uuid.New()
	configID := uuid.New()

	eventCh := make(chan model.Event, 1)
	sub := &mockEventSubscriber{ch: eventCh}

	cfgRepo := &mockNotificationConfigRepo{
		configs: []*model.NotificationConfig{
			{
				ID:           configID,
				ProjectID:    projectID,
				ChannelType:  "discord",
				Config:       map[string]string{"url": "https://example.com"},
				EventsFilter: []string{"run.completed"},
				Enabled:      true,
			},
		},
	}

	notifier := &mockNotifier{}
	dispatcher := service.NewNotificationDispatcher(
		sub,
		cfgRepo,
		&mockProjectRepo{},
		map[string]port.Notifier{"discord": notifier},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatcher.Start(ctx)

	eventCh <- model.Event{
		ID:         uuid.New(),
		ProjectID:  projectID,
		EntityType: "run",
		EntityID:   uuid.New(),
		Action:     "completed",
		CreatedAt:  time.Now(),
	}

	// Give dispatcher time to process
	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	if notifier.Calls() != 1 {
		t.Errorf("expected notifier called 1 time, got %d", notifier.Calls())
	}
}

func TestDispatcher_EventNotInFilter_DoesNotCallNotifier(t *testing.T) {
	projectID := uuid.New()

	eventCh := make(chan model.Event, 1)
	sub := &mockEventSubscriber{ch: eventCh}

	cfgRepo := &mockNotificationConfigRepo{
		configs: []*model.NotificationConfig{
			{
				ID:           uuid.New(),
				ProjectID:    projectID,
				ChannelType:  "discord",
				Config:       map[string]string{"url": "https://example.com"},
				EventsFilter: []string{"run.failed"},
				Enabled:      true,
			},
		},
	}

	notifier := &mockNotifier{}
	dispatcher := service.NewNotificationDispatcher(
		sub,
		cfgRepo,
		&mockProjectRepo{},
		map[string]port.Notifier{"discord": notifier},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatcher.Start(ctx)

	eventCh <- model.Event{
		ID:         uuid.New(),
		ProjectID:  projectID,
		EntityType: "run",
		EntityID:   uuid.New(),
		Action:     "completed", // not in filter (filter is run.failed)
		CreatedAt:  time.Now(),
	}

	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	if notifier.Calls() != 0 {
		t.Errorf("expected notifier NOT called, got %d calls", notifier.Calls())
	}
}

func TestDispatcher_NotifierError_DoesNotStopOtherDispatches(t *testing.T) {
	projectID := uuid.New()

	eventCh := make(chan model.Event, 2)
	sub := &mockEventSubscriber{ch: eventCh}

	errorNotifier := &mockNotifier{err: errors.New("send failed")}
	okNotifier := &mockNotifier{}

	cfgRepo := &mockNotificationConfigRepo{
		configs: []*model.NotificationConfig{
			{
				ID:           uuid.New(),
				ProjectID:    projectID,
				ChannelType:  "discord",
				Config:       map[string]string{"url": "https://example.com"},
				EventsFilter: []string{"run.completed"},
				Enabled:      true,
			},
			{
				ID:           uuid.New(),
				ProjectID:    projectID,
				ChannelType:  "webhook",
				Config:       map[string]string{"url": "https://example.com/hook"},
				EventsFilter: []string{"run.completed"},
				Enabled:      true,
			},
		},
	}

	dispatcher := service.NewNotificationDispatcher(
		sub,
		cfgRepo,
		&mockProjectRepo{},
		map[string]port.Notifier{
			"discord": errorNotifier,
			"webhook": okNotifier,
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatcher.Start(ctx)

	eventCh <- model.Event{
		ID:         uuid.New(),
		ProjectID:  projectID,
		EntityType: "run",
		EntityID:   uuid.New(),
		Action:     "completed",
		CreatedAt:  time.Now(),
	}

	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	if errorNotifier.Calls() != 1 {
		t.Errorf("expected error notifier called once, got %d", errorNotifier.Calls())
	}
	if okNotifier.Calls() != 1 {
		t.Errorf("expected ok notifier called once, got %d", okNotifier.Calls())
	}
}
