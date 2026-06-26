// Package planning hosts the PlanningSourceAdapter implementations (markdown,
// github_projects) and the factory that resolves one per source kind. Adapters
// are pure read: they fetch + normalize an external plan into the source-agnostic
// FetchResult DTO. ALL business decisions (status projection, identity merge,
// locking, hashing) live in service.PlanningImportService, never here.
package planning

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/adapter/markdown"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// MarkdownAdapter normalizes a generic frontmatter markdown blob (the existing
// markdown.ParseStoryMarkdown shape) into the planning import DTOs. It WRAPS the
// parser — it never reimplements it — so the parser tests stay authoritative.
type MarkdownAdapter struct{}

// NewMarkdownAdapter creates a new MarkdownAdapter.
func NewMarkdownAdapter() *MarkdownAdapter { return &MarkdownAdapter{} }

// Kind reports the source discriminator this adapter handles.
func (a *MarkdownAdapter) Kind() port.SourceKind { return port.SourceMarkdown }

// Fetch parses cfg.Markdown.Content and emits a FetchResult. It is lossless for
// every field the markdown shape carries:
//   - key            -> ExternalID + Key (markdown resolves by key)
//   - H1 title       -> Title
//   - body           -> AcceptanceCriteria
//   - scope          -> Scope (validated {backend,frontend,shared}; else nil + warning)
//   - depends_on     -> DependsOn
//   - epic           -> EpicRef + a deduped ImportedEpic named by the epic string
//   - status         -> RawStatus (the SERVICE projects it to {backlog,done})
//
// It carries NO objective (markdown has no slot — always nil so the service
// preserves any in-app value), no source_url (""), and never touches target_files.
// An empty / unparseable blob yields an empty FetchResult (the service treats that
// as a valid 200 zero-import).
func (a *MarkdownAdapter) Fetch(_ context.Context, _ uuid.UUID, cfg port.ImportConfig) (*port.FetchResult, error) {
	content := ""
	if cfg.Markdown != nil {
		content = cfg.Markdown.Content
	}

	parsed := markdown.ParseStoryMarkdown(content)
	res := &port.FetchResult{
		Epics:    []port.ImportedEpic{},
		Stories:  []port.ImportedStory{},
		Warnings: []port.ImportWarning{},
	}

	seenEpic := map[string]bool{}
	for _, p := range parsed {
		var epicRef *port.SourceRef
		if p.Epic != "" {
			ref := port.SourceRef{Source: port.SourceMarkdown, ExternalID: p.Epic}
			epicRef = &ref
			if !seenEpic[p.Epic] {
				seenEpic[p.Epic] = true
				// Frontmatter carries no epic title => name = key (the epic string).
				res.Epics = append(res.Epics, port.ImportedEpic{
					Ref:  ref,
					Key:  p.Epic,
					Name: p.Epic,
				})
			}
		}

		scope, scopeWarn := normalizeScope(p.Key, p.Scope)
		if scopeWarn != nil {
			res.Warnings = append(res.Warnings, *scopeWarn)
		}

		res.Stories = append(res.Stories, port.ImportedStory{
			Ref:                port.SourceRef{Source: port.SourceMarkdown, ExternalID: p.Key},
			Key:                p.Key,
			Title:              p.Title,
			Objective:          nil, // lossless: markdown has no objective slot
			AcceptanceCriteria: ptrIfNonEmpty(p.AcceptanceCriteria),
			Scope:              scope,
			DependsOn:          p.DependsOn,
			EpicRef:            epicRef,
			RawStatus:          p.Status, // projected to {backlog,done} by the service
			ParseError:         p.ParseError,
		})
	}

	return res, nil
}

// ptrIfNonEmpty returns a pointer to s, or nil when s is empty (so the service
// preserves an absent field rather than nulling it out).
func ptrIfNonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
