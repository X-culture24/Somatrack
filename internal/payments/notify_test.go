package payments

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// A realistic production link: portal domain + /r# + 22-char code.
const link = "https://portal.stmaryskabete.ac.ke/r#Q2xhdWRlIHdhcyBoZXJlIQ"

func isGSM7(s string) bool { return gsmSafe(s) == s }

func TestPaymentSMSFitsOneMessage(t *testing.T) {
	cases := []struct {
		first, last string
		amount, bal float64
		want        string
	}{
		{"Brian", "Kamau", 4000, 12500, "St.Mary's: KES 4,000 received for Brian Kamau. Bal KES 12,500. Receipt: " + link},
		{"Faith", "Otieno", 1250.5, 0, "St.Mary's: KES 1,250.50 received for Faith Otieno. Fees fully paid. Receipt: " + link},
		{"Faith", "Otieno", 40000, -1000, "St.Mary's: KES 40,000 received for Faith Otieno. Credit KES 1,000. Receipt: " + link},
	}
	for _, c := range cases {
		got := PaymentSMS(c.first, c.last, c.amount, c.bal, link)
		if got != c.want {
			t.Errorf("got  %q\nwant %q", got, c.want)
		}
	}
	// Worst case: long names and six-figure sums still fit in 160 GSM-7 chars.
	long := PaymentSMS("Christabel-Wanjiku", "Mwangi-Otieno-Njoroge", 1234567.89, 987654.32, link)
	if n := len([]rune(long)); n > 160 {
		t.Errorf("long-name SMS is %d chars: %q", n, long)
	}
	if !strings.Contains(long, "Christabel-Wanjiku M.") {
		t.Errorf("long names should shorten to first name + initial: %q", long)
	}
}

func TestPaymentSMSStaysGSM7(t *testing.T) {
	// One curly apostrophe or accented name would switch the SMS to UCS-2
	// (70 chars) and triple the cost.
	msg := PaymentSMS("Zoë", "O’Neill", 500, 100, link)
	if !isGSM7(msg) {
		t.Errorf("not GSM-7: %q", msg)
	}
	if !strings.Contains(msg, "O'Neill") {
		t.Errorf("curly apostrophe should become a plain one: %q", msg)
	}
}

func TestKESShort(t *testing.T) {
	for in, want := range map[float64]string{0: "0", 999: "999", 1000: "1,000", 12500: "12,500", 1234567.891: "1,234,567.89", 50.5: "50.50"} {
		if got := kesShort(in); got != want {
			t.Errorf("kesShort(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestManualPaymentValidation(t *testing.T) {
	ok := ManualPayment{StudentID: uuid.New(), Amount: 5000, Method: "cash"}
	if e := ok.validate(); len(e) != 0 {
		t.Errorf("valid cash payment rejected: %v", e)
	}
	bad := []ManualPayment{
		{Amount: 5000, Method: "cash"},                                                 // no student
		{StudentID: uuid.New(), Amount: 0, Method: "cash"},                             // no amount
		{StudentID: uuid.New(), Amount: 10, Method: "mpesa"},                           // M-Pesa comes via webhook only
		{StudentID: uuid.New(), Amount: 10, Method: "cheque"},                          // cheque needs a number
		{StudentID: uuid.New(), Amount: 10, Method: "bank_transfer"},                   // bank needs a reference
		{StudentID: uuid.New(), Amount: 10, Method: "cash", PaymentDate: "2099-01-01"}, // future
	}
	for i, m := range bad {
		if e := m.validate(); len(e) == 0 {
			t.Errorf("case %d should fail validation", i)
		}
	}
}

func TestReceiptNo(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		n, err := newReceiptNo(mustDate("2026-09-25"))
		if err != nil || !strings.HasPrefix(n, "RCT-20260925-") || len(n) != 19 {
			t.Fatalf("receipt no %q, %v", n, err)
		}
		if seen[n] {
			t.Fatalf("duplicate receipt no %s", n)
		}
		seen[n] = true
	}
}

func mustDate(s string) (t time.Time) {
	t, _ = time.Parse("2006-01-02", s)
	return
}
