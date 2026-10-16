package cli

import (
	"testing"
	"time"
)

func TestUnpaidKeepsPendingAndLate(t *testing.T) {
	rows := unpaidFromInvoices([]map[string]any{
		invoiceItem("Acme", "brand-a", 2026, 0, "pending", 12.5, "amazon.com"),
		invoiceItem("Paid Co", "brand-b", 2026, 1, "paid", 9, "amazon.com"),
		invoiceItem("Late Co", "brand-c", 2025, 11, "late", 3.25, "amazon.co.uk"),
	}, "", "")
	if len(rows) != 2 {
		t.Fatalf("len = %d, rows = %+v", len(rows), rows)
	}
	if rows[0].Brand != "Acme" || rows[0].Month != 0 || rows[0].Commission != 12.5 {
		t.Fatalf("first = %+v", rows[0])
	}
	if rows[1].Status != "late" || rows[1].Marketplace != "amazon.co.uk" {
		t.Fatalf("second = %+v", rows[1])
	}
}

func TestEarnedRollsUpByLink(t *testing.T) {
	end := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	start := end.AddDate(0, 0, -30)
	rows := earnedFromReports([]map[string]any{
		reportRow("2026-03-20T00:00:00.000Z", "B09G9FPHY6", "link-1", "https://lvnta.example/a", "Newsletter", 4.5, 10, "amazon.com", "brand-a", 2026, 2, "pending"),
		reportRow("2026-03-21T00:00:00.000Z", "B09G9FPHY6", "link-1", "https://lvnta.example/a", "Newsletter", 1.5, 2, "amazon.com", "brand-a", 2026, 2, "pending"),
		reportRow("2026-01-01T00:00:00.000Z", "B09G9FPHY6", "link-1", "https://lvnta.example/a", "Newsletter", 99, 1, "amazon.com", "brand-a", 2026, 0, "paid"),
		reportRow("2026-03-22T00:00:00.000Z", "B000000000", "link-2", "https://lvnta.example/b", "Blog", 0, 8, "amazon.com", "brand-a", 2026, 2, "pending"),
	}, start, end)
	if len(rows) != 1 {
		t.Fatalf("len = %d, rows = %+v", len(rows), rows)
	}
	if rows[0].Commission != 6 || rows[0].Clicks != 12 || rows[0].Source != "Newsletter" || rows[0].URL != "https://lvnta.example/a" {
		t.Fatalf("row = %+v", rows[0])
	}
}

func TestReconcileFlagsCommissionAndStatus(t *testing.T) {
	invoices := []map[string]any{
		invoiceItem("Acme", "brand-a", 2026, 2, "paid", 10, "amazon.com"),
		invoiceItem("Match", "brand-b", 2026, 2, "pending", 5, "amazon.com"),
	}
	reports := []map[string]any{
		reportRow("2026-03-02T00:00:00.000Z", "B09G9FPHY6", "link-1", "https://lvnta.example/a", "Newsletter", 8, 1, "amazon.com", "brand-a", 2026, 2, "pending"),
		reportRow("2026-03-03T00:00:00.000Z", "B09G9FPHY6", "link-2", "https://lvnta.example/b", "Blog", 5, 1, "amazon.com", "brand-b", 2026, 2, "pending"),
	}
	rows := reconcileInvoices(invoices, reports, 0.01)
	if len(rows) != 1 {
		t.Fatalf("len = %d, rows = %+v", len(rows), rows)
	}
	if rows[0].Brand != "Acme" || rows[0].InvoiceCommission != 10 || rows[0].ReportCommission != 8 || rows[0].Reason != "commission,status" {
		t.Fatalf("row = %+v", rows[0])
	}
}

func TestPayoutGapFindsMissingInvoice(t *testing.T) {
	invoices := []map[string]any{
		invoiceItem("Acme", "brand-a", 2026, 2, "pending", 10, "amazon.com"),
	}
	reports := []map[string]any{
		reportRow("2026-03-02T00:00:00.000Z", "B09G9FPHY6", "link-1", "https://lvnta.example/a", "Newsletter", 4, 1, "amazon.com", "brand-a", 2026, 2, "pending"),
		reportRow("2026-03-04T00:00:00.000Z", "B000000001", "link-9", "https://lvnta.example/c", "Site", 6.5, 1, "amazon.com", "brand-z", 2026, 2, "pending"),
	}
	brands := []map[string]any{{"id": "brand-z", "name": "Missing Brand"}}
	rows := payoutGaps(invoices, reports, brands)
	if len(rows) != 1 {
		t.Fatalf("len = %d, rows = %+v", len(rows), rows)
	}
	if rows[0].Brand != "Missing Brand" || rows[0].Commission != 6.5 || rows[0].Month != 2 {
		t.Fatalf("row = %+v", rows[0])
	}
}

func TestStalledRequiresPriorCommissionAndQuietCurrentWindow(t *testing.T) {
	now := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	windows := newLedgerWindows(30, now)
	reports := []map[string]any{
		reportRow("2026-03-15T00:00:00.000Z", "B09G9FPHY6", "link-quiet", "https://lvnta.example/quiet", "Newsletter", 7, 3, "amazon.com", "brand-a", 2026, 2, "pending"),
		reportRow("2026-03-15T00:00:00.000Z", "B09G9FPHY6", "link-live", "https://lvnta.example/live", "Blog", 4, 1, "amazon.com", "brand-a", 2026, 2, "pending"),
		reportRow("2026-04-15T00:00:00.000Z", "B09G9FPHY6", "link-live", "https://lvnta.example/live", "Blog", 2, 1, "amazon.com", "brand-a", 2026, 3, "pending"),
		reportRow("2026-03-15T00:00:00.000Z", "B09G9FPHY6", "link-off", "https://lvnta.example/off", "Old", 9, 1, "amazon.com", "brand-a", 2026, 2, "pending"),
	}
	links := []map[string]any{
		{"id": "link-quiet", "active": true, "url": "https://lvnta.example/quiet", "sourceName": "Newsletter"},
		{"id": "link-live", "active": true},
		{"id": "link-off", "active": false},
	}
	rows := stalledLinks(reports, links, windows)
	if len(rows) != 1 {
		t.Fatalf("len = %d, rows = %+v", len(rows), rows)
	}
	if rows[0].LinkID != "link-quiet" || rows[0].PriorCommission != 7 || rows[0].Source != "Newsletter" {
		t.Fatalf("row = %+v", rows[0])
	}
}

func TestCreatorROIRanksDelta(t *testing.T) {
	now := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	windows := newLedgerWindows(30, now)
	reports := []map[string]any{
		{"date": "2026-04-10T00:00:00.000Z", "creatorId": "c1", "commissionAmount": 10.0, "sales": 100.0},
		{"date": "2026-03-10T00:00:00.000Z", "creatorId": "c1", "commissionAmount": 20.0, "sales": 100.0},
		{"date": "2026-04-12T00:00:00.000Z", "creatorId": "c2", "commissionAmount": "25.00", "sales": "50.00"},
	}
	creators := []map[string]any{{"id": "c1", "name": "Alex"}, {"id": "c2", "name": "Sam"}}
	rows := creatorROI(reports, creators, windows)
	if len(rows) != 2 {
		t.Fatalf("len = %d, rows = %+v", len(rows), rows)
	}
	if rows[0].Creator != "Sam" || rows[0].ROI != 0.5 {
		t.Fatalf("first = %+v", rows[0])
	}
	if rows[1].Creator != "Alex" || rows[1].ROI != 0.1 || rows[1].Delta != -0.1 {
		t.Fatalf("second = %+v", rows[1])
	}
}

func invoiceItem(name, id string, year, month int, status string, commission float64, marketplace string) map[string]any {
	return map[string]any{
		"brand":       map[string]any{"id": id, "name": name},
		"year":        float64(year),
		"month":       float64(month),
		"status":      status,
		"commission":  commission,
		"marketplace": marketplace,
		"currency":    "USD",
	}
}

func reportRow(date, asin, linkID, url, source string, commission, clicks float64, marketplace, brandID string, year, month int, status string) map[string]any {
	return map[string]any{
		"date":             date,
		"asin":             asin,
		"marketplace":      marketplace,
		"commissionEarned": commission,
		"clicks":           clicks,
		"sales":            commission * 10,
		"link": map[string]any{
			"id":         linkID,
			"url":        url,
			"sourceName": source,
			"active":     true,
		},
		"invoice": map[string]any{
			"brandId": brandID,
			"year":    float64(year),
			"month":   float64(month),
			"status":  status,
		},
	}
}
