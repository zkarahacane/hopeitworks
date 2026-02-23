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
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// --- git_branch-specific mocks ---

type branchCall struct {
	WorkDir    string
	BranchName string
}

type gbMockGitProvider struct {
	mu             sync.Mutex
	createBranchFn func(ctx context.Context, workDir, branchName string) error
	branchCalls    []branchCall
}

func (m *gbMockGitProvider) CloneRepo(_ context.Context, _ string, _ string) error { return nil }
func (m *gbMockGitProvider) CreateBranch(ctx context.Context, workDir, branchName string) error {
	m.mu.Lock()
	m.branchCalls = append(m.branchCalls, branchCall{WorkDir: workDir, BranchName: branchName})
	m.mu.Unlock()
	if m.createBranchFn != nil {
		return m.createBranchFn(ctx, workDir, branchName)
	}
	return nil
}
func (m *gbMockGitProvider) Push(_ context.Context, _ string, _ string) error { return nil }
func (m *gbMockGitProvider) CreatePR(_ context.Context, _ string, _ string, _ string, _ string) (string, error) {
	return "", nil
}
func (m *gbMockGitProvider) MergePR(_ context.Context, _ string, _ string) error { return nil }
func (m *gbMockGitProvider) GetCIStatus(_ context.Context, _ string) (string, error) {
	return "pass", nil
}
func (m *gbMockGitProvider) GetPRDiff(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (m *gbMockGitProvider) getBranchCalls() []branchCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]branchCall, len(m.branchCalls))
	copy(result, m.branchCalls)
	return result
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
func (m *gbMockStoryRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

// --- Helpers ---

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

// --- Tests ---

func TestGitBranchAction_Name(t *testing.T) {
	a := action.NewGitBranchAction(nil, nil, testLogger())
	if a.Name() != "git_branch" {
		t.Fatalf("expected Name() = %q, got %q", "git_branch", a.Name())
	}
}

func TestGitBranchAction_Execute_HappyPath(t *testing.T) {
	storyID := uuid.New()
	gitProvider := &gbMockGitProvider{}
	storyRepo := &gbMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, Key: "S-03", Title: "Add login page"}, nil
		},
	}

	a := action.NewGitBranchAction(gitProvider, storyRepo, testLogger())

	cfg := map[string]string{
		"branch_pattern": "feat/{story_key}-{slug}",
		"base_branch":    "main",
		"work_dir":       "/tmp/repo",
	}
	runCtx := gbRunCtx(cfg, map[string]any{})
	runCtx.StoryID = storyID

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// Verify CreateBranch was called with correct args
	calls := gitProvider.getBranchCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 CreateBranch call, got %d", len(calls))
	}
	if calls[0].WorkDir != "/tmp/repo" {
		t.Fatalf("expected work_dir %q, got %q", "/tmp/repo", calls[0].WorkDir)
	}
	if calls[0].BranchName != "feat/S-03-add-login-page" {
		t.Fatalf("expected branch %q, got %q", "feat/S-03-add-login-page", calls[0].BranchName)
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
			gitProvider := &gbMockGitProvider{}
			storyRepo := &gbMockStoryRepo{
				getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
					return &model.Story{ID: id, Key: "S-01", Title: tc.title}, nil
				},
			}

			a := action.NewGitBranchAction(gitProvider, storyRepo, testLogger())

			cfg := map[string]string{
				"branch_pattern": "feat/{story_key}-{slug}",
				"work_dir":       "/tmp/repo",
			}
			runCtx := gbRunCtx(cfg, map[string]any{})

			err := a.Execute(context.Background(), runCtx)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}

			calls := gitProvider.getBranchCalls()
			if len(calls) != 1 {
				t.Fatalf("expected 1 CreateBranch call, got %d", len(calls))
			}

			expectedBranch := "feat/S-01-" + tc.expected
			if calls[0].BranchName != expectedBranch {
				t.Fatalf("expected branch %q, got %q", expectedBranch, calls[0].BranchName)
			}
		})
	}
}

func TestGitBranchAction_Execute_DefaultBaseBranch(t *testing.T) {
	gitProvider := &gbMockGitProvider{}
	storyRepo := &gbMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, Key: "S-05", Title: "Some feature"}, nil
		},
	}

	a := action.NewGitBranchAction(gitProvider, storyRepo, testLogger())

	// No base_branch in config — should default to "main"
	cfg := map[string]string{
		"work_dir": "/tmp/repo",
	}
	runCtx := gbRunCtx(cfg, map[string]any{})

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	calls := gitProvider.getBranchCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 CreateBranch call, got %d", len(calls))
	}

	// Verify branch was created (default pattern "feat/{story_key}-{slug}")
	if calls[0].BranchName != "feat/S-05-some-feature" {
		t.Fatalf("expected branch %q, got %q", "feat/S-05-some-feature", calls[0].BranchName)
	}
}

func TestGitBranchAction_Execute_CustomFixPattern(t *testing.T) {
	gitProvider := &gbMockGitProvider{}
	storyRepo := &gbMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, Key: "BUG-7", Title: "Fix null pointer"}, nil
		},
	}

	a := action.NewGitBranchAction(gitProvider, storyRepo, testLogger())

	cfg := map[string]string{
		"branch_pattern": "fix/{story_key}-{slug}",
		"work_dir":       "/tmp/repo",
	}
	runCtx := gbRunCtx(cfg, map[string]any{})

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	calls := gitProvider.getBranchCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 CreateBranch call, got %d", len(calls))
	}
	if !strings.HasPrefix(calls[0].BranchName, "fix/") {
		t.Fatalf("expected branch to start with %q, got %q", "fix/", calls[0].BranchName)
	}
	if calls[0].BranchName != "fix/BUG-7-fix-null-pointer" {
		t.Fatalf("expected branch %q, got %q", "fix/BUG-7-fix-null-pointer", calls[0].BranchName)
	}
}

func TestGitBranchAction_Execute_GitProviderFailure(t *testing.T) {
	gitProvider := &gbMockGitProvider{
		createBranchFn: func(_ context.Context, _, _ string) error {
			return fmt.Errorf("git checkout failed: branch already exists")
		},
	}
	storyRepo := &gbMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, Key: "S-01", Title: "Test"}, nil
		},
	}

	a := action.NewGitBranchAction(gitProvider, storyRepo, testLogger())

	cfg := map[string]string{
		"work_dir": "/tmp/repo",
	}
	runCtx := gbRunCtx(cfg, map[string]any{})

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error when CreateBranch fails")
	}
	if !strings.Contains(err.Error(), "create branch") {
		t.Fatalf("expected error containing %q, got %q", "create branch", err.Error())
	}

	// Verify metadata was NOT set
	if _, exists := runCtx.Metadata["branch_name"]; exists {
		t.Fatal("expected branch_name metadata to NOT be set on failure")
	}
}

func TestGitBranchAction_Execute_StoryNotFound(t *testing.T) {
	gitProvider := &gbMockGitProvider{}
	storyRepo := &gbMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return nil, apperrors.NewNotFound("story", id)
		},
	}

	a := action.NewGitBranchAction(gitProvider, storyRepo, testLogger())

	cfg := map[string]string{
		"work_dir": "/tmp/repo",
	}
	runCtx := gbRunCtx(cfg, map[string]any{})

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error when story not found")
	}
	if !strings.Contains(err.Error(), "fetch story") {
		t.Fatalf("expected error containing %q, got %q", "fetch story", err.Error())
	}

	// Verify GitProvider was NOT called
	calls := gitProvider.getBranchCalls()
	if len(calls) != 0 {
		t.Fatalf("expected no CreateBranch calls when story fetch fails, got %d", len(calls))
	}
}

func TestGitBranchAction_Execute_MissingWorkDir(t *testing.T) {
	gitProvider := &gbMockGitProvider{}
	storyRepo := &gbMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, Key: "S-01", Title: "Test"}, nil
		},
	}

	a := action.NewGitBranchAction(gitProvider, storyRepo, testLogger())

	// No work_dir in config or metadata
	cfg := map[string]string{}
	runCtx := gbRunCtx(cfg, map[string]any{})

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error when work_dir is missing")
	}
	if !strings.Contains(err.Error(), "work_dir") {
		t.Fatalf("expected error containing %q, got %q", "work_dir", err.Error())
	}

	// Verify GitProvider was NOT called
	calls := gitProvider.getBranchCalls()
	if len(calls) != 0 {
		t.Fatalf("expected no CreateBranch calls when work_dir missing, got %d", len(calls))
	}
}

func TestGitBranchAction_Execute_WorkDirFromMetadata(t *testing.T) {
	gitProvider := &gbMockGitProvider{}
	storyRepo := &gbMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, Key: "S-01", Title: "Test"}, nil
		},
	}

	a := action.NewGitBranchAction(gitProvider, storyRepo, testLogger())

	// work_dir only in metadata, not in config
	cfg := map[string]string{}
	runCtx := gbRunCtx(cfg, map[string]any{
		"work_dir": "/tmp/from-metadata",
	})

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	calls := gitProvider.getBranchCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 CreateBranch call, got %d", len(calls))
	}
	if calls[0].WorkDir != "/tmp/from-metadata" {
		t.Fatalf("expected work_dir %q, got %q", "/tmp/from-metadata", calls[0].WorkDir)
	}
}

func TestGitBranchAction_Execute_NilConfig(t *testing.T) {
	gitProvider := &gbMockGitProvider{}
	storyRepo := &gbMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, Key: "S-01", Title: "Test"}, nil
		},
	}

	a := action.NewGitBranchAction(gitProvider, storyRepo, testLogger())

	// nil Config on RunStep, work_dir in metadata
	runCtx := gbRunCtx(nil, map[string]any{
		"work_dir": "/tmp/repo",
	})

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	calls := gitProvider.getBranchCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 CreateBranch call, got %d", len(calls))
	}
	// Default pattern should be used
	if !strings.HasPrefix(calls[0].BranchName, "feat/S-01-") {
		t.Fatalf("expected branch to use default pattern, got %q", calls[0].BranchName)
	}
}
