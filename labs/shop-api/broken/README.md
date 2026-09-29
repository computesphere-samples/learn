# Broken deploys

Each file deploys `learn-shop-api` as a new service with one thing wrong. Deploy one, find out what's wrong from its status, deploy log and runtime log, fix it, then delete the service. Lesson 6.5.3 walks through them.

```bash
csph deploy --file case-1.yaml
csph deploy --file case-2.yaml
csph deploy --file case-3.yaml
csph deploy --file case-4.yaml
csph deploy --file case-5.yaml
```

Each is a new service (`shop-case-1` to `shop-case-5`), so none of them touches your working shop. A trial account runs two Flex spherelets in total: do them one at a time, and delete each when you're done:

```bash
csph deployments list
csph deployments delete <deployment-id>
```

`csph deploy` waits up to five minutes for Running. A broken one ends in an error; that's expected. Read the deploy log in the console or with `csph logs <service> --kind deploy`.

<details>
<summary>Answers (try first)</summary>

| Case | What's wrong | What you'll see | Fix |
| --- | --- | --- | --- |
| 1 | The service's port is 3000; the app listens on 8080 | The app logs `listening` on 8080, but the URL doesn't answer | Port 8080 |
| 2 | `SIGNING_KEY` is missing | A `fatal` line naming `SIGNING_KEY`; **Crash looping** | Add `SIGNING_KEY` as a secret, redeploy |
| 3 | `CACHE_MB=900` on a 512 MB Flex spherelet | `warming the product cache`, then nothing; **Out of memory** | Remove `CACHE_MB`, or a bigger shape |
| 4 | Tag `1.0.9` doesn't exist | Refused before anything starts | Tag `1.4.0` |
| 5 | The health check is on `/health`, which answers 404 | The app runs, but **Unhealthy** | Endpoint Path `/healthz` |

</details>

## Challenge 6.C

`challenge-setup.sh` deploys a working shop, then ships a release with several faults at once. Don't read the script before you've finished.

```bash
export COMPUTESPHERE_API_TOKEN=<your API token>
export PROJECT_ID=<lab project id> ENVIRONMENT_ID=<lab environment id>
./challenge-setup.sh
```

It needs `csph`, `curl` and `jq`, and takes about five minutes: the good release starts slowly, and it waits over a minute before the bad one so both show in deploy history. `DRY_RUN=1` prints the API calls without making them.
