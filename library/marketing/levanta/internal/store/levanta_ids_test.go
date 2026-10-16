package store

import "testing"

func TestLevantaInvoiceIDKeepsJanuaryAndDistinctBrands(t *testing.T) {
	january := map[string]any{
		"brand":       map[string]any{"id": "brand-a", "name": "Acme"},
		"year":        float64(2026),
		"month":       float64(0),
		"marketplace": "amazon.com",
	}
	other := map[string]any{
		"brand":       map[string]any{"id": "brand-b", "name": "Other"},
		"year":        float64(2026),
		"month":       float64(0),
		"marketplace": "amazon.com",
	}
	a := ExtractResourceID("invoices", january)
	b := ExtractResourceID("invoices", other)
	if a == "" || b == "" {
		t.Fatalf("ids = %q %q", a, b)
	}
	if a == b {
		t.Fatalf("distinct January invoices collapsed to %q", a)
	}
	if a != "inv:brand-a:2026:0:amazon.com" {
		t.Fatalf("january id = %q", a)
	}
}

func TestLevantaReportIDIncludesDateAndLink(t *testing.T) {
	day1 := map[string]any{
		"date": "2026-03-01T00:00:00.000Z",
		"asin": "B09G9FPHY6",
		"link": map[string]any{"id": "link-1"},
	}
	day2 := map[string]any{
		"date": "2026-03-02T00:00:00.000Z",
		"asin": "B09G9FPHY6",
		"link": map[string]any{"id": "link-1"},
	}
	a := ExtractResourceID("reports", day1)
	b := ExtractResourceID("reports", day2)
	if a == b {
		t.Fatalf("distinct report days collapsed to %q", a)
	}
}

func TestLevantaSellerReportV2KeepsExplicitID(t *testing.T) {
	row := map[string]any{"id": "test-001"}
	if got := ExtractResourceID("seller-reports-2", row); got != "test-001" {
		t.Fatalf("id = %q", got)
	}
}

func TestLevantaSellerReportV2IDUsesCreator(t *testing.T) {
	row := map[string]any{
		"creator": map[string]any{"id": "creator-1", "name": "Sam"},
		"metrics": map[string]any{"sales": "10.00", "commission": "1.00"},
	}
	if got := ExtractResourceID("seller-reports-2", row); got != "s2:creator-1" {
		t.Fatalf("id = %q", got)
	}
}
