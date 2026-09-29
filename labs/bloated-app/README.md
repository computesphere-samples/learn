# bloated-app

A tiny Go web app with a Dockerfile that works but is far too big. **This image is too big on purpose; lesson 3.4.5 shrinks it.**

| Request | Response |
| --- | --- |
| `GET /` | `200` and a short text greeting |
| `GET /healthz` | `200` and `{"status":"ok"}` |

## Build it and look at the size

```bash
docker build -t bloated-app .
docker images bloated-app          # well over 1 GB on disk for a few-MB program
docker run --rm -p 3000:3000 bloated-app
curl localhost:3000/healthz
```

## What's wrong with the Dockerfile

Read it before the lesson and list what you'd change. The comments hint at each problem: the base image, what gets copied, who the process runs as, and what ends up in the final image. There's also no `.dockerignore`.

Don't commit a fix here. Lesson 3.4.5 walks through it.
