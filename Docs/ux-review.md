# Accessible-screen UX review — 2026-11-03

Owner scoped review to screens accessible without supplying authentication.
Keep grayscale Kerala artwork, original pixel logo and readable body copy.

Removed the website's artificial two-second splash and workspace entrance fades.
Nested route loading retains navigation. Notification cleanup is bounded to eight
seconds; account checking has a thirty-second watchdog, listener error handling,
visible status and Retry. Flutter login returns actual backend auth status and
bounds token retrieval to twenty seconds. Tightened landing header/hero dimensions,
made “Plan a journey” the primary CTA, removed redundant copy and corrected the
recurring-lift claim. Login button uses a solid monochrome fill.

agent-browser visited `/`, `/login`, `/about`, `/privacy`, `/terms`,
`/account-deletion`, `/status/offline`, `/not-found` at 320×740, 390×844,
768×1024, 1024×768 and 1440×900. All 40 combinations had a main element and no
horizontal document overflow. Full-page captures are under `/tmp/orix-ux-review/`.
Landing desktop/mobile and mobile login received direct visual inspection.
These viewport checks do not represent every physical device.

axe: login 31 passes, no violations/incomplete checks; home 36 passes, no
violations, one incomplete contrast check on a decorative arrow. Website build
and all three existing error-copy checks passed. Focused Flutter analysis first
reported three existing brace-style infos, subsequently corrected.

No signed-in session was available. Private app/admin pages, onboarding, journey
forms, chat and moderation were not visually certified. Intermittent blank login
was not reproduced with a real account; identified loading hazards were addressed,
but authenticated and device follow-up is still required. No production fixtures,
fabricated people or authentication bypass were used. No backend tests, broad
regression checks or production performance benchmarks were run. Backend/account
verification hold otherwise remains in effect. Changes are not deployed by this review.

Focused Flutter analysis rerun: no issues found.
