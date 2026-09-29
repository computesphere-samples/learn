# compose-stack

A small Node web app (`app/`) and two Compose files for *Containers* (Path 3).

The app's `/healthz` reports on whatever it depends on:

| Setting | What `/healthz` checks | Healthy | Not healthy |
| --- | --- | --- | --- |
| `DATABASE_URL` | Can it connect to Postgres? | `200` `{"status":"ok","database":"connected"}` | `503` `"database":"unreachable"` |
| `API_URL` | Does `${API_URL}/hello` answer? | `200` `{"status":"ok","api":"reachable"}` | `503` `"api":"unreachable"` |

With neither set it just returns `{"status":"ok"}`. `GET /` returns a line of text. `app/Dockerfile` is a plain, correct Node Dockerfile that runs as the non-root `node` user.

## `compose.yaml`: the local stack (lesson 3.5.2)

The web app plus a Postgres 16 database, on your machine.

```bash
docker compose up --build
curl localhost:3000/healthz        # {"status":"ok","database":"connected"}
docker compose down -v             # -v also deletes the database volume
```

Things to notice: `web` reaches the database at the hostname `db` (the service name); `db` stores its data in the named volume `db-data`; and `depends_on` with `condition: service_healthy` holds `web` back until the database's healthcheck passes.

The password is `demo-password`. It's fine for a local demo and nothing else.

## `compose.deploy.yaml`: the stack you deploy (lab 3.L2)

The web app plus the Learn sample API (`quay.io/computesphere/learn-sample-api:1.0.0`). The web app calls `http://api:8080/hello`, and `/healthz` says whether the API answered.

There are no `build:` lines, because a platform runs images, not source folders. Before you deploy:

1. Build `./app` and push it to a registry you control.
2. Replace `<your-registry>` in `compose.deploy.yaml` with where you pushed it.

The lab walks through deploying the file to ComputeSphere.

## Why the database stays local

The deploy file has no database. ComputeSphere doesn't offer managed databases, so the Postgres version of the stack is for running on your own machine only.
