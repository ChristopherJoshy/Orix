# Ride notifications

The server emits `type=RIDE_ALARM`, `action=show_alarm`, `group_id`, `ride_time`
and `pickup_summary`. The client recognizes alarm types case-insensitively and
serializes local notification payloads as JSON. Legacy pipe payloads remain
readable. A nonempty group ID is required. Reminder and completion messages are
not departure alarms.

Background FCM delivery initializes Firebase and local notifications in its own
isolate. A high-priority reminder opens meeting details when tapped. Cold local
launches, initial FCM messages and background FCM taps feed pending state, consumed
when the user is authenticated and onboarded. Group actions remain subject to
server authorization. There is no OS lock-screen takeover or exact-alarm schedule.

Google Play restricts default full-screen intent access to core alarm/calling
functionality. ORIX uses an ordinary high-priority ride reminder instead:
https://support.google.com/googleplay/android-developer/answer/13392821
Firebase background handler requirements:
https://firebase.google.com/docs/cloud-messaging/flutter/receive-messages

Remaining verification: Android/iOS real-device foreground, background and cold
launches; notification permission denial/regrant; battery saver; delayed FCM;
reboot; signed-out taps; stale/cancelled groups; simultaneous reminders; server
outbox retry/deduplication; user preferences. FCM does not guarantee exact-time
arrival. The current repair does not establish end-to-end alarm reliability.

## Go scheduling and delivery follow-through (written, unverified)

Reminder scheduling now runs independently of delivery at a thirty-second cadence.
It projects only group/time/member fields and upserts stable per-schedule/member
outbox IDs in batches of at most 400 models. Four bounded delivery workers drain
claimed work immediately and back off on empty/error polls; one worker is reserved
for reminders and three general workers prioritize them ahead of ordinary backlog.
Durable leases, retry limits, inbox IDs and recipient receipts remain authoritative
across replicas. Idle/success heartbeats are throttled, and worker status describes
server processing rather than proof of device arrival.

Reminder delivery rechecks current active status, membership and the same departure
immediately before each recipient's inbox/push work. Near-departure reminders are
skipped. Push calls have a ten-second bound; reminder Android TTL is at most a
minute with departure/send-time allowance, and APNs has explicit expiration,
alert priority and a stable per-group/recipient collapse ID. Payloads now include
notification_id and the actual server title/body alongside ride_time. Already
accepted external pushes cannot be recalled after a later cancellation. Device
and client checks are still required, especially the existing client's hardcoded
fifteen-minute text and duplicate iOS presentation behavior.

Waiting singleton groups can no longer be confirmed, preventing readiness from
removing them from automatic matching. The migration planner resets legacy ready
flags on pending singleton groups; it has not run. Confirmation also rejects an
elapsed departure window. This does not change arrival/start/finish into automatic
completion.

An outstanding backend timing detail is lift passengers with pickup stops later
than the driver's departure. Current reminder schedules still use group departure;
per-member pickup offsets/planned times and their lifecycle need implementation
before claiming every mode's fifteen-minute reminder is correct. Later pickup
reminders may need to remain eligible after the driver starts. Road/traffic timing
and device execution remain estimates, not an exact alarm guarantee.

Official implementation references consulted on 2026-11-02:

- https://firebase.google.com/docs/cloud-messaging/customize-messages/setting-message-lifespan
- https://pkg.go.dev/firebase.google.com/go/v4/messaging

No scheduler, push, queue/database, compile, test, benchmark or device commands have
run for this slice under the user's code-first gate. Final QA must cover backlog,
reserved reminder capacity, scheduler concurrency/bulk retry, receipt/lease races,
stale schedules, expiry/collapse, opt-outs and late/failed provider acceptance.

## Per-member planned lift pickup reminders (written, not verified)

The earlier group-departure-only limitation now has implementation code. Verified
road routes record elapsed seconds at every ordered stop. At final confirmation,
the backend stores a private reminder_schedule with each member's planned pickup
time and own pickup label. Shared-auto/partner members retain the common meeting
time. Lift pickups use driver departure plus the actual selected route's leg
offsets, not direct-line distance or a guessed common time.

Group responses include mode and only the caller's my_pickup_time/
my_pickup_summary alongside the existing member-authorized road plan. The entire
private reminder array is not returned. New joins, singleton edits and departures
clear it; confirmation rebuilds it. Expiry/deletion clear it too. A legacy lift
without verified offsets is not assigned a guessed passenger reminder; migration
must flag it for route review before cutover. Existing shared-auto/partner groups
may fall back to their common departure.

The scheduler queries indexed personal pickup times for active/in-progress groups.
Lift passengers remain eligible after the driver starts; finished/removed members
and closed groups are ineligible. A reminder checks both the driver's current
departure and the recipient's pickup timestamp. Its stable v2 ID includes both
millisecond timestamps and the recipient, so changing either invalidates old work.
The payload title, ride_time, own pickup label and TTL/APNs expiry use that personal
time. Older Go lift reminder events without the personal timestamp are discarded;
any preexisting Go queue needs staged cutover/duplicate-delivery review.

Lift start is driver-controlled and requires all members ready plus the driver's
arrival at their origin. It does not require passengers to arrive at later stops
before the driver can start. A start at least one minute late shifts the planned
pickup schedule transactionally; sub-minute start jitter keeps it stable. Passenger
reminders may be reissued for a changed schedule, with the actual remaining minutes.
These are planned road-duration estimates, not live traffic or vehicle tracking.

Migration planner version 4 backfills derivable live-group schedules and reports
missing/unverified lift timing for explicit review. Nothing was migrated. Test
source covers later pickup after start, changed schedules, closed/finished/removed
members, absent offsets/labels, common departures and stable identity boundaries.
No tests or runtime scenarios have run; client pickup-time display, stale-group
checks and hardcoded alarm text still need the frontend pass and final device QA.

## Client reminder/lifecycle integration follow-through

Written client changes now derive remaining minutes from personal ride_time and
use a stable per-group local notification ID. Foreground duplicate checks prefer
the server notification_id; iOS APNs alerts are not duplicated as local alerts.
The reminder route refreshes current membership/pickup before showing, suppresses
stale/changed/finished/closed or inaccessible trips, shows the refreshed own label,
and offers dismissal, details and arrival. Arrival invokes mark-here only; it
never declares completion. OS sound/high-priority presentation remains in the
notification path; the app route no longer forces immersive UI or repeating
sounds/haptics. Widget/lifecycle/platform behavior is unverified.

WebSocket group updates and resume/reconnect trigger authoritative trip/chat refresh.
Chat now uses acknowledged WebSocket writes as requested, with same-ID REST fallback
for uncertain delivery. Background persistent deduplication, push-token ownership,
permissions/preferences and all end-to-end notification QA still need completion.

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


## Optional Manglish re-engagement (2026-11-03)

The Go notification worker schedules the 20-message rotation at 08:00, 12:00,
16:00 and 20:00 IST. Preferences > A little nudge is explicit opt-in. The
existing promotions field is reused and remains false by default. Scheduling
and the four-hour cooldown are transactional; stable slot/user outbox IDs
deduplicate replicas. Late delivery work expires after thirty minutes. Delivery
timing is still subject to FCM and platform restrictions. This is not an alarm.

Admin Operations > A little nudge reads/writes GET/PUT /api/v1/admin/engagement
through the authenticated BFF. Version conflicts preserve drafts and require an
explicit reload; updates are audited. Title limit: 80 Unicode characters; body:
240; maximum 200 messages. Individual messages and the entire schedule can pause.
Only public reminder copy enters web push payloads. Other private updates retain
generic push text. In-app copies use the existing monochrome pixel components;
Android/browser notification chrome remains native. The legacy Python hourly
promotion scheduler is disabled. Focused tests passed; authenticated multi-user
queue/push/device verification remains outstanding.
