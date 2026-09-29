# containerize-node

A tiny Node web app for *Containers* (Path 3). Your job is to put it in a container.

It has one dependency, [express](https://expressjs.com), pinned in `package-lock.json`.

| Request | Response |
| --- | --- |
| `GET /` | `200` and a short text greeting |
| `GET /healthz` | `200` and `{"status":"ok"}` |

It listens on `PORT` (default `3000`) and binds `0.0.0.0`, so it's reachable from outside a container.

## Run it without Docker

Node 20 or later:

```bash
npm ci        # installs exactly what package-lock.json lists
npm start     # http://localhost:3000
curl localhost:3000/healthz
```

## There's no Dockerfile on purpose

Writing it is the exercise. Start from the lesson, not from a copy of someone else's.
