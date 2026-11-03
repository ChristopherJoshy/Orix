import { readFile } from "node:fs/promises";

// Run only after the matching production deployment is live, never during a build.
// IndexNow's verification value is deliberately public, not an account credential.
const origin = "https://orixrideshare.tech";
const key = (await readFile("public/indexnow-key.txt", "utf8")).trim();
if (!/^[a-zA-Z0-9-]{8,128}$/.test(key))
  throw new Error("Invalid IndexNow verification value");
const sitemap = await readFile("dist/sitemap.xml", "utf8");
const urlList = [...sitemap.matchAll(/<loc>([^<]+)<\/loc>/g)].map(
  (match) => match[1],
);
if (!urlList.length || urlList.some((url) => new URL(url).origin !== origin))
  throw new Error("Sitemap must contain only canonical ORIX URLs");
const keyLocation = `${origin}/indexnow-key.txt`;
const verification = await fetch(keyLocation, {
  redirect: "error",
  signal: AbortSignal.timeout(20000),
});
if (!verification.ok || (await verification.text()).trim() !== key)
  throw new Error(
    "Production IndexNow verification file does not match; deploy first",
  );
const response = await fetch("https://api.indexnow.org/indexnow", {
  method: "POST",
  headers: { "Content-Type": "application/json; charset=utf-8" },
  body: JSON.stringify({
    host: new URL(origin).hostname,
    key,
    keyLocation,
    urlList,
  }),
  redirect: "error",
  signal: AbortSignal.timeout(30000),
});
if (![200, 202].includes(response.status))
  throw new Error(
    `IndexNow submission was not accepted (HTTP ${response.status})`,
  );
console.log(
  `IndexNow accepted ${urlList.length} public URLs (HTTP ${response.status}). Submission does not guarantee indexing or ranking.`,
);
