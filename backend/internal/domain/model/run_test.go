package model

import (
	"testing"

	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

func TestValidateRunTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    RunStatus
		to      RunStatus
		wantErr bool
	}{
		// Valid transitions from pending
		{"pending to running", RunStatusPending, RunStatusRunning, false},
		{"pending to cancelled", RunStatusPending, RunStatusCancelled, false},

		// Valid transitions from running
		{"running to completed", RunStatusRunning, RunStatusCompleted, false},
		{"running to failed", RunStatusRunning, RunStatusFailed, false},
		{"running to cancelled", RunStatusRunning, RunStatusCancelled, false},

		// Invalid transitions from pending
		{"pending to completed", RunStatusPending, RunStatusCompleted, true},
		{"pending to failed", RunStatusPending, RunStatusFailed, true},

		// Invalid transitions from completed (terminal state)
		{"completed to running", RunStatusCompleted, RunStatusRunning, true},
		{"completed to pending", RunStatusCompleted, RunStatusPending, true},
		{"completed to failed", RunStatusCompleted, RunStatusFailed, true},

		// Invalid transitions from failed (terminal state)
		{"failed to running", RunStatusFailed, RunStatusRunning, true},
		{"failed to pending", RunStatusFailed, RunStatusPending, true},
		{"failed to completed", RunStatusFailed, RunStatusCompleted, true},

		// Invalid transitions from cancelled (terminal state)
		{"cancelled to running", RunStatusCancelled, RunStatusRunning, true},
		{"cancelled to pending", RunStatusCancelled, RunStatusPending, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRunTransition(tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRunTransition(%s, %s) error = %v, wantErr %v", tt.from, tt.to, err, tt.wantErr)
			}
			if err != nil {
				domainErr, ok := err.(*errors.DomainError)
				if !ok {
					t.Errorf("expected *errors.DomainError, got %T", err)
				}
				if domainErr.Code != "errors.ErrCodeInvalidStateTransition" {
					t.Errorf("expected code errors.ErrCodeInvalidStateTransition, got %s", domainErr.Code)
				}
				if domainErr.Category != errors.CategoryInvalidState {
					t.Errorf("expected category invalid_state, got %s", domainErr.Category)
				}
			}
		})
	}
}

func TestValidateStepTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    StepStatus
		to      StepStatus
		wantErr bool
	}{
		// Valid transitions from pending
		{"pending to running", StepStatusPending, StepStatusRunning, false},
		{"pending to cancelled", StepStatusPending, StepStatusCancelled, false},

		// Valid transitions from running
		{"running to completed", StepStatusRunning, StepStatusCompleted, false},
		{"running to failed", StepStatusRunning, StepStatusFailed, false},
		{"running to cancelled", StepStatusRunning, StepStatusCancelled, false},

		// Invalid transitions from pending
		{"pending to completed", StepStatusPending, StepStatusCompleted, true},
		{"pending to failed", StepStatusPending, StepStatusFailed, true},

		// Invalid transitions from completed (terminal state)
		{"completed to running", StepStatusCompleted, StepStatusRunning, true},
		{"completed to pending", StepStatusCompleted, StepStatusPending, true},

		// Invalid transitions from failed (terminal state)
		{"failed to running", StepStatusFailed, StepStatusRunning, true},
		{"failed to pending", StepStatusFailed, StepStatusPending, true},

		// Invalid transitions from cancelled (terminal state)
		{"cancelled to running", StepStatusCancelled, StepStatusRunning, true},
		{"cancelled to pending", StepStatusCancelled, StepStatusPending, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStepTransition(tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStepTransition(%s, %s) error = %v, wantErr %v", tt.from, tt.to, err, tt.wantErr)
			}
			if err != nil {
				domainErr, ok := err.(*errors.DomainError)
				if !ok {
					t.Errorf("expected *errors.DomainError, got %T", err)
				}
				if domainErr.Code != "errors.ErrCodeInvalidStateTransition" {
					t.Errorf("expected code errors.ErrCodeInvalidStateTransition, got %s", domainErr.Code)
				}
				if domainErr.Category != errors.CategoryInvalidState {
					t.Errorf("expected category invalid_state, got %s", domainErr.Category)
				}
			}
		})
	}
}
