package cli

import (
	"fmt"
	"os"

	"github.com/mvanhorn/printing-press-library/library/payments/fidelity/internal/store"
	"github.com/spf13/cobra"
)

func openLedger(cmd *cobra.Command) (*store.Store, error) {
	db, err := store.OpenWithContext(cmd.Context(), defaultDBPath("fidelity-pp-cli"))
	if err != nil {
		return nil, fmt.Errorf("opening local ledger: %w", err)
	}
	return db, nil
}

func readFileArg(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading --file %s: %w", path, err)
	}
	return raw, nil
}
