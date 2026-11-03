# Orix app and web design refactor — in progress

## Accepted direction

The user requested new layouts for every Android screen, matching the website's
minimal monochrome pixel direction. Preserve the real authentication, safety,
moderation and trip lifecycle logic. Use the provided transparent wordmark in the
UI and symbol alone for launcher/splash/notification branding. App name: Orix.
Do not reuse the old screen compositions. The later request explicitly authorizes
emulator design testing and asks for a visible emulator window.

Additional requirements: email-based fuzzy contact lookup with photos; reports
begin with a reason/contact/first message and become private moderator
conversations; distinct pixel animations for loading, searching, found, no match,
lift offers, cancellation, travel, completion, report delivery and errors.

Latest route rule: show place suggestions while typing; selected places carry
coordinates and canonical IDs. Pickups must be within 300 m and destination must
be the same selected place. Coordinates must not contradict a matching ID.
Legacy coordinate-only destinations use a one-metre numerical tolerance, never
name similarity or a broad destination corridor. These rules also apply to lifts;
real road itinerary verification remains required.

## Implemented source work (not a completion claim)

- `frontend/lib/core/theme/app_theme.dart`: light grayscale tokens, local Manrope
  body fonts and Pixelify Sans headings, stepped borders, shared controls.
- `core/widgets/pixel_ui.dart`: OrixLogo, constrained OrixPage, PixelSection,
  four-tab navigation, stepped toggles, shared conversation bubbles and ten
  code-native pixel lifecycle scenes. Finite outcomes settle; waiting/travel
  loops respect reduced motion, route ticker state and application lifecycle.
- Rebuilt login, splash, onboarding, home, profile, edit profile, groups overview,
  group details, journey form/history, help hub, report creation/history/thread,
  blocked contacts, support creation/list, preferences, about, FAQ and system
  maintenance/restriction/update/offline layouts. Group and support chat use new
  stepped bubbles; the remaining complete-screen visual audit is still pending.
- Removed the portrait lock. Android launcher images, adaptive/monochrome vector,
  native splash and small notification icon use the symbol; app label stays Orix.
  Native notification colors remain subject to Android system rendering.
- `frontend/tool/generate_brand_assets.py` reproducibly draws density-specific
  launcher PNGs from the symbol geometry. Font licenses are bundled with fonts.
- App place autocomplete is debounced and invalidates a previous selection when
  the query changes. The map selector preserves place IDs; route maps show a
  300 m origin circle. Tiles and contact photos are grayscale.
- Web has a matching lifecycle animation family, outcome feedback tied to store
  changes, email/photo lookup, separate report conversation routes for reporter
  and admin, and place suggestions with canonical selection objects.
- Website remains explicitly a fictional local preview. Real Firebase sign-in,
  BFF/admin access and production APIs are not connected by these changes.
- Go has private report conversation read/reply endpoints, bounded messages,
  client message IDs for retry deduplication, sender/role derivation, conflict
  exclusion and closed-state checks. Python report conversations are also staged;
  Telegram report notification code was removed from that router.
- Contact lookup in Go/Python is bounded to previously encountered travel peers,
  ranks emails rather than names, supports substring/typo matching, and returns
  scoped profile/photo data. Mutations still use immutable user IDs.
- Go matching now checks all pickup pairs within 300 m, same destination and
  existing capacity/time/safety constraints. Generic `pool` requests can join
  compatible auto/companion/lift pools; lift inclusion requires opt-in. Assignment
  derives transport type and retains transactional road verification. Explicit
  legacy mode constraints remain for old clients. This needs full contract tests.
- Python matching/model source also carries the new distance/destination rule;
  legacy runtime parity and full group matching checks remain unverified.

## Provider configuration

Geoapify autocomplete was selected as an optional free-tier provider. Current
published quota is 3,000 credits/day, with attribution and rate limits. No account
was created and no private API key was invented or embedded.

Backend: `GEOCODING_BASE_URL=https://api.geoapify.com/v1/geocode`,
`GEOCODING_API_TOKEN`, `GEOCODING_ATTRIBUTION` and
`GEOCODING_ATTRIBUTION_URL=https://www.geoapify.com/`. The Go adapter supports this
format plus the existing contracted compatible provider. Public OSMF Nominatim
remains excluded. Aggregate quota provisioning across replicas is still required.

Web preview: optional origin-restricted public browser key
`VITE_GEOAPIFY_API_KEY`; without it, suggestions are explicitly labelled fictional
fixtures. Never put server/admin/service-account credentials into that variable.
Live provider execution is not verified without an operator key.

## Actual verification so far

- Multiple website TypeScript/Vite builds passed, including the autocomplete
  source revision. Full final route/responsiveness verification remains pending.
- Dart analysis passed compile errors/warnings at an intermediate revision; lint
  infos remained. Later edits need reanalysis. Initial analyzer sandbox cache
  failure was resolved by authorized execution with its normal cache.
- agent-browser in isolated session `orix-refactor` verified fuzzy `myaa` →
  `maya@example.com` with a visible fixture avatar, and report creation navigating
  to its local conversation. User/admin reply round-trip and responsive checks
  still need completion.
- Started the existing Pixel_10_Pro emulator headlessly, then stopped that instance
  at user request and reopened with a visible window. Boot completed. No AVD wipe
  or snapshot save was requested. The app has not yet been installed in this pass.
- First Android debug build failed: Android Studio bundles Java 25.0.4, incompatible
  with this project's Gradle 8.14. Flutter automatically added its documented
  `android.builtInKotlin=false` and `android.newDsl=false` compatibility flags in
  gradle.properties. No dependency or Gradle version was upgraded.
- Official Adoptium API returned HTTP 403. Official Temurin 21 GitHub metadata was
  retrieved; a checksum-verified temporary JDK download is in progress to build
  without changing global Java configuration. Preserve the live process handle
  in conversation continuity; do not restart a live download/build blindly.
- Meaningful Go tests for report privacy, spoofed replies, email ranking,
  Geoapify adapter, 300 m/same-destination rules and automatic lift opt-in have
  been added but not executed. No `go test -race`, Python suite, final widget
  suite or full authenticated emulator scenarios have passed in this revision.

## Remaining work and completion evidence

1. Finish complete-screen restructuring/audit, including notification center,
   support chat frame, location picker, reminders/dialogs and tablet navigation.
   Render every route at phone, narrow, landscape/tablet and enlarged text sizes.
2. Finish and verify recurring lift registration in the app/server to match web
   availability controls. Do not replace standing availability with a cosmetic
   toggle or a fake successful server response.
3. Finish Python place-adapter/parity and report conversation integration checks;
   verify report deletion/retention covers added messages and contact privacy.
4. Run required backend race tests and applicable Python checks once code work is
   ready under the testing gate. Validate actual Mongo transactions/authorization
   and concurrent membership/capacity rather than relying only on pure tests.
5. Build with a compatible JDK; install in the visible emulator. Use real emulator
   authentication/onboarding for protected flows, never an auth bypass. Existing
   internal test-account provisioning remains the route for four-plus personas.
6. Inspect all ten animations at initial/mid/final frames and reduced motion on
   both platforms. Check semantic meaning, grayscale, clipping and frame behavior.
7. Complete web report user/admin replies, closing behavior, block/unblock and
   matching exclusion, place-selection stale responses, and all responsive routes.
8. Record source hashes/screenshots and actual results. Do not call the goal
   complete until every requested screen and flow has current rendered evidence.

## Research used

- [Flutter rendering performance](https://docs.flutter.dev/perf/rendering-performance)
- [Flutter reduce motion](https://api.flutter.dev/flutter/dart-ui/AccessibilityFeatures/reduceMotion.html)
- [Android icon guidance](https://codelabs.developers.google.com/design-android-launcher)
- [Geoapify autocomplete](https://apidocs.geoapify.com/docs/geocoding/address-autocomplete/)
- [Geoapify pricing](https://www.geoapify.com/pricing/)
- [Gradle Java compatibility](https://docs.gradle.org/current/userguide/compatibility.html)
- [Adoptium download verification](https://adoptium.net/installation/archives)


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
