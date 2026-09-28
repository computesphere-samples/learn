const test = require("node:test");
const assert = require("node:assert/strict");
const { isValidDueDate, daysUntil } = require("../lib/dates");

test("isValidDueDate accepts a real calendar date", () => {
  assert.equal(isValidDueDate("2026-10-01"), true);
});

test("isValidDueDate rejects other formats and impossible dates", () => {
  assert.equal(isValidDueDate("10/01/2026"), false);
  assert.equal(isValidDueDate("2026-02-30"), false);
  assert.equal(isValidDueDate(""), false);
  assert.equal(isValidDueDate(null), false);
});

test("daysUntil counts whole days", () => {
  const now = new Date("2026-10-01T15:30:00Z");
  assert.equal(daysUntil("2026-10-04", now), 3);
  assert.equal(daysUntil("2026-09-30", now), -1);
});
