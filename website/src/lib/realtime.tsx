import { useEffect } from "react";
import { apiOrigin, firebaseAuth, useSession, verifyAccount } from "./session";
export const refreshEvent = "orix:refresh";
function refresh(type: string) {
  window.dispatchEvent(new CustomEvent(refreshEvent, { detail: { type } }));
}

/** Authenticated invalidations trigger authorized REST reads; no private data is cached here. */
export function RealtimeBridge() {
  const account = useSession((s) => s.account);
  useEffect(() => {
    if (!account?.onboarding_completed) return;
    let alive = true;
    let socket: WebSocket | null = null;
    let timer: number | undefined;
    let attempts = 0;
    let connecting = false;
    let lastMessage = Date.now();
    const owner = account.user_id;
    async function connect() {
      if (
        !alive ||
        connecting ||
        document.hidden ||
        socket?.readyState === WebSocket.OPEN ||
        socket?.readyState === WebSocket.CONNECTING
      )
        return;
      const user = firebaseAuth.currentUser;
      if (!user) return;
      connecting = true;
      try {
        const token = await user.getIdToken();
        if (
          !alive ||
          firebaseAuth.currentUser?.uid !== user.uid ||
          useSession.getState().account?.user_id !== owner
        )
          return;
        const url = new URL("/ws", apiOrigin());
        url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
        const next = new WebSocket(url, ["orix.v1", `firebase.${token}`]);
        socket = next;
        next.onopen = () => {
          if (!alive || socket !== next) return;
          attempts = 0;
          lastMessage = Date.now();
          refresh("connection");
          void verifyAccount().catch(() => {});
        };
        next.onmessage = (event) => {
          if (!alive || socket !== next || typeof event.data !== "string")
            return;
          lastMessage = Date.now();
          try {
            const value = JSON.parse(event.data);
            if (typeof value.type === "string" && value.type !== "pong") {
              refresh(value.type);
              if (value.type === "profile_update")
                void verifyAccount().catch(() => {});
            }
          } catch {
            /* Malformed frames do not replace REST state. */
          }
        };
        next.onerror = () => next.close();
        next.onclose = () => {
          if (socket === next) socket = null;
          schedule();
        };
      } catch {
        schedule();
      } finally {
        connecting = false;
      }
    }
    function schedule() {
      if (!alive || document.hidden || timer !== undefined) return;
      const delay =
        Math.min(30000, 1000 * 2 ** Math.min(attempts++, 5)) +
        Math.random() * 300;
      timer = window.setTimeout(() => {
        timer = undefined;
        void connect();
      }, delay);
    }
    const visible = () => {
      if (document.hidden) {
        socket?.close();
        return;
      }
      if (timer !== undefined) {
        window.clearTimeout(timer);
        timer = undefined;
      }
      refresh("connection");
      void connect();
    };
    const heartbeat = window.setInterval(() => {
      if (socket?.readyState === WebSocket.OPEN) {
        if (Date.now() - lastMessage > 75000) socket.close();
        else socket.send(JSON.stringify({ type: "ping" }));
      }
    }, 30000);
    document.addEventListener("visibilitychange", visible);
    window.addEventListener("online", visible);
    void connect();
    return () => {
      alive = false;
      if (timer !== undefined) window.clearTimeout(timer);
      window.clearInterval(heartbeat);
      socket?.close();
      document.removeEventListener("visibilitychange", visible);
      window.removeEventListener("online", visible);
    };
  }, [account?.user_id, account?.onboarding_completed]);
  return null;
}
export function useRealtimeRefresh(callback: () => void) {
  useEffect(() => {
    let timer: number | undefined;
    const listener = () => {
      if (timer !== undefined) return;
      timer = window.setTimeout(() => {
        timer = undefined;
        callback();
      }, 150);
    };
    window.addEventListener(refreshEvent, listener);
    return () => {
      window.removeEventListener(refreshEvent, listener);
      if (timer !== undefined) window.clearTimeout(timer);
    };
  }, [callback]);
}
