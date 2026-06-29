package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/zakari/hopeitworks/backend/internal/adapter/postgres"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

func TestIntegration_GitConnectionRepo_UpsertGetDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	repo := postgres.NewGitConnectionRepository(postgres.New(db.pool))
	projectID := createTestProject(t, db.pool)

	// Absent row -> not found.
	if _, err := repo.GetByProject(ctx, projectID); err == nil {
		t.Fatal("expected not-found for a project with no connection")
	}

	login := "octocat"
	last4 := "cd12"
	tokenType := string(model.GitTokenTypeClassic)
	exp := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	now := time.Now().UTC().Truncate(time.Second)

	created, err := repo.Upsert(ctx, port.UpsertGitConnectionParams{
		ProjectID:       projectID,
		Provider:        "github",
		EncryptedSecret: []byte("ciphertext-1"),
		SecretLast4:     &last4,
		TokenType:       &tokenType,
		Scopes:          []string{"repo", "read:project"},
		Status:          model.GitConnStatusConnected,
		AccountLogin:    &login,
		ExpiresAt:       &exp,
		LastValidatedAt: &now,
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if created.Status != model.GitConnStatusConnected || created.AccountLogin == nil || *created.AccountLogin != "octocat" {
		t.Fatalf("unexpected created row: %+v", created)
	}
	if len(created.Scopes) != 2 {
		t.Fatalf("scopes round-trip failed: %v", created.Scopes)
	}

	// Re-upsert (UNIQUE(project_id) -> ON CONFLICT updates the SAME row, resets status).
	updated, err := repo.Upsert(ctx, port.UpsertGitConnectionParams{
		ProjectID:       projectID,
		Provider:        "github",
		EncryptedSecret: []byte("ciphertext-2"),
		Status:          model.GitConnStatusConnected,
		Scopes:          []string{"read:project"},
	})
	if err != nil {
		t.Fatalf("re-Upsert: %v", err)
	}
	if updated.ID != created.ID {
		t.Fatalf("UNIQUE(project_id) violated: got a new row %s != %s", updated.ID, created.ID)
	}
	if string(updated.EncryptedSecret) != "ciphertext-2" {
		t.Fatalf("upsert did not replace ciphertext")
	}

	// MarkStatus flips status + validation_error in place, preserving the row id.
	if err := repo.MarkStatus(ctx, projectID, model.GitConnStatusInvalid, strPtr("unauthorized")); err != nil {
		t.Fatalf("MarkStatus: %v", err)
	}
	got, err := repo.GetByProject(ctx, projectID)
	if err != nil {
		t.Fatalf("GetByProject: %v", err)
	}
	if got.Status != model.GitConnStatusInvalid || got.ValidationError == nil || *got.ValidationError != "unauthorized" {
		t.Fatalf("MarkStatus did not apply: %+v", got)
	}

	// SetValidation full refresh.
	if err := repo.SetValidation(ctx, port.SetValidationParams{
		ProjectID: projectID, Status: model.GitConnStatusConnected,
		AccountLogin: &login, Scopes: []string{"read:project"}, ExpiresAt: nil, ValidationError: nil,
	}); err != nil {
		t.Fatalf("SetValidation: %v", err)
	}
	got, _ = repo.GetByProject(ctx, projectID)
	if got.Status != model.GitConnStatusConnected || got.ValidationError != nil {
		t.Fatalf("SetValidation did not apply: %+v", got)
	}

	// Delete is idempotent and reverts to not-found.
	if err := repo.Delete(ctx, projectID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := repo.Delete(ctx, projectID); err != nil {
		t.Fatalf("Delete should be idempotent: %v", err)
	}
	if _, err := repo.GetByProject(ctx, projectID); err == nil {
		t.Fatal("expected not-found after delete")
	}
}

func strPtr(s string) *string { return &s }
