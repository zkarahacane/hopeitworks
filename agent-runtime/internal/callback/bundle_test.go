package callback

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBundle_IsEmpty(t *testing.T) {
	var nilB *Bundle
	if !nilB.IsEmpty() {
		t.Error("nil bundle should be empty")
	}
	if !(&Bundle{}).IsEmpty() {
		t.Error("zero bundle should be empty")
	}
	if (&Bundle{SystemPrompt: "x"}).IsEmpty() {
		t.Error("bundle with system prompt should not be empty")
	}
	if (&Bundle{Skills: []BundleSkill{{Name: "s"}}}).IsEmpty() {
		t.Error("bundle with a skill should not be empty")
	}
}

func TestFetchBundle_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/internal/agent/callback/bundle" {
			http.Error(w, "bad route", http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer tok-123" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"system_prompt":"hello","skills":[{"name":"s","files":{"SKILL.md":"x"}}],"mcp":{"mcpServers":{}},"tool_policy":null,"credentials":{"K":"v"}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "tok-123", "run-1", "step-1")
	b, err := c.FetchBundle(context.Background())
	if err != nil {
		t.Fatalf("FetchBundle: %v", err)
	}
	if b == nil || b.SystemPrompt != "hello" {
		t.Fatalf("unexpected bundle: %+v", b)
	}
	if len(b.Skills) != 1 || b.Skills[0].Files["SKILL.md"] != "x" {
		t.Errorf("skills not decoded: %+v", b.Skills)
	}
	if b.Credentials["K"] != "v" {
		t.Errorf("credentials not decoded: %+v", b.Credentials)
	}
}

// An older API without the endpoint returns 404 -> treated as no bundle (nil, nil).
func TestFetchBundle_NotFoundIsNoBundle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", "run-1", "step-1")
	b, err := c.FetchBundle(context.Background())
	if err != nil {
		t.Fatalf("FetchBundle should not error on 404: %v", err)
	}
	if b != nil {
		t.Errorf("expected nil bundle on 404, got %+v", b)
	}
}
