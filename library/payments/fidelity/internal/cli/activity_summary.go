package cli

// pp:data-source local

import (
	"strings"

	"github.com/mvanhorn/printing-press-library/library/payments/fidelity/internal/ledger"
	"github.com/spf13/cobra"
)

func newNovelActivitySummaryCmd(flags *rootFlags) *cobra.Command {
	var from string
	var to string
	var action string
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Total imported history by action, including dividends, commissions, and fees",
		Example: strings.Trim(`
  fidelity-pp-cli activity summary --from 2026-01-01 --action DIVIDEND --agent
`, "\n"),
		Annotations: map[string]string{
			"mcp:read-only":       "true",
			"pp:data-source":      "local",
			"pp:happy-args":       "--from=2026-01-01;--action=DIVIDEND",
			"pp:typed-exit-codes": "0,2",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "activity summary")
			}
			db, err := openLedger(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			summary, err := ledger.SummarizeActivity(db, from, to, action)
			if err != nil {
				return usageErr(err)
			}
			return flags.printJSON(cmd, summary)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "Inclusive run-date start, as YYYY-MM-DD or MM/DD/YYYY")
	cmd.Flags().StringVar(&to, "to", "", "Inclusive run-date end, as YYYY-MM-DD or MM/DD/YYYY")
	cmd.Flags().StringVar(&action, "action", "", "Case-insensitive substring matched against the history Action column")
	return cmd
}
