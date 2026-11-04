import assert from 'node:assert/strict';
import fs from 'node:fs';

globalThis.window = { location: { origin: 'https://orix.example' }, dispatchEvent() {} };
globalThis.CustomEvent ??= class { constructor(type, init) { this.type = type; this.detail = init?.detail; } };
const source = fs.readFileSync(process.argv[2], 'utf8');
const { fetchWithRetry } = await import(`data:text/javascript;base64,${Buffer.from(source).toString('base64')}`);
const originalNow = Date.now;
const originalFetch = globalThis.fetch;
let now = Date.UTC(2026, 10, 4);
Date.now = () => now;
try {
  for (const guidance of ['60', 'date', 'missing', 'invalid']) {
    let calls = 0;
    globalThis.fetch = async () => {
      calls++;
      return calls === 1
        ? new Response(null, { status: 429, headers: guidance === 'missing' ? {} : {
          'Retry-After': guidance === 'date' ? new Date(now + 60000).toUTCString() : guidance,
        } })
        : new Response('{}', { status: 200 });
    };
    const origin = `https://${guidance}.example`;
    assert.equal((await fetchWithRetry(`${origin}/notifications`)).status, 429);
    assert.equal(calls, 1, '429 must not trigger rapid read retries');
    assert.equal((await fetchWithRetry(`${origin}/push-token`, { method: 'POST' })).status, 429);
    assert.equal(calls, 1, 'registration must respect the same origin cooldown');
    now += 61000;
    assert.equal((await fetchWithRetry(`${origin}/preferences`)).status, 200);
    assert.equal(calls, 2, 'the rejected mutation must not be replayed');
  }
  const controller = new AbortController();
  controller.abort();
  await assert.rejects(fetchWithRetry('https://cancel.example', { signal: controller.signal }));
  console.log('429 cooldown, HTTP-date guidance, no mutation replay and cancellation checks passed.');
} finally {
  Date.now = originalNow;
  globalThis.fetch = originalFetch;
}
