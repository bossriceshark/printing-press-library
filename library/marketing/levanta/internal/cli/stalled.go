// Copyright 2026 Cursor Agent and contributors. Licensed under Apache-2.0. See LICENSE.
// PATCH: active links that earned in the previous window and are quiet now. pp:data-source local

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newNovelStalledCmd(flags *rootFlags) *cobra.Command {
	var days int
	var limit int

	cmd := &cobra.Command{
		Use:   "stalled",
		Short: "List active links that earned commission in the previous window and earn nothing now.",
		Long: `Compares two adjacent windows of synced creator report rows, each --days
long. A link is stalled when it is still active, earned commission in the
older window, and earned none in the window that ends now.`,
		Example: stringsTrimExample(`
  levanta-pp-cli stalled --agent --select url,asin,source,prior_commission
  levanta-pp-cli stalled --days 14`),
		Annotations: map[string]string{
			"mcp:read-only":  "true",
			"pp:data-source": "local",
			"pp:happy-args":  "--limit=10",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "stalled")
			}
			if days < 1 {
				return usageErr(fmt.Errorf("--days must be 1 or greater"))
			}
			if limit < 0 {
				return usageErr(fmt.Errorf("--limit must be zero or greater"))
			}
			loaded, db, err := loadMany(cmd.Context(), ledgerReports, ledgerLinks)
			if err != nil {
				return fmt.Errorf("stalled: %w", err)
			}
			defer closeLedger(db)
			noteUnsynced(cmd, db, ledgerReports, flags.maxAge)
			windows := newLedgerWindows(days, ledgerNow())
			rows := applyLimit(stalledLinks(loaded[ledgerReports], loaded[ledgerLinks], windows), limit)
			if rows == nil {
				rows = []stalledRow{}
			}
			return emitLedger(cmd, flags, rows, "no stalled links in the local store for this window")
		},
	}
	cmd.Flags().IntVar(&days, "days", 30, "Length of each comparison window in days")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum rows to print (0 means no cap)")
	return cmd
}
