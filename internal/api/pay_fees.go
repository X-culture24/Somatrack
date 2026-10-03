package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/rpc"
	"github.com/stmaryskabete/lms/internal/shared/types"
)

// PayFeesService is the "M-Pesa Pay" button's actual backend — previously
// that button existed on both the web and mobile fee pages but did nothing
// at all. Proxies to payment-svc's Daraja STK Push client (internal/payments/stkpush.go),
// scoped the same way finance data already is: a parent can only trigger
// (or check on) a payment for their own child.
type PayFeesService struct {
	pool          *pgxpool.Pool
	paymentClient *rpc.PaymentClient
}

func NewPayFeesService(pool *pgxpool.Pool, paymentClient *rpc.PaymentClient) *PayFeesService {
	return &PayFeesService{pool: pool, paymentClient: paymentClient}
}

func (s *PayFeesService) RegisterRoutes(r chi.Router) {
	r.Post("/finance/pay/", s.initiate)
	r.Get("/finance/pay/{id}/status/", s.status)
	r.Post("/finance/pay/equity/", s.initiateEquity)
	r.Get("/finance/pay/equity/{id}/status/", s.equityStatus)
}

func (s *PayFeesService) canPayFor(r *http.Request, studentID uuid.UUID) (bool, error) {
	role, _ := r.Context().Value("role").(types.Role)
	userID, _ := r.Context().Value("user_id").(uuid.UUID)
	scope, err := resolveStudentScope(r.Context(), s.pool, role, userID)
	if err != nil {
		return false, err
	}
	return scope == nil || inScope(scope, studentID), nil
}

func (s *PayFeesService) initiate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StudentID uuid.UUID `json:"student_id" validate:"required"`
		Phone     string    `json:"phone" validate:"required"`
		Amount    float64   `json:"amount" validate:"required,gt=0"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	ok, err := s.canPayFor(r, body.StudentID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		httpx.ErrForbidden(w, "cannot pay fees for this student")
		return
	}
	userID, _ := r.Context().Value("user_id").(uuid.UUID)
	res, err := s.paymentClient.InitiateSTKPush(r.Context(), &rpc.STKPushInitiateRequest{
		StudentID: body.StudentID, InitiatedBy: userID, Phone: body.Phone, Amount: body.Amount,
	})
	if err != nil {
		WriteUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusAccepted, res)
}

func (s *PayFeesService) status(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	res, err := s.paymentClient.STKPushStatus(r.Context(), id)
	if err != nil {
		httpx.ErrNotFound(w, "payment request not found")
		return
	}
	ok, err := s.canPayFor(r, res.StudentID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		httpx.ErrForbidden(w, "cannot view this payment request")
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (s *PayFeesService) initiateEquity(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StudentID  uuid.UUID `json:"student_id" validate:"required"`
		Phone      string    `json:"phone" validate:"required"`
		Amount     float64   `json:"amount" validate:"required,gt=0"`
		WalletName string    `json:"wallet_name"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	ok, err := s.canPayFor(r, body.StudentID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		httpx.ErrForbidden(w, "cannot pay fees for this student")
		return
	}
	userID, _ := r.Context().Value("user_id").(uuid.UUID)
	res, err := s.paymentClient.InitiateEquityRTB(r.Context(), &rpc.EquityRTBInitiateRequest{
		StudentID: body.StudentID, InitiatedBy: userID,
		Phone: body.Phone, Amount: body.Amount, WalletName: body.WalletName,
	})
	if err != nil {
		WriteUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusAccepted, res)
}

func (s *PayFeesService) equityStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	res, err := s.paymentClient.EquityRTBStatus(r.Context(), id)
	if err != nil {
		httpx.ErrNotFound(w, "equity payment request not found")
		return
	}
	ok, err := s.canPayFor(r, res.StudentID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		httpx.ErrForbidden(w, "cannot view this payment request")
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}
