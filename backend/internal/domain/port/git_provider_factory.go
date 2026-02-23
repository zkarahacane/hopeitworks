package port

import (
	"context"

	"github.com/google/uuid"
)

// GitProviderFactory resolves the appropriate GitProvider for a given project.
type GitProviderFactory interface {
	ForProjectID(ctx context.Context, projectID uuid.UUID) (GitProvider, error)
}
