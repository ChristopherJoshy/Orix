# Orix integration — active

## Implemented source

- Android/web onboarding asks Student, Teacher or Other. Go persists occupation,
  and group member views expose it separately from authorization/driver roles.
- Go group rejection atomically cancels an unstarted group and requeues existing
  requests, retaining per-journey exclusions of the group and its members. Ranking
  and reservation check exclusions symmetrically. Repeated rejection is idempotent;
  editing keeps exclusions. Android and web controls call the endpoint.
- Website runtime mock records, demo store, simulated matching, fake activity chart
  and local-only moderation/support adapters removed. Store now holds motion only.
- Website dashboard, create/edit/detail/history/matching, groups/chat, messages,
  notifications, profile/photo, account preferences, blocks/reports, support and
  administrative controls call the actual API. One-time lift registration uses a
  real driver ride request. Recurring availability is staged in Go and both
  clients; concurrency, scheduling and account-switch behavior still need tests.
- Admin BFF allowlists stats, account/group lookup, role/restoration/cancellation,
  maintenance/jobs/delivery retry/audit, support, reports and release controls.
  Shared admin secret stays server-side. Every privileged API rechecks authority.
- Avatar uploads now work before onboarding completion, while preserving verified
  authentication, restricted-account rejection, own-account ownership and image
  size/normalization controls. Previously this blocked the onboarding photo step.
- Database change streams cover outbox inserts, ride requests, notifications,
  support tickets and profiles. Owner-only invalidations cause REST resync; profile
  changes invalidate current group member views. Admin support invalidations carry
  no private content and recheck role at delivery. Notification deletion writes a
  completed, non-push outbox invalidation atomically with deletion.
- Web resource reads serialize in-flight fetches and retain a trailing invalidation.
  Sockets guard duplicate CONNECTING instances and recover on visibility/network
  changes. Android refreshes ride/inbox/profile state and support threads/lists.
- Bounded read retries added for transient failures. No automatic non-idempotent
  write replay. Server error bodies/raw exceptions are not interface copy. Curated
  errors use light Kerala humour for transient/network failures; safety/auth remain
  straightforward. Dedicated monochrome connection-loss/chaya scene on app/web.
- Shared modal/toast/dialog/sheet/menu/date/time controls use neutral pixel themes.
  Native Google/OS permission and notification chrome stays platform-owned.
- About-builder copy on landing, website About and Flutter About, based on public
  GitHub profile; LinkedIn is linked as provided and did not permit research fetch.
  Quote edited to: “Life is like music. Its highs and lows are what make it beautiful.”
- Website deletion now reauthenticates Google and submits the real deletion request;
  a queued request is explicitly not described as completed cleanup.

## Remaining required work

1. Verify recurring lift registration and always-active scheduling, including
   concurrent edits, retries, worker restarts and account changes in staging.
2. Verify inbox/support/group-chat pagination, support retry deduplication and
   reconnect/concurrent-state edge cases. Verify web push background delivery;
   permission alone does not register a functioning web push service.
3. Complete legacy Python rejection/matching/realtime parity or verify the whole Go
   migration before retiring Python. Python runtime remains unverified.
4. Verify actual multi-account Mongo transaction races, replayed rejection,
   confirmation/start races, blocks/deletions, moderator BFF and reconnect recovery
   using isolated Firebase emulator/staging auth and persistence. No production
   fixture writes. Do not infer integration success from pure tests.
5. Verify deployment Google auth, API hosting/provider configuration, APK replacement,
   signed release publication and full responsive app/web flows. Missing provider
   config is an explicit error, never demo data or a silently changed provider.

## Isolated account and transaction check — 2026-11-03

An ephemeral localhost MongoDB 8.0.34 replica set and Firebase Auth Emulator
were started for `demo-orix` and `orix_internal_test`. Eight personas were
created using the guarded seed command and real emulator authentication. Against
the Go API, concurrent identical recurring-offer saves, pauses and resumes
returned the same revision and resulted in one departure; a second account saw
no schedule. Concurrent support-ticket retries returned one ticket, while a
replay with changed content returned a conflict. These are focused scenarios,
not proof of all transaction races, Google sign-in, FCM or production behavior.
The temporary API, Auth Emulator and MongoDB processes were stopped afterward.

## Executed verification — latest increment

- Website TypeScript/Vite production build passed after latest connected controls.
- Full Go `GOCACHE=/tmp/orix-go-cache go test -race ./...` passed, including new
  owned historical-ride reads, onboarding-avatar authentication and resource event
  classification. First sandbox run could not open httptest sockets; an approval
  review later timed out; the permitted retry ran and passed. No production DB used.
- Three direct web error-copy assertions passed via `npm run test:errors`.
  Node here lacks TS stripping support, so the command compiles the module first.
- Nine Flutter pixel/reduced-motion/narrow-layout/tour/notification tests passed.
- Focused Dart analysis has no errors/warnings; style lint infos remain. Duplicate
  picker theme arguments and an incorrect connectivity method were caught and fixed.
- Updated Android debug APK built successfully, installed without clearing data,
  and opened in existing visible emulator `emulator-5554`. No personal sign-in was
  automated. API binary rebuilt and started; public /health returned healthy2.1.0.
- Agent-browser public landing: desktop1440 screenshot, phone390 screenshot,
  width320 and tablet768 overflow checks; no horizontal overflow. Enlarged root text
  at200%/1440 had no horizontal overflow. Login showed Google only. Real browser
  offline mode displayed the chaya scene/Retry, and restoring network cleared it.
  Authenticated multi-account flows were not exercised by this public-page review.

Current runtime: API exec session63322 (`/tmp/orix-current-api`), log
`/tmp/orix-current-api.log`; current debug build log `/tmp/orix-current-android-build.log`.
Revalidate process/session liveness before relying on these handles.
