CREATE TABLE epic_run_stories (
    epic_run_id UUID NOT NULL REFERENCES epic_runs(id) ON DELETE CASCADE,
    story_id    UUID NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    run_id      UUID REFERENCES runs(id) ON DELETE SET NULL,
    group_index INTEGER NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',
    PRIMARY KEY (epic_run_id, story_id)
);

CREATE INDEX idx_epic_run_stories_epic_run_id ON epic_run_stories(epic_run_id);
CREATE INDEX idx_epic_run_stories_run_id ON epic_run_stories(run_id);
