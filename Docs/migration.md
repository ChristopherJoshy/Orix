# Go migration status

Python remains the application runtime until the Go transition and final testing
are complete. The Go implementation now covers authentication, profiles, rides,
matching/routing, groups/chat, notifications, safety/moderation, support, ratings,
account deletion and operational controls. It is **not yet verified end to end**.
See [backend-api.md](backend-api.md) for the current route/parity inventory and
[memeory.md](memeory.md) for remaining work. The sections below also retain the
chronological implementation record; later entries supersede earlier boundaries.

Implemented: auto, travel partner and lift modes; geometric shared-route scoring;
common UTC departure windows; member compatibility and safety checks; lift seat
limits; duplicate-user rejection; deterministic ranking and validation.

Required before cutover: REST/WebSocket contract ports, Firebase verification,
MongoDB repositories and indexes, atomic reservations, rate limiting, durable
notification delivery, alarm repairs, complete Telegram removal,
map integration, reports/moderation and Play policy verification. Road routing
must verify lift itineraries; geometric overlap cannot prove road accessibility.

Additional scope: clean up the codebase and comments, maintain AGENTS.md, verify
latest stable frontend/backend libraries and toolchains, apply compatible upgrades
with lockfiles and test the resulting application.

After code completion and environment setup, run
`GOCACHE=/tmp/orix-go-cache go test -race ./...` from `backend/`. Do not run it
during the current user-requested code-completion gate.

Public registration now accepts any email domain on both client and server.
College request screens, domain startup fetches, bot startup and webhook are
removed. Other Telegram integrations remain and must be removed before cutover.

## Go server foundation

After the testing gate opens, run `go run .` from `backend/` (default port 8080).
Go loads .env (or ENV_FILE); process environment takes precedence. LISTEN_ADDR, CORS_ORIGINS and
RATE_LIMIT_PER_MINUTE configure this foundation. The default differs from Python
port 8000 so both can be tested during migration. Ported routes are GET /health, GET /api/v1/auth/config and
POST /api/v1/auth/verify. Do not point the Flutter app at this
server as its full backend yet. Unsupported routes return 404.

HTTP requests have bounded body/header sizes, connection deadlines and graceful
shutdown. CORS uses explicit origins. Rate limits key on socket peers by default; explicit TRUSTED_PROXY_CIDRS enables
bounded right-to-left X-Forwarded-For resolution through controlled proxies. Limits
remain per instance/address. /health is liveness; /readyz probes configured auth
and writable transactional Mongo topology, not all providers/workers.

## Authentication port

Go uses Firebase Admin v4.22.0 and MongoDB driver v2.9.2, resolved from the official
module paths with @latest during this change and locked in go.mod/go.sum. Firebase
checks token validity, revocation/disabled users and verified email. Any email
domain is accepted. MongoDB uses the existing users collection with a unique
firebase_uid index and atomic upsert. Account IDs remain stable for existing users.

Configure FIREBASE_PROJECT_ID, MONGODB_URI, MONGODB_DATABASE and either Firebase
service-account path/JSON (including base64 JSON) or application default
credentials. When configuration is supplied, startup requires a MongoDB ping and
index creation; otherwise the public foundation can run but sign-in returns 503.
Failed token checks cannot create users. Database failures return generic errors.
Banned and currently suspended users receive account restrictions.

Live SDK/Mongo integration remains required, including concurrent upsert tests
against MongoDB, existing-data migration, identity email collisions, profile photo
sync, suspension expiry persistence and maintenance-mode parity. Only mocked HTTP
contracts have been exercised so far; the Flutter app still needs the remaining
user/onboarding and other APIs before it can switch to Go.

Official references:
- https://firebase.google.com/docs/auth/admin/verify-id-tokens
- https://www.mongodb.com/docs/drivers/go/current/get-started/

## Onboarding and own profile

POST /api/v1/auth/onboard and GET/PUT /api/v1/users/me now use the verified bearer
identity and MongoDB user repository. Onboarding records phone, optional M/F/O
gender, explicit safety consent and consent timestamp. Only the first completion
updates those fields. A phone must contain 10–15 digits with an optional leading
plus; this stores a supplied contact number, not a verified phone identity.

Own profiles include the existing Flutter fields (phone, gender, rider state,
status). Profile updates allow display_name, photo_url and fcm_token only. Push
tokens are never returned. Names are trimmed and limited; photos must use HTTPS
or an uploaded image path. User IDs, roles and safety flags cannot be supplied by
profile updates. Protected operations reject banned/currently suspended users.

Remaining parity: maintenance-mode gate, profile image uploads, ride offer
operations, user deletion/retention, moderation/search, live Mongo update races,
phone verification strategy and full client integration. The Python runtime still cannot be replaced based on these isolated route tests
alone.

## Ride request persistence

POST/GET /api/v1/rides and DELETE /api/v1/rides/{request_id} are ported. New
requests retain Flutter response fields and add mode (auto/partner/lift), driver,
seat count, GeoJSON coordinates and UTC departure bounds. Missing mode defaults
to auto. Validation requires explicit coordinates and timezone offset, nonzero
journeys, valid future windows and server profile gender for female-only requests.
Requests expire at the earlier of window end and fifteen hours after creation.

Mongo startup creates request ID, history and unique active-user indexes. Existing
duplicate active requests must be reconciled before this index can be created;
startup fails on such data rather than silently weakening the constraint. Pending
expired requests are released before insertion. Matched requests hold their slot
until the group lifecycle releases them; that integration is pending. Pending
cancellation is scoped to owner and status. Matched rides require group-aware
cancellation, so this route refuses to cancel those directly.

History currently returns at most 100 newest records. Pagination, matching-status,
indexed route candidates, transactions for group assignment, cancellation races,
full live Mongo tests and map/mode client integration remain required. Ride
creation is persistence-only for now; it does not yet assign a group or send push.

## Matching suggestions work in progress

Added GET /api/v1/rides/status and owner-only
GET /api/v1/rides/{request_id}/suggestions. Suggestions batch-load candidate group
members/routes/profiles, normalize legacy local dates and use the Go matcher.
Responses expose route labels, group counts and scores, not other members' private
coordinates or contact details. Past departure windows are excluded/clamped.
Candidate query currently caps at 200 groups and responses at 20 suggestions;
indexed candidate pagination, geographic pruning, assignment transactions and
blocking remain required. These additions are not a cutover-ready group service.
New regression tests were written; final verification is deferred under the
user's code-completion/environment gate. See testing.md.

## Transactional group reservations (unverified implementation)

POST /api/v1/rides/{request_id}/group accepts {} to create a group or a group_id to
join a suggested group. The request must belong to the authenticated onboarded
user. Mongo snapshot transactions read current profiles/routes, rerun all-member
matching constraints, write group membership and request assignment, and enqueue
a group_updated notification event atomically. Retried assignment returns the
already-assigned group; switching between groups requires explicit leave logic.
Active groups cannot be joined or reset by this operation. Candidate suggestions
now use pending/confirming groups consistently.

MongoDB must support transactions (replica set/Atlas); standalone MongoDB is not
sufficient. Concurrent reservation/retry testing remains deferred. The outbox
worker, group reads/readiness/chat/leave/finish and shared route summaries are
still pending. Lift group creation is supported but lift joins require the road
routing port; geometry alone does not authorize a driving itinerary. Blocked-user
exclusion, plan versioning, expiry/lifecycle and migration of existing groups must
also be integrated before cutover.

## Group read, readiness and lifecycle implementation

Added member-scoped GET /groups and GET /groups/{id}. Group profile lookup is
batched across the returned groups. Other members' phone numbers are available
only for ready members in active/in-progress groups; own phone remains available.
Group detail/list fields preserve the existing client model and add arrival and
completion state. Lists currently cap at 100 newest groups; pagination remains.

Transactional POST group actions now cover confirm, mark-here, start, finish and
leave. Confirmation requires the deadline and at least two ready members to
activate. Arrival marks presence only. Start requires everyone ready/present and
an eligible departure window. Finish requires an in-progress trip; the group is
completed after all members finish. Finishing releases that member's active ride
slot. Leaving before start cancels that member's request and resets remaining
readiness. A departing driver cancels the group rather than stranding passengers.
Membership history remains on completed groups.

Matched requests remain visible after their original search expiry until an
explicit lifecycle action releases them. Group state and notification outbox
records commit together. No Go job auto-completes rides at departure time.

Unverified code: concurrent transitions, retries, privacy boundaries and migration
parity must be checked after the final testing gate. Legacy frontend arrival
messaging still claims completion and needs updating when the client is ported;
it must use the explicit start/finish flow. Legacy Python lifecycle remains
separate until cutover. Discovery, chat/WebSocket, group expiry/cancellation,
route plans, blocked-user rules and notification delivery still need completion.

## Group chat persistence (written, verification deferred)

GET/POST /api/v1/groups/{group_id}/chat now use verified, onboarded membership.
Sending requires an open group and a current unrestricted sender profile; a group
document write serializes sends against transactional membership changes. Chat is
append-only and stores sender display name at send time. A notification outbox
event commits with each new message. Reusing a client_message_id returns the
existing message; different content under that ID is rejected. Clients that omit
the ID can send messages but cannot deduplicate retries across separate requests.

Bodies are trimmed and limited to 1000 Unicode characters. Clients cannot supply
sender identity, message type or timestamps. Group/history reads use a Mongo
snapshot. Pages contain up to 100 messages ordered oldest-to-newest, selected from
the newest matching records. The optional before cursor is returned in
X-Next-Cursor and exposed through CORS. Ordering uses created_at/message_id;
new compound and unique deduplication indexes are installed at startup.

The existing Flutter model can parse the REST response; its send path still uses
WebSocket and must be connected after the WebSocket port. Durable delivery worker,
blocked-user handling, report excerpts, moderation actions and retention still
need integration. Backend source is written without running checks, per the
user's final-testing gate. Do not treat this as real-time chat completion.

## WebSocket delivery implementation

Added /ws with legacy query-token support and preferred Authorization Bearer
header. Firebase validity/revocation, account restrictions and onboarding gate
connection and operations. Chat frames reuse transactional SendChat and return a
chat_sent acknowledgement with the persisted message and optional retry ID.
Unknown or invalid frames receive generic errors. Read size, inbound rate,
idle/lifetime, per-user/global connection counts and outbound queues are bounded.

One MongoDB change stream per server watches committed notification_outbox inserts.
Group recipients are resolved once before dispatch; each socket rechecks current
account and group membership before emitting a chat/group event. Every server
watches the database, so the design does not rely on a single-process broadcast
map to reach sessions on other replicas. Slow consumers disconnect instead of
creating an unbounded queue. The signal context cancels watchers and sockets on
server shutdown. MongoDB replica set/change-stream support is required.

Reconnect handling must fetch REST snapshots: WebSocket events are ephemeral and
currently do not replay events missed during a watch reconnect or client outage.
Push/outbox durable retry is a separate unfinished worker. Preserve compatibility
with connected/pong/chat_sent/new_chat_message/readiness_update/group_finalized/
group_cancelled payloads when updating the frontend. Frontend code still needs
acknowledgement handling, refreshed-token reconnects and retry IDs.

Do not log /ws query strings at a reverse proxy: legacy clients put Firebase tokens
there. Native client migration should prefer bearer headers. Explicit CORS origins
also gate browser upgrades. WebSocket compile, authorization, concurrency,
backpressure, reconnect and cross-replica tests are deferred under the user gate.

WebSocket dependency: github.com/coder/websocket v1.8.15, resolved with @latest.
Official API/release references used during implementation:
https://pkg.go.dev/github.com/coder/websocket
https://github.com/coder/websocket/releases

## Notification API and worker implementation

Ported own notification list/unread count, read-one/read-all and delete-one/all
routes. Mutations are owner-scoped. Added notification ID uniqueness, inbox and
outbox claim indexes. Go startup initializes the Firebase messaging client outside
Auth Emulator mode. Local auth testing records inbox items without contacting FCM.

A context-bound worker claims outbox events with a 90-second lease, records
per-recipient progress, retries transient failures with bounded exponential delay,
and retains failed events after eight attempts. Deterministic notification IDs
prevent repeated inbox rows. Invalid FCM registration tokens are removed only if
the stored token still matches the failed one. Existing group/chat events produce
inbox notifications and generic push content without exposing chat text on lockscreens.
Worker code reads per-user notification_preferences but UI/API preference editing
still needs to be connected.

The scheduler scans active groups every 30 seconds for departures in the next
fifteen minutes. Reminder identity includes group, schedule and recipient. This
allows a new member to receive their own reminder without repeating existing
members' delivery. Rescheduled/cancelled/started groups are checked again at send
time. Android departure reminders use high-priority data messages and the client
local notification path; iOS uses an alert/sound payload. No exact-alarm or
full-screen permission is required.

Delivery is at least once: a process can crash after FCM accepts a push but before
recording receipt. Collapse/tag IDs reduce visible duplicates; they cannot prove
exactly-once sound or exact-time arrival. Preference UI/API, event recipient
semantics, operational failure monitoring/replay, reminder date backfill, stale
job cleanup, final device tests and full notification-state client integration
remain unfinished. No checks or live sends ran under the user testing gate.

## Reports and moderation implementation

POST /reports files a categorized safety report; GET /reports returns the caller's
own reports/status. The accused must exist. Optional group context requires both
users' current membership before capturing up to 20 messages as immutable evidence.
Evidence, moderator identity and private review notes are excluded from user-facing
responses. Reporting yourself is rejected.

GET /admin/reports returns pending reports with evidence; POST
/admin/reports/{report_id}/resolve supports warn/suspend/ban/dismiss. Admin access
requires an authenticated database admin role and X-Admin-Secret matching the
configured server secret. A fresh role/restriction check occurs in the mutation
transaction. Moderators cannot review their own reports or reports against them;
account restriction of other admins is refused pending an escalation workflow.

Resolution, audit record, applicable account restriction, affected group/request
cancellations and notification events commit transactionally. Replay of the same
admin/action is idempotent; conflicting resolutions are rejected. Reporters receive
a generic reviewed notification, and warnings enqueue an account safety notice.
Direct-recipient notification events are now supported by the worker. No Telegram
calls exist in these Go paths. Supporting report indexes are installed at startup.

Still required: frontend report/history/admin review UI, blocking and report abuse
limits, notification preferences, moderation escalation, access/audit retention and
pagination. Live transaction/privacy/concurrency tests and Play UGC requirements
remain unverified under the final-testing gate. The old Python Telegram paths
remain until the full migration is tested and cut over.

## Blocking and notification preferences

Added GET /users/me/blocks plus PUT/DELETE /users/{user_id}/block. Only the caller's
outgoing block list is exposed. Blocks operate in both directions for matching and
communication, but unblocking removes only the caller's own block. No block notice
or reverse block list is sent to the target.

Block transactions touch both profiles and cancel common open coordination groups,
release their matched requests, and write generic cancellation events. Joining also
touches relevant profiles so concurrent block/join operations conflict and retry.
Unblocking never restores a cancelled group. Suggestions exclude groups containing
blocked peers; joins recheck relationships; chat history hides blocked authors,
new sends are refused in blocked shared groups, and chat WebSocket/push delivery
rechecks blocking. Other members' contact numbers are hidden across block relations.
Supporting directed/reverse indexes and local test-account block cleanup are added.

GET/PUT /users/me/notification-preferences stores ride_reminders, chat_messages,
group_updates and promotions. Only allowlisted boolean fields can change. Ride,
chat and group push default on; promotions default off. Inbox records remain
available when push is disabled. The worker already consults these values; the
frontend preference screen still needs API wiring. Expired suspensions no longer
suppress worker delivery indefinitely.

All new safety/preference code is unverified. Final checks must cover transactional
block/join/send races, in-flight notifications, history/privacy and contact
visibility, pagination, cancellation UX and notification preference consistency.
No checks or emulator execution ran under the user testing gate.

## Support ticket implementation

Ported POST/GET /support/tickets, GET /support/tickets/{id}, and message/close/reopen
POST actions. Responses retain the existing Flutter support model. User access is
owner-scoped; the ticket author/sender and admin status come from verified identity.
User-supplied priority cannot escalate a ticket; priority is normal at creation.
Input types and lengths are bounded. New tickets permit five open threads and ten
creations per UTC day, serialized by user document writes. User reopening also
respects the open-thread limit. Conversations cap at 200 messages without silently
truncating stored history.

Added admin support list/detail plus message/close/reopen/resolved/in_progress
actions behind the existing role and admin-secret gate. A fresh database role and
account check occurs in mutations. Admin replies display the generic support
identity; actual staff actor IDs remain in audit records. Replies and state changes
commit with audit and generic ticket-update outbox events, keeping reply text out
of push payloads. Go support has no Telegram calls.

Lists cap at 100 newest tickets and project out conversation bodies, returning an
empty messages list for client compatibility. Detail loads the stored conversation.
Indexing supports ticket identity, owner/day quota and queue status. Still required:
frontend admin tooling/notification navigation, cursor pagination, retry IDs,
moderation escalation/appeals for restricted accounts, retention and final
owner/admin/concurrency tests. Support code has not been compiled or exercised
under the testing gate. Legacy Python support remains until tested cutover.

## Ratings and release metadata implementation

Ported rating submission, pending group IDs, pending partners and average lookup.
Eligibility requires membership in the same completed trip, or an in-progress trip
where both partners explicitly finished. Ratings are unique per group/rater/target;
identical retries return the original record and changed retries conflict. Client
identity/type/time cannot be supplied. Pending queries batch existing ratings and
profiles. Average lookup is restricted to self or shared-group partners and uses
indexed persisted ratings rather than an unbackfilled cached total. No reviewer
identity or comments are returned by the average/submission response.

Added GET /app/version and admin PUT /admin/app/version. Stable major.minor.patch
versions support optional numeric build metadata; ordering ignores build metadata.
The minimum cannot exceed the current version. Store links must use official
Google Play/Apple HTTPS hosts and are absent by default instead of fake URLs.
Missing database config uses the staged app version and minimum 2.0.0. Admin
updates require role/header access and record an audit event. Stored malformed
legacy version config returns a repairable service error rather than silently
forcing clients to a placeholder store.

Unverified under the testing gate. Rating notifications, client finish/rating flow,
retention and paging remain to integrate. Existing duplicate ratings must be
reconciled before unique indexes can install. Version gating requires final
release/store configuration and client checks; these APIs do not prove publication
readiness. Older frontend store fallbacks still need removal during client cleanup.

## Account deletion implementation

DELETE /users/me requires a valid Firebase identity with auth_time within five
minutes, including token revocation checks. This operation remains accessible to
app-restricted users with valid recent identity. It queues a durable deletion job,
sets the profile to deleting, removes its push token and cancels shared open groups
and ride slots atomically. Other authenticated actions reject deleting/deleted
profiles. Client reauthentication/sign-out handling is still required.

A leased worker retries Firebase identity deletion and app cleanup; it does not
abandon deletions at a retry cap. Retry delay is bounded at one hour. Known private
records (profile, rides, notifications, tickets, ratings, blocks, authored chat)
are deleted or redacted. Shared group membership/labels and affected report
excerpts/identifiers are removed or minimized. Actor-linked audit free text is
removed instead of assuming it is anonymous. Completed job identity material is
removed, with a seven-day TTL on job receipts. Partial cleanup is designed for
retry, but that behavior remains unverified.

Required before publication: complete all-data/storage inventory (uploaded avatars,
legacy logs/attachments, nested/third-party records and backups), retention/privacy
text, restricted/disabled-account appeals, frontend reauthentication/deletion UX,
external web deletion request path, operational monitoring and deletion/retry tests.
Firebase deletion is not reversible. No deletion or service calls ran during this
implementation. Code does not yet prove Google Play account deletion compliance.

Official requirement:
https://support.google.com/googleplay/android-developer/answer/13327111

## Road-verified lift itinerary implementation

Added an OSRM-compatible routing client configured by ROUTING_BASE_URL. HTTPS is
required except for loopback development services; requests are bounded, redirects
are refused and there is no automatic public-demo endpoint. Two requests fetch a
road matrix and final geometry. Coordinates are shared only with the operator's
configured routing provider and group members; provider/privacy disclosure remains
part of release setup.

Lift reservations now plan legal pickup/drop orders using the road matrix with the
driver's endpoints fixed. Four-traveler groups have at most 90 legal orders. The
search enforces pickup before drop, simultaneous seats, and driver/passenger road
detour bounds (3 km / 10 minutes). Missing/unreachable routes and excessive road
snapping are rejected. Geometry and stop sequence are persisted on the group and
returned only to members, hidden for cancelled/expired or blocked relationships.
Passenger departures invalidate old plans; lift reactivation replans rather than
retaining the departed person's coordinates. Account deletion clears stored plans.

No routing calls or checks ran. Required final work includes pickup-arrival time
window constraints, exact-leg validation of matrix-vs-route differences, route
cache/version/revalidation, route provider terms, traffic assumptions, staging
availability and real device route/map presentation. External calls currently occur
inside reservation transactions and need performance/failure validation before
cutover. Auto/partner road visualization still needs integration. Configuring the
provider alone does not prove road itinerary correctness or publication readiness.

Official routing protocol used:
https://project-osrm.org/docs/v5.24.0/api/

### Lift timing and route-leg follow-through

The road planner now shifts each pickup window backward by cumulative travel time
to compute an admissible driver departure range. Candidate stop orders without a
common feasible departure are rejected before geometry fetch. Final road legs are
then checked again for pickup timing and each passenger's detour, with total-vs-leg
consistency bounds and finite numeric limits. Stored plans include earliest/latest
departure. Reservations and readiness use those bounds; trip start cannot occur
before departure. Lift geometric ranking may propose disjoint literal pickup windows
because the road stage accounts for the time to reach each traveler.

Added a two-minute routing response cache, capped by entry count and 32 MB total,
with large responses excluded from caching. Stale lift plans are recalculated on
activation. No checks/provider calls ran. Final work includes provider/version
failure behavior, traffic and waiting-time assumptions, cache/concurrency tests,
exact-geometry/privacy verification and device UX. Legacy group date/time_window
fields must yield to departure_start/road_plan bounds in the new client map flow.
Earlier notes listing pickup-window/exact-leg checks as pending are superseded by
this implementation, but their verification remains pending.

## Automatic matching worker implementation

Pending requests now enter a context-bound Go worker, polled in bounded batches
with oldest-due ordering and per-request leases. Current compatible groups are
ranked, then at most eight candidates are attempted through the transactional
reservation path. If none succeeds, an unconfirmed one-person waiting group is
created so future travelers can discover/join the same route. Lift road planning
still gates reservations where a driver itinerary is needed.

A second bounded pass revisits unconfirmed one-person pending groups. It can
transactionally move that member/request into a compatible destination, cancel
and empty the old group, and write the normal group-update event. This repairs
fragmentation when simultaneous requests initially create separate waiting groups.
Confirmed or multi-person source groups are never automatically moved. Profile
safety-version writes, group conflicts and current membership checks apply on each
reservation retry. Supporting due-work and group identity/candidate indexes are
installed at startup. Group origin markers allow account cleanup to find merged
source summaries as well as current memberships.

No checks, matching jobs, database writes or provider calls ran. Final work still
includes stale/expired group lifecycle processing, candidate pagination/geographic
pruning at scale, work failure visibility, worker failover/concurrency and map
client transitions. Fixed batch/candidate bounds need workload validation before
production claims. Existing pending/request data needs migration/backfill before
cutover. Prior persistence-only descriptions are superseded by this unverified
worker implementation.

## Expiry and worker operations implementation

Added a bounded lifecycle worker that expires timed-out search requests and groups.
Pending groups use an explicit search expiry; confirming groups use the readiness
deadline; pending/confirming/active groups also expire after their admissible
departure window. A one-person remainder after leaving gets a bounded renewed
search expiry. Group expiry, matched-slot release and cancellation event commit
together. No timer marks a ride completed or creates rating eligibility, and
in-progress rides are not expired automatically. Legacy groups without UTC/window
metadata still require backfill.

Added role/header-protected /admin/jobs, /admin/notifications/failed and
/admin/notifications/{event_id}/retry. Status exposes queue counts, attention
counts and worker heartbeats with generic errors, not tokens/push payloads. Failed
event retries preserve recipient receipts and record the admin action. Lifecycle,
matching, notification/reminder and deletion loops record worker status. Supporting
group expiry indexes are installed at startup.

No worker runs or checks were performed. Clock boundaries, readiness/expiry and
leave/merge races, timer backfill, retry/privacy behavior, multi-replica visibility,
heartbeat overhead and ops UI still need final integration/performance validation.

## Avatar port implementation

Added POST /users/me/avatar (the existing multipart file contract) and authenticated
GET /users/{user_id}/avatar. Uploads sniff/decode JPEG/PNG/WebP, cap input at 2 MB,
4096 pixels per side and 16 million pixels, limit parallel decodes, and re-encode a
512-pixel JPEG thumbnail on white. Re-encoding strips embedded metadata. Upload and
profile URL update commit together in the existing separate user_avatars collection.
A unique owner index prevents duplicate avatar records.

Reads require self or shared-group access, respect blocking/restrictions and return
private caching/ETag headers. Profile updates accept only the caller's own relative
avatar path rather than arbitrary /uploads paths. The upload body limit is scoped
to this endpoint. Account deletion and internal test-account cleanup now include
user_avatars. Existing Mongo avatar records remain readable; their metadata and
codec/dimension migration still need backfill review.

Frontend image components must send auth headers for API avatar URLs during the
client pass; the previous anonymous image endpoint is intentionally not retained.
No uploads, decoding checks, compilation, integration or emulator work ran. Final
resource/privacy/authorization and legacy data migration tests remain required.

## Known-partner search and maintenance

GET /users/search now searches a literal name prefix (2–100 characters) only among
people with shared ride-group history. It returns at most 20 identities, display
names and photos. Email stays an empty compatibility field; no global email/name
directory is exposed. Blocked partners remain reportable, with their image omitted.
The member-history index supports the scope query. Name-regex input is escaped.

Added GET /app/status, admin GET/POST /admin/maintenance and stored maintenance
state with audit. Admin mutation requires role/header access and rechecks the
current role in its transaction. Maintenance gates new ride/onboarding/profile
mutations and reservation transactions; automatic matching pauses as well.
Existing group coordination/chat, reporting, blocking, support, notification
management and deletion remain available. Push, cleanup and expiry workers keep
running. Already-running operations may complete before a maintenance change;
this is an activity pause, not a database freeze. Public status and client UI still
need to be connected during frontend work. Historical global-maintenance screen
behavior must not hide safety/deletion controls.

No checks, database queries or service execution ran. Final authorization/search
scope, query workload, maintenance races, availability and client behavior remain
unverified.

## Legacy data migration command (written, not executed)

`go run . migrate-data` reads MONGODB_URI/MONGODB_DATABASE directly and emits a
JSON plan without applying writes by default. It backfills ride UTC bounds from
explicit local date/window/offset, fills missing passenger modes as auto, derives
missing expiry, releases already-terminal/expired request states, and derives
missing group timing/mode/search/deadline/origin metadata. It preserves existing
IDs and does not convert separate legacy rider-profile offers into fake lift data.

Ambiguous/missing time zones, invalid live timing, missing/ambiguous member routes,
mixed modes, unplanned live lifts, multiple active requests and required unique-key
conflicts block all writes. Legacy ride/group TTL indexes are flagged because they
could silently delete history/active records outside the new lifecycle. Conflict
records must be reconciled explicitly; the tool never picks a winning active ride
or drops an index on its own. Diagnostic samples cap at 100 with the total issue
count reported; source collection reads cap at 50,000 records and refuse larger
single-run migrations rather than silently processing a subset.

Only after code completion and environment setup, stage and review the plan.
Application additionally requires MIGRATION_APPLY=true, MIGRATION_DATABASE exactly
matching MONGODB_DATABASE, and MIGRATION_WRITERS_STOPPED=true. All old/new writers
must actually be stopped; the environment assertion does not stop them. Each write
compares source fields and updated_at against its planned snapshot. A conflict
stops further writes. Application is incremental/idempotent, not a whole-database
transaction: an interrupted apply can be partially complete and must be replanned.
No backups or duplicate trees are created. User already maintains backups.

No migration queries, dry runs or writes were performed under the testing gate.
Large-dataset batching, canonical coordinate/profile validation, legacy rider offer
migration, old schema cases and end-to-end cutover/index verification remain.

## Environment loading and first staff setup

The Go entry point now loads .env (or ENV_FILE) before the server or CLI commands;
process environment takes precedence. This supersedes earlier process-only notes.
The parser never executes/interpolates shell text and does not echo secrets.

`go run . bootstrap-admin` requires an existing enabled/email-verified Firebase UID,
APP_ENV=local/staging/production, explicit BOOTSTRAP_APPLY=true and ADMIN_DATABASE
matching MONGODB_DATABASE, plus a 32+ character ADMIN_API_SECRET. Local mode uses
the existing demo-orix/loopback guards; cloud modes reject Auth Emulator settings.
It does not create Firebase users or rely on email matching alone. A singleton
bootstrap guard and transaction prevent assigning different first administrators
concurrently. Existing unrelated administrators close the bootstrap path; repeat
selection of the same administrator is idempotent.

PUT /admin/users/{user_id}/role now handles authenticated staff changes with the
role/header gate and fresh transactional role checks. Changes are serialized and
audited; removing the last active admin is rejected. Own sign-in/profile responses
include the server-derived role for the forthcoming client admin navigation.
No public sign-up can set that field. Last-admin identity loss/deletion recovery
still needs an explicit operator recovery procedure; do not use bootstrap as a
routine privilege elevation API.

No env/credential file was loaded by running the application, and no bootstrap,
role changes, compile checks or service calls ran during this implementation.

## Ride editing implementation

PUT /rides/{request_id} accepts the validated create-ride body and derives ownership
from the authenticated user. Pending/matching requests can be edited. Since the
matcher creates waiting groups quickly, an unconfirmed one-person pending group
can also be edited without losing its identity. Shared/confirmed/started or expired
requests reject this operation; shared changes require an explicit leave/new-trip
flow rather than silently changing other members' plans.

The transaction rechecks current account/safety state, serializes the owner profile,
updates request/group route/time/mode/expiry metadata, clears stale matching leases,
replaces lift geometry when applicable and emits the group update. Request ID and
original creation time remain stable. Matching resumes with the new route. Existing
source/group conflicts retry through Mongo transaction semantics.

No checks, route calls or writes ran. Frontend editing UX, retries/concurrency,
mode switches, expiry boundaries and validation/error mapping remain to integrate
and verify after the code-completion testing gate.

## Restricted-account appeals and restoration

User support routes now authenticate through a narrowly scoped identified-user
wrapper that permits app-banned/suspended identities. Other protected APIs keep the
normal restriction gate. Deleting/deleted identities are still rejected. Restricted
users can read their own support history and create/mutate appeal-type threads;
repository transactions recheck restrictions and do not permit ordinary ticket
mutations as an alternate privilege path. Existing ticket quotas and owner scope
apply. A valid enabled Firebase identity remains required; provider-disabled or
identity-lost accounts still need the external support/recovery process.

POST /admin/users/{user_id}/restore requires admin role/header and a review note.
The transaction rechecks the moderator, refuses self/admin/deleting restoration,
restores only app ban/suspension state and records audit plus a generic notification.
A repeated restore of an already-active account is idempotent. Cancelled groups and
ride slots stay cancelled; restoration never revives prior coordination or changes
resolved report history. This does not re-enable Firebase-disabled identities.

Frontend restricted-account appeal/history and staff restoration UI remain to wire.
No appeal submissions, restores, checks or service calls ran. Final auth isolation,
quota/ownership, restoration/deletion races and end-to-end user recovery require
verification after the code-completion gate.

## Main list cursor pagination implementation

Ride history, member group lists, notification inbox, own/admin report queues and
own/admin ticket lists now accept limit=1–100 and a before cursor. Response bodies
remain unchanged; X-Next-Cursor is returned only when a limit+1 fetch proves there
is another page. CORS already exposes that header. Cursors carry timestamp/ID and
an endpoint/caller scope; data authorization is independently enforced on every
query. Detail reads and internal worker repository calls keep their existing shape.

Queries use descending timestamp/ID keysets, not skip offsets, with supporting
owner/status/time/ID indexes. Group profiles are batched only for the visible page.
Unread notification counts continue to cover the inbox rather than the current
page. Ticket lists retain lightweight messages-free responses and sort by updated
activity; concurrent updates can move tickets between pages, so the client must
deduplicate IDs and refresh the first page on new activity. This is not a multi-page
snapshot guarantee.

No query/check/test ran. Frontend load-more state/header handling and final malformed
cursor, tied-time, empty/end-page, authorization and mutation-during-paging tests
remain. Earlier fixed-100 list cutoff notes are superseded for these endpoints;
candidate matching and other specialized queues have separate remaining bounds.

## Bounded candidate rotation, terms and admin operations

Background matching now keeps a private candidate_scan_after group-ID cursor on
each request. Each pass scans at most 200 groups plus one lookahead, advances past
incompatible groups and wraps at the end. Waiting singleton reconciliation uses
the same progress. Edits clear it. Requests filter modes and obvious non-lift
UTC-window incompatibility before loading batched member routes/profiles. Legacy
auto modes/missing UTC metadata remain eligible for strict member validation;
lift windows are verified by road routing, not a literal common pickup time.
No geographic pruning or global-best guarantee is claimed. Public suggestions
still return at most 20 matches from the first bounded batch; worker rotation is
internal and does not add public candidate pagination. Transactional reservation
remains authoritative for all compatibility/capacity/safety checks.

Go now supports own terms-status and explicit accept-terms. The server accepts
only current version 1.0, preserves acceptance time on identical retries, and
records acceptance/audit atomically after a fresh restriction check. Onboarding
safety consent never automatically accepts legal terms. Version must change with
the actual displayed/published policy. Final frontend gating and policy review
remain; this endpoint alone does not establish release compliance.

Admin endpoints now include exact email/user-ID lookup (ambiguous legacy emails
require stable ID), bounded/paged audit summaries, aggregate stats, an explicit
group summary, and transactional group cancellation with a required note. Every
route requires a current Firebase administrator and the admin secret. Cancellation
rechecks/touches the actor inside its transaction, cancels live groups/releases
matched request slots, queues notification delivery and writes an audit. Repeated
cancellation is idempotent; completed/expired groups are not rewritten. Raw Mongo
profiles/group documents, push tokens and private road plans are not returned.
Audit lists omit free-text notes, before/after snapshots and actor emails.

The legacy migration planner is now version 2 and includes admin_logs timestamp
backfill from a valid existing timestamp to created_at, retaining original data.
Missing timestamps or non-ObjectID IDs are conflicts for explicit review rather
than fabricated action dates. Existing stopped-writer/compare-and-set requirements
apply. No migration or provisioning has run. All code in this section is written
and formatted only; compilation, tests and integration remain deferred.

## Deployment boundary and data lifecycle follow-through

Added validated trusted-proxy CIDRs and bounded forwarded-client resolution;
no trusted configuration means forwarded headers remain ignored. Added /readyz
with generic 200/503 and a short primary/topology probe. Startup now rejects
standalone/nonwritable Mongo before indexes. Auth-emulator local Mongo also needs
a replica set. Health probes stay outside request quotas. Distributed limits and
shared-NAT quota fairness are not solved by proxy resolution.

Centralized API/CLI audit insertion in WriteAudit: every entry now has a unique
log_id plus created_at/timestamp. This fixes compatibility with the legacy unique
log_id index, where omitted values would otherwise cause duplicate-null failures.
Migration planning also backfills deterministic missing log IDs and detects
planned collisions, while preserving valid historical IDs and dates.

Deletion now redacts deleted moderator messages in other owners' support threads,
reported-user and resolution links, embedded report evidence, own hashed delivery
receipts, related trip-notification location fields and the final group-origin
identifier. Bootstrap markers referencing the Firebase identity are removed.
Dependent group redaction occurs before clearing the last discoverable origin link
so retries can repeat work. Cleanup retry errors now reach worker health rather
than appearing successful after only requeueing. See data-lifecycle.md for stores,
legacy Redis/Telegram and provider-retention boundaries. These additions remain
uncompiled/unexecuted under the testing gate; no deployed deletion claim is made.

## Public deletion flow and Go runtime image

Added embedded /account-deletion HTML/CSS/JS in the existing Go app directory.
The page loads only Firebase Auth modules (official CDN 13.1.0), uses Google popup
sign-in with memory-only sessions, displays the chosen account and requires explicit
confirmation before the existing recent-authenticated DELETE endpoint. It explains
accepted/background cleanup rather than claiming immediate completion. It offers
a configured support contact for provider-disabled/recovery cases; actual mailbox
monitoring and ownership review remain operational work. It does not weaken app
restriction handling or Firebase verification, and it sends no automatic email.

Public web identifiers are projected by a scoped config route, separate from
Admin SDK credentials. Partial configuration is rejected, and enabled Google web
sign-in requires a real support email and HTTPS privacy URL. Missing setup disables
sign-in; without any request method the page returns 503. CSP/no-frame/no-referrer
and popup-compatible opener headers are written. Browser/provider verification,
HTTPS hosting, actual policy text, in-app integration and Play Console setup remain.

Added a Go-only multi-stage Dockerfile and secret-excluding .dockerignore in backend.
Go minimum/builder now 1.27.3, verified from official releases/container tag sources.
Final runtime is a nonroot scratch image with CA certificates and the embedded page;
legacy Python source remains until transition testing. No builds or service/provider
operations ran. See deployment.md for configuration, sources and deferred scenarios.

## Ownership-reviewed support deletion backend

Added support-account-deletion CLI with default read-only planning and explicit
APP_ENV/database/actor/target/case selection. The Firebase administrator must be
enabled/verified and have a current active app role. Apply requires an explicit
ownership-reviewed assertion; that flag is not automated proof. No HTTP endpoint
or public UID/email deletion bypass was added. Operators must verify ownership
outside the app before selecting the target. See deployment.md.

Refactored the app/web cancellation/job transaction into queueAccountDeletion so
support applies the same invariants. Support application rechecks/touches actor
state, queues cleanup, attaches one opaque case and writes an audit atomically.
Identical case retries avoid duplicate review audits and conflicting cases are
refused. Orphan Firebase identities use identity-only jobs with no profile upsert.
Disabled/missing target identities remain eligible for proper reviewed cleanup.
Completion clears review metadata along with Firebase UID. Normal workers perform
the asynchronous deletion. No plans, provider/database calls, command executions,
builds or tests ran; the shared refactor and all scenarios still require final QA.

### Shared transaction authorization for administrator mutations

Consolidated current-admin/restriction checks into authorizeAdminMutation, which
also writes an authorization version on the actor's profile inside the mutation
transaction. A snapshot-only role read could otherwise coexist with another
transaction demoting/deleting the actor. The shared write forces a conflict/retry
against those profile changes. Wired report resolution, restoration, staff role
changes, support admin replies/status, maintenance/version controls, delivery retry,
group cancellation and support-origin deletion application. Existing domain
conflict/self-action/last-admin rules remain in their owning modules. Read-only
admin endpoints still use the authenticated admin middleware; this is transaction
mutation serialization, not a replacement for Firebase/header authorization.
Source is formatted only; role/deletion race tests and response contracts remain
part of the deferred final QA phase.

## Geographic candidate pruning and bounded matching data

Added versioned private route_search regions to group creation/join and singleton
route editing. Regions conservatively enclose each great-circle route and corridor
padding using endpoint balls of half the route length plus padding. Longitude
widens at the dateline/poles; routes over 1000 km skip geographic pruning. These
are coarse query bounds, not meeting points, road itineraries or match approval.
Every potential match still runs full route/member/safety/time/capacity checks and
transactional reservation. Region metadata is not returned in group/suggestion APIs.

Candidate lookup now excludes non-overlapping regions before member/profile batch
loads. Missing/old-version metadata remains eligible for full evaluation. Joins
anchor the region to a current member, leaving clears it, and deletion redacts it
along with shared route/location data. Policy/bound changes must bump the region
version so old bounds do not silently constrain the new matcher. A supporting
compound index is written; its real planner/latency characteristics remain unproven.

Migration plan version 3 derives waiting-group regions from unambiguous current
member routes, reports invalid coordinate conflicts, and retains source comparison
before apply. It has not run. Legacy groups are not excluded just because they
have not been backfilled. Existing timestamps/audit-ID backfills remain included.

Candidate loads now project only IDs, member state, labels, route matching fields
and safety/onboarding fields; push tokens/phone/profile payloads are not loaded.
Member IDs are deduplicated before profile batching; matched routes are bounded
against group capacity and malformed excess data produces an error rather than
unbounded allocation. Existing member onboarding is rechecked during suggestion
construction and reservation. Startup now explicitly enforces unique user_id as
well as firebase_uid, matching the legacy identity invariants.

The background matching passes now retain failures after releasing leases and
report them to worker health. Singleton reconciliation no longer discards provider/
matching errors; stale retry hints clear on success, and request failures do not
skip the singleton pass when time remains. Normal incompatible/contended candidate
reservations remain misses; unexpected failures are retained when none succeeds.

Added spherical containment, local pruning, dateline/pole/long-route and invalid
input test source. Tests are written but not executed under the code-first gate.
No compile, database query/explain, benchmark, migration or runtime validation has
run. Final QA must establish no eligible-match loss, correct edit/leave/deletion
invalidation, legacy fallback, index behavior and worker retry/health semantics.

## Group confirmation and reminder queue throughput

Confirm now refuses one-member waiting groups and elapsed departure windows. The
migration planner normalizes ready/here/finished flags on legacy ready pending
singletons so they remain matchable. No source deletion or migration was executed.

Split notification scheduling from four leased delivery workers, including one
reminder-only worker. General workers prioritize reminder events; batches of up to
400 stable-ID scheduling upserts replace one round trip per recipient. Added queue
sort/index support, bounded push calls, per-recipient schedule/member rechecks,
Android reminder TTL and APNs expiry/collapse, plus stable notification IDs and
server title/body in payloads. Moved expired-attempt lease recovery into the
scheduler and propagated its errors. Heartbeats are throttled on normal traffic.

See notifications.md for actual behavior/limitations and the still-needed lift
per-pickup timing work. No concurrency, provider, performance, compilation or
device verification occurred. Updated root/backend READMEs to public-app/migration
entry points linking canonical Docs instead of stale campus/Python/bot-only claims.

## Personal pickup times and lift start lifecycle

Road stops now store validated cumulative offset_seconds from final route legs.
Final confirmation requires usable lift offsets (replanning a missing/old plan),
loads bounded own pickup labels, and persists private per-member reminder times.
Groups expose mode and own my_pickup_time/my_pickup_summary, not the private array.
Shared-auto/partner keep their common departure; legacy lifts are never assigned
an invented driver-time pickup.

Scheduling/delivery now use per-member indexed times, including not-yet-finished
lift passengers after driver start. Stable event IDs include both driver/pickup
milliseconds; delivery rechecks current schedule/member/completion state and uses
own pickup time/label for payload/expiry. Full private road geometry is no longer
loaded for notification rendering. Join/edit/leave/expiry/deletion invalidate the
persisted schedule. Completed-trip own pickup metadata can remain in history.

Lift start requires the driver actor, all readiness and driver origin arrival,
rather than every passenger's arrival at a later pickup. A materially late start
(at least one minute) shifts driver departure/personal pickups in the transaction;
sub-minute jitter is tolerated. Current latest-departure/window checks still apply.

Migration plan is version 4: backfill derivable active/in-progress schedules or
report missing verified lift timing for explicit route review. Added pure timing/
stale identity test source; none executed. No provider queries, database work,
compile, checks or emulator runs occurred. Client mode/lifecycle/pickup metadata,
old Go queue transition, actual traffic/late timing and all delivery races still
need final integration/verification before any all-mode alarm reliability claim.

## Final-retirement operator code and frontend contract migration

Added read-only-by-default retire-legacy-data with explicit environment/database/
admin selection. Post-test apply requires stopped-writer/tested-transition
assertions and a real QA reference. Fixed obsolete bot/domain record allowlists
and legacy rider-profile fields are changed in authorized/audited 500-record
transactions; no whole database, active records, Redis, backups or source files
are deleted by this command. It has not run; see deployment.md for boundaries.

Admin mutation role revocation now has its own error mapped to 403 rather than
being misreported as a domain conflict or temporary server failure. Domain 409
rules remain separate. No transaction/auth HTTP checks ran.

Started client contract migration after the Go API/worker/operator implementation
work. Removed the hardcoded insecure production endpoint; debug uses Go port 8080,
while profile/release require an explicit validated HTTPS origin. Invalid builds
show service unavailability before Firebase/network setup. Time parsing now respects
explicit RFC3339 numeric offsets instead of appending an invalid extra Z.

Group models now parse mode, personal pickup/departure bounds, road geometry/stop
offsets and arrival/completion/profile fields. GeoJSON coordinate ordering/ranges
are explicit. Confirmation refetches full state rather than dropping new fields
through an optimistic legacy copy; active-group responses refresh when readiness/
status changes even if IDs stay identical. Leave now uses Go's POST contract.
ApiResponse exposes X-Next-Cursor for upcoming paged histories. Group API now also
exposes departure_end for client start/window decisions. Broader mode/map/lifecycle/
WS/history/safety/client integration is still incomplete. No frontend/backend
build, analyzer/check/test, environment execution or emulator QA ran.

## Frontend group lifecycle and acknowledged realtime chat

Rebuilt group state around account/chat scopes and stale-request guards. Account
switches clear trip/chat data; late responses cannot repopulate another account.
Group changes refetch authoritative state, and explicit controls now separate
readiness, arrival, start, finish and leave. Lift driver/passenger start rules and
personal pickup metadata reach the detail UI. Transient refresh errors stay visible
with retry instead of forcing navigation away. Active-group queries omit a member's
finished trip while preserving member/history detail access.

Per the user's explicit follow-up, chat sends prefer WebSockets and await persisted
chat_sent acknowledgements. Each message has a random stable client_message_id;
an eight-second missing acknowledgement/disconnect falls back to REST with the
same ID. Retry keeps that ID, pending/unconfirmed messages remain visible, and
matching never uses message text. Pagination/load-earlier and nested live-message
handling are wired. Reconnect refreshes REST snapshots to recover missed events.
Account identity is rechecked before fallback so a previous account's pending
message cannot be sent with the newly selected account's token.

Replaced stale-token reconnects with refreshed Firebase tokens, bounded/jittered
retries, heartbeat checks, generation-safe closure and app-resume renewal. Native
handshakes use Bearer headers; browser handshakes offer an auth subprotocol while
negotiating only orix.v1. Go no longer accepts token query parameters. No tokens
are placed in URLs or echoed as the selected protocol. Native/browser transport
code is in the existing services folder; no replacement tree was created.

The in-app reminder now refreshes membership/pickup state before opening, rejects
changed/past/closed trip alerts and offers ordinary dismissal/details/arrival.
Removed forced immersive UI, repeating sound/haptics and false arrival-completion
claims. Local reminders use per-group IDs and actual remaining pickup minutes;
iOS avoids adding a local alert when APNs already supplies one. Stable server IDs
replace FCM-attempt IDs for foreground deduplication. Further background duplicate,
permission, token-ownership and navigation work still needs the full client pass.

Source was formatted with the installed SDK binary after the snap wrapper refused
to launch. No analyzer, compile, tests/checks, server/provider/database operations or
emulator runs occurred. Maps/matching/home/history/safety/account/release integration
remains incomplete; these implementations do not prove realtime/alarm reliability.

WebSocket/matching continuation: client home, matching and request creation now
use the staged Go group/suggestion/edit flow. A positive pickup window replaces
the rejected equal-time payload. Readiness events refresh state and singleton
requests can still discover groups. Route previews and chat concurrency/lifecycle
changes are written; only formatting ran. Picker, remaining integration, latest
dependency reconciliation, release work and all final QA remain outstanding.

Location picker continuation: replaced campus recommendations/default center,
public Nominatim autocomplete/reverse lookups, fake map preload, deprecated tile
provider and uncredited satellite overlays. Picker uses an explicit Go place
search adapter with injected transient state, optional tap-triggered GPS, manual
coordinates, editable meeting label and prior-endpoint initialization. Shared map
tile/attribution config now serves picker and previews. Backend provider setup is
optional but required for name search; no provider was configured or contacted.
Legacy unused map_service.dart and manifest dependency remain retirement candidates
after the complete transition passes tests. Only formatting ran.

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

2026-11-03: Ride history now exposes the existing Go cursor pagination and
shows completed trips with a success status. Formatting only; tests deferred.

2026-11-03: Onboarding accepts international-format phone numbers instead of
forcing +91 and removes an inaccurate OTP comment and unverified tracking/crash
claims. Formatting only; no runtime or emulator checks ran.

2026-11-03: Profile edit now checks backend save success before reporting it,
guards account changes after avatar upload, and retains a committed avatar if
the subsequent name update fails. Formatting only; no app checks ran.
