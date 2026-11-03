import { listenForWebPush, restoreWebPush } from "./lib/webPush";
import { useResource } from "./lib/resource";
import type { InboxPage } from "./pages/Inbox";
import { ConnectionStatus } from "./components/ConnectionStatus";
import { RealtimeBridge } from "./lib/realtime";
import { AndroidDownload } from "./components/AndroidDownload";
import { Component, lazy, Suspense, useEffect } from "react";
import type { ErrorInfo, ReactNode } from "react";
import { BrowserRouter, Route, Routes, useLocation } from "react-router-dom";
import { LazyMotion, domAnimation, MotionConfig } from "motion/react";
import { Toaster, toast } from "sonner";
import { useStore } from "./lib/store";
import { initializeSession, useSession } from "./lib/session";
import { Button } from "./components/ui";
import { PixelMark } from "./components/PixelArt";
import { PixelStatus, LifecycleFeedback } from "./components/PixelStatus";
const Groups = lazy(() =>
  import("./pages/Groups").then((module) => ({ default: module.Groups })),
);
const GroupDetail = lazy(() =>
  import("./pages/Groups").then((module) => ({ default: module.GroupDetail })),
);
const ReportConversation = lazy(() => import("./pages/ReportConversation"));
const Landing = lazy(() => import("./pages/Landing"));
const Layout = lazy(() => import("./components/Layout"));
const Login = lazy(() =>
  import("./pages/Auth").then((m) => ({ default: m.Login })),
);
const Onboarding = lazy(() =>
  import("./pages/Auth").then((m) => ({ default: m.Onboarding })),
);
const Dashboard = lazy(() =>
  import("./pages/Journeys").then((m) => ({ default: m.Dashboard })),
);
const Explore = lazy(() =>
  import("./pages/Journeys").then((m) => ({ default: m.Explore })),
);
const CreateRide = lazy(() =>
  import("./pages/Journeys").then((m) => ({ default: m.CreateRide })),
);
const MyRides = lazy(() =>
  import("./pages/Journeys").then((m) => ({ default: m.MyRides })),
);
const RideDetail = lazy(() =>
  import("./pages/Journeys").then((m) => ({ default: m.RideDetail })),
);
const Messages = lazy(() =>
  import("./pages/Community").then((m) => ({ default: m.Messages })),
);
const Profile = lazy(() =>
  import("./pages/Community").then((m) => ({ default: m.Profile })),
);
const Notifications = lazy(() =>
  import("./pages/Community").then((m) => ({ default: m.Notifications })),
);
const Settings = lazy(() =>
  import("./pages/Community").then((m) => ({ default: m.Settings })),
);
const Safety = lazy(() =>
  import("./pages/Community").then((m) => ({ default: m.Safety })),
);
const Support = lazy(() =>
  import("./pages/Community").then((m) => ({ default: m.Support })),
);
const SupportThread = lazy(() =>
  import("./pages/Community").then((m) => ({ default: m.SupportThread })),
);
const RiderSetup = lazy(() => import("./pages/LiftRegistration"));
const AdminOverview = lazy(() =>
  import("./pages/Admin").then((m) => ({ default: m.AdminOverview })),
);
const AdminPeople = lazy(() =>
  import("./pages/Admin").then((m) => ({ default: m.AdminPeople })),
);
const AdminReports = lazy(() =>
  import("./pages/Admin").then((m) => ({ default: m.AdminReports })),
);
const AdminRides = lazy(() =>
  import("./pages/Admin").then((m) => ({ default: m.AdminRides })),
);
const Operations = lazy(() =>
  import("./pages/Admin").then((m) => ({ default: m.Operations })),
);
const Audit = lazy(() =>
  import("./pages/Admin").then((m) => ({ default: m.Audit })),
);
const PublicInfo = lazy(() =>
  import("./pages/Public").then((m) => ({ default: m.PublicInfo })),
);
const StatusPage = lazy(() =>
  import("./pages/Public").then((m) => ({ default: m.StatusPage })),
);
function Loading() {
  return (
    <div className="loading-screen" role="status">
      <PixelMark className="loading-brand" />
      <PixelStatus state="loading" label="Loading" />
    </div>
  );
}
// Tab titles follow local unread state; the favicon always retains the ORIX brand mark.
function PageIdentity() {
  const location = useLocation();
  const { data: inbox } = useResource<InboxPage>(
    "/notifications?limit=1&unread_only=true",
  );
  const unread = inbox?.unread_count ?? 0;
  useEffect(() => {
    const names: Record<string, string> = {
      "/": "Go together.",
      "/login": "A little hello.",
      "/app": "Your space",
      "/app/create": "Plan a journey",
      "/app/messages": "Your conversations",
      "/admin": "Control room",
    };
    const title = `${names[location.pathname] || location.pathname.split("/").at(-1)?.replaceAll("-", " ") || "Your journey"} · ORIX`;
    const onVisibility = () => {
      document.title =
        document.hidden && unread && location.pathname.startsWith("/app")
          ? `${unread} little updates · ORIX`
          : title;
    };
    onVisibility();
    document.addEventListener("visibilitychange", onVisibility);
    const favicon = document.querySelector<HTMLLinkElement>('link[rel="icon"]');
    // Keep the supplied brand mark stable; route/unread feedback belongs in the title.
    if (favicon) favicon.href = "/favicon.svg?v=orix-mark";
    window.scrollTo({ top: 0, behavior: "instant" });
    return () => {
      document.removeEventListener("visibilitychange", onVisibility);
      if (favicon) favicon.href = "/favicon.svg?v=orix-mark";
    };
  }, [location.pathname, unread]);
  return null;
}
class ErrorBoundary extends Component<
  { children: ReactNode },
  { error: boolean }
> {
  state = { error: false };
  static getDerivedStateFromError() {
    return { error: true };
  }
  componentDidCatch(_error: Error, _info: ErrorInfo) {
    /* Render a safe recovery state without logging potential user content. */
  }
  render() {
    if (this.state.error)
      return (
        <main className="recovery-page" aria-labelledby="recovery-title">
          <div className="recovery-content">
            <PixelMark className="recovery-brand" />
            <PixelStatus state="error" />
            <h1 id="recovery-title">A little bump in the road.</h1>
            <p>This page couldn’t open. Reload it to get back on your way.</p>
            <Button onClick={() => window.location.reload()}>
              Reload page
            </Button>
          </div>
        </main>
      );
    return this.props.children;
  }
}
function WebPushLifecycle() {
  const account = useSession((s) => s.account);
  useEffect(() => {
    if (!account?.onboarding_completed) return;
    let alive = true;
    let stop: undefined | (() => void);
    void listenForWebPush()
      .then((unsubscribe) => {
        if (alive) stop = unsubscribe;
        else unsubscribe();
      })
      .catch(() => {});
    const restore = () => {
      void restoreWebPush().catch(() => {
        if (
          !alive ||
          useSession.getState().account?.user_id !== account.user_id
        )
          return;
        toast.error("Browser notifications couldn't reconnect.", {
          action: { label: "Retry", onClick: restore },
        });
      });
    };
    restore();
    return () => {
      alive = false;
      stop?.();
    };
  }, [account?.user_id, account?.onboarding_completed]);
  return null;
}
export default function App() {
  useEffect(() => initializeSession(), []);
  const motion = useStore((s) => s.motion);
  useEffect(() => {
    document.documentElement.dataset.motion = motion ? "on" : "off";
  }, [motion]);
  return (
    <ErrorBoundary>
      <AndroidDownload />
      <ConnectionStatus />
      <RealtimeBridge />
      <WebPushLifecycle />
      <LazyMotion features={domAnimation}>
        <MotionConfig
          reducedMotion={motion ? "user" : "always"}
          transition={{ duration: motion ? 0.25 : 0 }}
        >
          <BrowserRouter>
            <PageIdentity />
            <Suspense fallback={<Loading />}>
              <Routes>
                <Route path="/" element={<Landing />} />
                <Route path="/login" element={<Login />} />
                <Route path="/onboarding" element={<Onboarding />} />
                {["about", "privacy", "terms", "account-deletion"].map(
                  (page) => (
                    <Route
                      key={page}
                      path={`/${page}`}
                      element={<PublicInfo page={page} />}
                    />
                  ),
                )}
                <Route path="/status/:kind" element={<StatusPage />} />
                <Route path="/app" element={<Layout />}>
                  <Route index element={<Dashboard />} />
                  <Route path="explore" element={<Explore />} />
                  <Route path="create" element={<CreateRide />} />
                  <Route path="rides" element={<MyRides />} />
                  <Route path="rides/:id" element={<RideDetail />} />
                  <Route path="history" element={<MyRides history />} />
                  <Route path="messages" element={<Messages />} />
                  <Route path="notifications" element={<Notifications />} />
                  <Route path="profile" element={<Profile />} />
                  <Route path="settings" element={<Settings />} />
                  <Route path="safety" element={<Safety />} />
                  <Route path="reports/:id" element={<ReportConversation />} />
                  <Route path="support" element={<Support />} />
                  <Route path="support/:id" element={<SupportThread />} />
                  <Route path="rider" element={<RiderSetup />} />
                  <Route path="groups" element={<Groups />} />
                  <Route path="groups/:id" element={<GroupDetail />} />
                  <Route path="onboarding" element={<Onboarding embedded />} />
                </Route>
                <Route path="/admin" element={<Layout admin />}>
                  <Route index element={<AdminOverview />} />
                  <Route path="users" element={<AdminPeople />} />
                  <Route path="rides" element={<AdminRides />} />
                  <Route path="reports" element={<AdminReports />} />
                  <Route
                    path="reports/:id"
                    element={<ReportConversation admin />}
                  />
                  <Route path="support" element={<Support admin />} />
                  <Route path="support/:id" element={<SupportThread admin />} />
                  <Route path="operations" element={<Operations />} />
                  <Route path="audit" element={<Audit />} />
                </Route>
                <Route path="*" element={<StatusPage />} />
              </Routes>
            </Suspense>
            <LifecycleFeedback />
            <Toaster
              position="bottom-right"
              toastOptions={{ className: "orix-toast" }}
            />
          </BrowserRouter>
        </MotionConfig>
      </LazyMotion>
    </ErrorBoundary>
  );
}
