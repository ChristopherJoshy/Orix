import { useEffect, useState } from 'react';
import { Modal } from './Modal';
import { PixelMark } from './PixelArt';
import { downloadSettings, type DownloadSettings } from '../lib/downloads';
import { apiOrigin } from '../lib/session';
export function AndroidDownload() {
  const [settings, setSettings] = useState<DownloadSettings | null>(null);
  const [open, setOpen] = useState(false);
  const [failed, setFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    if (!/Android/i.test(navigator.userAgent)) return;
    setFailed(false);
    const controller = new AbortController();
    void downloadSettings(controller.signal).then((value) => {
      setSettings(value);
      const release = value.apk_sha256 || value.play_store_url || "no-release";
      setOpen(value.android_prompt && sessionStorage.getItem('orix-android-dismissed-release') !== release);
    }).catch(() => { if (!controller.signal.aborted) { setFailed(true); setOpen(true); } });
    return () => controller.abort();
  }, [attempt]);
  return <Modal title="Orix on Android" open={open} onOpenChange={(value) => {
    setOpen(value); if (!value && settings) sessionStorage.setItem('orix-android-dismissed-release', settings.apk_sha256 || settings.play_store_url || 'no-release');
  }}>
    <PixelMark className="onboarding-brand" />
    <p>Keep your groups and pickup updates close.</p>
    {failed ? <><p>We couldn’t check the Android release just now. Please try again.</p><button className="button secondary" onClick={() => setAttempt(value => value + 1)}>Retry</button></> : settings?.play_store_url
      ? <a className="button primary" href={settings.play_store_url} rel="noopener noreferrer">Get it on Google Play</a>
      : <><p>Coming soon to Google Play.</p>{settings?.apk_version
        ? <><a className="button primary" href={`${apiOrigin()}/downloads/android.apk`}>Download APK · {settings.apk_version}</a><p className="muted">Install only the Orix APK from this website. Android may ask you to allow installation from your browser.</p><details><summary>Verify this download</summary><p className="download-hash">SHA-256: {settings.apk_sha256}</p></details></>
        : <p>The APK download will appear here when a release is published. You can continue on the website.</p>}</>}
  </Modal>;
}
