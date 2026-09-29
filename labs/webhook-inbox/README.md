# webhook-inbox

Somewhere to send webhooks and read them back, for Path 6, lab 6.L3. It keeps the last 20 messages for each token, in memory only: a restart empties it.

Image: `quay.io/computesphere/learn-webhook-inbox:1.4.0`. Listens on port 8080.

| Request | Response |
| --- | --- |
| `POST /hooks/<token>` | Stores the body (up to 64 KB); `202` `{"ok":true}` |
| `GET /<token>` | A page with the last 20 messages for that token, newest first. Refreshes every 5 seconds |
| `GET /<token>?format=json` | The same as JSON (so does `Accept: application/json`) |
| `GET /healthz` | `200` and `{"status":"ok"}` |
| `GET /` | How to use it |

A token is 6 to 64 letters, digits, `-` or `_`, chosen by you. Anyone who knows it can read that inbox, so pick one nobody would guess and don't send anything secret.

For a JSON body, the page shows its `text`, `content` or `message` field as the headline: the fields chat webhooks read.

## Deploy it

```bash
csph deploy --image quay.io/computesphere/learn-webhook-inbox:1.4.0 --name webhook-inbox --port 8080
```

Then add `https://<its URL>/hooks/<your token>` as a JSON webhook under Settings → Notifications, and open `https://<its URL>/<your token>`.

## Try it locally

```bash
docker run --rm -p 8080:8080 quay.io/computesphere/learn-webhook-inbox:1.4.0
curl -X POST localhost:8080/hooks/my-team-7f3a -H 'Content-Type: application/json' -d '{"text":"Hello from curl"}'
curl localhost:8080/my-team-7f3a?format=json
```
