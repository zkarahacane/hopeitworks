package planning

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// Compile-time check that Factory implements port.PlanningSourceFactory.
var _ port.PlanningSourceFactory = (*Factory)(nil)

// Factory resolves the PlanningSourceAdapter for a given source kind, parallel to
// git.DefaultGitProviderFactory. It obtains the GitHub PAT through the single
// resolution seam (port.GitCredentialResolver): a stored PAT connection, else the
// legacy env path.
type Factory struct {
	projectRepo port.ProjectRepository
	resolver    port.GitCredentialResolver
	logger      *slog.Logger
}

// NewFactory creates a new planning source Factory. The resolver is the
// GitConnectionService (token resolution + C1 status self-heal).
func NewFactory(projectRepo port.ProjectRepository, resolver port.GitCredentialResolver, logger *slog.Logger) *Factory {
	return &Factory{projectRepo: projectRepo, resolver: resolver, logger: logger}
}

// For resolves the adapter for kind. markdown is live; github_projects resolves the
// project's PAT through the credential seam, builds an authenticated githubv4
// client, and returns the adapter wrapped so a definitive auth failure (401/403)
// during import self-heals the stored connection status (C1). An unknown kind is
// rejected. The service maps a resolution error to SOURCE_ERROR (HTTP 422).
func (f *Factory) For(ctx context.Context, projectID uuid.UUID, kind port.SourceKind) (port.PlanningSourceAdapter, error) {
	switch kind {
	case port.SourceMarkdown:
		return NewMarkdownAdapter(), nil
	case port.SourceGitHub:
		if f.projectRepo == nil || f.resolver == nil {
			return nil, fmt.Errorf("github planning adapter unavailable: no credential resolver configured")
		}
		tok, err := f.resolver.TokenForProject(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("resolve github planning adapter: resolve token: %w", err)
		}
		if tok.Value == "" {
			return nil, fmt.Errorf("no github token available: connect this project to GitHub in project settings")
		}
		adapter := NewGitHubProjectsAdapter(NewGitHubClient(ctx, tok.Value), f.logger)
		return newReconcilingAdapter(adapter, f.resolver, projectID), nil
	default:
		return nil, fmt.Errorf("unsupported planning source: %s", kind)
	}
}

// reconcilingAdapter wraps a PlanningSourceAdapter so a Fetch auth failure self-heals
// the stored connection status (C1) on the import path.
type reconcilingAdapter struct {
	inner     port.PlanningSourceAdapter
	resolver  port.GitCredentialResolver
	projectID uuid.UUID
}

func newReconcilingAdapter(inner port.PlanningSourceAdapter, resolver port.GitCredentialResolver, projectID uuid.UUID) port.PlanningSourceAdapter {
	if resolver == nil {
		return inner
	}
	return &reconcilingAdapter{inner: inner, resolver: resolver, projectID: projectID}
}

func (a *reconcilingAdapter) Kind() port.SourceKind { return a.inner.Kind() }

func (a *reconcilingAdapter) Fetch(ctx context.Context, projectID uuid.UUID, cfg port.ImportConfig) (*port.FetchResult, error) {
	res, err := a.inner.Fetch(ctx, projectID, cfg)
	if err != nil {
		a.resolver.ReconcileFromOperationError(ctx, a.projectID, err)
	}
	return res, err
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
