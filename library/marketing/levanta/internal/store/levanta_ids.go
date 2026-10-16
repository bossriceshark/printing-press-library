// PATCH: stable storage keys for Levanta rows that have no unique id.
// Invoice items are one row per brand, year, month, and marketplace.
// Creator report rows are one row per date, ASIN, marketplace, and link.
// Seller performance-by-creator (v2) rows name a creator and carry no id.

package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
)

// levantaStableRowID returns a storage key for Levanta collections whose
// generated id field is not unique. An empty string means the generated
// extractor should run instead.
func levantaStableRowID(resourceType string, obj map[string]any) string {
	if obj == nil {
		return ""
	}
	switch resourceType {
	case "invoices":
		return levantaInvoiceID(obj)
	case "reports", "reports-cpc":
		return levantaReportID(obj)
	case "seller-reports-2":
		return levantaSellerReportV2ID(obj)
	default:
		return ""
	}
}

func levantaInvoiceID(obj map[string]any) string {
	brandID := levantaNestedString(obj, "brand", "id")
	year := levantaScalar(obj["year"])
	month := levantaScalar(obj["month"])
	if brandID == "" || year == "" || month == "" {
		return ""
	}
	marketplace := levantaScalar(obj["marketplace"])
	if marketplace == "" {
		marketplace = "amazon.com"
	}
	return strings.Join([]string{"inv", brandID, year, month, marketplace}, ":")
}

func levantaReportID(obj map[string]any) string {
	date := levantaScalar(obj["date"])
	asin := levantaScalar(obj["asin"])
	if date == "" || asin == "" {
		return ""
	}
	marketplace := levantaScalar(obj["marketplace"])
	linkID := levantaNestedString(obj, "link", "id")
	return strings.Join([]string{"rpt", date, asin, marketplace, linkID}, ":")
}

func levantaSellerReportV2ID(obj map[string]any) string {
	// Rows that already carry an id keep the generated extractor.
	if levantaScalar(obj["id"]) != "" {
		return ""
	}
	creatorID := levantaNestedString(obj, "creator", "id")
	if creatorID == "" {
		raw, err := json.Marshal(obj["metrics"])
		if err != nil || len(raw) == 0 {
			return ""
		}
		sum := sha256.Sum256(raw)
		creatorID = "unattributed-" + hex.EncodeToString(sum[:8])
	}
	return "s2:" + creatorID
}

func levantaNestedString(obj map[string]any, keys ...string) string {
	var cur any = obj
	for _, key := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur, ok = m[key]
		if !ok {
			return ""
		}
	}
	return levantaScalar(cur)
}

func levantaScalar(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return strings.TrimSpace(t.String())
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	default:
		return ""
	}
}
