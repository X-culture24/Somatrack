package finance

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
)

type Service struct {
	fee         *FeeStore
	payroll     *PayrollStore
	disbursement *DisbursementStore
}

func NewService(fee *FeeStore, payroll *PayrollStore, disbursement *DisbursementStore) *Service {
	return &Service{fee: fee, payroll: payroll, disbursement: disbursement}
}

// RegisterPublicRoutes exposes Safaricom's B2C result/timeout callback webhooks.
// These must be publicly reachable HTTPS URLs once DARAJA_B2C_RESULT_URL /
// DARAJA_B2C_TIMEOUT_URL point at them in production.
func (svc *Service) RegisterPublicRoutes(r chi.Router) {
	r.Post("/b2c/callback/result", svc.handleB2CResultCallback)
	r.Post("/b2c/callback/timeout", svc.handleB2CTimeoutCallback)
}

func (svc *Service) RegisterInternalRoutes(r chi.Router) {
	r.Get("/invoices", svc.handleListInvoices)
	r.Get("/invoices/{id}", svc.handleGetInvoice)
	r.Get("/payments", svc.handleListPayments)
	r.Get("/fee-structures", svc.handleListFeeStructures)
	r.Get("/discounts", svc.handleListDiscounts)

	r.Post("/payroll/runs", svc.handleGeneratePayrollRun)
	r.Get("/payroll/runs", svc.handleListPayrollRuns)
	r.Get("/payroll/runs/{id}/entries", svc.handleListPayrollEntries)
	r.Post("/payroll/entries/{id}/disburse", svc.handleDisburseEntry)
}

func (svc *Service) handleListInvoices(w http.ResponseWriter, r *http.Request) {
	var sid *uuid.UUID
	if s := r.URL.Query().Get("student_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid student_id")
			return
		}
		sid = &id
	}
	var status *string
	if s := r.URL.Query().Get("status"); s != "" {
		status = &s
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	rows, total, err := svc.fee.ListInvoices(r.Context(), sid, status, page, pageSize)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"data": rows, "total_count": total, "page": page, "page_size": pageSize,
	})
}

func (svc *Service) handleGetInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid invoice id")
		return
	}
	inv, err := svc.fee.GetInvoice(r.Context(), id)
	if err != nil {
		httpx.ErrNotFound(w, "invoice not found")
		return
	}
	httpx.JSON(w, http.StatusOK, inv)
}

func (svc *Service) handleListPayments(w http.ResponseWriter, r *http.Request) {
	var sid *uuid.UUID
	if s := r.URL.Query().Get("student_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid student_id")
			return
		}
		sid = &id
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	rows, total, err := svc.fee.ListPayments(r.Context(), sid, page, pageSize)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"data": rows, "total_count": total, "page": page, "page_size": pageSize,
	})
}

func (svc *Service) handleListFeeStructures(w http.ResponseWriter, r *http.Request) {
	var ayID *uuid.UUID
	if s := r.URL.Query().Get("academic_year_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid academic_year_id")
			return
		}
		ayID = &id
	}
	var grade *int
	if s := r.URL.Query().Get("grade"); s != "" {
		g, err := strconv.Atoi(s)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid grade")
			return
		}
		grade = &g
	}
	rows, err := svc.fee.ListFeeStructures(r.Context(), ayID, grade)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"data": rows})
}

func (svc *Service) handleListDiscounts(w http.ResponseWriter, r *http.Request) {
	var sid *uuid.UUID
	if s := r.URL.Query().Get("student_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid student_id")
			return
		}
		sid = &id
	}
	rows, err := svc.fee.ListDiscounts(r.Context(), sid)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"data": rows})
}

func (svc *Service) handleGeneratePayrollRun(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Month       int    `json:"month" validate:"required,min=1,max=12"`
		Year        int    `json:"year" validate:"required"`
		ProcessedBy string `json:"processed_by" validate:"required,uuid"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	processedBy, err := uuid.Parse(body.ProcessedBy)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid processed_by")
		return
	}
	run, entries, err := svc.payroll.GeneratePayrollRun(r.Context(), body.Month, body.Year, processedBy)
	if err != nil {
		if err == ErrPayrollRunExists {
			httpx.ErrConflict(w, err.Error())
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{"run": run, "entries": entries})
}

func (svc *Service) handleListPayrollRuns(w http.ResponseWriter, r *http.Request) {
	rows, err := svc.payroll.ListPayrollRuns(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"data": rows})
}

func (svc *Service) handleListPayrollEntries(w http.ResponseWriter, r *http.Request) {
	runID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid run id")
		return
	}
	rows, err := svc.payroll.ListPayrollEntries(r.Context(), runID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"data": rows})
}

func (svc *Service) handleDisburseEntry(w http.ResponseWriter, r *http.Request) {
	entryID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid entry id")
		return
	}
	tx, err := svc.disbursement.InitiateSalaryPayment(r.Context(), entryID)
	if err != nil {
		if err == ErrAlreadyDisbursed || err == ErrEntryNotFound {
			httpx.ErrConflict(w, err.Error())
			return
		}
		if _, ok := err.(*DarajaAPIError); ok {
			httpx.JSON(w, http.StatusBadGateway, map[string]interface{}{
				"error": "daraja_error", "message": err.Error(), "transaction": tx,
			})
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusAccepted, tx)
}

func (svc *Service) handleB2CResultCallback(w http.ResponseWriter, r *http.Request) {
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpx.JSON(w, http.StatusOK, map[string]interface{}{"ResultCode": 0, "ResultDesc": "invalid payload ignored"})
		return
	}
	log.Printf("[finance-svc] B2C result callback: %v", payload)
	if err := svc.disbursement.HandleResultCallback(r.Context(), payload); err != nil {
		log.Printf("[finance-svc] handle result callback: %v", err)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"ResultCode": 0, "ResultDesc": "Callback received successfully"})
}

func (svc *Service) handleB2CTimeoutCallback(w http.ResponseWriter, r *http.Request) {
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpx.JSON(w, http.StatusOK, map[string]interface{}{"ResultCode": 0, "ResultDesc": "invalid payload ignored"})
		return
	}
	log.Printf("[finance-svc] B2C timeout callback: %v", payload)
	if err := svc.disbursement.HandleTimeoutCallback(r.Context(), payload); err != nil {
		log.Printf("[finance-svc] handle timeout callback: %v", err)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"ResultCode": 0, "ResultDesc": "Callback received successfully"})
}
