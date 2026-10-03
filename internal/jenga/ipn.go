package jenga

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// IPN is the Instant Payment Notification body Jenga POSTs to the callback
// URL configured in Jenga HQ. Field names follow Jenga's IPN docs; amounts
// arrive as numbers in some payloads and strings in others, hence flexNum.
type IPN struct {
	CallbackType string `json:"callbackType"`
	Customer     struct {
		Name         string `json:"name"`
		MobileNumber string `json:"mobileNumber"`
		Reference    string `json:"reference"`
	} `json:"customer"`
	Transaction struct {
		Date           string  `json:"date"`
		Reference      string  `json:"reference"`
		PaymentMode    string  `json:"paymentMode"`
		Amount         flexNum `json:"amount"`
		BillNumber     string  `json:"billNumber"`
		ServedBy       string  `json:"servedBy"`
		AdditionalInfo string  `json:"additionalInfo"`
		Status         string  `json:"status"`
		Remarks        string  `json:"remarks"`
	} `json:"transaction"`
	Bank struct {
		Reference       string `json:"reference"`
		TransactionType string `json:"transactionType"`
		Account         string `json:"account"`
	} `json:"bank"`
}

type flexNum float64

func (f *flexNum) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	var v float64
	if _, err := fmt.Sscanf(strings.ReplaceAll(s, ",", ""), "%g", &v); err != nil {
		return fmt.Errorf("amount %q: %w", s, err)
	}
	*f = flexNum(v)
	return nil
}

// Payment is the IPN reduced to what the fee reconciler needs.
type Payment struct {
	TransID     string
	Amount      float64
	BillRef     string
	TransTime   string // "20060102150405", the layout the M-Pesa reconciler parses
	PayerName   string
	PayerPhone  string
	PaymentMode string
}

// IsCredit reports whether the IPN is money coming *in* successfully; debits
// and failed/reversed transactions must not be recorded as fee payments.
func (n *IPN) IsCredit() bool {
	if t := strings.ToUpper(strings.TrimSpace(n.Bank.TransactionType)); t != "" && t != "C" {
		return false
	}
	if s := strings.ToUpper(strings.TrimSpace(n.Transaction.Status)); s != "" && s != "SUCCESS" {
		return false
	}
	return true
}

// ToPayment extracts the reconciler fields. The payer's account entry
// ("137101#CHILDSNAME") can land in billNumber, customer.reference or inside
// additionalInfo depending on the channel, so accountPrefix is used to pick
// the right one. The raw payload is stored alongside, so the first live IPNs
// should be checked in mpesa_webhook_events to confirm which field carries it.
func (n *IPN) ToPayment(accountPrefix string, loc *time.Location) (Payment, error) {
	id := strings.TrimSpace(n.Transaction.Reference)
	if id == "" {
		id = strings.TrimSpace(n.Bank.Reference)
	}
	if id == "" {
		return Payment{}, fmt.Errorf("ipn has no transaction reference")
	}
	return Payment{
		TransID:     id,
		Amount:      float64(n.Transaction.Amount),
		BillRef:     n.billRef(accountPrefix),
		TransTime:   formatTransTime(n.Transaction.Date, loc),
		PayerName:   strings.TrimSpace(n.Customer.Name),
		PayerPhone:  strings.TrimSpace(n.Customer.MobileNumber),
		PaymentMode: n.Transaction.PaymentMode,
	}, nil
}

func (n *IPN) billRef(prefix string) string {
	candidates := []string{n.Transaction.BillNumber, n.Customer.Reference}
	for _, c := range candidates {
		if prefix != "" && strings.HasPrefix(strings.TrimSpace(c), prefix) {
			return strings.TrimSpace(c)
		}
	}
	if prefix != "" {
		for _, tok := range strings.FieldsFunc(n.Transaction.AdditionalInfo, func(r rune) bool {
			return r == ' ' || r == '/' || r == '|' || r == ','
		}) {
			if strings.HasPrefix(tok, prefix) {
				return tok
			}
		}
	}
	for _, c := range candidates {
		if s := strings.TrimSpace(c); s != "" {
			return s
		}
	}
	return strings.TrimSpace(n.Transaction.AdditionalInfo)
}

func formatTransTime(raw string, loc *time.Location) string {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{
		"2006-01-02 15:04:05.0",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		time.RFC3339,
		"2006-01-02",
	} {
		if t, err := time.ParseInLocation(layout, raw, loc); err == nil {
			return t.In(loc).Format("20060102150405")
		}
	}
	return time.Now().In(loc).Format("20060102150405")
}

// IPNAuth checks the credentials Jenga sends with each IPN. Jenga HQ lets you
// attach Basic auth to the callback URL; a ?secret= query parameter is
// accepted as well for setups that can't use headers.
type IPNAuth struct {
	Username string
	Password string
	Secret   string
	Sandbox  bool
}

func IPNAuthFromEnv() IPNAuth {
	return IPNAuth{
		Username: os.Getenv("JENGA_IPN_USERNAME"),
		Password: os.Getenv("JENGA_IPN_PASSWORD"),
		Secret:   os.Getenv("JENGA_IPN_SECRET"),
		Sandbox:  !strings.EqualFold(getEnv("JENGA_ENV", "sandbox"), "live"),
	}
}

func (a IPNAuth) configured() bool {
	return (a.Username != "" && a.Password != "") || a.Secret != ""
}

// Check fails closed in live mode when no IPN credentials are set, since an
// open endpoint would let anyone post fake fee payments.
func (a IPNAuth) Check(r *http.Request) bool {
	if !a.configured() {
		return a.Sandbox
	}
	if a.Username != "" && a.Password != "" {
		if u, p, ok := r.BasicAuth(); ok &&
			subtle.ConstantTimeCompare([]byte(u), []byte(a.Username)) == 1 &&
			subtle.ConstantTimeCompare([]byte(p), []byte(a.Password)) == 1 {
			return true
		}
	}
	if a.Secret != "" {
		if s := r.URL.Query().Get("secret"); s != "" &&
			subtle.ConstantTimeCompare([]byte(s), []byte(a.Secret)) == 1 {
			return true
		}
	}
	return false
}

// DecodeIPN reads the body, returning both the parsed IPN and the raw JSON
// (kept verbatim for the audit trail).
func DecodeIPN(r *http.Request) (*IPN, string, error) {
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, "", err
	}
	var n IPN
	if err := json.Unmarshal(raw, &n); err != nil {
		return nil, string(raw), err
	}
	return &n, string(raw), nil
}
