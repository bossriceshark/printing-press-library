# Levanta Absorb Manifest

## Absorbed (match or beat everything that exists)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-------------|--------------------|-------------|
| 1 | List seller brands | Levanta Seller MCP `seller_list_brands` | (generated endpoint) brands list | Cursor pagination, `--agent`, local sync |
| 2 | List active creators | Levanta Seller MCP `seller_list_creators` | (generated endpoint) creators list | Offline after sync |
| 3 | List seller products | Levanta Seller MCP `seller_list_products` | (generated endpoint) products list | Filterable catalog in the CLI |
| 4 | Product performance report | Levanta Seller MCP `seller_product_report` | (generated endpoint) reports performance product | Same metrics, scriptable date window |
| 5 | Creator performance report | Levanta Seller MCP `seller_creator_report` | (generated endpoint) reports performance creator | One-window mirror; delta is `creator roi` |
| 6 | Performance totals | Levanta Seller MCP `seller_get_performance_report_totals` | (behavior in levanta-pp-cli creator roi) | Totals fall out of the local seller rollup instead of a hosted-only tool |
| 7 | Account identity | Levanta Seller MCP `whoami` | (behavior in levanta-pp-cli doctor) | No REST whoami exists; doctor reports whether `LEVANTA_API_KEY` is set |
| 8 | List brand partnerships | affiliate-mcp listProgrammes (guessed `/partners`; official path is `/brands`) | (generated endpoint) brands list | Uses the resolving host `app.levanta.io` |
| 9 | Product catalogue | affiliate-mcp + creator docs `/products` | (generated endpoint) products list | Official OpenAPI field names |
| 10 | Performance reports as transactions | affiliate-mcp listTransactions | (generated endpoint) reports list | Keeps `commissionEarned`, `final`, and embedded link |
| 11 | Click reports | creator docs `/reports/clicks` (affiliate-mcp marks this unsupported) | (generated endpoint) reports clicks | Ships the click log the community adapter skipped |
| 12 | Create and list tracking links | creator docs `/links` | (generated endpoint) links create and links list | ASIN plus source, official bearer auth |
| 13 | Invoice / payout items | creator docs `/invoices/items` | (generated endpoint) invoices items | Status enum `pending`, `late`, `paid` |
| 14 | Deals, CPC, loyalty, storefront links, creator connections, BRB | official OpenAPI | (generated endpoint) matching resources | Full documented surface, not the seven-op adapter |

Hosted seller MCP is OAuth and read-only. It does not expose invoices, link create/list, or click logs. affiliate-mcp is experimental, documents `LEVANTA_API_KEY`, and targets `https://api.levanta.io`, which did not resolve on 2026-10-15.

## Transcendence (only possible with our approach)

| # | Feature | Command | Score | Buildability | How It Works | Evidence | Long Description |
|---|---------|---------|-------|--------------|--------------|----------|------------------|
| 1 | Unpaid payouts | unpaid | 10/10 | hand-code | Reads local invoice items from creator `/invoices/items` and returns rows whose status is pending or late. | Brief workflow 1; invoice list requires year and has no status query; seller MCP has no invoice tools. | Use this command for invoice items that already exist and are pending or late. Do NOT use this command for a brand-month with report commission and no invoice item; use 'payout gap' instead. Do NOT use this command when an invoice amount or finality disagrees with reports; use 'earnings reconcile' instead. |
| 2 | Earnings reconcile | earnings reconcile | 10/10 | hand-code | Joins local invoice items to creator report rows on brand, year, month, and marketplace and flags commission mismatches and paid items that still have non-final rows. | Brief workflow 5; report invoice embed has status but not the commission total. | Use this command when an invoice item exists and its commission or finality disagrees with report rows for the same brand and month. Do NOT use this command to list unpaid items; use 'unpaid' instead. Do NOT use this command when the invoice item is missing; use 'payout gap' instead. |
| 3 | Links that earned | earned | 10/10 | hand-code | Rolls up local creator report rows by embedded link id and returns url, ASIN, source, clicks, and summed commission. | Brief workflow 2 and build priority 4; daily `/reports` does not roll up by link. | Use this command for links whose summed commission in the selected window is greater than zero. Do NOT use this command for links that earned in the prior window and earn nothing now; use 'stalled' instead. |
| 4 | Payout gap | payout gap | 9/10 | hand-code | Anti-joins local creator report rows to invoice items on brand id, year, month, and marketplace. | Brief invoice sync key is separate from report rows; seller MCP cannot see invoices. | Use this command when summed report commission exists for a brand, year, month, and marketplace and no invoice item does. Do NOT use this command when the invoice item exists and is unpaid; use 'unpaid' instead. Do NOT use this command when the item exists and the amounts disagree; use 'earnings reconcile' instead. |
| 5 | Links that stopped earning | stalled | 9/10 | hand-code | Compares per-link summed commission on local creator report rows for the current window versus the previous equal-length window and returns active links that went quiet. | Brief product thesis; schema says inactive links do not earn. | Use this command for active links that earned commission in the prior equal-length window and earn nothing in the current window. Do NOT use this command to list links that are still earning; use 'earned' instead. |
| 6 | Creator ROI | creator roi | 9/10 | hand-code | Joins local seller report rows to creators and ranks each creator by commission-to-sales and by the change versus the previous window. | Brief workflow 4; seller MCP creator report is a single window. | none |

Hand-code count: 6 (`unpaid`, `earnings reconcile`, `earned`, `payout gap`, `stalled`, `creator roi`). Spec-emits among novel rows: 0. No stubs.

## User vision coverage

The invocation asked for payouts, affiliate/creator earnings, and links. `unpaid`, `payout gap`, and `earnings reconcile` cover payouts. `earned`, `stalled`, and the generated reports cover earnings. Generated link commands plus `earned` cover links. `creator roi` covers the Ozark seller roster question.
