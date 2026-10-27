// Copyright 2026 Matt Rice and contributors. Licensed under Apache-2.0. See LICENSE.
package central

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseWindow parses durations used by --since and --within.
// Go's time.ParseDuration does not accept days, so 7d and 90d are handled here.
// Supported units: m, h, d.
func ParseWindow(raw string) (time.Duration, error) {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	if len(s) < 2 {
		return 0, fmt.Errorf("duration %q must look like 6h, 24h, 7d, or 90d", raw)
	}
	unit := s[len(s)-1]
	n, err := strconv.Atoi(s[:len(s)-1])
	if err != nil || n < 0 {
		return 0, fmt.Errorf("duration %q must look like 6h, 24h, 7d, or 90d", raw)
	}
	switch unit {
	case 'm':
		return time.Duration(n) * time.Minute, nil
	case 'h':
		return time.Duration(n) * time.Hour, nil
	case 'd':
		return time.Duration(n) * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("duration %q must end in m, h, or d", raw)
	}
}
