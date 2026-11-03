import { useEffect, useState } from 'react';
import { Modal } from './Modal';
import { PixelMark } from './PixelArt';
import { downloadSettings, type DownloadSettings } from '../lib/downloads';
import { apiOrigin } from '../lib/session';
export function AndroidDownload() {
  const [settings, setSettings] = useState<DownloadSettings | null>(null);
  const [open, setOpen] = useState(false);
  useEffect(() => {
    if (!/Android/i.test(navigator.userAgent) || sessionStorage.getItem('orix-android-dismissed')) return;
    const controller = new AbortController();
    void downloadSettings(controller.signal).then((value) => {
      setSettings(value); setOpen(value.android_prompt);
    }).catch(() => { /* The website remains usable if download settings fail. */ });
    return () => controller.abort();
  }, []);
  return <Modal title="Orix on Android" open={open} onOpenChange={(value) => {
    setOpen(value); if (!value) sessionStorage.setItem('orix-android-dismissed', '1');
  }}>
    <PixelMark className="onboarding-brand" />
    <p>Keep your groups and pickup updates close.</p>
    {settings?.play_store_url
      ? <a className="button primary" href={settings.play_store_url} rel="noopener noreferrer">Get it on Google Play</a>
      : <><p>Coming soon to Google Play.</p>{settings?.apk_version
        ? <><a className="button primary" href={`${apiOrigin()}/downloads/android.apk`}>Download APK · {settings.apk_version}</a><p className="muted">Install only the Orix APK from this website. Android may ask you to allow installation from your browser.</p><details><summary>Verify this download</summary><p className="download-hash">SHA-256: {settings.apk_sha256}</p></details></>
        : <p>The APK download will appear here when a release is published. You can continue on the website.</p>}</>}
  </Modal>;
}
