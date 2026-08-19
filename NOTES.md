# NOTES.md — the user's world

## What I know so far (thin — must be fleshed out by interview)

- **Role:** engineer / technical lead building Avandab/MVTMS, a fleet-management SaaS in Go
  (repo: `/home/abhishek/Desktop/temux/basic`).
- **Way of working:** spec-driven — specs in `docs/tech-specs/`, migration ownership index,
  sub-tasks (1A→1G, 2A→2C), "Prove It" protocol (build/vet/test before claiming done),
  verification reports ending every task.
- **Tooling visible in repo:** Go 1.26, chi, SQLite (modernc), goose migrations, casbin RBAC,
  Datastar + HTMX templates, Telegram/founder notifications, MQTT telemetry, outbox-relay
  event bus, sqlc-generated queries, CI via GitHub Actions.
- **Channels I process (inferred, unverified):** GitHub issues/PRs, CI runs, migration
  conflicts, spec review.
