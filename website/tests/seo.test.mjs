import assert from "node:assert/strict";
import { readFile, access } from "node:fs/promises";

const routes = [
  "/",
  "/about",
  "/download",
  "/privacy",
  "/terms",
  "/account-deletion",
];
const titles = new Set();
const descriptions = new Set();
for (const route of routes) {
  const html = await readFile(
    `dist/${route === "/" ? "index" : route.slice(1)}.html`,
    "utf8",
  );
  const canonical = [...html.matchAll(/<link rel="canonical" href="([^"]+)"/g)];
  assert.equal(canonical.length, 1, `${route}: exactly one canonical`);
  assert.equal(canonical[0][1], `https://orixrideshare.tech${route}`);
  assert.match(
    html,
    /<meta name="robots" content="index, follow, max-image-preview:large"/,
  );
  assert.doesNotMatch(html, /noindex/);
  assert.equal(
    (html.match(/<h1[\s>]/g) ?? []).length,
    1,
    `${route}: one public page heading`,
  );
  assert.doesNotMatch(
    html,
    /class="loading-screen"/,
    `${route}: initial response contains the page, not a loading fallback`,
  );
  assert.match(
    html,
    /<h1[\s>]/,
    `${route}: heading available without JavaScript`,
  );
  assert.match(
    html,
    /<div id="root" data-prerendered="public">[\s\S]*<(?:div|main)/,
    `${route}: existing public component rendered`,
  );
  const title = html.match(/<title>([^<]+)<\/title>/)[1];
  const description = html.match(
    /<meta name="description" content="([^"]+)"/,
  )[1];
  assert.ok(!titles.has(title), `${route}: unique title`);
  assert.ok(!descriptions.has(description), `${route}: unique description`);
  titles.add(title);
  descriptions.add(description);
  const schema = JSON.parse(
    html.match(
      /<script id="orix-structured-data" type="application\/ld\+json">([^<]+)<\/script>/,
    )[1],
  );
  assert.equal(schema["@context"], "https://schema.org");
  assert.equal(schema["@graph"][0].url, canonical[0][1]);
  assert.match(
    html,
    new RegExp(
      `<meta property="og:url" content="https://orixrideshare.tech${route}"`,
    ),
  );
  for (const match of html.matchAll(/(?:href|src)="(\/assets\/[^"]+)"/g))
    await access(`dist${match[1]}`);
  for (const match of html.matchAll(/srcSet="([^"]+)"|srcset="([^"]+)"/g)) {
    for (const candidate of (match[1] ?? match[2]).split(",")) {
      const [url, width] = candidate.trim().split(/\s+/);
      assert.match(width, /^\d+w$/);
      await access(`dist${url}`);
      // Responsive editorial exports must retain their original transparent canvas.
      const bytes = await readFile(`dist${url}`);
      assert.equal(bytes.toString("ascii", 8, 12), "WEBP");
      assert.equal(bytes.toString("ascii", 12, 16), "VP8X");
      assert.ok(bytes[20] & 0x10, `${url}: alpha channel must be preserved`);
    }
  }
  if (route === "/") assert.match(html, /href="\/assets\/Landing-[^"]+\.css"/);
  if (["/about", "/privacy", "/terms", "/account-deletion"].includes(route))
    assert.match(html, /href="\/assets\/Auth-[^"]+\.css"/);
}
for (const name of ["spa", "404"]) {
  const html = await readFile(`dist/${name}.html`, "utf8");
  assert.match(html, /noindex, follow/);
  assert.doesNotMatch(html, /rel="canonical"/);
}
assert.match(
  await readFile("dist/404.html", "utf8"),
  /We couldn’t find that page/,
);
const sitemap = await readFile("dist/sitemap.xml", "utf8");
assert.deepEqual(
  [...sitemap.matchAll(/<loc>([^<]+)<\/loc>/g)].map((match) => match[1]),
  routes.map((route) => `https://orixrideshare.tech${route}`),
);
const config = JSON.parse(await readFile("vercel.json", "utf8"));
assert.equal(config.cleanUrls, true);
assert.deepEqual(
  config.rewrites.map((rule) => rule.source),
  ["/app/:path*", "/admin/:path*", "/login", "/onboarding", "/status/:path*"],
);
assert.ok(config.rewrites.every((rule) => rule.destination === "/spa"));
assert.deepEqual(config.redirects, [
  {
    source: "/",
    has: [{ type: "host", value: "www.orixrideshare.tech" }],
    destination: "https://orixrideshare.tech/",
    permanent: true,
  },
  {
    source: "/:path*",
    has: [{ type: "host", value: "www.orixrideshare.tech" }],
    destination: "https://orixrideshare.tech/:path*",
    permanent: true,
  },
]);
console.log(
  "SEO build checks passed: six public pages, canonical/social/schema consistency, CSS/assets, sitemap, private shell and genuine 404 routing.",
);
