const API_URL = window.API_URL || "/api";

const todoList = document.getElementById("todo-list");
const todoForm = document.getElementById("todo-form");
const todoInput = document.getElementById("todo-input");
const emptyMessage = document.getElementById("empty-message");

async function fetchTodos() {
  const res = await fetch(`${API_URL}/todos`);
  if (!res.ok) throw new Error("Failed to fetch todos");
  return res.json();
}

async function createTodo(title) {
  const res = await fetch(`${API_URL}/todos`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title }),
  });
  if (!res.ok) throw new Error("Failed to create todo");
  return res.json();
}

async function updateTodo(id, updates) {
  const res = await fetch(`${API_URL}/todos/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(updates),
  });
  if (!res.ok) throw new Error("Failed to update todo");
  return res.json();
}

async function deleteTodo(id) {
  const res = await fetch(`${API_URL}/todos/${id}`, {
    method: "DELETE",
  });
  if (!res.ok) throw new Error("Failed to delete todo");
}

function renderTodo(todo) {
  const li = document.createElement("li");
  li.className = `todo-item${todo.completed ? " completed" : ""}`;
  li.dataset.id = todo.id;

  const checkbox = document.createElement("input");
  checkbox.type = "checkbox";
  checkbox.checked = todo.completed;
  checkbox.addEventListener("change", async () => {
    await updateTodo(todo.id, { completed: checkbox.checked });
    li.classList.toggle("completed", checkbox.checked);
  });

  const title = document.createElement("span");
  title.className = "title";
  title.textContent = todo.title;

  const deleteBtn = document.createElement("button");
  deleteBtn.className = "delete-btn";
  deleteBtn.textContent = "\u00d7";
  deleteBtn.addEventListener("click", async () => {
    await deleteTodo(todo.id);
    li.remove();
    updateEmptyState();
  });

  li.append(checkbox, title, deleteBtn);
  return li;
}

function updateEmptyState() {
  const hasTodos = todoList.children.length > 0;
  emptyMessage.classList.toggle("hidden", hasTodos);
}

async function loadTodos() {
  try {
    const todos = await fetchTodos();
    todoList.innerHTML = "";
    todos.forEach((todo) => todoList.appendChild(renderTodo(todo)));
    updateEmptyState();
  } catch (err) {
    todoList.innerHTML =
      '<li class="todo-item" style="color:#e74c3c">Failed to load todos. Is the API running?</li>';
  }
}

todoForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  const title = todoInput.value.trim();
  if (!title) return;

  try {
    const todo = await createTodo(title);
    todoList.prepend(renderTodo(todo));
    todoInput.value = "";
    updateEmptyState();
  } catch (err) {
    alert("Failed to add todo");
  }
});

loadTodos();
