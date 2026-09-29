# containerize-go

A tiny Go web app for *Containers* (Path 3). Your job is to put it in a container.

Standard library only: there's a `go.mod` but nothing to download.

| Request | Response |
| --- | --- |
| `GET /` | `200` and a short text greeting |
| `GET /healthz` | `200` and `{"status":"ok"}` |

It listens on `PORT` (default `3000`) and binds `0.0.0.0`, so it's reachable from outside a container.

## Run it without Docker

Go 1.23 or later:

```bash
go run .              # http://localhost:3000
curl localhost:3000/healthz

go build -o server .  # or build a binary and run ./server
```

## There's no Dockerfile on purpose

Writing it is the exercise. Start from the lesson, not from a copy of someone else's.
