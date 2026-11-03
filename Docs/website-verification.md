# ORIX website verification

Started 2026-11-02; completed 2026-11-03 for the new `website/` frontend. These results do not verify the existing Flutter app, Go/Python runtime migration, real authentication, transport providers, or production services.

## 2026-11-03 — Landing revision after design review

Replaced the centered hero stack with a broad, left-aligned two-column composition and widened the navigation and landing sections. Replaced the hero’s washed-out isometric city with a side-view SVG autorickshaw, passenger, palms and skyline. Road motion, wheel movement and body suspension use CSS; the scene pauses offscreen/in hidden tabs and honours motion preferences. Removed the floating captions that overlapped the old art.

Created `Docs/assets/orix-logo-reference.svg` from the existing mark and outlined Pixelify lettering. It has a transparent background and no external font or image dependencies. It remains separate and is not imported or applied anywhere in the website, as requested.

Actual revision checks: TypeScript/Vite build and targeted Prettier check passed; inspected 2044×1167, 1440×900, 390×844 and 320×740 layouts; no document overflow in the inspected narrow layouts; desktop/mobile landing axe scans reported zero violations with gradient contrast incomplete cases. Verified offscreen pause and the global motion toggle. The original full-route results below predate this revision and are retained as historical evidence, not a newly repeated 36-route suite.

## Checks actually executed

| Check | Result |
| --- | --- |
| Website dependency installation | Completed; npm reported 0 vulnerabilities during installation. This is not a malware certification. |
| `npm run build` from `website/` | Passed: strict TypeScript checking and Vite production bundling. |
| Prettier source/config check | Passed after formatting. |
| Live desktop routes, 1440 × 1000 (development and final production build) | All 36 loaded with visible headings, no error-boundary screen and no document horizontal overflow. |
| Live mobile routes, 390 × 844 (development and final production build) | All 36 loaded with visible headings, no error-boundary screen and no document horizontal overflow. |
| Narrow layout boundary, 320 × 740 | All 36 routes loaded without document horizontal overflow after fixing intrinsic form-control sizing. |
| axe-core 4.12.1 scans | Final desktop/mobile route passes reported zero automated violations on each of the 36 routes. Incomplete checks remain, especially gradient/canvas contrast and modal focus guards. |
| Visual review | Desktop landing/workspace/admin and mobile landing, planner and admin screenshots inspected; selected-action contrast and text-spacing heading clipping corrected. |
| Neutral palette | Source scan found no non-neutral hexadecimal color literals in the website source. |
| Ignored artifacts | `website/node_modules/` and `website/dist/` confirmed ignored by Git. |
| Browser runtime errors | No application JavaScript errors reported in the inspected development/production sessions. |

The live route list covered landing, login, onboarding, about, privacy, terms, account deletion; dashboard, explore, create, rides, a seeded group detail, history, messages, notifications, profile, settings, safety, support queue/thread, driver setup, embedded onboarding; admin overview, users, rides, reports, support queue/thread, operations, audit; maintenance, offline, suspended, banned, update and not-found states.

Final screenshots: [landing](website-landing.png) and [admin workspace](website-admin.png).

Detailed browser outputs and representative screenshots were saved under `/tmp/orix-web-qa/` during the session. A compact durable summary is [website-verification.json](website-verification.json).

## Interactions verified live

- Entered preview through the landing action; verified workspace URL, page title and visible heading. Production-mode sign-out returned to login; direct admin entry while outside preview redirected to login with its destination preserved; explicit preview entry returned to admin.
- Sent a group message through the composer and confirmed it appeared in the local conversation. No remote recipient or real delivery was claimed.
- Confirmed an existing journey, marked arrival, and observed a separate trip-start action. Arrival did not finish the trip. Simulated start, marked completion, selected 4 stars and saved the rating.
- Created a new two-stage auto request for a future date, with route labels, note and a 15-minute flexibility value. Confirmed persisted preview state and the new detail URL.
- Edited the unmatched destination in its dialog and observed the updated detail heading.
- Submitted a fictional safety report, observed it in the admin queue, selected a warning with a reason, and confirmed reviewed state plus the corresponding audit entry.
- Created a support ticket, sent a reply, resolved it and verified the composer became read-only with a reopen action.
- Filtered suspended members, opened the management dialog, restored a fictional member with a reason and confirmed the audit entry.
- Paused motion in settings and verified the stored setting and document state. Compared canvas pixels after a delay and confirmed the paused scene stayed unchanged.
- Emulated OS reduced motion and confirmed the media query and disabled CSS sprite animation.
- Opened the mobile navigation drawer, verified focus moved to its close control and the background became inert, then pressed Escape and confirmed focus returned to the navigation trigger and the closed sidebar was hidden.
- Observed the favicon change between frames and route titles change on navigation.
- Reset the preview through its confirmation dialog and verified the original two rides/two reports and enabled motion were restored.
- Applied user text-spacing overrides to the mobile landing page. Visual review caught a non-wrapping heading; it was changed to allow reflow and re-inspected with the overrides active.

## Production-mode smoke measurement

Served the built assets locally with `npm run preview` on port 4173. The landing page loaded, its favicon changed, preview navigation reached `/app`, and the production workspace had no horizontal overflow or reported runtime errors.

One local Chromium measurement reported TTFB 1.3 ms, FCP 40 ms, LCP 388 ms and rounded CLS 0.01. INP was unavailable. This was a localhost smoke measurement on the development host, not a throttled device benchmark, field measurement, or a promise about every device/network. The viewport was set during the session, so these numbers should not be used as release performance claims.

## Failures found and corrected

- An open production-preview tab retained old lazy-chunk URLs while a later build replaced `dist/`; navigating to an unloaded route showed the recovery screen. A fresh navigation loaded the new build and the guarded preview flow passed. Production deployment must publish atomically and retain old hashed assets for existing clients.
- The host initially denied listening on the preview port; the local server was started with the required execution approval.
- Chrome's normal sandbox was unavailable on this host. The isolated browser sessions used `--no-sandbox` for localhost testing only; no system security settings were changed.
- Initial compilation identified missing Vite CSS declarations and unused imports; corrected before the successful build.
- Initial audits found small-text contrast and unnamed landmarks; corrected. Mobile login initially had its only `h1` inside the hidden desktop art panel; the actual form now owns the page heading.
- The mobile table's absolutely positioned hidden header label escaped its scroll container; establishing a containing block removed document overflow.
- The chat log needed an explicit keyboard focus target. The closed mobile sidebar also needed visibility/focus management.
- The selected mobile center action inherited dark active-link text on its dark background; an explicit inverted active state now has white text on `#222`.
- At 320 px, native form controls imposed a minimum grid width; zero-minimum tracks, constrained fields and a narrow form layout fixed it.
- A browser click on a low table row initially landed behind the fixed mobile navigation. Scroll padding/margins were added, and filtered member management was verified. Full touch ergonomics still need physical-device review.

## Verification limits

Automated scans leave some contrast/focus cases incomplete. Screen readers, full keyboard traversal of every state, browser zoom across all routes, international scripts, physical low-end phones, Safari/Firefox, real WebSocket reconnects, provider maps, denied permissions, real admin authorization, push delivery and production policy compliance remain outside this frontend-only verification. The site includes explicit motion controls and responsive safeguards, but universal smoothness cannot be established with one host.

No application backend/mobile tests, deployment, database jobs, environment setup or emulator QA were performed. The existing app rewrite/testing gate remains intact. See [website-integration.md](website-integration.md) for the backend/mobile handoff and launch requirements.


## 2026-11-03 minimal hero and unified pooling revision

The user rejected the route-form/diagram hero during review. The final design
uses left-aligned pixel type, open space, a primary login CTA, secondary preview
entry and a small static pixel street vignette. No route form or animated scene
remains in the hero. Google-only minimal login, transparent original wordmark,
automatic passenger pool UI and separate recurring lift registration are present.
The local demo does not establish real authentication, routing or matching.

Actual checks in this pass:

- `npm run build` passed after the final edits (TypeScript and Vite production build).
- `npx prettier --check src` passed.
- agent-browser inspected all 36 routes at 1440×900, 768×1024, 390×844 and
  320×740: 144 route/viewport checks, no document-wide horizontal overflow,
  unexpected redirects or recovery-screen text. This sweep preceded the last
  hero simplification and chat sizing adjustments.
- After final adjustments, landing and chat were rechecked at all four sizes:
  no document overflow or recovery-screen text; automated accessibility scans
  reported zero violations on these eight cases. Incomplete checks and the
  existing verification limits still apply.
- Visual screenshots were inspected for landing, login, dashboard, request,
  lift registration, chat and admin layouts. Mobile dashboard overlap,
  narrow hero wrapping, a dark-section caption contrast issue and mobile chat
  sizing were corrected. Screenshots are representative, not exhaustive proof
  of every interaction state or browser.

Current evidence and source hashes are under `minimal_hero_revision` in
`website-verification.json`; prior entries are historical. Current landing
screenshots are `website-landing.png` and `website-landing-mobile.png`.
No app/backend tests, live service integration, emulator QA or deployment ran.
