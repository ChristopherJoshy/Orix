import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { adminRequest, adminRequestPage } from "../lib/admin";
import { friendlyError } from "../lib/errors";
import { firebaseAuth, useSession } from "../lib/session";
import { useResource } from "../lib/resource";
import { refreshJourneys } from "../lib/journeys";
import { Badge, Button, Empty, Field, PageHead } from "../components/ui";
import { PixelStatus } from "../components/PixelStatus";
import { Modal } from "../components/Modal";
import { AdminRelease } from "../components/AdminRelease";
export { default as AdminReports } from "./AdminReports";
function readAdmin<T>(path: string): Promise<T> {
  const [action, search = ""] = path.split("?");
  return adminRequest<T>(
    action,
    "GET",
    undefined,
    Object.fromEntries(new URLSearchParams(search)),
  );
}
function Failure({ error, retry }: { error: string; retry: () => void }) {
  return error ? (
    <p role="alert">
      {error}{" "}
      <Button variant="secondary" onClick={retry}>
        Retry
      </Button>
    </p>
  ) : null;
}
interface Stats {
  users: { total: number; active: number };
  rides: { pending_requests: number; active_groups: number };
  reports: { pending: number };
  maintenance_mode: boolean;
}
export function AdminOverview() {
  const { data, error, load } = useResource<Stats>("stats", 30000, readAdmin);
  return (
    <>
      <PageHead title="ORIX control room." subtitle="Live service records" />
      <Failure error={error} retry={() => void load()} />
      {!data && !error ? (
        <PixelStatus state="loading" label="Loading service totals" />
      ) : (
        data && (
          <>
            <div className="quick-modes">
              {[
                ["Accounts", data.users.total],
                ["Active accounts", data.users.active],
                ["Searching requests", data.rides.pending_requests],
                ["Active groups", data.rides.active_groups],
                ["Pending reports", data.reports.pending],
              ].map(([label, value]) => (
                <section className="panel" key={label}>
                  <h2>{value}</h2>
                  <p>{label}</p>
                </section>
              ))}
            </div>
            <p>
              {data.maintenance_mode
                ? "Maintenance is enabled."
                : "Maintenance is off."}
            </p>
          </>
        )
      )}
      <div className="intro-actions">
        <Link className="button secondary" to="/admin/reports">
          Review reports
        </Link>
        <Link className="button secondary" to="/admin/support">
          Support inbox
        </Link>
        <Link className="button secondary" to="/admin/operations">
          Service operations
        </Link>
      </div>
    </>
  );
}
interface Person {
  user_id: string;
  display_name: string;
  email: string;
  role: string;
  status: string;
  occupation?: string;
  photo_url?: string | null;
  phone?: string | null;
  gender?: string | null;
}
function AdminPhoto({ url, name, userID }: { url?: string | null; name: string; userID: string }) {
  const owner = useSession(s => s.account?.user_id);
  const [src, setSrc] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0); const [failed, setFailed] = useState(false);
  useEffect(() => {
    setSrc(null); setFailed(false); if (!url || !owner) return;
    const controller = new AbortController(); let objectURL: string | undefined;
    if (url.startsWith("/api/v1/users/")) {
      const user = firebaseAuth.currentUser; if (!user) return;
      void user.getIdToken().then(token => fetch(`/api/admin/avatar?${new URLSearchParams({user_id:userID})}`, {headers:{Authorization:`Bearer ${token}`},signal:controller.signal,credentials:"omit"})).then(response => {if (!response.ok) throw new Error(); return response.blob();}).then(blob => { if (controller.signal.aborted || firebaseAuth.currentUser?.uid !== user.uid) return; objectURL=URL.createObjectURL(blob); setSrc(objectURL); }).catch(() => {if (!controller.signal.aborted) setFailed(true);});
    } else { try { const parsed = new URL(url); if (parsed.protocol === "https:" && !parsed.username && !parsed.password) setSrc(parsed.href); else setFailed(true); } catch { setFailed(true); } }
    return () => {controller.abort(); if (objectURL) URL.revokeObjectURL(objectURL);};
  }, [url,userID,owner,attempt]);
  if (src && !failed) return <img className="admin-person-photo" src={src} alt={`${name}'s profile`} referrerPolicy="no-referrer" onError={() => setFailed(true)} />;
  if (url && failed) return <button type="button" className="admin-person-photo admin-photo-empty" aria-label={`Retry photo for ${name}`} onClick={event => {event.stopPropagation();setAttempt(value => value+1);}}>Retry</button>;
  return <span className="admin-person-photo admin-photo-empty" aria-label={url ? "Loading photo" : "No profile photo"}>{url ? "…" : (name?.charAt(0).toUpperCase() || "?")}</span>;
}
function useAdminPages<T>(action: string, owner: string | undefined, groupId?: string | null) {
  const [rows, setRows] = useState<T[] | null>(null);
  const [cursor, setCursor] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const version = useRef(0);
  const scope = useRef({owner, groupId});
  const load = useCallback(async (before?: string) => {
    if (!owner || (action === "group-chat" && !groupId)) return;
    const current = ++version.current;
    setLoading(true); setError("");
    try {
      const page = await adminRequestPage<T[]>(action, "GET", undefined, { ...(groupId ? {group_id: groupId} : {}), ...(before ? {before} : {}) });
      if (!Array.isArray(page.data)) throw new Error("Invalid list response");
      if (current === version.current && owner === useSession.getState().account?.user_id) {
        setRows(previous => before ? [...(previous ?? []), ...page.data] : page.data); setCursor(page.nextCursor);
      }
    } catch (problem) { if (current === version.current) setError(friendlyError(problem)); }
    finally { if (current === version.current) setLoading(false); }
  }, [action, groupId, owner]);
  useEffect(() => { scope.current = {owner, groupId}; setRows(null); setCursor(null); setError(""); void load(); return () => { version.current++; }; }, [load, owner, groupId]);
  const sameScope = scope.current.owner === owner && scope.current.groupId === groupId;
  return {rows: sameScope ? rows : null, cursor: sameScope ? cursor : null, error: sameScope ? error : "", loading, load};
}
export function AdminPeople() {
  const owner = useSession((s) => s.account?.user_id);
  const directory = useAdminPages<Person>("people", owner);
  const [query, setQuery] = useState("");
  const [person, setPerson] = useState<Person | null>(null);
  const [role, setRole] = useState("user");
  const [note, setNote] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [confirm, setConfirm] = useState<"role" | "restore" | null>(null);
  const sending = useRef(false);
  const revision = useRef(0);
  useEffect(
    () => () => {
      revision.current++;
    },
    [],
  );
  async function lookup(event?: React.FormEvent) {
    event?.preventDefault();
    if (sending.current) return;
    sending.current = true;
    setBusy(true);
    setError("");
    setPerson(null);
    const request = ++revision.current;
    try {
      const result = await adminRequest<Person>(
        "lookup",
        "GET",
        undefined,
        query.includes("@")
          ? { email: query.trim() }
          : { user_id: query.trim() },
      );
      if (
        owner === useSession.getState().account?.user_id &&
        request === revision.current
      ) {
        setPerson(result);
        setRole(result.role);
      }
    } catch (problem) {
      if (request === revision.current) setError(friendlyError(problem));
    } finally {
      sending.current = false;
      if (request === revision.current) setBusy(false);
    }
  }
  async function change() {
    if (!person || !confirm || sending.current) return;
    sending.current = true;
    setBusy(true);
    setError("");
    try {
      await adminRequest(
        confirm === "role" ? "user-role" : "restore-account",
        confirm === "role" ? "PUT" : "POST",
        confirm === "role" ? { role } : { note: note.trim() },
        { user_id: person.user_id },
      );
      if (owner === useSession.getState().account?.user_id) {
        setConfirm(null);
        refreshJourneys();
        sending.current = false;
        await lookup();
      }
    } catch (problem) {
      if (owner === useSession.getState().account?.user_id)
        setError(friendlyError(problem));
    } finally {
      sending.current = false;
      if (owner === useSession.getState().account?.user_id) setBusy(false);
    }
  }
  return (
    <>
      <PageHead
        title="People."
        subtitle="Browse accounts or find someone by email or user ID"
      />
      <form className="panel" onSubmit={lookup}>
        <Field label="Email or user ID">
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            maxLength={320}
            required
            disabled={busy}
          />
        </Field>
        <Button disabled={busy || !query.trim()}>
          {busy ? "Searching…" : "Find account"}
        </Button>
      </form>
      <Failure error={directory.error} retry={() => void directory.load()} />
      <section className="panel admin-directory">
        <div className="admin-directory-heading"><h2>All people</h2><Button variant="secondary" disabled={directory.loading} onClick={() => void directory.load()}>Refresh</Button></div>
        {directory.rows === null && directory.loading && <PixelStatus state="loading" label="Loading people" />}
        {directory.rows?.length === 0 && <p>No accounts yet.</p>}
        {directory.rows?.map(item => <div className="admin-directory-row" key={item.user_id}><AdminPhoto url={item.photo_url} name={item.display_name} userID={item.user_id}/><button type="button" className="admin-directory-select" onClick={() => {setPerson(item);setQuery(item.user_id);setRole(item.role);setError("");}}><strong>{item.display_name || "Name unavailable"}</strong><small>{item.email}</small></button><Badge>{item.status}</Badge></div>)}
        {directory.cursor && <Button variant="secondary" disabled={directory.loading} onClick={() => void directory.load(directory.cursor!)}>Load more people</Button>}
      </section>
      <Failure error={error} retry={() => void lookup()} />
      {person && (
        <section className="panel">
          <AdminPhoto url={person.photo_url} name={person.display_name} userID={person.user_id} />
          <h2>{person.display_name}</h2>
          <p>{person.email}</p>
          <p>{person.phone || "Phone not provided"} · {person.gender || "Gender not provided"}</p>
          <Badge>{person.status}</Badge>
          <p>
            {person.user_id} · {person.role} · {person.occupation}
          </p>
          <Field label="Access role">
            <select
              value={role}
              disabled={busy}
              onChange={(event) => setRole(event.target.value)}
            >
              <option value="user">User</option>
              <option value="admin">Administrator</option>
            </select>
          </Field>
          <Button
            disabled={busy || role === person.role}
            onClick={() => setConfirm("role")}
          >
            Review role change
          </Button>
          {["banned", "suspended"].includes(person.status) && (
            <>
              <Field label="Restoration review note">
                <textarea
                  value={note}
                  maxLength={1000}
                  disabled={busy}
                  onChange={(event) => setNote(event.target.value)}
                />
              </Field>
              <Button
                variant="secondary"
                disabled={busy || !note.trim()}
                onClick={() => setConfirm("restore")}
              >
                Review restoration
              </Button>
            </>
          )}
        </section>
      )}
      <Modal
        title={
          confirm === "role"
            ? "Change this account’s access?"
            : "Restore this account?"
        }
        description="This affects the real account and is recorded in the audit log. The service rechecks your authorization before applying the change."
        open={confirm !== null}
        onOpenChange={(open) => {
          if (!open && !busy) setConfirm(null);
        }}
      >
        <Button disabled={busy} onClick={() => void change()}>
          {busy ? "Applying…" : "Confirm change"}
        </Button>
        <Button
          variant="secondary"
          disabled={busy}
          onClick={() => setConfirm(null)}
        >
          Keep current state
        </Button>
      </Modal>
    </>
  );
}
interface Group {
  group_id: string;
  status: string;
  mode: string;
  pickup_summary: string;
  drop_summary: string;
  departure_start: string;
  members: {
    user_id: string;
    display_name: string;
    email: string;
    photo_url?: string | null;
    occupation: string;
    phone?: string | null;
    gender?: string | null;
    status: string;
    role: string;
    readiness_confirmed: boolean;
    is_here: boolean;
    finished: boolean;
  }[];
}
export function AdminRides() {
  const owner = useSession((s) => s.account?.user_id);
  const directory = useAdminPages<Group>("groups", owner);
  const [id, setID] = useState("");
  const [selected, setSelected] = useState<string | null>(null);
  const {
    data: group,
    error,
    load,
  } = useResource<Group>(
    selected ? `group?group_id=${encodeURIComponent(selected)}` : null,
    30000,
    readAdmin,
  );
  const [note, setNote] = useState("");
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState("");
  const sending = useRef(false);
  async function cancel() {
    if (!selected || sending.current) return;
    sending.current = true;
    setBusy(true);
    setFailure("");
    try {
      await adminRequest(
        "cancel-group",
        "POST",
        { note: note.trim() },
        { group_id: selected },
      );
      if (owner === useSession.getState().account?.user_id) {
        setOpen(false);
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
  return (
    <>
      <PageHead
        title="Journey groups."
        subtitle="Browse journeys and review their members and conversations"
      />
      <form
        className="panel"
        onSubmit={(event) => {
          event.preventDefault();
          setSelected(id.trim());
          setFailure("");
          void load();
        }}
      >
        <Field label="Group ID">
          <input
            required
            maxLength={100}
            value={id}
            onChange={(event) => setID(event.target.value)}
          />
        </Field>
        <Button>Open group</Button>
      </form>
      <Failure error={directory.error} retry={() => void directory.load()} />
      <section className="panel admin-directory">
        <div className="admin-directory-heading"><h2>All journey groups</h2><Button variant="secondary" disabled={directory.loading} onClick={() => void directory.load()}>Refresh</Button></div>
        {directory.rows === null && directory.loading && <PixelStatus state="loading" label="Loading groups" />}
        {directory.rows?.length === 0 && <p>No journey groups yet.</p>}
        {directory.rows?.map(item => <div className="admin-directory-row" key={item.group_id}><span className="admin-group-photos">{item.members.slice(0, 4).map(member => <AdminPhoto key={member.user_id} url={member.photo_url} name={member.display_name} userID={member.user_id}/>)}</span><button type="button" className="admin-directory-select" onClick={() => {setID(item.group_id);setSelected(item.group_id);setFailure("");}}><strong>{item.pickup_summary} → {item.drop_summary}</strong><small>{new Date(item.departure_start).toLocaleString()} · {item.members.length} members</small></button><Badge>{item.status}</Badge></div>)}
        {directory.cursor && <Button variant="secondary" disabled={directory.loading} onClick={() => void directory.load(directory.cursor!)}>Load more groups</Button>}
      </section>
      <Failure error={failure || error} retry={() => void load()} />
      {group && (
        <section className="panel">
          <Badge>{group.status}</Badge>
          <h2>
            {group.pickup_summary} → {group.drop_summary}
          </h2>
          <p>{new Date(group.departure_start).toLocaleString()}</p>
          <h3>Members</h3>
          {group.members.map(member => <div className="admin-directory-row admin-member" key={member.user_id}><AdminPhoto url={member.photo_url} name={member.display_name} userID={member.user_id}/><span><strong>{member.display_name || "Name unavailable"}</strong><small>{member.email || "Email unavailable"} · {member.occupation || "Occupation not provided"} · {member.role || "Member"}</small><small>{member.phone || "Phone not provided"} · {member.gender || "Gender not provided"}</small><small>{member.finished ? "Finished" : member.is_here ? "At pickup" : member.readiness_confirmed ? "Confirmed" : "Awaiting confirmation"} · {member.status}</small></span></div>)}
          <AdminGroupChat key={`${owner}:${group.group_id}`} group={group} owner={owner} />
          {!["completed", "cancelled", "expired"].includes(group.status) && (
            <>
              <Field label="Cancellation review note">
                <textarea
                  required
                  minLength={3}
                  maxLength={1000}
                  value={note}
                  onChange={(event) => setNote(event.target.value)}
                />
              </Field>
              <Button
                variant="secondary"
                disabled={busy || note.trim().length < 3}
                onClick={() => setOpen(true)}
              >
                Review cancellation
              </Button>
            </>
          )}
        </section>
      )}
      <Modal
        title="Cancel this real group?"
        description="All affected members will be notified. Record the safety or operational reason before continuing."
        open={open}
        onOpenChange={(value) => {
          if (!busy) setOpen(value);
        }}
      >
        <Button disabled={busy} onClick={() => void cancel()}>
          Cancel group
        </Button>
        <Button
          variant="secondary"
          disabled={busy}
          onClick={() => setOpen(false)}
        >
          Keep group
        </Button>
      </Modal>
    </>
  );
}
interface AdminMessage { message_id: string; user_display_name: string; content: string; created_at: string; is_mine: boolean }
function AdminGroupChat({group, owner}: {group: Group; owner?: string}) {
  const chat = useAdminPages<AdminMessage>("group-chat", owner, group.group_id);
  const [draft, setDraft] = useState(""); const [error, setError] = useState(""); const [busy, setBusy] = useState(false);
  const sending = useRef(false); const clientID = useRef<string | null>(null);
  async function send(event: React.FormEvent) {
    event.preventDefault(); if (sending.current || !draft.trim()) return;
    sending.current = true; setBusy(true); setError("");
    const content = draft.trim(); const id = clientID.current ?? crypto.randomUUID(); clientID.current = id;
    try { await adminRequest("send-group-chat", "POST", {content, client_message_id:id}, {group_id:group.group_id});
      if (owner === useSession.getState().account?.user_id) { setDraft(""); clientID.current = null; await chat.load(); }
    } catch (problem) { if (owner === useSession.getState().account?.user_id) setError(friendlyError(problem)); }
    finally { sending.current = false; if (owner === useSession.getState().account?.user_id) setBusy(false); }
  }
  return <section className="admin-chat"><div className="admin-directory-heading"><h3>Group conversation</h3><Button variant="secondary" disabled={chat.loading} onClick={() => void chat.load()}>Refresh</Button></div>
    <p className="muted">Messages sent here are visible to group members as ORIX admin.</p>
    <Failure error={chat.error} retry={() => void chat.load()} />
    {chat.cursor && <Button variant="secondary" disabled={chat.loading} onClick={() => void chat.load(chat.cursor!)}>Load older messages</Button>}
    <div className="admin-chat-messages">{chat.rows === null && chat.loading && <PixelStatus state="loading" label="Loading messages" />}{chat.rows?.length === 0 && <p>No messages yet.</p>}{chat.rows && [...chat.rows].reverse().map(message => <article className="admin-chat-message" key={message.message_id}><strong>{message.user_display_name}</strong><time>{new Date(message.created_at).toLocaleString()}</time><p>{message.content}</p></article>)}</div>
    {!["completed", "cancelled", "expired"].includes(group.status) && <form onSubmit={send}><Field label="Message the group as an administrator"><textarea value={draft} maxLength={1000} disabled={busy} onChange={event => {setDraft(event.target.value); clientID.current = null;}} /></Field><Button disabled={busy || !draft.trim()}>{busy ? "Sending…" : "Send message"}</Button></form>}
    <Failure error={error} retry={() => void send({preventDefault(){}} as React.FormEvent)} />
  </section>;
}
interface Maintenance {
  maintenance_mode: boolean;
  message: string;
}
interface JobStatus {
  notification_outbox: Record<string, number>;
  account_deletions: Record<string, number>;
  workers: {
    _id: string;
    heartbeat_at?: string;
    last_success_at?: string;
    last_error?: string;
    last_error_at?: string;
  }[];
}
interface Failed {
  event_id: string;
  type: string;
  attempts: number;
  created_at: string;
}
interface VersionConfig {
  current_version: string;
  min_version: string;
  update_title: string | null;
  update_notes: string | null;
  play_store_url: string | null;
  app_store_url: string | null;
  update_required: boolean;
}
async function readFailedDeliveries(path: string): Promise<Failed[]> {
  const result = await readAdmin<{ events?: unknown }>(path);
  if (!result || !Array.isArray(result.events) || !result.events.every((event: unknown) => {
    if (!event || typeof event !== "object") return false;
    const item = event as Record<string, unknown>;
    return typeof item.event_id === "string" && typeof item.type === "string" &&
      typeof item.attempts === "number" && typeof item.created_at === "string";
  })) throw new Error("Invalid delivery response");
  return result.events as Failed[];
}
async function readJobStatus(path: string): Promise<JobStatus> {
  const result = await readAdmin<JobStatus>(path);
  if (!result || !result.notification_outbox || !result.account_deletions ||
      !Array.isArray(result.workers) || !result.workers.every((worker) =>
        worker && typeof worker._id === "string")) throw new Error("Invalid job response");
  return result;
}
function VersionControls() {
  const owner = useSession((s) => s.account?.user_id);
  const { data, error, loading, load } = useResource<VersionConfig>(
    "/app/version",
    30000,
  );
  const [draft, setDraft] = useState<VersionConfig | null>(null);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState("");
  const [confirm, setConfirm] = useState(false);
  const sending = useRef(false);
  useEffect(() => {
    setDraft(null);
    setFailure("");
    setConfirm(false);
  }, [owner]);
  async function save() {
    if (!draft || sending.current) return;
    sending.current = true;
    setBusy(true);
    setFailure("");
    try {
      await adminRequest("set-version", "PUT", draft);
      if (owner === useSession.getState().account?.user_id) {
        setConfirm(false);
        setDraft(null);
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
  return (
    <section className="panel">
      <h2>App version</h2>
      <p>
        Set the current version and the minimum supported version. Changes
        affect real users.
      </p>
      <Failure error={error || failure} retry={() => void load()} />
      {!data && !error ? (
        <PixelStatus state="loading" label="Loading version settings" />
      ) : data && !draft ? (
        <>
          <p>
            Current: {data.current_version} · Minimum: {data.min_version}
          </p>
          <Button
            variant="secondary"
            disabled={loading}
            onClick={() => setDraft({ ...data })}
          >
            Edit version settings
          </Button>
        </>
      ) : null}
      {draft && (
        <form
          onSubmit={(event) => {
            event.preventDefault();
            setConfirm(true);
          }}
        >
          <Field label="Current version">
            <input
              required
              value={draft.current_version}
              onChange={(event) =>
                setDraft({ ...draft, current_version: event.target.value })
              }
            />
          </Field>
          <Field label="Minimum supported version">
            <input
              required
              value={draft.min_version}
              onChange={(event) =>
                setDraft({ ...draft, min_version: event.target.value })
              }
            />
          </Field>
          <Field label="Update title">
            <input
              maxLength={200}
              value={draft.update_title ?? ""}
              onChange={(event) =>
                setDraft({ ...draft, update_title: event.target.value || null })
              }
            />
          </Field>
          <Field label="Update notes">
            <textarea
              maxLength={10000}
              value={draft.update_notes ?? ""}
              onChange={(event) =>
                setDraft({ ...draft, update_notes: event.target.value || null })
              }
            />
          </Field>
          <Field label="Google Play URL">
            <input
              type="url"
              value={draft.play_store_url ?? ""}
              onChange={(event) =>
                setDraft({
                  ...draft,
                  play_store_url: event.target.value || null,
                })
              }
            />
          </Field>
          <Field label="App Store URL">
            <input
              type="url"
              value={draft.app_store_url ?? ""}
              onChange={(event) =>
                setDraft({
                  ...draft,
                  app_store_url: event.target.value || null,
                })
              }
            />
          </Field>
          <Button disabled={busy}>Review version settings</Button>
          <Button
            type="button"
            variant="secondary"
            disabled={busy}
            onClick={() => setDraft(null)}
          >
            Cancel
          </Button>
        </form>
      )}
      <Modal
        title="Update app version settings?"
        description="The minimum version can prevent older app versions from accessing ORIX."
        open={confirm}
        onOpenChange={(value) => {
          if (!busy) setConfirm(value);
        }}
      >
        <Button disabled={busy} onClick={() => void save()}>
          Save settings
        </Button>
        <Button
          variant="secondary"
          disabled={busy}
          onClick={() => setConfirm(false)}
        >
          Keep editing
        </Button>
      </Modal>
    </section>
  );
}
export function Operations() {
  const owner = useSession((s) => s.account?.user_id);
  const maintenance = useResource<Maintenance>("maintenance", 30000, readAdmin);
  const jobs = useResource<JobStatus>("jobs", 30000, readJobStatus);
  const failed = useResource<Failed[]>("failed", 30000, readFailedDeliveries);
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [confirm, setConfirm] = useState(false);
  const sending = useRef(false);
  async function mutate(
    action: string,
    body?: unknown,
    query?: Record<string, string>,
  ) {
    if (sending.current) return;
    sending.current = true;
    setBusy(true);
    setError("");
    try {
      await adminRequest(action, "POST", body, query);
      if (owner === useSession.getState().account?.user_id) {
        setConfirm(false);
        refreshJourneys();
        await Promise.all([maintenance.load(), jobs.load(), failed.load()]);
      }
    } catch (problem) {
      if (owner === useSession.getState().account?.user_id)
        setError(friendlyError(problem));
    } finally {
      sending.current = false;
      if (owner === useSession.getState().account?.user_id) setBusy(false);
    }
  }
  return (
    <>
      <PageHead
        title="Operations."
        subtitle="Real service controls and delivery jobs"
      />
      <AdminRelease key={owner ?? "signed-out"} />
      <VersionControls key={owner ?? "signed-out"} />
      <Failure
        error={error}
        retry={() => {
          void maintenance.load();
          void jobs.load();
          void failed.load();
        }}
      />
      <section className="panel">
        <h2>Maintenance</h2>
        <Failure
          error={maintenance.error}
          retry={() => void maintenance.load()}
        />
        {maintenance.data ? (
          <>
            <p>
              {maintenance.data.maintenance_mode ? "Enabled" : "Disabled"} ·{" "}
              {maintenance.data.message}
            </p>
            <Field label="Message for users">
              <textarea
                value={note}
                maxLength={500}
                onChange={(event) => setNote(event.target.value)}
                placeholder="Why is the service unavailable?"
              />
            </Field>
            <Button
              variant="secondary"
              disabled={
                busy || (!maintenance.data.maintenance_mode && !note.trim())
              }
              onClick={() => setConfirm(true)}
            >
              {maintenance.data.maintenance_mode
                ? "Disable maintenance"
                : "Enable maintenance"}
            </Button>
          </>
        ) : (
          !maintenance.error && (
            <PixelStatus state="loading" label="Loading maintenance state" />
          )
        )}
      </section>
      <section className="panel">
        <h2>Background work</h2>
        <Failure error={jobs.error} retry={() => void jobs.load()} />
        {jobs.data &&
          (["notification_outbox", "account_deletions"] as const).map((key) => (
            <div key={key}>
              <h3>
                {key === "notification_outbox"
                  ? "Notifications"
                  : "Account deletion"}
              </h3>
              {Object.entries(jobs.data![key]).map(([state, count]) => (
                <p key={state}>
                  {state.replaceAll("_", " ")}: {count}
                </p>
              ))}
            </div>
          ))}
        {jobs.data && (
          <div>
            <h3>Worker heartbeats</h3>
            {jobs.data.workers.length === 0 ? (
              <p>No worker heartbeat has been recorded.</p>
            ) : (
              jobs.data.workers.map((worker) => (
                <article key={worker._id}>
                  <strong>{worker._id.replaceAll("_", " ")}</strong>
                  <p>
                    Last heartbeat:{" "}
                    {worker.heartbeat_at
                      ? new Date(worker.heartbeat_at).toLocaleString()
                      : "Not recorded"}
                  </p>
                  <p>
                    Last successful run:{" "}
                    {worker.last_success_at
                      ? new Date(worker.last_success_at).toLocaleString()
                      : "Not recorded"}
                  </p>
                  {worker.last_error && (
                    <p role="status">
                      The last run reported a problem. Check service logs and
                      retry the affected work.
                    </p>
                  )}
                </article>
              ))
            )}
          </div>
        )}
      </section>
      <section className="panel">
        <h2>Failed notification deliveries</h2>
        <Failure error={failed.error} retry={() => void failed.load()} />
        {failed.data?.length === 0 ? (
          <p>No failed deliveries.</p>
        ) : (
          failed.data?.map((event) => (
            <article key={event.event_id}>
              <h3>{event.type.replaceAll("_", " ")}</h3>
              <p>
                {event.attempts} attempts ·{" "}
                {new Date(event.created_at).toLocaleString()}
              </p>
              <Button
                variant="secondary"
                disabled={busy}
                onClick={() =>
                  void mutate(
                    "retry-delivery",
                    {},
                    { event_id: event.event_id },
                  )
                }
              >
                Retry delivery
              </Button>
            </article>
          ))
        )}
      </section>
      <Modal
        open={confirm}
        onOpenChange={(value) => {
          if (!busy) setConfirm(value);
        }}
        title="Change service availability?"
        description="This changes the connected service for real users and is audited."
      >
        <Button
          disabled={busy || !maintenance.data}
          onClick={() =>
            void mutate("set-maintenance", {
              enabled: !maintenance.data?.maintenance_mode,
              message: note.trim(),
            })
          }
        >
          Apply change
        </Button>
        <Button
          variant="secondary"
          disabled={busy}
          onClick={() => setConfirm(false)}
        >
          Keep current state
        </Button>
      </Modal>
    </>
  );
}
interface AuditRow {
  log_id: string;
  timestamp: string;
  actor_id: string;
  actor_name?: string;
  action: string;
  target_id: string;
}
export function Audit() {
  const owner = useSession((s) => s.account?.user_id);
  const [rows, setRows] = useState<AuditRow[]>([]);
  const [cursor, setCursor] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const revision = useRef(0);
  const pending = useRef(false);
  const load = useCallback(
    async (before?: string) => {
      if (pending.current) return;
      pending.current = true;
      setBusy(true);
      const request = ++revision.current;
      try {
        const page = await adminRequestPage<AuditRow[]>(
          "logs",
          "GET",
          undefined,
          before ? { before } : undefined,
        );
        if (!Array.isArray(page.data)) throw new Error("Invalid audit response");
        if (
          request === revision.current &&
          owner === useSession.getState().account?.user_id
        ) {
          setRows((current) =>
            before
              ? [
                  ...new Map(
                    [...current, ...page.data].map((row) => [row.log_id, row]),
                  ).values(),
                ]
              : page.data,
          );
          setCursor(page.nextCursor);
          setError("");
        }
      } catch (problem) {
        if (request === revision.current) setError(friendlyError(problem));
      } finally {
        pending.current = false;
        if (request === revision.current) setBusy(false);
      }
    },
    [owner],
  );
  useEffect(() => {
    setRows([]);
    void load();
    return () => {
      revision.current++;
    };
  }, [load]);
  return (
    <>
      <PageHead
        title="Activity log."
        subtitle="Server-recorded administrative actions"
      >
        <Button variant="secondary" disabled={busy} onClick={() => void load()}>
          Refresh
        </Button>
      </PageHead>
      <Failure error={error} retry={() => void load()} />
      {busy && rows.length === 0 ? (
        <PixelStatus state="loading" label="Loading audit records" />
      ) : rows.length === 0 && !error ? (
        <Empty
          title="No recorded actions."
          text="Administrative actions will appear here."
        />
      ) : (
        rows.map((row) => (
          <article key={row.log_id} className="panel">
            <h2>{row.action.replaceAll("_", " ")}</h2>
            <p>{new Date(row.timestamp).toLocaleString()}</p>
            <p>
              By: {row.actor_name || (row.actor_id === "system" ? "System" : "Name unavailable")}
              {row.target_id && <><br />Reference: {row.target_id}</>}
            </p>
          </article>
        ))
      )}
      {cursor && (
        <Button
          disabled={busy}
          variant="secondary"
          onClick={() => void load(cursor)}
        >
          Load older records
        </Button>
      )}
    </>
  );
}
