# vibe-starter

A small tasks app of the kind an AI coding agent might produce in an afternoon: Node 20 and Express, one static page, a JSON API and in-memory data. It is the starter app for [ComputeSphere Learn](https://learn.computesphere.com) Path 4, *Vibe coding to production*.

> **This app has deliberate problems for the lessons — don't use it as a template.** The lessons walk you through finding and fixing them. Spotting them yourself first is part of the exercise, so they aren't listed here.

## Run it

You need Node 20 or later.

```bash
cd labs/vibe-starter
npm ci
npm start
```

Open http://localhost:3000, or check it from a terminal:

```bash
curl localhost:3000/healthz
```

Run the tests:

```bash
npm test
```

## What's in it

| Path | What it is |
|---|---|
| `server.js` | The Express app: the page, the API and the health check |
| `lib/dates.js` | Due-date helpers |
| `lib/tasks.js` | The in-memory task store and input validation |
| `public/index.html` | The tasks page |
| `test/` | Tests, run with Node's built-in test runner |
| `Dockerfile` | Container build |

## API

| Method and path | What it does |
|---|---|
| `GET /healthz` | Returns `{"status":"ok"}` |
| `GET /api/tasks` | Lists tasks |
| `POST /api/tasks` | Adds a task. Body: `{"title": "...", "due": "2026-10-01"}` (`due` is optional) |
| `DELETE /api/tasks/:id` | Deletes a task |

Requests identify the user with an `x-user` header. Without one, you are `demo`. Tasks live in memory, so they reset whenever the app restarts.

## The agent's branch

The branch `vibe-starter/agent-export` holds a change an agent proposed: a CSV export of the task list. You review it in the lessons:

https://github.com/computesphere-samples/learn/compare/main...vibe-starter/agent-export
