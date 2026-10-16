---
name: pp-levanta
description: "The payout ledger for Levanta creator commerce: unpaid invoices, link earnings, and seller creator ROI from a local store. Trigger phrases: `which Levanta payouts are still pending`, `which affiliate links earned commission`, `Levanta creator earnings`, `reconcile Levanta invoices`, `use levanta`, `run levanta`."
author: "Matt Rice"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - levanta-pp-cli
    install:
      - kind: go
        bins: [levanta-pp-cli]
        module: github.com/mvanhorn/printing-press-library/library/marketing/levanta/cmd/levanta-pp-cli
---

# Levanta — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `levanta-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer. It defaults binaries to `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows:
   ```bash
   npx -y @mvanhorn/printing-press-library install levanta --cli-only
   ```
2. Verify: `levanta-pp-cli --version`
3. Ensure the reported install directory is on `$PATH` for the agent/runtime that will invoke this skill.

If the `npx` install fails (no Node, offline, etc.), fall back to a direct Go install (requires Go 1.26.9 or newer). This installs into `$GOPATH/bin` (default `$HOME/go/bin`), so add that directory to `$PATH` instead:

```bash
go install github.com/mvanhorn/printing-press-library/library/marketing/levanta/cmd/levanta-pp-cli@latest
```

If `--version` reports "command not found" after install, the runtime cannot see the binary directory on `$PATH`. Do not proceed with skill commands until verification succeeds.

Wraps the official Levanta creator and seller APIs, then answers the questions that today show up as email. unpaid lists pending and late invoice items. earned and stalled roll link commission across windows. earnings reconcile and payout gap compare invoices with report rows. creator roi ranks a seller's creators across two windows.

## When to Use This CLI

Use this CLI when an agent needs Levanta payout status, affiliate or creator earnings, or tracking links without reading mail. Creator commands cover invoice items, product links, and creator reports. Seller commands cover the brand roster and seller commission reports. Local commands unpaid, earned, stalled, payout gap, earnings reconcile, and creator roi answer the weekly ledger questions after a sync.

## Anti-triggers

Do not use this CLI for:
- Do not use this CLI to change bank details, send a payout, or edit a Levanta team.
- Do not use this CLI as a Gmail client. It reads the Levanta API, not the mailbox.
- Do not use a seller key to query creator invoice or link commands, or a creator key to query seller creator roster commands.

## Unique Capabilities

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

## Command Reference

**brands** — Manage brands

- `levanta-pp-cli brands list` — List Brands
- `levanta-pp-cli brands list-storefronts` — List all storefronts for a given brand

**cpc** — Manage cpc

- `levanta-pp-cli cpc get-campaign` — Get a Cost-Per-Click campaign by its ID
- `levanta-pp-cli cpc list-campaigns` — List all CPC campaigns

**creator-brands** — Manage creator brands

- `levanta-pp-cli creator-brands` — List all brands in Levanta's marketplace

**creator-connections** — Manage creator connections

- `levanta-pp-cli creator-connections creator-list-campaigns` — List your creator connections campaigns that you are a member of.
- `levanta-pp-cli creator-connections creator-list-reports` — List your creator connection reports. Reports are unique per campaign per date per asin.
- `levanta-pp-cli creator-connections get` — List Loyalty Programs for a Creator Connections Campaign.
- `levanta-pp-cli creator-connections get-creatorconnections` — List Products, Links, and Loyalty Programs for a creator connections campaign

**creator-deals** — Manage creator deals

- `levanta-pp-cli creator-deals` — List Deals

**creator-links** — Manage creator links

- `levanta-pp-cli creator-links creator-create-product-v2` — Create Product Link
- `levanta-pp-cli creator-links creator-get-status-v2` — Get Product Links Status

**creator-products** — Manage creator products

- `levanta-pp-cli creator-products` — List Products

**creator-reports** — Manage creator reports

- `levanta-pp-cli creator-reports creator-list-conversions-updates` — List conversion updates within a date range [start, end].
- `levanta-pp-cli creator-reports creator-list-conversions-v2` — List conversions within a date range [start, end].

**creators** — Manage creators

- `levanta-pp-cli creators get` — Get Creator by ID
- `levanta-pp-cli creators list` — List creators that you have an active partnership with. Filter by brand IDs.

**deals** — Manage deals

- `levanta-pp-cli deals` — List Deals

**invoices** — Manage invoices

- `levanta-pp-cli invoices` — Get a list of invoice items for a given month and year

**links** — Manage links

- `levanta-pp-cli links create-cpc-campaign` — Create a link for a CPC campaign
- `levanta-pp-cli links create-default-storefront` — Create a link for the default storefront of a brand
- `levanta-pp-cli links create-product` — Create a product link or retrieve an existing one via a unique asin/country/source_id combination.
- `levanta-pp-cli links create-storefront-by-id` — Get a storefront link by its ID
- `levanta-pp-cli links get-product` — Get a product link by its ID
- `levanta-pp-cli links list-cpc-campaign` — List all CPC links
- `levanta-pp-cli links list-product` — List all product links. Filter by asins, links, and sources.
- `levanta-pp-cli links list-storefront` — List all storefront links. Filter by brand_ids, link_ids, source_ids, source_sub_ids, and storefront_ids

**loyalty-programs** — Manage loyalty programs

- `levanta-pp-cli loyalty-programs` — List loyalty programs

**private** — Manage private

- `levanta-pp-cli private` — Find a match for a product from any marketplace to Walmart

**products** — Manage products

- `levanta-pp-cli products get-by-asin` — Get a product in amazon.com by asin
- `levanta-pp-cli products get-by-asin-marketplace-domain` — Get a product by its unique marketplace/asin pair
- `levanta-pp-cli products list` — List active products in Levanta's product catalog. Sort products by title, commission, or price.

**reports** — Manage reports

- `levanta-pp-cli reports list` — List per-product reports at the product/date/link granularity within a date range [start, end].
- `levanta-pp-cli reports list-click` — Get click reports at the date/link granularity within a date range [start, end]. Filter by link or source.
- `levanta-pp-cli reports list-cpc-campaign` — List all CPC reports

**seller-brands** — Manage seller brands

- `levanta-pp-cli seller-brands get` — Get Brand by ID
- `levanta-pp-cli seller-brands list` — List Brands

**seller-brands-2** — Manage seller brands 2

- `levanta-pp-cli seller-brands-2 seller-get-v2` — Get Brand by ID
- `levanta-pp-cli seller-brands-2 seller-list-v2` — List Brands

**seller-creator-connections** — Manage seller creator connections

- `levanta-pp-cli seller-creator-connections seller-list-campaigns` — List Your Creator Connections Campaigns
- `levanta-pp-cli seller-creator-connections seller-list-creators` — List creators that appear in your Creator Connections reports.
- `levanta-pp-cli seller-creator-connections seller-list-reports` — List your creator connection reports. Reports are unique per campaign, creator.creatorConnectionsId, date, and asin
- `levanta-pp-cli seller-creator-connections seller-lookup-creator-by-ids` — Look up Creator Connections creators by their Amazon associates tags (e.g. levanta-20).

**seller-creators** — Manage seller creators

- `levanta-pp-cli seller-creators seller-get-v2` — Get Creator by ID
- `levanta-pp-cli seller-creators seller-list-v2` — List creators that you have an active partnership with. Filter by brand IDs.

**seller-products** — Manage seller products

- `levanta-pp-cli seller-products` — List your products. Filter by brand IDs and ASINs.

**seller-products-2** — Manage seller products 2

- `levanta-pp-cli seller-products-2` — List your products across brands. Filter by brand IDs and product IDs.

**seller-reports** — Manage seller reports

- `levanta-pp-cli seller-reports list` — List per-product reports at the product/date/link granularity with a date range [start, end].
- `levanta-pp-cli seller-reports list-brbreports` — List brand referral bonus estimates for a given date range [start, end], bucketed by date.
- `levanta-pp-cli seller-reports list-click` — List click reports at the date/link granularity with a date range [start, end].

**seller-reports-2** — Manage seller reports 2

- `levanta-pp-cli seller-reports-2 seller-list-conversions-v2` — Conversion-level sales data for Walmart, Shopify, and Wayfair. Date range must not exceed 60 days.
- `levanta-pp-cli seller-reports-2 seller-list-performance-by-brand-v2` — Aggregated performance metrics across all marketplaces, grouped by brand.
- `levanta-pp-cli seller-reports-2 seller-list-performance-by-creator-v2` — Aggregated performance metrics across all marketplaces, grouped by creator.
- `levanta-pp-cli seller-reports-2 seller-list-performance-by-date-v2` — Aggregated performance metrics across all marketplaces, grouped by date.
- `levanta-pp-cli seller-reports-2 seller-list-performance-by-product-v2` — Aggregated performance metrics across all marketplaces, grouped by product.


**creator**


- `levanta-pp-cli creator roi` — Rank seller creators by commission-to-sales and by the change versus the previous window.


**earned**


- `levanta-pp-cli earned` — Roll creator report rows up to one line per link that earned commission.


**earnings**


- `levanta-pp-cli earnings reconcile` — Flag invoice items whose commission or paid status disagrees with stored report rows.


**payout**


- `levanta-pp-cli payout gap` — Show brand-months where reports earned commission and no invoice item exists.


**stalled**


- `levanta-pp-cli stalled` — List active links that earned commission in the previous window and earn nothing now.


**unpaid**


- `levanta-pp-cli unpaid` — List synced invoice items that are still pending or late.




### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
levanta-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query. `--json` (and other machine formats) keep that exit-2 contract and write `{"matches":[]}` on stdout so agents can inspect the envelope without treating a miss as success.

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

## Auth Setup

Create a key under Settings > API on app.levanta.io and export it as LEVANTA_API_KEY. The CLI sends Authorization: Bearer <api_key>. A creator key reads invoices, links, and creator reports. A seller key reads brands, creators, and seller reports. The two keys are not interchangeable. Do not paste the key into chat or commit it.

Run `levanta-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color`.

Global format flags share one contract on promoted, novel, sync, and `--deliver` paths:

- `--json` — one JSON document on stdout (sync progress events go to stderr)
- `--compact` — keep identity/status/timestamp fields; does not change the document vs stream shape
- `--csv` / `--plain` — tabular rows (collection envelopes unwrap to the row array)
- `--quiet` — one identity value per row, no envelope

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  levanta-pp-cli brands list --agent --select access,bio,brandId
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag
- **Explicit confirmation** — `--agent` does not imply `--yes`; pass `--yes` separately only after the target, arguments, and side effects are clear
- **Explicit retries** — use `--idempotent` only when an already-existing create should count as success

### Response envelope

Commands that read from the local store or the API wrap output in a provenance envelope:

```json
{
  "meta": {"source": "live" | "local", "synced_at": "...", "reason": "..."},
  "results": <data>
}
```

Parse `.results` for data and `.meta.source` to know whether it's live or local. A human-readable `N results (live)` summary is printed to stderr only when stdout is a terminal AND no machine-format flag (`--json`, `--csv`, `--compact`, `--quiet`, `--plain`, `--select`) is set — piped/agent consumers and explicit-format runs get pure JSON on stdout.

## Paths and state

Agents should treat the CLI's path resolver as part of the runtime contract:

- Use `--home <dir>` for one invocation, or set `LEVANTA_HOME=<dir>` to relocate all four path kinds under one root.
- Use per-kind env vars only when a specific kind must diverge: `LEVANTA_CONFIG_DIR`, `LEVANTA_DATA_DIR`, `LEVANTA_STATE_DIR`, `LEVANTA_CACHE_DIR`.
- Resolution order is per-kind env var, `--home`, `LEVANTA_HOME`, XDG (`XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`), then platform defaults.
- `config` contains settings like `config.toml` and profiles. `data` contains `credentials.toml`, `data.db`, cookies, and auth sidecars. `state` contains persisted queries, jobs, and `teach.log`. `cache` contains regenerable HTTP/cache files.
- Stored secrets live in `credentials.toml` under the data dir. Existing legacy `config.toml` secrets are read for compatibility and leave `config.toml` on the first auth write.
- Run `levanta-pp-cli doctor --fail-on warn` to surface path and credential-location warnings. `agent-context` exposes a schema v4 `paths` block for agents that need the resolved dirs.
- For MCP, pass relocation through the MCP host config. The MCP binary does not inherit CLI flags:

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

Fleet precedence: an inherited per-kind env var overrides an explicit `--home` for that kind. Use `LEVANTA_HOME` or per-kind vars as durable fleet levers, and use `--home` only for a single invocation. Relocation is not reversible by unsetting env vars; move files manually before clearing `LEVANTA_HOME`, or `doctor` will not find credentials left under the former root.

## Automatic learning

This CLI ships a self-capturing learning loop. The CLI does its own bookkeeping: every invocation is journaled locally, a failed flag followed by a corrected retry auto-derives a `flag_alias` candidate, and a `teach` on a query family without a playbook auto-synthesizes a `playbook_candidate` from the session's journal. Your job is judgment only: `recall` first, act on surfaced candidates, `teach` the final answer, `playbook amend` when you observe a correction. You never record failures by hand.

### Step 1: `recall` before any discovery

Before list/search/drill commands on a new user question, pass the question as an argv or MCP tool argument to `recall --agent`. Do not interpolate user-controlled text into a shell command line.

Quoted `recall "<question>"` breaks on an apostrophe, which is ordinary English. A quoted heredoc breaks when a body line equals the delimiter, and that delimiter is published in these docs. Write the question with a non-shell file-writing tool, then read it back as data:

```bash
# Write the question verbatim with your file-writing tool (no shell involved).
# Command substitution on a file only ever yields data — the shell never
# parses the file's bytes as syntax.
QUERY=$(cat /path/to/question.txt)
levanta-pp-cli recall "$QUERY" --agent
```

Prefer MCP: pass the question as the tool's query argument. `"$QUERY"` after a file read is argv-safe; putting the question itself in the command text is not.

The response envelope:

```json
{
  "query": "...",
  "normalized": "<normalized form>",
  "query_entities": ["..."],
  "found": true | false,
  "match_score": 0.0,
  "results": [
    { "resource_id": "...", "resource_type": "...", "venue": "...",
      "confidence": 2, "entity_match": "exact|partial|unknown",
      "source": "taught|preseed|pattern", "warnings": ["..."] }
  ],
  "mismatches": [ /* only when --debug-mismatches */ ],
  "warnings": [ /* top-level */ ],
  "candidates": [
    { "id": 12, "class": "flag_alias | playbook_candidate",
      "summary": "...", "sightings": 3, "last_seen": "...",
      "rationale": "...",
      "next_action": ["<trial command>", "levanta-pp-cli learnings confirm 12"] }
  ],
  "playbook": {
    "query_family": "...",
    "playbook": {
      "steps": [ { "cmd": "<command with {slot} substitution>", "purpose": "..." } ],
      "entity_slots": ["$ENTITY"],
      "expected_tool_calls": 3
    },
    "slots_resolved": { "$ENTITY": { "token": "<live token>", "canonical": "<canonical>" } },
    "notes": "<workarounds + gotchas for this query family>"
  },
  "notes": "<duplicate surface for non-playbook callers>"
}
```

Empty-store short-circuit: if the store has no learnings, playbooks, or candidates yet (recall finds nothing and `learnings list` and `learnings candidates` are both empty), skip recall for the rest of this session instead of taxing every query; resume recall-first once something has been taught.

### Step 2: decision tree

Read `candidates`, `playbook`, `notes`, `results[0]`, and warnings in that order:

```
if Candidates present (warnings include "candidates_present"):
    -> candidates are try-then-confirm, never facts. Follow each candidate's
       two-step next_action verbatim: run the trial command first, then run
       `learnings confirm <id>` only after the trial verified the behavior.
       Reject a wrong candidate with `learnings reject <id>`.
    -> NEVER re-teach something recall surfaced as a candidate; confirm or
       reject that candidate instead of teaching a duplicate.
    -> candidates ride alongside playbooks and resource hits, not instead of
       them; continue with the branches below after acting on them.

if Playbook present:
    -> READ Playbook.notes verbatim FIRST (workarounds + gotchas the CLI surface doesn't expose)
    -> replay Playbook.steps in order, substituting Playbook.slots_resolved entries
       for the entity slot tokens. If a step's slot is unresolved, fall back to
       discovery for that step only.
    -> the Playbook's expected_tool_calls is a budget; if you find yourself running
       materially more, record the divergence via `levanta-pp-cli playbook amend`
       at end-of-session.

elif Notes present (no Playbook):
    -> read Notes verbatim before any discovery step; they carry known gotchas
       for this query family even when no structured choreography exists yet.

elif Found AND Results[0].EntityMatch == "exact" AND Results[0].Confidence >= 2:
    -> skip discovery; fetch live data for Results[*].ResourceID in parallel

elif Found AND Results[0].EntityMatch == "partial":
    -> candidate hint, NOT a hit; read the resource title to validate before trusting

elif (any row in Mismatches[] when --debug-mismatches was passed):
    -> treat as cold start; the stored learning is for a different entity
       (different canonical resolved from query_entities)

else:  // Found == false, no playbook, no notes
    -> cold start; run discovery normally; teach the answer afterward (Step 4).
       If the family has no playbook yet, that teach auto-synthesizes a
       playbook candidate from this session's journal - you do not need to
       record one by hand.
```

Playbook and Notes are orthogonal to the per-resource path. A recall response can carry both a Playbook AND a `Results[]` hit - use both: the Playbook tells you which choreography to run; the resource hits short-circuit specific steps. Default to skipping `mismatches`; pass `--debug-mismatches` only when investigating cold-start surprises.

Candidate judgment details: `learnings confirm <id>` prints the candidate's full payload before materializing it - check that the printed payload matches the behavior you verified. `learnings reject <id>` tombstones the derivation signature so the same candidate does not resurface. The envelope carries only the few candidates worth acting on now; `levanta-pp-cli learnings candidates` lists the full open set.

Graceful degradation: if `learnings confirm` is an unknown command, you are driving an older binary - ignore the candidates guidance and follow the rest of the protocol.

### Step 3: always read `warnings`

- `low_confidence`: row exists at `confidence<2`. Treat as a hint, not a skip-discovery hit.
- `resource_not_in_store`: the local store doesn't have the resource the learning points at. The match validator couldn't classify entities — direct-fetch and re-evaluate.
- `cross_alias_match` (per-result): the row was taught under a different alias and matched the live query's canonical via `entity_lookups` (e.g., a "USA" teach satisfying a "United States" recall). Trust the resource_id.
- `similar_shape_different_entity:<canonical>` (top-level): a structurally matching row exists but its canonical entity differs from the live query's. Treated as cold start; the warning carries the conflicting canonical as a hint, but the row is NOT promoted into Results.
- `ambiguous_alias` (top-level): a single query entity resolved to multiple canonicals (e.g., "Cards" → Arizona Cardinals + St. Louis Cardinals). Surface the ambiguity from context before committing to a resource.
- `candidates_present` (top-level): the envelope carries a `candidates` section. Handle it via the candidates branch in Step 2 before anything else.
- `lookup_refresh_available` (top-level): an entity in the query has no lookup row yet, but synced data could provide one. Run `levanta-pp-cli sync` to refresh entity lookups.
- Top-level `no_learnings_for_query_family`: the table had no rows above the Jaccard floor. Pure cold start.

### Step 4: `teach &` after finalizing your response - always

Teaching is unconditional. After resolving a query the store could not answer, background-teach the final resource mapping - no call-count threshold, no judging whether it was "worth" learning. The teach is the anchor of the loop: it triggers playbook synthesis for a family without a playbook, and same-referent phrasings fold into one family so near-duplicate teaches do not fragment the store. Fire it after assembling your user-facing response but BEFORE emitting it, with a shell `&` so the call returns immediately. Pass the query the same way as recall — argv/MCP, or file-then-`$QUERY`. Do not splice the question into the command text:

```bash
QUERY=$(cat /path/to/question.txt)
levanta-pp-cli teach --query "$QUERY" --resource-type <type> --resource <id1> --resource <id2>
# (append shell `&` to background it)
```

Silent on success. Errors only land in `teach.log` under the resolved state dir. Teach the **most specific** resource - if the user asked a broad question and you walked through parent records to find the specific answer, teach the leaf id, not the parent. The CLI uses seeded `entity_lookups` for cross-alias resolution at recall time, so a teach under one alias (e.g., "Niners") satisfies future queries under another alias (e.g., "49ers", "San Francisco") automatically.

PII rule: teach the structural question with identifiers stripped - never include names, emails, phone numbers, account ids, or other personal identifiers in taught queries or notes. The CLI scans teach queries for obvious email/phone shapes and warns, but does not block; strip before teaching rather than relying on the warning.

### Step 5: playbooks - optional flags, automatic synthesis

You do not need to decide whether a session "deserves" a playbook: a teach on a family without one auto-synthesizes a `playbook_candidate` from the session's journal, and the next session judges it via confirm/reject. Attach explicit playbook flags only when you already hold choreography worth recording verbatim - workarounds the CLI didn't surface (silently-dropped flags, undocumented params, pagination tricks, payload gotchas). Prefer the **integrated one-call form** - record the resource learning and the playbook in the same `teach` invocation:

```bash
# Common case: record both the resource learning AND the playbook in one call.
QUERY=$(cat /path/to/question.txt)
levanta-pp-cli teach \
  --query "$QUERY" \
  --resource <id> \
  --playbook-file ~/playbooks/<shape>.json \
  --playbook-notes-file ~/playbooks/<shape>-notes.md
# (append shell `&` to background it)

# Alternate: playbook-only (no resource to record alongside).
QUERY=$(cat /path/to/question.txt)
levanta-pp-cli teach-playbook \
  --query "$QUERY" \
  --playbook-file ~/playbooks/<shape>.json \
  --notes-file ~/playbooks/<shape>-notes.md
```

Playbook files are JSON with `steps`, `entity_slots`, `expected_tool_calls`. Notes files are markdown carrying the gotchas verbatim. File-free callers (MCP-only agents) pass the same content inline: `--playbook-json` and `--playbook-notes` on the integrated `teach` form, `--playbook-json` and `--notes` on `teach-playbook`. On the integrated `teach` form, the playbook flags are optional - omit them entirely for a resource-only teach. On the standalone `teach-playbook` form, at least one of the playbook and notes flags must be set; both empty is rejected. Playbooks are keyed on the structural query family (entities stripped) so a recipe taught from one entity-shaped query applies to every other query of the same shape, with `slots_resolved` binding the live query's canonical at recall time.

When you DO find a playbook on a future recall, treat it as ground truth: replay the steps with `slots_resolved` substitutions, skip the discovery that the choreography already documents, and read `notes` before any step.

### Step 6: `playbook amend &` when your debug response identifies a correction

If your debug-protocol response identifies a concrete correction the notes or playbook should know — a workaround, an undocumented endpoint shape, a stale field name, observed schema drift, an empty-payload fallback — fire `playbook amend` BEFORE emitting your user-facing response. Same fire-and-forget posture as `teach`. Pass the query and note as argv/MCP arguments, or write each with a non-shell file tool and read them back (`QUERY=$(cat ...)`, `NOTE=$(cat ...)`). Do not interpolate either string into the command text:

```bash
QUERY=$(cat /path/to/question.txt)
NOTE=$(cat /path/to/note.txt)
levanta-pp-cli playbook amend \
  --query "$QUERY" \
  --add-note "$NOTE"
# (append shell `&` to background it)
```

What counts as worth amending: a behavior you OBSERVED this session that future-you would benefit from knowing. Examples worth amending:

- A workaround for a CLI surface that silently drops or misorders a flag.
- An undocumented endpoint shape (response wrapped in `{meta, results}`, payload nested two levels deeper than the docs claim).
- Observed schema drift (a field renamed, an index that shifted between seasons, a category label that the API now returns lower-cased).

What does NOT belong in notes:

- The year-specific or entity-specific answer to the user's question. That's the response, not a learning.
- Per-team / per-athlete / per-row data the playbook already retrieves at runtime.
- Statements that paraphrase what the existing notes already say.

The amend command appends to the family's existing notes with a timestamped marker (`[amend YYYY-MM-DDTHH:MMZ]: <text>`). Multiple amends accumulate; the audit trail is visible. If no playbook exists yet for the family, amend creates a notes-only one (so cold-start corrections still land).

#### PII discipline for amend notes

`playbook amend` notes are designed to potentially flow upstream as shared knowledge in future versions of the Printing Press. Keep them clean of user-identifying content so the upstream-contribution path stays open without retroactive scrubbing:

- **Do NOT embed** paths to user filesystems, personal API keys or tokens, user email addresses, user GitHub handles, or specific query histories tied to a single user.
- **Acceptable**: endpoint shapes, undocumented field names, API gotchas, observed schema drift, workarounds for CLI surfaces, generalizable pagination or retry tactics.

If a correction is only meaningful with user-specific context, it belongs in a personal note, not in the playbook amend.

### Measuring the loop

`levanta-pp-cli learnings stats` reports recall hit rate, teach-to-reuse, playbook resolution rate, and candidate confirm/reject counts from the local `learn_events` table. Rates are null until they have a denominator; everything stays on this machine. Use it to check whether the loop is earning its keep for this CLI.

### Disabling learning

- `--no-learn` on a single command short-circuits both `recall` and the `teach` write path. Use for deterministic agent flows or tests that must not be affected by accumulated learnings.
- `LEVANTA_NO_LEARN=true` in the environment globally disables the pipeline.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
levanta-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
levanta-pp-cli feedback --stdin < notes.txt
levanta-pp-cli feedback list --json --limit 10
```

Entries are stored locally as `feedback.jsonl` under the resolved data dir. They are never POSTed unless `LEVANTA_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `LEVANTA_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. Three sinks are supported:

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` (tmp + rename). Binary-response commands write decoded payload bytes (not the base64 JSON envelope) and print a small JSON receipt on stdout; `--json`/`--csv` do not refuse when this sink is set. |
| `webhook:<url>` | POST the output body to the URL (`application/json`) |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures return non-zero and log the URL + HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled or recurring agent reuses the same saved flags while providing different input each run.

```
levanta-pp-cli profile save briefing --json
levanta-pp-cli --profile briefing brands list
levanta-pp-cli profile list --json
levanta-pp-cli profile show briefing
levanta-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 4 | Authentication required |
| 5 | API error (upstream issue) |
| 6 | Partial failure |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `levanta-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

1. Install the MCP server:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/marketing/levanta/cmd/levanta-pp-mcp@latest
   ```
2. Register with Claude Code:
   ```bash
   claude mcp add levanta-pp-mcp -- levanta-pp-mcp
   ```
3. Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which levanta-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   levanta-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `levanta-pp-cli <command> --help`.
