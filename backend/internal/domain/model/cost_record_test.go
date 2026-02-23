package model

import (
	"testing"
)

func TestComputeCostUSD(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		inputTokens   int64
		outputTokens  int64
		wantCostAbove float64 // cost must be > wantCostAbove to verify non-zero
		wantKnown     bool
	}{
		{
			name:         "exact match claude-opus-4-6",
			model:        "claude-opus-4-6",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			// 15.0 + 75.0 = 90.0 USD
			wantCostAbove: 89.0,
			wantKnown:     true,
		},
		{
			name:         "prefix match full model ID claude-opus-4-6-20251101",
			model:        "claude-opus-4-6-20251101",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			// Same pricing as claude-opus-4-6 via prefix match
			wantCostAbove: 89.0,
			wantKnown:     true,
		},
		{
			name:         "prefix match full model ID claude-sonnet-4-6 with date suffix",
			model:        "claude-sonnet-4-6-20251101",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			// 3.0 + 15.0 = 18.0 USD
			wantCostAbove: 17.0,
			wantKnown:     true,
		},
		{
			name:         "prefix match claude-haiku-4-5 with date suffix",
			model:        "claude-haiku-4-5-20241022",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			// 0.25 + 1.25 = 1.5 USD
			wantCostAbove: 1.0,
			wantKnown:     true,
		},
		{
			name:          "unknown model returns not known",
			model:         "gpt-4-turbo",
			inputTokens:   1_000,
			outputTokens:  500,
			wantCostAbove: -1, // irrelevant
			wantKnown:     false,
		},
		{
			name:          "zero tokens returns zero cost but known",
			model:         "claude-opus-4-6",
			inputTokens:   0,
			outputTokens:  0,
			wantCostAbove: -1, // zero cost is fine
			wantKnown:     true,
		},
		{
			name:         "realistic usage from claude code result event",
			model:        "claude-opus-4-6-20251101",
			inputTokens:  12450,
			outputTokens: 2310,
			// 12450/1e6 * 15.0 + 2310/1e6 * 75.0 = 0.18675 + 0.17325 = 0.36 USD
			wantCostAbove: 0.0,
			wantKnown:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, known := ComputeCostUSD(tt.model, tt.inputTokens, tt.outputTokens)
			if known != tt.wantKnown {
				t.Errorf("expected known=%v, got %v", tt.wantKnown, known)
			}
			if tt.wantKnown && tt.wantCostAbove >= 0 && cost <= tt.wantCostAbove {
				t.Errorf("expected cost > %f, got %f", tt.wantCostAbove, cost)
			}
			if !tt.wantKnown && cost != 0 {
				t.Errorf("expected cost=0 for unknown model, got %f", cost)
			}
		})
	}
}

func TestComputeCostUSDPrefixMatchCorrectness(t *testing.T) {
	// Verify that prefix match returns the same result as exact match for base models.
	base := "claude-opus-4-6"
	versioned := "claude-opus-4-6-20251101"
	tokens := int64(500_000)

	baseCost, baseKnown := ComputeCostUSD(base, tokens, tokens)
	versionedCost, versionedKnown := ComputeCostUSD(versioned, tokens, tokens)

	if !baseKnown {
		t.Fatalf("base model %q should be known", base)
	}
	if !versionedKnown {
		t.Fatalf("versioned model %q should be known via prefix match", versioned)
	}
	if baseCost != versionedCost {
		t.Errorf("expected same cost for base and versioned model: base=%f versioned=%f", baseCost, versionedCost)
	}
}
