// A minimal migration runner for Application foundations lesson 2.3.5.
// Uses Node's built-in SQLite, so there's nothing to install.
//   node migrate.js status   which migrations have run
//   node migrate.js up       run every migration that hasn't run yet
//   node migrate.js down     undo the most recent migration
//   node migrate.js schema   print the tasks table's columns
const fs = require("node:fs");
const path = require("node:path");
const { DatabaseSync } = require("node:sqlite");

const db = new DatabaseSync(path.join(__dirname, "app.db"));
const dir = path.join(__dirname, "migrations");
db.exec("CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY)");

const all = fs
  .readdirSync(dir)
  .filter((f) => f.endsWith(".up.sql"))
  .map((f) => f.replace(".up.sql", ""))
  .sort();
const applied = () =>
  db.prepare("SELECT version FROM schema_migrations ORDER BY version").all().map((r) => r.version);

function run(version, direction) {
  const sql = fs.readFileSync(path.join(dir, `${version}.${direction}.sql`), "utf8");
  db.exec("BEGIN");
  try {
    db.exec(sql);
    if (direction === "up") db.prepare("INSERT INTO schema_migrations VALUES (?)").run(version);
    else db.prepare("DELETE FROM schema_migrations WHERE version = ?").run(version);
    db.exec("COMMIT");
    console.log(`${direction === "up" ? "Applied" : "Rolled back"} ${version}`);
  } catch (err) {
    db.exec("ROLLBACK");
    console.error(`Failed on ${version}: ${err.message}`);
    process.exit(1);
  }
}

const command = process.argv[2] || "status";
if (command === "up") {
  const pending = all.filter((v) => !applied().includes(v));
  if (pending.length === 0) console.log("Nothing to apply");
  pending.forEach((v) => run(v, "up"));
} else if (command === "down") {
  const last = applied().at(-1);
  if (!last) console.log("Nothing to roll back");
  else run(last, "down");
} else if (command === "schema") {
  const exists = db.prepare("SELECT 1 FROM sqlite_master WHERE type='table' AND name='tasks'").get();
  if (!exists) console.log("No tasks table yet");
  else console.table(db.prepare("PRAGMA table_info(tasks)").all().map((c) => ({ column: c.name, type: c.type })));
} else {
  const done = applied();
  all.forEach((v) => console.log(`${done.includes(v) ? "[x]" : "[ ]"} ${v}`));
}
