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
)

// HITLRequest records a human-in-the-loop gate triggered by a pipeline step.
type HITLRequest struct {
	ID              uuid.UUID
	RunStepID       uuid.UUID
	GateType        string  // default "approval"
	DiffContent     *string // PR diff fetched from GitProvider; nil if unavailable
	Status          HITLStatus
	ResolvedAt      *time.Time
	ResolvedBy      *uuid.UUID // user ID who resolved
	RejectionReason *string
	CreatedAt       time.Time
}
