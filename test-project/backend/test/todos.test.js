const { describe, it, before, after } = require("node:test");
const assert = require("node:assert/strict");
const http = require("node:http");
const createApp = require("../app");

const mockRows = [
  {
    id: 1,
    title: "Buy groceries",
    completed: false,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
];

const mockPool = {
  query: async (sql, params) => {
    if (sql.startsWith("SELECT * FROM todos ORDER")) {
      return { rows: mockRows };
    }
    if (sql.startsWith("SELECT * FROM todos WHERE")) {
      const id = params[0];
      const row = mockRows.find((r) => r.id === Number(id));
      return { rows: row ? [row] : [] };
    }
    if (sql.startsWith("INSERT")) {
      return {
        rows: [{ id: 2, title: params[0], completed: false }],
      };
    }
    if (sql.startsWith("UPDATE")) {
      return {
        rows: [
          {
            id: Number(params[2]),
            title: params[0],
            completed: params[1],
          },
        ],
      };
    }
    if (sql.startsWith("DELETE")) {
      const id = params[0];
      const row = mockRows.find((r) => r.id === Number(id));
      return { rows: row ? [row] : [] };
    }
    return { rows: [] };
  },
};

function makeRequest(server, method, path, body) {
  return new Promise((resolve, reject) => {
    const addr = server.address();
    const options = {
      hostname: "127.0.0.1",
      port: addr.port,
      path,
      method,
      headers: { "Content-Type": "application/json" },
    };
    const req = http.request(options, (res) => {
      let data = "";
      res.on("data", (chunk) => (data += chunk));
      res.on("end", () => {
        resolve({
          status: res.statusCode,
          body: data ? JSON.parse(data) : null,
        });
      });
    });
    req.on("error", reject);
    if (body) req.write(JSON.stringify(body));
    req.end();
  });
}

describe("Todo API", () => {
  let server;

  before(() => {
    const app = createApp(mockPool);
    server = app.listen(0);
  });

  after(() => {
    server.close();
  });

  it("GET /health returns ok", async () => {
    const res = await makeRequest(server, "GET", "/health");
    assert.equal(res.status, 200);
    assert.equal(res.body.status, "ok");
  });

  it("GET /api/todos returns list", async () => {
    const res = await makeRequest(server, "GET", "/api/todos");
    assert.equal(res.status, 200);
    assert.ok(Array.isArray(res.body));
    assert.equal(res.body.length, 1);
  });

  it("POST /api/todos creates a todo", async () => {
    const res = await makeRequest(server, "POST", "/api/todos", {
      title: "Test todo",
    });
    assert.equal(res.status, 201);
    assert.equal(res.body.title, "Test todo");
  });

  it("POST /api/todos rejects empty title", async () => {
    const res = await makeRequest(server, "POST", "/api/todos", { title: "" });
    assert.equal(res.status, 400);
    assert.equal(res.body.error.code, "VALIDATION_ERROR");
  });

  it("POST /api/todos rejects missing title", async () => {
    const res = await makeRequest(server, "POST", "/api/todos", {});
    assert.equal(res.status, 400);
    assert.equal(res.body.error.code, "VALIDATION_ERROR");
  });

  it("GET /api/todos/:id returns a todo", async () => {
    const res = await makeRequest(server, "GET", "/api/todos/1");
    assert.equal(res.status, 200);
    assert.equal(res.body.id, 1);
    assert.equal(res.body.title, "Buy groceries");
  });

  it("GET /api/todos/:id returns 404 for missing", async () => {
    const res = await makeRequest(server, "GET", "/api/todos/999");
    assert.equal(res.status, 404);
    assert.equal(res.body.error.code, "NOT_FOUND");
  });

  it("PUT /api/todos/:id updates a todo", async () => {
    const res = await makeRequest(server, "PUT", "/api/todos/1", {
      completed: true,
    });
    assert.equal(res.status, 200);
    assert.equal(res.body.completed, true);
  });

  it("PUT /api/todos/:id returns 404 for missing", async () => {
    const res = await makeRequest(server, "PUT", "/api/todos/999", {
      completed: true,
    });
    assert.equal(res.status, 404);
    assert.equal(res.body.error.code, "NOT_FOUND");
  });

  it("DELETE /api/todos/:id deletes a todo", async () => {
    const res = await makeRequest(server, "DELETE", "/api/todos/1");
    assert.equal(res.status, 204);
  });

  it("DELETE /api/todos/:id returns 404 for missing", async () => {
    const res = await makeRequest(server, "DELETE", "/api/todos/999");
    assert.equal(res.status, 404);
    assert.equal(res.body.error.code, "NOT_FOUND");
  });
});
