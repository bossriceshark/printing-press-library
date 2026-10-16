package cli

// pp:data-source local

import (
	"strings"

	"github.com/mvanhorn/printing-press-library/library/payments/fidelity/internal/ledger"
	"github.com/spf13/cobra"
)

func newNovelChangesCmd(flags *rootFlags) *cobra.Command {
	var account string
	cmd := &cobra.Command{
		Use:   "changes",
		Short: "Show holdings that appeared, disappeared, or changed quantity after a newer positions file",
		Example: strings.Trim(`
  fidelity-pp-cli changes --account Z1234567 --agent
`, "\n"),
		Annotations: map[string]string{
			"mcp:read-only":       "true",
			"pp:data-source":      "local",
			"pp:happy-args":       "--account=Z1234567",
			"pp:typed-exit-codes": "0,2",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "changes")
			}
			db, err := openLedger(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			diff, err := ledger.DiffChanges(db, account)
			if err != nil {
				return err
			}
			return flags.printJSON(cmd, diff)
		},
	}
	cmd.Flags().StringVar(&account, "account", "", "Limit the comparison to one account number from the positions export")
	return cmd
}
