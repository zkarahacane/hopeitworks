package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zakari/hopeitworks/agent-runtime/internal/callback"
	"github.com/zakari/hopeitworks/agent-runtime/internal/provider"
)

// provisionCapabilities fetches the agent's capability bundle and materialises it on disk,
// returning the RunOptions the provider applies.
//
// It is fail-soft by design: any error fetching the bundle or materialising a capability is
// logged (best-effort) and skipped — provisioning NEVER blocks the run. An empty or absent
// bundle returns the zero RunOptions, so a capability-less agent runs with the exact same
// command as before this layer existed (the back-compat invariant).
func (r *Runner) provisionCapabilities(ctx context.Context, workDir string) provider.RunOptions {
	bundle, err := r.callback.FetchBundle(ctx)
	if err != nil {
		_ = r.callback.SendLog(ctx, fmt.Sprintf("capabilities: bundle fetch failed, continuing without capabilities: %v", err))
		return provider.RunOptions{}
	}
	if bundle.IsEmpty() {
		// No capabilities: nothing materialised, zero options — identical to legacy runs.
		return provider.RunOptions{}
	}

	var opts provider.RunOptions
	opts.SystemPromptAppend = bundle.SystemPrompt

	// Skills -> <workDir>/.claude/skills/<name>/<relpath>
	for _, skill := range bundle.Skills {
		if err := materialiseSkill(workDir, skill); err != nil {
			_ = r.callback.SendLog(ctx, fmt.Sprintf("capabilities: skill %q skipped: %v", skill.Name, err))
		}
	}

	// MCP servers -> <workDir>/.mcp.json
	if len(bundle.MCP.MCPServers) > 0 {
		path, err := writeMCPConfig(workDir, bundle.MCP)
		if err != nil {
			_ = r.callback.SendLog(ctx, fmt.Sprintf("capabilities: .mcp.json skipped: %v", err))
		} else {
			opts.MCPConfigPath = path
		}
	}

	// Tool policy -> allow/deny tool lists
	if bundle.ToolPolicy != nil {
		opts.AllowedTools = bundle.ToolPolicy.Allow
		opts.DisallowedTools = bundle.ToolPolicy.Deny
	}

	// Resolved secrets -> harness child env only (never the container env, so they never
	// surface in `docker inspect`). The harness expands ${NAME} references from this env.
	for name, value := range bundle.Credentials {
		opts.ExtraEnv = append(opts.ExtraEnv, name+"="+value)
	}

	_ = r.callback.SendLog(ctx, fmt.Sprintf(
		"capabilities: provisioned %d skill(s), %d mcp server(s)%s",
		len(bundle.Skills), len(bundle.MCP.MCPServers), toolPolicyNote(bundle.ToolPolicy)))
	return opts
}

func toolPolicyNote(tp *callback.BundleToolPolicy) string {
	if tp == nil {
		return ""
	}
	return fmt.Sprintf(", tool policy (allow %d/deny %d)", len(tp.Allow), len(tp.Deny))
}

// materialiseSkill writes a skill's files under <workDir>/.claude/skills/<name>/, guarding
// against path traversal in both the skill name and its file paths.
func materialiseSkill(workDir string, skill callback.BundleSkill) error {
	name := safeName(skill.Name)
	if name == "" {
		return fmt.Errorf("invalid skill name %q", skill.Name)
	}
	base := filepath.Join(workDir, ".claude", "skills", name)
	for rel, content := range skill.Files {
		dest, err := safeJoin(base, rel)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// writeMCPConfig writes the bundle's MCP servers to <workDir>/.mcp.json (perms 0600, as the
// headers may carry secret references). Returns the path the harness loads via --mcp-config.
func writeMCPConfig(workDir string, mcp callback.BundleMCP) (string, error) {
	data, err := json.MarshalIndent(mcp, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(workDir, ".mcp.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// safeName rejects skill names that contain path separators or dot segments, so a name can
// never escape the skills directory.
func safeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return ""
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return ""
	}
	return name
}

// safeJoin joins rel onto base, rejecting any path that would escape base via an absolute
// path or ../ traversal.
func safeJoin(base, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("absolute skill file path %q", rel)
	}
	dest := filepath.Join(base, rel)
	within, err := filepath.Rel(base, dest)
	if err != nil || within == ".." || strings.HasPrefix(within, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("unsafe skill file path %q", rel)
	}
	return dest, nil
}
