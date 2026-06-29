package model

import (
	"time"

	"github.com/google/uuid"
)

// Story status constants.
const (
	StoryStatusBacklog = "backlog"
	StoryStatusRunning = "running"
	StoryStatusDone    = "done"
	StoryStatusFailed  = "failed"
)

// Story scope constants.
const (
	StoryScopeBackend  = "backend"
	StoryScopeFrontend = "frontend"
	StoryScopeShared   = "shared"
)

// StoryCounts holds the number of stories per lifecycle status. Used to populate
// an epic's aggregate progress on the board without N per-status queries.
type StoryCounts struct {
	Backlog int
	Running int
	Done    int
	Failed  int
}

// Story represents a user story within a project.
type Story struct {
	ID                 uuid.UUID
	ProjectID          uuid.UUID
	EpicID             *uuid.UUID
	Key                string
	Title              string
	Objective          *string
	TargetFiles        []string
	DependsOn          []string
	Scope              *string
	Status             string
	AcceptanceCriteria *string
	// CurrentStage is the name of the stage the card currently sits in, advanced by
	// the executor at stage boundaries. Nil means no stage (backlog before the first
	// run, or after run completion).
	CurrentStage *string
	// Planning provenance (see port.SourceKind). Source is the origin discriminator
	// ("manual" | "markdown" | "github_projects"); the pointer fields are nil for
	// in-app/seed rows. These are import-managed; the run engine never writes them.
	Source         string     // port.SourceManual / SourceMarkdown / SourceGitHub
	ExternalID     *string    // remote content node id (github_projects) or key (markdown); nil for manual
	ExternalItemID *string    // ProjectV2Item id (github_projects) — write-back target; nil otherwise
	SourceURL      *string    // deep-link to the source item; nil for manual/markdown
	SyncedAt       *time.Time // last successful import touch
	LastImportHash *string    // sha256 of the last normalized import payload (no-op gate)
	// WritebackStatus is the last-known state of the outbound status push to the
	// external tracker (disabled|pending|synced|failed). Nil for rows that never had
	// a write-back attempt. Managed solely by the write-back path.
	WritebackStatus *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
