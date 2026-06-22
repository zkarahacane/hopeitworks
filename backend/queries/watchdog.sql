-- name: ListRunningStepsForWatchdog :many
-- All currently-running run steps across every project, enriched with the run
-- context the watchdog needs (project, story) and the timestamp of the most
-- recent log.emitted event for the step (NULL when no log yet). The watchdog
-- compares now() against last_log_at (log_silence), started_at (wallclock) and
-- the run's cumulative cost (cost_batch). Only running steps of running runs are
-- returned — a paused run is already halted, skip it. The LEFT JOIN LATERAL
-- keeps last_log_at nullable (vs a scalar subquery, which sqlc infers non-null).
SELECT
    rs.id            AS step_id,
    rs.run_id        AS run_id,
    rs.step_name     AS step_name,
    rs.stage_id      AS stage_id,
    rs.stage_name    AS stage_name,
    rs.started_at    AS started_at,
    r.project_id     AS project_id,
    r.story_id       AS story_id,
    ll.last_log_at   AS last_log_at
FROM run_steps rs
JOIN runs r ON r.id = rs.run_id
LEFT JOIN LATERAL (
    SELECT max(e.created_at) AS last_log_at
    FROM events e
    WHERE e.entity_type = 'log'
      AND e.entity_id = rs.id
      AND e.action = 'emitted'
) ll ON true
WHERE rs.status = 'running'
  AND r.status = 'running';
