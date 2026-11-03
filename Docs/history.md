# Change history

## 2026-11-02

- Added root AGENTS.md and standalone Go matching domain with race-tested route,
  time, safety, duplicate-user and lift capacity constraints.
- Removed college-domain gates in backend verification and Google sign-in,
  domain list startup fetches, college-request endpoint and unsupported-domain UI.
  Updated login, FAQ and terms to describe public registration.
- Removed Python bot startup/shutdown, Telegram console logger initialization and
  webhook. Remaining Telegram services/source/dependencies still need removal.
- Created Docs, moved migration notes here, added memeory.md and configuration
  guidance; recorded user request to keep existing structure and create no backups.
- Added frontend/example.env and corrected backend/example.env; synchronized
  backend/.env.example. Fixed MongoDB setting names and disabled legacy bot settings
  by default. Frontend template uses compile-time API_BASE_URL.
- Verification: Go race tests passed in the earlier turn. Backend public auth tests:
  5 passed, 7 existing Pydantic deprecation warnings. Flutter analysis: no issues.
  Existing Flutter placeholder test: 1 passed. Full rewrite, real-device alarms,
  production integration and Play compliance remain unverified.

### Ride reminder delivery and taps

- Initialized local notifications in the Firebase background isolate with explicit
  Firebase options. Registered the background callback before permission handling.
- Unified RIDE_ALARM/ride_alarm recognition and JSON payload handling, retaining
  compatibility with delivered pipe payloads. Ordinary reminders and completed
  rides no longer trigger a departure alarm.
- Queued cold-start local taps, initial FCM messages and background FCM taps.
  Added live pending-alarm observation, auth/onboarding gating, navigator lookup,
  modal reentry protection and listener cleanup.
- Replaced full-screen intents with high-priority private ride reminders; removed
  lock-screen activity takeover and unused exact-alarm permission. Removed FCM
  token values from debug logs. Full Play audit is still pending.
- Verification: 5 Flutter tests passed (4 payload regressions and 1 existing
  placeholder); Flutter analysis clean. Device delivery, permission denial,
  background/battery behavior and server retry guarantees still need verification.

### Fifteen-minute departure requirement

- User clarified that obsolete Python and duplicate files must be removed after
  the Go transition is tested; frontend cleanup has the same requirement.
- Consolidated competing five-minute/zero-minute alarm jobs into the reminder
  scheduler. It runs every minute and alerts when a ride is fifteen minutes away,
  with catch-up until departure and no post-departure alerts.
- Fixed UTC/local calendar mismatch, removed the 100-group scan cap, deduplicated
  member IDs, and included schedule identity in reminder tracking.
- Fifteen-minute messages use the existing high-priority client reminder path.
- Verification: 14 backend auth/reminder tests passed (7 existing model deprecation
  warnings), Python compilation succeeded. Flutter checks recorded below.
- Remaining risks: notification tracker check/send is not atomic across replicas;
  FCM send failures are swallowed by legacy service. Durable outbox integration
  remains required. Cleanup and mark-here currently complete trips prematurely;
  replace that behavior with an explicit lifecycle in the full migration.
- Flutter verification: 5 tests passed; analysis reported no issues.

### Go HTTP foundation

- Added backend/main.go and backend/app/server.go in the existing structure.
  Ported GET /health and GET /api/v1/auth/config with public registration JSON.
- Added explicit CORS origins, bounded socket-peer rate limiting, body/header
  limits, connection timeouts, panic response handling and SIGTERM shutdown.
- Added HTTP contract, origin, quota, concurrent limiter expiry/capacity and
  forwarded-header bypass tests. No Firebase or persistence bypass routes added.
- Verification: `go test -race ./...` passed for HTTP and matching packages;
  `go vet ./...` passed. Python remains active until authenticated APIs,
  persistence, background jobs and frontend integration are ported and verified.

### Go sign-in and user persistence

- Added Firebase Admin token verification with revocation checks and verified
  email, accepting any email domain. Added POST /api/v1/auth/verify contract.
- Added MongoDB user repository with unique firebase_uid index, atomic upsert,
  existing-user IDs and generic storage failure responses. Banned and currently
  suspended accounts are rejected; absent auth dependencies return 503.
- Added real SDK initialization, configured-startup Mongo ping/index gate, bounded
  request contexts, service-account file/JSON/base64 JSON and ADC support.
- Resolved official Firebase Admin v4.22.0 and Mongo driver v2.9.2 using @latest;
  generated go.sum and tidied dependency declarations.
- Verification: Go race suite passed, including sign-in malformed/invalid token,
  public-email, account restriction and storage-error contracts. These use mock
  identity/storage services. Live Firebase/MongoDB and full frontend integration
  remain unverified; Python has not been removed.

### Go onboarding and current-user profile

- Ported POST /auth/onboard and GET/PUT /users/me beneath /api/v1. Added verified
  bearer authentication wrapper with account restriction checks and bounded context.
- Added MongoDB first-completion onboarding update with consent timestamp, phone
  and gender validation. Own profile responses preserve Flutter profile fields.
- Profile mutations allow only display name, HTTPS/uploaded photo path and push
  token, with size validation. Identity/role changes are rejected and tokens are
  not serialized in responses.
- Verification: Go race suite and go vet passed. Added anonymous/malformed bearer,
  profile response, allowlist/injection, invalid consent/contact, owned mutation
  and restricted account contract tests using mock identity/storage services.
  Inspected the corresponding Flutter provider contracts; live Mongo/Firebase and
  full device sign-in/onboarding remain unverified. Python remains active.

### Go ride request routes

- Ported create, list/history and owner-scoped pending cancellation. Preserved
  Flutter fields and added explicit auto/partner/lift modes and UTC route bounds.
- Validated required coordinates/offset, locations, calendars, time windows,
  zero-distance routes, mode, lift capacity and trusted profile gender.
- Added Mongo request ID/history indexes and unique active-user partial index;
  expired pending requests release their slot. Existing data conflicts fail startup.
- Verification: Go race suite and go vet passed. Added missing coordinate and
  client identity injection checks, time/UTC conversion, mode/capacity, date,
  safety preference and expiry validation regressions. Live Mongo concurrency and
  client integration remain unverified. Creation does not yet assign groups;
  matcher, status, lifecycle, outbox and map UI integration remain required.

### Final QA scheduling and internal accounts

- User requested deferring further checks until all code changes are complete;
  they will add environment values before full emulator testing.
- Recorded a required repeatable internal test-account provisioning command,
  with at least four accounts and a larger persona matrix for group/safety/admin
  coverage. Provisioning implementation and actual accounts remain pending.
- Added Docs/testing.md with timing, credentials boundaries and final coverage.

### Internal-account tooling

- Wrote local-only `seed-test-accounts` Go command for six deterministic Auth
  Emulator users, with backend API verification/onboarding and identifier-only
  output. Added runtime emulator guard requiring local demo-orix configuration.
- Added debug-only Flutter emulator authentication and account login dialog using
  a named demo Firebase app. Real auth verification remains required. Skipped FCM
  initialization in this local auth mode; staging push verification remains later.
- Updated both environment templates and testing instructions. No accounts were
  created and no compile/test/emulator checks ran, respecting the user's gate.

### Reservation and account lifecycle implementation

- Wrote transactional group creation/join for pending requests, checking ownership,
  current member routes/profiles, capacity, shared time and safety constraints.
  Group/request assignment and an outbox event commit together. Active groups
  cannot be changed by joining. Lift joining awaits verified road routing.
- Added local `remove-test-accounts` command and dedicated test-database guards.
  Recorded that local password identities do not verify real Google OAuth; final
  staging QA must cover Google sign-in, and test accounts are removed afterward.
- User confirmed emulator/tools are installed and requested finishing backend and
  all changes before testing. No compile, analysis, tests or emulator execution
  ran for these changes. Group endpoints and cleanup are unverified code.

### Go group reads and explicit trip lifecycle

- Added member-only group list/detail with batched profile reads and contact
  visibility tied to ready/active membership.
- Added transactional confirm, mark-here, start, finish and leave actions.
  Arrival no longer completes a trip. All-member readiness/presence gates start;
  all-member finish gates completion. Finished/cancelled members release slots.
- Preserved group history, reset consent on pre-trip membership changes and
  cancel groups when their driver leaves. Added transactional outbox events.
- Kept matched requests visible beyond search expiry until explicit lifecycle
  completion. No new checks or emulator runs; implementation remains unverified.

### Go chat persistence and paging

- Added member-only group chat read/send REST routes and strict input bounds.
- Added snapshot/transaction handling, append-only messages and transactional
  notification events. Group writes serialize sends with membership mutations.
- Added optional client-message retry deduplication and conflict handling, stable
  timestamp/ID cursor pagination, and supporting startup indexes.
- Kept private sender/deduplication fields out of other users' responses. Added
  CORS exposure for the next-page cursor. WebSocket and client send integration
  remain unfinished. No compile, tests, analysis or emulator runs performed.

### Go WebSocket implementation

- Wrote authenticated /ws chat frames/acknowledgements, bounded connections,
  messages and queues, idle/lifetime limits and generic errors.
- Added a Mongo change-stream bridge for committed chat/group outbox inserts,
  resolved recipients and fresh per-socket account/membership authorization.
- Connected background/socket cancellation to the server signal context. Added
  reconnect/REST resync requirements and query-token proxy logging guidance.
- No compile, analysis, tests or emulator execution ran. WebSocket delivery and
  client protocol integration remain unverified; durable push workers are pending.
- Downloaded and pinned github.com/coder/websocket v1.8.15 from the maintained
  module's latest release. No dependency code or validation commands were run.

### Go notification inbox, delivery and departure scheduling

- Wrote owner-scoped inbox/unread/read/delete APIs and notification/outbox indexes.
- Added worker leases, recipient receipts, deterministic inbox IDs, retry delay,
  retry cap and conditional invalid-token cleanup. Initialized FCM outside emulator
  mode; local auth mode stores notifications without sending push.
- Added per-schedule/per-member fifteen-minute reminder events for active groups,
  current-schedule/state rechecks, Android high-priority data and iOS sound alerts.
- No compile, checks, integration tests or live messages sent. Preference editing,
  failure monitoring/replay and client/device validation remain pending.

### Go reporting and moderation

- Wrote report creation/own history and authenticated admin pending/review routes.
  Captured bounded chat evidence only for valid shared-group context.
- Added moderator conflict checks, database-role/header authorization, transactional
  resolution/audit/restriction/group cancellations and reporter/warning events.
- Added report indexes and direct-recipient worker events. Go reporting does not
  call Telegram. User output excludes evidence and private moderation fields.
- No checks or service execution ran. Frontend reporting, blocking, abuse controls,
  escalation, retention and final integration/Play review remain unfinished.

### Blocking and server notification settings

- Wrote own block list/block/unblock APIs and two-way relationship lookup.
- Wired blocking into candidate suggestions, transactional joins, chat history/
  sends, WebSocket/push chat delivery and phone visibility. Blocking cancels shared
  open groups/reservations transactionally; unblocking does not restore them.
- Added boolean notification preference read/update APIs and worker suspension
  expiry handling; added block cleanup to the local test-account removal command.
- No checks, analysis or emulator runs. Frontend controls and final race/privacy/
  preference validation remain required.

### Go support tickets and staff responses

- Wrote owner-scoped ticket creation/list/detail/message/close/reopen, bounded input,
  per-user creation/open quotas and a bounded conversation without truncation.
- Added admin support queue/thread/reply/status endpoints with database-role/header
  authorization and fresh mutation checks. Staff replies use a support identity;
  actual staff actors are retained in private audit events.
- Added transactional ticket/audit/outbox writes, private support-update pushes and
  indexes. Lists omit conversation payloads while preserving client model shape.
- No checks or runtime/service calls ran. Admin frontend, retries/pagination,
  appeals, retention and final authorization/concurrency verification remain.

### Go ratings and app-version endpoints

- Wrote post-trip rating eligibility, unique partner/ride ratings and replay
  handling, pending partner/group reads, and scoped average queries with indexes.
- Added public version metadata and role/header protected admin updates with
  stable-version comparison and optional official HTTPS store links. Removed fake
  store defaults from the Go response.
- No checks or live operations ran. Rating/finish client flow, notifications,
  paging/retention, release metadata and final integration tests remain pending.

### Go account deletion workflow

- Wrote recent-authenticated deletion requests, durable jobs, immediate activity
  restriction and transactional shared-group/ride cancellation.
- Added retryable Firebase/app cleanup worker, known personal-record deletion/
  shared-content redaction, minimized job receipts and cleanup indexes.
- Kept deletion available to app-restricted users with valid recent Firebase auth.
  Documented remaining all-data/storage inventory, frontend/web paths and policy QA.
- No checks, service calls, account deletion or emulator runs performed.

### Road itinerary planning for lifts

- Wrote configured OSRM road matrix/geometry client and bounded legal stop-order
  search, with precedence, capacity, detour and snapping limits.
- Connected lift joins and driver-group creation to road planning. Persisted
  member-only geometry and invalidated plans on topology/privacy changes.
- Added ROUTING_BASE_URL to templates. No provider requests, compile/check/test or
  emulator operations ran. Arrival-window/exact-leg validation, route cache,
  provider setup/privacy and client map integration remain required.

### Lift pickup windows and bounded route caching

- Added pickup-window departure-range calculation in candidate search and final
  route validation. Final legs also recheck passenger detours and aggregate totals.
- Stored earliest/latest departure in plans and carried bounds into reservation,
  readiness activation and start timing. Lift candidates no longer require literal
  pickup windows to overlap before road travel time is considered.
- Added a two-minute/32-MB bounded response cache and activation-time stale-plan
  refresh. No checks, compilation, provider requests or emulator runs performed.

### Automatic route matching and waiting-group reconciliation

- Wrote leased, bounded pending-request matching with oldest-due ordering,
  candidate ranking and transactional reservation attempts before solo waiting.
- Added singleton group reconciliation so simultaneous waiting groups can merge
  without moving confirmed/multi-person groups. Source cancellation and request
  reassignment occur in the same transaction.
- Added due-work/candidate/identity indexes and group-origin metadata for deletion
  coverage after merges. No jobs/checks/services were run; implementation remains
  unverified and expiry/backfill/scale/failover work remains.

### Coordination expiry and worker operations

- Wrote transactional group expiry and slot release with cancellation events,
  separate from explicit trip completion. In-progress trips remain untouched.
- Added renewed singleton search bounds after departure of other members and
  lifecycle indexes for search/readiness/departure deadlines.
- Added admin queue/worker status, bounded failed-event listing, audited retry
  controls preserving recipient progress, and generic worker heartbeat/error state.
- No checks, background jobs, database/provider calls or emulator runs performed.

### Go profile avatar persistence

- Wrote bounded multipart JPEG/PNG/WebP decode/thumbnail re-encoding and atomic
  separate-collection image/profile update with versioned photo URL.
- Added owner/shared-group image authorization, block restrictions, private ETags,
  upload-only body allowance and own-avatar relative-path validation.
- Added avatar ownership index and account/test-account avatar cleanup. Authenticated
  image headers remain a frontend integration requirement. No checks or uploads ran.
- Resolved golang.org/x/image v0.47.0 via @latest for WebP decoding, with its
  required x/text, x/sync and x/sys updates recorded in go.mod/go.sum. No build or
  dependency compatibility checks were run during the testing gate.

### Scoped partner lookup and maintenance activity pause

- Wrote shared-history-only partner name search for reporting, with escaped prefix
  queries, bounded results, omitted emails and blocked-photo suppression.
- Added public/admin maintenance state and audited admin writes. Gated new ride
  activity/reservations and paused automatic matching while preserving safety,
  support, existing coordination and deletion paths.
- Consolidated working memory so superseded progress notes are not mistaken for
  current state. No checks, runtime calls or emulator runs performed.

### Explicit legacy-data migration tool

- Wrote default read-only migration planning for UTC bounds, passenger modes,
  expiry/terminal request states and missing group lifecycle/origin metadata.
- Added active-request, member-route, uniqueness and legacy TTL conflict reports;
  ambiguous data blocks application instead of choosing/deleting records.
- Explicit apply requires database selection/stopped-writer assertion and compares
  original source fields before incremental writes. No backups are created.
- Added environment/docs guidance. No migration reads, writes, checks or service
  calls ran; command and migration/cutover behavior remain unverified.

### Go environment setup and administrator provisioning

- Wrote nonexecuting .env/ENV_FILE loading with process precedence, strict keys/
  quoting and secret-free parse errors for server and tool entry points.
- Extracted shared Firebase initialization and wrote explicitly selected first-admin
  bootstrap with verified UID, local/cloud guards, singleton transaction and audit.
- Added authenticated staff-role mutation with last-admin protection and exposed
  server-derived roles only in the caller's own auth/profile response.
- Updated canonical templates/configuration guidance. No credentials were loaded,
  commands executed, roles changed or checks run. Recovery and client admin setup
  still need completion and final verification.

### Owner-scoped ride editing

- Wrote PUT /rides/{request_id} for pending/matching requests and unconfirmed
  singleton waiting groups, preserving identity while resetting matching work.
- Added transactional route/time/mode/safety revalidation, lift plan replacement,
  group metadata/expiry updates and group update events.
- Shared/confirmed/started/expired requests cannot be silently changed. No checks,
  routing requests or database writes ran; client editing and final race tests remain.

### Restricted-account appeal and restore APIs

- Added support-only authenticated access for app-banned/suspended identities,
  with fresh appeal-type restriction enforcement and existing owner/quota controls.
- Kept ordinary APIs and deleting identities restricted. Added audited admin restore
  with review notes, conflict checks and no resurrection of cancelled rides.
- Added generic account-restored notification content. No checks, submissions,
  restores, database/provider calls or emulator operations ran.

### Cursor paging for main histories and queues

- Added scoped timestamp/ID cursors, bounded limits and limit+1 next-page detection
  while preserving existing JSON bodies and X-Next-Cursor transport.
- Wired ride/group/inbox/report/ticket lists and supporting indexes. Kept own/admin
  filters, unpaged unread counts, lightweight ticket lists and detail/internal calls.
- No checks or data queries ran. Client load-more/deduplication and final cursor,
  access-control and concurrent-update verification remain pending.

## 2026-11-02 — Complete local test personas and cleanup scopes

- Added admin and newcomer to the existing six Firebase Auth Emulator personas.
  All identities authenticate through Firebase and API verification; travel/admin
  profiles use normal onboarding. Only the fixed local admin UID is promoted in
  an audited transaction inside the isolated test database. No accounts created.
- Newcomer skips initial onboarding; reruns preserve existing profile state and
  never silently restore restrictions. Documented actual suspend/ban/report/
  appeal/restore scenarios instead of fabricating client-only restricted states.
- Expanded removal to outbox target/actor/group events, queued deletion jobs and
  origin-owned emptied groups. Retain group records until dependent cleanup has
  succeeded so interrupted cleanup can rediscover them.
- Frontend ride mode fields and selection UI are drafted; not fully integrated.
  Backend completion remains the priority before client/emulator testing.
- Formatted edited Go command files only. No compilation, checks, provisioning,
  database calls or emulator tests were run under the user's code-first gate.

### Matching fairness and remaining backend API parity

- Added persisted candidate cursor rotation for background matching/singleton
  reconciliation. Bounded batches advance past unsuitable groups and wrap; edits
  reset progress. Added mode, capacity and non-lift UTC-window query pruning plus
  a supporting index. Final reservation continues to revalidate all members.
- Ported terms-status/acceptance, enforcing the current displayed version and
  preserving original acceptance timestamps on retries with transactional audit.
- Added typed admin lookup/group summaries, bounded keyset audit summaries and
  aggregate counts. Added note-required, transactionally authorized group cancel
  with request-slot release, outbox notification and audit in the same commit.
- Added legacy audit timestamp backfill/conflict planning to migration version 2.
  Replaced misleading migration document opening with current status; added a
  consolidated route/replacement inventory in Docs/backend-api.md.
- Source edits and Go formatting only. No builds, checks, tests, database writes,
  migrations, account provisioning or emulator sessions ran. Backend/client
  cutover, final cleanup and all requested verification remain unfinished.

## 2026-11-02 — Deployment trust and deletion inventory

- Added explicit trusted-proxy CIDRs with bounded right-to-left forwarded-chain
  resolution, while keeping socket peers as the default trust boundary.
- Added generic /readyz and startup checks for writable transactional Mongo
  topology. Probe routes remain exempt from request quotas; no service started.
- Fixed Go audit insertion compatibility with legacy unique log_id indexes through
  one API/CLI writer providing unique IDs and both timestamp fields. Extended the
  migration planner to backfill missing log IDs and flag planned collisions.
- Expanded account deletion for support/evidence/receipt/shared-location/origin
  references and own bootstrap marker; exposed deferred cleanup failures to worker
  health. Documented active, legacy and external data deletion/retention boundaries.
- Made backend example.env/.env.example Go-first with an explicit replica-set URI;
  updated configuration, migration and working-memory documents.
- Previous goal turn made source/documentation progress. This turn also changed
  source/docs only, with Go formatting. No compilation, test/check commands, DB
  connections, provider calls, account provisioning or emulator QA ran.

## 2026-11-02 — Public deletion code and Go container definition

- Prior goal turn made source/docs progress. Continued backend implementation with
  an embedded public Google-authenticated deletion page and explicit account/
  destructive-action confirmation, using the existing recent-auth deletion queue.
- Added public-only config projection, memory-only browser sessions, bounded request
  handling, clear queued/uncertain outcomes, CSP/framing/referrer/popup headers and
  configurable monitored support/privacy links. No bypass or automatic email.
- Added validated Firebase WEB/contact/policy environment settings and templates.
  Deployment/provider/manual recovery setup and real browser QA remain pending.
- Added Go-only multi-stage Dockerfile and allowlisted build context. Updated the
  minimum/builder to Go 1.27.3 from official release/container sources. Pinned public
  web Firebase Auth modules to official SDK 13.1.0. Documented staging/cutover QA.
- Browsed official implementation/release/policy documentation only. No compile,
  tests/checks, image build, services, DB/provider actions, mail or emulator runs.
  Full backend/client/release verification and Python/Telegram removal remain gated.

## 2026-11-02 — Operator workflow for externally requested deletion

- Prior goal turn made backend/web/container source and documentation progress.
  Added CLI-only support-account-deletion with default read-only planning, explicit
  environment/database/case selection and privileged Firebase/app-admin checks.
- Apply requires an explicit operator ownership-review assertion. Fresh role and
  identity checks are retained; no unauthenticated API or account-ownership bypass.
- Shared app/web cancellation and deletion queue logic with support application;
  added transactional case attachment/audit, retry/conflict behavior, orphan
  identity-only jobs and review metadata clearing on completion.
- Updated backend env examples, deployment/ownership procedure, data lifecycle,
  testing plan, API inventory, migration record and working memory.
- Source edits and formatting only. No commands were run against services, no
  plans/applies/identities/jobs created, and no compile/check/test/emulator QA ran.

### Consolidate administrator transaction authorization

- Replaced repeated snapshot-only role checks with a shared check/profile write,
  so concurrent role revocation/deletion conflicts with the transaction.
- Applied across moderation/restoration/staff/support, operational controls,
  delivery retry, group cancellation and operator support deletion. Kept domain
  conflicts and last-admin rules in their modules; read-only middleware unchanged.
- Updated deferred race scenarios and continuity docs. Source reading/edits and
  formatting only; no builds, checks, service/DB/provider operations or tests ran.

## 2026-11-02 — Conservative route filtering and matching data bounds

- Previous goal turn made backend source/docs progress. Added coarse spherical
  corridor regions and indexed overlap filtering with legacy/version fallbacks.
  Wired group join/create, singleton edits, leave invalidation and deletion cleanup.
- Extended migration plan to version 3 with waiting-group region backfill/conflicts.
  Reduced candidate/member/profile projections, deduplicated profile IDs, bounded
  route loads, and rechecked member onboarding. Added explicit unique user-ID index.
- Retained background match errors in health/retry state, cleared stale hints after
  success and kept singleton work eligible despite failures in the request pass.
- Added meaningful geometry test source and deferred workload/index/race scenarios.
  Updated migration, API, privacy-data, testing and working-memory records.
- Source reading/edits and formatting only. No compile, test/check, benchmarks,
  database/provider requests, migrations, explain plans or emulator runs occurred.

## 2026-11-02 — Reminder scheduling and waiting-group readiness

- Previous goal turn made matching/source/documentation progress. Refused singleton
  confirmation and elapsed windows; staged legacy singleton readiness normalization.
- Separated batched reminder scheduling from four bounded leased delivery workers,
  with a reserved reminder worker and priority/index ordering for the general pool.
- Added per-recipient stale schedule/member checks, ten-second push bounds, short
  reminder TTL/APNs expiry/collapse and stable payload IDs/server title/body.
  Moved expired lease-limit recovery into scheduling and throttled heartbeat writes.
- Updated notification/migration/test documentation and replaced stale campus/bot
  README narratives with current entry points. Consolidated working memory below
  around actual state and remaining work, retaining chronology in this history.
- Consulted official FCM lifespan/Go SDK documentation. No builds, tests/checks,
  runtime queues/provider requests, migration, benchmark or emulator QA ran.
  Lift passenger pickup-time reminders remain a known backend implementation gap.

## 2026-11-02 — Planned pickup reminders per lift traveler

- Previous goal turn made notification/lifecycle/docs progress. Added cumulative
  road-stop offsets, private member pickup schedules on confirmation and own-only
  pickup metadata/mode in group responses. Missing lift offsets require replan/
  explicit migration review, with no guessed passenger departure.
- Updated scheduler/delivery/expiry/IDs for personal pickup times, including lift
  passengers after driver start. Suppressed closed/finished/removed or rescheduled
  work and invalidated schedules across membership/edit/expiry/deletion changes.
- Made lift start driver-controlled with readiness and driver-origin arrival;
  later passengers can wait at their own stops. Materially late starts shift the
  plan; sub-minute jitter remains within the original confirmed schedule.
- Extended migration planner to version 4 and added meaningful timing/staleness/
  identity test source. Updated API, data, notification, migration and QA docs.
- Source edits/formatting only. No compile, test/check, routing/provider/database
  requests, migration or emulator runs. Client integration and real timing remain
  unverified; planned pickup estimates are not live traffic/vehicle tracking.

## 2026-11-02 — Retirement tooling and initial client contract migration

- Previous goal turn made personal pickup/lifecycle/source/docs progress. Added
  read-only retirement planning with fixed bot/domain record/profile-field scopes,
  explicit post-QA/stopped-writer assertions and bounded authorized/audited apply.
  No plan/apply, source deletion, backup or provider/data operations occurred.
- Separated revoked-admin 403 errors from review conflicts and provider failures.
- Began Flutter Go-contract integration: validated HTTPS release origin/debug Go
  port; removed the hardcoded HTTP endpoint; handled invalid build configuration
  before provider initialization; corrected numeric-offset timestamp parsing.
- Added mode/road/pickup/arrival/completion model data, full readiness refetch,
  unchanged-ID status refresh, POST leave, departure_end and cursor propagation.
- Updated env templates, operator/deletion/config/API/migration/testing/history and
  continuity docs. Source edits and Go formatting only; no compile, analyzer,
  tests/checks, cloud/database commands, retirement or emulator sessions ran.

## 2026-11-02 — Frontend group coordination and WebSocket-first chat

- Previous goal turn made backend retirement/client-contract progress. Rebuilt
  account-scoped group/chat state and stale-response guards, added complete group
  action controls, personal pickup UI and finished-member active/history handling.
- User explicitly requested WebSocket chat and faster continuation. Implemented
  primary socket sends with persisted ack, bounded wait and same-ID REST fallback;
  stable retry/deduplication, pending/unconfirmed UI, history paging and reconnect
  resynchronization. Messages are no longer matched by text or assumed sent.
- Added native Bearer/browser auth-subprotocol transports, refreshed-token bounded
  reconnect, heartbeat/resume and generation-safe teardown. Removed token-URL auth
  from Go and client. Scoped auth verification/fallback against account changes.
- Replaced misleading forced alarm screen with refreshed/dismissible pickup UI and
  arrival-only action. Added per-group local IDs, current minutes, stable foreground
  IDs and iOS APNs/local duplicate avoidance. Full notification work remains.
- The snap dart formatter failed before editing; direct installed SDK formatting
  then succeeded for the changed files. Consulted official socket transport/server
  API docs. No analyzer/build/test/check, service, provider/database or emulator QA
  ran. Broad client/release integration and final transition verification remain.


## 2026-11-02 — Reviewed global Codex skills installation

- Researched official/popular design and engineering skills; downloaded eight
  candidate repositories at fixed commits and selected 17 upstream packages.
- Added local instruction-only pixel-art-web and algorithm-design skills, for
  19 installed skills total in /home/chris/.codex/skills. Used bundled installer
  validation/copy helpers with no overwrite, hooks, package installs or services.
- Adapted Codex script paths and template branding, vendored pinned Vercel
  guidelines, and retained licenses. Recorded source commits, modifications,
  193 file hashes, review limits and usage in codex-skills.md and JSON artifacts.
- Actual verification: 19/19 metadata validations, 193/193 matching installed
  hashes, zero symlinks, three successful UI/UX Pro Max JSON smoke checks
  (UX, Flutter, design system). No ORIX application checks or emulator QA ran;
  existing user testing gate preserved. No malware guarantee is asserted.

## 2026-11-02 — WebSocket chat and public matching integration

- Chat uses persisted WebSocket acknowledgements, stable retry IDs and bounded
  HTTP fallback. Invalid socket messages now include the correlation ID; socket
  sends require an idempotency ID. New live/ack arrivals survive an older HTTP
  snapshot finishing afterward. The reader follows new arrivals when near the
  end, preserves reading position in older history, and pauses fallback polling
  while backgrounded or covered by another route.
- Home/matching now expose auto, partner and lift requests, compatible groups,
  join/edit/cancel actions and account-scoped own status. Readiness socket events
  trigger refresh; unconfirmed singleton requests retain suggestion access.
  Status includes the own request, group ID/status/size. Mutation-start invalidation
  prevents older status reads overwriting edited/cancelled requests.
- Trip creation sends a positive pickup window rather than identical timestamps,
  supports existing-request edits and validates selected coordinates. Own endpoint
  and member road-plan previews use flutter_map with visible attribution and
  cached default tiles. Removed fake map preloading from app startup; location
  picker replacement remains pending. Store links require explicit official HTTPS
  URLs instead of invented fallback listings.
- Actual tooling: direct installed Dart SDK formatted touched source successfully;
  gofmt formatted Go edits. No analyzer, compile, test, emulator, provider or
  database runs. Owner's finish-code-first gate remains in effect.

## 2026-11-02 — Public location picker and configured place search

- Applied the Flutter architecture skill: immutable place models, network/GPS
  service and transient picker provider separate from map UI; existing directory
  structure retained. Source refactor replaces campus defaults/recommendations,
  debounced public Nominatim autocomplete/reverse lookup, fake preload and
  satellite/label overlays with explicit search and cached shared tile setup.
- Added authenticated Go provider adapter with optional operator-selected
  Nominatim-compatible origin/token/attribution, bounded upstream concurrency,
  response/time limits, 256-entry memory cache and manual-selection fallback.
  Public OSMF origin is rejected; no direct public geocoder fallback exists.
- Picker supports map pin, coordinates, tap-triggered GPS, existing-endpoint
  initialization, editable bounded labels, stale-operation guards and attribution
  for selected results. It no longer reads or writes unscoped location history.
- Actual tooling: direct Dart SDK format and gofmt succeeded. Two source-edit
  scripts stopped on text-anchor mismatches; subsequent edits completed the
  affected map/attribution changes. No analyzer/build/tests/emulator/provider/DB
  runs. Backend test source is written but unexecuted. Existing user gate remains.

- Official geolocator LocationSettings and Nominatim search documentation were
  consulted for timeout and JSON/search contract details. This was documentation
  browsing, not an application/provider API execution.

### 2026-11-02 — Local typography skills

- Installed all 21 skills from jpoindexter/typography-skills, pinned to
  615006a142464d1a5abfa16304fbafa5e2ffef20, using the skill-installer helper.
- Added /.agents/skills/ to root .gitignore. Installer reported success for
  every skill. No application tests or emulator QA run.


## 2026-11-02–03 — Monochrome pixel-art website frontend

- Added website/ with a distinct pixel-art design, generated isometric city and
  autorickshaw sprites, self-hosted fonts, grayscale gradients, responsive app
  and admin layouts, splash/loading/tab feedback and motion controls.
- Implemented explicit fictional local workflows for journeys, readiness,
  chat, notifications, profiles, driver setup, safety/reports/blocks, support,
  ratings, moderation, member access/roles, operations and activity export.
- Documented actual API/auth/admin-secret boundary, places/road routing,
  persistence/WebSocket/notification requirements and mobile follow-up in
  website-integration.md. No app/backend source or configuration changed.
- Actual website checks: strict TypeScript/Vite production build and Prettier
  check passed; 36 routes each at 1440px and 390px loaded with zero automated
  axe violations, error screens or document overflow; 36 routes at 320px had
  no document overflow. Inspected screenshots and tested local journey, chat,
  reporting/admin, support, restoration, drawer, motion and reset flows.
- Corrected contrast, mobile heading/landmark/focus, table/native-control
  overflow and text-spacing issues found during QA. Gradient/modal automated
  checks retain incomplete cases; broader assistive/device QA is pending.
- Production localhost smoke reached landing and workspace, observed favicon
  and title changes, and reported no runtime errors. One local Web Vitals
  reading is recorded with limits in website-verification.md/.json. No
  deployment, backend/mobile test run, provider call or emulator QA performed.
- Final immutable-build pass: all 36 production routes at both 1440px and
  390px had zero reported axe violations, document overflow or error screens.
  Verified production sign-out, unconnected sign-in disclosure, protected
  preview redirect and explicit re-entry; recorded source hashes.

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


## 2026-11-03 — Wider landing and rebuilt hero art

- Responded to user screenshots: broadened landing/nav/section gutters and
  changed the centered hero to left-hand copy with right-hand pixel art.
- Added HeroJourney.tsx: a larger side-on autorickshaw, passenger, palms and
  skyline with road/wheel/suspension motion, offscreen/tab pause and reduced
  motion support. Removed old hero city and overlapping captions.
- Created standalone outlined Docs/assets/orix-logo-reference.svg; validated
  XML and absence of font/image dependencies. It is not used by the site.
- Actual checks: strict production build and targeted formatting passed;
  desktop/mobile screenshots inspected at 2044/1440/390/320 px; narrow layouts
  had no document overflow, inspected axe scans had zero violations with
  remaining gradient contrast incomplete cases; motion controls verified.
  Original full-route suite retained as historical evidence.

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

## 2026-11-03 — ORIX brand favicon

- Applied the supplied pixel symbol as a crisp SVG favicon with transparent
  background and light/dark foreground adaptation. Preserved the full logo SVG
  as an unused separate asset. Removed the old favicon replacement timer while
  retaining route/unread tab titles. TypeScript/Vite production build passed.


2026-11-03 — Website visual/product refinement. Replaced mode choice with a
single passenger request and assigned pool result; separate lift route/schedule
registration supports daily availability until paused. Google-only minimal login
remains an honest unconnected preview. Applied original transparent wordmark
throughout web branding. Simplified the hero after user rejected the form/diagram
version: pixel headline, open space, primary action, static small street vignette.
Fixed mobile dashboard overlap, controls, sidebar scrolling, caption contrast,
narrow headline wrapping and chat sizing. Final build and formatting passed.
agent-browser swept 36 routes at four widths (144 checks), then rechecked the
final affected landing/chat routes at four widths with zero automated audit
violations; no document overflow or recovery screens. See website-verification.md
for ordering and limitations. No Flutter/backend checks, emulator QA or deployment.

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

2026-11-03: Final group-tap APK build passed (6s), installed and reopened in the
visible emulator. Current integration API compiled and started on8080. Final
website build passed after accurate live-route/preview labels. Owner can test
these builds; complete cross-account/runtime integration is not yet claimed.

2026-11-03 — Replaced the public landing composition and generated a transparent
Kerala street-corner hero asset with built-in image_gen. Removed the old hero,
orbital avatars, repeated cards and filler copy. Added scoped monochrome styling,
responsive layouts, native FAQ disclosures and clear route/comfort/lift content.
Inspected the source and generated artwork; no build, tests or browser/emulator QA
executed for this change under the user-supplied current testing gate. Not deployed.

2026-11-03 — Replaced remaining website demo runtime with authenticated service
adapters for journeys/history/matching, inbox, account/preferences, support and
admin lookup/operations/audit; one-time lifts now create real driver requests.
Deleted simulated matching and fictional records; retained UI motion preference
only. Added owned historical ride GET, DB-level owner-scoped invalidations and
atomic notification-deletion invalidation. Fixed onboarding avatar upload gate.
Added builder bio/links and edited quote, safe error mapping, bounded read retries,
connection-loss pixel scenes and shared monochrome overlay styling. AGENTS.md
records no-mocks/retry and safe-error/overlay rules.

Verification: latest website build passed; full Go race suite passed after local
socket permissions/review timeout retry; three error-copy assertions and nine
Flutter tests passed; focused Dart analysis no errors/warnings, style infos remain.
Android debug build/install/launch passed, API /health returned healthy2.1.0.
Public browser inspection: no overflow at320/390/768/1440 or1440 with200% root text;
Google-only login rendered; offline scene and online recovery worked. No real
account sign-in/deletion, moderator mutations, production test data or full cross-
account integration QA performed. Remaining work tracked in integration-progress.

2026-11-03 — Added four distinct generated pixel-inspired grayscale Kerala assets:
website chaya-stop login and companions introduction/About; app ferry login and
umbrella companions introduction/About. Reworked responsive login compositions on
both platforms while preserving logo, animated pixel scenes and functional art.
Rejected unshipped shelter drafts with haze. Built-in image_gen used; prompt briefs
and asset paths recorded in Docs/illustration-design-phase.md.

Design verification only: website build passed; browser login screenshots at1440
and390, no overflow at320 or900 with200% root text. Two Flutter design/layout tests
passed including320/390/900 with1x/2x text and reachable sign-in. Debug APK build,
install and launch succeeded; actual emulator login visually reviewed. No personal
sign-in, backend tests or broad integration checks; those remain paused by owner.

2026-11-03 — Implemented transaction-scoped support reply retry keys in Go and
stable per-draft retry IDs on web. Android and cross-restart retry handling remain
outstanding. Verification not run under the owner's current hold.
2026-11-03 — Wired Android support reply retry IDs to the Go contract; no checks
run under the current verification hold.
2026-11-03 — Added cursor-based pagination and account-scoped refresh for web
notifications. No tests run under the current verification hold.
2026-11-03 — Added cursor pagination to both user and moderator web support
inboxes. No verification run under the current hold.
2026-11-03 — Added cursor pagination and explicit retry for web group chat reads.
No checks run under the current verification hold.

2026-11-03 — Added website/vercel.json for Vite client-side routes, excluding
/api so admin functions retain their own routing. Vercel project root must be
set to website/. Reviewed Vercel's official Vite and project-configuration docs.
Deployment routing has not been verified; broad testing remains on hold.
2026-11-03 — Scoped web resource snapshots to their authenticated owner and path
to avoid stale private data during account changes. Verification remains paused.
2026-11-03 — Added an initial pure daily lift schedule calculation with IANA
zone and DST-gap validation. Persistence, concurrency and UI remain outstanding;
no tests run under the verification hold.
2026-11-03 — Added recurring-lift cleanup to account deletion and rejected
clock-transition windows in the schedule calculation to prevent an inaccurate
fixed-offset ride. No checks run under the verification hold.
2026-11-03 — Added staged Go recurring-lift account-scoped persistence and API.
No scheduler or client wiring yet; no verification under the current hold.
2026-11-03 — Added staged recurring-lift worker and transaction-scoped account,
schedule and active-journey checks. No runtime or race verification under hold;
client wiring remains outstanding.
2026-11-03 — Added web recurring lift form and pause/resume actions against the
staged Go API. No verification under the current hold; Android UI remains pending.
2026-11-03 — Added Android recurring lift controls using shared place search and
staged Go endpoints. No checks run under the current hold.
2026-11-03 — Refreshed root and backend READMEs, removed Docker from their
workflow, and preserved the last recurring occurrence across schedule edits.
Implementation continues; testing deferred until the requested pass is complete.
2026-11-03 — Cleared recurring-offer drafts across account switches and retained
last occurrence on schedule saves to reduce duplicate departures. Tests deferred
until implementation is complete per owner request.
2026-11-03 — Hardened recurring-lift rescheduling against duplicate departures
after same-day edits and reduced maintenance-mode scheduler churn. Testing is
deferred until the implementation pass ends.
2026-11-03 — Connected recurring-offer changes to the realtime invalidation
stream and Android screen refresh. Testing remains deferred.
2026-11-03 — Added optimistic concurrency to recurring-offer saves and wired both
clients to send their loaded revision. No tests run during the implementation pass.
2026-11-03 — Connected admin app-version settings to the Go API through the
server-only BFF. Testing remains deferred until implementation is complete.
2026-11-03 — Added staged authenticated web-push token registration/removal with
account-scoped ownership and a device limit. Delivery/client wiring not yet done;
no tests run in the implementation pass.
2026-11-03 — Added staged per-device web-push delivery and outbox checkpoints;
provider acceptance/checkpoint crash window remains an at-least-once limitation.
No tests run during the implementation pass.
2026-11-03 — Added staged browser push registration controls and service worker.
No runtime verification; Firebase VAPID setup and account-transfer handling remain.
2026-11-03 — Added web-push sign-out cleanup, same-account re-registration and
foreground refresh lifecycle. No testing during this implementation pass.
2026-11-03 — Added browser push registration expiry and local token retirement on
external authentication changes. No tests run during the implementation pass.
2026-11-03 — Added per-device push preference/account checks and documented web
push configuration and unverified delivery limitations. Tests remain deferred.
2026-11-03 — Exposed existing worker heartbeat and last-success data in the admin
operations panel. Testing deferred during the implementation pass.
2026-11-03 — Hardened browser push registration against account changes within a
tab and made restoration failures visible with Retry. Tests remain deferred.
2026-11-03 — Preserved recurring-offer activation state while editing, resumed
deferred Flutter refreshes and cleared account-scoped profile/settings state on
account changes. Tests remain deferred.
2026-11-03 — Hardened recurring-offer transaction retries, safe edit replays and
worker fairness; preserve retryable failures instead of pausing the offer. No tests
run during the implementation pass.
2026-11-03 — Added deduplicated support-ticket creation, account-aware APK upload
cancellation and release-settings refresh; retired the local push subscription
when signing out after deletion. Removed unused Docker build files and updated
current deployment guidance. Testing remains deferred for the implementation pass.
2026-11-03 — Hardened browser push against cross-tab subscription races and
revoked permission; removed unused demo avatar assets. Added scheduling/replay
regression checks, not yet executed.
2026-11-03 — Ran Go race suite, website build/error-copy tests, focused Flutter
analysis and 12 Flutter tests successfully. Initial Go run exposed a DST schedule
search bound error and a missing/unused import; initial website build exposed
TypeScript pagination annotations; fixed and reran. Sandboxed Go HTTP tests could
not bind a socket; the approved retry passed. No production data was used.
2026-11-03 — Reviewed the current local landing at 320 and 1440 pixels with an
isolated browser; no horizontal overflow at either width. Isolated MongoDB and
Firebase Auth emulators were not available, so multi-account tests did not run.
2026-11-03 — Reopened the existing Android emulator, built and installed the
current debug APK using JDK 21, and visually confirmed the Google sign-in screen.
The default Java 25 build failed on Gradle compatibility before the successful
JDK 21 build. No account sign-in or production-database fixture was performed.
2026-11-03 — Ran focused concurrent recurring-offer and support retry checks
with eight Firebase Auth Emulator personas and a temporary localhost MongoDB
8.0.34 replica set. Both checks passed, including another account's empty schedule
and a conflicting support replay. Temporary services were stopped; no production
database was used. Real provider and broader multi-account scenarios remain.
2026-11-03 — At the owner's request, stopped further testing, corrected the
backend README environment-template instruction and added app/website setup
READMEs. No tests were run for this documentation change.
2026-11-03 — Installed Go and deployed the ORIX binary to the existing EC2
upstream port after Atlas access was enabled; stopped and disabled Typelo. HTTPS
health/readiness and ORIX-domain CORS checks passed. Deployed the website to
Vercel, set its production environment and VAPID public key, and configured the
GitHub-linked project root as website. Direct Geoapify autocomplete succeeded.
Live auth, push, routing and APK publication were not verified.
2026-11-03 — Added a Geoapify road-routing adapter to support lift itinerary
planning using the configured provider key. Go race suite passed; live lift
matching remains unverified.
2026-11-03 — Verified Geoapify adapter with focused contract and live provider
requests, and built Android 2.1.1+7 with a newly generated release key. APK v2
signature and package metadata verified. Firebase registration of the release
fingerprints and signed-device login remain outstanding.

2026-11-03 — Simplified the website session intro to the standard ORIX symbol
and animated loading squares with a small generated grayscale Kerala sprite.
Removed the slogan/skip action and enforced a two-second minimum independent of
motion preferences. Website production build passed; no broader QA was run.

2026-11-03 — Website onboarding now combines photo confirmation and advancing
into one explicit button after a successful upload or existing profile photo.
Removed the separate checkbox that blocked Continue, retained the server-required
face confirmation, and blocked submission during upload. Website build passed.

2026-11-03 — Onboarding permission buttons now show enabled labels and the
shared disabled gray styling once granted. Existing browser permission state is
read where supported and permission changes update the controls. Pending requests
disable repeated clicks; denied requests stay retryable. Website build passed.

2026-11-03 — Spaced the final web onboarding step into a short introduction,
tour action, consent group and separated footer actions. Replaced error-boundary
layout with a compact centered recovery group, removing the inherited auto-margin
that separated the illustration from its message. Website build passed.

2026-11-03 — Dashboard now uses the available workspace width with a larger
journey column and explicitly spaced safety/lift sections. Tablet uses two
support columns, mobile stacks them with consistent gaps and bounded columns.
Website build passed; authenticated visual QA was not run.

2026-11-03 — Group chat now uses bounded message bubbles, a responsive composer,
Enter-to-send and Shift+Enter newlines with IME composition preserved. Singleton
groups explain why confirmation is unavailable. Notification logo sizing is
bounded by default; metadata and long headings wrap on narrow screens. Missing
assets no longer rewrite to HTML, entry documents revalidate, and stale Vite
chunk failures reload at most once per minute. Website build passed. A complete
authenticated screen-by-screen responsive audit has not been performed.

2026-11-03 — Owner explicitly authorized clearing production user accounts and
activity while retaining configuration and APK releases. Paused the API, deleted
user-data documents from the allowlisted collections, checked empty user/activity
collections, and restarted it. Indexes were retained; Firebase identities were
not deleted. Operational worker health records regenerate normally.
Added permission-based current-city lookup via private Geoapify reverse geocoding
without location-history persistence, replacing the fixed city. Header/sidebar
now reuse the authenticated upload/Google photo loader with retry states. Website
build and Go race suite passed.

2026-11-03 — Redesigned group conversation as a bounded 840px chat surface
with a compact header, deliberate empty state, readable message text and subtle
author/time metadata. Composer grows from one line and preserves Enter/Shift+Enter
behavior. New messages follow the bottom only while the reader is near it. Mobile
uses larger body text and tighter gutters. Website build passed. Research: Material
Web text-field guidance and Carbon chat layout customization; no authenticated
production chat fixtures were created for this design change.

2026-11-03 — Installed the requested seo-audit skill locally. Added generated
monochrome social preview artwork, Open Graph and Twitter metadata, a sitemap
and robots file, and noindex response headers for private app/admin routes.
Verified the requested Firebase identity and assigned its admin role using the
guarded bootstrap command. Website build passed. Per-page server-rendered SEO
metadata and authenticated search indexing behavior remain unverified.

2026-11-03 — Corrected the admin failed-delivery response contract and added
response-shape checks for delivery, worker, audit and report lists. Website
production build passed. Authenticated production flows were not exercised.

2026-11-03 — Activity records now resolve actor display names from account
records and show them in the admin log. Website build and Go build passed; the
backend service was updated and reported active. Authenticated activity-log
verification was not performed.

2026-11-03 — Added paginated administrator people and journey-group directories,
profile photos and member details, plus audited administrator access to group
conversations and idempotent admin messages. Improved the Android release prompt
and retries. Go and website builds passed; authenticated group/chat and Android
browser verification were not performed. The live public settings reported no
published APK metadata at the time of the change.
The production HTTPS proxy rejected APK uploads larger than 2 MB. Increased its
limit to 256 MB only for the authenticated APK upload route, checked the nginx
configuration and reloaded the service. No APK was published by this change.

2026-11-03 — Fixed signed APK publication failing after about 15 seconds by
extending the Go HTTP read/write window to match the bounded upload handler.
The admin UI now separates integrity checking, upload, publication and verified
success, with a download action only after a matching published hash is read
back. Also constrained workspace header text within its flex layout. Website
and Go builds passed. A real APK upload and responsive browser QA were not run.

2026-11-03 — Avoid closing a connecting browser WebSocket during navigation or
visibility changes; close it after opening if the connection is no longer needed.
Website build passed. Production nginx recorded successful WebSocket upgrades;
no authenticated message delivery test was performed.

2026-11-03 — Added credential-free APK publication stage diagnostics and
separated archive validation failures from service errors. Go build passed and
the backend service was restarted. The reported upload failure requires another
real attempt to identify its stage; no production test APK was uploaded.

## 2026-11-03 — Accessible-screen UX and login recovery

Removed forced web splash and entry fades; kept workspace navigation while routes
load. Added bounded auth waits and Retry. Flutter login returns actual backend
authentication status. Tightened landing hero/action copy, corrected recurring-lift
claim and flattened login button fill, preserving grayscale pixel art/logo.
agent-browser: 40 public route/viewport checks with no horizontal overflow.
Login/home axe: no violations; home has one incomplete decorative-arrow check.
Website build and three error-copy tests passed. Focused Flutter analysis initially
found three existing brace infos, corrected before rerun. Owner limited review to
accessible screens; authenticated/device and production speed remain unverified.
See Docs/ux-review.md.

Focused Flutter analysis rerun: no issues found.

2026-11-03 — Production APK diagnostics showed failure during GridFS storage.
The authentication middleware passed its ten-second context into the entire
upload; the APK route now uses the original request context after authentication
and its own bounded upload deadline. Focused Go auth/upload tests passed. The
VPS backend was rebuilt and restarted; real APK publication remains unverified.

## 2026-11-03 — Remove chat refresh bottlenecks; reveal desktop navigation

Web chat previously reloaded every opened history page and group details after
sending or receiving a socket invalidation. It now consumes authorized socket
message bodies, merges by persisted/client IDs, displays pending/failure states,
and inserts the send response without waiting for another read. Older history
loads one cursor page independently and preserves scroll position. Refreshes
coalesce and keep new arrivals/history; account/group changes discard state.
Failed sends retain their message and use the same ID on explicit retry. Both
composers permit typing during sends. Flutter reconnect no longer clears readable
history; send scrolling uses a mounted-checked frame callback instead of a delay.

Desktop sidebar hides/reveals at the left edge, with a short slide, content space
reserved while open, keyboard/touch menu button and reduced-motion support.

Executed: website build and four chat regression test groups passed; focused
Flutter analysis initially reported twelve existing style infos, corrected, then
passed without issues. agent-browser isolated CSS harness confirmed desktop
hidden/reveal/hide and content alignment at 256px without horizontal overflow.
This harness does not validate the signed-in application. No backend code changes,
production fixtures, authenticated/device chat QA or latency benchmarks. Owner's
accessible-screen limitation and broader verification hold remain in effect.


2026-11-03 — Mobile monochrome/pixel layout refinement

Reduced shared heading sizes, card/list padding, doubled section margins and
button padding while keeping 48dp minimum button targets. Bottom navigation now
uses a black stepped selection indicator and explicit white selected icons.
Home, matching, journeys, groups and journey detail use a shared From/To route
component with two-line previews and expandable original full addresses; expansion
resets when endpoints change. Journey detail has a short page heading. Account
identity puts the name before email. Matching keeps its pixel animation at a
smaller size, collapses explanatory copy, uses a shorter map and avoids the extra
home loader during background refreshes. Matching timestamps render locally.

Executed: scoped Flutter analysis of the nine changed library files plus the new
layout test (no issues); flutter test --no-pub test/journey_layout_test.dart
test/pixel_design_test.dart (5 passed); git diff --check passed. Initial analysis
found existing brace lint infos in touched files, now resolved. Initial layout
checks exposed test scroll timing/state setup issues, corrected before passing.
Isolated layout checks covered 320x740, 390x844 and 740x390 at 100% and 200% text,
full-address expansion and endpoint reset; loaded bundled fonts and inspected the
390px rendered composition at /tmp/orix-journey-layout.png. This is a component
composition, not authenticated screen/API verification. No production personas,
backend integration, device map/permissions or broader regression checks ran.
The broader verification hold remains. Other in-progress website edits are outside
this mobile commit; no release artifact or version bump is part of this change.

## 2026-11-03 — Compact web route and card typography

Replaced large combined address headings with shared From/To rows and recoverable
full-address disclosure across journey, group and lift views. Made card/section
headings smaller, tightened mobile workspace spacing and date/time copy, and
reduced journey-detail illustration size. Original pixel headings/logo/art and
all full address values retained. Website only; unrelated Flutter work untouched.
Executed website build and agent-browser component layout checks using long labels
from the supplied screenshots: five widths (320–1440px), collapsed and expanded,
plus 200% text/spacing overrides, without horizontal document overflow. Actual
component was server-rendered into an isolated local harness; its Vite HMR socket
reported sandbox EPERM but rendering succeeded. This does not verify signed-in
screens or service flows. No production data was created and no auth bypass used.

## 2026-11-03 — Restore preferred web typography and journey design

At the owner's request, reversed the website changes from 28443b9. Restored
previous pixel address headings and card design, spacing, date formatting and
journey illustration size; removed the unused compact-route component. Kept
chat and desktop navigation improvements. No Flutter changes.
Verification: website production build and git diff whitespace check passed.
No authenticated UI or device checks were run for this restoration.


## 2026-11-03 — Optional donations, Manglish reminders and preview repair

Added a dismissible once-per-local-calendar-day donation prompt and permanent
Donate entries to Flutter and the web workspace/profile. Daily prompting is per
device/browser, stored as a preference, not a payment record. Copy asks users to
donate to keep ORIX going; any amount is welcome. Android targets Google Pay
through a small native channel; iOS uses its Google Pay URI. Missing-app failure
shows authored guidance and a copyable UPI ID. Desktop displays a code-generated
square-module monochrome QR with its four-module quiet zone intact. Its payload
was independently decoded from the owner's supplied QR and preserved exactly.
Both generated assets and the rendered desktop QR decoded back to that payload;
asset verification also passed at five sizes. No payment was initiated or claimed.

Go now owns optional re-engagement through the durable notification outbox.
Twenty original Manglish title/body pairs rotate deterministically. Scheduling
windows are 08:00, 12:00, 16:00 and 20:00 IST, with a four-hour per-user reservation
cooldown, transactional scheduling/deduplication and 30-minute expiry. Only
onboarded, unrestricted accounts with explicit promotions=true are eligible;
missing/default preferences remain off. Delivery rechecks preference/account
state and bounds retries through the existing outbox. The old Python worker's
automatic hourly topic broadcast is disabled to avoid duplicate coexistence sends.
Admin Operations can edit/add/pause messages and disable the schedule using
versioned, audited API updates; drafts survive save errors. Android uses a quieter
monochrome-logo channel, web background pushes show the same public copy, and
foreground web uses the existing themed toast. Private push content stays generic.
Reminder taps preserve account checks and open Home. OS notification chrome stays
platform-controlled; no custom OS typography claim is made.

Social previews reuse the generated artwork, exported as a 253,902-byte JPEG
instead of the 1.6MB PNG. Added explicit secure image URL, MIME type, actual
1732x908 dimensions, canonical and alt metadata; versioned URL avoids reusing an
old image cache entry. Live pre-change HTML and PNG returned 200 to the crawler
request, so this addresses robustness/cache issues rather than proving every
messaging platform's preview was broken. Existing third-party caches may require
a re-scrape. App version is bumped to 2.1.2+8; APK build/signature results follow.

Executed so far: website production build; Go build and go vet ./...; focused
GOCACHE=/tmp/orix-go-cache go test -race ./... -run
'TestEngagement|TestNotification' passed; Flutter scoped analysis clean after
fixing existing touched-file brace infos; donation/payload widget tests 9 passed;
web donation-link/date tests and push-worker privacy/validation checks passed;
Python worker py_compile passed; Android compileDebugKotlin succeeded using a
checksum-verified temporary official JDK21. Installed JDK25 failed Gradle Kotlin
DSL startup initially. The first unfiltered Go suite attempt hit sandbox localhost
listener restrictions; the focused authorized rerun passed. golangci-lint is not
installed. Isolated desktop/mobile web donation review used actual components;
browser reload confirmed no duplicate daily prompt. No account fixtures, actual
push deliveries, database concurrency scenarios, reboot/background device checks
or broader integration/regression QA ran. The owner's broader hold remains.

Backend deployment update: built a static Linux/amd64 binary, uploaded it to the
existing orix-backend service and restarted it successfully (active). No fixture
accounts or production push/test notifications were created.

Release-publication follow-through: added a public /download page and landing
footer link for all devices. Added the operator-only publish-android-release CLI,
using authenticated SSH/server credentials, explicit production database/apply
selection, pinned APK hash and the existing verified bootstrap-owner identity.
It does not mint Firebase user tokens or add an HTTP auth exception. Firebase
identity and current admin role are checked; normal single-use upload grants,
archive/hash validation and transactional audited publication are reused. Matching
already-published artifacts are not uploaded again. Current app version metadata
is updated while preserving the minimum supported version and official store
links. A computed android_apk_available flag lets new Android clients use only
their configured API's download path when a matching APK is actually published.
Focused operator guards/version tests and Go vet passed after correcting the
CLI's accidental Android-specific filename. Ten focused Flutter tests passed
after correcting a non-const test constructor. Public APK build, signature and
full-download verification are still pending at this checkpoint.

2026-11-03 — Android 2.1.2+8 release artifact built and verified. APK size
70,488,050 bytes; SHA-256 a4a1511872ed6e2e1dad397f8e034ffbd642132a7145f25f0cc35f86c9f35eb7.
Apksigner and aapt passed; certificate matches the existing release keystore.
Live website metadata/artwork and public download-page layout were checked.
The final backend is active and ready; unauthenticated admin reminder access is
rejected. Publication is pending explicit owner approval after automatic review
rejected reading the running service environment as an unintended credential
source. No credentials were read or copied. Public download remained 2.1.1 at
this checkpoint. Full hashes/fingerprints and verification: Docs/release-2.1.2.md.

2026-11-03 — Implemented web-only technical SEO under the owner's no-design-change
constraint; Play publication was explicitly cancelled by the owner due to cost.
Actual public route prerendering, six unique title/description/canonical/social
sets, factual JSON-LD, generated sitemap, noindex private shells, real 404 routing
and route module preloads. Existing PNG art gained lossless WebP derivatives;
all three decoded RGBA comparisons pass, saving ~38–39%. Website production build
and focused `test:seo` pass; diff whitespace check passes. Added Google public
verification tag and a live-verifying explicit IndexNow command. No live rankings
or traffic results claimed. Vercel preview first failed due to source root, then
missing Firebase preview config; website-only upload allowlist was checked and
isolated preview dpl_HZGFGpYUnME2a3qGEh9qgLDKqMmE built READY with only existing
public client config. Production staging was auto-review rejected after the failed
preview and did not run. Remaining actual verification/publication results follow.
Detailed evidence/research and limits: Docs/seo-audit.md. No broader held QA.

2026-11-03 — Owner reported a black rectangle behind the hero during the SEO
performance pass. The responsive export script had converted transparent PNGs to
RGB, dropping alpha. Regenerated all six hero/About responsive variants as RGBA
WebP with byte-for-byte alpha-plane checks against the resized originals; changed
the URLs with an `-alpha` suffix to avoid cached opaque versions. Added WebP alpha
flag assertions to `test:seo`. Build and focused checks pass. Connected Chrome
visually confirms the local hero blends into the existing page background. No
layout/CSS, typography or illustration composition changed. Deployment and live
verification results follow.

2026-11-03 — Transparency repair published and verified live. Six actual image
files match local alpha-verified bytes; Chrome loads the corrected `-alpha` hero,
and screenshot corner pixels match the light canvas. Current production
 dpl_CnfpAeTDRsB4GFJUsEJsaQq5Kf6p is READY on the main domain. Final focused live
HTTP checks pass for six public pages, six private shells, XML/robots/key, real404,
clean redirect and 22 assets. Added explicit www root rule after wildcard-only
checks found root still200; /, /about and /login now308 to matching apex paths.
Google ownership verified, sitemap processed successfully with six pages, live
inspection says available/indexable. Manual indexing request failed at Google’s
reCAPTCHA service connection; no bypass or accepted-request claim. Final IndexNow
accepted six URLs (200). Final mobile Lighthouse: SEO/accessibility/best practices
100 each, performance72; FCP3.2s/LCP5.2s/TBT0ms/CLS0. No actual index inclusion or
query ranking claimed. See Docs/seo-audit.md for evidence and remaining limits.
