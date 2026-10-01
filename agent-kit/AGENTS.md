# Deploying to ComputeSphere

Instructions for AI coding agents working in this repository. Read this before you deploy, redeploy, roll back or change anything on ComputeSphere.

## What ComputeSphere is

ComputeSphere runs your app as **services** (web services, background workers, cron jobs, static sites) from a container image or from a build of a connected Git repository. Services live in a **project** and an **environment** (for example `staging` and `prod`), and a web service gets a public URL. Each service runs on one or more **spherelets** of a fixed CPU-and-memory shape: `FLX` (Flex), `MAX` (Standard) or `PWR` (Performance). There are no GPU shapes, and ComputeSphere doesn't provide a database for you to provision: use one hosted elsewhere and pass its URL in as a secret. You work with it through the `csph` CLI, the console, or its MCP server.

## Before any deploy

- [ ] The app listens on `0.0.0.0`, not `127.0.0.1` or `localhost`, on the port in the `PORT` variable with a sensible default. The service's port is set to that same number.
- [ ] A web service has a health check on a path that answers 200–399 only once the app can really serve.
- [ ] No secret is in the code, the manifest, the Dockerfile, the image, a commit or a log line.
- [ ] Tests pass and `git status` is clean.
- [ ] You've shown the human the exact command and they said yes (see the rules).

## How to deploy

Install csph with `brew install computesphere/cli/csph`. The human signs in with `csph auth login`, or gives you a token through the environment (rule 2). `csph context` shows the active account, project and environment.

**A prebuilt image:**

```bash
csph deploy --image ghcr.io/your-org/your-app:1.2.0 --name web --port 8080 --write
```

`--write` also saves the manifest it generated to `./computesphere.yaml`, so the next deploy can come from the file.

**A manifest, `computesphere.yaml`, kept in the repository:**

```yaml
version: "2"
project:
  name: my-app
environments:
  - name: prod
    region: us-east-1          # `csph regions list` shows the names
    secrets:
      DATABASE_URL:
        secret: true           # declared here, value set outside the file
services:
  - name: web
    type: web-service          # or background-worker, cron-job, static-site
    image: ghcr.io/your-org/your-app:1.2.0
    port: 8080
    plan: FLX                  # always set it: FLX, MAX or PWR
    spherelets: 1
    health_check_path: /healthz
    env_vars:
      LOG_LEVEL: info          # non-secret configuration only
```

```bash
csph apply --file computesphere.yaml --dry-run   # validate, create nothing
csph apply --file computesphere.yaml             # or just `csph deploy` in this folder
```

Know what apply does:

- It's idempotent: re-applying an unchanged file reports `unchanged` and deploys nothing.
- For a service that already exists, a re-apply today picks up a **new image** only. Edits to its variables, spherelet count or health check in the file don't reach the running service; change those in the console or with csph.
- Apply never deletes. Removing a service from the file leaves it running.
- When one service is created or updated, apply waits for Running and prints the URL.

**From a Git repository:** a service that builds from a connected repository is set up by the human in the console, which is where they choose which repository ComputeSphere may read. `csph deploy` can't build local source. Never assume a push or merge deployed anything: check the status.

## Rules

1. **Secrets are never in code or the manifest.** Read them from environment variables. In `computesphere.yaml`, declare the name with `secret: true` and nothing else. The human sets the value in the console, or with `csph environments secrets set` or `csph services secrets set`. Both `set` commands store the whole set at once, and `--replace` deletes every secret not listed, so propose the command and let the human run it. Never print, log or echo a secret, and never read `.env` files unless asked.
2. **Use a project-scoped, expiring API token, never a full-access one.** The human creates it (console **Settings > User tokens**, **Project access**, 7 days or less, or `csph auth token create --scope project --restrictions <project-id> --expiry "<date>"`) and hands it over as `COMPUTESPHERE_API_TOKEN` in your terminal, never in the chat. Don't create, widen or revoke tokens yourself. If a token appears in the chat, tell the human to revoke it.
3. **Every web service has a health check on a real readiness path**, such as `/healthz`, that returns success only when start-up has finished. Without one, ComputeSphere only checks that the port accepts connections. Keep it shallow: don't fail it because a shared dependency is slow. If the app takes long to start, raise the health check's initial delay rather than removing the check.
4. **Listen on `0.0.0.0` and the `PORT` variable.** An app on `127.0.0.1`, or on a port different from the service's, never reaches Running.
5. **Ask before anything that changes or deletes a live service.** Reading is free: status, logs, history. Deploy, redeploy, restart, roll back, scale, stop, and variable changes each need the human's yes. Show the exact command, the deployment ID, the project and environment, and what it does to data: a restart or redeploy wipes anything the app keeps in memory or on its own disk. The human does these, or reads the exact action first: deleting a service, environment, project or volume; rotating or replacing production secrets; domains and DNS; tokens; plans and billing. When an MCP tool returns a confirmation request, show it to the human and let them approve it; never confirm on their behalf.
6. **Read the logs before you guess.** Quote the lines your conclusion rests on.
7. **Prefer rollback over hot-fixing a broken release.** Roll back to the last good version first, then diagnose while users are served. A rollback restores the earlier version's image, variables, secrets, port and health check, so a secret rotated since then must be set again.
8. **Don't overclaim.** Report what you checked (status, URL, log lines), not what you expect. Only use commands and tools that exist; `csph <command> --help` lists them.

## Redeploy, restart, rollback

| Command | What it does |
| --- | --- |
| `csph deployments redeploy <deployment-id>` | Applies the service's saved settings. If nothing changed, nothing rolls out and the same spherelets keep running |
| `csph deployments restart <deployment-id>` | Fresh spherelets running exactly what runs now. Use it for a stuck process; it doesn't apply settings changed since the last deploy |
| `csph deployments rollback <deployment-id>` | Back to the previous version; `--to <version-id>` for an earlier one |

Find IDs with `csph services list --project <project-id>`, then `csph deployments list --service <service-id>`. Deploy history keeps one version per minute, so leave a minute between releases you may want to roll back between. Neither redeploy nor restart picks up new code: that needs a new image or a new build.

## Troubleshooting order

1. **Status:** `csph deployments get <deployment-id>`. Is it Running, Failed, or still in progress?
2. **Build** (Git-built services): `csph logs <service> --kind build`.
3. **Rollout:** `csph logs <service> --kind deploy`. Image pull, health check and start-up events.
4. **App output:** `csph logs <service> --since 30m`, `csph logs <service> -q error`, or `-f` to follow.
5. **Then the usual causes:** listening on `127.0.0.1`; the service port doesn't match the app's; the health check path returns 404 or the app starts slower than the initial delay; a missing variable or secret; a crash at start-up.
6. **A release broke a live service?** Roll back (rule 7), then work through 1 to 5.

## Using the MCP server

ComputeSphere's MCP server is at `https://mcp.computesphere.com/mcp`. It acts **as the signed-in human**, with everything their account can reach, so use it mainly to read: `listProjects`, `listServices`, `listDeployments`, `getDeploymentStatus`, `getDeploymentRuntimeLogs`, `getDeploymentDeployLogs`, `getDeploymentBuildLogs`, `listDeploymentVersions`. Changes such as `applyManifest`, `restartDeployment` or `rollbackDeployment` follow rule 5. For unattended changes prefer csph with a project token (rule 2). The server publishes its current tool list when you connect; use only tools that appear there.

More: https://docs.computesphere.com/docs/cli/getting-started/ and https://learn.computesphere.com
