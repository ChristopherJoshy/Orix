<p align="center"><img src="website/public/orix-logo.svg" alt="ORIX" width="180"></p>

# ORIX

**Your route. Good company.**

[Use ORIX Ride Share on the web](https://orixrideshare.tech/) ·
[How ride sharing works](https://orixrideshare.tech/about)

ORIX helps people going the same way coordinate shared autos, travel companions
and lift offers. Choose your route, departure window and preferences; ORIX finds
compatible people. Review your group, agree on a pickup, and decide whether to go.

<p align="center"><img src="website/public/art/orix-street-corner.png" alt="Travelers meeting beside an auto-rickshaw" width="420"></p>

## Explore the project

- [`frontend/`](frontend/README.md) — Flutter app for planning and coordinating journeys.
- [`website/`](website/README.md) — web experience and administrator workspace.
- [`backend/`](backend/README.md) — Go API, matching and background services.
- [`Docs/`](Docs/) — setup, contracts, progress and verification notes.

Google sign-in connects the app and website to the API. Onboarding collects a
profile and travel preferences; groups include chat, confirmation and safety
controls. Moderators can review reports and support conversations.

## Get started

Configure each service from its example environment file. Run the Go API with
`go run .` inside `backend/`; run the website with `npm ci` and `npm run dev`
inside `website/`; use Flutter tooling from `frontend/`. Keep credentials in
ignored local files or a deployment secret manager.

Recurring lift availability is not implemented yet. Some integration and deployment
flows still need verification; [current progress](Docs/integration-progress.md)
records their status. See [environment setup](Docs/environment-setup.md) and
[contributor instructions](AGENTS.md) for details.
