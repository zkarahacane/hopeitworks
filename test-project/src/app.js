const express = require('express');
const path = require('path');
const todosRouter = require('./routes/todos');

const app = express();
const PORT = process.env.PORT || 3000;

app.use(express.json());
app.use(express.static(path.join(__dirname, 'public')));

app.use('/api/todos', todosRouter);

/**
 * GET /health - Liveness check
 */
app.get('/health', (_req, res) => {
  res.json({ status: 'ok' });
});

if (require.main === module) {
  app.listen(PORT, () => {
    process.stdout.write(`Todo app listening on port ${PORT}\n`);
  });
}

module.exports = app;
