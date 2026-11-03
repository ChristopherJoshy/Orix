# ORIX backend

Go replacement source lives in this existing backend tree: main.go for HTTP/CLI,
app/*.go for transport/repositories and domain modules, and internal/matching for
storage-independent route matching. Go minimum is declared in go.mod. Mongo must
support transactions and change streams. Firebase handles verified identity;
registration accepts any email domain.

Legacy Python remains until the complete transition passes final testing. The
Go-only Dockerfile is staged; it has not been built or deployed. Do not start
checks/services while the owner's code-completion gate is active.

See the canonical documentation in Docs:

- [API and changed legacy contracts](../Docs/backend-api.md)
- [Configuration and example.env](../Docs/configuration.md)
- [Deployment and public/support deletion workflows](../Docs/deployment.md)
- [Migration details](../Docs/migration.md)
- [Data lifecycle](../Docs/data-lifecycle.md)
- [Deferred test plan](../Docs/testing.md)
- [Current state](../Docs/memeory.md) and [history](../Docs/history.md)

example.env is canonical; .env.example mirrors it. Keep private values out of
version control. No Telegram credentials, college domain list or Redis service
are required by the Go replacement. Python-only settings are marked separately
while that source remains available for the final transition.
