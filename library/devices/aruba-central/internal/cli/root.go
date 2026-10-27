// Copyright 2026 Matt Rice and contributors. Licensed under Apache-2.0. See LICENSE.
// PATCH(hand-curated): New Central MVP written in this catalog because cli-printing-press was not available. This tree is not generator output.

package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/mvanhorn/printing-press-library/library/devices/aruba-central/internal/client"
	"github.com/mvanhorn/printing-press-library/library/devices/aruba-central/internal/config"
	"github.com/mvanhorn/printing-press-library/library/devices/aruba-central/internal/store"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

type rootFlags struct {
	asJSON     bool
	compact    bool
	dryRun     bool
	noInput    bool
	yes        bool
	agent      bool
	configPath string
	dbPath     string
	timeout    time.Duration
	rateLimit  float64
	dataSource string
}

func Execute() error {
	cmd := newRootCmd()
	return cmd.Execute()
}

func newRootCmd() *cobra.Command {
	var flags rootFlags
	rootCmd := &cobra.Command{
		Use:           "aruba-central-pp-cli",
		Short:         "Agent-native CLI for HPE Aruba New Central",
		Long:          "Read New Central into a local SQLite mirror, then answer fleet questions with brief and export. Read-only unless ENABLE_WRITES=1 and --yes. Classic Central is not supported.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}
	rootCmd.SetVersionTemplate("aruba-central-pp-cli {{ .Version }}\n")
	rootCmd.PersistentFlags().BoolVar(&flags.asJSON, "json", false, "Output JSON")
	rootCmd.PersistentFlags().BoolVar(&flags.compact, "compact", false, "Omit raw payloads")
	rootCmd.PersistentFlags().BoolVar(&flags.dryRun, "dry-run", false, "Show the request without sending it")
	rootCmd.PersistentFlags().BoolVar(&flags.noInput, "no-input", false, "Never prompt")
	rootCmd.PersistentFlags().BoolVar(&flags.yes, "yes", false, "Confirm a gated write")
	rootCmd.PersistentFlags().BoolVar(&flags.agent, "agent", false, "JSON output, no prompts, assume --yes for gates that still require ENABLE_WRITES=1")
	rootCmd.PersistentFlags().StringVar(&flags.configPath, "config", "", "TOML config file (base_url, client_id, client_secret)")
	rootCmd.PersistentFlags().StringVar(&flags.dbPath, "db", "", "SQLite mirror path")
	rootCmd.PersistentFlags().DurationVar(&flags.timeout, "timeout", 30*time.Second, "Per-request timeout")
	rootCmd.PersistentFlags().Float64Var(&flags.rateLimit, "rate-limit", 10, "Client-side requests per second (account cap is 10)")
	rootCmd.PersistentFlags().StringVar(&flags.dataSource, "data-source", "auto", "auto, live, or local")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if flags.agent {
			flags.asJSON = true
			flags.noInput = true
			flags.yes = true
		}
		switch flags.dataSource {
		case "auto", "live", "local":
			return nil
		default:
			return codeErr(2, fmt.Errorf("--data-source must be auto, live, or local"))
		}
	}

	rootCmd.AddCommand(newAuthCmd(&flags))
	rootCmd.AddCommand(newDoctorCmd(&flags))
	rootCmd.AddCommand(newSyncCmd(&flags))
	rootCmd.AddCommand(newSitesCmd(&flags))
	rootCmd.AddCommand(newDevicesCmd(&flags))
	rootCmd.AddCommand(newClientsCmd(&flags))
	rootCmd.AddCommand(newAlertsCmd(&flags))
	rootCmd.AddCommand(newEventsCmd(&flags))
	rootCmd.AddCommand(newFirmwareCmd(&flags))
	rootCmd.AddCommand(newSubscriptionsCmd(&flags))
	rootCmd.AddCommand(newWlansCmd(&flags))
	rootCmd.AddCommand(newAuditCmd(&flags))
	rootCmd.AddCommand(newBriefCmd(&flags))
	rootCmd.AddCommand(newExportCmd(&flags))
	rootCmd.AddCommand(newAPICmd(&flags))
	rootCmd.AddCommand(newDeviceCmd(&flags))
	rootCmd.AddCommand(newWhichCmd(&flags))
	return rootCmd
}

func (f *rootFlags) loadConfig() (config.Config, error) {
	cfg, err := config.Load(f.configPath)
	if err != nil {
		return cfg, codeErr(10, err)
	}
	if f.dbPath != "" {
		cfg.DBPath = f.dbPath
	}
	return cfg, nil
}

func (f *rootFlags) openStore(cfg config.Config) (*store.Store, error) {
	st, err := store.Open(cfg.DBPath)
	if err != nil {
		return nil, codeErr(10, err)
	}
	return st, nil
}

func (f *rootFlags) newClient(cfg config.Config) (*client.Client, error) {
	if err := cfg.Require(); err != nil {
		return nil, codeErr(10, err)
	}
	src := &client.TokenSource{
		ClientID:  cfg.ClientID,
		Secret:    cfg.ClientSecret,
		TokenURL:  cfg.TokenURL,
		CachePath: cfg.TokenPath,
	}
	c := client.New(cfg.BaseURL, src, f.rateLimit, f.timeout)
	c.DryRun = false
	return c, nil
}

func writesEnabled() bool { return os.Getenv("ENABLE_WRITES") == "1" }
