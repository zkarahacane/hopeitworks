package git

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// Compile-time check that DefaultGitProviderFactory implements GitProviderFactory.
var _ port.GitProviderFactory = (*DefaultGitProviderFactory)(nil)

// DefaultGitProviderFactory resolves a GitProvider based on project configuration.
type DefaultGitProviderFactory struct {
	projectRepo port.ProjectRepository
	runner      port.CommandRunner
	logger      *slog.Logger
}

// NewGitProviderFactory creates a new DefaultGitProviderFactory.
func NewGitProviderFactory(projectRepo port.ProjectRepository, runner port.CommandRunner, logger *slog.Logger) *DefaultGitProviderFactory {
	return &DefaultGitProviderFactory{
		projectRepo: projectRepo,
		runner:      runner,
		logger:      logger,
	}
}

// ForProjectID resolves the appropriate GitProvider for the given project.
func (f *DefaultGitProviderFactory) ForProjectID(ctx context.Context, projectID uuid.UUID) (port.GitProvider, error) {
	project, err := f.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("resolve git provider: get project: %w", err)
	}

	switch project.GitProvider {
	case "github", "":
		// API-based adapter (google/go-github) — no `gh` CLI dependency.
		// GhCliAdapter is kept in the package as a fallback/reference.
		token := resolveGitToken(project.GitTokenEnv)
		return NewGitHubAPIAdapter("", token, f.runner, f.logger), nil
	case "gitea":
		token := resolveGitToken(project.GitTokenEnv)
		baseURL := extractBaseURL(safeDeref(project.RepoURL))
		return NewGiteaAPIAdapter(baseURL, token, f.runner, f.logger), nil
	default:
		return nil, fmt.Errorf("unsupported git provider: %s", project.GitProvider)
	}
}

// resolveGitToken reads the git token from the environment variable specified
// by gitTokenEnv, falling back to GITHUB_TOKEN for backward compatibility.
func resolveGitToken(gitTokenEnv *string) string {
	if gitTokenEnv != nil && *gitTokenEnv != "" {
		if v := os.Getenv(*gitTokenEnv); v != "" {
			return v
		}
	}
	return os.Getenv("GITHUB_TOKEN")
}

// safeDeref safely dereferences a string pointer, returning empty string if nil.
func safeDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
