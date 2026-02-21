const express = require("express");
const cors = require("cors");

/** Creates the Express app with the given database pool. */
function createApp(pool) {
  const app = express();
  app.use(cors());
  app.use(express.json());

  // Health check
  app.get("/health", (_req, res) => {
    res.json({ status: "ok" });
  });

  // List all todos
  app.get("/api/todos", async (_req, res) => {
    try {
      const result = await pool.query(
        "SELECT * FROM todos ORDER BY created_at DESC"
      );
      res.json(result.rows);
    } catch (err) {
      res
        .status(500)
        .json({ error: { code: "DB_ERROR", message: err.message } });
    }
  });

  // Get a single todo
  app.get("/api/todos/:id", async (req, res) => {
    try {
      const result = await pool.query("SELECT * FROM todos WHERE id = $1", [
        req.params.id,
      ]);
      if (result.rows.length === 0) {
        return res
          .status(404)
          .json({ error: { code: "NOT_FOUND", message: "Todo not found" } });
      }
      res.json(result.rows[0]);
    } catch (err) {
      res
        .status(500)
        .json({ error: { code: "DB_ERROR", message: err.message } });
    }
  });

  // Create a todo
  app.post("/api/todos", async (req, res) => {
    const { title } = req.body;
    if (!title || typeof title !== "string" || title.trim() === "") {
      return res.status(400).json({
        error: { code: "VALIDATION_ERROR", message: "title is required" },
      });
    }
    try {
      const result = await pool.query(
        "INSERT INTO todos (title) VALUES ($1) RETURNING *",
        [title.trim()]
      );
      res.status(201).json(result.rows[0]);
    } catch (err) {
      res
        .status(500)
        .json({ error: { code: "DB_ERROR", message: err.message } });
    }
  });

  // Update a todo
  app.put("/api/todos/:id", async (req, res) => {
    const { title, completed } = req.body;
    try {
      const existing = await pool.query("SELECT * FROM todos WHERE id = $1", [
        req.params.id,
      ]);
      if (existing.rows.length === 0) {
        return res
          .status(404)
          .json({ error: { code: "NOT_FOUND", message: "Todo not found" } });
      }

      const updatedTitle =
        title !== undefined ? title.trim() : existing.rows[0].title;
      const updatedCompleted =
        completed !== undefined ? completed : existing.rows[0].completed;

      const result = await pool.query(
        "UPDATE todos SET title = $1, completed = $2, updated_at = NOW() WHERE id = $3 RETURNING *",
        [updatedTitle, updatedCompleted, req.params.id]
      );
      res.json(result.rows[0]);
    } catch (err) {
      res
        .status(500)
        .json({ error: { code: "DB_ERROR", message: err.message } });
    }
  });

  // Delete a todo
  app.delete("/api/todos/:id", async (req, res) => {
    try {
      const result = await pool.query(
        "DELETE FROM todos WHERE id = $1 RETURNING *",
        [req.params.id]
      );
      if (result.rows.length === 0) {
        return res
          .status(404)
          .json({ error: { code: "NOT_FOUND", message: "Todo not found" } });
      }
      res.status(204).send();
    } catch (err) {
      res
        .status(500)
        .json({ error: { code: "DB_ERROR", message: err.message } });
    }
  });

  return app;
}

module.exports = createApp;
