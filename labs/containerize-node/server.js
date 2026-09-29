// A tiny web app for Containers (Path 3). It has one dependency, express,
// pinned in package-lock.json, so `npm ci` installs exactly the same thing
// on your laptop and inside a container.
const express = require("express");

const app = express();

// Read the port from the environment so the platform can choose it.
const port = Number(process.env.PORT) || 3000;

app.get("/", (_req, res) => {
  res.type("text/plain").send("Hello from a Node app. Put me in a container!\n");
});

// Health checks call this. Keep it cheap: no database, no external calls.
app.get("/healthz", (_req, res) => {
  res.json({ status: "ok" });
});

// Bind 0.0.0.0, not localhost. Inside a container, localhost is only the
// container itself, so a server bound to 127.0.0.1 can't be reached from outside.
app.listen(port, "0.0.0.0", () => {
  console.log(`containerize-node listening on 0.0.0.0:${port}`);
});
