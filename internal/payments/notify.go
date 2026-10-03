package payments

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/rpc"
	"github.com/stmaryskabete/lms/internal/shortlink"
	"github.com/stmaryskabete/lms/internal/signedlink"
)

// Notifier texts the family when a payment is posted: amount, the balance
// left and a short link to the receipt. It runs for every way money arrives
// (M-Pesa C2B, Jenga/Equity IPN, STK push, cash/bank at the bursar), because
// they all end in Reconciler.enqueueNotifications.
type Notifier struct {
	pool      *pgxpool.Pool
	sms       *rpc.NotificationClient
	portalURL string
}

func NewNotifier(pool *pgxpool.Pool, sms *rpc.NotificationClient, portalURL string) *Notifier {
	return &Notifier{pool: pool, sms: sms, portalURL: portalURL}
}

// PaymentReceived sends the confirmation for paymentID at most once.
func (n *Notifier) PaymentReceived(ctx context.Context, paymentID uuid.UUID) error {
	var studentID uuid.UUID
	var amount, balance float64
	var status, first, last, payerPhone string
	var guardianPhone *string
	err := n.pool.QueryRow(ctx, `
		SELECT p.student_id, p.amount::float8, p.status, COALESCE(st.first_name, ''), COALESCE(st.last_name, ''),
		       COALESCE(p.payer_phone, ''),
		       COALESCE((SELECT SUM(i.balance) FROM invoices i WHERE i.student_id = st.id), 0)::float8,
		       (SELECT g.phone FROM student_guardians sg JOIN guardians g ON g.id = sg.guardian_id
		         WHERE sg.student_id = st.id AND COALESCE(g.phone, '') <> ''
		         ORDER BY sg.is_primary DESC, sg.contact_priority ASC LIMIT 1)
		FROM payments p JOIN students st ON st.id = p.student_id
		WHERE p.id = $1`, paymentID).Scan(&studentID, &amount, &status, &first, &last, &payerPhone, &balance, &guardianPhone)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if status != dbPaymentStatusConfirmed {
		return nil
	}
	// The guardian on file first — the payer may be a relative or an agent.
	to := payerPhone
	if guardianPhone != nil {
		to = *guardianPhone
	}
	if strings.TrimSpace(to) == "" {
		log.Printf("[payments] no phone to confirm payment %s", paymentID)
		return nil
	}

	link, err := shortlink.Issue(ctx, n.pool, n.portalURL, signedlink.PurposeReceipt, paymentID, studentID)
	if err != nil {
		return fmt.Errorf("receipt link: %w", err)
	}
	msg := PaymentSMS(first, last, amount, balance, link)

	// Claim before sending so a payment reported twice (IPN + C2B) or a retry
	// never texts the parent twice.
	tag, err := n.pool.Exec(ctx, `
		INSERT INTO payment_notifications (payment_id, recipient, message)
		VALUES ($1, $2, $3) ON CONFLICT (payment_id) DO NOTHING`, paymentID, to, msg)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	if err := n.sms.EnqueueSMS(ctx, &rpc.SendSMSRequest{
		To:      to,
		Message: msg,
		Meta:    map[string]string{"kind": "payment_received", "payment_id": paymentID.String(), "student_id": studentID.String()},
	}); err != nil {
		// Release the claim so a later retry can send it.
		_, _ = n.pool.Exec(ctx, `DELETE FROM payment_notifications WHERE payment_id = $1`, paymentID)
		return err
	}
	return nil
}

// PaymentSMS builds the confirmation text. It stays within one 160-character
// GSM-7 SMS for normal names and portal URLs: no em-dashes or curly quotes,
// which would switch the whole message to UCS-2 and a 70-character limit.
func PaymentSMS(first, last string, amount, balance float64, link string) string {
	name := strings.TrimSpace(first + " " + last)
	if len([]rune(name)) > 24 {
		name = strings.TrimSpace(first)
		if l := strings.TrimSpace(last); l != "" {
			name += " " + string([]rune(l)[0]) + "."
		}
	}
	bal := "Bal KES " + kesShort(balance)
	switch {
	case balance < -0.005:
		bal = "Credit KES " + kesShort(-balance)
	case math.Abs(balance) < 0.005:
		bal = "Fees fully paid"
	}
	return fmt.Sprintf("St.Mary's: KES %s received for %s. %s. Receipt: %s", kesShort(amount), gsmSafe(name), bal, link)
}

// kesShort: 4000 → "4,000"; 1250.5 → "1,250.50".
func kesShort(v float64) string {
	v = math.Round(v*100) / 100
	whole := int64(v)
	cents := int64(math.Round((v - float64(whole)) * 100))
	s := fmt.Sprintf("%d", whole)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if cents != 0 {
		return fmt.Sprintf("%s.%02d", b.String(), cents)
	}
	return b.String()
}

// gsmSafe replaces characters outside the GSM-7 basic set, so one odd
// letter in a name can't triple the cost of the message.
func gsmSafe(s string) string {
	const gsm = "@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞÆæßÉ !\"#¤%&'()*+,-./0123456789:;<=>?¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà"
	repl := map[rune]string{'’': "'", '‘': "'", '“': "\"", '”': "\"", '–': "-", '—': "-", 'á': "a", 'í': "i", 'ó': "o", 'ú': "u", 'â': "a", 'ê': "e", 'î': "i", 'ô': "o", 'û': "u", 'ç': "c", 'ë': "e", 'ï': "i"}
	var b strings.Builder
	for _, r := range s {
		switch {
		case strings.ContainsRune(gsm, r):
			b.WriteRune(r)
		case repl[r] != "":
			b.WriteString(repl[r])
		default:
			b.WriteByte('?')
		}
	}
	return b.String()
}

// notifyTimeout bounds the background send kicked off after a payment posts.
const notifyTimeout = 30 * time.Second
