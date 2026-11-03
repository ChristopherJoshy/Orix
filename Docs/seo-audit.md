# ORIX web SEO — 2026-11-03

The owner requested maximum organic discoverability, including “ride share”,
without changing the design. They subsequently cancelled Google Play publication
because of the registration cost and selected the website as the distribution
channel. No paid registration, advertising or SEO subscription was started.

## Findings and changes

| Priority | Finding and evidence | Change |
| --- | --- | --- |
| High | Every public path received the same empty React shell and homepage canonical. Source index.html had one title/description; the live DOM title was “Go together. · ORIX”. | Build-time rendering of the actual six public route components; unique initial HTML and matching client navigation metadata. |
| High | No ORIX property appeared in the connected Google account’s Search Console property list. | Added the HTTPS URL-prefix property and its public HTML verification tag; ownership and sitemap submission results are recorded below. |
| High | Generic app tab titles did not describe ride sharing; no structured data appeared in the live rendered DOM. | Homepage targets ride share, shared autos, travel partners and lifts. About explains ride sharing. Added factual WebSite, Organization, WebPage/AboutPage and breadcrumb JSON-LD, with no invented ratings, people, prices or service coverage. |
| Medium | Catch-all SPA rewriting returned an ordinary page for unknown paths, and private/login pages inherited the public shell. | Clean public URLs; explicit private SPA rewrites with noindex; existing not-found component rendered as a real HTTP 404. Authentication and authorization remain unchanged. |
| Medium | Account deletion was absent from the five-page sitemap. | Generate the six-page canonical sitemap from the same public metadata map; robots.txt points to it. No fake last-modified dates or private URLs. |
| Medium | Illustrations used 1.38–1.61 MB PNGs. | Lossless WebP versions: 919,184 / 992,544 / 839,188 bytes. Decoded RGBA pixel equality checked for all three; dimensions and artwork unchanged. Existing PNG source assets retained. |
| Medium | Public route JavaScript was discovered after the main bundle. | Preload the matching route module and include its actual styles in initial HTML. Existing fonts, layouts, colors and visible copy unchanged. |
| Medium | No IndexNow notification workflow. | Public verification file and an explicit post-deployment notification command for the six canonical pages. It verifies the live file before submitting and never runs inside a build. |

## Actual verification and deployment

- Website TypeScript/Vite production build passed, including six prerendered pages.
- `npm run test:seo` passed: distinct titles/descriptions, one canonical per page,
  original component content/H1 without JavaScript, JSON-LD/social consistency,
  actual referenced asset files/styles, sitemap contents, noindex shell and 404
  routing configuration. This is a focused build-contract check, not account QA.
- `git diff --check` passed at the implementation checkpoint.
- Each WebP was decoded and compared with the corresponding original RGBA pixels;
  all were identical. No visual redesign or creative artwork editing occurred.
- First CLI preview failed because its source root omitted the configured
  `website/` directory. Added a root website-only `.vercelignore`, then inspected
  the dry-run upload list: 92 website files, no local/private env files, credentials,
  backend source, Flutter builds or node_modules.
- Next preview failed because the Vercel Preview environment had no Firebase web
  configuration. No fake Firebase values or authentication fallback was added.
- Automatic approval review rejected staging a production build after that preview
  failure because behavior was unverified. No production action ran at that point.
- Rebuilt an isolated preview with only existing public VITE Firebase/API values;
  no server secrets were passed. Deployment dpl_HZGFGpYUnME2a3qGEh9qgLDKqMmE is READY.
- Corrected preview dpl_H3wEwNCzXhDhD5ZR7m64YQsM4THE passed actual HTTP checks:
  six public pages, six private noindex shells, robots/sitemap/ownership file,
  real 404, clean-URL redirect, and 16 referenced public assets. The first HTTP
  run caught a login 404 caused by using `.html` in a cleanUrls rewrite; fixed
  the destination to `/spa` per official Vercel guidance and retested.
- Production staging was approved after those checks; the earlier automatic
  rejection was resolved by verification, without weakening access controls.
- Initial production deployment dpl_4qmA9MHjX96q2KJsSJefZZqQre2H was promoted and
  passed the same live HTTP checks with no public noindex response header.
- Google verified ownership via the HTML tag. Search Console processed the
  sitemap successfully and discovered all six pages; its initial “Couldn't fetch”
  table state resolved on the detail screen. Independent Googlebot-header requests
  returned 200 application/xml, six valid XML entries and no noindex header.
- IndexNow accepted the six public canonical URLs with HTTP 202. This confirms
  submission acceptance, not indexing or a ranking result.
- First live PageSpeed mobile report (Lighthouse 13.6.0, slow 4G) recorded SEO 100,
  Accessibility 100, Best Practices 100, Performance 76, FCP 0.9s, LCP 7.3s,
  TBT 0ms and CLS 0. No real-user Core Web Vitals data was available.
  Report: https://pagespeed.web.dev/analysis/https-orixrideshare-tech/yzax7fndlr?form_factor=mobile
- Added responsive high-quality exports of the same hero (57,866 / 112,006 /
  187,532 bytes at 480 / 768 / 1122 pixels), keeping the original lossless fallback,
  crop, CSS and artwork. These resized/compressed delivery variants are not claimed
  pixel-identical to the full-resolution source. A subsequent mobile run retained
  SEO/Accessibility/Best Practices 100, with Performance 73, FCP 3.2s, LCP 4.8s,
  TBT 0ms and CLS 0. Report-to-report timing varies; do not claim an overall score
  improvement from this run. Live browser used the 480w variant at 380 CSS pixels
  and loaded it successfully without horizontal overflow.
- Google URL Inspection fetched the homepage successfully with indexing/crawling
  allowed, but reported a duplicate whose selected canonical was the www host.
  Added a permanent www-to-apex redirect retaining paths, and kept all public
  canonicals/sitemap URLs consistent. The final request/redirect checks follow.
- Public startup now waits for its actual route module before mounting the existing
  app, retaining the already visible prerendered content instead of flashing a lazy
  spinner. Existing error recovery, private routing and authenticated state remain.
- About-page illustration also gained responsive exports (29,406 / 79,132 /
  185,370 bytes at 384 / 768 / 1448 pixels) with the existing layout and original
  lossless fallback. All srcset URLs are checked in the focused build tests.
- Final production dpl_CnfpAeTDRsB4GFJUsEJsaQq5Kf6p is READY and assigned to
  https://orixrideshare.tech. Live checks pass for all six public pages, six
  private shells, sitemap/robots/ownership file, 404, clean URLs and 22 assets.
  Explicit root and wildcard host rules were required: www /, /about and /login
  now return 308 to the corresponding apex paths. IndexNow accepted the final
  six public URLs with HTTP 200.
- Google’s live test at 23:04 IST reported “URL is available to Google” and
  “Page can be indexed”. The manual indexing submission failed at Google with
  a reCAPTCHA-service connection error; it was not accepted and no CAPTCHA was
  bypassed. The verified sitemap is already processed, so ordinary crawling can
  continue. Actual index inclusion/canonical selection and “ride share” rankings
  remain unproven and need a later Search Console check.
- Final mobile PageSpeed report at 23:07 IST: SEO 100, Accessibility 100,
  Best Practices 100, Performance 72, FCP 3.2s, LCP 5.2s, TBT 0ms, CLS 0. It is
  a throttled lab result, not a search ranking or real-user CWV result. Mobile
  performance still needs work; no maximum-performance claim is made.
  Report: https://pagespeed.web.dev/analysis/https-orixrideshare-tech/f5h5w8nn4x?form_factor=mobile

## Constraints and remaining work

Technical eligibility and URL submission cannot guarantee indexing, placement or
visibility for every competitive generic query. Search Console initially has no
ORIX history; use its actual query/page reports after Google has collected data.
No traffic, keyword volume, rankings, backlinks or Core Web Vitals were fabricated.

Keep private journeys, identities, chat, support and moderation out of sitemaps and
search results. Do not publish private route details as SEO landing pages. The
current privacy notice explicitly calls itself a preview rather than a complete
service policy; completing truthful policy disclosures remains an operator/legal
content task, not something this SEO change can certify. Broader backend, account,
realtime, device and release QA remains on the existing owner hold.

For further organic growth, use actual query impressions to prioritize substantive
travel-sharing help and real service-area information. Avoid keyword stuffing,
doorway city pages, fake reviews, paid link schemes or unsupported claims that a
ride/match will be available. Genuine links from the owner's established public
profiles and relevant communities can help discovery; no unsolicited messages or
posts were sent.

## Official research used

- [Google Search Essentials](https://developers.google.com/search/docs/essentials)
- [Google SEO Starter Guide](https://developers.google.com/search/docs/fundamentals/seo-starter-guide)
- [JavaScript SEO and prerendering](https://developers.google.com/search/docs/crawling-indexing/javascript/javascript-seo-basics)
- [Canonical URL guidance](https://developers.google.com/search/docs/crawling-indexing/consolidate-duplicate-urls)
- [Site name structured data](https://developers.google.com/search/docs/appearance/site-names)
- [Crawlable link guidance](https://developers.google.com/search/docs/crawling-indexing/links-crawlable)
- [Bing IndexNow setup](https://www.bing.com/indexnow/getstarted)
- [IndexNow protocol FAQ](https://www.indexnow.org/faq)
- [Vercel upload exclusions](https://vercel.com/docs/deployments/vercel-ignore)

## Transparency regression and repair

The owner reported a black rectangle around the hero after responsive delivery
was introduced. The export script had converted the transparent PNGs to RGB,
which discarded alpha. This was an implementation regression; the earlier
lossless full-resolution RGBA exports were unaffected. Regenerated all six
responsive variants from the originals as RGBA and checked each decoded alpha
plane byte for byte against the corresponding resized source. New `-alpha` URLs
avoid stale opaque browser/CDN copies. Added WebP alpha flag assertions to the
focused build tests; production build and those tests pass. Chrome visual review
of the actual local website confirms the transparent hero on the existing light
background. No CSS/layout/type or composition changes were needed. Final live
verification passed: all six downloaded files match the locally verified bytes;
Chrome loads the 768w alpha hero, and screenshot corner samples are the existing
light page background rather than black. Corrected hero variants are 99,150 /
195,106 / 321,996 bytes; About variants are 48,164 / 129,082 / 310,526 bytes.
Original dimensions/layout/CSS and composition remain. The earlier opaque exports
were removed from the source tree; they must not be reintroduced.
