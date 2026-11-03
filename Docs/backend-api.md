# Go backend contract and legacy replacement inventory

Current implementation inventory, 2026-11-02. These routes are written, with final
compilation/integration testing deferred by the user's instruction. Python remains
the runtime until transition testing passes. Do not treat presence here as proof
of working behavior. API prefix is `/api/v1`, except `/health` and `/ws`.

## Authentication and access

Protected routes require Firebase `Authorization: Bearer <ID token>`. Verification
checks revocation, disabled identities and verified email. No college-domain gate.
Identity, role, safety attributes and group membership are server-derived.
Administrators additionally supply `X-Admin-Secret`; the secret alone grants no
role. Ordinary access rejects restricted/deleting identities. Support-only access
allows app-banned/suspended users to appeal; deletion requires recent sign-in.
Internal test accounts use actual emulator Firebase tokens, never an API bypass.

## Routes

| Area | Methods and paths after the prefix | Behavior |
| --- | --- | --- |
| Public status | GET `/auth/config`, `/app/status`, `/app/version` | Registration, maintenance, version/store metadata |
| Sign-in | POST `/auth/verify`, `/auth/onboard` | Firebase token verification; explicit onboarding safety consent |
| Profile | GET/PUT `/users/me`; POST `/users/me/avatar`; GET `/users/{user_id}/avatar` | Whitelisted own updates and authenticated, scoped images |
| Partners | GET `/users/search` | Literal name-prefix search among shared-group peers; no email discovery |
| Terms | GET `/users/me/terms-status`; POST `/users/me/accept-terms` | Server-owned terms version; explicit retry-safe acceptance |
| Deletion | DELETE `/users/me` | Recent sign-in, queued Firebase/app cleanup; no synchronous completion claim |
| Preferences | GET/PUT `/users/me/notification-preferences` | Own ride/chat/group/promotion preferences |
| Blocking | GET `/users/me/blocks`; PUT/DELETE `/users/{user_id}/block` | Own block management; two-way enforcement |
| Ride request | POST/GET `/rides`; PUT/DELETE `/rides/{request_id}` | Modes, UTC windows, one active assignment; safe edit/cancel |
| Matching | GET `/rides/status`, `/rides/{request_id}/suggestions`; POST `/rides/{request_id}/group` | Private status, bounded suggestions, atomic reservation |
| Groups | GET `/groups`, `/groups/{group_id}` | Membership-scoped list/detail |
| Group lifecycle | POST `/groups/{group_id}/confirm`, `/mark-here`, `/start`, `/finish`, `/leave` | Explicit readiness/arrival/start/completion; slot release |
| Chat | GET/POST `/groups/{group_id}/chat` | Membership, length bounds, keyset paging, retry idempotency |
| Inbox | GET/DELETE `/notifications`; PUT `/notifications/read-all`, `/notifications/{notification_id}/read`; DELETE `/notifications/{notification_id}` | Own inbox, unread count, read/delete |
| Reports | POST/GET `/reports` | Submit scoped evidence and view own report history |
| Support | POST/GET `/support/tickets`; GET `/support/tickets/{ticket_id}`; POST its `/message`, `/close`, `/reopen` | Own quota-bounded threads; restricted users may appeal |
| Ratings | POST `/ratings`; GET `/ratings/pending`, `/ratings/pending/{group_id}`, `/ratings/average/{user_id}` | Trip eligibility, one rating per pair/trip, private review details |
| Admin users | GET `/admin/users/lookup`, `/admin/users/{user_id}/history`; PUT `/admin/users/{user_id}/role`; POST `/admin/users/{user_id}/restore` | Exact lookup, audit summary, protected role management, audited restoration |
| Admin groups | GET `/admin/groups/{group_id}`; POST `/admin/groups/{group_id}/cancel` | Projected summary; atomic safety cancellation with note |
| Admin reports | GET `/admin/reports`; POST `/admin/reports/{report_id}/resolve` | Review/dismiss/warn/suspend/ban with audit and conflict rules |
| Admin support | GET `/admin/support/tickets`, `/admin/support/tickets/{ticket_id}`; POST its `/message`, `/close`, `/reopen`, `/resolved`, `/in_progress` | Authenticated support workflow |
| Admin controls | GET/POST `/admin/maintenance`; PUT `/admin/app/version`; GET `/admin/stats`, `/admin/logs`, `/admin/jobs` | Service controls, counts, bounded audit/worker status |
| Admin delivery | GET `/admin/notifications/failed`; POST `/admin/notifications/{event_id}/retry` | Failed-event inspection and audited retry retaining receipts |

`GET /health` is liveness; `/readyz` probes configured auth and writable transactional
Mongo topology. `/account-deletion` serves the public Google-authenticated deletion
page/support contact, requiring configured public web identifiers and real policy
/contact settings. See deployment.md; it has not been browser-tested or deployed. `/ws` authenticates Firebase
and rechecks membership/account/block rules. Native token transport is the Authorization header. Browsers offer orix.v1
plus firebase.<ID token>; only orix.v1 is negotiated. Query token authentication
is removed. Credentials must never be logged from headers/subprotocols. Client protocol/reconnect/REST resynchronization integration
still needs completion; sockets do not guarantee missed-event replay.

## Changed contracts that the client must adopt

Ride mode is `auto`, `partner` or `lift`. Lift drivers additionally supply `driver`
and passenger `seats`. Latitude/longitude, local date/time window and explicit
`timezone_offset_minutes` are required; persisted bounds are UTC. Lift itinerary
verification needs a configured routing provider. Suggestions disclose labels and
scores rather than other members' exact coordinates or contact data.

The matcher creates waiting singleton groups, then merges compatible requests.
Arrival does not finish a ride. Use explicit start/finish; do not simulate
completion from timers. Only editable pending requests or unconfirmed singleton
groups may change route/time/mode. Shared groups require leave/new request.

Main histories/queues accept `limit` (1–100) and scoped `before` cursors; read
`X-Next-Cursor` for subsequent pages. Ticket pages can move during replies: dedupe
by ticket ID. Admin audit cursors use timestamp/ObjectID; migrate old timestamps
before cutover. Public suggestions are capped at 20 from a bounded candidate batch;
background workers rotate through batches. No global-best ranking is promised.

Chat uses a client message ID for retry deduplication and server acknowledgement.
Notifications are persisted via outbox with leases/receipts/retries. Fifteen-minute
reminders check the current active group schedule; Android uses high-priority
push/local notification without full-screen or exact-alarm permissions. Delivery
and permission behavior must still be verified on device/staging.

## Legacy paths intentionally replaced or removed

| Legacy capability | Replacement / disposition |
| --- | --- |
| Separate `/users/me/rider` offer/profile architecture | Normal `lift` ride requests with driver/seats; remove old client callers during migration |
| Broad `/groups/discover/available` | Own-request route-compatible suggestions; no general enumeration of private groups |
| Direct admin ban/suspend/unban | Report-resolution restriction workflow and audited `/restore`; client/admin tooling must use these |
| Raw admin Mongo group responses | Explicit group summary and scoped report evidence |
| Redis scheduler metric reset/manual reconciliation endpoints | Durable Mongo matching/lifecycle workers plus `/admin/jobs`; no destructive metric reset port |
| Public test-push/test-broadcast routes | No public replacement; real local/staging delivery scenarios after environment setup |
| Client log/activity/notification telemetry forwarded to Telegram | Remove obsolete client callers and Telegram implementation after verified cutover; no generic free-text log collector |
| College requests/domain rejection and Telegram webhook/bot | Public registration; final Telegram source removal remains gated on completed transition testing |

## Commands and outstanding boundaries

`seed-test-accounts` / `remove-test-accounts` manage eight local personas;
`bootstrap-admin` selects the first real verified administrator;
`support-account-deletion` plans/applies ownership-reviewed deletion for users
who cannot sign in (CLI only, no public bypass);
`migrate-data` plans legacy backfills by default and requires explicit stopped-writer
assertions to apply. See configuration.md and testing.md. None ran during this
implementation phase.

Remaining backend/release work includes complete old data/storage/retention
inventory, operational deployment/provider configuration, scalable geographic
candidate pruning, deployed web deletion/manual recovery handling, and final error/race/
contract validation. Frontend integration, current-stack verification, policy and
release artifacts remain. Remove Python, Telegram and duplicate implementations
only after the completed replacement passes the final testing phase.

Candidate evaluation now uses private versioned coarse geographic bounds where
available, with legacy fallback and full final checks. Member/profile loads use
restricted projections and bounded routes. This changes implementation cost, not
the public suggestion JSON. Geographic index use, match completeness, migration
version 3 and worker failure/health behavior still require final QA.

Group list/detail now include mode plus own my_pickup_time/my_pickup_summary for
confirmed/current or completed coordination where available. Lift start is initiated
by the driver (the existing stored member role is rider); passengers need readiness
but not mark-here at a later pickup before driver departure. At-least-one-minute
late starts shift personal pickup plans. Road-plan stops include optional elapsed
offset_seconds. Whole reminder_schedule arrays remain private. The frontend must
adopt these fields/lifecycle rules; final contracts and behavior are unverified.

Admin mutations distinguish loss of administrator authorization from ordinary
review/domain conflicts: the former now maps to HTTP 403 even when a concurrent
role/deletion change occurs after initial middleware authentication. Existing 409
business conflicts remain separate. This error mapping and transaction races are
not tested yet. Operator retire-legacy-data is now written for post-QA fixed-scope
record/profile cleanup; it is not a public API or an executed retirement.

The Flutter client now uses chat_sent-acknowledged WebSocket sends as its primary
path, with same-client-ID REST fallback after uncertain delivery. Account-scoped
state, nested events, history cursor loading and reconnect REST resync are written.
Backend active-group lists omit finished members' trips; explicit/history detail
access is retained. Full client/transport/multi-account validation remains deferred.

Matching status additionally returns the authenticated user's full `request`,
`group_id`, and (when assigned) `group_status`/`group_size`. Route suggestions are
available for pending/matching requests and matched, unconfirmed single-member
pending groups; membership/capacity/route constraints are rechecked on join.
WebSocket `chat_message` requires a nonempty `client_message_id`; replies correlate
validation failures and persisted `chat_sent` responses with that ID. Native
Bearer/browser auth-subprotocol transport and reconnect REST catch-up remain
unverified until the final QA phase.

Place lookup: authenticated/onboarded `GET /places/search?query=...` accepts
3–200 characters and returns `{places:[{label,latitude,longitude}],attribution,
attribution_url}`. Results contain at most five valid points. Unconfigured or
unavailable upstream returns 503 with manual-selection guidance; provider pacing
returns 429/Retry-After. No autocomplete/reverse route is exposed. See configuration.md.
