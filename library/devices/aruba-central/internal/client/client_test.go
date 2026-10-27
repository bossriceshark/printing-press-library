// Copyright 2026 Cursor Agent. Licensed under Apache-2.0. See LICENSE.
package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestLimiterFakeClock(t *testing.T) {
	start := time.Unix(0, 0)
	now := start
	var slept time.Duration
	l := NewLimiter(10)
	l.burst = 1
	l.tokens = 1
	l.last = start
	l.now = func() time.Time { return now }
	l.sleep = func(d time.Duration) {
		slept += d
		now = now.Add(d)
	}
	l.fakeClock = true
	ctx := context.Background()
	if err := l.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if err := l.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if slept < 50*time.Millisecond {
		t.Fatalf("expected throttle sleep, got %s", slept)
	}
}

func TestTokenMintAndCache(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		u, p, ok := r.BasicAuth()
		if !ok || u != "id" || p != "secret" {
			http.Error(w, "nope", 401)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok-1", "expires_in": 7200})
	}))
	defer srv.Close()
	cache := filepath.Join(t.TempDir(), "token.json")
	src := &TokenSource{ClientID: "id", Secret: "secret", TokenURL: srv.URL, CachePath: cache, HTTP: srv.Client()}
	tok, exp, err := src.Token(context.Background())
	if err != nil || tok != "tok-1" || exp.IsZero() {
		t.Fatalf("%s %v %v", tok, exp, err)
	}
	if _, _, err := src.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatalf("hits %d", hits.Load())
	}
	src2 := &TokenSource{ClientID: "id", Secret: "secret", TokenURL: srv.URL, CachePath: cache, HTTP: srv.Client()}
	tok, _, err = src2.Token(context.Background())
	if err != nil || tok != "tok-1" || hits.Load() != 1 {
		t.Fatalf("cache reload tok=%s hits=%d err=%v", tok, hits.Load(), err)
	}
}

func TestRetryAfterAnd401(t *testing.T) {
	var apiHits atomic.Int32
	mux := http.NewServeMux()
	var token string
	mux.HandleFunc("/as/token.oauth2", func(w http.ResponseWriter, r *http.Request) {
		token = "tok-" + time.Now().Format("150405.000")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": token, "expires_in": 100})
	})
	mux.HandleFunc("/network-monitoring/v1/devices", func(w http.ResponseWriter, r *http.Request) {
		n := apiHits.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(429)
			_, _ = io.WriteString(w, `{"error":"slow"}`)
			return
		}
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(401)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{map[string]any{"serial": "SN1"}}, "total": 1})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	src := &TokenSource{ClientID: "id", Secret: "secret", TokenURL: srv.URL + "/as/token.oauth2", HTTP: srv.Client()}
	c := New(srv.URL, src, 1000, 5*time.Second)
	c.HTTP = srv.Client()
	body, err := c.GetJSON(context.Background(), "/network-monitoring/v1/devices", nil)
	if err != nil {
		t.Fatal(err)
	}
	items := body.(map[string]any)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("%v", body)
	}
}

func TestClassicRejected(t *testing.T) {
	c := New("https://apigw-prod2.central.arubanetworks.com", &TokenSource{}, 10, time.Second)
	if _, err := c.URL("/network-monitoring/v1/devices", nil); err == nil {
		t.Fatal("expected classic rejection")
	}
}
