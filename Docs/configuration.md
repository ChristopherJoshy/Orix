# Configuration

Copy backend/example.env to backend/.env and replace placeholders. Run the Go
server or tool commands from backend/. The Go entry point loads .env before startup;
process environment values take precedence. ENV_FILE selects another file. A missing
default .env is allowed for deployments using injected environment; a missing
explicit file is an error. Env parsing accepts KEY=value, optional export prefix,
full-line comments and single/double quoted values. Shell expansion/commands and
inline comments are not interpreted; use one-line JSON/base64 for service accounts.
Duplicate or malformed assignments fail without printing their values.

Go consumes FIREBASE_PROJECT_ID, service-account path/JSON or application default
credentials, MONGODB_URI/MONGODB_DATABASE, LISTEN_ADDR, CORS_ORIGINS,
RATE_LIMIT_PER_MINUTE, ADMIN_API_SECRET and optional ROUTING_BASE_URL. Mongo must
support transactions and change streams. Startup creates required indexes; existing
conflicts require explicit migration. Legacy Python-only template settings remain
marked by their names while Python is retained and are not proof that Go consumes
them. backend/.env.example stays synchronized with example.env.

No college domain list or Telegram credentials are required by Go. Keep server
credentials and admin secrets out of Flutter, logs and version control. Use explicit
browser CORS origins and a deployed HTTPS origin for public clients. Default Go
listen port is 8080; the legacy Python backend used 8000.

Frontend values are public compile-time values. Copy frontend/example.env to a
private build env file and pass --dart-define-from-file=<path> from frontend/. The
API_BASE_URL is a backend origin without /api/v1. Android emulators reach the host
at 10.0.2.2; iOS simulator/desktop generally use localhost, and physical devices need
a reachable host. API avatar image requests will need authenticated headers. Native
Firebase/client project configuration remains separate. Release HTTPS and obsolete
client store/HTTP fallbacks still need the frontend cleanup pass.

Internal seed/removal requires local demo-orix Auth Emulator and loopback MongoDB
orix_internal_test. See testing.md. Do not run commands during the code-completion
phase; the user will supply environment values before final testing.

First staff setup uses the bootstrap-admin command described in migration.md.
Legacy data planning/application uses migrate-data; never apply with active old/new
writers. Environment assertions select intent and do not stop other processes.

## Reverse proxies and health endpoints

TRUSTED_PROXY_CIDRS is an optional comma-separated list of explicit CIDRs for
proxy socket peers under your control. Default/empty trusts no forwarded headers.
Do not include broad client networks. Default routes are rejected. Ingress must
replace X-Forwarded-For or append its actual connected peer; the Go server walks
right-to-left to the first untrusted hop. Invalid/oversized chains fall back to
the socket peer. IPv4-mapped addresses normalize to IPv4. Rate limits are local to
each server instance; shared NAT users still share an address quota, and a
multi-instance distributed/authenticated quota remains a scaling consideration.

GET /health is process liveness. GET /readyz returns 200 only with configured auth
and a reachable writable Mongo replica set/sharded cluster with session support;
otherwise 503 with a generic status. Both are exempt from request quotas. Readiness
is a short database topology probe, not proof of OAuth, routing, FCM, worker
freshness, collection permissions or complete application readiness. Restrict probe
access at ingress where appropriate. Startup performs the same Mongo topology
requirement before creating indexes. Local Mongo must be configured as a replica
set; the template URI now states replicaSet=rs0 instead of implying standalone
Mongo is sufficient. Do not start services while the code-completion gate is shut.

The canonical backend example now groups actual Go settings first and marks all
remaining Python-only settings separately; .env.example mirrors it. Use a private
HTTPS deployment origin and independently generated secrets after environment
setup. Data stores/deletion boundaries are recorded in data-lifecycle.md.

## Public web deletion configuration and staged container

The new public web flow uses FIREBASE_WEB_API_KEY, FIREBASE_WEB_AUTH_DOMAIN and
FIREBASE_WEB_APP_ID with the existing FIREBASE_PROJECT_ID. These are public Firebase
web app identifiers, separate from Admin SDK credentials. ACCOUNT_SUPPORT_EMAIL
and PRIVACY_POLICY_URL must be real and are required when web sign-in is enabled.
Partial configuration fails startup. Authorize the page hostname in Firebase Auth,
enable Google provider and add the exact page origin to CORS_ORIGINS. Settings are
optional/disabled in example.env until the user supplies values. See deployment.md
for the Google web flow, support ownership, CSP, container and final QA boundaries.

backend/Dockerfile targets Go 1.27.3 with a nonroot scratch runtime and CA bundle.
Its build context excludes secrets/Python, while source Python remains untouched
pending tested cutover. No image build or runtime check has run.

The support-account-deletion CLI adds explicit DELETION_DATABASE, admin/target
Firebase UIDs and an opaque DELETION_CASE_ID. Default planning is read-only;
DELETION_APPLY=true additionally requires DELETION_OWNERSHIP_VERIFIED=true.
APP_ENV/provider/database guards and fresh administrator checks still apply.
See deployment.md for the ownership-review procedure and deferred verification.

retire-legacy-data plans counts by default. Use explicit RETIREMENT_DATABASE/admin
Firebase UID; post-test apply also requires stopped-writer/tested-transition flags
and a real RETIREMENT_TEST_REFERENCE. See deployment.md for exact allowlists and
incremental behavior. No data/source retirement ran during implementation.

Frontend origin configuration now defaults to the staged Go port 8080 in debug.
Profile/release builds require an explicit HTTPS API_BASE_URL origin, with no
credentials/path/query/fragment. Missing/invalid configuration shows an unavailable
build message before provider initialization; the old hardcoded HTTP production
address was removed. example.env now targets the Go port. Client migration and
all runtime/HTTPS/build behavior remain unverified under the testing gate.

Route previews read public Flutter build-time MAP_TILE_URL, MAP_ATTRIBUTION and
MAP_ATTRIBUTION_URL from frontend/example.env. Defaults use OSM standard tiles
with visible contributor attribution and the default flutter_map tile cache.
No tile prefetch runs at startup. Follow the tile provider's capacity and usage
terms; configure a suitable provider for release traffic. Location picker is
still being migrated and is not yet governed by this complete preview setup.

## Place search and location selection

The Go API exposes authenticated `GET /api/v1/places/search?query=...` for
onboarded, unrestricted accounts. Configure `GEOCODING_BASE_URL` for your own
or contracted Nominatim-compatible API whose base path exposes `/search`, plus
`GEOCODING_ATTRIBUTION` and its HTTPS `GEOCODING_ATTRIBUTION_URL`. An optional
`GEOCODING_API_TOKEN` is sent as a server-only Bearer header. URLs allow HTTPS or
loopback HTTP, reject URL credentials/query/fragment, and do not follow redirects.
No mobile release is needed to switch the backend provider.

No public geocoder is enabled by default. `nominatim.openstreetmap.org` is
explicitly rejected: its [usage policy](https://operations.osmfoundation.org/policies/nominatim/)
forbids autocomplete and restricts generic generated geocoding integrations.
The removed picker made debounced autocomplete and pan-driven reverse requests.
The replacement sends only explicit name searches; it does not send map movement
or GPS coordinates to the geocoder. Search failure/unconfigured service leaves
map-pin, manual-coordinate and optional device-GPS selection available.

The adapter allows one in-flight upstream request, spaces starts at least 1.1s
apart, times out after 6s, bounds responses to 256KiB/five valid results and
caches up to 256 normalized queries for 24 hours in process memory. Busy requests
return 429 with Retry-After; failed requests are not cached. These are process
limits, not a distributed quota: provision contracted quota for all replicas or
route through a shared provider gateway. Search queries may include private
addresses; no query logging or persistent place-history collection is added.
Provider terms, privacy disclosure and real environment setup need final review.

2026-11-03: Go Firebase verification requires the verified token's Google
sign-in provider in production. Local demo-orix Auth emulator retains password
personas; production password/other-provider tokens are rejected. Flutter Google
server client ID is a build-time public setting rather than a hardcoded project
ID; configure the Firebase console/native files and GOOGLE_SERVER_CLIENT_ID as
needed. No provider, build, test or emulator checks ran; Google OAuth still
requires end-to-end QA. Firebase token claim reference: https://firebase.google.com/docs/rules/rules-and-auth

2026-11-03: Flutter onboarding and profile link to operator-configured HTTPS
PRIVACY_POLICY_URL. Onboarding cannot accept policy consent without a configured
URL. Removed stale campus-rule and ad-consent claims from the terms screen. The
owner must publish and configure an accurate policy and review consent/terms; no
legal compliance is claimed. Formatting only; no app/device checks ran.
