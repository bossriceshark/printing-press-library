# Phase 4.85 output review

status: PASS

Sampled local commands against an empty store and against the unit fixtures in internal/cli/ledger_test.go.

- unpaid, earned, earnings reconcile, payout gap, stalled, and creator roi exit 0.
- --json emits [] when the store is empty.
- --agent --select returns {"meta":{"source":"local"},"results":[]}.
- --dry-run prints the dry-run line and does not open the network.
- Unit tests cover pending/late filtering, link rollup, commission and status mismatch, missing invoice gaps, stalled active links, and creator ROI delta.

No plausibility bug found in the local ledger math. Live row shapes are untested because no API key is available.
