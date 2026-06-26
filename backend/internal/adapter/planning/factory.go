package planning

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// Compile-time check that Factory implements port.PlanningSourceFactory.
var _ port.PlanningSourceFactory = (*Factory)(nil)

// Factory resolves the PlanningSourceAdapter for a given source kind, parallel to
// git.DefaultGitProviderFactory. It carries projectRepo + logger now so the
// Phase-3 github_projects case can resolve the project's PAT (via its git token
// env) without changing this constructor's signature or the main.go wiring.
type Factory struct {
	projectRepo port.ProjectRepository
	logger      *slog.Logger
}

// NewFactory creates a new planning source Factory.
func NewFactory(projectRepo port.ProjectRepository, logger *slog.Logger) *Factory {
	return &Factory{projectRepo: projectRepo, logger: logger}
}

// For resolves the adapter for kind. markdown is live; github_projects resolves the
// project's PAT (from its git token env, GITHUB_TOKEN fallback), builds an
// authenticated githubv4 client, and returns the adapter. An unknown kind is
// rejected. The service maps a resolution error to SOURCE_ERROR (HTTP 422).
func (f *Factory) For(ctx context.Context, projectID uuid.UUID, kind port.SourceKind) (port.PlanningSourceAdapter, error) {
	switch kind {
	case port.SourceMarkdown:
		return NewMarkdownAdapter(), nil
	case port.SourceGitHub:
		if f.projectRepo == nil {
			return nil, fmt.Errorf("github planning adapter unavailable: no project repository configured")
		}
		project, err := f.projectRepo.GetByID(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("resolve github planning adapter: get project: %w", err)
		}
		token := resolveGitHubToken(project.GitTokenEnv)
		if token == "" {
			return nil, fmt.Errorf("no github token available: set the project's git token env var or GITHUB_TOKEN")
		}
		return NewGitHubProjectsAdapter(NewGitHubClient(ctx, token), f.logger), nil
	default:
		return nil, fmt.Errorf("unsupported planning source: %s", kind)
	}
}

// resolveGitHubToken reads the PAT from the env var named by gitTokenEnv, falling
// back to GITHUB_TOKEN (replicates git.resolveGitToken, which is unexported).
func resolveGitHubToken(gitTokenEnv *string) string {
	if gitTokenEnv != nil && *gitTokenEnv != "" {
		if v := os.Getenv(*gitTokenEnv); v != "" {
			return v
		}
	}
	return os.Getenv("GITHUB_TOKEN")
}

// normalizeScope validates a raw scope string against the story scope enum
// {backend, frontend, shared} (case-insensitive, normalized to lowercase). A
// recognized value returns a pointer to the canonical form; anything else returns
// (nil, warning) so the importer never writes an out-of-enum scope (§16.8). An
// empty raw value is "absent" => (nil, nil) so the service preserves on update.
func normalizeScope(key, raw string) (*string, *port.ImportWarning) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	lower := strings.ToLower(trimmed)
	switch lower {
	case model.StoryScopeBackend, model.StoryScopeFrontend, model.StoryScopeShared:
		return &lower, nil
	default:
		return nil, &port.ImportWarning{
			Key:     key,
			Code:    "INVALID_SCOPE",
			Message: fmt.Sprintf("ignored out-of-enum scope %q (expected backend|frontend|shared)", raw),
		}
	}
}
