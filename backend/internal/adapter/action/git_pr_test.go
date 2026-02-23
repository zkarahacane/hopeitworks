package action_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/adapter/action"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

const (
	testPRURL     = "https://github.com/owner/repo/pull/42"
	testPRURLAlt  = "https://github.com/owner/repo/pull/1"
	testWorkDir   = "/tmp/clone/repo"
	testBranchPR  = "feat/S-03-add-login-page"
	testBranchAlt = "feat/S-03-test"
)

// --- Mocks for GitPRAction ---

type prMockGitProvider struct {
	createPRFn func(ctx context.Context, workDir, title, body, baseBranch string) (string, error)
	calls      []prCreatePRCall
}

type prCreatePRCall struct {
	WorkDir    string
	Title      string
	Body       string
	BaseBranch string
}

func (m *prMockGitProvider) CreatePR(ctx context.Context, workDir, title, body, baseBranch string) (string, error) {
	m.calls = append(m.calls, prCreatePRCall{
		WorkDir:    workDir,
		Title:      title,
		Body:       body,
		BaseBranch: baseBranch,
	})
	return m.createPRFn(ctx, workDir, title, body, baseBranch)
}

func (m *prMockGitProvider) CloneRepo(_ context.Context, _ string, _ string) error    { return nil }
func (m *prMockGitProvider) CreateBranch(_ context.Context, _ string, _ string) error { return nil }
func (m *prMockGitProvider) Push(_ context.Context, _ string, _ string) error         { return nil }
func (m *prMockGitProvider) MergePR(_ context.Context, _ string, _ string) error      { return nil }
func (m *prMockGitProvider) GetCIStatus(_ context.Context, _ string) (string, error)  { return "", nil }
func (m *prMockGitProvider) GetPRDiff(_ context.Context, _ string) (string, error)    { return "", nil }

type prMockGitProviderFactory struct {
	provider port.GitProvider
	err      error
}

func (m *prMockGitProviderFactory) ForProjectID(_ context.Context, _ uuid.UUID) (port.GitProvider, error) {
	return m.provider, m.err
}

type prMockStoryRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*model.Story, error)
}

func (m *prMockStoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Story, error) {
	return m.getByIDFn(ctx, id)
}

func (m *prMockStoryRepo) Create(_ context.Context, _ *model.Story) (*model.Story, error) {
	return nil, nil
}
func (m *prMockStoryRepo) GetByKey(_ context.Context, _ uuid.UUID, _ string) (*model.Story, error) {
	return nil, nil
}
func (m *prMockStoryRepo) ListByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *prMockStoryRepo) ListByStatus(_ context.Context, _ uuid.UUID, _ []string, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *prMockStoryRepo) ListByEpic(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *prMockStoryRepo) CountByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *prMockStoryRepo) CountByStatus(_ context.Context, _ uuid.UUID, _ []string) (int64, error) {
	return 0, nil
}
func (m *prMockStoryRepo) Update(_ context.Context, _ *model.Story) (*model.Story, error) {
	return nil, nil
}
func (m *prMockStoryRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

// --- Helpers ---

func buildPRRunCtx(metadata map[string]any) *model.RunContext {
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
			Action: "git_pr",
			Status: model.StepStatusRunning,
		},
		ProjectID: projectID,
		StoryID:   storyID,
		Metadata:  metadata,
	}
}

func newTestStory(storyID uuid.UUID) *model.Story {
	scope := "backend"
	objective := "Add login page with OAuth support"
	return &model.Story{
		ID:        storyID,
		Key:       "S-03",
		Title:     "Add login page",
		Scope:     &scope,
		Objective: &objective,
	}
}

// --- Tests ---

func TestGitPRAction_Name(t *testing.T) {
	a := action.NewGitPRAction(&prMockGitProviderFactory{}, nil, testLogger())
	if a.Name() != "git_pr" {
		t.Fatalf("expected Name() = %q, got %q", "git_pr", a.Name())
	}
}

func TestGitPRAction_Execute_HappyPath(t *testing.T) {
	gitProvider := &prMockGitProvider{
		createPRFn: func(_ context.Context, _, _, _, _ string) (string, error) {
			return testPRURL, nil
		},
	}
	factory := &prMockGitProviderFactory{provider: gitProvider}

	storyID := uuid.New()
	storyRepo := &prMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			if id != storyID {
				return nil, fmt.Errorf("story not found")
			}
			return newTestStory(storyID), nil
		},
	}

	a := action.NewGitPRAction(factory, storyRepo, testLogger())
	runCtx := buildPRRunCtx(map[string]any{
		"branch_name":    testBranchPR,
		"work_dir":       testWorkDir,
		"target_branch":  "develop",
		"title_template": "feat({scope}): {story_title}",
	})
	runCtx.StoryID = storyID

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// Verify pr_url set in metadata
	prURL, ok := runCtx.Metadata["pr_url"].(string)
	if !ok || prURL != testPRURL {
		t.Fatalf("expected Metadata[pr_url] = %q, got %q", testPRURL, prURL)
	}

	// Verify CreatePR was called with correct args
	if len(gitProvider.calls) != 1 {
		t.Fatalf("expected 1 CreatePR call, got %d", len(gitProvider.calls))
	}
	call := gitProvider.calls[0]
	if call.WorkDir != testWorkDir {
		t.Errorf("expected workDir = %q, got %q", testWorkDir, call.WorkDir)
	}
	if call.Title != "feat(backend): Add login page" {
		t.Errorf("expected title = %q, got %q", "feat(backend): Add login page", call.Title)
	}
	if call.BaseBranch != "develop" {
		t.Errorf("expected baseBranch = %q, got %q", "develop", call.BaseBranch)
	}
	// Verify body contains story context
	if !strings.Contains(call.Body, "S-03") {
		t.Errorf("expected body to contain story key, got %q", call.Body)
	}
	if !strings.Contains(call.Body, "Add login page") {
		t.Errorf("expected body to contain story title, got %q", call.Body)
	}
	if !strings.Contains(call.Body, "hopeitworks pipeline") {
		t.Errorf("expected body to contain footer, got %q", call.Body)
	}
}

func TestGitPRAction_Execute_DefaultTargetBranch(t *testing.T) {
	gitProvider := &prMockGitProvider{
		createPRFn: func(_ context.Context, _, _, _, _ string) (string, error) {
			return testPRURLAlt, nil
		},
	}
	factory := &prMockGitProviderFactory{provider: gitProvider}

	storyID := uuid.New()
	storyRepo := &prMockStoryRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Story, error) {
			return newTestStory(storyID), nil
		},
	}

	a := action.NewGitPRAction(factory, storyRepo, testLogger())
	runCtx := buildPRRunCtx(map[string]any{
		"branch_name": testBranchPR,
		"work_dir":    testWorkDir,
		// No target_branch — should default to "main"
	})
	runCtx.StoryID = storyID

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(gitProvider.calls) != 1 {
		t.Fatalf("expected 1 CreatePR call, got %d", len(gitProvider.calls))
	}
	if gitProvider.calls[0].BaseBranch != "main" {
		t.Errorf("expected default baseBranch = %q, got %q", "main", gitProvider.calls[0].BaseBranch)
	}
}

func TestGitPRAction_Execute_DraftFlag(t *testing.T) {
	gitProvider := &prMockGitProvider{
		createPRFn: func(_ context.Context, _, _, _, _ string) (string, error) {
			return testPRURLAlt, nil
		},
	}
	factory := &prMockGitProviderFactory{provider: gitProvider}

	storyID := uuid.New()
	storyRepo := &prMockStoryRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Story, error) {
			return newTestStory(storyID), nil
		},
	}

	a := action.NewGitPRAction(factory, storyRepo, testLogger())
	runCtx := buildPRRunCtx(map[string]any{
		"branch_name": testBranchPR,
		"work_dir":    testWorkDir,
		"draft":       "true",
	})
	runCtx.StoryID = storyID

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(gitProvider.calls) != 1 {
		t.Fatalf("expected 1 CreatePR call, got %d", len(gitProvider.calls))
	}
	if !strings.HasPrefix(gitProvider.calls[0].Title, "[Draft] ") {
		t.Errorf("expected title to start with %q, got %q", "[Draft] ", gitProvider.calls[0].Title)
	}
}

func TestGitPRAction_Execute_MissingBranchName(t *testing.T) {
	gitProvider := &prMockGitProvider{
		createPRFn: func(_ context.Context, _, _, _, _ string) (string, error) {
			return "", nil
		},
	}
	factory := &prMockGitProviderFactory{provider: gitProvider}
	storyRepo := &prMockStoryRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Story, error) {
			return newTestStory(uuid.New()), nil
		},
	}

	a := action.NewGitPRAction(factory, storyRepo, testLogger())
	runCtx := buildPRRunCtx(map[string]any{
		"work_dir": testWorkDir,
		// No branch_name
	})

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error when branch_name is missing")
	}
	if !strings.Contains(err.Error(), "branch_name") {
		t.Fatalf("expected error to mention branch_name, got %q", err.Error())
	}

	// CreatePR should NOT be called
	if len(gitProvider.calls) != 0 {
		t.Fatalf("expected 0 CreatePR calls, got %d", len(gitProvider.calls))
	}
}

func TestGitPRAction_Execute_MissingWorkDir(t *testing.T) {
	gitProvider := &prMockGitProvider{
		createPRFn: func(_ context.Context, _, _, _, _ string) (string, error) {
			return "", nil
		},
	}
	factory := &prMockGitProviderFactory{provider: gitProvider}
	storyRepo := &prMockStoryRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Story, error) {
			return newTestStory(uuid.New()), nil
		},
	}

	a := action.NewGitPRAction(factory, storyRepo, testLogger())
	runCtx := buildPRRunCtx(map[string]any{
		"branch_name": testBranchAlt,
		// No work_dir
	})

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error when work_dir is missing")
	}
	if !strings.Contains(err.Error(), "work_dir") {
		t.Fatalf("expected error to mention work_dir, got %q", err.Error())
	}

	// CreatePR should NOT be called
	if len(gitProvider.calls) != 0 {
		t.Fatalf("expected 0 CreatePR calls, got %d", len(gitProvider.calls))
	}
}

func TestGitPRAction_Execute_TitleRendering(t *testing.T) {
	tests := []struct {
		name          string
		template      string
		expectedTitle string
		scope         *string
	}{
		{
			name:          "conventional commit style",
			template:      "feat({scope}): {story_title}",
			expectedTitle: "feat(backend): Add login page",
			scope:         strPtr("backend"),
		},
		{
			name:          "default template",
			template:      "",
			expectedTitle: "S-03: Add login page",
			scope:         strPtr("backend"),
		},
		{
			name:          "with branch name",
			template:      "{story_key} ({branch_name}): {story_title}",
			expectedTitle: "S-03 (" + testBranchAlt + "): Add login page",
			scope:         strPtr("backend"),
		},
		{
			name:          "nil scope produces empty",
			template:      "feat({scope}): {story_title}",
			expectedTitle: "feat(): Add login page",
			scope:         nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitProvider := &prMockGitProvider{
				createPRFn: func(_ context.Context, _, _, _, _ string) (string, error) {
					return testPRURLAlt, nil
				},
			}
			factory := &prMockGitProviderFactory{provider: gitProvider}

			storyID := uuid.New()
			story := newTestStory(storyID)
			story.Scope = tt.scope
			storyRepo := &prMockStoryRepo{
				getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Story, error) {
					return story, nil
				},
			}

			a := action.NewGitPRAction(factory, storyRepo, testLogger())
			meta := map[string]any{
				"branch_name": testBranchAlt,
				"work_dir":    testWorkDir,
			}
			if tt.template != "" {
				meta["title_template"] = tt.template
			}
			runCtx := buildPRRunCtx(meta)
			runCtx.StoryID = storyID

			err := a.Execute(context.Background(), runCtx)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}

			if len(gitProvider.calls) != 1 {
				t.Fatalf("expected 1 CreatePR call, got %d", len(gitProvider.calls))
			}
			if gitProvider.calls[0].Title != tt.expectedTitle {
				t.Errorf("expected title = %q, got %q", tt.expectedTitle, gitProvider.calls[0].Title)
			}
		})
	}
}

func TestGitPRAction_Execute_GitProviderFailure(t *testing.T) {
	gitError := fmt.Errorf("gh CLI: command failed")
	gitProvider := &prMockGitProvider{
		createPRFn: func(_ context.Context, _, _, _, _ string) (string, error) {
			return "", gitError
		},
	}
	factory := &prMockGitProviderFactory{provider: gitProvider}

	storyID := uuid.New()
	storyRepo := &prMockStoryRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Story, error) {
			return newTestStory(storyID), nil
		},
	}

	a := action.NewGitPRAction(factory, storyRepo, testLogger())
	runCtx := buildPRRunCtx(map[string]any{
		"branch_name": testBranchAlt,
		"work_dir":    testWorkDir,
	})
	runCtx.StoryID = storyID

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error when GitProvider fails")
	}
	if !strings.Contains(err.Error(), "create PR") {
		t.Errorf("expected error to contain %q, got %q", "create PR", err.Error())
	}
	if !strings.Contains(err.Error(), "gh CLI") {
		t.Errorf("expected error to wrap GitProvider error, got %q", err.Error())
	}

	// pr_url should NOT be set
	if _, ok := runCtx.Metadata["pr_url"]; ok {
		t.Error("expected pr_url not to be set on error")
	}
}

func TestGitPRAction_Execute_StoryNotFound(t *testing.T) {
	gitProvider := &prMockGitProvider{
		createPRFn: func(_ context.Context, _, _, _, _ string) (string, error) {
			return "", nil
		},
	}
	factory := &prMockGitProviderFactory{provider: gitProvider}

	storyError := fmt.Errorf("story not found")
	storyRepo := &prMockStoryRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Story, error) {
			return nil, storyError
		},
	}

	a := action.NewGitPRAction(factory, storyRepo, testLogger())
	runCtx := buildPRRunCtx(map[string]any{
		"branch_name": testBranchAlt,
		"work_dir":    testWorkDir,
	})

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error when story not found")
	}
	if !strings.Contains(err.Error(), "fetch story") {
		t.Errorf("expected error to contain %q, got %q", "fetch story", err.Error())
	}

	// CreatePR should NOT be called
	if len(gitProvider.calls) != 0 {
		t.Fatalf("expected 0 CreatePR calls, got %d", len(gitProvider.calls))
	}
}

func TestGitPRAction_Execute_ObjectiveTruncation(t *testing.T) {
	longObjective := strings.Repeat("A", 600)
	gitProvider := &prMockGitProvider{
		createPRFn: func(_ context.Context, _, _, _, _ string) (string, error) {
			return testPRURLAlt, nil
		},
	}
	factory := &prMockGitProviderFactory{provider: gitProvider}

	storyID := uuid.New()
	story := newTestStory(storyID)
	story.Objective = &longObjective
	storyRepo := &prMockStoryRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Story, error) {
			return story, nil
		},
	}

	a := action.NewGitPRAction(factory, storyRepo, testLogger())
	runCtx := buildPRRunCtx(map[string]any{
		"branch_name": testBranchAlt,
		"work_dir":    testWorkDir,
	})
	runCtx.StoryID = storyID

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	body := gitProvider.calls[0].Body
	// The body should contain the truncated objective (500 chars + ellipsis)
	if strings.Contains(body, longObjective) {
		t.Error("expected objective to be truncated")
	}
	if !strings.Contains(body, "…") {
		t.Error("expected truncated objective to end with ellipsis")
	}
}
