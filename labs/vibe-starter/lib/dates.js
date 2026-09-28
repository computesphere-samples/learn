// Dates are plain "YYYY-MM-DD" strings and are compared in UTC.

function startOfDay(value) {
  const d = new Date(value);
  return Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate());
}

function isValidDueDate(value) {
  if (typeof value !== "string" || !/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    return false;
  }
  const d = new Date(value + "T00:00:00Z");
  return !Number.isNaN(d.getTime()) && d.toISOString().slice(0, 10) === value;
}

function daysUntil(due, now = new Date()) {
  const ms = startOfDay(due + "T00:00:00Z") - startOfDay(now);
  return Math.round(ms / 86400000);
}

function isOverdue(task, now = new Date()) {
  if (!task.due || task.done) {
    return false;
  }
  return startOfDay(task.due + "T00:00:00Z") <= startOfDay(now);
}

module.exports = { isValidDueDate, daysUntil, isOverdue };
