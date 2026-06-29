-- name: GetPlanningConnector :one
SELECT * FROM planning_connectors WHERE project_id = $1;

-- name: UpsertPlanningConnector :one
INSERT INTO planning_connectors (
    project_id, source, project_url, status_field, done_options,
    epic_issue_type, status_mapping, writeback_enabled, post_run_comment
) VALUES (
    @project_id, @source, @project_url, @status_field, @done_options,
    @epic_issue_type, @status_mapping, @writeback_enabled, @post_run_comment
)
ON CONFLICT (project_id) DO UPDATE SET
    source            = @source,
    project_url       = @project_url,
    status_field      = @status_field,
    done_options      = @done_options,
    epic_issue_type   = @epic_issue_type,
    status_mapping    = @status_mapping,
    writeback_enabled = @writeback_enabled,
    post_run_comment  = @post_run_comment,
    updated_at        = now()
RETURNING *;
