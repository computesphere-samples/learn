const test = require("node:test");
const assert = require("node:assert/strict");
const { createStore, validateTask } = require("../lib/tasks");

test("validateTask requires a title", () => {
  assert.deepEqual(validateTask({ title: "   " }), ["title is required"]);
});

test("validateTask rejects a bad due date", () => {
  assert.deepEqual(validateTask({ title: "Ship it", due: "tomorrow" }), [
    "due must be a date like 2026-10-01",
  ]);
});

test("validateTask accepts a title with no due date", () => {
  assert.deepEqual(validateTask({ title: "Ship it" }), []);
});

test("store adds and removes tasks", () => {
  const store = createStore([{ title: "One", owner: "demo" }]);
  const added = store.add({ title: "Two", owner: "demo" });
  assert.equal(store.all().length, 2);
  assert.equal(store.remove(added.id), true);
  assert.equal(store.remove(added.id), false);
  assert.equal(store.all().length, 1);
});
