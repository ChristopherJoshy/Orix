const transient = new Set([408, 425, 429, 500, 502, 503, 504]);
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
  for (let attempt = 0; attempt < attempts; attempt++) {
    init.signal?.throwIfAborted();
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
    if (!transient.has(response.status) || attempt === attempts - 1)
      return response;
    const guidance = response.headers.get("Retry-After");
    const parsed = guidance
      ? Number.isFinite(Number(guidance))
        ? Number(guidance) * 1000
        : Date.parse(guidance) - Date.now()
      : 0;
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
