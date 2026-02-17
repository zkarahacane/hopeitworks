package model

import (
	"time"

	"github.com/google/uuid"
)

// TemplateType represents the type of a prompt template.
type TemplateType string

// Prompt template type constants.
const (
	TemplateTypeImplement TemplateType = "implement"
	TemplateTypeRetry     TemplateType = "retry"
	TemplateTypeReview    TemplateType = "review"
	TemplateTypeMerge     TemplateType = "merge"
	TemplateTypeCustom    TemplateType = "custom"
)

// PromptTemplate represents a prompt template within a project.
type PromptTemplate struct {
	ID              uuid.UUID
	ProjectID       uuid.UUID
	Name            string
	TemplateContent string
	Type            TemplateType
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
