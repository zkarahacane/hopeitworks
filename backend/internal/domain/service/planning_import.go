package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// PlanningImportService imports an external plan (markdown, github_projects) into
// the internal epic->story model behind port.PlanningSourceAdapter. It owns EVERY
// business decision — status projection, identity resolution, source-guarded epic
// adoption, lock-while-running, hash no-op, preserve-on-absent, set-once epic_id —
// so adapters stay pure-read and adding a source needs no service change.
//
// There is NO outer transaction (the codebase has no Transactor used by import —
// only an unused Queries.WithTx). Each item is its own statement: a per-item
// failure is collected and the batch continues (partial success), mirroring the
// legacy markdown importer this replaces.
type PlanningImportService struct {
	stories port.StoryRepository
	epics   port.EpicRepository
	factory port.PlanningSourceFactory
}

// NewPlanningImportService wires the importer with both repos + the source factory.
func NewPlanningImportService(stories port.StoryRepository, epics port.EpicRepository, factory port.PlanningSourceFactory) *PlanningImportService {
	return &PlanningImportService{stories: stories, epics: epics, factory: factory}
}

// item decision actions / kinds (mirror the openapi enums).
const (
	actionCreate = "create"
	actionUpdate = "update"
	actionSkip   = "skip"
	actionLock   = "lock"
	actionFail   = "fail"

	kindEpic  = "epic"
	kindStory = "story"
)

// item error codes (mirror port.ImportItemError.Code enum).
const (
	codeKeyFormat    = "KEY_FORMAT"
	codeKeyConflict  = "KEY_CONFLICT"
	codeNameConflict = "NAME_CONFLICT"
	codeParseError   = "PARSE_ERROR"
	codeUpsertError  = "UPSERT_ERROR"
)

// Import fetches the external plan, then upserts epics (first, so stories can link)
// and stories. A successfully-fetched-but-empty board is a valid 200 zero-import;
// only an unreachable / unusable source surfaces as SOURCE_ERROR (HTTP 422).
func (s *PlanningImportService) Import(ctx context.Context, projectID uuid.UUID, cfg port.ImportConfig) (*port.ImportSummary, error) {
	adapter, err := s.factory.For(ctx, projectID, cfg.Source)
	if err != nil {
		return nil, sourceError(err)
	}

	res, err := adapter.Fetch(ctx, projectID, cfg)
	if err != nil {
		return nil, sourceError(err)
	}

	summary := &port.ImportSummary{
		Source:    cfg.Source,
		DryRun:    cfg.DryRun,
		SourceURL: res.SourceURL,
		Errors:    []port.ImportItemError{},
		Warnings:  append([]port.ImportWarning{}, res.Warnings...),
		Items:     []port.ImportItemDecision{},
	}

	// Epics first: build a map from epic external_id -> resolved DB id so stories
	// can attach their parent. A failed/skipped epic simply leaves no entry (its
	// children import as orphans).
	epicID := make(map[string]uuid.UUID)
	for i := range res.Epics {
		s.importEpic(ctx, projectID, cfg, res.Epics[i], epicID, summary)
	}

	for i := range res.Stories {
		s.importStory(ctx, projectID, cfg, res.Stories[i], epicID, summary)
	}

	return summary, nil
}

// importEpic applies the source-guarded adoption + idempotency + promote-only rules.
func (s *PlanningImportService) importEpic(ctx context.Context, projectID uuid.UUID, cfg port.ImportConfig, ie port.ImportedEpic, epicID map[string]uuid.UUID, summary *port.ImportSummary) {
	source := string(cfg.Source)
	extID := ie.Ref.ExternalID
	mapped := projectEpicStatus(cfg, ie.RawStatus)
	url := ie.Ref.URL

	// 1) Our own identity?
	existing, err := s.epics.GetBySourceRef(ctx, projectID, source, extID)
	if err != nil && !isNotFound(err) {
		s.failEpic(summary, ie, codeUpsertError, fmt.Sprintf("resolve epic by source ref: %v", err))
		return
	}
	if existing != nil {
		descMerged := mergeStrPtr(ie.Description, existing.Description)
		status := resolveEpicStatus(existing, mapped)
		if !epicChanged(existing, ie.Name, descMerged, status, source, extID, url) {
			epicID[extID] = existing.ID
			s.decide(summary, ie.Key, kindEpic, actionSkip, url, status, "unchanged")
			return
		}
		s.writeEpicUpdate(ctx, cfg, existing, ie.Name, descMerged, status, source, extID, url, epicID, summary)
		return
	}

	// 2) Adoption by name.
	byName, err := s.epics.GetByName(ctx, projectID, ie.Name)
	if err != nil && !isNotFound(err) {
		s.failEpic(summary, ie, codeUpsertError, fmt.Sprintf("resolve epic by name: %v", err))
		return
	}
	if byName == nil {
		s.createEpic(ctx, projectID, cfg, ie, mapped, source, extID, url, epicID, summary)
		return
	}

	switch {
	case byName.Source == string(port.SourceManual):
		// Adopt: self-heal an in-app epic to this source (provenance rewrite).
		descMerged := mergeStrPtr(ie.Description, byName.Description)
		status := resolveEpicStatus(byName, mapped)
		s.writeEpicUpdate(ctx, cfg, byName, ie.Name, descMerged, status, source, extID, url, epicID, summary)
	case byName.Source == source && derefStr(byName.ExternalID) != extID:
		// Same source, different external_id: duplicate name within the source.
		// Keep+link children, do NOT overwrite, warn.
		epicID[extID] = byName.ID
		summary.Warnings = append(summary.Warnings, port.ImportWarning{
			Key: ie.Key, Code: codeNameConflict,
			Message: fmt.Sprintf("epic name %q already exists for source %q under a different id; linked, not overwritten", ie.Name, source),
		})
		s.decide(summary, ie.Key, kindEpic, actionSkip, url, byName.Status, "name conflict — linked")
	default:
		// Different remote source shares this name: link-only, never rewrite
		// provenance (kills the markdown<->github_projects flip-flop).
		epicID[extID] = byName.ID
		summary.Warnings = append(summary.Warnings, port.ImportWarning{
			Key: ie.Key, Code: codeNameConflict,
			Message: fmt.Sprintf("epic name %q shared across sources (existing source %q); linked only", ie.Name, byName.Source),
		})
		s.decide(summary, ie.Key, kindEpic, actionSkip, url, byName.Status, "name shared across sources — linked")
	}
}

func (s *PlanningImportService) createEpic(ctx context.Context, projectID uuid.UUID, cfg port.ImportConfig, ie port.ImportedEpic, mapped, source, extID, url string, epicID map[string]uuid.UUID, summary *port.ImportSummary) {
	if cfg.DryRun {
		summary.EpicsCreated++
		s.decide(summary, ie.Key, kindEpic, actionCreate, url, mapped, "new epic")
		return
	}
	created, err := s.epics.CreateFromImport(ctx, &model.Epic{
		ProjectID:   projectID,
		Name:        ie.Name,
		Description: ie.Description,
		Status:      mapped,
		Source:      source,
		ExternalID:  &extID,
		SourceURL:   ptrIfNonEmpty(url),
	})
	if err != nil {
		code := codeUpsertError
		if isConflict(err) {
			code = codeNameConflict
		}
		s.failEpic(summary, ie, code, fmt.Sprintf("create epic: %v", err))
		return
	}
	epicID[extID] = created.ID
	summary.EpicsCreated++
	s.decide(summary, ie.Key, kindEpic, actionCreate, url, created.Status, "new epic")
}

func (s *PlanningImportService) writeEpicUpdate(ctx context.Context, cfg port.ImportConfig, existing *model.Epic, name string, descMerged *string, status, source, extID, url string, epicID map[string]uuid.UUID, summary *port.ImportSummary) {
	epicID[extID] = existing.ID
	if cfg.DryRun {
		summary.EpicsUpdated++
		s.decide(summary, name, kindEpic, actionUpdate, url, status, "updated")
		return
	}
	updated, err := s.epics.UpdateFromImport(ctx, &model.Epic{
		ID:          existing.ID,
		ProjectID:   existing.ProjectID,
		Name:        name,
		Description: descMerged,
		Status:      status,
		Source:      source,
		ExternalID:  &extID,
		SourceURL:   ptrIfNonEmpty(url),
	})
	if err != nil {
		code := codeUpsertError
		if isConflict(err) {
			code = codeNameConflict
		}
		s.failEpic(summary, port.ImportedEpic{Key: name, Ref: port.SourceRef{ExternalID: extID}}, code, fmt.Sprintf("update epic: %v", err))
		return
	}
	epicID[extID] = updated.ID
	summary.EpicsUpdated++
	s.decide(summary, name, kindEpic, actionUpdate, url, updated.Status, "updated")
}

// importStory applies key validation, identity resolution, lock-while-running,
// hash no-op, preserve-on-absent, set-once epic_id and promote-only status.
func (s *PlanningImportService) importStory(ctx context.Context, projectID uuid.UUID, cfg port.ImportConfig, is port.ImportedStory, epicID map[string]uuid.UUID, summary *port.ImportSummary) {
	source := string(cfg.Source)
	extID := is.Ref.ExternalID
	url := is.Ref.URL

	if is.ParseError != nil {
		s.failStory(summary, is, codeParseError, fmt.Sprintf("invalid markdown frontmatter: %v", is.ParseError))
		return
	}
	if !storyKeyPattern.MatchString(is.Key) {
		s.failStory(summary, is, codeKeyFormat, "key must match format [A-Z0-9]+-N (e.g. S-14)")
		return
	}
	if strings.TrimSpace(is.Title) == "" {
		s.failStory(summary, is, codeParseError, "title is required (no H1 heading found)")
		return
	}

	mapped := projectStoryStatus(cfg, is.RawStatus)

	// Resolve the parent epic id (set-once / orphan).
	var resolvedEpic *uuid.UUID
	if is.EpicRef != nil {
		if id, ok := epicID[is.EpicRef.ExternalID]; ok && id != uuid.Nil {
			idCopy := id
			resolvedEpic = &idCopy
		}
	}

	// Resolve identity.
	existing, finalKey, derr := s.resolveStoryIdentity(ctx, projectID, cfg, is, source, extID)
	if derr != nil {
		s.failStory(summary, is, derr.code, derr.message)
		return
	}

	if existing == nil {
		s.createStory(ctx, projectID, cfg, is, finalKey, mapped, source, extID, url, resolvedEpic, summary)
		return
	}
	s.updateStory(ctx, cfg, is, existing, mapped, source, extID, url, resolvedEpic, summary)
}

// storyResolveError carries a per-item failure out of identity resolution.
type storyResolveError struct {
	code    string
	message string
}

// resolveStoryIdentity finds the existing row for an imported story and the key to
// create under. markdown resolves by key (and only adopts manual/markdown rows);
// remote sources resolve by source ref and derive a collision-free key on create.
func (s *PlanningImportService) resolveStoryIdentity(ctx context.Context, projectID uuid.UUID, cfg port.ImportConfig, is port.ImportedStory, source, extID string) (*model.Story, string, *storyResolveError) {
	if cfg.Source == port.SourceMarkdown {
		existing, err := s.stories.GetByKey(ctx, projectID, is.Key)
		if err != nil {
			if isNotFound(err) {
				return nil, is.Key, nil
			}
			return nil, "", &storyResolveError{codeUpsertError, fmt.Sprintf("resolve story by key: %v", err)}
		}
		// Author-owned keys: only adopt manual/markdown rows; a different remote
		// owner is a hard KEY_CONFLICT (never clobber).
		if existing.Source != string(port.SourceManual) && existing.Source != string(port.SourceMarkdown) {
			return nil, "", &storyResolveError{codeKeyConflict, fmt.Sprintf("markdown key %q already owned by a %s story", is.Key, existing.Source)}
		}
		return existing, is.Key, nil
	}

	// Remote source: resolve by stable identity.
	existing, err := s.stories.GetBySourceRef(ctx, projectID, source, extID)
	if err != nil && !isNotFound(err) {
		return nil, "", &storyResolveError{codeUpsertError, fmt.Sprintf("resolve story by source ref: %v", err)}
	}
	if existing != nil {
		return existing, existing.Key, nil
	}
	// New remote item: derive a deterministic, collision-free key.
	finalKey, err := s.resolveRemoteCreateKey(ctx, projectID, is.Key, source, extID)
	if err != nil {
		return nil, "", &storyResolveError{codeUpsertError, fmt.Sprintf("resolve create key: %v", err)}
	}
	return nil, finalKey, nil
}

func (s *PlanningImportService) createStory(ctx context.Context, projectID uuid.UUID, cfg port.ImportConfig, is port.ImportedStory, finalKey, mapped, source, extID, url string, epic *uuid.UUID, summary *port.ImportSummary) {
	hash := canonicalStoryHash(is, mapped)
	if cfg.DryRun {
		summary.StoriesCreated++
		s.decide(summary, finalKey, kindStory, actionCreate, url, mapped, "new story")
		return
	}
	_, err := s.stories.CreateFromImport(ctx, &model.Story{
		ProjectID:          projectID,
		EpicID:             epic,
		Key:                finalKey,
		Title:              is.Title,
		Objective:          is.Objective,
		AcceptanceCriteria: is.AcceptanceCriteria,
		Scope:              is.Scope,
		DependsOn:          is.DependsOn,
		Status:             mapped,
		Source:             source,
		ExternalID:         &extID,
		ExternalItemID:     ptrIfNonEmpty(is.ExternalItemID),
		SourceURL:          ptrIfNonEmpty(url),
		LastImportHash:     &hash,
	})
	if err != nil {
		code := codeUpsertError
		if isConflict(err) {
			code = codeKeyConflict
		}
		s.failStory(summary, is, code, fmt.Sprintf("create story: %v", err))
		return
	}
	summary.StoriesCreated++
	s.decide(summary, finalKey, kindStory, actionCreate, url, mapped, "new story")
}

func (s *PlanningImportService) updateStory(ctx context.Context, cfg port.ImportConfig, is port.ImportedStory, existing *model.Story, mapped, source, extID, url string, epic *uuid.UUID, summary *port.ImportSummary) {
	locked := existing.Status == model.StoryStatusRunning ||
		existing.Status == model.StoryStatusFailed ||
		existing.CurrentStage != nil
	hash := canonicalStoryHash(is, mapped)

	// Hash no-op (unlocked, unchanged spec) => true no-op, no write. external_item_id
	// is provenance (not in the spec hash), so a row imported before write-back
	// shipped would never receive it; backfill it best-effort here without disturbing
	// the no-op gate (reuses the provenance-only write — same values, +item id).
	if !locked && existing.LastImportHash != nil && *existing.LastImportHash == hash {
		if !cfg.DryRun && is.ExternalItemID != "" && derefStr(existing.ExternalItemID) != is.ExternalItemID {
			upd := *existing
			upd.ExternalItemID = &is.ExternalItemID
			if _, err := s.stories.UpdateProvenanceOnly(ctx, &upd); err != nil {
				s.failStory(summary, is, codeUpsertError, fmt.Sprintf("backfill external_item_id: %v", err))
				return
			}
		}
		summary.Skipped++
		s.decide(summary, existing.Key, kindStory, actionSkip, url, existing.Status, "unchanged (hash match)")
		return
	}

	if locked {
		// Cosmetic title + provenance only; do NOT advance last_import_hash so the
		// frozen spec change re-applies on the first re-import after the run ends.
		provChanged := existing.Title != is.Title ||
			existing.Source != source ||
			derefStr(existing.ExternalID) != extID ||
			derefStr(existing.SourceURL) != url ||
			(is.ExternalItemID != "" && derefStr(existing.ExternalItemID) != is.ExternalItemID)
		if provChanged && !cfg.DryRun {
			upd := *existing
			upd.Title = is.Title
			upd.Source = source
			upd.ExternalID = &extID
			upd.ExternalItemID = mergeStrPtr(ptrIfNonEmpty(is.ExternalItemID), existing.ExternalItemID)
			upd.SourceURL = ptrIfNonEmpty(url)
			if _, err := s.stories.UpdateProvenanceOnly(ctx, &upd); err != nil {
				s.failStory(summary, is, codeUpsertError, fmt.Sprintf("update story provenance: %v", err))
				return
			}
		}
		summary.Locked++
		s.decide(summary, existing.Key, kindStory, actionLock, url, existing.Status, "running — status & spec frozen")
		return
	}

	// Unlocked + changed: merge in Go (preserve-on-absent, set-once epic, promote-only).
	merged := *existing
	merged.Title = is.Title
	merged.Objective = mergeStrPtr(is.Objective, existing.Objective)
	merged.AcceptanceCriteria = mergeStrPtr(is.AcceptanceCriteria, existing.AcceptanceCriteria)
	merged.Scope = mergeStrPtr(is.Scope, existing.Scope)
	if is.DependsOn != nil {
		merged.DependsOn = is.DependsOn
	}
	if merged.EpicID == nil {
		merged.EpicID = epic // set-once
	}
	merged.Status = resolveStoryStatus(existing, mapped)
	merged.Source = source
	merged.ExternalID = &extID
	merged.ExternalItemID = mergeStrPtr(ptrIfNonEmpty(is.ExternalItemID), existing.ExternalItemID)
	merged.SourceURL = ptrIfNonEmpty(url)
	merged.LastImportHash = &hash

	if cfg.DryRun {
		summary.StoriesUpdated++
		s.decide(summary, existing.Key, kindStory, actionUpdate, url, merged.Status, "updated")
		return
	}
	if _, err := s.stories.UpdateFromImport(ctx, &merged); err != nil {
		code := codeUpsertError
		if isConflict(err) {
			code = codeKeyConflict
		}
		s.failStory(summary, is, code, fmt.Sprintf("update story: %v", err))
		return
	}
	summary.StoriesUpdated++
	s.decide(summary, existing.Key, kindStory, actionUpdate, url, merged.Status, "updated")
}

// resolveRemoteCreateKey returns a regex-valid, deterministic key for a brand-new
// remote item whose natural key may already be owned by a different identity. The
// key is set-once: re-import resolves by source ref and keeps the stored key.
func (s *PlanningImportService) resolveRemoteCreateKey(ctx context.Context, projectID uuid.UUID, baseKey, source, externalID string) (string, error) {
	candidate := baseKey
	for attempt := 0; attempt < 8; attempt++ {
		existing, err := s.stories.GetByKey(ctx, projectID, candidate)
		if err != nil {
			if isNotFound(err) {
				return candidate, nil // free
			}
			return "", err
		}
		if existing.Source == source && derefStr(existing.ExternalID) == externalID {
			return candidate, nil // already ours (defensive; GetBySourceRef missed)
		}
		candidate = disambiguateKey(baseKey, externalID, attempt)
	}
	return candidate, nil
}

// --- pure helpers (no DB) ---

// disambiguateKey derives BASE+<hash slice>-NUMBER from a colliding key. The hash
// is fnv1a(externalID) base36 (deterministic), widened by one char per attempt.
func disambiguateKey(baseKey, externalID string, attempt int) string {
	base, number := splitStoryKey(baseKey)
	if base == "" {
		base = "GH"
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(externalID))
	suffix := strings.ToUpper(strconv.FormatUint(h.Sum64(), 36))
	n := 4 + attempt
	if n > len(suffix) {
		n = len(suffix)
	}
	return base + suffix[:n] + "-" + number
}

// splitStoryKey splits a "BASE-NUMBER" key. Callers validate the key matches
// ^[A-Z0-9]+-\d+$ first, so a dash is always present.
func splitStoryKey(key string) (base, number string) {
	idx := strings.LastIndex(key, "-")
	if idx < 0 {
		return key, "1"
	}
	return key[:idx], key[idx+1:]
}

// canonicalStoryHash hashes the import-significant spec of a story (excludes the
// volatile source_url/synced_at). Equal hash + unlocked => true no-op.
func canonicalStoryHash(is port.ImportedStory, mappedStatus string) string {
	deps := append([]string(nil), is.DependsOn...)
	sort.Strings(deps)
	epicExt := ""
	if is.EpicRef != nil {
		epicExt = is.EpicRef.ExternalID
	}
	parts := []string{
		is.Title,
		derefStr(is.Objective),
		derefStr(is.AcceptanceCriteria),
		derefStr(is.Scope),
		strings.Join(deps, ","),
		epicExt,
		mappedStatus,
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:])
}

// projectStoryStatus maps a raw external status to {backlog, done} — explicit only.
func projectStoryStatus(cfg port.ImportConfig, raw string) string {
	switch cfg.Source {
	case port.SourceGitHub:
		var done []string
		if cfg.GitHubProjects != nil {
			done = cfg.GitHubProjects.DoneOptions
		}
		if caseInsensitiveIn(raw, done) {
			return model.StoryStatusDone
		}
		return model.StoryStatusBacklog
	case port.SourceMarkdown:
		if strings.ToLower(strings.TrimSpace(raw)) == model.StoryStatusDone {
			return model.StoryStatusDone
		}
		return model.StoryStatusBacklog
	default:
		return model.StoryStatusBacklog
	}
}

// projectEpicStatus mirrors projectStoryStatus for an epic's own raw status.
func projectEpicStatus(cfg port.ImportConfig, raw string) string {
	switch cfg.Source {
	case port.SourceGitHub:
		var done []string
		if cfg.GitHubProjects != nil {
			done = cfg.GitHubProjects.DoneOptions
		}
		if caseInsensitiveIn(raw, done) {
			return model.EpicStatusDone
		}
		return model.EpicStatusBacklog
	case port.SourceMarkdown:
		if strings.ToLower(strings.TrimSpace(raw)) == model.EpicStatusDone {
			return model.EpicStatusDone
		}
		return model.EpicStatusBacklog
	default:
		return model.EpicStatusBacklog
	}
}

// resolveStoryStatus promotes only backlog->done on a never-executed story; it
// never downgrades and never produces running/failed.
func resolveStoryStatus(existing *model.Story, mapped string) string {
	if existing.Status == model.StoryStatusBacklog && existing.CurrentStage == nil && mapped == model.StoryStatusDone {
		return model.StoryStatusDone
	}
	return existing.Status
}

// resolveEpicStatus promotes only backlog->done; never touches in_progress/done.
func resolveEpicStatus(existing *model.Epic, mapped string) string {
	if existing.Status == model.EpicStatusBacklog && mapped == model.EpicStatusDone {
		return model.EpicStatusDone
	}
	return existing.Status
}

// epicChanged reports whether the merged epic target differs from the existing row.
func epicChanged(existing *model.Epic, name string, desc *string, status, source, extID, url string) bool {
	return existing.Name != name ||
		derefStr(existing.Description) != derefStr(desc) ||
		existing.Status != status ||
		existing.Source != source ||
		derefStr(existing.ExternalID) != extID ||
		derefStr(existing.SourceURL) != url
}

func caseInsensitiveIn(v string, set []string) bool {
	vl := strings.ToLower(strings.TrimSpace(v))
	if vl == "" {
		return false
	}
	for _, s := range set {
		if strings.ToLower(strings.TrimSpace(s)) == vl {
			return true
		}
	}
	return false
}

// mergeStrPtr returns incoming when it carries a value, else preserves existing
// (preserve-on-absent: the importer never nulls out a field the source omits).
func mergeStrPtr(incoming, existing *string) *string {
	if incoming != nil {
		return incoming
	}
	return existing
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptrIfNonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// isConflict classifies a repo conflict DomainError (23505 -> KEY/NAME_CONFLICT).
// isNotFound is shared with run_service.go in this package.
func isConflict(err error) bool {
	de, ok := err.(*errors.DomainError)
	return ok && de.Category == errors.CategoryConflict
}

// sourceError wraps an adapter/factory failure as SOURCE_ERROR (HTTP 422).
func sourceError(err error) error {
	return errors.NewInvalidState("SOURCE_ERROR", err.Error())
}

// --- summary mutators ---

func (s *PlanningImportService) decide(summary *port.ImportSummary, key, kind, action, url, mappedStatus, reason string) {
	summary.Items = append(summary.Items, port.ImportItemDecision{
		Key:          key,
		Kind:         kind,
		Action:       action,
		SourceURL:    url,
		MappedStatus: mappedStatus,
		Reason:       reason,
	})
}

func (s *PlanningImportService) failStory(summary *port.ImportSummary, is port.ImportedStory, code, message string) {
	summary.Failed++
	summary.Errors = append(summary.Errors, port.ImportItemError{
		Key:        is.Key,
		ExternalID: is.Ref.ExternalID,
		Code:       code,
		Message:    message,
	})
	s.decide(summary, is.Key, kindStory, actionFail, is.Ref.URL, "", message)
}

func (s *PlanningImportService) failEpic(summary *port.ImportSummary, ie port.ImportedEpic, code, message string) {
	summary.Failed++
	summary.Errors = append(summary.Errors, port.ImportItemError{
		Key:        ie.Key,
		ExternalID: ie.Ref.ExternalID,
		Code:       code,
		Message:    message,
	})
	s.decide(summary, ie.Key, kindEpic, actionFail, ie.Ref.URL, "", message)
}
