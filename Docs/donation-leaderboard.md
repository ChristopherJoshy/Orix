# Web donation leaderboard

## Scope and operation

- Web-only: `/app/leaderboard`, linked from the workspace navigation.
- Administrators manage `/admin/leaderboard`: add, edit or remove a supporter,
  enter their verified cumulative donation total in rupees, confirm permission
  to publish, then select **Publish leaderboard**.
- The reader view contains a thank-you message, decorative native-grid grayscale
  pixel artwork and a two-column name/INR amount table. No refresh timer, totals
  dashboard, invented donors or donation events appear.
- Entries are manually curated; this does not verify a UPI payment, process a
  payment, sync bank records or grant permissions. Use an approved display name,
  never an email, phone number or payment reference. Remove an entry when its
  supporter withdraws publication permission. These editorial entries are not
  linked to ORIX account identities or automatically removed on account deletion.
- The board starts genuinely empty until an administrator publishes entries.
  Reads do not seed or initialize production data. No donors were added during
  implementation or deployment. Android source and releases are unchanged.

## API and persistence

- `GET /api/v1/donations/leaderboard`: existing verified-account middleware;
  returns `{entries:[{name,amount_paise}]}`, without management IDs or version.
- `GET /api/v1/admin/donation-leaderboard`: administrator + server-held secret;
  returns `{version,entries:[{id,name,amount_paise}]}`.
- `PUT /api/v1/admin/donation-leaderboard`: same authorization; body includes
  `version`, complete `entries` and `publication_confirmed:true`.
- Vercel BFF allowlist actions are `donation-leaderboard` (GET) and
  `set-donation-leaderboard` (PUT). No administrator secret reaches browser code.
- Go persists `app_config/_id=donation_leaderboard`. MongoDB transaction checks
  administrator authorization again, compares the version and audits the update
  without names or payment details. Concurrent/stale edits return 409; access
  revoked during save returns 403. Initial creation uses the unique document ID.
- At most 50 entries; unique management IDs; display names at most 60 Unicode
  characters/160 UTF-8 bytes, without control/format characters; positive integer
  paise totals up to ₹1,00,00,000 each. Highest amounts first; name/ID break ties.
  UI accepts rupees with at most two decimal places, without floating-point
  rounding. Existing records are replaced, not incremented, on publish.
- Failed saves retain the draft. An explicit, confirmed reload discards unsaved
  edits. Account changes hide and discard the editor snapshot. Mutations are not
  automatically retried. Existing bounded read retries remain in use.

## Focused verification — 2026-11-04

- `GOCACHE=/tmp/orix-go-cache go test -race ./app -run '^TestDonationLeaderboard'`
  passed: validation, ordering, reader projection, auth/role/secret gates, revoked
  access, publication confirmation, fractional-paise rejection and conflict/error
  HTTP mapping. These use isolated transport doubles, not real Firebase/Mongo QA.
- `npm run build` passed; TypeScript initially caught a nullable editor snapshot,
  which was corrected before the successful build.
- Money parser/INR response tests and BFF allowlist/secret-isolation/body-bound
  tests passed. Helpers/BFF were compiled with the installed TypeScript into
  `/tmp/opencode/orix-leaderboard-tests` and `orix-leaderboard-bff` respectively.
- Isolated browser review rendered the actual reader/editor components with
  clearly labeled layout-only inputs outside the repository, without account
  bypasses, production API reads or writes. Checked 320/375/640/900/1440 widths,
  longest name and maximum amount, empty/loading/error states, Retry, and admin
  add/remove/edit/consent controls. No horizontal overflow after correcting a
  shared nowrap table-header style locally in the leaderboard stylesheet.
- 200% root text scaling plus WCAG text-spacing overrides at 320/640/1440 produced
  no horizontal overflow for the reader/editor. Reduced-motion emulation removed
  the new entrance animation. Automated axe checks found no WCAG A/AA violations;
  the admin check flagged one incomplete shared gradient-button contrast check.
  Full browser zoom, screen readers and authenticated end-to-end save/reload,
  persistence/concurrent Mongo transactions are not verified.
- Broader app/backend QA remains on hold. No Android changes or tests.

## Deployment

The static Linux/amd64 Go binary was uploaded to the existing VPS and
`orix-backend` restarted successfully. Local `/health` and `/readyz` returned
healthy/ready; both new reader/admin GET routes returned 401 without a token.
The installed binary hash matched the local build. No donor records, test
accounts or production mutations were created. Website publication is left to
the existing Git-triggered Vercel deployment, without manual Vercel changes.
