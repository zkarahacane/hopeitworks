# Story fix-7: Fix pipeline config API returning 404 for seeded project

Status: ready-for-dev

## Story

As a developer,
I want the pipeline config endpoint to return 200 for seeded projects,
so that E2E tests can verify the pipeline configuration page works.

## Context

The smoke test `pipeline config API returns 200 with config_yaml field` calls `GET /api/v1/projects/00000000-0000-0000-0000-000000000101/pipeline-config` and gets a 404 response. The seed SQL (`backend/testdata/seed.sql`) does INSERT a pipeline_config for that project, but the data may not be present after `e2e-stack.sh reset` if migrations changed the table schema, or the endpoint path may have changed.

## Acceptance Criteria (BDD)

**AC1: Pipeline config exists after seed**
- **Given** the database has been reset and seeded via `e2e-stack.sh reset`
- **When** I query `SELECT * FROM pipeline_configs WHERE project_id = '00000000-0000-0000-0000-000000000101'`
- **Then** exactly 1 row is returned with valid `config_yaml`

**AC2: API endpoint returns 200**
- **Given** the pipeline config is seeded
- **When** I call `GET /api/v1/projects/00000000-0000-0000-0000-000000000101/pipeline-config`
- **Then** the response is 200 with a JSON body containing `config_yaml`

**AC3: If endpoint path changed, update the test**
- **Given** the OpenAPI spec defines the pipeline config endpoint
- **When** I check `api/openapi.yaml` for the correct path
- **Then** the test uses the correct endpoint path

## Tasks / Subtasks

- [ ] Task 1: Check `api/openapi.yaml` for the pipeline config endpoint path
- [ ] Task 2: Verify the seed SQL inserts pipeline_configs correctly for the Todo App project
- [ ] Task 3: Check if the DB table schema matches the seed INSERT columns (migrations may have added/removed columns)
- [ ] Task 4: If the seed data is missing after reset, fix the seed SQL or the reset script
- [ ] Task 5: If the endpoint path differs from what the test uses, update `smoke-pipeline-config.spec.ts`
- [ ] Task 6: Verify `GET /api/v1/projects/{id}/pipeline-config` returns 200 after fix
