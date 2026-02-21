package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/adapter/postgres"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

func TestIntegration_EpicRunRepo_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	queries := postgres.New(db.pool)
	repo := postgres.NewEpicRunRepo(queries)

	projectID := createTestProject(t, db.pool)

	// Create an epic directly for FK
	epicID := uuid.New()
	_, err := db.pool.Exec(ctx,
		`INSERT INTO epics (id, project_id, name, status)
		 VALUES ($1, $2, $3, $4)`,
		epicID, projectID, "Test Epic", "draft",
	)
	if err != nil {
		t.Fatalf("failed to create test epic: %v", err)
	}

	// Create a story for FK
	storyID := createTestStory(t, db.pool, projectID)

	// Test CreateEpicRun
	epicRun, err := repo.CreateEpicRun(ctx, &model.EpicRun{
		ProjectID: projectID,
		EpicID:    epicID,
		Status:    model.EpicRunStatusPending,
	})
	if err != nil {
		t.Fatalf("CreateEpicRun() error = %v", err)
	}
	if epicRun.ID == uuid.Nil {
		t.Error("expected non-nil epic run ID")
	}
	if epicRun.Status != model.EpicRunStatusPending {
		t.Errorf("expected status pending, got %s", epicRun.Status)
	}

	// Test GetEpicRun
	got, err := repo.GetEpicRun(ctx, epicRun.ID)
	if err != nil {
		t.Fatalf("GetEpicRun() error = %v", err)
	}
	if got.ID != epicRun.ID {
		t.Errorf("GetEpicRun() ID = %v, want %v", got.ID, epicRun.ID)
	}
	if got.ProjectID != projectID {
		t.Errorf("GetEpicRun() ProjectID = %v, want %v", got.ProjectID, projectID)
	}

	// Test InsertEpicRunStory
	err = repo.InsertEpicRunStory(ctx, model.EpicRunStory{
		EpicRunID:  epicRun.ID,
		StoryID:    storyID,
		GroupIndex: 0,
		Status:     "pending",
	})
	if err != nil {
		t.Fatalf("InsertEpicRunStory() error = %v", err)
	}

	// Test ListEpicRunStories
	stories, err := repo.ListEpicRunStories(ctx, epicRun.ID)
	if err != nil {
		t.Fatalf("ListEpicRunStories() error = %v", err)
	}
	if len(stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(stories))
	}
	if stories[0].StoryID != storyID {
		t.Errorf("expected story ID %v, got %v", storyID, stories[0].StoryID)
	}

	// Test UpdateEpicRunStoryStatus
	runID := uuid.New()
	// Create a run for FK
	_, err = db.pool.Exec(ctx,
		`INSERT INTO runs (id, project_id, story_id, status, pipeline_config_snapshot)
		 VALUES ($1, $2, $3, $4, $5)`,
		runID, projectID, storyID, "running", `{"steps":[]}`,
	)
	if err != nil {
		t.Fatalf("failed to create test run: %v", err)
	}

	err = repo.UpdateEpicRunStoryStatus(ctx, epicRun.ID, storyID, "running", &runID)
	if err != nil {
		t.Fatalf("UpdateEpicRunStoryStatus() error = %v", err)
	}

	// Verify story status was updated
	stories, err = repo.ListEpicRunStories(ctx, epicRun.ID)
	if err != nil {
		t.Fatalf("ListEpicRunStories() after update error = %v", err)
	}
	if stories[0].Status != "running" {
		t.Errorf("expected story status running, got %s", stories[0].Status)
	}
	if stories[0].RunID == nil || *stories[0].RunID != runID {
		t.Errorf("expected story run_id %v, got %v", runID, stories[0].RunID)
	}

	// Test UpdateEpicRunStatus
	now := time.Now().UTC().Truncate(time.Microsecond)
	updated, err := repo.UpdateEpicRunStatus(ctx, epicRun.ID, model.EpicRunStatusCompleted, &now)
	if err != nil {
		t.Fatalf("UpdateEpicRunStatus() error = %v", err)
	}
	if updated.Status != model.EpicRunStatusCompleted {
		t.Errorf("expected status completed, got %s", updated.Status)
	}
	if updated.CompletedAt == nil {
		t.Error("expected non-nil completed_at")
	}
}

func TestIntegration_EpicRunRepo_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	queries := postgres.New(db.pool)
	repo := postgres.NewEpicRunRepo(queries)

	_, err := repo.GetEpicRun(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent epic run")
	}
}
