# Novel features brainstorm

Subagent `bc-5c1b18c6-5886-531c-8cc1-dbff94bdd804`. First print. Prior research: none.

## Customer model

### Weekend exporter
Downloads `Portfolio_Positions_*.csv` and `History_for_Account_*.csv` after the week and wants "what changed" without a spreadsheet diff.

### Owner-side agent
Answers balances, holdings, and activity from files the owner already saved. Cannot log into fidelity.com.

### Books importer
Today stops at OFX conversion. Wants dividends, interest, commissions, and fees as ledger totals, and a report when one account's history file is missing.

### Multi-account reader
One positions file lists brokerage, IRA, Roth, and CMA. Wants market value, cash versus invested, and a symbol that appears in more than one account, without double-counting statement cash and core money-market positions.

## Candidates (pre-cut)

1. portfolio summary — KEEP
2. activity summary — KEEP
3. changes — KEEP
4. symbols total — KEEP
5. coverage — KEEP
6. balances — CUT, fields belong on portfolio summary
7. freshness — CUT, as-of belongs on portfolio summary
8. allocation — CUT, weekly question is cash versus invested
9. performance — CUT, no price series
10. benchmark — CUT, live quotes forbidden
11. accounts compare — CUT, portfolio summary already has per-account rows
12. contributions — CUT, activity summary already groups actions; name collides with money movement
13. realized — CUT, no closed-lot cost basis
14. transactions since — CUT, changes already lists new history rows
15. location — CUT, account names are owner-defined
16. imports list — CUT, doctor plus coverage cover it

## Survivors and kills

### Survivors

| # | Feature | Command | Score | Buildability | How It Works | Evidence | Long Description |
|---|---------|---------|-------|--------------|--------------|----------|------------------|
| 1 | Portfolio summary | fidelity-pp-cli portfolio summary | 10/10 | hand-code | Latest snapshots plus imports. | Top Workflow 3. | Redirects to symbols total, activity summary, changes, coverage. |
| 2 | Activity summary | fidelity-pp-cli activity summary | 9/10 | hand-code | Sums Amount, Commission, Fees by Action. | Dividends this year. | Redirects to portfolio summary, symbols total, changes. |
| 3 | Changes since previous import | fidelity-pp-cli changes | 8/10 | hand-code | Prior snapshot retained on sha256 change. | Top Workflow 4. | Redirects to portfolio summary, activity summary, symbols total. |
| 4 | Cross-account symbol total | fidelity-pp-cli symbols total | 8/10 | hand-code | Sums one symbol across accounts. | Where is my VTI. | Redirects to portfolio summary, activity summary, changes. |
| 5 | Per-account export coverage | fidelity-pp-cli coverage | 7/10 | hand-code | Positions vs history presence per account. | Separate files. | Redirects to portfolio summary and changes. |

### Killed candidates

| feature | kill reason | closest-surviving-sibling |
|---|---|---|
| balances | Labeled fields on portfolio summary | portfolio summary |
| freshness | As-of is a header on portfolio summary | portfolio summary |
| allocation | Cash versus invested is the weekly question | portfolio summary |
| performance | No price series | portfolio summary |
| benchmark | Live quotes forbidden | portfolio summary |
| accounts compare | Per-account rows already exist | portfolio summary |
| contributions | Action groups plus money-movement name collision | activity summary |
| realized | No closed-lot basis | activity summary |
| transactions since | changes lists new history rows | changes |
| location | Account names are not a taxonomy | symbols total |
| imports list | doctor plus coverage | coverage |
