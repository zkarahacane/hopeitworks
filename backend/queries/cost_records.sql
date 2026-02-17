-- name: InsertCostRecord :one
INSERT INTO cost_records (run_step_id, project_id, tokens_input, tokens_output, cost_usd, model)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetCostByRunStep :one
SELECT * FROM cost_records WHERE run_step_id = $1 LIMIT 1;

-- name: SumCostByProject :one
SELECT COALESCE(SUM(cost_usd), 0)::DECIMAL(10,6) AS total_cost,
       COALESCE(SUM(tokens_input), 0)             AS total_input,
       COALESCE(SUM(tokens_output), 0)            AS total_output
FROM cost_records
WHERE project_id = $1 AND created_at >= $2;

-- name: SumCostByRun :one
SELECT COALESCE(SUM(cr.cost_usd), 0)::DECIMAL(10,6) AS total_cost
FROM cost_records cr
JOIN run_steps rs ON rs.id = cr.run_step_id
WHERE rs.run_id = $1;
