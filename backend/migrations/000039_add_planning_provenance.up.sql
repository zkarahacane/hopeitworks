-- Planning provenance: each story/epic records where it came from so the in-app
-- kanban is a faithful projection of an external plan (markdown, GitHub Projects).
-- This replaces the fake git_provider heuristic the board used to derive on the fly.
--
-- All additive: NOT NULL DEFAULT 'manual' backfills every existing (in-app/seed) row
-- to source='manual', external_id=NULL automatically — no separate UPDATE. A legacy
-- markdown-imported row stays labelled 'manual' until its next markdown re-import,
-- which self-heals it (resolution is by key, so no duplicate is created).
--
-- source is VARCHAR(20): the widest value 'github_projects' (15 chars) fits.

ALTER TABLE stories
  ADD COLUMN source           VARCHAR(20)  NOT NULL DEFAULT 'manual',
  ADD COLUMN external_id      TEXT,
  ADD COLUMN source_url       TEXT,
  ADD COLUMN synced_at        TIMESTAMPTZ,
  ADD COLUMN last_import_hash VARCHAR(64);

ALTER TABLE epics
  ADD COLUMN source      VARCHAR(20)  NOT NULL DEFAULT 'manual',
  ADD COLUMN external_id TEXT,
  ADD COLUMN source_url  TEXT,
  ADD COLUMN synced_at   TIMESTAMPTZ;

-- Idempotency identity for remote sources. PARTIAL so manual rows
-- (external_id IS NULL) never participate and never collide.
CREATE UNIQUE INDEX stories_uq_source_external
  ON stories (project_id, source, external_id) WHERE external_id IS NOT NULL;
CREATE UNIQUE INDEX epics_uq_source_external
  ON epics (project_id, source, external_id) WHERE external_id IS NOT NULL;
