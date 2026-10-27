// Copyright 2026 Matt Rice and contributors. Licensed under Apache-2.0. See LICENSE.
package cli

import (
	"fmt"

	"github.com/mvanhorn/printing-press-library/library/devices/aruba-central/internal/central"
	"github.com/mvanhorn/printing-press-library/library/devices/aruba-central/internal/syncer"
	"github.com/spf13/cobra"
)

func newSyncCmd(flags *rootFlags) *cobra.Command {
	var only, since string
	var maxPages int
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Mirror sites, devices, clients, alerts, and events into SQLite",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := syncer.ParseOnly(only)
			if err != nil {
				return codeErr(2, err)
			}
			window, err := central.ParseWindow(since)
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
			if flags.dryRun {
				if len(names) == 0 {
					names = syncer.DefaultResources
				}
				plan := []syncer.Result{}
				for _, n := range names {
					plan = append(plan, syncer.Result{Resource: n, DryRun: true, Requests: 1})
				}
				return flags.print(cmd, wrap("dry-run", "no requests sent", map[string]any{"db": cfg.DBPath, "resources": plan, "since": since}))
			}
			if miss := cfg.Missing(); len(miss) > 0 {
				return doctorFail(cmd, miss)
			}
			c, err := flags.newClient(cfg)
			if err != nil {
				return err
			}
			results, err := syncer.Run(cmd.Context(), c, st, syncer.Options{Only: names, Since: window, MaxPages: maxPages})
			if err != nil {
				return codeErr(5, err)
			}
			failed := 0
			for _, r := range results {
				if r.Error != "" {
					failed++
				}
			}
			body := wrap("live", "sync writes the local mirror", map[string]any{"db": cfg.DBPath, "resources": results, "failed": failed})
			if err := flags.print(cmd, body); err != nil {
				return err
			}
			if failed > 0 {
				return codeErr(5, fmt.Errorf("%d resource syncs failed", failed))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&only, "only", "", "Comma-separated resources: sites,devices,clients,alerts,events,firmware,subscriptions,wlans,audits")
	cmd.Flags().StringVar(&since, "since", "24h", "Window for events and audits")
	cmd.Flags().IntVar(&maxPages, "max-pages", 200, "Page cap per resource")
	return cmd
}
