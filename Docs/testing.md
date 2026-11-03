# Final testing plan

## User-requested timing

Do not begin further test runs, integration checks or emulator QA while the code
rewrite is in progress. Finish the code changes first. The user will then supply
the frontend/backend environment values, after which use the emulator to test all
features. Unit/static checks already run are historical evidence only and do not
prove the completed application. A Go race test process had already been started
before this instruction arrived; do not launch further checks until the gate opens.

## Internal testing accounts

Implement a repeatable internal-account provisioning command before final QA.
Use Firebase Auth Emulator for local tests or an explicitly selected nonproduction
Firebase project for real-device/FCM integration. Create at least four distinct
accounts; six or more are preferred. Do not embed credentials, reuse production
users, weaken token verification or expose a public test-login bypass. Keep test
credentials in local ignored environment files and show account identifiers only
in provisioning output. Provision corresponding app profiles through the actual
onboarding flow/API so normal validation and database invariants are exercised.

Required personas:

| Persona | Purpose |
| --- | --- |
| Traveler A | Creates a shared-auto request and coordinates a group |
| Traveler B | Same route with nearby pickup/drop, group confirmation and chat |
| Traveler C | Partially overlapping route; tests corridor suggestions and capacity |
| Lift driver | Offers a lift with a bounded seat count |
| Safety traveler | Female profile and female-only preference checks |
| Incompatible traveler | Opposite route/time and negative matching checks |
| Moderation admin | Reviews reports through authenticated admin controls |

Use separate additional restricted/new profiles for banned/suspended/onboarding
states. Test access control with the restricted accounts, not by changing their
client UI alone. Nonproduction account provisioning must refuse production unless
an explicit environment selection is supplied and reviewed. Implement provisioning
against emulator/staging only by default; do not create actual cloud accounts
until the environment is supplied.

## Emulator and integration coverage

After code and environment setup, test sign-in, onboarding, account lifecycle,
map selection and similar-route suggestions, all travel modes, multiple concurrent
group joins/capacity, readiness, cancellation/rescheduling, chat and reconnects,
notification preferences, reminders fifteen minutes before departure, foreground/
background/cold launches, denied permissions, battery constraints, reports,
blocking/moderation, ratings, support, deletion and offline/retry behavior.

Use at least four simultaneously usable sessions/accounts where needed. Check
server state as well as screens. Firebase Auth Emulator does not prove real Google
sign-in or FCM delivery; verify those with a staging Firebase project after values
are available. Verify release configuration and Google Play policy requirements
with official current references before declaring publication readiness.

## Provisioning implementation (written; not run)

`backend/seed_accounts.go` adds `go run . seed-test-accounts` from backend.
It requires process environment values APP_ENV=local,
FIREBASE_PROJECT_ID=demo-orix, FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:9099,
TEST_API_BASE_URL=http://127.0.0.1:8080 and TEST_ACCOUNT_PASSWORD (12+ characters).
Start the Auth Emulator for demo-orix and the Go server with the same local
configuration after the testing gate opens. A separate local Mongo database is
required for the backend. No service-account credentials are needed for emulator
auth. Seed tooling is restricted to local loopback services and never provisions
cloud accounts.

The command creates or updates eight deterministic users, signs into the Auth
Emulator, calls /auth/verify and /auth/onboard, and prints identifiers only. It
resets the local test password on reruns; it does not reset existing app profile
or ride state. All personas share the supplied local password. Emails are
traveler-a, traveler-b, traveler-c, driver, safety, opposite, admin and newcomer
at orix.example.
Traveler A/B and Safety Traveler have female profiles. Driver's ride offer must
still be created through the normal ride flow. The admin persona is onboarded normally, then a local-only database transaction
assigns its admin role and records an audit entry. The role helper accepts only
orix-test-admin in the validated demo project/loopback test database. Admin API
requests still require its Firebase token and ADMIN_API_SECRET; no HTTP bypass is
added. The newcomer persona signs in but skips onboarding on a fresh database;
reruns preserve any existing onboarding state. Remove/reseed to restart that flow.
Banned and suspended scenarios should be created through actual reports and admin
moderation actions so cancellation, notifications, appeal and restore can also be
exercised. Seeding does not silently restore restricted accounts; restore them
through moderation or remove/reseed before retrying the provisioning command.

For Flutter debug builds set ENABLE_INTERNAL_TEST_LOGIN=true and
TEST_AUTH_EMULATOR_HOST=10.0.2.2 (Android emulator),
TEST_AUTH_EMULATOR_PORT=9099, and API_BASE_URL=http://10.0.2.2:8080 in the private
build-time env file. The login screen provides an internal account email/password
dialog only in debug mode. A separate named Firebase app uses demo-orix; backend
tokens still go through normal Firebase verification and account authorization.
Release builds ignore the internal-login flag. Passwords are never build defines.
Push initialization is skipped in this Auth Emulator mode. Storage uploads, real
Google sign-in and notifications require the later staging/device pass. A demo
project's local auth does not emulate other Firebase services automatically.

Implementation was intentionally not compiled, analyzed, executed or emulator
verified during the code-completion gate. Run final security/access and behavior
checks when the user supplies environment settings.

Firebase reference:
https://firebase.google.com/docs/emulator-suite/connect_auth

### Google sign-in and removal after QA

Production frontend authentication remains Google sign-in. The debug-only local
switcher uses Firebase email/password to choose among emulator identities. This
exercises the normal Firebase ID-token/backend authorization flow, but does not
prove Google account-picker, OAuth configuration or real Google token exchange.
Include a separate staging Google sign-in pass with actual Google accounts after
environment setup. Do not claim emulator password login validates those steps.

Both local commands and emulator server startup now require loopback MongoDB and
MONGODB_DATABASE=orix_internal_test as well as the demo-orix/local Auth settings.
After the testing phase, run `go run . remove-test-accounts` from backend with the
same local environment. This command removes the eight deterministic emulator
identities and their local profiles, rides, notifications/outbox events, queued
account deletions, related groups/chat,
reports, tickets, ratings and actor-linked audit records. It refuses cleanup if a
test group includes a non-seeded user. It does not create/remove cloud Google
accounts or perform cloud staging cleanup. Stop local app sessions before cleanup.
The command is written but unrun; deletion and interrupted-retry behavior must be
included in final QA. Keep the internal-login build flag disabled for release;
remove internal test users after QA, and never provision them into production.

### Moderation fixture coverage (implementation only)

Use admin@orix.example for authenticated moderation, safety@orix.example for
reporting/appeal scenarios, and an ordinary travel persona as the report target.
Create a shared group before reporting where group evidence is required. Apply
suspend/ban via the real admin resolution API, then verify the target cannot use
travel/chat APIs but can create an appeal ticket and request deletion. Restore
through the audited API and confirm cancelled trips stay cancelled. Do not disable
the Firebase identity for this scenario: provider-disabled recovery is a separate
flow. Use newcomer@orix.example for first-run onboarding only on fresh seed state.
No accounts have been created and none of these scenarios have run yet.

## Public deletion and container QA (not run)

After the code/environment gate, execute the web sign-in/deletion and deployment
scenarios in deployment.md. This needs a real staging Firebase WEB app, authorized
hostname, monitored support mailbox and published policy. Local Flutter Auth
Emulator personas do not prove browser Google OAuth, cloud deletion, external
mail delivery or Play Console link acceptance. Build/run the staged Go container
only during final checks; confirm the image contains no environment credentials or
Python runtime and handles signal shutdown, probe behavior and worker failures.

## Support-origin deletion QA (not run)

Use the local admin persona and ordinary targets to test read-only plans, required
ownership/database/environment assertions, non-admin and disabled-operator denial,
self-target denial, actor demotion races, disabled/missing target identities,
identity-only jobs, case conflicts and same-case retries. Verify no profile/index/
audit/job writes from planning. Run the normal worker and inspect actual deletion,
case metadata clearing and retry recovery. Do not run while the code-first gate
remains shut. Staging ownership/mailbox procedures need separate verification.

Administrator mutation race coverage must include report resolution, restoration,
staff changes, support replies/status, maintenance/version updates, delivery retry,
group cancellation and support-origin deletion. Interleave actor demotion/deletion
with the transaction and verify its shared authorization write conflicts/retries,
then refuses the mutation when authorization is lost. Retest self-action/domain
conflicts and last-admin protection separately. No race scenarios have run yet.

## Geographic filtering and matching worker QA (not run)

Run the new matching/bounds_test.go along with existing route/ranking tests only
after code/environment setup. Compare filtered and unfiltered eligible results
for realistic nearby/partial/opposite routes, longitude wrap, poles, long trips,
all travel modes, safety preferences, legacy/missing/versioned bounds and floating
boundary cases. Inspect database state after edits/leave/join/deletion; stale or
nonmember anchor metadata must not suppress valid candidates. Run the migration
plan/backfill in staging and verify repeatability and conflict handling.

Use actual Mongo explain plans and a representative dense multi-user workload to
measure index use, candidate/member documents loaded, pagination fairness and
latency. A new index declaration does not prove a speedup. Confirm malformed
excess/duplicate routes fail boundedly, user-ID uniqueness conflicts are detected,
and candidate projections never need excluded profile/credential fields.

Inject provider/database failures into both background matching passes: leases
must release/expire, retry hints back off/clear correctly, failures reach worker
health, and one failed request should not skip singleton reconciliation when the
shared work deadline still permits it. Reservation contention/no-fit results
remain normal misses. None of these scenarios has been executed yet.

## Reminder throughput and singleton lifecycle QA (not run)

Verify pending one-member confirmation is refused, legacy ready singleton backfill
is idempotent, matching remains possible and expired windows cannot activate.
Simulate a large ordinary backlog and slow FCM calls: scheduling must continue,
the reserved reminder worker must remain available, total provider work must stay
bounded, and duplicate schedulers/workers must respect stable IDs and leases.
Inspect bulk partial-failure recovery, terminal lease recovery, retry/backoff,
heartbeat write cadence and indexed priority selection.

Validate per-recipient stale status/departure/membership rejection, stable payload
IDs, server minutes/title, short Android TTL, absolute APNs expiry/collapse and
near-departure suppression. An FCM accepted response is not device delivery. Test
provider/receipt crash windows and stale accepted pushes against client group
refresh. Lift per-pickup timing is still an implementation gap and must be completed
before those final scenarios can establish fifteen-minute behavior for all modes.

## Personal lift pickup and start QA (written tests, not run)

Run reminder_schedule_test.go in the final Go suite. Add live multi-user scenarios
where a passenger's pickup is 30+ minutes after driver departure: their reminder
must occur fifteen minutes before that pickup even when the group is in progress.
Check driver-only start, all readiness, driver origin arrival, passengers who have
not yet marked arrival, a start at least a minute late and sub-minute jitter.
Inspect committed personal schedules, new event IDs, old-event suppression and
own-only pickup fields for group current/history responses.

Verify route-leg offset accuracy/fractional milliseconds, pickup labels, missing/
old plans triggering replan or migration conflict, legacy common-time fallback,
removed/finished users, leave/edit/rejoin, completion/cancellation/expiry/deletion,
permissions/preferences and rescheduling race boundaries. Review preexisting Go
v1 reminder queues before cutover to avoid migration duplicate/lost alerts. Road
duration is an estimate; no live traffic behavior is implemented. Client group/
alarm navigation and display must use current personal pickup metadata. None of
these database/provider/device scenarios has been executed.

## Retirement and first client contract changes (not run)

After complete implementation/environment setup, verify retirement planning causes
no writes; apply assertions, enabled/current administrator checks, fixed allowlist,
profile field-only changes, non-target preservation, interrupted incremental retry,
audit atomicity and concurrent role/mutation handling. Never run retirement apply
before full transition QA. Test revoked-admin mutations returning 403 separately
from actual domain conflicts returning 409.

Client QA must cover missing/invalid/insecure release origins and a valid HTTPS
origin, debug Android/other host defaults, no requests before invalid configuration
is handled, RFC3339 Z/positive/negative offsets and legacy naive UTC dates. Verify
road GeoJSON ordering, personal pickup times/stop offsets/member arrival/finished
fields, status refresh with unchanged group IDs, confirmation full-state refresh,
POST leave and cursor header propagation. No build/static/runtime checks ran yet;
frontend integration remains in progress, not ready for emulator acceptance.

## WebSocket-first chat and group action QA (not run)

Use at least four personas to test socket chat_sent acknowledgement, new_chat_message
nested delivery, identical text with distinct IDs, lost acknowledgements, disconnected
REST fallback with the same ID, retry without duplication, bounded pending requests,
server rejection, old-session callbacks and rapid account switches. Verify unread/
older message paging, reconnect REST resync, missed outbox events, idle heartbeat,
fresh-token reconnect and resume. Native Bearer/browser auth-subprotocol handshakes
must work without token URLs; ambiguous/absent credentials and old query auth must
fail. Confirm only orix.v1 is selected and tokens never reach logs/UI/errors.

Test explicit confirm/arrival/start/finish/leave controls, singleton waiting UI,
late pickup plans, driver-only lift start, read-only closed-group chat and finished
members leaving the active list while history stays readable. Simulate transient
refresh failures and overlapping polls/actions to detect stale replacement/flicker.

Reminder UI must refresh current membership/pickup, reject changed/past/finished/
closed or inaccessible coordination, allow dismissal/back, and record arrival
without completing a trip. Check per-group local IDs, actual remaining-time text,
foreground stable-ID deduplication and duplicate iOS APNs/local presentation. No
unit, static, widget, socket, database/provider or emulator scenarios have run.

Additional pending QA for the matching/chat continuation: create each travel mode
with a positive local-time window; edit pending and unconfirmed singleton requests;
join a compatible suggested group concurrently; refresh stale status during edits
and cancellation. Check date/time near midnight and coordinate-only locations.
Test a delayed HTTP chat snapshot completing after a new WS message/ack: the
new message must remain visible exactly once. Background the app, navigate away
from chat and return; fallback polls must pause/resume and history must catch up.
Reading older chat must preserve scroll position; near-bottom arrivals follow.
These cases are documented, not executed. Maps, picker/provider policy, endpoint
marker order and member-only road itineraries need final emulator QA.

## Location picker and place provider QA (not run)

- With no geocoder configured, search explains availability; map, coordinates and
  GPS remain usable. Configure a real private/contracted provider only after the
  coding phase. Verify attribution, search failure, quota/Retry-After and timeout.
- Typing must make no request; tapping Search or submitting makes an explicit
  request. There must be no autocomplete or pan-driven reverse request.
- Deny GPS, deny permanently, turn location services off, and test timeout. Map
  selection must remain available. Opening the picker must not prompt for GPS.
- Delay GPS/search, then move the map, choose another point or edit query: stale
  completions must not replace the current point/result list. Test disposal too.
- Check editing an existing pickup/drop reopens the correct point, identical
  place names at different coordinates, negative/NaN/out-of-range coordinates,
  label bounds, small screens, keyboard, panning, pin alignment and attribution.
- New backend source tests cover URL restrictions/attribution, cached lookup,
  invalid coordinate filtering, provider pacing and redirect rejection. These
  test files were authored and formatted; no tests were executed.

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

2026-11-03: Onboarding now sends explicit privacy acknowledgement and the Go
API requires and records it atomically with a timestamp. Local seed payloads
include the acknowledgement. Historical users are not backfilled; policy
versioning and legal review remain outstanding. Existing legacy test fixtures
need updating before the final test run. Formatting only; no tests run.

2026-11-03: Updated Go onboarding validation fixtures for privacy consent and
added missing/false acknowledgement cases. Test source formatted; no tests run.

2026-11-03: The terms screen loads account acceptance status and provides an
explicit Go API acceptance action only for the displayed version 1.0; a server
version mismatch cannot be silently accepted. Published/legal terms review and
any enforcement of acceptance before app use remain outstanding. Formatting only;
no tests ran.

2026-11-03: Trip submission uses the selected date's local UTC offset instead
of today's offset. The form rejects nonexistent local times and windows spanning
an offset change because the Go contract accepts one offset. Ambiguous repeated
local times need device/backend design review; tests remain deferred.
