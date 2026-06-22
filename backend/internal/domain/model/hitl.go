package model

import (
	"time"

	"github.com/google/uuid"
)

// HITLStatus represents the approval state of a HITL request.
type HITLStatus string

const (
	HITLStatusPending  HITLStatus = "pending"
	HITLStatusApproved HITLStatus = "approved"
	HITLStatusRejected HITLStatus = "rejected"
	// HITLStatusResolved is the terminal status for a probe_halt gate closed via
	// an enriched resolution action (resume/override/send_back/skip/abort). The
	// specific action taken is recorded in ResolutionAction.
	HITLStatusResolved HITLStatus = "resolved"
)

// HITL gate_type values. The column is a free VARCHAR; probe_halt is the
// watchdog-raised variant (vs the "approval"/"human" review gates).
const (
	HITLGateApproval  = "approval"
	HITLGateHuman     = "human"
	HITLGateProbeHalt = "probe_halt"
)

// Halt-gate resolution actions (§7 of the agents model). A review gate uses
// approve/reject; a probe_halt gate uses this richer set. Each is a durable
// stage transition recorded with the resolving human.
const (
	// HITLActionResume retries the halted step fresh (re-enqueue the run).
	HITLActionResume = "resume"
	// HITLActionOverride accepts the partial result and advances (false positive:
	// the agent finished but did not signal). Explicit + audited.
	HITLActionOverride = "override"
	// HITLActionSendBack returns the card to an earlier stage (e.g. Needs Spec).
	HITLActionSendBack = "send_back"
	// HITLActionSkip advances past this stage/step without running it.
	HITLActionSkip = "skip"
	// HITLActionAbort fails the card.
	HITLActionAbort = "abort"
)

// HaltReason is the structured reason a probe halted a run. It is persisted on
// the HITL request (halt_reason JSONB) so the resolution UI can show what
// breached and suggest a remedy (cost → resume +budget; log_silence/wallclock →
// retry fresh).
type HaltReason struct {
	// Probe is the guard kind that breached: log_silence | wallclock | cost_batch.
	Probe string `json:"probe"`
	// OnFail is the configured action for the breached guard (always halt-gate
	// here, since fail/retry don't raise a gate).
	OnFail string `json:"on_fail,omitempty"`
	// Observed is the measured value at breach (seconds of silence/runtime, or USD).
	Observed float64 `json:"observed"`
	// Threshold is the configured limit that was exceeded.
	Threshold float64 `json:"threshold"`
	// Unit is the unit of Observed/Threshold: "seconds" or "usd".
	Unit string `json:"unit,omitempty"`
	// Detail is an optional human-readable summary.
	Detail string `json:"detail,omitempty"`
}

// HITLRequest records a human-in-the-loop gate triggered by a pipeline step.
type HITLRequest struct {
	ID              uuid.UUID
	RunStepID       uuid.UUID
	GateType        string  // "approval", "human" or "probe_halt"
	DiffContent     *string // PR diff fetched from GitProvider; nil if unavailable
	Message         *string // optional human-readable message for the reviewer
	Status          HITLStatus
	ResolvedAt      *time.Time
	ResolvedBy      *uuid.UUID // user ID who resolved
	RejectionReason *string
	// HaltReason is the structured probe-breach reason for a probe_halt gate; nil
	// for review gates.
	HaltReason *HaltReason
	// ResolutionAction is the enriched halt-gate action taken (resume/override/
	// send_back/skip/abort); nil for plain approve/reject.
	ResolutionAction *string
	CreatedAt        time.Time
}

// ValidHITLResolutionAction reports whether s is a recognized halt-gate
// resolution action.
func ValidHITLResolutionAction(s string) bool {
	switch s {
	case HITLActionResume, HITLActionOverride, HITLActionSendBack, HITLActionSkip, HITLActionAbort:
		return true
	default:
		return false
	}
}

// PendingHITLRequest is a denormalized view of a pending HITL request
// including run and story context for the pending listing endpoint.
type PendingHITLRequest struct {
	ID        uuid.UUID
	RunID     uuid.UUID
	StepID    uuid.UUID
	StoryKey  string
	DiffURL   *string
	CreatedAt time.Time
}

// ProbeHalt is a denormalized view of a pending probe_halt gate for the batch
// triage inbox: enough context to group by reason and act in bulk.
type ProbeHalt struct {
	ID         uuid.UUID
	RunStepID  uuid.UUID
	RunID      uuid.UUID
	ProjectID  uuid.UUID
	StoryKey   string
	StoryTitle string
	StepName   string
	StageName  string
	HaltReason *HaltReason
	CreatedAt  time.Time
}
