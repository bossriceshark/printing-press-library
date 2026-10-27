// Copyright 2026 Matt Rice and contributors. Licensed under Apache-2.0. See LICENSE.
package central

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Str reads the first non-empty alias from an API object.
func Str(m map[string]any, keys ...string) string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch t := v.(type) {
		case string:
			if strings.TrimSpace(t) != "" {
				return strings.TrimSpace(t)
			}
		case float64:
			if t == float64(int64(t)) {
				return strconv.FormatInt(int64(t), 10)
			}
			return strconv.FormatFloat(t, 'f', -1, 64)
		case json.Number:
			return t.String()
		case bool:
			if t {
				return "true"
			}
		}
	}
	return ""
}

// Num reads the first numeric alias, including numeric strings.
func Num(m map[string]any, keys ...string) float64 {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch t := v.(type) {
		case float64:
			return t
		case json.Number:
			f, err := t.Float64()
			if err == nil {
				return f
			}
		case string:
			f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
			if err == nil {
				return f
			}
		case map[string]any:
			if n := Num(t, "count", "value", "total", "score"); n != 0 || Str(t, "count", "value", "score") != "" {
				return Num(t, "count", "value", "total", "score")
			}
		}
	}
	return 0
}

// NormMAC keeps hex digits only, lowercase, so aa:bb and aabb compare equal.
func NormMAC(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// DeviceFamily maps Central device-type strings onto ap, switch, or gateway.
func DeviceFamily(raw string) string {
	s := strings.ToLower(strings.ReplaceAll(raw, "_", " "))
	s = strings.ReplaceAll(s, "-", " ")
	switch {
	case strings.Contains(s, "gateway") || s == "gw" || strings.Contains(s, "mobility gw"):
		return "gateway"
	case strings.Contains(s, "switch") || s == "cx" || strings.Contains(s, "aos s"):
		return "switch"
	case strings.Contains(s, "access point") || strings.Contains(s, "ap") || strings.Contains(s, "iap"):
		return "ap"
	default:
		return strings.TrimSpace(strings.ToLower(raw))
	}
}

// IsDown reports device status values agents treat as offline.
func IsDown(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	if s == "" {
		return false
	}
	switch s {
	case "down", "offline", "disconnected", "unreachable", "inactive":
		return true
	}
	return strings.Contains(s, "down") || strings.Contains(s, "offline")
}

// IsFailedClient reports client rows that failed onboarding or association.
func IsFailedClient(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	return strings.Contains(s, "fail")
}

// IsActiveAlert is true unless the alert is cleared, closed, or acknowledged.
func IsActiveAlert(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	if s == "" {
		return true
	}
	switch s {
	case "cleared", "clear", "closed", "close", "ack", "acked", "acknowledged", "resolved":
		return false
	}
	return true
}

// HealthScore pulls a numeric score from a sites-health record.
// Lower is worse when the API sends an explicit score. When only good/fair/poor
// buckets exist, the score is good-minus-poor so the worst sites sort first.
func HealthScore(m map[string]any) float64 {
	if raw, ok := m["health"]; ok {
		switch t := raw.(type) {
		case float64:
			return t
		case string:
			if f, err := strconv.ParseFloat(t, 64); err == nil {
				return f
			}
		case map[string]any:
			if s := Str(t, "score", "healthScore"); s != "" {
				return Num(t, "score", "healthScore")
			}
			return bucketScore(t)
		}
	}
	if s := Str(m, "healthScore", "score"); s != "" {
		return Num(m, "healthScore", "score")
	}
	return bucketScore(m)
}

func bucketScore(m map[string]any) float64 {
	good := groupCount(m, "good")
	poor := groupCount(m, "poor")
	fair := groupCount(m, "fair")
	if good == 0 && poor == 0 && fair == 0 {
		return 0
	}
	return good - poor
}

func groupCount(m map[string]any, name string) float64 {
	for _, key := range []string{"groups", "health", "alerts"} {
		raw, ok := m[key]
		if !ok {
			continue
		}
		switch t := raw.(type) {
		case []any:
			for _, item := range t {
				gm, ok := item.(map[string]any)
				if !ok {
					continue
				}
				if strings.EqualFold(Str(gm, "name", "key", "severity"), name) {
					return Num(gm, "count", "value")
				}
			}
		case map[string]any:
			if v := Num(t, name); Str(t, name) != "" || v != 0 {
				return v
			}
			if nested, ok := t["groups"]; ok {
				b, _ := json.Marshal(map[string]any{"groups": nested})
				var wrap map[string]any
				if json.Unmarshal(b, &wrap) == nil {
					return groupCount(wrap, name)
				}
			}
		}
	}
	return 0
}

// CriticalAlerts reads a critical-alert count from a site record.
func CriticalAlerts(m map[string]any) int {
	if raw, ok := m["alerts"]; ok {
		if am, ok := raw.(map[string]any); ok {
			if n := groupCount(am, "critical"); n != 0 || hasGroup(am, "critical") {
				return int(n)
			}
			if s := Str(am, "critical"); s != "" {
				return int(Num(am, "critical"))
			}
		}
	}
	return int(Num(m, "criticalAlerts", "critical_alerts", "critical"))
}

func hasGroup(m map[string]any, name string) bool {
	raw, ok := m["groups"]
	if !ok {
		return false
	}
	items, ok := raw.([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		gm, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if strings.EqualFold(Str(gm, "name", "key", "severity"), name) {
			return true
		}
	}
	return false
}

// NestedCount reads {"devices":{"count":N}} style fields.
func NestedCount(m map[string]any, keys ...string) int {
	for _, k := range keys {
		raw, ok := m[k]
		if !ok {
			continue
		}
		switch t := raw.(type) {
		case float64:
			return int(t)
		case map[string]any:
			return int(Num(t, "count", "total", "value"))
		case string:
			f, err := strconv.ParseFloat(t, 64)
			if err == nil {
				return int(f)
			}
		}
	}
	return 0
}

// ParseTime tries the timestamp shapes New Central and GLP return.
func ParseTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return time.Time{}, false
		}
		layouts := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02T15:04:05.000Z0700",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, layout := range layouts {
			if ts, err := time.Parse(layout, s); err == nil {
				return ts.UTC(), true
			}
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return unixish(n), true
		}
	case float64:
		if t > 1_000_000_000 {
			return unixish(int64(t)), true
		}
	}
	return time.Time{}, false
}

func unixish(n int64) time.Time {
	if n > 1_000_000_000_000 {
		return time.UnixMilli(n).UTC()
	}
	return time.Unix(n, 0).UTC()
}

// FirstTime returns the first parseable timestamp among aliases.
func FirstTime(m map[string]any, keys ...string) (time.Time, bool) {
	for _, k := range keys {
		if ts, ok := ParseTime(m[k]); ok {
			return ts, true
		}
	}
	return time.Time{}, false
}

// AsMap converts a decoded JSON value into an object when it is one.
func AsMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

// AsMaps reads a list of objects from either a bare array or an envelope key.
func AsMaps(v any, keys ...string) []map[string]any {
	switch t := v.(type) {
	case []any:
		return mapsOf(t)
	case map[string]any:
		for _, k := range keys {
			if raw, ok := t[k]; ok {
				if items, ok := raw.([]any); ok {
					return mapsOf(items)
				}
			}
		}
	}
	return nil
}

func mapsOf(items []any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if ok {
			out = append(out, m)
		}
	}
	return out
}

// Total reads a page total when the API sent one.
func Total(v any) (int, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return 0, false
	}
	if _, ok := m["total"]; !ok {
		return 0, false
	}
	return int(Num(m, "total")), true
}

// IsClassicHost reports Classic Central API gateways, which this CLI does not call.
func IsClassicHost(baseURL string) bool {
	u, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	h := strings.ToLower(u.Hostname())
	return strings.Contains(h, "apigw") || strings.HasPrefix(h, "classic.")
}

// LoopbackHTTP is the only cleartext base URL allowed, so tests can use httptest.
func LoopbackHTTP(baseURL string) bool {
	u, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	if u.Scheme != "http" {
		return false
	}
	h := u.Hostname()
	return h == "127.0.0.1" || h == "localhost" || h == "::1"
}
