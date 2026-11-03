# ORIX API

The Go service handles accounts, matching, groups, chat, notifications,
safety, moderation and support. MongoDB stores the service data and Firebase
verifies sign-ins. A MongoDB deployment with transactions and change streams
is required.

## Run locally

1. Copy `example.env` to `.env` and supply the required local values.
2. Configure Firebase credentials outside Git and provide the credential path.
3. Run `go run .` from this directory.

The sample environment file describes provider and operational settings. A
contracted place-search provider and road-routing service are needed for the
corresponding live features. Administrative secrets belong only on the server.

See [configuration](../Docs/configuration.md), [API reference](../Docs/backend-api.md),
[environment setup](../Docs/environment-setup.md) and [current progress](../Docs/integration-progress.md).
The legacy Python implementation remains in this tree while its contracts and
background jobs are being migrated and verified. Do not treat the Go transition
as complete until those checks pass.
