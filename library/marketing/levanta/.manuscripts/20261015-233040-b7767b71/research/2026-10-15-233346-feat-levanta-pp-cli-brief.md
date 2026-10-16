# Levanta CLI Brief

## API Identity
- Domain: Amazon (plus Walmart and Shopify on v2) affiliate / creator-commerce payouts. Brands recruit creators; creators earn commission on attributed sales and get paid on an invoice cycle. Ozark Armament uses Levanta as the brand/seller side of that program. The Gmail trail agents currently read is payout status, creator earnings, and tracking links.
- Users: seller teams (brands) and creator teams (affiliates, publishers). One API key is minted per team under Settings > API. A seller key and a creator key are not interchangeable.
- Data profile: cursor-paginated JSON. Creator v1 is the payout-and-link ledger (invoice items, product links, per-link reports). Seller v1/v2 is the brand ledger (active creators, commission reports, performance rollups). Creator v2 adds multi-retailer conversions and link-create status. No published rate-limit table.

## Reachability Risk
- Low. Official OpenAPI documents returned HTTP 200 on 2026-10-15:
  - `https://app.levanta.io/api/creator/v1/openapi.json` (25 paths, Creator API 1.0.0)
  - `https://app.levanta.io/api/creator/v2/openapi.json` (8 paths, Creator API v2 2.0.0)
  - `https://app.levanta.io/api/seller/v1/openapi.json` (12 paths, Seller API 1.0.0)
  - `https://app.levanta.io/api/seller/v2/openapi.json` (10 paths, Seller API v2 2.0.0)
- Docs index `https://api-docs.levanta.io/llms.txt` HTTP 200. Auth guide HTTP 200. Pagination guide HTTP 200. Help guide HTTP 200 (no rate-limit section; support is the vendor support contact listed in the help guide).
- No npm, PyPI, or GitHub SDK for `app.levanta.io`. Search hits for "levanta" are unrelated Spanish "levantar" projects. Nothing to mine for 403/deprecated issues.
- Onboarding risk, not a transport risk: the creator knowledge base says API keys are requested in the in-app chatbot and can take up to a week. This run has no `LEVANTA_*` token in the environment. The user forbade pasting secrets, so live smoke is deferred.
- Auth (exact): `Authorization: Bearer {api_key}`. Security scheme name in every spec: `Bearer`, type `http`, scheme `bearer`. Key is created at Settings > API on https://app.levanta.io. Placeholder only: `Authorization: Bearer <api_key>`.
- Pagination (exact): response field `cursor` (`string | null`). Query `cursor` (maxLength 512) and `limit` (1–100, default 25 on the invoice endpoint). `null` cursor means no further page.
- Marketplace enum on creator v1 (exact): `all`, `amazon.com`, `amazon.co.uk`, `amazon.com.mx`, `amazon.ca`, `amazon.de`, `amazon.es`, `amazon.fr`, `amazon.it`, `amazon.nl`. Seller v2 adds `walmart.com` and `shopify.com`.
- Invoice status enum (exact): `pending`, `late`, `paid`.
- Report windows: creator/seller v1 reports and clicks require `start` and `end` as ISO UTC `YYYY-MM-DDTHH:mm:ss.SSSZ`, span at most 1 year. Seller BRB reports span at most 90 days. Creator v2 and seller v2 conversions span at most 60 days. Invoice list requires `year` (2023–2030); `month` is 0–11 and optional.

## Top Workflows
1. Payout check: list invoice items for a year/month and read `status`, `commission`, `cpcCommission`, `sales`, `currency`, and brand name. This replaces the "you got paid / still pending" email.
2. Earnings by link: list reports for a date window filtered by ASIN, link id, or source, and read `commissionEarned`, `sales`, `conversions`, `clicks`, and `final`.
3. Link desk: list product links (url, active, source) and create one for an ASIN (`asin` required; `source_id` defaults to `default`, marketplace defaults to `amazon.com`).
4. Brand desk (Ozark seller account): list active creators, pull seller reports (`commissionAmount`, `linkId`, `creatorId`, `brandId`), and roll performance up by creator, product, brand, or date.
5. Reconcile: join invoice status to report rows that point at the same brand and month, and flag non-final reports that still disagree with a paid invoice.

## Table Stakes
- Levanta's own Seller MCP (`https://app.levanta.io/api/mcp`, OAuth) exposes `whoami`, `seller_list_brands`, `seller_list_creators`, `seller_list_products`, `seller_get_performance_report_totals`, `seller_product_report`, `seller_creator_report`. It does not expose creator invoices, link create/list, click logs, or creator-connections.
- Official docs already document bearer auth, cursor pagination, and the four OpenAPI files. No third-party CLI exists to absorb.
- Knowledge-base creator surface (v1) names these paths: `/brands`, `/brands/storefronts`, `/products`, `/products/{asin}`, `/products/{marketplace}/{asin}`, `/deals`, `/links`, `/links/{link_id}`, `/links/storefronts`, `/reports`, `/reports/clicks`, `/invoices/items`.

## Data Layer
- Primary entities: invoice items (payout status), product links, performance reports, clicks, creators, brands, products, deals, creator-connection campaigns.
- Sync cursor: the API cursor is opaque and query-scoped, not a global replication token. Persist the last `cursor` per command invocation. For reports, the incremental key is the date window plus `final=false` rows that must be re-fetched. Invoice sync key is `(year, month, marketplace, brand.id)`.
- FTS/search: link `url` / `sourceId` / ASIN, creator name, brand name, product title. Earnings queries filter by date, ASIN, brand id, and invoice status more often than free text.

## User Vision
- Print one `levanta-pp-cli` agents can use for the Levanta account ops that today exist only in Ozark Gmail: payouts, affiliate/creator earnings, and links.
- Prefer the public documented API and bearer auth over scraping. Env var names and placeholders only. Do not commit secrets. Do not ask for a pasted key.
- Land the package in this library fork at `library/<category>/levanta/`, on a branch and PR against the fork. Do not open a PR to `mvanhorn/printing-press-library` in this run.

## Product Thesis
- Name: levanta
- Why it should exist: Levanta isn't just an affiliate-link generator. It's the payout ledger for creator commerce. Every invoice status, click, and link is a signal about which partnerships are actually getting paid. Gmail is a lagging copy of that ledger. A CLI that can list payouts, earnings, and links in one command is the thing agents can call instead of reading mail.

## Build Priorities
1. Merge the four official OpenAPI specs into one `levanta` CLI. Creator v1 is first so invoices, links, and reports keep the short resource names. Seller and v2 resources that collide take a spec-name prefix. Same host `https://app.levanta.io`; fold `/api/creator/v1`, `/api/creator/v2`, `/api/seller/v1`, and `/api/seller/v2` into endpoint paths.
2. Bearer auth via an env var (generator default; expected `LEVANTA_TOKEN` or the scheme-derived name). Document that a creator key serves creator routes and a seller key serves seller routes. Do not invent a second scheme.
3. Cursor pagination on list commands. Required query flags for `year` (invoices) and `start`/`end` (reports) must match the spec, including the ISO UTC format.
4. Local store for invoice items, links, and reports so an agent can answer "what is still pending?" and "which links earned commission this month?" without re-pulling every page.
5. Category: `marketing`. Skip live smoke until a key exists in the environment. Structural `go build`, `--help`, and shipcheck still have to pass.

## Spec Selection
- Combined surface, umbrella name `levanta`, because the user asked for one CLI covering payouts (creator invoices), earnings (creator and seller reports), and links (creator link endpoints). Generating only the seller spec would drop invoices and link URLs. Generating only the creator spec would drop the Ozark brand roster and seller commission reports.
- Local copies: `research/creator-v1.openapi.json`, `research/creator-v2.openapi.json`, `research/seller-v1.openapi.json`, `research/seller-v2.openapi.json`.
- Provenance: official.
