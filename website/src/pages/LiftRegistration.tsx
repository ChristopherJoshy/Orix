import { useEffect, useRef, useState } from "react";
import { useSession, apiRequest } from "../lib/session";
import { useResource } from "../lib/resource";
import { friendlyError } from "../lib/errors";
import { refreshJourneys, type JourneyInput } from "../lib/journeys";
import type { SelectedPlace } from "../lib/types";
import { PlaceAutocomplete } from "../components/PlaceAutocomplete";
import { Button, Field, PageHead, Toggle } from "../components/ui";
import { PixelStatus } from "../components/PixelStatus";
import { CreateRide } from "./Journeys";

type Offer = {
  revision: number;
  route: JourneyInput;
  schedule: {
    time_zone: string;
    start_minute: number;
    end_minute: number;
    weekdays: number[];
  };
  active: boolean;
  next_departure: string;
};
type Response = { offer: Offer | null };
const names = [
  "Sunday",
  "Monday",
  "Tuesday",
  "Wednesday",
  "Thursday",
  "Friday",
  "Saturday",
];
function hhmm(value: number) {
  return `${String(Math.floor(value / 60)).padStart(2, "0")}:${String(value % 60).padStart(2, "0")}`;
}
function minutes(value: string) {
  const [h, m] = value.split(":").map(Number);
  return h * 60 + m;
}
export default function LiftRegistration() {
  const owner = useSession((s) => s.account?.user_id);
  const { data, error, loading, load } =
    useResource<Response>("/lifts/recurring");
  const [editing, setEditing] = useState(false);
  const [from, setFrom] = useState<SelectedPlace | null>(null);
  const [to, setTo] = useState<SelectedPlace | null>(null);
  const [start, setStart] = useState("09:00");
  const [duration, setDuration] = useState(30);
  const [seats, setSeats] = useState(2);
  const [women, setWomen] = useState(false);
  const [weekdays, setWeekdays] = useState<number[]>([1, 2, 3, 4, 5]);
  const [zone, setZone] = useState(
    Intl.DateTimeFormat().resolvedOptions().timeZone || "",
  );
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState("");
  const [saved, setSaved] = useState("");
  const sending = useRef(false);
  const editRevision = useRef(0);
  const editActive = useRef(true);
  useEffect(() => {
    setEditing(false);
    setFrom(null);
    setTo(null);
    setStart("09:00");
    setDuration(30);
    setSeats(2);
    setWomen(false);
    setWeekdays([1, 2, 3, 4, 5]);
    setZone(Intl.DateTimeFormat().resolvedOptions().timeZone || "");
    setFailure("");
    setSaved("");
    setBusy(false);
    sending.current = false;
  }, [owner]);
  function edit() {
    editRevision.current = data?.offer?.revision ?? 0;
    const offer = data?.offer;
    editActive.current = offer?.active ?? true;
    if (offer) {
      const route = offer.route;
      setFrom({
        id: route.pickup_place_id,
        label: route.pickup_label,
        latitude: route.pickup_lat,
        longitude: route.pickup_lng,
      });
      setTo({
        id: route.drop_place_id,
        label: route.drop_label,
        latitude: route.drop_lat,
        longitude: route.drop_lng,
      });
      setStart(hhmm(offer.schedule.start_minute));
      setDuration(offer.schedule.end_minute - offer.schedule.start_minute);
      setSeats(route.seats);
      setWomen(route.female_only);
      setWeekdays(offer.schedule.weekdays);
      setZone(offer.schedule.time_zone);
    }
    setFailure("");
    setEditing(true);
  }
  async function put(offer: Offer) {
    if (sending.current) return;
    sending.current = true;
    setBusy(true);
    setFailure("");
    setSaved("");
    try {
      await apiRequest("/lifts/recurring", {
        method: "PUT",
        body: JSON.stringify({
          route: offer.route,
          schedule: offer.schedule,
          active: offer.active,
          revision: offer.revision,
        }),
      });
      if (owner === useSession.getState().account?.user_id) {
        setEditing(false);
        setSaved(
          offer.active ? "Recurring offer saved." : "Recurring offer paused.",
        );
        refreshJourneys();
        await load();
      }
    } catch (problem) {
      if (owner === useSession.getState().account?.user_id)
        setFailure(friendlyError(problem));
    } finally {
      if (owner === useSession.getState().account?.user_id) {
        sending.current = false;
        setBusy(false);
      }
    }
  }
  function save(event: React.FormEvent) {
    event.preventDefault();
    if (
      !from ||
      !to ||
      from.id === to.id ||
      !weekdays.length ||
      !zone ||
      !Number.isInteger(seats) ||
      seats < 1 ||
      seats > 8
    ) {
      setFailure(
        "Choose different places, at least one day, a timezone and 1–8 seats.",
      );
      return;
    }
    const begin = minutes(start);
    if (!Number.isInteger(begin) || begin < 0 || begin + duration >= 1440) {
      setFailure("Choose a departure window within the same day.");
      return;
    }
    const route: JourneyInput = {
      pickup_place_id: from.id,
      drop_place_id: to.id,
      pickup_label: from.label,
      drop_label: to.label,
      pickup_lat: from.latitude,
      pickup_lng: from.longitude,
      drop_lat: to.latitude,
      drop_lng: to.longitude,
      date: "",
      time_window_start: "",
      time_window_end: "",
      timezone_offset_minutes: 0,
      female_only: women,
      allow_riders: true,
      mode: "lift",
      driver: true,
      seats,
    };
    void put({
      route,
      revision: editRevision.current,
      schedule: {
        time_zone: zone,
        start_minute: begin,
        end_minute: begin + duration,
        weekdays,
      },
      active: editActive.current,
      next_departure: "",
    });
  }
  return (
    <>
      <PageHead
        title="Offer a lift."
        subtitle="Share a seat on a route you already travel."
      />
      <section className="panel">
        <h2>Recurring availability</h2>
        <p>
          Set your route and days. An active offer creates a journey before each
          departure when you have no other active journey. Pause it whenever you
          need to.
        </p>
        {error && (
          <p role="alert">
            {error}{" "}
            <Button variant="secondary" onClick={() => void load()}>
              Retry
            </Button>
          </p>
        )}
        {failure && <p role="alert">{failure}</p>}
        {saved && <p role="status">{saved}</p>}
        {!data && !error ? (
          <PixelStatus state="loading" label="Loading recurring offer" />
        ) : data?.offer && !editing ? (
          <>
            <p>
              {data.offer.route.pickup_label} → {data.offer.route.drop_label}
            </p>
            <p>
              {data.offer.schedule.weekdays.map((day) => names[day]).join(", ")}{" "}
              · {hhmm(data.offer.schedule.start_minute)}–
              {hhmm(data.offer.schedule.end_minute)} ·{" "}
              {data.offer.schedule.time_zone}
            </p>
            <p>
              {data.offer.active
                ? `Active · next departure ${new Date(data.offer.next_departure).toLocaleString()}`
                : "Paused"}
            </p>
            <div className="intro-actions">
              <Button
                variant="secondary"
                disabled={busy || loading}
                onClick={edit}
              >
                Edit schedule
              </Button>
              <Button
                variant="secondary"
                disabled={busy || loading}
                onClick={() =>
                  void put({ ...data.offer!, active: !data.offer!.active })
                }
              >
                {data.offer.active ? "Pause future offers" : "Resume offer"}
              </Button>
            </div>
            <p>Pausing does not cancel a journey already created.</p>
          </>
        ) : !editing && data ? (
          <Button disabled={loading} onClick={edit}>
            Set up recurring offer
          </Button>
        ) : null}
        {editing && (
          <form onSubmit={save}>
            <PlaceAutocomplete
              label="Starting from"
              pickup
              value={from}
              onChange={setFrom}
              disabled={busy}
            />
            <PlaceAutocomplete
              label="Going to"
              value={to}
              onChange={setTo}
              disabled={busy}
            />
            <Field label="Departure time">
              <input
                type="time"
                value={start}
                onChange={(event) => setStart(event.target.value)}
                required
                disabled={busy}
              />
            </Field>
            <Field label="Departure window">
              <select
                value={duration}
                onChange={(event) => setDuration(Number(event.target.value))}
                disabled={busy}
              >
                {[15, 30, 60, 120].map((value) => (
                  <option key={value} value={value}>
                    {value} minutes
                  </option>
                ))}
              </select>
            </Field>
            <Field label="Passenger seats">
              <input
                type="number"
                min={1}
                max={8}
                value={seats}
                onChange={(event) => setSeats(Number(event.target.value))}
                required
                disabled={busy}
              />
            </Field>
            <Field label="IANA timezone">
              <input
                value={zone}
                onChange={(event) => setZone(event.target.value)}
                required
                disabled={busy}
              />
            </Field>
            <fieldset disabled={busy}>
              <legend>Travel days</legend>
              {names.map((day, index) => (
                <label key={day} className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={weekdays.includes(index)}
                    onChange={(event) =>
                      setWeekdays((current) =>
                        event.target.checked
                          ? [...current, index].sort()
                          : current.filter((value) => value !== index),
                      )
                    }
                  />
                  <span>{day}</span>
                </label>
              ))}
            </fieldset>
            <Toggle
              label="Women-only group"
              checked={women}
              onChange={setWomen}
            />
            <div className="intro-actions">
              <Button disabled={busy}>
                {busy
                  ? "Saving…"
                  : data?.offer
                    ? "Save changes"
                    : "Save and activate"}
              </Button>
              <Button
                type="button"
                variant="secondary"
                disabled={busy}
                onClick={() => setEditing(false)}
              >
                Cancel
              </Button>
            </div>
          </form>
        )}
      </section>
      <CreateRide offer />
    </>
  );
}
