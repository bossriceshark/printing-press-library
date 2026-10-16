// Copyright 2026 Cursor Agent and contributors. Licensed under Apache-2.0. See LICENSE.
// PATCH: brand-months with report commission and no invoice item. pp:data-source local

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newNovelPayoutGapCmd(flags *rootFlags) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "gap",
		Short: "Show brand-months where reports earned commission and no invoice item exists.",
		Long: `Groups synced creator report rows by the invoice brand, year, month, and
marketplace stamped on each row. A group is a gap when its commission is
above zero and the local invoice ledger has no matching item.`,
		Example: stringsTrimExample(`
  levanta-pp-cli payout gap --agent --select brand,year,month,marketplace,commission
  levanta-pp-cli payout gap --limit 20`),
		Annotations: map[string]string{
			"mcp:read-only":  "true",
			"pp:data-source": "local",
			"pp:happy-args":  "--limit=10",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "payout gap")
			}
			if limit < 0 {
				return usageErr(fmt.Errorf("--limit must be zero or greater"))
			}
			loaded, db, err := loadMany(cmd.Context(), ledgerReports, ledgerInvoices, ledgerBrands)
			if err != nil {
				return fmt.Errorf("payout gap: %w", err)
			}
			defer closeLedger(db)
			noteUnsynced(cmd, db, ledgerReports, flags.maxAge)
			rows := applyLimit(payoutGaps(loaded[ledgerInvoices], loaded[ledgerReports], loaded[ledgerBrands]), limit)
			if rows == nil {
				rows = []payoutGapRow{}
			}
			return emitLedger(cmd, flags, rows, "no payout gaps in the local store")
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum rows to print (0 means no cap)")
	return cmd
}
