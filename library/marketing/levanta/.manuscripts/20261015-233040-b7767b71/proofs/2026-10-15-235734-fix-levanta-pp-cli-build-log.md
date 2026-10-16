Manifest transcendence rows: 6 planned, 6 built. Phase 3 will not pass until all 6 ship.

Built:
- unpaid
- earnings reconcile
- earned
- payout gap
- stalled
- creator roi

All six read the local SQLite store (`pp:data-source local`). Empty stores exit 0 with `[]`.

Stable row ids: invoice items key on brand, year, month (including January = 0), and marketplace. Creator report rows key on date, ASIN, marketplace, and link. Seller v2 creator performance rows key on creator id. See `internal/store/levanta_ids.go`.

Deferred: live dogfood against app.levanta.io. No LEVANTA_API_KEY is available in this session, and the print request forbids pasting secrets.

Generator limitation: the profiler picked `month` and `asin` as id fields. The hand patch runs before that override so sync does not collapse distinct rows.
