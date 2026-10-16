# Fidelity CLI

**Read a Fidelity book from the official CSV exports, with portfolio, activity, and coverage queries no spreadsheet has.**

Fidelity Investments does not publish a retail brokerage API. fidelity-pp-cli imports the positions and history CSV files the owner downloads from fidelity.com into a local SQLite ledger, then answers balances, holdings, dividends, and what changed. It never stores a password, cookie, or token, and it has no trade or transfer commands.

## Install

The recommended path installs both the `fidelity-pp-cli` binary and the `pp-fidelity` agent skill (Claude Code, Codex, Cursor, Gemini CLI, GitHub Copilot, and other agents supported by the upstream [`skills`](https://github.com/vercel-labs/skills) CLI) in one shot:

```bash
npx -y @mvanhorn/printing-press-library install fidelity
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press-library install fidelity --cli-only
```

For skill only — installs the skill into the same agents as the default command above, but skips the CLI binary (use this to update or reinstall just the skill):

```bash
npx -y @mvanhorn/printing-press-library install fidelity --skill-only
```

To constrain the skill install to one or more specific agents (repeatable — agent names match the [`skills`](https://github.com/vercel-labs/skills) CLI):

```bash
npx -y @mvanhorn/printing-press-library install fidelity --agent claude-code
npx -y @mvanhorn/printing-press-library install fidelity --agent claude-code --agent codex
```

### Without Node (Go fallback)

If `npx` isn't available (no Node, offline), install the CLI directly via Go (requires Go 1.26.9 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/payments/fidelity/cmd/fidelity-pp-cli@latest
```

This installs the CLI only — no skill.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/fidelity-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

Install the CLI binary first. The installer writes binaries to a per-user managed bin directory by default: `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows.

```bash
npx -y @mvanhorn/printing-press-library install fidelity --cli-only
```

Then install the focused Hermes skill.

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-fidelity --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-fidelity --force
```

Restart the Hermes session or gateway if the newly installed skill is not visible immediately.

## Install for OpenClaw
Install both the CLI binary and the focused OpenClaw skill. The installer defaults binaries to a per-user bin directory (`$HOME/.local/bin` on macOS/Linux, `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows):

```bash
npx -y @mvanhorn/printing-press-library install fidelity --agent openclaw
```

Restart the OpenClaw session or gateway if the newly installed skill is not visible immediately.

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/fidelity-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


```bash
go install github.com/mvanhorn/printing-press-library/library/payments/fidelity/cmd/fidelity-pp-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "fidelity": {
      "command": "fidelity-pp-mcp"
    }
  }
}
```

</details>

## Authentication

No credential is required. Fidelity does not offer a self-serve retail API key. Download Portfolio_Positions_MMM-DD-YYYY.csv from Accounts & Trade, Portfolio, Positions, and History_for_Account_<id>.csv from Activity & Orders. Import those files locally. Do not set FIDELITY_PASSWORD, FIDELITY_TOKEN, or a session cookie. WorkplaceXchange, Fidelity Access, and aggregator OAuth are separate partner programs and are not used by this CLI.

## Quick Start

```bash
# Confirm the binary is the local-ledger CLI and that no brokerage credential is required.
fidelity-pp-cli doctor --dry-run

# Preview a positions export before writing the ledger. Use a file you downloaded; do not commit it.
fidelity-pp-cli import positions --file ./Portfolio_Positions_Oct-16-2026.csv --dry-run

# Load holdings into the local ledger.
fidelity-pp-cli import positions --file ./Portfolio_Positions_Oct-16-2026.csv

# Load activity for one account. The account id is the one in the filename.
fidelity-pp-cli import history --file ./History_for_Account_Z1234567.csv

# Read per-account market value, cash versus invested, and unrealized gain.
fidelity-pp-cli portfolio summary --agent

```

## Unique Features

### Local ledger
- **`portfolio summary`** — See per-account and whole-book market value, cash versus invested, unrealized gain, and the largest positions from the latest export.

  _Use this when the owner asks what the book is worth right now, split by account, without opening www.fidelity.com._

  ```bash
  fidelity-pp-cli portfolio summary --agent
  ```
- **`activity summary`** — Total imported history by action, including dividends, commissions, and fees, over a run-date window.

  _Use this for dividends, interest, buys, or fees over a period instead of scanning the activity CSV._

  ```bash
  fidelity-pp-cli activity summary --from 2026-01-01 --action DIVIDEND --agent
  ```
- **`changes`** — Show holdings that appeared, disappeared, or changed quantity after a newer positions file, plus history rows from the latest history import.

  _Use this after a fresh download when the question is what moved, not what is held now._

  ```bash
  fidelity-pp-cli changes --account Z1234567 --agent
  ```
- **`symbols total`** — Sum one symbol's shares, market value, cost basis, and gain across brokerage and retirement accounts.

  _Use this when the owner asks where a ticker is held and how much, across accounts._

  ```bash
  fidelity-pp-cli symbols total VTI --agent
  ```
- **`coverage`** — Show which accounts have a positions snapshot, which have history, and the run-date span of that history.

  _Use this before trusting a book total, to see which account export is missing._

  ```bash
  fidelity-pp-cli coverage --agent
  ```

## Recipes

### Import and summarize

```bash
fidelity-pp-cli portfolio summary --agent --select book.market_value,accounts.account_number,accounts.market_value
```

Narrow the portfolio summary to book value and per-account market value.

### Dividends this year

```bash
fidelity-pp-cli activity summary --from 2026-01-01 --action DIVIDEND --agent
```

Sum dividend rows in the imported history.

### Where is VTI

```bash
fidelity-pp-cli symbols total VTI --agent
```

Sum one symbol across brokerage and retirement accounts.

### What changed

```bash
fidelity-pp-cli changes --account Z1234567 --agent
```

Compare the latest positions file with the previous snapshot for one account.

### Which export is missing

```bash
fidelity-pp-cli coverage --agent
```

See which accounts have positions, which have history, and the history date span.

## Usage

Run `fidelity-pp-cli --help` for the full command reference and flag list.

## Paths & environment variables

This CLI separates local files into four path kinds:

| Kind | Contents |
|------|----------|
| `config` | User-editable settings such as `config.toml` and saved profiles |
| `data` | Durable local data such as `data.db` |
| `state` | Runtime state such as persisted queries, jobs, and `teach.log` |
| `cache` | Regenerable HTTP/cache files |

Each kind resolves independently. The ladder is:

1. Per-kind env var: `FIDELITY_CONFIG_DIR`, `FIDELITY_DATA_DIR`, `FIDELITY_STATE_DIR`, or `FIDELITY_CACHE_DIR`
2. `--home <dir>` for this invocation
3. `FIDELITY_HOME` for a flat relocated root
4. XDG env vars: `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`
5. Platform defaults matching existing installs

For containers and agent sandboxes, prefer a single relocated root:

```bash
export FIDELITY_HOME=/srv/fidelity
fidelity-pp-cli doctor
```

Under `FIDELITY_HOME=/srv/fidelity`, the four dirs resolve to `/srv/fidelity/config`, `/srv/fidelity/data`, `/srv/fidelity/state`, and `/srv/fidelity/cache`.

MCP servers do not receive CLI flags from the host. Put relocation in the host `env` block:

```json
{
  "mcpServers": {
    "fidelity": {
      "command": "fidelity-pp-mcp",
      "env": {
        "FIDELITY_HOME": "/srv/fidelity"
      }
    }
  }
}
```

Precedence matters in fleets: an ambient per-kind variable such as `FIDELITY_DATA_DIR` overrides an explicit `--home` for that kind. Use `FIDELITY_HOME` or the per-kind variables for durable fleet relocation; treat `--home` as the weaker per-invocation lever.

Relocation is one-way. Unsetting `FIDELITY_HOME` does not move files back to platform defaults, and `doctor` cannot find files left under a former root. Move the files manually before unsetting relocation variables.

Existing installs keep working because the platform-default rung matches the legacy layout. Run `fidelity-pp-cli doctor --fail-on warn` to check path warnings in automation.

## Commands

### accounts

Accounts present in imported Fidelity exports

- **`fidelity-pp-cli accounts`** - List accounts imported from positions and history CSV files

### positions

Holdings from the latest positions CSV snapshot

- **`fidelity-pp-cli positions`** - List holdings from the latest imported positions snapshot

### transactions

Activity rows imported from history CSV files

- **`fidelity-pp-cli transactions`** - List imported activity rows


### activity


- **`fidelity-pp-cli activity summary`** - Total imported history by action, including dividends, commissions, and fees, over a run-date window.


### changes


- **`fidelity-pp-cli changes`** - Show holdings that appeared, disappeared, or changed quantity after a newer positions file, plus history rows from the latest history import.


### coverage


- **`fidelity-pp-cli coverage`** - Show which accounts have a positions snapshot, which have history, and the run-date span of that history.


### portfolio


- **`fidelity-pp-cli portfolio summary`** - See per-account and whole-book market value, cash versus invested, unrealized gain, and the largest positions from the latest export.


### symbols


- **`fidelity-pp-cli symbols total`** - Sum one symbol's shares, market value, cost basis, and gain across brokerage and retirement accounts.




### Self-learning loop

This CLI caches per-question discovery so repeat queries skip the walk and structurally similar queries get answered via entity substitution. The loop also self-captures: every invocation is journaled locally, and failed-flag corrections plus fresh teaches surface as candidates on the next `recall` for confirm/reject judgment. Agents call `recall` before discovery and fire `teach &` after answering. See the `## Automatic learning` section in `SKILL.md` for the full protocol.

- **`fidelity-pp-cli recall <query>`** - Look up cached resources for a query before running discovery
- **`fidelity-pp-cli teach`** - Record a query -> resource mapping (silent on success, safe to background with `&`)
- **`fidelity-pp-cli learnings list`** - Inspect taught rows
- **`fidelity-pp-cli learnings forget <query>`** - Undo a teach
- **`fidelity-pp-cli learnings candidates`** - List auto-captured candidates awaiting confirm/reject
- **`fidelity-pp-cli learnings stats`** - Local loop metrics: recall hit rate, teach-to-reuse, playbook resolution, candidate counts
- **`fidelity-pp-cli teach-pattern`** - Install a query/resource template up front
- **`fidelity-pp-cli teach-lookup`** - Add an entity mapping (e.g. country code, team alias) for pattern substitution

Pass `--no-learn` or set `FIDELITY_NO_LEARN=true` to disable the loop for deterministic flows.

The local store's schema version stamp is one-way: once this version of `fidelity-pp-cli` opens the database, older binaries refuse it with a version error — upgrade the binary rather than downgrading.

## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
fidelity-pp-cli accounts

# JSON for scripting and agents
fidelity-pp-cli accounts --json
# Filter to specific fields
fidelity-pp-cli accounts --json --select account_number,account_name

# Dry run — show the request without sending
fidelity-pp-cli accounts --dry-run

# Agent mode — JSON + compact + no prompts in one flag
fidelity-pp-cli accounts --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select <field>[,<field>...]` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Read-only by default** - this CLI does not create, update, delete, publish, send, or mutate remote resources
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `5` API error, `7` rate limited or blocked, `10` config error.

## Health Check

```bash
fidelity-pp-cli doctor
```

Verifies configuration and connectivity to the API.

## Configuration

Run `fidelity-pp-cli doctor` to see the resolved config, data, state, and cache directories. The platform-default config path is `~/.config/fidelity/config.toml`; `--home`, `FIDELITY_HOME`, and per-kind env vars can relocate it.

Static request headers can be configured under `headers`; per-command header overrides take precedence.

## Troubleshooting
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific
- **import positions reports no data rows** — Confirm the header row contains Account Number and Symbol. Disclaimer footer rows are skipped; a file that is only the footer will import nothing.
- **portfolio summary cash looks doubled** — Cash versus invested uses position Type and SPAXX or FDRXX. A history Cash Balance, when present, is a separate labeled field. Do not add them.
- **doctor says the ledger is empty** — Import a positions CSV first. There is no network sync, because Fidelity publishes no retail API.

## Sources & Inspiration

This CLI was built by studying these projects and resources:

- [**portfolioSummary**](https://github.com/rabinnh/portfolioSummary) — Python (3 stars)
- [**fidelity-account-overview**](https://github.com/gerardrbentley/fidelity-account-overview) — Python
- [**fidelity_csv_converter**](https://github.com/sztupkay/fidelity_csv_converter) — Python
- [**MoneydanceScripts**](https://github.com/tkralphs/MoneydanceScripts) — Python
- [**felix-august**](https://github.com/justfetz/felix-august) — Python
- [**alpha-terminal**](https://github.com/ronitg1/alpha-terminal) — Python
- [**claude-fidelity-mcp**](https://github.com/TylerFlar/claude-fidelity-mcp) — TypeScript

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
