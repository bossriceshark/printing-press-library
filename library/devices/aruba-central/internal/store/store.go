// Copyright 2026 Matt Rice and contributors. Licensed under Apache-2.0. See LICENSE.
package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/devices/aruba-central/internal/central"
	_ "modernc.org/sqlite"
)

// Store is the local SQLite mirror agents query instead of spending the 10 req/s budget.
type Store struct {
	DB   *sql.DB
	Path string
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &Store{DB: db, Path: path}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.DB.Close() }

func (s *Store) migrate() error {
	_, err := s.DB.Exec(`
CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS sites (
  id TEXT PRIMARY KEY, name TEXT, health REAL, device_count INTEGER, client_count INTEGER,
  critical_alerts INTEGER, raw TEXT, synced_at TEXT);
CREATE TABLE IF NOT EXISTS devices (
  id TEXT PRIMARY KEY, name TEXT, serial TEXT, mac TEXT, mac_norm TEXT, dev_type TEXT,
  status TEXT, site_id TEXT, site_name TEXT, model TEXT, firmware TEXT, raw TEXT, synced_at TEXT);
CREATE TABLE IF NOT EXISTS clients (
  mac TEXT PRIMARY KEY, mac_norm TEXT, username TEXT, site_id TEXT, site_name TEXT,
  status TEXT, usage REAL, failed INTEGER, raw TEXT, synced_at TEXT);
CREATE TABLE IF NOT EXISTS alerts (
  id TEXT PRIMARY KEY, severity TEXT, status TEXT, active INTEGER, site_id TEXT, site_name TEXT,
  device TEXT, summary TEXT, raw TEXT, synced_at TEXT);
CREATE TABLE IF NOT EXISTS events (
  id TEXT PRIMARY KEY, device TEXT, site_id TEXT, severity TEXT, ts TEXT, name TEXT, raw TEXT, synced_at TEXT);
CREATE TABLE IF NOT EXISTS firmware (
  serial TEXT PRIMARY KEY, model TEXT, running TEXT, recommended TEXT, drift INTEGER, raw TEXT, synced_at TEXT);
CREATE TABLE IF NOT EXISTS subscriptions (
  id TEXT PRIMARY KEY, name TEXT, expires_at TEXT, raw TEXT, synced_at TEXT);
CREATE TABLE IF NOT EXISTS wlans (
  id TEXT PRIMARY KEY, name TEXT, band TEXT, status TEXT, raw TEXT, synced_at TEXT);
CREATE TABLE IF NOT EXISTS audits (
  id TEXT PRIMARY KEY, ts TEXT, actor TEXT, action TEXT, raw TEXT, synced_at TEXT);
CREATE TABLE IF NOT EXISTS insights (
  id TEXT PRIMARY KEY, severity TEXT, summary TEXT, site_name TEXT, raw TEXT, synced_at TEXT);
`)
	return err
}

func (s *Store) SetMeta(key, value string) error {
	_, err := s.DB.Exec(`INSERT INTO meta(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

func (s *Store) Meta(key string) string {
	var v string
	_ = s.DB.QueryRow(`SELECT value FROM meta WHERE key=?`, key).Scan(&v)
	return v
}

func (s *Store) replace(table string, cols []string, rows [][]any) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec("DELETE FROM " + table); err != nil {
		return err
	}
	if len(rows) > 0 {
		ph := strings.Repeat("?,", len(cols))
		ph = ph[:len(ph)-1]
		stmt, err := tx.Prepare("INSERT INTO " + table + "(" + strings.Join(cols, ",") + ") VALUES (" + ph + ")")
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, row := range rows {
			if _, err := stmt.Exec(row...); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func rawOf(m map[string]any) string {
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

func (s *Store) ReplaceSites(items []map[string]any) error {
	ts := now()
	rows := make([][]any, 0, len(items))
	seen := map[string]struct{}{}
	for i, m := range items {
		id := central.Str(m, "id", "siteId", "site_id")
		name := central.Str(m, "siteName", "name", "site_name")
		if id == "" {
			id = name
		}
		if id == "" {
			id = fmt.Sprintf("site-%d", i)
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		rows = append(rows, []any{id, name, central.HealthScore(m), central.NestedCount(m, "devices", "deviceCount"), central.NestedCount(m, "clients", "clientCount"), central.CriticalAlerts(m), rawOf(m), ts})
	}
	if err := s.replace("sites", []string{"id", "name", "health", "device_count", "client_count", "critical_alerts", "raw", "synced_at"}, rows); err != nil {
		return err
	}
	return s.SetMeta("synced_at_sites", ts)
}

func (s *Store) ReplaceDevices(items []map[string]any) error {
	ts := now()
	rows := make([][]any, 0, len(items))
	seen := map[string]struct{}{}
	for i, m := range items {
		serial := central.Str(m, "serialNumber", "serial", "serial_number")
		mac := central.Str(m, "macAddress", "macaddr", "mac", "deviceMac")
		name := central.Str(m, "deviceName", "name", "hostName", "hostname")
		id := serial
		if id == "" {
			id = mac
		}
		if id == "" {
			id = name
		}
		if id == "" {
			id = fmt.Sprintf("device-%d", i)
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		status := central.Str(m, "status", "deviceStatus", "state")
		typ := central.DeviceFamily(central.Str(m, "deviceType", "type", "device_type", "persona", "deviceFunction"))
		rows = append(rows, []any{id, name, serial, mac, central.NormMAC(mac), typ, status, central.Str(m, "siteId", "site_id"), central.Str(m, "siteName", "site_name", "site"), central.Str(m, "model", "deviceModel"), central.Str(m, "firmwareVersion", "firmware", "softwareVersion"), rawOf(m), ts})
	}
	if err := s.replace("devices", []string{"id", "name", "serial", "mac", "mac_norm", "dev_type", "status", "site_id", "site_name", "model", "firmware", "raw", "synced_at"}, rows); err != nil {
		return err
	}
	return s.SetMeta("synced_at_devices", ts)
}

func (s *Store) ReplaceClients(items []map[string]any) error {
	ts := now()
	rows := make([][]any, 0, len(items))
	seen := map[string]struct{}{}
	for i, m := range items {
		mac := central.Str(m, "macAddress", "mac", "clientMac", "macaddr")
		if mac == "" {
			mac = fmt.Sprintf("client-%d", i)
		}
		if _, ok := seen[mac]; ok {
			continue
		}
		seen[mac] = struct{}{}
		status := central.Str(m, "status", "connectionStatus", "clientStatus")
		usage := central.Num(m, "usage", "totalUsage", "bytes", "totalBytes")
		if usage == 0 {
			usage = central.Num(m, "rxBytes") + central.Num(m, "txBytes")
		}
		failed := 0
		if central.IsFailedClient(status) {
			failed = 1
		}
		rows = append(rows, []any{mac, central.NormMAC(mac), central.Str(m, "username", "userName", "name", "hostname"), central.Str(m, "siteId", "site_id"), central.Str(m, "siteName", "site_name"), status, usage, failed, rawOf(m), ts})
	}
	if err := s.replace("clients", []string{"mac", "mac_norm", "username", "site_id", "site_name", "status", "usage", "failed", "raw", "synced_at"}, rows); err != nil {
		return err
	}
	return s.SetMeta("synced_at_clients", ts)
}

func (s *Store) ReplaceAlerts(items []map[string]any) error {
	ts := now()
	rows := make([][]any, 0, len(items))
	seen := map[string]struct{}{}
	for i, m := range items {
		id := central.Str(m, "id", "alertId", "uuid")
		if id == "" {
			id = fmt.Sprintf("alert-%d", i)
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		status := central.Str(m, "status", "state", "alertStatus")
		active := 0
		if central.IsActiveAlert(status) {
			active = 1
		}
		rows = append(rows, []any{id, central.Str(m, "severity", "alertSeverity", "priority"), status, active, central.Str(m, "siteId", "site_id"), central.Str(m, "siteName", "site_name"), central.Str(m, "deviceName", "hostname", "serialNumber", "device"), central.Str(m, "summary", "description", "name", "alertType", "message"), rawOf(m), ts})
	}
	if err := s.replace("alerts", []string{"id", "severity", "status", "active", "site_id", "site_name", "device", "summary", "raw", "synced_at"}, rows); err != nil {
		return err
	}
	return s.SetMeta("synced_at_alerts", ts)
}

func (s *Store) ReplaceEvents(items []map[string]any) error {
	ts := now()
	rows := make([][]any, 0, len(items))
	seen := map[string]struct{}{}
	for i, m := range items {
		id := central.Str(m, "uuid", "id", "eventId")
		when, ok := central.FirstTime(m, "timestamp", "ts", "time", "occurredAt")
		whenS := ""
		if ok {
			whenS = when.Format(time.RFC3339)
		}
		if id == "" {
			id = fmt.Sprintf("event-%d-%s", i, whenS)
		}
		if _, dup := seen[id]; dup {
			id = fmt.Sprintf("%s-%d", id, i)
		}
		seen[id] = struct{}{}
		rows = append(rows, []any{id, central.Str(m, "serialNumber", "deviceName", "hostname", "macAddress", "device"), central.Str(m, "siteId", "site_id"), central.Str(m, "severity"), whenS, central.Str(m, "eventName", "name", "description"), rawOf(m), ts})
	}
	if err := s.replace("events", []string{"id", "device", "site_id", "severity", "ts", "name", "raw", "synced_at"}, rows); err != nil {
		return err
	}
	return s.SetMeta("synced_at_events", ts)
}

func (s *Store) ReplaceFirmware(items []map[string]any) error {
	ts := now()
	rows := make([][]any, 0, len(items))
	seen := map[string]struct{}{}
	for i, m := range items {
		serial := central.Str(m, "serialNumber", "serial", "deviceSerial")
		if serial == "" {
			serial = fmt.Sprintf("fw-%d", i)
		}
		if _, ok := seen[serial]; ok {
			continue
		}
		seen[serial] = struct{}{}
		running := central.Str(m, "firmwareVersion", "runningVersion", "currentVersion", "version")
		rec := central.Str(m, "recommendedVersion", "recommendedFirmware", "recommended")
		drift := 0
		if running != "" && rec != "" && !strings.EqualFold(running, rec) {
			drift = 1
		}
		rows = append(rows, []any{serial, central.Str(m, "model", "deviceModel"), running, rec, drift, rawOf(m), ts})
	}
	if err := s.replace("firmware", []string{"serial", "model", "running", "recommended", "drift", "raw", "synced_at"}, rows); err != nil {
		return err
	}
	return s.SetMeta("synced_at_firmware", ts)
}

func (s *Store) ReplaceSubscriptions(items []map[string]any) error {
	ts := now()
	rows := make([][]any, 0, len(items))
	seen := map[string]struct{}{}
	for i, m := range items {
		id := central.Str(m, "id", "subscriptionKey", "key", "subscriptionId")
		if id == "" {
			id = fmt.Sprintf("sub-%d", i)
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		exp := ""
		if when, ok := central.FirstTime(m, "expirationDate", "expiresAt", "endDate", "expiry", "end_date"); ok {
			exp = when.Format(time.RFC3339)
		}
		rows = append(rows, []any{id, central.Str(m, "name", "subscriptionType", "type", "sku"), exp, rawOf(m), ts})
	}
	if err := s.replace("subscriptions", []string{"id", "name", "expires_at", "raw", "synced_at"}, rows); err != nil {
		return err
	}
	return s.SetMeta("synced_at_subscriptions", ts)
}

func (s *Store) ReplaceWlans(items []map[string]any) error {
	ts := now()
	rows := make([][]any, 0, len(items))
	seen := map[string]struct{}{}
	for i, m := range items {
		id := central.Str(m, "id", "wlanId")
		name := central.Str(m, "wlanName", "ssid", "name")
		if id == "" {
			id = name
		}
		if id == "" {
			id = fmt.Sprintf("wlan-%d", i)
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		rows = append(rows, []any{id, name, central.Str(m, "band"), central.Str(m, "status"), rawOf(m), ts})
	}
	if err := s.replace("wlans", []string{"id", "name", "band", "status", "raw", "synced_at"}, rows); err != nil {
		return err
	}
	return s.SetMeta("synced_at_wlans", ts)
}

func (s *Store) ReplaceAudits(items []map[string]any) error {
	ts := now()
	rows := make([][]any, 0, len(items))
	seen := map[string]struct{}{}
	for i, m := range items {
		id := central.Str(m, "id", "uuid", "auditId")
		when, ok := central.FirstTime(m, "timestamp", "ts", "time", "createdAt")
		whenS := ""
		if ok {
			whenS = when.Format(time.RFC3339)
		}
		if id == "" {
			id = fmt.Sprintf("audit-%d-%s", i, whenS)
		}
		if _, dup := seen[id]; dup {
			id = fmt.Sprintf("%s-%d", id, i)
		}
		seen[id] = struct{}{}
		rows = append(rows, []any{id, whenS, central.Str(m, "user", "actor", "username", "principal"), central.Str(m, "action", "description", "message", "event"), rawOf(m), ts})
	}
	if err := s.replace("audits", []string{"id", "ts", "actor", "action", "raw", "synced_at"}, rows); err != nil {
		return err
	}
	return s.SetMeta("synced_at_audits", ts)
}

func (s *Store) ReplaceInsights(items []map[string]any) error {
	ts := now()
	rows := make([][]any, 0, len(items))
	seen := map[string]struct{}{}
	for i, m := range items {
		id := central.Str(m, "id", "insightId", "uuid")
		if id == "" {
			id = fmt.Sprintf("insight-%d", i)
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		rows = append(rows, []any{id, central.Str(m, "severity"), central.Str(m, "summary", "description", "name", "title"), central.Str(m, "siteName", "site_name"), rawOf(m), ts})
	}
	if err := s.replace("insights", []string{"id", "severity", "summary", "site_name", "raw", "synced_at"}, rows); err != nil {
		return err
	}
	return s.SetMeta("synced_at_insights", ts)
}

func (s *Store) Count(table string) int {
	var n int
	_ = s.DB.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n)
	return n
}

// Query runs a SELECT and returns JSON-friendly objects. table must be a known name.
func (s *Store) Query(table, where string, args ...any) ([]map[string]any, error) {
	if !knownTable(table) {
		return nil, fmt.Errorf("unknown table %s", table)
	}
	q := "SELECT * FROM " + table
	if where != "" {
		q += " WHERE " + where
	}
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMaps(rows)
}

func knownTable(t string) bool {
	switch t {
	case "sites", "devices", "clients", "alerts", "events", "firmware", "subscriptions", "wlans", "audits", "insights", "meta":
		return true
	}
	return false
}

func scanMaps(rows *sql.Rows) ([]map[string]any, error) {
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
	return out, rows.Err()
}

// FindDevice matches serial, MAC, then a unique name substring.
func (s *Store) FindDevice(q string) ([]map[string]any, error) {
	q = strings.TrimSpace(q)
	if rows, err := s.Query("devices", "serial = ? OR id = ?", q, q); err != nil {
		return nil, err
	} else if len(rows) > 0 {
		return rows, nil
	}
	norm := central.NormMAC(q)
	if norm != "" {
		if rows, err := s.Query("devices", "mac_norm = ?", norm); err != nil {
			return nil, err
		} else if len(rows) > 0 {
			return rows, nil
		}
	}
	return s.Query("devices", "lower(name) = lower(?) OR lower(name) LIKE ?", q, "%"+strings.ToLower(q)+"%")
}

func (s *Store) FindClient(q string) ([]map[string]any, error) {
	norm := central.NormMAC(q)
	if norm != "" {
		if rows, err := s.Query("clients", "mac_norm = ?", norm); err != nil {
			return nil, err
		} else if len(rows) > 0 {
			return rows, nil
		}
	}
	return s.Query("clients", "lower(username) = lower(?) OR lower(mac) = lower(?)", q, q)
}

// SiteIDs returns id/name pairs already mirrored. Events sync needs them.
func (s *Store) SiteIDs() ([][2]string, error) {
	rows, err := s.DB.Query(`SELECT id, name FROM sites`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out [][2]string
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out = append(out, [2]string{id, name})
	}
	return out, rows.Err()
}

// Brief is the one-call fleet rollup from the local mirror.
type Brief struct {
	Source                string            `json:"source"`
	SyncedAt              map[string]string `json:"synced_at"`
	Site                  string            `json:"site,omitempty"`
	Empty                 bool              `json:"empty"`
	Hint                  string            `json:"hint,omitempty"`
	DownDevices           []map[string]any  `json:"down_devices"`
	CriticalAlerts        []map[string]any  `json:"critical_alerts"`
	FirmwareDrift         []map[string]any  `json:"firmware_drift"`
	ExpiringSubscriptions []map[string]any  `json:"expiring_subscriptions"`
	WorstSites            []map[string]any  `json:"worst_sites"`
}

func (s *Store) Brief(site string, within time.Duration, worst int) (Brief, error) {
	if worst <= 0 {
		worst = 5
	}
	b := Brief{
		Source:                "local",
		SyncedAt:              map[string]string{},
		Site:                  site,
		DownDevices:           []map[string]any{},
		CriticalAlerts:        []map[string]any{},
		FirmwareDrift:         []map[string]any{},
		ExpiringSubscriptions: []map[string]any{},
		WorstSites:            []map[string]any{},
	}
	for _, key := range []string{"sites", "devices", "clients", "alerts", "events", "firmware", "subscriptions"} {
		b.SyncedAt[key] = s.Meta("synced_at_" + key)
	}
	total := s.Count("sites") + s.Count("devices") + s.Count("alerts")
	if total == 0 {
		b.Empty = true
		b.Hint = "local mirror is empty; run sync before brief or export"
		return b, nil
	}
	siteClause, args := siteFilter(site)
	downWhere := "(lower(status) LIKE '%down%' OR lower(status) LIKE '%offline%' OR lower(status) IN ('disconnected','unreachable','inactive'))"
	if siteClause != "" {
		downWhere += " AND " + siteClause
	}
	var err error
	if b.DownDevices, err = s.Query("devices", downWhere, args...); err != nil {
		return b, err
	}
	alertWhere := "active = 1 AND lower(severity) = 'critical'"
	if siteClause != "" {
		alertWhere += " AND " + siteClause
	}
	if b.CriticalAlerts, err = s.Query("alerts", alertWhere, args...); err != nil {
		return b, err
	}
	if b.FirmwareDrift, err = s.Query("firmware", "drift = 1"); err != nil {
		return b, err
	}
	subs, err := s.Query("subscriptions", "")
	if err != nil {
		return b, err
	}
	cutoff := time.Now().UTC().Add(within)
	for _, row := range subs {
		exp, _ := row["expires_at"].(string)
		if exp == "" {
			continue
		}
		ts, err := time.Parse(time.RFC3339, exp)
		if err != nil {
			continue
		}
		if !ts.After(cutoff) {
			b.ExpiringSubscriptions = append(b.ExpiringSubscriptions, row)
		}
	}
	worstWhere := ""
	var worstArgs []any
	if strings.TrimSpace(site) != "" {
		worstWhere = "(lower(id) = lower(?) OR lower(name) = lower(?) OR lower(name) LIKE ?)"
		worstArgs = []any{site, site, "%" + strings.ToLower(site) + "%"}
	}
	q := "SELECT * FROM sites"
	if worstWhere != "" {
		q += " WHERE " + worstWhere
	}
	q += " ORDER BY health ASC LIMIT ?"
	worstArgs = append(worstArgs, worst)
	rows, err := s.DB.Query(q, worstArgs...)
	if err != nil {
		return b, err
	}
	defer rows.Close()
	b.WorstSites, err = scanMaps(rows)
	return b, err
}

func siteFilter(site string) (string, []any) {
	site = strings.TrimSpace(site)
	if site == "" {
		return "", nil
	}
	return "(lower(site_id) = lower(?) OR lower(site_name) = lower(?) OR lower(site_name) LIKE ?)", []any{site, site, "%" + strings.ToLower(site) + "%"}
}

// ExportTables returns every mirrored table for agent ingest.
func (s *Store) ExportTables() (map[string]any, error) {
	out := map[string]any{}
	for _, table := range []string{"sites", "devices", "clients", "alerts", "events", "firmware", "subscriptions", "wlans", "audits", "insights", "meta"} {
		rows, err := s.Query(table, "")
		if err != nil {
			return nil, err
		}
		if rows == nil {
			rows = []map[string]any{}
		}
		out[table] = rows
	}
	return out, nil
}
