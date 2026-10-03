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
	stkStore        *STKPushStore
	jenga           *jengaDeps
	equityRTB       *EquityRTBStore
}

func NewService(matcher *Matcher, reconciler *Reconciler, processor *AsynqProcessor, sharedSecret string, allowedIPs []string, notifCli *rpc.NotificationClient, loc *time.Location, useAsync bool, stkStore *STKPushStore) *Service {
	return &Service{
		matcher:         matcher,
		reconciler:      reconciler,
		processor:       processor,
		sharedSecret:    sharedSecret,
		allowedIPs:      allowedIPs,
		notificationCli: notifCli,
		nairobiLoc:      loc,
		useAsync:        useAsync,
		stkStore:        stkStore,
	}
}

func (s *Service) EnableEquityRTB(store *EquityRTBStore) {
	s.equityRTB = store
}

// RegisterPublicRoutes exposes the C2B callback endpoints. The path
// deliberately avoids the word "mpesa" — Safaricom's RegisterURL API
// rejects any Confirmation/Validation URL containing it
// ("400.003.02 Bad Request - Invalid ValidationURL - URL has the word MPESA"),
// confirmed against their own sandbox while wiring this up.
func (s *Service) RegisterPublicRoutes(r chi.Router) {
	r.Post("/webhooks/payments/confirmation", s.handleMpesaConfirmation)
	r.Post("/webhooks/payments/validation", s.handleMpesaValidation)
	r.Post("/webhooks/payments/stk-callback", s.handleSTKCallback)
	s.registerJengaPublicRoutes(r)
}

func (s *Service) RegisterInternalRoutes(r chi.Router) {
	r.Post("/payments/mpesa/process", s.handleInternalProcessPayment)
	r.Post("/invoices/{invoiceID}/recompute", s.handleRecomputeInvoice)
	r.Post("/reminders/send", s.handleSendReminders)
	r.Post("/payments/manual", s.handleManualPayment)
	r.Post("/payments/stk-push/initiate", s.handleSTKInitiate)
	r.Get("/payments/stk-push/{id}/status", s.handleSTKStatus)
	// Same handler as the public webhook — the gateway relays Safaricom's
	// callback here over the trusted internal channel when only the gateway
	// (not payment-svc directly) has a public tunnel, e.g. behind ngrok's
	// free-tier one-tunnel limit. See internal/api/pay_fees.go's callback relay.
	r.Post("/payments/stk-push/callback", s.handleSTKCallback)
	// Equity EazzyPay / Request to Pay (C2B)
	r.Post("/payments/equity/rtb/initiate", s.handleEquityRTBInitiate)
	r.Get("/payments/equity/rtb/{id}/status", s.handleEquityRTBStatus)
	s.registerJengaInternalRoutes(r)
}

func (s *Service) handleSTKInitiate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StudentID   uuid.UUID `json:"student_id"`
		InitiatedBy uuid.UUID `json:"initiated_by"`
		Phone       string    `json:"phone"`
		Amount      float64   `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if body.StudentID == uuid.Nil || body.Phone == "" || body.Amount <= 0 {
		httpx.ErrBadRequest(w, "student_id, phone, and a positive amount are required")
		return
	}
	req, err := s.stkStore.Initiate(r.Context(), body.StudentID, body.InitiatedBy, body.Phone, body.Amount)
	if err != nil && req == nil {
		httpx.Error(w, http.StatusBadGateway, err)
		return
	}
	httpx.JSON(w, http.StatusAccepted, req)
}

func (s *Service) handleSTKStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	req, err := s.stkStore.GetStatus(r.Context(), id)
	if err != nil {
		httpx.ErrNotFound(w, "stk push request not found")
		return
	}
	httpx.JSON(w, http.StatusOK, req)
}

func (s *Service) handleSTKCallback(w http.ResponseWriter, r *http.Request) {
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if err := s.stkStore.HandleCallback(r.Context(), payload); err != nil {
		// Still 200 — Safaricom retries aggressively on non-2xx, and a
		// processing error here is ours to fix, not theirs to resend for.
		httpx.JSON(w, http.StatusOK, map[string]string{"ResultCode": "0", "ResultDesc": "Accepted (processing error logged)"})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"ResultCode": "0", "ResultDesc": "Accepted"})
}

func (s *Service) handleEquityRTBInitiate(w http.ResponseWriter, r *http.Request) {
	if s.equityRTB == nil {
		httpx.JSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":   "equity_not_configured",
			"message": "Equity Request to Pay is not enabled — check JENGA_* environment variables.",
		})
		return
	}
	var body struct {
		StudentID  uuid.UUID `json:"student_id"`
		InitiatedBy uuid.UUID `json:"initiated_by"`
		Phone      string    `json:"phone"`
		Amount     float64   `json:"amount"`
		WalletName string    `json:"wallet_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if body.StudentID == uuid.Nil || body.Phone == "" || body.Amount <= 0 {
		httpx.ErrBadRequest(w, "student_id, phone, and a positive amount are required")
		return
	}
	req, err := s.equityRTB.Initiate(r.Context(), body.StudentID, body.InitiatedBy, body.Phone, body.WalletName, body.Amount)
	if err != nil && req == nil {
		httpx.Error(w, http.StatusBadGateway, err)
		return
	}
	httpx.JSON(w, http.StatusAccepted, req)
}

func (s *Service) handleEquityRTBStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if s.equityRTB == nil {
		httpx.ErrNotFound(w, "equity rtb not configured")
		return
	}
	req, err := s.equityRTB.GetStatus(r.Context(), id)
	if err != nil {
		httpx.ErrNotFound(w, "request not found")
		return
	}
	httpx.JSON(w, http.StatusOK, req)
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

// checkSecret verifies the shared secret. Safaricom's C2B confirmation/
// validation callbacks never carry custom headers — it just POSTs to
// whatever URL you registered — so the secret has to live in the URL
// (query string) you give Safaricom at registration time. The header
// checks remain for local/manual curl testing convenience.
func (s *Service) checkSecret(r *http.Request) bool {
	if s.sharedSecret == "" {
		return true
	}
	h := r.Header.Get("X-Mpesa-Secret")
	if h == "" {
		h = r.Header.Get("X-Shared-Secret")
	}
	if h == "" {
		h = r.URL.Query().Get("secret")
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
		s.dispatch(r.Context(), params)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ResultCode": "0",
		"ResultDesc": "Accepted",
	})
}

// dispatch hands a payment to the async worker when Redis is available,
// falling back to processing inline.
func (s *Service) dispatch(ctx context.Context, params ProcessPaymentParams) {
	if s.useAsync && s.processor != nil {
		if _, err := s.processor.EnqueueProcess(ctx, params); err == nil {
			return
		}
	}
	_, _, _, _, _ = s.reconciler.ProcessPayment(ctx, params, s.matcher)
}

func (s *Service) handleInternalProcessPayment(w http.ResponseWriter, r *http.Request) {
	var p rpc.ProcessMpesaPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if p.TransID == "" {
		httpx.ErrBadRequest(w, "trans_id required")
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
	// ProcessPayment expects a row already locked in mpesa_webhook_events (the
	// public webhook handler creates it via EnsureWebhookEvent before calling
	// this) — this internal entry point skipped that step entirely, so every
	// call here failed with "lock event: no rows in result set". This was
	// dead/broken code with no caller until the admin "simulate payment" tool
	// started using it.
	_, skip, err := s.reconciler.EnsureWebhookEvent(r.Context(), p.TransID, p.RawPayload)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if skip {
		httpx.ErrConflict(w, fmt.Sprintf("trans_id %q already processed", p.TransID))
		return
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
	// One reminder per student (summed over their current-term invoices), not
	// one per invoice — otherwise a parent gets several texts for one child.
	rows, err := s.reconciler.pool.Query(ctx, `
		WITH bal AS (
			SELECT i.student_id, SUM(i.balance) AS balance
			FROM invoices i
			JOIN terms t ON t.id = i.term_id AND t.is_current = TRUE
			WHERE i.balance > 0
			GROUP BY i.student_id
		)
		SELECT DISTINCT ON (s.id)
		       s.id, b.balance, s.first_name, s.last_name, s.admission_no,
		       g.phone AS guardian_phone
		FROM bal b
		JOIN students s ON s.id = b.student_id AND s.is_active = TRUE
		LEFT JOIN student_guardians sg ON sg.student_id = s.id
		LEFT JOIN guardians g ON g.id = sg.guardian_id
		ORDER BY s.id, sg.is_primary DESC, sg.contact_priority ASC`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	type reminderRow struct {
		StudentID   uuid.UUID
		Balance     float64
		FirstName   string
		LastName    string
		AdmissionNo string
		Phone       *string
	}
	var pending []reminderRow
	for rows.Next() {
		var r reminderRow
		if err := rows.Scan(&r.StudentID, &r.Balance, &r.FirstName, &r.LastName, &r.AdmissionNo, &r.Phone); err != nil {
			continue
		}
		pending = append(pending, r)
	}
	count := 0
	for _, r := range pending {
		if r.Phone == nil || strings.TrimSpace(*r.Phone) == "" {
			continue
		}
		// The account reference is the admission number — the key
		// Matcher.MatchStudent tries first, so the payment posts automatically.
		msg := fmt.Sprintf("ACK St.Mary's: Fee reminder for %s %s - balance KES %.2f. Pay via Paybill 247247, Acc %s%s.",
			r.FirstName, r.LastName, r.Balance, s.matcher.accountPrefix, r.AdmissionNo)
		if err := s.notificationCli.EnqueueSMS(ctx, &rpc.SendSMSRequest{
			To:      *r.Phone,
			Message: msg,
			Meta:    map[string]string{"student_id": r.StudentID.String(), "kind": "fee_reminder"},
		}); err != nil {
			continue
		}
		count++
	}
	return count, nil
}

func init() {
	_ = types.PaginatedResponse[string]{}
}
