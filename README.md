<p align="center"><img src="website/public/orix-logo.svg" alt="ORIX" width="180"></p>

# ORIX

ORIX helps people coordinate shared autos, travel companions and lift offers.
Travelers choose a starting point, destination, departure window and safety
preferences; compatible requests are matched into groups. Group members can
coordinate their pickup, confirm a plan or reject a group and resume matching.

The project includes a Flutter app, a React website and an API backed by MongoDB.
Google sign-in, onboarding, profile photos, support, blocking and reporting are
part of the shared account experience. The website also includes administrative
controls and Android release management.

The Go backend migration remains in progress. The legacy Python backend is
retained until the API, persistence, background jobs and client contracts have
been verified. Recurring lift availability, background web push and further
integration work are still outstanding; see the current progress document.

## Project directories

- `frontend/` — Flutter mobile client.
- `website/` — React website and server-side administrator proxy.
- `backend/` — Python API and Go replacement under development.
- `Docs/` — configuration, implementation notes and verification records.

Start the website locally with `npm ci` and `npm run dev` from `website/`. Configure
services using the example environment files; keep real credentials in ignored
local files or your deployment secret manager.

See [integration progress](Docs/integration-progress.md), [environment setup](Docs/environment-setup.md),
[backend API](Docs/backend-api.md), [testing](Docs/testing.md) and [contributor instructions](AGENTS.md).
