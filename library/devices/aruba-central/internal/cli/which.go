// Copyright 2026 Matt Rice and contributors. Licensed under Apache-2.0. See LICENSE.
package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newWhichCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "which <query>",
		Short: "Find a command from a short capability description",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			q := strings.ToLower(args[0])
			type hit struct {
				Command string `json:"command"`
				Why     string `json:"why"`
			}
			catalog := []hit{
				{"doctor", "credentials token cluster"},
				{"sync", "sqlite mirror cache sites devices clients alerts events"},
				{"brief", "down devices critical alerts drift subscriptions worst sites"},
				{"export", "dump json csv mirror"},
				{"sites list", "sites inventory"},
				{"sites health", "worst site health"},
				{"devices list", "aps switches gateways offline down"},
				{"devices get", "serial mac name"},
				{"clients list", "failed clients"},
				{"clients top", "usage"},
				{"alerts list", "critical active alerts"},
				{"events list", "events device"},
				{"firmware status", "firmware drift"},
				{"subscriptions expiring", "expiring subscriptions"},
				{"api", "raw request"},
			}
			var matches []hit
			for _, item := range catalog {
				if strings.Contains(item.Why, q) || strings.Contains(item.Command, q) || strings.Contains(q, item.Command) {
					matches = append(matches, item)
				}
			}
			if len(matches) == 0 {
				_ = flags.print(cmd, map[string]any{"matches": []hit{}})
				return codeErr(2, fmt.Errorf("no match for %q", args[0]))
			}
			return flags.print(cmd, map[string]any{"matches": matches})
		},
	}
}
