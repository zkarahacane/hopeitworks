package action

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// GitBranchAction implements model.Action for creating a Git feature branch.
// It renders the branch name from a configurable pattern using RunContext variables,
// delegates creation to GitProvider, and stores the result in RunContext.Metadata.
type GitBranchAction struct {
	gitProvider port.GitProvider
	storyRepo   port.StoryRepository
	logger      *slog.Logger
}

// NewGitBranchAction creates a new GitBranchAction.
func NewGitBranchAction(gitProvider port.GitProvider, storyRepo port.StoryRepository, logger *slog.Logger) *GitBranchAction {
	return &GitBranchAction{
		gitProvider: gitProvider,
		storyRepo:   storyRepo,
		logger:      logger,
	}
}

// Name returns the action identifier.
func (a *GitBranchAction) Name() string { return "git_branch" }

// Execute creates a feature branch from a configurable pattern.
// It reads branch_pattern and base_branch from step config, derives a slug from
// the story title, and calls GitProvider.CreateBranch. On success, the rendered
// branch name is stored in runCtx.Metadata["branch_name"].
func (a *GitBranchAction) Execute(ctx context.Context, runCtx *model.RunContext) error {
	cfg := runCtx.RunStep.Config
	if cfg == nil {
		cfg = make(map[string]string)
	}

	pattern := cfg["branch_pattern"]
	if pattern == "" {
		pattern = "feat/{story_key}-{slug}"
	}
	baseBranch := cfg["base_branch"]
	if baseBranch == "" {
		baseBranch = "main"
	}

	workDir := cfg["work_dir"]
	if workDir == "" {
		if wd, ok := runCtx.Metadata["work_dir"].(string); ok && wd != "" {
			workDir = wd
		} else {
			return fmt.Errorf("git_branch: work_dir not configured and not in metadata")
		}
	}

	story, err := a.storyRepo.GetByID(ctx, runCtx.StoryID)
	if err != nil {
		return fmt.Errorf("fetch story: %w", err)
	}

	slug := slugify(story.Title)
	branchName := strings.ReplaceAll(pattern, "{story_key}", story.Key)
	branchName = strings.ReplaceAll(branchName, "{slug}", slug)

	a.logger.Info("creating branch",
		"branch", branchName,
		"base", baseBranch,
		"story_key", story.Key,
		"work_dir", workDir,
	)

	if err := a.gitProvider.CreateBranch(ctx, workDir, branchName); err != nil {
		return fmt.Errorf("create branch %q: %w", branchName, err)
	}

	runCtx.Metadata["branch_name"] = branchName
	return nil
}

// slugify converts a story title to a URL-safe lowercase slug.
func slugify(title string) string {
	lower := strings.ToLower(title)
	slug := nonAlphanumeric.ReplaceAllString(lower, "-")
	return strings.Trim(slug, "-")
}
