package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey: apiKey,
	}
}

// RPCError preserves the internal service's real status code and response
// body so callers (api-gateway proxies) can pass through the true HTTP
// status — e.g. a 409 conflict — instead of collapsing every upstream error
// into a generic 502 Bad Gateway.
type RPCError struct {
	StatusCode int
	Body       string
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("upstream returned %d: %s", e.StatusCode, e.Body)
}

func (c *Client) do(ctx context.Context, method, path string, body, out interface{}) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal req: %w", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return fmt.Errorf("build req: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Service-Key", c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do req: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return &RPCError{StatusCode: resp.StatusCode, Body: string(b)}
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type FingerprintClient struct {
	*Client
}

func NewFingerprintClient(baseURL, apiKey string) *FingerprintClient {
	return &FingerprintClient{Client: NewClient(baseURL, apiKey)}
}

type DeviceRegisterRequest struct {
	Name        string `json:"name"`
	Location    string `json:"location"`
	DeviceType  string `json:"device_type"`
	SharedSecret string `json:"shared_secret"`
}

type DeviceRegisterResponse struct {
	DeviceID uuid.UUID `json:"device_id"`
	Name     string    `json:"name"`
	Location string    `json:"location"`
}

func (c *FingerprintClient) RegisterDevice(ctx context.Context, req *DeviceRegisterRequest) (*DeviceRegisterResponse, error) {
	var out DeviceRegisterResponse
	if err := c.do(ctx, http.MethodPost, "/internal/devices/register", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type StoreTemplateRequest struct {
	StudentID     uuid.UUID `json:"student_id"`
	TemplateB64   string    `json:"template_b64"`
	EnrolledBy    uuid.UUID `json:"enrolled_by"`
	DeviceID      uuid.UUID `json:"device_id"`
}

func (c *FingerprintClient) StoreTemplate(ctx context.Context, req *StoreTemplateRequest) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/internal/students/%s/template", req.StudentID), req, nil)
}

type GetTemplateResponse struct {
	TemplateB64   string    `json:"template_b64"`
	TemplateVersion int     `json:"template_version"`
	EnrolledAt    time.Time `json:"enrolled_at"`
	DeviceID      uuid.UUID `json:"device_id"`
	IsActive      bool      `json:"is_active"`
}

func (c *FingerprintClient) GetTemplate(ctx context.Context, studentID uuid.UUID) (*GetTemplateResponse, error) {
	var out GetTemplateResponse
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/internal/students/%s/template", studentID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *FingerprintClient) DeleteTemplate(ctx context.Context, studentID uuid.UUID) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/internal/students/%s/template", studentID), nil, nil)
}

type ScanEvent struct {
	DeviceID             uuid.UUID  `json:"device_id"`
	StudentID            *uuid.UUID `json:"student_id,omitempty"`
	StaffID              *uuid.UUID `json:"staff_id,omitempty"`
	TemplateID           *uuid.UUID `json:"template_id,omitempty"`
	FingerprintMatchScore *float64   `json:"fingerprint_match_score,omitempty"`
	ScanType             string     `json:"scan_type"`
	Timestamp            time.Time  `json:"timestamp"`
	RawScore             *float64   `json:"raw_score,omitempty"`
}

type ProcessScanResponse struct {
	Duplicate   bool       `json:"duplicate"`
	ScanID      uuid.UUID  `json:"scan_id"`
	AttendanceID *uuid.UUID `json:"attendance_id,omitempty"`
	StudentID   uuid.UUID  `json:"student_id"`
	StaffID     uuid.UUID  `json:"staff_id"`
}

type StoreStaffTemplateRequest struct {
	StaffID     uuid.UUID `json:"staff_id"`
	TemplateB64 string    `json:"template_b64"`
	EnrolledBy  uuid.UUID `json:"enrolled_by"`
	DeviceID    uuid.UUID `json:"device_id"`
}

func (c *FingerprintClient) StoreStaffTemplate(ctx context.Context, req *StoreStaffTemplateRequest) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/internal/staff/%s/template", req.StaffID), req, nil)
}

func (c *FingerprintClient) GetStaffTemplate(ctx context.Context, staffID uuid.UUID) (*GetTemplateResponse, error) {
	var out GetTemplateResponse
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/internal/staff/%s/template", staffID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *FingerprintClient) DeleteStaffTemplate(ctx context.Context, staffID uuid.UUID) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/internal/staff/%s/template", staffID), nil, nil)
}

func (c *FingerprintClient) ProcessScan(ctx context.Context, scan *ScanEvent) (*ProcessScanResponse, error) {
	var out ProcessScanResponse
	if err := c.do(ctx, http.MethodPost, "/internal/scans/process", scan, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type AttendanceQuery struct {
	StudentID *uuid.UUID `json:"student_id,omitempty"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	ScanType  *string    `json:"scan_type,omitempty"`
	Page      int        `json:"page"`
	PageSize  int        `json:"page_size"`
}

type AttendanceRecord struct {
	ID          uuid.UUID `json:"id"`
	StudentID   uuid.UUID `json:"student_id"`
	Date        string    `json:"date"`
	ScanType    string    `json:"scan_type"`
	Timestamp   time.Time `json:"timestamp"`
	DeviceID    uuid.UUID `json:"device_id"`
	RawScore    *float64  `json:"raw_score,omitempty"`
	MatchScore  *float64  `json:"match_score,omitempty"`
}

type AttendanceListResponse struct {
	Data       []AttendanceRecord `json:"data"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalCount int64              `json:"total_count"`
	TotalPages int                `json:"total_pages"`
}

func (c *FingerprintClient) ListAttendance(ctx context.Context, q *AttendanceQuery) (*AttendanceListResponse, error) {
	var out AttendanceListResponse
	if err := c.do(ctx, http.MethodPost, "/internal/attendance/list", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type DailyAttendanceSummaryRequest struct {
	ClassID *uuid.UUID `json:"class_id,omitempty"`
	Date    string     `json:"date"`
}

type DailyAttendanceSummaryResponse struct {
	Date          string  `json:"date"`
	PresentCount  int     `json:"present_count"`
	LateCount     int     `json:"late_count"`
	AbsentCount   int     `json:"absent_count"`
	TotalExpected int     `json:"total_expected"`
	AttendancePct float64 `json:"attendance_pct"`
}

func (c *FingerprintClient) DailyAttendanceSummary(ctx context.Context, req *DailyAttendanceSummaryRequest) (*DailyAttendanceSummaryResponse, error) {
	var out DailyAttendanceSummaryResponse
	if err := c.do(ctx, http.MethodPost, "/internal/attendance/summary/daily", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type ClassRosterAttendanceRequest struct {
	ClassID uuid.UUID `json:"class_id"`
	Date    string    `json:"date"`
}

type ClassRosterStudent struct {
	StudentID     uuid.UUID  `json:"student_id"`
	StudentName   string     `json:"student_name"`
	AdmissionNo   string     `json:"admission_no"`
	ClassID       *uuid.UUID `json:"class_id,omitempty"`
	ClassName     string     `json:"class_name,omitempty"`
	PresentDays   int        `json:"present_days"`
	LateDays      int        `json:"late_days"`
	AbsentDays    int        `json:"absent_days"`
	ExpectedDays  int        `json:"expected_days"`
	AttendancePct float64    `json:"attendance_pct"`
}

func (c *FingerprintClient) ClassRosterAttendance(ctx context.Context, req *ClassRosterAttendanceRequest) ([]ClassRosterStudent, error) {
	var out []ClassRosterStudent
	if err := c.do(ctx, http.MethodPost, "/internal/attendance/class/roster", req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type StudentAttendanceRangeRequest struct {
	StudentID uuid.UUID `json:"student_id"`
	StartDate string    `json:"start_date"`
	EndDate   string    `json:"end_date"`
}

type StudentAttendanceEvent struct {
	Date    string  `json:"date"`
	Status  string  `json:"status"`
	TimeIn  *string `json:"time_in,omitempty"`
	TimeOut *string `json:"time_out,omitempty"`
	Remarks string  `json:"remarks,omitempty"`
	Subject *string `json:"subject,omitempty"`
}

type StudentAttendanceRangeResponse struct {
	Events         []StudentAttendanceEvent `json:"events"`
	PresentDays    int                      `json:"present_days"`
	LateDays       int                      `json:"late_days"`
	AbsentDays     int                      `json:"absent_days"`
	ExpectedDays   int                      `json:"expected_days"`
	AttendancePct  float64                  `json:"attendance_pct"`
}

func (c *FingerprintClient) StudentAttendanceRange(ctx context.Context, req *StudentAttendanceRangeRequest) (*StudentAttendanceRangeResponse, error) {
	var out StudentAttendanceRangeResponse
	if err := c.do(ctx, http.MethodPost, "/internal/attendance/student/range", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type StaffAttendanceRangeRequest struct {
	StaffID   uuid.UUID `json:"staff_id"`
	StartDate string    `json:"start_date"`
	EndDate   string    `json:"end_date"`
}

type StaffAttendanceEvent struct {
	Date    string  `json:"date"`
	Status  string  `json:"status"`
	TimeIn  *string `json:"time_in,omitempty"`
	TimeOut *string `json:"time_out,omitempty"`
}

type StaffAttendanceRangeResponse struct {
	Events        []StaffAttendanceEvent `json:"events"`
	PresentDays   int                    `json:"present_days"`
	LateDays      int                    `json:"late_days"`
	AbsentDays    int                    `json:"absent_days"`
	ExpectedDays  int                    `json:"expected_days"`
	AttendancePct float64                `json:"attendance_pct"`
}

func (c *FingerprintClient) StaffAttendanceRange(ctx context.Context, req *StaffAttendanceRangeRequest) (*StaffAttendanceRangeResponse, error) {
	var out StaffAttendanceRangeResponse
	if err := c.do(ctx, http.MethodPost, "/internal/attendance/staff/range", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type FinanceClient struct {
	*Client
}

func NewFinanceClient(baseURL, apiKey string) *FinanceClient {
	return &FinanceClient{Client: NewClient(baseURL, apiKey)}
}

func withQuery(path string, q url.Values) string {
	if len(q) == 0 {
		return path
	}
	return path + "?" + q.Encode()
}

type InvoiceSummary struct {
	ID          uuid.UUID `json:"id"`
	StudentID   uuid.UUID `json:"student_id"`
	InvoiceNo   string    `json:"invoice_no"`
	TermID      uuid.UUID `json:"term_id"`
	TermName    string    `json:"term_name,omitempty"`
	InvoiceDate string    `json:"invoice_date"`
	DueDate     string    `json:"due_date"`
	Status      string    `json:"status"`
	TotalDue    float64   `json:"total_due"`
	TotalPaid   float64   `json:"total_paid"`
	Balance     float64   `json:"balance"`
}

type InvoiceLine struct {
	ID          uuid.UUID `json:"id"`
	LineType    string    `json:"line_type"`
	Description string    `json:"description"`
	Quantity    int       `json:"quantity"`
	UnitPrice   float64   `json:"unit_price"`
	Amount      float64   `json:"amount"`
}

type InvoiceDetail struct {
	InvoiceSummary
	Lines []InvoiceLine `json:"lines"`
}

type ListResponse[T any] struct {
	Data       []T   `json:"data"`
	TotalCount int64 `json:"total_count"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
}

func (c *FinanceClient) ListInvoices(ctx context.Context, studentID *uuid.UUID, status *string, page, pageSize int) (*ListResponse[InvoiceSummary], error) {
	q := url.Values{}
	if studentID != nil {
		q.Set("student_id", studentID.String())
	}
	if status != nil {
		q.Set("status", *status)
	}
	if page > 0 {
		q.Set("page", fmt.Sprintf("%d", page))
	}
	if pageSize > 0 {
		q.Set("page_size", fmt.Sprintf("%d", pageSize))
	}
	var out ListResponse[InvoiceSummary]
	if err := c.do(ctx, http.MethodGet, withQuery("/internal/invoices", q), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *FinanceClient) GetInvoice(ctx context.Context, id uuid.UUID) (*InvoiceDetail, error) {
	var out InvoiceDetail
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/internal/invoices/%s", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type PaymentSummary struct {
	ID            uuid.UUID `json:"id"`
	StudentID     uuid.UUID `json:"student_id"`
	ReceiptNo     string    `json:"receipt_no"`
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"payment_method"`
	PaymentDate   string    `json:"payment_date"`
	TransactionID string    `json:"transaction_id,omitempty"`
	Status        string    `json:"status"`
	PayerName     string    `json:"payer_name,omitempty"`
}

func (c *FinanceClient) ListPayments(ctx context.Context, studentID *uuid.UUID, page, pageSize int) (*ListResponse[PaymentSummary], error) {
	q := url.Values{}
	if studentID != nil {
		q.Set("student_id", studentID.String())
	}
	if page > 0 {
		q.Set("page", fmt.Sprintf("%d", page))
	}
	if pageSize > 0 {
		q.Set("page_size", fmt.Sprintf("%d", pageSize))
	}
	var out ListResponse[PaymentSummary]
	if err := c.do(ctx, http.MethodGet, withQuery("/internal/payments", q), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type FeeStructureRow struct {
	ID            uuid.UUID `json:"id"`
	Grade         int       `json:"grade"`
	TermNumber    int       `json:"term_number"`
	TotalDay      float64   `json:"total_day"`
	TotalBoarding float64   `json:"total_boarding"`
	IsActive      bool      `json:"is_active"`
}

func (c *FinanceClient) ListFeeStructures(ctx context.Context, academicYearID *uuid.UUID, grade *int) ([]FeeStructureRow, error) {
	q := url.Values{}
	if academicYearID != nil {
		q.Set("academic_year_id", academicYearID.String())
	}
	if grade != nil {
		q.Set("grade", fmt.Sprintf("%d", *grade))
	}
	var out struct {
		Data []FeeStructureRow `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, withQuery("/internal/fee-structures", q), nil, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

type DiscountRow struct {
	ID            uuid.UUID `json:"id"`
	StudentID     uuid.UUID `json:"student_id"`
	WaiverType    string    `json:"waiver_type"`
	Amount        float64   `json:"amount"`
	Percentage    *float64  `json:"percentage,omitempty"`
	Reason        string    `json:"reason,omitempty"`
	EffectiveDate string    `json:"effective_date"`
}

func (c *FinanceClient) ListDiscounts(ctx context.Context, studentID *uuid.UUID) ([]DiscountRow, error) {
	q := url.Values{}
	if studentID != nil {
		q.Set("student_id", studentID.String())
	}
	var out struct {
		Data []DiscountRow `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, withQuery("/internal/discounts", q), nil, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

type PayrollRun struct {
	ID          uuid.UUID `json:"id"`
	Month       int       `json:"month"`
	Year        int       `json:"year"`
	RunDate     string    `json:"run_date"`
	Status      string    `json:"status"`
	StaffCount  int       `json:"staff_count"`
	TotalNetPay float64   `json:"total_net_pay"`
}

type PayrollEntry struct {
	ID                 uuid.UUID `json:"id"`
	PayrollRunID       uuid.UUID `json:"payroll_run_id"`
	StaffID            uuid.UUID `json:"staff_id"`
	StaffName          string    `json:"staff_name"`
	EmployeeNo         string    `json:"employee_no"`
	BasicSalary        float64   `json:"basic_salary"`
	Allowances         float64   `json:"allowances"`
	Deductions         float64   `json:"deductions"`
	GrossPay           float64   `json:"gross_pay"`
	NetPay             float64   `json:"net_pay"`
	PAYE               float64   `json:"paye"`
	NHIF               float64   `json:"nhif"`
	NSSF               float64   `json:"nssf"`
	DisbursementStatus string    `json:"disbursement_status"`
	DisbursementRef    string    `json:"disbursement_ref,omitempty"`
}

func (c *FinanceClient) GeneratePayrollRun(ctx context.Context, month, year int, processedBy uuid.UUID) (*PayrollRun, []PayrollEntry, error) {
	body := map[string]interface{}{"month": month, "year": year, "processed_by": processedBy}
	var out struct {
		Run     PayrollRun     `json:"run"`
		Entries []PayrollEntry `json:"entries"`
	}
	if err := c.do(ctx, http.MethodPost, "/internal/payroll/runs", body, &out); err != nil {
		return nil, nil, err
	}
	return &out.Run, out.Entries, nil
}

func (c *FinanceClient) ListPayrollRuns(ctx context.Context) ([]PayrollRun, error) {
	var out struct {
		Data []PayrollRun `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/internal/payroll/runs", nil, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

func (c *FinanceClient) ListPayrollEntries(ctx context.Context, runID uuid.UUID) ([]PayrollEntry, error) {
	var out struct {
		Data []PayrollEntry `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/internal/payroll/runs/%s/entries", runID), nil, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

type B2CTransaction struct {
	ID                       uuid.UUID  `json:"id"`
	PayrollEntryID           *uuid.UUID `json:"payroll_entry_id,omitempty"`
	StaffID                  *uuid.UUID `json:"staff_id,omitempty"`
	PhoneNumber              string     `json:"phone_number"`
	Amount                   float64    `json:"amount"`
	Status                   string     `json:"status"`
	OriginatorConversationID string     `json:"originator_conversation_id"`
	ConversationID           string     `json:"conversation_id,omitempty"`
	ResultDesc               string     `json:"result_desc,omitempty"`
	TransactionID            string     `json:"transaction_id,omitempty"`
}

func (c *FinanceClient) DisburseEntry(ctx context.Context, entryID uuid.UUID) (*B2CTransaction, error) {
	var out B2CTransaction
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/internal/payroll/entries/%s/disburse", entryID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type PaymentClient struct {
	*Client
}

func NewPaymentClient(baseURL, apiKey string) *PaymentClient {
	return &PaymentClient{Client: NewClient(baseURL, apiKey)}
}

type ProcessMpesaPaymentRequest struct {
	TransID         string    `json:"trans_id"`
	TransAmount     float64   `json:"trans_amount"`
	BillRefNumber   string    `json:"bill_ref_number"`
	TransTime       string    `json:"trans_time"`
	FirstName       string    `json:"first_name,omitempty"`
	MiddleName      string    `json:"middle_name,omitempty"`
	LastName        string    `json:"last_name,omitempty"`
	MSISDN          string    `json:"msisdn,omitempty"`
	RawPayload      string    `json:"raw_payload"`
}

type ProcessMpesaPaymentResponse struct {
	Status                string     `json:"status"`
	PaymentID             *uuid.UUID `json:"payment_id,omitempty"`
	MatchedStudentID      *uuid.UUID `json:"matched_student_id,omitempty"`
	InvoiceID             *uuid.UUID `json:"invoice_id,omitempty"`
	ReconciliationStatus  string     `json:"reconciliation_status"`
}

func (c *PaymentClient) ProcessMpesaPayment(ctx context.Context, req *ProcessMpesaPaymentRequest) (*ProcessMpesaPaymentResponse, error) {
	var out ProcessMpesaPaymentResponse
	if err := c.do(ctx, http.MethodPost, "/internal/payments/mpesa/process", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type InvoiceTotalRequest struct {
	InvoiceID uuid.UUID `json:"invoice_id"`
}

type InvoiceTotalResponse struct {
	InvoiceID  uuid.UUID `json:"invoice_id"`
	TotalDue   float64   `json:"total_due"`
	TotalPaid  float64   `json:"total_paid"`
	Balance    float64   `json:"balance"`
	Status     string    `json:"status"`
}

func (c *PaymentClient) RecomputeInvoiceTotals(ctx context.Context, invoiceID uuid.UUID) (*InvoiceTotalResponse, error) {
	var out InvoiceTotalResponse
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/internal/invoices/%s/recompute", invoiceID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *PaymentClient) SendReminders(ctx context.Context) (int, error) {
	var out struct{ Sent int `json:"sent"` }
	if err := c.do(ctx, http.MethodPost, "/internal/reminders/send", nil, &out); err != nil {
		return 0, err
	}
	return out.Sent, nil
}

type NotificationClient struct {
	*Client
}

func NewNotificationClient(baseURL, apiKey string) *NotificationClient {
	return &NotificationClient{Client: NewClient(baseURL, apiKey)}
}

type SendSMSRequest struct {
	To      string            `json:"to"`
	Message string            `json:"message"`
	Meta    map[string]string `json:"meta,omitempty"`
}

func (c *NotificationClient) EnqueueSMS(ctx context.Context, req *SendSMSRequest) error {
	return c.do(ctx, http.MethodPost, "/internal/sms/enqueue", req, nil)
}

type SendEmailRequest struct {
	To          []string          `json:"to"`
	Subject     string            `json:"subject"`
	HTMLBody    string            `json:"html_body,omitempty"`
	TextBody    string            `json:"text_body,omitempty"`
	Attachments map[string][]byte `json:"attachments,omitempty"`
}

func (c *NotificationClient) EnqueueEmail(ctx context.Context, req *SendEmailRequest) error {
	return c.do(ctx, http.MethodPost, "/internal/email/enqueue", req, nil)
}

type GuardianNotification struct {
	GuardianID   uuid.UUID         `json:"guardian_id"`
	StudentID    uuid.UUID         `json:"student_id"`
	TemplateKey  string            `json:"template_key"`
	TemplateData map[string]string `json:"template_data"`
	SMS          bool              `json:"sms"`
	Email        bool              `json:"email"`
}

func (c *NotificationClient) DispatchGuardianNotification(ctx context.Context, req *GuardianNotification) error {
	return c.do(ctx, http.MethodPost, "/internal/notifications/guardian", req, nil)
}

type ReportClient struct {
	*Client
}

func NewReportClient(baseURL, apiKey string) *ReportClient {
	return &ReportClient{Client: NewClient(baseURL, apiKey)}
}

type AttendanceReportRequest struct {
	StartDate  string     `json:"start_date"`
	EndDate    string     `json:"end_date"`
	ClassID    *uuid.UUID `json:"class_id,omitempty"`
	StreamID   *uuid.UUID `json:"stream_id,omitempty"`
	StudentID  *uuid.UUID `json:"student_id,omitempty"`
}

type AttendanceReportRow struct {
	StudentID     uuid.UUID `json:"student_id"`
	StudentName   string    `json:"student_name"`
	ClassID       uuid.UUID `json:"class_id,omitempty"`
	ClassName     string    `json:"class_name,omitempty"`
	ExpectedDays  int       `json:"expected_days"`
	PresentDays   int       `json:"present_days"`
	AbsentDays    int       `json:"absent_days"`
	AttendancePct float64   `json:"attendance_pct"`
}

func (c *ReportClient) AttendanceReport(ctx context.Context, req *AttendanceReportRequest) ([]AttendanceReportRow, error) {
	var out []AttendanceReportRow
	if err := c.do(ctx, http.MethodPost, "/internal/reports/attendance", req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type GradesReportRequest struct {
	TermID    uuid.UUID  `json:"term_id"`
	ClassID   *uuid.UUID `json:"class_id,omitempty"`
	SubjectID *uuid.UUID `json:"subject_id,omitempty"`
	StudentID *uuid.UUID `json:"student_id,omitempty"`
}

type GradesReportRow struct {
	StudentID     uuid.UUID `json:"student_id"`
	StudentName   string    `json:"student_name"`
	ClassID       uuid.UUID `json:"class_id,omitempty"`
	ClassName     string    `json:"class_name,omitempty"`
	SubjectID     uuid.UUID `json:"subject_id,omitempty"`
	SubjectName   string    `json:"subject_name,omitempty"`
	MeanScore     float64   `json:"mean_score"`
	MeanGrade     string    `json:"mean_grade,omitempty"`
	TotalStudents int       `json:"total_students,omitempty"`
	Rank          int       `json:"rank,omitempty"`
}

func (c *ReportClient) GradesReport(ctx context.Context, req *GradesReportRequest) ([]GradesReportRow, error) {
	var out []GradesReportRow
	if err := c.do(ctx, http.MethodPost, "/internal/reports/grades", req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type FeeCollectionRequest struct {
	TermID    *uuid.UUID `json:"term_id,omitempty"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}

type FeeCollectionReportRow struct {
	ClassID        uuid.UUID `json:"class_id,omitempty"`
	ClassName      string    `json:"class_name,omitempty"`
	TotalStudents  int       `json:"total_students"`
	ExpectedAmount float64   `json:"expected_amount"`
	CollectedAmount float64  `json:"collected_amount"`
	Outstanding    float64   `json:"outstanding"`
	CollectionPct  float64   `json:"collection_pct"`
}

func (c *ReportClient) FeeCollectionReport(ctx context.Context, req *FeeCollectionRequest) ([]FeeCollectionReportRow, error) {
	var out []FeeCollectionReportRow
	if err := c.do(ctx, http.MethodPost, "/internal/reports/fee-collection", req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type ClassPerformanceRequest struct {
	TermID    uuid.UUID `json:"term_id"`
	ClassID   uuid.UUID `json:"class_id"`
}

type ClassPerformanceRow struct {
	SubjectID    uuid.UUID `json:"subject_id"`
	SubjectName  string    `json:"subject_name"`
	MeanScore    float64   `json:"mean_score"`
	PassCount    int       `json:"pass_count"`
	TotalStudents int      `json:"total_students"`
	PassPct      float64   `json:"pass_pct"`
}

func (c *ReportClient) ClassPerformanceReport(ctx context.Context, req *ClassPerformanceRequest) ([]ClassPerformanceRow, error) {
	var out []ClassPerformanceRow
	if err := c.do(ctx, http.MethodPost, "/internal/reports/class-performance", req, &out); err != nil {
		return nil, err
	}
	return out, nil
}
