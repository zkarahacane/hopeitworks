package git

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// Compile-time check that DefaultGitProviderFactory implements GitProviderFactory.
var _ port.GitProviderFactory = (*DefaultGitProviderFactory)(nil)

// DefaultGitProviderFactory resolves a GitProvider based on project configuration.
// It obtains the credential through the single resolution seam
// (port.GitCredentialResolver): a stored PAT connection, else the legacy env path.
type DefaultGitProviderFactory struct {
	projectRepo port.ProjectRepository
	resolver    port.GitCredentialResolver
	runner      port.CommandRunner
	logger      *slog.Logger
}

// NewGitProviderFactory creates a new DefaultGitProviderFactory. The resolver is the
// GitConnectionService (token resolution + C1 status self-heal).
func NewGitProviderFactory(projectRepo port.ProjectRepository, resolver port.GitCredentialResolver, runner port.CommandRunner, logger *slog.Logger) *DefaultGitProviderFactory {
	return &DefaultGitProviderFactory{
		projectRepo: projectRepo,
		resolver:    resolver,
		runner:      runner,
		logger:      logger,
	}
}

// ForProjectID resolves the appropriate GitProvider for the given project. The
// returned provider is wrapped so a definitive auth failure (401/403) during real
// operations self-heals the stored connection status (C1).
func (f *DefaultGitProviderFactory) ForProjectID(ctx context.Context, projectID uuid.UUID) (port.GitProvider, error) {
	project, err := f.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("resolve git provider: get project: %w", err)
	}

	switch project.GitProvider {
	case "github", "":
		tok, terr := f.resolver.TokenForProject(ctx, projectID)
		if terr != nil {
			return nil, fmt.Errorf("resolve git provider: resolve token: %w", terr)
		}
		// API-based adapter (google/go-github) — no `gh` CLI dependency.
		inner := NewGitHubAPIAdapter("", tok.Value, f.runner, f.logger)
		return newReconcilingProvider(inner, f.resolver, projectID), nil
	case "gitea":
		tok, terr := f.resolver.TokenForProject(ctx, projectID)
		if terr != nil {
			return nil, fmt.Errorf("resolve git provider: resolve token: %w", terr)
		}
		baseURL := extractBaseURL(safeDeref(project.RepoURL))
		inner := NewGiteaAPIAdapter(baseURL, tok.Value, f.runner, f.logger)
		return newReconcilingProvider(inner, f.resolver, projectID), nil
	default:
		return nil, fmt.Errorf("unsupported git provider: %s", project.GitProvider)
	}
}

// safeDeref safely dereferences a string pointer, returning empty string if nil.
func safeDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
