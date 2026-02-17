package port

import "github.com/zakari/hopeitworks/backend/internal/domain/model"

// TemplateRenderer renders Handlebars templates with story context.
type TemplateRenderer interface {
	// Render parses a Handlebars template string and renders it with the given context.
	// Returns the rendered string or an error if the template syntax is invalid.
	Render(templateContent string, ctx *model.TemplateContext) (string, error)
}
