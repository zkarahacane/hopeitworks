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
				if domainErr.Code != errors.ErrCodeInvalidStateTransition {
					t.Errorf("expected code %s, got %s", errors.ErrCodeInvalidStateTransition, domainErr.Code)
				}
				if domainErr.Category != errors.CategoryInvalidState {
					t.Errorf("expected category invalid_state, got %s", domainErr.Category)
				}
			}
		})
	}
}

func TestComputeProgress(t *testing.T) {
	tests := []struct {
		name     string
		steps    []RunStep
		expected int
	}{
		{
			name:     "zero steps returns 0",
			steps:    []RunStep{},
			expected: 0,
		},
		{
			name:     "nil steps returns 0",
			steps:    nil,
			expected: 0,
		},
		{
			name: "no completed steps returns 0",
			steps: []RunStep{
				{Status: StepStatusPending},
				{Status: StepStatusRunning},
				{Status: StepStatusPending},
			},
			expected: 0,
		},
		{
			name: "2 of 3 completed returns 66",
			steps: []RunStep{
				{Status: StepStatusCompleted},
				{Status: StepStatusCompleted},
				{Status: StepStatusRunning},
			},
			expected: 66,
		},
		{
			name: "all 3 completed returns 100",
			steps: []RunStep{
				{Status: StepStatusCompleted},
				{Status: StepStatusCompleted},
				{Status: StepStatusCompleted},
			},
			expected: 100,
		},
		{
			name: "1 of 1 completed returns 100",
			steps: []RunStep{
				{Status: StepStatusCompleted},
			},
			expected: 100,
		},
		{
			name: "1 of 2 completed returns 50",
			steps: []RunStep{
				{Status: StepStatusCompleted},
				{Status: StepStatusPending},
			},
			expected: 50,
		},
		{
			name: "1 of 3 completed returns 33",
			steps: []RunStep{
				{Status: StepStatusCompleted},
				{Status: StepStatusFailed},
				{Status: StepStatusPending},
			},
			expected: 33,
		},
		{
			name: "mixed statuses with cancelled",
			steps: []RunStep{
				{Status: StepStatusCompleted},
				{Status: StepStatusCancelled},
				{Status: StepStatusPending},
				{Status: StepStatusPending},
			},
			expected: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Run{}
			result := r.ComputeProgress(tt.steps)
			if result != tt.expected {
				t.Errorf("ComputeProgress() = %d, want %d", result, tt.expected)
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
		{"running to waiting_approval", StepStatusRunning, StepStatusWaitingApproval, false},

		// Valid transitions from waiting_approval
		{"waiting_approval to running", StepStatusWaitingApproval, StepStatusRunning, false},
		{"waiting_approval to completed", StepStatusWaitingApproval, StepStatusCompleted, false},
		{"waiting_approval to failed", StepStatusWaitingApproval, StepStatusFailed, false},
		{"waiting_approval to cancelled", StepStatusWaitingApproval, StepStatusCancelled, false},

		// Invalid transitions from waiting_approval
		{"waiting_approval to pending", StepStatusWaitingApproval, StepStatusPending, true},

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
				if domainErr.Code != errors.ErrCodeInvalidStateTransition {
					t.Errorf("expected code %s, got %s", errors.ErrCodeInvalidStateTransition, domainErr.Code)
				}
				if domainErr.Category != errors.CategoryInvalidState {
					t.Errorf("expected category invalid_state, got %s", domainErr.Category)
				}
			}
		})
	}
}
