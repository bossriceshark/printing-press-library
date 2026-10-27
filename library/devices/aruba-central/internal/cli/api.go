// Copyright 2026 Matt Rice and contributors. Licensed under Apache-2.0. See LICENSE.
package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newAPICmd(flags *rootFlags) *cobra.Command {
	var data string
	cmd := &cobra.Command{
		Use:   "api <method> <path>",
		Short: "Raw New Central request. Writes stay gated.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			method := strings.ToUpper(args[0])
			path := args[1]
			switch method {
			case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			default:
				return codeErr(2, fmt.Errorf("method %s is not supported", method))
			}
			if strings.Contains(path, "://") || strings.Contains(strings.ToLower(path), "apigw") || strings.Contains(strings.ToLower(path), "/classic/") {
				return codeErr(2, fmt.Errorf("path must be a relative New Central path; Classic Central is not supported"))
			}
			var body any
			if data != "" {
				raw := data
				if strings.HasPrefix(data, "@") {
					b, err := os.ReadFile(strings.TrimPrefix(data, "@"))
					if err != nil {
						return codeErr(2, err)
					}
					raw = string(b)
				}
				if err := json.Unmarshal([]byte(raw), &body); err != nil {
					return codeErr(2, fmt.Errorf("data must be JSON: %w", err))
				}
			}
			if method != http.MethodGet && method != http.MethodHead {
				if flags.dryRun || !writesEnabled() || !flags.yes {
					plan := map[string]any{"dry_run": true, "method": method, "path": path, "body": body, "sent": false}
					if flags.yes && !writesEnabled() && !flags.dryRun {
						_ = flags.print(cmd, plan)
						return codeErr(10, fmt.Errorf("ENABLE_WRITES=1 is required before a write is sent"))
					}
					return flags.print(cmd, plan)
				}
			}
			if flags.dryRun {
				return flags.print(cmd, map[string]any{"dry_run": true, "method": method, "path": path, "sent": false})
			}
			cfg, err := flags.loadConfig()
			if err != nil {
				return err
			}
			if miss := cfg.Missing(); len(miss) > 0 {
				return doctorFail(cmd, miss)
			}
			c, err := flags.newClient(cfg)
			if err != nil {
				return err
			}
			out, err := c.DoJSON(cmd.Context(), method, path, nil, body)
			if err != nil {
				return codeErr(5, err)
			}
			return flags.print(cmd, wrap("live", method+" "+path, out))
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "JSON body, or @file.json")
	return cmd
}

func newDeviceCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "device", Short: "Gated device actions. Dry-run unless ENABLE_WRITES=1 and --yes."}
	cmd.AddCommand(newDeviceBouncePortCmd(flags))
	return cmd
}

func newDeviceBouncePortCmd(flags *rootFlags) *cobra.Command {
	var serial, kind, ports string
	cmd := &cobra.Command{
		Use:   "bounce-port",
		Short: "Plan or send a port bounce. Not sent unless writes are enabled.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			family := "cx"
			switch strings.ToLower(kind) {
			case "ap":
				family = "aps"
			case "gateway":
				family = "gateways"
			case "aos-s":
				family = "aos-s"
			case "switch", "cx", "":
				family = "cx"
			default:
				return codeErr(2, fmt.Errorf("--type must be ap, switch, gateway, cx, or aos-s"))
			}
			path := fmt.Sprintf("network-troubleshooting/v1/%s/%s/portBounce", family, serial)
			body := map[string]any{"ports": strings.Split(ports, ",")}
			if flags.dryRun || !writesEnabled() || !flags.yes {
				plan := map[string]any{"dry_run": true, "method": "POST", "path": path, "body": body, "sent": false}
				if flags.yes && !writesEnabled() && !flags.dryRun {
					_ = flags.print(cmd, plan)
					return codeErr(10, fmt.Errorf("ENABLE_WRITES=1 is required before bounce-port is sent"))
				}
				return flags.print(cmd, plan)
			}
			cfg, err := flags.loadConfig()
			if err != nil {
				return err
			}
			c, err := flags.newClient(cfg)
			if err != nil {
				return err
			}
			out, err := c.DoJSON(cmd.Context(), http.MethodPost, path, nil, body)
			if err != nil {
				return codeErr(5, err)
			}
			return flags.print(cmd, wrap("live", "port bounce sent", out))
		},
	}
	cmd.Flags().StringVar(&serial, "serial", "", "Device serial")
	cmd.Flags().StringVar(&kind, "type", "switch", "ap, switch, gateway, cx, or aos-s")
	cmd.Flags().StringVar(&ports, "ports", "", "Comma-separated port ids")
	_ = cmd.MarkFlagRequired("serial")
	_ = cmd.MarkFlagRequired("ports")
	return cmd
}
