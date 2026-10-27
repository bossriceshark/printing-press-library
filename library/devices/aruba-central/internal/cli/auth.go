// Copyright 2026 Matt Rice and contributors. Licensed under Apache-2.0. See LICENSE.
package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/mvanhorn/printing-press-library/library/devices/aruba-central/internal/central"
	"github.com/mvanhorn/printing-press-library/library/devices/aruba-central/internal/client"
	"github.com/spf13/cobra"
)

func newAuthCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "auth", Short: "Mint and inspect the OAuth2 client-credentials token"}
	cmd.AddCommand(newAuthLoginCmd(flags))
	cmd.AddCommand(newAuthStatusCmd(flags))
	cmd.AddCommand(newAuthRefreshCmd(flags))
	return cmd
}

func newAuthLoginCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Mint a Central access token and cache it",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mint(cmd, flags, false)
		},
	}
}

func newAuthStatusCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show token cache TTL without printing the token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := flags.loadConfig()
			if err != nil {
				return err
			}
			if miss := cfg.Missing(); len(miss) > 0 {
				return doctorFail(cmd, miss)
			}
			src := &client.TokenSource{ClientID: cfg.ClientID, Secret: cfg.ClientSecret, TokenURL: cfg.TokenURL, CachePath: cfg.TokenPath}
			exp, ok := src.Cached()
			body := map[string]any{
				"ok":            ok && exp.After(time.Now()),
				"token_cached":  ok,
				"expires_at":    "",
				"ttl_seconds":   0,
				"base_url_host": hostOf(cfg.BaseURL),
			}
			if ok {
				body["expires_at"] = exp.UTC().Format(time.RFC3339)
				body["ttl_seconds"] = int(time.Until(exp).Seconds())
			}
			return flags.print(cmd, body)
		},
	}
}

func newAuthRefreshCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Force a new client-credentials token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mint(cmd, flags, true)
		},
	}
}

func mint(cmd *cobra.Command, flags *rootFlags, force bool) error {
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
	var exp time.Time
	if force {
		_, exp, err = c.Tokens.Force(cmd.Context())
	} else {
		_, exp, err = c.Tokens.Token(cmd.Context())
	}
	if err != nil {
		return codeErr(4, err)
	}
	return flags.print(cmd, map[string]any{
		"ok":          true,
		"expires_at":  exp.UTC().Format(time.RFC3339),
		"ttl_seconds": int(time.Until(exp).Seconds()),
		"token_path":  cfg.TokenPath,
	})
}

func newDoctorCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check the three Central env vars, token TTL, and cluster",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := flags.loadConfig()
			if err != nil {
				_ = writeDoctor(cmd, map[string]any{"ok": false, "error": err.Error()})
				return err
			}
			if miss := cfg.Missing(); len(miss) > 0 {
				return doctorFail(cmd, miss)
			}
			if central.IsClassicHost(cfg.BaseURL) {
				_ = writeDoctor(cmd, map[string]any{"ok": false, "error": "Classic Central base URL is not supported", "base_url_host": hostOf(cfg.BaseURL)})
				return codeErr(10, fmt.Errorf("classic central is not supported"))
			}
			body := map[string]any{
				"ok":                  false,
				"base_url_host":       hostOf(cfg.BaseURL),
				"rate_limit_rps":      flags.rateLimit,
				"read_only_default":   true,
				"writes_enabled":      writesEnabled(),
				"db":                  cfg.DBPath,
				"config_file":         cfg.ConfigFile,
				"classic_central":     false,
				"credentials_bundled": false,
			}
			c, err := flags.newClient(cfg)
			if err != nil {
				body["error"] = err.Error()
				_ = writeDoctor(cmd, body)
				return err
			}
			_, exp, err := c.Tokens.Token(cmd.Context())
			if err != nil {
				body["error"] = "token mint failed"
				_ = writeDoctor(cmd, body)
				return codeErr(4, fmt.Errorf("token mint failed"))
			}
			body["token_cached"] = true
			body["token_expires_at"] = exp.UTC().Format(time.RFC3339)
			body["token_ttl_seconds"] = int(time.Until(exp).Seconds())
			probe, err := c.GetJSON(cmd.Context(), "network-monitoring/v1/sites-health", url.Values{"limit": []string{"1"}})
			if err != nil {
				body["error"] = err.Error()
				body["probe"] = "network-monitoring/v1/sites-health"
				_ = writeDoctor(cmd, body)
				return codeErr(5, err)
			}
			_ = probe
			body["ok"] = true
			body["probe"] = "network-monitoring/v1/sites-health"
			return writeDoctor(cmd, body)
		},
	}
}

func doctorFail(cmd *cobra.Command, missing []string) error {
	body := map[string]any{
		"ok":      false,
		"error":   "missing credentials",
		"missing": missing,
		"hint":    "Export CENTRAL_BASE_URL, CENTRAL_CLIENT_ID, and CENTRAL_CLIENT_SECRET from a GreenLake Personal API client scoped to the Central instance. You can also set base_url, client_id, and client_secret in ~/.config/aruba-central-pp-cli/config.toml. This CLI ships with no credentials.",
	}
	_ = writeDoctor(cmd, body)
	b, _ := json.Marshal(body)
	return codeErr(10, fmt.Errorf("%s", string(b)))
}

func writeDoctor(cmd *cobra.Command, body map[string]any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(body)
}

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Hostname()
}
