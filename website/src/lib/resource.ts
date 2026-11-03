import { friendlyError } from "./errors";
import { useCallback, useEffect, useRef, useState } from "react";
import { apiRequest, useSession } from "./session";
import { useRealtimeRefresh } from "./realtime";

/** Serial account-scoped reads retain one trailing invalidation during a slow fetch. */
export function useResource<T>(
  path: string | null,
  pollMs = 30000,
  request: (path: string) => Promise<T> = apiRequest,
) {
  const owner = useSession((s) =>
    s.account?.onboarding_completed ? s.account.user_id : null,
  );
  const version = useRef(0);
  const pending = useRef(false);
  const queued = useRef(false);
  const [snapshot, setSnapshot] = useState<{
    owner: string;
    path: string;
    value: T;
  } | null>(null);
  const data =
    snapshot?.owner === owner && snapshot.path === path ? snapshot.value : null;
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const load = useCallback(
    async function read() {
      if (!path || !owner) return;
      if (pending.current) {
        queued.current = true;
        return;
      }
      pending.current = true;
      const revision = ++version.current;
      setLoading(true);
      try {
        const value = await request(path);
        if (
          revision === version.current &&
          owner === useSession.getState().account?.user_id
        ) {
          setSnapshot({ owner, path, value });
          setError("");
        }
      } catch (problem) {
        if (
          revision === version.current &&
          owner === useSession.getState().account?.user_id
        )
          setError(friendlyError(problem));
      } finally {
        if (revision === version.current) {
          pending.current = false;
          setLoading(false);
          if (queued.current) {
            queued.current = false;
            void read();
          }
        }
      }
    },
    [owner, path, request],
  );
  useRealtimeRefresh(load);
  useEffect(() => {
    version.current++;
    pending.current = false;
    queued.current = false;
    setSnapshot(null);
    setError("");
    void load();
    const timer = pollMs
      ? window.setInterval(() => {
          if (!document.hidden && !pending.current) void load();
        }, pollMs)
      : undefined;
    return () => {
      version.current++;
      queued.current = false;
      if (timer !== undefined) window.clearInterval(timer);
    };
  }, [load, pollMs]);
  return { data, error, loading, load };
}
