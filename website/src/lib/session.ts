import { PublicError, statusMessage } from "./errors";
import { friendlyError } from "./errors";
import { fetchWithRetry } from "./retry";
import { initializeApp, getApps } from "firebase/app";
import {
  getAuth,
  GoogleAuthProvider,
  onAuthStateChanged,
  signInWithPopup,
  signInWithRedirect,
  signOut as firebaseSignOut,
  getRedirectResult,
} from "firebase/auth";
import { create } from "zustand";

const env = import.meta.env;
const app =
  getApps()[0] ??
  initializeApp({
    apiKey: env.VITE_FIREBASE_API_KEY,
    authDomain: env.VITE_FIREBASE_AUTH_DOMAIN,
    projectId: env.VITE_FIREBASE_PROJECT_ID,
    storageBucket: env.VITE_FIREBASE_STORAGE_BUCKET,
    messagingSenderId: env.VITE_FIREBASE_MESSAGING_SENDER_ID,
    appId: env.VITE_FIREBASE_APP_ID,
  });
export const firebaseAuth = getAuth(app);
export interface Account {
  user_id: string;
  email: string;
  display_name: string;
  photo_url?: string;
  onboarding_completed: boolean;
  role: string;
  occupation?: "student" | "teacher" | "other";
}
export const useSession = create<{
  account: Account | null;
  loading: boolean;
  error: string | null;
}>(() => ({ account: null, loading: true, error: null }));

export function apiOrigin(): string {
  if (!env.VITE_API_BASE_URL)
    throw new PublicError(
      "Orix can’t connect right now. Please contact support.",
    );
  let url: URL;
  try {
    url = new URL(env.VITE_API_BASE_URL);
  } catch {
    throw new PublicError(
      "Orix can’t connect right now. Please contact support.",
    );
  }
  if (
    url.username ||
    url.password ||
    url.search ||
    url.hash ||
    url.pathname !== "/" ||
    (url.protocol !== "https:" &&
      !["localhost", "127.0.0.1"].includes(url.hostname))
  ) {
    throw new PublicError(
      "Orix can’t connect right now. Please contact support.",
    );
  }
  return url.origin;
}
async function requestAPI<T>(
  path: string,
  init: RequestInit = {},
): Promise<{ data: T; nextCursor: string | null }> {
  if (!path.startsWith("/") || path.startsWith("//"))
    throw new PublicError(
      "That action isn’t available. Please refresh and try again.",
    );
  const user = firebaseAuth.currentUser;
  if (!user) throw new PublicError("Sign in with Google to continue.");
  const headers = new Headers(init.headers);
  headers.set("Authorization", `Bearer ${await user.getIdToken()}`);
  if (init.body && !(init.body instanceof FormData))
    headers.set("Content-Type", "application/json");
  const response = await fetchWithRetry(`${apiOrigin()}/api/v1${path}`, {
    ...init,
    headers,
    credentials: "omit",
    redirect: "error",
    signal: init.signal
      ? AbortSignal.any([init.signal, AbortSignal.timeout(30000)])
      : AbortSignal.timeout(30000),
  });
  if (firebaseAuth.currentUser?.uid !== user.uid)
    throw new PublicError("The signed-in account changed.");
  const data = await response.json().catch(() => null);
  if (!response.ok) throw new PublicError(statusMessage(response.status));
  const method = (init.method ?? "GET").toUpperCase();
  let outcome: string | undefined;
  if (method === "POST" && path === "/rides")
    outcome = data?.driver ? "lift-offered" : "searching";
  if (method === "DELETE" && /^\/rides\/[^/]+$/.test(path))
    outcome = "cancelled";
  if (method === "POST" && path === "/reports") outcome = "report-sent";
  if (method === "POST" && /^\/groups\/[^/]+\/reject$/.test(path))
    outcome = "searching";
  if (method === "POST" && /^\/groups\/[^/]+\/start$/.test(path))
    outcome = "travelling";
  if (method === "POST" && /^\/groups\/[^/]+\/finish$/.test(path))
    outcome = "completed";
  if (outcome)
    window.dispatchEvent(
      new CustomEvent("orix:outcome", { detail: { state: outcome } }),
    );
  return { data: data as T, nextCursor: response.headers.get("X-Next-Cursor") };
}
export async function apiRequest<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  return (await requestAPI<T>(path, init)).data;
}
export async function apiRequestPage<T>(path: string, init: RequestInit = {}) {
  return requestAPI<T>(path, init);
}
export async function verifyAccount(): Promise<Account> {
  const user = firebaseAuth.currentUser;
  if (!user) throw new PublicError("Sign in to continue.");
  const response = await fetch(`${apiOrigin()}/api/v1/auth/verify`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id_token: await user.getIdToken() }),
    credentials: "omit",
    redirect: "error",
    signal: AbortSignal.timeout(20000),
  });
  const data = await response.json().catch(() => null);
  if (firebaseAuth.currentUser?.uid !== user.uid)
    throw new PublicError("The signed-in account changed.");
  if (!response.ok) {
    const message = statusMessage(response.status);
    if (response.status === 401 || response.status === 403)
      useSession.setState({ account: null, loading: false, error: message });
    throw new PublicError(message);
  }
  useSession.setState({
    account: data as Account,
    loading: false,
    error: null,
  });
  return data as Account;
}
export function initializeSession() {
  let revision = 0;
  void getRedirectResult(firebaseAuth).catch(() => {
    useSession.setState({
      error: "Google sign-in did not finish. Please try again.",
    });
  });
  const stop = onAuthStateChanged(firebaseAuth, async (user) => {
    const current = ++revision;
    useSession.setState({ account: null, loading: !!user, error: null });
    try {
      const { reconcileWebPushOwner } = await import("./webPush");
      await reconcileWebPushOwner(user?.uid ?? null);
      if (current !== revision) return;
      if (!user) return;
      await verifyAccount();
    } catch (error) {
      if (current === revision)
        useSession.setState({
          account: null,
          loading: false,
          error: friendlyError(error),
        });
    }
  });
  return () => {
    revision++;
    stop();
  };
}
export async function signInGoogle() {
  const provider = new GoogleAuthProvider();
  provider.setCustomParameters({ prompt: "select_account" });
  try {
    await signInWithPopup(firebaseAuth, provider);
  } catch (error) {
    const code = (error as { code?: string }).code;
    if (code === "auth/popup-blocked") {
      await signInWithRedirect(firebaseAuth, provider);
      return null;
    }
    if (code === "auth/popup-closed-by-user")
      throw new PublicError("Sign-in was cancelled.");
    if (code === "auth/unauthorized-domain")
      throw new PublicError(
        "This website must be authorized in Firebase Authentication before sign-in.",
      );
    throw new PublicError("Google sign-in could not finish. Please try again.");
  }
  return verifyAccount();
}
export async function signOutAccount(
  options: { accountDeleted?: boolean } = {},
) {
  const user = firebaseAuth.currentUser;
  if (!user) return;
  if (!options.accountDeleted) {
    const { disableWebPush } = await import("./webPush");
    await disableWebPush();
  } else {
    const { reconcileWebPushOwner } = await import("./webPush");
    await reconcileWebPushOwner(null, user.uid);
  }
  if (firebaseAuth.currentUser?.uid !== user.uid)
    throw new PublicError("Your account changed. Please try again.");
  await firebaseSignOut(firebaseAuth);
  useSession.setState({ account: null, loading: false, error: null });
}
