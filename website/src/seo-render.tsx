import { renderToString } from "react-dom/server";
import { StaticRouter } from "react-router-dom";
import Landing from "./pages/Landing";
import { PublicInfo, StatusPage } from "./pages/Public";
import DownloadAndroid from "./pages/DownloadAndroid";
export { publicPages, pageSchema, siteOrigin } from "./lib/seo";

// Render only anonymous public content. Effects, sign-in and API mutations never run here.
export function renderPublicPage(path: string) {
  return renderToString(
    <StaticRouter location={path}>
      {path === "/" ? (
        <Landing />
      ) : path === "/download" ? (
        <DownloadAndroid />
      ) : (
        <PublicInfo page={path.slice(1)} />
      )}
    </StaticRouter>,
  );
}

export function renderNotFound() {
  return renderToString(
    <StaticRouter location="/404">
      <StatusPage />
    </StaticRouter>,
  );
}
