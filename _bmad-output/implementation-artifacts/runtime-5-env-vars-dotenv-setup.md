# Story runtime-5: Environment Variables and .env Setup

**Status:** ready-for-dev
**Branch:** `feat/runtime-5-env-vars-dotenv`
**Commit scope:** `deploy`

---

## Story

As a developer setting up hopeitworks for the first time, I need a documented `.env.example` file and a properly configured Docker Compose stack that forwards all required environment variables — so that agent pipeline execution works out of the box without hunting through source code for undocumented secrets.

---

## Acceptance Criteria

**AC #1 — .env.example exists with all required variables documented**
- Given a new developer clones the repo
- When they look at `deploy/.env.example`
- Then all env vars needed for full pipeline execution are listed with descriptions
- And sensitive vars have placeholder values (e.g., `your-github-token-here`)
- And non-sensitive vars have sensible defaults

**AC #2 — docker-compose forwards agent-critical env vars**
- Given `deploy/.env` contains `GITHUB_TOKEN=xxx` and `CLAUDE_CODE_OAUTH_TOKEN=yyy`
- When `docker compose up` starts the api container
- Then the api container has `GITHUB_TOKEN=xxx` and `CLAUDE_CODE_OAUTH_TOKEN=yyy` in its environment
- And when the api spawns an agent container, those tokens are passed through

**AC #3 — .env is gitignored**
- Given `deploy/.env` exists locally
- When `git status` is run
- Then `.env` does not appear as untracked

**AC #4 — JWT_SECRET is configurable**
- Given `JWT_SECRET=my-secure-secret` in `.env`
- When the api container starts
- Then it uses `my-secure-secret` instead of the hardcoded default

---

## Tasks / Subtasks

- [ ] **T1.** Audit all env var reads in the backend (AC: #1)
  - [ ] T1.1 Read `backend/cmd/api/main.go` — list all `getEnvOrDefault` and `os.Getenv` calls
  - [ ] T1.2 Read `backend/internal/adapter/action/agent_run.go` lines 191-192 — confirm token var names
  - [ ] T1.3 Read `backend/internal/config/loader.go` — check for additional env overrides
  - [ ] T1.4 Read `deploy/docker-compose.yml` — list all currently declared env vars

- [ ] **T2.** Create `deploy/.env.example` (AC: #1, #4)
  - [ ] T2.1 Group vars by category: Database, Auth, Docker, Agent, SMTP
  - [ ] T2.2 Add `GITHUB_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN` with placeholder values
  - [ ] T2.3 Add `JWT_SECRET` with placeholder and a comment warning about the insecure default
  - [ ] T2.4 Add `AGENT_IMAGE` with default `hopeitworks/agent:latest`
  - [ ] T2.5 Add `CLAUDE_MD_PATH` with default `agent/claude-md`
  - [ ] T2.6 Carry over all existing vars already in `docker-compose.yml` (Postgres, SMTP, etc.)

- [ ] **T3.** Update `deploy/docker-compose.yml` (AC: #2, #4)
  - [ ] T3.1 Add `GITHUB_TOKEN: ${GITHUB_TOKEN}` to the `api` service `environment:` section
  - [ ] T3.2 Add `CLAUDE_CODE_OAUTH_TOKEN: ${CLAUDE_CODE_OAUTH_TOKEN}` to the `api` service
  - [ ] T3.3 Add `JWT_SECRET: ${JWT_SECRET:-dev-secret-key-change-in-production}` to the `api` service
  - [ ] T3.4 Keep all existing env vars unchanged
  - [ ] T3.5 Add comments grouping env vars by category for readability

- [ ] **T4.** Update `.gitignore` (AC: #3)
  - [ ] T4.1 Check if `deploy/.env` or `*.env` is already gitignored
  - [ ] T4.2 If not, add `deploy/.env` entry to `.gitignore`

---

## Dev Notes

### Dependencies

- No code changes — deploy/config only
- `GITHUB_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` must be set in the host environment (or `deploy/.env`) before running `docker compose up`

### File Paths

| File | Action | Purpose |
|------|--------|---------|
| `deploy/.env.example` | CREATE | Documents all env vars with placeholders and descriptions |
| `deploy/docker-compose.yml` | MODIFY | Forward `GITHUB_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`, `JWT_SECRET` to api container |
| `.gitignore` | MODIFY (if needed) | Ensure `deploy/.env` is not committed |

### Technical Specifications

**deploy/.env.example structure:**

```dotenv
# =============================================================================
# hopeitworks — environment variables
# Copy this file to deploy/.env and fill in the values.
# NEVER commit deploy/.env to git.
# =============================================================================

# -----------------------------------------------------------------------------
# Database
# -----------------------------------------------------------------------------
POSTGRES_USER=hopeitworks
POSTGRES_PASSWORD=hopeitworks
POSTGRES_DB=hopeitworks

# -----------------------------------------------------------------------------
# Auth
# -----------------------------------------------------------------------------
# SECURITY: Change this before deploying. The default is insecure.
JWT_SECRET=your-jwt-secret-here

# -----------------------------------------------------------------------------
# Agent tokens (required for pipeline execution)
# -----------------------------------------------------------------------------
# GitHub personal access token or GitHub App installation token.
# Needs repo read/write scope for the target repository.
GITHUB_TOKEN=your-github-token-here

# Claude Code OAuth token — obtained from https://claude.ai/
CLAUDE_CODE_OAUTH_TOKEN=your-claude-code-oauth-token-here

# -----------------------------------------------------------------------------
# Docker / Agent runtime
# -----------------------------------------------------------------------------
# Docker image used for agent containers (must be built locally or pulled).
AGENT_IMAGE=hopeitworks/agent:latest

# Path to the CLAUDE.md template directory (relative to repo root inside container).
CLAUDE_MD_PATH=agent/claude-md

# -----------------------------------------------------------------------------
# SMTP (optional — for email notifications)
# -----------------------------------------------------------------------------
SMTP_HOST=mailhog
SMTP_PORT=1025
SMTP_FROM=noreply@hopeitworks.local
```

**docker-compose.yml api service environment additions:**

```yaml
environment:
  # --- existing vars (unchanged) ---
  DATABASE_URL: postgres://...
  # ...

  # --- auth ---
  JWT_SECRET: ${JWT_SECRET:-dev-secret-key-change-in-production}

  # --- agent tokens ---
  GITHUB_TOKEN: ${GITHUB_TOKEN}
  CLAUDE_CODE_OAUTH_TOKEN: ${CLAUDE_CODE_OAUTH_TOKEN}
```

**Why `${VAR}` vs `${VAR:-default}`:**

- `GITHUB_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN`: no default — if missing, `docker compose` will warn and the var will be empty, causing agent runs to fail at runtime (intentional — fail fast rather than silently skip auth)
- `JWT_SECRET`: use `${JWT_SECRET:-dev-secret-key-change-in-production}` to keep the existing behavior (insecure default for local dev) while allowing override

### Source Code References

| File | Line(s) | What it reads |
|------|---------|---------------|
| `backend/cmd/api/main.go` | ~97 | `JWT_SECRET` via `getEnvOrDefault` |
| `backend/cmd/api/main.go` | ~211 | `AGENT_IMAGE` via `getEnvOrDefault` |
| `backend/cmd/api/main.go` | ~216 | `CLAUDE_MD_PATH` via `getEnvOrDefault` |
| `backend/internal/adapter/action/agent_run.go` | 191 | `GITHUB_TOKEN` via `os.Getenv` |
| `backend/internal/adapter/action/agent_run.go` | 192 | `CLAUDE_CODE_OAUTH_TOKEN` via `os.Getenv` |

### Testing Requirements

- Run `docker compose --env-file deploy/.env.example config` and verify no warnings about missing required vars (after filling in dummy values)
- Verify `deploy/.env` appears in `.gitignore` output: `git check-ignore -v deploy/.env`
- Verify `docker compose up api` picks up `JWT_SECRET` by checking the api startup log for no "insecure default" warning (if such a log exists)

### References

- `backend/cmd/api/main.go` — all `getEnvOrDefault` calls
- `backend/internal/adapter/action/agent_run.go` — agent token reads
- `deploy/docker-compose.yml` — current state of env var forwarding
- `backend/internal/config/loader.go` — env override logic

---

## Change Log

- Created 2026-02-22
