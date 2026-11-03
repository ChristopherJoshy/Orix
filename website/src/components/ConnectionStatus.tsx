import { useEffect, useRef, useState } from "react";
import { apiOrigin } from "../lib/session";
import { fetchWithRetry } from "../lib/retry";
import { refreshEvent } from "../lib/realtime";
import { PixelStatus } from "./PixelStatus";
import { Button } from "./ui";
export function ConnectionStatus() {
  const [lost, setLost] = useState(!navigator.onLine);
  const [busy, setBusy] = useState(false);
  const pending = useRef(false);
  const alive = useRef(true);
  async function retry() {
    if (pending.current) return;
    pending.current = true;
    setBusy(true);
    try {
      const response = await fetchWithRetry(`${apiOrigin()}/health`, {
        credentials: "omit",
        redirect: "error",
      });
      if (alive.current && response.ok) {
        setLost(false);
        window.dispatchEvent(
          new CustomEvent(refreshEvent, { detail: { type: "resync" } }),
        );
      }
    } catch {
      /* Curated connection copy stays visible. */
    } finally {
      pending.current = false;
      if (alive.current) setBusy(false);
    }
  }
  useEffect(() => {
    alive.current = true;
    const offline = () => setLost(true);
    const online = () => {
      void retry();
    };
    const status = (event: Event) =>
      setLost(!(event as CustomEvent<{ connected: boolean }>).detail.connected);
    window.addEventListener("offline", offline);
    window.addEventListener("online", online);
    window.addEventListener("orix:connection", status);
    return () => {
      alive.current = false;
      window.removeEventListener("offline", offline);
      window.removeEventListener("online", online);
      window.removeEventListener("orix:connection", status);
    };
  }, []);
  return lost ? (
    <aside className="connection-notice" aria-label="Connection lost">
      <PixelStatus state="connection-lost" compact label="Connection lost" />
      <div role="status">
        <strong>Connection lost.</strong>
        <p>
          The network’s on a chaya break. Check your connection and try again.
        </p>
      </div>
      <Button disabled={busy} onClick={() => void retry()}>
        {busy ? "Reconnecting…" : "Retry"}
      </Button>
    </aside>
  ) : null;
}
