package model

// RuntimeBundle is the fetch-at-startup contract. The agent-runtime GETs it from the
// API at boot (authenticated by the short-lived container token) and materialises it
// before launching the harness. It is the assembled, runtime-agnostic projection of an
// agent's composed capabilities (skills + MCP servers + tool policy + resolved secrets).
//
// INVARIANT (back-compat): an agent with no capabilities yields a zero-value bundle —
// no system prompt, no skills, no MCP servers, no tool policy, no credentials. The
// runtime then materialises nothing and behaves exactly as it did before this layer
// existed. IsEmpty() is the single guard the runtime uses for that fast path.
type RuntimeBundle struct {
	SystemPrompt string            `json:"system_prompt"`
	Skills       []SkillSpec       `json:"skills"`
	MCP          MCPBundle         `json:"mcp"`
	ToolPolicy   *ToolPolicySpec   `json:"tool_policy"`
	Credentials  map[string]string `json:"credentials"`
}

// IsEmpty reports whether the bundle carries nothing to materialise. When true the
// runtime skips all materialisation and runs the harness exactly as before.
func (b RuntimeBundle) IsEmpty() bool {
	return b.SystemPrompt == "" &&
		len(b.Skills) == 0 &&
		len(b.MCP.MCPServers) == 0 &&
		b.ToolPolicy == nil &&
		len(b.Credentials) == 0
}

// MCPBundle is the `.mcp.json` projection: a map of server name -> connection entry.
// The wrapper struct keeps the JSON shape `{"mcpServers": {...}}` that Claude Code and
// other harnesses expect.
type MCPBundle struct {
	MCPServers map[string]MCPServerEntry `json:"mcpServers"`
}

// MCPServerEntry is one `.mcp.json` server definition. HTTP transport carries url +
// headers; stdio carries command + args. Header values may contain ${CRED} placeholders
// that the runtime resolves from RuntimeBundle.Credentials (the harness expands env-style
// references), so the secret value itself never travels inside the spec.
type MCPServerEntry struct {
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
}
