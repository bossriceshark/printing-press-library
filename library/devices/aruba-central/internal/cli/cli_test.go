// Copyright 2026 Cursor Agent. Licensed under Apache-2.0. See LICENSE.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mvanhorn/printing-press-library/library/devices/aruba-central/internal/client"
	"github.com/mvanhorn/printing-press-library/library/devices/aruba-central/internal/store"
	"github.com/mvanhorn/printing-press-library/library/devices/aruba-central/internal/syncer"
)

func TestDoctorMissingCreds(t *testing.T) {
	t.Setenv("CENTRAL_BASE_URL", "")
	t.Setenv("CENTRAL_CLIENT_ID", "")
	t.Setenv("CENTRAL_CLIENT_SECRET", "")
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(cfgPath, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	buf := &bytes.Buffer{}
	cmd := newRootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--config", cfgPath, "doctor"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if ExitCode(err) != 10 {
		t.Fatalf("exit %d err %v", ExitCode(err), err)
	}
	var body map[string]any
	if err := json.Unmarshal(buf.Bytes(), &body); err != nil {
		t.Fatalf("stdout %s: %v", buf.String(), err)
	}
	if body["ok"] != false {
		t.Fatalf("%v", body)
	}
	missing, _ := body["missing"].([]any)
	joined := buf.String()
	for _, name := range []string{"CENTRAL_BASE_URL", "CENTRAL_CLIENT_ID", "CENTRAL_CLIENT_SECRET"} {
		if !strings.Contains(joined, name) {
			t.Fatalf("missing %s in %s", name, joined)
		}
	}
	if len(missing) != 3 {
		t.Fatalf("%v", body)
	}
}

func TestHelpListsMVP(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := newRootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sync", "doctor", "brief", "export", "auth", "sites", "devices", "clients", "alerts", "events", "firmware", "subscriptions", "wlans", "audit", "api"} {
		if !strings.Contains(buf.String(), name) {
			t.Fatalf("help missing %s\n%s", name, buf.String())
		}
	}
}

func TestSyncBriefExport(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/as/token.oauth2", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 7200})
	})
	mux.HandleFunc("/network-monitoring/v1/sites-health", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{
			map[string]any{"id": "s1", "siteName": "HQ", "health": map[string]any{"score": float64(10)}, "devices": map[string]any{"count": float64(2)}, "alerts": map[string]any{"groups": []any{map[string]any{"name": "critical", "count": float64(1)}}}},
		}, "total": 1})
	})
	mux.HandleFunc("/network-monitoring/v1/devices", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{
			map[string]any{"serialNumber": "SN1", "deviceName": "ap-lobby", "macAddress": "aa:bb:cc:dd:ee:ff", "deviceType": "ACCESS_POINT", "status": "Down", "siteId": "s1", "siteName": "HQ", "firmwareVersion": "1.0"},
		}, "total": 1})
	})
	mux.HandleFunc("/network-monitoring/v1/clients", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{
			map[string]any{"macAddress": "11:22:33:44:55:66", "username": "ada", "status": "Failed", "siteName": "HQ", "usage": float64(90)},
		}, "total": 1})
	})
	mux.HandleFunc("/network-notifications/v1/alerts", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{
			map[string]any{"id": "a1", "severity": "critical", "status": "open", "siteName": "HQ", "summary": "AP down"},
		}, "total": 1})
	})
	mux.HandleFunc("/network-notifications/v1/insights", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "total": 0})
	})
	mux.HandleFunc("/network-troubleshooting/v1/events", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"events": []any{
			map[string]any{"uuid": "e1", "serialNumber": "SN1", "severity": "major", "timestamp": "2026-10-27T00:00:00Z", "eventName": "down"},
		}, "total": 1})
	})
	mux.HandleFunc("/network-services/v1/firmware-details", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{
			map[string]any{"serialNumber": "SN1", "firmwareVersion": "1.0", "recommendedVersion": "2.0", "model": "AP-635"},
		}, "total": 1})
	})
	mux.HandleFunc("/network-monitoring/v1/subscription-expiring-summary", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{
			map[string]any{"id": "sub1", "name": "foundation", "expirationDate": "2026-11-01T00:00:00Z"},
		}, "total": 1})
	})
	mux.HandleFunc("/network-monitoring/v1/wlans", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{map[string]any{"id": "w1", "wlanName": "corp", "band": "5GHz", "status": "up"}}, "total": 1})
	})
	mux.HandleFunc("/network-services/v1/audits", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{map[string]any{"id": "au1", "timestamp": "2026-10-26T00:00:00Z", "user": "admin", "action": "login"}}, "total": 1})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "central.db")
	tokenPath := filepath.Join(dir, "token.json")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	src := &client.TokenSource{ClientID: "id", Secret: "secret", TokenURL: srv.URL + "/as/token.oauth2", CachePath: tokenPath, HTTP: srv.Client()}
	c := client.New(srv.URL, src, 1000, 0)
	c.HTTP = srv.Client()
	results, err := syncer.Run(context.Background(), c, st, syncer.Options{Since: 24 * time.Hour, MaxPages: 5})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Error != "" {
			t.Fatalf("%s: %s", r.Resource, r.Error)
		}
		if r.Count == 0 && r.Resource != "audits" {
			// audits should be 1; allow insights not in default
			if r.Resource != "insights" {
				t.Fatalf("empty %s %+v", r.Resource, r)
			}
		}
	}
	b, err := st.Brief("", 90*24*time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}
	if b.Empty || len(b.DownDevices) != 1 || len(b.CriticalAlerts) != 1 || len(b.FirmwareDrift) != 1 {
		t.Fatalf("%+v", b)
	}
	tables, err := st.ExportTables()
	if err != nil {
		t.Fatal(err)
	}
	sites, _ := tables["sites"].([]map[string]any)
	if len(sites) != 1 {
		t.Fatalf("export sites %#v", tables["sites"])
	}
	_ = st.Close()
}

func TestBouncePortDryRun(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := newRootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"device", "bounce-port", "--serial", "SN1", "--ports", "1"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"dry_run": true`) || !strings.Contains(buf.String(), "portBounce") {
		t.Fatalf("%s", buf.String())
	}
}
