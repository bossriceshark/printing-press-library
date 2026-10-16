package cli

// pp:data-source local

import (
	"strings"

	"github.com/mvanhorn/printing-press-library/library/payments/fidelity/internal/ledger"
	"github.com/spf13/cobra"
)

func newNovelCoverageCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "coverage",
		Short: "Show which accounts have a positions snapshot, which have history, and the run-date span of that history.",
		Example: strings.Trim(`
  fidelity-pp-cli coverage --agent
`, "\n"),
		Annotations: map[string]string{
			"mcp:read-only":       "true",
			"pp:data-source":      "local",
			"pp:typed-exit-codes": "0,2",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "coverage")
			}
			db, err := openLedger(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			report, err := ledger.ReportCoverage(db)
			if err != nil {
				return err
			}
			return flags.printJSON(cmd, report)
		},
	}
	return cmd
}
