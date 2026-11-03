# Data lifecycle and deletion implementation

Implementation inventory, not a published privacy promise. Final production data,
retention settings, provider configuration and deletion behavior still require the
post-code environment/testing phase. No database or provider operations ran while
writing this inventory. No personal values belong in these documents.

## Active Go data

| Store | Content and scope | Account-deletion behavior |
| --- | --- | --- |
| Firebase Authentication | Google identity / local emulator identity | Retryable Admin SDK identity deletion; user-not-found is idempotent |
| users | Profile, contact, preferences, onboarding/terms consent, role/restriction, push token | Immediately marked deleting and push token cleared; document removed after dependent cleanup |
| user_avatars | Sanitized uploaded avatar bytes | Own image record deleted; no Go upload to Firebase Storage |
| ride_requests | Own route/time/mode and assignment | Active slots cancelled, then all own requests deleted |
| ride_groups | Shared lifecycle, members, labels, optional private road itinerary | Active groups cancelled; deleted member removed; route labels/road plan redacted; origin identity removed after dependent redaction |
| chat_messages | Member-authored content and retry key | Own messages anonymized and content removed; client/deduplication IDs cleared |
| notifications | Own inbox and structured event data | Own inbox deleted; related shared-trip notification labels/location payloads redacted |
| notification_outbox | Durable event, actor/direct target, group, per-recipient hashed delivery receipts | Direct-target events removed; actor cleared; own hashed receipts removed; shared events remain for other members |
| reports | Reporter/accused, scoped chat evidence, moderation outcome | Identity fields anonymized; linked text/evidence removed, including matching evidence-sender references |
| support_tickets | Own ticket/contact context, messages, reported account, resolution actor | Own tickets deleted; own authored messages in other tickets redacted; targeted-account text and resolution references cleared |
| ratings / user_blocks | Pair-scoped feedback and blocking edges | Both directions involving this account deleted |
| admin_logs | Structured action/actor/target and private action context | Actor/target-linked records deleted; full legacy indirect free-text inventory still required |
| user_logs | Legacy app-linked logs | Own user_id-linked records deleted |
| account_deletions | Retry job with Firebase UID while pending | Completed job clears Firebase UID; seven-day expires_at TTL removes its temporary job identifier |
| app_config / worker_health | Service configuration, bootstrap marker, generic worker state | Configuration remains; an own bootstrap Firebase UID marker is removed during deletion; explicit bootstrap can recover when no admin remains |

The queue continues retrying failed deletion and exposes attention counts after
repeated attempts. A queued response means accepted for processing, not finished.
`/admin/jobs` reports failures/attention. Local test cleanup is a separate command;
it is not a replacement for testing the real account deletion flow.

Shared-record redaction is conservative: it can remove conversation/evidence text
for a group or ticket to avoid retaining the deleted person's details in embedded
copies. Other users' unrelated profile/ride records are retained. Arbitrary text
that mentions a person without a stored relationship cannot be reliably located by
ID alone; the final privacy review must account for that limitation.

## Legacy and external stores requiring cutover work

The Python implementation stores avatars in Mongo user_avatars. Source review did
not find an application Firebase Storage upload or GridFS/file-upload path. The
Firebase Admin dependency on Cloud Storage is not proof that the app stores files
there. Confirm the actual deployed project's buckets, exports and integrations
after environment setup; inspect source/configuration without disclosing secrets.

Legacy Redis holds match requests/queues, user active-request keys, sessions,
tentative groups and maintenance/scheduler state. Go does not connect to Redis.
Stop Python writers at cutover, inventory the deployed key namespace and retire
only ORIX-owned keys through an explicit reviewed cleanup. Do not FLUSHDB or erase
unrelated applications' keys. TTLs are not a substitute for confirming retirement.

Telegram AI collections include ai_chat_context and ai_pending_confirmations,
keyed by Telegram numeric user/chat IDs rather than the app's Firebase user ID.
They cannot safely be joined to an app account from the current User schema.
Retire these obsolete collections and related bot configuration at the final
Telegram cutover after the user's transition/testing gate; do not pretend ordinary
app-account deletion already reaches them. Review bot/cloud logs and previously
sent Telegram copies separately. Deleting source code does not erase those copies.

Externally configured provider logs, reverse-proxy access logs, crash tooling,
Mongo backups/exports and Firebase/Google operational records require owner-set
retention and deletion procedures. The user already maintains backups; no backups
or parallel trees were created by this migration. Define actual production
retention before publishing policy text or completing Play Data safety.

## Still required before release

- Deploy and verify the written /account-deletion web flow and monitored
  recovery/contact path, including identities that cannot currently sign in. The
  page uses the same recent-authenticated self-service deletion endpoint; contact
  handling uses the written support-account-deletion CLI after an owner-operated
  ownership-verification procedure. Mailbox operation and this command remain
  unverified; no automatic request ownership proof is implied.
- Complete legacy/external inventory with the supplied environment, establish
  concrete retention periods and publish truthful privacy/deletion disclosures.
- Test restricted-user deletion, shared data redaction, queued retries, interrupted
  cleanup, already-missing identities and concurrent group/chat/moderation writes.
- Remove Python/Telegram and obsolete data only after completed transition testing;
  then inspect and test the final deployed architecture again.

## Support-origin queue records

The ownership-reviewed CLI shares account_deletions and the same transactionally
cancelled trip state with app/web deletion. It can queue an identity-only job for
an orphan Firebase UID without adding a users record. Support case/actor metadata
is attached once, audited and cleared at completion; ordinary seven-day completed
job expiry applies. Disabled/missing Firebase identity handling does not bypass
ownership review. Read-only planning has no database writes. The command and
shared-helper refactor remain unexecuted under the user's testing gate.

Private group route_search metadata now includes an anchor user ID and conservative
coordinate bounds for candidate filtering. It is derived location data, not public
analytics. API projections omit it; member leave clears it and account deletion
redacts the whole region for affected groups. Waiting-group backfill is part of
the explicit migration planner. Final data deletion tests must inspect this field
too. No live bounds/backfill/query or deletion verification has run.

Private group reminder_schedule now stores current member IDs, planned pickup
timestamps and own pickup labels. It is derived personal location/time data.
Responses project only the requesting member's pickup fields; member-only road
plans also include elapsed stop offsets. Join/edit/leave invalidate schedules,
expiry clears them, and account deletion redacts the complete related schedule.
Notification events/inboxes store only that recipient's pickup data. Completed
trip history may retain own pickup metadata until deletion/retention cleanup.
Include these fields in the final database/export/retention and deletion review.

Post-QA retire-legacy-data tooling is now written to inventory/remove records in
explicit bot/domain collections and obsolete rider-profile fields with bounded,
role-authorized/audited transactions. No retirement ran. This supersedes the lack
of a Mongo retirement implementation, not the need to inspect actual deployments,
Redis/provider/Telegram copies, backups or retention. Full scope/conditions are in
deployment.md; source deletion still waits for completed transition testing.

Place-search results: backend retains up to 256 normalized queries/results in
process memory for at most 24 hours; no user IDs are attached and no Mongo
location-history collection is added. Cache is shared by the process, so searched
private addresses may remain until eviction/expiry/restart. Include the selected
provider's processing/retention in the published privacy inventory. Flutter picker
state is transient and no longer reads/writes the old device-wide location_history
preference. Removal/migration of that legacy preference on existing installs
remains part of final data-retention cleanup. No device data was changed now.

2026-11-03: Go profile updates now serialize ownership of a nonempty FCM token
through a SHA-256 keyed push_token_claims document and transaction, removing that
token from other profiles before assigning it. Claims retain owner ID and timestamp,
not the token itself. Account deletion removes claims owned by the deleted user. Migration/
retention review remains; sign-out unlink and multi-device semantics remain outstanding.
Source formatted only; no database, compiler, test, or emulator run.
