"""A tiny web app for Containers (Path 3).

Flask handles the routes; gunicorn is the production server that runs it.
Both are pinned in requirements.txt so every install gets the same versions.
"""
from flask import Flask, jsonify

app = Flask(__name__)


@app.get("/")
def index():
    return "Hello from a Python app. Put me in a container!\n", 200, {
        "Content-Type": "text/plain; charset=utf-8"
    }


# Health checks call this. Keep it cheap: no database, no external calls.
@app.get("/healthz")
def healthz():
    return jsonify(status="ok")
