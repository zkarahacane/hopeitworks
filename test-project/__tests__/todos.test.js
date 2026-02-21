// Set DB_PATH to in-memory before requiring modules
process.env.DB_PATH = ':memory:';

const request = require('supertest');
const app = require('../src/app');
const { closeDb } = require('../src/db');

beforeEach(() => {
  closeDb();
});

afterEach(() => {
  closeDb();
});

describe('GET /health', () => {
  it('returns ok status', async () => {
    const res = await request(app).get('/health');
    expect(res.status).toBe(200);
    expect(res.body.status).toBe('ok');
  });
});

describe('GET /api/todos', () => {
  it('returns empty list initially', async () => {
    const res = await request(app).get('/api/todos');
    expect(res.status).toBe(200);
    expect(res.body.data).toEqual([]);
  });

  it('returns todos after creation', async () => {
    await request(app).post('/api/todos').send({ title: 'Test todo' });
    const res = await request(app).get('/api/todos');
    expect(res.status).toBe(200);
    expect(res.body.data).toHaveLength(1);
    expect(res.body.data[0].title).toBe('Test todo');
  });
});

describe('GET /api/todos/:id', () => {
  it('returns a single todo', async () => {
    const created = await request(app).post('/api/todos').send({ title: 'Find me' });
    const res = await request(app).get(`/api/todos/${created.body.id}`);
    expect(res.status).toBe(200);
    expect(res.body.title).toBe('Find me');
  });

  it('returns 404 for non-existent todo', async () => {
    const res = await request(app).get('/api/todos/999');
    expect(res.status).toBe(404);
    expect(res.body.error.code).toBe('TODO_NOT_FOUND');
  });
});

describe('POST /api/todos', () => {
  it('creates a new todo', async () => {
    const res = await request(app).post('/api/todos').send({ title: 'New todo' });
    expect(res.status).toBe(201);
    expect(res.body.title).toBe('New todo');
    expect(res.body.completed).toBe(0);
    expect(res.body.id).toBeDefined();
  });

  it('trims whitespace from title', async () => {
    const res = await request(app).post('/api/todos').send({ title: '  Trimmed  ' });
    expect(res.status).toBe(201);
    expect(res.body.title).toBe('Trimmed');
  });

  it('rejects empty title', async () => {
    const res = await request(app).post('/api/todos').send({ title: '' });
    expect(res.status).toBe(400);
    expect(res.body.error.code).toBe('INVALID_INPUT');
  });

  it('rejects missing title', async () => {
    const res = await request(app).post('/api/todos').send({});
    expect(res.status).toBe(400);
    expect(res.body.error.code).toBe('INVALID_INPUT');
  });

  it('rejects whitespace-only title', async () => {
    const res = await request(app).post('/api/todos').send({ title: '   ' });
    expect(res.status).toBe(400);
    expect(res.body.error.code).toBe('INVALID_INPUT');
  });
});

describe('PUT /api/todos/:id', () => {
  it('updates todo title', async () => {
    const created = await request(app).post('/api/todos').send({ title: 'Original' });
    const res = await request(app).put(`/api/todos/${created.body.id}`).send({ title: 'Updated' });
    expect(res.status).toBe(200);
    expect(res.body.title).toBe('Updated');
  });

  it('marks todo as completed', async () => {
    const created = await request(app).post('/api/todos').send({ title: 'Complete me' });
    const res = await request(app).put(`/api/todos/${created.body.id}`).send({ completed: true });
    expect(res.status).toBe(200);
    expect(res.body.completed).toBe(1);
  });

  it('marks todo as not completed', async () => {
    const created = await request(app).post('/api/todos').send({ title: 'Uncomplete me' });
    await request(app).put(`/api/todos/${created.body.id}`).send({ completed: true });
    const res = await request(app).put(`/api/todos/${created.body.id}`).send({ completed: false });
    expect(res.status).toBe(200);
    expect(res.body.completed).toBe(0);
  });

  it('returns 404 for non-existent todo', async () => {
    const res = await request(app).put('/api/todos/999').send({ title: 'Nope' });
    expect(res.status).toBe(404);
  });

  it('rejects empty title', async () => {
    const created = await request(app).post('/api/todos').send({ title: 'Valid' });
    const res = await request(app).put(`/api/todos/${created.body.id}`).send({ title: '' });
    expect(res.status).toBe(400);
    expect(res.body.error.code).toBe('INVALID_INPUT');
  });
});

describe('DELETE /api/todos/:id', () => {
  it('deletes a todo', async () => {
    const created = await request(app).post('/api/todos').send({ title: 'Delete me' });
    const res = await request(app).delete(`/api/todos/${created.body.id}`);
    expect(res.status).toBe(204);

    const check = await request(app).get(`/api/todos/${created.body.id}`);
    expect(check.status).toBe(404);
  });

  it('returns 404 for non-existent todo', async () => {
    const res = await request(app).delete('/api/todos/999');
    expect(res.status).toBe(404);
  });
});
