// Package ledger is the read-only Fidelity book built from owner CSV exports.
package ledger

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/payments/fidelity/internal/fidelitycsv"
	"github.com/mvanhorn/printing-press-library/library/payments/fidelity/internal/store"
)

const (
	resourcePositions = "positions"
	resourceAccounts  = "accounts"
	resourceTxns      = "transactions"
	resourceImports   = "fidelity_imports"
	resourcePrevious  = "fidelity_position_previous"
	resourceLatest    = "fidelity_latest"
)

// ImportResult is the agent-facing report for one CSV import.
type ImportResult struct {
	Kind             string   `json:"kind"`
	SHA256           string   `json:"sha256"`
	Noop             bool     `json:"noop"`
	Rows             int      `json:"rows"`
	Accounts         []string `json:"accounts"`
	ReplacedAccounts []string `json:"replaced_accounts,omitempty"`
	Message          string   `json:"message,omitempty"`
}

// AccountTotal is one account inside a portfolio summary.
type AccountTotal struct {
	AccountNumber      string `json:"account_number"`
	AccountName        string `json:"account_name"`
	MarketValue        string `json:"market_value"`
	CashValue          string `json:"cash_value"`
	InvestedValue      string `json:"invested_value"`
	UnrealizedGainLoss string `json:"unrealized_gain_loss"`
	HistoryCashBalance string `json:"history_cash_balance,omitempty"`
	HistoryCashAsOf    string `json:"history_cash_as_of,omitempty"`
}

// BookTotal is the whole imported book.
type BookTotal struct {
	MarketValue        string `json:"market_value"`
	CashValue          string `json:"cash_value"`
	InvestedValue      string `json:"invested_value"`
	UnrealizedGainLoss string `json:"unrealized_gain_loss"`
	HistoryCashBalance string `json:"history_cash_balance,omitempty"`
}

// LargestPosition is one row in the portfolio summary.
type LargestPosition struct {
	AccountNumber string `json:"account_number"`
	Symbol        string `json:"symbol"`
	Description   string `json:"description"`
	MarketValue   string `json:"market_value"`
	Quantity      string `json:"quantity"`
}

// PortfolioSummary is the book-level read model.
type PortfolioSummary struct {
	Book             BookTotal         `json:"book"`
	Accounts         []AccountTotal    `json:"accounts"`
	LargestPositions []LargestPosition `json:"largest_positions"`
	Note             string            `json:"note,omitempty"`
}

// ActionTotal is one action bucket in an activity summary.
type ActionTotal struct {
	Action     string `json:"action"`
	Rows       int    `json:"rows"`
	Amount     string `json:"amount"`
	Commission string `json:"commission"`
	Fees       string `json:"fees"`
}

// ActivitySummary totals imported history.
type ActivitySummary struct {
	From       string        `json:"from,omitempty"`
	To         string        `json:"to,omitempty"`
	Action     string        `json:"action,omitempty"`
	Rows       int           `json:"rows"`
	Amount     string        `json:"amount"`
	Commission string        `json:"commission"`
	Fees       string        `json:"fees"`
	ByAction   []ActionTotal `json:"by_action"`
	Note       string        `json:"note,omitempty"`
}

// PositionDelta is one holding that moved between snapshots.
type PositionDelta struct {
	AccountNumber string `json:"account_number"`
	Symbol        string `json:"symbol"`
	Description   string `json:"description"`
	Quantity      string `json:"quantity"`
	PreviousQty   string `json:"previous_quantity,omitempty"`
	CurrentValue  string `json:"current_value,omitempty"`
}

// HistoryChange is one activity row from the latest history import.
type HistoryChange struct {
	AccountNumber string `json:"account_number"`
	RunDate       string `json:"run_date"`
	Action        string `json:"action"`
	Symbol        string `json:"symbol"`
	Amount        string `json:"amount"`
}

// Changes is the delta since the previous positions import.
type Changes struct {
	Account         string          `json:"account,omitempty"`
	Appeared        []PositionDelta `json:"appeared"`
	Disappeared     []PositionDelta `json:"disappeared"`
	QuantityChanged []PositionDelta `json:"quantity_changed"`
	HistoryRows     []HistoryChange `json:"history_rows"`
	Note            string          `json:"note,omitempty"`
}

// SymbolAccount is one account's slice of a symbol total.
type SymbolAccount struct {
	AccountNumber      string `json:"account_number"`
	AccountName        string `json:"account_name"`
	Quantity           string `json:"quantity"`
	MarketValue        string `json:"market_value"`
	CostBasis          string `json:"cost_basis"`
	UnrealizedGainLoss string `json:"unrealized_gain_loss"`
}

// SymbolTotal sums one ticker across accounts.
type SymbolTotal struct {
	Symbol             string          `json:"symbol"`
	Quantity           string          `json:"quantity"`
	MarketValue        string          `json:"market_value"`
	CostBasis          string          `json:"cost_basis"`
	UnrealizedGainLoss string          `json:"unrealized_gain_loss"`
	Accounts           []SymbolAccount `json:"accounts"`
	Note               string          `json:"note,omitempty"`
}

// CoverageAccount says which exports exist for one account.
type CoverageAccount struct {
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
	HasPositions  bool   `json:"has_positions"`
	HasHistory    bool   `json:"has_history"`
	HistoryFrom   string `json:"history_from,omitempty"`
	HistoryTo     string `json:"history_to,omitempty"`
}

// Coverage is the per-account export inventory.
type Coverage struct {
	Accounts []CoverageAccount `json:"accounts"`
	Note     string            `json:"note,omitempty"`
}

type storedPosition struct {
	ID               string `json:"id"`
	Account          string `json:"account"`
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

type storedAccount struct {
	ID                 string `json:"id"`
	Account            string `json:"account"`
	AccountNumber      string `json:"account_number"`
	AccountName        string `json:"account_name"`
	HistoryCashBalance string `json:"history_cash_balance,omitempty"`
	HistoryCashAsOf    string `json:"history_cash_as_of,omitempty"`
}

type storedTxn struct {
	ID            string `json:"id"`
	Account       string `json:"account"`
	AccountNumber string `json:"account_number"`
	RunDate       string `json:"run_date"`
	Action        string `json:"action"`
	Symbol        string `json:"symbol"`
	Description   string `json:"description"`
	Quantity      string `json:"quantity"`
	Amount        string `json:"amount"`
	Commission    string `json:"commission"`
	Fees          string `json:"fees"`
	ImportSHA     string `json:"import_sha"`
}

type importRecord struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	SHA256   string   `json:"sha256"`
	At       string   `json:"imported_at"`
	Accounts []string `json:"accounts"`
	Rows     int      `json:"rows"`
}

type previousSnapshot struct {
	ID        string           `json:"id"`
	Account   string           `json:"account"`
	Positions []storedPosition `json:"positions"`
}

type latestPointer struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	SHA256 string `json:"sha256"`
}

// ImportPositions loads a positions CSV. The same file bytes are a no-op.
// A new file replaces each account in the file and keeps the prior snapshot
// for changes.
func ImportPositions(db *store.Store, filename string, raw []byte) (ImportResult, error) {
	sum := fidelitycsv.SHA256(raw)
	if seen, err := importSeen(db, sum); err != nil {
		return ImportResult{}, err
	} else if seen {
		return ImportResult{Kind: "positions", SHA256: sum, Noop: true, Message: "positions file already imported"}, nil
	}
	parsed, err := fidelitycsv.ParsePositions(raw)
	if err != nil {
		return ImportResult{}, err
	}
	if len(parsed) == 0 {
		return ImportResult{}, fmt.Errorf("positions file %s has a header but no account rows; disclaimer footers are skipped", filename)
	}
	grouped := collapsePositions(parsed)
	names := map[string]string{}
	for _, row := range parsed {
		if row.AccountName != "" {
			names[row.AccountNumber] = row.AccountName
		}
	}
	var accounts []string
	var replaced []string
	var batch []json.RawMessage
	keep := map[string]map[string]struct{}{}
	prior := map[string][]storedPosition{}
	for account, rows := range grouped {
		accounts = append(accounts, account)
		keep[account] = map[string]struct{}{}
		current, err := positionsForAccount(db, account)
		if err != nil {
			return ImportResult{}, err
		}
		if len(current) > 0 {
			prior[account] = current
			replaced = append(replaced, account)
		}
		for _, row := range rows {
			keep[account][row.ID] = struct{}{}
			rawRow, err := json.Marshal(row)
			if err != nil {
				return ImportResult{}, err
			}
			batch = append(batch, rawRow)
		}
	}
	for account, rows := range prior {
		if err := savePrevious(db, account, rows); err != nil {
			return ImportResult{}, err
		}
	}
	if _, _, err := db.UpsertBatch(resourcePositions, batch); err != nil {
		return ImportResult{}, err
	}
	for account := range grouped {
		current, err := positionsForAccount(db, account)
		if err != nil {
			return ImportResult{}, err
		}
		var stale []string
		for _, row := range current {
			if _, ok := keep[account][row.ID]; !ok {
				stale = append(stale, row.ID)
			}
		}
		if len(stale) > 0 {
			if err := db.DeleteIDs(resourcePositions, stale); err != nil {
				return ImportResult{}, err
			}
		}
		if err := upsertAccount(db, storedAccount{ID: account, Account: account, AccountNumber: account, AccountName: names[account]}); err != nil {
			return ImportResult{}, err
		}
	}
	sort.Strings(accounts)
	sort.Strings(replaced)
	rec := importRecord{ID: sum, Kind: "positions", SHA256: sum, At: time.Now().UTC().Format(time.RFC3339), Accounts: accounts, Rows: len(batch)}
	if err := putJSON(db, resourceImports, sum, rec); err != nil {
		return ImportResult{}, err
	}
	if err := db.SaveSyncState(resourcePositions, sum, len(batch)); err != nil {
		return ImportResult{}, err
	}
	if err := db.SaveSyncState(resourceAccounts, sum, len(accounts)); err != nil {
		return ImportResult{}, err
	}
	return ImportResult{Kind: "positions", SHA256: sum, Rows: len(batch), Accounts: accounts, ReplacedAccounts: replaced}, nil
}

// ImportHistory loads an activity CSV. accountOverride wins over the filename.
func ImportHistory(db *store.Store, filename, accountOverride string, raw []byte) (ImportResult, error) {
	sum := fidelitycsv.SHA256(raw)
	if seen, err := importSeen(db, sum); err != nil {
		return ImportResult{}, err
	} else if seen {
		return ImportResult{Kind: "history", SHA256: sum, Noop: true, Message: "history file already imported"}, nil
	}
	account := strings.TrimSpace(accountOverride)
	if account == "" {
		if id, ok := fidelitycsv.AccountFromHistoryFilename(filename); ok {
			account = id
		}
	}
	parsed, err := fidelitycsv.ParseHistory(raw, account)
	if err != nil {
		return ImportResult{}, err
	}
	if len(parsed.Rows) == 0 {
		return ImportResult{}, fmt.Errorf("history file %s has a header but no activity rows; pass --account when the filename has no account id", filename)
	}
	seenRow := map[string]int{}
	var batch []json.RawMessage
	accounts := map[string]struct{}{}
	cashByAccount := map[string]storedAccount{}
	for _, row := range parsed.Rows {
		key := row.AccountNumber + "|" + row.RunDate + "|" + row.Action + "|" + row.Symbol + "|" + row.Amount + "|" + row.Quantity
		seenRow[key]++
		id := rowID(key + "|" + fmt.Sprint(seenRow[key]))
		item := storedTxn{
			ID: id, Account: row.AccountNumber, AccountNumber: row.AccountNumber, RunDate: row.RunDate,
			Action: row.Action, Symbol: row.Symbol, Description: row.Description, Quantity: row.Quantity,
			Amount: row.Amount, Commission: row.Commission, Fees: row.Fees, ImportSHA: sum,
		}
		rawRow, err := json.Marshal(item)
		if err != nil {
			return ImportResult{}, err
		}
		batch = append(batch, rawRow)
		accounts[row.AccountNumber] = struct{}{}
		if parsed.CashBalancePresent && strings.TrimSpace(row.CashBalance) != "" {
			prev := cashByAccount[row.AccountNumber]
			if prev.HistoryCashAsOf == "" || row.RunDate >= prev.HistoryCashAsOf {
				cashByAccount[row.AccountNumber] = storedAccount{
					HistoryCashBalance: row.CashBalance,
					HistoryCashAsOf:    row.RunDate,
				}
			}
		}
	}
	if _, _, err := db.UpsertBatch(resourceTxns, batch); err != nil {
		return ImportResult{}, err
	}
	var accountList []string
	for account := range accounts {
		accountList = append(accountList, account)
		existing, _ := loadAccount(db, account)
		existing.ID = account
		existing.Account = account
		existing.AccountNumber = account
		if cash, ok := cashByAccount[account]; ok {
			existing.HistoryCashBalance = cash.HistoryCashBalance
			existing.HistoryCashAsOf = cash.HistoryCashAsOf
		}
		if err := upsertAccount(db, existing); err != nil {
			return ImportResult{}, err
		}
	}
	sort.Strings(accountList)
	rec := importRecord{ID: sum, Kind: "history", SHA256: sum, At: time.Now().UTC().Format(time.RFC3339), Accounts: accountList, Rows: len(batch)}
	if err := putJSON(db, resourceImports, sum, rec); err != nil {
		return ImportResult{}, err
	}
	if err := putJSON(db, resourceLatest, "history", latestPointer{ID: "history", Kind: "history", SHA256: sum}); err != nil {
		return ImportResult{}, err
	}
	if err := db.SaveSyncState(resourceTxns, sum, len(batch)); err != nil {
		return ImportResult{}, err
	}
	return ImportResult{Kind: "history", SHA256: sum, Rows: len(batch), Accounts: accountList}, nil
}

// SummarizePortfolio rolls up the latest position snapshots.
func SummarizePortfolio(db *store.Store) (PortfolioSummary, error) {
	rows, err := loadPositions(db)
	if err != nil {
		return PortfolioSummary{}, err
	}
	accounts, err := loadAccounts(db)
	if err != nil {
		return PortfolioSummary{}, err
	}
	if len(rows) == 0 {
		return PortfolioSummary{
			Book:             emptyBook(),
			Accounts:         []AccountTotal{},
			LargestPositions: []LargestPosition{},
			Note:             "no positions imported; run fidelity-pp-cli import positions --file <Portfolio_Positions.csv>",
		}, nil
	}
	type acc struct {
		name     string
		market   *big.Rat
		cash     *big.Rat
		invested *big.Rat
		gain     *big.Rat
	}
	by := map[string]*acc{}
	var largest []LargestPosition
	for _, row := range rows {
		bucket := by[row.AccountNumber]
		if bucket == nil {
			bucket = &acc{market: new(big.Rat), cash: new(big.Rat), invested: new(big.Rat), gain: new(big.Rat)}
			by[row.AccountNumber] = bucket
		}
		if row.AccountName != "" {
			bucket.name = row.AccountName
		}
		value := mustRat(row.CurrentValue)
		gain := mustRat(row.TotalGainLoss)
		bucket.market.Add(bucket.market, value)
		bucket.gain.Add(bucket.gain, gain)
		if isCash(row) {
			bucket.cash.Add(bucket.cash, value)
		} else {
			bucket.invested.Add(bucket.invested, value)
		}
		largest = append(largest, LargestPosition{
			AccountNumber: row.AccountNumber, Symbol: row.Symbol, Description: row.Description,
			MarketValue: fidelitycsv.FormatMoney(value), Quantity: row.Quantity,
		})
	}
	sort.Slice(largest, func(i, j int) bool {
		return mustRat(largest[i].MarketValue).Cmp(mustRat(largest[j].MarketValue)) > 0
	})
	if len(largest) > 5 {
		largest = largest[:5]
	}
	book := emptyBook()
	var out []AccountTotal
	var keys []string
	for key := range by {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	historyCash := new(big.Rat)
	historyCount := 0
	for _, key := range keys {
		bucket := by[key]
		item := AccountTotal{
			AccountNumber: key, AccountName: bucket.name,
			MarketValue: fidelitycsv.FormatMoney(bucket.market), CashValue: fidelitycsv.FormatMoney(bucket.cash),
			InvestedValue: fidelitycsv.FormatMoney(bucket.invested), UnrealizedGainLoss: fidelitycsv.FormatMoney(bucket.gain),
		}
		if stored, ok := accounts[key]; ok {
			if item.AccountName == "" {
				item.AccountName = stored.AccountName
			}
			item.HistoryCashBalance = stored.HistoryCashBalance
			item.HistoryCashAsOf = stored.HistoryCashAsOf
			if stored.HistoryCashBalance != "" {
				historyCash.Add(historyCash, mustRat(stored.HistoryCashBalance))
				historyCount++
			}
		}
		book.MarketValue = addMoney(book.MarketValue, item.MarketValue)
		book.CashValue = addMoney(book.CashValue, item.CashValue)
		book.InvestedValue = addMoney(book.InvestedValue, item.InvestedValue)
		book.UnrealizedGainLoss = addMoney(book.UnrealizedGainLoss, item.UnrealizedGainLoss)
		out = append(out, item)
	}
	if historyCount > 0 {
		book.HistoryCashBalance = fidelitycsv.FormatMoney(historyCash)
	}
	return PortfolioSummary{Book: book, Accounts: out, LargestPositions: largest}, nil
}

// SummarizeActivity totals history rows in an inclusive run-date window.
// action is a case-insensitive substring of the Action cell.
func SummarizeActivity(db *store.Store, from, to, action string) (ActivitySummary, error) {
	from, err := optionalDate(from, "--from")
	if err != nil {
		return ActivitySummary{}, err
	}
	to, err = optionalDate(to, "--to")
	if err != nil {
		return ActivitySummary{}, err
	}
	rows, err := loadTxns(db)
	if err != nil {
		return ActivitySummary{}, err
	}
	if len(rows) == 0 {
		return ActivitySummary{
			From: from, To: to, Action: action, Amount: zeroMoney(), Commission: zeroMoney(), Fees: zeroMoney(),
			ByAction: []ActionTotal{},
			Note:     "no history imported; run fidelity-pp-cli import history --file <History_for_Account_ID.csv>",
		}, nil
	}
	type bucket struct {
		rows       int
		amount     *big.Rat
		commission *big.Rat
		fees       *big.Rat
	}
	by := map[string]*bucket{}
	for _, row := range rows {
		if from != "" && row.RunDate < from {
			continue
		}
		if to != "" && row.RunDate > to {
			continue
		}
		if action != "" && !strings.Contains(strings.ToUpper(row.Action), strings.ToUpper(action)) {
			continue
		}
		b := by[row.Action]
		if b == nil {
			b = &bucket{amount: new(big.Rat), commission: new(big.Rat), fees: new(big.Rat)}
			by[row.Action] = b
		}
		b.rows++
		b.amount.Add(b.amount, mustRat(row.Amount))
		b.commission.Add(b.commission, mustRat(row.Commission))
		b.fees.Add(b.fees, mustRat(row.Fees))
	}
	var actions []string
	for key := range by {
		actions = append(actions, key)
	}
	sort.Strings(actions)
	out := ActivitySummary{From: from, To: to, Action: action, Amount: zeroMoney(), Commission: zeroMoney(), Fees: zeroMoney(), ByAction: []ActionTotal{}}
	for _, key := range actions {
		b := by[key]
		item := ActionTotal{Action: key, Rows: b.rows, Amount: fidelitycsv.FormatMoney(b.amount), Commission: fidelitycsv.FormatMoney(b.commission), Fees: fidelitycsv.FormatMoney(b.fees)}
		out.ByAction = append(out.ByAction, item)
		out.Rows += b.rows
		out.Amount = addMoney(out.Amount, item.Amount)
		out.Commission = addMoney(out.Commission, item.Commission)
		out.Fees = addMoney(out.Fees, item.Fees)
	}
	return out, nil
}

// DiffChanges compares the current positions snapshot with the retained
// previous one and lists history rows from the latest history import.
func DiffChanges(db *store.Store, account string) (Changes, error) {
	account = strings.TrimSpace(account)
	current, err := loadPositions(db)
	if err != nil {
		return Changes{}, err
	}
	txns, err := loadTxns(db)
	if err != nil {
		return Changes{}, err
	}
	latest := ""
	var pointer latestPointer
	if err := getJSON(db, resourceLatest, "history", &pointer); err == nil {
		latest = pointer.SHA256
	}
	out := Changes{Account: account, Appeared: []PositionDelta{}, Disappeared: []PositionDelta{}, QuantityChanged: []PositionDelta{}, HistoryRows: []HistoryChange{}}
	if len(current) == 0 && len(txns) == 0 {
		out.Note = "no imports yet; run fidelity-pp-cli import positions --file <Portfolio_Positions.csv>"
		return out, nil
	}
	accounts := map[string]struct{}{}
	if account != "" {
		accounts[account] = struct{}{}
	} else {
		for _, row := range current {
			accounts[row.AccountNumber] = struct{}{}
		}
	}
	var notes []string
	for acct := range accounts {
		prev, err := loadPrevious(db, acct)
		if err != nil {
			return Changes{}, err
		}
		cur := filterPositions(current, acct)
		if prev == nil {
			if len(cur) > 0 {
				notes = append(notes, acct+" has no previous positions snapshot")
			}
			for _, row := range cur {
				out.Appeared = append(out.Appeared, deltaFrom(row, ""))
			}
			continue
		}
		prevBy := map[string]storedPosition{}
		for _, row := range prev.Positions {
			prevBy[row.ID] = row
		}
		curBy := map[string]storedPosition{}
		for _, row := range cur {
			curBy[row.ID] = row
			old, ok := prevBy[row.ID]
			if !ok {
				out.Appeared = append(out.Appeared, deltaFrom(row, ""))
				continue
			}
			if mustRat(old.Quantity).Cmp(mustRat(row.Quantity)) != 0 {
				item := deltaFrom(row, old.Quantity)
				out.QuantityChanged = append(out.QuantityChanged, item)
			}
		}
		for id, old := range prevBy {
			if _, ok := curBy[id]; !ok {
				out.Disappeared = append(out.Disappeared, deltaFrom(old, ""))
			}
		}
	}
	for _, row := range txns {
		if account != "" && row.AccountNumber != account {
			continue
		}
		if latest != "" && row.ImportSHA != latest {
			continue
		}
		out.HistoryRows = append(out.HistoryRows, HistoryChange{
			AccountNumber: row.AccountNumber, RunDate: row.RunDate, Action: row.Action, Symbol: row.Symbol, Amount: row.Amount,
		})
	}
	sort.Slice(out.HistoryRows, func(i, j int) bool {
		if out.HistoryRows[i].RunDate == out.HistoryRows[j].RunDate {
			return out.HistoryRows[i].Symbol < out.HistoryRows[j].Symbol
		}
		return out.HistoryRows[i].RunDate > out.HistoryRows[j].RunDate
	})
	if len(notes) > 0 {
		sort.Strings(notes)
		out.Note = strings.Join(notes, "; ")
	}
	return out, nil
}

// TotalSymbol sums one symbol across accounts. symbol is matched case-insensitively.
func TotalSymbol(db *store.Store, symbol string) (SymbolTotal, error) {
	symbol = strings.TrimSpace(symbol)
	rows, err := loadPositions(db)
	if err != nil {
		return SymbolTotal{}, err
	}
	out := SymbolTotal{Symbol: symbol, Quantity: "0", MarketValue: zeroMoney(), CostBasis: zeroMoney(), UnrealizedGainLoss: zeroMoney(), Accounts: []SymbolAccount{}}
	if len(rows) == 0 {
		out.Note = "no positions imported; run fidelity-pp-cli import positions --file <Portfolio_Positions.csv>"
		return out, nil
	}
	type acc struct {
		name string
		qty  *big.Rat
		mv   *big.Rat
		cost *big.Rat
		gain *big.Rat
	}
	by := map[string]*acc{}
	for _, row := range rows {
		if !strings.EqualFold(row.Symbol, symbol) {
			continue
		}
		b := by[row.AccountNumber]
		if b == nil {
			b = &acc{qty: new(big.Rat), mv: new(big.Rat), cost: new(big.Rat), gain: new(big.Rat)}
			by[row.AccountNumber] = b
		}
		b.name = row.AccountName
		b.qty.Add(b.qty, mustRat(row.Quantity))
		b.mv.Add(b.mv, mustRat(row.CurrentValue))
		b.cost.Add(b.cost, mustRat(row.CostBasis))
		b.gain.Add(b.gain, mustRat(row.TotalGainLoss))
	}
	if len(by) == 0 {
		out.Note = fmt.Sprintf("no imported position uses symbol %s", symbol)
		return out, nil
	}
	var keys []string
	for key := range by {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	qty, mv, cost, gain := new(big.Rat), new(big.Rat), new(big.Rat), new(big.Rat)
	for _, key := range keys {
		b := by[key]
		qty.Add(qty, b.qty)
		mv.Add(mv, b.mv)
		cost.Add(cost, b.cost)
		gain.Add(gain, b.gain)
		out.Accounts = append(out.Accounts, SymbolAccount{
			AccountNumber: key, AccountName: b.name,
			Quantity: fidelitycsv.FormatQuantity(b.qty), MarketValue: fidelitycsv.FormatMoney(b.mv),
			CostBasis: fidelitycsv.FormatMoney(b.cost), UnrealizedGainLoss: fidelitycsv.FormatMoney(b.gain),
		})
	}
	out.Quantity = fidelitycsv.FormatQuantity(qty)
	out.MarketValue = fidelitycsv.FormatMoney(mv)
	out.CostBasis = fidelitycsv.FormatMoney(cost)
	out.UnrealizedGainLoss = fidelitycsv.FormatMoney(gain)
	return out, nil
}

// ReportCoverage lists which accounts have positions and history.
func ReportCoverage(db *store.Store) (Coverage, error) {
	positions, err := loadPositions(db)
	if err != nil {
		return Coverage{}, err
	}
	txns, err := loadTxns(db)
	if err != nil {
		return Coverage{}, err
	}
	accounts, err := loadAccounts(db)
	if err != nil {
		return Coverage{}, err
	}
	type span struct {
		name string
		pos  bool
		hist bool
		from string
		to   string
	}
	by := map[string]*span{}
	ensure := func(account, name string) *span {
		item := by[account]
		if item == nil {
			item = &span{}
			by[account] = item
		}
		if name != "" && item.name == "" {
			item.name = name
		}
		return item
	}
	for _, row := range positions {
		ensure(row.AccountNumber, row.AccountName).pos = true
	}
	for _, row := range txns {
		item := ensure(row.AccountNumber, "")
		item.hist = true
		if item.from == "" || row.RunDate < item.from {
			item.from = row.RunDate
		}
		if row.RunDate > item.to {
			item.to = row.RunDate
		}
	}
	for account, stored := range accounts {
		ensure(account, stored.AccountName)
	}
	if len(by) == 0 {
		return Coverage{Accounts: []CoverageAccount{}, Note: "no imports yet; run fidelity-pp-cli import positions --file <Portfolio_Positions.csv>"}, nil
	}
	var keys []string
	for key := range by {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := Coverage{Accounts: []CoverageAccount{}}
	for _, key := range keys {
		item := by[key]
		out.Accounts = append(out.Accounts, CoverageAccount{
			AccountNumber: key, AccountName: item.name, HasPositions: item.pos, HasHistory: item.hist,
			HistoryFrom: item.from, HistoryTo: item.to,
		})
	}
	return out, nil
}

func collapsePositions(parsed []fidelitycsv.Position) map[string][]storedPosition {
	type key struct{ account, id string }
	order := []key{}
	by := map[key]storedPosition{}
	for _, row := range parsed {
		id := positionID(row.AccountNumber, row.Symbol, row.Type)
		k := key{row.AccountNumber, id}
		existing, ok := by[k]
		if !ok {
			order = append(order, k)
			by[k] = storedPosition{
				ID: id, Account: row.AccountNumber, AccountNumber: row.AccountNumber, AccountName: row.AccountName,
				Symbol: row.Symbol, Description: row.Description, Quantity: row.Quantity, LastPrice: row.LastPrice,
				CurrentValue: row.CurrentValue, CostBasis: row.CostBasis, TotalGainLoss: row.TotalGainLoss,
				PercentOfAccount: row.PercentOfAccount, Type: row.Type,
			}
			continue
		}
		existing.Quantity = fidelitycsv.FormatQuantity(new(big.Rat).Add(mustRat(existing.Quantity), mustRat(row.Quantity)))
		existing.CurrentValue = fidelitycsv.FormatMoney(new(big.Rat).Add(mustRat(existing.CurrentValue), mustRat(row.CurrentValue)))
		existing.CostBasis = fidelitycsv.FormatMoney(new(big.Rat).Add(mustRat(existing.CostBasis), mustRat(row.CostBasis)))
		existing.TotalGainLoss = fidelitycsv.FormatMoney(new(big.Rat).Add(mustRat(existing.TotalGainLoss), mustRat(row.TotalGainLoss)))
		by[k] = existing
	}
	grouped := map[string][]storedPosition{}
	for _, k := range order {
		grouped[k.account] = append(grouped[k.account], by[k])
	}
	return grouped
}

func positionID(account, symbol, typ string) string {
	return account + "|" + strings.ToUpper(symbol) + "|" + strings.ToUpper(typ)
}

func rowID(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// isCash reports spendable cash. In the official positions export, Type is
// Cash versus Margin (which sleeve holds the shares), so a stock typed Cash
// is still invested. Cash is a money-market symbol, a cash description, or a
// blank symbol in the cash sleeve.
func isCash(row storedPosition) bool {
	symbol := strings.TrimRight(strings.ToUpper(strings.TrimSpace(row.Symbol)), "*")
	switch symbol {
	case "SPAXX", "FDRXX", "FCASH", "FZFXX", "SPRXX", "FDLXX":
		return true
	}
	desc := strings.ToUpper(strings.TrimSpace(row.Description))
	if strings.Contains(desc, "MONEY MARKET") || strings.HasPrefix(desc, "CASH") || strings.Contains(desc, "HELD IN MONEY") {
		return true
	}
	return symbol == "" && strings.Contains(strings.ToLower(row.Type), "cash")
}

func emptyBook() BookTotal {
	return BookTotal{MarketValue: zeroMoney(), CashValue: zeroMoney(), InvestedValue: zeroMoney(), UnrealizedGainLoss: zeroMoney()}
}

func zeroMoney() string {
	return fidelitycsv.FormatMoney(new(big.Rat))
}

func mustRat(raw string) *big.Rat {
	r, ok := fidelitycsv.ParseDecimal(raw)
	if !ok {
		return new(big.Rat)
	}
	return r
}

func addMoney(left, right string) string {
	sum := new(big.Rat).Add(mustRat(left), mustRat(right))
	return fidelitycsv.FormatMoney(sum)
}

func optionalDate(raw, flag string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	normalized, ok := fidelitycsv.NormalizeDate(raw)
	if !ok {
		return "", fmt.Errorf("%s must be a date such as 2026-01-01 or 01/01/2026, got %q", flag, raw)
	}
	return normalized, nil
}

func importSeen(db *store.Store, sum string) (bool, error) {
	_, err := db.Get(resourceImports, sum)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func putJSON(db *store.Store, resource, id string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return db.Upsert(resource, id, raw)
}

func getJSON(db *store.Store, resource, id string, dest any) error {
	raw, err := db.Get(resource, id)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}

func upsertAccount(db *store.Store, item storedAccount) error {
	if item.ID == "" {
		item.ID = item.AccountNumber
	}
	existing, err := loadAccount(db, item.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil {
		if item.AccountName == "" {
			item.AccountName = existing.AccountName
		}
		if item.HistoryCashBalance == "" {
			item.HistoryCashBalance = existing.HistoryCashBalance
			item.HistoryCashAsOf = existing.HistoryCashAsOf
		}
	}
	raw, err := json.Marshal(item)
	if err != nil {
		return err
	}
	if _, _, err := db.UpsertBatch(resourceAccounts, []json.RawMessage{raw}); err != nil {
		return err
	}
	return nil
}

func loadAccount(db *store.Store, id string) (storedAccount, error) {
	var item storedAccount
	err := getJSON(db, resourceAccounts, id, &item)
	return item, err
}

func loadPositions(db *store.Store) ([]storedPosition, error) {
	return listJSON[storedPosition](db, resourcePositions)
}

func loadTxns(db *store.Store) ([]storedTxn, error) {
	return listJSON[storedTxn](db, resourceTxns)
}

func loadAccounts(db *store.Store) (map[string]storedAccount, error) {
	rows, err := listJSON[storedAccount](db, resourceAccounts)
	if err != nil {
		return nil, err
	}
	out := map[string]storedAccount{}
	for _, row := range rows {
		out[row.AccountNumber] = row
	}
	return out, nil
}

func positionsForAccount(db *store.Store, account string) ([]storedPosition, error) {
	rows, err := loadPositions(db)
	if err != nil {
		return nil, err
	}
	return filterPositions(rows, account), nil
}

func filterPositions(rows []storedPosition, account string) []storedPosition {
	var out []storedPosition
	for _, row := range rows {
		if row.AccountNumber == account {
			out = append(out, row)
		}
	}
	return out
}

func savePrevious(db *store.Store, account string, rows []storedPosition) error {
	return putJSON(db, resourcePrevious, account, previousSnapshot{ID: account, Account: account, Positions: rows})
}

func loadPrevious(db *store.Store, account string) (*previousSnapshot, error) {
	var snap previousSnapshot
	err := getJSON(db, resourcePrevious, account, &snap)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

func deltaFrom(row storedPosition, previousQty string) PositionDelta {
	return PositionDelta{
		AccountNumber: row.AccountNumber, Symbol: row.Symbol, Description: row.Description,
		Quantity: row.Quantity, PreviousQty: previousQty, CurrentValue: row.CurrentValue,
	}
}

func listJSON[T any](db *store.Store, resource string) ([]T, error) {
	raws, err := db.List(resource, 0)
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(raws))
	for _, raw := range raws {
		var item T
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, fmt.Errorf("decoding %s row: %w", resource, err)
		}
		out = append(out, item)
	}
	return out, nil
}
