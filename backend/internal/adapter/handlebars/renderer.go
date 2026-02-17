package handlebars

import (
	"fmt"

	"github.com/aymerick/raymond"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure Renderer implements port.TemplateRenderer at compile time.
var _ port.TemplateRenderer = (*Renderer)(nil)

// Renderer implements port.TemplateRenderer using the raymond Handlebars library.
type Renderer struct{}

// NewRenderer creates a new Handlebars Renderer.
func NewRenderer() *Renderer {
	return &Renderer{}
}

// Render parses and renders a Handlebars template with the given TemplateContext.
func (r *Renderer) Render(templateContent string, ctx *model.TemplateContext) (string, error) {
	data := map[string]interface{}{
		"story_key":           ctx.StoryKey,
		"story_title":         ctx.StoryTitle,
		"story_objective":     ctx.StoryObjective,
		"target_files":        ctx.TargetFiles,
		"acceptance_criteria": ctx.AcceptanceCriteria,
		"error_context":       ctx.ErrorContext,
		"diff_content":        ctx.DiffContent,
		"branch_name":         ctx.BranchName,
		"repo_url":            ctx.RepoURL,
	}

	result, err := raymond.Render(templateContent, data)
	if err != nil {
		return "", &errors.DomainError{
			Category: errors.CategoryValidation,
			Code:     "TEMPLATE_RENDER_FAILED",
			Message:  fmt.Sprintf("failed to render template: %v", err),
			Cause:    err,
		}
	}

	return result, nil
}
