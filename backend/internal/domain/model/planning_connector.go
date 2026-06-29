package model

import (
	"time"

	"github.com/google/uuid"
)

// PlanningWritebackStatus is the last-known state of a story's outbound status push
// to its external tracker. It mirrors the openapi enum and is managed solely by the
// write-back path (the importer and run engine never write it).
type PlanningWritebackStatus string

const (
	// WritebackDisabled means no connector / write-back off / source is not a remote
	// tracker (no push will ever happen for this story).
	WritebackDisabled PlanningWritebackStatus = "disabled"
	// WritebackPending means a write-back is queued or in flight.
	WritebackPending PlanningWritebackStatus = "pending"
	// WritebackSynced means the tracker reflects the internal status.
	WritebackSynced PlanningWritebackStatus = "synced"
	// WritebackFailed means the last write-back attempt errored.
	WritebackFailed PlanningWritebackStatus = "failed"
)

// PlanningStatusMapping maps each internal story status to an external tracker status
// option id. A nil pointer means "do not write back for this status". For GitHub
// Projects v2 the values are single-select option ids.
type PlanningStatusMapping struct {
	Backlog *string `json:"backlog,omitempty"`
	Running *string `json:"running,omitempty"`
	Done    *string `json:"done,omitempty"`
	Failed  *string `json:"failed,omitempty"`
}

// OptionFor returns the configured option id for an internal story status, or ""
// when the status has no usable mapping target.
func (m PlanningStatusMapping) OptionFor(internalStatus string) string {
	var p *string
	switch internalStatus {
	case StoryStatusBacklog:
		p = m.Backlog
	case StoryStatusRunning:
		p = m.Running
	case StoryStatusDone:
		p = m.Done
	case StoryStatusFailed:
		p = m.Failed
	}
	if p == nil {
		return ""
	}
	return *p
}

// HasAnyTarget reports whether at least one internal status maps to a non-empty
// option id (the minimum for write-back to be useful).
func (m PlanningStatusMapping) HasAnyTarget() bool {
	for _, p := range []*string{m.Backlog, m.Running, m.Done, m.Failed} {
		if p != nil && *p != "" {
			return true
		}
	}
	return false
}

// PlanningConnector is a project's persisted planning connector configuration. It
// consolidates the import knobs (status field, done options, epic issue type) and
// the write-back knobs (status mapping, toggles) behind a single per-project row.
type PlanningConnector struct {
	ProjectID        uuid.UUID
	Source           string // port.SourceKind value (github_projects)
	ProjectURL       *string
	StatusField      string
	DoneOptions      []string
	EpicIssueType    string
	StatusMapping    PlanningStatusMapping
	WritebackEnabled bool
	PostRunComment   bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// PlanningWriteBack is one append-only audit row recording a write-back attempt.
type PlanningWriteBack struct {
	ID             uuid.UUID
	ProjectID      uuid.UUID
	StoryID        uuid.UUID
	RunID          *uuid.UUID
	Source         *string
	ExternalID     *string
	InternalStatus *string
	RemoteStatus   *string
	Success        bool
	ErrorCode      *string
	ErrorMessage   *string
	CreatedAt      time.Time
}
