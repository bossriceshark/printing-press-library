// Copyright 2026 Matt Rice and contributors. Licensed under Apache-2.0. See LICENSE.
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	EnvBaseURL      = "CENTRAL_BASE_URL"
	EnvClientID     = "CENTRAL_CLIENT_ID"
	EnvClientSecret = "CENTRAL_CLIENT_SECRET"

	// TokenURL is the New Central client_credentials endpoint.
	// Central-scoped Personal API clients mint ~2h tokens here.
	// GLP-scoped clients use a different host and are not the default.
	TokenURL = "https://sso.common.cloud.hpe.com/as/token.oauth2"

	appDir = "aruba-central-pp-cli"
)

// Config is everything the CLI needs to talk to New Central.
// Secrets stay in env, dotenv, or the user's config file. Nothing here is baked in.
type Config struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	TokenURL     string
	TokenPath    string
	DBPath       string
	ConfigFile   string
}

type fileShape struct {
	BaseURL      string `toml:"base_url"`
	ClientID     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
}

// MissingError lists the env vars a customer still has to set.
type MissingError struct {
	Missing []string
}

func (e *MissingError) Error() string {
	return "missing credentials: " + strings.Join(e.Missing, ", ")
}

// Load resolves config for a run.
// explicitPath, when set, is the only file consulted. Env values still win when non-empty.
// When explicitPath is empty, blanks are filled from ./.env, then the user config dotenv, then config.toml.
func Load(explicitPath string) (Config, error) {
	cfg := Config{TokenURL: TokenURL}
	if explicitPath != "" {
		cfg.ConfigFile = explicitPath
		fileCfg, err := readFile(explicitPath)
		if err != nil {
			return cfg, err
		}
		cfg.BaseURL = fileCfg.BaseURL
		cfg.ClientID = fileCfg.ClientID
		cfg.ClientSecret = fileCfg.ClientSecret
	} else {
		applyDotenv(filepath.Join(".", ".env"), &cfg)
		if dir, err := os.UserConfigDir(); err == nil {
			base := filepath.Join(dir, appDir)
			applyDotenv(filepath.Join(base, ".env"), &cfg)
			tomlPath := filepath.Join(base, "config.toml")
			if _, err := os.Stat(tomlPath); err == nil {
				fileCfg, err := readFile(tomlPath)
				if err != nil {
					return cfg, err
				}
				cfg.ConfigFile = tomlPath
				if cfg.BaseURL == "" {
					cfg.BaseURL = fileCfg.BaseURL
				}
				if cfg.ClientID == "" {
					cfg.ClientID = fileCfg.ClientID
				}
				if cfg.ClientSecret == "" {
					cfg.ClientSecret = fileCfg.ClientSecret
				}
			}
		}
	}
	overrideEnv(&cfg)
	if cfg.TokenPath == "" {
		cfg.TokenPath = defaultTokenPath(explicitPath)
	}
	if cfg.DBPath == "" {
		cfg.DBPath = defaultDBPath()
	}
	if os.Getenv("ARUBA_CENTRAL_ALLOW_TOKEN_URL_OVERRIDE") == "1" {
		if v := strings.TrimSpace(os.Getenv("ARUBA_CENTRAL_TOKEN_URL")); v != "" {
			cfg.TokenURL = v
		}
	}
	if v := strings.TrimSpace(os.Getenv("ARUBA_CENTRAL_TOKEN_PATH")); v != "" {
		cfg.TokenPath = v
	}
	if v := strings.TrimSpace(os.Getenv("ARUBA_CENTRAL_DB")); v != "" {
		cfg.DBPath = v
	}
	return cfg, nil
}

func overrideEnv(cfg *Config) {
	if v := strings.TrimSpace(os.Getenv(EnvBaseURL)); v != "" {
		cfg.BaseURL = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvClientID)); v != "" {
		cfg.ClientID = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvClientSecret)); v != "" {
		cfg.ClientSecret = v
	}
}

// Missing returns the customer-facing env var names that are still empty.
func (c Config) Missing() []string {
	var out []string
	if strings.TrimSpace(c.BaseURL) == "" {
		out = append(out, EnvBaseURL)
	}
	if strings.TrimSpace(c.ClientID) == "" {
		out = append(out, EnvClientID)
	}
	if strings.TrimSpace(c.ClientSecret) == "" {
		out = append(out, EnvClientSecret)
	}
	return out
}

// Require returns MissingError when any of the three variables is empty.
func (c Config) Require() error {
	missing := c.Missing()
	if len(missing) == 0 {
		return nil
	}
	return &MissingError{Missing: missing}
}

func readFile(path string) (fileShape, error) {
	var shape fileShape
	b, err := os.ReadFile(path)
	if err != nil {
		return shape, fmt.Errorf("read config %s: %w", path, err)
	}
	if strings.HasSuffix(strings.ToLower(path), ".env") {
		applyDotenvContent(string(b), nil)
		return shape, nil
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return shape, nil
	}
	if err := toml.Unmarshal(b, &shape); err != nil {
		return shape, fmt.Errorf("parse config %s: %w", path, err)
	}
	return shape, nil
}

func applyDotenv(path string, cfg *Config) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	vals := map[string]string{}
	applyDotenvContent(string(b), vals)
	if cfg.BaseURL == "" {
		cfg.BaseURL = vals[EnvBaseURL]
	}
	if cfg.ClientID == "" {
		cfg.ClientID = vals[EnvClientID]
	}
	if cfg.ClientSecret == "" {
		cfg.ClientSecret = vals[EnvClientSecret]
	}
	if cfg.ConfigFile == "" {
		cfg.ConfigFile = path
	}
}

func applyDotenvContent(text string, into map[string]string) {
	if into == nil {
		return
	}
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		into[k] = v
	}
}

func defaultTokenPath(explicit string) string {
	if explicit != "" {
		return filepath.Join(filepath.Dir(explicit), "token.json")
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, appDir, "token.json")
	}
	return filepath.Join(".", "token.json")
}

func defaultDBPath() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, appDir, "central.db")
	}
	return filepath.Join(".", "central.db")
}

// UserConfigDir is the directory customers edit.
func UserConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(".", "."+appDir)
	}
	return filepath.Join(dir, appDir)
}
