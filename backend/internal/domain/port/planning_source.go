package port

import (
	"context"

	"github.com/google/uuid"
)

// SourceKind is THE discriminator keying adapter resolution + provenance.
// Adding a source = a new constant + a factory case. No interface change.
type SourceKind string

const (
	// SourceManual marks in-app/seed rows. It is never produced by an adapter.
	SourceManual SourceKind = "manual"
	// SourceMarkdown marks rows imported from generic frontmatter markdown.
	SourceMarkdown SourceKind = "markdown"
	// SourceGitHub marks rows imported from a GitHub Projects v2 board.
	// The string value is "github_projects" (precise vs a future GitHub Issues source).
	SourceGitHub SourceKind = "github_projects"
)

// SourceRef is the stable provenance of one imported node.
type SourceRef struct {
	Source     SourceKind
	ExternalID string // markdown: the key; github_projects: content node_id (opaque, NEVER the number)
	URL        string // markdown: ""; github_projects: content.url
}

// ImportedEpic is the source-agnostic epic an adapter emits.
type ImportedEpic struct {
	Ref         SourceRef
	Key         string // markdown: the epic string; github_projects: derived "<REPO>-<n>"
	Name        string
	Description *string // nil => preserve existing on re-import
	RawStatus   string  // UNMAPPED external status; the SERVICE projects it to {backlog, done}
}

// ImportedStory is the source-agnostic story an adapter emits.
// nil / empty for a field means "source does not carry this" => service PRESERVES.
type ImportedStory struct {
	Ref                SourceRef
	Key                string // MUST match ^[A-Z0-9]+-\d+$ (validated by service)
	Title              string
	Objective          *string // markdown: always nil
	AcceptanceCriteria *string
	Scope              *string    // "backend"|"frontend"|"shared"|nil
	DependsOn          []string   // story keys; nil => preserve
	EpicRef            *SourceRef // parent epic identity; nil => orphan
	RawStatus          string     // external option / frontmatter status; service projects it
	ParseError         error      // per-item soft failure -> ImportItemError, batch continues
}

// MarkdownConfig / GitHubProjectsConfig are typed per-source knobs (flat openapi sub-objects).
type MarkdownConfig struct {
	Content string
}

// GitHubProjectsConfig holds the GitHub Projects v2 import knobs.
type GitHubProjectsConfig struct {
	ProjectURL    string   // https://github.com/orgs/<o>/projects/<n> | /users/<u>/projects/<n>
	StatusField   string   // single-select field to read; default "Status"
	DoneOptions   []string // option names mapped to "done" (case-insensitive); default empty => all backlog
	EpicIssueType string   // issueType.name that means epic; default "Epic"
}

// ImportConfig is the validated, source-discriminated request.
type ImportConfig struct {
	Source         SourceKind
	DryRun         bool                  // true => Fetch + plan decisions, NO writes
	Markdown       *MarkdownConfig       // set iff Source == markdown
	GitHubProjects *GitHubProjectsConfig // set iff Source == github_projects
}

// FetchResult is the normalized snapshot — the adapter<->service contract.
type FetchResult struct {
	SourceURL string // canonical URL of the imported board/file ("" for markdown)
	Epics     []ImportedEpic
	Stories   []ImportedStory
	Warnings  []ImportWarning // non-fatal: skipped DRAFT/REDACTED, unmapped status, etc.
}

// PlanningSourceAdapter normalizes an external source. Pure read; NO DB writes.
type PlanningSourceAdapter interface {
	Kind() SourceKind
	Fetch(ctx context.Context, projectID uuid.UUID, cfg ImportConfig) (*FetchResult, error)
}

// PlanningSourceFactory resolves the adapter for a kind (parallels GitProviderFactory).
type PlanningSourceFactory interface {
	For(ctx context.Context, projectID uuid.UUID, kind SourceKind) (PlanningSourceAdapter, error)
}

// ImportSummary powers BOTH the dry-run preview UI and the post-import result UI.
type ImportSummary struct {
	Source         SourceKind
	DryRun         bool
	SourceURL      string
	EpicsCreated   int
	EpicsUpdated   int
	StoriesCreated int
	StoriesUpdated int
	Skipped        int // hash-identical, unlocked => true no-op
	Locked         int // running/failed/in-stage => spec frozen, cosmetic refresh only
	Failed         int
	Errors         []ImportItemError
	Warnings       []ImportWarning
	Items          []ImportItemDecision // per-item, drives the preview/result table
}

// ImportItemError is a per-item failure (code ∈
// {KEY_FORMAT, KEY_CONFLICT, NAME_CONFLICT, PARSE_ERROR, UPSERT_ERROR, SOURCE_ERROR}).
type ImportItemError struct {
	Key        string
	ExternalID string
	Code       string
	Message    string
}

// ImportWarning is a non-fatal advisory surfaced to the user.
type ImportWarning struct {
	Key     string
	Code    string
	Message string
}

// ImportItemDecision is the per-item plan/outcome that drives the preview/result table.
type ImportItemDecision struct {
	Key          string
	Kind         string // "epic" | "story"
	Action       string // "create" | "update" | "skip" | "lock" | "fail"
	SourceURL    string
	MappedStatus string // story: backlog|done ; epic: backlog|in_progress|done
	Reason       string // e.g. "running — status & spec frozen", "unchanged (hash match)"
}
