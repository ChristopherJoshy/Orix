# ORIX contributor guide

## Product direction

ORIX is becoming a public travel coordination app for shared autos, travel
partners and lift offers. College email restrictions and Telegram integration
must be removed. Keep verified authentication, safety preferences, blocking,
reporting and moderation intact throughout the migration.

## Current architecture

- `frontend/`: Flutter client with Firebase authentication and messaging.
- `backend/app/`: existing Python API, still the runtime implementation.
- `backend/internal/`: Go replacement modules under active development.
- `backend/main.go`: Go HTTP/CLI entry point. `backend/app/*.go` now covers auth,
  profiles, rides/matching/routing, groups/chat, notifications, safety/moderation,
  support, ratings, deletion and operational controls. Server registration is in
  server.go; domain matching stays in backend/internal/matching. See the current
  route/replacement inventory in Docs/backend-api.md.
- Go serves the staged public /account-deletion page using embedded assets and
  explicit Firebase WEB/support/privacy configuration. backend/Dockerfile is a
  staged Go-only runtime image. Neither page nor image has been runtime-verified.
- backend/app/places.go is the staged authenticated place-search adapter. Use
  an explicitly configured private/contracted provider; public OSMF Nominatim
  is excluded. Flutter picker state and network/GPS adapters live in existing
  core/providers and core/services directories; tile configuration is shared.
- WebSocket delivery and client protocol migration remain unverified. Python
  stays until the completed transition passes the final test phase.

Do not describe the migration as complete until the full API and WebSocket
contracts, persistence, background jobs and client integration are verified.

## Implementation conventions

Use small domain-focused modules and explicit types. Keep transport, domain
logic and persistence separate. Prefer readable names and straightforward control
flow. Comments should explain invariants, intent, units or non-obvious tradeoffs;
remove stale, misleading, abusive and redundant comments when touching a module.
Never include credentials, personal data or authentication tokens in logs.

Verify the latest stable releases using official sources before upgrading the
stack. Update lockfiles alongside manifests, assess breaking changes and platform
requirements, and test the affected flows. Do not blindly replace compatible
constraints with unverified version numbers or use deprecated APIs.

## Matching and safety

Normalize times to UTC and validate coordinates. Match all members' time windows
and safety preferences, enforce capacity atomically, and prevent duplicate active
assignments. Geometric overlap is a suggestion; actual lift itineraries need road
routing. Authenticate and authorize membership, reports, chat and admin actions
on the server. Never trust client-supplied identity or safety attributes.

Notifications need durable scheduling, bounded retries and deduplication. Respect
user preferences and platform permission decisions. Verify alarms on a device,
including background and reboot behavior.

## Verification and releases

Run `go test -race ./...` from `backend/` for Go changes. Where the default Go
cache is not writable, use `GOCACHE=/tmp/orix-go-cache`. Run applicable Python
checks while the legacy runtime remains. Run Flutter analysis and relevant tests
from `frontend/`; use device tests for maps, permissions and notifications.

Bump the app version for release work. Record checks actually executed and any
failures or unavailable environments. Do not claim production readiness or Google
Play compliance from unit tests alone. Verify current official Play policies,
privacy disclosures, deletion, UGC moderation, permissions and release artifacts.

## Documentation and continuity

Keep the existing frontend and backend structure. Do not create backups or
parallel replacement trees; the user already maintains backups. Place project
documentation in `Docs/`. Maintain `Docs/memeory.md` (spelling requested by the
user) as the current state, constraints and next actions, and `Docs/history.md`
as a dated record of changes and actual verification results. Keep this root
AGENTS.md as the contributor entry point. Do not record secrets in documentation.

## Current task and testing scope

The user now authorizes small focused tests while completing the redesign and
full app/API/website integration; they will do broader manual testing. Keep the
visible emulator available. Do not test fixtures against Orix-Prod or claim full
migration, realtime or release readiness from these small checks. Use the existing
Firebase emulator/staging personas for automated authenticated scenarios.

All app/website owned surfaces, artwork and notification branding must be pixel
inspired and strictly black/white/grayscale. Native permission/system notification
chrome remains controlled by the OS; do not claim it can be fully restyled.

Onboarding asks Student, Teacher or Other, stored as `occupation` and shown in
group member profiles. This self-description is separate from admin/driver roles
and grants no permissions or women-only eligibility.

Rejecting a group before travel restarts the existing journey's matching, rather
than creating a duplicate request. Persist request-scoped rejected groups and
members and enforce them symmetrically at candidate ranking and transactional
reservation. Requeue all affected members, reset stale readiness/pickup plans and
notify them atomically. Repeated rejection requests must be idempotent. New
journeys may start with fresh exclusions; editing the same journey must retain them.

Keep Docs/memeory.md and Docs/history.md current with code changes, executed checks
and remaining integration limits. Implement and use real four-plus test accounts
(prefer six-plus), never an authentication bypass; see Docs/testing.md.

## No mocks or silent fallbacks in the product

The user requires real service-backed product flows: no seeded people, simulated
matches, illustrative analytics presented as data, local-only support/moderation,
or fabricated success. Remove these adapters as each remaining flow is connected;
do not describe unfinished routes as working. Preserve user input on failure,
show the actual unavailable/error state and provide Retry. Use bounded retries
with backoff for transient failures; honor cancellation and server retry guidance.
Never automatically replay a non-idempotent mutation unless the server deduplicates
it. Missing credentials, denied permissions and invalid input require an explicit
action, not fake data or another provider silently substituted. Account changes
must discard private snapshots. Verification must distinguish real integration
results from isolated unit checks; test personas require actual emulator/staging
authentication and persistence, never an authentication bypass.

## User-facing errors and overlays

Never render raw exceptions, server detail strings, stack traces, hostnames or
provider diagnostics. Map failures to authored, actionable copy. Light Kerala
humour (for example a chaya break) is welcome for connection/transient failures;
authentication, safety, moderation and privacy messages stay clear and respectful.
Connection loss needs the dedicated grayscale pixel scene and a real Retry action.
All owned dialogs, toasts, alerts, sheets, menus and date/time pickers use the
shared monochrome pixel theme, including borders, focus and typography. Native
Google/browser/Android prompts remain platform-owned. Do not claim to restyle them.

Website runtime state now comes from the authenticated API/BFF. `lib/store.ts`
contains UI motion preference only; do not reintroduce demo people, journeys,
analytics or local-only moderation/support. One-time lift offers create driver
requests; recurring availability is still incomplete and must not be advertised
as implemented. See Docs/integration-progress.md for the actual verification and
remaining contracts. Builder profile source notes live in Docs/builder-profile.md.

## Current verification hold / illustration direction

The owner paused broader verifications. The subsequent illustration design phase
explicitly authorized responsiveness checks for those changed layouts only; those
checks are recorded in Docs/illustration-design-phase.md. Do not resume broad
backend, integration, regression or account QA until the owner authorizes it.
Implementation work may continue while that verification hold is in effect.

Use distinct generated Kerala editorial illustrations for app and website, from
the approved grayscale street-scene family. Keep the original logo and good
animated/functional code-native pixel art. App art belongs under frontend assets;
web art under website/public/art. Do not copy the same scene to both platforms.
Keep real routes, identities, errors and controls in semantic UI. Generated people
are decorative art, never mock travel contacts or fabricated service outcomes.

## Commit and push

After the appropriate tests have passed, commit and push each completed change,
large or small, with a clear, high-quality commit message. Do not leave completed
changes uncommitted. Before staging or pushing, inspect the changes and ignore
rules for credentials and private configuration; never commit or push secrets.
The owner's current verification hold still applies until they lift it.
