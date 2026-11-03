import { readFile, writeFile } from "node:fs/promises";
import { createServer } from "vite";

const server = await createServer({
  mode: "production",
  server: { middlewareMode: true, hmr: false, ws: false },
  appType: "custom",
});
try {
  const {
    renderPublicPage,
    renderNotFound,
    publicPages,
    pageSchema,
    siteOrigin,
  } = await server.ssrLoadModule("/src/seo-render.tsx");
  const template = await readFile("dist/index.html", "utf8");
  const manifest = JSON.parse(
    await readFile("dist/.vite/manifest.json", "utf8"),
  );
  const escape = (value) =>
    value
      .replaceAll("&", "&amp;")
      .replaceAll('"', "&quot;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;");
  function metadata(html, page, path) {
    html = html.replace(
      /<title>[^<]*<\/title>/,
      `<title>${escape(page.title)}</title>`,
    );
    for (const [attribute, key, value] of [
      ["name", "description", page.description],
      ["property", "og:title", page.title],
      ["property", "og:description", page.description],
      ["property", "og:url", `${siteOrigin}${path}`],
      ["name", "twitter:title", page.title],
      ["name", "twitter:description", page.description],
    ])
      html = html.replace(
        new RegExp(`<meta\\s+${attribute}="${key}"\\s+content="[^"]*"\\s*/?>`),
        `<meta ${attribute}="${key}" content="${escape(value)}" />`,
      );
    return html.replace(
      /<link rel="canonical"[^>]*>/,
      `<link rel="canonical" href="${siteOrigin}${path}" />`,
    );
  }
  // The SPA shell never inherits the homepage's canonical or indexability.
  const shell = template
    .replace(/<link rel="canonical"[^>]*>/, "")
    .replace(
      "</head>",
      '<meta name="robots" content="noindex, follow" /></head>',
    );
  await writeFile("dist/spa.html", shell);
  const statusCSS = (manifest["src/pages/Public.tsx"]?.css ?? [])
    .map((file) => `<link rel="stylesheet" href="/${file}" />`)
    .join("\n");
  await writeFile(
    "dist/404.html",
    shell
      .replace(
        '<div id="root"></div>',
        `<div id="root">${renderNotFound()}</div>`,
      )
      .replace("</head>", `${statusCSS}</head>`)
      .replace(/<title>[^<]*<\/title>/, "<title>Page Not Found | ORIX</title>"),
  );
  for (const [path, page] of Object.entries(publicPages)) {
    let html = metadata(template, page, path);
    const css = new Set();
    const seen = new Set();
    function collect(key) {
      if (seen.has(key)) return;
      seen.add(key);
      for (const file of manifest[key]?.css ?? []) css.add(file);
      for (const dependency of manifest[key]?.imports ?? [])
        collect(dependency);
    }
    collect("index.html");
    const entry =
      path === "/"
        ? "src/pages/Landing.tsx"
        : path === "/download"
          ? "src/pages/DownloadAndroid.tsx"
          : "src/pages/Public.tsx";
    collect(entry);
    const styles = [...css]
      .filter((file) => !html.includes(`/${file}`))
      .map((file) => `<link rel="stylesheet" href="/${file}" />`)
      .join("\n");
    const schema = JSON.stringify(pageSchema(path)).replaceAll("<", "\\u003c");
    html = html.replace(
      "</head>",
      `${styles}\n<link rel="modulepreload" href="/${manifest[entry].file}" />\n<meta name="robots" content="index, follow, max-image-preview:large" />\n<script id="orix-structured-data" type="application/ld+json">${schema}</script>\n</head>`,
    );
    html = html.replace(
      '<div id="root"></div>',
      `<div id="root">${renderPublicPage(path)}</div>`,
    );
    await writeFile(
      `dist/${path === "/" ? "index" : path.slice(1)}.html`,
      html,
    );
  }
  await writeFile(
    "dist/sitemap.xml",
    `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${Object.keys(
      publicPages,
    )
      .map((path) => `  <url><loc>${siteOrigin}${path}</loc></url>`)
      .join("\n")}\n</urlset>\n`,
  );
  console.log(
    `Pre-rendered ${Object.keys(publicPages).length} public pages with matching metadata and styles.`,
  );
} finally {
  await server.close();
}
