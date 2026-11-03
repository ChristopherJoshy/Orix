import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "@fontsource/pixelify-sans/latin-500.css";
import "@fontsource/pixelify-sans/latin-600.css";
import "@fontsource/manrope/latin-400.css";
import "@fontsource/manrope/latin-500.css";
import "@fontsource/manrope/latin-600.css";
import "./styles.css";
import App from "./App";

// A tab opened before a deployment can still request the previous build's
// chunks. Refresh once to fetch the new entry point, preserving the route.
window.addEventListener("vite:preloadError", (event) => {
  if (!navigator.onLine) return;
  try {
    const key = "orix-chunk-recovery";
    const previous = Number(sessionStorage.getItem(key) || 0);
    if (Date.now() - previous < 60000) return;
    sessionStorage.setItem(key, String(Date.now()));
    event.preventDefault();
    window.location.reload();
  } catch {
    // Storage restrictions leave the normal recovery UI available.
  }
});

async function mount() {
  const path = window.location.pathname;
  let page;
  try {
    // Keep the real prerendered content visible until its interactive component
    // is ready, instead of replacing an already painted page with a lazy spinner.
    if (path === "/") {
      const { default: Landing } = await import("./pages/Landing");
      page = <Landing />;
    } else if (path === "/download") {
      const { default: DownloadAndroid } =
        await import("./pages/DownloadAndroid");
      page = <DownloadAndroid />;
    } else if (
      ["/about", "/privacy", "/terms", "/account-deletion"].includes(path)
    ) {
      const { PublicInfo } = await import("./pages/Public");
      page = <PublicInfo page={path.slice(1)} />;
    }
  } catch {
    // Let the existing lazy-route error boundary show its authored recovery UI.
    // The existing preload-error listener also retries a stale deployment once.
  }
  createRoot(document.getElementById("root")!).render(
    <StrictMode>
      <App
        initialPublicPage={page}
        initialPublicPath={page ? path : undefined}
      />
    </StrictMode>,
  );
}
void mount();
