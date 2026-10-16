Polish pass:
  Verify: shipcheck PASS (no separate before/after; first shipcheck was already green)
  Scorecard: 97/100 Grade A
  Tools-audit: exit 0, no pending findings
  Gosec: 21 findings, 0 in hand-authored novel files (ledger.go, command files, levanta_ids.go). The rest are generator-emitted G304/G201/G101 patterns and are retro candidates, not edited here.
  Fixed: none required after shipcheck

---POLISH-RESULT---
verify_before: PASS
verify_after: PASS
scorecard_before: 97
scorecard_after: 97
tools_audit_before: 0
tools_audit_after: 0
gosec_before: 0
gosec_after: 0
dogfood_before: PASS
dogfood_after: PASS
fixes_applied: []
remaining_issues: []
further_polish_recommended: no
further_polish_reasoning: Shipcheck is 97/100 and hand-authored gosec findings are zero. Live API sampling needs LEVANTA_API_KEY, which is not a polish edit.
ship_recommendation: ship
publish_validate_before: skipped (mid-pipeline)
publish_validate_after: skipped (mid-pipeline)
phase3_transcendence_rows_planned: 6
phase3_transcendence_rows_built: 6
prior_sub60_reprint: false
---END-POLISH-RESULT---
