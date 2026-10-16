// Copyright 2026 Cursor Agent and contributors. Licensed under Apache-2.0. See LICENSE.
// Novel command scaffold. Implement the RunE body before shipping.
// generate --force preserves implemented bodies; untouched TODO scaffolds may refresh.
// pp:data-source auto
// Supported strategies: auto, local, live, or computed. Change this default deliberately.

package cli

import (
	"github.com/spf13/cobra"
)

func newNovelPayoutCmd(flags *rootFlags) *cobra.Command {

	cmd := &cobra.Command{
		Use:         "payout",
		Short:       "Work with payout",
		Example:     "  levanta-pp-cli payout gap --agent --select brand,year,month,marketplace,commission",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "auto", "pp:typed-exit-codes": "0,2"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	addNovelCommandIfAbsent(cmd, newNovelPayoutGapCmd(flags))
	return cmd
}
