import {
  mergeMessages,
  isChatMessage,
  type ChatMessage as Message,
} from "../lib/chat";
import { refreshEvent } from "../lib/realtime";
import { friendlyError } from "../lib/errors";
import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { apiRequest, apiRequestPage, useSession } from "../lib/session";
import { useRealtimeRefresh } from "../lib/realtime";
import { Badge, Button, PageHead } from "../components/ui";
import { Modal } from "../components/Modal";
import { PersonIdentity } from "../components/EmailPersonPicker";
import { PixelStatus } from "../components/PixelStatus";
import { MessageSquare, ArrowUp } from "lucide-react";
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
  const readBusy = useRef(false);
  const readAgain = useRef(false);
  const olderBusy = useRef(false);
  const historyOpened = useRef(false);
  const [sending, setSending] = useState(false);
  const sendBusy = useRef(false);
  const acknowledged = useRef(new Set<string>());
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [draft, setDraft] = useState("");
  const [reject, setReject] = useState(false);
  const messageLog = useRef<HTMLDivElement>(null);
  const composer = useRef<HTMLTextAreaElement>(null);
  const followMessages = useRef(true);
  const lastMessage = messages.at(-1)?.message_id;
  useEffect(() => {
    const log = messageLog.current;
    if (log && followMessages.current) log.scrollTop = log.scrollHeight;
  }, [lastMessage]);
  useEffect(() => {
    const input = composer.current;
    if (!input) return;
    input.style.height = "0px";
    input.style.height = `${Math.min(144, Math.max(44, input.scrollHeight))}px`;
  }, [draft]);
  const load = useCallback(async () => {
    if (!id) return;
    if (readBusy.current) {
      readAgain.current = true;
      return;
    }
    readBusy.current = true;
    const request = version.current;
    const valid = () =>
      request === version.current &&
      owner === useSession.getState().account?.user_id;
    try {
      // A slow group read must not delay the conversation snapshot.
      await Promise.all([
        apiRequest<Group>(`/groups/${encodeURIComponent(id)}`)
          .then((value) => {
            if (valid()) {
              setGroup(value);
              setError("");
            }
          })
          .catch(() => {
            if (valid()) setError("The group could not be refreshed.");
          }),
        apiRequestPage<Message[]>(
          `/groups/${encodeURIComponent(id)}/chat?limit=50`,
        )
          .then((result) => {
            if (!valid()) return;
            setMessages((previous) => mergeMessages(previous, result.data));
            // Keep the earliest cursor once older history has been opened.
            if (!historyOpened.current) setChatCursor(result.nextCursor);
            setChatError("");
          })
          .catch(() => {
            if (valid())
              setChatError(
                "The conversation could not be refreshed. Try again.",
              );
          }),
      ]);
    } finally {
      if (valid()) {
        readBusy.current = false;
        if (!olderBusy.current) setChatLoading(false);
        if (readAgain.current) {
          readAgain.current = false;
          void load();
        }
      }
    }
  }, [id, owner]);
  const loadOlder = async () => {
    if (!id || !chatCursor || olderBusy.current) return;
    const request = version.current;
    olderBusy.current = true;
    historyOpened.current = true;
    setChatLoading(true);
    const log = messageLog.current;
    const height = log?.scrollHeight ?? 0;
    const top = log?.scrollTop ?? 0;
    try {
      const result = await apiRequestPage<Message[]>(
        `/groups/${encodeURIComponent(id)}/chat?limit=50&before=${encodeURIComponent(chatCursor)}`,
      );
      if (
        request !== version.current ||
        owner !== useSession.getState().account?.user_id
      )
        return;
      setMessages((previous) => mergeMessages(previous, result.data));
      setChatCursor(result.nextCursor);
      setChatError("");
      requestAnimationFrame(() => {
        if (request === version.current && log)
          log.scrollTop = top + log.scrollHeight - height;
      });
    } catch {
      if (request === version.current)
        setChatError("Older messages couldn't load. Try again.");
    } finally {
      if (request === version.current) {
        olderBusy.current = false;
        setChatLoading(false);
      }
    }
  };
  useEffect(() => {
    version.current++;
    readBusy.current = false;
    readAgain.current = false;
    olderBusy.current = false;
    historyOpened.current = false;
    sendBusy.current = false;
    acknowledged.current.clear();
    setSending(false);
    setGroup(null);
    setMessages([]);
    setChatCursor(null);
    setChatError("");
    setChatLoading(true);
    setDraft("");
    followMessages.current = true;
    void load();
    const onUpdate = (event: Event) => {
      const detail = (event as CustomEvent).detail;
      if (owner !== useSession.getState().account?.user_id) return;
      if (detail?.type === "new_chat_message") {
        if (detail.group_id !== id) return;
        if (isChatMessage(detail.message)) {
          if (detail.message.is_mine && detail.message.client_message_id)
            acknowledged.current.add(detail.message.client_message_id);
          setMessages((previous) => mergeMessages(previous, [detail.message]));
        } else void load();
        return;
      }
      if (!detail?.group_id || detail.group_id === id) void load();
    };
    const recover = () => {
      if (!document.hidden) void load();
    };
    window.addEventListener(refreshEvent, onUpdate);
    document.addEventListener("visibilitychange", recover);
    const timer = window.setInterval(recover, 30000);
    return () => {
      version.current++;
      window.clearInterval(timer);
      window.removeEventListener(refreshEvent, onUpdate);
      document.removeEventListener("visibilitychange", recover);
    };
  }, [load, id, owner]);
  async function sendMessage(
    text: string,
    clientId: string = crypto.randomUUID(),
  ) {
    if (!text || sendBusy.current || !id) return;
    sendBusy.current = true;
    setSending(true);
    setChatError("");
    const request = version.current;
    const localId = `local-${clientId}`;
    setMessages((previous) =>
      mergeMessages(
        previous.filter((m) => m.message_id !== localId),
        [
          {
            message_id: localId,
            client_message_id: clientId,
            content: text,
            created_at: new Date().toISOString(),
            is_mine: true,
            pending: true,
          },
        ],
      ),
    );
    followMessages.current = true;
    setDraft((previous) => (previous.trim() === text ? "" : previous));
    composer.current?.focus();
    try {
      const saved = await apiRequest<Message>(
        `/groups/${encodeURIComponent(id)}/chat`,
        {
          method: "POST",
          body: JSON.stringify({
            content: text,
            client_message_id: clientId,
          }),
        },
      );
      if (
        request !== version.current ||
        owner !== useSession.getState().account?.user_id
      )
        return;
      setMessages((previous) =>
        mergeMessages(
          previous.filter((m) => m.message_id !== localId),
          [saved],
        ),
      );
    } catch (problem) {
      if (
        request === version.current &&
        owner === useSession.getState().account?.user_id &&
        !acknowledged.current.has(clientId)
      ) {
        setChatError(friendlyError(problem));
        setMessages((previous) =>
          previous.map((m) =>
            m.message_id === localId
              ? { ...m, pending: false, failed: true }
              : m,
          ),
        );
      }
    } finally {
      if (
        request === version.current &&
        owner === useSession.getState().account?.user_id
      ) {
        sendBusy.current = false;
        setSending(false);
      }
    }
  }
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
      <PageHead
        title="Your journey group."
        subtitle="Your route, travel partners and conversation"
      >
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
          {waiting && group.members.length < 2 && (
            <p className="group-confirmation-note" role="status">
              Looking for your travel partners. You can confirm this group once
              someone joins.
            </p>
          )}
          {waiting && group.members.length > 1 && me?.readiness_confirmed && (
            <p className="group-confirmation-note" role="status">
              You’ve confirmed this group.{" "}
              {group.all_confirmed
                ? "Everyone is ready."
                : "Waiting for your travel partners to confirm."}
            </p>
          )}
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
      <section
        className="group-conversation"
        aria-labelledby="group-chat-title"
      >
        <header className="group-chat-header">
          <div>
            <h2 id="group-chat-title">Group conversation</h2>
            <p>Pickup plans. Quick hellos. All in one place.</p>
          </div>
          <span className="group-chat-member-count">
            {group.members.length}{" "}
            {group.members.length === 1 ? "traveller" : "travellers"}
          </span>
        </header>
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
            onClick={() => void loadOlder()}
          >
            {chatLoading ? "Loading…" : "Load older messages"}
          </Button>
        )}
        <div
          ref={messageLog}
          className="group-message-log"
          role="log"
          aria-label="Group messages"
          tabIndex={0}
          onScroll={(event) => {
            const log = event.currentTarget;
            followMessages.current =
              log.scrollHeight - log.scrollTop - log.clientHeight < 80;
          }}
        >
          {messages.length === 0 && chatLoading ? (
            <PixelStatus state="loading" label="Loading conversation…" />
          ) : messages.length === 0 ? (
            <div className="group-chat-empty">
              <MessageSquare size={28} strokeWidth={1.5} aria-hidden="true" />
              <p>A good journey starts with hello.</p>
              <span>
                {group.members.length < 2
                  ? "Your conversation will be here when a travel partner joins."
                  : "Say hello or share a detail about your pickup."}
              </span>
            </div>
          ) : (
            messages.map((message) => (
              <article
                key={message.message_id}
                className={`group-chat-bubble ${message.is_mine ? "mine" : ""}`}
              >
                <small>
                  {message.is_mine
                    ? "You"
                    : message.user_display_name || "Travel partner"}
                </small>
                <p>{message.content}</p>
                {message.pending && <small role="status">Sending…</small>}
                {message.failed && (
                  <>
                    <small>Not confirmed.</small>
                    <Button
                      variant="secondary"
                      disabled={sending || !canChat}
                      onClick={() => {
                        if (message.client_message_id)
                          void sendMessage(
                            message.content,
                            message.client_message_id,
                          );
                      }}
                    >
                      Retry message
                    </Button>
                  </>
                )}
                <time dateTime={message.created_at}>
                  {new Date(message.created_at).toLocaleTimeString([], {
                    hour: "2-digit",
                    minute: "2-digit",
                  })}
                </time>
              </article>
            ))
          )}
        </div>
        {canChat && (
          <form
            className="group-chat-composer"
            onSubmit={(event) => {
              event.preventDefault();
              void sendMessage(draft.trim());
            }}
          >
            <label className="group-chat-input">
              <span className="group-chat-input-label">Message your group</span>
              <textarea
                ref={composer}
                maxLength={1000}
                rows={1}
                placeholder="Write a message…"
                aria-describedby="group-chat-keyboard"
                required
                value={draft}
                onChange={(event) => setDraft(event.target.value)}
                onKeyDown={(event) => {
                  if (
                    event.key === "Enter" &&
                    !event.shiftKey &&
                    !event.nativeEvent.isComposing
                  ) {
                    event.preventDefault();
                    if (!sending && draft.trim())
                      event.currentTarget.form?.requestSubmit();
                  }
                }}
              />
            </label>
            <Button
              className="group-chat-send"
              aria-label={sending ? "Sending message" : "Send message"}
              disabled={sending || !draft.trim()}
            >
              <ArrowUp size={20} aria-hidden="true" />
            </Button>
            <small className="group-chat-keyboard" id="group-chat-keyboard">
              Enter to send · Shift + Enter for a new line
            </small>
          </form>
        )}
        {!canChat && (
          <p className="group-chat-closed">
            This journey has ended. You can still read the conversation.
          </p>
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
