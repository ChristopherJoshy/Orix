import { adminRequest } from "./admin";
export { adminRequest } from "./admin";
import { apiOrigin, firebaseAuth } from "./session";
import { onAuthStateChanged } from "firebase/auth";
import { PublicError } from "./errors";
import { fetchWithRetry } from "./retry";
export interface DownloadSettings {
  android_prompt: boolean;
  play_store_url: string;
  support_email: string;
  apk_version: string;
  apk_sha256: string;
  apk_size: number;
}
export async function downloadSettings(
  signal?: AbortSignal,
): Promise<DownloadSettings> {
  const response = await fetchWithRetry(`${apiOrigin()}/api/v1/public/settings`, {
    signal,
    credentials: "omit",
  });
  if (!response.ok) throw new Error("Download settings are unavailable.");
  return response.json();
}
export async function uploadAPK(
  file: File,
  version: string,
  progress: (percent: number) => void,
): Promise<DownloadSettings> {
  const user = firebaseAuth.currentUser;
  if (!user) throw new PublicError("Sign in again to publish a release.");
  if (
    !file.name.toLowerCase().endsWith(".apk") ||
    !file.size ||
    file.size > 256 * 1024 * 1024
  )
    throw new PublicError("Choose an APK smaller than 256 MB.");
  const digest = await crypto.subtle.digest(
    "SHA-256",
    await file.arrayBuffer(),
  );
  const sha256 = Array.from(new Uint8Array(digest), (v) =>
    v.toString(16).padStart(2, "0"),
  ).join("");
  if (firebaseAuth.currentUser?.uid !== user.uid)
    throw new PublicError("Your account changed. Start the upload again.");
  const grant = await adminRequest<{ upload_ticket: string }>(
    "android-upload-ticket",
    "POST",
    { version, sha256, size: file.size },
  );
  if (firebaseAuth.currentUser?.uid !== user.uid)
    throw new PublicError("Your account changed. Start the upload again.");
  const token = await user.getIdToken();
  if (firebaseAuth.currentUser?.uid !== user.uid)
    throw new PublicError("Your account changed. Start the upload again.");
  return new Promise((resolve, reject) => {
    const request = new XMLHttpRequest();
    let settled = false;
    let stopAuth = () => {};
    const finish = (error?: PublicError, result?: DownloadSettings) => {
      if (settled) return;
      settled = true;
      stopAuth();
      if (error) reject(error);
      else resolve(result!);
    };
    request.open("PUT", `${apiOrigin()}/api/v1/admin/android-apk`);
    request.timeout = 300000;
    request.setRequestHeader("Authorization", `Bearer ${token}`);
    request.setRequestHeader("X-Upload-Ticket", grant.upload_ticket);
    request.setRequestHeader(
      "Content-Type",
      "application/vnd.android.package-archive",
    );
    request.upload.onprogress = (event) => {
      if (event.lengthComputable && firebaseAuth.currentUser?.uid === user.uid)
        progress(Math.min(99, Math.floor((event.loaded / event.total) * 100)));
    };
    request.onerror =
      request.ontimeout =
      request.onabort =
        () =>
          finish(
            new PublicError(
              "Upload interrupted. Refresh release settings before trying again.",
            ),
          );
    request.onload = () => {
      if (firebaseAuth.currentUser?.uid !== user.uid) {
        finish(
          new PublicError("Your account changed. Refresh release settings."),
        );
        return;
      }
      if (request.status < 200 || request.status >= 300) {
        finish(
          new PublicError(
            "Publication was not confirmed. Refresh release settings before trying again.",
          ),
        );
        return;
      }
      try {
        progress(100);
        finish(undefined, JSON.parse(request.responseText) as DownloadSettings);
      } catch {
        finish(
          new PublicError(
            "Publication was not confirmed. Refresh release settings before trying again.",
          ),
        );
      }
    };
    stopAuth = onAuthStateChanged(firebaseAuth, (current) => {
      if (current?.uid !== user.uid) request.abort();
    });
    request.send(file);
  });
}
