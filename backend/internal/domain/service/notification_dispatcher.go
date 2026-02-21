package service

import (
	"context"
	"log/slog"
	"slices"
	"sync"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// NotificationDispatcher subscribes to the EventBus for all projects and
// routes events to the appropriate Notifier adapters.
type NotificationDispatcher struct {
	eventSub    port.EventSubscriber
	repo        port.NotificationConfigRepository
	projectRepo port.ProjectRepository
	notifiers   map[string]port.Notifier

	mu       sync.Mutex
	cleanups []func()
}

// NewNotificationDispatcher creates a new NotificationDispatcher.
func NewNotificationDispatcher(
	eventSub port.EventSubscriber,
	repo port.NotificationConfigRepository,
	projectRepo port.ProjectRepository,
	notifiers map[string]port.Notifier,
) *NotificationDispatcher {
	return &NotificationDispatcher{
		eventSub:    eventSub,
		repo:        repo,
		projectRepo: projectRepo,
		notifiers:   notifiers,
	}
}

// Start subscribes to the EventBus for all known projects and begins
// dispatching notifications until ctx is cancelled.
func (d *NotificationDispatcher) Start(ctx context.Context) {
	go func() {
		if err := d.run(ctx); err != nil && err != context.Canceled {
			slog.Warn("notification dispatcher exited with error", "err", err)
		}
	}()
}

// run fetches all known projects, subscribes to events for each, and fans in
// all event channels to dispatch loop. It also handles context cancellation.
func (d *NotificationDispatcher) run(ctx context.Context) error {
	projects, err := d.projectRepo.List(ctx, 1000, 0)
	if err != nil {
		slog.Warn("notification dispatcher: failed to list projects at startup", "err", err)
		// Run with no subscriptions — will still handle future projects if subscribed
		projects = nil
	}

	merged := make(chan model.Event, 256)

	for _, p := range projects {
		ch, cleanup, subErr := d.eventSub.Subscribe(ctx, p.ID)
		if subErr != nil {
			slog.Warn("notification dispatcher: subscribe failed for project", "project_id", p.ID, "err", subErr)
			continue
		}
		d.mu.Lock()
		d.cleanups = append(d.cleanups, cleanup)
		d.mu.Unlock()

		go d.fanIn(ctx, ch, merged)
	}

	// Also subscribe to the nil UUID to catch any events without a project context
	nilProjectID := uuid.Nil
	if len(projects) == 0 {
		ch, cleanup, subErr := d.eventSub.Subscribe(ctx, nilProjectID)
		if subErr == nil {
			d.mu.Lock()
			d.cleanups = append(d.cleanups, cleanup)
			d.mu.Unlock()
			go d.fanIn(ctx, ch, merged)
		}
	}

	for {
		select {
		case <-ctx.Done():
			d.stopAll()
			return ctx.Err()
		case event, ok := <-merged:
			if !ok {
				return nil
			}
			d.dispatch(ctx, event)
		}
	}
}

// fanIn reads from src and writes to dst until src is closed or ctx is done.
func (d *NotificationDispatcher) fanIn(ctx context.Context, src <-chan model.Event, dst chan<- model.Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-src:
			if !ok {
				return
			}
			select {
			case dst <- event:
			case <-ctx.Done():
				return
			}
		}
	}
}

// stopAll calls all registered cleanup functions.
func (d *NotificationDispatcher) stopAll() {
	d.mu.Lock()
	cleanups := d.cleanups
	d.cleanups = nil
	d.mu.Unlock()

	for _, cleanup := range cleanups {
		cleanup()
	}
}

// dispatch fetches enabled configs for the event's project, filters by event
// name, and calls the matching Notifier. Errors are logged but never fatal.
func (d *NotificationDispatcher) dispatch(ctx context.Context, event model.Event) {
	configs, err := d.repo.ListEnabledByProject(ctx, event.ProjectID)
	if err != nil {
		slog.Warn("notification dispatch: list configs failed",
			"project_id", event.ProjectID,
			"err", err,
		)
		return
	}

	eventName := event.EventName()
	for _, cfg := range configs {
		if !slices.Contains(cfg.EventsFilter, eventName) {
			continue
		}
		notifier, ok := d.notifiers[cfg.ChannelType]
		if !ok {
			continue
		}
		if sendErr := notifier.Send(ctx, event, cfg.Config); sendErr != nil {
			slog.Warn("notification send failed",
				"channel_type", cfg.ChannelType,
				"config_id", cfg.ID,
				"err", sendErr,
			)
		}
	}
}
