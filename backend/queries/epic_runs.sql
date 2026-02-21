-- name: CreateEpicRun :one
INSERT INTO epic_runs (project_id, epic_id, status)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetEpicRun :one
SELECT * FROM epic_runs WHERE id = $1;

-- name: UpdateEpicRunStatus :one
UPDATE epic_runs
SET status = $1,
    completed_at = $2
WHERE id = $3
RETURNING *;

-- name: InsertEpicRunStory :exec
INSERT INTO epic_run_stories (epic_run_id, story_id, run_id, group_index, status)
VALUES ($1, $2, $3, $4, $5);

-- name: UpdateEpicRunStoryStatus :exec
UPDATE epic_run_stories
SET status = $1,
    run_id = $2
WHERE epic_run_id = $3
  AND story_id = $4;

-- name: ListEpicRunStories :many
SELECT * FROM epic_run_stories
WHERE epic_run_id = $1
ORDER BY group_index, story_id;
