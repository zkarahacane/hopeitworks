package service

import (
	"context"
	"testing"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

func TestSeedStacks_UpsertsEachEntry(t *testing.T) {
	repo := newMockStackRepo()
	catalogue := []model.Stack{
		{Key: model.StackKeyGo, ImageRef: "ghcr.io/x/agent-go:latest", Toolchain: []byte(`{"go":"1.23"}`)},
		{Key: model.StackKeyNode, ImageRef: "ghcr.io/x/agent-node:latest", Toolchain: []byte(`{"node":"22"}`)},
	}

	if err := SeedStacks(context.Background(), repo, catalogue, discardLogger()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(repo.upsertLog) != 2 {
		t.Fatalf("expected 2 upserts, got %d", len(repo.upsertLog))
	}
	if repo.upsertLog[0].Key != model.StackKeyGo || repo.upsertLog[1].Key != model.StackKeyNode {
		t.Errorf("unexpected upsert order: %+v", repo.upsertLog)
	}
	if string(repo.upsertLog[0].Toolchain) != `{"go":"1.23"}` {
		t.Errorf("toolchain not passed through: %q", repo.upsertLog[0].Toolchain)
	}
}

func TestSeedStacks_EmptyCatalogueIsNoop(t *testing.T) {
	repo := newMockStackRepo()

	if err := SeedStacks(context.Background(), repo, nil, discardLogger()); err != nil {
		t.Fatalf("expected no error for nil catalogue, got %v", err)
	}
	if err := SeedStacks(context.Background(), repo, []model.Stack{}, discardLogger()); err != nil {
		t.Fatalf("expected no error for empty catalogue, got %v", err)
	}
	if len(repo.upsertLog) != 0 {
		t.Fatalf("expected no upserts for empty catalogue, got %d", len(repo.upsertLog))
	}
}

func TestSeedStacks_PropagatesUpsertError(t *testing.T) {
	repo := newMockStackRepo()
	wantErr := errors.NewInternal("boom", nil)
	repo.upsertFn = func(_ *model.Stack) (*model.Stack, error) {
		return nil, wantErr
	}

	catalogue := []model.Stack{{Key: model.StackKeyGo, ImageRef: "ghcr.io/x/agent-go:latest"}}
	err := SeedStacks(context.Background(), repo, catalogue, discardLogger())
	if err == nil {
		t.Fatal("expected error to propagate from Upsert, got nil")
	}
}
