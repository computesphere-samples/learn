# containerize-python

A tiny Python web app for *Containers* (Path 3). Your job is to put it in a container.

It uses [Flask](https://flask.palletsprojects.com) for the routes and [gunicorn](https://gunicorn.org) as the server, both pinned in `requirements.txt`.

| Request | Response |
| --- | --- |
| `GET /` | `200` and a short text greeting |
| `GET /healthz` | `200` and `{"status":"ok"}` |

## Run it without Docker

Python 3.10 or later:

```bash
python3 -m venv .venv && . .venv/bin/activate
pip install -r requirements.txt
gunicorn --bind 0.0.0.0:${PORT:-3000} app:app    # http://localhost:3000
curl localhost:3000/healthz
```

That `gunicorn` line is the start command. `app:app` means "the `app` object in `app.py`". Binding `0.0.0.0` (not `127.0.0.1`) is what makes it reachable from outside a container, and `${PORT:-3000}` uses `PORT` when it's set.

## There's no Dockerfile on purpose

Writing it is the exercise. Start from the lesson, not from a copy of someone else's.
