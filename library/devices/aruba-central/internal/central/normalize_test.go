// Copyright 2026 Matt Rice and contributors. Licensed under Apache-2.0. See LICENSE.
package central

import (
	"testing"
	"time"
)

func TestParseWindow(t *testing.T) {
	d, err := ParseWindow("90d")
	if err != nil || d != 90*24*time.Hour {
		t.Fatalf("90d: %v %v", d, err)
	}
	if _, err := ParseWindow("6h"); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseWindow("nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestDeviceFamilyAndStatus(t *testing.T) {
	if DeviceFamily("ACCESS_POINT") != "ap" {
		t.Fatal(DeviceFamily("ACCESS_POINT"))
	}
	if DeviceFamily("CX_SWITCH") != "switch" {
		t.Fatal(DeviceFamily("CX_SWITCH"))
	}
	if !IsDown("Offline") || IsDown("Up") {
		t.Fatal("down detection")
	}
	if !IsFailedClient("Failed") || !IsActiveAlert("") || IsActiveAlert("cleared") {
		t.Fatal("status helpers")
	}
}

func TestHealthAndCritical(t *testing.T) {
	site := map[string]any{
		"siteName": "HQ",
		"id":       "9",
		"health": map[string]any{
			"groups": []any{
				map[string]any{"name": "poor", "count": float64(3)},
				map[string]any{"name": "good", "count": float64(1)},
			},
		},
		"alerts": map[string]any{
			"groups": []any{map[string]any{"name": "critical", "count": float64(2)}},
		},
		"devices": map[string]any{"count": float64(4)},
	}
	if HealthScore(site) != -2 {
		t.Fatalf("score %v", HealthScore(site))
	}
	if CriticalAlerts(site) != 2 {
		t.Fatalf("crit %d", CriticalAlerts(site))
	}
	if NestedCount(site, "devices") != 4 {
		t.Fatal("devices")
	}
}

func TestClassicHost(t *testing.T) {
	if !IsClassicHost("https://apigw-prod2.central.arubanetworks.com") {
		t.Fatal("classic")
	}
	if IsClassicHost("https://us5.api.central.arubanetworks.com") {
		t.Fatal("new central misread")
	}
	if !LoopbackHTTP("http://127.0.0.1:9") || LoopbackHTTP("http://example.com") {
		t.Fatal("loopback")
	}
}

func TestNormMAC(t *testing.T) {
	if NormMAC("AA:BB:CC:DD:EE:FF") != NormMAC("aabb.ccdd.eeff") {
		t.Fatal("mac")
	}
}
