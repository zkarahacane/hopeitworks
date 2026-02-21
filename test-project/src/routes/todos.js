const express = require('express');
const { getDb } = require('../db');

const router = express.Router();

/**
 * GET /api/todos - List all todos
 */
router.get('/', (_req, res) => {
  const db = getDb();
  const todos = db.prepare('SELECT * FROM todos ORDER BY created_at DESC').all();
  res.json({ data: todos });
});

/**
 * GET /api/todos/:id - Get a single todo by ID
 */
router.get('/:id', (req, res) => {
  const db = getDb();
  const todo = db.prepare('SELECT * FROM todos WHERE id = ?').get(req.params.id);
  if (!todo) {
    return res.status(404).json({ error: { code: 'TODO_NOT_FOUND', message: `Todo ${req.params.id} not found` } });
  }
  res.json(todo);
});

/**
 * POST /api/todos - Create a new todo
 */
router.post('/', (req, res) => {
  const { title } = req.body;
  if (!title || typeof title !== 'string' || title.trim().length === 0) {
    return res.status(400).json({ error: { code: 'INVALID_INPUT', message: 'Title is required and must be a non-empty string' } });
  }

  const db = getDb();
  const result = db.prepare('INSERT INTO todos (title) VALUES (?)').run(title.trim());
  const todo = db.prepare('SELECT * FROM todos WHERE id = ?').get(result.lastInsertRowid);
  res.status(201).json(todo);
});

/**
 * PUT /api/todos/:id - Update an existing todo
 */
router.put('/:id', (req, res) => {
  const db = getDb();
  const existing = db.prepare('SELECT * FROM todos WHERE id = ?').get(req.params.id);
  if (!existing) {
    return res.status(404).json({ error: { code: 'TODO_NOT_FOUND', message: `Todo ${req.params.id} not found` } });
  }

  const { title, completed } = req.body;
  const updatedTitle = title !== undefined ? title : existing.title;
  const updatedCompleted = completed !== undefined ? (completed ? 1 : 0) : existing.completed;

  if (title !== undefined && (typeof title !== 'string' || title.trim().length === 0)) {
    return res.status(400).json({ error: { code: 'INVALID_INPUT', message: 'Title must be a non-empty string' } });
  }

  db.prepare("UPDATE todos SET title = ?, completed = ?, updated_at = datetime('now') WHERE id = ?")
    .run(updatedTitle, updatedCompleted, req.params.id);

  const todo = db.prepare('SELECT * FROM todos WHERE id = ?').get(req.params.id);
  res.json(todo);
});

/**
 * DELETE /api/todos/:id - Delete a todo
 */
router.delete('/:id', (req, res) => {
  const db = getDb();
  const existing = db.prepare('SELECT * FROM todos WHERE id = ?').get(req.params.id);
  if (!existing) {
    return res.status(404).json({ error: { code: 'TODO_NOT_FOUND', message: `Todo ${req.params.id} not found` } });
  }

  db.prepare('DELETE FROM todos WHERE id = ?').run(req.params.id);
  res.status(204).send();
});

module.exports = router;
