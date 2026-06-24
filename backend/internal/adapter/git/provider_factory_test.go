package git

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// factoryMockProjectRepo implements port.ProjectRepository for factory testing.
type factoryMockProjectRepo struct {
	project *model.Project
	err     error
}

func (m *factoryMockProjectRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.Project, error) {
	return m.project, m.err
}

func (m *factoryMockProjectRepo) Create(_ context.Context, p *model.Project) (*model.Project, error) {
	return p, nil
}
func (m *factoryMockProjectRepo) List(_ context.Context, _, _ int32) ([]*model.Project, error) {
	return nil, nil
}
func (m *factoryMockProjectRepo) Count(_ context.Context) (int64, error) { return 0, nil }
func (m *factoryMockProjectRepo) Update(_ context.Context, p *model.Project) (*model.Project, error) {
	return p, nil
}
func (m *factoryMockProjectRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (m *factoryMockProjectRepo) IncrementCircuitBreakerCount(_ context.Context, _ uuid.UUID) (*model.Project, error) {
	return nil, nil
}
func (m *factoryMockProjectRepo) ResetCircuitBreaker(_ context.Context, _ uuid.UUID) (*model.Project, error) {
	return nil, nil
}

func TestDefaultGitProviderFactory_ForProjectID_GitHub(t *testing.T) {
	projectID := uuid.New()
	repo := &factoryMockProjectRepo{
		project: &model.Project{
			ID:          projectID,
			GitProvider: "github",
		},
	}

	runner := newMockCommandRunner()

	factory := NewGitProviderFactory(repo, runner, testLogger())

	provider, err := factory.ForProjectID(context.Background(), projectID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it returns the API-based GitHub adapter (no gh CLI dependency).
	if _, ok := provider.(*GitHubAPIAdapter); !ok {
		t.Fatalf("expected *GitHubAPIAdapter, got %T", provider)
	}
}

func TestDefaultGitProviderFactory_ForProjectID_EmptyDefaultsToGitHub(t *testing.T) {
	projectID := uuid.New()
	repo := &factoryMockProjectRepo{
		project: &model.Project{
			ID:          projectID,
			GitProvider: "",
		},
	}

	runner := newMockCommandRunner()

	factory := NewGitProviderFactory(repo, runner, testLogger())

	provider, err := factory.ForProjectID(context.Background(), projectID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := provider.(*GitHubAPIAdapter); !ok {
		t.Fatalf("expected *GitHubAPIAdapter for empty git_provider, got %T", provider)
	}
}

func TestDefaultGitProviderFactory_ForProjectID_Gitea(t *testing.T) {
	projectID := uuid.New()
	repoURL := "https://gitea.example.com/org/project"
	repo := &factoryMockProjectRepo{
		project: &model.Project{
			ID:          projectID,
			GitProvider: "gitea",
			RepoURL:     &repoURL,
		},
	}

	runner := newMockCommandRunner()

	factory := NewGitProviderFactory(repo, runner, testLogger())

	provider, err := factory.ForProjectID(context.Background(), projectID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := provider.(*GiteaAPIAdapter); !ok {
		t.Fatalf("expected *GiteaAPIAdapter, got %T", provider)
	}
}

func TestDefaultGitProviderFactory_ForProjectID_UnsupportedProvider(t *testing.T) {
	projectID := uuid.New()
	repo := &factoryMockProjectRepo{
		project: &model.Project{
			ID:          projectID,
			GitProvider: "gitlab",
		},
	}

	factory := NewGitProviderFactory(repo, nil, testLogger())

	_, err := factory.ForProjectID(context.Background(), projectID)
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
}

func TestDefaultGitProviderFactory_ForProjectID_ProjectNotFound(t *testing.T) {
	repo := &factoryMockProjectRepo{
		err: fmt.Errorf("project not found"),
	}

	factory := NewGitProviderFactory(repo, nil, testLogger())

	_, err := factory.ForProjectID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error when project not found")
	}
}

// Ensure the factory implements the interface.
var _ port.GitProviderFactory = (*DefaultGitProviderFactory)(nil)
