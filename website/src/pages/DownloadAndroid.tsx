import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { downloadSettings, type DownloadSettings } from "../lib/downloads";
import { apiOrigin } from "../lib/session";
import { Logo, PageHead, Button } from "../components/ui";

export default function DownloadAndroid() {
  const [settings, setSettings] = useState<DownloadSettings | null>(null);
  const [error, setError] = useState(false);
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    setError(false);
    void downloadSettings(controller.signal)
      .then((value) => {
        if (!controller.signal.aborted) setSettings(value);
      })
      .catch(() => {
        if (!controller.signal.aborted) setError(true);
      });
    return () => controller.abort();
  }, [attempt]);
  return (
    <main className="android-download-page">
      <Logo />
      <PageHead
        title="ORIX, in your pocket."
        subtitle="Download the latest Android release."
      />
      {error ? (
        <p role="alert">
          We couldn’t check the release. Please try again.{" "}
          <Button
            variant="secondary"
            onClick={() => setAttempt((value) => value + 1)}
          >
            Retry
          </Button>
        </p>
      ) : !settings ? (
        <p role="status">Checking the latest release…</p>
      ) : settings.apk_version &&
        settings.apk_sha256 &&
        settings.apk_size > 0 ? (
        <section className="panel">
          <h2>ORIX {settings.apk_version}</h2>
          <p>
            Refined pixel layouts, optional donations and a little Manglish
            nudge.
          </p>
          <a
            className="button primary"
            href={`${apiOrigin()}/downloads/android.apk`}
          >
            Download Android APK
          </a>
          <p>{(settings.apk_size / 1024 / 1024).toFixed(1)} MB · Android</p>
          <p>
            Install only the APK from this website. Your browser or Android may
            ask you to allow installation.
          </p>
          <details>
            <summary>Verify your download</summary>
            <p>SHA-256</p>
            <code className="download-hash">{settings.apk_sha256}</code>
          </details>
          {settings.play_store_url && (
            <p>
              <a href={settings.play_store_url} rel="noopener noreferrer">
                View on Google Play
              </a>
            </p>
          )}
        </section>
      ) : (
        <p>
          A release is not available yet.{" "}
          <Button
            variant="secondary"
            onClick={() => setAttempt((value) => value + 1)}
          >
            Check again
          </Button>
        </p>
      )}
      <Link className="button secondary" to="/">
        Back to ORIX
      </Link>
    </main>
  );
}
