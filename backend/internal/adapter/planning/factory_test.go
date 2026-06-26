package planning

import (
	"testing"
)

// TestNormalizeScope verifies §16.8: normalizeScope maps known values to
// canonical lowercase pointers and rejects out-of-enum values with a warning.
func TestNormalizeScope(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantNil   bool   // true => scope pointer must be nil
		wantWarn  bool   // true => a non-nil warning must be returned
		wantCode  string // warning code when wantWarn==true
		wantScope string // expected pointer value when !wantNil
	}{
		{"empty string is absent (nil, nil)", "", true, false, "", ""},
		{"whitespace-only is absent", "   ", true, false, "", ""},
		{"backend lower", "backend", false, false, "", "backend"},
		{"frontend lower", "frontend", false, false, "", "frontend"},
		{"shared lower", "shared", false, false, "", "shared"},
		{"Backend uppercase normalized", "Backend", false, false, "", "backend"},
		{"FRONTEND all-caps normalized", "FRONTEND", false, false, "", "frontend"},
		{"  shared  trimmed", "  shared  ", false, false, "", "shared"},
		{"infra invalid scope → nil + INVALID_SCOPE", "infra", true, true, "INVALID_SCOPE", ""},
		{"devops invalid scope → nil + INVALID_SCOPE", "devops", true, true, "INVALID_SCOPE", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, warn := normalizeScope("K-1", tt.raw)
			if tt.wantNil && got != nil {
				t.Errorf("expected nil scope, got %q", *got)
			}
			if !tt.wantNil {
				if got == nil {
					t.Errorf("expected scope %q, got nil", tt.wantScope)
				} else if *got != tt.wantScope {
					t.Errorf("expected scope %q, got %q", tt.wantScope, *got)
				}
			}
			if tt.wantWarn && warn == nil {
				t.Errorf("expected warning with code %q, got nil", tt.wantCode)
			}
			if !tt.wantWarn && warn != nil {
				t.Errorf("expected no warning, got %+v", warn)
			}
			if tt.wantWarn && warn != nil && warn.Code != tt.wantCode {
				t.Errorf("expected warning code %q, got %q", tt.wantCode, warn.Code)
			}
		})
	}
}
