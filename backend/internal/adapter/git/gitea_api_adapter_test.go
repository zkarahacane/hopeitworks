package git

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// giteaFnRunner implements port.CommandRunner using a function callback.
// Unlike mockCommandRunner (sequential results), it allows per-call logic.
type giteaFnRunner struct {
	fn func(ctx context.Context, workDir string, name string, args ...string) (string, error)
}

func (r *giteaFnRunner) Run(ctx context.Context, workDir string, name string, args ...string) (string, error) {
	return r.fn(ctx, workDir, name, args...)
}

func TestParseGiteaRepoOwnerAndName(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "simple HTTPS URL",
			url:       "https://gitea.example.com/myorg/myrepo",
			wantOwner: "myorg",
			wantRepo:  "myrepo",
		},
		{
			name:      "HTTPS URL with .git suffix",
			url:       "https://gitea.example.com/myorg/myrepo.git",
			wantOwner: "myorg",
			wantRepo:  "myrepo",
		},
		{
			name:      "HTTP URL with port",
			url:       "http://localhost:3030/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "URL with embedded token",
			url:       "https://abc123@gitea.example.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:    "invalid URL - no repo",
			url:     "https://gitea.example.com/owner",
			wantErr: true,
		},
		{
			name:    "invalid URL - empty",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseGiteaRepoOwnerAndName(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if owner != tt.wantOwner {
				t.Errorf("owner: got %q, want %q", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo: got %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}

func TestParseGiteaPRIndex(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantIndex int
		wantErr   bool
	}{
		{
			name:      "standard PR URL",
			url:       "https://gitea.example.com/owner/repo/pulls/123",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantIndex: 123,
		},
		{
			name:      "PR URL with port",
			url:       "http://localhost:3030/org/project/pulls/42",
			wantOwner: "org",
			wantRepo:  "project",
			wantIndex: 42,
		},
		{
			name:    "not a PR URL",
			url:     "https://gitea.example.com/owner/repo",
			wantErr: true,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, index, err := parseGiteaPRIndex(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if owner != tt.wantOwner {
				t.Errorf("owner: got %q, want %q", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo: got %q, want %q", repo, tt.wantRepo)
			}
			if index != tt.wantIndex {
				t.Errorf("index: got %d, want %d", index, tt.wantIndex)
			}
		})
	}
}

func TestExtractBaseURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://gitea.example.com/owner/repo", "https://gitea.example.com"},
		{"http://localhost:3030/owner/repo.git", "http://localhost:3030"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractBaseURL(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInjectTokenInURL(t *testing.T) {
	got, err := injectTokenInURL("https://gitea.example.com/owner/repo.git", "mytoken")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "mytoken@") {
		t.Errorf("expected token in URL, got %q", got)
	}
	if !strings.Contains(got, "gitea.example.com") {
		t.Errorf("expected host preserved, got %q", got)
	}
}

func TestGiteaAPIAdapter_CreatePR(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/v1/repos/myorg/myrepo/pulls") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Verify auth header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "token test-token" {
			t.Errorf("expected auth header %q, got %q", "token test-token", authHeader)
		}

		var body giteaPRRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		resp := giteaPRResponse{
			Number:  42,
			HTMLURL: fmt.Sprintf("%s/myorg/myrepo/pulls/42", srv.URL),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	runner := &giteaFnRunner{
		fn: func(_ context.Context, _ string, _ string, args ...string) (string, error) {
			// Handle git remote get-url origin
			if len(args) >= 2 && args[0] == "remote" && args[1] == "get-url" {
				return fmt.Sprintf("%s/myorg/myrepo.git", srv.URL), nil
			}
			// Handle git rev-parse --abbrev-ref HEAD
			if len(args) >= 2 && args[0] == "rev-parse" && args[1] == "--abbrev-ref" {
				return "feat/S-01-test", nil
			}
			return "", fmt.Errorf("unexpected command: %v", args)
		},
	}

	adapter := NewGiteaAPIAdapter(srv.URL, "test-token", runner, testLogger())

	prURL, err := adapter.CreatePR(context.Background(), "/tmp/repo", "test PR", "body text", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(prURL, "/pulls/42") {
		t.Errorf("expected PR URL with /pulls/42, got %q", prURL)
	}
}

func TestGiteaAPIAdapter_MergePR(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/v1/repos/owner/repo/pulls/10/merge") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body giteaMergeRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Do != "squash" {
			t.Errorf("expected Do = %q, got %q", "squash", body.Do)
		}
		if !body.DeleteBranchAfterMerge {
			t.Error("expected delete_branch_after_merge = true")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	adapter := NewGiteaAPIAdapter(srv.URL, "test-token", nil, testLogger())

	prURL := fmt.Sprintf("%s/owner/repo/pulls/10", srv.URL)
	err := adapter.MergePR(context.Background(), "/tmp/repo", prURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGiteaAPIAdapter_GetCIStatus(t *testing.T) {
	tests := []struct {
		name       string
		statuses   []giteaCommitStatus
		wantStatus string
	}{
		{
			name:       "no statuses",
			statuses:   []giteaCommitStatus{},
			wantStatus: "no_checks",
		},
		{
			name:       "all success",
			statuses:   []giteaCommitStatus{{Status: "success"}, {Status: "success"}},
			wantStatus: "pass",
		},
		{
			name:       "one failure",
			statuses:   []giteaCommitStatus{{Status: "success"}, {Status: "failure"}},
			wantStatus: "fail",
		},
		{
			name:       "one pending",
			statuses:   []giteaCommitStatus{{Status: "success"}, {Status: "pending"}},
			wantStatus: "pending",
		},
		{
			name:       "error state",
			statuses:   []giteaCommitStatus{{Status: "error"}},
			wantStatus: "fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tt.statuses)
			}))
			defer srv.Close()

			runner := &giteaFnRunner{
				fn: func(_ context.Context, _ string, _ string, args ...string) (string, error) {
					if len(args) >= 2 && args[0] == "remote" && args[1] == "get-url" {
						return fmt.Sprintf("%s/owner/repo.git", srv.URL), nil
					}
					if len(args) >= 1 && args[0] == "rev-parse" {
						return "abc123def", nil
					}
					return "", fmt.Errorf("unexpected command: %v", args)
				},
			}

			adapter := NewGiteaAPIAdapter(srv.URL, "test-token", runner, testLogger())

			status, err := adapter.GetCIStatus(context.Background(), "/tmp/repo")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if status != tt.wantStatus {
				t.Errorf("got status %q, want %q", status, tt.wantStatus)
			}
		})
	}
}

func TestGiteaAPIAdapter_GetPRDiff(t *testing.T) {
	expectedDiff := "diff --git a/file.go b/file.go\n+new line"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/pulls/5.diff") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Accept") != "text/plain" {
			t.Errorf("expected Accept: text/plain, got %s", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(expectedDiff))
	}))
	defer srv.Close()

	adapter := NewGiteaAPIAdapter(srv.URL, "test-token", nil, testLogger())

	prURL := fmt.Sprintf("%s/owner/repo/pulls/5", srv.URL)
	diff, err := adapter.GetPRDiff(context.Background(), prURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff != expectedDiff {
		t.Errorf("got diff %q, want %q", diff, expectedDiff)
	}
}

func TestGiteaAPIAdapter_CloneRepo(t *testing.T) {
	var capturedArgs []string
	runner := &giteaFnRunner{
		fn: func(_ context.Context, _ string, _ string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}

	adapter := NewGiteaAPIAdapter("https://gitea.example.com", "mytoken", runner, testLogger())

	err := adapter.CloneRepo(context.Background(), "https://gitea.example.com/owner/repo.git", "/tmp/clone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the clone URL has the token injected
	if len(capturedArgs) < 2 {
		t.Fatalf("expected at least 2 args, got %d", len(capturedArgs))
	}
	cloneURL := capturedArgs[1]
	if !strings.Contains(cloneURL, "mytoken@") {
		t.Errorf("expected token in clone URL, got %q", cloneURL)
	}
}

func TestStripCredentials(t *testing.T) {
	got := stripCredentials("https://mytoken@gitea.example.com/owner/repo.git")
	if strings.Contains(got, "mytoken") {
		t.Errorf("expected credentials stripped, got %q", got)
	}
	if !strings.Contains(got, "gitea.example.com/owner/repo.git") {
		t.Errorf("expected path preserved, got %q", got)
	}
}
