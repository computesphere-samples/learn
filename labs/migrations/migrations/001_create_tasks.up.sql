CREATE TABLE tasks (
  id    INTEGER PRIMARY KEY,
  title TEXT NOT NULL,
  done  INTEGER NOT NULL DEFAULT 0
);
INSERT INTO tasks (title) VALUES ('Write the first migration');
