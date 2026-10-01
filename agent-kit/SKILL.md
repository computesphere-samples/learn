---
name: deploy-to-computesphere
description: Use when deploying, redeploying, restarting, rolling back or debugging an app on ComputeSphere, writing or applying a computesphere.yaml, setting ComputeSphere secrets or tokens, or reading ComputeSphere logs with csph or the ComputeSphere MCP server.
---

# Deploy to ComputeSphere

ComputeSphere runs apps as services (web service, background worker, cron job, static site) from a container image or a build of a connected Git repository, inside a project and an environment. Each service runs on spherelets of a CPU-and-memory shape: `FLX` (Flex), `MAX` (Standard) or `PWR` (Performance). There are no GPU shapes and no database to provision: use one hosted elsewhere, with its URL as a secret. Tools: the `csph` CLI, the console, and the MCP server at `https://mcp.computesphere.com/mcp`.

## Steps

### 1. Check the app is ready to run online

- It listens on `0.0.0.0` (not `127.0.0.1` or `localhost`) on the port in `PORT`, with a default. The service's port must match.
- A web service has a readiness path, such as `/healthz`, that returns 200–399 only once start-up has finished. Keep it shallow.
- Secrets come from environment variables. Search the diff and the repository for keys, passwords and tokens; none may be in code, the Dockerfile, the image or the manifest.
- Tests pass and `git status` is clean.

Stop and report anything that fails. Don't deploy around it.

### 2. Check access

- `csph context` shows the active account, project and environment.
- csph should run with a **project-scoped, expiring** token in `COMPUTESPHERE_API_TOKEN`, put there by the human outside the chat. If there isn't one, ask the human to create it (console **Settings > User tokens**, **Project access**, 7 days or less, or `csph auth token create --scope project --restrictions <project-id> --expiry "<date>"`). Never ask for a full-access token, never ask for it in the chat, and never print it.

### 3. Write or update the manifest

```yaml
version: "2"
project:
  name: my-app
environments:
  - name: prod
    region: us-east-1          # `csph regions list`
    secrets:
      DATABASE_URL:
        secret: true           # name only; the value is set outside the file
services:
  - name: web
    type: web-service
    image: ghcr.io/your-org/your-app:1.2.0
    port: 8080
    plan: FLX                  # always set: FLX, MAX or PWR
    spherelets: 1
    health_check_path: /healthz
    env_vars:
      LOG_LEVEL: info          # non-secret values only
```

For a single prebuilt image, `csph deploy --image <ref> --name <name> --port <port> --write` deploys and saves the generated manifest.

Validate with `csph apply --file computesphere.yaml --dry-run`.

### 4. Ask, then deploy

Show the human the exact command, the project and environment, and what it does to data. Wait for an explicit yes. Then:

```bash
csph apply --file computesphere.yaml
```

Know the limits: re-applying an unchanged file deploys nothing; for an existing service a re-apply only picks up a new image (change variables, spherelet count and health check in the console or with csph); apply never deletes. A Git-built service is set up and built from the console; a push or merge doesn't mean it deployed.

### 5. Verify

- `csph deployments get <deployment-id>` until it's Running or Failed.
- `curl` the URL's health path.
- Report what you saw: status, URL, and the log lines that matter. Not what you expected.

### 6. If it fails, read before guessing

1. `csph logs <service> --kind build` (Git-built services)
2. `csph logs <service> --kind deploy` (rollout: image pull, health check, start-up)
3. `csph logs <service> --since 30m`, or `-q error`
4. Usual causes: listening on `127.0.0.1`; port mismatch; health path 404, or a start slower than the initial delay; a missing secret; a crash at start-up.

Quote the lines your diagnosis rests on.

### 7. If a live release is broken, roll back first

Propose `csph deployments rollback <deployment-id>` (previous version) or `--to <version-id>`, get a yes, then diagnose while users are served. Don't stack quick fixes on a broken release. A rollback restores that version's image, variables, secrets, port and health check, so a secret rotated since then must be set again. Deploy history keeps one version per minute, so leave a minute between releases.

## Lifecycle commands

| Command | Use it when |
| --- | --- |
| `csph deployments redeploy <deployment-id>` | Settings were saved and need applying. With nothing changed, nothing rolls out |
| `csph deployments restart <deployment-id>` | A process is stuck and you need fresh spherelets running the same thing |
| `csph deployments rollback <deployment-id>` | The current release is bad |

IDs: `csph services list --project <project-id>`, then `csph deployments list --service <service-id>`. Restart and redeploy wipe anything the app keeps in memory or on its own disk. Neither picks up new code.

## Never without the human

- Any change to a live service: deploy, redeploy, restart, roll back, scale, stop, variables. Propose it; they approve.
- Secret values. You propose `csph environments secrets set` or `csph services secrets set` with key names; the human runs it. Both store the whole set at once, and `--replace` deletes every secret not listed.
- Deleting a service, environment, project or volume; domains and DNS; creating or revoking tokens; plans and billing.
- MCP confirmation requests: show them to the human. Never confirm on their behalf.

## With the MCP server

It acts as the signed-in human, with everything their account reaches. Use it to read: `listProjects`, `listServices`, `listDeployments`, `getDeploymentStatus`, `getDeploymentDeployLogs`, `getDeploymentRuntimeLogs`, `getDeploymentBuildLogs`, `listDeploymentVersions`. Changes (`applyManifest`, `restartDeployment`, `rollbackDeployment`) follow the same approval rule; for unattended work prefer csph with a project token. Use only tools the server lists when you connect.

Docs: https://docs.computesphere.com/docs/cli/getting-started/
