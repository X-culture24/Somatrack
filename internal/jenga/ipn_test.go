package jenga

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const sampleIPN = `{
  "callbackType": "IPN",
  "customer": {"name": "JANE DOE", "mobileNumber": "254712345678", "reference": "137101#FAITH OTIENO"},
  "transaction": {"date": "2026-01-15 10:45:00.0", "reference": "QK7TEST001", "paymentMode": "MPESA",
    "amount": "5000.00", "billNumber": "", "status": "SUCCESS", "additionalInfo": "MPS 254712345678 QK7TEST001"},
  "bank": {"reference": "FT26015ABC", "transactionType": "C", "account": "1370263402101"}
}`

func TestIPNToPayment(t *testing.T) {
	loc, _ := time.LoadLocation("Africa/Nairobi")
	r := httptest.NewRequest("POST", "/", strings.NewReader(sampleIPN))
	n, _, err := DecodeIPN(r)
	if err != nil {
		t.Fatal(err)
	}
	if !n.IsCredit() {
		t.Fatal("expected credit")
	}
	p, err := n.ToPayment("137101", loc)
	if err != nil {
		t.Fatal(err)
	}
	if p.TransID != "QK7TEST001" || p.Amount != 5000 || p.BillRef != "137101#FAITH OTIENO" || p.TransTime != "20260115104500" {
		t.Fatalf("unexpected payment: %+v", p)
	}
}

func TestIPNDebitIgnored(t *testing.T) {
	n := &IPN{}
	n.Bank.TransactionType = "D"
	if n.IsCredit() {
		t.Fatal("debit treated as credit")
	}
}

func TestIPNAuth(t *testing.T) {
	a := IPNAuth{Username: "u", Password: "p"}
	r := httptest.NewRequest("POST", "/", nil)
	if a.Check(r) {
		t.Fatal("accepted without credentials")
	}
	r.SetBasicAuth("u", "p")
	if !a.Check(r) {
		t.Fatal("rejected valid credentials")
	}
	if (IPNAuth{Sandbox: false}).Check(httptest.NewRequest("POST", "/", nil)) {
		t.Fatal("live mode with no auth configured must fail closed")
	}
}
