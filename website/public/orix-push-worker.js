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
    data?.type !== "inbox_update" ||
    !/^[a-f0-9]{16,128}$/i.test(data.notification_id || "")
  )
    return;
  event.waitUntil(
    self.registration.showNotification("ORIX", {
      body: "You have a new update. Open ORIX to see it.",
      icon: "/favicon.svg",
      badge: "/favicon.svg",
      tag: `orix-${data.notification_id}`,
      data: { path: "/app/notifications" },
    }),
  );
});
self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  event.waitUntil(
    (async () => {
      const url = new URL("/app/notifications", self.location.origin).href;
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
