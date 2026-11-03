import type { IncomingMessage, ServerResponse } from "node:http";

// Server-only BFF. Administrator secrets are never sent to the browser.
export default async function handler(
  req: IncomingMessage,
  res: ServerResponse,
) {
  const requestURL = new URL(req.url || "/", "http://local");
  const action = requestURL.pathname.split("/").at(-1);
  const reportID = requestURL.searchParams.get("report_id");
  const validReportID =
    reportID !== null && /^[A-Za-z0-9_-]{1,100}$/.test(reportID);
  const ticketID = requestURL.searchParams.get("ticket_id");
  const validTicketID =
    ticketID !== null && /^[A-Za-z0-9_-]{1,100}$/.test(ticketID);
  const operation = requestURL.searchParams.get("operation");
  const validOperation =
    operation !== null &&
    ["message", "close", "reopen", "resolved", "in_progress"].includes(
      operation,
    );
  const userID = requestURL.searchParams.get("user_id");
  const groupID = requestURL.searchParams.get("group_id");
  const eventID = requestURL.searchParams.get("event_id");
  const validID = (value: string | null): value is string =>
    value !== null && /^[A-Za-z0-9_-]{1,100}$/.test(value);
  const routes: Record<string, { method: string; path: string }> = {
    "set-version": { method: "PUT", path: "/api/v1/admin/app/version" },
    people: { method: "GET", path: "/api/v1/admin/people" },
    groups: { method: "GET", path: "/api/v1/admin/groups" },
    stats: { method: "GET", path: "/api/v1/admin/stats" },
    lookup: { method: "GET", path: "/api/v1/admin/users/lookup" },
    logs: { method: "GET", path: "/api/v1/admin/logs" },
    maintenance: { method: "GET", path: "/api/v1/admin/maintenance" },
    engagement: { method: "GET", path: "/api/v1/admin/engagement" },
    "set-engagement": { method: "PUT", path: "/api/v1/admin/engagement" },
    jobs: { method: "GET", path: "/api/v1/admin/jobs" },
    failed: { method: "GET", path: "/api/v1/admin/notifications/failed" },
    "set-maintenance": { method: "POST", path: "/api/v1/admin/maintenance" },
    ...(validID(userID)
      ? {
          avatar: { method: "GET", path: `/api/v1/admin/users/${encodeURIComponent(userID)}/avatar` },
          "user-role": {
            method: "PUT",
            path: `/api/v1/admin/users/${encodeURIComponent(userID)}/role`,
          },
          "restore-account": {
            method: "POST",
            path: `/api/v1/admin/users/${encodeURIComponent(userID)}/restore`,
          },
        }
      : {}),
    ...(validID(groupID)
      ? {
          "group-chat": { method: "GET", path: `/api/v1/admin/groups/${encodeURIComponent(groupID)}/chat` },
          "send-group-chat": { method: "POST", path: `/api/v1/admin/groups/${encodeURIComponent(groupID)}/chat` },
          group: {
            method: "GET",
            path: `/api/v1/admin/groups/${encodeURIComponent(groupID)}`,
          },
          "cancel-group": {
            method: "POST",
            path: `/api/v1/admin/groups/${encodeURIComponent(groupID)}/cancel`,
          },
        }
      : {}),
    ...(validID(eventID)
      ? {
          "retry-delivery": {
            method: "POST",
            path: `/api/v1/admin/notifications/${encodeURIComponent(eventID)}/retry`,
          },
        }
      : {}),
    "public-settings": { method: "PUT", path: "/api/v1/admin/public-settings" },
    "android-upload-ticket": {
      method: "POST",
      path: "/api/v1/admin/android-upload-ticket",
    },
    "support-tickets": { method: "GET", path: "/api/v1/admin/support/tickets" },
    ...(validTicketID
      ? {
          "support-thread": {
            method: "GET",
            path: `/api/v1/admin/support/tickets/${encodeURIComponent(ticketID!)}`,
          },
          ...(validOperation
            ? {
                "support-action": {
                  method: "POST",
                  path: `/api/v1/admin/support/tickets/${encodeURIComponent(ticketID!)}/${operation}`,
                },
              }
            : {}),
        }
      : {}),
    reports: { method: "GET", path: "/api/v1/admin/reports" },
    ...(validReportID
      ? {
          "report-thread": {
            method: "GET",
            path: `/api/v1/admin/reports/${encodeURIComponent(reportID!)}`,
          },
          "report-reply": {
            method: "POST",
            path: `/api/v1/admin/reports/${encodeURIComponent(reportID!)}/messages`,
          },
          "resolve-report": {
            method: "POST",
            path: `/api/v1/admin/reports/${encodeURIComponent(reportID!)}/resolve`,
          },
        }
      : {}),
  };
  const route = routes[action || ""];
  res.setHeader("Cache-Control", "no-store");
  if (!route || req.method !== route.method) {
    res.writeHead(405);
    res.end();
    return;
  }
  const authorization = req.headers.authorization;
  if (!authorization?.startsWith("Bearer ") || authorization.length > 17000) {
    res.writeHead(401);
    res.end();
    return;
  }
  const origin = process.env.ORIX_API_ORIGIN;
  const secret = process.env.ORIX_ADMIN_API_SECRET;
  if (!origin || !secret) {
    res.writeHead(503);
    res.end(JSON.stringify({ detail: "The admin service is not configured." }));
    return;
  }
  try {
    const target = new URL(origin);
    if (
      target.username ||
      target.password ||
      target.pathname !== "/" ||
      target.search ||
      target.hash ||
      (target.protocol !== "https:" &&
        !["localhost", "127.0.0.1"].includes(target.hostname))
    )
      throw new Error("Invalid service origin");
    let body = "";
    for await (const chunk of req) {
      body += chunk.toString();
      if (Buffer.byteLength(body) > 16384) {
        res.writeHead(413);
        res.end();
        return;
      }
    }
    let path = route.path;
    if (
      action === "people" || action === "groups" || action === "group-chat" ||
      action === "reports" ||
      action === "support-tickets" ||
      action === "logs"
    ) {
      const before = requestURL.searchParams.get("before");
      if (before && before.length > 4096) {
        res.writeHead(400);
        res.end();
        return;
      }
      path += `?${new URLSearchParams({ limit: "100", ...(before ? { before } : {}) })}`;
    }
    if (action === "lookup") {
      const email = requestURL.searchParams.get("email");
      if (email && email.length > 320) {
        res.writeHead(400);
        res.end();
        return;
      }
      path += `?${new URLSearchParams(email ? { email } : userID ? { user_id: userID } : {})}`;
    }
    const response = await fetch(`${target.origin}${path}`, {
      method: route.method,
      headers: {
        "Content-Type": "application/json",
        Authorization: authorization,
        "X-Admin-Secret": secret,
      },
      body: route.method === "GET" ? undefined : body,
      redirect: "error",
      signal: AbortSignal.timeout(30000),
    });
    res.setHeader("Content-Type", action === "avatar" && response.ok ? (response.headers.get("content-type") || "application/octet-stream") : "application/json");
    const nextCursor = response.headers.get("X-Next-Cursor");
    if (nextCursor) res.setHeader("X-Next-Cursor", nextCursor);
    res.writeHead(response.status);
    if (action === "avatar" && response.ok) res.end(Buffer.from(await response.arrayBuffer()));
    else res.end(await response.text());
  } catch {
    res.writeHead(502);
    res.end(
      JSON.stringify({
        detail: "The admin service is unavailable. Try again.",
      }),
    );
  }
}
