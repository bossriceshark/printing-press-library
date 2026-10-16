package fidelitycsv

import "testing"

func TestParsePositionsSkipsFooter(t *testing.T) {
	raw := []byte(`Account Number,Account Name,Symbol,Description,Quantity,Last Price,Current Value,Total Gain/Loss Dollar,Total Gain/Loss Percent,Percent Of Account,Cost Basis,Type
Z1234567,Brokerage,VTI,Vanguard Total Stock Market,10,"$25.00","$250.00","+$10.00",4%,80%,"$240.00",Cash
Z1234567,Brokerage,SPAXX,Fidelity Government Money Market,100,"$1.00","$100.00","$0.00",0%,20%,"$100.00",Cash

Date downloaded 10/16/2026
The data and information is for informational purposes only.
`)
	rows, err := ParsePositions(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0].Symbol != "VTI" || rows[0].CurrentValue != "$250.00" {
		t.Fatalf("first row = %+v", rows[0])
	}
	got, ok := ParseDecimal(rows[0].TotalGainLoss)
	if !ok || FormatMoney(got) != "10.00" {
		t.Fatalf("gain = %s ok=%v", FormatMoney(got), ok)
	}
}

func TestParseHistoryFilenameAndDates(t *testing.T) {
	id, ok := AccountFromHistoryFilename("/tmp/History_for_Account_Z1234567.csv")
	if !ok || id != "Z1234567" {
		t.Fatalf("account = %q ok=%v", id, ok)
	}
	raw := []byte("Run Date,Action,Symbol,Description,Type,Price ($),Quantity,Commission ($),Fees ($),Amount ($),Cash Balance ($)\n10/01/2026,DIVIDEND RECEIVED,VTI,VANGUARD,Cash,, ,,,\"$12.50\",\"$1,000.00\"\nThe disclaimer is not a transaction\n")
	file, err := ParseHistory(raw, id)
	if err != nil {
		t.Fatal(err)
	}
	if !file.CashBalancePresent || len(file.Rows) != 1 {
		t.Fatalf("file = %+v", file)
	}
	if file.Rows[0].RunDate != "2026-10-01" || file.Rows[0].AccountNumber != "Z1234567" {
		t.Fatalf("row = %+v", file.Rows[0])
	}
	if file.Rows[0].AccountNumber != id {
		t.Fatal("account mismatch")
	}
}
