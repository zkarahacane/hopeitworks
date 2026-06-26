package action_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/adapter/action"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// --- git_branch-specific mocks ---

type remoteBranchCall struct {
	RepoURL    string
	BranchName string
	BaseBranch string
}

type gbMockGitProvider struct {
	mu                   sync.Mutex
	createRemoteBranchFn func(ctx context.Context, repoURL, branchName, baseBranch string) error
	remoteBranchCalls    []remoteBranchCall
}

func (m *gbMockGitProvider) CloneRepo(_ context.Context, _ string, _ string) error { return nil }
func (m *gbMockGitProvider) CreateBranch(_ context.Context, _ string, _ string) error {
	return nil
}
func (m *gbMockGitProvider) CreateRemoteBranch(ctx context.Context, repoURL, branchName, baseBranch string) error {
	m.mu.Lock()
	m.remoteBranchCalls = append(m.remoteBranchCalls, remoteBranchCall{RepoURL: repoURL, BranchName: branchName, BaseBranch: baseBranch})
	m.mu.Unlock()
	if m.createRemoteBranchFn != nil {
		return m.createRemoteBranchFn(ctx, repoURL, branchName, baseBranch)
	}
	return nil
}
func (m *gbMockGitProvider) Push(_ context.Context, _ string, _ string) error { return nil }
func (m *gbMockGitProvider) CreatePR(_ context.Context, _ string, _ string, _ string, _ string) (string, error) {
	return "", nil
}
func (m *gbMockGitProvider) CreateRemotePR(_ context.Context, _ string, _ string, _ string, _ string, _ string) (string, error) {
	return "", nil
}
func (m *gbMockGitProvider) MergePR(_ context.Context, _ string, _ string) error { return nil }
func (m *gbMockGitProvider) GetCIStatus(_ context.Context, _ string) (string, error) {
	return ciStatusPass, nil
}
func (m *gbMockGitProvider) GetRemoteCIStatus(_ context.Context, _ string) (string, error) {
	return ciStatusPass, nil
}
func (m *gbMockGitProvider) GetPRDiff(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (m *gbMockGitProvider) getRemoteBranchCalls() []remoteBranchCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]remoteBranchCall, len(m.remoteBranchCalls))
	copy(result, m.remoteBranchCalls)
	return result
}

type gbMockGitProviderFactory struct {
	provider port.GitProvider
	err      error
}

func (m *gbMockGitProviderFactory) ForProjectID(_ context.Context, _ uuid.UUID) (port.GitProvider, error) {
	return m.provider, m.err
}

type gbMockStoryRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*model.Story, error)
}

func (m *gbMockStoryRepo) Create(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *gbMockStoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Story, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, apperrors.NewNotFound("story", id)
}
func (m *gbMockStoryRepo) GetByKey(_ context.Context, _ uuid.UUID, _ string) (*model.Story, error) {
	return nil, nil
}
func (m *gbMockStoryRepo) GetBySourceRef(_ context.Context, _ uuid.UUID, _, _ string) (*model.Story, error) {
	return nil, nil
}
func (m *gbMockStoryRepo) CreateFromImport(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *gbMockStoryRepo) UpdateFromImport(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *gbMockStoryRepo) UpdateProvenanceOnly(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *gbMockStoryRepo) ListByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *gbMockStoryRepo) ListByStatus(_ context.Context, _ uuid.UUID, _ []string, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *gbMockStoryRepo) ListByEpic(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *gbMockStoryRepo) CountByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *gbMockStoryRepo) CountByStatus(_ context.Context, _ uuid.UUID, _ []string) (int64, error) {
	return 0, nil
}
func (m *gbMockStoryRepo) Update(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *gbMockStoryRepo) UpdateStoryCurrentStage(_ context.Context, id uuid.UUID, currentStage *string) (*model.Story, error) {
	return &model.Story{ID: id, CurrentStage: currentStage}, nil
}
func (m *gbMockStoryRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

type gbMockProjectRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*model.Project, error)
}

func (m *gbMockProjectRepo) Create(_ context.Context, p *model.Project) (*model.Project, error) {
	return p, nil
}
func (m *gbMockProjectRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, apperrors.NewNotFound("project", id)
}
func (m *gbMockProjectRepo) List(_ context.Context, _, _ int32) ([]*model.Project, error) {
	return nil, nil
}
func (m *gbMockProjectRepo) Count(_ context.Context) (int64, error) { return 0, nil }
func (m *gbMockProjectRepo) Update(_ context.Context, p *model.Project) (*model.Project, error) {
	return p, nil
}
func (m *gbMockProjectRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (m *gbMockProjectRepo) IncrementCircuitBreakerCount(_ context.Context, _ uuid.UUID) (*model.Project, error) {
	return nil, nil
}
func (m *gbMockProjectRepo) ResetCircuitBreaker(_ context.Context, _ uuid.UUID) (*model.Project, error) {
	return nil, nil
}

// --- Helpers ---

const (
	testRepoURL    = "https://github.com/owner/repo"
	testBaseBranch = "main"
)

func gbRunCtx(cfg map[string]string, metadata map[string]any) *model.RunContext {
	runID := uuid.New()
	stepID := uuid.New()
	projectID := uuid.New()
	storyID := uuid.New()

	return &model.RunContext{
		Run: &model.Run{
			ID:        runID,
			ProjectID: projectID,
			StoryID:   storyID,
			Status:    model.RunStatusRunning,
		},
		RunStep: &model.RunStep{
			ID:     stepID,
			RunID:  runID,
			Action: "git_branch",
			Status: model.StepStatusRunning,
			Config: cfg,
		},
		ProjectID: projectID,
		StoryID:   storyID,
		Metadata:  metadata,
	}
}

func gbDefaultProjectRepo(_ uuid.UUID) *gbMockProjectRepo {
	repoURL := testRepoURL
	return &gbMockProjectRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Project, error) {
			return &model.Project{ID: id, RepoURL: &repoURL}, nil
		},
	}
}

// --- Tests ---

func TestGitBranchAction_Name(t *testing.T) {
	a := action.NewGitBranchAction(&gbMockGitProviderFactory{}, nil, nil, testLogger())
	if a.Name() != "git_branch" {
		t.Fatalf("expected Name() = %q, got %q", "git_branch", a.Name())
	}
}

func TestGitBranchAction_Execute_HappyPath(t *testing.T) {
	storyID := uuid.New()
	projectID := uuid.New()
	gitProvider := &gbMockGitProvider{}
	factory := &gbMockGitProviderFactory{provider: gitProvider}
	storyRepo := &gbMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, Key: "S-03", Title: "Add login page"}, nil
		},
	}
	projectRepo := gbDefaultProjectRepo(projectID)

	a := action.NewGitBranchAction(factory, storyRepo, projectRepo, testLogger())

	cfg := map[string]string{
		"branch_pattern": "feat/{story_key}-{slug}",
		"base_branch":    testBaseBranch,
	}
	runCtx := gbRunCtx(cfg, map[string]any{})
	runCtx.StoryID = storyID
	runCtx.ProjectID = projectID

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// Verify CreateRemoteBranch was called with correct args
	calls := gitProvider.getRemoteBranchCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 CreateRemoteBranch call, got %d", len(calls))
	}
	if calls[0].RepoURL != testRepoURL {
		t.Fatalf("expected repo_url %q, got %q", testRepoURL, calls[0].RepoURL)
	}
	if calls[0].BranchName != "feat/S-03-add-login-page" {
		t.Fatalf("expected branch %q, got %q", "feat/S-03-add-login-page", calls[0].BranchName)
	}
	if calls[0].BaseBranch != testBaseBranch {
		t.Fatalf("expected base_branch %q, got %q", testBaseBranch, calls[0].BaseBranch)
	}

	// Verify metadata was set
	branchName, ok := runCtx.Metadata["branch_name"].(string)
	if !ok || branchName != "feat/S-03-add-login-page" {
		t.Fatalf("expected Metadata[branch_name] = %q, got %v", "feat/S-03-add-login-page", runCtx.Metadata["branch_name"])
	}
}

func TestGitBranchAction_SlugDerivation(t *testing.T) {
	tests := []struct {
		title    string
		expected string
	}{
		{"Add login page", "add-login-page"},
		{"Add login (OAuth)", "add-login-oauth"},
		{"Hello World!", "hello-world"},
		{"  spaces  ", "spaces"},
		{"UPPERCASE Title", "uppercase-title"},
		{"multiple---hyphens", "multiple-hyphens"},
		{"special!@#$%^&*chars", "special-chars"},
		{"trailing-", "trailing"},
		{"-leading", "leading"},
		{"a", "a"},
		{"123-numeric", "123-numeric"},
	}

	for _, tc := range tests {
		t.Run(tc.title, func(t *testing.T) {
			projectID := uuid.New()
			gitProvider := &gbMockGitProvider{}
			factory := &gbMockGitProviderFactory{provider: gitProvider}
			storyRepo := &gbMockStoryRepo{
				getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
					return &model.Story{ID: id, Key: "S-01", Title: tc.title}, nil
				},
			}
			projectRepo := gbDefaultProjectRepo(projectID)

			a := action.NewGitBranchAction(factory, storyRepo, projectRepo, testLogger())

			cfg := map[string]string{
				"branch_pattern": "feat/{story_key}-{slug}",
			}
			runCtx := gbRunCtx(cfg, map[string]any{})
			runCtx.ProjectID = projectID

			err := a.Execute(context.Background(), runCtx)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}

			calls := gitProvider.getRemoteBranchCalls()
			if len(calls) != 1 {
				t.Fatalf("expected 1 CreateRemoteBranch call, got %d", len(calls))
			}

			expectedBranch := "feat/S-01-" + tc.expected
			if calls[0].BranchName != expectedBranch {
				t.Fatalf("expected branch %q, got %q", expectedBranch, calls[0].BranchName)
			}
		})
	}
}

func TestGitBranchAction_Execute_DefaultBaseBranch(t *testing.T) {
	projectID := uuid.New()
	gitProvider := &gbMockGitProvider{}
	factory := &gbMockGitProviderFactory{provider: gitProvider}
	storyRepo := &gbMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, Key: "S-05", Title: "Some feature"}, nil
		},
	}
	projectRepo := gbDefaultProjectRepo(projectID)

	a := action.NewGitBranchAction(factory, storyRepo, projectRepo, testLogger())

	// No base_branch in config — should default to "main"
	cfg := map[string]string{}
	runCtx := gbRunCtx(cfg, map[string]any{})
	runCtx.ProjectID = projectID

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	calls := gitProvider.getRemoteBranchCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 CreateRemoteBranch call, got %d", len(calls))
	}

	// Verify branch was created (default pattern "feat/{story_key}-{slug}")
	if calls[0].BranchName != "feat/S-05-some-feature" {
		t.Fatalf("expected branch %q, got %q", "feat/S-05-some-feature", calls[0].BranchName)
	}
	if calls[0].BaseBranch != testBaseBranch {
		t.Fatalf("expected base_branch %q, got %q", testBaseBranch, calls[0].BaseBranch)
	}
}

func TestGitBranchAction_Execute_CustomFixPattern(t *testing.T) {
	projectID := uuid.New()
	gitProvider := &gbMockGitProvider{}
	factory := &gbMockGitProviderFactory{provider: gitProvider}
	storyRepo := &gbMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, Key: "BUG-7", Title: "Fix null pointer"}, nil
		},
	}
	projectRepo := gbDefaultProjectRepo(projectID)

	a := action.NewGitBranchAction(factory, storyRepo, projectRepo, testLogger())

	cfg := map[string]string{
		"branch_pattern": "fix/{story_key}-{slug}",
	}
	runCtx := gbRunCtx(cfg, map[string]any{})
	runCtx.ProjectID = projectID

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	calls := gitProvider.getRemoteBranchCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 CreateRemoteBranch call, got %d", len(calls))
	}
	if !strings.HasPrefix(calls[0].BranchName, "fix/") {
		t.Fatalf("expected branch to start with %q, got %q", "fix/", calls[0].BranchName)
	}
	if calls[0].BranchName != "fix/BUG-7-fix-null-pointer" {
		t.Fatalf("expected branch %q, got %q", "fix/BUG-7-fix-null-pointer", calls[0].BranchName)
	}
}

func TestGitBranchAction_Execute_GitProviderFailure(t *testing.T) {
	projectID := uuid.New()
	gitProvider := &gbMockGitProvider{
		createRemoteBranchFn: func(_ context.Context, _, _, _ string) error {
			return fmt.Errorf("git API failed: generic error")
		},
	}
	factory := &gbMockGitProviderFactory{provider: gitProvider}
	storyRepo := &gbMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, Key: "S-01", Title: "Test"}, nil
		},
	}
	projectRepo := gbDefaultProjectRepo(projectID)

	a := action.NewGitBranchAction(factory, storyRepo, projectRepo, testLogger())

	cfg := map[string]string{}
	runCtx := gbRunCtx(cfg, map[string]any{})
	runCtx.ProjectID = projectID

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error when CreateRemoteBranch fails with non-idempotent error")
	}
	if !strings.Contains(err.Error(), "create branch") {
		t.Fatalf("expected error containing %q, got %q", "create branch", err.Error())
	}

	// Verify metadata was NOT set
	if _, exists := runCtx.Metadata["branch_name"]; exists {
		t.Fatal("expected branch_name metadata to NOT be set on failure")
	}
}

func TestGitBranchAction_Execute_BranchAlreadyExists_Idempotent(t *testing.T) {
	projectID := uuid.New()
	gitProvider := &gbMockGitProvider{
		createRemoteBranchFn: func(_ context.Context, _, _, _ string) error {
			return fmt.Errorf("API returned status 409: branch already exists")
		},
	}
	factory := &gbMockGitProviderFactory{provider: gitProvider}
	storyRepo := &gbMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, Key: "S-02", Title: "Idempotent test"}, nil
		},
	}
	projectRepo := gbDefaultProjectRepo(projectID)

	a := action.NewGitBranchAction(factory, storyRepo, projectRepo, testLogger())

	cfg := map[string]string{}
	runCtx := gbRunCtx(cfg, map[string]any{})
	runCtx.ProjectID = projectID

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error when branch already exists (idempotent), got %v", err)
	}

	// Verify metadata WAS set (idempotent success)
	branchName, exists := runCtx.Metadata["branch_name"].(string)
	if !exists {
		t.Fatal("expected branch_name metadata to be set on 409 (already exists)")
	}
	if !strings.Contains(branchName, "S-02") {
		t.Fatalf("expected branch_name to contain story key, got %q", branchName)
	}
}

func TestGitBranchAction_Execute_StoryNotFound(t *testing.T) {
	projectID := uuid.New()
	gitProvider := &gbMockGitProvider{}
	factory := &gbMockGitProviderFactory{provider: gitProvider}
	storyRepo := &gbMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return nil, apperrors.NewNotFound("story", id)
		},
	}
	projectRepo := gbDefaultProjectRepo(projectID)

	a := action.NewGitBranchAction(factory, storyRepo, projectRepo, testLogger())

	cfg := map[string]string{}
	runCtx := gbRunCtx(cfg, map[string]any{})
	runCtx.ProjectID = projectID

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error when story not found")
	}
	if !strings.Contains(err.Error(), "fetch story") {
		t.Fatalf("expected error containing %q, got %q", "fetch story", err.Error())
	}

	// Verify GitProvider was NOT called
	calls := gitProvider.getRemoteBranchCalls()
	if len(calls) != 0 {
		t.Fatalf("expected no CreateRemoteBranch calls when story fetch fails, got %d", len(calls))
	}
}

func TestGitBranchAction_Execute_MissingRepoURL(t *testing.T) {
	projectID := uuid.New()
	gitProvider := &gbMockGitProvider{}
	factory := &gbMockGitProviderFactory{provider: gitProvider}
	storyRepo := &gbMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, Key: "S-01", Title: "Test"}, nil
		},
	}
	// Project has no repo_url
	projectRepo := &gbMockProjectRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Project, error) {
			return &model.Project{ID: id}, nil
		},
	}

	a := action.NewGitBranchAction(factory, storyRepo, projectRepo, testLogger())

	cfg := map[string]string{}
	runCtx := gbRunCtx(cfg, map[string]any{})
	runCtx.ProjectID = projectID

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error when repo_url is missing")
	}
	if !strings.Contains(err.Error(), "repo_url") {
		t.Fatalf("expected error containing %q, got %q", "repo_url", err.Error())
	}

	// Verify GitProvider was NOT called
	calls := gitProvider.getRemoteBranchCalls()
	if len(calls) != 0 {
		t.Fatalf("expected no CreateRemoteBranch calls when repo_url missing, got %d", len(calls))
	}
}

func TestGitBranchAction_Execute_NilConfig(t *testing.T) {
	projectID := uuid.New()
	gitProvider := &gbMockGitProvider{}
	factory := &gbMockGitProviderFactory{provider: gitProvider}
	storyRepo := &gbMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, Key: "S-01", Title: "Test"}, nil
		},
	}
	projectRepo := gbDefaultProjectRepo(projectID)

	a := action.NewGitBranchAction(factory, storyRepo, projectRepo, testLogger())

	// nil Config on RunStep
	runCtx := gbRunCtx(nil, map[string]any{})
	runCtx.ProjectID = projectID

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	calls := gitProvider.getRemoteBranchCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 CreateRemoteBranch call, got %d", len(calls))
	}
	// Default pattern should be used
	if !strings.HasPrefix(calls[0].BranchName, "feat/S-01-") {
		t.Fatalf("expected branch to use default pattern, got %q", calls[0].BranchName)
	}
}

func (m *gbMockStoryRepo) CountByEpicGroupedByStatus(_ context.Context, _ uuid.UUID) (model.StoryCounts, error) {
	return model.StoryCounts{}, nil
}
