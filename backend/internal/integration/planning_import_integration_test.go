package integration

import (
	"context"
	"testing"

	"github.com/zakari/hopeitworks/backend/internal/adapter/postgres"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/testutil"
)

const planningImportMarkdown = `---
key: PI-1
epic: PI-Epic
scope: backend
status: done
---
# First imported story

Acceptance one.

---
key: PI-2
epic: PI-Epic
status: backlog
---
# Second imported story

Acceptance two.`

// TestIntegration_PlanningImport_MarkdownIdempotent validates the provenance-aware
// upsert against a real Postgres: a first import creates rows + stamps provenance
// (partial unique index participation), and an unchanged re-import is a true no-op
// (Skipped) that never duplicates.
func TestIntegration_PlanningImport_MarkdownIdempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer db.Cleanup()

	ctx := context.Background()
	queries := postgres.New(db.Pool)
	storyRepo := postgres.NewStoryRepo(queries)
	epicRepo := postgres.NewEpicRepo(queries)
	importSvc := newMarkdownImportService(storyRepo, queries)

	projectID := testutil.CreateProject(t, db.Pool)
	cfg := port.ImportConfig{Source: port.SourceMarkdown, Markdown: &port.MarkdownConfig{Content: planningImportMarkdown}}

	first, err := importSvc.Import(ctx, projectID, cfg)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if first.StoriesCreated != 2 {
		t.Errorf("expected 2 stories created, got %d (errors=%v)", first.StoriesCreated, first.Errors)
	}
	if first.EpicsCreated != 1 {
		t.Errorf("expected 1 epic created, got %d", first.EpicsCreated)
	}

	// Provenance persisted + explicit status:done honored on a brand-new row.
	s1, err := storyRepo.GetByKey(ctx, projectID, "PI-1")
	if err != nil {
		t.Fatalf("GetByKey PI-1: %v", err)
	}
	if s1.Source != string(port.SourceMarkdown) {
		t.Errorf("PI-1 source = %q, want markdown", s1.Source)
	}
	if s1.ExternalID == nil || *s1.ExternalID != "PI-1" {
		t.Errorf("PI-1 external_id = %v, want PI-1", s1.ExternalID)
	}
	if s1.LastImportHash == nil {
		t.Errorf("PI-1 last_import_hash should be set")
	}
	if s1.Status != model.StoryStatusDone {
		t.Errorf("PI-1 status = %q, want done", s1.Status)
	}
	if s1.EpicID == nil {
		t.Errorf("PI-1 should be linked to the PI-Epic")
	}
	s2, err := storyRepo.GetByKey(ctx, projectID, "PI-2")
	if err != nil {
		t.Fatalf("GetByKey PI-2: %v", err)
	}
	if s2.Status != model.StoryStatusBacklog {
		t.Errorf("PI-2 status = %q, want backlog", s2.Status)
	}

	// Unchanged re-import is hash-gated to a true no-op, no duplicates.
	second, err := importSvc.Import(ctx, projectID, cfg)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if second.Skipped != 2 {
		t.Errorf("expected 2 skipped on unchanged re-import, got %d", second.Skipped)
	}
	if second.StoriesCreated != 0 || second.StoriesUpdated != 0 {
		t.Errorf("re-import must be a no-op, got created=%d updated=%d", second.StoriesCreated, second.StoriesUpdated)
	}
	if second.EpicsCreated != 0 || second.EpicsUpdated != 0 {
		t.Errorf("epic re-import must be a no-op, got created=%d updated=%d", second.EpicsCreated, second.EpicsUpdated)
	}

	storyCount, err := storyRepo.CountByProject(ctx, projectID)
	if err != nil {
		t.Fatalf("CountByProject: %v", err)
	}
	if storyCount != 2 {
		t.Errorf("re-import duplicated rows: expected 2 stories, got %d", storyCount)
	}
	epicCount, err := epicRepo.CountByProject(ctx, projectID)
	if err != nil {
		t.Fatalf("epic CountByProject: %v", err)
	}
	if epicCount != 1 {
		t.Errorf("re-import duplicated epics: expected 1 epic, got %d", epicCount)
	}
}

// TestIntegration_PlanningImport_SelfHealManualRow validates that a markdown
// re-import resolves a legacy source='manual' row BY KEY and stamps it in place
// (no duplicate), exercising the partial unique index once external_id is set.
func TestIntegration_PlanningImport_SelfHealManualRow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer db.Cleanup()

	ctx := context.Background()
	queries := postgres.New(db.Pool)
	storyRepo := postgres.NewStoryRepo(queries)
	importSvc := newMarkdownImportService(storyRepo, queries)

	projectID := testutil.CreateProject(t, db.Pool)

	// A legacy in-app/seed row (source defaults to 'manual', external_id NULL).
	manual, err := storyRepo.Create(ctx, &model.Story{
		ProjectID: projectID, Key: "PI-1", Title: "old title", Status: model.StoryStatusBacklog,
	})
	if err != nil {
		t.Fatalf("seed manual story: %v", err)
	}
	if manual.Source != string(port.SourceManual) {
		t.Fatalf("seed row should be source=manual, got %q", manual.Source)
	}

	md := "---\nkey: PI-1\n---\n# new title\n\nBody."
	cfg := port.ImportConfig{Source: port.SourceMarkdown, Markdown: &port.MarkdownConfig{Content: md}}
	sum, err := importSvc.Import(ctx, projectID, cfg)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if sum.StoriesUpdated != 1 || sum.StoriesCreated != 0 {
		t.Errorf("self-heal should update in place, got created=%d updated=%d", sum.StoriesCreated, sum.StoriesUpdated)
	}

	count, err := storyRepo.CountByProject(ctx, projectID)
	if err != nil {
		t.Fatalf("CountByProject: %v", err)
	}
	if count != 1 {
		t.Errorf("self-heal must not duplicate: expected 1 story, got %d", count)
	}

	healed, err := storyRepo.GetByKey(ctx, projectID, "PI-1")
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if healed.ID != manual.ID {
		t.Errorf("self-heal must reuse the same row id")
	}
	if healed.Source != string(port.SourceMarkdown) {
		t.Errorf("healed source = %q, want markdown", healed.Source)
	}
	if healed.ExternalID == nil || *healed.ExternalID != "PI-1" {
		t.Errorf("healed external_id = %v, want PI-1", healed.ExternalID)
	}
	if healed.Title != "new title" {
		t.Errorf("healed title = %q, want 'new title'", healed.Title)
	}
}
