import { friendlyError } from "../lib/errors";
import { useRealtimeRefresh } from "../lib/realtime";
import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Flag, RefreshCw } from "lucide-react";
import { Modal } from "../components/Modal";
import { Button, Badge, Field, PageHead } from "../components/ui";
import { PixelStatus } from "../components/PixelStatus";
import {
  EmailPersonPicker,
  PersonIdentity,
} from "../components/EmailPersonPicker";
import {
  blockedContacts,
  createReport,
  ownReports,
  setContactBlocked,
  type SafetyRecord,
  type TravelContact,
} from "../lib/safety";
import { useSession } from "../lib/session";
const reasons = [
  ["harassment", "Harassment"],
  ["unsafe_driving", "Unsafe driving"],
  ["no_show", "Missed pickup"],
  ["spam", "Spam"],
  ["other", "Other"],
];
export default function Safety() {
  const identity = useSession((s) => s.account?.user_id);
  const navigate = useNavigate();
  const revision = useRef(0);
  const [blocked, setBlocked] = useState<TravelContact[] | null>(null);
  const [reports, setReports] = useState<SafetyRecord[] | null>(null);
  const [cursor, setCursor] = useState<string | null>(null);
  const [selected, setSelected] = useState<TravelContact | null>(null);
  const [reported, setReported] = useState<TravelContact | null>(null);
  const [reportOpen, setReportOpen] = useState(false);
  const [blockAction, setBlockAction] = useState<{
    person: TravelContact;
    blocked: boolean;
  } | null>(null);
  const [reason, setReason] = useState("harassment");
  const [note, setNote] = useState("");
  const [error, setError] = useState("");
  const [formError, setFormError] = useState("");
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const load = useCallback(
    async (signal?: AbortSignal) => {
      if (!identity) return;
      const version = ++revision.current;
      setLoading(true);
      setError("");
      const results = await Promise.allSettled([
        blockedContacts(signal),
        ownReports(signal),
      ]);
      if (
        signal?.aborted ||
        version !== revision.current ||
        useSession.getState().account?.user_id !== identity
      )
        return;
      const failures: string[] = [];
      if (results[0].status === "fulfilled") setBlocked(results[0].value);
      else failures.push("Blocked contacts could not be refreshed.");
      if (results[1].status === "fulfilled") {
        setReports(results[1].value.reports);
        setCursor(results[1].value.nextCursor);
      } else failures.push("Report conversations could not be refreshed.");
      setError(failures.join(" "));
      setLoading(false);
    },
    [identity],
  );
  useRealtimeRefresh(load);
  useEffect(() => {
    setBlocked(null);
    setReports(null);
    setSelected(null);
    setReported(null);
    setNote("");
    const controller = new AbortController();
    void load(controller.signal);
    return () => {
      controller.abort();
      revision.current++;
    };
  }, [load]);
  return (
    <>
      <PageHead
        title="Safety"
        subtitle="Contacts and private report conversations"
      >
        <Button
          variant="ghost"
          disabled={loading || busy}
          onClick={() => void load()}
          aria-label="Refresh safety information"
        >
          <RefreshCw size={18} />
        </Button>
        <Button
          onClick={() => {
            setFormError("");
            setReportOpen(true);
          }}
        >
          <Flag size={16} />
          Start a report
        </Button>
      </PageHead>
      {error && (
        <p role="alert">
          {error}{" "}
          <Button
            variant="ghost"
            disabled={loading}
            onClick={() => void load()}
          >
            Retry
          </Button>
        </p>
      )}
      <div className="two-column">
        <section className="panel">
          <h2>Block a contact</h2>
          <p>
            Search by email. Blocking excludes this contact from future matches
            and may cancel a group you currently share.
          </p>
          <EmailPersonPicker
            selected={selected}
            onSelect={setSelected}
            disabled={busy}
          />
          <Button
            variant="secondary"
            disabled={!selected || busy}
            onClick={() =>
              selected && setBlockAction({ person: selected, blocked: true })
            }
          >
            Block selected contact
          </Button>
        </section>
        <section className="panel">
          <h2>Your blocked contacts</h2>
          {loading && blocked === null ? (
            <PixelStatus state="loading" label="Refreshing blocked contacts" />
          ) : blocked?.length === 0 ? (
            <p>No blocked contacts.</p>
          ) : (
            blocked?.map((person) => (
              <div className="member-row" key={person.userId}>
                <PersonIdentity person={person} />
                <Button
                  variant="ghost"
                  disabled={busy}
                  onClick={() => setBlockAction({ person, blocked: false })}
                >
                  Unblock
                </Button>
              </div>
            ))
          )}
        </section>
      </div>
      <section className="panel">
        <div className="section-title">
          <h2>Report conversations</h2>
          {reports && <Badge>{reports.length}</Badge>}
        </div>
        {loading && reports === null ? (
          <PixelStatus state="loading" label="Refreshing reports" />
        ) : reports?.length === 0 ? (
          <p>No report conversations yet.</p>
        ) : (
          <div className="report-history">
            {reports?.map((report) => (
              <Link
                className="report-history-link"
                key={report.report_id}
                to={`/app/reports/${encodeURIComponent(report.report_id)}`}
              >
                <Flag size={18} />
                <div>
                  <strong>{report.category.replaceAll("_", " ")}</strong>
                  <p>{new Date(report.created_at).toLocaleDateString()}</p>
                  <small>{report.note}</small>
                </div>
                <Badge>{report.status}</Badge>
              </Link>
            ))}
          </div>
        )}
      </section>
      {cursor && (
        <Button
          variant="ghost"
          disabled={loading || busy}
          onClick={async () => {
            const version = ++revision.current;
            setLoading(true);
            try {
              const page = await ownReports(undefined, cursor);
              if (
                version !== revision.current ||
                identity !== useSession.getState().account?.user_id
              )
                return;
              setReports((previous) => [
                ...new Map(
                  [...(previous ?? []), ...page.reports].map((report) => [
                    report.report_id,
                    report,
                  ]),
                ).values(),
              ]);
              setCursor(page.nextCursor);
            } catch (problem) {
              if (version === revision.current)
                setError(friendlyError(problem));
            } finally {
              if (version === revision.current) setLoading(false);
            }
          }}
        >
          Earlier conversations
        </Button>
      )}
      <div className="inline-notice">
        Choose your travel and safety preferences for each journey. For
        immediate danger, contact local emergency services.{" "}
        <Link to="/app/support">Contact support</Link> for other help.
      </div>
      <Modal
        title="Start a report"
        description="Your first message opens a private conversation with Orix moderation. The reported contact cannot read it."
        open={reportOpen}
        onOpenChange={(value) => {
          if (!busy) setReportOpen(value);
        }}
      >
        <form
          onSubmit={async (event) => {
            event.preventDefault();
            if (!reported || busy || !note.trim()) return;
            const owner = identity;
            setBusy(true);
            setFormError("");
            try {
              const result = await createReport(reported, reason, note.trim());
              if (useSession.getState().account?.user_id !== owner) return;
              setReportOpen(false);
              navigate(`/app/reports/${encodeURIComponent(result.report_id)}`);
            } catch (problem) {
              if (useSession.getState().account?.user_id === owner)
                setFormError(friendlyError(problem));
            } finally {
              if (useSession.getState().account?.user_id === owner)
                setBusy(false);
            }
          }}
        >
          <EmailPersonPicker
            selected={reported}
            onSelect={setReported}
            disabled={busy}
          />
          <Field label="Reason">
            <select
              value={reason}
              disabled={busy}
              onChange={(event) => setReason(event.target.value)}
            >
              {reasons.map(([value, label]) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </select>
          </Field>
          <Field label="First message">
            <textarea
              required
              minLength={1}
              maxLength={1000}
              rows={4}
              disabled={busy}
              value={note}
              onChange={(event) => setNote(event.target.value)}
              placeholder="Describe what happened and when."
            />
          </Field>
          {formError && <p role="alert">{formError}</p>}
          <Button disabled={!reported || !note.trim() || busy}>
            {busy ? "Starting conversation…" : "Start conversation"}
          </Button>
        </form>
      </Modal>
      <Modal
        title={
          blockAction?.blocked ? "Block this contact?" : "Unblock this contact?"
        }
        open={blockAction !== null}
        onOpenChange={(value) => {
          if (!value && !busy) setBlockAction(null);
        }}
      >
        {blockAction && (
          <>
            <PersonIdentity person={blockAction.person} />
            <p>
              {blockAction.blocked
                ? "This contact will be excluded from future matches. A group you currently share may be cancelled."
                : "You may be matched with this contact again."}
            </p>
            {formError && <p role="alert">{formError}</p>}
            <div className="intro-actions">
              <Button
                variant="ghost"
                disabled={busy}
                onClick={() => setBlockAction(null)}
              >
                Cancel
              </Button>
              <Button
                disabled={busy}
                onClick={async () => {
                  if (busy) return;
                  const owner = identity;
                  setBusy(true);
                  setFormError("");
                  try {
                    await setContactBlocked(
                      blockAction.person,
                      blockAction.blocked,
                    );
                    if (useSession.getState().account?.user_id !== owner)
                      return;
                    setBlockAction(null);
                    setSelected(null);
                    await load();
                  } catch (problem) {
                    if (useSession.getState().account?.user_id === owner)
                      setFormError(friendlyError(problem));
                  } finally {
                    if (useSession.getState().account?.user_id === owner)
                      setBusy(false);
                  }
                }}
              >
                {busy
                  ? "Saving…"
                  : blockAction.blocked
                    ? "Block contact"
                    : "Unblock contact"}
              </Button>
            </div>
          </>
        )}
      </Modal>
    </>
  );
}
