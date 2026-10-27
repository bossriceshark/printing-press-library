// Copyright 2026 Cursor Agent. Licensed under Apache-2.0. See LICENSE.
package cli

import (
	"github.com/mvanhorn/printing-press-library/library/devices/aruba-central/internal/central"
	"github.com/spf13/cobra"
)

func newBriefCmd(flags *rootFlags) *cobra.Command {
	var site, within string
	var worst int
	cmd := &cobra.Command{
		Use:   "brief",
		Short: "Down devices, critical alerts, firmware drift, expiring subscriptions, and worst sites",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			window, err := central.ParseWindow(within)
			if err != nil {
				return codeErr(2, err)
			}
			cfg, err := flags.loadConfig()
			if err != nil {
				return err
			}
			st, err := flags.openStore(cfg)
			if err != nil {
				return err
			}
			defer st.Close()
			b, err := st.Brief(site, window, worst)
			if err != nil {
				return err
			}
			return flags.print(cmd, b)
		},
	}
	cmd.Flags().StringVar(&site, "site", "", "Limit the brief to one site name or id")
	cmd.Flags().StringVar(&within, "within", "90d", "Subscription expiry window")
	cmd.Flags().IntVar(&worst, "worst", 5, "How many worst sites to include")
	return cmd
}
