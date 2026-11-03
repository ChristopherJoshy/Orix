import { friendlyError } from "../lib/errors";
import { useEffect, useState } from "react";
import { Button, Field, Toggle } from "./ui";
import { PixelStatus } from "./PixelStatus";
import {
  adminRequest,
  downloadSettings,
  uploadAPK,
  type DownloadSettings,
} from "../lib/downloads";
export function AdminRelease() {
  const [settings, setSettings] = useState<DownloadSettings | null>(null);
  const [version, setVersion] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [verified, setVerified] = useState(false);
  const [busy, setBusy] = useState(false);
  const [percent, setPercent] = useState(0);
  const [message, setMessage] = useState("");
  const [published, setPublished] = useState(false);
  const [loadVersion, setLoadVersion] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    void downloadSettings(controller.signal)
      .then((value) => {
        setSettings(value);
        setMessage("");
      })
      .catch(() => {
        if (!controller.signal.aborted)
          setMessage("Download settings could not be loaded. Try again.");
      });
    return () => controller.abort();
  }, [loadVersion]);
  return (
    <section className="panel">
      <h2>Android release</h2>
      <p>
        Publish the APK visitors can download until Orix is available on Google
        Play. A successful replacement removes the previous file from the
        database.
      </p>
      {message && <p role="status">{message}</p>}
      {settings && <p>Currently published: {settings.apk_version ? `version ${settings.apk_version}` : "No APK published"}</p>}
      <Button
        type="button"
        variant="secondary"
        disabled={busy}
        onClick={() => setLoadVersion((value) => value + 1)}
      >
        Refresh release settings
      </Button>
      {settings && (
        <form
          onSubmit={async (event) => {
            event.preventDefault();
            setBusy(true);
            setMessage("");
            try {
              setSettings(
                await adminRequest("public-settings", "PUT", settings),
              );
              setMessage("Download settings saved.");
            } catch (error) {
              setMessage(friendlyError(error));
            } finally {
              setBusy(false);
            }
          }}
        >
          <fieldset
            disabled={busy}
            style={{ border: 0, padding: 0, margin: 0 }}
          >
            <Toggle
              label="Suggest the Android app to Android visitors"
              checked={settings.android_prompt}
              onChange={(value) =>
                setSettings({ ...settings, android_prompt: value })
              }
            />
            <Field label="Google Play URL (leave empty until published)">
              <input
                type="url"
                value={settings.play_store_url}
                onChange={(e) =>
                  setSettings({ ...settings, play_store_url: e.target.value })
                }
              />
            </Field>
            <Field label="Support email">
              <input
                type="email"
                required
                value={settings.support_email}
                onChange={(e) =>
                  setSettings({ ...settings, support_email: e.target.value })
                }
              />
            </Field>
            <Button disabled={busy}>Save download settings</Button>
          </fieldset>
        </form>
      )}
      <hr />
      <form
        onSubmit={async (event) => {
          event.preventDefault();
          if (!file || !verified) return;
          setBusy(true);
          setPercent(0);
          setPublished(false);
          setMessage("Preparing upload…");
          try {
            setSettings(await uploadAPK(file, version, setPercent));
            setPublished(true);
            setMessage(
              "Release published. Previous APK cleanup has been scheduled.",
            );
          } catch (error) {
            setMessage(friendlyError(error));
          } finally {
            setBusy(false);
          }
        }}
      >
        <Field label="Release version">
          <input
            required
            pattern="[0-9]+\.[0-9]+\.[0-9]+(\+[0-9]+)?"
            value={version}
            onChange={(e) => setVersion(e.target.value)}
            placeholder="2.1.0"
          />
        </Field>
        <Field label="Signed release APK">
          <input
            type="file"
            accept=".apk,application/vnd.android.package-archive"
            disabled={busy}
            onChange={(e) => {
              setFile(e.target.files?.[0] ?? null);
              setVerified(false);
            }}
          />
        </Field>
        <label className="checkbox-label">
          <input
            type="checkbox"
            checked={verified}
            onChange={(e) => setVerified(e.target.checked)}
          />
          <span>
            I verified this APK’s package, release signature and install
            behavior.
          </span>
        </label>
        {busy && (
          <>
            <PixelStatus state="uploading" label="Uploading APK" />
            <progress
              max={100}
              value={percent}
              aria-label="APK upload progress"
            />
            <p>
              {percent}% uploaded{percent === 99 ? " · Publishing…" : ""}
            </p>
          </>
        )}
        {published && <PixelStatus state="uploaded" label="APK published" />}
        <Button disabled={busy || !file || !verified}>Publish APK</Button>
      </form>
    </section>
  );
}
