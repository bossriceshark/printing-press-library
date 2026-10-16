package cli

// pp:data-source local

import (
	"strings"

	"github.com/mvanhorn/printing-press-library/library/payments/fidelity/internal/ledger"
	"github.com/spf13/cobra"
)

func newNovelPortfolioSummaryCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "See per-account and whole-book market value, cash versus invested, unrealized gain",
		Example: strings.Trim(`
  fidelity-pp-cli portfolio summary --agent
  fidelity-pp-cli portfolio summary --agent --select book.market_value,accounts.account_number,accounts.market_value
`, "\n"),
		Annotations: map[string]string{
			"mcp:read-only":       "true",
			"pp:data-source":      "local",
			"pp:typed-exit-codes": "0,2",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "portfolio summary")
			}
			db, err := openLedger(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			summary, err := ledger.SummarizePortfolio(db)
			if err != nil {
				return err
			}
			return flags.printJSON(cmd, summary)
		},
	}
	return cmd
}
