// Copyright 2026 Cursor Agent and contributors. Licensed under Apache-2.0. See LICENSE.
// PATCH: local unpaid invoice ledger. pp:data-source local

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newNovelUnpaidCmd(flags *rootFlags) *cobra.Command {
	var status string
	var marketplace string
	var limit int

	cmd := &cobra.Command{
		Use:   "unpaid",
		Short: "List synced invoice items that are still pending or late.",
		Long: `Reads invoice items already stored by sync and keeps rows whose status
is pending or late. The Levanta invoice endpoint requires a year and has
no status filter, so the open payout list only exists after the pages are
local. Pass --status paid to inspect settled rows instead.`,
		Example: stringsTrimExample(`
  levanta-pp-cli unpaid --agent --select status,brand,commission,year,month
  levanta-pp-cli unpaid --status late --marketplace amazon.com`),
		Annotations: map[string]string{
			"mcp:read-only":  "true",
			"pp:data-source": "local",
			"pp:happy-args":  "--limit=10",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "unpaid")
			}
			if !validLedgerStatus(status) {
				return usageErr(fmt.Errorf("--status must be pending, late, paid, or a comma-separated list of those values"))
			}
			if limit < 0 {
				return usageErr(fmt.Errorf("--limit must be zero or greater"))
			}
			items, db, err := loadResourceMaps(cmd.Context(), ledgerInvoices)
			if err != nil {
				return fmt.Errorf("unpaid: %w", err)
			}
			defer closeLedger(db)
			noteUnsynced(cmd, db, ledgerInvoices, flags.maxAge)
			rows := applyLimit(unpaidFromInvoices(items, status, marketplace), limit)
			if rows == nil {
				rows = []unpaidRow{}
			}
			return emitLedger(cmd, flags, rows, "no unpaid invoice items in the local store")
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "Invoice status to keep: pending, late, paid, or a comma-separated list (default pending and late)")
	cmd.Flags().StringVar(&marketplace, "marketplace", "", "Marketplace to keep, for example amazon.com")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum rows to print (0 means no cap)")
	return cmd
}
