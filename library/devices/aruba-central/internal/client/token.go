// Copyright 2026 Cursor Agent. Licensed under Apache-2.0. See LICENSE.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const defaultTokenTTL = 2 * time.Hour

// TokenSource mints and caches OAuth2 client_credentials tokens.
// Central-scoped Personal API clients do not receive a refresh token; the CLI re-mints on expiry.
type TokenSource struct {
	ClientID  string
	Secret    string
	TokenURL  string
	CachePath string
	HTTP      *http.Client
	Now       func() time.Time

	mu     sync.Mutex
	access string
	exp    time.Time
}

type tokenFile struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

func (t *TokenSource) now() time.Time {
	if t.Now != nil {
		return t.Now()
	}
	return time.Now()
}

func (t *TokenSource) httpClient() *http.Client {
	if t.HTTP != nil {
		return t.HTTP
	}
	return http.DefaultClient
}

// Token returns a valid access token, reading the cache and re-minting when it is inside 60s of expiry.
func (t *TokenSource) Token(ctx context.Context) (string, time.Time, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.access != "" && t.exp.After(t.now().Add(60*time.Second)) {
		return t.access, t.exp, nil
	}
	if err := t.loadCache(); err != nil {
		return "", time.Time{}, err
	}
	if t.access != "" && t.exp.After(t.now().Add(60*time.Second)) {
		return t.access, t.exp, nil
	}
	return t.mint(ctx)
}

// Force mints a new token and replaces the cache.
func (t *TokenSource) Force(ctx context.Context) (string, time.Time, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.access = ""
	t.exp = time.Time{}
	return t.mint(ctx)
}

// Invalidate drops the in-memory token so the next call re-mints. Used after HTTP 401.
func (t *TokenSource) Invalidate() {
	t.mu.Lock()
	t.access = ""
	t.exp = time.Time{}
	t.mu.Unlock()
}

// Cached reports the current cache without contacting the token endpoint.
func (t *TokenSource) Cached() (expires time.Time, ok bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	_ = t.loadCache()
	if t.access == "" {
		return time.Time{}, false
	}
	return t.exp, true
}

func (t *TokenSource) loadCache() error {
	if t.CachePath == "" || t.access != "" {
		return nil
	}
	b, err := os.ReadFile(t.CachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var f tokenFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil
	}
	t.access = f.AccessToken
	t.exp = f.ExpiresAt
	return nil
}

func (t *TokenSource) mint(ctx context.Context) (string, time.Time, error) {
	if t.ClientID == "" || t.Secret == "" {
		return "", time.Time{}, fmt.Errorf("client id and client secret are required to mint a token")
	}
	endpoint := t.TokenURL
	if endpoint == "" {
		endpoint = "https://sso.common.cloud.hpe.com/as/token.oauth2"
	}
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", t.ClientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", time.Time{}, err
	}
	req.SetBasicAuth(t.ClientID, t.Secret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := t.httpClient().Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("token endpoint: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", time.Time{}, fmt.Errorf("token endpoint returned HTTP %d", resp.StatusCode)
	}
	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil || tr.AccessToken == "" {
		return "", time.Time{}, fmt.Errorf("token endpoint returned no access_token")
	}
	ttl := defaultTokenTTL
	if tr.ExpiresIn > 0 {
		ttl = time.Duration(tr.ExpiresIn) * time.Second
	}
	exp := t.now().Add(ttl)
	t.access = tr.AccessToken
	t.exp = exp
	if err := t.save(tr.AccessToken, exp); err != nil {
		return "", time.Time{}, err
	}
	return tr.AccessToken, exp, nil
}

func (t *TokenSource) save(token string, exp time.Time) error {
	if t.CachePath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(t.CachePath), 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(tokenFile{AccessToken: token, ExpiresAt: exp.UTC()})
	if err != nil {
		return err
	}
	return os.WriteFile(t.CachePath, b, 0o600)
}
