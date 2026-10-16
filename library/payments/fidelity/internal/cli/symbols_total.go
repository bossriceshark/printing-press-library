package cli

// pp:data-source local

import (
	"fmt"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/payments/fidelity/internal/ledger"
	"github.com/spf13/cobra"
)

func newNovelSymbolsTotalCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "total [symbol]",
		Short: "Sum one symbol's shares, market value, cost basis, and gain across brokerage and retirement accounts.",
		Example: strings.Trim(`
  fidelity-pp-cli symbols total VTI --agent
`, "\n"),
		Annotations: map[string]string{
			"mcp:read-only":       "true",
			"pp:data-source":      "local",
			"pp:happy-args":       "symbol=VTI",
			"pp:typed-exit-codes": "0,2",
			// PATCH: an unknown ticker is an empty total, not a bad argument.
			"pp:no-error-path-probe": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 && !requiredInputWaived() {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "symbols total")
			}
			symbol := ""
			if len(args) > 0 {
				symbol = args[0]
			}
			if symbol == "" {
				if requiredInputWaived() {
					symbol = "VTI"
				} else {
					return usageErr(fmt.Errorf("a symbol is required, for example: fidelity-pp-cli symbols total VTI"))
				}
			}
			if len(args) > 1 {
				return usageErr(fmt.Errorf("symbols total accepts one symbol, for example: fidelity-pp-cli symbols total VTI"))
			}
			db, err := openLedger(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			total, err := ledger.TotalSymbol(db, symbol)
			if err != nil {
				return err
			}
			return flags.printJSON(cmd, total)
		},
	}
	return cmd
}
