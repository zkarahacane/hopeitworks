package model

import (
	"time"

	"github.com/google/uuid"
)

// NOTE: This file is P0 scaffolding for the runtime/capabilities rework. The
// types describe the agnostic capability model that runtime adapters translate
// to their native mechanism (see port.AgentRuntime). Persistence (the
// `capabilities` / `agent_capabilities` tables), the fetch-at-startup bundle and
// the per-adapter Provision implementations land in P1 — nothing here is wired
// yet.

// Capability kinds. A capability is platform data, agnostic to the runtime; each
// adapter advertises which kinds it can translate via AgentRuntime.SupportedCapabilities.
const (
	CapabilityKindSkill      = "skill"       // SKILL.md + resources (pure data)
	CapabilityKindMCPServer  = "mcp_server"  // HTTP service (default) or in-image stdio binary
	CapabilityKindToolPolicy = "tool_policy" // allow/deny tools, by role
)

// Capability scopes. Global capabilities form the admin-curated catalogue;
// project capabilities are owned by a single project.
const (
	CapabilityScopeGlobal  = "global"
	CapabilityScopeProject = "project"
)

// Capability is a versioned, runtime-agnostic capability (skill, MCP server or
// tool policy) that can be composed onto an agent.
type Capability struct {
	ID        uuid.UUID  `json:"id"`
	Kind      string     `json:"kind"` // CapabilityKind*
	Name      string     `json:"name"`
	Version   int        `json:"version"` // bumped on edit; agents pin or follow latest
	Scope     string     `json:"scope"`   // CapabilityScope*
	ProjectID *uuid.UUID `json:"project_id"`
	Spec      []byte     `json:"spec"` // agnostic spec (jsonb): skill text / url+auth / allow-deny
	CreatedAt time.Time  `json:"created_at"`
}

// AgentCapability is the composition join binding a capability onto an agent.
type AgentCapability struct {
	AgentID      uuid.UUID `json:"agent_id"`
	CapabilityID uuid.UUID `json:"capability_id"`
}

// CapabilitySpec is the assembled, runtime-agnostic set of capabilities for a
// single agent run. The platform builds it from the agent's composed
// capabilities; an adapter's Provision turns it into the harness-native form.
type CapabilitySpec struct {
	Skills     []SkillSpec     `json:"skills"`
	MCPServers []MCPServerSpec `json:"mcp_servers"`
	ToolPolicy *ToolPolicySpec `json:"tool_policy"`
}

// SkillSpec is a skill rendered as files (e.g. SKILL.md plus scripts/rubrics).
// Keyed by relative path so the runtime can materialise it on disk.
type SkillSpec struct {
	Name  string            `json:"name"`
	Files map[string]string `json:"files"`
}

// MCPServerSpec describes one MCP server. Transport "http" carries a URL plus
// headers (the default); "stdio" references an in-image binary for the
// ultra-sensitive case. CredentialRef points at a secret resolved at runtime —
// secrets are never baked nor stored in the spec in clear.
type MCPServerSpec struct {
	Name          string            `json:"name"`
	Transport     string            `json:"transport"` // "http" | "stdio"
	URL           string            `json:"url,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	Command       string            `json:"command,omitempty"`
	CredentialRef string            `json:"credential_ref,omitempty"`
}

// ToolPolicySpec is an allow/deny list of tools, applied per role.
type ToolPolicySpec struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

// CapabilitySet declares which capability kinds an adapter can translate. The
// invariant is warn+skip: an unsupported capability degrades the run, never
// blocks it (see the capability × runtime matrix in the plan).
type CapabilitySet struct {
	Skills          bool `json:"skills"`
	MCPServersHTTP  bool `json:"mcp_servers_http"`
	MCPServersStdio bool `json:"mcp_servers_stdio"`
	ToolPolicy      bool `json:"tool_policy"`
}

// ProvisionResult reports what an adapter applied and what it skipped.
type ProvisionResult struct {
	Applied  []string           `json:"applied"`
	Warnings []ProvisionWarning `json:"warnings"`
}

// ProvisionWarning records a capability that was skipped (warn+skip), with the reason.
type ProvisionWarning struct {
	Capability string `json:"capability"`
	Reason     string `json:"reason"`
}
