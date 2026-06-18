-- =============================================================================
-- hopeitworks dev seed data
-- =============================================================================
-- Purpose: Pre-populate local dev database with test data.
-- Run:     cd backend && make seed
-- Reset:   cd backend && make reset-db
--
-- Credentials:
--   admin@hopeitworks.dev  / admin1234  (admin)
--   sarah@hopeitworks.dev  / admin1234  (admin)
--   marc@hopeitworks.dev   / admin1234  (admin)
--   dev@hopeitworks.dev    / user1234   (user)
--   alice@hopeitworks.dev  / user1234   (user)
--   bob@hopeitworks.dev    / user1234   (user)
--
-- Idempotent: safe to run multiple times (uses ON CONFLICT).
-- =============================================================================

BEGIN;

-- ---------------------------------------------------------------------------
-- Users  (3 admins + 3 users)
-- ---------------------------------------------------------------------------

INSERT INTO users (id, email, password_hash, name, role) VALUES
('00000000-0000-0000-0000-000000000001', 'admin@hopeitworks.dev', '$2a$10$OxGZnxAwWA6XQt45Z5RUxOcKuknitNZFYNTDYGl.52yR35cBPdN/C', 'Admin User', 'admin'),
('00000000-0000-0000-0000-000000000002', 'sarah@hopeitworks.dev', '$2a$10$OxGZnxAwWA6XQt45Z5RUxOcKuknitNZFYNTDYGl.52yR35cBPdN/C', 'Sarah Chen', 'admin'),
('00000000-0000-0000-0000-000000000003', 'marc@hopeitworks.dev',  '$2a$10$OxGZnxAwWA6XQt45Z5RUxOcKuknitNZFYNTDYGl.52yR35cBPdN/C', 'Marc Dupont', 'admin'),
('00000000-0000-0000-0000-000000000011', 'dev@hopeitworks.dev',   '$2a$10$WrlGk50pss6bpaGdnA1HjO1qXqs5/DdbFMX2L9Uq.Z4IXtiaECaou', 'Dev User', 'user'),
('00000000-0000-0000-0000-000000000012', 'alice@hopeitworks.dev', '$2a$10$WrlGk50pss6bpaGdnA1HjO1qXqs5/DdbFMX2L9Uq.Z4IXtiaECaou', 'Alice Martin', 'user'),
('00000000-0000-0000-0000-000000000013', 'bob@hopeitworks.dev',   '$2a$10$WrlGk50pss6bpaGdnA1HjO1qXqs5/DdbFMX2L9Uq.Z4IXtiaECaou', 'Bob Nguyen', 'user')
ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    role = EXCLUDED.role;

-- ---------------------------------------------------------------------------
-- Project: Todo App  (Gitea local)
-- ---------------------------------------------------------------------------

INSERT INTO projects (id, name, description, owner_id, repo_url, git_provider, git_token_env, default_model)
VALUES (
    '00000000-0000-0000-0000-000000000101',
    'Todo App',
    'Simple todo application — Go backend + Vue 3 frontend. Used as test project for pipeline validation.',
    '00000000-0000-0000-0000-000000000001',
    'http://localhost:3030/devops/todo-app',
    'gitea',
    'GITEA_TOKEN',
    'claude-sonnet-4-6'
) ON CONFLICT (name) DO UPDATE SET
    description = EXCLUDED.description,
    owner_id = EXCLUDED.owner_id,
    repo_url = EXCLUDED.repo_url,
    git_provider = EXCLUDED.git_provider,
    git_token_env = EXCLUDED.git_token_env,
    default_model = EXCLUDED.default_model;

INSERT INTO projects (id, name, description, owner_id, repo_url, git_provider, git_token_env, default_model)
VALUES (
    '00000000-0000-0000-0000-000000000102',
    'E-commerce API',
    'REST API for an e-commerce backend',
    '00000000-0000-0000-0000-000000000001',
    'http://localhost:3030/devops/todo-app',
    'gitea',
    'GITEA_TOKEN',
    'claude-sonnet-4-6'
) ON CONFLICT (name) DO UPDATE SET
    description = EXCLUDED.description,
    owner_id = EXCLUDED.owner_id,
    repo_url = EXCLUDED.repo_url,
    git_provider = EXCLUDED.git_provider,
    git_token_env = EXCLUDED.git_token_env,
    default_model = EXCLUDED.default_model;

-- ---------------------------------------------------------------------------
-- Project memberships
-- ---------------------------------------------------------------------------

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'project_users'
    ) THEN
        INSERT INTO project_users (project_id, user_id, role) VALUES
        ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000001', 'owner'),
        ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000002', 'member'),
        ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000011', 'member'),
        ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000012', 'member'),
        ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000013', 'member')
        ON CONFLICT DO NOTHING;
        RAISE NOTICE 'Seed: project_users memberships inserted';
    END IF;
END $$;

-- ---------------------------------------------------------------------------
-- Epic: Foundation
-- ---------------------------------------------------------------------------

INSERT INTO epics (id, project_id, name, description, status)
VALUES (
    '00000000-0000-0000-0000-000000000201',
    '00000000-0000-0000-0000-000000000101',
    'Foundation',
    'Foundation — project scaffolding, CI, and core infrastructure',
    'in_progress'
) ON CONFLICT (project_id, name) DO UPDATE SET
    description = EXCLUDED.description,
    status = EXCLUDED.status;

-- ---------------------------------------------------------------------------
-- Epic: Task Management
-- ---------------------------------------------------------------------------

INSERT INTO epics (id, project_id, name, description, status)
VALUES (
    '00000000-0000-0000-0000-000000000202',
    '00000000-0000-0000-0000-000000000101',
    'Task Management',
    'Task management features — CRUD, assignments, and status tracking',
    'in_progress'
) ON CONFLICT (project_id, name) DO UPDATE SET
    description = EXCLUDED.description,
    status = EXCLUDED.status;

-- ---------------------------------------------------------------------------
-- Stories (3 stories in the MVP epic)
-- ---------------------------------------------------------------------------

-- S-01: Project scaffolding
INSERT INTO stories (id, project_id, epic_id, key, title, objective, scope, status, depends_on, acceptance_criteria)
VALUES (
    '00000000-0000-0000-0000-000000000301',
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000201',
    'S-01',
    'Scaffold Go backend + Vue frontend',
    'Create the initial project structure with a Go chi backend and a Vue 3 + PrimeVue frontend. Both should compile and serve a hello world page.',
    'fullstack',
    'backlog',
    '[]',
    'Go backend starts on :8080 with GET /health returning 200. Vue frontend builds and serves on :5173 with a hello world page. Docker Compose brings up both services.'
) ON CONFLICT (project_id, key) DO UPDATE SET
    title = EXCLUDED.title,
    objective = EXCLUDED.objective,
    status = EXCLUDED.status;

-- S-02: CI pipeline
INSERT INTO stories (id, project_id, epic_id, key, title, objective, scope, status, depends_on, acceptance_criteria)
VALUES (
    '00000000-0000-0000-0000-000000000302',
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000201',
    'S-02',
    'Setup CI pipeline with GitHub Actions',
    'Configure GitHub Actions workflows for linting, testing, and building both backend and frontend on every push and PR.',
    'ci',
    'backlog',
    '["S-01"]',
    'CI runs on push to main and on PRs. Backend: golangci-lint + go test. Frontend: eslint + vitest + build. All jobs pass on a clean repo.'
) ON CONFLICT (project_id, key) DO UPDATE SET
    title = EXCLUDED.title,
    objective = EXCLUDED.objective,
    status = EXCLUDED.status;

-- S-03: Linting & formatting
INSERT INTO stories (id, project_id, epic_id, key, title, objective, scope, status, depends_on, acceptance_criteria)
VALUES (
    '00000000-0000-0000-0000-000000000303',
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000201',
    'S-03',
    'Configure linting and code formatting',
    'Setup golangci-lint for the backend and ESLint + Prettier for the frontend with shared configs. Add pre-commit hooks via lefthook.',
    'fullstack',
    'backlog',
    '["S-01"]',
    'golangci-lint runs clean on backend. ESLint + Prettier run clean on frontend. Lefthook pre-commit hook runs both. npm run format and gofmt produce no changes on committed code.'
) ON CONFLICT (project_id, key) DO UPDATE SET
    title = EXCLUDED.title,
    objective = EXCLUDED.objective,
    status = EXCLUDED.status;

-- S-04: Task CRUD endpoints (Task Management epic)
INSERT INTO stories (id, project_id, epic_id, key, title, objective, scope, status, depends_on, acceptance_criteria)
VALUES (
    '00000000-0000-0000-0000-000000000304',
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000202',
    'S-04',
    'Implement task CRUD endpoints',
    'Create REST endpoints to create, read, update, and delete tasks.',
    'backend',
    'completed',
    '[]',
    'POST/GET/PUT/DELETE /tasks all return correct status codes and persist to the database.'
) ON CONFLICT (project_id, key) DO UPDATE SET
    title = EXCLUDED.title,
    objective = EXCLUDED.objective,
    status = EXCLUDED.status;

-- S-05: Task list UI with status filter (Task Management epic)
INSERT INTO stories (id, project_id, epic_id, key, title, objective, scope, status, depends_on, acceptance_criteria)
VALUES (
    '00000000-0000-0000-0000-000000000305',
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000202',
    'S-05',
    'Task list UI with status filter',
    'Build the task list view with filtering by status.',
    'frontend',
    'running',
    '["S-04"]',
    'The task list renders, supports filtering by status, and updates live.'
) ON CONFLICT (project_id, key) DO UPDATE SET
    title = EXCLUDED.title,
    objective = EXCLUDED.objective,
    status = EXCLUDED.status;

-- ---------------------------------------------------------------------------
-- Agents (2 global + 2 project-scoped)
-- ---------------------------------------------------------------------------

-- Global: Opus Dev Agent
INSERT INTO agents (id, project_id, name, template_content, type, scope, model, image)
VALUES (
    '00000000-0000-0000-0000-000000000901',
    NULL,
    'Opus Dev Agent',
    'Implement story {{story_key}}: {{story_title}}

## Objective
{{story_objective}}

## Acceptance Criteria
{{acceptance_criteria}}

## Branch
{{branch_name}}',
    'implement',
    'global',
    'claude-opus-4-6',
    'hopeitworks/agent-go-node:latest'
) ON CONFLICT DO NOTHING;

-- Global: Sonnet Review Agent
INSERT INTO agents (id, project_id, name, template_content, type, scope, model, image)
VALUES (
    '00000000-0000-0000-0000-000000000902',
    NULL,
    'Sonnet Review Agent',
    'Review changes for {{story_key}}: {{story_title}}

## Acceptance Criteria
{{acceptance_criteria}}

## Changes
{{diff_content}}

## Checklist
- All acceptance criteria met
- Linters pass
- Tests added
- No hardcoded secrets',
    'review',
    'global',
    'claude-sonnet-4-6',
    'hopeitworks/agent-go-node:latest'
) ON CONFLICT DO NOTHING;

-- Project: Todo Dev Agent (overrides global with project context)
INSERT INTO agents (id, project_id, name, template_content, type, scope, model, image)
VALUES (
    '00000000-0000-0000-0000-000000000903',
    '00000000-0000-0000-0000-000000000101',
    'Todo Dev Agent',
    'Implement story {{story_key}}: {{story_title}}

## Project
Todo App — Go backend (chi, pgx, sqlc) + Vue 3 frontend (PrimeVue, Tailwind).

## Objective
{{story_objective}}

## Acceptance Criteria
{{acceptance_criteria}}

## Branch
{{branch_name}}',
    'implement',
    'project',
    'claude-sonnet-4-6',
    'hopeitworks/agent-go-node:latest'
) ON CONFLICT (project_id, name) DO UPDATE SET
    template_content = EXCLUDED.template_content,
    model = EXCLUDED.model;

-- Project: Todo Merge Agent
INSERT INTO agents (id, project_id, name, template_content, type, scope, model, image)
VALUES (
    '00000000-0000-0000-0000-000000000904',
    '00000000-0000-0000-0000-000000000101',
    'Todo Merge Agent',
    'Merge story {{story_key}} (branch {{branch_name}}) into main.

1. Verify CI green
2. Rebase on main
3. Create PR with squash merge
4. Confirm CI green post-merge',
    'merge',
    'project',
    'claude-opus-4-6',
    'hopeitworks/agent-go-node:latest'
) ON CONFLICT (project_id, name) DO UPDATE SET
    template_content = EXCLUDED.template_content,
    model = EXCLUDED.model;

-- ---------------------------------------------------------------------------
-- Pipeline config (groups + preconfigured steps)
-- ---------------------------------------------------------------------------

INSERT INTO pipeline_configs (id, project_id, config_yaml, version)
VALUES (
    '00000000-0000-0000-0000-000000000401',
    '00000000-0000-0000-0000-000000000101',
    'groups:
  - id: setup
    name: Setup
    steps:
      - id: 10000000-0000-0000-0000-000000000001
        name: Create Branch
        action_type: git_branch
        auto_approve: true
        config:
          base_branch: main
        retry_policy:
          max_retries: 0
          retry_type: none
  - id: development
    name: Development
    steps:
      - id: 10000000-0000-0000-0000-000000000002
        name: Implement Story
        action_type: agent_run
        model: claude-sonnet-4-6
        auto_approve: false
        config:
          role: dev
          phase: dev-story
        retry_policy:
          max_retries: 2
          retry_type: on-failure
  - id: review-merge
    name: Review & Merge
    steps:
      - id: 10000000-0000-0000-0000-000000000003
        name: Code Review
        action_type: agent_run
        model: claude-sonnet-4-6
        auto_approve: true
        config:
          role: review
          phase: code-review
        retry_policy:
          max_retries: 1
          retry_type: on-failure
      - id: 10000000-0000-0000-0000-000000000004
        name: Approval Gate
        action_type: human
        auto_approve: false
        config:
          message: "Review the changes and approve to proceed with merge"
        retry_policy:
          max_retries: 0
          retry_type: none
      - id: 10000000-0000-0000-0000-000000000005
        name: Create PR
        action_type: git_pr
        auto_approve: true
        config:
          target_branch: main
          strategy: squash
        retry_policy:
          max_retries: 1
          retry_type: on-failure
  - id: delivery
    name: Delivery
    steps:
      - id: 10000000-0000-0000-0000-000000000006
        name: Wait for CI
        action_type: ci_poll
        auto_approve: true
        config:
          timeout_minutes: "30"
        retry_policy:
          max_retries: 0
          retry_type: none
      - id: 10000000-0000-0000-0000-000000000007
        name: Notify Completion
        action_type: notification
        auto_approve: true
        config:
          channel: default
          message: "Story {story_key} pipeline completed successfully"
        retry_policy:
          max_retries: 0
          retry_type: none
',
    1
) ON CONFLICT (project_id) DO UPDATE SET
    config_yaml = EXCLUDED.config_yaml,
    version = pipeline_configs.version + 1;

-- ---------------------------------------------------------------------------
-- Runs (1 completed, 1 running, 1 pending — for UI testing)
-- ---------------------------------------------------------------------------

-- Run 1: completed (S-01 scaffolding — done)
INSERT INTO runs (id, project_id, story_id, status, started_at, completed_at)
VALUES (
    '00000000-0000-0000-0000-000000000501',
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000301',
    'completed',
    now() - interval '2 days',
    now() - interval '2 days' + interval '14 minutes'
) ON CONFLICT DO NOTHING;

-- Run 2: running (S-02 CI setup — in progress)
INSERT INTO runs (id, project_id, story_id, status, started_at)
VALUES (
    '00000000-0000-0000-0000-000000000502',
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000302',
    'running',
    now() - interval '8 minutes'
) ON CONFLICT DO NOTHING;

-- Run 3: pending (S-03 linting — queued)
INSERT INTO runs (id, project_id, story_id, status)
VALUES (
    '00000000-0000-0000-0000-000000000503',
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000303',
    'pending'
) ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- Run steps — Run 1 (completed, all 7 steps done)
-- ---------------------------------------------------------------------------

INSERT INTO run_steps (id, run_id, step_name, step_order, action, status, started_at, completed_at) VALUES
('00000000-0000-0000-0000-000000000601', '00000000-0000-0000-0000-000000000501', 'Create Branch',      1, 'git_branch',    'completed', now() - interval '2 days',                          now() - interval '2 days' + interval '12 seconds'),
('00000000-0000-0000-0000-000000000602', '00000000-0000-0000-0000-000000000501', 'Implement Story',    2, 'agent_run',     'completed', now() - interval '2 days' + interval '15 seconds',   now() - interval '2 days' + interval '8 minutes'),
('00000000-0000-0000-0000-000000000603', '00000000-0000-0000-0000-000000000501', 'Code Review',        3, 'agent_run',     'completed', now() - interval '2 days' + interval '8 minutes',    now() - interval '2 days' + interval '10 minutes'),
('00000000-0000-0000-0000-000000000604', '00000000-0000-0000-0000-000000000501', 'Approval Gate',      4, 'hitl_gate',     'completed', now() - interval '2 days' + interval '10 minutes',   now() - interval '2 days' + interval '11 minutes'),
('00000000-0000-0000-0000-000000000605', '00000000-0000-0000-0000-000000000501', 'Create PR',          5, 'git_pr',        'completed', now() - interval '2 days' + interval '11 minutes',   now() - interval '2 days' + interval '11 minutes' + interval '20 seconds'),
('00000000-0000-0000-0000-000000000606', '00000000-0000-0000-0000-000000000501', 'Wait for CI',        6, 'ci_poll',       'completed', now() - interval '2 days' + interval '11 minutes' + interval '25 seconds', now() - interval '2 days' + interval '13 minutes'),
('00000000-0000-0000-0000-000000000607', '00000000-0000-0000-0000-000000000501', 'Notify Completion',  7, 'notification',  'completed', now() - interval '2 days' + interval '13 minutes',   now() - interval '2 days' + interval '14 minutes')
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- Run steps — Run 2 (running, dev agent in progress)
-- ---------------------------------------------------------------------------

INSERT INTO run_steps (id, run_id, step_name, step_order, action, status, started_at, completed_at) VALUES
('00000000-0000-0000-0000-000000000611', '00000000-0000-0000-0000-000000000502', 'Create Branch', 1, 'git_branch', 'completed', now() - interval '8 minutes', now() - interval '7 minutes' - interval '45 seconds')
ON CONFLICT DO NOTHING;

INSERT INTO run_steps (id, run_id, step_name, step_order, action, status, started_at, container_id, log_tail)
VALUES (
    '00000000-0000-0000-0000-000000000612',
    '00000000-0000-0000-0000-000000000502',
    'Implement Story',
    2,
    'agent_run',
    'running',
    now() - interval '7 minutes',
    'abc123def456',
    E'[14:32:01] Creating .github/workflows/ci.yml\n[14:32:05] Adding Go lint job\n[14:32:08] Adding frontend build job\n[14:32:12] Running initial test...'
) ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- HITL requests
-- ---------------------------------------------------------------------------

-- Run 1: approved approval gate
INSERT INTO hitl_requests (id, run_step_id, gate_type, diff_content, status, resolved_at, resolved_by)
VALUES (
    '00000000-0000-0000-0000-000000000701',
    '00000000-0000-0000-0000-000000000604',
    'approval',
    E'diff --git a/main.go b/main.go\n+++ b/main.go\n@@ -0,0 +1,25 @@\n+package main\n+\n+import (\n+\t"net/http"\n+\t"github.com/go-chi/chi/v5"\n+)\n+\n+func main() {\n+\tr := chi.NewRouter()\n+\tr.Get("/health", func(w http.ResponseWriter, r *http.Request) {\n+\t\tw.Write([]byte("ok"))\n+\t})\n+\thttp.ListenAndServe(":8080", r)\n+}',
    'approved',
    now() - interval '2 days' + interval '11 minutes',
    '00000000-0000-0000-0000-000000000001'
) ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- Cost records (Run 1 steps that used agents)
-- ---------------------------------------------------------------------------

-- Run 1: Implement Story (Sonnet dev)
INSERT INTO cost_records (id, run_step_id, project_id, tokens_input, tokens_output, cost_usd, model, agent_id)
VALUES (
    '00000000-0000-0000-0000-000000001001',
    '00000000-0000-0000-0000-000000000602',
    '00000000-0000-0000-0000-000000000101',
    128000, 16500, 0.631500,
    'claude-sonnet-4-6',
    '00000000-0000-0000-0000-000000000903'
) ON CONFLICT DO NOTHING;

-- Run 1: Code Review (Sonnet review)
INSERT INTO cost_records (id, run_step_id, project_id, tokens_input, tokens_output, cost_usd, model, agent_id)
VALUES (
    '00000000-0000-0000-0000-000000001002',
    '00000000-0000-0000-0000-000000000603',
    '00000000-0000-0000-0000-000000000101',
    42000, 3800, 0.183000,
    'claude-sonnet-4-6',
    '00000000-0000-0000-0000-000000000902'
) ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- Events
-- ---------------------------------------------------------------------------

INSERT INTO events (id, project_id, entity_type, entity_id, action, payload) VALUES
('00000000-0000-0000-0000-000000000801', '00000000-0000-0000-0000-000000000101', 'run', '00000000-0000-0000-0000-000000000501', 'started',   '{"story_key": "S-01"}'),
('00000000-0000-0000-0000-000000000802', '00000000-0000-0000-0000-000000000101', 'run', '00000000-0000-0000-0000-000000000501', 'completed', '{"story_key": "S-01", "duration_ms": 840000}'),
('00000000-0000-0000-0000-000000000803', '00000000-0000-0000-0000-000000000101', 'run', '00000000-0000-0000-0000-000000000502', 'started',   '{"story_key": "S-02"}'),
('00000000-0000-0000-0000-000000000804', '00000000-0000-0000-0000-000000000101', 'hitl', '00000000-0000-0000-0000-000000000701', 'approved', '{"resolved_by": "00000000-0000-0000-0000-000000000001", "run_id": "00000000-0000-0000-0000-000000000501"}')
ON CONFLICT DO NOTHING;

COMMIT;
