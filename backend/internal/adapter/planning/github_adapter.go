package planning

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shurcooL/githubv4"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"golang.org/x/oauth2"
)

// Compile-time checks: GitHubProjectsAdapter is BOTH the inbound import adapter and
// the outbound write-back sink (it shares URL/project resolution + the rate-limited
// client between the two directions).
var (
	_ port.PlanningSourceAdapter = (*GitHubProjectsAdapter)(nil)
	_ port.PlanningSourceSink    = (*GitHubProjectsAdapter)(nil)
)

// gqlClient is the minimal GraphQL surface the adapter needs. It is satisfied by
// *githubv4.Client and lets tests inject a client pointed at an httptest server
// returning recorded JSON (no network). Mutate is used by the outbound sink
// (write-back); the import path uses only Query.
type gqlClient interface {
	Query(ctx context.Context, q interface{}, variables map[string]interface{}) error
	Mutate(ctx context.Context, m interface{}, input githubv4.Input, variables map[string]interface{}) error
}

// GitHubProjectsAdapter normalizes a GitHub Projects v2 board (read via GraphQL)
// into the source-agnostic planning DTOs. It is PURE READ: it never decides
// "done", never resolves identity, never writes — all of that lives in
// service.PlanningImportService. It only emits a correct *port.FetchResult.
//
// Status projection is service-owned: the adapter merely reports the raw Status
// single-select option name in RawStatus. It deliberately does NOT carry any
// Terminal/Cancelled signal (dropped in the normative spec §16.0).
type GitHubProjectsAdapter struct {
	client gqlClient
	logger *slog.Logger

	// fieldCache memoizes resolved status field id/options per (projectURL|field) for
	// the lifetime of this adapter instance (best-effort; the factory builds a fresh
	// adapter per request so the cache mostly de-dupes within a single WriteBack).
	mu         sync.Mutex
	fieldCache map[string]port.PlanningStatusOptions
}

// NewGitHubProjectsAdapter builds an adapter over an already-authenticated client.
func NewGitHubProjectsAdapter(client gqlClient, logger *slog.Logger) *GitHubProjectsAdapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &GitHubProjectsAdapter{client: client, logger: logger, fieldCache: map[string]port.PlanningStatusOptions{}}
}

// Kind reports the source discriminator this adapter handles ("github_projects").
func (a *GitHubProjectsAdapter) Kind() port.SourceKind { return port.SourceGitHub }

// ---- URL resolution ---------------------------------------------------------

var (
	orgProjectRe  = regexp.MustCompile(`^https://github\.com/orgs/([^/]+)/projects/(\d+)`)
	userProjectRe = regexp.MustCompile(`^https://github\.com/users/([^/]+)/projects/(\d+)`)
)

// parseProjectURL extracts (login, number, ownerIsOrg) from an anchored Projects
// v2 URL. The /orgs/ vs /users/ path segment authoritatively distinguishes the
// owner kind; an unrecognized URL is a SOURCE_ERROR (the service maps it to 422).
func parseProjectURL(raw string) (login string, number int, ownerIsOrg bool, err error) {
	if m := orgProjectRe.FindStringSubmatch(raw); m != nil {
		n, _ := strconv.Atoi(m[2])
		return m[1], n, true, nil
	}
	if m := userProjectRe.FindStringSubmatch(raw); m != nil {
		n, _ := strconv.Atoi(m[2])
		return m[1], n, false, nil
	}
	return "", 0, false, fmt.Errorf(
		"unrecognized GitHub Projects URL %q (expected https://github.com/orgs/<org>/projects/<n> or https://github.com/users/<user>/projects/<n>)",
		raw)
}

// resolveProjectID resolves the opaque ProjectV2 node id. It leads with the owner
// kind named in the URL, then falls back to the other (organization{...} /
// user{...}). A GitHub "Could not resolve to an Organization" surfaces as a
// GraphQL error with a null payload, so (err != nil || id == "") means "try the
// other path". Null after both => SOURCE_ERROR with a PAT-scope hint (fine-grained
// PATs cannot read user-owned projects — a known gap).
func (a *GitHubProjectsAdapter) resolveProjectID(ctx context.Context, ownerIsOrg bool, login string, number int, rawURL string) (string, error) {
	tryOrg := func() (string, error) {
		var q struct {
			Organization struct {
				ProjectV2 struct {
					ID githubv4.String
				} `graphql:"projectV2(number: $number)"`
			} `graphql:"organization(login: $login)"`
		}
		err := a.client.Query(ctx, &q, map[string]interface{}{
			"login":  githubv4.String(login),
			"number": githubv4.Int(int32(number)),
		})
		return string(q.Organization.ProjectV2.ID), err
	}
	tryUser := func() (string, error) {
		var q struct {
			User struct {
				ProjectV2 struct {
					ID githubv4.String
				} `graphql:"projectV2(number: $number)"`
			} `graphql:"user(login: $login)"`
		}
		err := a.client.Query(ctx, &q, map[string]interface{}{
			"login":  githubv4.String(login),
			"number": githubv4.Int(int32(number)),
		})
		return string(q.User.ProjectV2.ID), err
	}

	first, second := tryOrg, tryUser
	if !ownerIsOrg {
		first, second = tryUser, tryOrg
	}

	if id, err := first(); err == nil && id != "" {
		return id, nil
	}
	if id, err := second(); err == nil && id != "" {
		return id, nil
	}
	return "", fmt.Errorf(
		"github project %q not found or inaccessible: verify the URL and that the token has the read:project scope (note: fine-grained PATs cannot read user-owned projects)",
		rawURL)
}

// ---- typed GraphQL query structs (§8, with §16.7 fieldValues pagination) -----

type projectItemsQuery struct {
	Node struct {
		ProjectV2 struct {
			Title githubv4.String
			Items struct {
				PageInfo struct {
					HasNextPage githubv4.Boolean
					EndCursor   githubv4.String
				}
				Nodes []githubProjectItem
			} `graphql:"items(first: 20, after: $cursor)"`
		} `graphql:"... on ProjectV2"`
	} `graphql:"node(id: $projectId)"`
}

type githubProjectItem struct {
	ID          githubv4.String
	Type        githubv4.String // ISSUE | PULL_REQUEST | DRAFT_ISSUE | REDACTED
	FieldValues struct {
		PageInfo struct {
			HasNextPage githubv4.Boolean
		}
		Nodes []githubFieldValue
	} `graphql:"fieldValues(first: 50)"`
	Content githubContent
}

// githubFieldValue is a union; only single-select + text fragments are requested.
// The jsonutil decoder fills fragment fields in parallel, so the concrete kind is
// determined by __typename, never by which sub-struct is populated.
type githubFieldValue struct {
	Typename     githubv4.String `graphql:"__typename"`
	SingleSelect struct {
		Name  githubv4.String
		Field struct {
			Common struct {
				Name githubv4.String
			} `graphql:"... on ProjectV2FieldCommon"`
		}
	} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
	Text struct {
		Text  githubv4.String
		Field struct {
			Common struct {
				Name githubv4.String
			} `graphql:"... on ProjectV2FieldCommon"`
		}
	} `graphql:"... on ProjectV2ItemFieldTextValue"`
}

// githubContent is the item content union. Body/state/stateReason are NOT selected:
// §16.0 drops the Terminal/Cancelled signals and v1 parses no body, so those fields
// would only add to the secondary-rate-limit budget (§8 flags it as the real cap).
type githubContent struct {
	Typename githubv4.String `graphql:"__typename"`
	Issue    struct {
		ID         githubv4.String
		Number     githubv4.Int
		URL        githubv4.String
		Title      githubv4.String
		Repository githubRepo
		IssueType  struct {
			Name githubv4.String
		}
		Parent struct {
			ID githubv4.String
		}
		SubIssues struct {
			TotalCount githubv4.Int
		} `graphql:"subIssues(first: 1)"`
	} `graphql:"... on Issue"`
	PullRequest struct {
		ID         githubv4.String
		Number     githubv4.Int
		URL        githubv4.String
		Title      githubv4.String
		Repository githubRepo
	} `graphql:"... on PullRequest"`
	DraftIssue struct {
		ID    githubv4.String
		Title githubv4.String
	} `graphql:"... on DraftIssue"`
}

type githubRepo struct {
	Name  githubv4.String
	Owner struct {
		Login githubv4.String
	}
}

// ---- fetch ------------------------------------------------------------------

// Fetch resolves the board node id, then cursor-walks its items, normalizing each
// into an ImportedEpic/ImportedStory. A board that resolves but has zero items is
// a valid empty FetchResult with a nil error (the service returns HTTP 200, §16.5);
// only an unreachable / unresolvable source returns an error (=> SOURCE_ERROR / 422).
func (a *GitHubProjectsAdapter) Fetch(ctx context.Context, _ uuid.UUID, cfg port.ImportConfig) (*port.FetchResult, error) {
	gh := cfg.GitHubProjects
	if gh == nil {
		return nil, fmt.Errorf("github_projects import config is missing")
	}
	statusField := gh.StatusField
	if statusField == "" {
		statusField = "Status"
	}
	epicType := gh.EpicIssueType
	if epicType == "" {
		epicType = "Epic"
	}

	login, number, ownerIsOrg, err := parseProjectURL(gh.ProjectURL)
	if err != nil {
		return nil, err
	}

	nodeID, err := a.resolveProjectID(ctx, ownerIsOrg, login, number, gh.ProjectURL)
	if err != nil {
		return nil, err
	}

	res := &port.FetchResult{
		SourceURL: gh.ProjectURL,
		Epics:     []port.ImportedEpic{},
		Stories:   []port.ImportedStory{},
		Warnings:  []port.ImportWarning{},
	}

	var cursor *githubv4.String // nil => first page (after: null)
	for {
		var q projectItemsQuery
		vars := map[string]interface{}{
			"projectId": githubv4.ID(nodeID),
			"cursor":    cursor,
		}
		if err := a.client.Query(ctx, &q, vars); err != nil {
			return nil, fmt.Errorf("fetch project items: %w", err)
		}

		for i := range q.Node.ProjectV2.Items.Nodes {
			a.mapItem(&q.Node.ProjectV2.Items.Nodes[i], gh, statusField, epicType, res)
		}

		if !bool(q.Node.ProjectV2.Items.PageInfo.HasNextPage) {
			break
		}
		end := q.Node.ProjectV2.Items.PageInfo.EndCursor
		cursor = &end
	}

	return res, nil
}

// mapItem normalizes a single project item, appending an epic or story (or a
// warning for skipped/unsupported content) onto res.
func (a *GitHubProjectsAdapter) mapItem(it *githubProjectItem, gh *port.GitHubProjectsConfig, statusField, epicType string, res *port.FetchResult) {
	rawStatus, scopeRaw := readSingleSelects(it, statusField)

	var (
		key     string
		extID   string
		url     string
		title   string
		isEpic  bool
		epicRef *port.SourceRef
	)

	switch string(it.Content.Typename) {
	case "Issue":
		iss := it.Content.Issue
		key = deriveKey(string(iss.Repository.Name), int(iss.Number))
		extID = string(iss.ID)
		url = string(iss.URL)
		title = string(iss.Title)
		isEpic = strings.EqualFold(string(iss.IssueType.Name), epicType) || int(iss.SubIssues.TotalCount) > 0
		if pid := string(iss.Parent.ID); pid != "" {
			epicRef = &port.SourceRef{Source: port.SourceGitHub, ExternalID: pid}
		}
	case "PullRequest":
		pr := it.Content.PullRequest
		key = deriveKey(string(pr.Repository.Name), int(pr.Number))
		extID = string(pr.ID)
		url = string(pr.URL)
		title = string(pr.Title)
		// A PR is always a story (no epic detection, no parent traversal in v1).
	case "DraftIssue":
		d := it.Content.DraftIssue
		key = draftKey(string(it.ID)) // §16.6: stable key from the immutable project-item id
		extID = string(it.ID)         // §16.6: external_id = project-item id (not the draft node)
		url = gh.ProjectURL           // a draft has no url => deep-link to the board
		title = string(d.Title)
	default:
		// REDACTED, null content, or an unsupported content type: skip, never panic.
		res.Warnings = append(res.Warnings, port.ImportWarning{
			Key:     string(it.ID),
			Code:    "CONTENT_SKIPPED",
			Message: fmt.Sprintf("skipped project item %s: unsupported or unavailable content type %q", it.ID, it.Content.Typename),
		})
		return
	}

	// §16.7: a truncated fieldValues page may hide the Status/Scope option, so the
	// item silently falls back to backlog/no-scope — surface it.
	if bool(it.FieldValues.PageInfo.HasNextPage) {
		res.Warnings = append(res.Warnings, port.ImportWarning{
			Key:     key,
			Code:    "FIELD_VALUES_TRUNCATED",
			Message: "field values exceeded one page; the Status/Scope option may be unread (item falls back to backlog/no-scope)",
		})
	}

	if isEpic {
		res.Epics = append(res.Epics, port.ImportedEpic{
			Ref:       port.SourceRef{Source: port.SourceGitHub, ExternalID: extID, URL: url},
			Key:       key,
			Name:      title,
			RawStatus: rawStatus, // the service projects {backlog,done}
		})
		return
	}

	// §16.8: only an in-enum scope is emitted; anything else => nil + warning.
	scope, scopeWarn := normalizeScope(key, scopeRaw)
	if scopeWarn != nil {
		res.Warnings = append(res.Warnings, *scopeWarn)
	}

	res.Stories = append(res.Stories, port.ImportedStory{
		Ref: port.SourceRef{Source: port.SourceGitHub, ExternalID: extID, URL: url},
		// The ProjectV2Item id (it.ID) is the write-back target for the field
		// mutation, distinct from the content node id stored in Ref.ExternalID.
		ExternalItemID:     string(it.ID),
		Key:                key,
		Title:              title,
		Objective:          nil, // v1: no body parsing
		AcceptanceCriteria: nil, // v1: no body parsing
		Scope:              scope,
		DependsOn:          nil, // v1: native dependencies deferred
		EpicRef:            epicRef,
		RawStatus:          rawStatus, // the service projects {backlog,done}
	})
}

// readSingleSelects pulls the raw Status option name and the raw Scope option name
// from an item's single-select field values. Field matching is case-insensitive.
func readSingleSelects(it *githubProjectItem, statusField string) (rawStatus, scopeRaw string) {
	for i := range it.FieldValues.Nodes {
		fv := &it.FieldValues.Nodes[i]
		if string(fv.Typename) != "ProjectV2ItemFieldSingleSelectValue" {
			continue
		}
		fieldName := string(fv.SingleSelect.Field.Common.Name)
		switch {
		case strings.EqualFold(fieldName, statusField):
			rawStatus = string(fv.SingleSelect.Name)
		case strings.EqualFold(fieldName, "Scope"):
			scopeRaw = string(fv.SingleSelect.Name)
		}
	}
	return rawStatus, scopeRaw
}

// deriveKey builds a regex-valid story key UPPER(sanitize(repo))-<number>, e.g.
// "HOPEITWORKS-42". sanitize keeps [A-Z0-9]; an empty result falls back to "GH".
func deriveKey(repo string, number int) string {
	return sanitizeRepoName(repo) + "-" + strconv.Itoa(number)
}

func sanitizeRepoName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(name) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "GH"
	}
	return b.String()
}

// draftKey derives a stable, collision-resistant, regex-valid key for a DraftIssue
// from its immutable project-item id (§16.6): "DRAFT"+UPPER(base36(fnv1a(id)))[:8]+"-1".
// A 64-bit hash keeps the base36 slice within bounds.
func draftKey(itemID string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(itemID))
	b36 := strings.ToUpper(strconv.FormatUint(h.Sum64(), 36))
	if len(b36) > 8 {
		b36 = b36[:8]
	}
	return "DRAFT" + b36 + "-1"
}

// ---- authenticated client construction (used by the factory) ----------------

// NewGitHubClient builds a githubv4 client authenticated with a PAT, wrapping the
// transport with secondary-rate-limit backoff. The factory calls this; tests
// construct the adapter with a client pointed at httptest instead.
func NewGitHubClient(ctx context.Context, token string) *githubv4.Client {
	httpClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
	httpClient.Transport = &rateLimitRetryTransport{
		base:       httpClient.Transport, // oauth2.Transport (adds Authorization on every attempt)
		maxRetries: 4,
		sleep:      time.Sleep,
	}
	return githubv4.NewClient(httpClient)
}

// rateLimitRetryTransport retries a request when GitHub signals a (primary or
// secondary) rate limit, honoring Retry-After. GraphQL non-200s surface as opaque
// errors above the githubv4 boundary, so backoff lives here where the real
// *http.Response is visible. The GraphQL POST body is replayed via req.GetBody
// (net/http sets it for the *bytes.Buffer body shurcooL/graphql uses).
type rateLimitRetryTransport struct {
	base       http.RoundTripper
	maxRetries int
	sleep      func(time.Duration)
}

func (t *rateLimitRetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}

	for attempt := 0; ; attempt++ {
		r := req
		if attempt > 0 && req.GetBody != nil {
			body, gerr := req.GetBody()
			if gerr != nil {
				return nil, gerr
			}
			r = req.Clone(req.Context())
			r.Body = body
		}

		resp, err := base.RoundTrip(r)
		if err != nil {
			return nil, err
		}
		if attempt >= t.maxRetries || !isRateLimited(resp) {
			return resp, nil
		}

		wait := retryAfter(resp)
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		if err := req.Context().Err(); err != nil {
			return nil, err
		}
		if t.sleep != nil {
			t.sleep(wait)
		}
	}
}

// isRateLimited reports whether resp is a GitHub rate-limit rejection worth
// retrying: a 403/429 carrying a Retry-After header or X-RateLimit-Remaining: 0.
func isRateLimited(resp *http.Response) bool {
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusTooManyRequests {
		return false
	}
	if resp.Header.Get("Retry-After") != "" {
		return true
	}
	return resp.Header.Get("X-RateLimit-Remaining") == "0"
}

// retryAfter returns the backoff duration from Retry-After (clamped to [0,60s]),
// defaulting to 1s when the header is absent or unparseable.
func retryAfter(resp *http.Response) time.Duration {
	if v := resp.Header.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			switch {
			case secs < 0:
				secs = 0
			case secs > 60:
				secs = 60
			}
			return time.Duration(secs) * time.Second
		}
	}
	return time.Second
}
