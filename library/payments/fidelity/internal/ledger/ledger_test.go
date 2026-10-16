package ledger

import (
	"path/filepath"
	"testing"

	"github.com/mvanhorn/printing-press-library/library/payments/fidelity/internal/store"
)

func TestImportPositionsAndQueries(t *testing.T) {
	db := openTestDB(t)
	first := []byte(`Account Number,Account Name,Symbol,Description,Quantity,Last Price,Current Value,Total Gain/Loss Dollar,Percent Of Account,Cost Basis,Type
Z1234567,Brokerage,VTI,Vanguard Total Stock Market,10,$25.00,$250.00,+$10.00,70%,$240.00,Cash
Z1234567,Brokerage,SPAXX,Fidelity Government Money Market,100,$1.00,$100.00,$0.00,30%,$100.00,Cash
12345678,Roth IRA,VTI,Vanguard Total Stock Market,2,$25.00,$50.00,+$5.00,100%,$45.00,Cash
Date downloaded 10/16/2026
"The data and information is not a position."
`)
	got, err := ImportPositions(db, "Portfolio_Positions_Oct-16-2026.csv", first)
	if err != nil {
		t.Fatal(err)
	}
	if got.Noop || got.Rows != 3 {
		t.Fatalf("import = %+v", got)
	}
	again, err := ImportPositions(db, "Portfolio_Positions_Oct-16-2026.csv", first)
	if err != nil {
		t.Fatal(err)
	}
	if !again.Noop {
		t.Fatal("expected idempotent re-import")
	}
	summary, err := SummarizePortfolio(db)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Book.MarketValue != "400.00" || summary.Book.CashValue != "100.00" || summary.Book.InvestedValue != "300.00" {
		t.Fatalf("book = %+v", summary.Book)
	}
	if summary.Book.UnrealizedGainLoss != "15.00" {
		t.Fatalf("gain = %s", summary.Book.UnrealizedGainLoss)
	}
	total, err := TotalSymbol(db, "vti")
	if err != nil {
		t.Fatal(err)
	}
	if total.Quantity != "12" || total.MarketValue != "300.00" || len(total.Accounts) != 2 {
		t.Fatalf("symbol = %+v", total)
	}
	second := []byte(`Account Number,Account Name,Symbol,Description,Quantity,Last Price,Current Value,Total Gain/Loss Dollar,Percent Of Account,Cost Basis,Type
Z1234567,Brokerage,VTI,Vanguard Total Stock Market,8,$25.00,$200.00,+$10.00,100%,$190.00,Cash
`)
	if _, err := ImportPositions(db, "next.csv", second); err != nil {
		t.Fatal(err)
	}
	changes, err := DiffChanges(db, "Z1234567")
	if err != nil {
		t.Fatal(err)
	}
	if len(changes.Disappeared) != 1 || changes.Disappeared[0].Symbol != "SPAXX" {
		t.Fatalf("disappeared = %+v", changes.Disappeared)
	}
	if len(changes.QuantityChanged) != 1 || changes.QuantityChanged[0].Symbol != "VTI" || changes.QuantityChanged[0].PreviousQty != "10" {
		t.Fatalf("changed = %+v", changes.QuantityChanged)
	}
	history := []byte("Run Date,Action,Symbol,Description,Quantity,Amount ($),Commission ($),Fees ($),Cash Balance ($)\n01/15/2026,DIVIDEND RECEIVED,VTI,VANGUARD, ,\"$12.50\",$0.00,$0.00,\"$1,000.00\"\n")
	imported, err := ImportHistory(db, "History_for_Account_Z1234567.csv", "", history)
	if err != nil {
		t.Fatal(err)
	}
	if imported.Rows != 1 || imported.Accounts[0] != "Z1234567" {
		t.Fatalf("history = %+v", imported)
	}
	activity, err := SummarizeActivity(db, "2026-01-01", "", "dividend")
	if err != nil {
		t.Fatal(err)
	}
	if activity.Rows != 1 || activity.Amount != "12.50" {
		t.Fatalf("activity = %+v", activity)
	}
	summary, err = SummarizePortfolio(db)
	if err != nil {
		t.Fatal(err)
	}
	var cash string
	for _, account := range summary.Accounts {
		if account.AccountNumber == "Z1234567" {
			cash = account.HistoryCashBalance
		}
	}
	if cash != "$1,000.00" {
		t.Fatalf("history cash = %q, accounts=%+v", cash, summary.Accounts)
	}
	if summary.Book.CashValue == summary.Book.HistoryCashBalance {
		t.Fatalf("history cash was folded into invested cash: %+v", summary.Book)
	}
	coverage, err := ReportCoverage(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(coverage.Accounts) != 2 {
		t.Fatalf("coverage = %+v", coverage)
	}
}

func TestEmptyLedgerExitsClean(t *testing.T) {
	db := openTestDB(t)
	summary, err := SummarizePortfolio(db)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Note == "" || summary.Book.MarketValue != "0.00" {
		t.Fatalf("summary = %+v", summary)
	}
	coverage, err := ReportCoverage(db)
	if err != nil || coverage.Note == "" {
		t.Fatalf("coverage = %+v err=%v", coverage, err)
	}
}

func openTestDB(t *testing.T) *store.Store {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
