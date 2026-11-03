import { PublicError, statusMessage } from "./errors";
import { fetchWithRetry } from "./retry";
import { firebaseAuth } from "./session";

async function requestAdmin<T>(
  action: string,
  method: string,
  body?: unknown,
  query?: Record<string, string>,
): Promise<{ data: T; nextCursor: string | null }> {
  const user = firebaseAuth.currentUser;
  if (!user) throw new Error("Sign in with an administrator account.");
  const suffix = query ? `?${new URLSearchParams(query)}` : "";
  const response = await fetchWithRetry(`/api/admin/${action}${suffix}`, {
    method,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${await user.getIdToken()}`,
    },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: AbortSignal.timeout(35000),
    credentials: "omit",
    redirect: "error",
  });
  if (firebaseAuth.currentUser?.uid !== user.uid)
    throw new Error("The signed-in account changed.");
  const data = await response.json().catch(() => null);
  if (!response.ok) throw new PublicError(statusMessage(response.status));
  return { data: data as T, nextCursor: response.headers.get("X-Next-Cursor") };
}

export async function adminRequest<T>(
  action: string,
  method: string,
  body?: unknown,
  query?: Record<string, string>,
): Promise<T> {
  return (await requestAdmin<T>(action, method, body, query)).data;
}
export async function adminRequestPage<T>(
  action: string,
  method: string,
  body?: unknown,
  query?: Record<string, string>,
) {
  return requestAdmin<T>(action, method, body, query);
}
