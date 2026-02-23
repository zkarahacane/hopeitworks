package model

import (
	"time"

	"github.com/google/uuid"
)

// Agent scope constants.
const (
	AgentScopeGlobal  = "global"
	AgentScopeProject = "project"
)

// Agent represents an AI agent definition with its runtime configuration and prompt template.
// Agents can be scoped globally (available to all projects) or to a specific project.
type Agent struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	Model           string     `json:"model"`
	Image           string     `json:"image"`
	TemplateContent string     `json:"template_content"`
	Type            string     `json:"type"`  // "implement", "review", "merge", "retry", "custom"
	Scope           string     `json:"scope"` // "global" or "project"
	ProjectID       *uuid.UUID `json:"project_id"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
