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
	fee              *FeeStore
	payroll          *PayrollStore
	disbursement     *DisbursementStore
	equityDisburse   *EquityDisbursementStore
}

func NewService(fee *FeeStore, payroll *PayrollStore, disbursement *DisbursementStore) *Service {
	return &Service{fee: fee, payroll: payroll, disbursement: disbursement}
}

func (svc *Service) EnableEquityDisbursement(eq *EquityDisbursementStore) {
	svc.equityDisburse = eq
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
	r.Post("/fee-structures", svc.handleCreateFeeStructure)
	r.Get("/fee-structures/{id}", svc.handleGetFeeStructure)
	r.Put("/fee-structures/{id}", svc.handleUpdateFeeStructure)
	r.Delete("/fee-structures/{id}", svc.handleDeleteFeeStructure)

	r.Get("/discounts", svc.handleListDiscounts)
	r.Post("/discounts", svc.handleCreateDiscount)

	r.Post("/billing/generate", svc.handleGenerateBilling)
	r.Post("/students/{id}/bill", svc.handleBillStudent)

	r.Get("/adhoc-fees", svc.handleListAdhocFees)
	r.Post("/adhoc-fees", svc.handleCreateAdhocFee)
	r.Delete("/adhoc-fees/{id}", svc.handleDeleteAdhocFee)
	r.Post("/invoices/{id}/charges", svc.handleAddInvoiceCharge)

	r.Post("/payroll/runs", svc.handleGeneratePayrollRun)
	r.Get("/payroll/runs", svc.handleListPayrollRuns)
	r.Get("/payroll/runs/{id}/entries", svc.handleListPayrollEntries)
	r.Post("/payroll/entries/{id}/disburse", svc.handleDisburseEntry)
	r.Post("/payroll/entries/{id}/disburse/equity", svc.handleDisburseEntryEquity)
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

func (svc *Service) handleCreateFeeStructure(w http.ResponseWriter, r *http.Request) {
	var body FeeStructureInput
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	row, err := svc.fee.CreateFeeStructure(r.Context(), body)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, row)
}

func (svc *Service) handleGetFeeStructure(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid fee structure id")
		return
	}
	row, err := svc.fee.GetFeeStructure(r.Context(), id)
	if err != nil {
		httpx.ErrNotFound(w, "fee structure not found")
		return
	}
	httpx.JSON(w, http.StatusOK, row)
}

func (svc *Service) handleUpdateFeeStructure(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid fee structure id")
		return
	}
	var body FeeStructureInput
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	row, err := svc.fee.UpdateFeeStructure(r.Context(), id, body)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, row)
}

func (svc *Service) handleDeleteFeeStructure(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid fee structure id")
		return
	}
	if err := svc.fee.DeleteFeeStructure(r.Context(), id); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}

func (svc *Service) handleCreateDiscount(w http.ResponseWriter, r *http.Request) {
	var body DiscountInput
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	row, err := svc.fee.CreateDiscount(r.Context(), body)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, row)
}

func (svc *Service) handleGenerateBilling(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TermID string `json:"term_id" validate:"required,uuid"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	termID, err := uuid.Parse(body.TermID)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid term_id")
		return
	}
	result, err := svc.fee.GenerateInvoicesForTerm(r.Context(), termID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}

func (svc *Service) handleBillStudent(w http.ResponseWriter, r *http.Request) {
	studentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid student id")
		return
	}
	invoiceID, err := svc.fee.BillStudentForCurrentTerm(r.Context(), studentID)
	if err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"invoice_id": invoiceID})
}

func (svc *Service) handleListAdhocFees(w http.ResponseWriter, r *http.Request) {
	var termID *uuid.UUID
	if s := r.URL.Query().Get("term_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid term_id")
			return
		}
		termID = &id
	}
	rows, err := svc.fee.ListAdhocFees(r.Context(), termID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"data": rows})
}

func (svc *Service) handleCreateAdhocFee(w http.ResponseWriter, r *http.Request) {
	var body AdhocFeeInput
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	row, err := svc.fee.CreateAdhocFee(r.Context(), body)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, row)
}

func (svc *Service) handleDeleteAdhocFee(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid adhoc fee id")
		return
	}
	if err := svc.fee.DeleteAdhocFee(r.Context(), id); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}

func (svc *Service) handleAddInvoiceCharge(w http.ResponseWriter, r *http.Request) {
	invoiceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid invoice id")
		return
	}
	var body struct {
		AdhocFeeID  *uuid.UUID `json:"adhoc_fee_id,omitempty"`
		Description string     `json:"description"`
		Amount      float64    `json:"amount"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if err := svc.fee.AddInvoiceCharge(r.Context(), invoiceID, body.AdhocFeeID, body.Description, body.Amount); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	inv, err := svc.fee.GetInvoice(r.Context(), invoiceID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, inv)
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

func (svc *Service) handleDisburseEntryEquity(w http.ResponseWriter, r *http.Request) {
	if svc.equityDisburse == nil {
		httpx.JSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "equity_not_configured",
			"message": "Equity EFT disbursement is not enabled — check JENGA_* environment variables.",
		})
		return
	}
	entryID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid entry id")
		return
	}
	tx, err := svc.equityDisburse.InitiateEFT(r.Context(), entryID)
	if err != nil {
		switch err {
		case ErrAlreadyDisbursed, ErrEntryNotFound:
			httpx.ErrConflict(w, err.Error())
		case ErrNoBankAccount:
			httpx.ErrBadRequest(w, err.Error())
		default:
			httpx.JSON(w, http.StatusBadGateway, map[string]interface{}{
				"error": "equity_eft_error", "message": err.Error(), "transaction": tx,
			})
		}
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
