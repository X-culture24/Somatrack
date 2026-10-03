//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/stmaryskabete/lms/internal/api"
	"github.com/stmaryskabete/lms/internal/shortlink"
	"github.com/stmaryskabete/lms/internal/signedlink"
	"github.com/stmaryskabete/lms/internal/testutil"
)

// mpesa posts a Safaricom-shaped payment through the real reconciliation
// pipeline (the same one Jenga IPNs and the C2B webhook use).
func (e *env) mpesa(transID, billRef string, amount float64) resp {
	return e.anon().post("/internal/payments/mpesa/process", map[string]interface{}{
		"trans_id": transID, "trans_amount": amount, "bill_ref_number": billRef,
		"trans_time": "20260210093000", "first_name": "John", "last_name": "Kamau", "msisdn": "254711000111",
	})
}

func balance(t *testing.T, e *env, student uuid.UUID) float64 {
	return testutil.Scalar[float64](t, e.pool, `SELECT COALESCE(SUM(balance), 0)::float8 FROM invoices WHERE student_id = $1`, student)
}

func TestMpesaPaymentTextsReceiptLinkOnce(t *testing.T) {
	e := newEnv(t)
	// Paid with the reference reminders give: account prefix + admission no.
	r := want(t, e.mpesa("QK1001", accountPrefix+"ADM-002", 4000), 200).JSON(t)
	if r["reconciliation_status"] != "posted" || r["matched_student_id"] != brianID.String() {
		t.Fatalf("not matched by admission number: %v", r)
	}
	if b := balance(t, e, brianID); b != 16000 {
		t.Errorf("balance = %v, want 16000", b)
	}

	sms := e.sms.waitFor(t, "payment_received", 1)[0]
	if sms.To != "0722000111" { // the guardian, not the payer's M-Pesa number
		t.Errorf("sent to %s", sms.To)
	}
	wantPrefix := "St.Mary's: KES 4,000 received for Brian Kamau. Bal KES 16,000. Receipt: " + portalURL + "/r#"
	if !strings.HasPrefix(sms.Message, wantPrefix) || len(sms.Message) > 160 {
		t.Errorf("SMS = %q (%d chars)", sms.Message, len(sms.Message))
	}

	// The same money reported again (Jenga IPN + C2B) posts and texts once.
	want(t, e.mpesa("QK1001", accountPrefix+"ADM-002", 4000), 409)
	if n := testutil.Scalar[int](t, e.pool, `SELECT count(*) FROM payments`); n != 1 {
		t.Errorf("payments = %d", n)
	}
	if err := e.notifyAgain(r["payment_id"].(string)); err != nil {
		t.Fatal(err)
	}
	if got := e.sms.all(); len(got) != 1 {
		t.Errorf("sent %d SMS, want exactly 1", len(got))
	}

	// The link in the SMS opens that receipt for the parent — and only them.
	code := strings.SplitN(sms.Message, "#", 2)[1]
	res := want(t, e.parent().post("/api/links/resolve/", map[string]string{"code": code}), 200).JSON(t)
	if res["target"] != "/parent/fees?receipt="+r["payment_id"].(string) {
		t.Errorf("target: %v", res)
	}
	want(t, e.westT().post("/api/links/resolve/", map[string]string{"code": code}), 403)
	want(t, e.anon().post("/api/links/resolve/", map[string]string{"code": code}), 401)
}

// notifyAgain re-runs the notifier as a retry would.
func (e *env) notifyAgain(paymentID string) error {
	// Rebuild a notifier on the same pool/SMS stub and call it directly.
	return e.notifier().PaymentReceived(context.Background(), uuid.MustParse(paymentID))
}

func TestNameMatchingStillWorks(t *testing.T) {
	e := newEnv(t)
	r := want(t, e.mpesa("QK2001", "brian kamau", 1000), 200).JSON(t)
	if r["matched_student_id"] != brianID.String() {
		t.Errorf("name match: %v", r)
	}
	r = want(t, e.mpesa("QK2002", "nobody at all", 1000), 200).JSON(t)
	if r["reconciliation_status"] != "needs_review" {
		t.Errorf("unknown reference should need review: %v", r)
	}
}

func TestManualPayment(t *testing.T) {
	e := newEnv(t)
	post := func(body map[string]interface{}) resp { return e.anon().post("/internal/payments/manual", body) }

	f := want(t, post(map[string]interface{}{"student_id": brianID, "amount": 100, "payment_method": "bank_transfer"}), 400).JSON(t)
	if f["fields"].(map[string]interface{})["reference"] == nil {
		t.Errorf("bank transfer without slip: %v", f)
	}
	want(t, post(map[string]interface{}{"student_id": brianID, "amount": 100, "payment_method": "cheque"}), 400)
	want(t, post(map[string]interface{}{"student_id": uuid.New(), "amount": 100, "payment_method": "cash"}), 400)

	key := uuid.NewString()
	body := map[string]interface{}{"student_id": brianID, "amount": 1500, "payment_method": "cash", "payer_name": "Mary", "idempotency_key": key}
	first := want(t, post(body), 201).JSON(t)
	if !strings.HasPrefix(first["receipt_no"].(string), "RCT-") {
		t.Errorf("receipt no: %v", first)
	}
	again := want(t, post(body), 200).JSON(t) // double-click
	if again["payment_id"] != first["payment_id"] || again["duplicate"] != true {
		t.Errorf("idempotency: %v vs %v", again, first)
	}
	if b := balance(t, e, brianID); b != 18500 {
		t.Errorf("balance = %v", b)
	}
	sms := e.sms.waitFor(t, "payment_received", 1)
	if len(e.sms.all()) != 1 || !strings.Contains(sms[0].Message, "KES 1,500 received") {
		t.Errorf("SMS: %v", e.sms.all())
	}
	if m := testutil.Scalar[string](t, e.pool, `SELECT payment_method FROM payments`); m != "cash" {
		t.Errorf("method %s", m)
	}
}

func TestFeeRemindersOnePerStudentWithAdmissionNo(t *testing.T) {
	e := newEnv(t)
	// A second unpaid invoice line for Brian must not mean a second text.
	testutil.MustExec(t, e.pool, `
		INSERT INTO invoices (student_id, term_id, invoice_no, invoice_date, due_date, total_amount, total_paid, balance, status)
		VALUES ($1, $2, 'INV-ADM-001', '2026-01-05', '2026-02-01', 5000, 0, 5000, 'unpaid')`, faithID, termID)
	r := want(t, e.anon().post("/internal/reminders/send", nil), 200).JSON(t)
	if r["sent"] != 2.0 {
		t.Errorf("sent %v, want 2 (one per student)", r["sent"])
	}
	msgs := e.sms.waitFor(t, "fee_reminder", 2)
	var brian string
	for _, m := range msgs {
		if m.Meta["student_id"] == brianID.String() {
			brian = m.Message
		}
	}
	if !strings.Contains(brian, "balance KES 20000.00") || !strings.Contains(brian, "Acc "+accountPrefix+"ADM-002") {
		t.Errorf("reminder: %q", brian)
	}
	testutil.MustExec(t, e.pool, `UPDATE students SET is_active = FALSE WHERE id = $1`, faithID)
	e.sms.reset()
	if r := want(t, e.anon().post("/internal/reminders/send", nil), 200).JSON(t); r["sent"] != 1.0 {
		t.Errorf("inactive students must be skipped: %v", r)
	}
}

func TestReceiptPDFAndVerification(t *testing.T) {
	e := newEnv(t)
	pid := want(t, e.mpesa("QK3001", accountPrefix+"ADM-002", 4000), 200).JSON(t)["payment_id"].(string)

	pdf := want(t, e.parent().get("/api/payments/"+pid+"/receipt/"), 200)
	if !bytes.HasPrefix(pdf.Body, []byte("%PDF")) || !bytes.Contains(pdf.Body, []byte("/Subtype /Image")) {
		t.Error("receipt PDF missing or without QR code")
	}
	testutil.MustExec(t, e.pool, `INSERT INTO users (id, password, email, role) VALUES ($1, 'x', 'p2@it', 'parent')`, uuid.MustParse("40000000-0000-0000-0000-0000000000ff"))
	other := e.as(uuid.MustParse("40000000-0000-0000-0000-0000000000ff"), "PARENT")
	want(t, other.get("/api/payments/"+pid+"/receipt/"), 403)

	links := api.NewLinksService(e.pool, e.keys, portalURL)
	tok := func(amount float64) string {
		u, err := links.ReceiptVerifyURL(uuid.MustParse(pid), amount)
		if err != nil {
			t.Fatal(err)
		}
		return strings.SplitN(u, "#", 2)[1]
	}
	verify := func(token string) map[string]interface{} {
		return want(t, e.anon().post("/api/verify/", map[string]string{"token": token}), 200).JSON(t)
	}
	if v := verify(tok(4000)); v["valid"] != true || v["amount"] != 4000.0 || v["student_initials"] != "B. K." || v["number"] != "MPESA-QK3001" {
		t.Errorf("genuine receipt: %v", v)
	}
	if v := verify(tok(9000)); v["valid"] != false || v["status"] != "amount_changed" {
		t.Errorf("edited amount: %v", v)
	}
	raw, _ := base64.RawURLEncoding.DecodeString(tok(4000))
	raw[30] ^= 1
	if v := verify(base64.RawURLEncoding.EncodeToString(raw)); v["valid"] != false {
		t.Errorf("tampered token accepted: %v", v)
	}
	testutil.MustExec(t, e.pool, `UPDATE payments SET status = 'reversed' WHERE id = $1`, pid)
	if v := verify(tok(4000)); v["valid"] != false || v["status"] != "reversed" {
		t.Errorf("reversed payment still valid: %v", v)
	}

	keys := want(t, e.anon().get("/.well-known/stmarys-keys.json"), 200).JSON(t)["keys"].([]interface{})
	if len(keys) != 1 || keys[0].(map[string]interface{})["alg"] != "Ed25519" {
		t.Errorf("published keys: %v", keys)
	}
}

func TestVerifyIsRateLimited(t *testing.T) {
	e := newEnv(t)
	limited := false
	for i := 0; i < 15; i++ {
		if e.anon().post("/api/verify/", map[string]string{"token": "x"}).Status == 429 {
			limited = true
			break
		}
	}
	if !limited {
		t.Error("15 rapid verify calls were never rate limited")
	}
}

func TestShortLinkLifecycle(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	record := uuid.New()
	link, err := shortlink.Issue(ctx, e.pool, portalURL, signedlink.PurposeReceipt, record, brianID)
	if err != nil {
		t.Fatal(err)
	}
	code := strings.SplitN(link, "#", 2)[1]
	if len(code) != 22 {
		t.Errorf("code length %d", len(code))
	}
	if n := testutil.Scalar[int](t, e.pool, `SELECT count(*) FROM short_links WHERE position($1::bytea in code_hash) > 0`, []byte(code)); n != 0 {
		t.Error("code stored in the clear")
	}
	p := e.parent()
	want(t, p.post("/api/links/resolve/", map[string]string{"code": code}), 200)
	want(t, p.post("/api/links/resolve/", map[string]string{"code": "AAAAAAAAAAAAAAAAAAAAAA"}), 404)
	if n := testutil.Scalar[int](t, e.pool, `SELECT use_count FROM short_links`); n != 1 {
		t.Errorf("use_count %d", n)
	}

	testutil.MustExec(t, e.pool, `UPDATE short_links SET expires_at = now() - interval '1 minute'`)
	want(t, p.post("/api/links/resolve/", map[string]string{"code": code}), 404)

	link2, _ := shortlink.Issue(ctx, e.pool, portalURL, signedlink.PurposeReceipt, record, brianID)
	if err := shortlink.Revoke(ctx, e.pool, signedlink.PurposeReceipt, record); err != nil {
		t.Fatal(err)
	}
	want(t, p.post("/api/links/resolve/", map[string]string{"code": strings.SplitN(link2, "#", 2)[1]}), 404)
}
