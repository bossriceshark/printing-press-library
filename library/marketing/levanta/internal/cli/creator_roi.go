// Copyright 2026 Cursor Agent and contributors. Licensed under Apache-2.0. See LICENSE.
// PATCH: rank seller creators by commission-to-sales across two windows. pp:data-source local

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newNovelCreatorRoiCmd(flags *rootFlags) *cobra.Command {
	var days int
	var limit int

	cmd := &cobra.Command{
		Use:   "roi",
		Short: "Rank seller creators by commission-to-sales and by the change versus the previous window.",
		Long: `Reads synced seller report rows, groups them by creator, and ranks the
current --days window by commission divided by sales. Delta is that ratio
minus the same ratio on the previous window of equal length. Creator names
come from synced seller creators when those rows are present.

This command reads the seller report collection. A creator API key does
not populate it; sync seller reports with a seller key first.`,
		Example: stringsTrimExample(`
  levanta-pp-cli creator roi --agent --select creator,commission,sales,roi,delta
  levanta-pp-cli creator roi --days 14 --limit 20`),
		Annotations: map[string]string{
			"mcp:read-only":  "true",
			"pp:data-source": "local",
			"pp:happy-args":  "--limit=10",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "creator roi")
			}
			if days < 1 {
				return usageErr(fmt.Errorf("--days must be 1 or greater"))
			}
			if limit < 0 {
				return usageErr(fmt.Errorf("--limit must be zero or greater"))
			}
			loaded, db, err := loadMany(cmd.Context(), ledgerSellerReports, ledgerSellerCreators)
			if err != nil {
				return fmt.Errorf("creator roi: %w", err)
			}
			defer closeLedger(db)
			noteUnsynced(cmd, db, ledgerSellerReports, flags.maxAge)
			windows := newLedgerWindows(days, ledgerNow())
			rows := applyLimit(creatorROI(loaded[ledgerSellerReports], loaded[ledgerSellerCreators], windows), limit)
			if rows == nil {
				rows = []creatorROIRow{}
			}
			return emitLedger(cmd, flags, rows, "no seller creator earnings in the local store for this window")
		},
	}
	cmd.Flags().IntVar(&days, "days", 30, "Length of each comparison window in days")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum rows to print (0 means no cap)")
	return cmd
}
