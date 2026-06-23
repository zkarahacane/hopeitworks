package microsandbox

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// Test-only constants to satisfy goconst (literals reused ≥3×).
const (
	skillName    = "review-checklist"
	httpMCPName  = "search"
	stdioMCPName = "secret-fetcher"
)

// compile-time conformance is also asserted in runtime.go via the package-level
// var; this test makes the guarantee visible to the test suite too.
func TestRuntime_ImplementsAgentRuntime(_ *testing.T) {
	var _ port.AgentRuntime = NewRuntime(false, nil, nil)
}

func TestRuntime_SupportedCapabilities(t *testing.T) {
	got := NewRuntime(false, nil, nil).SupportedCapabilities()
	want := model.CapabilitySet{
		Skills:          true,
		MCPServersHTTP:  true,
		MCPServersStdio: false,
		ToolPolicy:      true,
	}
	if got != want {
		t.Fatalf("SupportedCapabilities() = %+v, want %+v", got, want)
	}
}

func TestRuntime_Provision_WarnSkip(t *testing.T) {
	tests := []struct {
		name         string
		spec         model.CapabilitySpec
		wantApplied  []string
		wantWarnings []string // capability refs expected to warn
	}{
		{
			name:        "empty spec applies nothing and warns nothing",
			spec:        model.CapabilitySpec{},
			wantApplied: nil,
		},
		{
			name: "skill is supported and applied",
			spec: model.CapabilitySpec{
				Skills: []model.SkillSpec{{Name: skillName}},
			},
			wantApplied: []string{model.CapabilityKindSkill + "/" + skillName},
		},
		{
			name: "http mcp server is supported and applied",
			spec: model.CapabilitySpec{
				MCPServers: []model.MCPServerSpec{{Name: httpMCPName, Transport: transportHTTP}},
			},
			wantApplied: []string{model.CapabilityKindMCPServer + "/" + httpMCPName},
		},
		{
			name: "stdio mcp server is warn+skip, never error",
			spec: model.CapabilitySpec{
				MCPServers: []model.MCPServerSpec{{Name: stdioMCPName, Transport: transportStdio}},
			},
			wantWarnings: []string{model.CapabilityKindMCPServer + "/" + stdioMCPName},
		},
		{
			name: "unknown transport is warn+skip",
			spec: model.CapabilitySpec{
				MCPServers: []model.MCPServerSpec{{Name: "weird", Transport: "grpc"}},
			},
			wantWarnings: []string{model.CapabilityKindMCPServer + "/weird"},
		},
		{
			name: "tool policy is supported and applied",
			spec: model.CapabilitySpec{
				ToolPolicy: &model.ToolPolicySpec{Allow: []string{"Bash"}},
			},
			wantApplied: []string{model.CapabilityKindToolPolicy},
		},
		{
			name: "mixed spec: applies supported, warns unsupported, never errors",
			spec: model.CapabilitySpec{
				Skills: []model.SkillSpec{{Name: skillName}},
				MCPServers: []model.MCPServerSpec{
					{Name: httpMCPName, Transport: transportHTTP},
					{Name: stdioMCPName, Transport: transportStdio},
				},
				ToolPolicy: &model.ToolPolicySpec{Deny: []string{"WebFetch"}},
			},
			wantApplied: []string{
				model.CapabilityKindSkill + "/" + skillName,
				model.CapabilityKindMCPServer + "/" + httpMCPName,
				model.CapabilityKindToolPolicy,
			},
			wantWarnings: []string{model.CapabilityKindMCPServer + "/" + stdioMCPName},
		},
	}

	rt := NewRuntime(false, nil, nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := rt.Provision(context.Background(), tt.spec)
			if err != nil {
				t.Fatalf("Provision returned error %v; warn+skip must never error", err)
			}
			assertRefs(t, "applied", res.Applied, tt.wantApplied)
			gotWarnRefs := make([]string, 0, len(res.Warnings))
			for _, w := range res.Warnings {
				if w.Reason == "" {
					t.Errorf("warning for %q has empty reason", w.Capability)
				}
				gotWarnRefs = append(gotWarnRefs, w.Capability)
			}
			assertRefs(t, "warnings", gotWarnRefs, tt.wantWarnings)
		})
	}
}

func TestRuntime_LiveStubs_ReturnNotImplemented(t *testing.T) {
	rt := NewRuntime(true, nil, nil) // even enabled, P3a stays a stub
	ctx := context.Background()

	if _, err := rt.Launch(ctx, port.RunSpec{}); !errors.Is(err, ErrNotImplemented) {
		t.Errorf("Launch err = %v, want ErrNotImplemented", err)
	}
	if _, err := rt.Wait(ctx, port.RunHandle{}); !errors.Is(err, ErrNotImplemented) {
		t.Errorf("Wait err = %v, want ErrNotImplemented", err)
	}
	if err := rt.Stop(ctx, port.RunHandle{}); !errors.Is(err, ErrNotImplemented) {
		t.Errorf("Stop err = %v, want ErrNotImplemented", err)
	}
}

func TestRuntime_ResolveImage(t *testing.T) {
	const (
		freeFormImage = "ghcr.io/acme/custom:1.2.3"
		catalogueRef  = "ghcr.io/zakari/hopeitworks-go@sha256:deadbeef"
	)

	tests := []struct {
		name    string
		stacks  port.StackRepository
		image   string
		want    string
		wantErr bool
	}{
		{
			name:  "no catalogue: free-form image passes through",
			image: freeFormImage,
			want:  freeFormImage,
		},
		{
			name:   "catalogue key resolves to catalogued image",
			stacks: &stubStackRepo{byKey: map[string]*model.Stack{model.StackKeyGo: {ImageRef: catalogueRef}}},
			image:  model.StackKeyGo,
			want:   catalogueRef,
		},
		{
			name:   "catalogue miss falls back to free-form image",
			stacks: &stubStackRepo{byKey: map[string]*model.Stack{}},
			image:  freeFormImage,
			want:   freeFormImage,
		},
		{
			name:    "empty image with no catalogue is an error",
			image:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := NewRuntime(false, tt.stacks, nil)
			got, err := rt.ResolveImage(context.Background(), port.RunSpec{Image: tt.image})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ResolveImage() expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveImage() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ResolveImage() = %q, want %q", got, tt.want)
			}
		})
	}
}

// assertRefs compares an unordered set of refs (applied/warning) ignoring order.
func assertRefs(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %v (len %d), want %v (len %d)", label, got, len(got), want, len(want))
	}
	gotSet := make(map[string]int, len(got))
	for _, g := range got {
		gotSet[g]++
	}
	for _, w := range want {
		if gotSet[w] == 0 {
			t.Fatalf("%s: missing %q in %v", label, w, got)
		}
		gotSet[w]--
	}
}

// stubStackRepo is a minimal port.StackRepository for ResolveImage tests.
type stubStackRepo struct {
	byKey map[string]*model.Stack
}

func (s *stubStackRepo) List(_ context.Context) ([]*model.Stack, error) { return nil, nil }

func (s *stubStackRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.Stack, error) {
	return nil, errors.New("not implemented in stub")
}

func (s *stubStackRepo) GetByKey(_ context.Context, key string) (*model.Stack, error) {
	if st, ok := s.byKey[key]; ok {
		return st, nil
	}
	return nil, errors.New("stack not found")
}

func (s *stubStackRepo) Upsert(_ context.Context, st *model.Stack) (*model.Stack, error) {
	return st, nil
}
