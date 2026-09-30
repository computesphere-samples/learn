# Broken deploys

Each file deploys `learn-shop-api` as a new service with one thing wrong. Deploy one, find out what's wrong from its status, deploy log and runtime log, fix it, then delete the service. Lesson 6.5.3 walks through them.

```bash
csph deploy --file case-1.yaml --no-wait
csph logs shop-case-1 --kind deploy     # newest event first
csph logs shop-case-1
```

Each is a new service (`shop-case-1` to `shop-case-5`), so none of them touches your working shop. A trial account runs two Flex spherelets in total: do them one at a time, and delete each when you're done.

## The fix loop

Most faults you fix in the file and deploy it again: variables you add or remove, the image tag and the health-check path all apply to the existing service. A port change is the exception: on a service that already exists it doesn't reach the URL. So for case 1, delete the broken service, fix the file, and deploy it again:

```bash
csph services list
csph services delete <service-id>
csph services list -o json    # the service is no longer listed
csph deploy --file case-1.yaml
```

Case 2's fix is a secret, which you set on the running service. Delete each service when you're done with it.

<details>
<summary>Answers (try first)</summary>

| Case | What's wrong | What you'll see | Fix |
| --- | --- | --- | --- |
| 1 | The service's port is 3000; the app listens on 8080 | The app logs `listening` on 8080, but the URL doesn't answer | Delete the service, set `port: 8080`, deploy the file again |
| 2 | `SIGNING_KEY` is missing | A `fatal` line naming `SIGNING_KEY`; the deploy log says **Restarting repeatedly** | `csph services secrets set SIGNING_KEY=demo-not-a-real-key-0000 --service <service-id>` (that redeploys it) |
| 3 | `CACHE_MB=900` on a 512 MB Flex spherelet | `warming the product cache`, then nothing; **Out of memory** | Delete the `CACHE_MB` line and deploy the file again |
| 4 | Tag `1.0.9` doesn't exist | An error at once, and no logs. It still leaves an empty `shop-case-4` service | Set the tag to `1.4.0` and deploy the file again |
| 5 | The health check is on `/health`, which answers 404 | The app runs, but **Unhealthy**; requests to `/health` answered 404 | Set `health_check_path: /healthz` and deploy the file again |

Don't use `csph deployments port` for case 1: fix the file and create the service again.

</details>

## Challenge 6.C

`challenge-setup.sh` deploys a working shop as `shop`, then ships a release with several faults at once. Don't read the script before you've finished.

```bash
read -s COMPUTESPHERE_API_TOKEN && export COMPUTESPHERE_API_TOKEN   # an API token with Project Access to your lab project
export PROJECT_ID=<lab-project-id> ENVIRONMENT_ID=<lab-environment-id>
./challenge-setup.sh
```

It needs `csph` (signed in), `curl` and `jq`, and takes about three minutes. It only creates `shop`, in the project and environment you name.

**Your account ID.** The API needs it. The script uses, in order: `COMPUTESPHERE_ACCOUNT_ID` if you set it; the default account `csph` saved when you signed in (`csph context` shows it); or your only account. If it can't tell, it stops and says so: run `csph accounts list` and `export COMPUTESPHERE_ACCOUNT_ID=<account-id>`, then run it again.

`DRY_RUN=1` prints the API calls without making them.

<details>
<summary>Challenge answers (try first)</summary>

The status shows **Running** throughout, even while the shop is broken: the URL is the truth. The bad release records no new version, so there's nothing to roll back to. Fix it forward.

| Fault | Evidence |
| --- | --- |
| The port is 3000; the app listens on 8080 | `/products` answers 500; the runtime log says `listening` with port 8080; **Networking** shows **Port** 3000 |
| The health check is on port 3000 with a 5 s initial delay; the app needs about 90 s to load its catalog | **Health Check** shows **Port** 3000 and **Initial Delay** 5s; the runtime log says `loading the catalog` with `start_delay` 90 |
| It runs on 2 spherelets | The **Spherelets** card on **Settings** shows 2 |
| The `SIGNING_KEY` secret is gone | `csph services secrets list --service <service-id>` prints `No secret overrides.` A spherelet that starts without it logs a `fatal` line naming `SIGNING_KEY` |

The fix, in this order:

1. **Health Check**, **Edit**: **Port** 8080, **Initial Delay** 100. **Update**, then **Later**.
2. **Networking**, **Edit**: **Port** 8080, then **Save**.
3. `csph deployments list --service <service-id>`, then `csph deployments scale <deployment-id> --spheres 1`.
4. `csph services secrets set SIGNING_KEY=demo-not-a-real-key-0000 --service <service-id>`. This redeploys the service with all four fixes.

Wait for **Running** (the app takes about two minutes to pass its health check), then `curl https://<shop-url>/products` answers 200 every time.

</details>
