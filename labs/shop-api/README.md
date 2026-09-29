# shop-api

A small shop API for Path 6, *Operate your application*. It works like a normal service, and a few environment variables make it misbehave on purpose, so you can practise reading logs, metrics and health checks on something that's really failing.

Image: `quay.io/computesphere/learn-shop-api:1.4.0`. Listens on port 8080.

## Endpoints

| Request | Response |
| --- | --- |
| `GET /` | A short list of these endpoints |
| `GET /products` | The catalog |
| `GET /products/{id}` | One product (`p-100` to `p-104`), or `404` |
| `GET /orders` | The last 100 orders, newest first (in memory) |
| `POST /orders` | `{"product_id":"p-100","quantity":2}` creates an order, signed with `SIGNING_KEY`; `201` |
| `GET /healthz` | `503` `{"status":"starting"}` until `START_DELAY` has passed, then `200` `{"status":"ok"}` |
| `GET /status` | Version, uptime, whether it's ready, memory held, CPU burn left. Always `200` |
| `GET /work?ms=200` | Does about that many milliseconds of CPU work on a full core, inside the request (up to 10000). On a small or busy spherelet it takes longer: compare `took_ms` with `worked_ms` |
| `GET /burn?seconds=60` | Burns CPU in the background for that long (up to 3600). `&threads=2` uses more cores; `seconds=0` stops it |
| `GET /alloc?mb=64` | Holds that much more memory (up to 1024 per call) until `/free` or a restart |
| `GET /free` | Releases the memory from `/alloc` |

`/products` and `/orders` answer `503` while the app is starting, like `/healthz`.

## Variables

| Variable | Default | What it does |
| --- | --- | --- |
| `SIGNING_KEY` | none | Signs orders. **Required**: without it the app logs a `fatal` line naming the missing variable and exits. Set it as a secret |
| `REQUIRE_SIGNING_KEY` | `true` | `false` lets the app run without `SIGNING_KEY` (orders are unsigned). The broken-deploy exercise uses it so each case has one fault |
| `START_DELAY` | `0` | Seconds before the app is ready. Until then `/healthz` answers `503` |
| `CACHE_MB` | `0` | Megabytes of memory the app fills at start and keeps. More than the shape's memory ends in **Out of memory** |
| `ERROR_RATE` | `0` | Share of `/products` and `/orders` requests that fail with `500`: `0.2` or `20%` |
| `SHOP_NAME` | `Learn shop` | Shown on `/`, `/products` and `/status` |
| `PORT` | `8080` | The port it listens on |

A value that doesn't parse (`START_DELAY=soon`) stops the app with a `fatal` line that says what's wrong.

## Logs

One JSON object per line on stdout. Each request logs `level`, `msg`, `request_id` (from `X-Request-Id`, or a new one, returned in the response header), `method`, `route`, `status` and `duration_ms`; a simulated failure adds `error`. Passing health checks aren't logged; failing ones are.

```json
{"time":"2026-09-29T12:00:01Z","level":"error","msg":"request failed","request_id":"9f2c1a7e4b0d3c11","method":"GET","route":"/products","status":500,"duration_ms":0,"error":"simulated failure (ERROR_RATE=0.2)"}
{"time":"2026-09-29T12:00:00Z","level":"fatal","msg":"SIGNING_KEY is not set: the shop can't sign orders without it. Add SIGNING_KEY as a secret variable and redeploy","missing":"SIGNING_KEY"}
```

## Run it locally

```bash
docker run --rm -p 8080:8080 -e SIGNING_KEY=dev quay.io/computesphere/learn-shop-api:1.4.0
curl localhost:8080/products
curl -X POST localhost:8080/orders -d '{"product_id":"p-100","quantity":2}'
```

Some traffic to watch on the graphs:

```bash
while true; do curl -s -o /dev/null -w '%{http_code}\n' "$URL/products"; sleep 0.2; done
```

## Broken deploys and the challenge

[`broken/`](broken/) has the five broken deploys for lesson 6.5.3 and the setup script for challenge 6.C.
