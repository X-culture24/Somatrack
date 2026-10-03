# Reference screens → API coverage

Every menu item visible in the SmartShule reference screenshots (`mock-up/`, 24 images),
mapped to what serves it in this codebase.

- **✅ R**: declarative resource (`internal/api/resources.go`). It has a form and table at `/records/<name>`, and its API is at `/api/<name>/`.
- **✅ S**: hand-written service (existing).
- **🔶**: partial. The records exist, but some behaviour is missing (noted).
- **❌**: not built. It needs custom logic, not just a form.

The resource engine (`internal/resource`) gives every **R** item list, search, filters, paging,
create, edit, delete, validation, role permissions and parent scoping. It also serves a form
schema at `/api/meta/resources/<name>/`.

## Students management

| Screen item | Status | Served by |
|---|---|---|
| New admission (3-step wizard) | ✅ S | `/admissions/` (fields from migrations 011, 014) |
| All students | ✅ S | `/students/` |
| Admission enquiries | ✅ R | `admission-enquiries` |
| Move students to next class | ❌ | Needs a bulk promotion endpoint (class → next class, per year) |
| Siblings | 🔶 | Data exists (`student_guardians`); needs a "grouped by guardian" view |
| Alumni students | 🔶 | `students.status`; needs an alumni list/filter |
| Pocket money | ✅ R | `pocket-money` + `/pocket-money-balances/` (overdrawing is blocked) |
| Students discipline | ✅ R | `discipline-cases` (parents see their own child's) |
| Medical records | ✅ R | `medical-records` (was a placeholder stub) |
| Extra-curricular activities | ✅ S | `/clubs/` |
| Hobbies & favourites | ✅ R | `hobbies` |
| Birthday reminders | ❌ | Computed list from `students.date_of_birth` + optional SMS |
| Leaving certificate | 🔶 R | `leaving-certificates`. Records only; a PDF with a signed QR code is still to do |
| Leadership position | ✅ R | `leadership-positions` |
| Withdrawals | ✅ R | `student-withdrawals` (was a placeholder stub) |
| Hostels / dormitories, rooms, beds | ✅ R | `hostels`, `hostel-rooms`, `hostel-beds` |
| Bed allocation | ✅ R | `bed-allocations`: one active bed per student; bed status follows allocations; beds under maintenance can't be allocated |
| Class rooms (physical) | ❌ | Not modelled |

## Accounts

| Screen item | Status | Served by |
|---|---|---|
| Receive payment | ✅ S | M-Pesa C2B, Jenga/Equity IPN and STK push post automatically; cash/cheque/bank/card at the bursar via `/finance/manual-payment/` (no double-posting). **Every payment texts the family** the amount, new balance and a receipt link (one SMS, sent once) |
| Student ledger | ❌ | Invoices + payments + balance b/f in one statement |
| Priority config / fee vote heads | 🔶 R | `vote-heads` (with priority). Payment allocation doesn't use priority yet |
| Fee statements / payment receipts | ✅ S | `/invoices/{id}/pdf/`, `/payments/{id}/receipt/` (signed QR). Parents download every receipt from Fees; the SMS link opens straight to that receipt |
| Pending voided | ❌ | Void-request approval workflow |
| SMS fee balance | ✅ S | `/finance/send-reminders/` (Africa's Talking) |
| Print fee balance | ❌ | Class-level balance PDF |
| Fee waivers / discounts | ✅ S | `/discounts/` |
| Pending waivers (approval) | ❌ | Approval workflow on `discount_waivers` |
| Fee pledges | ✅ R | `fee-pledges` |
| Balance brought forward | 🔶 R | `balances-brought-forward`. Records only; not yet added to invoices |
| Manage fee structure | ✅ S | `/fee-structures/` |
| Generate / invoice tuition | ✅ S | `/billing/generate/` |
| Bulk manage invoices, student journals, proforma invoice | ❌ | Custom finance logic |
| Bank accounts | ❌ | Not modelled (Equity account is config only) |
| Payroll | ✅ S | `/payroll-runs/` |
| Uniforms | ✅ R | `uniforms` (was a placeholder stub) |

## Transport

| Screen item | Status | Served by |
|---|---|---|
| Routes | ✅ S | `/transport-routes/`, `/vehicles/` |
| Add / manage students | 🔶 S | `students.transport_route_id`; no zone filter yet |
| Invoice students (transport) | ❌ | Bill route fees per term |
| Transport report | ❌ | |

## Procurement

| Screen item | Status | Served by |
|---|---|---|
| Supplier invoices | ✅ R | `supplier-invoices`: status is set from the amount paid; overpayment is blocked |
| Other revenue | ✅ R | `other-revenue` |
| Purchase orders | ✅ R | `purchase-orders` (items as text; no line items yet) |
| Expenses, expenses category | ✅ R | `expenses`, `expense-categories` (finance roles only) |
| Requisitions | ✅ R | `requisitions` |
| Petty cash | ✅ R | `petty-cash` |
| Sales items, record sales | ✅ R | `sales-items`, `sales`: a sale reduces stock, and overselling is blocked |
| Sales items stock trend | ❌ | Report |

## Academics

| Screen item | Status | Served by |
|---|---|---|
| All classes, subjects | ✅ S | `/classes/`, `/subjects/` |
| Assign educators subjects | ✅ S | `/class-subjects/` |
| Subject categories / CBC subjects | 🔶 S | `subjects.learning_area` (migration 015) |
| Class timetable | ✅ R | `timetable` (was a placeholder stub; the Timetable page is fixed) |
| Students roll call | ✅ S | `/attendance/…`, fingerprint |
| Assignments | ✅ S | Nova LMS |
| Exams (exam threads, CATs/main weights) | ✅ R | `exam-threads` (weights must total 100; lockable) |
| Set exam comments | ✅ R | `exam-comments` |
| Grading system | ✅ S | `/grade-requirements/` |
| Marks entry / assessments | ✅ S | `/assessments/`, `/assessments/{id}/marksheet/`, bulk `PUT /assessments/{id}/marks/` (teacher-scoped, respects locks, graded with school bands else CBC EE/ME/AE/BE) |
| **Results trail** (client priority 3) | ✅ S | `/results/students/{id}/history/` (every term, labelled with the class sat in; CATs/main blended by exam-thread weights) + `/transcript/` PDF with a signed QR code. Results can no longer be deleted with a term, class, subject or student |
| Exams report, exam subjects report, CBC formative/summative/yearly | ❌ | Marks now exist; report layouts still to build (per-term report card = one term of the trail) |
| National / other certificates | ✅ R | `certificates` |

## E-Classroom & library

| Screen item | Status | Served by |
|---|---|---|
| E-learning, lesson plans | ✅ S | Nova LMS, `/lesson-plans/` |
| E-past papers, e-videos, e-notes, e-books, schemes of work | ❌ | Need file uploads per level/subject |
| Books, borrow/return, due/overdue | ✅ S | `/library/books/`, `/library/loans/` |
| Lost books, book fund | ❌ | |

## Inventory

| Screen item | Status | Served by |
|---|---|---|
| Manage items, category, units | ✅ R | `inventory` (was a placeholder stub; the Inventory page is fixed) |
| Add stock / giving out items / stock log | ✅ R | `inventory-transactions`: movements update stock, and negative stock is blocked |

## People management

| Screen item | Status | Served by |
|---|---|---|
| Employees attendance | ✅ S | `/attendance/staff/…` |
| Teaching / non-teaching / supporting staff | ✅ S | `/staff/` |
| Teachers appraisal (+ results) | ✅ R | `staff-appraisals` (leadership only) |
| Board members | ✅ R | `board-members` |
| Parents, students | ✅ S | `/guardians/`, `/students/` |
| Emergency contacts | ✅ R | `emergency-contacts` |
| Students / staff clearance | ✅ R | `clearances` |
| Add staff, list users | ✅ S | `/users/`, `/staff/` |
| Staff signatures, user groups / permissions | ❌ | |

## Communication

| Screen item | Status | Served by |
|---|---|---|
| Message templates, contact groups | ✅ R | `message-templates`, `contact-groups` |
| Send bulk SMS to a group / class / arrears list | ❌ | Uses the Africa's Talking sender already built |

## Reports

Existing: attendance, grades, fee collection, class performance (`/reports/…`).
Still missing: books of accounts (cash book, general ledger, trial balance, P&L, balance
sheet), plus report screens that depend on marks.

## Mistakes in the reference screens not to repeat

- A cashflow pie that's meaningless when expenses are 0.
- `false` shown as a date.
- Blank or duplicate sibling rows.
- Duplicate subjects.
- Stray dots in class names ("Grade 4 .").
- A content pane that doesn't follow the menu.
