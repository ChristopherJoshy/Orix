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

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
