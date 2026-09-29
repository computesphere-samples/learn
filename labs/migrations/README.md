# migrations

A two-migration project for *Application foundations*, lesson 2.3.5. It uses Node's built-in SQLite, so there's nothing to install: Node 22.13 or later (Node 24 LTS recommended). The database is a local file, `app.db`; delete it to start over.

```bash
node migrate.js status   # which migrations have run
node migrate.js up       # apply every pending migration
node migrate.js schema   # show the tasks table's columns
node migrate.js down     # roll back the most recent migration
```

Each migration is a pair of SQL files in `migrations/`: `*.up.sql` makes the change and `*.down.sql` undoes it. The runner records what has run in a `schema_migrations` table.
