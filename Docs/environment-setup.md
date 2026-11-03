# Orix configuration and deployment

Local configuration was created from the owner-provided Firebase Android/Web
clients, Admin SDK file, MongoDB and Redis connection details. Credentials are
not repeated in documentation. Database name: **Orix-Prod**. The local API points
to that database; user testing can create real account records. Automated tests
must continue to use their separate local test database and emulator project.

## Files

- `backend/.env`: server-only connections, Firebase project/file path, independent
  generated admin/JWT secrets, CORS origins and public service URLs.
- `frontend/.env`: public Flutter defines. Run from frontend with
  `flutter run --dart-define-from-file=.env`. Android emulator uses
  `http://10.0.2.2:8080`; never ship this as a public release origin.
- `frontend/android/app/google-services.json`: moved from frontend root after
  matching package `com.orixridesharestudents.app` and Firebase project identity.
- `frontend/lib/firebase_options.dart`: Android and web public identifiers updated
  to the supplied project. Removed the old test-project/iOS placeholder options.
- `website/.env.local`: public `VITE_*` client values plus server-only
  `ORIX_API_ORIGIN` and `ORIX_ADMIN_API_SECRET` for the admin BFF.
- The owner-provided Admin SDK file stays in backend; its path is referenced by
  the backend env. All local envs, the native Firebase config and Admin SDK file
  are Git-ignored; credential-bearing local files have mode0600.

The `.env.example` files contain placeholders, never real passwords. Backend
`example.env` points to the canonical template instead of duplicating values.

## Public deployment

Website: `https://orixrideshare.tech/` (Vercel; not deployed by this work).
Support: `Christopherjoshy4@gmail.com`.

The website host is not a deployed API. Deploy the long-running Go API to a host
supporting its WebSockets/workers and set a reachable HTTPS API origin in both
clients and the Vercel server environment. Configure Atlas network access and a
least-privileged database user scoped to Orix-Prod. The Go transition remains
staged; unit/build results alone do not verify the full runtime migration.

In Vercel set the public values from website/.env.example and set
`ORIX_API_ORIGIN` to the API's HTTPS origin. `ORIX_ADMIN_API_SECRET` must equal the
backend `ADMIN_API_SECRET`; never rename it with a VITE_ prefix. The Vercel admin
function forwards a Firebase bearer token plus the server-only shared secret to
an allowlisted backend route. Database and service-account credentials never go
to Vercel browser builds or editable public settings.

Enable Google sign-in in Firebase Authentication, add the actual site domains to
Authorized Domains, and register the Android signing SHA fingerprints. Public
Firebase keys identify a project; they do not authorize data access. Server-side
Firebase verification, account status and administrator role checks remain in
force. Analytics is deliberately not initialized before a tracking-consent flow.
A real published and reviewed privacy policy is still needed before launch.

Rotate the DB/Redis passwords shared in the conversation before launch. The
supplied Redis URI uses plain redis://; obtain/enable the provider's TLS endpoint
and update REDIS_URL to rediss:// before using it in production. Do not assume
that changing only the scheme makes an existing endpoint TLS-capable. Go does
not consume Redis/JWT; these values support the retained Python runtime.

## Administrator and Android releases

No administrator role was inferred from a support email or granted automatically.
Use the existing reviewed `go run . bootstrap-admin` workflow with an existing
verified Firebase UID and explicit APP_ENV/database/apply selections. Never put
bootstrap flags into the ordinary runtime env. See backend/bootstrap_admin.go.

Operations now includes a connected Android release panel. Public settings are
stored in app_config/public: Android prompt, official Orix Google Play URL and
support email. Secrets, signing keys, database connections and provider keys
remain infrastructure settings rather than browser-admin fields.

APK uploads use a five-minute, single-use, administrator/file-scoped ticket from
the BFF, then transfer directly to the API to avoid Vercel's function body limit.
Actual upload bytes drive progress; the server verifies size, SHA-256 and APK ZIP
structure. Limit:256MiB, two concurrent uploads per API process. The administrator
must independently verify the release signature/package/install behavior; this
code does not claim cryptographic APK signer verification. Publish signed release
APKs only; the emulator debug build has NOT been published as a public download.

GridFS bucket android_releases stores the file; a transaction switches the single
current pointer and queues the previous file for deletion. Cleanup runs immediately
and retries every30seconds from persistent apk_cleanup records. It removes the
previous GridFS file and chunks; old metadata is not retained as another release.
The public Android prompt falls back to APK when no Play Store URL exists, and
shows an honest unavailable state until an administrator publishes an APK.
Reverse proxies must support the documented upload size and five-minute timeout.

## Implementation and evidence

- First configured Android build passed; the installed app then exposed a plain
  Provider/ChangeNotifier registration error. Corrected NotificationService to
  ChangeNotifierProvider.value. Rebuild/install/relaunch passed, and a captured
  emulator frame confirmed the Google login screen instead of the red error.
- New Android onboarding includes photo upload/confirmation, lift interest,
  optional notification/location prompts, safety/privacy consent and a replayable
  pixel tour. Permissions are no longer requested before the login screen.
- New web Google auth and onboarding call the real API. The remaining journey/
  community/admin preview adapters are NOT represented as production integration.
- Initial web production build passed after Firebase integration. npm install
  reported four high findings through a transitive grpc package. Official SDK
  release notes and npm registry were consulted; a grpc-js1.14.7 override addresses
  those findings. Patched install reports zero vulnerabilities. Final UI/auth,
  upload/replacement and policy verification are still required.

## Research

- https://firebase.google.com/docs/admin/setup
- https://firebase.google.com/docs/auth/web/google-signin
- https://firebase.google.com/docs/web/learn-more
- https://firebase.google.com/support/release-notes/js
- https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html
- https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html
- https://www.mongodb.com/docs/drivers/go/current/crud/gridfs/
- https://github.com/grpc/grpc-node/releases

## Browser push configuration (staged)

Set `VITE_FIREBASE_VAPID_KEY` to the public Web Push certificate key from the
Firebase project's Cloud Messaging settings. Configure the matching Firebase
project for the deployed HTTPS origin and authorize that origin for sign-in.
The key is public configuration; keep Firebase Admin credentials and the admin
API secret server-side. Browser permission alone is insufficient: users opt in
from Settings, which registers the browser with the API.

The implementation sends a generic notification with an inbox identifier. The
service worker opens the inbox; it does not include private journey or message
content. Tokens are separate from Android and expire after 30 days without a
registration refresh. Delivery is at least once: a crash between provider
acceptance and the outbox checkpoint can result in duplicate provider sends.
Browser/FCM registration, permission revocation, multi-account transitions,
background delivery and token expiry have not yet been verified end to end.
