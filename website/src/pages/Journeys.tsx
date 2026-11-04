import { friendlyError } from "../lib/errors";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  Link,
  useNavigate,
  useParams,
  useSearchParams,
} from "react-router-dom";
import { ArrowRight, ArrowUpRight, Plus, ShieldCheck } from "lucide-react";
import { apiRequest, useSession } from "../lib/session";
import { useRealtimeRefresh } from "../lib/realtime";
import {
  canEditJourney,
  journeyInput,
  journeyTime,
  refreshJourneys,
  useJourneyHistory,
  useMatchingStatus,
} from "../lib/journeys";
import type { Journey } from "../lib/journeys";
import type { SelectedPlace } from "../lib/types";
import { localDate } from "../lib/data";
import {
  Badge,
  Button,
  Empty,
  Field,
  LinkButton,
  PageHead,
  Toggle,
} from "../components/ui";
import { PixelMark, RouteArt } from "../components/PixelArt";
import { PixelStatus } from "../components/PixelStatus";
import { PlaceAutocomplete } from "../components/PlaceAutocomplete";
import { Modal } from "../components/Modal";

function JourneyCard({ ride }: { ride: Journey }) {
  return (
    <section className="panel">
      <Badge>
        {ride.status === "matched" ? "Group assigned" : ride.status}
      </Badge>
      <h2>
        {ride.pickup_label} → {ride.drop_label}
      </h2>
      <p>{journeyTime(ride)}</p>
      <p>
        {ride.female_only ? "Women-only group" : "All eligible travellers"} ·{" "}
        {ride.allow_riders ? "Open to a lift" : "No lifts"}
      </p>
      <LinkButton
        to={`/app/rides/${encodeURIComponent(ride.request_id)}`}
        secondary
      >
        View journey <ArrowRight size={16} />
      </LinkButton>
    </section>
  );
}
function LoadError({ error, retry }: { error: string; retry: () => void }) {
  return error ? (
    <div className="inline-notice" role="alert">
      {error}{" "}
      <Button variant="secondary" onClick={retry}>
        Retry
      </Button>
    </div>
  ) : null;
}

export function Dashboard() {
  const account = useSession((s) => s.account);
  const { status, error, load } = useMatchingStatus();
  return (
    <>
      <PageHead
        title={`Hello, ${account?.display_name.split(" ")[0] ?? "traveller"}.`}
        subtitle="Your route. Your people."
      >
        <LinkButton to="/app/create">
          <Plus size={16} /> Plan a journey
        </LinkButton>
      </PageHead>
      <LoadError error={error} retry={() => void load()} />
      <div className="dashboard-grid dashboard-home">
        <section className="dashboard-journey">
          <div className="section-title">
            <h2>Your journey</h2>
            <Link to="/app/rides">All journeys →</Link>
          </div>
          {!status && !error ? (
            <PixelStatus state="loading" label="Refreshing your journey" />
          ) : status?.request ? (
            <JourneyCard ride={status.request} />
          ) : status ? (
            <Empty
              title="Where are you heading?"
              text="Choose your route, departure window and preferences."
            >
              <LinkButton to="/app/create">
                Find my pool <ArrowRight size={16} />
              </LinkButton>
            </Empty>
          ) : null}
        </section>
        <section className="dashboard-support">
          <Link to="/app/safety" className="safety-panel">
            <ShieldCheck size={28} />
            <div>
              <h3>Your comfort comes first.</h3>
              <p>Manage blocks, report a concern, and keep your boundaries.</p>
            </div>
            <ArrowUpRight size={20} />
          </Link>
          <div className="panel">
            <h2>Have a seat to share?</h2>
            <p>Register your route and availability to offer a lift.</p>
            <LinkButton to="/app/rider" secondary>
              Register a lift
            </LinkButton>
          </div>
          <Link className="text-link" to="/app/notifications">
            Your updates <ArrowRight size={16} />
          </Link>
        </section>
      </div>
    </>
  );
}
export function Explore() {
  const { status, error, loading, load } = useMatchingStatus();
  const ride = status?.request;
  const searching = !!ride && canEditJourney(status);
  return (
    <>
      <PageHead
        title="Your route. Your pool."
        subtitle="Matching follows your preferences"
      >
        <Button
          variant="secondary"
          disabled={loading}
          onClick={() => void load()}
        >
          Refresh
        </Button>
      </PageHead>
      <LoadError error={error} retry={() => void load()} />
      {!status && !error ? (
        <PixelStatus state="loading" label="Refreshing matching" />
      ) : ride ? (
        <>
          <PixelStatus
            state={searching ? "searching" : "found"}
            label={
              searching
                ? "Looking for compatible people"
                : "Your group is ready"
            }
          />
          <JourneyCard ride={ride} />
          {status?.group_id && (
            <LinkButton
              to={`/app/groups/${encodeURIComponent(status.group_id)}`}
            >
              Meet your group <ArrowRight size={16} />
            </LinkButton>
          )}
          <p>
            Matching considers nearby starts and destinations, departure
            windows and everyone’s safety preferences.
          </p>
        </>
      ) : status ? (
        <Empty
          title="Start with your route."
          text="Tell us where and when. We’ll find people going your way."
        >
          <LinkButton to="/app/create">Find my pool</LinkButton>
        </Empty>
      ) : null}
    </>
  );
}
export function CreateRide({ offer = false }: { offer?: boolean }) {
  const [params] = useSearchParams();
  const editing = params.get("edit");
  const navigate = useNavigate();
  const { status, error: statusError, load } = useMatchingStatus();
  const [from, setFrom] = useState<SelectedPlace | null>(null);
  const [to, setTo] = useState<SelectedPlace | null>(null);
  const [date, setDate] = useState(localDate(1));
  const [time, setTime] = useState("09:00");
  const [flexibility, setFlexibility] = useState("15");
  const [seats, setSeats] = useState(2);
  const [women, setWomen] = useState(false);
  const [lift, setLift] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const initialized = useRef<string | null>(null);
  const saving = useRef(false);
  const owner = useSession((s) => s.account?.user_id);
  useEffect(() => {
    if (
      !editing ||
      initialized.current === editing ||
      status?.request?.request_id !== editing
    )
      return;
    const ride = status.request;
    initialized.current = editing;
    setFrom({
      id: ride.pickup_place_id,
      label: ride.pickup_label,
      latitude: ride.pickup.coordinates[1],
      longitude: ride.pickup.coordinates[0],
    });
    setTo({
      id: ride.drop_place_id,
      label: ride.drop_label,
      latitude: ride.drop.coordinates[1],
      longitude: ride.drop.coordinates[0],
    });
    const start = new Date(ride.departure_start);
    setDate(
      `${start.getFullYear()}-${String(start.getMonth() + 1).padStart(2, "0")}-${String(start.getDate()).padStart(2, "0")}`,
    );
    setTime(
      `${String(start.getHours()).padStart(2, "0")}:${String(start.getMinutes()).padStart(2, "0")}`,
    );
    setFlexibility(
      String(
        Math.round(
          (new Date(ride.departure_end).getTime() - start.getTime()) / 60000,
        ),
      ),
    );
    setWomen(ride.female_only);
    setLift(ride.allow_riders);
    setSeats(ride.seats || 2);
  }, [editing, status]);
  const allowed =
    !!status &&
    (editing
      ? status.request?.request_id === editing && canEditJourney(status)
      : !status.has_active_request);
  async function save(event: React.FormEvent) {
    event.preventDefault();
    if (!allowed || saving.current || !from || !to) return;
    saving.current = true;
    setBusy(true);
    setError("");
    try {
      const input = journeyInput(
        from,
        to,
        date,
        time,
        Number(flexibility),
        women,
        lift,
      );
      if (offer || (editing && status?.request?.driver)) {
        input.mode = "lift";
        input.driver = true;
        input.seats = seats;
      }
      const ride = await apiRequest<Journey>(
        editing ? `/rides/${encodeURIComponent(editing)}` : "/rides",
        { method: editing ? "PUT" : "POST", body: JSON.stringify(input) },
      );
      if (owner !== useSession.getState().account?.user_id) return;
      refreshJourneys();
      navigate(`/app/rides/${encodeURIComponent(ride.request_id)}`, {
        replace: !!editing,
      });
    } catch (problem) {
      if (owner === useSession.getState().account?.user_id) {
        setError(friendlyError(problem));
        void load();
      }
    } finally {
      saving.current = false;
      if (owner === useSession.getState().account?.user_id) setBusy(false);
    }
  }
  return (
    <>
      <PageHead
        title={
          editing
            ? "Adjust your journey."
            : offer
              ? "Offer a seat."
              : "Where to, together?"
        }
        subtitle={
          offer
            ? "Register your route, time and available seats"
            : "One request. Automatic matching."
        }
      />
      <LoadError error={statusError} retry={() => void load()} />
      {status && !allowed && (
        <div className="inline-notice">
          {editing
            ? "This journey can no longer be edited."
            : "You already have an active journey."}{" "}
          <Link to="/app/explore">View matching</Link>
        </div>
      )}
      <div className="create-layout">
        <section className="panel create-panel unified-request">
          <div className="request-intro">
            <PixelMark />
            <div>
              <h2>Just your route.</h2>
              <p>We’ll find the people who fit.</p>
            </div>
          </div>
          <form onSubmit={save}>
            <fieldset
              disabled={busy || !allowed}
              style={{ border: 0, padding: 0, margin: 0, minWidth: 0 }}
            >
              <div className="route-fields">
                <PlaceAutocomplete
                  label="Starting from"
                  value={from}
                  onChange={setFrom}
                  pickup
                  disabled={busy || !allowed}
                />
                <PlaceAutocomplete
                  label="Going to"
                  value={to}
                  onChange={setTo}
                  disabled={busy || !allowed}
                />
              </div>
              <div className="form-row">
                <Field label="Travel date">
                  <input
                    type="date"
                    min={localDate()}
                    value={date}
                    onChange={(event) => setDate(event.target.value)}
                    required
                  />
                </Field>
                <Field label="Departure time">
                  <input
                    type="time"
                    value={time}
                    onChange={(event) => setTime(event.target.value)}
                    required
                  />
                </Field>
              </div>
              <Field label="Departure window">
                <select
                  value={flexibility}
                  onChange={(event) => setFlexibility(event.target.value)}
                >
                  {[1, 15, 30, 60, 90, 120].map((value) => (
                    <option key={value} value={value}>
                      {value === 1
                        ? "Within one minute"
                        : `Up to ${value} minutes later`}
                    </option>
                  ))}
                </select>
              </Field>
              {(offer || (editing && status?.request?.driver)) && (
                <Field label="Available passenger seats">
                  <input
                    type="number"
                    min={1}
                    max={8}
                    step={1}
                    required
                    value={seats}
                    onChange={(event) => setSeats(Number(event.target.value))}
                  />
                </Field>
              )}
              <div className="request-preferences">
                <h3>Your comfort</h3>
                <Toggle
                  label="Women-only group"
                  description="Eligibility is checked against your profile by the service."
                  checked={women}
                  onChange={setWomen}
                />
                <Toggle
                  label="Open to a lift"
                  description="Include compatible registered lift offers."
                  checked={lift}
                  onChange={setLift}
                />
              </div>
              <Button
                className="wide"
                disabled={!from || !to || busy || !allowed}
              >
                {busy
                  ? "Saving your request…"
                  : editing
                    ? "Save journey"
                    : offer
                      ? "Register lift"
                      : "Find my pool"}{" "}
                <ArrowUpRight size={17} />
              </Button>
            </fieldset>
            {error && <p role="alert">{error}</p>}
            <small className="muted">
              No transport is booked. Confirm your pickup plan with your group.
            </small>
          </form>
        </section>
        <aside className="create-aside">
          <RouteArt
            from={from?.label ?? "Your starting point"}
            to={to?.label ?? "Your destination"}
          />
          <div className="panel">
            <ShieldCheck />
            <h3>Nearby starts. Nearby destinations.</h3>
            <p>
              Pickups must align within 300 metres and destinations within 100 m. Departure windows and
              everyone’s safety preferences must also fit.
            </p>
            <Link className="text-link" to="/app/rider">
              Offering a lift? Register here <ArrowRight size={16} />
            </Link>
          </div>
        </aside>
      </div>
    </>
  );
}
export function MyRides({ history = false }: { history?: boolean }) {
  const { rows, cursor, error, loading, load } = useJourneyHistory();
  const [filter, setFilter] = useState(history ? "past" : "active");
  const past = (ride: Journey) =>
    ["completed", "cancelled", "expired"].includes(ride.status);
  const filtered = rows?.filter(
    (ride) =>
      filter === "all" || (filter === "past" ? past(ride) : !past(ride)),
  );
  return (
    <>
      <PageHead
        title={history ? "Your journey history." : "Your journeys."}
        subtitle="Plans and people along the way"
      >
        <LinkButton to="/app/create">
          <Plus size={16} /> Plan a journey
        </LinkButton>
      </PageHead>
      <div className="tabs">
        {[
          ["active", "Upcoming"],
          ["past", "Past journeys"],
          ["all", "All journeys"],
        ].map(([value, label]) => (
          <button
            key={value}
            className={value === filter ? "active" : ""}
            aria-pressed={value === filter}
            onClick={() => setFilter(value)}
          >
            {label}
          </button>
        ))}
      </div>
      <LoadError error={error} retry={() => void load()} />
      {rows === null && !error ? (
        <PixelStatus state="loading" label="Opening journeys" />
      ) : filtered?.length ? (
        filtered.map((ride) => (
          <JourneyCard key={ride.request_id} ride={ride} />
        ))
      ) : rows ? (
        <Empty
          title="No journeys in this view."
          text={
            cursor
              ? "Load more to see older journeys."
              : "Your journeys will appear here."
          }
        />
      ) : null}
      {cursor && (
        <Button
          variant="secondary"
          disabled={loading}
          onClick={() => void load(cursor)}
        >
          {loading ? "Loading…" : "Load older journeys"}
        </Button>
      )}
    </>
  );
}
export function RideDetail() {
  const { id } = useParams();
  const owner = useSession((s) => s.account?.user_id);
  const { status } = useMatchingStatus();
  const navigate = useNavigate();
  const version = useRef(0);
  const sending = useRef(false);
  const [ride, setRide] = useState<Journey | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [cancel, setCancel] = useState(false);
  const load = useCallback(async () => {
    if (!id) return;
    const revision = ++version.current;
    try {
      const row = await apiRequest<Journey>(`/rides/${encodeURIComponent(id)}`);
      if (
        revision === version.current &&
        owner === useSession.getState().account?.user_id
      ) {
        setRide(row);
        setError("");
      }
    } catch (problem) {
      if (revision === version.current) setError(friendlyError(problem));
    }
  }, [id, owner]);
  useRealtimeRefresh(load);
  useEffect(() => {
    setRide(null);
    setError("");
    setCancel(false);
    void load();
    return () => {
      version.current++;
    };
  }, [load]);
  async function cancelRide() {
    if (!id || sending.current) return;
    sending.current = true;
    setBusy(true);
    try {
      await apiRequest(`/rides/${encodeURIComponent(id)}`, {
        method: "DELETE",
      });
      if (owner === useSession.getState().account?.user_id) {
        refreshJourneys();
        navigate("/app/rides");
      }
    } catch (problem) {
      if (owner === useSession.getState().account?.user_id) {
        setError(friendlyError(problem));
        setCancel(false);
        await load();
      }
    } finally {
      sending.current = false;
      if (owner === useSession.getState().account?.user_id) setBusy(false);
    }
  }
  const editable = status?.request?.request_id === id && canEditJourney(status);
  return (
    <>
      <Link className="back-link" to="/app/rides">
        ← Your journeys
      </Link>
      <PageHead title="Your journey" />
      <LoadError error={error} retry={() => void load()} />
      {!ride && !error ? (
        <PixelStatus state="loading" label="Opening journey" />
      ) : ride ? (
        <>
          <RouteArt from={ride.pickup_label} to={ride.drop_label} />
          <section className="panel">
            <Badge>{ride.status}</Badge>
            <h2>
              {ride.pickup_label} → {ride.drop_label}
            </h2>
            <p>{journeyTime(ride)}</p>
            <p>
              {ride.female_only
                ? "Women-only group"
                : "All eligible travellers"}{" "}
              · {ride.allow_riders ? "Open to a lift" : "No lifts"}
            </p>
            <div className="intro-actions">
              {ride.group_id && (
                <LinkButton
                  to={`/app/groups/${encodeURIComponent(ride.group_id)}`}
                >
                  Open group
                </LinkButton>
              )}
              {editable && (
                <LinkButton
                  to={`/app/create?edit=${encodeURIComponent(ride.request_id)}`}
                  secondary
                >
                  Edit journey
                </LinkButton>
              )}
              {["pending", "matching"].includes(ride.status) && (
                <Button
                  variant="secondary"
                  disabled={busy}
                  onClick={() => setCancel(true)}
                >
                  Cancel request
                </Button>
              )}
            </div>
            {ride.status === "matched" ? (
              <p>
                Manage confirmation, rejection and pickup details in your group.
              </p>
            ) : ["pending", "matching"].includes(ride.status) ? (
              <>
                <PixelStatus
                  state="searching"
                  label="Looking for your people"
                />
                <Link className="text-link" to="/app/explore">
                  View matching status →
                </Link>
              </>
            ) : null}
          </section>
        </>
      ) : null}
      <Modal
        open={cancel}
        onOpenChange={(open) => {
          if (!busy) setCancel(open);
        }}
        title="Cancel this request?"
        description="Matching will stop for this journey."
      >
        <Button disabled={busy} onClick={() => void cancelRide()}>
          {busy ? "Cancelling…" : "Cancel request"}
        </Button>
        <Button
          variant="secondary"
          disabled={busy}
          onClick={() => setCancel(false)}
        >
          Keep searching
        </Button>
      </Modal>
    </>
  );
}
