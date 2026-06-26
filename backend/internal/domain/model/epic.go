package model

import (
	"time"

	"github.com/google/uuid"
)

// Epic status constants.
const (
	EpicStatusBacklog    = "backlog"
	EpicStatusInProgress = "in_progress"
	EpicStatusDone       = "done"
)

// Epic represents an epic within a project.
type Epic struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	Name        string
	Description *string
	Status      string
	// Planning provenance (see port.SourceKind). Source is the origin discriminator
	// ("manual" | "markdown" | "github_projects"); the pointer fields are nil for
	// in-app/seed rows. Import-managed; the run engine never writes them. Epics carry
	// no last_import_hash (idempotency is computed from the merged tuple in Go).
	Source     string     // port.SourceManual / SourceMarkdown / SourceGitHub
	ExternalID *string    // remote node id (github_projects); nil for manual/markdown-by-name
	SourceURL  *string    // deep-link to the source item; nil for manual/markdown
	SyncedAt   *time.Time // last successful import touch
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
