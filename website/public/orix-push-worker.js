self.addEventListener("install", () => self.skipWaiting());
self.addEventListener("activate", (event) => event.waitUntil(self.clients.claim()));

/* Data-only FCM messages carry no private content. Notification tags replace
   duplicate deliveries for the same durable inbox item. */
self.addEventListener("push", (event) => {
  let payload;
  try {
    payload = event.data?.json();
  } catch {
    return;
  }
  const data = payload?.data;
  if (
    !["inbox_update", "engagement_reminder"].includes(data?.type) ||
    !/^[a-f0-9]{16,128}$/i.test(data.notification_id || "")
  )
    return;
  const engagement = data.type === "engagement_reminder";
  if (
    engagement &&
    (typeof data.title !== "string" ||
      typeof data.body !== "string" ||
      data.title.length > 160 ||
      data.body.length > 480)
  )
    return;
  event.waitUntil(
    self.registration.showNotification(engagement ? data.title : "ORIX", {
      body: engagement
        ? data.body
        : "You have a new update. Open ORIX to see it.",
      icon: "/favicon.svg",
      badge: "/favicon.svg",
      tag: `orix-${data.notification_id}`,
      data: { path: engagement ? "/app" : "/app/notifications" },
      silent: engagement,
    }),
  );
});
self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  event.waitUntil(
    (async () => {
      const path =
        event.notification.data?.path === "/app"
          ? "/app"
          : "/app/notifications";
      const url = new URL(path, self.location.origin).href;
      const windows = await self.clients.matchAll({
        type: "window",
        includeUncontrolled: true,
      });
      const existing = windows.find(
        (client) => new URL(client.url).origin === self.location.origin,
      );
      if (existing) {
        await existing.navigate(url);
        return existing.focus();
      }
      return self.clients.openWindow(url);
    })(),
  );
});
