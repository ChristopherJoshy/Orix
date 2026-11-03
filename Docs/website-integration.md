# ORIX website integration plan

## Scope and current boundary

`website/` is a new React/TypeScript frontend. It leaves the existing Flutter client and Python/Go backend in place. This task implements design and browser-local demonstrations, not production authentication, booking, matching, chat delivery, or administration.

The website explicitly labels its workspace as an interactive preview. `src/lib/store.ts` is the local demonstration adapter; `src/lib/data.ts` contains fictional people, trips, messages, and queues. Browser storage must never become the production authentication or personal-data store. Account roles, verification, counts, driver data, and group transitions shown here do not establish real permissions or service state.

The Go inventory in [backend-api.md](backend-api.md) describes staged implementation, not verified runtime behavior. Python remains until the existing migration gate passes. This website does not change that status.

## Run the frontend

From `website/`, install the locked dependencies with `npm ci`, then run `npm run dev`. `npm run build` runs TypeScript checking and produces the static application in `website/dist/`. `npm run preview` serves that build for inspection. Node 24 was used during implementation. Check the locked Vite engine requirements before using another Node version.

The static host must route application URLs to `index.html` while serving actual assets normally. Deploy over HTTPS with an appropriate CSP, security headers, immutable caching for hashed assets, and short-lived/revalidated HTML. No environment variables or external API keys are needed for the preview. Publish releases atomically and retain previous hashed assets long enough for open tabs to finish loading their routes; rebuilding a served `dist/` in place can invalidate an old tab’s lazy imports. No deployment was performed by this task.

## Feature inventory

| Browser route | Frontend behavior | Real integration needed |
| --- | --- | --- |
| `/` | Minimal pixel landing, small static street vignette, pooling explanation, safety story, FAQs, motion control | Final operator/brand and service availability copy |
| `/login` | Minimal Google-only sign-in design; honest unconnected state; explicit preview entry | Firebase Google sign-in and server verification |
| `/onboarding`, `/app/onboarding` | Name, explicit community agreement, preview profile | Authenticated onboarding and versioned consent |
| `/app` | Personal workspace, current journey, notification count, planning shortcuts | Own profile, rides, groups, unread state |
| `/app/explore` | Automatic pool result, route and safety preferences; no manual mode or pool selection | Server automatic assignment, eligibility, atomic capacity |
| `/app/create` | Single route/date/time request, flexibility, women-only and lift acceptance preferences | Place coordinates, validated route/time window, request submission |
| `/app/rides`, `/app/history` | Upcoming/past/all filters and journey detail links | Cursor-based own ride/group history |
| `/app/rides/:id` | Group members, pickup illustration, unmatched edit, confirmation, arrival, start/finish preview, leave, ratings | Membership and role-authorized lifecycle, persisted pickups/ratings |
| `/app/messages` | Group selection/search, message composer, local append, completed-thread read-only state | Membership-scoped WebSocket and REST history/resync |
| `/app/notifications` | All/unread views, individual/read-all, delete, preference link | Own inbox/read/delete APIs and delivery state |
| `/app/profile` | Pixel avatar, own profile edit, history and driver links | Profile API, server verification, scoped avatar upload |
| `/app/rider` | Separate lift route registration with endpoints, date/time, seats, daily availability and pause controls | Driver onboarding/verification and routing contract |
| `/app/settings` | Notification preferences, motion setting, preview reset, sign-out, deletion explanation | Preference APIs, permission handling, recent-auth deletion |
| `/app/safety` | Preference, contact form, block/unblock, report creation/history | Trusted safety attributes, two-way blocks, scoped reports; emergency-contact contract if supported |
| `/app/support`, `/app/support/:id` | Ticket creation, conversation, replies, resolve/reopen | Own support threads and restricted-account appeals |
| `/admin` | Sample community overview, queue links, labelled illustrative pixel chart | Authorized stats with defined metrics/time ranges |
| `/admin/users` | Member search/filter, status/role/restore preview, reason notes | Exact lookup, history, protected role/restoration APIs |
| `/admin/rides` | Route/ID search, review, cancellation reason, activity entry | Authorized group lookup and atomic safety cancellation |
| `/admin/reports` | Queue filters, report context, reasoned decisions, local restriction | Evidence access, conflict-safe resolution and audit |
| `/admin/support`, `/admin/support/:id` | Support queue, staff reply, resolve/reopen | Admin support API and staff attribution |
| `/admin/operations` | Local maintenance confirmation, version form, sample delivery retry | Maintenance, version, worker/notification operations |
| `/admin/audit` | Local action log, search, CSV export | Bounded server audit log and access-controlled export |
| `/about`, `/privacy`, `/terms`, `/account-deletion` | Concise explanation and explicit preview/policy limitations | Final legal copy, actual contacts, retention and deletion policy |
| `/status/:kind`, unmatched paths | Maintenance, offline, restricted, update and not-found designs | Trusted service/account state; support access when restricted |

The city and route art are illustrations. They do not claim to locate people or provide navigation. The preview does not request geolocation, push permissions, payment, analytics, or live provider calls.

## Backend work to connect the site

1. **Choose and verify the runtime.** Exercise the staged API/WebSocket contracts against the actual persistence, auth and job infrastructure before connecting a public client. Keep a typed frontend transport adapter separate from the domain/view state. Replace fixtures through that adapter; do not scatter requests through page components.
2. **Authentication.** Configure Firebase public web identifiers and authorized domains. Obtain real Firebase ID tokens, verify email and token revocation server-side, call `/api/v1/auth/verify`, then `/auth/onboard` as required. Add real pending/error/retry/sign-out/expired-session states. No college-domain gate or Telegram dependency. Use supported Firebase persistence/token handling; never copy tokens to the preview store or logs.
3. **Browser access.** Define exact CORS origins, supported methods and headers, cursor-header exposure (`X-Next-Cursor`), timeouts, request IDs and error shapes. If a BFF uses cookies, define Secure/HttpOnly/SameSite settings and CSRF protections appropriate to the deployment. Do not blindly enable credentialed wildcard CORS.
4. **Administration requires a server boundary.** The current staged admin APIs require Firebase authorization, a server-derived role, and `X-Admin-Secret`. A static frontend must never receive that secret, including through a `VITE_` variable. Add an authorized BFF/server capability or redesign the trusted admin contract. The server must validate every action and permission independently; hiding a navigation link is insufficient. Audit actor, target, reason, and result without sensitive content in logs.
5. **Places and road routes.** Connect to the configured authenticated `/places/search` adapter and permitted tile provider after verifying its final route/payload. Use a contracted/private provider; public OSMF Nominatim is excluded. Add cancellable/debounced search, a usable list fallback, attribution, clear selected coordinates and optional user-initiated GPS with denied/timeout handling. Replace the illustrated map only with data permitted by the provider. Lift itineraries require real road routing.
6. **Ride payloads.** Translate local date/time plus the selected flexibility into validated UTC bounds and the explicit timezone offset expected by the API. Carry precise coordinates separately from display labels. Passenger requests must carry route/time/preferences without requiring a mode choice; pool type is an assigned result. Lift registrations carry available passenger seats. Reconcile vehicle/driver eligibility and women-only eligibility with trusted server state. Enforce capacity and duplicate-active-assignment constraints atomically server-side.
7. **Matching and lifecycle.** Reconcile own `/rides`, `/rides/status`, and group APIs with automatic server assignment, replacing manual suggestion/reservation selection in the passenger flow. Revalidate membership/state after every mutation and on reconnect. A frontend preview match is not a match engine. Arrival must not mark a journey complete. Only authorized host/driver roles may start/finish real journeys; other members wait for the trusted event. Preserve personal pickup times, UTC schedules and stale-event suppression.
8. **Chat.** Implement the existing authenticated browser protocol: `orix.v1` plus `firebase.<ID token>` offerings, negotiating only `orix.v1`. Never place credentials in query strings or logs. Use stable client message IDs, persisted acknowledgements, bounded retries, idempotency, membership/block checks, cursor history and reconnect REST resynchronization. Sockets do not guarantee missed-event replay. Local messages currently have no real delivery receipts.
9. **Safety and support.** Wire `/users/me/blocks`, scoped reports, tickets, preferences and ratings. Use stable user IDs, not the display-name identifiers used by the fixtures. Apply blocks in both directions and preserve access to appeals/deletion for restricted identities. Define evidence visibility, moderation permissions, conflicts, limits and retention. Emergency-contact saving needs a supported service contract and consent; the preview never notifies contacts.
10. **Notifications and operations.** Connect read/delete/preference APIs. Decide whether browser push is a supported capability; request permission only after an explicit user action and honour denial. Scheduling, retries, expiry and deduplication remain server-owned. Maintenance/version displays must reflect actual server state. Validate operations reason notes and retries against current state; retain receipts.
11. **Privacy, deletion and launch.** Replace preview policy copy with reviewed operator-specific terms, support contacts, retention and deletion disclosures. Real deletion needs recent sign-in, queued job handling and honest pending/completed states. Confirm notification, location, UGC and account policies across all clients before release. Never show a successful real operation based on a local update alone.

## Mobile-app follow-up (documentation only)

- Preserve the existing Flutter directories and continue the staged Go-contract migration already recorded in the project docs. The website does not replace the app.
- Share semantic contracts, consent versions, assigned pool types, safety definitions, UTC bounds, group lifecycle and deep-link destinations. The UI implementations can differ.
- Configure validated web/app links for group, chat, support and account-deletion URLs, with an appropriate web fallback and access checks. Avoid exposing private data in link previews.
- Keep member readiness, personal pickups, route changes, message IDs, unread state and notification preferences consistent across devices. Revalidate after edits, leave, cancellation, account changes and reconnect.
- Provide real driver and safety verification. No client may supply its own trusted identity, admin role, or safety eligibility.
- Device-test maps, permission denial, background/reboot notifications, route links and account deletion during the existing final environment/emulator phase. No mobile code, manifests, app versions or backend code were changed for this website task.

## Design and performance decisions

The whole UI uses a pixel vocabulary, not just a decorative hero: pixel display type, stepped control silhouettes, generated avatars, grid maps, route marks, pixel charts, and sprite loaders. Manrope is retained for readable user content. All colors are literal black/white or equal-channel greys; fonts are self-hosted.

The dashboard city is deterministic code-native art with cached static geometry. Animation composites two tiny autorickshaws, caps rendering at 24 fps on desktop and 18 fps on narrow screens, and cancels its frame loop offscreen/in hidden tabs. The landing uses a small static code-native pixel street vignette. The hero has no moving scene, route diagram or input form. Layout and control feedback use brief transform/opacity transitions. Route code is lazy-loaded, large image/video assets are absent, and motion can be paused globally. OS reduced motion takes precedence. A short first-visit splash is skippable; route loading uses the transparent ORIX mark. Tab titles follow route/unread state, and the favicon uses the ORIX mark with light/dark contrast adaptation.

These measures reduce work; they do not prove smoothness on every device. Measure production Web Vitals and test representative low-end phones before launch. Automated accessibility scans do not replace keyboard, screen-reader, zoom, text-spacing, contrast-over-gradient and assistive-device testing.

Implementation references: [Vite](https://vite.dev/guide/), [Motion accessibility](https://motion.dev/docs/react-accessibility), [MDN reduced motion](https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/At-rules/@media/prefers-reduced-motion), [MDN fluid sizing](https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/Values/clamp). Exact dependency versions are pinned in `website/package-lock.json`.

Actual build and browser results are recorded separately in [website-verification.md](website-verification.md).

## Latest product and brand direction

Passengers submit starting point, destination, travel window and safety preferences. They do not choose auto, companion or lift mode or manually select a pool. Lift acceptance is opt-in; women-only eligibility must come from trusted server data. Drivers register their route, schedule, seats and availability separately. “Always active” in this preview means recurring daily within the registered time window until paused, not continuous 24-hour availability. Production recurrence needs an explicit timezone, DST policy, expiry and server-owned scheduling.

The existing Flutter/backend contracts still contain mode-specific controls and need a separate coordinated migration to this direction. No Flutter/backend changes were made in this website pass. The local matcher uses fixed fictional pools, endpoint distances and directed geometric lift overlap. It is not road routing or a verified server matching system; own lift registrations are local availability records, not live supply.

Following the user's later instruction to apply it throughout the web UI, the standalone outlined wordmark is now served as `website/public/orix-logo.svg`. The original reference remains in `Docs/assets/orix-logo-reference.svg`. Brand placeholders use the transparent ORIX mark; transport drawings remain only where they illustrate journeys. Login offers Google only, with a separate clearly labelled local preview entry; Google authentication remains unconnected.
