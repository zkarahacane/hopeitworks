-- name: InsertCostRecord :one
INSERT INTO cost_records (run_step_id, project_id, tokens_input, tokens_output, cost_usd, model, agent_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetCostByRunStep :one
SELECT * FROM cost_records WHERE run_step_id = $1 LIMIT 1;

-- name: SumCostByProject :one
SELECT COALESCE(SUM(cost_usd), 0)::DECIMAL(10,6) AS total_cost,
       COALESCE(SUM(tokens_input), 0)::BIGINT     AS total_input,
       COALESCE(SUM(tokens_output), 0)::BIGINT    AS total_output
FROM cost_records
WHERE project_id = $1 AND created_at >= $2;

-- name: SumCostByRun :one
SELECT COALESCE(SUM(cr.cost_usd), 0)::DECIMAL(10,6) AS total_cost
FROM cost_records cr
JOIN run_steps rs ON rs.id = cr.run_step_id
WHERE rs.run_id = $1;

-- name: SumCostByStory :one
SELECT COALESCE(SUM(cr.cost_usd), 0)::DECIMAL(10,6) AS total_cost,
       COALESCE(SUM(cr.tokens_input), 0)             AS total_input,
       COALESCE(SUM(cr.tokens_output), 0)            AS total_output,
       COUNT(DISTINCT rs.run_id)::INT                 AS run_count
FROM cost_records cr
JOIN run_steps rs ON rs.id = cr.run_step_id
JOIN runs r ON r.id = rs.run_id
WHERE r.story_id = $1;

-- name: ListCostsByProjectByStory :many
SELECT r.story_id,
       s.key AS story_key,
       COALESCE(SUM(cr.cost_usd), 0)::DECIMAL(10,6) AS total_cost
FROM cost_records cr
JOIN run_steps rs ON rs.id = cr.run_step_id
JOIN runs r ON r.id = rs.run_id
JOIN stories s ON s.id = r.story_id
WHERE cr.project_id = $1 AND cr.created_at >= $2
GROUP BY r.story_id, s.key
ORDER BY total_cost DESC;

-- name: ListCostsByProjectByRun :many
SELECT rs2.run_id,
       s.key AS story_key,
       r.status,
       COALESCE(SUM(cr.cost_usd), 0)::DECIMAL(10,6) AS total_cost,
       r.created_at
FROM cost_records cr
JOIN run_steps rs2 ON rs2.id = cr.run_step_id
JOIN runs r ON r.id = rs2.run_id
JOIN stories s ON s.id = r.story_id
WHERE cr.project_id = $1 AND cr.created_at >= $2
GROUP BY rs2.run_id, s.key, r.status, r.created_at
ORDER BY r.created_at DESC;

-- name: ListCostsByProjectByModel :many
SELECT cr.model,
       COALESCE(SUM(cr.cost_usd), 0)::DECIMAL(10,6) AS total_cost,
       COALESCE(SUM(cr.tokens_input), 0)             AS tokens_input,
       COALESCE(SUM(cr.tokens_output), 0)            AS tokens_output
FROM cost_records cr
WHERE cr.project_id = $1 AND cr.created_at >= $2
GROUP BY cr.model
ORDER BY total_cost DESC;

-- name: ListStepCostsByRun :many
SELECT cr.run_step_id AS step_id,
       rs.step_name,
       cr.model,
       cr.tokens_input,
       cr.tokens_output,
       cr.cost_usd
FROM cost_records cr
JOIN run_steps rs ON rs.id = cr.run_step_id
WHERE rs.run_id = $1
ORDER BY rs.step_order ASC;

-- name: ListDailyCostsByProject :many
SELECT
    DATE(cr.created_at)::text AS date,
    COALESCE(SUM(cr.cost_usd), 0)::DECIMAL(10,6) AS total_cost_usd
FROM cost_records cr
WHERE cr.project_id = $1
  AND cr.created_at >= $2
GROUP BY DATE(cr.created_at)
ORDER BY date ASC;

-- name: ListCostsByProjectByRunPaginated :many
SELECT rs2.run_id,
       s.key    AS story_key,
       r.status,
       r.created_at AS started_at,
       COALESCE(SUM(cr.cost_usd), 0)::DECIMAL(10,6) AS total_cost_usd
FROM cost_records cr
JOIN run_steps rs2 ON rs2.id = cr.run_step_id
JOIN runs r ON r.id = rs2.run_id
JOIN stories s ON s.id = r.story_id
WHERE cr.project_id = $1 AND cr.created_at >= $2
GROUP BY rs2.run_id, s.key, r.status, r.created_at
ORDER BY r.created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountCostsByProjectByRun :one
SELECT COUNT(DISTINCT rs2.run_id)
FROM cost_records cr
JOIN run_steps rs2 ON rs2.id = cr.run_step_id
WHERE cr.project_id = $1 AND cr.created_at >= $2;

-- name: ListCostsByProjectByAgent :many
SELECT
  cr.agent_id,
  COALESCE(a.name, 'Unknown') AS agent_name,
  SUM(cr.tokens_input)::bigint AS tokens_input,
  SUM(cr.tokens_output)::bigint AS tokens_output,
  SUM(cr.cost_usd)::DECIMAL(10,6) AS cost_usd,
  COUNT(DISTINCT rs.run_id)::int AS runs_count
FROM cost_records cr
LEFT JOIN agents a ON a.id = cr.agent_id
JOIN run_steps rs ON rs.id = cr.run_step_id
JOIN runs r ON r.id = rs.run_id
WHERE r.project_id = $1
  AND cr.agent_id IS NOT NULL
GROUP BY cr.agent_id, a.name
ORDER BY cost_usd DESC;
