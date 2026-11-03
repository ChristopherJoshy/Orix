import { friendlyError } from "../lib/errors";
import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { apiRequest, apiRequestPage, useSession } from "../lib/session";
import { adminRequest, adminRequestPage } from "../lib/admin";
import { useResource } from "../lib/resource";
import { refreshJourneys } from "../lib/journeys";
import { Badge, Button, Empty, Field, PageHead } from "../components/ui";
import { PixelStatus } from "../components/PixelStatus";
import { Modal } from "../components/Modal";
interface Ticket {
  ticket_id: string;
  subject: string;
  description: string;
  type: string;
  status: string;
  created_at: string;
  updated_at: string;
  messages: {
    sender_id: string;
    sender_name: string;
    content: string;
    timestamp: string;
    is_admin: boolean;
  }[];
}
function adminTickets<T>(path: string): Promise<T> {
  const id = path.split("?")[0].split("/")[3];
  return adminRequest<T>(
    id ? "support-thread" : "support-tickets",
    "GET",
    undefined,
    id ? { ticket_id: id } : undefined,
  );
}
export function Support({ admin = false }: { admin?: boolean }) {
  const owner = useSession((s) => s.account?.user_id);
  const pageCount = useRef(1);
  const readPages = useCallback(
    async (
      _path: string,
    ): Promise<{ tickets: Ticket[]; nextCursor: string | null }> => {
      if (!owner) throw new Error("Account unavailable");
      let cursor: string | null = null;
      const rows = new Map<string, Ticket>();
      for (let page = 0; page < pageCount.current; page++) {
        if (owner !== useSession.getState().account?.user_id)
          throw new Error("Account changed");
        const result = admin
          ? await adminRequestPage<Ticket[]>(
              "support-tickets",
              "GET",
              undefined,
              cursor ? { before: cursor } : undefined,
            )
          : await apiRequestPage<Ticket[]>(
              `/support/tickets?limit=50${cursor ? `&before=${encodeURIComponent(cursor)}` : ""}`,
            );
        if (owner !== useSession.getState().account?.user_id)
          throw new Error("Account changed");
        for (const ticket of result.data) rows.set(ticket.ticket_id, ticket);
        cursor = result.nextCursor;
        if (!cursor) break;
      }
      return { tickets: [...rows.values()], nextCursor: cursor };
    },
    [owner, admin],
  );
  useEffect(() => {
    pageCount.current = 1;
  }, [owner, admin]);
  const { data, error, loading, load } = useResource<{
    tickets: Ticket[];
    nextCursor: string | null;
  }>("/support/tickets", 30000, readPages);
  const [open, setOpen] = useState(false);
  const [subject, setSubject] = useState("");
  const [description, setDescription] = useState("");
  const [type, setType] = useState("issue");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState("");
  const sending = useRef(false);
  const navigate = useNavigate();
  async function create(event: React.FormEvent) {
    event.preventDefault();
    if (sending.current) return;
    sending.current = true;
    setBusy(true);
    setFailure("");
    try {
      const ticket = await apiRequest<Ticket>("/support/tickets", {
        method: "POST",
        body: JSON.stringify({
          type,
          subject: subject.trim(),
          description: description.trim(),
          priority: "normal",
        }),
      });
      if (owner === useSession.getState().account?.user_id) {
        refreshJourneys();
        navigate(`/app/support/${encodeURIComponent(ticket.ticket_id)}`);
      }
    } catch (problem) {
      if (owner === useSession.getState().account?.user_id)
        setFailure(friendlyError(problem));
    } finally {
      sending.current = false;
      if (owner === useSession.getState().account?.user_id) setBusy(false);
    }
  }
  return (
    <>
      <PageHead
        title={admin ? "Support inbox." : "A little help."}
        subtitle={
          admin
            ? "Private support conversations"
            : "Ask a question or tell us what went wrong"
        }
      >
        {!admin && <Button onClick={() => setOpen(true)}>New ticket</Button>}
      </PageHead>
      {error && (
        <p role="alert">
          {error}{" "}
          <Button variant="secondary" onClick={() => void load()}>
            Retry
          </Button>
        </p>
      )}
      {!data && !error ? (
        <PixelStatus state="loading" label="Opening support" />
      ) : data?.tickets.length === 0 ? (
        <Empty
          title="No support conversations."
          text={
            admin
              ? "New tickets will appear here."
              : "Your questions and replies will stay together here."
          }
        />
      ) : (
        data?.tickets.map((ticket) => (
          <article key={ticket.ticket_id} className="panel">
            <Badge>{ticket.status}</Badge>
            <h2>{ticket.subject}</h2>
            <p>{new Date(ticket.updated_at).toLocaleString()}</p>
            <Link
              className="button secondary"
              to={`${admin ? "/admin" : "/app"}/support/${encodeURIComponent(ticket.ticket_id)}`}
            >
              Open conversation
            </Link>
          </article>
        ))
      )}
      {data?.nextCursor && (
        <Button
          variant="secondary"
          disabled={loading || busy}
          onClick={() => {
            pageCount.current++;
            void load();
          }}
        >
          Load older conversations
        </Button>
      )}
      <Modal
        open={open}
        onOpenChange={(value) => {
          if (!busy) setOpen(value);
        }}
        title="How can we help?"
        description="For a safety report about a travel partner, use the private report flow."
      >
        <form onSubmit={create}>
          <Field label="Topic">
            <select
              value={type}
              disabled={busy}
              onChange={(event) => setType(event.target.value)}
            >
              <option value="issue">Something isn’t working</option>
              <option value="feature">Suggest an improvement</option>
              <option value="other">Another question</option>
            </select>
          </Field>
          <Field label="Subject">
            <input
              required
              maxLength={150}
              value={subject}
              disabled={busy}
              onChange={(event) => setSubject(event.target.value)}
            />
          </Field>
          <Field label="What happened?">
            <textarea
              required
              maxLength={4000}
              rows={5}
              value={description}
              disabled={busy}
              onChange={(event) => setDescription(event.target.value)}
            />
          </Field>
          {failure && <p role="alert">{failure}</p>}
          {busy && (
            <PixelStatus
              state="uploading"
              compact
              label="Sending your ticket"
            />
          )}
          <Button disabled={busy || !subject.trim() || !description.trim()}>
            {busy ? "Sending…" : "Send to support"}
          </Button>
        </form>
        <p>
          <Link to="/app/safety">Report a safety concern →</Link>
        </p>
      </Modal>
    </>
  );
}
export function SupportThread({ admin = false }: { admin?: boolean }) {
  const { id } = useParams();
  const owner = useSession((s) => s.account?.user_id);
  const {
    data: ticket,
    error,
    load,
  } = useResource<Ticket>(
    id ? `/support/tickets/${encodeURIComponent(id)}` : null,
    15000,
    admin ? adminTickets : apiRequest,
  );
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState("");
  const [closing, setClosing] = useState(false);
  const sending = useRef(false);
  const retry = useRef<{
    owner: string | undefined;
    ticket: string;
    content: string;
    id: string;
  } | null>(null);
  async function act(action: string) {
    if (!id || sending.current) return;
    sending.current = true;
    setBusy(true);
    setFailure("");
    const content = draft.trim();
    if (
      action === "message" &&
      (!retry.current ||
        retry.current.owner !== owner ||
        retry.current.ticket !== id ||
        retry.current.content !== content)
    )
      retry.current = { owner, ticket: id, content, id: crypto.randomUUID() };
    const message =
      action === "message"
        ? { content, client_message_id: retry.current!.id }
        : {};
    try {
      if (admin)
        await adminRequest("support-action", "POST", message, {
          ticket_id: id,
          operation: action,
        });
      else
        await apiRequest(
          `/support/tickets/${encodeURIComponent(id)}/${action}`,
          {
            method: "POST",
            body: JSON.stringify(message),
          },
        );
      if (owner === useSession.getState().account?.user_id) {
        if (action === "message") {
          retry.current = null;
          setDraft((current) => (current.trim() === content ? "" : current));
        }
        setClosing(false);
        refreshJourneys();
        await load();
      }
    } catch (problem) {
      if (owner === useSession.getState().account?.user_id)
        setFailure(friendlyError(problem));
    } finally {
      sending.current = false;
      if (owner === useSession.getState().account?.user_id) setBusy(false);
    }
  }
  const closed = ticket && ["closed", "resolved"].includes(ticket.status);
  return (
    <>
      <Link
        className="back-link"
        to={admin ? "/admin/support" : "/app/support"}
      >
        ← Support conversations
      </Link>
      <PageHead
        title={ticket?.subject ?? "Your conversation"}
        subtitle={ticket?.ticket_id}
      />
      {(error || failure) && (
        <p role="alert">
          {failure || error}{" "}
          <Button variant="secondary" onClick={() => void load()}>
            Refresh
          </Button>
        </p>
      )}
      {!ticket && !error ? (
        <PixelStatus state="loading" label="Opening conversation" />
      ) : (
        ticket && (
          <>
            <section className="panel">
              <Badge>{ticket.status}</Badge>
              <h2>Your question</h2>
              <p style={{ whiteSpace: "pre-wrap" }}>{ticket.description}</p>
            </section>
            <section className="panel">
              <h2>Conversation</h2>
              <div
                className="group-message-log"
                role="log"
                aria-label="Support replies"
                tabIndex={0}
              >
                {ticket.messages.length === 0 ? (
                  <p>No replies yet. Support will respond here.</p>
                ) : (
                  ticket.messages.map((message, index) => (
                    <article
                      key={`${message.timestamp}:${index}`}
                      className={`report-bubble ${message.sender_id === owner ? "mine" : ""}`}
                    >
                      <small>
                        {message.is_admin
                          ? "ORIX Support"
                          : message.sender_id === owner
                            ? "You"
                            : message.sender_name}
                      </small>
                      <p style={{ whiteSpace: "pre-wrap" }}>
                        {message.content}
                      </p>
                      <time>
                        {new Date(message.timestamp).toLocaleString()}
                      </time>
                    </article>
                  ))
                )}
              </div>
              {!closed && ticket.messages.length < 200 ? (
                <form
                  onSubmit={(event) => {
                    event.preventDefault();
                    void act("message");
                  }}
                >
                  <Field label="Your reply">
                    <textarea
                      value={draft}
                      disabled={busy}
                      maxLength={4000}
                      rows={3}
                      onChange={(event) => setDraft(event.target.value)}
                      required
                    />
                  </Field>
                  <Button disabled={busy || !draft.trim()}>
                    {busy ? "Sending…" : "Send reply"}
                  </Button>
                </form>
              ) : (
                <p>
                  {closed
                    ? "This conversation is closed."
                    : "This conversation has reached its reply limit."}
                </p>
              )}
              <div className="intro-actions">
                {!closed && (
                  <Button
                    variant="secondary"
                    disabled={busy}
                    onClick={() => setClosing(true)}
                  >
                    Close conversation
                  </Button>
                )}
                {closed && (
                  <Button
                    variant="secondary"
                    disabled={busy}
                    onClick={() => void act("reopen")}
                  >
                    Reopen conversation
                  </Button>
                )}
                {admin && ticket.status === "open" && (
                  <Button
                    variant="secondary"
                    disabled={busy}
                    onClick={() => void act("in_progress")}
                  >
                    Mark in progress
                  </Button>
                )}
              </div>
            </section>
          </>
        )
      )}
      <Modal
        open={closing}
        onOpenChange={(value) => {
          if (!busy) setClosing(value);
        }}
        title="Close this conversation?"
        description="Your messages remain available. You can reopen the conversation when you need more help."
      >
        <Button disabled={busy} onClick={() => void act("close")}>
          Close conversation
        </Button>
        <Button
          variant="secondary"
          disabled={busy}
          onClick={() => setClosing(false)}
        >
          Keep open
        </Button>
      </Modal>
    </>
  );
}
