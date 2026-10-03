package payments

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/stmaryskabete/lms/internal/jenga"
	"github.com/stmaryskabete/lms/internal/shared/httpx"
)

type jengaDeps struct {
	client        *jenga.Client
	auth          jenga.IPNAuth
	accountPrefix string
}

// EnableJenga wires Equity's Jenga IPN into the same match-and-reconcile path
// the Safaricom C2B confirmation uses. Paybill 247247 belongs to Equity, so
// its payments reach us through Jenga rather than through a C2B callback.
func (s *Service) EnableJenga(client *jenga.Client, auth jenga.IPNAuth, accountPrefix string) {
	s.jenga = &jengaDeps{client: client, auth: auth, accountPrefix: accountPrefix}
}

func (s *Service) registerJengaPublicRoutes(r chi.Router) {
	r.Post("/webhooks/payments/jenga-ipn", s.handleJengaIPNPublic)
}

func (s *Service) registerJengaInternalRoutes(r chi.Router) {
	// Relay target for the gateway's public IPN route; the gateway has
	// already checked Jenga's credentials before forwarding.
	r.Post("/payments/jenga/ipn", s.handleJengaIPNInternal)
	r.Get("/payments/jenga/status", s.handleJengaStatus)
}

func (s *Service) handleJengaIPNPublic(w http.ResponseWriter, r *http.Request) {
	if s.jenga == nil {
		http.Error(w, "jenga not enabled", http.StatusServiceUnavailable)
		return
	}
	if !s.jenga.auth.Check(r) {
		log.Printf("[payment-svc] jenga ipn rejected: bad credentials from %s", realIP(r))
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	s.handleJengaIPNInternal(w, r)
}

func (s *Service) handleJengaIPNInternal(w http.ResponseWriter, r *http.Request) {
	if s.jenga == nil {
		http.Error(w, "jenga not enabled", http.StatusServiceUnavailable)
		return
	}
	n, raw, err := jenga.DecodeIPN(r)
	if err != nil {
		log.Printf("[payment-svc] jenga ipn decode: %v raw=%s", err, raw)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.processJengaIPN(r.Context(), n, raw); err != nil {
		log.Printf("[payment-svc] jenga ipn: %v raw=%s", err, raw)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "received"})
}

func (s *Service) processJengaIPN(ctx context.Context, n *jenga.IPN, raw string) error {
	if !n.IsCredit() {
		log.Printf("[payment-svc] jenga ipn ignored (type=%q status=%q ref=%q)",
			n.Bank.TransactionType, n.Transaction.Status, n.Transaction.Reference)
		return nil
	}
	p, err := n.ToPayment(s.jenga.accountPrefix, s.nairobiLoc)
	if err != nil {
		log.Printf("[payment-svc] jenga ipn ignored: %v", err)
		return nil
	}
	// Keyed on the transaction reference, which for M-Pesa paybill payments
	// is the M-Pesa receipt, so a payment reported by both Jenga and a C2B
	// callback is only recorded once.
	_, skip, err := s.reconciler.EnsureWebhookEvent(ctx, p.TransID, raw)
	if err != nil {
		return err
	}
	if skip {
		return nil
	}
	s.dispatch(ctx, ProcessPaymentParams{
		TransID:       p.TransID,
		TransAmount:   p.Amount,
		BillRefNumber: p.BillRef,
		TransTime:     p.TransTime,
		FirstName:     p.PayerName,
		MSISDN:        p.PayerPhone,
		RawPayload:    raw,
	})
	return nil
}

// handleJengaStatus reports which credentials are still missing and, once
// they're all set, makes a live balance call to prove they actually work.
func (s *Service) handleJengaStatus(w http.ResponseWriter, r *http.Request) {
	cfg := jenga.ConfigFromEnv()
	out := map[string]interface{}{
		"env":                 cfg.Env,
		"missing":             cfg.Missing(),
		"ipn_auth_configured": s.jenga != nil && (s.jenga.auth.Username != "" || s.jenga.auth.Secret != ""),
	}
	if s.jenga == nil || !s.jenga.client.Configured() {
		out["api_ok"] = false
		httpx.JSON(w, http.StatusOK, out)
		return
	}
	bal, err := s.jenga.client.AccountBalance()
	if err != nil {
		out["api_ok"] = false
		out["error"] = err.Error()
	} else {
		out["api_ok"] = true
		b, _ := json.Marshal(bal)
		out["balance"] = json.RawMessage(b)
	}
	httpx.JSON(w, http.StatusOK, out)
}
