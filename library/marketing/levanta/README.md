# Levanta CLI

**The payout ledger for Levanta creator commerce: unpaid invoices, link earnings, and seller creator ROI from a local store.**

Wraps the official Levanta creator and seller APIs, then answers the questions that today show up as email. unpaid lists pending and late invoice items. earned and stalled roll link commission across windows. earnings reconcile and payout gap compare invoices with report rows. creator roi ranks a seller's creators across two windows.

## Install

The recommended path installs both the `levanta-pp-cli` binary and the `pp-levanta` agent skill (Claude Code, Codex, Cursor, Gemini CLI, GitHub Copilot, and other agents supported by the upstream [`skills`](https://github.com/vercel-labs/skills) CLI) in one shot:

```bash
npx -y @mvanhorn/printing-press-library install levanta
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press-library install levanta --cli-only
```

For skill only — installs the skill into the same agents as the default command above, but skips the CLI binary (use this to update or reinstall just the skill):

```bash
npx -y @mvanhorn/printing-press-library install levanta --skill-only
```

To constrain the skill install to one or more specific agents (repeatable — agent names match the [`skills`](https://github.com/vercel-labs/skills) CLI):

```bash
npx -y @mvanhorn/printing-press-library install levanta --agent claude-code
npx -y @mvanhorn/printing-press-library install levanta --agent claude-code --agent codex
```

### Without Node (Go fallback)

If `npx` isn't available (no Node, offline), install the CLI directly via Go (requires Go 1.26.9 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/marketing/levanta/cmd/levanta-pp-cli@latest
```

This installs the CLI only — no skill.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/levanta-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

Install the CLI binary first. The installer writes binaries to a per-user managed bin directory by default: `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows.

```bash
npx -y @mvanhorn/printing-press-library install levanta --cli-only
```

Then install the focused Hermes skill.

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-levanta --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-levanta --force
```

Restart the Hermes session or gateway if the newly installed skill is not visible immediately.

## Install for OpenClaw
Install both the CLI binary and the focused OpenClaw skill. The installer defaults binaries to a per-user bin directory (`$HOME/.local/bin` on macOS/Linux, `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows):

```bash
npx -y @mvanhorn/printing-press-library install levanta --agent openclaw
```

Restart the OpenClaw session or gateway if the newly installed skill is not visible immediately.

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/levanta-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.
3. Fill in `LEVANTA_API_KEY` when Claude Desktop prompts you.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


```bash
go install github.com/mvanhorn/printing-press-library/library/marketing/levanta/cmd/levanta-pp-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "levanta": {
      "command": "levanta-pp-mcp",
      "env": {
        "LEVANTA_API_KEY": "<your-key>"
      }
    }
  }
}
```

</details>

## Authentication

Create a key under Settings > API on app.levanta.io and export it as LEVANTA_API_KEY. The CLI sends Authorization: Bearer <api_key>. A creator key reads invoices, links, and creator reports. A seller key reads brands, creators, and seller reports. The two keys are not interchangeable. Do not paste the key into chat or commit it.

## Quick Start

```bash
# Confirm the binary and auth wiring before any network call.
levanta-pp-cli doctor --dry-run

# Read pending and late invoice items already stored locally.
levanta-pp-cli unpaid --agent

# See which stored links earned commission.
levanta-pp-cli earned --agent

# Rank stored seller creators by commission relative to sales.
levanta-pp-cli creator roi --agent

```

## Unique Features

### Payout ledger
- **`unpaid`** — List synced invoice items that are still pending or late.

  _Use this when the question is which Levanta payouts are still unpaid, instead of paging every invoice year._

  ```bash
  levanta-pp-cli unpaid --agent --select status,brand,commission,year,month
  ```
- **`earnings reconcile`** — Flag invoice items whose commission or paid status disagrees with stored report rows.

  _Use this when a payout email says paid but the report rows for that brand-month are still moving._

  ```bash
  levanta-pp-cli earnings reconcile --agent --select brand,status,invoice_commission,report_commission
  ```
- **`payout gap`** — Show brand-months where reports earned commission and no invoice item exists.

  _Use this when earnings are already on the reports and the invoice ledger has no matching line._

  ```bash
  levanta-pp-cli payout gap --agent --select brand,year,month,marketplace,commission
  ```

### Link earnings
- **`earned`** — Roll creator report rows up to one line per link that earned commission.

  _Use this when the question is which affiliate links earned commission, not which daily rows exist._

  ```bash
  levanta-pp-cli earned --agent --select url,asin,source,commission,clicks
  ```
- **`stalled`** — List active links that earned commission in the previous window and earn nothing now.

  _Use this when a link used to earn and the current window is quiet._

  ```bash
  levanta-pp-cli stalled --agent --select url,asin,source,prior_commission
  ```

### Seller program
- **`creator roi`** — Rank seller creators by commission-to-sales and by the change versus the previous window.

  _Use this on the seller account when the question is which creators' return moved, not a single performance page._

  ```bash
  levanta-pp-cli creator roi --agent --select creator,commission,sales,roi,delta
  ```

## Recipes

### See unpaid payouts

```bash
levanta-pp-cli unpaid --agent --select status,brand,commission,year,month
```

Returns pending and late invoice items without paging every invoice year by hand.

### See which links earned

```bash
levanta-pp-cli earned --agent --select url,asin,source,commission,clicks
```

Rolls stored daily report rows up to one line per earning link.

### Reconcile a paid invoice against reports

```bash
levanta-pp-cli earnings reconcile --agent --select brand,status,invoice_commission,report_commission
```

Flags invoice totals that disagree with stored report commission.

### Find links that stopped earning

```bash
levanta-pp-cli stalled --agent --select url,asin,source,prior_commission
```

Compares the current window with the previous one for active links that went quiet.

## Usage

Run `levanta-pp-cli --help` for the full command reference and flag list.

## Paths & environment variables

This CLI separates local files into four path kinds:

| Kind | Contents |
|------|----------|
| `config` | User-editable settings such as `config.toml` and saved profiles |
| `data` | Durable local data: `credentials.toml`, `data.db`, cookies, browser-session proof files, and other auth sidecars |
| `state` | Runtime state such as persisted queries, jobs, and `teach.log` |
| `cache` | Regenerable HTTP/cache files |

Each kind resolves independently. The ladder is:

1. Per-kind env var: `LEVANTA_CONFIG_DIR`, `LEVANTA_DATA_DIR`, `LEVANTA_STATE_DIR`, or `LEVANTA_CACHE_DIR`
2. `--home <dir>` for this invocation
3. `LEVANTA_HOME` for a flat relocated root
4. XDG env vars: `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`
5. Platform defaults matching existing installs

For containers and agent sandboxes, prefer a single relocated root:

```bash
export LEVANTA_HOME=/srv/levanta
levanta-pp-cli doctor
```

Under `LEVANTA_HOME=/srv/levanta`, the four dirs resolve to `/srv/levanta/config`, `/srv/levanta/data`, `/srv/levanta/state`, and `/srv/levanta/cache`.

MCP servers do not receive CLI flags from the host. Put relocation in the host `env` block:

```json
{
  "mcpServers": {
    "levanta": {
      "command": "levanta-pp-mcp",
      "env": {
        "LEVANTA_HOME": "/srv/levanta"
      }
    }
  }
}
```

Precedence matters in fleets: an ambient per-kind variable such as `LEVANTA_DATA_DIR` overrides an explicit `--home` for that kind. Use `LEVANTA_HOME` or the per-kind variables for durable fleet relocation; treat `--home` as the weaker per-invocation lever.

Relocation is one-way. Unsetting `LEVANTA_HOME` does not move files back to platform defaults, and `doctor` cannot find credentials left under a former root. Move the files manually before unsetting relocation variables.

Existing installs keep working because the platform-default rung matches the legacy layout. On the first auth write, stored secrets leave `config.toml` and are consolidated into `credentials.toml` under the data directory. Run `levanta-pp-cli doctor --fail-on warn` to check path and credential-location warnings in automation.

## Commands

### brands

Manage brands

- **`levanta-pp-cli brands list`** - List Brands
- **`levanta-pp-cli brands list-storefronts`** - List all storefronts for a given brand

### cpc

Manage cpc

- **`levanta-pp-cli cpc get-campaign`** - Get a Cost-Per-Click campaign by its ID
- **`levanta-pp-cli cpc list-campaigns`** - List all CPC campaigns

### creator-brands

Manage creator brands

- **`levanta-pp-cli creator-brands`** - List all brands in Levanta's marketplace

### creator-connections

Manage creator connections

- **`levanta-pp-cli creator-connections creator-list-campaigns`** - List your creator connections campaigns that you are a member of.
- **`levanta-pp-cli creator-connections creator-list-reports`** - List your creator connection reports. Reports are unique per campaign per date per asin.
- **`levanta-pp-cli creator-connections get`** - List Loyalty Programs for a Creator Connections Campaign.
     Lists any loyalty program that was active for the campaign at some point
- **`levanta-pp-cli creator-connections get-creatorconnections`** - List Products, Links, and Loyalty Programs for a creator connections campaign

### creator-deals

Manage creator deals

- **`levanta-pp-cli creator-deals`** - List Deals

### creator-links

Manage creator links

- **`levanta-pp-cli creator-links creator-create-product-v2`** - Create Product Link
- **`levanta-pp-cli creator-links creator-get-status-v2`** - Get Product Links Status

### creator-products

Manage creator products

- **`levanta-pp-cli creator-products`** - List Products

### creator-reports

Manage creator reports

- **`levanta-pp-cli creator-reports creator-list-conversions-updates`** - List conversion updates within a date range [start, end]. Start date must be before end date and the date range must not be greater than 60 days
- **`levanta-pp-cli creator-reports creator-list-conversions-v2`** - List conversions within a date range [start, end]. Start date must be before end date and the date range must not be greater than 60 days

### creators

Manage creators

- **`levanta-pp-cli creators get`** - Get Creator by ID
- **`levanta-pp-cli creators list`** - List creators that you have an active partnership with. Filter by brand IDs.

### deals

Manage deals

- **`levanta-pp-cli deals`** - List Deals

### invoices

Manage invoices

- **`levanta-pp-cli invoices`** - Get a list of invoice items for a given month and year

### links

Manage links

- **`levanta-pp-cli links create-cpc-campaign`** - Create a link for a CPC campaign
- **`levanta-pp-cli links create-default-storefront`** - Create a link for the default storefront of a brand
- **`levanta-pp-cli links create-product`** - Create a product link or retrieve an existing one via a unique asin/country/source_id combination. If source_id is not provided the value "default" will be used. If country is not provided the value "US" will be used.
- **`levanta-pp-cli links create-storefront-by-id`** - Get a storefront link by its ID
- **`levanta-pp-cli links get-product`** - Get a product link by its ID
- **`levanta-pp-cli links list-cpc-campaign`** - List all CPC links
- **`levanta-pp-cli links list-product`** - List all product links. Filter by asins, links, and sources.
- **`levanta-pp-cli links list-storefront`** - List all storefront links. Filter by brand_ids, link_ids, source_ids, source_sub_ids, and storefront_ids

### loyalty-programs

Manage loyalty programs

- **`levanta-pp-cli loyalty-programs`** - List loyalty programs

### private

Manage private

- **`levanta-pp-cli private`** - Find a match for a product from any marketplace to Walmart

### products

Manage products

- **`levanta-pp-cli products get-by-asin`** - Get a product in amazon.com by asin
- **`levanta-pp-cli products get-by-asin-marketplace-domain`** - Get a product by its unique marketplace/asin pair
- **`levanta-pp-cli products list`** - List active products in Levanta's product catalog. Sort products by title, commission, or price.
                Filter products by asin, brand, access, and in stock status. Access represents whether or not a creator has an active partnership
                with the brand that the product belongs to.

### reports

Manage reports

- **`levanta-pp-cli reports list`** - List per-product reports at the product/date/link granularity within a date range [start, end].
    Filter by asins, links, or source. The difference in time between the start date and end date can be no longer than 1 year.
    Dates must be specified in ISO format with no offset (UTC), format: "yyyy-MM-ddTHH:mm:ss.SSSZ".
- **`levanta-pp-cli reports list-click`** - Get click reports at the date/link granularity within a date range [start, end]. Filter by link or source.
    The difference in time between the start date and end date can be no longer than 1 year. Dates must be specified in ISO format
    with no offset (UTC), format: "yyyy-MM-ddTHH:mm:ss.SSSZ".
- **`levanta-pp-cli reports list-cpc-campaign`** - List all CPC reports

### seller-brands

Manage seller brands

- **`levanta-pp-cli seller-brands get`** - Get Brand by ID
- **`levanta-pp-cli seller-brands list`** - List Brands

### seller-brands-2

Manage seller brands 2

- **`levanta-pp-cli seller-brands-2 seller-get-v2`** - Get Brand by ID
- **`levanta-pp-cli seller-brands-2 seller-list-v2`** - List Brands

### seller-creator-connections

Manage seller creator connections

- **`levanta-pp-cli seller-creator-connections seller-list-campaigns`** - List Your Creator Connections Campaigns
- **`levanta-pp-cli seller-creator-connections seller-list-creators`** - List creators that appear in your Creator Connections reports. This includes both claimed (active on Levanta) and unclaimed (not yet on Levanta) creators
- **`levanta-pp-cli seller-creator-connections seller-list-reports`** - List your creator connection reports. Reports are unique per campaign, creator.creatorConnectionsId, date, and asin
- **`levanta-pp-cli seller-creator-connections seller-lookup-creator-by-ids`** - Look up Creator Connections creators by their Amazon associates tags (e.g. levanta-20). Returns Levanta creator team IDs and metadata for the matched associates tags.

### seller-creators

Manage seller creators

- **`levanta-pp-cli seller-creators seller-get-v2`** - Get Creator by ID
- **`levanta-pp-cli seller-creators seller-list-v2`** - List creators that you have an active partnership with. Filter by brand IDs.

### seller-products

Manage seller products

- **`levanta-pp-cli seller-products`** - List your products. Filter by brand IDs and ASINs.

### seller-products-2

Manage seller products 2

- **`levanta-pp-cli seller-products-2`** - List your products across brands. Filter by brand IDs and product IDs.

### seller-reports

Manage seller reports

- **`levanta-pp-cli seller-reports list`** - List per-product reports at the product/date/link granularity with a date range [start, end].
                The difference in time between start and end date must be less than 1 year.
                Dates must be specified in ISO format with no offset (UTC), format: "yyyy-MM-ddTHH:mm:ss.SSSZ".
- **`levanta-pp-cli seller-reports list-brbreports`** - List brand referral bonus estimates for a given date range [start, end], bucketed by date.
                The difference in time between start and end date must be less than or equal to 90 days
                Dates must be specified in ISO format with no offset (UTC), format: "yyyy-MM-ddTHH:mm:ss.SSSZ".
- **`levanta-pp-cli seller-reports list-click`** - List click reports at the date/link granularity with a date range [start, end].
                The difference in time between start and end date must be less than 1 year.
                Dates must be specified in ISO format with no offset (UTC), format: "yyyy-MM-ddTHH:mm:ss.SSSZ".

### seller-reports-2

Manage seller reports 2

- **`levanta-pp-cli seller-reports-2 seller-list-conversions-v2`** - Conversion-level sales data for Walmart, Shopify, and Wayfair. Date range must not exceed 60 days.
- **`levanta-pp-cli seller-reports-2 seller-list-performance-by-brand-v2`** - Aggregated performance metrics across all marketplaces, grouped by brand.
- **`levanta-pp-cli seller-reports-2 seller-list-performance-by-creator-v2`** - Aggregated performance metrics across all marketplaces, grouped by creator.
- **`levanta-pp-cli seller-reports-2 seller-list-performance-by-date-v2`** - Aggregated performance metrics across all marketplaces, grouped by date.
- **`levanta-pp-cli seller-reports-2 seller-list-performance-by-product-v2`** - Aggregated performance metrics across all marketplaces, grouped by product.


### creator


- **`levanta-pp-cli creator roi`** - Rank seller creators by commission-to-sales and by the change versus the previous window.


### earned


- **`levanta-pp-cli earned`** - Roll creator report rows up to one line per link that earned commission.


### earnings


- **`levanta-pp-cli earnings reconcile`** - Flag invoice items whose commission or paid status disagrees with stored report rows.


### payout


- **`levanta-pp-cli payout gap`** - Show brand-months where reports earned commission and no invoice item exists.


### stalled


- **`levanta-pp-cli stalled`** - List active links that earned commission in the previous window and earn nothing now.


### unpaid


- **`levanta-pp-cli unpaid`** - List synced invoice items that are still pending or late.




### Self-learning loop

This CLI caches per-question discovery so repeat queries skip the walk and structurally similar queries get answered via entity substitution. The loop also self-captures: every invocation is journaled locally, and failed-flag corrections plus fresh teaches surface as candidates on the next `recall` for confirm/reject judgment. Agents call `recall` before discovery and fire `teach &` after answering. See the `## Automatic learning` section in `SKILL.md` for the full protocol.

- **`levanta-pp-cli recall <query>`** - Look up cached resources for a query before running discovery
- **`levanta-pp-cli teach`** - Record a query -> resource mapping (silent on success, safe to background with `&`)
- **`levanta-pp-cli learnings list`** - Inspect taught rows
- **`levanta-pp-cli learnings forget <query>`** - Undo a teach
- **`levanta-pp-cli learnings candidates`** - List auto-captured candidates awaiting confirm/reject
- **`levanta-pp-cli learnings stats`** - Local loop metrics: recall hit rate, teach-to-reuse, playbook resolution, candidate counts
- **`levanta-pp-cli teach-pattern`** - Install a query/resource template up front
- **`levanta-pp-cli teach-lookup`** - Add an entity mapping (e.g. country code, team alias) for pattern substitution

Pass `--no-learn` or set `LEVANTA_NO_LEARN=true` to disable the loop for deterministic flows.

The local store's schema version stamp is one-way: once this version of `levanta-pp-cli` opens the database, older binaries refuse it with a version error — upgrade the binary rather than downgrading.

## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
levanta-pp-cli brands list

# JSON for scripting and agents
levanta-pp-cli brands list --json
# Filter to specific fields
levanta-pp-cli brands list --json --select access,bio,brandId

# Dry run — show the request without sending
levanta-pp-cli brands list --dry-run

# Agent mode — JSON + compact + no prompts in one flag
levanta-pp-cli brands list --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select <field>[,<field>...]` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Explicit retries** - add `--idempotent` to create retries when a no-op success is acceptable
- **Explicit confirmation** - `--agent` does not imply `--yes`; pass `--yes` separately only after the target, arguments, and side effects are clear
- **Piped input** - write commands can accept structured input when their help lists `--stdin`
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `6` partial failure, `7` rate limited, `10` config error.

## Health Check

```bash
levanta-pp-cli doctor
```

Verifies configuration, credentials, and connectivity to the API.

## Configuration

Run `levanta-pp-cli doctor` to see the resolved config, data, state, and cache directories. The platform-default config path is `~/.config/levanta-pp-cli/config.toml`; `--home`, `LEVANTA_HOME`, and per-kind env vars can relocate it.

Static request headers can be configured under `headers`; per-command header overrides take precedence.

Environment variables:

| Name | Kind | Required | Description |
| --- | --- | --- | --- |
| `LEVANTA_API_KEY` | per_call | Yes | Set to your API credential. |

### agentcookie (optional)

If you use agentcookie to sync secrets across machines, this CLI auto-adopts agentcookie-managed credentials with no extra setup. When the daemon writes to this CLI's config, `levanta-pp-cli doctor` reports `agentcookie: detected` and `auth-status` labels the source as `agentcookie`. Skip this section if you don't use agentcookie - the CLI works the same as any other.

## Troubleshooting
**Authentication errors (exit code 4)**
- Run `levanta-pp-cli doctor` to check credentials
- Verify the environment variable is set: `echo $LEVANTA_API_KEY`
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific
- **HTTP 401 saying an Authorization bearer token is required** — Export LEVANTA_API_KEY with the bearer token from Settings > API. Use a creator key for invoice and link commands and a seller key for creator roster and seller reports.
- **A report command rejects the date window** — Pass start and end as UTC timestamps YYYY-MM-DDTHH:mm:ss.SSSZ and keep creator or seller v1 report spans at or under one year. Creator v2 and seller v2 conversion spans must stay at or under 60 days.
- **Invoice month looks wrong** — The invoice month query is 0 through 11, not 1 through 12. Omit month to list the whole year.
