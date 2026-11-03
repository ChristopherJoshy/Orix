# ORIX Android 2.1.2+8

Built 2026-11-03 from commit cbb4d12. Package: `com.orixridesharestudents.app`.
Size: 70,488,050 bytes (70.5 MB). Release build completed in 170 seconds.

Publication is pending explicit owner approval for the server-side credential
source after automatic approval review blocked reading the live service environment.
The command did not run and no service credentials were read or copied. The public
download still advertised 2.1.1 at the last check. The verified new APK is local at
`frontend/build/app/outputs/flutter-apk/app-release.apk` and
`frontend/build/app/outputs/apk/release/Orix-v2.1.2.apk`.

## APK file checksums

- SHA-1: `bf304e890623a6408e1bfd99e94ffdb6db5ee92e`
- SHA-256: `a4a1511872ed6e2e1dad397f8e034ffbd642132a7145f25f0cc35f86c9f35eb7`

## Release signing certificate fingerprints

Use these certificate fingerprints when a service asks for the Android signing
key (for example Firebase). They are different from the APK file checksums.

- SHA-1: `AE:F6:91:27:49:FD:EE:67:DB:FB:69:0D:96:4C:62:F2:51:6A:65:A5`
- SHA-256: `64:5A:E8:43:BA:EB:FF:7C:C1:23:1B:91:11:4D:D4:29:37:3A:B2:9B:96:1F:71:DC:80:A2:1B:57:57:AC:07:02`

## Executed verification

- Android apksigner verification passed; its certificate matches the existing
  private release keystore. The APK is not signed with the Android debug certificate.
- Android aapt reported versionName 2.1.2, versionCode 8 and the expected package.
- The packaged donation QR matches the verified source asset exactly. Both shipped
  QR assets and the rendered desktop QR decode to the supplied payment payload;
  generated QR decoding also passed at five sizes. No payment was initiated.
- Ten focused Flutter donation/payload/update-link checks passed; scoped analysis
  is clean. Android Kotlin compilation passed with official checksum-verified JDK21.
- Focused Go race tests for reminders, notification/APK/version contracts and the
  operator selection guards passed. Go vet and static Linux backend build passed.
- Website production build, donation-link/date checks, push-worker public-copy/
  privacy/validation checks and Python worker py_compile passed.
- The updated backend is active; its configured API readiness returned 200 and the
  new admin reminder route returned 401 without authentication.
- The live public /download page loads at mobile width without document overflow.
  Live crawler HTML includes complete image metadata; the 253,902-byte deployed
  JPEG matches the export from the generated source art byte for byte.

No real push delivery, authenticated multi-user database concurrency, device
background/reboot, Google Pay handoff/payment or full regression checks ran. Native
notification chrome remains OS-controlled. Optional Manglish reminders require
explicit opt-in and are scheduled four hours apart in daytime IST windows. The
owner's broader verification hold remains in effect.

Build warnings: the existing Gradle/AGP/Kotlin versions report future-support
warnings; one Flutter map package is discontinued. Dependencies were not upgraded
or compatibility checks bypassed in this release. The installed JDK25 initially
failed Gradle startup; the official JDK21 build succeeded.
