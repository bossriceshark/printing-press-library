# Aruba Central printed CLI

This directory is `aruba-central-pp-cli`, a curated New Central client in the printing-press catalog. The generator binary was not available when this package was published, so the tree is hand-curated rather than a full OpenAPI print. Keep later edits narrow and record them in `.printing-press-patches/`.

## Local operating contract

```bash
aruba-central-pp-cli doctor
aruba-central-pp-cli sync --since 24h
aruba-central-pp-cli brief
aruba-central-pp-cli export --format json
```

Add `--agent` for JSON, no prompts, and `--yes` on gated writes. Writes still require `ENABLE_WRITES=1`. Classic Central is rejected.

Credentials are the same three variables as `central-mcp-server`:

- `CENTRAL_BASE_URL`
- `CENTRAL_CLIENT_ID`
- `CENTRAL_CLIENT_SECRET`

Never commit those values. `doctor` exits 10 with a JSON error when any of them is missing.
