# Go deployment and public account deletion

Staged implementation only. The user requested completing code changes before
checks/tests, then supplying environment values and using the installed emulator.
No container builds, server starts, provider setup, browser tests or deployments
were performed while writing this material. Python remains until verified cutover.

## Backend service

Build the Go service from `backend/` with `go build -o orix .` using the Go
version specified in `go.mod`. Run the binary as a dedicated unprivileged service
account. Supply private values through the host's secret mechanism; keep credential
files outside the repository and restrict their permissions. Application default
credentials are also supported. Use a Mongo replica set or sharded cluster and
configure the correct network addresses for MongoDB and routing services.

Put HTTPS ingress in front of 8080. Forward WebSocket upgrades for /ws and allow
its connection lifetime; avoid logging authorization headers or query tokens.
Configure exact CORS_ORIGINS and only controlled TRUSTED_PROXY_CIDRS. Route probes
using /health for liveness and /readyz for transactional Mongo readiness; inspect
/admin/jobs for worker state. Readiness does not validate Google OAuth/FCM/routing.
Forward SIGTERM and allow at least the server's ten-second shutdown period.

The binary exposes the existing CLI subcommands too; they are not run automatically.
Local emulator seed/removal commands intentionally reject nonloopback databases and
cloud projects. Do not weaken those guards to make a remote deployment seed work.

## Public deletion page

Go serves `/account-deletion`, its scoped script/styles and public config endpoint.
Users sign in with Google, inspect their account, explicitly confirm permanent
ORIX deletion and call the existing recent-authenticated DELETE /api/v1/users/me.
No app installation, ordinary profile verification endpoint or onboarding is
required by this page. App-banned/suspended identities can still request deletion
with valid recent Firebase authentication. Disabling a Firebase identity itself
requires the separate monitored support path for ownership verification.

The page uses Firebase modular web SDK 13.1.0 from Firebase's official CDN, pinned
from current official release/setup documentation. Only authentication modules are
loaded. It selects memory-only auth persistence and keeps ID tokens out of URLs,
local/session storage and logs. There is no background analytics or service worker.
Deletion requests use bearer headers, omit cookies, reject redirects and have a
bounded wait. Accepted requests stay accepted even if subsequent browser sign-out
fails; transport failures explicitly leave acceptance uncertain for safe retry.

The page has an external script/style, restrictive CSP, no framing, no referrer
and popup-compatible opener policy. These controls and popup behavior are written,
not browser-verified. Google popup scripts/authorized Firebase auth iframe must be
allowed by the deployed policy, including any stricter ingress headers.

Required environment for Google web sign-in:

| Setting | Purpose |
| --- | --- |
| FIREBASE_PROJECT_ID | Same Firebase project as the backend/mobile app |
| FIREBASE_WEB_API_KEY | Public web app API key |
| FIREBASE_WEB_AUTH_DOMAIN | Firebase/custom Auth host, without scheme/path/port |
| FIREBASE_WEB_APP_ID | Public registered web app identifier |
| ACCOUNT_SUPPORT_EMAIL | Real monitored deletion/recovery contact |
| PRIVACY_POLICY_URL | Published HTTPS policy with truthful retention/provider details |
| CORS_ORIGINS | Must include the exact public backend/page origin for DELETE requests |

Partial web auth configuration fails startup. Google web sign-in also requires
the support email and privacy URL; do not publish placeholders. Without web config,
the page offers a configured support email. With neither, it returns 503 and says
the service is unavailable rather than showing a working-looking request button.
Only the explicit public Firebase fields reach the config response; server/admin
credentials never do. Configured contact/policy links are validated and rendered
through html/template escaping.

Register a Firebase WEB app in the same project. Enable Google as an Auth provider,
authorize the public page hostname in Firebase Authentication, and configure API-key
restrictions for the intended web Auth use. Publish a valid certificate/HTTPS page
and the policy/contact service. Enter the final public account-deletion page URL
in Play Console only after deployment and verification. Nothing was published here.

Support email is a user-initiated mailto link, not an automatic message or a public
unauthenticated destructive API. The owner must monitor it, verify requestor
ownership, track requests, and use an established privileged deletion/recovery
procedure. A disabled Google/Firebase account is not bypassed in the web page.
No email was sent or cloud identity/configuration created by this implementation.

## Final verification after code and environment setup

Test successful Google sign-in/deletion, the wrong-account switch, cancelled/blocked
popups, blocked third-party storage, denied network/CDN, expired recent sign-in,
429/503/timeouts, response loss and retry, account restrictions, provider-disabled
identities and manual support handling. Inspect queue completion and stored-data
redaction rather than treating the browser's accepted message as completion.

Verify mobile in-app deletion separately. Confirm public page availability without
app installation, support delivery/ownership review, final privacy retention text,
all external-store cleanup and the Play Console URL. Review CSP on deployed HTTPS
and Google sign-in on supported browsers. The deployed service needs ordinary Go race/
contract checks, multi-user database integration, graceful shutdown and deployment
probe tests. All remain deferred; neither code presence nor policy references
prove Google Play compliance.

Official references used for implementation:

- Google Play account-deletion requirements: https://support.google.com/googleplay/android-developer/answer/13327111?hl=en
- Google web sign-in: https://firebase.google.com/docs/auth/web/google-signin
- Memory-only auth persistence: https://firebase.google.com/docs/auth/web/auth-state-persistence
- Firebase CDN setup: https://firebase.google.com/docs/web/alt-setup
- Firebase SDK releases: https://firebase.google.com/support/release-notes/js
- Go releases: https://go.dev/dl/

## Ownership-reviewed support deletion command

`go run . support-account-deletion` (or the equivalent binary subcommand) plans a
support request by default. This command is written but has not been run. Use it
after the code/environment/testing gate, with server credentials held by an
operator. It is not an HTTP endpoint and does not weaken Firebase token checks
on the app/web self-service routes.

The operator must receive an actual deletion request, verify account ownership
through an established support procedure, and use the exact Firebase UID found
through trusted administration. An email address or someone knowing a UID alone
is not proof of ownership. Keep proof documents outside app logs/environment files;
use an opaque case reference without names/email/content in this command.

Required settings (in addition to ordinary Firebase/Mongo credentials):

| Setting | Requirement |
| --- | --- |
| APP_ENV | Explicit local, staging or production; local retains isolated emulator/database guards |
| DELETION_DATABASE | Exact match for explicit MONGODB_DATABASE |
| DELETION_ADMIN_FIREBASE_UID | Existing enabled, verified Firebase identity with an active app admin role |
| DELETION_TARGET_FIREBASE_UID | A distinct, ownership-reviewed target UID; disabled/missing provider identities are supported |
| DELETION_CASE_ID | 1–100 letters/digits/underscore/hyphen, starting with a letter/digit |
| DELETION_APPLY | Empty/false plans only; true requests application |
| DELETION_OWNERSHIP_VERIFIED | Must be true before application; an operator assertion, not automated proof |

Planning reads identity/database records without upserting profiles, creating
indexes, recording audits or queueing deletion. Its output contains internal IDs,
case reference, app-account presence and applied status; no emails or tokens.
Inspect the actual target against the support request before applying. A changed
role/target state is reloaded inside the apply transaction, not trusted from an
old plan. Possession of privileged server credentials is part of this operational
trust boundary; supplying an administrator UID is not a new remote authentication
mechanism.

Application touches/rechecks the actor against concurrent demotion/deletion,
queues the same account/trip cleanup as self-service, attaches the first support
case and records an audit in that transaction. A different case cannot silently
replace a queued case. Same-case retries do not append another review audit. An
identity without an app profile gets an identity-only deletion job rather than a
new app profile. Missing provider identities still allow app-data cleanup. No
provider deletion happens synchronously in the command: normal Go workers must
be running to process the job.

Pending jobs retain the case/actor only for processing. Completion clears those
fields and Firebase UID; the temporary job expires after seven days. Actor/target
linked audit removal follows the documented account-deletion policy. Do not treat
command success as final erasure; inspect worker state and actual provider/data
cleanup. Final QA must cover plan side effects, authorization, ownership-review
assertions, concurrent demotion, disabled/missing identities, orphan identities,
case conflicts, identical retries and interrupted worker cleanup. No such tests or
support requests have been executed yet.

## Post-test legacy data retirement

`go run . retire-legacy-data` is written, defaulting to a read-only inventory.
Do not execute even the plan during the current code-first gate. Apply belongs
AFTER the full backend/client transition has passed QA, alongside the owner's
requested final Python/Telegram/source cleanup. No retirement has been run.

The command has a fixed source-derived collection allowlist: ai_chat_context,
ai_pending_confirmations, ai_config, promo_config, telegram_admins,
telegram_head_admins, domain_names and domain_requests. It removes their records,
not active app data or whole databases. It also unsets only obsolete is_rider and
rider_info fields from users after the new lift-request client is ready. Ordinary
profile/auth/role/contact/consent/preferences remain. App configuration/version,
reports, shared trips, support and audit history are outside the deletion list.
Empty retired collection/index metadata remains; dropping namespaces is separate
from removing their records and is not needed for data erasure here.

Use explicit APP_ENV, Firebase/Mongo credentials, RETIREMENT_DATABASE matching
MONGODB_DATABASE, and RETIREMENT_ADMIN_FIREBASE_UID. The actor must have an enabled,
verified Firebase identity and a current active app administrator role. Plans only
read identity/database counts; they do not create indexes/profiles/jobs/audits.
Output contains collection names/counts, internal actor ID and incremental result
counts; retired document contents, API secrets and Telegram messages are not shown.

Apply additionally requires RETIREMENT_APPLY=true,
RETIREMENT_WRITERS_STOPPED=true, RETIREMENT_TRANSITION_TESTED=true and an opaque
RETIREMENT_TEST_REFERENCE (1–100 letters/digits/underscore/hyphen). These flags are
operator assertions: they do not stop processes or inspect evidence automatically.
Reference real recorded QA evidence, then stop old/new app and bot writers before
applying. Server credentials are an operational privilege, not an HTTP auth bypass.

Application uses transactions of at most 500 records. Each nonempty batch rechecks
and writes the actor's authorization state, mutates only its fixed scope, and
records count/reference audit atomically. A role change or concurrent mutation
causes conflict/refusal. Interrupted runs are incremental and retryable, not an
all-database atomic operation. Results report completed steps even if a later step
fails; do not equate applied=false after an error with no prior changes.

This command does not erase Redis, backups/exports, provider or Telegram copies,
or any source files. Retire only ORIX-owned Redis namespaces using the separately
reviewed environment inventory after writers stop; never FLUSHDB on a shared
service. Source deletion remains explicitly gated on finished transition testing.
No backup folders/files are created. Final QA needs plan side-effect checks,
allowlist/non-target preservation, role races, interrupted retry and actual retired
record removal before using this operator action on the selected environment.

The production HTTPS proxy must allow uploads up to 256 MB for the exact
`/api/v1/admin/android-apk` endpoint. Keep the smaller general request limit on
other routes and disable request buffering for this upload. Validate nginx's
configuration before reloading it.
