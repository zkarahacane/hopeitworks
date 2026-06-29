package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/adapter/postgres"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

func TestIntegration_PlanningConnectorRepo_UpsertGetRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	repo := postgres.NewPlanningConnectorRepository(postgres.New(db.pool))
	projectID := createTestProject(t, db.pool)

	// Absent row -> not found.
	if _, err := repo.Get(ctx, projectID); err == nil {
		t.Fatal("expected not-found for a project with no connector")
	}

	url := "https://github.com/orgs/acme/projects/7"
	done := "OPT_DONE"
	running := "OPT_RUNNING"
	created, err := repo.Upsert(ctx, &model.PlanningConnector{
		ProjectID:        projectID,
		Source:           string(port.SourceGitHub),
		ProjectURL:       &url,
		StatusField:      "Status",
		DoneOptions:      []string{"Done", "Shipped"},
		EpicIssueType:    "Epic",
		StatusMapping:    model.PlanningStatusMapping{Done: &done, Running: &running},
		WritebackEnabled: true,
		PostRunComment:   true,
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if created.ProjectURL == nil || *created.ProjectURL != url {
		t.Fatalf("project_url round-trip failed: %+v", created.ProjectURL)
	}
	if len(created.DoneOptions) != 2 || created.DoneOptions[1] != "Shipped" {
		t.Fatalf("done_options JSONB round-trip failed: %v", created.DoneOptions)
	}
	if created.StatusMapping.Done == nil || *created.StatusMapping.Done != "OPT_DONE" {
		t.Fatalf("status_mapping JSONB round-trip failed: %+v", created.StatusMapping)
	}
	if created.StatusMapping.Backlog != nil || created.StatusMapping.Failed != nil {
		t.Fatalf("unset mapping targets must stay nil: %+v", created.StatusMapping)
	}
	if !created.WritebackEnabled || !created.PostRunComment {
		t.Fatalf("toggles round-trip failed: %+v", created)
	}

	// Get returns the same row.
	got, err := repo.Get(ctx, projectID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ProjectID != projectID || !got.WritebackEnabled {
		t.Fatalf("unexpected fetched connector: %+v", got)
	}

	// Re-upsert (project_id PK -> ON CONFLICT updates the same row).
	updated, err := repo.Upsert(ctx, &model.PlanningConnector{
		ProjectID:        projectID,
		Source:           string(port.SourceGitHub),
		StatusField:      "State",
		EpicIssueType:    "Epic",
		WritebackEnabled: false,
	})
	if err != nil {
		t.Fatalf("re-Upsert: %v", err)
	}
	if updated.StatusField != "State" || updated.WritebackEnabled {
		t.Fatalf("re-upsert did not replace fields: %+v", updated)
	}
	if len(updated.DoneOptions) != 0 {
		t.Fatalf("done_options should reset to empty on re-upsert, got %v", updated.DoneOptions)
	}
}

func TestIntegration_PlanningWriteBackRepo_CreateAndList(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	repo := postgres.NewPlanningWriteBackRepository(postgres.New(db.pool))

	projectID := uuid.New()
	storyID := uuid.New()
	runID := uuid.New()
	source := string(port.SourceGitHub)
	extID := "ITEM_ID"
	internal := model.StoryStatusDone
	remote := "Done"

	ok, err := repo.Create(ctx, &model.PlanningWriteBack{
		ProjectID:      projectID,
		StoryID:        storyID,
		RunID:          &runID,
		Source:         &source,
		ExternalID:     &extID,
		InternalStatus: &internal,
		RemoteStatus:   &remote,
		Success:        true,
	})
	if err != nil {
		t.Fatalf("Create success row: %v", err)
	}
	if ok.ID == uuid.Nil || ok.RunID == nil || *ok.RunID != runID {
		t.Fatalf("unexpected created row: %+v", ok)
	}

	code := "UNAUTHORIZED"
	msg := "bad credentials"
	if _, err := repo.Create(ctx, &model.PlanningWriteBack{
		ProjectID:      projectID,
		StoryID:        storyID,
		Source:         &source,
		InternalStatus: &internal,
		Success:        false,
		ErrorCode:      &code,
		ErrorMessage:   &msg,
	}); err != nil {
		t.Fatalf("Create failure row: %v", err)
	}

	rows, err := repo.ListByStory(ctx, storyID, 10)
	if err != nil {
		t.Fatalf("ListByStory: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 audit rows, got %d", len(rows))
	}
	// Newest first: the failure row was inserted last.
	if rows[0].Success || rows[0].ErrorCode == nil || *rows[0].ErrorCode != "UNAUTHORIZED" {
		t.Fatalf("unexpected ordering / failure row: %+v", rows[0])
	}
}
