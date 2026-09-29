# ComputeSphere Learn

Lab apps and starter code for [ComputeSphere Learn](https://learn.computesphere.com). Each lab lives in its own directory under `labs/` and builds on its own.

## Lab images

Tiny Go programs on a distroless, non-root base. Each listens on port 8080.

| Image | What it does | Source |
|---|---|---|
| `quay.io/computesphere/learn-hello-web:1.0.0` | Serves a page on `/` and `{"status":"ok"}` on `/healthz` | `labs/hello-web` |
| `quay.io/computesphere/learn-hello-web-broken:1.0.0` | The same page, but `/healthz` returns 503. The bad release in the rollback lab | `labs/hello-web` (`Dockerfile.broken`) |
| `quay.io/computesphere/learn-sample-api:1.0.0` | `GET /hello` returns the `GREETING` variable and whether `API_KEY` is set, never its value | `labs/sample-api` |
| `quay.io/computesphere/learn-request-demo:1.1.0` | The service learners inspect in Cloud foundations: `?code=<your code>` adds an `X-Learn-Nonce` response header; `/verify` checks it. Run by ComputeSphere; needs `NONCE_SECRET` (32+ characters) | `labs/request-demo` |
| `quay.io/computesphere/learn-shop-api:1.0.0` | The shop API for Path 6, *Operate*: `/products`, `/orders`, JSON logs, and switches for a slow start, a missing secret, memory, CPU and an error rate. Needs `SIGNING_KEY` | `labs/shop-api` |
| `quay.io/computesphere/learn-webhook-inbox:1.0.0` | Receives webhooks at `POST /hooks/<token>` and shows the last 20 at `/<token>` (page or JSON), in memory | `labs/webhook-inbox` |

Deploy one:

```bash
csph deploy --image quay.io/computesphere/learn-hello-web:1.0.0 --name hello-web --port 8080
```

Run one locally:

```bash
docker run --rm -p 8080:8080 quay.io/computesphere/learn-hello-web:1.0.0
curl localhost:8080/healthz
```

## Starter apps

| Directory | What it is |
|---|---|
| `labs/vibe-starter` | A small Node tasks app for Path 4, *Vibe coding to production*. It has deliberate problems for the lessons, so don't use it as a template |
| `labs/local-api` | A tiny notes API in Node, no dependencies, for calling an API with curl (Path 2, lesson 2.2.4) |
| `labs/migrations` | Two up/down migrations and a minimal runner on Node's built-in SQLite (Path 2, lesson 2.3.5) |
| `labs/containerize-node` | A tiny Express app with no Dockerfile: you write it (Path 3, *Containers*) |
| `labs/containerize-python` | The same app in Flask with gunicorn, no Dockerfile (Path 3) |
| `labs/containerize-go` | The same app in Go, standard library only, no Dockerfile (Path 3) |
| `labs/bloated-app` | A Go app whose Dockerfile builds a far-too-big image on purpose (Path 3, lesson 3.4.5) |
| `labs/compose-stack` | A Node web app with a local Compose stack (Postgres) and a deployable one (web + sample API) (Path 3, lesson 3.5.2 and lab 3.L2) |

## Releasing

Images are built by the **Publish lab images** workflow (`.github/workflows/publish-images.yaml`) when a version tag (`1.0.0`, `1.1.0`, …) is pushed: every image in `labs/images.json` is built for `linux/amd64` and pushed to `quay.io/computesphere/<repository>:<tag>` by the `computesphere+learn_samples` robot. Pushes to `main` build nothing. To release a change, bump the lab's `VERSION` file, merge, then tag `main` with the new version. Lessons pin exact versions, so a published tag is never moved: the workflow refuses to overwrite one.

To add a new lab image: add it to `labs/images.json`, and create its public repository in the `computesphere` Quay org with write access for the robot, before the next tag.
