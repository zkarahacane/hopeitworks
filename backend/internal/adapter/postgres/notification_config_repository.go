package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure NotificationConfigRepo implements port.NotificationConfigRepository at compile time.
var _ port.NotificationConfigRepository = (*NotificationConfigRepo)(nil)

// NotificationConfigRepo implements port.NotificationConfigRepository using sqlc-generated queries.
type NotificationConfigRepo struct {
	queries *Queries
}

// NewNotificationConfigRepository creates a new NotificationConfigRepo.
func NewNotificationConfigRepository(queries *Queries) *NotificationConfigRepo {
	return &NotificationConfigRepo{queries: queries}
}

// Insert creates a new notification config and returns it.
func (r *NotificationConfigRepo) Insert(ctx context.Context, cfg *model.NotificationConfig) (*model.NotificationConfig, error) {
	configJSON, err := marshalStringMap(cfg.Config)
	if err != nil {
		return nil, apperrors.NewInternal("failed to marshal notification config", err)
	}

	eventsJSON, err := marshalStringSlice(cfg.EventsFilter)
	if err != nil {
		return nil, apperrors.NewInternal("failed to marshal events_filter", err)
	}

	row, err := r.queries.InsertNotificationConfig(ctx, InsertNotificationConfigParams{
		ProjectID:    cfg.ProjectID,
		ChannelType:  cfg.ChannelType,
		Config:       configJSON,
		EventsFilter: eventsJSON,
		Enabled:      cfg.Enabled,
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to insert notification config", err)
	}

	return toDomainNotificationConfig(row)
}

// Get retrieves a notification config by ID.
func (r *NotificationConfigRepo) Get(ctx context.Context, id uuid.UUID) (*model.NotificationConfig, error) {
	row, err := r.queries.GetNotificationConfig(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("notification_config", id)
		}
		return nil, apperrors.NewInternal("failed to get notification config", err)
	}

	return toDomainNotificationConfig(row)
}

// ListByProject returns all notification configs for a project.
func (r *NotificationConfigRepo) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*model.NotificationConfig, error) {
	rows, err := r.queries.ListNotificationConfigsByProject(ctx, projectID)
	if err != nil {
		return nil, apperrors.NewInternal("failed to list notification configs", err)
	}

	return toDomainNotificationConfigs(rows)
}

// Update updates an existing notification config and returns it.
func (r *NotificationConfigRepo) Update(ctx context.Context, cfg *model.NotificationConfig) (*model.NotificationConfig, error) {
	configJSON, err := marshalStringMap(cfg.Config)
	if err != nil {
		return nil, apperrors.NewInternal("failed to marshal notification config", err)
	}

	eventsJSON, err := marshalStringSlice(cfg.EventsFilter)
	if err != nil {
		return nil, apperrors.NewInternal("failed to marshal events_filter", err)
	}

	row, err := r.queries.UpdateNotificationConfig(ctx, UpdateNotificationConfigParams{
		ID:           cfg.ID,
		ChannelType:  cfg.ChannelType,
		Config:       configJSON,
		EventsFilter: eventsJSON,
		Enabled:      cfg.Enabled,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("notification_config", cfg.ID)
		}
		return nil, apperrors.NewInternal("failed to update notification config", err)
	}

	return toDomainNotificationConfig(row)
}

// Delete removes a notification config by ID.
func (r *NotificationConfigRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteNotificationConfig(ctx, id); err != nil {
		return apperrors.NewInternal("failed to delete notification config", err)
	}
	return nil
}

// ListEnabledByProject returns only enabled notification configs for a project.
func (r *NotificationConfigRepo) ListEnabledByProject(ctx context.Context, projectID uuid.UUID) ([]*model.NotificationConfig, error) {
	rows, err := r.queries.ListEnabledConfigsByProject(ctx, projectID)
	if err != nil {
		return nil, apperrors.NewInternal("failed to list enabled notification configs", err)
	}

	return toDomainNotificationConfigs(rows)
}

// toDomainNotificationConfig maps a sqlc NotificationConfig to a domain NotificationConfig.
func toDomainNotificationConfig(row NotificationConfig) (*model.NotificationConfig, error) {
	config, err := unmarshalStringMap(row.Config)
	if err != nil {
		return nil, apperrors.NewInternal("failed to unmarshal notification config.config", err)
	}

	eventsFilter, err := unmarshalStringSlice(row.EventsFilter)
	if err != nil {
		return nil, apperrors.NewInternal("failed to unmarshal notification config.events_filter", err)
	}

	return &model.NotificationConfig{
		ID:           row.ID,
		ProjectID:    row.ProjectID,
		ChannelType:  row.ChannelType,
		Config:       config,
		EventsFilter: eventsFilter,
		Enabled:      row.Enabled,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}

// toDomainNotificationConfigs maps a slice of sqlc NotificationConfig rows to domain models.
func toDomainNotificationConfigs(rows []NotificationConfig) ([]*model.NotificationConfig, error) {
	result := make([]*model.NotificationConfig, 0, len(rows))
	for _, row := range rows {
		cfg, err := toDomainNotificationConfig(row)
		if err != nil {
			return nil, err
		}
		result = append(result, cfg)
	}
	return result, nil
}

// marshalStringMap serialises map[string]string to JSON bytes for JSONB storage.
func marshalStringMap(m map[string]string) ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

// unmarshalStringMap deserialises JSON bytes from JSONB into map[string]string.
func unmarshalStringMap(data []byte) (map[string]string, error) {
	if len(data) == 0 {
		return map[string]string{}, nil
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// marshalStringSlice serialises []string to JSON bytes for JSONB storage.
func marshalStringSlice(s []string) ([]byte, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(s)
}

// unmarshalStringSlice deserialises JSON bytes from JSONB into []string.
func unmarshalStringSlice(data []byte) ([]string, error) {
	if len(data) == 0 {
		return []string{}, nil
	}
	var s []string
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return s, nil
}
