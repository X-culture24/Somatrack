package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/resource"
	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/types"
)

// Record-keeping forms served by the declarative resource engine. Module
// names follow the left rail of the reference screens in mock-up/. Tables
// come from migrations 000 and 018.
//
// Forms with real business rules (fees, payments, marks entry, admissions)
// stay in their hand-written services.

// Shorthands keep the specs readable.
type f = resource.Field

var (
	studentRef = &resource.RefSpec{Resource: "students", Table: "students", Label: "x.full_name || ' (' || x.admission_no || ')'"}
	staffRef   = &resource.RefSpec{Resource: "staff", Table: "staff_profiles",
		Label: "(SELECT u.first_name || ' ' || u.last_name FROM users u WHERE u.id = x.user_id)"}
	userRef    = &resource.RefSpec{Resource: "users", Table: "users", Label: "x.first_name || ' ' || x.last_name"}
	termRef    = &resource.RefSpec{Resource: "terms", Table: "terms", Label: "x.name"}
	yearRef    = &resource.RefSpec{Resource: "academic-years", Table: "academic_years", Label: "x.name"}
	classRef   = &resource.RefSpec{Resource: "classes", Table: "class_groups", Label: "x.name"}
	subjectRef = &resource.RefSpec{Resource: "subjects", Table: "subjects", Label: "x.name"}
	deptRef    = &resource.RefSpec{Resource: "departments", Table: "departments", Label: "x.name"}
)

func student(required bool) f {
	return f{Name: "student_id", Label: "Student", Type: resource.Ref, Ref: studentRef, Required: required, Filter: true, List: true}
}

var payMethods = []string{"cash", "mpesa", "bank", "cheque"}

func roles(groups ...[]types.Role) []types.Role {
	var out []types.Role
	seen := map[types.Role]bool{}
	for _, g := range groups {
		for _, r := range g {
			if !seen[r] {
				seen[r] = true
				out = append(out, r)
			}
		}
	}
	return out
}

var (
	financeStaff = roles(types.FinanceRoles, []types.Role{types.RoleHeadTeacher, types.RoleDeputyHeadTeacher})
	adminStaff   = types.AdminRoles
	adminFinance = roles(types.AdminRoles, types.FinanceRoles)
	teachingAdm  = roles(types.AdminRoles, types.TeacherRoles)
	leadership   = []types.Role{types.RoleSystemAdmin, types.RoleSchoolDirector, types.RoleHeadTeacher, types.RoleDeputyHeadTeacher}
)

func schoolResources() []resource.Resource {
	return []resource.Resource{
		// ------------------------------------------------------------ Students
		{
			Name: "admission-enquiries", Title: "Admission enquiry", Module: "Students", Table: "admission_enquiries",
			CreatorField: "created_by_id", HasUpdatedAt: true,
			Fields: []f{
				{Name: "child_name", Label: "Child's name", Type: resource.Text, Required: true, MaxLen: 200, Search: true, List: true, Section: "Child"},
				{Name: "date_of_birth", Label: "Date of birth", Type: resource.Date, Section: "Child"},
				{Name: "grade_sought", Label: "Grade sought", Type: resource.Text, MaxLen: 50, List: true, Filter: true, Section: "Child"},
				{Name: "parent_name", Label: "Parent/guardian name", Type: resource.Text, Required: true, MaxLen: 200, Search: true, List: true, Section: "Parent"},
				{Name: "parent_phone", Label: "Parent phone", Type: resource.Phone, Required: true, Search: true, List: true, Section: "Parent"},
				{Name: "parent_email", Label: "Parent email", Type: resource.Email, Section: "Parent"},
				{Name: "enquiry_date", Label: "Enquiry date", Type: resource.Date, List: true, Section: "Follow-up"},
				{Name: "source", Label: "How they heard of us", Type: resource.Enum, Options: []string{"walk_in", "phone", "website", "referral", "social_media", "other"}, Filter: true, Section: "Follow-up"},
				{Name: "status", Label: "Status", Type: resource.Enum, Options: []string{"new", "contacted", "visited", "applied", "admitted", "declined"}, Filter: true, List: true, Section: "Follow-up"},
				{Name: "follow_up_date", Label: "Follow-up date", Type: resource.Date, List: true, Section: "Follow-up"},
				{Name: "notes", Label: "Notes", Type: resource.LongText, Section: "Follow-up"},
			},
		},
		{
			Name: "pocket-money", Title: "Pocket money transaction", Module: "Students", Table: "pocket_money_transactions",
			StudentField: "student_id", CreatorField: "created_by_id", OrderBy: "t.txn_date DESC, t.created_at DESC",
			WriteRoles:  adminFinance,
			Constraints: map[string]string{"pocket_money_nonnegative": "This withdrawal is more than the student's pocket money balance."},
			Fields: []f{
				student(true),
				{Name: "txn_type", Label: "Type", Type: resource.Enum, Options: []string{"deposit", "withdrawal"}, Required: true, Filter: true, List: true},
				{Name: "amount", Label: "Amount (KES)", Type: resource.Decimal, Required: true, Min: resource.Float(0.01), List: true},
				{Name: "txn_date", Label: "Date", Type: resource.Date, List: true},
				{Name: "description", Label: "Description", Type: resource.Text, MaxLen: 255, Search: true, List: true},
			},
		},
		{
			Name: "discipline-cases", Title: "Discipline case", Module: "Students", Table: "discipline_cases",
			StudentField: "student_id", CreatorField: "created_by_id", HasUpdatedAt: true, OrderBy: "t.incident_date DESC",
			WriteRoles: teachingAdm,
			Fields: []f{
				student(true),
				{Name: "incident_date", Label: "Incident date", Type: resource.Date, Required: true, List: true},
				{Name: "severity", Label: "Severity", Type: resource.Enum, Options: []string{"minor", "major", "severe"}, Filter: true, List: true},
				{Name: "offence", Label: "Offence", Type: resource.Text, Required: true, MaxLen: 200, Search: true, List: true},
				{Name: "description", Label: "What happened", Type: resource.LongText},
				{Name: "action_taken", Label: "Action taken", Type: resource.LongText},
				{Name: "reported_by", Label: "Reported by", Type: resource.Text, MaxLen: 200},
				{Name: "parent_notified", Label: "Parent notified", Type: resource.Bool, List: true},
				{Name: "status", Label: "Status", Type: resource.Enum, Options: []string{"open", "resolved"}, Filter: true, List: true},
			},
		},
		{
			Name: "medical-records", Title: "Medical record", Module: "Students", Table: "medical_records",
			StudentField: "student_id", CreatorField: "administered_by_id", OrderBy: "t.visit_date DESC",
			ReadRoles: teachingAdm,
			Fields: []f{
				student(true),
				{Name: "visit_date", Label: "Visit date", Type: resource.Date, Required: true, List: true},
				{Name: "symptoms", Label: "Symptoms", Type: resource.LongText, List: true},
				{Name: "diagnosis", Label: "Diagnosis", Type: resource.LongText},
				{Name: "treatment_given", Label: "Treatment given", Type: resource.LongText},
				{Name: "medication_prescribed", Label: "Medication prescribed", Type: resource.LongText},
				{Name: "referred_to_hospital", Label: "Referred to hospital", Type: resource.Bool, List: true},
				{Name: "hospital_name", Label: "Hospital", Type: resource.Text, MaxLen: 200},
				{Name: "follow_up_date", Label: "Follow-up date", Type: resource.Date},
				{Name: "notes", Label: "Notes", Type: resource.LongText},
			},
		},
		{
			Name: "hobbies", Title: "Hobby / favourite", Module: "Students", Table: "student_hobbies",
			StudentField: "student_id", WriteRoles: teachingAdm,
			Fields: []f{
				student(true),
				{Name: "hobby", Label: "Hobby", Type: resource.Text, Required: true, MaxLen: 120, Search: true, List: true},
				{Name: "category", Label: "Category", Type: resource.Enum, Options: []string{"sport", "art", "music", "reading", "technology", "outdoors", "other"}, Filter: true, List: true},
				{Name: "notes", Label: "Notes", Type: resource.LongText},
			},
		},
		{
			Name: "leadership-positions", Title: "Leadership position", Module: "Students", Table: "leadership_positions",
			StudentField: "student_id", WriteRoles: teachingAdm,
			Fields: []f{
				student(true),
				{Name: "position", Label: "Position", Type: resource.Text, Required: true, MaxLen: 120, Search: true, List: true},
				{Name: "academic_year_id", Label: "Academic year", Type: resource.Ref, Ref: yearRef, Filter: true, List: true},
				{Name: "start_date", Label: "Start date", Type: resource.Date, List: true},
				{Name: "end_date", Label: "End date", Type: resource.Date},
				{Name: "notes", Label: "Notes", Type: resource.LongText},
			},
		},
		{
			Name: "leaving-certificates", Title: "Leaving certificate", Module: "Students", Table: "leaving_certificates",
			StudentField: "student_id", CreatorField: "created_by_id", OrderBy: "t.leaving_date DESC",
			WriteRoles: leadership,
			Fields: []f{
				student(true),
				{Name: "certificate_no", Label: "Certificate no.", Type: resource.Text, Required: true, MaxLen: 50, Search: true, List: true},
				{Name: "leaving_date", Label: "Leaving date", Type: resource.Date, Required: true, List: true},
				{Name: "reason", Label: "Reason", Type: resource.Enum, Options: []string{"completed", "transfer", "other"}, Filter: true, List: true},
				{Name: "last_grade", Label: "Last grade attended", Type: resource.Text, MaxLen: 50},
				{Name: "conduct", Label: "Conduct", Type: resource.Enum, Options: []string{"excellent", "good", "fair", "poor"}},
				{Name: "remarks", Label: "Remarks", Type: resource.LongText},
				{Name: "issued_by", Label: "Issued by", Type: resource.Text, MaxLen: 200},
			},
		},
		{
			Name: "student-withdrawals", Title: "Withdrawal", Module: "Students", Table: "student_withdrawals",
			StudentField: "student_id", CreatorField: "approved_by_id", OrderBy: "t.withdrawal_date DESC", WriteRoles: leadership,
			Fields: []f{
				student(true),
				{Name: "withdrawal_date", Label: "Withdrawal date", Type: resource.Date, Required: true, List: true},
				{Name: "reason_code", Label: "Reason", Type: resource.Enum, Options: []string{"financial", "academic", "disciplinary", "health", "family_relocation", "transfer", "other"}, Filter: true, List: true},
				{Name: "reason", Label: "Details", Type: resource.LongText},
				{Name: "refund_amount", Label: "Refund (KES)", Type: resource.Decimal, Min: resource.Float(0)},
			},
		},
		{
			Name: "emergency-contacts", Title: "Emergency contact", Module: "People", Table: "emergency_contacts",
			StudentField: "student_id",
			Fields: []f{
				student(true),
				{Name: "name", Label: "Name", Type: resource.Text, Required: true, MaxLen: 200, Search: true, List: true},
				{Name: "relationship", Label: "Relationship", Type: resource.Text, Required: true, MaxLen: 60, List: true},
				{Name: "phone", Label: "Phone", Type: resource.Phone, Required: true, Search: true, List: true},
				{Name: "alt_phone", Label: "Alternative phone", Type: resource.Phone},
				{Name: "address", Label: "Address", Type: resource.Text, MaxLen: 255},
			},
		},
		{
			Name: "certificates", Title: "Certificate", Module: "Academics", Table: "certificates",
			StudentField: "student_id",
			Fields: []f{
				student(true),
				{Name: "cert_type", Label: "Type", Type: resource.Enum, Options: []string{"national", "other"}, Filter: true, List: true},
				{Name: "title", Label: "Title", Type: resource.Text, Required: true, MaxLen: 200, Search: true, List: true},
				{Name: "certificate_no", Label: "Certificate no.", Type: resource.Text, MaxLen: 80, Search: true, List: true},
				{Name: "issuer", Label: "Issued by", Type: resource.Text, MaxLen: 200},
				{Name: "issue_date", Label: "Issue date", Type: resource.Date, List: true},
				{Name: "notes", Label: "Notes", Type: resource.LongText},
			},
		},

		// ------------------------------------------------------------ Boarding
		{
			Name: "hostels", Title: "Hostel / dormitory", Module: "Boarding", Table: "hostels", OrderBy: "t.name",
			Fields: []f{
				{Name: "name", Label: "Name", Type: resource.Text, Required: true, MaxLen: 120, Search: true, List: true},
				{Name: "gender", Label: "Gender", Type: resource.Enum, Options: []string{"boys", "girls", "mixed"}, Filter: true, List: true},
				{Name: "capacity", Label: "Capacity", Type: resource.Int, Min: resource.Float(0), List: true},
				{Name: "warden_name", Label: "Warden / matron", Type: resource.Text, MaxLen: 200, List: true},
				{Name: "warden_phone", Label: "Warden phone", Type: resource.Phone},
				{Name: "notes", Label: "Notes", Type: resource.LongText},
			},
		},
		{
			Name: "hostel-rooms", Title: "Hostel room", Module: "Boarding", Table: "hostel_rooms", OrderBy: "t.room_no",
			Fields: []f{
				{Name: "hostel_id", Label: "Hostel", Type: resource.Ref, Ref: &resource.RefSpec{Resource: "hostels", Table: "hostels", Label: "x.name"}, Required: true, Filter: true, List: true},
				{Name: "room_no", Label: "Room no.", Type: resource.Text, Required: true, MaxLen: 30, Search: true, List: true},
				{Name: "floor", Label: "Floor", Type: resource.Text, MaxLen: 30, List: true},
				{Name: "capacity", Label: "Capacity", Type: resource.Int, Min: resource.Float(0), List: true},
			},
		},
		{
			Name: "hostel-beds", Title: "Bed", Module: "Boarding", Table: "hostel_beds", OrderBy: "t.bed_no",
			Fields: []f{
				{Name: "room_id", Label: "Room", Type: resource.Ref, Required: true, Filter: true, List: true, Ref: &resource.RefSpec{Resource: "hostel-rooms", Table: "hostel_rooms",
					Label: "(SELECT h.name FROM hostels h WHERE h.id = x.hostel_id) || ' – ' || x.room_no"}},
				{Name: "bed_no", Label: "Bed no.", Type: resource.Text, Required: true, MaxLen: 30, Search: true, List: true},
				{Name: "status", Label: "Status", Type: resource.Enum, Options: []string{"available", "occupied", "maintenance"}, Filter: true, List: true},
			},
		},
		{
			Name: "bed-allocations", Title: "Bed allocation", Module: "Boarding", Table: "bed_allocations",
			StudentField: "student_id",
			Constraints:  map[string]string{"bed_not_in_maintenance": "This bed is under maintenance and can't be allocated."},
			Fields: []f{
				student(true),
				{Name: "bed_id", Label: "Bed", Type: resource.Ref, Required: true, Filter: true, List: true, Ref: &resource.RefSpec{Resource: "hostel-beds", Table: "hostel_beds",
					Label: "(SELECT h.name || ' – ' || r.room_no FROM hostel_rooms r JOIN hostels h ON h.id = r.hostel_id WHERE r.id = x.room_id) || ' / bed ' || x.bed_no"}},
				{Name: "term_id", Label: "Term", Type: resource.Ref, Ref: termRef, Filter: true, List: true},
				{Name: "start_date", Label: "From", Type: resource.Date, List: true},
				{Name: "end_date", Label: "To", Type: resource.Date},
				{Name: "status", Label: "Status", Type: resource.Enum, Options: []string{"active", "ended"}, Filter: true, List: true},
			},
		},

		// ------------------------------------------------------------ Front office
		{
			Name: "visitors", Title: "Visitor", Module: "Front office", Table: "visitors",
			CreatorField: "created_by_id", OrderBy: "t.time_in DESC",
			Fields: []f{
				{Name: "visitor_name", Label: "Visitor name", Type: resource.Text, Required: true, MaxLen: 200, Search: true, List: true},
				{Name: "phone", Label: "Phone", Type: resource.Phone, Search: true, List: true},
				{Name: "id_number", Label: "ID number", Type: resource.Text, MaxLen: 30, Search: true},
				{Name: "purpose", Label: "Purpose of visit", Type: resource.Text, Required: true, MaxLen: 255, List: true},
				{Name: "person_to_see", Label: "Person to see", Type: resource.Text, MaxLen: 200, List: true},
				{Name: "vehicle_reg", Label: "Vehicle reg.", Type: resource.Text, MaxLen: 20},
				{Name: "time_in", Label: "Time in", Type: resource.DateTime, List: true},
				{Name: "time_out", Label: "Time out", Type: resource.DateTime, List: true},
				{Name: "notes", Label: "Notes", Type: resource.LongText},
			},
		},

		// ------------------------------------------------------------ Procurement
		{
			Name: "suppliers", Title: "Supplier", Module: "Procurement", Table: "suppliers", OrderBy: "t.name",
			ReadRoles: adminFinance, WriteRoles: financeStaff,
			Fields: []f{
				{Name: "name", Label: "Name", Type: resource.Text, Required: true, MaxLen: 200, Search: true, List: true},
				{Name: "contact_person", Label: "Contact person", Type: resource.Text, MaxLen: 200, List: true},
				{Name: "phone", Label: "Phone", Type: resource.Phone, Search: true, List: true},
				{Name: "email", Label: "Email", Type: resource.Email},
				{Name: "kra_pin", Label: "KRA PIN", Type: resource.Text, MaxLen: 20, Search: true},
				{Name: "address", Label: "Address", Type: resource.Text, MaxLen: 255},
				{Name: "payment_terms", Label: "Payment terms", Type: resource.Text, MaxLen: 100},
				{Name: "is_active", Label: "Active", Type: resource.Bool, Filter: true, List: true},
			},
		},
		{
			Name: "expense-categories", Title: "Expense category", Module: "Procurement", Table: "expense_categories", OrderBy: "t.name",
			ReadRoles: adminFinance, WriteRoles: financeStaff,
			Fields: []f{
				{Name: "name", Label: "Name", Type: resource.Text, Required: true, MaxLen: 120, Search: true, List: true},
				{Name: "description", Label: "Description", Type: resource.LongText, List: true},
			},
		},
		{
			Name: "expenses", Title: "Expense", Module: "Procurement", Table: "expenses",
			CreatorField: "created_by_id", OrderBy: "t.expense_date DESC, t.created_at DESC",
			ReadRoles: financeStaff, WriteRoles: financeStaff,
			Fields: []f{
				{Name: "category_id", Label: "Category", Type: resource.Ref, Ref: &resource.RefSpec{Resource: "expense-categories", Table: "expense_categories", Label: "x.name"}, Required: true, Filter: true, List: true},
				{Name: "supplier_id", Label: "Supplier", Type: resource.Ref, Ref: &resource.RefSpec{Resource: "suppliers", Table: "suppliers", Label: "x.name"}, Filter: true},
				{Name: "expense_date", Label: "Date", Type: resource.Date, List: true},
				{Name: "payee", Label: "Paid to", Type: resource.Text, Required: true, MaxLen: 200, Search: true, List: true},
				{Name: "amount", Label: "Amount (KES)", Type: resource.Decimal, Required: true, Min: resource.Float(0.01), List: true},
				{Name: "payment_method", Label: "Paid by", Type: resource.Enum, Options: payMethods, Filter: true, List: true},
				{Name: "reference", Label: "Reference", Type: resource.Text, MaxLen: 100, Search: true},
				{Name: "description", Label: "Description", Type: resource.LongText},
			},
		},
		{
			Name: "supplier-invoices", Title: "Supplier invoice", Module: "Procurement", Table: "supplier_invoices", OrderBy: "t.invoice_date DESC",
			ReadRoles: financeStaff, WriteRoles: financeStaff,
			Constraints: map[string]string{"supplier_invoice_not_overpaid": "Amount paid can't be more than the invoice amount."},
			Fields: []f{
				{Name: "supplier_id", Label: "Supplier", Type: resource.Ref, Ref: &resource.RefSpec{Resource: "suppliers", Table: "suppliers", Label: "x.name"}, Required: true, Filter: true, List: true},
				{Name: "invoice_no", Label: "Invoice no.", Type: resource.Text, Required: true, MaxLen: 60, Search: true, List: true},
				{Name: "invoice_date", Label: "Invoice date", Type: resource.Date, Required: true, List: true},
				{Name: "due_date", Label: "Due date", Type: resource.Date, List: true},
				{Name: "amount", Label: "Amount (KES)", Type: resource.Decimal, Required: true, Min: resource.Float(0), List: true},
				{Name: "amount_paid", Label: "Paid (KES)", Type: resource.Decimal, Min: resource.Float(0), List: true},
				{Name: "status", Label: "Status", Type: resource.Enum, Options: []string{"unpaid", "partial", "paid"}, Filter: true, List: true, ReadOnly: true,
					Help: "Set automatically from the amount paid."},
				{Name: "description", Label: "Description", Type: resource.LongText},
			},
		},
		{
			Name: "purchase-orders", Title: "Purchase order", Module: "Procurement", Table: "purchase_orders",
			CreatorField: "created_by_id", OrderBy: "t.order_date DESC",
			ReadRoles: adminFinance, WriteRoles: financeStaff,
			Fields: []f{
				{Name: "po_number", Label: "PO number", Type: resource.Text, Required: true, MaxLen: 40, Search: true, List: true},
				{Name: "supplier_id", Label: "Supplier", Type: resource.Ref, Ref: &resource.RefSpec{Resource: "suppliers", Table: "suppliers", Label: "x.name"}, Required: true, Filter: true, List: true},
				{Name: "order_date", Label: "Order date", Type: resource.Date, List: true},
				{Name: "expected_date", Label: "Expected delivery", Type: resource.Date},
				{Name: "items", Label: "Items ordered", Type: resource.LongText, Required: true},
				{Name: "total_amount", Label: "Total (KES)", Type: resource.Decimal, Min: resource.Float(0), List: true},
				{Name: "status", Label: "Status", Type: resource.Enum, Options: []string{"draft", "sent", "received", "cancelled"}, Filter: true, List: true},
				{Name: "notes", Label: "Notes", Type: resource.LongText},
			},
		},
		{
			Name: "requisitions", Title: "Requisition", Module: "Procurement", Table: "requisitions",
			CreatorField: "created_by_id", HasUpdatedAt: true, OrderBy: "t.request_date DESC",
			WriteRoles: adminFinance,
			Fields: []f{
				{Name: "department_id", Label: "Department", Type: resource.Ref, Ref: deptRef, Filter: true, List: true},
				{Name: "requested_by", Label: "Requested by", Type: resource.Text, Required: true, MaxLen: 200, Search: true, List: true},
				{Name: "request_date", Label: "Date", Type: resource.Date, List: true},
				{Name: "items", Label: "Items needed", Type: resource.LongText, Required: true, List: true},
				{Name: "estimated_cost", Label: "Estimated cost (KES)", Type: resource.Decimal, Min: resource.Float(0), List: true},
				{Name: "status", Label: "Status", Type: resource.Enum, Options: []string{"pending", "approved", "rejected", "fulfilled"}, Filter: true, List: true},
				{Name: "approved_by", Label: "Approved by", Type: resource.Text, MaxLen: 200},
				{Name: "notes", Label: "Notes", Type: resource.LongText},
			},
		},
		{
			Name: "petty-cash", Title: "Petty cash entry", Module: "Procurement", Table: "petty_cash_transactions",
			CreatorField: "created_by_id", OrderBy: "t.txn_date DESC, t.created_at DESC",
			ReadRoles: financeStaff, WriteRoles: financeStaff,
			Fields: []f{
				{Name: "txn_date", Label: "Date", Type: resource.Date, List: true},
				{Name: "txn_type", Label: "Type", Type: resource.Enum, Options: []string{"top_up", "expense"}, Required: true, Filter: true, List: true},
				{Name: "amount", Label: "Amount (KES)", Type: resource.Decimal, Required: true, Min: resource.Float(0.01), List: true},
				{Name: "description", Label: "Description", Type: resource.Text, Required: true, MaxLen: 255, Search: true, List: true},
				{Name: "received_by", Label: "Received by", Type: resource.Text, MaxLen: 200, List: true},
			},
		},
		{
			Name: "other-revenue", Title: "Other revenue", Module: "Procurement", Table: "other_revenue",
			CreatorField: "created_by_id", OrderBy: "t.received_date DESC",
			ReadRoles: financeStaff, WriteRoles: financeStaff,
			Fields: []f{
				{Name: "source", Label: "Source", Type: resource.Text, Required: true, MaxLen: 200, Search: true, List: true},
				{Name: "amount", Label: "Amount (KES)", Type: resource.Decimal, Required: true, Min: resource.Float(0.01), List: true},
				{Name: "received_date", Label: "Date received", Type: resource.Date, List: true},
				{Name: "payment_method", Label: "Received via", Type: resource.Enum, Options: payMethods, Filter: true, List: true},
				{Name: "reference", Label: "Reference", Type: resource.Text, MaxLen: 100, Search: true},
				{Name: "description", Label: "Description", Type: resource.LongText},
			},
		},
		{
			Name: "sales-items", Title: "Sales item", Module: "Procurement", Table: "sales_items", OrderBy: "t.name", HasUpdatedAt: true,
			ReadRoles: adminFinance, WriteRoles: financeStaff,
			Fields: []f{
				{Name: "name", Label: "Item", Type: resource.Text, Required: true, MaxLen: 200, Search: true, List: true},
				{Name: "unit_price", Label: "Unit price (KES)", Type: resource.Decimal, Required: true, Min: resource.Float(0), List: true},
				{Name: "stock_qty", Label: "In stock", Type: resource.Int, List: true},
				{Name: "reorder_level", Label: "Reorder level", Type: resource.Int, Min: resource.Float(0), List: true},
				{Name: "description", Label: "Description", Type: resource.LongText},
				{Name: "is_active", Label: "Active", Type: resource.Bool, Filter: true, List: true},
			},
		},
		{
			Name: "sales", Title: "Sale", Module: "Procurement", Table: "sales",
			CreatorField: "created_by_id", OrderBy: "t.sale_date DESC, t.created_at DESC",
			ReadRoles: adminFinance, WriteRoles: financeStaff,
			Constraints: map[string]string{"sales_stock_nonnegative": "Not enough of this item in stock."},
			Fields: []f{
				{Name: "item_id", Label: "Item", Type: resource.Ref, Ref: &resource.RefSpec{Resource: "sales-items", Table: "sales_items", Label: "x.name"}, Required: true, Filter: true, List: true},
				{Name: "quantity", Label: "Quantity", Type: resource.Int, Required: true, Min: resource.Float(1), List: true},
				{Name: "unit_price", Label: "Unit price (KES)", Type: resource.Decimal, Required: true, Min: resource.Float(0), List: true},
				{Name: "sale_date", Label: "Date", Type: resource.Date, List: true},
				{Name: "buyer_name", Label: "Buyer", Type: resource.Text, MaxLen: 200, Search: true, List: true},
				student(false),
				{Name: "payment_method", Label: "Paid by", Type: resource.Enum, Options: []string{"cash", "mpesa", "bank", "invoice"}, Filter: true, List: true},
			},
		},

		// ------------------------------------------------------------ Accounts
		{
			Name: "vote-heads", Title: "Vote head", Module: "Accounts", Table: "vote_heads", OrderBy: "t.priority, t.name",
			ReadRoles: adminFinance, WriteRoles: types.FinanceRoles,
			Fields: []f{
				{Name: "name", Label: "Name", Type: resource.Text, Required: true, MaxLen: 120, Search: true, List: true},
				{Name: "code", Label: "Code", Type: resource.Text, MaxLen: 20, Search: true, List: true},
				{Name: "priority", Label: "Payment priority", Type: resource.Int, Min: resource.Float(1), List: true,
					Help: "Lower numbers are paid first when a payment is split across vote heads."},
				{Name: "description", Label: "Description", Type: resource.LongText},
				{Name: "is_active", Label: "Active", Type: resource.Bool, Filter: true, List: true},
			},
		},
		{
			Name: "fee-pledges", Title: "Fee pledge", Module: "Accounts", Table: "fee_pledges",
			StudentField: "student_id", CreatorField: "created_by_id", HasUpdatedAt: true, OrderBy: "t.due_date",
			ReadRoles: adminFinance, WriteRoles: types.FinanceRoles,
			Fields: []f{
				student(true),
				{Name: "amount", Label: "Amount pledged (KES)", Type: resource.Decimal, Required: true, Min: resource.Float(0.01), List: true},
				{Name: "pledge_date", Label: "Pledge date", Type: resource.Date, List: true},
				{Name: "due_date", Label: "Pay by", Type: resource.Date, Required: true, List: true},
				{Name: "pledged_by", Label: "Pledged by", Type: resource.Text, MaxLen: 200, List: true},
				{Name: "phone", Label: "Phone", Type: resource.Phone},
				{Name: "status", Label: "Status", Type: resource.Enum, Options: []string{"open", "fulfilled", "defaulted"}, Filter: true, List: true},
				{Name: "notes", Label: "Notes", Type: resource.LongText},
			},
		},
		{
			Name: "balances-brought-forward", Title: "Balance brought forward", Module: "Accounts", Table: "balances_brought_forward",
			StudentField: "student_id", CreatorField: "created_by_id", OrderBy: "t.as_of_date DESC",
			ReadRoles: adminFinance, WriteRoles: types.FinanceRoles,
			Fields: []f{
				student(true),
				{Name: "amount", Label: "Amount (KES)", Type: resource.Decimal, Required: true, List: true,
					Help: "Positive = the student owes; negative = the school holds a credit."},
				{Name: "as_of_date", Label: "As of", Type: resource.Date, Required: true, List: true},
				{Name: "term_id", Label: "Carried into term", Type: resource.Ref, Ref: termRef, Filter: true, List: true},
				{Name: "notes", Label: "Notes", Type: resource.LongText},
			},
		},
		{
			Name: "uniforms", Title: "Uniform item", Module: "Accounts", Table: "uniform_items", OrderBy: "t.name", HasUpdatedAt: true,
			WriteRoles: adminFinance,
			Fields: []f{
				{Name: "name", Label: "Item", Type: resource.Text, Required: true, MaxLen: 200, Search: true, List: true},
				{Name: "category", Label: "Category", Type: resource.Text, MaxLen: 50, Filter: true, List: true},
				{Name: "gender", Label: "Gender", Type: resource.Enum, Options: []string{"unisex", "boys", "girls"}, Filter: true, List: true},
				{Name: "size", Label: "Size", Type: resource.Text, MaxLen: 20, List: true},
				{Name: "unit_price", Label: "Unit price (KES)", Type: resource.Decimal, Min: resource.Float(0), List: true},
				{Name: "stock_quantity", Label: "In stock", Type: resource.Int, List: true},
				{Name: "reorder_level", Label: "Reorder level", Type: resource.Int, Min: resource.Float(0)},
				{Name: "is_active", Label: "Active", Type: resource.Bool, Filter: true},
			},
		},

		// ------------------------------------------------------------ Academics
		{
			Name: "exam-threads", Title: "Exam thread", Module: "Academics", Table: "exam_threads",
			WriteRoles:  teachingAdm,
			Constraints: map[string]string{"exam_threads_check": "CATs weight and main exam weight must add up to 100."},
			Fields: []f{
				{Name: "name", Label: "Exam thread", Type: resource.Text, Required: true, MaxLen: 200, Search: true, List: true},
				{Name: "term_id", Label: "Term", Type: resource.Ref, Ref: termRef, Required: true, Filter: true, List: true},
				{Name: "curriculum", Label: "Curriculum", Type: resource.Enum, Options: []string{"cbc", "844_igcse"}, Filter: true, List: true},
				{Name: "cats_weight", Label: "CATs weight (%)", Type: resource.Decimal, Min: resource.Float(0), Max: resource.Float(100), List: true},
				{Name: "main_exam_weight", Label: "Main exam weight (%)", Type: resource.Decimal, Min: resource.Float(0), Max: resource.Float(100), List: true,
					Help: "CATs and main exam weights must add up to 100."},
				{Name: "description", Label: "Description", Type: resource.LongText},
				{Name: "is_locked", Label: "Locked (no more mark changes)", Type: resource.Bool, List: true},
			},
		},
		{
			Name: "exam-comments", Title: "Exam comment", Module: "Academics", Table: "exam_comments", OrderBy: "t.comment_by, t.min_score DESC",
			WriteRoles:  teachingAdm,
			Constraints: map[string]string{"exam_comments_check": "The 'from' score must not be higher than the 'to' score."},
			Fields: []f{
				{Name: "comment_by", Label: "Comment by", Type: resource.Enum, Options: []string{"class_teacher", "head_teacher"}, Filter: true, List: true},
				{Name: "min_score", Label: "From score", Type: resource.Decimal, Required: true, Min: resource.Float(0), Max: resource.Float(100), List: true},
				{Name: "max_score", Label: "To score", Type: resource.Decimal, Required: true, Min: resource.Float(0), Max: resource.Float(100), List: true},
				{Name: "comment", Label: "Comment", Type: resource.LongText, Required: true, List: true},
			},
		},
		{
			Name: "timetable", Title: "Timetable lesson", Module: "Academics", Table: "timetable_slots",
			OrderBy: "t.day_of_week, t.start_time", WriteRoles: teachingAdm,
			Fields: []f{
				{Name: "term_id", Label: "Term", Type: resource.Ref, Ref: termRef, Required: true, Filter: true},
				{Name: "class_id", Label: "Class", Type: resource.Ref, Ref: classRef, Required: true, Filter: true, List: true},
				{Name: "day_of_week", Label: "Day", Type: resource.Int, Required: true, Min: resource.Float(1), Max: resource.Float(7), Filter: true, List: true,
					Help: "1 = Monday … 7 = Sunday"},
				{Name: "period_number", Label: "Period", Type: resource.Int, Required: true, Min: resource.Float(1), List: true},
				{Name: "start_time", Label: "Starts", Type: resource.Time, Required: true, List: true},
				{Name: "end_time", Label: "Ends", Type: resource.Time, Required: true, List: true},
				{Name: "subject_id", Label: "Subject", Type: resource.Ref, Ref: subjectRef, Required: true, Filter: true, List: true},
				{Name: "teacher_id", Label: "Teacher", Type: resource.Ref, Ref: userRef, Filter: true, List: true},
				{Name: "venue", Label: "Venue", Type: resource.Text, MaxLen: 100},
			},
		},

		// ------------------------------------------------------------ Inventory
		{
			Name: "inventory", Title: "Inventory item", Module: "Inventory", Table: "inventory_items", OrderBy: "t.name", HasUpdatedAt: true,
			WriteRoles: adminFinance,
			Fields: []f{
				{Name: "name", Label: "Item", Type: resource.Text, Required: true, MaxLen: 200, Search: true, List: true},
				{Name: "sku", Label: "SKU / code", Type: resource.Text, Required: true, MaxLen: 50, Search: true, List: true},
				{Name: "category", Label: "Category", Type: resource.Enum, Options: []string{"stationery", "uniform", "books", "cleaning", "kitchen", "sports", "furniture", "electronics", "medical", "other"}, Filter: true, List: true},
				{Name: "unit_of_measure", Label: "Unit", Type: resource.Text, MaxLen: 20, List: true},
				{Name: "unit_price", Label: "Unit price (KES)", Type: resource.Decimal, Min: resource.Float(0)},
				{Name: "quantity_in_stock", Label: "In stock", Type: resource.Int, List: true,
					Help: "Updated automatically by stock movements; edit only to correct a count."},
				{Name: "reorder_level", Label: "Reorder level", Type: resource.Int, Min: resource.Float(0), List: true},
				{Name: "location", Label: "Store location", Type: resource.Text, MaxLen: 100, List: true},
				{Name: "supplier", Label: "Usual supplier", Type: resource.Text, MaxLen: 200},
				{Name: "is_active", Label: "Active", Type: resource.Bool, Filter: true},
			},
		},
		{
			Name: "inventory-transactions", Title: "Stock movement", Module: "Inventory", Table: "inventory_transactions",
			CreatorField: "processed_by_id", OrderBy: "t.transaction_date DESC, t.created_at DESC", WriteRoles: adminFinance,
			Constraints: map[string]string{"inventory_stock_nonnegative": "Not enough stock for this movement.", "inventory_txn_quantity_sign": "Quantity must be more than zero (only adjustments may be negative)."},
			Fields: []f{
				{Name: "item_id", Label: "Item", Type: resource.Ref, Ref: &resource.RefSpec{Resource: "inventory", Table: "inventory_items", Label: "x.name"}, Required: true, Filter: true, List: true},
				{Name: "transaction_type", Label: "Movement", Type: resource.Enum, Options: []string{"purchase", "issue", "return", "adjustment", "damage", "loss", "transfer"}, Required: true, Filter: true, List: true},
				{Name: "quantity", Label: "Quantity", Type: resource.Int, Required: true, List: true},
				{Name: "unit_price", Label: "Unit price (KES)", Type: resource.Decimal, Min: resource.Float(0)},
				{Name: "total_amount", Label: "Total (KES)", Type: resource.Decimal, Min: resource.Float(0), List: true},
				{Name: "recipient", Label: "Issued to", Type: resource.Text, MaxLen: 200, Search: true, List: true},
				{Name: "supplier", Label: "Supplier", Type: resource.Text, MaxLen: 200},
				{Name: "reference_no", Label: "Reference", Type: resource.Text, MaxLen: 100, Search: true},
				{Name: "transaction_date", Label: "Date", Type: resource.Date, Required: true, List: true},
				{Name: "notes", Label: "Notes", Type: resource.LongText},
			},
		},

		// ------------------------------------------------------------ People
		{
			Name: "board-members", Title: "Board member", Module: "People", Table: "board_members", OrderBy: "t.full_name",
			WriteRoles: leadership,
			Fields: []f{
				{Name: "full_name", Label: "Full name", Type: resource.Text, Required: true, MaxLen: 200, Search: true, List: true},
				{Name: "position", Label: "Position", Type: resource.Text, Required: true, MaxLen: 120, List: true},
				{Name: "phone", Label: "Phone", Type: resource.Phone, List: true},
				{Name: "email", Label: "Email", Type: resource.Email},
				{Name: "term_start", Label: "Term starts", Type: resource.Date},
				{Name: "term_end", Label: "Term ends", Type: resource.Date, List: true},
				{Name: "notes", Label: "Notes", Type: resource.LongText},
			},
		},
		{
			Name: "staff-appraisals", Title: "Staff appraisal", Module: "People", Table: "staff_appraisals", OrderBy: "t.appraisal_date DESC",
			ReadRoles: leadership, WriteRoles: leadership,
			Fields: []f{
				{Name: "staff_id", Label: "Staff member", Type: resource.Ref, Ref: staffRef, Required: true, Filter: true, List: true},
				{Name: "appraisal_date", Label: "Date", Type: resource.Date, Required: true, List: true},
				{Name: "period", Label: "Period covered", Type: resource.Text, MaxLen: 60, List: true},
				{Name: "score", Label: "Score (%)", Type: resource.Decimal, Min: resource.Float(0), Max: resource.Float(100), List: true},
				{Name: "rating", Label: "Rating", Type: resource.Enum, Options: []string{"outstanding", "exceeds", "meets", "below", "poor"}, Filter: true, List: true},
				{Name: "appraiser", Label: "Appraiser", Type: resource.Text, MaxLen: 200},
				{Name: "strengths", Label: "Strengths", Type: resource.LongText},
				{Name: "improvements", Label: "Areas to improve", Type: resource.LongText},
				{Name: "comments", Label: "Comments", Type: resource.LongText},
			},
		},
		{
			Name: "clearances", Title: "Clearance", Module: "People", Table: "clearances", HasUpdatedAt: true,
			StudentField: "student_id",
			Constraints:  map[string]string{"clearances_check": "Choose a student when clearing a student, or a staff member when clearing staff — not both."},
			Fields: []f{
				{Name: "subject_type", Label: "Clearing a", Type: resource.Enum, Options: []string{"student", "staff"}, Required: true, Filter: true, List: true},
				student(false),
				{Name: "staff_id", Label: "Staff member", Type: resource.Ref, Ref: staffRef, Filter: true, List: true},
				{Name: "department", Label: "Department", Type: resource.Enum, Options: []string{"library", "finance", "boarding", "sports", "laboratory", "stores", "other"}, Required: true, Filter: true, List: true},
				{Name: "status", Label: "Status", Type: resource.Enum, Options: []string{"pending", "cleared", "blocked"}, Filter: true, List: true},
				{Name: "cleared_by", Label: "Cleared by", Type: resource.Text, MaxLen: 200},
				{Name: "cleared_on", Label: "Cleared on", Type: resource.Date, List: true},
				{Name: "notes", Label: "Notes", Type: resource.LongText},
			},
		},

		// ------------------------------------------------------------ Communication
		{
			Name: "message-templates", Title: "Message template", Module: "Communication", Table: "message_templates",
			CreatorField: "created_by_id", HasUpdatedAt: true, OrderBy: "t.name",
			Fields: []f{
				{Name: "template_key", Label: "Key", Type: resource.Text, Required: true, MaxLen: 60, Search: true, List: true},
				{Name: "name", Label: "Name", Type: resource.Text, Required: true, MaxLen: 200, Search: true, List: true},
				{Name: "channel", Label: "Channel", Type: resource.Enum, Options: []string{"sms", "email", "in_app", "push"}, Filter: true, List: true},
				{Name: "subject", Label: "Subject (email)", Type: resource.Text, MaxLen: 200},
				{Name: "body_template", Label: "Message", Type: resource.LongText, Required: true,
					Help: "Placeholders like {student_first} and {balance} are filled in when sending."},
				{Name: "description", Label: "Description", Type: resource.LongText},
				{Name: "is_active", Label: "Active", Type: resource.Bool, Filter: true, List: true},
			},
		},
		{
			Name: "contact-groups", Title: "Contact group", Module: "Communication", Table: "contact_groups",
			CreatorField: "created_by_id", OrderBy: "t.name",
			Fields: []f{
				{Name: "name", Label: "Name", Type: resource.Text, Required: true, MaxLen: 200, Search: true, List: true},
				{Name: "group_type", Label: "Type", Type: resource.Text, MaxLen: 30, Filter: true, List: true},
				{Name: "description", Label: "Description", Type: resource.LongText, List: true},
			},
		},
	}
}

// pocketMoneyBalances: GET /pocket-money-balances/?student_id= — running
// balances from the pocket_money_balances view. Parents see their own
// children only.
func pocketMoneyBalances(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rl, _ := r.Context().Value("role").(types.Role)
		uid, _ := r.Context().Value("user_id").(uuid.UUID)
		if !hasRole(rl, adminFinance) && !hasRole(rl, types.TeacherRoles) && rl != types.RoleParent && rl != types.RoleStudent {
			httpx.ErrForbidden(w, "not allowed")
			return
		}
		scope, err := resolveStudentScope(r.Context(), pool, rl, uid)
		if hasRole(rl, types.TeacherRoles) || hasRole(rl, adminFinance) {
			scope, err = nil, nil
		}
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		var filter *uuid.UUID
		if v := r.URL.Query().Get("student_id"); v != "" {
			id, err := uuid.Parse(v)
			if err != nil {
				httpx.ErrBadRequest(w, "invalid student_id")
				return
			}
			filter = &id
		}
		rows, err := pool.Query(r.Context(), `
			SELECT b.student_id, st.full_name, st.admission_no, b.balance::float8, b.last_txn_date::text
			FROM pocket_money_balances b JOIN students st ON st.id = b.student_id
			WHERE ($1::uuid IS NULL OR b.student_id = $1)
			  AND ($2::uuid[] IS NULL OR b.student_id = ANY($2))
			ORDER BY st.full_name`, filter, scope)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()
		type bal struct {
			StudentID   uuid.UUID `json:"student_id"`
			FullName    string    `json:"full_name"`
			AdmissionNo string    `json:"admission_no"`
			Balance     float64   `json:"balance"`
			LastTxnDate string    `json:"last_txn_date"`
		}
		out := []bal{}
		for rows.Next() {
			var b bal
			if rows.Scan(&b.StudentID, &b.FullName, &b.AdmissionNo, &b.Balance, &b.LastTxnDate) == nil {
				out = append(out, b)
			}
		}
		httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
	}
}

// NewResourceRegistry builds the engine with every school resource and the
// parent/student scoping shared with documents and finance.
func NewResourceRegistry(pool *pgxpool.Pool) *resource.Registry {
	g := resource.NewRegistry(pool, func(r *http.Request) ([]uuid.UUID, bool, error) {
		rl, _ := r.Context().Value("role").(types.Role)
		uid, _ := r.Context().Value("user_id").(uuid.UUID)
		ids, err := resolveStudentScope(r.Context(), pool, rl, uid)
		if err != nil {
			return nil, true, err
		}
		return ids, ids != nil, nil
	})
	g.Add(schoolResources()...)
	return g
}
