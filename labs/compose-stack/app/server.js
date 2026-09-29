// The web app for the Compose lessons (Path 3). /healthz reports on whatever
// this app depends on, and which dependencies it has depends on configuration:
//   DATABASE_URL set -> it checks it can reach Postgres (compose.yaml)
//   API_URL set      -> it checks it can reach ${API_URL}/hello (compose.deploy.yaml)
// If a dependency is down, /healthz returns 503 so the platform knows.
const express = require("express");
const { Client } = require("pg");

const app = express();
const port = Number(process.env.PORT) || 3000;
const databaseUrl = process.env.DATABASE_URL;
const apiUrl = process.env.API_URL;

// Open a short-lived connection and run a trivial query.
async function checkDatabase() {
  const client = new Client({ connectionString: databaseUrl, connectionTimeoutMillis: 2000 });
  try {
    await client.connect();
    await client.query("SELECT 1");
    return true;
  } catch (err) {
    console.error("database check failed:", err.message);
    return false;
  } finally {
    await client.end().catch(() => {});
  }
}

// Call the other service by its name, e.g. http://api:8080/hello.
async function checkApi() {
  try {
    const res = await fetch(`${apiUrl}/hello`, { signal: AbortSignal.timeout(2000) });
    return res.ok;
  } catch (err) {
    console.error("api check failed:", err.message);
    return false;
  }
}

app.get("/", (_req, res) => {
  res.type("text/plain").send("Hello from the compose-stack web app. Try /healthz.\n");
});

app.get("/healthz", async (_req, res) => {
  const body = { status: "ok" };
  if (databaseUrl) body.database = (await checkDatabase()) ? "connected" : "unreachable";
  if (apiUrl) body.api = (await checkApi()) ? "reachable" : "unreachable";

  const healthy = body.database !== "unreachable" && body.api !== "unreachable";
  if (!healthy) body.status = "unhealthy";
  res.status(healthy ? 200 : 503).json(body);
});

app.listen(port, "0.0.0.0", () => {
  console.log(`compose-stack web listening on 0.0.0.0:${port}`);
});
