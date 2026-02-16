# CLAUDE.md Templates — Composition Rules

This directory contains modular CLAUDE.md templates that are composed at runtime and injected into agent containers. Each agent receives a single `CLAUDE.md` file assembled from these templates.

## Directory Structure

```
agent/claude-md/
├── README.md          # This file — composition rules
├── base.md            # Common: git, commits, quality, testing principles
├── backend.md         # Go: hexagonal, chi, sqlc, DomainError, slog, testing
├── frontend.md        # Vue: Composition API, PrimeVue, Tailwind, Pinia, testing
└── project.md         # Current state: phase, key paths, openapi.yaml, status
```

## Composition Rule

Each agent receives: **base + (backend OR frontend) + project**

| Agent Type | Composed From |
|------------|---------------|
| Backend agent | `base.md` + `backend.md` + `project.md` |
| Frontend agent | `base.md` + `frontend.md` + `project.md` |

## Composition Logic

The `agent/scripts/inject-claude-md.sh` script concatenates the appropriate template files into a single `CLAUDE.md` placed at the repository root inside the agent container.

```bash
# Backend agent composition
cat base.md backend.md project.md > /repo/CLAUDE.md

# Frontend agent composition
cat base.md frontend.md project.md > /repo/CLAUDE.md
```

## Scoping Rules

- **Backend agents** MUST NEVER touch the `frontend/` directory
- **Frontend agents** MUST NEVER touch the `backend/` directory
- Both agents can reference `api/openapi.yaml` (read-only; coordinated changes only)
- Each template section is self-contained — agents may not have access to the architecture document at runtime

## Editing Guidelines

- Each file must be self-contained (no cross-references between template files)
- Include exact library versions where critical (e.g., "pgx/v5", "PrimeVue 4", "Vue 3")
- Include exact commands (e.g., `make generate`, `npm run generate-api`)
- Include folder structure conventions explicitly
- Include testing commands and patterns explicitly
- All patterns must match the architecture document (`_bmad-output/planning-artifacts/architecture.md`)
