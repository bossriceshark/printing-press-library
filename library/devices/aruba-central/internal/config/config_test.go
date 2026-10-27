// Copyright 2026 Matt Rice and contributors. Licensed under Apache-2.0. See LICENSE.
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExplicitFileDoesNotReadHome(t *testing.T) {
	t.Setenv(EnvBaseURL, "")
	t.Setenv(EnvClientID, "")
	t.Setenv(EnvClientSecret, "")
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("base_url = \"https://us5.api.central.arubanetworks.com\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL == "" || cfg.ClientID != "" {
		t.Fatalf("%+v", cfg)
	}
	missing := cfg.Missing()
	if len(missing) != 2 {
		t.Fatalf("missing %#v", missing)
	}
}

func TestEnvOverridesFile(t *testing.T) {
	t.Setenv(EnvBaseURL, "https://us1.api.central.arubanetworks.com")
	t.Setenv(EnvClientID, "id")
	t.Setenv(EnvClientSecret, "secret")
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("base_url = \"https://us5.api.central.arubanetworks.com\"\nclient_id = \"file\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Require(); err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "https://us1.api.central.arubanetworks.com" || cfg.ClientID != "id" {
		t.Fatalf("%+v", cfg)
	}
	if cfg.ClientSecret != "secret" {
		t.Fatal("secret not loaded")
	}
}
