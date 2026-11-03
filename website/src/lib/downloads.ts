import { adminRequest } from './admin';
export { adminRequest } from './admin';
import { apiOrigin, firebaseAuth } from './session';
export interface DownloadSettings {
  android_prompt: boolean; play_store_url: string; support_email: string;
  apk_version: string; apk_sha256: string; apk_size: number;
}
export async function downloadSettings(signal?: AbortSignal): Promise<DownloadSettings> {
  const response = await fetch(`${apiOrigin()}/api/v1/public/settings`, { signal, credentials: 'omit' });
  if (!response.ok) throw new Error('Download settings are unavailable.');
  return response.json();
}
export async function uploadAPK(file: File, version: string, progress: (percent: number) => void): Promise<DownloadSettings> {
  if (!file.name.toLowerCase().endsWith('.apk') || !file.size || file.size > 256 * 1024 * 1024) throw new Error('Choose an APK smaller than 256 MB.');
  const digest = await crypto.subtle.digest('SHA-256', await file.arrayBuffer());
  const sha256 = Array.from(new Uint8Array(digest), (v) => v.toString(16).padStart(2, '0')).join('');
  const grant = await adminRequest<{upload_ticket: string}>('android-upload-ticket', 'POST', { version, sha256, size: file.size });
  const user = firebaseAuth.currentUser;
  if (!user) throw new Error('Sign in again.');
  const token = await user.getIdToken();
  return new Promise((resolve, reject) => {
    const request = new XMLHttpRequest();
    request.open('PUT', `${apiOrigin()}/api/v1/admin/android-apk`);
    request.timeout = 300000;
    request.setRequestHeader('Authorization', `Bearer ${token}`);
    request.setRequestHeader('X-Upload-Ticket', grant.upload_ticket);
    request.setRequestHeader('Content-Type', 'application/vnd.android.package-archive');
    request.upload.onprogress = (event) => { if (event.lengthComputable) progress(Math.min(99, Math.floor(event.loaded / event.total * 100))); };
    request.onerror = request.ontimeout = () => reject(new Error('Upload interrupted. Request a fresh upload to retry.'));
    request.onload = () => {
      if (firebaseAuth.currentUser?.uid !== user.uid) { reject(new Error('The signed-in account changed.')); return; }
      if (request.status < 200 || request.status >= 300) { reject(new Error('The APK was not published. Check the file and try again.')); return; }
      try { progress(100); resolve(JSON.parse(request.responseText) as DownloadSettings); }
      catch { reject(new Error('Publication was not confirmed. Refresh before retrying.')); }
    };
    request.send(file);
  });
}
