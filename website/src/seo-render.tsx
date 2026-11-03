import { renderToString } from "react-dom/server";
import { StrictMode } from "react";
import type { ReactNode } from "react";
import { StaticRouter } from "react-router-dom";
import App from "./App";
import Landing from "./pages/Landing";
import { PublicInfo, StatusPage } from "./pages/Public";
import DownloadAndroid from "./pages/DownloadAndroid";
export { publicPages, pageSchema, siteOrigin } from "./lib/seo";

// Render only anonymous public content. Effects, sign-in and API mutations never run here.
export function renderPublicPage(path: string) {
  function PublicRouter({ children }: { children: ReactNode }) {
    return <StaticRouter location={path}>{children}</StaticRouter>;
  }
  const page =
    path === "/" ? (
      <Landing />
    ) : path === "/download" ? (
      <DownloadAndroid />
    ) : (
      <PublicInfo page={path.slice(1)} />
    );
  return renderToString(
    <StrictMode>
      <App
        Router={PublicRouter}
        initialPublicPage={page}
        initialPublicPath={path}
      />
    </StrictMode>,
  );
}

export function renderNotFound() {
  return renderToString(
    <StaticRouter location="/404">
      <StatusPage />
    </StaticRouter>,
  );
}
