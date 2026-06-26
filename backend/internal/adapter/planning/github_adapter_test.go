package planning

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shurcooL/githubv4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// gqlHandler returns the recorded GraphQL JSON body for a given request. The query
// string + variables let a test branch on the resolution phase (organization/user)
// vs the items phase (node), and on the pagination cursor.
type gqlHandler func(query string, vars map[string]any) string

// newTestAdapter wires a GitHubProjectsAdapter to an httptest server that replays
// recorded JSON, exercising the real githubv4 query-build + decode path (no network).
func newTestAdapter(t *testing.T, h gqlHandler) *GitHubProjectsAdapter {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, h(body.Query, body.Variables))
	}))
	t.Cleanup(srv.Close)
	client := githubv4.NewEnterpriseClient(srv.URL, srv.Client())
	return NewGitHubProjectsAdapter(client, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func ghConfig(url string) port.ImportConfig {
	return port.ImportConfig{
		Source: port.SourceGitHub,
		GitHubProjects: &port.GitHubProjectsConfig{
			ProjectURL:  url,
			DoneOptions: []string{"Done"},
		},
	}
}

const orgURL = "https://github.com/orgs/acme/projects/7"
const userURL = "https://github.com/users/octocat/projects/3"

// resolveOrg/resolveUser are the canned resolution responses.
func resolveOrg(id string) string {
	return `{"data":{"organization":{"projectV2":{"id":"` + id + `"}}}}`
}
func resolveUser(id string) string {
	return `{"data":{"user":{"projectV2":{"id":"` + id + `"}}}}`
}

// itemsPage wraps node fixtures into the data envelope.
func itemsPage(hasNext bool, endCursor, nodesJSON string) string {
	next := "false"
	if hasNext {
		next = "true"
	}
	return `{"data":{"node":{"title":"Roadmap","items":{` +
		`"pageInfo":{"hasNextPage":` + next + `,"endCursor":"` + endCursor + `"},` +
		`"nodes":[` + nodesJSON + `]}}}}`
}

func isResolution(query string) bool {
	return strings.Contains(query, "organization(login:") || strings.Contains(query, "user(login:")
}
func isOrgQuery(query string) bool { return strings.Contains(query, "organization(login:") }

// ---- fixtures ---------------------------------------------------------------

// epicIssue: an Issue whose issueType is "Epic" => epic.
const epicIssueNode = `{
  "id":"PVTI_epic","type":"ISSUE",
  "fieldValues":{"pageInfo":{"hasNextPage":false},"nodes":[
    {"__typename":"ProjectV2ItemFieldSingleSelectValue","name":"Done","field":{"name":"Status"}}
  ]},
  "content":{"__typename":"Issue","id":"I_kwEPIC","number":1,
    "url":"https://github.com/acme/repo/issues/1","title":"Auth epic",
    "repository":{"name":"repo","owner":{"login":"acme"}},
    "issueType":{"name":"Epic"},"parent":{"id":""},"subIssues":{"totalCount":3}}
}`

// storyIssue: a child Issue (parent = the epic above), scope backend.
const storyIssueNode = `{
  "id":"PVTI_story","type":"ISSUE",
  "fieldValues":{"pageInfo":{"hasNextPage":false},"nodes":[
    {"__typename":"ProjectV2ItemFieldSingleSelectValue","name":"In Progress","field":{"name":"Status"}},
    {"__typename":"ProjectV2ItemFieldSingleSelectValue","name":"Backend","field":{"name":"Scope"}}
  ]},
  "content":{"__typename":"Issue","id":"I_kwSTORY","number":42,
    "url":"https://github.com/acme/repo/issues/42","title":"Login form",
    "repository":{"name":"my-repo","owner":{"login":"acme"}},
    "issueType":{"name":"Task"},"parent":{"id":"I_kwEPIC"},"subIssues":{"totalCount":0}}
}`

func TestFetch_OrgResolution_IssueEpicAndStory(t *testing.T) {
	a := newTestAdapter(t, func(query string, _ map[string]any) string {
		if isResolution(query) {
			assert.True(t, isOrgQuery(query), "org URL must resolve via organization()")
			return resolveOrg("PVT_org123")
		}
		assert.Contains(t, query, "node(id:")
		return itemsPage(false, "END", epicIssueNode+","+storyIssueNode)
	})

	res, err := a.Fetch(context.Background(), uuid.New(), ghConfig(orgURL))
	require.NoError(t, err)
	require.Equal(t, orgURL, res.SourceURL)

	// one epic (issueType Epic), one story (the child).
	require.Len(t, res.Epics, 1)
	require.Len(t, res.Stories, 1)

	epic := res.Epics[0]
	assert.Equal(t, "I_kwEPIC", epic.Ref.ExternalID, "external_id is the opaque node id, never the number")
	assert.Equal(t, port.SourceGitHub, epic.Ref.Source)
	assert.Equal(t, "REPO-1", epic.Key)
	assert.Equal(t, "Auth epic", epic.Name)
	assert.Equal(t, "Done", epic.RawStatus, "adapter reports raw status; the service projects done")

	story := res.Stories[0]
	assert.Equal(t, "I_kwSTORY", story.Ref.ExternalID)
	assert.Equal(t, "MYREPO-42", story.Key, "key = UPPER(sanitize(repo))-number")
	assert.Equal(t, "https://github.com/acme/repo/issues/42", story.Ref.URL)
	assert.Equal(t, "Login form", story.Title)
	assert.Equal(t, "In Progress", story.RawStatus)
	require.NotNil(t, story.Scope)
	assert.Equal(t, "backend", *story.Scope, "scope normalized to lowercase enum")
	require.NotNil(t, story.EpicRef)
	assert.Equal(t, "I_kwEPIC", story.EpicRef.ExternalID, "EpicRef = content.parent.id")
	assert.Nil(t, story.Objective)
	assert.Nil(t, story.AcceptanceCriteria)
	assert.Empty(t, res.Warnings)
}

func TestFetch_SubIssuesPromotesToEpic(t *testing.T) {
	// issueType is not Epic, but subIssues.totalCount > 0 => epic.
	node := `{
      "id":"PVTI_x","type":"ISSUE",
      "fieldValues":{"pageInfo":{"hasNextPage":false},"nodes":[]},
      "content":{"__typename":"Issue","id":"I_parent","number":5,
        "url":"https://github.com/acme/repo/issues/5","title":"Parent feature",
        "repository":{"name":"repo","owner":{"login":"acme"}},
        "issueType":{"name":"Feature"},"parent":{"id":""},"subIssues":{"totalCount":2}}
    }`
	a := newTestAdapter(t, func(query string, _ map[string]any) string {
		if isResolution(query) {
			return resolveOrg("PVT_org123")
		}
		return itemsPage(false, "END", node)
	})
	res, err := a.Fetch(context.Background(), uuid.New(), ghConfig(orgURL))
	require.NoError(t, err)
	require.Len(t, res.Epics, 1)
	assert.Empty(t, res.Stories)
	assert.Equal(t, "I_parent", res.Epics[0].Ref.ExternalID)
}

func TestFetch_UserResolution(t *testing.T) {
	node := `{
      "id":"PVTI_u","type":"ISSUE",
      "fieldValues":{"pageInfo":{"hasNextPage":false},"nodes":[]},
      "content":{"__typename":"Issue","id":"I_u1","number":9,
        "url":"https://github.com/octocat/site/issues/9","title":"User story",
        "repository":{"name":"site","owner":{"login":"octocat"}},
        "issueType":{"name":"Task"},"parent":{"id":""},"subIssues":{"totalCount":0}}
    }`
	a := newTestAdapter(t, func(query string, _ map[string]any) string {
		if isResolution(query) {
			assert.False(t, isOrgQuery(query), "user URL must resolve via user()")
			return resolveUser("PVT_user1")
		}
		return itemsPage(false, "END", node)
	})
	res, err := a.Fetch(context.Background(), uuid.New(), ghConfig(userURL))
	require.NoError(t, err)
	require.Len(t, res.Stories, 1)
	assert.Equal(t, "SITE-9", res.Stories[0].Key)
}

func TestFetch_PullRequestAndDraft(t *testing.T) {
	prNode := `{
      "id":"PVTI_pr","type":"PULL_REQUEST",
      "fieldValues":{"pageInfo":{"hasNextPage":false},"nodes":[]},
      "content":{"__typename":"PullRequest","id":"PR_node","number":17,
        "url":"https://github.com/acme/repo/pull/17","title":"Fix bug",
        "repository":{"name":"repo","owner":{"login":"acme"}}}
    }`
	draftNode := `{
      "id":"PVTI_draftABC","type":"DRAFT_ISSUE",
      "fieldValues":{"pageInfo":{"hasNextPage":false},"nodes":[]},
      "content":{"__typename":"DraftIssue","id":"DI_inner","title":"Spike idea"}
    }`
	a := newTestAdapter(t, func(query string, _ map[string]any) string {
		if isResolution(query) {
			return resolveOrg("PVT_org123")
		}
		return itemsPage(false, "END", prNode+","+draftNode)
	})
	res, err := a.Fetch(context.Background(), uuid.New(), ghConfig(orgURL))
	require.NoError(t, err)
	require.Len(t, res.Stories, 2)

	pr := res.Stories[0]
	assert.Equal(t, "PR_node", pr.Ref.ExternalID)
	assert.Equal(t, "REPO-17", pr.Key)
	assert.Equal(t, "https://github.com/acme/repo/pull/17", pr.Ref.URL)
	assert.Equal(t, "Fix bug", pr.Title)

	draft := res.Stories[1]
	// §16.6: external_id = project-item id; key = "DRAFT"+base36(fnv1a(itemID))[:8]+"-1"
	assert.Equal(t, "PVTI_draftABC", draft.Ref.ExternalID, "draft external_id is the project-item id")
	assert.Equal(t, draftKey("PVTI_draftABC"), draft.Key)
	assert.True(t, strings.HasPrefix(draft.Key, "DRAFT"))
	assert.True(t, strings.HasSuffix(draft.Key, "-1"))
	assert.Equal(t, orgURL, draft.Ref.URL, "draft deep-links to the board")
	assert.Equal(t, "Spike idea", draft.Title)
}

func TestFetch_RedactedContentSkipped(t *testing.T) {
	node := `{
      "id":"PVTI_redacted","type":"REDACTED",
      "fieldValues":{"pageInfo":{"hasNextPage":false},"nodes":[]},
      "content":null
    }`
	a := newTestAdapter(t, func(query string, _ map[string]any) string {
		if isResolution(query) {
			return resolveOrg("PVT_org123")
		}
		return itemsPage(false, "END", node)
	})
	res, err := a.Fetch(context.Background(), uuid.New(), ghConfig(orgURL))
	require.NoError(t, err)
	assert.Empty(t, res.Epics)
	assert.Empty(t, res.Stories)
	require.Len(t, res.Warnings, 1)
	assert.Equal(t, "CONTENT_SKIPPED", res.Warnings[0].Code)
}

func TestFetch_InvalidScopeWarns(t *testing.T) {
	node := `{
      "id":"PVTI_s","type":"ISSUE",
      "fieldValues":{"pageInfo":{"hasNextPage":false},"nodes":[
        {"__typename":"ProjectV2ItemFieldSingleSelectValue","name":"Todo","field":{"name":"Status"}},
        {"__typename":"ProjectV2ItemFieldSingleSelectValue","name":"infra","field":{"name":"Scope"}}
      ]},
      "content":{"__typename":"Issue","id":"I_s","number":3,
        "url":"https://github.com/acme/repo/issues/3","title":"Bad scope",
        "repository":{"name":"repo","owner":{"login":"acme"}},
        "issueType":{"name":"Task"},"parent":{"id":""},"subIssues":{"totalCount":0}}
    }`
	a := newTestAdapter(t, func(query string, _ map[string]any) string {
		if isResolution(query) {
			return resolveOrg("PVT_org123")
		}
		return itemsPage(false, "END", node)
	})
	res, err := a.Fetch(context.Background(), uuid.New(), ghConfig(orgURL))
	require.NoError(t, err)
	require.Len(t, res.Stories, 1)
	assert.Nil(t, res.Stories[0].Scope, "out-of-enum scope is dropped")
	require.Len(t, res.Warnings, 1)
	assert.Equal(t, "INVALID_SCOPE", res.Warnings[0].Code)
	assert.Equal(t, "REPO-3", res.Warnings[0].Key)
}

func TestFetch_Pagination(t *testing.T) {
	page1 := `{
      "id":"PVTI_p1","type":"ISSUE",
      "fieldValues":{"pageInfo":{"hasNextPage":false},"nodes":[]},
      "content":{"__typename":"Issue","id":"I_p1","number":1,
        "url":"https://github.com/acme/repo/issues/1","title":"First",
        "repository":{"name":"repo","owner":{"login":"acme"}},
        "issueType":{"name":"Task"},"parent":{"id":""},"subIssues":{"totalCount":0}}
    }`
	page2 := `{
      "id":"PVTI_p2","type":"ISSUE",
      "fieldValues":{"pageInfo":{"hasNextPage":false},"nodes":[]},
      "content":{"__typename":"Issue","id":"I_p2","number":2,
        "url":"https://github.com/acme/repo/issues/2","title":"Second",
        "repository":{"name":"repo","owner":{"login":"acme"}},
        "issueType":{"name":"Task"},"parent":{"id":""},"subIssues":{"totalCount":0}}
    }`
	calls := 0
	a := newTestAdapter(t, func(query string, vars map[string]any) string {
		if isResolution(query) {
			return resolveOrg("PVT_org123")
		}
		calls++
		if vars["cursor"] == nil {
			return itemsPage(true, "CUR1", page1) // first page, more to come
		}
		assert.Equal(t, "CUR1", vars["cursor"], "second page must pass the prior endCursor")
		return itemsPage(false, "", page2)
	})
	res, err := a.Fetch(context.Background(), uuid.New(), ghConfig(orgURL))
	require.NoError(t, err)
	assert.Equal(t, 2, calls, "must walk both pages")
	require.Len(t, res.Stories, 2)
	assert.Equal(t, "REPO-1", res.Stories[0].Key)
	assert.Equal(t, "REPO-2", res.Stories[1].Key)
}

func TestFetch_FieldValuesTruncationWarns(t *testing.T) {
	node := `{
      "id":"PVTI_t","type":"ISSUE",
      "fieldValues":{"pageInfo":{"hasNextPage":true},"nodes":[]},
      "content":{"__typename":"Issue","id":"I_t","number":8,
        "url":"https://github.com/acme/repo/issues/8","title":"Many fields",
        "repository":{"name":"repo","owner":{"login":"acme"}},
        "issueType":{"name":"Task"},"parent":{"id":""},"subIssues":{"totalCount":0}}
    }`
	a := newTestAdapter(t, func(query string, _ map[string]any) string {
		if isResolution(query) {
			return resolveOrg("PVT_org123")
		}
		return itemsPage(false, "END", node)
	})
	res, err := a.Fetch(context.Background(), uuid.New(), ghConfig(orgURL))
	require.NoError(t, err)
	require.Len(t, res.Stories, 1)
	require.Len(t, res.Warnings, 1)
	assert.Equal(t, "FIELD_VALUES_TRUNCATED", res.Warnings[0].Code)
	assert.Equal(t, "REPO-8", res.Warnings[0].Key)
}

func TestFetch_EmptyBoardIsSuccess(t *testing.T) {
	a := newTestAdapter(t, func(query string, _ map[string]any) string {
		if isResolution(query) {
			return resolveOrg("PVT_org123")
		}
		return itemsPage(false, "", "")
	})
	res, err := a.Fetch(context.Background(), uuid.New(), ghConfig(orgURL))
	require.NoError(t, err, "empty board is a valid 200, not a SOURCE_ERROR")
	assert.Empty(t, res.Epics)
	assert.Empty(t, res.Stories)
	assert.Empty(t, res.Warnings)
	assert.Equal(t, orgURL, res.SourceURL)
}

func TestFetch_ProjectNotFound(t *testing.T) {
	// org returns null + a GraphQL error; user also null => unresolvable.
	a := newTestAdapter(t, func(query string, _ map[string]any) string {
		if isOrgQuery(query) {
			return `{"data":{"organization":null},"errors":[{"message":"Could not resolve to an Organization with the login of 'acme'."}]}`
		}
		return `{"data":{"user":null},"errors":[{"message":"Could not resolve to a User with the login of 'acme'."}]}`
	})
	_, err := a.Fetch(context.Background(), uuid.New(), ghConfig(orgURL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found or inaccessible")
	assert.Contains(t, err.Error(), "read:project")
}

func TestParseProjectURL(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantLogin  string
		wantNumber int
		wantOrg    bool
		wantErr    bool
	}{
		{"org", "https://github.com/orgs/acme/projects/7", "acme", 7, true, false},
		{"user", "https://github.com/users/octocat/projects/3", "octocat", 3, false, false},
		{"org trailing", "https://github.com/orgs/acme/projects/7/views/1", "acme", 7, true, false},
		{"bogus", "https://example.com/foo", "", 0, false, true},
		{"repo project (unsupported)", "https://github.com/acme/repo/projects/2", "", 0, false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			login, number, org, err := parseProjectURL(tc.url)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantLogin, login)
			assert.Equal(t, tc.wantNumber, number)
			assert.Equal(t, tc.wantOrg, org)
		})
	}
}

func TestDeriveKey(t *testing.T) {
	assert.Equal(t, "HOPEITWORKS-42", deriveKey("hopeitworks", 42))
	assert.Equal(t, "MYREPO-1", deriveKey("my-repo", 1))
	assert.Equal(t, "GH-5", deriveKey("___", 5), "empty sanitized repo falls back to GH")
	assert.Equal(t, "REPO2-9", deriveKey("repo.2", 9))
	assert.Regexp(t, `^[A-Z0-9]+-\d+$`, deriveKey("my-repo", 1))
}

func TestDraftKey(t *testing.T) {
	k1 := draftKey("PVTI_lADO123")
	k2 := draftKey("PVTI_lADO123")
	assert.Equal(t, k1, k2, "stable for the same item id")
	assert.NotEqual(t, draftKey("PVTI_lADO123"), draftKey("PVTI_other"))
	assert.True(t, strings.HasPrefix(k1, "DRAFT"))
	assert.True(t, strings.HasSuffix(k1, "-1"))
	assert.Regexp(t, `^[A-Z0-9]+-\d+$`, k1)
	assert.LessOrEqual(t, len(k1), len("DRAFT")+8+len("-1"))
}

func TestRateLimitRetryTransport(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		// Drain the body to prove it is replayed on retry.
		b, _ := io.ReadAll(r.Body)
		require.NotEmpty(t, b, "body must be replayed on every attempt")
		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"message":"You have exceeded a secondary rate limit"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, resolveOrg("PVT_after_retry"))
	}))
	t.Cleanup(srv.Close)

	httpClient := srv.Client()
	slept := 0
	httpClient.Transport = &rateLimitRetryTransport{
		base:       httpClient.Transport,
		maxRetries: 3,
		sleep:      func(time.Duration) { slept++ },
	}
	client := githubv4.NewEnterpriseClient(srv.URL, httpClient)
	a := NewGitHubProjectsAdapter(client, slog.New(slog.NewTextHandler(io.Discard, nil)))

	id, err := a.resolveProjectID(context.Background(), true, "acme", 7, orgURL)
	require.NoError(t, err)
	assert.Equal(t, "PVT_after_retry", id)
	assert.Equal(t, 2, calls, "one rate-limited attempt + one success")
	assert.Equal(t, 1, slept, "backed off once")
}

func hdr(pairs ...string) http.Header {
	h := http.Header{}
	for i := 0; i+1 < len(pairs); i += 2 {
		h.Set(pairs[i], pairs[i+1]) // Set canonicalizes keys (matches Header.Get lookups)
	}
	return h
}

func TestIsRateLimited(t *testing.T) {
	// Composite literals (not call results) keep the bodyclose linter quiet; these
	// synthetic responses carry no body to close.
	assert.True(t, isRateLimited(&http.Response{StatusCode: 403, Header: hdr("Retry-After", "30")}))
	assert.True(t, isRateLimited(&http.Response{StatusCode: 429, Header: hdr("Retry-After", "5")}))
	assert.True(t, isRateLimited(&http.Response{StatusCode: 403, Header: hdr("X-RateLimit-Remaining", "0")}))
	assert.False(t, isRateLimited(&http.Response{StatusCode: 403, Header: hdr("X-RateLimit-Remaining", "12")}))
	assert.False(t, isRateLimited(&http.Response{StatusCode: 200, Header: hdr()}))

	assert.Equal(t, 30*time.Second, retryAfter(&http.Response{StatusCode: 403, Header: hdr("Retry-After", "30")}))
	assert.Equal(t, 60*time.Second, retryAfter(&http.Response{StatusCode: 403, Header: hdr("Retry-After", "9999")}))
	assert.Equal(t, time.Second, retryAfter(&http.Response{StatusCode: 403, Header: hdr()}))
}
