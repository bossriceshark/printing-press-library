# Phase 4.85 output review — fidelity-pp-cli

status: PASS

Review path: direct sample against a synthetic owner CSV (no live fidelity.com session).

Imported a 3-row Portfolio Positions file and a 1-row history file, then ran the headline commands with `--agent`.

- `portfolio summary` book market value 400.00, cash 100.00 (SPAXX), invested 300.00, unrealized gain 15.00. Per-account rows match the file.
- `activity summary --from 2026-01-01 --action DIVIDEND` returned one DIVIDEND RECEIVED row for 12.50.
- `symbols total VTI` summed 12 shares, market value 300.00, cost basis 285.00, gain 15.00 across Brokerage and Roth IRA.
- `coverage` marked the Roth account positions-only and the brokerage account positions plus history from 2026-01-15.
- `doctor --json` reports auth not required and an empty base URL.

No wrong totals, silent row drops, or trade/output side effects. Unknown symbols return a zero total with a note, which is the intended empty-ledger shape.
