import { PublicError } from "./errors";
import { useCallback, useEffect, useRef, useState } from "react";
import { apiRequestPage, useSession } from "./session";
import { refreshEvent, useRealtimeRefresh } from "./realtime";
import { useResource } from "./resource";
import type { SelectedPlace } from "./types";

export interface Journey {
  request_id: string;
  pickup_place_id: string;
  drop_place_id: string;
  pickup_label: string;
  drop_label: string;
  pickup: { coordinates: [number, number] };
  drop: { coordinates: [number, number] };
  date: string;
  time_window: { start: string; end: string };
  timezone_offset_minutes: number;
  departure_start: string;
  departure_end: string;
  created_at: string;
  expires_at: string;
  status: string;
  group_id: string | null;
  female_only: boolean;
  allow_riders: boolean;
  mode: string;
  driver: boolean;
  seats: number;
}
export interface MatchingStatus {
  has_active_request: boolean;
  request?: Journey;
  group_id?: string | null;
  group_size?: number;
  group_status?: string;
  my_readiness_confirmed?: boolean;
}
export interface JourneyInput {
  pickup_place_id: string;
  drop_place_id: string;
  pickup_label: string;
  drop_label: string;
  pickup_lat: number;
  pickup_lng: number;
  drop_lat: number;
  drop_lng: number;
  date: string;
  time_window_start: string;
  time_window_end: string;
  timezone_offset_minutes: number;
  female_only: boolean;
  allow_riders: boolean;
  mode: "pool" | "lift";
  driver: boolean;
  seats: number;
}
export function refreshJourneys() {
  window.dispatchEvent(
    new CustomEvent(refreshEvent, { detail: { type: "ride_update" } }),
  );
}
export function journeyInput(
  from: SelectedPlace,
  to: SelectedPlace,
  date: string,
  time: string,
  minutes: number,
  women: boolean,
  lift: boolean,
): JourneyInput {
  const start = new Date(`${date}T${time}`);
  const end = new Date(start.getTime() + minutes * 60000);
  if (!Number.isFinite(start.getTime()) || start.getTime() <= Date.now())
    throw new PublicError("Choose a future departure time.");
  if (!Number.isInteger(minutes) || minutes < 1 || minutes > 120)
    throw new PublicError("Choose a departure window of 1–120 minutes.");
  if (
    start.getFullYear() !== end.getFullYear() ||
    start.getMonth() !== end.getMonth() ||
    start.getDate() !== end.getDate() ||
    start.getTimezoneOffset() !== end.getTimezoneOffset()
  )
    throw new PublicError(
      "Choose a time window within one calendar day and timezone offset.",
    );
  if (from.id === to.id)
    throw new PublicError("Choose a different destination.");
  return {
    pickup_place_id: from.id,
    drop_place_id: to.id,
    pickup_label: from.label,
    drop_label: to.label,
    pickup_lat: from.latitude,
    pickup_lng: from.longitude,
    drop_lat: to.latitude,
    drop_lng: to.longitude,
    date,
    time_window_start: time,
    time_window_end: `${String(end.getHours()).padStart(2, "0")}:${String(end.getMinutes()).padStart(2, "0")}`,
    timezone_offset_minutes: -start.getTimezoneOffset(),
    female_only: women,
    allow_riders: lift,
    mode: "pool",
    driver: false,
    seats: 0,
  };
}
export function useMatchingStatus() {
  const { data: status, ...rest } = useResource<MatchingStatus>(
    "/rides/status",
    15000,
  );
  return { status, ...rest };
}
export function useJourneyHistory() {
  const owner = useSession((s) => s.account?.user_id);
  const sequence = useRef(0);
  const loadingRef = useRef(false);
  const [rows, setRows] = useState<Journey[] | null>(null);
  const [cursor, setCursor] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const load = useCallback(
    async (before?: string) => {
      // A refresh supersedes pagination so old pages cannot overwrite a newer snapshot.
      if (!owner || (before && loadingRef.current)) return;
      const revision = ++sequence.current;
      loadingRef.current = true;
      setLoading(true);
      try {
        const result = await apiRequestPage<Journey[]>(
          `/rides?include_history=true&limit=50${before ? `&before=${encodeURIComponent(before)}` : ""}`,
        );
        if (
          revision === sequence.current &&
          owner === useSession.getState().account?.user_id
        ) {
          setRows((previous) =>
            before
              ? [
                  ...new Map(
                    [...(previous ?? []), ...result.data].map((row) => [
                      row.request_id,
                      row,
                    ]),
                  ).values(),
                ]
              : result.data,
          );
          setCursor(result.nextCursor);
          setError("");
        }
      } catch (e) {
        if (revision === sequence.current)
          setError(e instanceof Error ? e.message : "Could not load journeys.");
      } finally {
        if (revision === sequence.current) {
          loadingRef.current = false;
          setLoading(false);
        }
      }
    },
    [owner],
  );
  const refresh = useCallback(() => {
    void load();
  }, [load]);
  useRealtimeRefresh(refresh);
  useEffect(() => {
    setRows(null);
    setCursor(null);
    setError("");
    void load();
    return () => {
      sequence.current++;
      loadingRef.current = false;
    };
  }, [load]);
  return { rows, cursor, error, loading, load };
}
export function canEditJourney(status: MatchingStatus | null) {
  return (
    !!status?.request &&
    (["pending", "matching"].includes(status.request.status) ||
      (status.request.status === "matched" &&
        status.group_status === "pending" &&
        status.group_size === 1 &&
        !status.my_readiness_confirmed))
  );
}
export function journeyTime(ride: Journey) {
  return `${new Date(ride.departure_start).toLocaleString()} – ${new Date(ride.departure_end).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}`;
}
