import { friendlyError } from "../lib/errors";
import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { apiRequest, apiRequestPage, useSession } from "../lib/session";
import { useRealtimeRefresh } from "../lib/realtime";
import { Badge, Button, Field, PageHead } from "../components/ui";
import { Modal } from "../components/Modal";
import { PersonIdentity } from "../components/EmailPersonPicker";
import { PixelStatus } from "../components/PixelStatus";
interface Member {
  user_id: string;
  display_name: string;
  email?: string;
  photo_url?: string;
  occupation?: string;
  role: string;
  is_me: boolean;
  readiness_confirmed: boolean;
  is_here: boolean;
  finished: boolean;
}
interface Group {
  group_id: string;
  route_summary: string;
  pickup_summary: string;
  drop_summary: string;
  status: string;
  departure_start: string;
  departure_end: string;
  mode: string;
  members: Member[];
  all_confirmed: boolean;
}
interface Message {
  message_id: string;
  content: string;
  is_mine: boolean;
  user_display_name?: string;
  created_at: string;
}
const occupation = (value?: string) =>
  value === "student"
    ? "Student"
    : value === "teacher"
      ? "Teacher"
      : value === "other"
        ? "Other"
        : "Not provided";
export function Groups() {
  const owner = useSession((s) => s.account?.user_id);
  const version = useRef(0);
  const [groups, setGroups] = useState<Group[] | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const load = useCallback(async () => {
    const request = ++version.current;
    setLoading(true);
    try {
      const rows = await apiRequest<Group[]>("/groups");
      if (
        request === version.current &&
        owner === useSession.getState().account?.user_id
      ) {
        setGroups(rows);
        setError("");
      }
    } catch (problem) {
      if (request === version.current) setError(friendlyError(problem));
    } finally {
      if (request === version.current) setLoading(false);
    }
  }, [owner]);
  useRealtimeRefresh(load);
  useEffect(() => {
    setGroups(null);
    void load();
    return () => {
      version.current++;
    };
  }, [load]);
  return (
    <>
      <PageHead title="Your groups" subtitle="Pickup plans and travel partners">
        <Button
          variant="secondary"
          disabled={loading}
          onClick={() => void load()}
        >
          Refresh
        </Button>
      </PageHead>
      {error && <p role="alert">{error}</p>}
      {groups === null && loading ? (
        <PixelStatus state="loading" label="Refreshing groups" />
      ) : groups?.length === 0 ? (
        <section className="panel">
          <p>Your group will appear here when you match.</p>
          <Link className="button primary" to="/app/create">
            Plan a journey
          </Link>
        </section>
      ) : (
        groups?.map((group) => (
          <section key={group.group_id} className="panel">
            <Badge>
              {group.members.length < 2 && group.status === "pending"
                ? "Searching"
                : group.status}
            </Badge>
            <h2>{group.route_summary}</h2>
            <p>
              {group.members.length}{" "}
              {group.members.length === 1 ? "person" : "people"} ·{" "}
              {new Date(group.departure_start).toLocaleString()}
            </p>
            <Link
              className="button secondary"
              to={`/app/groups/${encodeURIComponent(group.group_id)}`}
            >
              Open group
            </Link>
          </section>
        ))
      )}
    </>
  );
}
export function GroupDetail() {
  const { id } = useParams();
  const owner = useSession((s) => s.account?.user_id);
  const navigate = useNavigate();
  const version = useRef(0);
  const [group, setGroup] = useState<Group | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [chatCursor, setChatCursor] = useState<string | null>(null);
  const [chatError, setChatError] = useState("");
  const [chatLoading, setChatLoading] = useState(false);
  const pages = useRef(1);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [draft, setDraft] = useState("");
  const [reject, setReject] = useState(false);
  const retry = useRef<{ text: string; id: string } | null>(null);
  const load = useCallback(async () => {
    if (!id) return;
    const request = ++version.current;
    const results = await Promise.allSettled([
      apiRequest<Group>(`/groups/${encodeURIComponent(id)}`),
      (async () => {
        let cursor: string | null = null;
        let combined: Message[] = [];
        for (let page = 0; page < pages.current; page++) {
          if (
            owner !== useSession.getState().account?.user_id ||
            request !== version.current
          )
            throw new Error("Account changed");
          const result: { data: Message[]; nextCursor: string | null } =
            await apiRequestPage<Message[]>(
              `/groups/${encodeURIComponent(id)}/chat?limit=50${cursor ? `&before=${encodeURIComponent(cursor)}` : ""}`,
            );
          combined = [...result.data, ...combined];
          cursor = result.nextCursor;
          if (!cursor) break;
        }
        return {
          messages: [
            ...new Map(
              combined.map((message) => [message.message_id, message]),
            ).values(),
          ],
          cursor,
        };
      })(),
    ]);
    if (
      request !== version.current ||
      owner !== useSession.getState().account?.user_id
    )
      return;
    if (results[0].status === "fulfilled") {
      setGroup(results[0].value);
      setError("");
    } else setError("The group could not be refreshed.");
    if (results[1].status === "fulfilled") {
      setMessages(results[1].value.messages);
      setChatCursor(results[1].value.cursor);
      setChatError("");
    } else setChatError("The conversation could not be refreshed. Try again.");
    setChatLoading(false);
  }, [id, owner]);
  useRealtimeRefresh(load);
  useEffect(() => {
    setGroup(null);
    setMessages([]);
    setChatCursor(null);
    setChatError("");
    setChatLoading(false);
    pages.current = 1;
    setDraft("");
    retry.current = null;
    void load();
    return () => {
      version.current++;
    };
  }, [load]);
  async function action(name: string) {
    if (!id || busy) return;
    setBusy(true);
    setError("");
    try {
      await apiRequest(`/groups/${encodeURIComponent(id)}/${name}`, {
        method: "POST",
        ...(name === "confirm"
          ? { body: JSON.stringify({ confirmed: true }) }
          : {}),
      });
      if (owner !== useSession.getState().account?.user_id) return;
      if (name === "reject") {
        navigate("/app/explore");
        return;
      }
      await load();
    } catch (problem) {
      if (owner === useSession.getState().account?.user_id)
        setError(friendlyError(problem));
    } finally {
      if (owner === useSession.getState().account?.user_id) setBusy(false);
    }
  }
  if (!group)
    return (
      <>
        <PageHead title="Your group" />
        {error ? (
          <p role="alert">{error}</p>
        ) : (
          <PixelStatus state="loading" label="Opening group" />
        )}
        <Button onClick={() => void load()}>Refresh</Button>
      </>
    );
  const me = group.members.find((member) => member.is_me);
  const waiting = ["pending", "confirming", "active"].includes(group.status);
  const canReject =
    waiting &&
    group.members.length > 1 &&
    new Date(group.departure_end).getTime() > Date.now();
  const canChat = ["pending", "confirming", "active", "in_progress"].includes(
    group.status,
  );
  const arrived =
    group.mode === "lift"
      ? me?.role === "rider" && me.is_here
      : group.members.every((member) => member.is_here);
  return (
    <>
      <Link className="back-link" to="/app/groups">
        Your groups
      </Link>
      <PageHead title={group.route_summary} subtitle={group.pickup_summary}>
        <Badge>{group.status}</Badge>
      </PageHead>
      {error && <p role="alert">{error}</p>}
      <div className="two-column">
        <section className="panel">
          <PixelStatus
            state={
              group.status === "in_progress"
                ? "travelling"
                : group.status === "completed"
                  ? "completed"
                  : group.status === "cancelled"
                    ? "cancelled"
                    : group.members.length < 2
                      ? "searching"
                      : "found"
            }
            label={group.status}
          />
          <h2>Pickup plan</h2>
          <p>{new Date(group.departure_start).toLocaleString()}</p>
          <p>
            {group.pickup_summary} → {group.drop_summary}
          </p>
          <div className="intro-actions">
            {["pending", "confirming"].includes(group.status) &&
              group.members.length > 1 &&
              !me?.readiness_confirmed && (
                <Button disabled={busy} onClick={() => void action("confirm")}>
                  Confirm this group
                </Button>
              )}
            {group.status === "active" && !me?.is_here && (
              <Button disabled={busy} onClick={() => void action("mark-here")}>
                I am at my pickup point
              </Button>
            )}
            {group.status === "active" &&
              arrived &&
              new Date(group.departure_start).getTime() <= Date.now() && (
                <Button disabled={busy} onClick={() => void action("start")}>
                  Start journey
                </Button>
              )}
            {group.status === "in_progress" && !me?.finished && (
              <Button disabled={busy} onClick={() => void action("finish")}>
                I reached my destination
              </Button>
            )}
            {canReject && (
              <Button
                variant="secondary"
                disabled={busy}
                onClick={() => setReject(true)}
              >
                Reject and search again
              </Button>
            )}
          </div>
        </section>
        <section className="panel">
          <h2>Your travel partners</h2>
          {group.members.map((member) => (
            <div className="group-contact" key={member.user_id}>
              <PersonIdentity
                person={{
                  userId: member.user_id,
                  name: member.display_name,
                  email: member.email ?? "",
                  photoUrl: member.photo_url,
                }}
              />
              <p>
                {occupation(member.occupation)} ·{" "}
                {member.role === "rider" ? "Lift driver" : "Passenger"}
                {member.is_me ? " · You" : ""}
              </p>
              <small>
                {member.readiness_confirmed ? "Confirmed" : "Not confirmed"}
              </small>
            </div>
          ))}
        </section>
      </div>
      <section className="panel">
        <h2>Group conversation</h2>
        {chatError && (
          <p role="alert">
            {chatError}{" "}
            <Button variant="secondary" onClick={() => void load()}>
              Retry
            </Button>
          </p>
        )}
        {chatCursor && (
          <Button
            variant="secondary"
            disabled={chatLoading}
            onClick={() => {
              pages.current++;
              setChatLoading(true);
              void load();
            }}
          >
            {chatLoading ? "Loading…" : "Load older messages"}
          </Button>
        )}
        <div
          className="group-message-log"
          role="log"
          aria-label="Group messages"
          tabIndex={0}
        >
          {messages.length === 0 ? (
            <p>No messages yet.</p>
          ) : (
            messages.map((message) => (
              <article
                key={message.message_id}
                className={`report-bubble ${message.is_mine ? "mine" : ""}`}
              >
                <small>
                  {message.is_mine
                    ? "You"
                    : message.user_display_name || "Travel partner"}
                </small>
                <p>{message.content}</p>
                <time>{new Date(message.created_at).toLocaleTimeString()}</time>
              </article>
            ))
          )}
        </div>
        {canChat && (
          <form
            onSubmit={async (event) => {
              event.preventDefault();
              const text = draft.trim();
              if (!text || busy || !id) return;
              setBusy(true);
              setError("");
              if (retry.current?.text !== text)
                retry.current = { text, id: crypto.randomUUID() };
              try {
                await apiRequest(`/groups/${encodeURIComponent(id)}/chat`, {
                  method: "POST",
                  body: JSON.stringify({
                    content: text,
                    client_message_id: retry.current.id,
                  }),
                });
                if (owner !== useSession.getState().account?.user_id) return;
                setDraft("");
                retry.current = null;
                await load();
              } catch (problem) {
                if (owner === useSession.getState().account?.user_id)
                  setError(friendlyError(problem));
              } finally {
                if (owner === useSession.getState().account?.user_id)
                  setBusy(false);
              }
            }}
          >
            <Field label="Message your group">
              <textarea
                maxLength={1000}
                rows={2}
                required
                disabled={busy}
                value={draft}
                onChange={(event) => setDraft(event.target.value)}
              />
            </Field>
            <Button disabled={busy || !draft.trim()}>Send message</Button>
          </form>
        )}
      </section>
      <Modal
        title="Find another group?"
        open={reject}
        onOpenChange={(value) => {
          if (!busy) setReject(value);
        }}
      >
        <p>
          Matching will restart for this journey. We will avoid this group and
          its members for the rest of this search. Everyone in the group will be
          notified and matched again.
        </p>
        <Button disabled={busy} onClick={() => void action("reject")}>
          {busy ? "Restarting…" : "Reject and search again"}
        </Button>
      </Modal>
    </>
  );
}
