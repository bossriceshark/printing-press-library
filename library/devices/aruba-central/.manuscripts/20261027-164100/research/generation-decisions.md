# Generation decision — 2026-10-27

`cli-printing-press` / `printing-press` was not on `PATH` and no local generator checkout was present in the publish environment. The owner ask was a customer-shareable draft on `feat/aruba-central` with sync → SQLite, brief, export, doctor, and a Try this (customer) README, using the same three env vars as `central-mcp-server`, New Central only.

That ruled out waiting on a full generator reprint of ~1,736 operations. The tree under `library/devices/aruba-central/` is therefore a curated v0.1:

- OAuth2 client-credentials against `https://sso.common.cloud.hpe.com/as/token.oauth2`, token file mode 0600, no secret in `doctor` output.
- Token bucket at 10 requests/second and Retry-After backoff.
- SQLite mirror (`modernc.org/sqlite`, no CGO) for sites, devices, clients, alerts, events, firmware, subscriptions, WLANs, audits, and insights.
- `brief` and `export` read that mirror.
- Writes default to a dry-run plan.

Live Central dogfood was not run. The environment has no `CENTRAL_CLIENT_ID` or `CENTRAL_CLIENT_SECRET`. Structural coverage is `go test ./...` plus a httptest server that implements the sync paths. `doctor` against an empty config file exits 10 and names the three variables.

Category is `devices`, next to the other hardware CLIs in this catalog. `registry.json` and `cli-skills/` are left for the post-merge generators.
