package payments

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/rpc"
	"github.com/stmaryskabete/lms/internal/shared/types"
)

type Service struct {
	matcher         *Matcher
	reconciler      *Reconciler
	processor       *AsynqProcessor
	sharedSecret    string
	allowedIPs      []string
	notificationCli *rpc.NotificationClient
	nairobiLoc      *time.Location
	useAsync        bool
}

func NewService(matcher *Matcher, reconciler *Reconciler, processor *AsynqProcessor, sharedSecret string, allowedIPs []string, notifCli *rpc.NotificationClient, loc *time.Location, useAsync bool) *Service {
	return &Service{
		matcher:         matcher,
		reconciler:      reconciler,
		processor:       processor,
		sharedSecret:    sharedSecret,
		allowedIPs:      allowedIPs,
		notificationCli: notifCli,
		nairobiLoc:      loc,
		useAsync:        useAsync,
	}
}

func (s *Service) RegisterPublicRoutes(r chi.Router) {
	r.Post("/webhooks/mpesa/confirmation", s.handleMpesaConfirmation)
	r.Post("/webhooks/mpesa/validation", s.handleMpesaValidation)
}

func (s *Service) RegisterInternalRoutes(r chi.Router) {
	r.Post("/payments/mpesa/process", s.handleInternalProcessPayment)
	r.Post("/invoices/{invoiceID}/recompute", s.handleRecomputeInvoice)
	r.Post("/reminders/send", s.handleSendReminders)
}

// realIP resolves the client IP for the MPESA_ALLOWED_IPS allowlist check.
// X-Forwarded-For is attacker-controlled end to end: a caller can prepend
// any IP they like before the real chain, so trusting its *first* entry (as
// this used to) lets anyone bypass the allowlist by sending
// "X-Forwarded-For: <an-allowed-ip>". nginx (the only reverse proxy in this
// deployment — see deploy/nginx.conf) always sets X-Real-IP to the true
// connecting peer, overwriting any client-supplied value, so that's
// trustworthy; the *last* X-Forwarded-For entry (appended by that same
// nearest hop) is the next best fallback when serving traffic directly.
func realIP(r *http.Request) string {
	if rip := strings.TrimSpace(r.Header.Get("X-Real-IP")); rip != "" && net.ParseIP(rip) != nil {
		return rip
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(parts[i])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *Service) checkIP(ip string) bool {
	if len(s.allowedIPs) == 0 {
		return true
	}
	for _, allowed := range s.allowedIPs {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			continue
		}
		if strings.Contains(allowed, "/") {
			_, n, err := net.ParseCIDR(allowed)
			if err == nil && n.Contains(net.ParseIP(ip)) {
				return true
			}
			continue
		}
		if allowed == ip {
			return true
		}
	}
	return false
}

func (s *Service) checkSecret(r *http.Request) bool {
	h := r.Header.Get("X-Mpesa-Secret")
	if h == "" {
		h = r.Header.Get("X-Shared-Secret")
	}
	if s.sharedSecret == "" {
		return true
	}
	return subtle.ConstantTimeCompare([]byte(h), []byte(s.sharedSecret)) == 1
}

type mpesaConfirmationPayload struct {
	TransID       string  `json:"TransID"`
	TransAmount   float64 `json:"TransAmount,string"`
	BillRefNumber string  `json:"BillRefNumber"`
	TransTime     string  `json:"TransTime"`
	FirstName     string  `json:"FirstName"`
	MiddleName    string  `json:"MiddleName"`
	LastName      string  `json:"LastName"`
	MSISDN        string  `json:"MSISDN"`
}

func (s *Service) handleMpesaValidation(w http.ResponseWriter, r *http.Request) {
	ip := realIP(r)
	if !s.checkIP(ip) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if !s.checkSecret(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ResultCode": "0",
		"ResultDesc": "Accepted",
	})
}

func (s *Service) handleMpesaConfirmation(w http.ResponseWriter, r *http.Request) {
	ip := realIP(r)
	if !s.checkIP(ip) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if !s.checkSecret(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var raw map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	b, _ := json.Marshal(raw)
	rawStr := string(b)

	bb, _ := json.Marshal(raw)
	var payload mpesaConfirmationPayload
	_ = json.Unmarshal(bb, &payload)

	if payload.TransID == "" {
		http.Error(w, "TransID required", http.StatusBadRequest)
		return
	}
	eventID, skip, err := s.reconciler.EnsureWebhookEvent(r.Context(), payload.TransID, rawStr)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	_ = eventID
	params := ProcessPaymentParams{
		TransID:       payload.TransID,
		TransAmount:   payload.TransAmount,
		BillRefNumber: payload.BillRefNumber,
		TransTime:     payload.TransTime,
		FirstName:     payload.FirstName,
		MiddleName:    payload.MiddleName,
		LastName:      payload.LastName,
		MSISDN:        payload.MSISDN,
		RawPayload:    rawStr,
	}
	if !skip {
		if s.useAsync && s.processor != nil {
			_, err = s.processor.EnqueueProcess(r.Context(), params)
			if err != nil {
				_, _, _, _, err = s.reconciler.ProcessPayment(r.Context(), params, s.matcher)
			}
		} else {
			_, _, _, _, err = s.reconciler.ProcessPayment(r.Context(), params, s.matcher)
		}
		_ = err
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ResultCode": "0",
		"ResultDesc": "Accepted",
	})
}

func (s *Service) handleInternalProcessPayment(w http.ResponseWriter, r *http.Request) {
	var p rpc.ProcessMpesaPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	params := ProcessPaymentParams{
		TransID:       p.TransID,
		TransAmount:   p.TransAmount,
		BillRefNumber: p.BillRefNumber,
		TransTime:     p.TransTime,
		FirstName:     p.FirstName,
		MiddleName:    p.MiddleName,
		LastName:      p.LastName,
		MSISDN:        p.MSISDN,
		RawPayload:    p.RawPayload,
	}
	paymentID, studentID, invoiceID, status, err := s.reconciler.ProcessPayment(r.Context(), params, s.matcher)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	out := rpc.ProcessMpesaPaymentResponse{
		Status:                "ok",
		PaymentID:             paymentID,
		MatchedStudentID:      studentID,
		InvoiceID:             invoiceID,
		ReconciliationStatus:  status,
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (s *Service) handleRecomputeInvoice(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "invoiceID")
	invoiceID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid invoice id")
		return
	}
	_, due, paid, bal, status, err := s.reconciler.RecomputeInvoiceTotals(r.Context(), invoiceID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"invoice_id": invoiceID,
		"total_due":  due,
		"total_paid": paid,
		"balance":    bal,
		"status":     status,
	})
}

func (s *Service) handleSendReminders(w http.ResponseWriter, r *http.Request) {
	sent, err := s.SendFeeReminders(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]int{"sent": sent})
}

func (s *Service) SendFeeReminders(ctx context.Context) (int, error) {
	if s.notificationCli == nil {
		return 0, nil
	}
	rows, err := s.reconciler.pool.Query(ctx, `
		SELECT DISTINCT ON (i.id)
		       i.id, i.student_id, i.balance, s.first_name, s.last_name,
		       g.phone AS guardian_phone, g.email AS guardian_email
		FROM invoices i
		JOIN students s ON s.id = i.student_id
		LEFT JOIN student_guardians sg ON sg.student_id = s.id
		LEFT JOIN guardians g ON g.id = sg.guardian_id
		WHERE i.balance > 0
		  AND EXISTS (SELECT 1 FROM terms t WHERE t.id = i.term_id AND t.is_current = TRUE)
		ORDER BY i.id, sg.is_primary DESC, sg.contact_priority ASC`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	type reminderRow struct {
		InvoiceID uuid.UUID
		StudentID uuid.UUID
		Balance   float64
		FirstName string
		LastName  string
		Phone     *string
		Email     *string
	}
	var pending []reminderRow
	for rows.Next() {
		var r reminderRow
		if err := rows.Scan(&r.InvoiceID, &r.StudentID, &r.Balance, &r.FirstName, &r.LastName, &r.Phone, &r.Email); err != nil {
			continue
		}
		pending = append(pending, r)
	}
	for _, r := range pending {
		if r.Phone != nil {
			msg := fmt.Sprintf("ACK St.Mary's Reminder: %s %s has a balance of KES %.2f. Pay via M-Pesa Paybill 247247 Acc %s.",
				r.FirstName, r.LastName, r.Balance, s.matcher.accountPrefix+r.StudentID.String()[:6])
			_ = s.notificationCli.EnqueueSMS(ctx, &rpc.SendSMSRequest{
				To:      *r.Phone,
				Message: msg,
				Meta:    map[string]string{"student_id": r.StudentID.String(), "invoice_id": r.InvoiceID.String()},
			})
			count++
		}
	}
	return count, nil
}

func init() {
	_ = types.PaginatedResponse[string]{}
}
