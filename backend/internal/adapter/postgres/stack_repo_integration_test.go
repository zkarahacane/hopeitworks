package postgres_test

import (
	"context"
	"testing"

	"github.com/zakari/hopeitworks/backend/internal/adapter/postgres"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

func TestIntegration_StackRepo_ListSeededCatalogue(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	repo := postgres.NewStackRepo(postgres.New(db.pool))

	stacks, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	// Migration 000034 seeds exactly these four catalogued stacks with ghcr refs.
	wantRefs := map[string]string{
		model.StackKeyGo:     "ghcr.io/zkarahacane/hopeitworks/agent-go:latest",
		model.StackKeyNode:   "ghcr.io/zkarahacane/hopeitworks/agent-node:latest",
		model.StackKeyPython: "ghcr.io/zkarahacane/hopeitworks/agent-python:latest",
		model.StackKeyGoNode: "ghcr.io/zkarahacane/hopeitworks/agent-go-node:latest",
	}
	if len(stacks) != len(wantRefs) {
		t.Fatalf("expected %d seeded stacks, got %d", len(wantRefs), len(stacks))
	}
	for _, s := range stacks {
		want, ok := wantRefs[s.Key]
		if !ok {
			t.Errorf("unexpected stack key %q", s.Key)
			continue
		}
		if s.ImageRef != want {
			t.Errorf("stack %q: expected image_ref %q, got %q", s.Key, want, s.ImageRef)
		}
		if len(s.Toolchain) == 0 {
			t.Errorf("stack %q: expected non-empty toolchain jsonb", s.Key)
		}
	}
}

func TestIntegration_StackRepo_GetByKeyAndByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	repo := postgres.NewStackRepo(postgres.New(db.pool))

	byKey, err := repo.GetByKey(ctx, model.StackKeyGoNode)
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if byKey.ImageRef != "ghcr.io/zkarahacane/hopeitworks/agent-go-node:latest" {
		t.Errorf("unexpected image_ref %q", byKey.ImageRef)
	}

	byID, err := repo.GetByID(ctx, byKey.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if byID.Key != model.StackKeyGoNode {
		t.Errorf("expected key %q, got %q", model.StackKeyGoNode, byID.Key)
	}

	if _, err := repo.GetByKey(ctx, "does-not-exist"); err == nil {
		t.Error("expected not-found error for unknown key")
	}
}
