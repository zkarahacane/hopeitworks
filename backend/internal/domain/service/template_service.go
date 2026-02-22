package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Default template names used for fallback resolution.
const (
	TemplateNameImplement      = "implement"
	TemplateNameImplementRetry = "implement-retry"
	TemplateNameReview         = "review"
	TemplateNameMerge          = "merge"
	TemplateNameMergeConflict  = "merge-conflict"
)

// TemplateService resolves prompt templates from the database (with fallback to defaults)
// and renders them with story context using a TemplateRenderer.
type TemplateService struct {
	templateRepo port.PromptTemplateRepository
	renderer     port.TemplateRenderer
	logger       *slog.Logger
}

// NewTemplateService creates a new TemplateService.
func NewTemplateService(
	templateRepo port.PromptTemplateRepository,
	renderer port.TemplateRenderer,
	logger *slog.Logger,
) *TemplateService {
	return &TemplateService{
		templateRepo: templateRepo,
		renderer:     renderer,
		logger:       logger,
	}
}

// RenderForStory resolves a template by name for a project, falls back to defaults, and renders it.
func (s *TemplateService) RenderForStory(
	ctx context.Context,
	projectID uuid.UUID,
	templateName string,
	tmplCtx *model.TemplateContext,
) (string, error) {
	tmpl, err := s.templateRepo.GetByProjectAndName(ctx, projectID, templateName)
	if err != nil {
		// Check if the error is a not-found error
		domainErr, ok := err.(*errors.DomainError)
		if !ok || domainErr.Category != errors.CategoryNotFound {
			return "", err
		}

		// Template not found in DB, try default
		s.logger.Debug("template not found in DB, trying default",
			"project_id", projectID,
			"template_name", templateName,
		)

		defaultContent := getDefaultTemplate(templateName)
		if defaultContent == "" {
			return "", &errors.DomainError{
				Category: errors.CategoryNotFound,
				Code:     "TEMPLATE_NOT_FOUND",
				Message:  "template '" + templateName + "' not found in database and no default exists",
			}
		}

		return s.renderer.Render(defaultContent, tmplCtx)
	}

	return s.renderer.Render(tmpl.TemplateContent, tmplCtx)
}

// getDefaultTemplate returns hardcoded default template content for known template names.
// Returns empty string if no default exists for the given name.
func getDefaultTemplate(name string) string {
	defaults := map[string]string{
		TemplateNameImplement: `Implement story {{story_key}}: {{story_title}}

## Objective
{{story_objective}}

## Instructions
Implement the story according to the acceptance criteria below. Work on branch {{branch_name}} and modify only the target files listed.

## Target Files
{{#each target_files}}
- {{this}}
{{/each}}

## Acceptance Criteria
{{acceptance_criteria}}

## Branch
{{branch_name}}`,

		TemplateNameImplementRetry: `Retry implementation for {{story_key}}: {{story_title}}

## Previous Error
{{error_context}}

## Log Tail
{{log_tail}}

## Existing Changes
{{diff_content}}

## Objective
{{story_objective}}

Fix the issues described above while preserving the existing changes.`,

		TemplateNameReview: `Review changes for {{story_key}}: {{story_title}}

## Story Context
**Objective:** {{story_objective}}

**Acceptance Criteria:**
{{acceptance_criteria}}

## Changes to Review
{{diff_content}}

## Review Checklist
- [ ] All acceptance criteria are met
- [ ] Code passes golangci-lint (backend) or ESLint (frontend)
- [ ] Tests are added or updated for new behavior
- [ ] No secrets, tokens, or credentials are committed
- [ ] No console.log or fmt.Println in production code
- [ ] Code quality and adherence to project conventions
- [ ] Error messages are actionable with sufficient context

## Review Instructions
Report findings as a list of issues with severity (blocker, warning, suggestion). If no issues are found, approve the changes.`,

		TemplateNameMerge: `Merge changes for {{story_key}}: {{story_title}}

## Story Context
**Objective:** {{story_objective}}
**Branch:** {{branch_name}}

## Merge Steps
1. Check that CI is green on the feature branch: gh pr checks or gh run list --branch {{branch_name}}
2. Rebase the feature branch on develop: git fetch origin develop && git rebase origin/develop
3. Push the rebased branch: git push --force-with-lease
4. Create a PR following conventional commit format: gh pr create --title "feat(scope): summary" --body "..."
5. Squash merge after CI passes: gh pr merge --squash --auto
6. Verify that CI passes on develop after merge`,

		TemplateNameMergeConflict: `Resolve merge conflict for {{story_key}}: {{story_title}}

## Story Context
**Objective:** {{story_objective}}

## Conflict Details
{{error_context}}

## Current Changes
{{diff_content}}

## Resolution Instructions
- Review the conflict markers in the diff
- Resolve conflicts while preserving the story objective
- Ensure all acceptance criteria remain satisfied after resolution`,
	}
	return defaults[name]
}
