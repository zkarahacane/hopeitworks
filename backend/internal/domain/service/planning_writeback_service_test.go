package service

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// ─── shared planning write-back / connector test mocks (package service) ─────────

// wbStoryRepo embeds the full StoryRepository mock and overrides only GetByID +
// SetWritebackStatus so write-back tests can assert the status transitions.
type wbStoryRepo struct {
	*mockStoryRepoForExecutor
	story       *model.Story
	getErr      error
	statusCalls []string
	setErr      error
}

func newWBStoryRepo(s *model.Story) *wbStoryRepo {
	return &wbStoryRepo{mockStoryRepoForExecutor: &mockStoryRepoForExecutor{}, story: s}
}

func (m *wbStoryRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.Story, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.story, nil
}

func (m *wbStoryRepo) SetWritebackStatus(_ context.Context, _ uuid.UUID, status string) error {
	m.statusCalls = append(m.statusCalls, status)
	return m.setErr
}

// wbConnectorRepo is a configurable PlanningConnectorRepository.
type wbConnectorRepo struct {
	conn      *model.PlanningConnector
	getErr    error
	upserted  *model.PlanningConnector
	upsertErr error
	upsertCnt int
}

func (m *wbConnectorRepo) Get(_ context.Context, _ uuid.UUID) (*model.PlanningConnector, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.conn, nil
}

func (m *wbConnectorRepo) Upsert(_ context.Context, c *model.PlanningConnector) (*model.PlanningConnector, error) {
	m.upsertCnt++
	if m.upsertErr != nil {
		return nil, m.upsertErr
	}
	m.upserted = c
	return c, nil
}

// wbWriteBackRepo captures audit rows.
type wbWriteBackRepo struct {
	rows []*model.PlanningWriteBack
}

func (m *wbWriteBackRepo) Create(_ context.Context, w *model.PlanningWriteBack) (*model.PlanningWriteBack, error) {
	m.rows = append(m.rows, w)
	return w, nil
}

func (m *wbWriteBackRepo) ListByStory(_ context.Context, _ uuid.UUID, _ int32) ([]*model.PlanningWriteBack, error) {
	return m.rows, nil
}

// wbSink captures write-back requests and returns a canned result/error.
type wbSink struct {
	reqs       []port.WriteBackRequest
	result     port.WriteBackResult
	err        error
	statusOpts port.PlanningStatusOptions
	statusErr  error
}

func (s *wbSink) StatusOptions(_ context.Context, _, _ string) (port.PlanningStatusOptions, error) {
	return s.statusOpts, s.statusErr
}

func (s *wbSink) WriteBack(_ context.Context, req port.WriteBackRequest) (port.WriteBackResult, error) {
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return port.WriteBackResult{}, s.err
	}
	return s.result, nil
}

// wbSinkFactory returns a fixed sink (or a resolution error).
type wbSinkFactory struct {
	sink port.PlanningSourceSink
	err  error
}

func (f *wbSinkFactory) Sink(_ context.Context, _ uuid.UUID) (port.PlanningSourceSink, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.sink, nil
}

// wbResolver is a GitCredentialResolver stub.
type wbResolver struct {
	token    string
	tokenErr error
}

func (r *wbResolver) TokenForProject(_ context.Context, _ uuid.UUID) (port.GitToken, error) {
	if r.tokenErr != nil {
		return port.GitToken{}, r.tokenErr
	}
	return port.GitToken{Value: r.token}, nil
}

func (r *wbResolver) ReconcileFromOperationError(_ context.Context, _ uuid.UUID, _ error) {}

func strptrT(s string) *string { return &s }

// githubStory builds a github_projects story with an item id for write-back tests.
func githubStory() *model.Story {
	return &model.Story{
		ID:             uuid.New(),
		ProjectID:      uuid.New(),
		Key:            "REPO-1",
		Source:         string(port.SourceGitHub),
		ExternalID:     strptrT("CONTENT_NODE"),
		ExternalItemID: strptrT("ITEM_ID"),
	}
}

// enabledConnector returns a write-back-enabled github connector mapping done->OPT_DONE.
func enabledConnector(projectID uuid.UUID) *model.PlanningConnector {
	return &model.PlanningConnector{
		ProjectID:        projectID,
		Source:           string(port.SourceGitHub),
		ProjectURL:       strptrT("https://github.com/orgs/acme/projects/7"),
		StatusField:      "Status",
		WritebackEnabled: true,
		StatusMapping:    model.PlanningStatusMapping{Done: strptrT("OPT_DONE")},
	}
}

func newWBService(story *wbStoryRepo, conn *wbConnectorRepo, audit *wbWriteBackRepo, sink port.PlanningSourceSink) *PlanningWriteBackService {
	return NewPlanningWriteBackService(conn, story, audit, &wbSinkFactory{sink: sink}, "", testLogger())
}

func TestWriteBack_NoConnector_NoOp(t *testing.T) {
	story := newWBStoryRepo(githubStory())
	conn := &wbConnectorRepo{getErr: notFound("planning_connector")}
	audit := &wbWriteBackRepo{}
	sink := &wbSink{}
	svc := newWBService(story, conn, audit, sink)

	if err := svc.SyncStatus(context.Background(), story.story.ID, nil, model.StoryStatusDone); err != nil {
		t.Fatalf("SyncStatus: %v", err)
	}
	if len(sink.reqs) != 0 {
		t.Fatalf("expected no write-back call, got %d", len(sink.reqs))
	}
	if len(audit.rows) != 0 {
		t.Fatalf("expected no audit row, got %d", len(audit.rows))
	}
	assertStatus(t, story, model.WritebackDisabled)
}

func TestWriteBack_Disabled_NoOp(t *testing.T) {
	s := githubStory()
	story := newWBStoryRepo(s)
	c := enabledConnector(s.ProjectID)
	c.WritebackEnabled = false
	conn := &wbConnectorRepo{conn: c}
	audit := &wbWriteBackRepo{}
	sink := &wbSink{}
	svc := newWBService(story, conn, audit, sink)

	if err := svc.SyncStatus(context.Background(), s.ID, nil, model.StoryStatusDone); err != nil {
		t.Fatalf("SyncStatus: %v", err)
	}
	if len(sink.reqs) != 0 {
		t.Fatalf("expected no write-back call")
	}
	assertStatus(t, story, model.WritebackDisabled)
}

func TestWriteBack_MappingAbsent_NoOp(t *testing.T) {
	s := githubStory()
	story := newWBStoryRepo(s)
	c := enabledConnector(s.ProjectID) // maps Done only
	conn := &wbConnectorRepo{conn: c}
	audit := &wbWriteBackRepo{}
	sink := &wbSink{}
	svc := newWBService(story, conn, audit, sink)

	// running has no mapping target => no-op, no error, status disabled.
	if err := svc.SyncStatus(context.Background(), s.ID, nil, model.StoryStatusRunning); err != nil {
		t.Fatalf("SyncStatus: %v", err)
	}
	if len(sink.reqs) != 0 {
		t.Fatalf("expected no write-back call for unmapped status")
	}
	if len(audit.rows) != 0 {
		t.Fatalf("expected no audit row for no-op")
	}
	assertStatus(t, story, model.WritebackDisabled)
}

func TestWriteBack_NonGithubStory_NoOp(t *testing.T) {
	s := githubStory()
	s.Source = string(port.SourceManual)
	story := newWBStoryRepo(s)
	conn := &wbConnectorRepo{conn: enabledConnector(s.ProjectID)}
	audit := &wbWriteBackRepo{}
	sink := &wbSink{}
	svc := newWBService(story, conn, audit, sink)

	if err := svc.SyncStatus(context.Background(), s.ID, nil, model.StoryStatusDone); err != nil {
		t.Fatalf("SyncStatus: %v", err)
	}
	if len(sink.reqs) != 0 {
		t.Fatalf("expected no write-back call for non-github story")
	}
	assertStatus(t, story, model.WritebackDisabled)
}

func TestWriteBack_Success_SyncedAndAudited(t *testing.T) {
	s := githubStory()
	story := newWBStoryRepo(s)
	c := enabledConnector(s.ProjectID)
	c.PostRunComment = true
	conn := &wbConnectorRepo{conn: c}
	audit := &wbWriteBackRepo{}
	sink := &wbSink{result: port.WriteBackResult{RemoteStatus: "Done", CommentPosted: true}}
	svc := newWBService(story, conn, audit, sink)

	runID := uuid.New()
	if err := svc.SyncStatus(context.Background(), s.ID, &runID, model.StoryStatusDone); err != nil {
		t.Fatalf("SyncStatus: %v", err)
	}
	if len(sink.reqs) != 1 {
		t.Fatalf("expected 1 write-back call, got %d", len(sink.reqs))
	}
	req := sink.reqs[0]
	if req.ItemID != "ITEM_ID" || req.ContentNodeID != "CONTENT_NODE" || req.OptionID != "OPT_DONE" {
		t.Fatalf("unexpected write-back request: %+v", req)
	}
	if req.Comment == "" {
		t.Fatalf("expected a comment when post_run_comment is on")
	}
	assertStatus(t, story, model.WritebackSynced)
	if len(audit.rows) != 1 || !audit.rows[0].Success {
		t.Fatalf("expected 1 successful audit row, got %+v", audit.rows)
	}
	if got := derefStr(audit.rows[0].RemoteStatus); got != "Done" {
		t.Fatalf("audit remote_status = %q, want Done", got)
	}
	if got := derefStr(audit.rows[0].InternalStatus); got != model.StoryStatusDone {
		t.Fatalf("audit internal_status = %q, want done", got)
	}
}

func TestWriteBack_DefinitiveFailure_FailedAndAudited(t *testing.T) {
	s := githubStory()
	story := newWBStoryRepo(s)
	conn := &wbConnectorRepo{conn: enabledConnector(s.ProjectID)}
	audit := &wbWriteBackRepo{}
	sink := &wbSink{err: stderrors.New("non-200 OK: 401 Unauthorized bad credentials")}
	svc := newWBService(story, conn, audit, sink)

	// definitive auth failure => recorded as failed, SyncStatus returns nil (no retry).
	if err := svc.SyncStatus(context.Background(), s.ID, nil, model.StoryStatusDone); err != nil {
		t.Fatalf("SyncStatus should swallow a definitive failure, got %v", err)
	}
	assertStatus(t, story, model.WritebackFailed)
	if len(audit.rows) != 1 || audit.rows[0].Success {
		t.Fatalf("expected 1 failed audit row, got %+v", audit.rows)
	}
	if got := derefStr(audit.rows[0].ErrorCode); got != "UNAUTHORIZED" {
		t.Fatalf("audit error_code = %q, want UNAUTHORIZED", got)
	}
}

func TestWriteBack_TransientFailure_ReturnsErrorForRetry(t *testing.T) {
	s := githubStory()
	story := newWBStoryRepo(s)
	conn := &wbConnectorRepo{conn: enabledConnector(s.ProjectID)}
	audit := &wbWriteBackRepo{}
	sink := &wbSink{err: stderrors.New("non-200 OK: 429 Too Many Requests rate limit")}
	svc := newWBService(story, conn, audit, sink)

	// transient => SyncStatus returns the error so River retries; status stays pending
	// (no SetWritebackStatus call), no failed audit row.
	if err := svc.SyncStatus(context.Background(), s.ID, nil, model.StoryStatusDone); err == nil {
		t.Fatalf("expected a transient error to be returned for retry")
	}
	if len(story.statusCalls) != 0 {
		t.Fatalf("transient failure must not change writeback_status, got %v", story.statusCalls)
	}
	if len(audit.rows) != 0 {
		t.Fatalf("transient failure must not write an audit row, got %d", len(audit.rows))
	}
}

func TestWriteBack_MissingItemID_FailedNoRemoteCall(t *testing.T) {
	s := githubStory()
	s.ExternalItemID = nil // never captured (pre-feature import)
	story := newWBStoryRepo(s)
	conn := &wbConnectorRepo{conn: enabledConnector(s.ProjectID)}
	audit := &wbWriteBackRepo{}
	sink := &wbSink{}
	svc := newWBService(story, conn, audit, sink)

	if err := svc.SyncStatus(context.Background(), s.ID, nil, model.StoryStatusDone); err != nil {
		t.Fatalf("SyncStatus: %v", err)
	}
	if len(sink.reqs) != 0 {
		t.Fatalf("expected no remote call without an item id")
	}
	assertStatus(t, story, model.WritebackFailed)
	if len(audit.rows) != 1 || derefStr(audit.rows[0].ErrorCode) != "MISSING_TARGET" {
		t.Fatalf("expected a MISSING_TARGET audit row, got %+v", audit.rows)
	}
}

// assertStatus checks the last writeback_status transition recorded on the story.
func assertStatus(t *testing.T, story *wbStoryRepo, want model.PlanningWritebackStatus) {
	t.Helper()
	if len(story.statusCalls) == 0 {
		t.Fatalf("expected a writeback_status transition, got none")
	}
	if got := story.statusCalls[len(story.statusCalls)-1]; got != string(want) {
		t.Fatalf("writeback_status = %q, want %q", got, want)
	}
}

// notFound builds a not-found DomainError the way the repos do (so isNotFound matches).
func notFound(resource string) error {
	return apperrors.NewNotFound(resource, uuid.New())
}

// ─── executor enqueue hook ──────────────────────────────────────────────────────

// fakeEnqueuer records the statuses the executor enqueues a write-back for.
type fakeEnqueuer struct {
	statuses []string
}

func (f *fakeEnqueuer) EnqueueWriteBack(_ context.Context, _, _, _ uuid.UUID, status string) error {
	f.statuses = append(f.statuses, status)
	return nil
}

func TestExecutor_EnqueueWriteBack_OnlyLifecycleTransitions(t *testing.T) {
	storyRepo := &wbStoryRepo{mockStoryRepoForExecutor: &mockStoryRepoForExecutor{}, story: githubStory()}
	enq := &fakeEnqueuer{}
	exec := NewPipelineExecutor(nil, storyRepo, nil, newMockEventPublisher(), testLogger())
	exec.SetWriteBackEnqueuer(enq)

	run := &model.Run{ID: uuid.New(), ProjectID: uuid.New(), StoryID: storyRepo.story.ID}

	for _, status := range []string{
		model.StoryStatusRunning,
		model.StoryStatusDone,
		model.StoryStatusFailed,
		model.StoryStatusBacklog, // must NOT be written back
	} {
		exec.updateStoryStatus(context.Background(), run, status)
	}

	want := []string{model.StoryStatusRunning, model.StoryStatusDone, model.StoryStatusFailed}
	if len(enq.statuses) != len(want) {
		t.Fatalf("enqueued statuses = %v, want %v", enq.statuses, want)
	}
	for i, s := range want {
		if enq.statuses[i] != s {
			t.Fatalf("enqueued[%d] = %q, want %q", i, enq.statuses[i], s)
		}
	}
	// backlog set pending? No: pending is only set for the three lifecycle transitions.
	for _, s := range storyRepo.statusCalls {
		if s != string(model.WritebackPending) {
			t.Fatalf("unexpected writeback_status transition %q (only pending expected at enqueue)", s)
		}
	}
	if len(storyRepo.statusCalls) != 3 {
		t.Fatalf("expected 3 pending transitions, got %d", len(storyRepo.statusCalls))
	}
}
