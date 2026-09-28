const { isValidDueDate, isOverdue } = require("./dates");

function createStore(seed = []) {
  let nextId = 1;
  const tasks = [];

  function add({ title, due = null, owner }) {
    const task = { id: String(nextId++), title, due, done: false, owner };
    tasks.push(task);
    return task;
  }

  for (const t of seed) {
    add(t);
  }

  return {
    all: () => tasks.slice(),
    add,
    remove(id) {
      const i = tasks.findIndex((t) => t.id === id);
      if (i === -1) {
        return false;
      }
      tasks.splice(i, 1);
      return true;
    },
  };
}

function validateTask(body) {
  const errors = [];
  const title = typeof body.title === "string" ? body.title.trim() : "";
  if (!title) {
    errors.push("title is required");
  } else if (title.length > 200) {
    errors.push("title must be 200 characters or fewer");
  }
  if (body.due != null && body.due !== "" && !isValidDueDate(body.due)) {
    errors.push("due must be a date like 2026-10-01");
  }
  return errors;
}

function withStatus(task, now = new Date()) {
  return { ...task, overdue: isOverdue(task, now) };
}

module.exports = { createStore, validateTask, withStatus };
