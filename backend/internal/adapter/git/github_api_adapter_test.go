package git

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

const (
	testRepoURL  = "https://github.com/octocat/hello"
	testPRURL    = "https://github.com/octocat/hello/pull/42"
	testBaseSHA  = "abc123def456"
	testToken    = "ghp_testtoken"
	testHTMLURL  = "https://github.com/octocat/hello/pull/42"
	headRunsPath = "/repos/octocat/hello/commits/" + testBaseSHA + "/check-runs"
	prPath       = "/repos/octocat/hello/pulls/42"
	baseMain     = "main"
)

// newTestAdapter builds a GitHubAPIAdapter pointing at the given test server.
func newTestAdapter(t *testing.T, serverURL string) *GitHubAPIAdapter {
	t.Helper()
	return NewGitHubAPIAdapter(serverURL, testToken, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

// writeJSON is a small helper to marshal a value to the response writer.
func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func TestGitHubAPIAdapter_CreateRemoteBranch(t *testing.T) {
	var gotRefBody map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/octocat/hello/git/ref/heads/main":
			writeJSON(t, w, map[string]any{
				"ref":    "refs/heads/main",
				"object": map[string]any{"sha": testBaseSHA, "type": "commit"},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/repos/octocat/hello/git/refs":
			_ = json.NewDecoder(r.Body).Decode(&gotRefBody)
			w.WriteHeader(http.StatusCreated)
			writeJSON(t, w, map[string]any{
				"ref":    "refs/heads/feat/s-01-thing",
				"object": map[string]any{"sha": testBaseSHA},
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	a := newTestAdapter(t, srv.URL)
	err := a.CreateRemoteBranch(context.Background(), testRepoURL, "feat/s-01-thing", baseMain)
	if err != nil {
		t.Fatalf("CreateRemoteBranch: unexpected error: %v", err)
	}
	if gotRefBody["ref"] != "refs/heads/feat/s-01-thing" {
		t.Errorf("ref = %q, want refs/heads/feat/s-01-thing", gotRefBody["ref"])
	}
	if gotRefBody["sha"] != testBaseSHA {
		t.Errorf("sha = %q, want %q", gotRefBody["sha"], testBaseSHA)
	}
}

func TestGitHubAPIAdapter_CreateRemotePR(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/repos/octocat/hello/pulls" {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["head"] != "feat/s-01-thing" || body["base"] != baseMain {
				t.Errorf("unexpected PR body: %+v", body)
			}
			w.WriteHeader(http.StatusCreated)
			writeJSON(t, w, map[string]any{"number": 42, "html_url": testHTMLURL})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	a := newTestAdapter(t, srv.URL)
	url, err := a.CreateRemotePR(context.Background(), testRepoURL, "title", "body", "feat/s-01-thing", baseMain)
	if err != nil {
		t.Fatalf("CreateRemotePR: unexpected error: %v", err)
	}
	if url != testHTMLURL {
		t.Errorf("prURL = %q, want %q", url, testHTMLURL)
	}
}

// ciCheckRunsHandler returns an httptest server that serves PR head SHA and
// the supplied check-runs payload for GetRemoteCIStatus tests.
func ciCheckRunsHandler(t *testing.T, checkRuns []map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case prPath:
			writeJSON(t, w, map[string]any{
				"number": 42,
				"head":   map[string]any{"sha": testBaseSHA, "ref": "feat/s-01-thing"},
			})
		case headRunsPath:
			writeJSON(t, w, map[string]any{
				"total_count": len(checkRuns),
				"check_runs":  checkRuns,
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestGitHubAPIAdapter_GetRemoteCIStatus(t *testing.T) {
	tests := []struct {
		name      string
		checkRuns []map[string]any
		want      string
	}{
		{
			name: "all completed success -> pass",
			checkRuns: []map[string]any{
				{"name": "build", "status": "completed", "conclusion": "success"},
				{"name": "test", "status": "completed", "conclusion": "success"},
			},
			want: CIStatusPass,
		},
		{
			name: "one failure -> fail",
			checkRuns: []map[string]any{
				{"name": "build", "status": "completed", "conclusion": "success"},
				{"name": "test", "status": "completed", "conclusion": "failure"},
			},
			want: CIStatusFail,
		},
		{
			name: "timed_out conclusion -> fail",
			checkRuns: []map[string]any{
				{"name": "build", "status": "completed", "conclusion": "timed_out"},
			},
			want: CIStatusFail,
		},
		{
			name: "action_required conclusion -> fail",
			checkRuns: []map[string]any{
				{"name": "build", "status": "completed", "conclusion": "action_required"},
			},
			want: CIStatusFail,
		},
		{
			name: "in_progress with success -> pending",
			checkRuns: []map[string]any{
				{"name": "build", "status": "completed", "conclusion": "success"},
				{"name": "test", "status": "in_progress", "conclusion": ""},
			},
			want: CIStatusPending,
		},
		{
			name: "queued -> pending",
			checkRuns: []map[string]any{
				{"name": "build", "status": "queued", "conclusion": ""},
			},
			want: CIStatusPending,
		},
		{
			name:      "no checks -> no_checks",
			checkRuns: []map[string]any{},
			want:      CIStatusNoChecks,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := ciCheckRunsHandler(t, tt.checkRuns)
			defer srv.Close()

			a := newTestAdapter(t, srv.URL)
			got, err := a.GetRemoteCIStatus(context.Background(), testPRURL)
			if err != nil {
				t.Fatalf("GetRemoteCIStatus: unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("status = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGitHubAPIAdapter_MergePR(t *testing.T) {
	var merged, deleted bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/repos/octocat/hello/pulls/42/merge":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["merge_method"] != mergeMethodSquash {
				t.Errorf("merge_method = %v, want squash", body["merge_method"])
			}
			merged = true
			writeJSON(t, w, map[string]any{"merged": true, "sha": testBaseSHA})
		case r.Method == http.MethodGet && r.URL.Path == prPath:
			writeJSON(t, w, map[string]any{
				"number": 42,
				"head":   map[string]any{"sha": testBaseSHA, "ref": "feat/s-01-thing"},
			})
		case r.Method == http.MethodDelete && r.URL.Path == "/repos/octocat/hello/git/refs/heads/feat/s-01-thing":
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	a := newTestAdapter(t, srv.URL)
	if err := a.MergePR(context.Background(), "", testPRURL); err != nil {
		t.Fatalf("MergePR: unexpected error: %v", err)
	}
	if !merged {
		t.Error("expected squash merge to be called")
	}
	if !deleted {
		t.Error("expected source branch deletion to be called")
	}
}

func TestGitHubAPIAdapter_MergePR_Conflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == "/repos/octocat/hello/pulls/42/merge" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			writeJSON(t, w, map[string]any{"message": "Merge conflict"})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	a := newTestAdapter(t, srv.URL)
	err := a.MergePR(context.Background(), "", testPRURL)
	if err == nil {
		t.Fatal("expected error for merge conflict")
	}
	var de *errors.DomainError
	if !stderrors.As(err, &de) || de.Code != errors.ErrCodeMergeConflict {
		t.Errorf("expected MERGE_CONFLICT domain error, got %v", err)
	}
}

func TestGitHubAPIAdapter_GetPRDiff(t *testing.T) {
	const diffBody = "diff --git a/foo.go b/foo.go\n+added line\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == prPath {
			if !strings.Contains(r.Header.Get("Accept"), "diff") {
				t.Errorf("Accept header = %q, want diff media type", r.Header.Get("Accept"))
			}
			w.Header().Set("Content-Type", "application/vnd.github.v3.diff")
			_, _ = io.WriteString(w, diffBody)
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	a := newTestAdapter(t, srv.URL)
	diff, err := a.GetPRDiff(context.Background(), testPRURL)
	if err != nil {
		t.Fatalf("GetPRDiff: unexpected error: %v", err)
	}
	if diff != diffBody {
		t.Errorf("diff = %q, want %q", diff, diffBody)
	}
}

func TestGitHubAPIAdapter_TokenAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		writeJSON(t, w, map[string]any{
			"ref":    "refs/heads/main",
			"object": map[string]any{"sha": testBaseSHA},
		})
	}))
	defer srv.Close()

	a := newTestAdapter(t, srv.URL)
	// Any API call exercises the auth header; ignore the (expected) failure
	// from the second request having no handler.
	_ = a.CreateRemoteBranch(context.Background(), testRepoURL, "feat/s-01-thing", baseMain)
	if gotAuth != "Bearer "+testToken {
		t.Errorf("Authorization = %q, want Bearer %s", gotAuth, testToken)
	}
}
