// Package fidelitycsv parses the positions and history CSV files an owner
// downloads from Fidelity Investments. It does not contact the network.
package fidelitycsv

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// AccountID matches the account numbers printed in Fidelity exports.
var AccountID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{2,31}$`)

// Position is one holding row from a positions export.
type Position struct {
	AccountNumber    string `json:"account_number"`
	AccountName      string `json:"account_name"`
	Symbol           string `json:"symbol"`
	Description      string `json:"description"`
	Quantity         string `json:"quantity"`
	LastPrice        string `json:"last_price"`
	CurrentValue     string `json:"current_value"`
	CostBasis        string `json:"cost_basis"`
	TotalGainLoss    string `json:"total_gain_loss"`
	PercentOfAccount string `json:"percent_of_account"`
	Type             string `json:"type"`
}

// Transaction is one activity row from a history export.
type Transaction struct {
	AccountNumber string `json:"account_number"`
	RunDate       string `json:"run_date"`
	Action        string `json:"action"`
	Symbol        string `json:"symbol"`
	Description   string `json:"description"`
	Quantity      string `json:"quantity"`
	Amount        string `json:"amount"`
	Commission    string `json:"commission"`
	Fees          string `json:"fees"`
	CashBalance   string `json:"cash_balance,omitempty"`
}

// HistoryFile is a parsed activity export.
type HistoryFile struct {
	Rows               []Transaction
	CashBalancePresent bool
}

// SHA256 returns the hex digest used as the import idempotency key.
func SHA256(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// ParsePositions reads a Portfolio Positions CSV, skipping the disclaimer
// footer and any pending-activity section that has no account number.
func ParsePositions(raw []byte) ([]Position, error) {
	records, err := readRecords(raw)
	if err != nil {
		return nil, err
	}
	headerAt := -1
	var index map[string]int
	for i, row := range records {
		idx := headerIndex(row)
		if _, ok := idx["account number"]; ok {
			if _, sym := idx["symbol"]; sym {
				headerAt = i
				index = idx
				break
			}
		}
	}
	if headerAt < 0 {
		return nil, fmt.Errorf("positions CSV has no header row containing Account Number and Symbol")
	}
	var out []Position
	for _, row := range records[headerAt+1:] {
		account := field(row, index, "account number")
		if !AccountID.MatchString(account) {
			continue
		}
		symbol := field(row, index, "symbol")
		typ := field(row, index, "type")
		if symbol == "" && !isCashType(typ) {
			continue
		}
		out = append(out, Position{
			AccountNumber:    account,
			AccountName:      field(row, index, "account name"),
			Symbol:           symbol,
			Description:      field(row, index, "description"),
			Quantity:         field(row, index, "quantity"),
			LastPrice:        field(row, index, "last price"),
			CurrentValue:     field(row, index, "current value"),
			CostBasis:        firstField(row, index, "cost basis total", "cost basis"),
			TotalGainLoss:    firstField(row, index, "total gain loss dollar", "total gain loss"),
			PercentOfAccount: field(row, index, "percent of account"),
			Type:             typ,
		})
	}
	return out, nil
}

// ParseHistory reads an Activity & Orders history CSV. account is used when
// the file has no account column. The caller should pass the id from
// History_for_Account_<id>.csv or --account.
func ParseHistory(raw []byte, account string) (HistoryFile, error) {
	records, err := readRecords(raw)
	if err != nil {
		return HistoryFile{}, err
	}
	headerAt := -1
	var index map[string]int
	for i, row := range records {
		idx := headerIndex(row)
		if _, ok := idx["run date"]; ok {
			if _, action := idx["action"]; action {
				headerAt = i
				index = idx
				break
			}
		}
	}
	if headerAt < 0 {
		return HistoryFile{}, fmt.Errorf("history CSV has no header row containing Run Date and Action")
	}
	_, cashPresent := index["cash balance"]
	var out []Transaction
	for _, row := range records[headerAt+1:] {
		runDate, ok := NormalizeDate(field(row, index, "run date"))
		if !ok {
			continue
		}
		rowAccount := firstField(row, index, "account number", "account")
		if rowAccount == "" {
			rowAccount = account
		}
		if !AccountID.MatchString(rowAccount) {
			continue
		}
		tx := Transaction{
			AccountNumber: rowAccount,
			RunDate:       runDate,
			Action:        field(row, index, "action"),
			Symbol:        field(row, index, "symbol"),
			Description:   field(row, index, "description"),
			Quantity:      field(row, index, "quantity"),
			Amount:        firstField(row, index, "amount", "amount dollar"),
			Commission:    field(row, index, "commission"),
			Fees:          field(row, index, "fees"),
		}
		if cashPresent {
			tx.CashBalance = field(row, index, "cash balance")
		}
		if tx.Action == "" && tx.Symbol == "" && tx.Amount == "" {
			continue
		}
		out = append(out, tx)
	}
	return HistoryFile{Rows: out, CashBalancePresent: cashPresent}, nil
}

// AccountFromHistoryFilename returns the id embedded in
// History_for_Account_<id>.csv. The second result is false when the name
// does not carry an id.
func AccountFromHistoryFilename(name string) (string, bool) {
	base := name
	if i := strings.LastIndexAny(base, `/\`); i >= 0 {
		base = base[i+1:]
	}
	base = strings.TrimSuffix(base, ".csv")
	base = strings.TrimSuffix(base, ".CSV")
	const prefix = "history_for_account_"
	lower := strings.ToLower(base)
	if !strings.HasPrefix(lower, prefix) {
		return "", false
	}
	id := base[len(prefix):]
	if !AccountID.MatchString(id) {
		return "", false
	}
	return id, true
}

// NormalizeDate accepts the date spellings Fidelity prints and returns
// YYYY-MM-DD.
func NormalizeDate(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	layouts := []string{
		"2006-01-02",
		"01/02/2006",
		"1/2/2006",
		"01-02-2006",
		"1-2-2006",
		"Jan-02-2006",
		"Jan 2, 2006",
		"January 2, 2006",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("2006-01-02"), true
		}
	}
	return "", false
}

// ParseDecimal reads Fidelity money and quantity cells. The boolean is false
// when the cell is blank or a placeholder such as n/a.
func ParseDecimal(raw string) (*big.Rat, bool) {
	s := strings.TrimSpace(raw)
	if s == "" || s == "--" || s == "—" || strings.EqualFold(s, "n/a") || strings.EqualFold(s, "na") {
		return new(big.Rat), false
	}
	neg := false
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		neg = true
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	s = strings.TrimPrefix(s, "+")
	if strings.HasPrefix(s, "-") {
		neg = !neg
		s = strings.TrimSpace(s[1:])
	}
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	s = strings.TrimSpace(s)
	r := new(big.Rat)
	if _, ok := r.SetString(s); !ok {
		return new(big.Rat), false
	}
	if neg {
		r.Neg(r)
	}
	return r, true
}

// FormatMoney renders a sum with two decimal places.
func FormatMoney(r *big.Rat) string {
	if r == nil {
		r = new(big.Rat)
	}
	return new(big.Rat).Set(r).FloatString(2)
}

// FormatQuantity renders a share count without trailing zeros.
func FormatQuantity(r *big.Rat) string {
	if r == nil {
		return "0"
	}
	s := new(big.Rat).Set(r).FloatString(8)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

func isCashType(typ string) bool {
	return strings.Contains(strings.ToLower(typ), "cash")
}

func readRecords(raw []byte) ([][]string, error) {
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	if !utf8.Valid(raw) {
		return nil, fmt.Errorf("CSV is not valid UTF-8")
	}
	reader := csv.NewReader(bytes.NewReader(raw))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	var records [][]string
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading CSV: %w", err)
		}
		records = append(records, row)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("CSV is empty")
	}
	return records, nil
}

func headerIndex(row []string) map[string]int {
	idx := map[string]int{}
	for i, cell := range row {
		key := normalizeHeader(cell)
		if key == "" {
			continue
		}
		if _, exists := idx[key]; exists {
			continue
		}
		idx[key] = i
	}
	return idx
}

func normalizeHeader(cell string) string {
	cell = strings.ToLower(strings.TrimSpace(cell))
	replacer := strings.NewReplacer("$", "", "/", " ", "(", " ", ")", " ", "%", "", ",", " ", "_", " ", "-", " ")
	cell = replacer.Replace(cell)
	fields := strings.Fields(cell)
	return strings.Join(fields, " ")
}

func field(row []string, index map[string]int, name string) string {
	i, ok := index[name]
	if !ok || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func firstField(row []string, index map[string]int, names ...string) string {
	for _, name := range names {
		if value := field(row, index, name); value != "" {
			return value
		}
	}
	return ""
}
