import { useEffect, useRef, useState } from "react";
import { apiRequest, useSession } from "../lib/session";

export function CurrentCity() {
  const owner = useSession((s) => s.account?.user_id);
  const [city, setCity] = useState("");
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState(false);
  const version = useRef(0);
  const locating = useRef(false);
  const locate = useRef<() => void>(() => {});
  useEffect(() => {
    const revision = ++version.current;
    setCity("");
    setBusy(false);
    setFailed(false);
    locating.current = false;
    let permission: PermissionStatus | undefined;
    let timer: number | undefined;
    const valid = () =>
      revision === version.current &&
      owner === useSession.getState().account?.user_id;
    const update = () => {
      if (!valid() || locating.current || !owner) return;
      if (!navigator.geolocation) {
        setFailed(true);
        return;
      }
      locating.current = true;
      setBusy(true);
      setFailed(false);
      navigator.geolocation.getCurrentPosition(
        async (position) => {
          try {
            if (!valid()) return;
            const result = await apiRequest<{ city: string }>("/places/city", {
              method: "POST",
              body: JSON.stringify({
                lat: position.coords.latitude,
                lng: position.coords.longitude,
              }),
            });
            if (valid()) setCity(result.city);
          } catch {
            if (valid()) {
              setCity("");
              setFailed(true);
            }
          } finally {
            if (valid()) {
              locating.current = false;
              setBusy(false);
            }
          }
        },
        () => {
          if (valid()) {
            locating.current = false;
            setBusy(false);
            setCity("");
            setFailed(true);
          }
        },
        { timeout: 10000, maximumAge: 60000 },
      );
    };
    locate.current = update;
    if (navigator.permissions)
      void navigator.permissions
        .query({ name: "geolocation" })
        .then((status) => {
          if (!valid()) return;
          permission = status;
          const sync = () => {
            if (status.state === "granted") update();
            else if (valid()) setCity("");
          };
          status.onchange = sync;
          sync();
          timer = window.setInterval(() => {
            if (!document.hidden && status.state === "granted") update();
          }, 300000);
        })
        .catch(() => {});
    return () => {
      version.current++;
      if (permission) permission.onchange = null;
      if (timer) clearInterval(timer);
    };
  }, [owner]);
  return (
    <button
      type="button"
      className="city-chip city-location"
      disabled={busy}
      onClick={() => locate.current()}
      title={
        city
          ? "Update your current city"
          : "Use your location to show your city"
      }
    >
      <span className="tiny-pixel" />
      {busy
        ? "Locating…"
        : city || (failed ? "Retry location" : "Set location")}
    </button>
  );
}
