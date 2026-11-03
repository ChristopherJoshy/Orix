import { useEffect, useState } from "react";
import { adminRequest } from "../lib/admin";
import { friendlyError } from "../lib/errors";
import { useSession } from "../lib/session";
import { Button, Field, Toggle } from "./ui";

type Message = { id: string; title: string; body: string; enabled: boolean };
type Config = { version: number; enabled: boolean; messages: Message[] };
export function AdminEngagement() {
  const owner = useSession((s) => s.account?.user_id);
  const [draft, setDraft] = useState<Config | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  async function load() {
    setError("");
    setBusy(true);
    try {
      const config = await adminRequest<Config>("engagement", "GET");
      if (!Array.isArray(config.messages) || !Number.isInteger(config.version))
        throw new Error("Invalid reminder settings");
      if (owner === useSession.getState().account?.user_id) setDraft(config);
    } catch (problem) {
      if (owner === useSession.getState().account?.user_id)
        setError(friendlyError(problem));
    } finally {
      if (owner === useSession.getState().account?.user_id) setBusy(false);
    }
  }
  useEffect(() => {
    void load();
  }, [owner]);
  async function save() {
    if (!draft || busy) return;
    setBusy(true);
    setError("");
    setNotice("");
    try {
      const config = await adminRequest<Config>("set-engagement", "PUT", draft);
      if (owner === useSession.getState().account?.user_id) {
        setDraft(config);
        setNotice("Reminder messages saved.");
      }
    } catch (problem) {
      if (owner === useSession.getState().account?.user_id)
        setError(friendlyError(problem));
    } finally {
      if (owner === useSession.getState().account?.user_id) setBusy(false);
    }
  }
  function change(id: string, fields: Partial<Message>) {
    setNotice("");
    setDraft((value) =>
      value
        ? {
            ...value,
            messages: value.messages.map((message) =>
              message.id === id ? { ...message, ...fields } : message,
            ),
          }
        : value,
    );
  }
  return (
    <section className="panel engagement-editor">
      <h2>A little nudge</h2>
      <p>
        Optional Manglish reminders at 8 am, noon, 4 pm and 8 pm IST. Only
        people who enable these reminders receive them. Minimum four hours
        apart; old messages expire after 30 minutes.
      </p>
      {error && (
        <p role="alert">
          {error}{" "}
          <Button
            variant="secondary"
            disabled={busy}
            onClick={() => void load()}
          >
            Reload saved settings
          </Button>
        </p>
      )}
      {!draft && !error && <p>Loading reminder messages…</p>}
      {draft && (
        <fieldset disabled={busy}>
          <Toggle
            checked={draft.enabled}
            onChange={(value) => {
              setNotice("");
              setDraft({ ...draft, enabled: value });
            }}
            label="Enable reminder schedule"
          />
          <p>
            {draft.messages.length} messages. Changes take effect after saving.
            Paused messages remain editable.
          </p>
          <div className="engagement-messages">
            {draft.messages.map((message, index) => (
              <details key={message.id}>
                <summary>
                  {index + 1}. {message.title || "New reminder"}
                  {!message.enabled && " · paused"}
                </summary>
                <Field label="Title">
                  <input
                    maxLength={80}
                    value={message.title}
                    disabled={busy}
                    onChange={(event) =>
                      change(message.id, { title: event.target.value })
                    }
                  />
                </Field>
                <Field label="Message">
                  <textarea
                    maxLength={240}
                    rows={3}
                    value={message.body}
                    disabled={busy}
                    onChange={(event) =>
                      change(message.id, { body: event.target.value })
                    }
                  />
                </Field>
                <Toggle
                  checked={message.enabled}
                  onChange={(value) => change(message.id, { enabled: value })}
                  label="Include in rotation"
                />
                <p className="engagement-preview">
                  <strong>{message.title}</strong>
                  <br />
                  {message.body}
                </p>
              </details>
            ))}
          </div>
          <div className="intro-actions">
            <Button
              variant="secondary"
              disabled={busy || draft.messages.length >= 200}
              onClick={() =>
                setDraft({
                  ...draft,
                  messages: [
                    ...draft.messages,
                    {
                      id: crypto.randomUUID(),
                      title: "",
                      body: "",
                      enabled: true,
                    },
                  ],
                })
              }
            >
              Add message
            </Button>
            <Button
              disabled={
                busy ||
                draft.messages.some(
                  (message) => !message.title.trim() || !message.body.trim(),
                )
              }
              onClick={() => void save()}
            >
              {busy ? "Saving…" : "Save reminder messages"}
            </Button>
          </div>
          <p role="status">{notice}</p>
        </fieldset>
      )}
    </section>
  );
}
