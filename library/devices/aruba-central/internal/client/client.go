// Copyright 2026 Cursor Agent. Licensed under Apache-2.0. See LICENSE.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/devices/aruba-central/internal/central"
)

const userAgent = "aruba-central-pp-cli/0.1.0"

// APIError is an HTTP failure from New Central after retries are exhausted.
type APIError struct {
	Method string
	Path   string
	Status int
	Body   string
}

func (e *APIError) Error() string {
	msg := e.Body
	if len(msg) > 300 {
		msg = msg[:300]
	}
	return fmt.Sprintf("%s %s: HTTP %d: %s", e.Method, e.Path, e.Status, strings.TrimSpace(msg))
}

// Planned is a request that was not sent because of --dry-run or the write gate.
type Planned struct {
	Method string         `json:"method"`
	URL    string         `json:"url"`
	Body   map[string]any `json:"body,omitempty"`
	DryRun bool           `json:"dry_run"`
}

// Client calls the regional New Central gateway.
type Client struct {
	BaseURL    string
	HTTP       *http.Client
	Limiter    *Limiter
	Tokens     *TokenSource
	MaxRetries int
	DryRun     bool
}

func New(baseURL string, tokens *TokenSource, rate float64, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTP:       &http.Client{Timeout: timeout},
		Limiter:    NewLimiter(rate),
		Tokens:     tokens,
		MaxRetries: 3,
	}
}

// GetJSON performs a GET and decodes a JSON object or array.
func (c *Client) GetJSON(ctx context.Context, path string, q url.Values) (any, error) {
	return c.DoJSON(ctx, http.MethodGet, path, q, nil)
}

// DoJSON performs a JSON request. Non-GET calls still go through the caller's write gate.
func (c *Client) DoJSON(ctx context.Context, method, path string, q url.Values, body any) (any, error) {
	var payload []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		payload = b
	}
	respBody, _, err := c.do(ctx, method, path, q, payload, true)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(respBody)) == 0 {
		return map[string]any{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(respBody))
	dec.UseNumber()
	var out any
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("decode %s %s: %w", method, path, err)
	}
	return out, nil
}

func (c *Client) do(ctx context.Context, method, path string, q url.Values, payload []byte, retryAuth bool) ([]byte, http.Header, error) {
	if c.DryRun {
		return nil, nil, fmt.Errorf("dry-run")
	}
	full, err := c.URL(path, q)
	if err != nil {
		return nil, nil, err
	}
	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if err := c.Limiter.Wait(ctx); err != nil {
			return nil, nil, err
		}
		req, err := http.NewRequestWithContext(ctx, method, full, bytes.NewReader(payload))
		if err != nil {
			return nil, nil, err
		}
		token, _, err := c.Tokens.Token(ctx)
		if err != nil {
			return nil, nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", userAgent)
		if len(payload) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return nil, nil, err
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized && retryAuth && attempt == 0 {
			c.Tokens.Invalidate()
			if _, _, err := c.Tokens.Force(ctx); err != nil {
				return nil, nil, err
			}
			continue
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < c.MaxRetries {
			wait := retryAfter(resp.Header.Get("Retry-After"))
			if wait <= 0 {
				wait = time.Duration(attempt+1) * time.Second
			}
			if err := sleepCtx(ctx, wait); err != nil {
				return nil, nil, err
			}
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, resp.Header, &APIError{Method: method, Path: path, Status: resp.StatusCode, Body: string(body)}
		}
		return body, resp.Header, nil
	}
	if lastErr != nil {
		return nil, nil, lastErr
	}
	return nil, nil, fmt.Errorf("%s %s: retries exhausted", method, path)
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func retryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if n, err := strconv.Atoi(v); err == nil && n >= 0 {
		return time.Duration(n) * time.Second
	}
	if ts, err := http.ParseTime(v); err == nil {
		d := time.Until(ts)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}

// URL joins the regional base URL with a relative API path.
func (c *Client) URL(path string, q url.Values) (string, error) {
	if strings.Contains(path, "://") {
		return "", fmt.Errorf("path must be relative to CENTRAL_BASE_URL, not an absolute URL")
	}
	if err := checkBase(c.BaseURL); err != nil {
		return "", err
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", err
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(path, "/")
	if q != nil {
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

func checkBase(base string) error {
	if central.IsClassicHost(base) {
		return fmt.Errorf("base URL %s looks like Classic Central; this CLI talks to New Central only", base)
	}
	u, err := url.Parse(base)
	if err != nil {
		return fmt.Errorf("base URL: %w", err)
	}
	if u.Scheme == "https" {
		return nil
	}
	if central.LoopbackHTTP(base) {
		return nil
	}
	return fmt.Errorf("CENTRAL_BASE_URL must be https (got %s)", u.Scheme)
}

// ListPages walks offset or next pagination and returns every object.
// style is "offset" (sites-health) or "next" (devices, clients, alerts, events).
func (c *Client) ListPages(ctx context.Context, path, style, itemKey string, extra url.Values, limit, maxPages int) (items []map[string]any, pages int, truncated bool, err error) {
	if limit <= 0 {
		limit = 100
	}
	if maxPages <= 0 {
		maxPages = 200
	}
	if itemKey == "" {
		itemKey = "items"
	}
	offset := 0
	next := "1"
	seen := map[string]struct{}{}
	for page := 1; page <= maxPages; page++ {
		q := url.Values{}
		for k, vs := range extra {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
		q.Set("limit", strconv.Itoa(limit))
		switch style {
		case "offset":
			q.Set("offset", strconv.Itoa(offset))
		default:
			q.Set("next", next)
		}
		body, err := c.GetJSON(ctx, path, q)
		if err != nil {
			return items, pages, truncated, err
		}
		pages = page
		batch := central.AsMaps(body, itemKey, "items", "events")
		items = append(items, batch...)
		total, hasTotal := central.Total(body)
		if len(batch) == 0 {
			return items, pages, false, nil
		}
		if hasTotal && len(items) >= total {
			return items, pages, false, nil
		}
		if len(batch) < limit && style == "offset" {
			return items, pages, false, nil
		}
		if style == "offset" {
			offset += len(batch)
			if hasTotal && offset >= total {
				return items, pages, false, nil
			}
			continue
		}
		m, _ := body.(map[string]any)
		rawNext, ok := m["next"]
		if !ok || rawNext == nil || fmt.Sprint(rawNext) == "" || fmt.Sprint(rawNext) == "<nil>" {
			if len(batch) < limit {
				return items, pages, false, nil
			}
			// Page number APIs omit next and expect the caller to increment.
			n, _ := strconv.Atoi(next)
			if n == 0 {
				n = page
			}
			next = strconv.Itoa(n + 1)
		} else {
			next = fmt.Sprint(rawNext)
		}
		if _, dup := seen[next]; dup {
			return items, pages, false, nil
		}
		seen[next] = struct{}{}
	}
	return items, pages, true, nil
}
