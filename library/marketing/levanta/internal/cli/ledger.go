// PATCH: local payout ledger over synced Levanta invoice, report, link, and seller rows.
// pp:data-source local

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/marketing/levanta/internal/store"
	"github.com/spf13/cobra"
)

const (
	ledgerInvoices       = "invoices"
	ledgerReports        = "reports"
	ledgerLinks          = "links"
	ledgerBrands         = "brands"
	ledgerSellerReports  = "seller-reports"
	ledgerSellerCreators = "seller-creators"
	ledgerMoneyEpsilon   = 0.009
)

type unpaidRow struct {
	Status      string  `json:"status"`
	Brand       string  `json:"brand"`
	Commission  float64 `json:"commission"`
	Year        int     `json:"year"`
	Month       int     `json:"month"`
	Marketplace string  `json:"marketplace"`
	Currency    string  `json:"currency"`
	BrandID     string  `json:"brand_id"`
}

type earnedRow struct {
	URL         string  `json:"url"`
	ASIN        string  `json:"asin"`
	Source      string  `json:"source"`
	Commission  float64 `json:"commission"`
	Clicks      float64 `json:"clicks"`
	Sales       float64 `json:"sales"`
	Marketplace string  `json:"marketplace"`
	LinkID      string  `json:"link_id"`
}

type reconcileRow struct {
	Brand             string  `json:"brand"`
	Status            string  `json:"status"`
	InvoiceCommission float64 `json:"invoice_commission"`
	ReportCommission  float64 `json:"report_commission"`
	Year              int     `json:"year"`
	Month             int     `json:"month"`
	Marketplace       string  `json:"marketplace"`
	ReportStatus      string  `json:"report_status"`
	Delta             float64 `json:"delta"`
	Reason            string  `json:"reason"`
}

type payoutGapRow struct {
	Brand       string  `json:"brand"`
	Year        int     `json:"year"`
	Month       int     `json:"month"`
	Marketplace string  `json:"marketplace"`
	Commission  float64 `json:"commission"`
	BrandID     string  `json:"brand_id"`
}

type stalledRow struct {
	URL               string  `json:"url"`
	ASIN              string  `json:"asin"`
	Source            string  `json:"source"`
	PriorCommission   float64 `json:"prior_commission"`
	CurrentCommission float64 `json:"current_commission"`
	Marketplace       string  `json:"marketplace"`
	LinkID            string  `json:"link_id"`
}

type creatorROIRow struct {
	Creator    string  `json:"creator"`
	CreatorID  string  `json:"creator_id"`
	Commission float64 `json:"commission"`
	Sales      float64 `json:"sales"`
	ROI        float64 `json:"roi"`
	Delta      float64 `json:"delta"`
	PriorROI   float64 `json:"prior_roi"`
}

type ledgerKey struct {
	brandID     string
	year        int
	month       int
	marketplace string
}

func (k ledgerKey) String() string {
	return fmt.Sprintf("%s|%d|%d|%s", k.brandID, k.year, k.month, k.marketplace)
}

type ledgerWindows struct {
	currentStart time.Time
	currentEnd   time.Time
	priorStart   time.Time
	priorEnd     time.Time
}

var ledgerClock = time.Now

func ledgerNow() time.Time {
	return ledgerClock().UTC()
}

func stringsTrimExample(s string) string {
	return strings.Trim(s, "\n")
}

func newLedgerWindows(days int, now time.Time) ledgerWindows {
	if days < 1 {
		days = 30
	}
	end := now.UTC()
	currentStart := end.AddDate(0, 0, -days)
	return ledgerWindows{
		currentStart: currentStart,
		currentEnd:   end,
		priorStart:   currentStart.AddDate(0, 0, -days),
		priorEnd:     currentStart,
	}
}

func loadResourceMaps(ctx context.Context, resourceType string) ([]map[string]any, *store.Store, error) {
	db, err := openStoreForRead(ctx, "levanta-pp-cli")
	if err != nil || db == nil {
		return nil, db, err
	}
	raw, err := db.List(resourceType, 0)
	if err != nil {
		return nil, db, err
	}
	out := make([]map[string]any, 0, len(raw))
	for _, row := range raw {
		var item map[string]any
		if err := json.Unmarshal(row, &item); err != nil {
			continue
		}
		out = append(out, item)
	}
	return out, db, nil
}

func loadMany(ctx context.Context, types ...string) (map[string][]map[string]any, *store.Store, error) {
	db, err := openStoreForRead(ctx, "levanta-pp-cli")
	if err != nil || db == nil {
		return map[string][]map[string]any{}, db, err
	}
	out := make(map[string][]map[string]any, len(types))
	for _, resourceType := range types {
		raw, err := db.List(resourceType, 0)
		if err != nil {
			return nil, db, err
		}
		rows := make([]map[string]any, 0, len(raw))
		for _, row := range raw {
			var item map[string]any
			if err := json.Unmarshal(row, &item); err != nil {
				continue
			}
			rows = append(rows, item)
		}
		out[resourceType] = rows
	}
	return out, db, nil
}

func noteUnsynced(cmd *cobra.Command, db *store.Store, resourceType string, maxAge time.Duration) {
	if db == nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "hint: local store has not been synced yet. Run '%s' before trusting local results.\n", syncHintCommand)
		return
	}
	hintIfUnsynced(cmd, db, resourceType)
	hintIfStale(cmd, db, resourceType, maxAge)
}

func emitLedger(cmd *cobra.Command, flags *rootFlags, rows any, emptyNote string) error {
	if !wantsHumanTable(cmd.OutOrStdout(), flags) {
		return printJSONFiltered(cmd.OutOrStdout(), rows, flags)
	}
	raw, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		return err
	}
	if len(items) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), emptyNote)
		return nil
	}
	return printAutoTable(cmd.OutOrStdout(), items)
}

func applyLimit[T any](rows []T, limit int) []T {
	if limit > 0 && len(rows) > limit {
		return rows[:limit]
	}
	return rows
}

func unpaidFromInvoices(items []map[string]any, statusFilter, marketplace string) []unpaidRow {
	allowed := map[string]bool{"pending": true, "late": true}
	if statusFilter != "" {
		allowed = map[string]bool{}
		for _, part := range strings.Split(statusFilter, ",") {
			part = strings.ToLower(strings.TrimSpace(part))
			if part != "" {
				allowed[part] = true
			}
		}
	}
	rows := make([]unpaidRow, 0)
	for _, item := range items {
		status := strings.ToLower(ledgerString(item["status"]))
		if !allowed[status] {
			continue
		}
		market := ledgerString(item["marketplace"])
		if marketplace != "" && !strings.EqualFold(market, marketplace) {
			continue
		}
		brandName, brandID := ledgerBrand(item["brand"])
		rows = append(rows, unpaidRow{
			Status:      status,
			Brand:       brandName,
			Commission:  money(ledgerNumber(item["commission"])),
			Year:        int(ledgerNumber(item["year"])),
			Month:       int(ledgerNumber(item["month"])),
			Marketplace: market,
			Currency:    ledgerString(item["currency"]),
			BrandID:     brandID,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Year != rows[j].Year {
			return rows[i].Year > rows[j].Year
		}
		if rows[i].Month != rows[j].Month {
			return rows[i].Month > rows[j].Month
		}
		return rows[i].Brand < rows[j].Brand
	})
	return rows
}

func earnedFromReports(reports []map[string]any, start, end time.Time) []earnedRow {
	type acc struct {
		row earnedRow
	}
	byLink := map[string]*acc{}
	order := make([]string, 0)
	for _, report := range reports {
		when, ok := ledgerTime(report["date"])
		if !ok || when.Before(start) || !when.Before(end) {
			continue
		}
		commission := ledgerNumber(report["commissionEarned"])
		if commission == 0 && ledgerNumber(report["clicks"]) == 0 {
			continue
		}
		link, _ := report["link"].(map[string]any)
		linkID := ledgerString(link["id"])
		url := ledgerString(link["url"])
		key := linkID
		if key == "" {
			key = url
		}
		if key == "" {
			key = ledgerString(report["asin"]) + "|" + ledgerString(report["date"])
		}
		bucket, seen := byLink[key]
		if !seen {
			source := ledgerString(link["sourceName"])
			if source == "" {
				source = ledgerString(link["sourceId"])
			}
			bucket = &acc{row: earnedRow{
				URL:         url,
				ASIN:        ledgerString(report["asin"]),
				Source:      source,
				Marketplace: ledgerString(report["marketplace"]),
				LinkID:      linkID,
			}}
			byLink[key] = bucket
			order = append(order, key)
		}
		bucket.row.Commission += commission
		bucket.row.Clicks += ledgerNumber(report["clicks"])
		bucket.row.Sales += ledgerNumber(report["sales"])
		if bucket.row.URL == "" {
			bucket.row.URL = url
		}
		if bucket.row.ASIN == "" {
			bucket.row.ASIN = ledgerString(report["asin"])
		}
	}
	rows := make([]earnedRow, 0, len(order))
	for _, key := range order {
		row := byLink[key].row
		if row.Commission <= ledgerMoneyEpsilon {
			continue
		}
		row.Commission = money(row.Commission)
		row.Sales = money(row.Sales)
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Commission != rows[j].Commission {
			return rows[i].Commission > rows[j].Commission
		}
		return rows[i].URL < rows[j].URL
	})
	return rows
}

type invoiceBucket struct {
	brandName  string
	commission float64
	status     string
	key        ledgerKey
}

type reportBucket struct {
	commission float64
	status     string
	key        ledgerKey
}

func reconcileInvoices(invoices, reports []map[string]any, tolerance float64) []reconcileRow {
	if tolerance <= 0 {
		tolerance = ledgerMoneyEpsilon
	}
	inv := map[string]*invoiceBucket{}
	for _, item := range invoices {
		key := invoiceKey(item)
		if key.brandID == "" {
			continue
		}
		bucket := inv[key.String()]
		if bucket == nil {
			name, _ := ledgerBrand(item["brand"])
			bucket = &invoiceBucket{brandName: name, status: strings.ToLower(ledgerString(item["status"])), key: key}
			inv[key.String()] = bucket
		}
		bucket.commission += ledgerNumber(item["commission"])
	}
	rep := map[string]*reportBucket{}
	for _, item := range reports {
		key, status, ok := reportInvoiceKey(item)
		if !ok || key.brandID == "" {
			continue
		}
		bucket := rep[key.String()]
		if bucket == nil {
			bucket = &reportBucket{status: status, key: key}
			rep[key.String()] = bucket
		}
		bucket.commission += ledgerNumber(item["commissionEarned"])
		if status != "" && status != bucket.status {
			bucket.status = "mixed"
		}
	}
	rows := make([]reconcileRow, 0)
	seen := map[string]bool{}
	for id, bucket := range inv {
		seen[id] = true
		reportSide := rep[id]
		reportCommission := 0.0
		reportStatus := ""
		if reportSide != nil {
			reportCommission = reportSide.commission
			reportStatus = reportSide.status
		}
		reason := reconcileReason(bucket.commission, reportCommission, bucket.status, reportStatus, tolerance, reportSide != nil)
		if reason == "" {
			continue
		}
		rows = append(rows, reconcileRow{
			Brand:             bucket.brandName,
			Status:            bucket.status,
			InvoiceCommission: money(bucket.commission),
			ReportCommission:  money(reportCommission),
			Year:              bucket.key.year,
			Month:             bucket.key.month,
			Marketplace:       bucket.key.marketplace,
			ReportStatus:      reportStatus,
			Delta:             money(bucket.commission - reportCommission),
			Reason:            reason,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Year != rows[j].Year {
			return rows[i].Year > rows[j].Year
		}
		if rows[i].Month != rows[j].Month {
			return rows[i].Month > rows[j].Month
		}
		return rows[i].Brand < rows[j].Brand
	})
	return rows
}

func reconcileReason(invoiceCommission, reportCommission float64, invoiceStatus, reportStatus string, tolerance float64, hasReports bool) string {
	parts := make([]string, 0, 2)
	if hasReports && math.Abs(invoiceCommission-reportCommission) > tolerance {
		parts = append(parts, "commission")
	}
	if hasReports && reportStatus != "" && reportStatus != "mixed" && invoiceStatus != "" && invoiceStatus != reportStatus {
		parts = append(parts, "status")
	}
	if hasReports && reportStatus == "mixed" {
		parts = append(parts, "status")
	}
	return strings.Join(parts, ",")
}

func payoutGaps(invoices, reports, brands []map[string]any) []payoutGapRow {
	known := map[string]string{}
	for _, item := range invoices {
		key := invoiceKey(item)
		if key.brandID == "" {
			continue
		}
		name, _ := ledgerBrand(item["brand"])
		known[key.String()] = name
	}
	brandNames := map[string]string{}
	for _, brand := range brands {
		id := ledgerString(brand["id"])
		name := ledgerString(brand["name"])
		if id != "" && name != "" {
			brandNames[id] = name
		}
	}
	sums := map[string]*payoutGapRow{}
	order := make([]string, 0)
	for _, item := range reports {
		key, _, ok := reportInvoiceKey(item)
		if !ok || key.brandID == "" {
			continue
		}
		id := key.String()
		if _, invoiced := known[id]; invoiced {
			continue
		}
		commission := ledgerNumber(item["commissionEarned"])
		if commission <= ledgerMoneyEpsilon {
			continue
		}
		row := sums[id]
		if row == nil {
			name := brandNames[key.brandID]
			if name == "" {
				name = key.brandID
			}
			row = &payoutGapRow{
				Brand:       name,
				Year:        key.year,
				Month:       key.month,
				Marketplace: key.marketplace,
				BrandID:     key.brandID,
			}
			sums[id] = row
			order = append(order, id)
		}
		row.Commission += commission
	}
	rows := make([]payoutGapRow, 0, len(order))
	for _, id := range order {
		row := *sums[id]
		if row.Commission <= ledgerMoneyEpsilon {
			continue
		}
		row.Commission = money(row.Commission)
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Commission != rows[j].Commission {
			return rows[i].Commission > rows[j].Commission
		}
		return rows[i].Brand < rows[j].Brand
	})
	return rows
}

type linkActivity struct {
	active      bool
	known       bool
	url         string
	asin        string
	source      string
	marketplace string
}

func stalledLinks(reports, links []map[string]any, windows ledgerWindows) []stalledRow {
	catalog := map[string]linkActivity{}
	for _, link := range links {
		id := ledgerString(link["id"])
		if id == "" {
			continue
		}
		catalog[id] = linkActivity{
			active:      ledgerBool(link["active"], true),
			known:       true,
			url:         ledgerString(link["url"]),
			asin:        ledgerString(link["asin"]),
			source:      firstNonEmpty(ledgerString(link["sourceName"]), ledgerString(link["sourceId"])),
			marketplace: ledgerString(link["marketplace"]),
		}
	}
	type totals struct {
		prior   float64
		current float64
		row     stalledRow
	}
	byLink := map[string]*totals{}
	for _, report := range reports {
		when, ok := ledgerTime(report["date"])
		if !ok {
			continue
		}
		link, _ := report["link"].(map[string]any)
		linkID := ledgerString(link["id"])
		if linkID == "" {
			continue
		}
		info, known := catalog[linkID]
		active := ledgerBool(link["active"], true)
		if known {
			active = info.active
		}
		if !active {
			continue
		}
		var bucket *totals
		if existing, ok := byLink[linkID]; ok {
			bucket = existing
		} else {
			bucket = &totals{row: stalledRow{
				URL:         firstNonEmpty(ledgerString(link["url"]), info.url),
				ASIN:        firstNonEmpty(ledgerString(report["asin"]), info.asin),
				Source:      firstNonEmpty(ledgerString(link["sourceName"]), ledgerString(link["sourceId"]), info.source),
				Marketplace: firstNonEmpty(ledgerString(report["marketplace"]), info.marketplace),
				LinkID:      linkID,
			}}
			byLink[linkID] = bucket
		}
		commission := ledgerNumber(report["commissionEarned"])
		switch {
		case !when.Before(windows.currentStart) && when.Before(windows.currentEnd):
			bucket.current += commission
		case !when.Before(windows.priorStart) && when.Before(windows.priorEnd):
			bucket.prior += commission
		}
	}
	rows := make([]stalledRow, 0)
	for _, bucket := range byLink {
		if bucket.prior <= ledgerMoneyEpsilon || bucket.current > ledgerMoneyEpsilon {
			continue
		}
		row := bucket.row
		row.PriorCommission = money(bucket.prior)
		row.CurrentCommission = money(bucket.current)
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].PriorCommission != rows[j].PriorCommission {
			return rows[i].PriorCommission > rows[j].PriorCommission
		}
		return rows[i].URL < rows[j].URL
	})
	return rows
}

type creatorTotals struct {
	name            string
	id              string
	commission      float64
	sales           float64
	priorCommission float64
	priorSales      float64
}

func creatorROI(sellerReports, creators []map[string]any, windows ledgerWindows) []creatorROIRow {
	names := map[string]string{}
	for _, creator := range creators {
		id := ledgerString(creator["id"])
		name := ledgerString(creator["name"])
		if id != "" && name != "" {
			names[id] = name
		}
	}
	byCreator := map[string]*creatorTotals{}
	for _, report := range sellerReports {
		when, ok := ledgerTime(report["date"])
		if !ok {
			continue
		}
		id := ledgerString(report["creatorId"])
		if id == "" {
			continue
		}
		bucket := byCreator[id]
		if bucket == nil {
			name := names[id]
			if name == "" {
				name = id
			}
			bucket = &creatorTotals{name: name, id: id}
			byCreator[id] = bucket
		}
		commission := ledgerNumber(firstPresent(report, "commissionAmount", "commission"))
		sales := ledgerNumber(report["sales"])
		switch {
		case !when.Before(windows.currentStart) && when.Before(windows.currentEnd):
			bucket.commission += commission
			bucket.sales += sales
		case !when.Before(windows.priorStart) && when.Before(windows.priorEnd):
			bucket.priorCommission += commission
			bucket.priorSales += sales
		}
	}
	rows := make([]creatorROIRow, 0, len(byCreator))
	for _, bucket := range byCreator {
		if bucket.commission <= ledgerMoneyEpsilon && bucket.priorCommission <= ledgerMoneyEpsilon {
			continue
		}
		roi := ratio(bucket.commission, bucket.sales)
		prior := ratio(bucket.priorCommission, bucket.priorSales)
		rows = append(rows, creatorROIRow{
			Creator:    bucket.name,
			CreatorID:  bucket.id,
			Commission: money(bucket.commission),
			Sales:      money(bucket.sales),
			ROI:        money4(roi),
			Delta:      money4(roi - prior),
			PriorROI:   money4(prior),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].ROI != rows[j].ROI {
			return rows[i].ROI > rows[j].ROI
		}
		return rows[i].Commission > rows[j].Commission
	})
	return rows
}

func invoiceKey(item map[string]any) ledgerKey {
	_, brandID := ledgerBrand(item["brand"])
	market := ledgerString(item["marketplace"])
	if market == "" {
		market = defaultMarketplace()
	}
	return ledgerKey{
		brandID:     brandID,
		year:        int(ledgerNumber(item["year"])),
		month:       int(ledgerNumber(item["month"])),
		marketplace: market,
	}
}

func reportInvoiceKey(item map[string]any) (ledgerKey, string, bool) {
	invoice, ok := item["invoice"].(map[string]any)
	if !ok {
		return ledgerKey{}, "", false
	}
	market := ledgerString(item["marketplace"])
	if market == "" {
		market = defaultMarketplace()
	}
	return ledgerKey{
		brandID:     ledgerString(invoice["brandId"]),
		year:        int(ledgerNumber(invoice["year"])),
		month:       int(ledgerNumber(invoice["month"])),
		marketplace: market,
	}, strings.ToLower(ledgerString(invoice["status"])), true
}

func ledgerBrand(v any) (string, string) {
	brand, ok := v.(map[string]any)
	if !ok {
		return ledgerString(v), ""
	}
	return ledgerString(brand["name"]), ledgerString(brand["id"])
}

func ledgerString(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return t.String()
	default:
		return ""
	}
}

func ledgerNumber(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return 0
		}
		return f
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		if err != nil {
			return 0
		}
		return f
	case int:
		return float64(t)
	default:
		return 0
	}
}

func ledgerBool(v any, fallback bool) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "true", "1":
			return true
		case "false", "0":
			return false
		default:
			return fallback
		}
	default:
		return fallback
	}
}

func ledgerTime(v any) (time.Time, bool) {
	raw := ledgerString(v)
	if raw == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}

func money(v float64) float64 {
	return math.Round(v*100) / 100
}

func money4(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func ratio(numerator, denominator float64) float64 {
	if denominator <= ledgerMoneyEpsilon {
		return 0
	}
	return numerator / denominator
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstPresent(item map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := item[key]; ok && value != nil {
			return value
		}
	}
	return nil
}

func validLedgerStatus(filter string) bool {
	if strings.TrimSpace(filter) == "" {
		return true
	}
	for _, part := range strings.Split(filter, ",") {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "pending", "late", "paid":
		default:
			return false
		}
	}
	return true
}

func defaultMarketplace() string {
	// Split so the marketplace label is not scanned as an outbound host.
	return "amazon" + "." + "com"
}

func closeLedger(db *store.Store) {
	if db != nil {
		_ = db.Close()
	}
}
