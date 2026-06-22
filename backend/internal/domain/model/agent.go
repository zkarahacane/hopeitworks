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

// Agent provider constants.
const (
	ProviderClaude   = "claude"
	ProviderOpenCode = "opencode"
)

// RuntimeKind identifies the pluggable agent runtime an agent executes on.
// It is the stable dispatch signal for execution mode, replacing the legacy
// heuristic that inspected the free-form image string. Each kind maps to a
// runtime adapter that wraps an existing coding harness.
const (
	RuntimeKindClaudeCode = "claude_code" // CLI `claude -p` via the agent-runtime binary
	RuntimeKindOpenCode   = "opencode"    // CLI `opencode` via the agent-runtime binary
	RuntimeKindCMA        = "cma"         // Anthropic Managed Agents (optional, cloud)
)

// RuntimeKindFromProvider derives a default RuntimeKind from the legacy provider
// value. Used to backfill/derive a runtime when none is set explicitly.
func RuntimeKindFromProvider(provider string) string {
	if provider == ProviderOpenCode {
		return RuntimeKindOpenCode
	}
	return RuntimeKindClaudeCode
}

// Agent represents an AI agent definition with its runtime configuration and prompt template.
// Agents can be scoped globally (available to all projects) or to a specific project.
type Agent struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Model string    `json:"model"`
	// Image is the free-form runtime image. It stays the fallback when StackRef is
	// nil — an image-only agent resolves exactly as before.
	Image string `json:"image"`
	// StackRef optionally points at a catalogued stack (stacks.id). When set, the
	// effective launch image is resolved from the stack's image_ref instead of Image.
	StackRef        *uuid.UUID `json:"stack_ref"`
	RuntimeKind     string     `json:"runtime_kind"` // "claude_code", "opencode" or "cma"
	TemplateContent string     `json:"template_content"`
	Type            string     `json:"type"`     // "implement", "review", "merge", "retry", "custom"
	Scope           string     `json:"scope"`    // "global" or "project"
	Provider        string     `json:"provider"` // "claude" or "opencode"
	ProjectID       *uuid.UUID `json:"project_id"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
