# Acceptance / shipcheck — run 20261027-164100

Live New Central was not called. No Personal API client was available. The checks below are the ones that ran in the publish environment on 2026-10-27.

## go test

```
ok  internal/central
ok  internal/config
ok  internal/client
ok  internal/cli
```

`TestSyncBriefExport` stands up httptest routes for the token endpoint and the sync paths, runs `syncer.Run`, and checks the SQLite brief:

- one down device (`status` Down, serial SN1)
- one critical alert
- one firmware drift row (running 1.0, recommended 2.0)
- export table `sites` length 1

`TestDoctorMissingCreds` executes `doctor` with an empty TOML file and empty `CENTRAL_*` variables. Exit code 10. Stdout JSON includes `missing` = `CENTRAL_BASE_URL`, `CENTRAL_CLIENT_ID`, `CENTRAL_CLIENT_SECRET`.

`TestBouncePortDryRun` runs `device bounce-port --serial SN1 --ports 1` with writes disabled. Stdout has `dry_run: true` and path `portBounce`. Nothing is sent.

`TestLimiterFakeClock` checks the 10/s bucket sleeps. `TestTokenMintAndCache` checks Basic auth and that a second call does not remint. `TestClassicRejected` rejects `apigw-prod2.central.arubanetworks.com`.

## Binary smoke

`go build -o` `./cmd/aruba-central-pp-cli` succeeded.

```
aruba-central-pp-cli 0.1.0
```

`--help` lists auth, doctor, sync, sites, devices, clients, alerts, events, firmware, subscriptions, wlans, audit, brief, export, api, device, which.

`doctor --config` of an empty file, with the three variables unset:

```json
{
  "ok": false,
  "error": "missing credentials",
  "missing": ["CENTRAL_BASE_URL", "CENTRAL_CLIENT_ID", "CENTRAL_CLIENT_SECRET"]
}
```

Process exit status 10.

`sync --dry-run --only sites,devices --since 24h` prints a plan and does not open a token request.

## Not run

- Live `doctor` probe of `network-monitoring/v1/sites-health` on a customer cluster
- `govulncheck ./...` with the module's `toolchain go1.26.10` reported no reachable vulnerabilities. The catalog CI image is Go 1.26.3; the toolchain line is what clears the stdlib findings fixed after that release.
- Full ReadMe-hub operation generation
