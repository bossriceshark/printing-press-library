package cli

import (
	"fmt"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/payments/fidelity/internal/ledger"
	"github.com/spf13/cobra"
)

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		root.AddCommand(newImportCmd(flags))
	})
}

func newImportCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Load official Fidelity CSV exports into the local ledger",
		Example: strings.Trim(`
  fidelity-pp-cli import positions --file ./Portfolio_Positions_Oct-16-2026.csv --dry-run
  fidelity-pp-cli import history --file ./History_for_Account_Z1234567.csv
`, "\n"),
		RunE: parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newImportPositionsCmd(flags))
	cmd.AddCommand(newImportHistoryCmd(flags))
	return cmd
}

func newImportPositionsCmd(flags *rootFlags) *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "positions",
		Short: "Import a Portfolio Positions CSV and skip the disclaimer footer",
		Example: strings.Trim(`
  fidelity-pp-cli import positions --file ./Portfolio_Positions_Oct-16-2026.csv
  fidelity-pp-cli import positions --file ./Portfolio_Positions_Oct-16-2026.csv --dry-run
`, "\n"),
		Annotations: map[string]string{
			"mcp:read-only":       "false",
			"pp:data-source":      "local",
			"pp:typed-exit-codes": "0,2",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if file == "" {
				if dryRunOK(flags) || requiredInputWaived() {
					return writeDryRun(cmd.OutOrStdout(), flags, "import positions")
				}
				return usageErr(fmt.Errorf("--file is required, for example --file ./Portfolio_Positions_Oct-16-2026.csv"))
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "import positions from "+file)
			}
			raw, err := readFileArg(file)
			if err != nil {
				return err
			}
			db, err := openLedger(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			result, err := ledger.ImportPositions(db, file, raw)
			if err != nil {
				return err
			}
			return flags.printJSON(cmd, result)
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "Path to a Portfolio_Positions CSV downloaded by the account owner")
	return cmd
}

func newImportHistoryCmd(flags *rootFlags) *cobra.Command {
	var file string
	var account string
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Import an Activity and Orders history CSV for one account",
		Example: strings.Trim(`
  fidelity-pp-cli import history --file ./History_for_Account_Z1234567.csv
  fidelity-pp-cli import history --file ./activity.csv --account Z1234567
`, "\n"),
		Annotations: map[string]string{
			"mcp:read-only":       "false",
			"pp:data-source":      "local",
			"pp:typed-exit-codes": "0,2",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if file == "" {
				if dryRunOK(flags) || requiredInputWaived() {
					return writeDryRun(cmd.OutOrStdout(), flags, "import history")
				}
				return usageErr(fmt.Errorf("--file is required, for example --file ./History_for_Account_Z1234567.csv"))
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "import history from "+file)
			}
			raw, err := readFileArg(file)
			if err != nil {
				return err
			}
			db, err := openLedger(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			result, err := ledger.ImportHistory(db, file, account, raw)
			if err != nil {
				return err
			}
			return flags.printJSON(cmd, result)
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "Path to a History_for_Account CSV downloaded by the account owner")
	cmd.Flags().StringVar(&account, "account", "", "Account number when the filename does not contain History_for_Account_<id>")
	return cmd
}
