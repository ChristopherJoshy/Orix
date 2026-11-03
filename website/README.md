<p align="center"><img src="public/orix-logo.svg" alt="ORIX" width="160"></p>

# ORIX website

The website connects travelers to the same journeys, groups and support flows as
the app. It also contains the administrator workspace.

<p align="center"><img src="public/art/orix-street-corner.png" alt="Travelers meeting beside an auto-rickshaw" width="400"></p>

## Run locally

1. Copy `.env.example` to `.env.local` and fill in the Firebase web settings and
   API origin. Configure the server-only admin values for the admin workspace.
2. Run `npm ci`, then `npm run dev`.
3. Start the Go API separately using the [backend setup](../backend/README.md).

Browser notification opt-in requires the public Firebase VAPID key. Keep the
admin secret and Firebase Admin credentials server-side; never expose them as
`VITE_` variables. For Vercel, use this directory as the project root and set
the production environment values there. See [environment setup](../Docs/environment-setup.md)
and [current integration status](../Docs/integration-progress.md).

The production API origin must use HTTPS and be reachable from browsers. The
admin proxy uses the same API origin and a separately configured server-only
secret. Vercel's Git integration deploys the production branch from this folder.
The project uses `main` as its production branch.

## Public search pages

`npm run build` renders the existing public components to six HTML pages with
route-specific metadata and generates their canonical sitemap. Real public
Firebase/API web configuration is required in each build environment; private
service credentials must never use `VITE_` names. Private routes use a separate
noindex SPA shell; unknown routes serve the existing not-found design with HTTP
404. `npm run test:seo` checks the generated artifacts.

After verifying the matching production deployment, run `npm run seo:notify` to
notify IndexNow of the public URLs. This verifies the live public ownership file
first. Submission does not guarantee indexing or ranking. Google ownership and
sitemap submissions are managed in Search Console. See the [SEO audit](../Docs/seo-audit.md)
for research, actual execution evidence and remaining limits.
