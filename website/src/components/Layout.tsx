import { toast } from "sonner";
import { useResource } from "../lib/resource";
import type { InboxPage } from "../pages/Inbox";
import { useSession, signOutAccount } from "../lib/session";
import { NavLink, Link, Navigate, Outlet, useLocation } from "react-router-dom";
import { useEffect, useState, useRef } from "react";
import { m } from "motion/react";
import {
  Home,
  Compass,
  Navigation,
  Route,
  MessageSquare,
  Bell,
  User,
  Shield,
  LifeBuoy,
  Settings,
  LayoutDashboard,
  Users,
  Flag,
  Activity,
  ScrollText,
  Plus,
  ArrowUpRight,
  Menu,
  X,
  LogOut,
} from "lucide-react";
import { useStore } from "../lib/store";
import { PixelMark } from "./PixelArt";
import { ContactPhoto } from "./EmailPersonPicker";
import { CurrentCity } from "./CurrentCity";
import { Logo } from "./ui";
const userNav = [
  { to: "/app", label: "Your space", icon: Home, end: true },
  { to: "/app/explore", label: "Your pool", icon: Compass },
  { to: "/app/rides", label: "Your journeys", icon: Route },
  { to: "/app/rider", label: "Offer a lift", icon: Navigation },
  { to: "/app/groups", label: "Your groups", icon: Route },
  { to: "/app/messages", label: "Messages", icon: MessageSquare },
  { to: "/app/notifications", label: "Notifications", icon: Bell },
];
const adminNav = [
  { to: "/admin", label: "Overview", icon: LayoutDashboard, end: true },
  { to: "/admin/users", label: "People", icon: Users },
  { to: "/admin/rides", label: "Journeys", icon: Route },
  { to: "/admin/reports", label: "Reports", icon: Flag },
  { to: "/admin/support", label: "Support desk", icon: LifeBuoy },
  { to: "/admin/operations", label: "Operations", icon: Activity },
  { to: "/admin/audit", label: "Activity log", icon: ScrollText },
];
export default function Layout({ admin = false }: { admin?: boolean }) {
  const motion = useStore((s) => s.motion);
  const { account, loading } = useSession();
  const { data: inbox } = useResource<InboxPage>(
    "/notifications?limit=1&unread_only=true",
  );
  const profile = {
    userId: account?.user_id ?? "",
    name: account?.display_name ?? "",
    email: account?.email ?? "",
    photoUrl: account?.photo_url,
  };
  const signOut = () => {
    void signOutAccount().catch(() =>
      toast.error("Could not sign out. Check your connection and try again."),
    );
  };
  const location = useLocation();
  const [menu, setMenu] = useState(false);
  const sidebarRef = useRef<HTMLElement>(null);
  useEffect(() => {
    const viewport = window.matchMedia("(max-width: 700px)");
    const resize = () => {
      if (!viewport.matches) setMenu(false);
    };
    viewport.addEventListener("change", resize);
    return () => viewport.removeEventListener("change", resize);
  }, []);
  useEffect(() => {
    if (!menu) return;
    // Keep keyboard focus inside the mobile drawer and restore its trigger on dismissal.
    const previous = document.activeElement as HTMLElement | null;
    const sidebar = sidebarRef.current;
    const focusable = () =>
      Array.from(
        sidebar?.querySelectorAll<HTMLElement>("a,button") || [],
      ).filter((el) => el.getClientRects().length);
    sidebar
      ?.querySelector<HTMLElement>('[aria-label="Close navigation"]')
      ?.focus();
    const keydown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setMenu(false);
        return;
      }
      if (event.key !== "Tab") return;
      const elements = focusable(),
        first = elements[0],
        last = elements.at(-1);
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last?.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first?.focus();
      }
    };
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    document.addEventListener("keydown", keydown);
    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener("keydown", keydown);
      previous?.focus();
    };
  }, [menu]);
  useEffect(() => setMenu(false), [location.pathname]);
  if (loading)
    return (
      <main className="loading-screen" aria-label="Verifying account">
        <Logo />
      </main>
    );
  if (!account)
    return (
      <Navigate
        to={`/login?next=${encodeURIComponent(location.pathname)}`}
        replace
      />
    );
  if (!account.onboarding_completed)
    return <Navigate to="/onboarding" replace />;
  if (admin && account.role !== "admin") return <Navigate to="/app" replace />;
  const unread = inbox?.unread_count ?? 0;
  return (
    <div className={`workspace ${admin ? "admin-workspace" : ""}`}>
      <a className="skip-link" href="#main">
        Skip to content
      </a>
      <aside
        ref={sidebarRef}
        role={menu ? "dialog" : undefined}
        aria-modal={menu || undefined}
        aria-label="Workspace sidebar"
        className={`sidebar ${menu ? "open" : ""}`}
      >
        <div className="sidebar-brand">
          <Logo />
          <button
            className="icon-button mobile-only"
            aria-label="Close navigation"
            onClick={() => setMenu(false)}
          >
            <X />
          </button>
        </div>
        {admin ? (
          <span className="admin-label">Control room</span>
        ) : (
          <Link className="button primary sidebar-create" to="/app/create">
            <Plus size={17} /> Plan a journey
          </Link>
        )}
        <nav aria-label={admin ? "Admin navigation" : "Workspace navigation"}>
          {(admin ? adminNav : userNav).map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                `sidebar-link ${isActive ? "active" : ""}`
              }
            >
              <item.icon size={18} />
              <span>{item.label}</span>
              {item.to === "/app/notifications" && unread > 0 && (
                <span className="nav-count">{unread}</span>
              )}
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-bottom">
          {!admin && (
            <nav aria-label="Account navigation">
              {[
                { to: "/app/safety", label: "Safety & trust", icon: Shield },
                { to: "/app/support", label: "Help & support", icon: LifeBuoy },
                { to: "/app/settings", label: "Settings", icon: Settings },
              ].map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  className={({ isActive }) =>
                    `sidebar-link ${isActive ? "active" : ""}`
                  }
                >
                  <item.icon size={18} />
                  <span>{item.label}</span>
                </NavLink>
              ))}
            </nav>
          )}
          {account.role === "admin" && (
            <Link className="preview-switch" to={admin ? "/app" : "/admin"}>
              {admin ? "Back to your space" : "Administration"}
              <ArrowUpRight size={15} />
            </Link>
          )}
          <div className="sidebar-profile">
            <Link to="/app/profile">
              <span className="account-photo">
                <ContactPhoto
                  key={`${profile.userId}:${profile.photoUrl}`}
                  person={profile}
                />
              </span>
              <span>
                <strong>{profile.name}</strong>
                <small>Signed in</small>
              </span>
            </Link>
            <button
              aria-label="Sign out"
              className="icon-button"
              onClick={signOut}
            >
              <LogOut size={17} />
            </button>
          </div>
        </div>
      </aside>
      {menu && (
        <button
          className="menu-shade"
          aria-label="Close navigation"
          onClick={() => setMenu(false)}
        />
      )}
      <div className="workspace-body" inert={menu}>
        <header className="workspace-header">
          <div>
            <button
              className="icon-button mobile-only"
              aria-label="Open navigation"
              onClick={() => setMenu(true)}
            >
              <Menu size={21} />
            </button>
            <span className="workspace-caption">
              {admin
                ? "A little care. A better community."
                : "Your city. Your people."}
            </span>
          </div>
          <div>
            <CurrentCity />

            <Link
              className="icon-button notification-link"
              to="/app/notifications"
              aria-label={`Notifications, ${unread} unread`}
            >
              <Bell size={19} />
              {unread > 0 && <i />}
            </Link>
            <Link to="/app/profile" aria-label="Your profile">
              <span className="account-photo compact">
                <ContactPhoto
                  key={`${profile.userId}:${profile.photoUrl}`}
                  person={profile}
                />
              </span>
            </Link>
          </div>
        </header>
        <main
          id="main"
          className={`workspace-main${location.pathname === "/app" ? " workspace-dashboard" : ""}`}
        >
          <m.div
            key={location.pathname}
            initial={motion ? { opacity: 0, y: 8 } : false}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: motion ? 0.22 : 0 }}
          >
            <Outlet />
          </m.div>
        </main>
        <footer className="workspace-footer">
          <span>
            <PixelMark className="footer-mark" />A little company goes a long
            way.
          </span>
          <Link to="/">
            Back to ORIX <ArrowUpRight size={14} />
          </Link>
        </footer>
      </div>
      <nav
        className="mobile-nav"
        inert={menu}
        aria-label={admin ? "Admin quick navigation" : "Quick navigation"}
      >
        {admin ? (
          <>
            <NavLink to="/admin" end>
              <LayoutDashboard />
              <span>Overview</span>
            </NavLink>
            <NavLink to="/admin/users">
              <Users />
              <span>People</span>
            </NavLink>
            <NavLink to="/admin/reports" className="mobile-plan">
              <Flag />
              <span>Reports</span>
            </NavLink>
            <NavLink to="/admin/support">
              <LifeBuoy />
              <span>Support</span>
            </NavLink>
            <button onClick={() => setMenu(true)}>
              <Menu />
              <span>More</span>
            </button>
          </>
        ) : (
          <>
            <NavLink to="/app" end>
              <Home />
              <span>Home</span>
            </NavLink>
            <NavLink to="/app/explore">
              <Compass />
              <span>Explore</span>
            </NavLink>
            <NavLink to="/app/create" className="mobile-plan">
              <Plus />
              <span>Plan</span>
            </NavLink>
            <NavLink to="/app/messages">
              <MessageSquare />
              <span>Chat</span>
            </NavLink>
            <NavLink to="/app/profile">
              <User />
              <span>You</span>
            </NavLink>
          </>
        )}
      </nav>
    </div>
  );
}
