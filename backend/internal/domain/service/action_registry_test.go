package service

import (
	"context"
	"testing"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// stubAction is a minimal model.Action implementation for testing the registry.
type stubAction struct {
	name string
}

func (a *stubAction) Name() string                                         { return a.name }
func (a *stubAction) Execute(_ context.Context, _ *model.RunContext) error { return nil }

func TestInMemoryActionRegistry_RegisterAlias(t *testing.T) {
	reg := NewActionRegistry()
	action := &stubAction{name: "agent_run"}
	reg.Register(action)

	// Register aliases
	for _, alias := range []string{"implement", "review", "merge"} {
		reg.RegisterAlias(alias, action)
	}

	// Original name still resolves (AC #7)
	got, err := reg.Get("agent_run")
	if err != nil {
		t.Fatalf("Get(agent_run) error: %v", err)
	}
	if got != action {
		t.Fatalf("Get(agent_run) returned different instance")
	}

	// Aliases resolve to the same instance (AC #3, #4, #5)
	for _, alias := range []string{"implement", "review", "merge"} {
		got, err := reg.Get(alias)
		if err != nil {
			t.Fatalf("Get(%s) error: %v", alias, err)
		}
		if got != action {
			t.Fatalf("Get(%s) returned different instance than agent_run", alias)
		}
	}

	// Unregistered names still fail
	_, err = reg.Get("nonexistent")
	if err == nil {
		t.Fatal("Get(nonexistent) should have returned an error")
	}
}
