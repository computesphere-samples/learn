# ComputeSphere Learn

Lab apps and starter code for [ComputeSphere Learn](https://learn.computesphere.com). Each lab lives in its own directory under `labs/` and builds on its own.

## Lab images

Tiny Go programs on a distroless, non-root base. Each listens on port 8080.

| Image | What it does | Source |
|---|---|---|
| `quay.io/computesphere/learn-hello-web:1.0.0` | Serves a page on `/` and `{"status":"ok"}` on `/healthz` | `labs/hello-web` |
| `quay.io/computesphere/learn-hello-web-broken:1.0.0` | The same page, but `/healthz` returns 503. The bad release in the rollback lab | `labs/hello-web` (`Dockerfile.broken`) |
| `quay.io/computesphere/learn-sample-api:1.0.0` | `GET /hello` returns the `GREETING` variable and whether `API_KEY` is set, never its value | `labs/sample-api` |

Deploy one:

```bash
csph deploy --image quay.io/computesphere/learn-hello-web:1.0.0 --name hello-web --port 8080
```

Run one locally:

```bash
docker run --rm -p 8080:8080 quay.io/computesphere/learn-hello-web:1.0.0
curl localhost:8080/healthz
```

## Releasing

Images are built by Quay when a version tag (`1.0.0`, `1.1.0`, …) is pushed. Pushes to `main` build nothing. To release a change, bump the lab's `VERSION` file, merge, then tag `main` with the new version. Lessons pin exact versions, so a published tag is never moved.
