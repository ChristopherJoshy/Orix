import { useEffect, useRef, useState } from "react";
import { adminRequest } from "../lib/admin";
import { friendlyError, PublicError } from "../lib/errors";
import { useSession } from "../lib/session";
import {
  rupeesToPaise,
  validDonationRows,
  type DonationLeaderboard,
} from "../lib/donation-leaderboard";
import { Button, Field, PageHead } from "../components/ui";
import { Modal } from "../components/Modal";
import { PixelStatus } from "../components/PixelStatus";
import "./leaderboard.css";

type Draft = {
  version: number;
  entries: { id: string; name: string; rupees: string }[];
};
function toDraft(board: DonationLeaderboard): Draft {
  if (
    !Number.isSafeInteger(board?.version) ||
    board.version < 0 ||
    !validDonationRows(board.entries) ||
    board.entries.some((row) => typeof row.id !== "string")
  ) {
    throw new PublicError("The saved leaderboard couldn’t load. Please retry.");
  }
  return {
    version: board.version,
    entries: board.entries.map((row) => ({
      id: row.id,
      name: row.name,
      rupees: (row.amount_paise / 100).toFixed(2),
    })),
  };
}

export default function AdminLeaderboard() {
  const owner = useSession((s) => s.account?.user_id);
  const [snapshot, setSnapshot] = useState<{
    owner: string;
    draft: Draft;
  } | null>(null);
  const draft = snapshot && snapshot.owner === owner ? snapshot.draft : null;
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [confirmed, setConfirmed] = useState(false);
  const [reload, setReload] = useState(false);
  const revision = useRef(0);
  const sending = useRef(false);

  useEffect(() => {
    setSnapshot(null);
    setError("");
    setNotice("");
    setConfirmed(false);
    setReload(false);
    sending.current = false;
    void load();
    return () => {
      revision.current++;
    };
  }, [owner]);

  async function load() {
    if (!owner || sending.current) return;
    const current = ++revision.current;
    sending.current = true;
    setBusy(true);
    setError("");
    try {
      const result = toDraft(
        await adminRequest<DonationLeaderboard>("donation-leaderboard", "GET"),
      );
      if (
        current === revision.current &&
        owner === useSession.getState().account?.user_id
      ) {
        setSnapshot({ owner, draft: result });
        setConfirmed(false);
        setNotice("");
      }
    } catch (problem) {
      if (current === revision.current) setError(friendlyError(problem));
    } finally {
      if (current === revision.current) {
        sending.current = false;
        setBusy(false);
      }
    }
  }
  function edit(next: Draft) {
    if (!owner) return;
    setSnapshot({ owner, draft: next });
    setConfirmed(false);
    setNotice("");
  }
  async function save(event: React.FormEvent) {
    event.preventDefault();
    if (!owner || !draft || !confirmed || sending.current) return;
    const entries = draft.entries.map((row) => ({
      id: row.id,
      name: row.name.trim(),
      amount_paise: rupeesToPaise(row.rupees),
    }));
    if (entries.some((row) => !row.name || row.amount_paise === null)) {
      setError(
        "Add a name and a valid total in rupees for each supporter. Use no more than two decimal places.",
      );
      return;
    }
    const current = ++revision.current;
    sending.current = true;
    setBusy(true);
    setError("");
    setNotice("");
    try {
      const result = toDraft(
        await adminRequest<DonationLeaderboard>(
          "set-donation-leaderboard",
          "PUT",
          {
            version: draft.version,
            entries,
            publication_confirmed: confirmed,
          },
        ),
      );
      if (
        current === revision.current &&
        owner === useSession.getState().account?.user_id
      ) {
        setSnapshot({ owner, draft: result });
        setConfirmed(false);
        setNotice(
          "Leaderboard published. Thank you for keeping every contribution accurate.",
        );
      }
    } catch (problem) {
      if (current === revision.current) setError(friendlyError(problem));
    } finally {
      if (current === revision.current) {
        sending.current = false;
        setBusy(false);
      }
    }
  }

  return (
    <div className="donation-editor">
      <PageHead title="Donation leaderboard" subtitle="Admin · Supporters" />
      <p>
        Add one row per supporter with their verified total donation in rupees,
        up to ₹1,00,00,000 with two decimal places. The highest totals appear
        first. Publish only names and totals the supporter has agreed to share.
        No payment is processed here.
      </p>
      {error && (
        <div role="alert">
          <p>{error} Your unsaved entries stay here.</p>
          <Button
            variant="secondary"
            disabled={busy}
            onClick={() => (draft ? setReload(true) : void load())}
          >
            {draft ? "Reload saved version" : "Retry"}
          </Button>
        </div>
      )}
      {!draft && !error && (
        <PixelStatus state="loading" label="Loading saved donations" />
      )}
      {draft && (
        <DonationEditorForm
          draft={draft}
          busy={busy}
          confirmed={confirmed}
          edit={edit}
          setConfirmed={setConfirmed}
          save={save}
        />
      )}
      <p role="status">{notice}</p>
      <Modal
        open={reload}
        onOpenChange={setReload}
        title="Reload the saved leaderboard?"
      >
        <p>
          This replaces your unsaved entries with the last published version.
        </p>
        <div className="intro-actions">
          <Button variant="secondary" onClick={() => setReload(false)}>
            Keep editing
          </Button>
          <Button
            onClick={() => {
              setReload(false);
              void load();
            }}
          >
            Reload saved version
          </Button>
        </div>
      </Modal>
    </div>
  );
}

export function DonationEditorForm({
  draft,
  busy,
  confirmed,
  edit,
  setConfirmed,
  save,
}: {
  draft: Draft;
  busy: boolean;
  confirmed: boolean;
  edit: (draft: Draft) => void;
  setConfirmed: (value: boolean) => void;
  save: (event: React.FormEvent) => void | Promise<void>;
}) {
  return (
    <form onSubmit={(event) => void save(event)}>
      <fieldset disabled={busy}>
        <legend className="sr-only">Published supporters</legend>
        <div className="donation-editor-rows">
          {draft.entries.map((row, index) => (
            <div className="donation-editor-row" key={row.id}>
              <Field label={`Supporter ${index + 1} · Name`}>
                <input
                  value={row.name}
                  required
                  maxLength={60}
                  autoComplete="off"
                  onChange={(event) =>
                    edit({
                      ...draft,
                      entries: draft.entries.map((item) =>
                        item.id === row.id
                          ? { ...item, name: event.target.value }
                          : item,
                      ),
                    })
                  }
                />
              </Field>
              <Field label="Total donated (₹)">
                <input
                  value={row.rupees}
                  required
                  inputMode="decimal"
                  maxLength={11}
                  autoComplete="off"
                  aria-label={`Total donated in rupees for supporter ${index + 1}`}
                  onChange={(event) =>
                    edit({
                      ...draft,
                      entries: draft.entries.map((item) =>
                        item.id === row.id
                          ? { ...item, rupees: event.target.value }
                          : item,
                      ),
                    })
                  }
                />
              </Field>
              <Button
                type="button"
                variant="ghost"
                aria-label={`Remove supporter ${index + 1}`}
                onClick={() =>
                  edit({
                    ...draft,
                    entries: draft.entries.filter((item) => item.id !== row.id),
                  })
                }
              >
                Remove
              </Button>
            </div>
          ))}
        </div>
        {draft.entries.length === 0 && (
          <p>No supporters published yet. Add a verified donation to begin.</p>
        )}
        <Button
          type="button"
          variant="secondary"
          disabled={draft.entries.length >= 50}
          onClick={() =>
            edit({
              ...draft,
              entries: [
                ...draft.entries,
                { id: crypto.randomUUID(), name: "", rupees: "" },
              ],
            })
          }
        >
          Add supporter
        </Button>
        <label className="donation-editor-consent">
          <input
            type="checkbox"
            checked={confirmed}
            onChange={(event) => setConfirmed(event.target.checked)}
            required
          />
          <span>
            I have verified these totals and have permission to publish these
            names and amounts.
          </span>
        </label>
        <Button type="submit" disabled={!confirmed || busy}>
          {busy ? "Publishing…" : "Publish leaderboard"}
        </Button>
      </fieldset>
    </form>
  );
}
