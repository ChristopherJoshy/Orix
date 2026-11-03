# Working memory

## Full objective and constraints

Complete the entire Python-to-Go backend rewrite; improve algorithms, matching,
architecture and notification reliability; remove Telegram completely; repair
fifteen-minute reminders; make Flutter a public travel app for shared autos,
travel partners and lift offers with maps/similar-route groups; complete reports,
moderation and practical app features; upgrade both stacks to current stable
libraries; clean readable code/comments; establish Google Play release readiness.

Backend completion comes first. Keep the existing structure, no backups or
parallel replacement folders; the owner already has backups. Project docs live
in Docs. Maintain this intentionally spelled memeory.md, history.md, root AGENTS.md
and corrected frontend/backend example.env. Remove Python/Telegram/duplicates only
AFTER the completed transition passes final testing, then revalidate the final tree.

**Testing gate:** finish all code changes first; the owner will supply environment
values afterward. Only then run checks/builds/tests/provider/database operations
and multi-account emulator/staging QA. Emulator/tools are installed. Do not start
further checks now. Source reading/editing and formatting are allowed. Historical
isolated test results are not evidence for later unverified implementation.

## Current architecture and status

Legacy Python remains pending verified cutover. Go source is in backend/main.go,
backend/app/*.go and backend/internal/matching. Go minimum/builder is 1.27.3 from
official current sources. Firebase Admin, Mongo v2, coder/websocket and x/image
versions are locked in go.mod/go.sum. Final full-stack latest-version/platform
verification remains. Existing frontend is Flutter, mostly legacy contracts.

Go is WRITTEN but not compiled/end-to-end verified for recent changes. It includes:

- Firebase revoked/verified-email identity, open registration, onboarding, own
  profile/terms/preferences/avatar, known-partner search and explicit CORS/limits.
- UTC auto/partner/lift ride create/edit/cancel/history/status/suggestions, one
  active assignment, all-member route/safety/capacity checks and transactional joins.
- Leased request/singleton matching, rotating bounded candidate scans, conservative
  versioned geographic bounds with legacy/dateline/pole/long-route fallback,
  projected/deduplicated profile loads and bounded member routes. Geographic
  performance/completeness still needs tests/explain/representative workloads.
- OSRM-compatible road matrix/legal stop ordering/pickup windows/final-leg detours,
  bounded caching and member-scoped private route geometry. Provider not exercised.
- Group readiness/arrival/start/finish/leave, expiry/slot release and transactional
  edits for unmatched or unconfirmed singleton trips. Arrival never auto-finishes.
  Singletons cannot confirm; migration resets older ready singleton flags.
- Member-only REST chat with retry IDs and paging; bounded authenticated sockets
  and Mongo outbox change-stream delivery. Client protocol/reconnect work remains.
- Inbox/preferences, stable outbox IDs/recipient receipts, leases/retries/token
  cleanup; independent thirty-second reminder scheduler with batched upserts and
  four delivery workers (one reserved for reminders), priority indexes, short
  Android TTL/APNs expiry/collapse, bounded sends and current-schedule/member checks.
- Reports/evidence, own history, audited review/warn/suspend/ban/restoration,
  two-way blocking and restricted-account support appeals; ratings and user/admin
  support tickets; service maintenance, version, stats/audit/queue operations.
- Shared transaction admin authorization touches actor state to conflict with
  concurrent role/deletion changes. Domain/self-action/last-admin rules remain.
- Recent-auth deletion queue with retryable Firebase/app cleanup and shared-data
  redaction, including avatars/support/evidence/receipt/location/origin/bootstrap
  references. Legacy/external retention still needs actual environment inventory.

Mongo replica set/sharded cluster, sessions/transactions/change streams required;
startup rejects standalone/nonwritable topology. /health is liveness; /readyz is
limited auth/Mongo readiness, not proof of provider or worker health. Trusted proxy
CIDRs optionally resolve forwarded chains; quotas remain per address/instance.
Go loads .env or ENV_FILE, process wins. Templates are Go-first; Python-only knobs
are marked. No credentials have been loaded/executed during this code phase.

## Operational code written, unrun

Go-only nonroot scratch Dockerfile/.dockerignore are staged; no image built.
/account-deletion embeds a Google web sign-in/explicit confirmation page using
Firebase web SDK 13.1.0 from official CDN with memory-only sessions. Requires real
same-project WEB identifiers, authorized host, support email/privacy URL and CORS
origin. No page was deployed or browser/provider-verified. Support email is a
user-initiated link; mailbox operation/ownership verification remain owner work.

support-account-deletion CLI plans read-only by default, validates explicit env/
database/actor/target/opaque case and an enabled Firebase/current app admin, then
requires an ownership-review assertion for apply. Uses shared cancellation/queue
logic, transactional case/audit, case conflict/retry handling and orphan identity
jobs without profile creation. Completion clears review metadata. No public bypass.

bootstrap-admin and staff role management are written. migrate-data plan version 4
includes UTC/mode/lifecycle, audit timestamp/log IDs and waiting-group search region
backfills, plus singleton normalization. Defaults read-only; apply requires explicit
database/stopped-writer assertions and source compare-and-set. Bounded to 50k rows
per collection, incremental, no backups. No bootstrap/migration/provisioning ran.

Local seed/remove commands support EIGHT deterministic Firebase Auth Emulator
personas: traveler-a/b/c, driver, safety, opposite, admin and newcomer @orix.example.
Require APP_ENV=local, demo-orix, loopback Mongo orix_internal_test, local API and
private password. Profiles use real auth/onboarding; fixed local admin promotion
is audited, newcomer skips fresh onboarding. Actual restrictions use moderation
workflows. Cleanup includes related outbox/jobs/groups and refuses nonseeded group
members. A debug-only named Firebase app switches emulator accounts; production
remains Google. Emulator password auth does not prove Google OAuth/FCM. Remove
internal users after QA. None created. See testing.md and deployment.md.

## Next implementation work

1. Backend API/worker/migration/operator code is now written; remaining backend
   integration gaps can surface while finishing the client and final QA. Fixed-
   scope retire-legacy-data now plans read-only and applies only after tested/
   stopped-writer assertions with audited batches. Actual inventory/retirement,
   retention decisions, old queue transition and all verification remain. Do not
   claim the port or all-mode fifteen-minute reliability is complete.
2. Finish frontend mode/map/suggestion/group lifecycle integration. Remove old
   rider/discovery/Telegram telemetry callers; wire REST/WS acknowledgement, token
   refresh/reconnect and resync. Repair authenticated images, preferences, reports,
   blocking/history/admin/support/appeals/ratings/recent-auth deletion flows.
3. Finish notification client integration: consume stable IDs/current server text,
   expiry/stale-group validation, avoid duplicate iOS alerts, remove fixed-ID/
   hardcoded-fifteen-minute behavior, and wire reliable navigation. Existing
   background/tap payload fixes are partial; device alarm behavior is unverified.
4. Complete current-stable frontend/platform libraries and lockfiles, remove fake
   store/insecure production defaults/placeholders, finalize public policy/retention,
   hosted deletion/manual recovery, UGC/moderation, Play Data safety and artifacts.
5. Only after ALL code changes: obtain private environment setup, run complete
   backend/frontend/static/race/contract/migration and multi-account emulator/staging
   tests, repair failures, audit every requirement. Then remove legacy Python,
   Telegram/duplicates, run final checks again, and assess actual release readiness.

Primary docs: backend-api.md, configuration.md, deployment.md, data-lifecycle.md,
notifications.md, testing.md, migration.md; chronology/actual results in history.md.

Latest backend slice (written, unrun): road-stop offset_seconds and private personal
pickup schedules; own my_pickup_time/my_pickup_summary + mode in group API; pickup-
time reminders for active/in-progress lifts, finished/removed/stale suppression,
versioned driver/pickup event identity and own payload/expiry. Join/edit/leave/
expiry/deletion invalidate schedules. Lift driver starts with all ready and driver
origin arrival; materially late start shifts pickups, sub-minute jitter retained.
Migration version 4 plans schedule backfill or flags unknown lift timing. Pure
schedule tests added but no compile/check/provider/DB/migration/emulator actions.
No live traffic/vehicle tracking; frontend adoption and final device/race QA pending.

Latest changes (unverified): distinct revoked-admin 403 mapping; post-QA read-only/
bounded legacy retirement CLI; initial Flutter Go-contract migration. Client origin
is validated, debug Go port8080, HTTPS profile/release required, old insecure host
removed; invalid builds stop before Firebase/network initialization. Numeric-offset
UTC parsing fixed. Group models carry mode/road/personal pickup/arrival/finished
fields; confirmation refetches, same-ID status changes refresh, leave uses POST.
API responses expose paging cursor; backend group includes departure_end. Frontend
mode/map/lifecycle/WS and feature UX remain incomplete. No checks/tests/runtime ran.

Latest client slice: WebSocket-FIRST chat per the user's follow-up, persisted ack
with eight-second bound and same-ID REST fallback/retry; nested live events,
load-earlier paging, no text-based matching, account/chat scopes and stale-response
suppression. Native Bearer/browser auth subprotocol, no query tokens, fresh-token
reconnect/resume/heartbeat; REST resync on reconnect. Group detail has explicit
readiness/arrival/start/finish/leave and own pickup UI; completed member omitted
from active lists. Catch reminder is current-group-validated/dismissible and never
claims arrival completes the ride. Per-group local IDs/current-minute text, stable
foreground IDs and iOS duplicate avoidance are written. Dart snap launcher failed;
direct installed SDK formatter succeeded. No analyzer/build/tests/provider/DB or
emulator runs. Next: home/matching/map flow still uses legacy contracts; finish
that, remaining notification/token/preferences/safety/account UI and stack/release
work before final environment/testing gate. User asked to continue quickly.


Skills setup (2026-11-02): 19 reviewed global Codex skills installed in
/home/chris/.codex/skills for premium UI, pixel art, generative art, Go,
Flutter/Dart, algorithms and security. See codex-skills.md for names, usage,
source-review limits and adaptations; codex-skills-manifest.json records commits
and 193 file hashes. All entrypoints validated and installed hashes matched;
three UI/UX Pro Max search smoke checks passed. Skills are discoverable next turn.
This task changed no app code and ran no application checks; the existing
implementation/environment/emulator testing gate and migration status remain.

Current continuation (2026-11-02): home/matching/create now use Go mode/suggestion/
join/edit/status contracts, including singleton suggestions and positive windows.
Own endpoint/member road maps are written; fake startup preloading removed.
Location picker still needs campus/autocomplete/provider cleanup. Chat is WS-first
with ack/idempotent HTTP fallback; added live-arrival vs older REST snapshot race
protection, reader-aware scrolling and background/covered-route polling pause.
Only source formatting ran, successfully. All current behavior remains untested.
Next: finish location picker, notification token/preferences/background handling,
remaining safety/account UI, dependency/lockfile upgrades and release cleanup;
then owner env + full backend/frontend/emulator QA. Preserve Python/duplicates
until transition testing succeeds. Eight Firebase-backed personas remain a QA
requirement; remove them after testing. No new runtime/DB accounts were created.

Latest progress (2026-11-02): replaced legacy picker with explicit backend place
search, map/coordinate selection and optional tap-triggered GPS; no campus
recommendations, startup permission request, public autocomplete or pan reverse
lookup. Existing endpoint initializes picker. Model operation IDs suppress stale
GPS/search; bounded meeting labels, selected-result credits and shared MapConfig
are written. Go /places/search uses optional private/contracted Nominatim-compatible
provider env (both backend env examples updated), no public OSMF origin, 1 in-flight/
1.1s pacing, 6s timeout, 256KiB/five-point bounds, 256-entry 24h memory cache.
No external provider configured/contacted. Source tests authored but not run;
Dart format/gofmt succeeded. Unused map_service.dart + cancellable dependency
remain until tested retirement. Legacy location_history preference is no longer
read/written; removal from existing devices remains a data-retention follow-up.
Next: notification token/background/preferences, safety/account UI completion,
stack/lockfile/release reconciliation, then owner env and full QA. Full goal active.

Local tooling (2026-11-02): installed all 21 jpoindexter/typography-skills
skills at commit 615006a142464d1a5abfa16304fbafa5e2ffef20 into
.agents/skills/, excluded by root .gitignore. Available on the next turn.
Application verification gate remains unchanged; no application checks run.


Website frontend (2026-11-02–03): new website/ React/TypeScript/Vite client,
separate from the unchanged Flutter and backend implementations. Entire design
uses grayscale pixel typography, sprite art, stepped controls and animated city
scenes. Landing/login, rider workspace, journeys/chat/safety/support/profile,
admin queues/roles/operations/audit, public policy and system-state pages are
implemented as an explicitly labelled browser-local preview. No real sign-in,
matching, chat delivery or admin authorization is connected. See website-
integration.md for backend/mobile work and website-verification.md/.json for
actual checks: production build, formatting, 36 routes at desktop and mobile
with zero reported axe violations/overflow, all 36 at 320px without overflow,
and live journey/chat/report/support/admin/motion interactions. Automated
contrast/focus checks retain limitations; physical-device QA remains. No
backend/mobile code or application test gate was changed. Next for the web:
user design review, then separately authorized API/auth/provider integration
and production/device/security validation.

2026-11-03: Push-token synchronization now tracks the Firebase identity as well
as the token, rejects stale responses after account changes, and retries when an
identity or token changes during a request. Backend token ownership across accounts
and sign-out cleanup remain outstanding. Dart formatting only; no tests run.

2026-11-03: Go profile updates now serialize ownership of a nonempty FCM token
through a SHA-256 keyed push_token_claims document and transaction, removing that
token from other profiles before assigning it. Claims retain owner ID and timestamp,
not the token itself. Account deletion removes claims owned by the deleted user. Migration/
retention review remains; sign-out unlink and multi-device semantics remain outstanding.
Source formatted only; no database, compiler, test, or emulator run.

2026-11-03: Added token-specific authenticated unregister on sign-out. The Go
transaction removes only the matching user's current token and matching claim;
Flutter attempts it before Firebase sign-out with a five-second bound. Offline
failure and concurrent registration can still leave a token registered, and the
existing model supports only one token per profile. Source formatted only; no
compiler, database, test or emulator checks ran.

2026-11-03: Sign-out now suppresses new client token-sync attempts and waits
up to 35 seconds for the current sync before attempting unregister. If that wait
times out, sign-out proceeds without claiming unregister succeeded; a request
may remain in flight. Offline/server-side delivery races and one-token-per-profile
limitations still require attention. Source formatting only; no tests run.

2026-11-03: Token rotation removes the previous owner-matching claim in the
same transaction; an FCM unregistered-token response now removes both the
matching profile token and claim, and retries if cleanup fails. Actual delivery
still supports one token per profile; multi-device registration and per-device
receipts remain open. Go source formatted; no tests or database checks run.

2026-11-03: Push delivery re-reads account status, preferences and the current
token immediately before sending, reducing stale-destination sends. An expired
pickup no longer stops processing later recipients. A change after the final
read or an already accepted push cannot be recalled. Formatting only; no tests,
provider or database checks ran. Multi-device delivery remains outstanding.

2026-11-03: Notification preferences screen now reads and writes the Go account
preference API, preserves server values on failure, and guards account switches.
Legacy automatic promotion-topic opt-in was removed; existing installations try
to unsubscribe at initialization. Device permission remains separate. Source
formatted only; no analyzer, runtime, provider or emulator checks ran.

2026-11-03: Retired the frontend rider-profile API calls and misleading active
rider status. The existing profile route now introduces a per-trip lift offer and
opens the current lift request form. Source formatted only; no tests run.

2026-11-03: Report-a-traveler now searches known group peers and submits the
Go moderation report contract with explicit category/target/details, instead
of creating a support ticket without the selected user. Group-specific evidence
selection and report history UI remain pending. Formatting only; no tests run.

2026-11-03: Added a paged account-scoped report history screen linked from
help/support, showing own report status and details. Source formatted only;
no analyzer, test, database or emulator checks ran.

2026-11-03: Group details can open a report preselecting another member and
including the group ID. The form tells users that recent group chat is included
for moderation; backend membership validation remains authoritative. Formatted
only; no tests or emulator checks ran.

2026-11-03: Group detail can block another member using the Go safety API,
with a confirmation explaining shared-group cancellation. A blocked-user
management/unblock screen remains to be wired. Formatting only; no tests run.

2026-11-03: Added blocked-traveler management and unblock confirmation via
the Go safety API. Block listings resolve display names with one bounded batch
lookup; deleted/missing profiles fall back to ID. Source formatting only;
no tests or database/emulator checks ran.

2026-11-03: Support ticket list uses Go cursor pagination, deduplicates moved
tickets by ID, refreshes from the server and guards account switches. Ticket
paths encode IDs; user-search query parameters are encoded. Source formatted only;
no tests, analyzer, database or emulator checks ran.

2026-11-03: Support ticket detail now offers the Go reopen action for closed
or resolved tickets, treats both statuses as read-only, and checks account
identity before applying ticket responses. Formatting only; no tests run.


2026-11-03 design revision: user rejected centered landing composition and
old hero art. Landing now uses full-width gutters, left-aligned copy beside
a large side-view autorickshaw SVG scene; sections/nav widened and overlapping
art captions removed. Build/format checks and targeted desktop/mobile visual,
axe and motion checks passed (see website-verification.md). Separate outlined
logo created at Docs/assets/orix-logo-reference.svg; user explicitly requested
not applying it, and it remains unused. Existing app/backend untouched.

2026-11-03: Go Firebase verification requires the verified token's Google
sign-in provider in production. Local demo-orix Auth emulator retains password
personas; production password/other-provider tokens are rejected. Flutter Google
server client ID is a build-time public setting rather than a hardcoded project
ID; configure the Firebase console/native files and GOOGLE_SERVER_CLIENT_ID as
needed. No provider, build, test or emulator checks ran; Google OAuth still
requires end-to-end QA. Firebase token claim reference: https://firebase.google.com/docs/rules/rules-and-auth

2026-11-03: Ride history now exposes the existing Go cursor pagination and
shows completed trips with a success status. Formatting only; tests deferred.

2026-11-03: Profile has an account deletion request flow with confirmation,
Google reauthentication, forced Firebase token refresh, Go deletion request and
sign-out after acceptance. Internal password-emulator reauthentication and web
Google reauthentication remain unsupported in this UI; use documented support
path during QA. No device/provider/emulator tests or builds ran.

2026-11-03: Flutter web Google sign-in and sensitive-action reauthentication
use Firebase popup APIs; deletion requests are guarded against concurrent
submissions. Popup configuration and account deletion remain untested. Local
password-emulator reauthentication is still not supported by this screen.
Official API reference: https://firebase.google.com/docs/auth/flutter/federated-auth

2026-11-03: The debug-only local Firebase Auth emulator deletion flow asks
for the test password and reauthenticates the current emulator account; the
password is not persisted. Production continues to require Google. Formatting
only; no emulator, provider or application checks ran.

2026-11-03: Flutter onboarding and profile link to operator-configured HTTPS
PRIVACY_POLICY_URL. Onboarding cannot accept policy consent without a configured
URL. Removed stale campus-rule and ad-consent claims from the terms screen. The
owner must publish and configure an accurate policy and review consent/terms; no
legal compliance is claimed. Formatting only; no app/device checks ran.

2026-11-03: Onboarding now sends explicit privacy acknowledgement and the Go
API requires and records it atomically with a timestamp. Local seed payloads
include the acknowledgement. Historical users are not backfilled; policy
versioning and legal review remain outstanding. Existing legacy test fixtures
need updating before the final test run. Formatting only; no tests run.

2026-11-03: Updated Go onboarding validation fixtures for privacy consent and
added missing/false acknowledgement cases. Test source formatted; no tests run.

2026-11-03: Onboarding accepts international-format phone numbers instead of
forcing +91 and removes an inaccurate OTP comment and unverified tracking/crash
claims. Formatting only; no runtime or emulator checks ran.

2026-11-03: Profile edit now checks backend save success before reporting it,
guards account changes after avatar upload, and retains a committed avatar if
the subsequent name update fails. Formatting only; no app checks ran.

2026-11-03: Profile provider guards asynchronous edits by Firebase identity and
backend user ID, clears errors before edits, and avoids updates after disposal.
Formatting only; no app tests or analyzer checks ran.

2026-11-03: The terms screen loads account acceptance status and provides an
explicit Go API acceptance action only for the displayed version 1.0; a server
version mismatch cannot be silently accepted. Published/legal terms review and
any enforcement of acceptance before app use remain outstanding. Formatting only;
no tests ran.

2026-11-03: Trip submission uses the selected date's local UTC offset instead
of today's offset. The form rejects nonexistent local times and windows spanning
an offset change because the Go contract accepts one offset. Ambiguous repeated
local times need device/backend design review; tests remain deferred.

2026-11-03: user approved the isolated ORIX symbol as the favicon. Replaced
public/favicon.svg, added light/dark contrast adaptation and cache-busted URL,
and removed the timer that overwrote it with an autorickshaw. Dynamic tab
titles remain. Standalone full wordmark SVG stays unused. Build passed.


2026-11-03 website design revision: passengers now submit one route/time request
with women-only and lift acceptance preferences; pool type is an assigned result.
Separate lift registration includes route, schedule, seats and daily-until-paused
availability. These are local preview behaviors, not real service integration.
Existing Flutter/backend mode contracts still need coordinated follow-up.
Google-only minimal login remains unconnected; preview entry is explicit.
The original outlined transparent wordmark is now applied throughout the web UI
following the user's later instruction; the ORIX mark remains the favicon.
The user rejected the route-form/diagram hero. Final hero returns to minimal
pixel typography, open space, one primary CTA and a small static street vignette.
Mobile dashboard overlap, form control visibility, sidebar scrolling and mobile
chat sizing were refined. See website-verification.md for this pass's checks.
No Flutter/backend tests or emulator QA were started.

2026-11-03: Removed the pause/play animation button from public and workspace
website headers at user request. Motion preference remains in Settings and OS
reduced-motion handling remains. Website production build passed.


2026-11-03 — Active app/web pixel design refactor. User requested entirely new
screen compositions, monochrome only, email/photo fuzzy lookup, private report
conversations, distinct lifecycle animations, and a visible Android emulator.
Later added autocomplete, pickups within 300 m and the same selected destination.
Shared theme/fonts/branding and many new app layouts plus web/report/matching
source changes are in progress. See Docs/design-refactor.md for exact scope,
implemented modules, verification order and remaining work. Website builds pass
at the autocomplete revision; Dart compile errors/warnings cleared at an earlier
revision (lint infos remain; later edits require reanalysis). One local web report
creation and fuzzy email lookup were browser-inspected. Visible Pixel_10_Pro
emulator booted, but app installation/full-screen QA is not complete. Android
build failed on Java 25/Gradle 8.14 mismatch; temporary compatible JDK setup is in
progress. Backend race/Python suites and protected emulator flows have NOT been
verified for these changes. Goal remains active; no completion/readiness claim.

2026-11-03: Restored the transparent Orix symbol above the web loading animation.
App loaders now retain the symbol too (compact indicators show the symbol within
their existing control constraints). Splash keeps its symbol with pixels beneath.
User explicitly requested restoring the new logo on both clients.

2026-11-03: Removed duplicate web loading road tracks, including the ground
line inside the loading sprite. Loading now shows the Orix mark and one compact
pixel-dot indicator. Splash has no extra road bar.

2026-11-03: Removed visible Loading text from the web route loader at user
request; the logo and single dot animation remain with an accessible status label.


2026-11-03 verification update: Go `GOCACHE=/tmp/orix-go-cache go test -race
./...` passed after updating an old suggestion fixture to the new 300 m/same
point rule. Initial sandbox run failed because local httptest listeners were
unavailable; the authorized run passed. Website build passed after loading bar
and caption fixes. Focused Flutter pixel-design tests passed (3 cases: ten
reduced-motion scene renders, logo preservation in full/compact loaders, 320 px
layout at 2× text with toggle interaction). Forty web route/viewport checks
passed with no document overflow/recovery screen; reporter→admin→reporter reply
round-trip verified in the fictional local preview; report contrast scan had no
violations. These do not prove production auth or DB behavior.

Java21 GitHub download checksum verified and extracted to /tmp/orix-jdk21.
Direct Gradle build uses that temporary JDK, not global Java config. Its live
process completed NDK28.2 and SDK36/34 installation and has advanced to Android
plugin build tasks. APK installation and authenticated emulator QA still pending.
Visible emulator rebooted after its earlier process exited; no user data wipe.
Python adapter/source parses only; runtime dependencies are absent, suite not run.
New matching panel and journey form have no passenger mode choice. Go generic
pool assignment respects lift opt-in, derives pool transport and retains road
checks. App standing daily lift availability remains unfinished; do not claim
all web/app behaviors or every screen are fully verified.


2026-11-03 — Firebase registration and continued layout source work:
Confirmed Android applicationId/namespace `com.orixridesharestudents.app` in
frontend/android/app/build.gradle.kts for the user's Firebase registration.
The Java21 Android build reached processDebugGoogleServices and then failed
because google-services.json is absent. It did not produce an installable APK.
The operator must download the registered Android client configuration to
frontend/android/app/google-services.json; the root .gitignore already excludes
that filename. No placeholder configuration or authentication bypass was added.

Continued shared Flutter source restructuring: OrixPage now uses scrollable
side navigation at window widths >=840 for its four-tab root pages, retaining
bottom navigation on narrower windows. Side labels can wrap and navigation
scrolls in short windows. Narrow pages use 16 px horizontal padding below360px.
Updated location search hints to describe automatic suggestions rather than
requiring Search. Corrected animation-loop documentation to include travelling.
These latest edits are unverified. Per the current user testing gate, no new
build, analysis, tests or emulator QA were started; wait until code changes and
operator environment setup are complete. Remaining recurring lift availability,
legacy parity, real client/backend integration and full visual QA stay pending.


2026-11-03 — Continued full-screen layout source work under the testing gate:
- Rebuilt location picker with a scrollable narrow layout and a search/map split
  at900px. Search results show full place labels, with provider attribution and
  explicit location/coordinate actions. One compact branded loader replaces the
  standard progress bar. The selection and confirmation stay in the scroll area
  to avoid a large fixed footer when the keyboard or enlarged text reduces space.
- Query edits clear canonical selection; selecting a suggestion or requesting
  GPS/manual coordinates cancels pending debounce. Submit cancels delayed search
  in shared autocomplete and email lookup too. Lookup loaders now use compact
  branded sizing; disabled email fields cannot launch a new request.
- Rebuilt pickup reminder using the shared page, a compact scene, grouped
  pickup time/place and scrollable actions; retained membership/time refresh
  checks, arrival API and notification dismissal behavior.
- Report conversation headings, status and failures now scroll with messages;
  the reply composer remains below the log. Shared Orix bubbles and original
  wordmark are used. Support conversation also rebuilt with the same structure,
  explicit refresh/retry/close/reopen and retained drafts on failed sends.
  Async support results are scoped to captured account identity and request
  version; refreshed details bypass the client GET cache.
- Dart formatting applied to edited files only. No analysis, builds, test runs,
  browser QA or emulator QA were started in this continuation. Latest source is
  unverified, and full completion is not claimed. Firebase Android configuration
  is still operator-provided; the last build failure is the missing
  google-services.json, not a successful APK build.


2026-11-03 — Group conversation source now shares the branded app bar,
constrained message log and compact multiline composer used by support/reports.
Moved title, connection state, errors and earlier-message action into the log
so they do not compete with the keyboard for fixed vertical space. Preserved
older-message scroll offset correction, polling/WebSocket fallback, per-message
retry and server canChat checks. Added a trip-details action and removed the
submit callback from the multiline field so newline entry is distinct from the
send button. Dart formatting only; rendering and runtime checks remain deferred.


2026-11-03 — Continued app/web interaction polish (source only):
Added a shared24px PixelButtonLoader with an accessible busy announcement and
original symbol. Applied it to login, onboarding, edit profile, journey submit,
ratings and support/report creation to avoid button height jumps from full-page
loaders. Corrected false empty states when journeys, groups, support tickets,
blocked contacts or notifications fail to load. Report/support/blocked lists
retain current same-account rows during refresh and replace the first page only
after success, reducing avoidable flicker without retaining another user's rows.

Web report conversation now follows incoming messages only near the bottom;
readers viewing older messages get a View new replies action. Sending a reply
resumes following, and changing report IDs clears the previous draft. Email picker
synchronizes externally selected contacts, recovers photos when the source URL
changes, and allows the selected contact/change action to wrap on narrow screens.
React source uses unconditional hooks, captured scroll refs and semantic buttons;
no new dependencies or production claims. Website still a fictional local demo.

Dart formatting applied to edited files. No analyzer, build, test, browser or
emulator checks were run in this continuation under the current testing gate.
Latest edits remain unverified. Remaining full integration, recurring lift
availability, legacy matching parity and complete rendered QA are still open.


2026-11-03 — Report follow-up notification source integration:
Go moderator replies now enqueue a report_reply event in the same Mongo
transaction as the message append. The existing idempotency branch returns
before event insertion on a retry. Event ID is derived from report, sender and
client message ID. Existing worker retry/lease/delivery-preference behavior is
reused; push text contains no report content or accused identity. Report resolved
events also carry report_id, and delivery maps report replies to report_action
with a private-conversation notification title/body and route identifier.

Flutter now retains scoped report/support tap identifiers from foreground local
notifications, background FCM taps and initial-message taps. Cold-start routing
waits for backend authentication/onboarding and account-provider binding, then
checks both backend user ID and Firebase UID. Invalid-account taps are discarded.
Pending ride reminders take priority. Added a guarded GoRouter report thread
route; report creation, report history and inbox use the same route. Foreground
local notification payload contains scoped identifiers only, not report text.

Dart formatting and gofmt applied. No analysis, build, test, browser or emulator
QA started under the current user testing gate. This integration is staged and
unverified; Python report-reply notification delivery still lacks equivalent
durable scheduling, and web remains a fictional local preview. Full design and
integration goal remains incomplete; do not infer runtime delivery from source.


2026-11-03 — Owner-supplied configuration, onboarding/tour and manual emulator pass:
Created ignored mode0600 backend/.env, frontend/.env and website/.env.local from
owner-provided connection/client values. No secret values recorded here. Selected
MongoDB database Orix-Prod, website https://orixrideshare.tech and supplied support
email. Moved Android google-services.json to android/app after checking package
and Firebase project; updated Android/Web Dart public Firebase options. Go local
API started for the owner's manual testing using these connections. This uses
real account data, not an auth bypass or internal-test account. Production DB
must not be used for automated test fixtures.

Implemented four-step app/web onboarding: phone/optional gender; clear-photo
upload/confirmation; lift interest; optional location/notification permissions;
safety/privacy acknowledgements and replayable pixel tour. App opens tour at
last onboarding step and offers it from Profile. Permissions moved out of startup.
Go/Python persist introduction preferences at first onboarding completion; clear
photo acknowledgement is not automated identity/face verification. Shared pixel
upload/uploaded scenes added. Contact/profile image requests now add bearer
headers only to the API avatar endpoint and keep thumbnails grayscale.

Website Google-only Firebase authentication now verifies with the actual backend;
removed public preview-entry buttons and all Take a look CTA text. Other journey,
community and operational preview adapters remain explicitly labelled and are
not claimed migrated. Admin routing requires the backend-verified admin role.
Android prompt/public download settings, admin release UI and server-only BFF
added. GridFS stores the current APK; pointer swap queues old blob/chunk deletion
with immediate/persistent retries. Admin upload ticket is single-use/short-lived,
file-scoped and size/digest checked. Uploads bypass Vercel body limits and report
actual progress. No release APK was published; release signing must be verified
by the administrator. No admin role was inferred from a support email.

Runtime correction: first installed build exposed NotificationService registered
with plain Provider. Fixed to ChangeNotifierProvider.value, rebuilt/installed and
confirmed a real emulator login frame. Latest debug APK rebuild passed and was
installed for owner testing. SHA certificate fingerprints were read from that
APK with apksigner and supplied in conversation (debug certificate only).

Small checks requested by owner: four Flutter checks passed (12 scenes in reduced
motion, logo loaders, narrow layout at2x, tour progression/close at320px and2x).
The new tour check first found a50px overflow; scrollable AlertDialog fixed it.
Five focused Go race tests passed for APK grant/store-link validation, report
participant/reply privacy and Geoapify adapter, using fixtures rather than cloud
DB. Website TypeScript/Vite build passed after final source edits. Changed Python
onboarding sources parse; dependencies/runtime suite not run. Complete emulator,
Google sign-in, onboarding upload, APK replacement and production integration QA
are left to the owner as requested, not claimed successful here.

Firebase JS13.1.0 verified against official release notes. Installation reported
four high transitive grpc findings; registry/release guidance checked and pinned
override grpc-js1.14.7 applied with lockfile. Patched install reports zero known
npm vulnerabilities. This does not establish overall application security.
Rotate passwords shared in chat before launch, provision managed Redis TLS and
deploy a real HTTPS API origin. See Docs/environment-setup.md for configuration,
public/admin distinction, release lifecycle and remaining deployment work.


2026-11-03 — Active goal expanded to complete app/API/web integration and realtime
sync. Added explicit Student/Teacher/Other onboarding and group member display.
Added atomic Go group rejection/requeue using existing request IDs, persistent
per-journey exclusions checked symmetrically in ranking/reservation, stale lease
reset, audit and notifications. Android and live web group controls call it.
Web safety, email/photo lookup, blocks, report history/thread and admin review
now use the actual API/BFF rather than fixture mutations. Added authenticated
web realtime recovery and app readiness/resync/inbox handling. Admin queue
participant privacy tightened. AGENTS.md updated to current small-test scope,
style and rejection invariants. See Docs/integration-progress.md for exact live
routes, tests and remaining adapters. Go race suite and website build passed;
9 focused Flutter tests passed; Dart had lint infos but no errors/warnings.
Python runtime and Mongo concurrency checks remain unverified. Goal remains active.

2026-11-03 — User temporarily prioritised the public landing redesign over the
remaining integration work. Rewrote Landing.tsx with isolated Landing.css and a
custom built-in image_gen Kerala street illustration in website/public/art/.
Wide editorial hero, original logo, grayscale pixel-inspired artwork, simpler
route-first explanation, comfort section, lift entry and native FAQ. Responsive
source layouts added; browser/build/emulator checks deferred under the supplied
current testing gate. See Docs/landing-art-direction.md. Full integration remains
unfinished and is the next workstream after this landing change.

2026-11-03 — Continued full integration and implemented new owner rules: no mock
product data/simulated success, no silent service substitution, bounded read retries,
curated nontechnical errors with restrained Kerala humour, monochrome pixel overlays.
Removed web demo matching/data/store and fake admin chart; connected journeys,
inbox, profile/settings, support/admin and one-time lifts to real API/BFF. Fixed
onboarding photo upload incorrectly requiring completed onboarding. Added owner-
scoped DB invalidations, profile/group/support sync, connection-loss scenes and
builder sections in both clients; research notes in Docs/builder-profile.md.
Go race suite, website build, three error-copy assertions and nine Flutter tests
passed; Dart has lint infos only. Updated debug APK built/installed/opened in visible
emulator; rebuilt API started (/health healthy). Public browser checks passed at
320/390/768/1440, plus enlarged text; offline/recovery scene exercised. No personal
Google sign-in or production fixture mutation automated. Recurring lifts, complete
pagination/deduplication, web push and isolated multi-account runtime verification
remain. See Docs/integration-progress.md for exact current state and limitations.

2026-11-03 — Owner paused broader verifications, then requested an illustration
phase with responsiveness checks only. Generated and shipped four approved-style
Kerala scenes with built-in image_gen: web chaya-stop/three companions, app ferry-
landing/shared umbrella. Distinct assets per platform. Reworked both logins;
placed secondary scenes in introductory onboarding/About. Preserved original logo,
working pixel animations and functional route art. Rejected hazy shelter drafts.
Web build and targeted browser responsive review passed; two Flutter design tests
passed at320/390/900 with1x/2x text. Design APK built/installed and actual login
visually inspected in visible emulator. No account interaction or business tests.
See Docs/illustration-design-phase.md for prompts/assets/review. Broader verification
stays paused; remaining implementation list in integration-progress.md still applies.

2026-11-03 — Began remaining implementation under the owner's broad verification
hold. Go support replies accept an optional client message ID and deduplicate it
per actor within the ticket transaction; reusing an ID for different text is
rejected. Web retains the same ID for retries of the same draft. Legacy clients
without IDs remain compatible; Android retry IDs and cross-restart persistence
remain to implement. No tests or builds run for this increment.
2026-11-03 — Android support replies now retain a cryptographically random
per-draft retry ID and send it to the Go deduplication contract. IDs reset on
account change, successful send or changed draft; app restart persistence and
legacy Python parity remain outstanding. No verification run under the hold.
2026-11-03 — Website notification inbox now follows server cursors beyond the
first 100 records, refreshes the requested pages after invalidations, deduplicates
rows and guards account switches. Broader verification remains paused; this code
has not yet been tested.
2026-11-03 — Web support list now follows authenticated user/admin cursors and
refreshes previously requested pages with account-switch checks. Source change
is unverified while the owner's broad testing hold remains in effect.
2026-11-03 — Web group chat now reads earlier messages using the server cursor,
reconstructs requested pages on realtime refresh, deduplicates message IDs, and
surfaces chat read failures with retry. This implementation is unverified under
the owner's current testing hold.

2026-11-03 — Website Vercel config added for client-side deep links with /api
excluded. Set Vercel project root to website/. Deployment verification remains
pending under the testing hold; production API origin and server-only admin secret
must be configured in Vercel before use.
2026-11-03 — Scoped website resource snapshots to account ID and request path at
render time, preventing a previous account's data appearing for one render after
an account or route change. No tests run under the verification hold.
2026-11-03 — Began recurring lift scheduling with a pure bounded local-calendar
calculation in backend/internal/liftschedule. It validates timezone, weekdays and
window, skips nonexistent local times, and returns UTC. This is not yet connected
to persistence, workers or clients; recurring lifts remain unavailable. No checks
run under the current verification hold.
2026-11-03 — Recurring lift calculations now skip windows spanning a timezone
offset transition because existing ride contracts store a single offset. Account
cleanup includes the planned recurring_lifts collection. Persistence, scheduling,
clients and verification remain incomplete; no test run under the hold.
2026-11-03 — Added initial Go recurring-offer storage and authenticated GET/PUT
routes; each account has at most one stored schedule, checked against current
profile and ride validation. Worker, client controls and verification remain
outstanding. This API is staged and must not be advertised as active scheduling.
2026-11-03 — Added a staged transactional recurring-lift materializer to the Go
matching worker. It checks the current account and maintenance state, observes the
one-active-journey limit, retries while an account is busy, and advances missed
occurrences. Client controls, runtime and concurrency verification remain pending;
no tests run under the verification hold.
2026-11-03 — Added website recurring lift route/day/time/seats/timezone form and
pause/resume controls, backed by the staged API. Existing one-time form remains.
The app controls, validation of live scheduling, races and deployment remain
unfinished. No tests run under the verification hold.
2026-11-03 — Added Android recurring lift route, weekday, timezone, time and seats
form with pause/resume against the staged API. Existing one-time flow retained.
No build or tests run under the owner verification hold; runtime and concurrency
correctness remain unproven.
2026-11-03 — User requested a single implementation pass, tests afterward, no
Docker workflow, organized utility scripts, refreshed READMEs and complete Go/
app/web/admin integration. Recurring-offer persistence now retains the last
materialized occurrence when editing or resuming, reducing duplicate departures.
Work remains incomplete; no tests run during this implementation pass.
2026-11-03 — Recurring-lift forms clear private route fields on account changes;
place-picker text also clears when an existing selection is removed. The worker
retains its last materialized departure across offer edits and pause/resume.
Implementation pass continues. No verification run yet by owner request.
2026-11-03 — Recurring-offer edits preserve the last created local calendar day
when calculating the next occurrence, preventing a time edit from scheduling a
second departure on that day. Maintenance now postpones scheduler rechecks rather
than repeatedly scanning the same offers. No tests run in the implementation pass.
2026-11-03 — Added owner-scoped recurring-lift realtime invalidations and app
refresh handling; edits retain their draft and refresh after completion. This is
unverified until the requested implementation pass finishes.
2026-11-03 — Recurring offers carry a server revision; saves check the last
loaded revision inside the transaction and reject stale updates. Web and Android
capture the edit revision so a background refresh cannot silently overwrite a
change from another device. This is staged and untested during implementation.
2026-11-03 — Added admin app-version controls through the allowlisted server-only
BFF; operators can edit current/minimum versions and store links after a review
prompt. Backend remains authoritative for validation and authorization. No tests
run during the implementation pass.
2026-11-03 — Researched official Firebase Web Messaging setup and added a staged,
account-scoped web push token registry separate from Android's single token.
Registration is bounded and serialized; account deletion includes the registry.
Delivery, browser registration, worker handling and tests remain outstanding.
2026-11-03 — Staged browser push delivery in the Go notification worker using
separate web tokens and per-device outbox checkpoints. Browser payloads contain
only an inbox identifier and type. Provider acceptance followed by a process crash
before checkpointing can still duplicate delivery; clients must deduplicate.
Browser worker and registration remain outstanding. No tests run in this pass.
2026-11-03 — Added opt-in browser push registration/settings, a data-only worker
with generic notification tags, and a public VAPID configuration placeholder.
Firebase setup guidance consulted. Sign-out/account-transfer handling and runtime
verification remain outstanding; no tests run in this implementation pass.
2026-11-03 — Browser push now unregisters on ordinary sign-out, restores an
explicitly opted-in registration for the same signed-in account, and listens for
foreground invalidations. Account deletion clears its local marker after server
acceptance. External account switches and stale server tokens still need review;
no testing during implementation pass.
2026-11-03 — Staged web push registration expiry after 30 days of inactivity and
local subscription retirement when Firebase auth changes outside normal sign-out.
This limits stale delivery after account switching; live browser/provider behavior
and sign-out failure recovery still require verification after implementation.
2026-11-03 — Browser push worker now rechecks account restrictions and notification
preferences immediately before each device delivery; documented public VAPID
configuration and at-least-once delivery limitations. No tests run in the
implementation pass.
2026-11-03 — Admin operations now shows recorded worker heartbeat and last-success
times alongside queue counts, with a safe message for failed work. These are raw
recorded timestamps, not a claim that workers are currently healthy. Untested in
this implementation pass.
2026-11-03 — Serialized browser subscription changes within a tab, recorded
subscription ownership before registration, and added an actionable retry for
failed restoration. Settings feedback is scoped to the signed-in account.
Cross-tab behavior and live push delivery still need verification; no tests run.
2026-11-03 — Editing an existing recurring offer now preserves its paused or
active state in both clients. Flutter also schedules deferred refresh after an
edit ends. Profile and settings transient state is cleared on account changes.
These implementation changes have not yet been tested.
2026-11-03 — Recurring-offer transactions now reset their candidate on MongoDB
callback retries, accept an exact replay of the immediately preceding edit,
and defer failed worker records so other offers can progress. Non-validation
ride creation failures no longer silently pause an offer. These changes require
focused and concurrent staging verification; no tests run in this pass.
2026-11-03 — Added account-scoped idempotency for support ticket creation across
Go, Flutter and web, and bound APK uploads to the initiating account. Release
settings can be refreshed after an uncertain upload; account-deletion sign-out
retires the local push subscription. Removed the unused Docker build files and
updated current deployment guidance for direct Go binaries. No tests run yet.
2026-11-03 — Browser push subscription changes now use browser-wide Web Locks;
new opt-in requires that capability. Revoked notification permission can be
retired without trapping sign-out, and queued auth changes are checked before
touching another account's subscription. Removed unused demo avatars. Added
focused schedule/replay regression checks; not yet executed.
2026-11-03 — Verification after the implementation batch: Go race suite passed
(local HTTP test sockets required sandbox escalation), website production build
and error-copy tests passed, focused Flutter analysis passed, and 12 Flutter
tour/notification/design/art tests passed. The new DST-gap test exposed a search
bound error, now corrected. These checks do not verify real multi-account Mongo,
Firebase/FCM delivery, provider services, release publication or deployed flows.
2026-11-03 — Rechecked the public landing in an isolated local browser at widths
320 and 1440: screenshots were reviewed and document width matched the viewport
at both sizes. The isolated MongoDB and Firebase Auth emulator services were not
available in this environment; no production database or authenticated account
was used. This check does not cover authenticated or device-specific layouts.
2026-11-03 — Reopened Pixel_10_Pro without wiping data, built the current Android
debug APK with the existing temporary JDK 21, installed it and observed the
Google-only sign-in screen. The first Flutter build selected Java 25 and failed
against Gradle 8.14; direct Gradle with JDK 21 succeeded. No sign-in or production
database operation was performed, and no API was started for the emulator.
2026-11-03 — Subsequently started a checksum-verified temporary MongoDB 8.0.34
replica set and Firebase Auth Emulator on localhost, provisioned eight local
personas and ran focused API checks. Concurrent recurring-offer save/pause/resume
replays produced one departure; another account had no schedule. Concurrent
support retries produced one ticket and changed-content replay conflicted. Stopped
the isolated services afterward. This does not verify Google sign-in, FCM, every
race, external providers or deployed behavior; no production data was used.
