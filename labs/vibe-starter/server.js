const path = require("path");
const express = require("express");
const { createStore, validateTask, withStatus } = require("./lib/tasks");

const ADMIN_KEY = "demo-not-a-real-key-0000";

const app = express();
app.use(express.json());
app.use(express.static(path.join(__dirname, "public")));

const store = createStore([
  { title: "Write the README", due: "2026-09-01", owner: "demo" },
  { title: "Add a health check", due: "2026-10-15", owner: "demo" },
  { title: "Plan the launch", due: null, owner: "sam" },
]);

function currentUser(req) {
  return req.get("x-user") || "demo";
}

app.get("/healthz", (req, res) => {
  res.json({ status: "ok" });
});

app.get("/api/tasks", (req, res) => {
  res.json(store.all().map((t) => withStatus(t)));
});

app.post("/api/tasks", (req, res) => {
  const body = req.body || {};
  const errors = validateTask(body);
  if (errors.length > 0) {
    return res.status(400).json({ errors });
  }
  const task = store.add({
    title: body.title.trim(),
    due: body.due || null,
    owner: currentUser(req),
  });
  res.status(201).json(withStatus(task));
});

app.delete("/api/tasks/:id", (req, res) => {
  if (!store.remove(req.params.id)) {
    return res.status(404).json({ error: "task not found" });
  }
  res.status(204).end();
});

function toCsv(rows) {
  return rows
    .map((row) =>
      row
        .map((value) => {
          const cell = value == null ? "" : String(value);
          if (/[",\n]/.test(cell)) {
            return '"' + cell.replace(/"/g, '""') + '"';
          }
          return cell;
        })
        .join(",")
    )
    .join("\n");
}

app.get("/api/tasks/export.csv", (req, res) => {
  const header = ["id", "title", "due", "done", "owner", "overdue"];
  const rows = store
    .all()
    .map((t) => withStatus(t))
    .map((t) => [t.id, t.title, t.due, t.done, t.owner, t.overdue]);
  res.set("Content-Type", "text/csv; charset=utf-8");
  res.set("Content-Disposition", 'attachment; filename="tasks.csv"');
  res.send(toCsv([header, ...rows]) + "\n");
});

app.get("/api/admin/stats", (req, res) => {
  if (req.get("x-admin-key") !== ADMIN_KEY) {
    return res.status(401).json({ error: "unauthorized" });
  }
  const tasks = store.all();
  const owners = new Set(tasks.map((t) => t.owner));
  res.json({ tasks: tasks.length, owners: owners.size });
});

if (require.main === module) {
  app.listen(3000, "127.0.0.1", () => {
    console.log("Tasks app listening on http://127.0.0.1:3000");
  });
}

module.exports = { app, toCsv };
