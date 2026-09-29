// A tiny JSON API for Application foundations lesson 2.2.4.
// Node only, no dependencies: `node server.js`, then call it with curl.
// Data lives in memory, so it resets every time the server starts.
const http = require("node:http");

const port = Number(process.env.PORT) || 3000;
const notes = [{ id: 1, text: "Buy milk" }];
let nextId = 2;

function send(res, status, body) {
  res.writeHead(status, { "Content-Type": "application/json" });
  res.end(JSON.stringify(body));
}

function readJson(req) {
  return new Promise((resolve) => {
    let raw = "";
    req.on("data", (chunk) => (raw += chunk));
    req.on("end", () => {
      try {
        resolve(JSON.parse(raw || "{}"));
      } catch {
        resolve(null);
      }
    });
  });
}

const server = http.createServer(async (req, res) => {
  const url = new URL(req.url, `http://${req.headers.host}`);
  const match = url.pathname.match(/^\/notes\/(\d+)$/);

  if (req.method === "GET" && url.pathname === "/healthz") {
    return send(res, 200, { status: "ok" });
  }
  if (req.method === "GET" && url.pathname === "/notes") {
    return send(res, 200, notes);
  }
  if (req.method === "POST" && url.pathname === "/notes") {
    if (!(req.headers["content-type"] || "").startsWith("application/json")) {
      return send(res, 415, { error: "Send JSON with Content-Type: application/json" });
    }
    const body = await readJson(req);
    if (!body || typeof body.text !== "string" || body.text.trim() === "") {
      return send(res, 400, { error: "A note needs a non-empty \"text\" field" });
    }
    const note = { id: nextId++, text: body.text.trim() };
    notes.push(note);
    return send(res, 201, note);
  }
  if (req.method === "GET" && match) {
    const note = notes.find((n) => n.id === Number(match[1]));
    return note ? send(res, 200, note) : send(res, 404, { error: "No note with that id" });
  }
  send(res, 404, { error: "Not found" });
});

server.listen(port, () => console.log(`Notes API on http://localhost:${port}`));
