-- =============================================================================
-- hopeitworks dev seed data
-- =============================================================================
-- Purpose: Pre-populate local dev database with test users and projects.
-- Run:     cd backend && make seed
-- Reset:   cd backend && make reset-db
--
-- Credentials:
--   admin@hopeitworks.dev / admin123  (role: admin)
--   dev@hopeitworks.dev   / dev123    (role: user)
--   alice@hopeitworks.dev / alice123  (role: user)
--
-- Idempotent: safe to run multiple times (uses ON CONFLICT).
-- =============================================================================

BEGIN;

-- ---------------------------------------------------------------------------
-- Users
-- ---------------------------------------------------------------------------

INSERT INTO users (id, email, password_hash, name, role)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin@hopeitworks.dev',
    crypt('admin123', gen_salt('bf', 10)),
    'Admin User',
    'admin'
) ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    role = EXCLUDED.role;

INSERT INTO users (id, email, password_hash, name, role)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    'dev@hopeitworks.dev',
    crypt('dev123', gen_salt('bf', 10)),
    'Dev User',
    'user'
) ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    role = EXCLUDED.role;

INSERT INTO users (id, email, password_hash, name, role)
VALUES (
    '00000000-0000-0000-0000-000000000003',
    'alice@hopeitworks.dev',
    crypt('alice123', gen_salt('bf', 10)),
    'Alice Developer',
    'user'
) ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    role = EXCLUDED.role;

-- ---------------------------------------------------------------------------
-- Projects
-- ---------------------------------------------------------------------------

INSERT INTO projects (id, name, description, owner_id, repo_url, default_model)
VALUES (
    '00000000-0000-0000-0000-000000000101',
    'Todo App',
    'Reference todo application for pipeline validation and baseline testing',
    '00000000-0000-0000-0000-000000000001',
    'https://github.com/hopeitworks/todo-app',
    'claude-sonnet-4-20250514'
) ON CONFLICT (name) DO UPDATE SET
    description = EXCLUDED.description,
    owner_id = EXCLUDED.owner_id,
    repo_url = EXCLUDED.repo_url,
    default_model = EXCLUDED.default_model;

INSERT INTO projects (id, name, description, owner_id, repo_url, default_model)
VALUES (
    '00000000-0000-0000-0000-000000000102',
    'E-commerce API',
    'Sample e-commerce REST API for multi-project testing scenarios',
    '00000000-0000-0000-0000-000000000001',
    'https://github.com/hopeitworks/ecommerce-api',
    'claude-sonnet-4-20250514'
) ON CONFLICT (name) DO UPDATE SET
    description = EXCLUDED.description,
    owner_id = EXCLUDED.owner_id,
    repo_url = EXCLUDED.repo_url,
    default_model = EXCLUDED.default_model;

INSERT INTO projects (id, name, description, owner_id, repo_url, default_model)
VALUES (
    '00000000-0000-0000-0000-000000000103',
    'Frontend Kit',
    'Vue 3 component library project owned by dev user',
    '00000000-0000-0000-0000-000000000002',
    'https://github.com/hopeitworks/frontend-kit',
    'claude-sonnet-4-20250514'
) ON CONFLICT (name) DO UPDATE SET
    description = EXCLUDED.description,
    owner_id = EXCLUDED.owner_id,
    repo_url = EXCLUDED.repo_url,
    default_model = EXCLUDED.default_model;

-- ---------------------------------------------------------------------------
-- Project memberships (conditional: only if project_users table exists)
-- ---------------------------------------------------------------------------

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'project_users'
    ) THEN
        -- Admin owns Todo App and E-commerce API
        INSERT INTO project_users (project_id, user_id, role)
        VALUES ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000001', 'owner')
        ON CONFLICT DO NOTHING;

        INSERT INTO project_users (project_id, user_id, role)
        VALUES ('00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000001', 'owner')
        ON CONFLICT DO NOTHING;

        -- Dev user is member of Todo App
        INSERT INTO project_users (project_id, user_id, role)
        VALUES ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000002', 'member')
        ON CONFLICT DO NOTHING;

        -- Alice is member of E-commerce API
        INSERT INTO project_users (project_id, user_id, role)
        VALUES ('00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000003', 'member')
        ON CONFLICT DO NOTHING;

        -- Dev user owns Frontend Kit
        INSERT INTO project_users (project_id, user_id, role)
        VALUES ('00000000-0000-0000-0000-000000000103', '00000000-0000-0000-0000-000000000002', 'owner')
        ON CONFLICT DO NOTHING;

        RAISE NOTICE 'Seed: project_users memberships inserted';
    ELSE
        RAISE NOTICE 'Seed: project_users table not found (Story 1-6 not applied), skipping memberships';
    END IF;
END $$;

COMMIT;
