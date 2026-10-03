package api

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jung-kurt/gofpdf"
	qrcode "github.com/skip2/go-qrcode"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/types"
)

// DocumentsService renders invoices and payment receipts as downloadable
// PDFs — previously there was no export path at all, just the on-screen
// tables on InvoicesPage/PaymentsPage. It reads straight off the gateway's
// own pool (invoices/invoice_lines/payments/invoice_payment_links all live
// in the same Postgres database finance-svc writes to), the same
// direct-to-DB pattern fee_analytics.go and dashboard.go already use for
// read-only reporting.
type DocumentsService struct {
	pool  *pgxpool.Pool
	links *LinksService // signs the QR code on receipts
}

func NewDocumentsService(pool *pgxpool.Pool, links *LinksService) *DocumentsService {
	return &DocumentsService{pool: pool, links: links}
}

func (s *DocumentsService) RegisterRoutes(r chi.Router) {
	r.Get("/invoices/{id}/pdf/", s.invoicePDF)
	r.Get("/payments/{id}/receipt/", s.paymentReceiptPDF)
	r.Get("/documents/cbc-report/", s.cbcReportPDF)
}

// canViewStudent reuses finance_proxy.go's resolveStudentScope so a parent
// can only ever download their own children's invoices/receipts (and a
// student only their own) — these handlers originally did no such check at
// all, so any authenticated user could fetch any other student's PDF by
// guessing/iterating a UUID.
func (s *DocumentsService) canViewStudent(r *http.Request, studentID uuid.UUID) bool {
	return canViewStudent(r, s.pool, studentID)
}

// canViewStudent is the shared "may this caller see this student's records"
// check, used by documents and SMS links alike.
func canViewStudent(r *http.Request, pool *pgxpool.Pool, studentID uuid.UUID) bool {
	role, _ := r.Context().Value("role").(types.Role)
	userID, _ := r.Context().Value("user_id").(uuid.UUID)
	scope, err := resolveStudentScope(r.Context(), pool, role, userID)
	if err != nil {
		return false
	}
	return scope == nil || inScope(scope, studentID)
}

const schoolName = "ACK St. Mary's School Kabete"

func kesAmount(v float64) string {
	return fmt.Sprintf("KES %s", formatThousands(v))
}

func formatThousands(v float64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	whole := int64(v)
	frac := int64((v-float64(whole))*100 + 0.5)
	s := fmt.Sprintf("%d", whole)
	out := ""
	for i, c := range reverseString(s) {
		if i > 0 && i%3 == 0 {
			out = "," + out
		}
		out = string(c) + out
	}
	sign := ""
	if neg {
		sign = "-"
	}
	return fmt.Sprintf("%s%s.%02d", sign, out, frac)
}

func reverseString(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func newDocPDF(title string) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 8, schoolName, "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, "P.O. Box, Kabete, Nairobi, Kenya", "", 1, "C", false, 0, "")
	pdf.Ln(4)
	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(0, 8, title, "", 1, "C", false, 0, "")
	pdf.Ln(4)
	pdf.SetFont("Helvetica", "", 10)
	return pdf
}

// addVerifyQR prints the signed verification QR code with a short caption.
// A failure to render just leaves the code off; the receipt still prints.
func addVerifyQR(pdf *gofpdf.Fpdf, url, heading, caption string) {
	png, err := qrcode.Encode(url, qrcode.Medium, 512)
	if err != nil {
		return
	}
	pdf.Ln(6)
	x, y := pdf.GetX(), pdf.GetY()
	const size = 32.0
	pdf.RegisterImageOptionsReader("verify-qr", gofpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(png))
	pdf.ImageOptions("verify-qr", x, y, size, size, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	pdf.SetXY(x+size+4, y+8)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(0, 5, heading, "", 2, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 8)
	pdf.MultiCell(0, 4, caption, "", "L", false)
	pdf.SetXY(x, y+size+2)
}

func pdfKeyValue(pdf *gofpdf.Fpdf, label, value string) {
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(45, 6, label, "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, value, "", 1, "L", false, 0, "")
}

func streamPDF(w http.ResponseWriter, pdf *gofpdf.Fpdf, filename string) {
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	if err := pdf.Output(w); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
	}
}

func (s *DocumentsService) invoicePDF(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}

	var studentID uuid.UUID
	var invoiceNo, invoiceDate, dueDate, status, studentName, admissionNo, className, termName, yearName string
	var totalAmount, totalPaid, totalWaivers, balance float64
	err = s.pool.QueryRow(r.Context(), `
		SELECT i.student_id, i.invoice_no, i.invoice_date::text, i.due_date::text, i.status,
		       st.full_name, st.admission_no, COALESCE(cg.name, ''),
		       t.name, ay.name,
		       i.total_amount, i.total_paid, i.total_waivers, i.balance
		FROM invoices i
		JOIN students st ON st.id = i.student_id
		LEFT JOIN class_groups cg ON cg.id = st.current_class_id
		JOIN terms t ON t.id = i.term_id
		JOIN academic_years ay ON ay.id = t.academic_year_id
		WHERE i.id = $1`, id).Scan(
		&studentID, &invoiceNo, &invoiceDate, &dueDate, &status, &studentName, &admissionNo, &className,
		&termName, &yearName, &totalAmount, &totalPaid, &totalWaivers, &balance)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.ErrNotFound(w, "invoice not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if !s.canViewStudent(r, studentID) {
		httpx.ErrForbidden(w, "cannot view this invoice")
		return
	}

	type line struct {
		Description string
		Quantity    int
		UnitPrice   float64
		Amount      float64
	}
	var lines []line
	rows, err := s.pool.Query(r.Context(), `
		SELECT description, quantity, unit_price, amount FROM invoice_lines
		WHERE invoice_id = $1 ORDER BY created_at`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var l line
			if rows.Scan(&l.Description, &l.Quantity, &l.UnitPrice, &l.Amount) == nil {
				lines = append(lines, l)
			}
		}
	}

	pdf := newDocPDF("Fee Invoice")
	pdfKeyValue(pdf, "Invoice No:", invoiceNo)
	pdfKeyValue(pdf, "Invoice Date:", invoiceDate)
	pdfKeyValue(pdf, "Due Date:", dueDate)
	pdfKeyValue(pdf, "Status:", status)
	pdf.Ln(2)
	pdfKeyValue(pdf, "Student:", fmt.Sprintf("%s (%s)", studentName, admissionNo))
	pdfKeyValue(pdf, "Class:", className)
	pdfKeyValue(pdf, "Term:", fmt.Sprintf("%s %s", termName, yearName))
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(27, 54, 93)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(90, 8, "Description", "1", 0, "L", true, 0, "")
	pdf.CellFormat(20, 8, "Qty", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 8, "Unit Price", "1", 0, "R", true, 0, "")
	pdf.CellFormat(35, 8, "Amount", "1", 1, "R", true, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "", 10)
	for _, l := range lines {
		pdf.CellFormat(90, 7, l.Description, "1", 0, "L", false, 0, "")
		pdf.CellFormat(20, 7, fmt.Sprintf("%d", l.Quantity), "1", 0, "C", false, 0, "")
		pdf.CellFormat(35, 7, formatThousands(l.UnitPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(35, 7, formatThousands(l.Amount), "1", 1, "R", false, 0, "")
	}
	pdf.Ln(4)

	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(145, 7, "Total Billed", "", 0, "R", false, 0, "")
	pdf.CellFormat(35, 7, kesAmount(totalAmount), "", 1, "R", false, 0, "")
	pdf.CellFormat(145, 7, "Total Paid", "", 0, "R", false, 0, "")
	pdf.CellFormat(35, 7, kesAmount(totalPaid), "", 1, "R", false, 0, "")
	if totalWaivers > 0 {
		pdf.CellFormat(145, 7, "Waivers", "", 0, "R", false, 0, "")
		pdf.CellFormat(35, 7, kesAmount(totalWaivers), "", 1, "R", false, 0, "")
	}
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(145, 9, "Balance", "", 0, "R", false, 0, "")
	pdf.CellFormat(35, 9, kesAmount(balance), "", 1, "R", false, 0, "")

	pdf.Ln(8)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.CellFormat(0, 5, "Generated on "+time.Now().Format("02 Jan 2006 15:04"), "", 1, "L", false, 0, "")

	streamPDF(w, pdf, "invoice-"+invoiceNo+".pdf")
}

func newReceiptPDF() *gofpdf.Fpdf {
	// A5 portrait (148 × 210 mm) — narrow receipt slip, not a full A4 document.
	pdf := gofpdf.New("P", "mm", "A5", "")
	pdf.SetMargins(10, 10, 10)
	pdf.AddPage()
	// School header
	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(0, 7, schoolName, "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 8)
	pdf.CellFormat(0, 5, "P.O. Box, Kabete, Nairobi, Kenya", "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 5, "Tel: +254 (0) 20 012 3456", "", 1, "C", false, 0, "")
	// Divider
	pdf.Ln(2)
	pdf.SetDrawColor(27, 54, 93)
	pdf.SetLineWidth(0.5)
	pdf.Line(10, pdf.GetY(), 138, pdf.GetY())
	pdf.Ln(3)
	// Receipt title
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(0, 7, "OFFICIAL PAYMENT RECEIPT", "", 1, "C", false, 0, "")
	pdf.Line(10, pdf.GetY(), 138, pdf.GetY())
	pdf.Ln(4)
	pdf.SetFont("Helvetica", "", 9)
	return pdf
}

const receiptCW = 128.0 // A5 (148mm) minus 10mm left+right margins

func receiptKV(pdf *gofpdf.Fpdf, label, value string) {
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(receiptCW*0.30, 5.5, label, "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(receiptCW*0.70, 5.5, value, "", 1, "L", false, 0, "")
}

func (s *DocumentsService) paymentReceiptPDF(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}

	var studentID uuid.UUID
	var receiptNo, paymentDate, method, status, transactionID, payerName, payerPhone, studentName, admissionNo, className string
	var amount float64
	err = s.pool.QueryRow(r.Context(), `
		SELECT p.student_id, p.receipt_no, p.payment_date::text, p.payment_method, p.status,
		       COALESCE(p.transaction_id,''), p.payer_name, p.payer_phone,
		       st.full_name, st.admission_no, COALESCE(cg.name,''), p.amount
		FROM payments p
		JOIN students st ON st.id = p.student_id
		LEFT JOIN class_groups cg ON cg.id = st.current_class_id
		WHERE p.id = $1`, id).Scan(
		&studentID, &receiptNo, &paymentDate, &method, &status, &transactionID, &payerName, &payerPhone,
		&studentName, &admissionNo, &className, &amount)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.ErrNotFound(w, "payment not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if !s.canViewStudent(r, studentID) {
		httpx.ErrForbidden(w, "cannot view this receipt")
		return
	}

	type alloc struct {
		InvoiceNo string
		TermLabel string
		Applied   float64
	}
	var allocs []alloc
	rows, err := s.pool.Query(r.Context(), `
		SELECT i.invoice_no, t.name || ' ' || ay.name, l.amount_applied
		FROM invoice_payment_links l
		JOIN invoices i ON i.id = l.invoice_id
		JOIN terms t ON t.id = i.term_id
		JOIN academic_years ay ON ay.id = t.academic_year_id
		WHERE l.payment_id = $1`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var a alloc
			if rows.Scan(&a.InvoiceNo, &a.TermLabel, &a.Applied) == nil {
				allocs = append(allocs, a)
			}
		}
	}

	pdf := newReceiptPDF()
	const cw = receiptCW

	// Student & payment info
	receiptKV(pdf, "Receipt No:", receiptNo)
	receiptKV(pdf, "Date:", paymentDate)
	receiptKV(pdf, "Student:", fmt.Sprintf("%s (%s)", studentName, admissionNo))
	if className != "" {
		receiptKV(pdf, "Class:", className)
	}
	receiptKV(pdf, "Method:", method)
	if transactionID != "" {
		receiptKV(pdf, "Txn ID:", transactionID)
	}
	if payerName != "" {
		receiptKV(pdf, "Paid by:", fmt.Sprintf("%s  %s", payerName, payerPhone))
	}

	// Amount box — prominent, centered
	pdf.Ln(5)
	pdf.SetFillColor(27, 54, 93)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(cw, 6, "AMOUNT RECEIVED", "", 1, "C", true, 0, "")
	pdf.SetFont("Helvetica", "B", 18)
	pdf.CellFormat(cw, 12, "KES "+formatThousands(amount), "", 1, "C", true, 0, "")
	pdf.SetTextColor(0, 0, 0)

	// Applied to invoices
	if len(allocs) > 0 {
		pdf.Ln(4)
		pdf.SetFont("Helvetica", "B", 8)
		pdf.SetFillColor(230, 235, 242)
		col1 := cw * 0.35
		col2 := cw * 0.40
		col3 := cw * 0.25
		pdf.CellFormat(col1, 6, "Invoice", "1", 0, "L", true, 0, "")
		pdf.CellFormat(col2, 6, "Term", "1", 0, "L", true, 0, "")
		pdf.CellFormat(col3, 6, "Applied", "1", 1, "R", true, 0, "")
		pdf.SetFont("Helvetica", "", 8)
		for _, a := range allocs {
			pdf.CellFormat(col1, 5.5, a.InvoiceNo, "1", 0, "L", false, 0, "")
			pdf.CellFormat(col2, 5.5, a.TermLabel, "1", 0, "L", false, 0, "")
			pdf.CellFormat(col3, 5.5, formatThousands(a.Applied), "1", 1, "R", false, 0, "")
		}
	}

	// Separator before QR / footer
	pdf.Ln(4)
	pdf.SetDrawColor(200, 200, 200)
	pdf.Line(10, pdf.GetY(), 138, pdf.GetY())
	pdf.Ln(3)
	pdf.SetFont("Helvetica", "I", 7)
	pdf.MultiCell(cw, 4,
		"Generated: "+time.Now().Format("02 Jan 2006 15:04")+
			"  |  This receipt confirms payment received, not fee balance cleared.",
		"", "C", false)

	// QR code (confirmed payments only)
	if status == "confirmed" {
		if url, err := s.links.ReceiptVerifyURL(id, amount); err == nil {
			addVerifyQR(pdf, url, "Scan to verify",
				"Digitally signed. Confirms receipt number, amount and date are genuine.")
		}
	}

	streamPDF(w, pdf, "receipt-"+receiptNo+".pdf")
}

// ---------------------------------------------------------------------------
// cbcReportPDF — GET /api/documents/cbc-report/?student_id=X&term_id=Y
//
// Generates a PDF matching the physical Summative Assessment Report cards
// used by ACK St. Mary's: school header, learner info, performance key table,
// learning-areas grid (3 assessments × EE/ME/AE/BE columns per subject),
// teacher comments, headteacher sign-off, opening/closing dates.
func (s *DocumentsService) cbcReportPDF(w http.ResponseWriter, r *http.Request) {
	studentID, err := uuid.Parse(r.URL.Query().Get("student_id"))
	if err != nil {
		httpx.ErrBadRequest(w, "student_id required")
		return
	}
	termID, err := uuid.Parse(r.URL.Query().Get("term_id"))
	if err != nil {
		httpx.ErrBadRequest(w, "term_id required")
		return
	}
	// Same rule as the results trail: staff see any learner, a parent or
	// student only their own.
	if rl := callerRole(r); !academicLead(rl) && !hasRole(rl, types.TeacherRoles) && !s.canViewStudent(r, studentID) {
		httpx.ErrForbidden(w, "cannot view this student's report card")
		return
	}

	// --- fetch student / class / term ---
	var studentName, admNo, className, gradeName, termName, teacherName string
	var termYear int
	err = s.pool.QueryRow(r.Context(), `
		SELECT s.full_name, s.admission_no,
		       COALESCE(cg.name,''),
		       CASE WHEN cg.grade IS NULL THEN '' ELSE 'Grade '||cg.grade::text END,
		       COALESCE(t.name,''),
		       COALESCE(EXTRACT(YEAR FROM t.start_date)::int, 0),
		       COALESCE(u.first_name||' '||u.last_name,'')
		FROM students s
		LEFT JOIN class_groups cg ON cg.id = s.current_class_id
		LEFT JOIN class_subjects cs ON cs.class_id = cg.id
		LEFT JOIN users u ON u.id = cs.teacher_id
		LEFT JOIN terms t ON t.id = $2
		WHERE s.id = $1
		LIMIT 1`, studentID, termID,
	).Scan(&studentName, &admNo, &className, &gradeName, &termName, &termYear, &teacherName)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.ErrNotFound(w, "student not found")
		} else {
			httpx.Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	// --- fetch signoff ---
	var teacherComment, perfLevel, htComment, openingDate, closingDate string
	var htSigned bool
	s.pool.QueryRow(r.Context(), `
		SELECT teacher_comment, general_performance_level,
		       headteacher_comment, headteacher_signed, opening_date, closing_date
		FROM cbc_signoffs WHERE student_id=$1 AND term_id=$2`, studentID, termID,
	).Scan(&teacherComment, &perfLevel, &htComment, &htSigned, &openingDate, &closingDate)

	// --- fetch marks: subject → cat1/cat2/cat3 level ---
	type subjectMarks struct {
		Name       string
		Cat1, Cat2, Cat3 string
		Reflection string
	}
	mrows, err := s.pool.Query(r.Context(), `
		SELECT subj.name, a.assessment_type, COALESCE(m.competency_level,''), COALESCE(m.reflection,'')
		FROM assessments a
		JOIN subjects subj ON subj.id = a.subject_id
		LEFT JOIN marks m ON m.assessment_id = a.id AND m.student_id = $1
		WHERE a.term_id = $2
		  AND a.class_id = (SELECT current_class_id FROM students WHERE id=$1)
		  AND a.assessment_type IN ('cat1','cat2','cat3')
		ORDER BY subj.name, a.assessment_type`, studentID, termID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer mrows.Close()
	subjectMap := map[string]*subjectMarks{}
	var subjectOrder []string
	for mrows.Next() {
		var subName, atype, level, refl string
		if mrows.Scan(&subName, &atype, &level, &refl) != nil {
			continue
		}
		if _, ok := subjectMap[subName]; !ok {
			subjectMap[subName] = &subjectMarks{Name: subName}
			subjectOrder = append(subjectOrder, subName)
		}
		sm := subjectMap[subName]
		switch atype {
		case "cat1":
			sm.Cat1 = level
			sm.Reflection = refl
		case "cat2":
			sm.Cat2 = level
		case "cat3":
			sm.Cat3 = level
		}
	}

	// --- build PDF ---
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	pageW, _ := pdf.GetPageSize()
	contentW := pageW - 30 // left+right margins

	// Header
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(contentW, 7, schoolName, "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(contentW, 5, "P.O. Box, Kabete, Nairobi, Kenya", "", 1, "C", false, 0, "")
	pdf.Ln(3)

	// Report title + ORIGINAL box side by side
	titleLine := fmt.Sprintf("SUMMATIVE ASSESSMENT REPORT  -  %s - %s %d", gradeName, termName, termYear)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(contentW-25, 7, titleLine, "", 0, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(25, 7, "ORIGINAL", "1", 1, "C", false, 0, "")
	pdf.Ln(3)

	// Learner info lines
	dotLine := func(label, value string) {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(40, 6, label, "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.CellFormat(contentW-40, 6, value, "B", 1, "L", false, 0, "")
	}
	dotLine("Learner's Name:", studentName)
	dotLine("Admission No:", admNo)
	dotLine("Name of the School:", schoolName)
	dotLine("Name of the Facilitator:", teacherName)
	dotLine("Class:", className)
	pdf.Ln(4)

	// Performance KEY table
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetFillColor(220, 220, 220)
	colWidths := []float64{42, 22, 28, 18, 28}
	headers := []string{"Performance Level", "Actual Level", "Raw Marks (%)", "Points", "Learner's Mark"}
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 5, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetFillColor(255, 255, 255)
	keyRows := []struct{ perf, actual, raw, pts string }{
		{"Exceeding Expectations", "EE1", "90-99", "8.0"},
		{"", "EE2", "75-89", "7.0"},
		{"Meeting Expectations", "ME1", "58-70", "6.0"},
		{"", "ME2", "41-57", "5.0"},
		{"Approaching Expectations", "AE1", "31-40", "4.0"},
		{"", "AE2", "21-30", "3.0"},
		{"Below Expectations", "BE1", "11-20", "2.0"},
		{"", "BE2", "1-10", "1.0"},
	}
	for _, kr := range keyRows {
		pdf.CellFormat(colWidths[0], 4, kr.perf, "LR", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[1], 4, kr.actual, "LR", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[2], 4, kr.raw, "LR", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[3], 4, kr.pts, "LR", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[4], 4, "", "LR", 0, "C", false, 0, "")
		pdf.Ln(-1)
	}
	// bottom border on key table
	pdf.CellFormat(colWidths[0]+colWidths[1]+colWidths[2]+colWidths[3]+colWidths[4], 0, "", "T", 1, "", false, 0, "")
	pdf.Ln(3)

	// Learning Areas grid
	// Column layout: Subject(35) + [Ass1: EE(7) ME(7) AE(7) BE(7)] + [Ass2: same] + [Ass3: same] + Reflection(20)
	subjectColW := 35.0
	levelColW := 7.0
	reflColW := contentW - subjectColW - levelColW*12 // remaining
	if reflColW < 14 {
		reflColW = 14
	}

	// Group header row
	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetFillColor(220, 220, 220)
	pdf.CellFormat(subjectColW, 5, "LEARNING AREAS", "1", 0, "C", true, 0, "")
	for _, ass := range []string{"1ST ASSESSMENT", "2ND ASSESSMENT", "3RD ASSESSMENT"} {
		pdf.CellFormat(levelColW*4, 5, ass, "1", 0, "C", true, 0, "")
	}
	pdf.CellFormat(reflColW, 5, "REFLECTION", "1", 1, "C", true, 0, "")

	// Level sub-headers
	pdf.CellFormat(subjectColW, 4, "", "LR", 0, "C", false, 0, "")
	for range []int{0, 1, 2} {
		for _, lv := range []string{"EE", "ME", "AE", "BE"} {
			pdf.CellFormat(levelColW, 4, lv, "1", 0, "C", true, 0, "")
		}
	}
	pdf.CellFormat(reflColW, 4, "", "LR", 1, "C", false, 0, "")

	// Subject rows
	pdf.SetFont("Helvetica", "", 7)
	pdf.SetFillColor(255, 255, 255)
	markCell := func(rowLevel, target string) {
		filled := rowLevel == target
		if filled {
			pdf.SetFillColor(40, 40, 40)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		pdf.CellFormat(levelColW, 5, "", "1", 0, "C", filled, 0, "")
		pdf.SetFillColor(255, 255, 255)
	}
	for _, name := range subjectOrder {
		sm := subjectMap[name]
		pdf.CellFormat(subjectColW, 5, name, "1", 0, "L", false, 0, "")
		for _, level := range []string{sm.Cat1, sm.Cat2, sm.Cat3} {
			for _, lv := range []string{"EE", "ME", "AE", "BE"} {
				markCell(level, lv)
			}
		}
		pdf.CellFormat(reflColW, 5, sm.Reflection, "1", 1, "L", false, 0, "")
	}
	// If no subjects recorded, add empty rows as placeholders
	if len(subjectOrder) == 0 {
		for i := 0; i < 8; i++ {
			pdf.CellFormat(subjectColW, 5, "", "1", 0, "L", false, 0, "")
			for j := 0; j < 12; j++ {
				pdf.CellFormat(levelColW, 5, "", "1", 0, "C", false, 0, "")
			}
			pdf.CellFormat(reflColW, 5, "", "1", 1, "L", false, 0, "")
		}
	}
	pdf.Ln(4)

	// Teacher comments
	pdf.SetFont("Helvetica", "B", 8)
	pdf.CellFormat(contentW, 5, "TEACHER'S COMMENTS ON COMPETENCIES ACHIEVED / BEHAVIOUR / ACQUISITION OF VALUES", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	if teacherComment == "" {
		for i := 0; i < 3; i++ {
			pdf.CellFormat(contentW, 6, "", "B", 1, "L", false, 0, "")
		}
	} else {
		pdf.MultiCell(contentW, 6, teacherComment, "B", "L", false)
	}
	pdf.Ln(3)

	// Headteacher comment
	if htComment != "" {
		pdf.SetFont("Helvetica", "B", 8)
		pdf.CellFormat(contentW, 5, "HEAD TEACHER'S REMARKS", "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.MultiCell(contentW, 6, htComment, "B", "L", false)
		pdf.Ln(3)
	}

	// General performance level
	pdf.SetFont("Helvetica", "B", 8)
	pdf.CellFormat(70, 5, "LEARNER'S GENERAL PERFORMANCE LEVEL:", "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(contentW-70, 5, perfLevel, "B", 1, "L", false, 0, "")
	pdf.Ln(5)

	// Signatures row
	sigW := contentW / 2
	pdf.SetFont("Helvetica", "", 8)
	pdf.CellFormat(sigW, 5, "Facilitator's Signature ..................................", "", 0, "L", false, 0, "")
	htSig := "Head Teacher's Signature .................................."
	if htSigned {
		htSig = "Head Teacher's Signature  [SIGNED " + time.Now().Format("02 Jan 2006") + "]"
	}
	pdf.CellFormat(sigW, 5, htSig, "", 1, "L", false, 0, "")
	pdf.Ln(4)

	// Opening/Closing dates
	od := openingDate
	if od == "" {
		od = "................................."
	}
	cd := closingDate
	if cd == "" {
		cd = "................................."
	}
	pdf.CellFormat(sigW, 5, "Opening Date: "+od, "", 0, "L", false, 0, "")
	pdf.CellFormat(sigW, 5, "Closing Date: "+cd, "", 1, "L", false, 0, "")
	pdf.Ln(6)

	// Footer
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(120, 120, 120)
	pdf.CellFormat(sigW, 5, gradeName, "", 0, "L", false, 0, "")
	pdf.CellFormat(sigW, 5, fmt.Sprintf("Generated %s", time.Now().Format("02 Jan 2006 15:04")), "", 1, "R", false, 0, "")

	fname := fmt.Sprintf("cbc-report-%s-%s-%d.pdf", admNo, termName, termYear)
	streamPDF(w, pdf, fname)
}
