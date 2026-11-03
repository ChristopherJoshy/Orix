import { friendlyError } from "../lib/errors";
import { useCallback, useEffect, useRef, useState } from "react";
import { Link, Navigate, useSearchParams } from "react-router-dom";
import { apiRequest, apiRequestPage, useSession } from "../lib/session";
import { useResource } from "../lib/resource";
import { refreshJourneys } from "../lib/journeys";
import { Badge, Button, Empty, PageHead } from "../components/ui";
import { PixelMark } from "../components/PixelArt";
import { PixelStatus } from "../components/PixelStatus";
import { Groups } from "./Groups";
import { Modal } from "../components/Modal";
interface Notice {
  notification_id: string;
  title: string;
  body: string;
  type: string;
  read: boolean;
  created_at: string;
  data?: Record<string, string>;
}
export interface InboxPage {
  notifications: Notice[];
  unread_count: number;
}
function noticeLink(notice: Notice) {
  const data = notice.data ?? {};
  if (notice.type === "group_rejected") return "/app/explore";
  if (data.report_id)
    return `/app/reports/${encodeURIComponent(data.report_id)}`;
  if (data.ticket_id)
    return `/app/support/${encodeURIComponent(data.ticket_id)}`;
  if (data.group_id) return `/app/groups/${encodeURIComponent(data.group_id)}`;
  return null;
}
export function Messages() {
  const [params] = useSearchParams();
  const group = params.get("group");
  return group ? (
    <Navigate to={`/app/groups/${encodeURIComponent(group)}`} replace />
  ) : (
    <Groups />
  );
}
export function Notifications() {
  const owner = useSession((s) => s.account?.user_id);
  const pageCount = useRef(1);
  const [loadingMore, setLoadingMore] = useState(false);
  const readPages = useCallback(
    async (
      _path: string,
    ): Promise<InboxPage & { nextCursor: string | null }> => {
      if (!owner) throw new Error("Account unavailable");
      let cursor: string | null = null;
      let unread = 0;
      const rows = new Map<string, Notice>();
      for (let page = 0; page < pageCount.current; page++) {
        if (owner !== useSession.getState().account?.user_id)
          throw new Error("Account changed");
        const result = await apiRequestPage<InboxPage>(
          `/notifications?limit=50${cursor ? `&before=${encodeURIComponent(cursor)}` : ""}`,
        );
        if (owner !== useSession.getState().account?.user_id)
          throw new Error("Account changed");
        if (page === 0) unread = result.data.unread_count;
        for (const item of result.data.notifications)
          rows.set(item.notification_id, item);
        cursor = result.nextCursor;
        if (!cursor) break;
      }
      return {
        notifications: [...rows.values()],
        unread_count: unread,
        nextCursor: cursor,
      };
    },
    [owner],
  );
  useEffect(() => {
    pageCount.current = 1;
    setLoadingMore(false);
  }, [owner]);
  const { data, error, loading, load } = useResource<
    InboxPage & { nextCursor: string | null }
  >("/notifications", 30000, readPages);
  useEffect(() => {
    if (!loading) setLoadingMore(false);
  }, [loading]);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState("");
  const [clear, setClear] = useState(false);
  const sending = useRef(false);
  async function change(path: string, method: string) {
    if (sending.current) return;
    sending.current = true;
    setBusy(true);
    setFailure("");
    try {
      await apiRequest(path, { method });
      if (owner === useSession.getState().account?.user_id) {
        setClear(false);
        refreshJourneys();
        await load();
      }
    } catch (problem) {
      if (owner === useSession.getState().account?.user_id)
        setFailure(friendlyError(problem));
    } finally {
      sending.current = false;
      if (owner === useSession.getState().account?.user_id) setBusy(false);
    }
  }
  return (
    <>
      <PageHead
        title="Your updates."
        subtitle={
          data
            ? `${data.unread_count} unread`
            : "People, plans and pickup details"
        }
      >
        <Button
          variant="secondary"
          disabled={busy || !data?.unread_count}
          onClick={() => void change("/notifications/read-all", "PUT")}
        >
          Mark all read
        </Button>
      </PageHead>
      {(error || failure) && (
        <p role="alert">
          {failure || error}{" "}
          <Button variant="secondary" onClick={() => void load()}>
            Refresh
          </Button>
        </p>
      )}
      {!data && !error ? (
        <PixelStatus state="loading" label="Opening updates" />
      ) : data?.notifications.length === 0 ? (
        <Empty
          title="You’re all caught up."
          text="Journey updates and replies will appear here."
        />
      ) : (
        data?.notifications.map((notice) => (
          <article className="panel" key={notice.notification_id}>
            <div className="section-title">
              <PixelMark className="notification-brand" />
              {!notice.read && <Badge>Unread</Badge>}
              <time className="muted">
                {new Date(notice.created_at).toLocaleString()}
              </time>
            </div>
            <h2>{notice.title}</h2>
            <p>{notice.body}</p>
            <div className="intro-actions">
              {noticeLink(notice) && (
                <Link className="button secondary" to={noticeLink(notice)!}>
                  Open update
                </Link>
              )}
              {!notice.read && (
                <Button
                  disabled={busy}
                  variant="secondary"
                  onClick={() =>
                    void change(
                      `/notifications/${encodeURIComponent(notice.notification_id)}/read`,
                      "PUT",
                    )
                  }
                >
                  Mark read
                </Button>
              )}
              <Button
                disabled={busy}
                variant="ghost"
                onClick={() =>
                  void change(
                    `/notifications/${encodeURIComponent(notice.notification_id)}`,
                    "DELETE",
                  )
                }
              >
                Delete
              </Button>
            </div>
          </article>
        ))
      )}
      {data?.nextCursor && (
        <Button
          disabled={busy || loading || loadingMore}
          onClick={() => {
            pageCount.current++;
            setLoadingMore(true);
            void load();
          }}
        >
          {loadingMore ? "Loading…" : "Load older updates"}
        </Button>
      )}
      {data && data.notifications.length > 0 && (
        <Button
          variant="secondary"
          disabled={busy}
          onClick={() => setClear(true)}
        >
          Clear notifications
        </Button>
      )}
      <Modal
        title="Clear your notifications?"
        description="This removes your saved updates. Your journeys and conversations remain available."
        open={clear}
        onOpenChange={(open) => {
          if (!busy) setClear(open);
        }}
      >
        <Button
          disabled={busy}
          onClick={() => void change("/notifications", "DELETE")}
        >
          Clear notifications
        </Button>
        <Button
          variant="secondary"
          disabled={busy}
          onClick={() => setClear(false)}
        >
          Keep updates
        </Button>
      </Modal>
    </>
  );
}
