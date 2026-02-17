package service

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// mockCostRepo is a mock implementation of port.CostRepository for testing.
type mockCostRepo struct {
	inserted []*model.CostRecord
	insertErr error
}

func (m *mockCostRepo) InsertCostRecord(_ context.Context, record *model.CostRecord) (*model.CostRecord, error) {
	if m.insertErr != nil {
		return nil, m.insertErr
	}
	out := *record
	out.ID = uuid.New()
	out.CreatedAt = time.Now()
	m.inserted = append(m.inserted, &out)
	return &out, nil
}

func (m *mockCostRepo) GetCostByRunStep(_ context.Context, _ uuid.UUID) (*model.CostRecord, error) {
	return nil, nil
}

func (m *mockCostRepo) SumCostByProject(_ context.Context, _ uuid.UUID, _ time.Time) (float64, int64, int64, error) {
	return 0, 0, 0, nil
}

func (m *mockCostRepo) SumCostByRun(_ context.Context, _ uuid.UUID) (float64, error) {
	return 0, nil
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestCostService_RecordStepCost_EmptyEvents(t *testing.T) {
	repo := &mockCostRepo{}
	svc := NewCostService(repo, newTestLogger())

	err := svc.RecordStepCost(context.Background(), uuid.New(), uuid.New(), nil)
	if err != nil {
		t.Fatalf("expected nil error for empty events, got: %v", err)
	}
	if len(repo.inserted) != 0 {
		t.Errorf("expected no inserts for empty events, got %d", len(repo.inserted))
	}
}

func TestCostService_RecordStepCost_KnownModels(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		inputTokens  int64
		outputTokens int64
		wantCost     float64
	}{
		{
			name:         "opus",
			model:        "claude-opus-4-6",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			wantCost:     15.0 + 75.0,
		},
		{
			name:         "sonnet",
			model:        "claude-sonnet-4-5",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			wantCost:     3.0 + 15.0,
		},
		{
			name:         "haiku",
			model:        "claude-haiku-4-3",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			wantCost:     0.25 + 1.25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockCostRepo{}
			svc := NewCostService(repo, newTestLogger())

			stepID := uuid.New()
			projectID := uuid.New()
			events := []model.CostEvent{
				{InputTokens: tt.inputTokens, OutputTokens: tt.outputTokens, Model: tt.model},
			}

			err := svc.RecordStepCost(context.Background(), stepID, projectID, events)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(repo.inserted) != 1 {
				t.Fatalf("expected 1 insert, got %d", len(repo.inserted))
			}
			rec := repo.inserted[0]
			if rec.RunStepID != stepID {
				t.Errorf("expected step_id %v, got %v", stepID, rec.RunStepID)
			}
			if rec.ProjectID != projectID {
				t.Errorf("expected project_id %v, got %v", projectID, rec.ProjectID)
			}
			const epsilon = 0.000001
			if diff := rec.CostUSD - tt.wantCost; diff > epsilon || diff < -epsilon {
				t.Errorf("expected cost %.6f, got %.6f", tt.wantCost, rec.CostUSD)
			}
		})
	}
}

func TestCostService_RecordStepCost_UnknownModel(t *testing.T) {
	repo := &mockCostRepo{}
	svc := NewCostService(repo, newTestLogger())

	events := []model.CostEvent{
		{InputTokens: 1000, OutputTokens: 500, Model: "unknown-model-xyz"},
	}

	err := svc.RecordStepCost(context.Background(), uuid.New(), uuid.New(), events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.inserted) != 1 {
		t.Fatalf("expected 1 insert, got %d", len(repo.inserted))
	}
	if repo.inserted[0].CostUSD != 0 {
		t.Errorf("expected cost_usd 0 for unknown model, got %f", repo.inserted[0].CostUSD)
	}
}

func TestCostService_RecordStepCost_MultipleEventsAggregated(t *testing.T) {
	repo := &mockCostRepo{}
	svc := NewCostService(repo, newTestLogger())

	events := []model.CostEvent{
		{InputTokens: 500_000, OutputTokens: 250_000, Model: "claude-sonnet-4-5"},
		{InputTokens: 500_000, OutputTokens: 250_000, Model: "claude-sonnet-4-5"},
	}

	err := svc.RecordStepCost(context.Background(), uuid.New(), uuid.New(), events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Must be a single insert, not two
	if len(repo.inserted) != 1 {
		t.Fatalf("expected 1 insert (aggregated), got %d", len(repo.inserted))
	}
	rec := repo.inserted[0]
	if rec.TokensInput != 1_000_000 {
		t.Errorf("expected total_input 1000000, got %d", rec.TokensInput)
	}
	if rec.TokensOutput != 500_000 {
		t.Errorf("expected total_output 500000, got %d", rec.TokensOutput)
	}
	// sonnet: (1M/1M)*3 + (500K/1M)*15 = 3 + 7.5 = 10.5
	const wantCost = 10.5
	const epsilon = 0.000001
	if diff := rec.CostUSD - wantCost; diff > epsilon || diff < -epsilon {
		t.Errorf("expected cost %.6f, got %.6f", wantCost, rec.CostUSD)
	}
}

func TestCostService_RecordStepCost_RepoError(t *testing.T) {
	repo := &mockCostRepo{insertErr: context.DeadlineExceeded}
	svc := NewCostService(repo, newTestLogger())

	events := []model.CostEvent{
		{InputTokens: 100, OutputTokens: 50, Model: "claude-opus-4-6"},
	}

	err := svc.RecordStepCost(context.Background(), uuid.New(), uuid.New(), events)
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
}
