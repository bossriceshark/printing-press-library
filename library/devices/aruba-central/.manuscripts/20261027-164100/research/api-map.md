# New Central API map used by aruba-central-pp-cli 0.1.0

Research date: 2026-10-27. Sources: the catalog scout for this print, HPE's public REST getting-started and token docs, pycentral v2 (`aruba/pycentral` branch `v2`, MIT), and the community `central-mcp-server` tool modules. No customer tenant was available, so none of these paths were called against a live cluster during this run.

## Scope

Target is New Central plus the monitoring view of subscription expiry that New Central exposes on the regional gateway. Classic Central (`apigw-*` hosts, username/password OAuth, refresh tokens) is out of scope and the client rejects those hosts. A GLP-scoped token (`POST https://global.api.greenlake.hpe.com/authorization/v2/oauth2/{workspace_id}/token`, about 15 minutes, needs a workspace id) is a different credential. v0.1 uses the Central-scoped Personal API client only, which is the same three variables as `central-mcp-server`.

## Auth

- `CENTRAL_BASE_URL` — regional gateway, for example `https://us5.api.central.arubanetworks.com`. Public cluster list includes us1, us2, us4, us5, us6, de1–de3, gb1, ca1, in1, jp1, au1, ae1, cn1, and an internal cluster.
- `CENTRAL_CLIENT_ID` and `CENTRAL_CLIENT_SECRET` — created under GreenLake → Manage Workspace → Personal API clients, scoped to the Central instance.
- Token: `POST https://sso.common.cloud.hpe.com/as/token.oauth2` with HTTP Basic and `grant_type=client_credentials` plus `client_id` in the body. That matches pycentral's `BackendApplicationClient` plus `HTTPBasicAuth` against `AUTHENTICATION["OAUTH"]`.
- Response field `expires_in` is cached. Central-scoped tokens are about 2 hours and there is no refresh token, so the CLI re-mints. A 401 invalidates the cache and mints once more.

## Rate limit

HPE documents an account-wide cap of about 10 requests/second, shared by every token on the tenant, with HTTP 429 on overflow. The client uses a token bucket defaulting to 10/s and sleeps on `Retry-After` before retrying. The product answer to repeated agent questions is `sync` into SQLite, then `brief` and `export` from that file.

## Paths the syncer calls

| Resource | Method and path | Pagination | Notes |
|---|---|---|---|
| sites | `GET network-monitoring/v1/sites-health` | `limit` + `offset`, `items` + `total` | pycentral `MonitoringSites.get_sites` |
| devices | `GET network-monitoring/v1/devices` | `limit` + `next` | Monitored inventory. Type values include `ACCESS_POINT`, switch, gateway |
| clients | `GET network-monitoring/v1/clients` | `limit` + `next` | Failed onboarding is a status filter on the stored rows |
| alerts | `GET network-notifications/v1/alerts` | `limit` + `next` | Same path as central-mcp-server `tools/alerts.py` |
| insights | `GET network-notifications/v1/insights` | `limit` + `next` | Fetched with alerts |
| events | `GET network-troubleshooting/v1/events` | `limit` + `next`, body key `events` | Requires `context-type=SITE`, `context-identifier`, `site-id`, `start-at`, `end-at`. Synced per site inside `--since` |
| firmware | `GET network-services/v1/firmware-details` | `limit` + `next` | Drift when running and recommended versions both exist and differ |
| subscriptions | `GET network-monitoring/v1/subscription-expiring-summary` | `limit` + `next` | New Central monitoring summary, not the GLP host |
| wlans | `GET network-monitoring/v1/wlans` | `limit` + `next` | pycentral `WLAN.get_wlans` |
| audits | `GET network-services/v1/audits` | `limit` + `next`, `start-at`/`end-at` | Central audit on the regional base URL |

`sites health` sorts the mirrored health score worst-first. Device lookup matches serial, then normalized MAC, then a unique name substring. Client top-N by usage is a SQL order on the mirror. WLAN stats are counts by band and status derived from that mirror.

## Writes

`device bounce-port` plans `POST network-troubleshooting/v1/{aps|cx|gateways|aos-s}/{serial}/portBounce`. It is not sent unless `ENABLE_WRITES=1` and `--yes`. `api` allows GET against a relative path and applies the same gate to other methods. Absolute URLs and `apigw` paths are refused.

## What was not generated

The public New Central surface is on the order of 32 OpenAPI files and about 1,736 operations, mostly split across the ReadMe hub, with config still largely `v1alpha1`. This package vendors a curated `spec.json` of the paths above instead of copying the hub or the `hpe-networking-mcp` vendor tree. The `api` command is the escape hatch for the rest.
