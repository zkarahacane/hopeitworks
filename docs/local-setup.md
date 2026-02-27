# Local Setup Guide

This guide covers setting up the **hopeitworks** stack on your local machine (not the devcontainer).

## Prerequisites

- Docker Desktop (or compatible: OrbStack, Colima, Rancher Desktop with dockerd)
- Docker Compose v2+
- Go 1.23+
- Node.js 20+ and npm
- `gh` CLI (for GitHub integration)

## /etc/hosts

Add the following entry so the frontend can reach the API and Gitea:

```
127.0.0.1  localhost
```

If you use a custom hostname for the local Gitea instance:

```
127.0.0.1  gitea.local
```

## Start the Stack

```bash
cd deploy
docker compose up -d
```

This starts 4 services:

| Service | Port | URL |
|---------|------|-----|
| API | 8080 | http://localhost:8080 |
| Postgres | 5432 | `postgres://hopeitworks:hopeitworks_dev_password@localhost:5432/hopeitworks_dev` |
| MailHog (SMTP) | 1025 | — |
| MailHog (UI) | 8025 | http://localhost:8025 |
| Socket Proxy | 2375 (internal) | — |

All services use `restart: unless-stopped` so they survive Docker/machine restarts.

## First Seed

After starting the stack for the first time, seed the database:

```bash
./scripts/reset-dev.sh
```

This creates test users, a project, stories, agents, and a pipeline config. See the script header for credentials.

Default login: `admin@hopeitworks.dev` / `admin1234`

## Frontend Dev Server

The frontend runs separately (not in docker-compose):

```bash
cd frontend
npm install
npm run dev
```

Access at http://localhost:5173

## Data Persistence

Postgres data is stored in a Docker named volume (`postgres_data`). It persists across `docker compose down` and machine restarts. Only `docker compose down -v` destroys the volume.

To reset data without destroying the volume:

```bash
./scripts/reset-dev.sh
```

## Relationship with Devcontainer

The devcontainer and the local machine share the same Docker engine (the socket is bind-mounted into the devcontainer). To prevent conflicts:

- **Devcontainer** = code-only (lint, tests, code generation)
- **Local machine** = stable stack (docker-compose, reset-dev, e2e tests)

The devcontainer sets `HOPEITWORKS_ENV=devcontainer`, which blocks docker-related scripts. See the "Development Environments" section in `CLAUDE.md` for details.

## Stopping the Stack

```bash
cd deploy
docker compose down       # stop services, keep data
docker compose down -v    # stop services AND destroy data
```
