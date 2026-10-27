# Aruba Central CLI

Agent-native New Central client. It syncs a fleet into SQLite so later questions do not spend the account-wide 10 requests/second budget. Read-only unless `ENABLE_WRITES=1` and `--yes`. Classic Central is not implemented. No credentials are bundled.

Created by [@bossriceshark](https://github.com/bossriceshark) (Matt Rice).

## Try this (customer)

You bring your own HPE GreenLake Personal API client. The partner who shared this CLI does not have your tokens and cannot mint them for you.

### 1. Prerequisites

Install Go, then:

```bash
go install github.com/mvanhorn/printing-press-library/library/devices/aruba-central/cmd/aruba-central-pp-cli@latest
aruba-central-pp-cli --version
```

After this catalog entry is on the default branch, the installer path is:

```bash
npx -y @mvanhorn/printing-press-library install aruba-central --cli-only
```

Optional day-0 chat access uses the same three variables with the community MCP server. That server is read-only and does not build the local mirror:

```bash
uvx --prerelease=allow central-mcp-server
```

Setup notes: https://developer.arubanetworks.com/new-central/docs/central-mcp-setup

### 2. Create a GreenLake Personal API client

In HPE GreenLake, open Manage Workspace, then Personal API clients. Create a client scoped to the Central instance (Central-scoped clients mint about a 2 hour token). Copy the client id and secret once. They are shown only at creation.

Pick the regional New Central base URL for the workspace, for example `https://us5.api.central.arubanetworks.com`. The cluster list is on https://developer.arubanetworks.com/new-central/docs/getting-started-with-rest-apis. Token details: https://developer.arubanetworks.com/new-central/docs/generating-and-managing-access-tokens.

### 3. Export the three environment variables

These are the same names `central-mcp-server` uses:

```bash
export CENTRAL_BASE_URL="https://us5.api.central.arubanetworks.com"
export CENTRAL_CLIENT_ID="paste-client-id"
export CENTRAL_CLIENT_SECRET="paste-client-secret"
```

Alternatively, put them in `~/.config/aruba-central-pp-cli/config.toml` (mode 0600):

```toml
base_url = "https://us5.api.central.arubanetworks.com"
client_id = "paste-client-id"
client_secret = "paste-client-secret"
```

A `.env` file in the working directory is also read. Do not commit it.

### 4. doctor, sync, brief, export

```bash
aruba-central-pp-cli doctor
aruba-central-pp-cli sync --since 24h
aruba-central-pp-cli brief
aruba-central-pp-cli export --format json --out fleet.json
```

`doctor` prints JSON and exits non-zero when any variable is missing, before it calls Central. `sync` walks sites, devices, clients, alerts, events, firmware, subscriptions, WLANs, and audit into `~/.config/aruba-central-pp-cli/central.db`, staying under a client-side 10 requests/second budget and honoring `Retry-After`. `brief` reads that mirror: down devices, critical alerts, firmware drift, subscriptions inside 90 days, and the worst sites. `export` writes the mirror as one JSON document (or a directory of CSV files with `--format csv --out ./fleet`).

Narrow a sync while you learn the tenant:

```bash
aruba-central-pp-cli sync --only sites,devices,clients,alerts,events --since 24h
aruba-central-pp-cli devices list --status down --data-source local
aruba-central-pp-cli alerts list --severity critical --active --data-source local
```

### 5. What stays off

Reads are the default. `device bounce-port` and non-GET `api` calls print the request and do not send it unless `ENABLE_WRITES=1` and `--yes` are both set. There is no interactive prompt. Classic Central hosts (the `apigw` gateways) are rejected.

## Install

```bash
npx -y @mvanhorn/printing-press-library install aruba-central
```

CLI only:

```bash
npx -y @mvanhorn/printing-press-library install aruba-central --cli-only
```

Go fallback (Go 1.26.10 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/devices/aruba-central/cmd/aruba-central-pp-cli@latest
```

## Commands

| Command | What it reads |
|---|---|
| `auth login\|status\|refresh` | Client-credentials token cache |
| `doctor` | Env vars, token TTL, one sites-health probe |
| `sync [--only ...] [--since 24h]` | SQLite mirror |
| `sites list` / `sites health --worst N` | Site health |
| `devices list [--type ap\|switch\|gateway] [--status down]` / `devices get` | Inventory |
| `clients list [--site] [--failed]` / `clients get` / `clients top --by usage` | Clients |
| `alerts list [--severity critical] [--active]` / `alerts insights` | Notifications |
| `events list --device X --since 6h` | Events from the mirror |
| `firmware status [--drift]` | Running vs recommended |
| `subscriptions expiring [--within 90d]` | Monitoring expiry summary |
| `wlans list` / `wlans stats` | WLANs |
| `audit list --since 7d` | Central audit |
| `brief [--site]` | Fleet rollup |
| `export [--format json\|csv] [--out path]` | Mirror dump |
| `api METHOD path [--data @file.json]` | Relative New Central path |

## Rate limit

New Central allows about 10 requests/second for the whole account, shared with every other integration. The CLI's default `--rate-limit` is 10. Prefer `sync` once, then `--data-source local` for repeated agent questions.
