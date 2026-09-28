const test = require("node:test");
const assert = require("node:assert/strict");
const { app, toCsv } = require("../server");

test("toCsv quotes cells that contain commas, quotes or newlines", () => {
  assert.equal(
    toCsv([["a", "b, c", 'say "hi"'], ["line\nbreak", null, true]]),
    'a,"b, c","say ""hi"""\n"line\nbreak",,true'
  );
});

test("GET /api/tasks/export.csv returns every task as CSV", async () => {
  const server = app.listen(0);
  try {
    const { port } = server.address();
    const res = await fetch(`http://127.0.0.1:${port}/api/tasks/export.csv`);
    assert.equal(res.status, 200);
    assert.match(res.headers.get("content-type"), /^text\/csv/);
    const lines = (await res.text()).trim().split("\n");
    assert.equal(lines[0], "id,title,due,done,owner,overdue");
    assert.equal(lines.length, 4);
  } finally {
    server.close();
  }
});
