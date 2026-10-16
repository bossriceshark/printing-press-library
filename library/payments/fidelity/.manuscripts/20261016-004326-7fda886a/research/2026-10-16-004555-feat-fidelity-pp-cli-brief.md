# Fidelity CLI Brief

## API Identity
- Domain: Fidelity Investments retail brokerage and retirement accounts at https://www.fidelity.com/ (personal brokerage, IRA, Roth IRA, CMA, workplace accounts the customer can export from the website). This is not Fidelity International (`developer.fil.com`) and not the WorkplaceXchange plan-sponsor portal.
- Users: the account owner and an agent acting for that owner. Read-only. No advisor, plan-sponsor, or trading desk workflows.
- Data profile: account identity, cash and margin balances, positions (symbol, quantity, price, market value, cost basis, gain/loss), and activity/transaction history. All of it is owner-private. Nothing in this CLI calls a brokerage order or money-movement endpoint.

## Reachability Risk
- High for any unofficial fidelity.com session, cookie, or stored-password client. Fidelity Investments does not publish a retail brokerage API. API Evangelist index `apis: []` with `x-no-public-api: true` (fetched 2026-10-16, HTTP 200, https://raw.githubusercontent.com/api-evangelist/fidelity-investments/refs/heads/main/apis.yml). Fidelity Investments moderators on r/fidelityinvestments have repeatedly said there is no public API and pointed customers at CSV/XLS download from Fidelity.com and Active Trader Pro (https://www.reddit.com/r/fidelityinvestments/comments/vk8gp2/ and https://www.reddit.com/r/fidelityinvestments/comments/1fgfxwu/).
- None for the chosen surface: local import of files the owner downloads while logged in. No network credential, no session cookie, no MFA.
- Fidelity International developer catalog https://developer.fil.com/apis returned HTTP 200 and the page text "No API documentation is currently available." That portal is a different company (FIL) and is empty.
- WorkplaceXchange WI Balances (https://workplacexchange.fidelity.com/public/wpx/docs/wi-balances, HTTP 200) is a plan-sponsor / investment-professional partner API ("For plan sponsor and investment professional use only"). The public page is marketing copy; the authentication URL https://workplacexchange.fidelity.com/public/wpx/api-catalog/authentication returned HTTP 200 with a rendered title "Not Found | Fidelity WorkplaceXchange". No downloadable OpenAPI file was published. It is not a self-serve personal-brokerage API and it includes mutating HR surfaces (elections, loans, participant create). Out of scope.
- Fidelity Access / Akoya is the official third-party read path (OAuth on Fidelity's page, data recipient receives tokens). It is not a public client-credentials API an individual can call with an env var. Aggregators such as SnapTrade sit on that rail. Out of scope for this CLI; documented so agents do not scrape instead.
- Official export help URL https://www.fidelity.com/customer-service/faqs-exporting-account-information redirected (HTTP 200) to https://digital.fidelity.com/prgw/digital/customer-service/ and did not expose a machine-readable spec.
- Probe-safe endpoint: none. There is no unauthenticated retail HTTP probe.

## Top Workflows
1. Import a Positions CSV (`Portfolio_Positions_MMM-DD-YYYY.csv`) the owner downloaded from Accounts & Trade → Portfolio → Positions, then list holdings by account and symbol.
2. Import an Activity CSV (`History_for_Account_<id>.csv`) and list transactions (dividends, buys, sells, transfers) filtered by account and date. Read-only; no order placement.
3. Summarize the imported book: per-account market value, cash vs invested, unrealized gain/loss, and largest positions.
4. Re-import a newer export idempotently (same file or newer snapshot replaces positions for that account; history rows dedupe) and answer "what changed since the last import".
5. `doctor` reports whether a local ledger exists, which export kinds have been imported, and that no brokerage credential is configured or required.

## Table Stakes
- Positions CSV columns, from the public fixture https://github.com/gerardrbentley/fidelity-account-overview/blob/main/example.csv (HTTP 200): `Account Number`, `Account Name`, `Symbol`, `Description`, `Quantity`, `Last Price`, `Last Price Change`, `Current Value`, `Today's Gain/Loss Dollar`, `Today's Gain/Loss Percent`, `Total Gain/Loss Dollar`, `Total Gain/Loss Percent`, `Percent Of Account`, `Cost Basis`, `Cost Basis Per Share`, `Type`. Files include a disclaimer footer and a "Date downloaded …" row that must be skipped. Money cells use `$` and `+`/`-` prefixes and percents use `%`.
- History CSV required columns, from https://github.com/sztupkay/fidelity_csv_converter (HTTP 200): `Run Date`, `Action`, `Symbol`, `Description`, `Type`, `Price ($)`, `Quantity`, `Commission ($)`, `Fees ($)`, `Amount ($)`. Community docs also cite `Accrued Interest ($)`, `Cash Balance ($)`, `Settlement Date`. Filename pattern `History_for_Account_<account>.csv`.
- Account number and account name come from the positions file (and from the history filename when the history file has no account column).
- Idempotent re-import. Do not duplicate history rows. Do not commit raw CSVs or account numbers into git.
- Currency-aware parsing. Footer/disclaimer rows are not positions.
- No trade, transfer, bill-pay, or wire commands.
- Competitors: `sztupkay/fidelity_csv_converter` (history CSV → OFX only), `redstreet/beancount_reds_importers` (OFX plus Fidelity CSV importers for beancount), `gerardrbentley/fidelity-account-overview` (positions CSV dashboard), Finance-Guru skills (manual sheet sync from the same CSVs). None of them is an agent CLI with a local SQLite ledger and compound portfolio queries. OFX direct download is fragile; community issues (redstreet/beancount_reds_importers#66) treat CSV as the fallback when OFX breaks. This CLI standardizes on CSV.
- npm/PyPI: PyPI search `fidelity brokerage` returned HTTP 200 HTML and no maintained official SDK. No official npm package.

## Data Layer
- Primary entities: `accounts`, `position_snapshots` (one row per account+symbol for the latest import), `transactions`, `imports` (source file basename, sha256, kind, imported_at, row_count).
- Sync cursor: content sha256 of each imported file plus the export's "Date downloaded" line when present. Re-importing the same bytes is a no-op.
- FTS/search: symbol, description, action, account name. Agents ask "where is my VTI" and "dividends this year" against the local ledger.
- Store location: operator-local SQLite (default under the user config dir). The database holds account numbers; it must stay out of the repo. `.gitignore` covers `*.csv` imports and `*.db`.

## User Vision
- Read-oriented Fidelity Investments personal brokerage and retirement accounts: balances, holdings, positions, statements/transactions where a documented interface allows.
- Prefer a public or official API. None exists for retail. The documented interface is the owner's own CSV export. Do not scrape fidelity.com and do not store brokerage passwords, MFA codes, cookies, or tokens.
- No trades and no money movement.
- Land the package in this fork at `library/<category>/fidelity/` and open a PR on bossriceshark/printing-press-library. Do not publish to mvanhorn/printing-press-library from this run.
- Auth env vars: none. Do not invent `FIDELITY_TOKEN` / `FIDELITY_PASSWORD`. Document that absence explicitly.

## Product Thesis
- Name: fidelity-pp-cli
- Why it should exist: agents cannot log into fidelity.com, and Fidelity will not give them an API. The owner already has a supported export. A local, read-only ledger turns those files into `positions`, `transactions`, and a portfolio summary without a session, a scraper, or a third-party aggregator key.

## Build Priorities
1. Internal spec with `source: local-sqlite`, auth `none`, read-only resources for accounts, positions, transactions, and a portfolio summary. No HTTP base URL. No mutating endpoints.
2. CSV import commands for positions and history that parse the documented headers, skip disclaimer footers, and upsert into SQLite. This is the sync path, because there is no HTTP API to sync.
3. List/get commands and one compound `portfolio` summary that read the local store. `doctor` explains the no-API constraint and the export steps.
4. Ship MCP only if the press emits the cobra-tree mirror for local-sqlite. Keep it read-only.
5. Tests use the public synthetic positions fixture shape (fake account `Z1234567`) plus a synthetic history fixture. Never a real account export.

## Explicit non-goals
- Browser-sniff of fidelity.com. It would require the owner's live session, which this run does not have and must not request, and automated credentialed access conflicts with Fidelity's own guidance against bot logins.
- WorkplaceXchange, Wealthscape, Fidelity Institutional, and Fidelity Access/Akoya/SnapTrade client implementations.
- Order entry, transfers, ACAT, bill pay, wires, or any money movement.

## Reachability Gate
- Decision: PASS (local-sqlite carve-out)
- Reason: no HTTP API origin; account data is owner-exported CSV
- Public site probe: HTTP 200 for https://www.fidelity.com/ (marketing/login shell, not an account API)
- Evidence: source local-sqlite, auth none, AUTH_SESSION_AVAILABLE=false
