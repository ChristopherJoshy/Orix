import assert from "node:assert/strict";
import { Readable } from "node:stream";
import handler from "/tmp/opencode/orix-leaderboard-bff/[action].js";

// Isolated BFF contract test, never a live service or authentication scenario.
process.env.ORIX_API_ORIGIN = "https://api.example.invalid";
process.env.ORIX_ADMIN_API_SECRET = "unit-test-secret";
let calls = [];
globalThis.fetch = async (url, init) => {
  calls.push({ url, init });
  return new Response(JSON.stringify({ version: 1, entries: [] }), { status: 200 });
};
async function invoke(action, method, token = "Bearer unit-test", body = "") {
  const req = Readable.from(body ? [body] : []);
  req.url = `/api/admin/${action}`;
  req.method = method;
  req.headers = token ? { authorization: token } : {};
  let status;
  let output;
  const headers = {};
  await handler(req, {
    setHeader: (name, value) => { headers[name] = value; },
    writeHead: (value) => { status = value; },
    end: (value) => { output = value; },
  });
  return { status, output, headers };
}
assert.equal((await invoke("donation-leaderboard", "GET", "")).status, 401);
assert.equal(calls.length, 0);
assert.equal((await invoke("donation-leaderboard", "PUT")).status, 405);
assert.equal((await invoke("set-donation-leaderboard", "POST")).status, 405);
assert.equal(calls.length, 0);
const read = await invoke("donation-leaderboard", "GET");
assert.equal(read.status, 200);
assert.equal(read.headers["Cache-Control"], "no-store");
assert.equal(calls[0].url, "https://api.example.invalid/api/v1/admin/donation-leaderboard");
assert.equal(calls[0].init.headers["X-Admin-Secret"], "unit-test-secret");
assert.equal(read.output.includes("unit-test-secret"), false);
const body = JSON.stringify({ version: 0, entries: [], publication_confirmed: true });
assert.equal((await invoke("set-donation-leaderboard", "PUT", "Bearer unit-test", body)).status, 200);
assert.equal(calls[1].init.method, "PUT");
assert.equal(calls[1].init.body, body);
assert.equal((await invoke("set-donation-leaderboard", "PUT", "Bearer unit-test", "a".repeat(32769))).status, 413);
assert.equal(calls.length, 2);
console.log("Donation leaderboard BFF allowlist, credential isolation and body bounds passed.");
