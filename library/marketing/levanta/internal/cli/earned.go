// Copyright 2026 Cursor Agent and contributors. Licensed under Apache-2.0. See LICENSE.
// PATCH: local link earnings rollup. pp:data-source local

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newNovelEarnedCmd(flags *rootFlags) *cobra.Command {
	var days int
	var limit int

	cmd := &cobra.Command{
		Use:   "earned",
		Short: "Roll creator report rows up to one line per link that earned commission.",
		Long: `Groups synced creator report rows by link over the last --days and
keeps links whose commission is above zero. Daily report rows stay in the
store; this command answers which links earned.`,
		Example: stringsTrimExample(`
  levanta-pp-cli earned --agent --select url,asin,source,commission,clicks
  levanta-pp-cli earned --days 7`),
		Annotations: map[string]string{
			"mcp:read-only":  "true",
			"pp:data-source": "local",
			"pp:happy-args":  "--limit=10",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "earned")
			}
			if days < 1 {
				return usageErr(fmt.Errorf("--days must be 1 or greater"))
			}
			if limit < 0 {
				return usageErr(fmt.Errorf("--limit must be zero or greater"))
			}
			reports, db, err := loadResourceMaps(cmd.Context(), ledgerReports)
			if err != nil {
				return fmt.Errorf("earned: %w", err)
			}
			defer closeLedger(db)
			noteUnsynced(cmd, db, ledgerReports, flags.maxAge)
			windows := newLedgerWindows(days, ledgerNow())
			rows := applyLimit(earnedFromReports(reports, windows.currentStart, windows.currentEnd), limit)
			if rows == nil {
				rows = []earnedRow{}
			}
			return emitLedger(cmd, flags, rows, "no links earned commission in the local store for this window")
		},
	}
	cmd.Flags().IntVar(&days, "days", 30, "Length of the earnings window in days, ending now")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum rows to print (0 means no cap)")
	return cmd
}
