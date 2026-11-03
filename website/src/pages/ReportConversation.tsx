import { friendlyError } from "../lib/errors";
import { refreshEvent } from "../lib/realtime";
import { useEffect, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { ArrowLeft, ArrowUp } from "lucide-react";
import { Button, Badge, Empty, Field, PageHead } from "../components/ui";
import { PixelStatus } from "../components/PixelStatus";
import { Modal } from "../components/Modal";
import { reportThread, replyToReport, type ReportThread } from "../lib/safety";
import { adminRequest } from "../lib/admin";
import { useSession } from "../lib/session";
export default function ReportConversation({
  admin = false,
}: {
  admin?: boolean;
}) {
  const { id } = useParams();
  const identity = useSession((s) => s.account?.user_id);
  const [thread, setThread] = useState<{
    id: string;
    owner: string;
    data: ReportThread;
  } | null>(null);
  const [draft, setDraft] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [refresh, setRefresh] = useState(0);
  const [newReplies, setNewReplies] = useState(false);
  const [review, setReview] = useState(false);
  const [action, setAction] = useState("dismiss");
  const [reviewNote, setReviewNote] = useState("");
  const [hours, setHours] = useState(24);
  const log = useRef<HTMLDivElement>(null);
  const follow = useRef(true);
  const busy = useRef(false);
  const revision = useRef(0);
  const retry = useRef<{ text: string; id: string } | null>(null);
  const data =
    thread && thread.id === id && thread.owner === identity
      ? thread.data
      : null;
  useEffect(() => {
    const scope = ++revision.current;
    let alive = true;
    let fetching = false;
    const controller = new AbortController();
    setThread(null);
    setDraft("");
    setError("");
    setLoading(true);
    setSending(false);
    setReview(false);
    setReviewNote("");
    setNewReplies(false);
    retry.current = null;
    busy.current = false;
    follow.current = true;
    async function load() {
      if (!id || !identity || fetching || busy.current || document.hidden)
        return;
      fetching = true;
      try {
        const value = await reportThread(id, admin, controller.signal);
        if (
          !alive ||
          revision.current !== scope ||
          useSession.getState().account?.user_id !== identity
        )
          return;
        if (!value.report || !Array.isArray(value.messages))
          throw new Error("Conversation response is unavailable.");
        setThread({ id, owner: identity, data: value });
        setError("");
      } catch (problem) {
        if (alive && !controller.signal.aborted && revision.current === scope)
          setError(friendlyError(problem));
      } finally {
        fetching = false;
        if (alive && revision.current === scope) setLoading(false);
      }
    }
    void load();
    const timer = window.setInterval(() => void load(), 15000);
    const visible = () => {
      if (!document.hidden) void load();
    };
    document.addEventListener("visibilitychange", visible);
    window.addEventListener(refreshEvent, visible);
    return () => {
      alive = false;
      controller.abort();
      window.clearInterval(timer);
      document.removeEventListener("visibilitychange", visible);
      window.removeEventListener(refreshEvent, visible);
      revision.current++;
    };
  }, [id, identity, admin, refresh]);
  useEffect(() => {
    if (!log.current) return;
    if (follow.current) log.current.scrollTop = log.current.scrollHeight;
    else setNewReplies(true);
  }, [id, data?.messages.length]);
  async function send() {
    const text = draft.trim();
    if (
      !id ||
      !identity ||
      !data ||
      data.report.status !== "pending" ||
      !text ||
      busy.current
    )
      return;
    const scope = revision.current;
    busy.current = true;
    setSending(true);
    setError("");
    if (retry.current?.text !== text)
      retry.current = { text, id: crypto.randomUUID() };
    try {
      const value = await replyToReport(id, text, retry.current.id, admin);
      if (
        scope !== revision.current ||
        useSession.getState().account?.user_id !== identity
      )
        return;
      follow.current = true;
      setNewReplies(false);
      setThread({ id, owner: identity, data: value });
      setDraft("");
      retry.current = null;
    } catch (problem) {
      if (scope === revision.current) setError(friendlyError(problem));
    } finally {
      if (scope === revision.current) {
        busy.current = false;
        setSending(false);
      }
    }
  }
  if (!data && loading)
    return (
      <div className="loading-screen" role="status">
        <PixelStatus state="loading" label="Opening report conversation" />
      </div>
    );
  if (!data)
    return (
      <>
        <Empty
          title="Report unavailable"
          text={error || "Return to your report conversations."}
        />
        <Button onClick={() => setRefresh((v) => v + 1)}>Retry</Button>
        <Link to={admin ? "/admin/reports" : "/app/safety"}>
          Back to reports
        </Link>
      </>
    );
  const report = data.report;
  return (
    <>
      <Link className="back-link" to={admin ? "/admin/reports" : "/app/safety"}>
        <ArrowLeft size={16} />
        Reports
      </Link>
      <PageHead
        title={report.category.replaceAll("_", " ")}
        subtitle="Private report conversation"
      >
        <Badge>{report.status}</Badge>
      </PageHead>
      {error && (
        <p role="alert">
          {error}{" "}
          <Button
            variant="ghost"
            disabled={sending}
            onClick={() => setRefresh((v) => v + 1)}
          >
            Refresh
          </Button>
        </p>
      )}
      <div className="report-thread-layout">
        <section
          className="report-conversation"
          aria-label="Private report conversation"
        >
          <header>
            <strong>Orix moderation</strong>
            <p>Follow-ups and replies stay in this conversation.</p>
          </header>
          <div
            className="report-messages"
            ref={log}
            role="log"
            tabIndex={0}
            aria-label="Report messages"
            onScroll={() => {
              if (!log.current) return;
              follow.current =
                log.current.scrollHeight -
                  log.current.scrollTop -
                  log.current.clientHeight <
                80;
              if (follow.current) setNewReplies(false);
            }}
          >
            {data.messages.map((message) => {
              const mine = admin ? message.is_admin : !message.is_admin;
              return (
                <article
                  key={`${message.is_admin}:${message.message_id}`}
                  className={`report-bubble ${mine ? "mine" : ""}`}
                >
                  <small>
                    {message.is_admin ? "Orix moderation" : "Reporter"}
                  </small>
                  <p>{message.content}</p>
                  <time dateTime={message.created_at}>
                    {new Date(message.created_at).toLocaleString()}
                  </time>
                </article>
              );
            })}
          </div>
          {newReplies && (
            <div className="report-new-replies" role="status">
              <Button
                variant="ghost"
                onClick={() => {
                  follow.current = true;
                  setNewReplies(false);
                  if (log.current)
                    log.current.scrollTop = log.current.scrollHeight;
                }}
              >
                View new replies
              </Button>
            </div>
          )}
          {report.status === "pending" && data.messages.length < 200 ? (
            <form
              className="report-composer"
              onSubmit={(event) => {
                event.preventDefault();
                void send();
              }}
            >
              <Field
                label={admin ? "Reply to reporter" : "Reply to moderation"}
              >
                <textarea
                  value={draft}
                  onChange={(event) => setDraft(event.target.value)}
                  maxLength={2000}
                  rows={3}
                  required
                  disabled={sending}
                />
              </Field>
              <Button disabled={!draft.trim() || sending}>
                {sending ? "Sending…" : "Send reply"}
                <ArrowUp size={16} />
              </Button>
            </form>
          ) : (
            <p className="report-closed">
              {report.status === "pending"
                ? "This conversation has reached its message limit."
                : "This conversation is closed."}
              {report.resolution ? ` Decision: ${report.resolution}.` : ""}
            </p>
          )}
        </section>
        <aside className="panel report-context">
          <PixelStatus
            state="report-sent"
            label="Private report conversation"
          />
          <h2>Report details</h2>
          <p>Submitted {new Date(report.created_at).toLocaleDateString()}.</p>
          <p>
            Only the reporter and authorized moderation team can read or reply.
            The reported contact is excluded.
          </p>
          {admin && report.status === "pending" && (
            <Button
              variant="secondary"
              disabled={sending}
              onClick={() => setReview(true)}
            >
              Review report
            </Button>
          )}
        </aside>
      </div>
      {admin && (
        <Modal
          title="Review report"
          open={review}
          onOpenChange={(value) => {
            if (!sending) setReview(value);
          }}
        >
          <p>
            This decision closes the conversation. Account restrictions affect
            the reported contact.
          </p>
          <form
            onSubmit={async (event) => {
              event.preventDefault();
              if (!id || !identity || busy.current || !reviewNote.trim())
                return;
              const scope = revision.current;
              busy.current = true;
              setSending(true);
              setError("");
              try {
                await adminRequest(
                  "resolve-report",
                  "POST",
                  {
                    resolution: action,
                    note: reviewNote.trim(),
                    suspend_hours: action === "suspend" ? hours : 0,
                  },
                  { report_id: id },
                );
                if (
                  scope !== revision.current ||
                  useSession.getState().account?.user_id !== identity
                )
                  return;
                const value = await reportThread(id, true);
                if (scope !== revision.current) return;
                setThread({ id, owner: identity, data: value });
                setReview(false);
              } catch (problem) {
                if (scope === revision.current)
                  setError(friendlyError(problem));
              } finally {
                if (scope === revision.current) {
                  busy.current = false;
                  setSending(false);
                }
              }
            }}
          >
            <Field label="Decision">
              <select
                value={action}
                onChange={(event) => setAction(event.target.value)}
                disabled={sending}
              >
                <option value="dismiss">Dismiss report</option>
                <option value="warn">Warn contact</option>
                <option value="suspend">Suspend contact</option>
                <option value="ban">Ban contact</option>
              </select>
            </Field>
            {action === "suspend" && (
              <Field label="Suspension hours">
                <input
                  type="number"
                  min={1}
                  max={720}
                  required
                  value={hours}
                  disabled={sending}
                  onChange={(event) => setHours(Number(event.target.value))}
                />
              </Field>
            )}
            <Field label="Review note">
              <textarea
                required
                maxLength={1000}
                rows={4}
                value={reviewNote}
                disabled={sending}
                onChange={(event) => setReviewNote(event.target.value)}
              />
            </Field>
            {error && <p role="alert">{error}</p>}
            <Button disabled={sending || !reviewNote.trim()}>
              {sending ? "Saving…" : "Confirm decision"}
            </Button>
          </form>
        </Modal>
      )}
    </>
  );
}
