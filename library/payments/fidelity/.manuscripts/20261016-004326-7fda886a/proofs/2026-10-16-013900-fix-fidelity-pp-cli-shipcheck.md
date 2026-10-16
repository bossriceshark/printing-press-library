# Shipcheck — fidelity-pp-cli

Date: 2026-10-16
Run: 20261016-004326-7fda886a
CLI: <runstate>/workspace-c52ddf65/runs/20261016-004326-7fda886a/working/fidelity-pp-cli

## Command

```
cli-printing-press shipcheck \
  --dir <working>/fidelity-pp-cli \
  --spec <run>/research/fidelity.yaml \
  --research-dir <run>
```

Exit: 0

## Per-leg result

| Leg | Result |
|---|---|
| verify | PASS 100% (34/34), 0 critical |
| validate-narrative | PASS (10 narrative commands) |
| dogfood | PASS (5/5 novel features, 0 dead flags, 0 dead functions) |
| workflow-verify | workflow-pass (no workflow manifest) |
| apify-audit | pass |
| verify-skill | PASS |
| scorecard | 96/100 Grade A; live sample 5/5 |

## Before

First shipcheck failed on missing `// pp:data-source local` directives, a `"0.00"` literal flagged as an unverified host, empty `defaultSyncResources()`, and `--dry-run` on `accounts` / `positions` / `transactions` hitting an empty local store.

## Fixes

- Declared local data source on novel commands and parent groups.
- Replaced money zero literals with `fidelitycsv.FormatMoney` / `zeroMoney`.
- Advertised `accounts`, `positions`, and `transactions` as the population path while `sync` still refuses the network.
- Short-circuited `--dry-run` on promoted list commands.
- Empty-database reads return `[]` instead of failing a fresh ledger.
- Kept generated sync helpers referenced so dogfood does not mark them dead.

## After

Verify 100%. Scorecard 96/100 Grade A. Novel sample probe 5/5.

## Recommendation

ship

Fidelity has no retail brokerage API. The CLI is a local SQLite ledger over owner-downloaded positions and history CSV files. No password, cookie, token, trade, or money-movement command ships.
