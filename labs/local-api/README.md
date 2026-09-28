# local-api

A tiny notes API for *Application foundations*, lesson 2.2.4. Node 20 or later, no dependencies. Data lives in memory and resets when the server stops.

```bash
node server.js                # listens on http://localhost:3000 (set PORT to change it)
```

| Request | Response |
| --- | --- |
| `GET /notes` | `200` and the list of notes |
| `GET /notes/1` | `200` and one note; `404` if there's no note with that id |
| `POST /notes` with `{"text":"…"}` and `Content-Type: application/json` | `201` and the new note; `400` if `text` is missing or empty; `415` without the JSON content type |
| `GET /healthz` | `200` and `{"status":"ok"}` |
