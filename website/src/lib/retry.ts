const transient = new Set([408, 425, 429, 500, 502, 503, 504]);
// Quotas are shared by endpoints at an origin. Polls and registration must not
// start another request while the server has told this tab to back off.
const cooldowns = new Map<string, number>();
function retryDelay(guidance: string | null): number {
  if (!guidance) return 0;
  const value = Number.isFinite(Number(guidance))
    ? Number(guidance) * 1000
    : Date.parse(guidance) - Date.now();
  return Number.isFinite(value) ? Math.max(0, value) : 0;
}
async function wait(ms: number, signal?: AbortSignal | null) {
  signal?.throwIfAborted();
  await new Promise<void>((resolve, reject) => {
    const abort = () => {
      clearTimeout(timer);
      reject(signal?.reason ?? new DOMException("Cancelled", "AbortError"));
    };
    const timer = setTimeout(() => {
      signal?.removeEventListener("abort", abort);
      resolve();
    }, ms);
    signal?.addEventListener("abort", abort, { once: true });
  });
}
/** Retry reads only. Writes require server idempotency before automatic replay. */
export async function fetchWithRetry(
  url: string,
  init: RequestInit = {},
): Promise<Response> {
  const attempts = (init.method ?? "GET").toUpperCase() === "GET" ? 3 : 1;
  const origin = new URL(url, window.location.origin).origin;
  for (let attempt = 0; attempt < attempts; attempt++) {
    init.signal?.throwIfAborted();
    const remaining = (cooldowns.get(origin) ?? 0) - Date.now();
    if (remaining > 0)
      return new Response(null, {
        status: 429,
        headers: { "Retry-After": String(Math.ceil(remaining / 1000)) },
      });
    cooldowns.delete(origin);
    let response: Response;
    try {
      response = await fetch(url, {
        ...init,
        signal: init.signal
          ? AbortSignal.any([init.signal, AbortSignal.timeout(10000)])
          : AbortSignal.timeout(10000),
      });
    } catch (problem) {
      if (init.signal?.aborted || attempt === attempts - 1) {
        if (!init.signal?.aborted)
          window.dispatchEvent(
            new CustomEvent("orix:connection", {
              detail: { connected: false },
            }),
          );
        throw problem;
      }
      await wait(500 * 2 ** attempt + Math.random() * 150, init.signal);
      continue;
    }
    window.dispatchEvent(
      new CustomEvent("orix:connection", { detail: { connected: true } }),
    );
    const parsed = retryDelay(response.headers.get("Retry-After"));
    if (response.status === 429) {
      // A missing/unexposed header is not permission to hammer the API.
      cooldowns.set(origin, Date.now() + (parsed || 60000));
      return response;
    }
    if (!transient.has(response.status) || attempt === attempts - 1)
      return response;
    // Long server cooldowns are surfaced to the user instead of shortened.
    if (parsed > 10000) return response;
    await response.body?.cancel();
    await wait(
      Math.max(
        500 * 2 ** attempt + Math.random() * 150,
        Number.isFinite(parsed) ? parsed : 0,
      ),
      init.signal,
    );
  }
  throw new Error(
    "The service could not be reached. Retry when your connection is available.",
  );
}
