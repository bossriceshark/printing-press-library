// Copyright 2026 Cursor Agent and contributors. Licensed under Apache-2.0. See LICENSE.
// PATCH: compare invoice items with stored creator report rows. pp:data-source local

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newNovelEarningsReconcileCmd(flags *rootFlags) *cobra.Command {
	var tolerance float64
	var limit int

	cmd := &cobra.Command{
		Use:   "reconcile",
		Short: "Flag invoice items whose commission or paid status disagrees with stored report rows.",
		Long: `Joins synced invoice items to creator report rows on brand, year, month,
and marketplace. A row is printed when the invoice commission and the
summed report commission differ by more than --tolerance, or when the
invoice status disagrees with the status stored on those report rows.`,
		Example: stringsTrimExample(`
  levanta-pp-cli earnings reconcile --agent --select brand,status,invoice_commission,report_commission
  levanta-pp-cli earnings reconcile --tolerance 1`),
		Annotations: map[string]string{
			"mcp:read-only":  "true",
			"pp:data-source": "local",
			"pp:happy-args":  "--limit=10",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "earnings reconcile")
			}
			if tolerance < 0 {
				return usageErr(fmt.Errorf("--tolerance must be zero or greater"))
			}
			if limit < 0 {
				return usageErr(fmt.Errorf("--limit must be zero or greater"))
			}
			loaded, db, err := loadMany(cmd.Context(), ledgerInvoices, ledgerReports)
			if err != nil {
				return fmt.Errorf("earnings reconcile: %w", err)
			}
			defer closeLedger(db)
			noteUnsynced(cmd, db, ledgerInvoices, flags.maxAge)
			rows := applyLimit(reconcileInvoices(loaded[ledgerInvoices], loaded[ledgerReports], tolerance), limit)
			if rows == nil {
				rows = []reconcileRow{}
			}
			return emitLedger(cmd, flags, rows, "no invoice and report mismatches in the local store")
		},
	}
	cmd.Flags().Float64Var(&tolerance, "tolerance", 0.01, "Absolute commission difference that still counts as a match")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum rows to print (0 means no cap)")
	return cmd
}
