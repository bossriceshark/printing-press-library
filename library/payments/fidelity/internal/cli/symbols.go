// Copyright 2026 Cursor Agent and contributors. Licensed under Apache-2.0. See LICENSE.
// Novel command scaffold. Implement the RunE body before shipping.
// generate --force preserves implemented bodies; untouched TODO scaffolds may refresh.
// pp:data-source auto
// Supported strategies: auto, local, live, or computed. Change this default deliberately.

package cli

// pp:data-source local

import (
	"github.com/spf13/cobra"
)

func newNovelSymbolsCmd(flags *rootFlags) *cobra.Command {

	cmd := &cobra.Command{
		Use:         "symbols",
		Short:       "Work with symbols",
		Example:     "  fidelity-pp-cli symbols total VTI --agent",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "auto", "pp:typed-exit-codes": "0,2"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	addNovelCommandIfAbsent(cmd, newNovelSymbolsTotalCmd(flags))
	return cmd
}
