# Fidelity Absorb Manifest

Approved scope: read-only local ledger built from official Fidelity.com CSV exports. No public retail HTTP API. No trades, transfers, login, cookies, or passwords.

## Tools surveyed
- gerardrbentley/fidelity-account-overview — positions CSV fixture and columns
- sztupkay/fidelity_csv_converter — history CSV to OFX
- redstreet/beancount_reds_importers — OFX/CSV importers; OFX is fragile
- justfetz/felix-august — positions import, footer skip, dry-run
- tkralphs/MoneydanceScripts — positions and history CSV import
- rabinnh/portfolioSummary — cross-account cash and symbol rollup
- ronitg1/alpha-terminal — idempotent CSV import; rejects stored credentials
- TylerFlar/claude-fidelity-mcp — Playwright MCP. Read tools absorbed. Login, 2FA, transfer, and order tools rejected (user constraint and Fidelity ToS)
- Hosted aggregators (Synci, Ledgerwire, SnapTrade, Plaid) — not dependencies

## Absorbed (match or beat everything that exists)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-------------|--------------------|-------------|
| 1 | Import positions CSV and skip disclaimer footer and pending-activity rows | gerardrbentley/fidelity-account-overview, justfetz/felix-august | fidelity-pp-cli import positions | Idempotent SQLite snapshot, --dry-run, --json |
| 2 | Import activity/history CSV | sztupkay/fidelity_csv_converter | fidelity-pp-cli import history | Queryable ledger instead of an OFX file that stops at conversion |
| 3 | List accounts seen in the export | claude-fidelity-mcp fidelity_get_accounts (read tool only) | fidelity-pp-cli accounts list | Offline, no browser session |
| 4 | List holdings with quantity, price, value, cost basis, gain/loss | claude-fidelity-mcp fidelity_get_positions, rabinnh/portfolioSummary | fidelity-pp-cli positions list | Filter by account and symbol, --json --select |
| 5 | List transactions | sztupkay/fidelity_csv_converter | fidelity-pp-cli transactions list | Date and symbol filters against the local store |
| 6 | Idempotent re-import | ronitg1/alpha-terminal | (behavior in fidelity-pp-cli import positions) same sha256 is a no-op; a new positions file replaces that account | Safe to re-run |
| 7 | Dry-run preview of an import | justfetz/felix-august | (behavior in fidelity-pp-cli import positions) --dry-run | No write until confirmed |
| 8 | Cash and money-market rows stay visible | rabinnh/portfolioSummary | (behavior in fidelity-pp-cli positions list) Type and symbol filters | SPAXX and FDRXX remain positions |

## Rejected (not shipping)
- fidelity_login, fidelity_submit_2fa, fidelity_save_session, fidelity_screenshot, fidelity_page_content — stored credentials and browser automation. Out of scope.
- fidelity_transfer, fidelity_place_order, fidelity_batch_orders — money movement and trades. Out of scope.
- Live quotes, OFX network download, SnapTrade, Plaid, WorkplaceXchange writes.

## Transcendence (only possible with our approach)

| # | Feature | Command | Score | Buildability | How It Works | Evidence | Long Description |
|---|---------|---------|-------|--------------|--------------|----------|------------------|
| 1 | Portfolio summary | fidelity-pp-cli portfolio summary | 10/10 | hand-code | Latest position snapshots plus imports: per-account and book market value, unrealized gain/loss, largest positions, cash versus invested (cash Type plus SPAXX/FDRXX), and a separately labeled last Cash Balance when that history column was stored. | Top Workflow 3. Competitors stop at a dashboard, an OFX file, or a sheet. | Use this command for current per-account and book totals: market value, cash versus invested, unrealized gain/loss, largest positions, and the snapshot as-of. Do NOT use this command for one symbol summed across accounts; use 'symbols total' instead. Do NOT use this command for dividend, interest, buy, sell, or fee totals; use 'activity summary' instead. Do NOT use this command for what changed since the previous import; use 'changes' instead. Do NOT use this command to see which accounts lack a positions or history import; use 'coverage' instead. |
| 2 | Activity summary | fidelity-pp-cli activity summary | 9/10 | hand-code | Local transactions summed by Action over an inclusive run-date window, including commission and fees. | Data-layer question "dividends this year". History columns from fidelity_csv_converter. | Use this command to total imported history amounts by Action over a run-date window, including Commission and Fees. Do NOT use this command for current market value, cash versus invested, or unrealized gain; use 'portfolio summary' instead. Do NOT use this command for one symbol's current shares and value; use 'symbols total' instead. Do NOT use this command for rows introduced by the latest import only; use 'changes' instead. |
| 3 | Changes since previous import | fidelity-pp-cli changes | 8/10 | hand-code | Retains the previous per-account position snapshot when a new file sha256 replaces that account, and lists history rows from the latest history import. | Top Workflow 4. | Use this command for holding deltas against the previous retained positions snapshot and for history rows introduced by the latest history import. Do NOT use this command for current book totals; use 'portfolio summary' instead. Do NOT use this command for a date-windowed action total such as dividends this year; use 'activity summary' instead. Do NOT use this command for one symbol's current cross-account total; use 'symbols total' instead. |
| 4 | Cross-account symbol total | fidelity-pp-cli symbols total | 8/10 | hand-code | Sums quantity, market value, cost basis, and unrealized gain/loss for one symbol across accounts, with a per-account breakdown. | "Where is my VTI" across brokerage and IRA rows in one positions file. | Use this command for one symbol's current quantity, market value, cost basis, and unrealized gain/loss summed across accounts, with a per-account breakdown. Do NOT use this command for book-level cash versus invested or largest positions; use 'portfolio summary' instead. Do NOT use this command for dividends or other history totals on that symbol; use 'activity summary' instead. Do NOT use this command for quantity or value deltas since the previous import; use 'changes' instead. |
| 5 | Per-account export coverage | fidelity-pp-cli coverage | 7/10 | hand-code | Per account, whether a positions snapshot and a history import exist, plus the min and max run date. | History is one file per account; positions can list every account. doctor only reports kinds globally. | Use this command to see, per account, whether a positions snapshot and a history import exist and the run-date span of imported transactions. Do NOT use this command for market value, cash, or gain/loss; use 'portfolio summary' instead. Do NOT use this command for holding deltas after an import; use 'changes' instead. |

Hand-code count: 5 (portfolio summary, activity summary, changes, symbols total, coverage). Spec-emits count: 0. No stubs.

## Gate
The invoking task already fixed the product: read-oriented balances, holdings, positions, and transactions from a documented interface, with no trades and no secrets in chat. This manifest is that product. Cloud run has no interactive approval prompt; generation proceeds on that briefing.
