# notes

The app for the volume lab in Path 5, lesson 5.5.4. It keeps one note in `/data/note.txt`, so you can see a file on the spherelet's own filesystem vanish on a redeploy, then mount a SphereStor volume at `/data` and watch the note survive.

Image: `quay.io/computesphere/learn-notes:1.5.0`. Listens on port 8080. Logs are JSON, one line per request.

| Request | Response |
| --- | --- |
| `POST /note` | Saves the request body (up to 16 KB) as the note, replacing the last one; `204` |
| `GET /note` | The note as plain text, or an empty reply if none is saved |
| `GET /healthz` | `200` and `{"status":"ok"}` |
| `GET /` | The version and where the note is kept |

`DATA_DIR` changes the folder (default `/data`). The image creates `/data` for its non-root user, so saving works with or without a volume there.

## Deploy it

```bash
csph deploy --image quay.io/computesphere/learn-notes:1.5.0 --name notes --port 8080
curl -X POST --data "before redeploy" https://<your notes URL>/note
curl https://<your notes URL>/note
```

## Try it locally

```bash
mkdir -p notes-data
docker run --rm -p 8080:8080 -v "$PWD/notes-data:/data" quay.io/computesphere/learn-notes:1.5.0
curl -X POST --data "hello" localhost:8080/note
curl localhost:8080/note
```

Stop the container and start it again: the note is still there, because it's in `notes-data` on your machine. Start it without `-v` and it's gone.
