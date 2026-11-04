import { toast } from "sonner";
import { getApps } from "firebase/app";
import {
  getMessaging,
  getToken,
  deleteToken,
  isSupported,
  onMessage,
} from "firebase/messaging";
import { apiRequest, firebaseAuth } from "./session";
import { PublicError } from "./errors";
import { refreshEvent } from "./realtime";

const key = import.meta.env.VITE_FIREBASE_VAPID_KEY?.trim();
const storageKey = "orix-web-push-owner";
// A subscription belongs to the browser profile, so serialize changes across
// tabs as well as within this tab when the browser supports Web Locks.
let subscriptionChange: Promise<unknown> = Promise.resolve();
function serialize<T>(change: () => Promise<T>): Promise<T> {
  const run = () =>
    navigator.locks
      ? navigator.locks.request("orix-web-push", change)
      : change();
  const result = subscriptionChange.then(run, run);
  subscriptionChange = result.catch(() => {});
  return result;
}
export function enableWebPush() {
  const requestedUser = firebaseAuth.currentUser?.uid;
  return serialize(async () => {
    const user = firebaseAuth.currentUser;
    if (!user || user.uid !== requestedUser)
      throw new PublicError("Sign in again to enable notifications.");
    if (!key)
      throw new PublicError("Browser notifications are not configured yet.");
    if (!navigator.locks)
      throw new PublicError(
        "Browser notifications are unavailable in this browser.",
      );
    if (!(await isSupported()))
      throw new PublicError("Notifications are unavailable in this browser.");
    const permission =
      Notification.permission === "granted"
        ? "granted"
        : await Notification.requestPermission();
    if (permission !== "granted")
      throw new PublicError(
        "Notifications are off. You can change this in browser settings.",
      );
    const worker = await navigator.serviceWorker.register(
      "/orix-push-worker.js",
      { scope: "/" },
    );
    await navigator.serviceWorker.ready;
    const messaging = getMessaging(getApps()[0]);
    const token = await getToken(messaging, {
      vapidKey: key,
      serviceWorkerRegistration: worker,
    });
    if (!token)
      throw new PublicError("Notifications could not be enabled. Try again.");
    if (firebaseAuth.currentUser?.uid !== user.uid)
      throw new PublicError("Your account changed. Try again.");
    // Record ownership before the network request: its outcome can be uncertain
    // if authentication changes while the server is registering the token.
    localStorage.setItem(storageKey, user.uid);
    await apiRequest("/users/me/web-push-token", {
      method: "POST",
      body: JSON.stringify({ token }),
    });
    if (firebaseAuth.currentUser?.uid !== user.uid)
      throw new PublicError("Your account changed. Try again.");
    return messaging;
  });
}
export function disableWebPush() {
  const requestedUser = firebaseAuth.currentUser?.uid;
  return serialize(async () => {
    const user = firebaseAuth.currentUser;
    if (!user || user.uid !== requestedUser)
      throw new PublicError("Your account changed. Try again.");
    if (localStorage.getItem(storageKey) !== user.uid) return;
    if (!(await isSupported()))
      throw new PublicError(
        "Browser notifications could not be updated. Try again.",
      );
    const messaging = getMessaging(getApps()[0]);
    // Firebase cannot retrieve a token after browser permission is revoked.
    // Revoke the local subscription instead; the server removes the stale
    // registration on a failed delivery or when it expires.
    if (Notification.permission !== "granted") {
      await deleteToken(messaging);
      localStorage.removeItem(storageKey);
      return;
    }
    const registration = await navigator.serviceWorker.getRegistration("/");
    if (!registration || !key)
      throw new PublicError("Could not update notifications. Try again.");
    const token = await getToken(messaging, {
      vapidKey: key,
      serviceWorkerRegistration: registration,
    });
    if (firebaseAuth.currentUser?.uid !== user.uid)
      throw new PublicError("Your account changed. Try again.");
    if (token)
      await apiRequest("/users/me/web-push-token", {
        method: "DELETE",
        body: JSON.stringify({ token }),
      });
    await deleteToken(messaging);
    localStorage.removeItem(storageKey);
  });
}
export async function listenForWebPush() {
  if (!(await isSupported()) || !firebaseAuth.currentUser) return () => {};
  const owner = firebaseAuth.currentUser.uid;
  return onMessage(getMessaging(getApps()[0]), (message) => {
    if (firebaseAuth.currentUser?.uid !== owner) return;
    const data = message.data;
    if (
      data?.type === "engagement_reminder" &&
      typeof data.title === "string" &&
      typeof data.body === "string"
    ) {
      toast(data.title, {
        description: data.body,
        action: {
          label: "Open ORIX",
          onClick: () => window.location.assign("/app"),
        },
      });
    }
    window.dispatchEvent(
      new CustomEvent(refreshEvent, {
        detail: { type: "notification_update" },
      }),
    );
  });
}

/** Re-register only after explicit opt-in by the current account. */
export async function restoreWebPush() {
  const user = firebaseAuth.currentUser;
  if (
    !user ||
    localStorage.getItem(storageKey) !== user.uid ||
    !("Notification" in window) ||
    Notification.permission !== "granted"
  )
    return;
  await enableWebPush();
}

/** Retire a local browser subscription when auth changes outside the sign-out flow. */
export function reconcileWebPushOwner(
  uid: string | null,
  retireForUser?: string,
) {
  return serialize(async () => {
    if ((firebaseAuth.currentUser?.uid ?? null) !== (retireForUser ?? uid))
      return;
    const previous = localStorage.getItem(storageKey);
    if (!previous || previous === uid) return;
    if (await isSupported()) {
      await deleteToken(getMessaging(getApps()[0]));
    }
    localStorage.removeItem(storageKey);
  });
}
