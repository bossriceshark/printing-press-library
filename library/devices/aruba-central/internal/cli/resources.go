// Copyright 2026 Matt Rice and contributors. Licensed under Apache-2.0. See LICENSE.
package cli

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/devices/aruba-central/internal/central"
	"github.com/mvanhorn/printing-press-library/library/devices/aruba-central/internal/client"
	"github.com/spf13/cobra"
)

func newSitesCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "sites", Short: "Sites and site health from the mirror or New Central"}
	cmd.AddCommand(newSitesListCmd(flags))
	cmd.AddCommand(newSitesHealthCmd(flags))
	return cmd
}

func newSitesListCmd(flags *rootFlags) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use: "list", Short: "List sites", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return readRows(cmd, flags, "sites", "", nil, limit, func(c *client.Client) ([]map[string]any, error) {
				items, _, _, err := c.ListPages(cmd.Context(), "network-monitoring/v1/sites-health", "offset", "items", nil, 100, 50)
				return siteRows(items), err
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 100, "Max rows to print")
	return cmd
}

func newSitesHealthCmd(flags *rootFlags) *cobra.Command {
	var worst int
	cmd := &cobra.Command{
		Use: "health", Short: "Sites sorted by health, worst first", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return readRows(cmd, flags, "sites", "1=1 ORDER BY health ASC", nil, worst, func(c *client.Client) ([]map[string]any, error) {
				items, _, _, err := c.ListPages(cmd.Context(), "network-monitoring/v1/sites-health", "offset", "items", nil, 100, 50)
				return siteRows(items), err
			})
		},
	}
	cmd.Flags().IntVar(&worst, "worst", 10, "How many worst sites to show")
	return cmd
}

func siteRows(items []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, m := range items {
		out = append(out, map[string]any{
			"id": central.Str(m, "id", "siteId"), "name": central.Str(m, "siteName", "name"),
			"health": central.HealthScore(m), "device_count": central.NestedCount(m, "devices"),
			"client_count": central.NestedCount(m, "clients"), "critical_alerts": central.CriticalAlerts(m),
		})
	}
	return out
}

func newDevicesCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "devices", Short: "Inventory and status"}
	cmd.AddCommand(newDevicesListCmd(flags))
	cmd.AddCommand(newDevicesGetCmd(flags))
	return cmd
}

func newDevicesListCmd(flags *rootFlags) *cobra.Command {
	var typ, status string
	var limit int
	cmd := &cobra.Command{
		Use: "list", Short: "List devices", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			where, qargs := deviceWhere(typ, status)
			return readRows(cmd, flags, "devices", where, qargs, limit, func(c *client.Client) ([]map[string]any, error) {
				items, _, _, err := c.ListPages(cmd.Context(), "network-monitoring/v1/devices", "next", "items", nil, 100, 50)
				return filterDevices(deviceMaps(items), typ, status), err
			})
		},
	}
	cmd.Flags().StringVar(&typ, "type", "", "ap, switch, or gateway")
	cmd.Flags().StringVar(&status, "status", "", "Status filter, for example down")
	cmd.Flags().IntVar(&limit, "limit", 100, "Max rows")
	return cmd
}

func deviceWhere(typ, status string) (string, []any) {
	var parts []string
	var args []any
	if typ != "" {
		parts = append(parts, "dev_type = ?")
		args = append(args, central.DeviceFamily(typ))
	}
	if strings.EqualFold(status, "down") {
		parts = append(parts, "(lower(status) LIKE '%down%' OR lower(status) LIKE '%offline%')")
	} else if status != "" {
		parts = append(parts, "lower(status) = lower(?)")
		args = append(args, status)
	}
	return strings.Join(parts, " AND "), args
}

func filterDevices(rows []map[string]any, typ, status string) []map[string]any {
	var out []map[string]any
	for _, row := range rows {
		if typ != "" && fmt.Sprint(row["dev_type"]) != central.DeviceFamily(typ) {
			continue
		}
		st := fmt.Sprint(row["status"])
		if strings.EqualFold(status, "down") && !central.IsDown(st) {
			continue
		}
		if status != "" && !strings.EqualFold(status, "down") && !strings.EqualFold(st, status) {
			continue
		}
		out = append(out, row)
	}
	return out
}

func deviceMaps(items []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, m := range items {
		out = append(out, map[string]any{
			"name": central.Str(m, "deviceName", "name"), "serial": central.Str(m, "serialNumber", "serial"),
			"mac": central.Str(m, "macAddress", "mac"), "dev_type": central.DeviceFamily(central.Str(m, "deviceType", "type")),
			"status": central.Str(m, "status", "deviceStatus"), "site_name": central.Str(m, "siteName", "site_name"),
			"model": central.Str(m, "model"), "firmware": central.Str(m, "firmwareVersion", "firmware"),
		})
	}
	return out
}

func newDevicesGetCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use: "get <name|serial|mac>", Short: "Look up one device by name, serial, or MAC", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := flags.loadConfig()
			if err != nil {
				return err
			}
			st, err := flags.openStore(cfg)
			if err != nil {
				return err
			}
			defer st.Close()
			rows, err := st.FindDevice(args[0])
			if err != nil {
				return err
			}
			if len(rows) == 0 {
				return codeErr(3, fmt.Errorf("device %q not found in the local mirror; run sync first", args[0]))
			}
			return flags.print(cmd, wrap("local", "fuzzy name, serial, or mac", rows))
		},
	}
}

func newClientsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "clients", Short: "Wireless and wired clients"}
	cmd.AddCommand(newClientsListCmd(flags))
	cmd.AddCommand(newClientsGetCmd(flags))
	cmd.AddCommand(newClientsTopCmd(flags))
	return cmd
}

func newClientsListCmd(flags *rootFlags) *cobra.Command {
	var site string
	var failed bool
	var limit int
	cmd := &cobra.Command{
		Use: "list", Short: "List clients", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			where, qargs := clientWhere(site, failed)
			return readRows(cmd, flags, "clients", where, qargs, limit, func(c *client.Client) ([]map[string]any, error) {
				items, _, _, err := c.ListPages(cmd.Context(), "network-monitoring/v1/clients", "next", "items", nil, 100, 50)
				return filterClientMaps(items, site, failed), err
			})
		},
	}
	cmd.Flags().StringVar(&site, "site", "", "Site name or id")
	cmd.Flags().BoolVar(&failed, "failed", false, "Only failed onboarding clients")
	cmd.Flags().IntVar(&limit, "limit", 100, "Max rows")
	return cmd
}

func clientWhere(site string, failed bool) (string, []any) {
	var parts []string
	var args []any
	if site != "" {
		parts = append(parts, "(lower(site_id) = lower(?) OR lower(site_name) = lower(?) OR lower(site_name) LIKE ?)")
		args = append(args, site, site, "%"+strings.ToLower(site)+"%")
	}
	if failed {
		parts = append(parts, "failed = 1")
	}
	return strings.Join(parts, " AND "), args
}

func filterClientMaps(items []map[string]any, site string, failed bool) []map[string]any {
	var out []map[string]any
	for _, m := range items {
		status := central.Str(m, "status", "connectionStatus")
		if failed && !central.IsFailedClient(status) {
			continue
		}
		siteName := central.Str(m, "siteName", "site_name")
		if site != "" && !strings.EqualFold(siteName, site) && !strings.Contains(strings.ToLower(siteName), strings.ToLower(site)) && central.Str(m, "siteId") != site {
			continue
		}
		out = append(out, map[string]any{"mac": central.Str(m, "macAddress", "mac"), "username": central.Str(m, "username", "name"), "site_name": siteName, "status": status})
	}
	return out
}

func newClientsGetCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use: "get <mac|user>", Short: "Look up a client by MAC or username", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := flags.loadConfig()
			if err != nil {
				return err
			}
			st, err := flags.openStore(cfg)
			if err != nil {
				return err
			}
			defer st.Close()
			rows, err := st.FindClient(args[0])
			if err != nil {
				return err
			}
			if len(rows) == 0 {
				return codeErr(3, fmt.Errorf("client %q not found in the local mirror; run sync first", args[0]))
			}
			return flags.print(cmd, wrap("local", "mac or username", rows))
		},
	}
}

func newClientsTopCmd(flags *rootFlags) *cobra.Command {
	var by string
	var limit int
	cmd := &cobra.Command{
		Use: "top", Short: "Top clients by usage from the local mirror", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if by != "usage" {
				return codeErr(2, fmt.Errorf("--by must be usage"))
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
			rows, err := st.DB.Query(`SELECT mac, username, site_name, status, usage FROM clients ORDER BY usage DESC LIMIT ?`, limit)
			if err != nil {
				return err
			}
			defer rows.Close()
			mapped, err := scanSQL(rows)
			if err != nil {
				return err
			}
			return flags.print(cmd, wrap("local", "ordered by usage", mapped))
		},
	}
	cmd.Flags().StringVar(&by, "by", "usage", "Rank clients by usage")
	cmd.Flags().IntVar(&limit, "limit", 10, "How many clients")
	return cmd
}

func newAlertsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "alerts", Short: "Alerts and insights"}
	cmd.AddCommand(newAlertsListCmd(flags))
	cmd.AddCommand(newAlertsInsightsCmd(flags))
	return cmd
}

func newAlertsListCmd(flags *rootFlags) *cobra.Command {
	var severity string
	var active bool
	var limit int
	cmd := &cobra.Command{
		Use: "list", Short: "List alerts", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var parts []string
			var qargs []any
			if severity != "" {
				parts = append(parts, "lower(severity) = lower(?)")
				qargs = append(qargs, severity)
			}
			if active {
				parts = append(parts, "active = 1")
			}
			return readRows(cmd, flags, "alerts", strings.Join(parts, " AND "), qargs, limit, func(c *client.Client) ([]map[string]any, error) {
				return listPages(c, cmd, "network-notifications/v1/alerts", "next", "items")
			})
		},
	}
	cmd.Flags().StringVar(&severity, "severity", "", "Filter by severity, for example critical")
	cmd.Flags().BoolVar(&active, "active", false, "Only alerts that are not cleared")
	cmd.Flags().IntVar(&limit, "limit", 100, "Max rows")
	return cmd
}

func newAlertsInsightsCmd(flags *rootFlags) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use: "insights", Short: "Notification insights", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return readRows(cmd, flags, "insights", "", nil, limit, func(c *client.Client) ([]map[string]any, error) {
				return listPages(c, cmd, "network-notifications/v1/insights", "next", "items")
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "Max rows")
	return cmd
}

func newEventsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "events", Short: "Network events"}
	cmd.AddCommand(newEventsListCmd(flags))
	return cmd
}

func newEventsListCmd(flags *rootFlags) *cobra.Command {
	var device, since string
	var limit int
	cmd := &cobra.Command{
		Use: "list", Short: "List events for a device since a window", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := central.ParseWindow(since); err != nil {
				return codeErr(2, err)
			}
			where := ""
			var qargs []any
			if device != "" {
				where = "(lower(device) = lower(?) OR lower(device) LIKE ?)"
				qargs = []any{device, "%" + strings.ToLower(device) + "%"}
			}
			return readRows(cmd, flags, "events", where, qargs, limit, func(c *client.Client) ([]map[string]any, error) {
				return nil, fmt.Errorf("live events need a site id; run sync, then query with --data-source local")
			})
		},
	}
	cmd.Flags().StringVar(&device, "device", "", "Device name, serial, or MAC")
	cmd.Flags().StringVar(&since, "since", "6h", "Lookback window such as 6h")
	cmd.Flags().IntVar(&limit, "limit", 100, "Max rows")
	return cmd
}

func newFirmwareCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "firmware", Short: "Running versus recommended firmware"}
	cmd.AddCommand(newFirmwareStatusCmd(flags))
	return cmd
}

func newFirmwareStatusCmd(flags *rootFlags) *cobra.Command {
	var drift bool
	var limit int
	cmd := &cobra.Command{
		Use: "status", Short: "Firmware status", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			where := ""
			if drift {
				where = "drift = 1"
			}
			return readRows(cmd, flags, "firmware", where, nil, limit, func(c *client.Client) ([]map[string]any, error) {
				return listPages(c, cmd, "network-services/v1/firmware-details", "next", "items")
			})
		},
	}
	cmd.Flags().BoolVar(&drift, "drift", false, "Only rows whose running version differs from recommended")
	cmd.Flags().IntVar(&limit, "limit", 100, "Max rows")
	return cmd
}

func newSubscriptionsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "subscriptions", Short: "Subscription expiry from New Central monitoring"}
	cmd.AddCommand(newSubscriptionsExpiringCmd(flags))
	return cmd
}

func newSubscriptionsExpiringCmd(flags *rootFlags) *cobra.Command {
	var within string
	var limit int
	cmd := &cobra.Command{
		Use: "expiring", Short: "Subscriptions expiring inside a window", Args: cobra.NoArgs,
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
			b, err := st.Brief("", window, 5)
			if err != nil {
				return err
			}
			rows := b.ExpiringSubscriptions
			if limit > 0 && len(rows) > limit {
				rows = rows[:limit]
			}
			return flags.print(cmd, wrap("local", "parsed expiry within "+within, rows))
		},
	}
	cmd.Flags().StringVar(&within, "within", "90d", "Expiry window such as 90d")
	cmd.Flags().IntVar(&limit, "limit", 100, "Max rows")
	return cmd
}

func newWlansCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "wlans", Short: "WLANs"}
	cmd.AddCommand(newWlansListCmd(flags))
	cmd.AddCommand(newWlansStatsCmd(flags))
	return cmd
}

func newWlansListCmd(flags *rootFlags) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use: "list", Short: "List WLANs", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return readRows(cmd, flags, "wlans", "", nil, limit, func(c *client.Client) ([]map[string]any, error) {
				return listPages(c, cmd, "network-monitoring/v1/wlans", "next", "items")
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 100, "Max rows")
	return cmd
}

func newWlansStatsCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use: "stats", Short: "WLAN counts by band and status from the local mirror", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := flags.loadConfig()
			if err != nil {
				return err
			}
			st, err := flags.openStore(cfg)
			if err != nil {
				return err
			}
			defer st.Close()
			rows, err := st.Query("wlans", "")
			if err != nil {
				return err
			}
			byBand, byStatus := map[string]int{}, map[string]int{}
			for _, row := range rows {
				byBand[fmt.Sprint(row["band"])]++
				byStatus[fmt.Sprint(row["status"])]++
			}
			return flags.print(cmd, wrap("local", "derived from mirrored wlans", map[string]any{"count": len(rows), "by_band": byBand, "by_status": byStatus}))
		},
	}
}

func newAuditCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "audit", Short: "Central audit log"}
	cmd.AddCommand(newAuditListCmd(flags))
	return cmd
}

func newAuditListCmd(flags *rootFlags) *cobra.Command {
	var since string
	var limit int
	cmd := &cobra.Command{
		Use: "list", Short: "List audit records", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := central.ParseWindow(since); err != nil {
				return codeErr(2, err)
			}
			return readRows(cmd, flags, "audits", "", nil, limit, func(c *client.Client) ([]map[string]any, error) {
				return listPages(c, cmd, "network-services/v1/audits", "next", "items")
			})
		},
	}
	cmd.Flags().StringVar(&since, "since", "7d", "Lookback window such as 7d")
	cmd.Flags().IntVar(&limit, "limit", 100, "Max rows")
	return cmd
}

func listPages(c *client.Client, cmd *cobra.Command, path, style, key string) ([]map[string]any, error) {
	items, _, _, err := c.ListPages(cmd.Context(), path, style, key, nil, 100, 20)
	return items, err
}

func readRows(cmd *cobra.Command, flags *rootFlags, table, where string, args []any, limit int, live func(*client.Client) ([]map[string]any, error)) error {
	cfg, err := flags.loadConfig()
	if err != nil {
		return err
	}
	source := flags.dataSource
	st, err := flags.openStore(cfg)
	if err != nil {
		return err
	}
	defer st.Close()
	if source == "auto" {
		if st.Count(table) > 0 {
			source = "local"
		} else {
			source = "live"
		}
	}
	if source == "local" {
		rows, err := st.Query(table, where, args...)
		if err != nil {
			return err
		}
		if limit > 0 && len(rows) > limit {
			rows = rows[:limit]
		}
		if rows == nil {
			rows = []map[string]any{}
		}
		return flags.print(cmd, wrap("local", "sqlite mirror", rows))
	}
	if flags.dryRun {
		return flags.print(cmd, wrap("dry-run", "live fetch not sent", map[string]any{"table": table}))
	}
	if miss := cfg.Missing(); len(miss) > 0 {
		return doctorFail(cmd, miss)
	}
	c, err := flags.newClient(cfg)
	if err != nil {
		return err
	}
	rows, err := live(c)
	if err != nil {
		return codeErr(5, err)
	}
	if rows == nil {
		rows = []map[string]any{}
	}
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return flags.print(cmd, wrap("live", "New Central", rows))
}

func scanSQL(rows *sql.Rows) ([]map[string]any, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		m := map[string]any{}
		for i, c := range cols {
			switch v := vals[i].(type) {
			case []byte:
				m[c] = string(v)
			default:
				m[c] = v
			}
		}
		out = append(out, m)
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out, rows.Err()
}
