package payments

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var msisdnDigitsSTK = regexp.MustCompile(`\D`)

// NormalizeMSISDN converts common Kenyan phone formats to the 2547XXXXXXXX /
// 2541XXXXXXXX form Daraja expects. Duplicated from internal/finance/daraja.go
// (payment-svc and finance-svc are separate binaries/packages) rather than a
// cross-package import between two otherwise-independent microservices.
func NormalizeMSISDN(phone string) (string, error) {
	digits := msisdnDigitsSTK.ReplaceAllString(phone, "")
	switch {
	case len(digits) == 12 && digits[:3] == "254":
		return digits, nil
	case len(digits) == 10 && digits[0] == '0':
		return "254" + digits[1:], nil
	case len(digits) == 9 && (digits[0] == '7' || digits[0] == '1'):
		return "254" + digits, nil
	}
	return "", fmt.Errorf("%q is not a recognisable Safaricom MSISDN", phone)
}

// DarajaSTKClient drives Lipa Na M-Pesa Online (STK Push) — the flow that
// pops a PIN prompt on the payer's own phone, as opposed to the passive C2B
// listener elsewhere in this package that only reacts to a Paybill payment
// someone already made. This is what lets a parent tap "Pay" in the app and
// get charged directly, rather than having to go dial *334# themselves.
//
// Sandbox defaults (174379 / the well-known public sandbox passkey) are
// Safaricom's own published test values for exercising this API — every
// Daraja STK Push tutorial uses them, and a real production shortcode +
// passkey from the developer portal is a drop-in env var override.
type DarajaSTKClient struct {
	BaseURL        string
	ConsumerKey    string
	ConsumerSecret string
	ShortCode      string
	PassKey        string
	CallbackURL    string
	httpClient     *http.Client

	tokenMu     sync.Mutex
	cachedToken string
	tokenExpiry time.Time
}

func NewDarajaSTKClient() *DarajaSTKClient {
	env := getEnvSTK("DARAJA_ENV", "sandbox")
	base := "https://sandbox.safaricom.co.ke"
	if env == "production" {
		base = "https://api.safaricom.co.ke"
	}
	return &DarajaSTKClient{
		BaseURL:        getEnvSTK("DARAJA_BASE_URL", base),
		ConsumerKey:    os.Getenv("DARAJA_CONSUMER_KEY"),
		ConsumerSecret: os.Getenv("DARAJA_CONSUMER_SECRET"),
		ShortCode:      getEnvSTK("DARAJA_STK_SHORTCODE", "174379"),
		PassKey:        getEnvSTK("DARAJA_STK_PASSKEY", "bfb279f9aa9bdbcf158e97dd71a467cd2e0c893059b10f78e6b72ada1ed2c919"),
		CallbackURL:    os.Getenv("DARAJA_STK_CALLBACK_URL"),
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

func getEnvSTK(k, fallback string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return fallback
}

func (c *DarajaSTKClient) getAccessToken(forceRefresh bool) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	if !forceRefresh && c.cachedToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.cachedToken, nil
	}
	if c.ConsumerKey == "" || c.ConsumerSecret == "" {
		return "", fmt.Errorf("DARAJA_CONSUMER_KEY / DARAJA_CONSUMER_SECRET are not configured")
	}
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/oauth/v1/generate?grant_type=client_credentials", nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.ConsumerKey, c.ConsumerSecret)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to reach Daraja OAuth endpoint: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Daraja OAuth request failed (%d): %s", resp.StatusCode, string(body))
	}
	var data struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   string `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &data); err != nil || data.AccessToken == "" {
		return "", fmt.Errorf("Daraja OAuth response missing access_token: %s", string(body))
	}
	expiresIn, _ := strconv.Atoi(data.ExpiresIn)
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	c.cachedToken = data.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(expiresIn-60) * time.Second)
	return c.cachedToken, nil
}

type STKPushResult struct {
	MerchantRequestID string
	CheckoutRequestID string
	ResponseCode      string
	ResponseDesc      string
	CustomerMessage   string
	Raw               map[string]interface{}
}

// InitiatePush triggers the PIN prompt. amount is truncated to whole
// shillings — Daraja's Amount field is integer-only.
func (c *DarajaSTKClient) InitiatePush(phone string, amount float64, accountRef, description string) (*STKPushResult, error) {
	msisdn, err := NormalizeMSISDN(phone)
	if err != nil {
		return nil, err
	}
	if len(accountRef) > 12 {
		accountRef = accountRef[:12]
	}
	if len(description) > 13 {
		description = description[:13]
	}
	timestamp := time.Now().Format("20060102150405")
	password := base64.StdEncoding.EncodeToString([]byte(c.ShortCode + c.PassKey + timestamp))

	payload := map[string]interface{}{
		"BusinessShortCode": c.ShortCode,
		"Password":          password,
		"Timestamp":         timestamp,
		"TransactionType":   "CustomerPayBillOnline",
		"Amount":            strconv.Itoa(int(amount)),
		"PartyA":            msisdn,
		"PartyB":            c.ShortCode,
		"PhoneNumber":       msisdn,
		"CallBackURL":       c.CallbackURL,
		"AccountReference":  accountRef,
		"TransactionDesc":   description,
	}
	raw, err := c.post("/mpesa/stkpush/v1/processrequest", payload)
	if err != nil {
		return nil, err
	}
	res := &STKPushResult{Raw: raw}
	if v, ok := raw["MerchantRequestID"].(string); ok {
		res.MerchantRequestID = v
	}
	if v, ok := raw["CheckoutRequestID"].(string); ok {
		res.CheckoutRequestID = v
	}
	if v, ok := raw["ResponseCode"]; ok {
		res.ResponseCode = fmt.Sprintf("%v", v)
	}
	if v, ok := raw["ResponseDescription"].(string); ok {
		res.ResponseDesc = v
	}
	if v, ok := raw["CustomerMessage"].(string); ok {
		res.CustomerMessage = v
	}
	return res, nil
}

func (c *DarajaSTKClient) post(path string, payload map[string]interface{}) (map[string]interface{}, error) {
	token, err := c.getAccessToken(false)
	if err != nil {
		return nil, err
	}
	body, _ := json.Marshal(payload)
	do := func(tok string) (*http.Response, error) {
		req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		return c.httpClient.Do(req)
	}
	resp, err := do(token)
	if err != nil {
		return nil, fmt.Errorf("failed to reach Daraja at %s: %w", path, err)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		token, err = c.getAccessToken(true)
		if err != nil {
			return nil, err
		}
		resp, err = do(token)
		if err != nil {
			return nil, fmt.Errorf("failed to reach Daraja at %s: %w", path, err)
		}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		data = map[string]interface{}{"raw": string(raw)}
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Daraja request failed (%d): %v", resp.StatusCode, data)
	}
	return data, nil
}

// STKPushStore persists request state (stk_push_requests) and reconciles a
// successful callback into a real payments/invoice row via the shared
// Reconciler — same downstream effect as a C2B payment, just initiated by
// the parent instead of arriving passively.
type STKPushStore struct {
	pool        *pgxpool.Pool
	reconciler  *Reconciler
	client      *DarajaSTKClient
}

func NewSTKPushStore(pool *pgxpool.Pool, reconciler *Reconciler, client *DarajaSTKClient) *STKPushStore {
	return &STKPushStore{pool: pool, reconciler: reconciler, client: client}
}

type STKPushRequest struct {
	ID                uuid.UUID `json:"id"`
	StudentID         uuid.UUID `json:"student_id"`
	PhoneNumber       string    `json:"phone_number"`
	Amount            float64   `json:"amount"`
	Status            string    `json:"status"`
	CheckoutRequestID string    `json:"checkout_request_id,omitempty"`
	ResultDesc        string    `json:"result_desc,omitempty"`
	MpesaReceiptNo    string    `json:"mpesa_receipt_no,omitempty"`
}

// Initiate looks up the student's admission number for a human-readable
// account reference, fires the STK push, and records the attempt regardless
// of outcome — a rejected push is still worth showing in "payment history"
// so the parent isn't left wondering what happened.
func (s *STKPushStore) Initiate(ctx context.Context, studentID, initiatedBy uuid.UUID, phone string, amount float64) (*STKPushRequest, error) {
	var admissionNo, fullName string
	if err := s.pool.QueryRow(ctx, `SELECT admission_no, full_name FROM students WHERE id = $1`, studentID).
		Scan(&admissionNo, &fullName); err != nil {
		return nil, fmt.Errorf("student not found: %w", err)
	}

	id := uuid.New()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO stk_push_requests (id, student_id, initiated_by_id, phone_number, amount, account_reference, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending')`,
		id, studentID, initiatedBy, phone, amount, admissionNo)
	if err != nil {
		return nil, fmt.Errorf("record stk push request: %w", err)
	}

	res, sendErr := s.client.InitiatePush(phone, amount, admissionNo, "School fees - "+fullName)

	status := "failed"
	var checkoutID, merchantID, respDesc string
	var rawJSON []byte
	if sendErr == nil {
		checkoutID = res.CheckoutRequestID
		merchantID = res.MerchantRequestID
		respDesc = res.ResponseDesc
		rawJSON, _ = json.Marshal(res.Raw)
		if res.ResponseCode == "0" {
			status = "accepted"
		}
	} else {
		respDesc = sendErr.Error()
	}

	_, updErr := s.pool.Exec(ctx, `
		UPDATE stk_push_requests SET
			status = $1, checkout_request_id = NULLIF($2, ''), merchant_request_id = NULLIF($3, ''),
			result_desc = $4, raw_initiate_response = $5, updated_at = NOW()
		WHERE id = $6`,
		status, checkoutID, merchantID, respDesc, rawJSON, id)
	if updErr != nil {
		return nil, updErr
	}

	if sendErr != nil {
		return &STKPushRequest{ID: id, StudentID: studentID, PhoneNumber: phone, Amount: amount, Status: status, ResultDesc: respDesc}, sendErr
	}
	return &STKPushRequest{
		ID: id, StudentID: studentID, PhoneNumber: phone, Amount: amount,
		Status: status, CheckoutRequestID: checkoutID, ResultDesc: respDesc,
	}, nil
}

// HandleCallback processes Safaricom's async STK result. Correlates by
// CheckoutRequestID (set when the push was initiated) — an unknown one is
// acked anyway since there's nothing to reconcile against.
func (s *STKPushStore) HandleCallback(ctx context.Context, payload map[string]interface{}) error {
	body, _ := payload["Body"].(map[string]interface{})
	cb, _ := body["stkCallback"].(map[string]interface{})
	if cb == nil {
		return fmt.Errorf("missing Body.stkCallback in STK callback payload")
	}
	checkoutID, _ := cb["CheckoutRequestID"].(string)
	resultCodeRaw := cb["ResultCode"]
	resultDesc, _ := cb["ResultDesc"].(string)
	resultCode := fmt.Sprintf("%v", resultCodeRaw)

	var id, studentID uuid.UUID
	var phone string
	var amount float64
	err := s.pool.QueryRow(ctx, `
		SELECT id, student_id, phone_number, amount FROM stk_push_requests WHERE checkout_request_id = $1`,
		checkoutID).Scan(&id, &studentID, &phone, &amount)
	if err != nil {
		return nil // nothing to correlate — ack anyway, matches the C2B webhook's own tolerance
	}

	rawJSON, _ := json.Marshal(payload)
	status := "failed"
	var receiptNo string
	var transactionAmount float64 = amount

	if resultCode == "0" {
		status = "success"
		if meta, ok := cb["CallbackMetadata"].(map[string]interface{}); ok {
			if items, ok := meta["Item"].([]interface{}); ok {
				for _, it := range items {
					m, ok := it.(map[string]interface{})
					if !ok {
						continue
					}
					name, _ := m["Name"].(string)
					switch name {
					case "MpesaReceiptNumber":
						receiptNo, _ = m["Value"].(string)
					case "Amount":
						if v, ok := m["Value"].(float64); ok {
							transactionAmount = v
						}
					}
				}
			}
		}
	}

	var paymentID *uuid.UUID
	if status == "success" && receiptNo != "" {
		// ProcessKnownStudentPayment (like ProcessPayment) expects a row
		// already locked in mpesa_webhook_events — EnsureWebhookEvent
		// creates it. Without this it 500s with "lock event: no rows".
		if _, _, err := s.reconciler.EnsureWebhookEvent(ctx, receiptNo, string(rawJSON)); err != nil {
			return fmt.Errorf("ensure webhook event: %w", err)
		}
		pid, _, err := s.reconciler.ProcessKnownStudentPayment(ctx, studentID, ProcessPaymentParams{
			TransID:       receiptNo,
			TransAmount:   transactionAmount,
			BillRefNumber: "STK-" + checkoutID,
			MSISDN:        phone,
			RawPayload:    string(rawJSON),
		})
		if err != nil {
			return fmt.Errorf("reconcile stk push payment: %w", err)
		}
		paymentID = pid
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE stk_push_requests SET
			status = $1, result_code = $2, result_desc = $3, mpesa_receipt_no = NULLIF($4, ''),
			payment_id = $5, raw_callback = $6, updated_at = NOW()
		WHERE id = $7`,
		status, resultCode, resultDesc, receiptNo, paymentID, rawJSON, id)
	return err
}

// GetStatus lets the app poll while waiting for the async callback (the PIN
// prompt + callback round trip is typically 5-20s — polling every couple of
// seconds from the app is the standard STK Push UX pattern).
func (s *STKPushStore) GetStatus(ctx context.Context, id uuid.UUID) (*STKPushRequest, error) {
	var r STKPushRequest
	var checkoutID, resultDesc, receiptNo *string
	err := s.pool.QueryRow(ctx, `
		SELECT id, student_id, phone_number, amount, status, checkout_request_id, result_desc, mpesa_receipt_no
		FROM stk_push_requests WHERE id = $1`, id).
		Scan(&r.ID, &r.StudentID, &r.PhoneNumber, &r.Amount, &r.Status, &checkoutID, &resultDesc, &receiptNo)
	if err != nil {
		return nil, err
	}
	if checkoutID != nil {
		r.CheckoutRequestID = *checkoutID
	}
	if resultDesc != nil {
		r.ResultDesc = *resultDesc
	}
	if receiptNo != nil {
		r.MpesaReceiptNo = *receiptNo
	}
	return &r, nil
}
