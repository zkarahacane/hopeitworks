package service

import (
	"context"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// --- fakes -----------------------------------------------------------------

// fakeAdapter returns a canned FetchResult so the SERVICE logic can be tested
// directly without depending on the markdown parser / a live source.
type fakeAdapter struct {
	kind port.SourceKind
	res  *port.FetchResult
	err  error
}

func (a *fakeAdapter) Kind() port.SourceKind { return a.kind }
func (a *fakeAdapter) Fetch(_ context.Context, _ uuid.UUID, _ port.ImportConfig) (*port.FetchResult, error) {
	return a.res, a.err
}

type fakeFactory struct {
	adapter port.PlanningSourceAdapter
	err     error
}

func (f *fakeFactory) For(_ context.Context, _ uuid.UUID, _ port.SourceKind) (port.PlanningSourceAdapter, error) {
	return f.adapter, f.err
}

func newImportSvc(stories port.StoryRepository, epics port.EpicRepository, source port.SourceKind, res *port.FetchResult) *PlanningImportService {
	return NewPlanningImportService(stories, epics, &fakeFactory{adapter: &fakeAdapter{kind: source, res: res}})
}

func mdConfig() port.ImportConfig {
	return port.ImportConfig{Source: port.SourceMarkdown, Markdown: &port.MarkdownConfig{Content: "x"}}
}

func ghConfig(done ...string) port.ImportConfig {
	return port.ImportConfig{Source: port.SourceGitHub, GitHubProjects: &port.GitHubProjectsConfig{ProjectURL: "u", DoneOptions: done}}
}

func mdStory(key, title, rawStatus string) port.ImportedStory {
	return port.ImportedStory{
		Ref:       port.SourceRef{Source: port.SourceMarkdown, ExternalID: key},
		Key:       key,
		Title:     title,
		RawStatus: rawStatus,
	}
}

// --- status projection (explicit only) -------------------------------------

func TestPlanningImport_StatusProjection_ExplicitOnly(t *testing.T) {
	tests := []struct {
		name   string
		cfg    port.ImportConfig
		raw    string
		expect string
	}{
		{"markdown done literal", mdConfig(), "done", model.StoryStatusBacklog + "->done"},
		{"markdown DONE uppercase", mdConfig(), "DONE", "done"},
		{"markdown in_progress -> backlog", mdConfig(), "in_progress", "backlog"},
		{"markdown running -> backlog", mdConfig(), "running", "backlog"},
		{"markdown closed -> backlog (no broad allowlist)", mdConfig(), "closed", "backlog"},
		{"markdown empty -> backlog", mdConfig(), "", "backlog"},
		{"github no done options -> backlog", ghConfig(), "Done", "backlog"},
		{"github Done in options -> done", ghConfig("Done"), "Done", "done"},
		{"github case-insensitive match", ghConfig("done"), "DONE", "done"},
		{"github In Progress -> backlog", ghConfig("Done"), "In Progress", "backlog"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := projectStoryStatus(tt.cfg, tt.raw)
			want := tt.expect
			if want == "backlog->done" {
				want = "done"
			}
			if got != want {
				t.Errorf("projectStoryStatus(%q) = %q, want %q", tt.raw, got, want)
			}
		})
	}

	// running/failed are NEVER produced by import.
	for _, raw := range []string{"running", "failed", "error", "wip"} {
		if s := projectStoryStatus(mdConfig(), raw); s != model.StoryStatusBacklog {
			t.Errorf("markdown %q projected to %q; import must never emit running/failed", raw, s)
		}
	}
}

// --- set-once epic_id -------------------------------------------------------

func TestPlanningImport_SetOnceEpicID(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()
	stories := newMockStoryRepo()
	epics := newMockEpicRepo()

	epicA := uuid.New()
	storyID := uuid.New()
	stories.stories[storyID] = &model.Story{
		ID: storyID, ProjectID: projectID, Key: "S-1", Title: "old",
		Status: model.StoryStatusBacklog, Source: string(port.SourceMarkdown),
		ExternalID: storyStrPtr("S-1"), EpicID: &epicA,
	}

	// Re-import the same story but linking a DIFFERENT epic E-2, with a new title
	// (so it lands in the update path rather than the hash no-op).
	res := &port.FetchResult{
		Epics:   []port.ImportedEpic{{Ref: port.SourceRef{Source: port.SourceMarkdown, ExternalID: "E-2"}, Key: "E-2", Name: "E-2"}},
		Stories: []port.ImportedStory{{Ref: port.SourceRef{Source: port.SourceMarkdown, ExternalID: "S-1"}, Key: "S-1", Title: "new", EpicRef: &port.SourceRef{Source: port.SourceMarkdown, ExternalID: "E-2"}}},
	}
	svc := newImportSvc(stories, epics, port.SourceMarkdown, res)
	if _, err := svc.Import(ctx, projectID, mdConfig()); err != nil {
		t.Fatalf("Import: %v", err)
	}

	got := stories.stories[storyID]
	if got.EpicID == nil || *got.EpicID != epicA {
		t.Errorf("epic_id must be set-once (stay %v), got %v", epicA, got.EpicID)
	}
}

// --- epic adoption (no NAME_CONFLICT abort) ---------------------------------

func TestPlanningImport_EpicAdoption(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()
	stories := newMockStoryRepo()
	epics := newMockEpicRepo()

	manualEpicID := uuid.New()
	epics.epics[manualEpicID] = &model.Epic{
		ID: manualEpicID, ProjectID: projectID, Name: "Auth",
		Status: model.EpicStatusBacklog, Source: string(port.SourceManual),
	}

	res := &port.FetchResult{
		Epics:   []port.ImportedEpic{{Ref: port.SourceRef{Source: port.SourceMarkdown, ExternalID: "Auth"}, Key: "Auth", Name: "Auth"}},
		Stories: []port.ImportedStory{{Ref: port.SourceRef{Source: port.SourceMarkdown, ExternalID: "S-1"}, Key: "S-1", Title: "Login", EpicRef: &port.SourceRef{Source: port.SourceMarkdown, ExternalID: "Auth"}}},
	}
	svc := newImportSvc(stories, epics, port.SourceMarkdown, res)
	sum, err := svc.Import(ctx, projectID, mdConfig())
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if sum.Failed != 0 {
		t.Fatalf("adoption must not abort: failed=%d errors=%v", sum.Failed, sum.Errors)
	}
	adopted := epics.epics[manualEpicID]
	if adopted.Source != string(port.SourceMarkdown) {
		t.Errorf("manual epic should be adopted (source=markdown), got %q", adopted.Source)
	}
	if adopted.ExternalID == nil || *adopted.ExternalID != "Auth" {
		t.Errorf("adopted epic should be stamped external_id=Auth, got %v", adopted.ExternalID)
	}
	// The story must link to the adopted (same) epic.
	for _, s := range stories.stories {
		if s.Key == "S-1" && (s.EpicID == nil || *s.EpicID != manualEpicID) {
			t.Errorf("story should link to adopted epic %v, got %v", manualEpicID, s.EpicID)
		}
	}
}

// --- epic link-only anti-flip-flop ------------------------------------------

func TestPlanningImport_EpicLinkOnly_AntiFlipFlop(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()
	stories := newMockStoryRepo()
	epics := newMockEpicRepo()

	// An epic owned by a DIFFERENT remote source shares the name.
	remoteEpicID := uuid.New()
	epics.epics[remoteEpicID] = &model.Epic{
		ID: remoteEpicID, ProjectID: projectID, Name: "Auth",
		Status: model.EpicStatusBacklog, Source: string(port.SourceGitHub),
		ExternalID: storyStrPtr("ghnode"),
	}

	res := &port.FetchResult{
		Epics:   []port.ImportedEpic{{Ref: port.SourceRef{Source: port.SourceMarkdown, ExternalID: "Auth"}, Key: "Auth", Name: "Auth"}},
		Stories: []port.ImportedStory{{Ref: port.SourceRef{Source: port.SourceMarkdown, ExternalID: "S-1"}, Key: "S-1", Title: "Login", EpicRef: &port.SourceRef{Source: port.SourceMarkdown, ExternalID: "Auth"}}},
	}
	svc := newImportSvc(stories, epics, port.SourceMarkdown, res)
	sum, err := svc.Import(ctx, projectID, mdConfig())
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	// Provenance must NOT flip to markdown (anti flip-flop) — link only.
	e := epics.epics[remoteEpicID]
	if e.Source != string(port.SourceGitHub) {
		t.Errorf("link-only: epic source must stay github_projects, got %q", e.Source)
	}
	if e.ExternalID == nil || *e.ExternalID != "ghnode" {
		t.Errorf("link-only: external_id must be untouched, got %v", e.ExternalID)
	}
	if sum.EpicsUpdated != 0 {
		t.Errorf("link-only must not count as EpicsUpdated, got %d", sum.EpicsUpdated)
	}
	if len(sum.Warnings) == 0 {
		t.Errorf("expected a NAME_CONFLICT warning on cross-source name share")
	}
	// Children still attach to the existing epic.
	for _, s := range stories.stories {
		if s.Key == "S-1" && (s.EpicID == nil || *s.EpicID != remoteEpicID) {
			t.Errorf("story should link to existing epic %v, got %v", remoteEpicID, s.EpicID)
		}
	}
}

// --- lock while running -----------------------------------------------------

func TestPlanningImport_LockWhileRunning(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()
	stories := newMockStoryRepo()
	epics := newMockEpicRepo()

	storyID := uuid.New()
	stories.stories[storyID] = &model.Story{
		ID: storyID, ProjectID: projectID, Key: "S-1", Title: "old",
		Status: model.StoryStatusRunning, Objective: storyStrPtr("old objective"),
		Source: string(port.SourceMarkdown), ExternalID: storyStrPtr("S-1"),
	}

	res := &port.FetchResult{Stories: []port.ImportedStory{{
		Ref: port.SourceRef{Source: port.SourceMarkdown, ExternalID: "S-1"}, Key: "S-1",
		Title: "new title", Objective: storyStrPtr("new objective"), RawStatus: "done",
	}}}
	svc := newImportSvc(stories, epics, port.SourceMarkdown, res)
	sum, err := svc.Import(ctx, projectID, mdConfig())
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	got := stories.stories[storyID]
	if sum.Locked != 1 {
		t.Errorf("expected Locked=1, got %d", sum.Locked)
	}
	if got.Status != model.StoryStatusRunning {
		t.Errorf("locked story status must stay running, got %q", got.Status)
	}
	if got.Objective == nil || *got.Objective != "old objective" {
		t.Errorf("locked story spec (objective) must be frozen, got %v", got.Objective)
	}
	if got.LastImportHash != nil {
		t.Errorf("locked import must NOT advance last_import_hash, got %v", got.LastImportHash)
	}
	if sum.StoriesUpdated != 0 {
		t.Errorf("locked row must not count as StoriesUpdated, got %d", sum.StoriesUpdated)
	}
}

// --- hash no-op idempotency -------------------------------------------------

func TestPlanningImport_HashNoOp(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()
	stories := newMockStoryRepo()
	epics := newMockEpicRepo()

	res := &port.FetchResult{Stories: []port.ImportedStory{mdStory("S-1", "Title", "")}}
	svc := newImportSvc(stories, epics, port.SourceMarkdown, res)

	first, err := svc.Import(ctx, projectID, mdConfig())
	if err != nil {
		t.Fatalf("Import 1: %v", err)
	}
	if first.StoriesCreated != 1 {
		t.Fatalf("expected 1 created, got %d", first.StoriesCreated)
	}

	second, err := svc.Import(ctx, projectID, mdConfig())
	if err != nil {
		t.Fatalf("Import 2: %v", err)
	}
	if second.Skipped != 1 {
		t.Errorf("expected unchanged re-import Skipped=1, got %d", second.Skipped)
	}
	if second.StoriesCreated != 0 || second.StoriesUpdated != 0 {
		t.Errorf("unchanged re-import must be a no-op, got created=%d updated=%d", second.StoriesCreated, second.StoriesUpdated)
	}
	if len(stories.stories) != 1 {
		t.Errorf("re-import must not duplicate, got %d stories", len(stories.stories))
	}
}

// --- preserve on absent -----------------------------------------------------

func TestPlanningImport_PreserveOnAbsent(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()
	stories := newMockStoryRepo()
	epics := newMockEpicRepo()

	storyID := uuid.New()
	stories.stories[storyID] = &model.Story{
		ID: storyID, ProjectID: projectID, Key: "S-1", Title: "old",
		Status:             model.StoryStatusBacklog,
		Objective:          storyStrPtr("keep objective"),
		AcceptanceCriteria: storyStrPtr("keep ac"),
		Scope:              storyStrPtr("backend"),
		DependsOn:          []string{"DEP-1"},
		Source:             string(port.SourceMarkdown), ExternalID: storyStrPtr("S-1"),
	}

	// Incoming carries only a new title; every other field is absent (nil).
	res := &port.FetchResult{Stories: []port.ImportedStory{{
		Ref: port.SourceRef{Source: port.SourceMarkdown, ExternalID: "S-1"}, Key: "S-1", Title: "new title",
	}}}
	svc := newImportSvc(stories, epics, port.SourceMarkdown, res)
	if _, err := svc.Import(ctx, projectID, mdConfig()); err != nil {
		t.Fatalf("Import: %v", err)
	}

	got := stories.stories[storyID]
	if got.Title != "new title" {
		t.Errorf("title should update, got %q", got.Title)
	}
	if got.Objective == nil || *got.Objective != "keep objective" {
		t.Errorf("objective must be preserved, got %v", got.Objective)
	}
	if got.AcceptanceCriteria == nil || *got.AcceptanceCriteria != "keep ac" {
		t.Errorf("acceptance_criteria must be preserved, got %v", got.AcceptanceCriteria)
	}
	if got.Scope == nil || *got.Scope != "backend" {
		t.Errorf("scope must be preserved, got %v", got.Scope)
	}
	if len(got.DependsOn) != 1 || got.DependsOn[0] != "DEP-1" {
		t.Errorf("depends_on must be preserved, got %v", got.DependsOn)
	}
}

// --- deterministic key collision fallback (github_projects) -----------------

func TestPlanningImport_DeterministicKeyFallback(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()
	stories := newMockStoryRepo()
	epics := newMockEpicRepo()

	// A manual story already owns the natural derived key.
	manualID := uuid.New()
	stories.stories[manualID] = &model.Story{
		ID: manualID, ProjectID: projectID, Key: "GH-1", Title: "manual",
		Status: model.StoryStatusBacklog, Source: string(port.SourceManual),
	}

	ghItem := port.ImportedStory{
		Ref:   port.SourceRef{Source: port.SourceGitHub, ExternalID: "node-1", URL: "https://gh/1"},
		Key:   "GH-1",
		Title: "remote",
	}
	res := &port.FetchResult{Stories: []port.ImportedStory{ghItem}}
	svc := newImportSvc(stories, epics, port.SourceGitHub, res)

	sum, err := svc.Import(ctx, projectID, ghConfig())
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if sum.StoriesCreated != 1 || sum.Failed != 0 {
		t.Fatalf("expected 1 created 0 failed (deterministic fallback), got created=%d failed=%d errs=%v", sum.StoriesCreated, sum.Failed, sum.Errors)
	}

	// Find the created github story and capture its disambiguated key.
	keyRe := regexp.MustCompile(`^[A-Z0-9]+-\d+$`)
	var ghKey string
	for _, s := range stories.stories {
		if s.Source == string(port.SourceGitHub) {
			ghKey = s.Key
		}
	}
	if ghKey == "" || ghKey == "GH-1" {
		t.Fatalf("expected a disambiguated key != GH-1, got %q", ghKey)
	}
	if !keyRe.MatchString(ghKey) {
		t.Errorf("disambiguated key %q must match ^[A-Z0-9]+-\\d+$", ghKey)
	}

	// Re-import resolves by source ref and keeps the SAME key (stable, set-once).
	sum2, err := svc.Import(ctx, projectID, ghConfig())
	if err != nil {
		t.Fatalf("re-import: %v", err)
	}
	if sum2.StoriesCreated != 0 {
		t.Errorf("re-import must not create a second row, got %d", sum2.StoriesCreated)
	}
	var ghKey2 string
	for _, s := range stories.stories {
		if s.Source == string(port.SourceGitHub) {
			ghKey2 = s.Key
		}
	}
	if ghKey2 != ghKey {
		t.Errorf("derived key must be stable across re-import: %q -> %q", ghKey, ghKey2)
	}
}

// --- markdown self-heal of a backfilled manual row by key -------------------

func TestPlanningImport_MarkdownSelfHealManualRow(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()
	stories := newMockStoryRepo()
	epics := newMockEpicRepo()

	// A legacy markdown-imported row backfilled to source='manual' by migration.
	manualID := uuid.New()
	stories.stories[manualID] = &model.Story{
		ID: manualID, ProjectID: projectID, Key: "TODO-1", Title: "old",
		Status: model.StoryStatusBacklog, Source: string(port.SourceManual),
	}

	res := &port.FetchResult{Stories: []port.ImportedStory{mdStory("TODO-1", "Updated title", "")}}
	svc := newImportSvc(stories, epics, port.SourceMarkdown, res)
	sum, err := svc.Import(ctx, projectID, mdConfig())
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if sum.StoriesUpdated != 1 {
		t.Errorf("self-heal should update in place, got updated=%d created=%d", sum.StoriesUpdated, sum.StoriesCreated)
	}
	if len(stories.stories) != 1 {
		t.Errorf("self-heal must not duplicate the row, got %d stories", len(stories.stories))
	}
	healed := stories.stories[manualID]
	if healed.Source != string(port.SourceMarkdown) {
		t.Errorf("row should self-heal to source=markdown, got %q", healed.Source)
	}
	if healed.ExternalID == nil || *healed.ExternalID != "TODO-1" {
		t.Errorf("row should be stamped external_id=TODO-1, got %v", healed.ExternalID)
	}
	if healed.LastImportHash == nil {
		t.Errorf("self-heal should stamp last_import_hash")
	}
}

// --- epic idempotency (§16.14b) ---------------------------------------------

func TestPlanningImport_EpicIdempotency(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()
	stories := newMockStoryRepo()
	epics := newMockEpicRepo()

	res := &port.FetchResult{Epics: []port.ImportedEpic{{Ref: port.SourceRef{Source: port.SourceMarkdown, ExternalID: "E-1"}, Key: "E-1", Name: "E-1"}}}
	svc := newImportSvc(stories, epics, port.SourceMarkdown, res)

	first, err := svc.Import(ctx, projectID, mdConfig())
	if err != nil {
		t.Fatalf("Import 1: %v", err)
	}
	if first.EpicsCreated != 1 {
		t.Fatalf("expected 1 epic created, got %d", first.EpicsCreated)
	}

	second, err := svc.Import(ctx, projectID, mdConfig())
	if err != nil {
		t.Fatalf("Import 2: %v", err)
	}
	if second.EpicsCreated != 0 || second.EpicsUpdated != 0 {
		t.Errorf("unchanged epic re-import must be a no-op, got created=%d updated=%d", second.EpicsCreated, second.EpicsUpdated)
	}
	if len(epics.epics) != 1 {
		t.Errorf("epic re-import must not duplicate, got %d epics", len(epics.epics))
	}
}

// --- empty board is a valid 200 zero-import ---------------------------------

func TestPlanningImport_EmptyBoardIsZeroImport(t *testing.T) {
	ctx := context.Background()
	svc := newImportSvc(newMockStoryRepo(), newMockEpicRepo(), port.SourceMarkdown, &port.FetchResult{})
	sum, err := svc.Import(ctx, uuid.New(), mdConfig())
	if err != nil {
		t.Fatalf("empty board must be a valid 200, got error: %v", err)
	}
	if sum.StoriesCreated != 0 || sum.EpicsCreated != 0 || sum.Failed != 0 {
		t.Errorf("expected all-zero counts, got %+v", sum)
	}
}

// --- epic promote-only: in_progress frozen, backlog→done promoted (§16.4) ----

func TestPlanningImport_EpicPromoteOnly(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()

	t.Run("in_progress not touched by import even when raw maps done", func(t *testing.T) {
		stories := newMockStoryRepo()
		epics := newMockEpicRepo()

		epicID := uuid.New()
		epics.epics[epicID] = &model.Epic{
			ID: epicID, ProjectID: projectID, Name: "E-1",
			Status: model.EpicStatusInProgress, Source: string(port.SourceMarkdown),
			ExternalID: storyStrPtr("E-1"),
		}

		// Raw "done" would project to done, but in_progress must be frozen.
		res := &port.FetchResult{Epics: []port.ImportedEpic{{
			Ref: port.SourceRef{Source: port.SourceMarkdown, ExternalID: "E-1"},
			Key: "E-1", Name: "E-1", RawStatus: "done",
		}}}
		svc := newImportSvc(stories, epics, port.SourceMarkdown, res)
		sum, err := svc.Import(ctx, projectID, mdConfig())
		if err != nil {
			t.Fatalf("Import: %v", err)
		}
		if sum.Failed != 0 {
			t.Fatalf("expected no failures, got %d: %v", sum.Failed, sum.Errors)
		}
		got := epics.epics[epicID]
		if got.Status != model.EpicStatusInProgress {
			t.Errorf("in_progress epic must not be touched by import, got %q", got.Status)
		}
		// The epic must not have been counted as updated either.
		if sum.EpicsUpdated != 0 {
			t.Errorf("in_progress epic must not count as EpicsUpdated, got %d", sum.EpicsUpdated)
		}
	})

	t.Run("backlog promoted to done when raw maps done", func(t *testing.T) {
		stories := newMockStoryRepo()
		epics := newMockEpicRepo()

		epicID := uuid.New()
		epics.epics[epicID] = &model.Epic{
			ID: epicID, ProjectID: projectID, Name: "E-2",
			Status: model.EpicStatusBacklog, Source: string(port.SourceMarkdown),
			ExternalID: storyStrPtr("E-2"),
		}

		res := &port.FetchResult{Epics: []port.ImportedEpic{{
			Ref: port.SourceRef{Source: port.SourceMarkdown, ExternalID: "E-2"},
			Key: "E-2", Name: "E-2", RawStatus: "done",
		}}}
		svc := newImportSvc(stories, epics, port.SourceMarkdown, res)
		sum, err := svc.Import(ctx, projectID, mdConfig())
		if err != nil {
			t.Fatalf("Import: %v", err)
		}
		if sum.Failed != 0 {
			t.Fatalf("expected no failures, got %d: %v", sum.Failed, sum.Errors)
		}
		got := epics.epics[epicID]
		if got.Status != model.EpicStatusDone {
			t.Errorf("backlog epic with done raw status must be promoted to done, got %q", got.Status)
		}
		if sum.EpicsUpdated != 1 {
			t.Errorf("expected EpicsUpdated=1, got %d", sum.EpicsUpdated)
		}
	})
}

// --- markdown KEY_CONFLICT: key owned by github_projects (§16.1) --------------

func TestPlanningImport_MarkdownKeyConflict(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()
	stories := newMockStoryRepo()
	epics := newMockEpicRepo()

	// A story owned by github_projects already holds key "S-1".
	ghID := uuid.New()
	stories.stories[ghID] = &model.Story{
		ID: ghID, ProjectID: projectID, Key: "S-1", Title: "github story",
		Status: model.StoryStatusBacklog, Source: string(port.SourceGitHub),
		ExternalID: storyStrPtr("node-gh-1"),
	}

	// Markdown import sends a story with the same key.
	res := &port.FetchResult{Stories: []port.ImportedStory{mdStory("S-1", "markdown story", "")}}
	svc := newImportSvc(stories, epics, port.SourceMarkdown, res)
	sum, err := svc.Import(ctx, projectID, mdConfig())
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	if sum.Failed != 1 {
		t.Errorf("expected Failed=1, got %d", sum.Failed)
	}
	if len(sum.Errors) == 0 || sum.Errors[0].Code != "KEY_CONFLICT" {
		t.Errorf("expected KEY_CONFLICT error, got %v", sum.Errors)
	}
	// The github story must NOT be overwritten.
	got := stories.stories[ghID]
	if got.Title != "github story" {
		t.Errorf("github story must not be overwritten, got title %q", got.Title)
	}
	if got.Source != string(port.SourceGitHub) {
		t.Errorf("source must stay %s, got %q", port.SourceGitHub, got.Source)
	}
	if sum.StoriesCreated != 0 || sum.StoriesUpdated != 0 {
		t.Errorf("no story must be created or updated on KEY_CONFLICT, got created=%d updated=%d", sum.StoriesCreated, sum.StoriesUpdated)
	}
}

// --- epic same source, external_id different: NAME_CONFLICT warning (§16.2) ---

func TestPlanningImport_EpicSameSourceDifferentExternalID(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()
	stories := newMockStoryRepo()
	epics := newMockEpicRepo()

	// An existing github_projects epic holds name "Auth" with external_id "node-1".
	existingEpicID := uuid.New()
	epics.epics[existingEpicID] = &model.Epic{
		ID: existingEpicID, ProjectID: projectID, Name: "Auth",
		Status: model.EpicStatusBacklog, Source: string(port.SourceGitHub),
		ExternalID: storyStrPtr("node-1"),
	}

	// A second github_projects import brings an epic with the SAME name but a
	// DIFFERENT external_id ("node-2"), plus a child story linked to it.
	res := &port.FetchResult{
		Epics: []port.ImportedEpic{{
			Ref: port.SourceRef{Source: port.SourceGitHub, ExternalID: "node-2"},
			Key: "GH-2", Name: "Auth",
		}},
		Stories: []port.ImportedStory{{
			Ref:     port.SourceRef{Source: port.SourceGitHub, ExternalID: "S-child"},
			Key:     "GH-1",
			Title:   "child story",
			EpicRef: &port.SourceRef{Source: port.SourceGitHub, ExternalID: "node-2"},
		}},
	}
	svc := newImportSvc(stories, epics, port.SourceGitHub, res)
	sum, err := svc.Import(ctx, projectID, ghConfig())
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	// Must produce a NAME_CONFLICT warning (not a failure).
	var hasNameConflict bool
	for _, w := range sum.Warnings {
		if w.Code == "NAME_CONFLICT" {
			hasNameConflict = true
		}
	}
	if !hasNameConflict {
		t.Errorf("expected NAME_CONFLICT warning, got warnings=%v", sum.Warnings)
	}
	if sum.Failed != 0 {
		t.Errorf("same-source name collision must not fail the batch: failed=%d errors=%v", sum.Failed, sum.Errors)
	}

	// Provenance of the existing epic must NOT be overwritten.
	got := epics.epics[existingEpicID]
	if got.Source != string(port.SourceGitHub) {
		t.Errorf("source must stay %s, got %q", port.SourceGitHub, got.Source)
	}
	if got.ExternalID == nil || *got.ExternalID != "node-1" {
		t.Errorf("external_id must stay node-1, got %v", got.ExternalID)
	}

	// The child story must link to the existing (not a phantom new) epic.
	for _, s := range stories.stories {
		if s.Key == "GH-1" && (s.EpicID == nil || *s.EpicID != existingEpicID) {
			t.Errorf("child story must link to existing epic %v, got %v", existingEpicID, s.EpicID)
		}
	}
}

// --- story done (by execution), current_stage nil → not locked (§16.9) --------

func TestPlanningImport_DoneStoryNotLocked(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()
	stories := newMockStoryRepo()
	epics := newMockEpicRepo()

	oldHash := "old-hash-value"
	storyID := uuid.New()
	stories.stories[storyID] = &model.Story{
		ID: storyID, ProjectID: projectID, Key: "S-1", Title: "old title",
		Status: model.StoryStatusDone, CurrentStage: nil,
		Source: string(port.SourceMarkdown), ExternalID: storyStrPtr("S-1"),
		LastImportHash: &oldHash,
	}

	// Upstream changes the title; raw status does not map to done (falls back to backlog).
	res := &port.FetchResult{Stories: []port.ImportedStory{{
		Ref: port.SourceRef{Source: port.SourceMarkdown, ExternalID: "S-1"}, Key: "S-1",
		Title: "new title", RawStatus: "backlog",
	}}}
	svc := newImportSvc(stories, epics, port.SourceMarkdown, res)
	sum, err := svc.Import(ctx, projectID, mdConfig())
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	// done+nil-stage is NOT locked: spec changes must be applied.
	if sum.Locked != 0 {
		t.Errorf("done+nil-stage story must not be locked, got Locked=%d", sum.Locked)
	}
	if sum.StoriesUpdated != 1 {
		t.Errorf("expected StoriesUpdated=1 (title changed), got %d", sum.StoriesUpdated)
	}

	got := stories.stories[storyID]
	if got.Title != "new title" {
		t.Errorf("title must be updated, got %q", got.Title)
	}
	// Status must stay done — never downgraded by import.
	if got.Status != model.StoryStatusDone {
		t.Errorf("done status must never be downgraded by import, got %q", got.Status)
	}
}

// --- current_stage != nil blocks backlog→done promotion (§16.0) ---------------

func TestPlanningImport_InStageBlocksPromotion(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()
	stories := newMockStoryRepo()
	epics := newMockEpicRepo()

	stage := "review"
	storyID := uuid.New()
	stories.stories[storyID] = &model.Story{
		ID: storyID, ProjectID: projectID, Key: "S-1", Title: "in-stage story",
		Status: model.StoryStatusBacklog, CurrentStage: &stage,
		Source: string(port.SourceMarkdown), ExternalID: storyStrPtr("S-1"),
	}

	// Raw "done" would normally promote backlog→done, but current_stage != nil must block it.
	res := &port.FetchResult{Stories: []port.ImportedStory{{
		Ref: port.SourceRef{Source: port.SourceMarkdown, ExternalID: "S-1"}, Key: "S-1",
		Title: "in-stage story", RawStatus: "done",
	}}}
	svc := newImportSvc(stories, epics, port.SourceMarkdown, res)
	sum, err := svc.Import(ctx, projectID, mdConfig())
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	if sum.Locked != 1 {
		t.Errorf("expected Locked=1 for in-stage story, got %d", sum.Locked)
	}
	got := stories.stories[storyID]
	if got.Status != model.StoryStatusBacklog {
		t.Errorf("backlog+in-stage must not be promoted to done, got %q", got.Status)
	}
	if sum.StoriesUpdated != 0 {
		t.Errorf("locked story must not count as StoriesUpdated, got %d", sum.StoriesUpdated)
	}
}

// --- source error -> 422 (InvalidState) -------------------------------------

func TestPlanningImport_SourceErrorMapsTo422(t *testing.T) {
	ctx := context.Background()
	svc := NewPlanningImportService(newMockStoryRepo(), newMockEpicRepo(),
		&fakeFactory{err: errors.NewInternal("github_projects adapter not implemented", nil)})
	_, err := svc.Import(ctx, uuid.New(), ghConfig())
	if err == nil {
		t.Fatal("expected an error when the source factory fails")
	}
	de, ok := err.(*errors.DomainError)
	if !ok || de.Category != errors.CategoryInvalidState || de.Code != "SOURCE_ERROR" {
		t.Errorf("expected SOURCE_ERROR/invalid_state (HTTP 422), got %+v", err)
	}
}
