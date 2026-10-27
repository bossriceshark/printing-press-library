---
name: pp-aruba-central
description: "Printing Press CLI for HPE Aruba New Central. Syncs sites, devices, clients, alerts, and events into SQLite so agents can brief and export without burning the account rate limit."
author: "Matt Rice"
license: "Apache-2.0"
argument-hint: "<command> [args]"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - aruba-central-pp-cli
---

# Aruba Central — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `aruba-central-pp-cli` binary. Verify it is installed before invoking any command. If it is missing, install it first:

1. Install via the Printing Press installer:
   ```bash
   npx -y @mvanhorn/printing-press-library install aruba-central --cli-only
   ```
2. Verify: `aruba-central-pp-cli --version`
3. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

The customer creates a GreenLake Personal API client scoped to the Central instance and exports `CENTRAL_BASE_URL`, `CENTRAL_CLIENT_ID`, and `CENTRAL_CLIENT_SECRET`. This package contains no credentials.

## Command Reference

**auth** — OAuth2 client-credentials token cache

- `aruba-central-pp-cli auth login` — Mint and cache a token
- `aruba-central-pp-cli auth status` — Show TTL without printing the token
- `aruba-central-pp-cli auth refresh` — Force a new token

**doctor** — Credential and cluster check

- `aruba-central-pp-cli doctor` — Fail with JSON when the three env vars are missing

**sync** — Local SQLite mirror

- `aruba-central-pp-cli sync` — Pull sites, devices, clients, alerts, events, firmware, subscriptions, wlans, and audits

**sites** — Site inventory and health

- `aruba-central-pp-cli sites list` — List sites
- `aruba-central-pp-cli sites health` — Worst sites first

**devices** — Inventory

- `aruba-central-pp-cli devices list` — Filter by type and status
- `aruba-central-pp-cli devices get` — Lookup by name, serial, or MAC

**clients** — Associated clients

- `aruba-central-pp-cli clients list` — Filter by site or failed onboarding
- `aruba-central-pp-cli clients get` — Lookup by MAC or username
- `aruba-central-pp-cli clients top` — Rank by usage

**alerts** — Alerts and insights

- `aruba-central-pp-cli alerts list` — Filter by severity and active state
- `aruba-central-pp-cli alerts insights` — Notification insights

**events** — Time-bounded events from the mirror

- `aruba-central-pp-cli events list` — Filter by device and window

**firmware** — Running versus recommended

- `aruba-central-pp-cli firmware status` — Show drift

**subscriptions** — Expiry summary from New Central monitoring

- `aruba-central-pp-cli subscriptions expiring` — Window such as 90d

**wlans** — SSIDs

- `aruba-central-pp-cli wlans list` — List WLANs
- `aruba-central-pp-cli wlans stats` — Counts by band and status

**audit** — Audit log

- `aruba-central-pp-cli audit list` — Records since a window

**brief** — One-call fleet rollup from the mirror

- `aruba-central-pp-cli brief` — Down devices, critical alerts, drift, expiring subscriptions, worst sites

**export** — Dump the mirror

- `aruba-central-pp-cli export` — JSON or CSV

**api** — Relative New Central path. Non-GET stays dry-run unless writes are enabled.

- `aruba-central-pp-cli api GET network-monitoring/v1/sites-health` — Raw method and relative path

**device** — Gated writes

- `aruba-central-pp-cli device bounce-port` — Dry-run unless ENABLE_WRITES=1 and --yes

**which** — Command lookup

- `aruba-central-pp-cli which` — Match a short capability query

## Auth Setup

```bash
export CENTRAL_BASE_URL="https://us5.api.central.arubanetworks.com"
export CENTRAL_CLIENT_ID="your-client-id"
export CENTRAL_CLIENT_SECRET="your-client-secret"
aruba-central-pp-cli doctor --json
```

Or set `base_url`, `client_id`, and `client_secret` in `~/.config/aruba-central-pp-cli/config.toml`.

## Agent mode

`--agent` turns on `--json`, `--no-input`, and `--yes`. Writes still do nothing until `ENABLE_WRITES=1`.

Read commands use `--data-source auto` (local mirror when it has rows, otherwise live). `brief` and `export` always read the SQLite mirror.

```bash
aruba-central-pp-cli sync --only sites,devices,clients,alerts,events --since 24h
aruba-central-pp-cli brief --site HQ
aruba-central-pp-cli export --format json --out fleet.json
aruba-central-pp-cli devices list --type ap --status down
aruba-central-pp-cli devices get ap-lobby
aruba-central-pp-cli clients list --site HQ --failed
aruba-central-pp-cli clients top --by usage
aruba-central-pp-cli alerts list --severity critical --active
aruba-central-pp-cli events list --device SN1 --since 6h
aruba-central-pp-cli firmware status --drift
aruba-central-pp-cli subscriptions expiring --within 90d
aruba-central-pp-cli audit list --since 7d
aruba-central-pp-cli which "offline devices"
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error |
| 3 | Not found in the mirror |
| 4 | Authentication failed |
| 5 | API error |
| 7 | Rate limited |
| 10 | Missing configuration |
