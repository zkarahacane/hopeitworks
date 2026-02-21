-- =============================================================================
-- Todo App seed data
-- =============================================================================
-- Purpose: Pre-populate the todo app database with sample todos for testing.
-- Run:     npm run seed
-- Reset:   rm -f todos.db && npm run seed
--
-- Idempotent: Uses INSERT OR REPLACE for safe re-runs.
-- =============================================================================

CREATE TABLE IF NOT EXISTS todos (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  completed INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

INSERT OR REPLACE INTO todos (id, title, completed, created_at, updated_at) VALUES
  (1, 'Buy groceries', 0, '2026-02-01 10:00:00', '2026-02-01 10:00:00'),
  (2, 'Write unit tests', 1, '2026-02-02 09:00:00', '2026-02-02 14:30:00'),
  (3, 'Review pull request', 0, '2026-02-03 11:00:00', '2026-02-03 11:00:00'),
  (4, 'Deploy to staging', 0, '2026-02-04 08:30:00', '2026-02-04 08:30:00'),
  (5, 'Update documentation', 1, '2026-02-05 13:00:00', '2026-02-05 16:45:00'),
  (6, 'Fix login bug', 0, '2026-02-06 10:15:00', '2026-02-06 10:15:00'),
  (7, 'Set up CI pipeline', 1, '2026-02-07 09:00:00', '2026-02-07 12:00:00'),
  (8, 'Refactor database layer', 0, '2026-02-08 14:00:00', '2026-02-08 14:00:00');
