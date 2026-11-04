# ORIX Android 2.1.3+9

Built 2026-11-04 from application source at 9b6d115 with the pubspec version
bumped to 2.1.3+9. Package: `com.orixridesharestudents.app`.

Artifact: `frontend/build/app/outputs/apk/release/Orix-v2.1.3.apk`.
The standard Flutter `outputs/flutter-apk/app-release.apk` also contains this build.
Size: 70,487,922 bytes. SHA-256: `447264a441a8490eeb43dbca1e9bb6cb8b3c04b53cefc3df580f877ea3461913`.

## Included changes

Automatic notification enrollment after verified sign-in/onboarding, one-time
native permission prompt, eager device-token synchronization, foreground recovery
and bounded transient registration retries. Donation dialogs and notification
settings use the updated monochrome layout. User-denied permissions remain off.

## Executed verification

- Signed release assembly succeeded in 3m 7s using checksum-verified official
  Amazon Corretto JDK 21.0.13. The initial Flutter invocation selected Java 25
  despite JAVA_HOME and failed Gradle startup; direct Gradle assembly pinned JDK21
  without modifying global Flutter settings or upgrading dependencies.
- apksigner passed (APK v2 signature); certificate SHA-256 matches the existing
  release certificate: `645ae843baebff7cc1231b91114dd429373ab29b961f71dc80a21b5757ac0702`.
- aapt confirmed package, versionName 2.1.3, versionCode 9, minSdk 24, targetSdk 36.
- Release API configuration was checked for HTTPS, server-only credential fields
  were absent from app defines, and internal test login was explicitly disabled.
- Scoped Flutter analysis and six notification payload/privacy tests passed in
  the preceding implementation change. These are not phone delivery tests.
- No backend source changes since the deployed backend release were found.
  VPS orix-backend service was active; /health returned healthy version 2.1.2 and
  /readyz returned ready. No backend restart or deployment was necessary.

No actual notification delivery, device install, background/reboot behavior or
payment was exercised. This APK has not been published to the public download
endpoint by this build task. Existing toolchain future-support/deprecation
warnings remain; no validation bypass flags were used.
