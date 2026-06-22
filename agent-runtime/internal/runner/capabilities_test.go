package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/zakari/hopeitworks/agent-runtime/internal/callback"
)

func TestSafeName(t *testing.T) {
	cases := map[string]string{
		"code-review": "code-review",
		"  spaced  ":  "spaced",
		"":            "",
		".":           "",
		"..":          "",
		"a/b":         "",
		`a\b`:         "",
		"../escape":   "",
		"a..b":        "",
	}
	for in, want := range cases {
		if got := safeName(in); got != want {
			t.Errorf("safeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSafeJoin_RejectsTraversal(t *testing.T) {
	base := "/tmp/skills/x"
	for _, rel := range []string{"../../etc/passwd", "../sibling", "/abs/escape"} {
		if _, err := safeJoin(base, rel); err == nil {
			t.Errorf("safeJoin(%q) expected error, got nil", rel)
		}
	}
	got, err := safeJoin(base, "scripts/run.sh")
	if err != nil {
		t.Fatalf("safeJoin valid path errored: %v", err)
	}
	if got != filepath.Join(base, "scripts/run.sh") {
		t.Errorf("safeJoin = %q", got)
	}
}

func TestMaterialiseSkill(t *testing.T) {
	dir := t.TempDir()
	skill := callback.BundleSkill{
		Name: "code-review",
		Files: map[string]string{
			"SKILL.md":       "# Review",
			"scripts/lint.sh": "echo hi",
		},
	}
	if err := materialiseSkill(dir, skill); err != nil {
		t.Fatalf("materialiseSkill: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, ".claude", "skills", "code-review", "SKILL.md"))
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	if string(got) != "# Review" {
		t.Errorf("SKILL.md = %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "skills", "code-review", "scripts", "lint.sh")); err != nil {
		t.Errorf("nested skill file missing: %v", err)
	}
}

func TestMaterialiseSkill_RejectsBadName(t *testing.T) {
	dir := t.TempDir()
	if err := materialiseSkill(dir, callback.BundleSkill{Name: "../evil", Files: map[string]string{"x": "y"}}); err == nil {
		t.Error("expected error for traversal skill name")
	}
}

func TestWriteMCPConfig(t *testing.T) {
	dir := t.TempDir()
	mcp := callback.BundleMCP{MCPServers: map[string]json.RawMessage{
		"kanban": json.RawMessage(`{"url":"http://mcp/sse"}`),
	}}
	path, err := writeMCPConfig(dir, mcp)
	if err != nil {
		t.Fatalf("writeMCPConfig: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .mcp.json: %v", err)
	}
	var parsed struct {
		MCPServers map[string]struct {
			URL string `json:"url"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal .mcp.json: %v", err)
	}
	if parsed.MCPServers["kanban"].URL != "http://mcp/sse" {
		t.Errorf("unexpected .mcp.json content: %s", raw)
	}
}
