---
name: pp-digg
description: "Tail the Digg AI 1000's news cycle from the terminal — read-only, with the full pipeline event stream and... Trigger phrases: `what's trending on Digg AI`, `digg ai top stories`, `what climbed the AI news rankings`, `Digg AI 1000 leaderboard`, `what got replaced on Digg`, `tail the Digg pipeline`, `use digg`, `run digg-pp-cli`."
author: "Matt Van Horn"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - digg-pp-cli
---

# Digg AI — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `digg-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer:
   ```bash
   npx -y @mvanhorn/printing-press install digg --cli-only
   ```
2. Verify: `digg-pp-cli --version`
3. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

If the `npx` install fails before this CLI has a public-library category, install Node or use the category-specific Go fallback after publish.

If `--version` reports "command not found" after install, the install step did not put the binary on `$PATH`. Do not proceed with skill commands until verification succeeds.

Digg AI is a curated leaderboard of 1,000 AI accounts on X and the story clusters they surface. The web UI shows you today's snapshot. This CLI tails the pipeline events, keeps a local rank-history that survives daily overwrites, and exposes Digg's own replacement rationale and gravity components so an agent can answer 'why this story?' and 'what got dropped overnight?' with structured data.

## When to Use This CLI

Use this CLI when an agent or power user needs structured access to Digg AI's rankings, ranking-change history, pipeline events, or per-cluster transparency record. It is the right tool for tracking AI-news cycle movement, building cross-aggregator research over HN+Techmeme+Digg, or exposing Digg AI signals into a larger automation. Do NOT use it for vote, comment, or post automation — those mutations are explicitly out of scope.

## When Not to Use This CLI

Do not activate this CLI for requests that require creating, updating, deleting, publishing, commenting, upvoting, inviting, ordering, sending messages, booking, purchasing, or changing remote state. This printed CLI exposes read-only commands for inspection, export, sync, and analysis.

## Unique Capabilities

These capabilities aren't available in any other tool for this API.

### Live pipeline observability
- **`events`** — Tail Digg's ingestion pipeline in real time — see clusters as they're detected, stories fast-climbing the leaderboard with explicit rank deltas, X posts being processed, batch breakdowns.

  _When an agent needs 'tell me when story X just climbed N ranks' or 'what new clusters did Digg detect in the last hour', this is the only way._

  ```bash
  digg-pp-cli events --since 1h --type fast_climb --json --select clusterId,label,delta,currentRank,previousRank
  ```
- **`watch`** — Poll /ai, diff against last snapshot, alert when any cluster moves N+ ranks.

  _Read-only operational watcher; never writes anything back to Digg._

  ```bash
  digg-pp-cli watch --alert 'rank.delta>=10'
  ```
- **`pipeline status`** — One-screen view of /api/trending/status: isFetching, nextFetchAt, storiesToday, clustersToday, last 5 events.

  _Lets ops and power users see when a fresh batch is about to land and what's been ingested in the last hour._

  ```bash
  digg-pp-cli pipeline status --watch
  ```

### Local state that compounds
- **`replaced`** — Show stories that were knocked out of the rankings since the last sync, with Digg's own published replacement rationale.

  _Best-of-feed shifts faster than people remember. This makes 'what did Digg drop and why' queryable._

  ```bash
  digg-pp-cli replaced --since 24h --json
  ```
- **`crossref`** — Show this cluster's Hacker News and Techmeme mirrors when Digg has detected the story is being discussed there.

  _Removes the manual 'is HN talking about this too' step from any cross-aggregator research workflow._

  ```bash
  digg-pp-cli crossref iq7usf9e
  ```
- **`authors top`** — Top accounts in the Digg AI 1000 ranked by Digg's influence score, story count, or reach.

  _Investors and AI scouts care which accounts move the news cycle. Now queryable, sortable, scriptable._

  ```bash
  digg-pp-cli authors top --by influence --limit 50 --json
  ```
- **`history`** — Full trajectory of one cluster's currentRank, peakRank, and delta over local snapshot history.

  _'Entered at #18, peaked at #4 over 6h, dropped to #22 by 24h' is impossible to learn from the live site._

  ```bash
  digg-pp-cli history iq7usf9e --json
  ```
- **`author`** — Every cluster a given X account contributed to, with post type (original, retweet, quote, reply).

  _'Show me every story this account surfaced this week' is the investor-scout query._

  ```bash
  digg-pp-cli author Scobleizer --since 7d --json
  ```

### Transparency
- **`evidence`** — Print the full ranking transparency record for one cluster — scoreComponents, evidence array, numeratorLabel, percentAboveAverage.

  _When a user asks 'why is THIS the top story', the answer is structured data; agents can compose with it._

  ```bash
  digg-pp-cli evidence iq7usf9e --json
  ```
- **`sentiment`** — Read per-time-window positivity ratios (pos6h, pos12h, pos24h, posLast) for a cluster.

  _Tells an agent whether the conversation around a story is still net-positive or has soured; useful before quoting a story._

  ```bash
  digg-pp-cli sentiment iq7usf9e --window 6h --json
  ```

## HTTP Transport

This CLI uses Chrome-compatible HTTP transport for browser-facing endpoints. It does not require a resident browser process for normal API calls.

## Command Reference

**feed** — Top-level AI story feed (HTML page; CLI parses the embedded RSC stream)

- `digg-pp-cli feed raw` — Fetch the raw /ai HTML page. The CLI's sync command parses this; most users should run `sync` then `top` instead of...
- `digg-pp-cli feed story_raw` — Fetch the raw /ai/{clusterUrlId} story detail page (HTML). The CLI's `story` command parses this; users should not...

**trending** — Public ingestion-pipeline status and event stream

- `digg-pp-cli trending` — Read the current pipeline status: storiesToday, clustersToday, isFetching, nextFetchAt, and the recent event stream...


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
digg-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Recipes


### What climbed >=10 ranks in the last hour

```bash
digg-pp-cli events --since 1h --type fast_climb --json --select clusterId,label,delta,currentRank,previousRank
```

Reads the public events stream, filters to fast-climb events only, and narrows the JSON to the five fields an agent actually needs.

### Why is a story the top story

```bash
digg-pp-cli evidence 65idu2x5 --json
```

Print the scoreComponents and evidence array for one cluster. Get a clusterUrlId from `digg-pp-cli top --json --select clusterUrlId`.

### Show every cluster a given X account contributed to this week

```bash
digg-pp-cli author Scobleizer --since 7d --json --select label,clusterUrlId,activityAt
```

Queries the local store for clusters where the named author was a contributor; output is narrowed for agent consumption.

### Cross-reference a story across HN and Techmeme

```bash
digg-pp-cli crossref 65idu2x5
```

Uses Digg's own hackerNews/techmeme reference fields so you don't have to search those sites manually. Pass any clusterUrlId from `top --json --select clusterUrlId`.

### Tail the pipeline live

```bash
digg-pp-cli pipeline status --watch
```

One-screen dashboard of isFetching, nextFetchAt, storiesToday, clustersToday, and the last few pipeline events.

## Auth Setup

No authentication required.

Run `digg-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  digg-pp-cli feed raw --agent --select id,name,status
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag
- **Read-only** — do not use this CLI for create, update, delete, publish, comment, upvote, invite, order, send, or other mutating requests

### Response envelope

Commands that read from the local store or the API wrap output in a provenance envelope:

```json
{
  "meta": {"source": "live" | "local", "synced_at": "...", "reason": "..."},
  "results": <data>
}
```

Parse `.results` for data and `.meta.source` to know whether it's live or local. A human-readable `N results (live)` summary is printed to stderr only when stdout is a terminal — piped/agent consumers get pure JSON on stdout.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
digg-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
digg-pp-cli feedback --stdin < notes.txt
digg-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.digg-pp-cli/feedback.jsonl`. They are never POSTed unless `DIGG_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `DIGG_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. Three sinks are supported:

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` (tmp + rename) |
| `webhook:<url>` | POST the output body to the URL (`application/json` or `application/x-ndjson` when `--compact`) |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures return non-zero and log the URL + HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled agent calls the same command every run with the same configuration - HeyGen's "Beacon" pattern.

```
digg-pp-cli profile save briefing --json
digg-pp-cli --profile briefing feed raw
digg-pp-cli profile list --json
digg-pp-cli profile show briefing
digg-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `digg-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

Install the MCP binary from this CLI's published public-library entry or pre-built release, then register it:

```bash
claude mcp add digg-pp-mcp -- digg-pp-mcp
```

Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which digg-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   digg-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `digg-pp-cli <command> --help`.
