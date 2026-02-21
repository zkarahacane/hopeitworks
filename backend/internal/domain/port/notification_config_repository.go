package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// NotificationConfigRepository defines the persistence interface for notification configs.
type NotificationConfigRepository interface {
	// Insert creates a new notification config and returns it.
	Insert(ctx context.Context, cfg *model.NotificationConfig) (*model.NotificationConfig, error)

	// Get retrieves a notification config by ID.
	Get(ctx context.Context, id uuid.UUID) (*model.NotificationConfig, error)

	// ListByProject returns all notification configs for a project.
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*model.NotificationConfig, error)

	// Update updates an existing notification config and returns it.
	Update(ctx context.Context, cfg *model.NotificationConfig) (*model.NotificationConfig, error)

	// Delete removes a notification config by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// ListEnabledByProject returns only enabled notification configs for a project.
	ListEnabledByProject(ctx context.Context, projectID uuid.UUID) ([]*model.NotificationConfig, error)
}
