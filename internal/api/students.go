package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/rpc"
	"github.com/stmaryskabete/lms/internal/shared/types"
)

// StudentsService implements the real student registration/lifecycle CRUD
// that registerPlaceholderCRUD used to stub out. The previous React form
// (and its API client) used field names and an ID scheme (integers) from a
// different, never-deployed backend — this defines the actual contract
// against the live Postgres schema, including every field the students
// table has (medical, admission, boarding, KCPE, etc.), not just the
// handful the old form exposed. It also lets a registration call link
// guardians (existing or brand new) in the same request, since "register a
// student" and "who is their guardian" are one workflow in practice.
type StudentsService struct {
	pool    *pgxpool.Pool
	finance *rpc.FinanceClient
}

func NewStudentsService(pool *pgxpool.Pool, finance *rpc.FinanceClient) *StudentsService {
	return &StudentsService{pool: pool, finance: finance}
}

// billIfEnrolled fires the new-student auto-bill (BillStudentForCurrentTerm)
// once a student has a class assigned — a student with no class yet can't be
// billed (no grade to look up a fee structure by), so this is a no-op until
// then. Failure is deliberately non-fatal: finance not having set up a fee
// structure for this grade yet shouldn't block registering the student.
func (s *StudentsService) billIfEnrolled(ctx context.Context, studentID uuid.UUID, hasClass bool) {
	if !hasClass || s.finance == nil {
		return
	}
	if _, err := s.finance.BillStudent(ctx, studentID); err != nil {
		log.Printf("[students] auto-bill for %s: %v", studentID, err)
	}
}

func (s *StudentsService) RegisterRoutes(r chi.Router) {
	r.Get("/student-ledger/", s.ledger)
	r.Route("/students/", func(r chi.Router) {
		r.Get("/", s.list)
		r.Post("/", s.create)
		r.Post("/promote/", adminOnly()(http.HandlerFunc(s.promote)).ServeHTTP)
		r.Get("/fee-arrears/", s.feeArrears)
		r.Get("/{id}/", s.get)
		r.Put("/{id}/", s.update)
		r.Patch("/{id}/", s.update)
		r.Delete("/{id}/", adminOnly()(http.HandlerFunc(s.delete)).ServeHTTP)
		r.Post("/{id}/guardians/", s.linkGuardian)
		r.Delete("/{id}/guardians/{guardian_id}/", s.unlinkGuardian)
		r.Post("/{id}/clubs/", s.linkClub)
		r.Delete("/{id}/clubs/{club_id}/", s.unlinkClub)
	})
}

// Student is the full registration/update payload — every column on the
// students table the API previously left inaccessible.
type Student struct {
	ID                 uuid.UUID    `json:"id"`
	AdmissionNo        string       `json:"admission_no" validate:"required"`
	FirstName          string       `json:"first_name" validate:"required"`
	MiddleName         string       `json:"middle_name"`
	LastName           string       `json:"last_name" validate:"required"`
	FullName           string       `json:"full_name,omitempty"`
	Gender             string       `json:"gender" validate:"required,oneof=male female other"`
	DateOfBirth        *string      `json:"date_of_birth,omitempty"`
	NationalID         string       `json:"national_id"`
	BirthCertificateNo string       `json:"birth_certificate_no"`
	Phone              string       `json:"phone"`
	Email              string       `json:"email"`
	ClassID            OptionalUUID `json:"class_id,omitempty"`
	CurrentClassID     OptionalUUID `json:"current_class_id,omitempty"`
	StreamID           OptionalUUID `json:"stream_id,omitempty"`
	AcademicYearID     OptionalUUID `json:"academic_year_id,omitempty"`
	AdmissionDate      *string      `json:"admission_date,omitempty"`
	BoardingStatus     string       `json:"boarding_status" validate:"omitempty,oneof=day boarding"`
	BloodGroup         string       `json:"blood_group"`
	Allergies          string       `json:"allergies"`
	ChronicConditions  string       `json:"chronic_conditions"`
	ResidentialAddress string       `json:"residential_address"`
	FormerSchool       string       `json:"former_school"`
	KcpeMarks          *int         `json:"kcpe_marks,omitempty"`
	KcpeGrade          string       `json:"kcpe_grade"`
	UPI                string       `json:"upi"`
	Status             string       `json:"status" validate:"omitempty,oneof=active inactive transferred withdrawn suspended graduated admitted"`
	IsActive           bool         `json:"is_active"`

	// Fields from the paper admission form added after the initial pass.
	KemisNumber           string       `json:"kemis_number"`
	AssessmentNumber      string       `json:"assessment_number"`
	UsesTransport         bool         `json:"uses_transport"`
	TransportRouteID      OptionalUUID `json:"transport_route_id,omitempty"`
	PhotoPath             string       `json:"photo_path"`
	BirthCertDocumentPath string       `json:"birth_cert_document_path"`

	StudentWelfare

	// Optional: link guardians / co-curricular clubs in the same call as registration.
	Guardians []GuardianLink `json:"guardians,omitempty"`
	ClubIDs   []uuid.UUID    `json:"club_ids,omitempty"`
}

// StudentWelfare holds the rest of the school's admission form: consent,
// family and welfare status, medical contacts, sponsorship and where the
// learner is from. Kept apart from the core columns so create/update write
// it with one statement of its own.
type StudentWelfare struct {
	PhotoConsent          bool   `json:"photo_consent"`
	ParentalStatus        string `json:"parental_status" validate:"omitempty,oneof=both_alive father_deceased mother_deceased both_deceased"`
	HasSpecialNeeds       bool   `json:"has_special_needs"`
	SpecialNeeds          string `json:"special_needs"`
	Religion              string `json:"religion"`
	DoctorName            string `json:"doctor_name"`
	DoctorPhone           string `json:"doctor_phone"`
	PreferredHospital     string `json:"preferred_hospital"`
	HasScholarship        bool   `json:"has_scholarship"`
	ScholarshipType       string `json:"scholarship_type" validate:"omitempty,oneof=full partial bursary sponsorship other"`
	SponsorName           string `json:"sponsor_name"`
	SponsorPhone          string `json:"sponsor_phone"`
	SponsorLocation       string `json:"sponsor_location"`
	SponsorContactPerson  string `json:"sponsor_contact_person"`
	SponsorContactDetails string `json:"sponsor_contact_details"`
	Citizenship           string `json:"citizenship"`
	HomeCounty            string `json:"home_county"`
	ResidenceCounty       string `json:"residence_county"`
	SubCounty             string `json:"sub_county"`
}

const welfareCols = `photo_consent, parental_status, has_special_needs, special_needs, religion, doctor_name,
	doctor_phone, preferred_hospital, has_scholarship, scholarship_type, sponsor_name, sponsor_phone,
	sponsor_location, sponsor_contact_person, sponsor_contact_details, citizenship, home_county,
	residence_county, sub_county`

func (w *StudentWelfare) scanDest() []interface{} {
	return []interface{}{&w.PhotoConsent, &w.ParentalStatus, &w.HasSpecialNeeds, &w.SpecialNeeds, &w.Religion,
		&w.DoctorName, &w.DoctorPhone, &w.PreferredHospital, &w.HasScholarship, &w.ScholarshipType,
		&w.SponsorName, &w.SponsorPhone, &w.SponsorLocation, &w.SponsorContactPerson, &w.SponsorContactDetails,
		&w.Citizenship, &w.HomeCounty, &w.ResidenceCounty, &w.SubCounty}
}

// studentWelfareUpdate is the PATCH mirror: nil leaves a column unchanged.
type studentWelfareUpdate struct {
	PhotoConsent          *bool   `json:"photo_consent,omitempty"`
	ParentalStatus        *string `json:"parental_status,omitempty"`
	HasSpecialNeeds       *bool   `json:"has_special_needs,omitempty"`
	SpecialNeeds          *string `json:"special_needs,omitempty"`
	Religion              *string `json:"religion,omitempty"`
	DoctorName            *string `json:"doctor_name,omitempty"`
	DoctorPhone           *string `json:"doctor_phone,omitempty"`
	PreferredHospital     *string `json:"preferred_hospital,omitempty"`
	HasScholarship        *bool   `json:"has_scholarship,omitempty"`
	ScholarshipType       *string `json:"scholarship_type,omitempty"`
	SponsorName           *string `json:"sponsor_name,omitempty"`
	SponsorPhone          *string `json:"sponsor_phone,omitempty"`
	SponsorLocation       *string `json:"sponsor_location,omitempty"`
	SponsorContactPerson  *string `json:"sponsor_contact_person,omitempty"`
	SponsorContactDetails *string `json:"sponsor_contact_details,omitempty"`
	Citizenship           *string `json:"citizenship,omitempty"`
	HomeCounty            *string `json:"home_county,omitempty"`
	ResidenceCounty       *string `json:"residence_county,omitempty"`
	SubCounty             *string `json:"sub_county,omitempty"`
}

func (u studentWelfareUpdate) args() []interface{} {
	return []interface{}{u.PhotoConsent, u.ParentalStatus, u.HasSpecialNeeds, u.SpecialNeeds, u.Religion,
		u.DoctorName, u.DoctorPhone, u.PreferredHospital, u.HasScholarship, u.ScholarshipType,
		u.SponsorName, u.SponsorPhone, u.SponsorLocation, u.SponsorContactPerson, u.SponsorContactDetails,
		u.Citizenship, u.HomeCounty, u.ResidenceCounty, u.SubCounty}
}

func (w StudentWelfare) update() studentWelfareUpdate {
	return studentWelfareUpdate{&w.PhotoConsent, &w.ParentalStatus, &w.HasSpecialNeeds, &w.SpecialNeeds, &w.Religion,
		&w.DoctorName, &w.DoctorPhone, &w.PreferredHospital, &w.HasScholarship, &w.ScholarshipType,
		&w.SponsorName, &w.SponsorPhone, &w.SponsorLocation, &w.SponsorContactPerson, &w.SponsorContactDetails,
		&w.Citizenship, &w.HomeCounty, &w.ResidenceCounty, &w.SubCounty}
}

// welfareUpdateSQL sets every welfare column from $2..$20, keeping the
// current value where the argument is NULL.
var welfareUpdateSQL = func() string {
	cols := strings.Split(welfareCols, ",")
	sets := make([]string, len(cols))
	for i, c := range cols {
		c = strings.TrimSpace(c)
		sets[i] = fmt.Sprintf("%s = COALESCE($%d, %s)", c, i+2, c)
	}
	return "UPDATE students SET " + strings.Join(sets, ", ") + " WHERE id = $1"
}()

func validWelfareUpdate(u studentWelfareUpdate) map[string]string {
	errs := map[string]string{}
	if u.ParentalStatus != nil && !containsStr([]string{"", "both_alive", "father_deceased", "mother_deceased", "both_deceased"}, *u.ParentalStatus) {
		errs["parental_status"] = "must be both_alive, father_deceased, mother_deceased or both_deceased"
	}
	if u.ScholarshipType != nil && !containsStr([]string{"", "full", "partial", "bursary", "sponsorship", "other"}, *u.ScholarshipType) {
		errs["scholarship_type"] = "must be full, partial, bursary, sponsorship or other"
	}
	return errs
}

type GuardianLink struct {
	GuardianID OptionalUUID `json:"guardian_id,omitempty"` // link an existing guardian by id...

	FirstName          string `json:"first_name,omitempty"` // ...or create one inline with these fields
	LastName           string `json:"last_name,omitempty"`
	Email              string `json:"email,omitempty"`
	Phone              string `json:"phone,omitempty"`
	AlternatePhone     string `json:"alternate_phone,omitempty"`
	NationalID         string `json:"national_id,omitempty"`
	Occupation         string `json:"occupation,omitempty"`
	Employer           string `json:"employer,omitempty"`
	ResidentialAddress string `json:"residential_address,omitempty"`
	PostalAddress      string `json:"postal_address,omitempty"`
	IDDocumentPath     string `json:"id_document_path,omitempty"`

	Relation        string `json:"relation"`
	IsPrimary       bool   `json:"is_primary"`
	HasCustody      bool   `json:"has_custody"`
	ContactPriority int    `json:"contact_priority"`
}

const studentCols = `id, admission_no, first_name, middle_name, last_name, full_name, gender,
	date_of_birth::text, national_id, birth_certificate_no, phone, email, class_id, current_class_id,
	stream_id, academic_year_id, admission_date::text, boarding_status, blood_group, allergies,
	chronic_conditions, residential_address, former_school, kcpe_marks, kcpe_grade, upi, status, is_active,
	kemis_number, assessment_number, uses_transport, transport_route_id, photo_path, birth_cert_document_path, ` + welfareCols

func scanStudent(row interface {
	Scan(dest ...interface{}) error
}) (Student, error) {
	var st Student
	var classID, currentClassID, streamID, academicYearID, transportRouteID *uuid.UUID
	dest := []interface{}{&st.ID, &st.AdmissionNo, &st.FirstName, &st.MiddleName, &st.LastName, &st.FullName, &st.Gender,
		&st.DateOfBirth, &st.NationalID, &st.BirthCertificateNo, &st.Phone, &st.Email, &classID, &currentClassID,
		&streamID, &academicYearID, &st.AdmissionDate, &st.BoardingStatus, &st.BloodGroup, &st.Allergies,
		&st.ChronicConditions, &st.ResidentialAddress, &st.FormerSchool, &st.KcpeMarks, &st.KcpeGrade, &st.UPI,
		&st.Status, &st.IsActive, &st.KemisNumber, &st.AssessmentNumber, &st.UsesTransport, &transportRouteID,
		&st.PhotoPath, &st.BirthCertDocumentPath}
	err := row.Scan(append(dest, st.StudentWelfare.scanDest()...)...)
	st.ClassID = OptionalUUIDFromPtr(classID)
	st.CurrentClassID = OptionalUUIDFromPtr(currentClassID)
	st.StreamID = OptionalUUIDFromPtr(streamID)
	st.AcademicYearID = OptionalUUIDFromPtr(academicYearID)
	st.TransportRouteID = OptionalUUIDFromPtr(transportRouteID)
	return st, err
}

func (s *StudentsService) list(w http.ResponseWriter, r *http.Request) {
	where := []string{"true"}
	args := []interface{}{}
	argIdx := 1

	// A parent has no legitimate reason to browse the full student directory —
	// this endpoint had no such restriction at all before, and the mobile app's
	// "my wards" screen (?my_wards=true) relied on a filter that never existed,
	// so it silently returned every student in the school. Reuses the same
	// guardian-scoping resolveStudentScope already enforces for finance data.
	role, _ := r.Context().Value("role").(types.Role)
	userID, _ := r.Context().Value("user_id").(uuid.UUID)
	if role == types.RoleParent {
		scope, err := resolveStudentScope(r.Context(), s.pool, role, userID)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		if len(scope) == 0 {
			httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": []interface{}{}, "count": 0})
			return
		}
		where = append(where, "id = ANY($"+strconv.Itoa(argIdx)+")")
		args = append(args, scope)
		argIdx++
	}

	// alumni=true shows inactive/graduated students; default shows only active.
	if r.URL.Query().Get("alumni") == "true" {
		where = append(where, "is_active = false")
	} else if role != types.RoleParent {
		where = append(where, "is_active = true")
	}

	if v := r.URL.Query().Get("status"); v != "" {
		where = append(where, "status = $"+strconv.Itoa(argIdx))
		args = append(args, v)
		argIdx++
	}
	if v := r.URL.Query().Get("class_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			where = append(where, "current_class_id = $"+strconv.Itoa(argIdx))
			args = append(args, id)
			argIdx++
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("search")); v != "" {
		where = append(where, "(full_name ILIKE $"+strconv.Itoa(argIdx)+" OR admission_no ILIKE $"+strconv.Itoa(argIdx)+")")
		args = append(args, "%"+v+"%")
		argIdx++
	}
	page, pageSize, offset := pageParams(r)

	var total int
	_ = s.pool.QueryRow(r.Context(), `SELECT COUNT(*) FROM students WHERE `+joinAnd(where), args...).Scan(&total)

	q := `SELECT ` + studentCols + ` FROM students WHERE ` + joinAnd(where) +
		` ORDER BY last_name, first_name LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)
	args = append(args, pageSize, offset)
	rows, err := s.pool.Query(r.Context(), q, args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	var students []Student
	for rows.Next() {
		st, err := scanStudent(rows)
		if err != nil {
			continue
		}
		students = append(students, st)
	}

	ids := make([]uuid.UUID, len(students))
	for i, st := range students {
		ids[i] = st.ID
	}
	balances := map[uuid.UUID]float64{}
	if len(ids) > 0 {
		if brows, err := s.pool.Query(r.Context(), `
			SELECT student_id, COALESCE(SUM(balance), 0) FROM invoices
			WHERE student_id = ANY($1) GROUP BY student_id`, ids); err == nil {
			defer brows.Close()
			for brows.Next() {
				var sid uuid.UUID
				var bal float64
				if brows.Scan(&sid, &bal) == nil {
					balances[sid] = bal
				}
			}
		}
	}

	classNames := map[uuid.UUID]string{}
	classIDs := make([]uuid.UUID, 0, len(students))
	for _, st := range students {
		if st.CurrentClassID.Valid {
			classIDs = append(classIDs, st.CurrentClassID.UUID)
		}
	}
	if len(classIDs) > 0 {
		if crows, err := s.pool.Query(r.Context(), `SELECT id, name FROM class_groups WHERE id = ANY($1)`, classIDs); err == nil {
			defer crows.Close()
			for crows.Next() {
				var cid uuid.UUID
				var name string
				if crows.Scan(&cid, &name) == nil {
					classNames[cid] = name
				}
			}
		}
	}

	// Primary guardian name/phone for each student — shown in the All Students list.
	type parentInfo struct{ Name, Phone string }
	parents := map[uuid.UUID]parentInfo{}
	if len(ids) > 0 {
		if prows, err := s.pool.Query(r.Context(), `
			SELECT sg.student_id, g.first_name || ' ' || g.last_name, COALESCE(g.phone,'')
			FROM student_guardians sg JOIN guardians g ON g.id = sg.guardian_id
			WHERE sg.student_id = ANY($1) AND sg.is_primary = TRUE`, ids); err == nil {
			defer prows.Close()
			for prows.Next() {
				var sid uuid.UUID
				var name, phone string
				if prows.Scan(&sid, &name, &phone) == nil {
					parents[sid] = parentInfo{name, phone}
				}
			}
		}
	}

	type studentWithExtras struct {
		Student
		FeeBalance  float64 `json:"fee_balance"`
		ClassName   string  `json:"class_name,omitempty"`
		ParentName  string  `json:"parent_name,omitempty"`
		ParentPhone string  `json:"parent_phone,omitempty"`
	}
	out := make([]studentWithExtras, len(students))
	for i, st := range students {
		p := parents[st.ID]
		out[i] = studentWithExtras{
			Student:     st,
			FeeBalance:  balances[st.ID],
			ClassName:   classNames[st.CurrentClassID.UUID],
			ParentName:  p.Name,
			ParentPhone: p.Phone,
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"results": out, "count": total, "page": page, "page_size": pageSize,
	})
}

// ledger returns the full fee/payment history for one student (identified by ?student_id=).
func (s *StudentsService) ledger(w http.ResponseWriter, r *http.Request) {
	sid, err := uuid.Parse(r.URL.Query().Get("student_id"))
	if err != nil {
		httpx.ErrBadRequest(w, "student_id required")
		return
	}
	var name, admNo, className string
	_ = s.pool.QueryRow(r.Context(), `
		SELECT s.full_name, s.admission_no, COALESCE(cg.name,'')
		FROM students s LEFT JOIN class_groups cg ON cg.id = s.current_class_id
		WHERE s.id = $1`, sid).Scan(&name, &admNo, &className)

	invoices := []map[string]interface{}{}
	irows, err := s.pool.Query(r.Context(), `
		SELECT i.id, COALESCE(t.name,''), COALESCE(t.year::text,''),
		       i.total_amount::float8, i.total_paid::float8, i.total_waivers::float8, i.balance::float8,
		       to_char(i.created_at,'DD/MM/YYYY')
		FROM invoices i LEFT JOIN terms t ON t.id = i.term_id
		WHERE i.student_id = $1 ORDER BY i.created_at DESC`, sid)
	if err == nil {
		defer irows.Close()
		for irows.Next() {
			var iid uuid.UUID
			var term, year, date string
			var total, paid, waivers, balance float64
			if irows.Scan(&iid, &term, &year, &total, &paid, &waivers, &balance, &date) == nil {
				invoices = append(invoices, map[string]interface{}{
					"invoice_id": iid, "term": term, "year": year,
					"total_amount": total, "total_paid": paid, "total_waivers": waivers,
					"balance": balance, "date": date,
				})
			}
		}
	}

	payments := []map[string]interface{}{}
	prows, err := s.pool.Query(r.Context(), `
		SELECT id, amount::float8, to_char(payment_date,'DD/MM/YYYY'),
		       payment_method, COALESCE(payer_name,''), COALESCE(transaction_ref,''), status
		FROM payments WHERE student_id = $1 ORDER BY payment_date DESC`, sid)
	if err == nil {
		defer prows.Close()
		for prows.Next() {
			var pid uuid.UUID
			var amount float64
			var date, method, payer, ref, status string
			if prows.Scan(&pid, &amount, &date, &method, &payer, &ref, &status) == nil {
				payments = append(payments, map[string]interface{}{
					"payment_id": pid, "amount": amount, "date": date,
					"method": method, "payer": payer, "ref": ref, "status": status,
				})
			}
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"student":  map[string]interface{}{"id": sid, "full_name": name, "admission_no": admNo, "class_name": className},
		"invoices": invoices,
		"payments": payments,
	})
}

// promote bulk-moves a set of students to a target class (end-of-year promotion).
func (s *StudentsService) promote(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StudentIDs    []uuid.UUID `json:"student_ids"`
		TargetClassID uuid.UUID   `json:"target_class_id"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if len(body.StudentIDs) == 0 {
		httpx.ErrBadRequest(w, "student_ids required")
		return
	}
	_, err := s.pool.Exec(r.Context(), `
		UPDATE students SET current_class_id = $1, updated_at = NOW()
		WHERE id = ANY($2)`, body.TargetClassID, body.StudentIDs)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"promoted": len(body.StudentIDs), "target_class_id": body.TargetClassID,
	})
}

// feeArrears returns active students who have an outstanding invoice balance > 0.
func (s *StudentsService) feeArrears(w http.ResponseWriter, r *http.Request) {
	var args []interface{}
	classFilter := ""
	if v := r.URL.Query().Get("class_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			classFilter = " AND s.current_class_id = $1"
			args = append(args, id)
		}
	}
	rows, err := s.pool.Query(r.Context(), `
		SELECT s.id, s.full_name, s.admission_no, COALESCE(s.phone,''),
		       COALESCE(cg.name,'') AS class_name,
		       COALESCE(g.first_name || ' ' || g.last_name,'') AS parent_name,
		       COALESCE(g.phone,'') AS parent_phone,
		       COALESCE(SUM(i.balance),0) AS outstanding
		FROM students s
		LEFT JOIN class_groups cg ON cg.id = s.current_class_id
		LEFT JOIN student_guardians sg ON sg.student_id = s.id AND sg.is_primary = TRUE
		LEFT JOIN guardians g ON g.id = sg.guardian_id
		LEFT JOIN invoices i ON i.student_id = s.id
		WHERE s.is_active = TRUE`+classFilter+`
		GROUP BY s.id, s.full_name, s.admission_no, s.phone, cg.name, g.first_name, g.last_name, g.phone
		HAVING COALESCE(SUM(i.balance),0) > 0
		ORDER BY outstanding DESC, s.last_name`, args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var sid uuid.UUID
		var name, admNo, phone, className, parentName, parentPhone string
		var outstanding float64
		if rows.Scan(&sid, &name, &admNo, &phone, &className, &parentName, &parentPhone, &outstanding) == nil {
			out = append(out, map[string]interface{}{
				"student_id": sid, "full_name": name, "admission_no": admNo, "phone": phone,
				"class_name": className, "parent_name": parentName, "parent_phone": parentPhone,
				"outstanding": outstanding,
			})
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *StudentsService) create(w http.ResponseWriter, r *http.Request) {
	var body Student
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if body.BoardingStatus == "" {
		body.BoardingStatus = "day"
	}
	if body.Status == "" {
		body.Status = "active"
	}
	body.IsActive = true
	fullName := strings.TrimSpace(strings.Join(filterEmpty(body.FirstName, body.MiddleName, body.LastName), " "))
	searchName := strings.TrimSpace(body.FirstName + " " + body.LastName)

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback(r.Context())

	err = tx.QueryRow(r.Context(), `
		INSERT INTO students (admission_no, first_name, middle_name, last_name, full_name, search_name, gender,
			date_of_birth, national_id, birth_certificate_no, phone, email, class_id, current_class_id, stream_id,
			academic_year_id, admission_date, boarding_status, blood_group, allergies, chronic_conditions,
			residential_address, former_school, kcpe_marks, kcpe_grade, upi, status, is_active,
			kemis_number, assessment_number, uses_transport, transport_route_id, photo_path, birth_cert_document_path)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,
			$29,$30,$31,$32,$33,$34)
		RETURNING id`,
		body.AdmissionNo, body.FirstName, body.MiddleName, body.LastName, fullName, searchName, body.Gender,
		body.DateOfBirth, body.NationalID, body.BirthCertificateNo, body.Phone, body.Email, body.ClassID.Ptr(), body.CurrentClassID.Ptr(),
		body.StreamID.Ptr(), body.AcademicYearID.Ptr(), body.AdmissionDate, body.BoardingStatus, body.BloodGroup, body.Allergies,
		body.ChronicConditions, body.ResidentialAddress, body.FormerSchool, body.KcpeMarks, body.KcpeGrade, body.UPI,
		body.Status, body.IsActive,
		body.KemisNumber, body.AssessmentNumber, body.UsesTransport, body.TransportRouteID.Ptr(), body.PhotoPath, body.BirthCertDocumentPath,
	).Scan(&body.ID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	if body.Citizenship == "" {
		body.Citizenship = "Kenya"
	}
	if _, err := tx.Exec(r.Context(), welfareUpdateSQL,
		append([]interface{}{body.ID}, body.StudentWelfare.update().args()...)...); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}

	for _, clubID := range body.ClubIDs {
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO student_clubs (student_id, club_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			body.ID, clubID); err != nil {
			httpx.Error(w, http.StatusBadRequest, err)
			return
		}
	}

	for _, g := range body.Guardians {
		if err := linkOrCreateGuardianTx(r.Context(), tx, body.ID, g); err != nil {
			httpx.Error(w, http.StatusBadRequest, err)
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	s.billIfEnrolled(r.Context(), body.ID, body.CurrentClassID.Valid)
	body.FullName = fullName
	httpx.JSON(w, http.StatusCreated, body)
}

func filterEmpty(parts ...string) []string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}

// linkOrCreateGuardianTx is shared by student creation (inline guardians) and
// POST /students/{id}/guardians/ (linking after the fact). If g.GuardianID
// is set it links that existing guardian; otherwise it creates a new
// guardian row from the inline fields first.
func linkOrCreateGuardianTx(ctx context.Context, tx pgx.Tx, studentID uuid.UUID, g GuardianLink) error {
	guardianID := g.GuardianID.Ptr()
	if guardianID == nil {
		if g.FirstName == "" || g.LastName == "" || g.Phone == "" {
			return errors.New("guardian_id or (first_name, last_name, phone) required to link a guardian")
		}
		relation := g.Relation
		if relation == "" {
			relation = "parent"
		}
		var newID uuid.UUID
		err := tx.QueryRow(ctx, `
			INSERT INTO guardians (first_name, last_name, email, phone, alternate_phone, national_id,
				relation, occupation, employer, residential_address, postal_address, is_active, id_document_path)
			VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10,$11,true,$12) RETURNING id`,
			g.FirstName, g.LastName, g.Email, g.Phone, g.AlternatePhone, g.NationalID,
			relation, g.Occupation, g.Employer, g.ResidentialAddress, g.PostalAddress, g.IDDocumentPath,
		).Scan(&newID)
		if err != nil {
			return err
		}
		guardianID = &newID
	}
	relation := g.Relation
	if relation == "" {
		relation = "parent"
	}
	contactPriority := g.ContactPriority
	if contactPriority == 0 {
		contactPriority = 1
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO student_guardians (student_id, guardian_id, relation, is_primary, has_custody, contact_priority)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (student_id, guardian_id) DO UPDATE SET
			relation = EXCLUDED.relation, is_primary = EXCLUDED.is_primary,
			has_custody = EXCLUDED.has_custody, contact_priority = EXCLUDED.contact_priority`,
		studentID, *guardianID, relation, g.IsPrimary, g.HasCustody, contactPriority)
	return err
}

func (s *StudentsService) get(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	st, err := scanStudent(s.pool.QueryRow(r.Context(), `SELECT `+studentCols+` FROM students WHERE id = $1`, id))
	if err != nil {
		httpx.ErrNotFound(w, "student not found")
		return
	}

	guardians := []map[string]interface{}{}
	rows, err := s.pool.Query(r.Context(), `
		SELECT g.id, g.first_name, g.last_name, g.phone, COALESCE(g.email,''), sg.relation, sg.is_primary, sg.contact_priority, g.user_id,
		       g.alternate_phone, g.national_id, g.occupation, g.employer, g.residential_address, g.postal_address, sg.has_custody
		FROM student_guardians sg JOIN guardians g ON g.id = sg.guardian_id
		WHERE sg.student_id = $1 ORDER BY sg.is_primary DESC, sg.contact_priority ASC`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var gid uuid.UUID
			var fn, ln, phone, email, rel, alt, nid, occ, emp, addr, postal string
			var primary, custody bool
			var prio int
			var userID *uuid.UUID
			if rows.Scan(&gid, &fn, &ln, &phone, &email, &rel, &primary, &prio, &userID,
				&alt, &nid, &occ, &emp, &addr, &postal, &custody) == nil {
				guardians = append(guardians, map[string]interface{}{
					"guardian_id": gid, "first_name": fn, "last_name": ln, "phone": phone, "email": email,
					"relation": rel, "is_primary": primary, "contact_priority": prio, "has_login": userID != nil,
					"alternate_phone": alt, "national_id": nid, "occupation": occ, "employer": emp,
					"residential_address": addr, "postal_address": postal, "has_custody": custody,
				})
			}
		}
	}

	clubs := []map[string]interface{}{}
	clubRows, err := s.pool.Query(r.Context(), `
		SELECT c.id, c.name, sc.joined_date::text
		FROM student_clubs sc JOIN clubs c ON c.id = sc.club_id
		WHERE sc.student_id = $1 ORDER BY c.name`, id)
	if err == nil {
		defer clubRows.Close()
		for clubRows.Next() {
			var cid uuid.UUID
			var name, joined string
			if clubRows.Scan(&cid, &name, &joined) == nil {
				clubs = append(clubs, map[string]interface{}{"club_id": cid, "name": name, "joined_date": joined})
			}
		}
	}

	var totalDue, totalPaid, totalWaivers, balance float64
	_ = s.pool.QueryRow(r.Context(), `
		SELECT COALESCE(SUM(total_amount), 0), COALESCE(SUM(total_paid), 0),
		       COALESCE(SUM(total_waivers), 0), COALESCE(SUM(balance), 0)
		FROM invoices WHERE student_id = $1`, id).
		Scan(&totalDue, &totalPaid, &totalWaivers, &balance)

	resp := studentToMap(st)
	resp["guardians"] = guardians
	resp["clubs"] = clubs
	// fee_balance is the single number that matters for "does this student
	// owe money" — the sum of every invoice's balance across all terms. It
	// updates automatically because it's read live from invoices.balance,
	// which the fee listener and billing generator both keep current; there
	// is no separate copy of this value stored on the student row itself.
	resp["fee_balance"] = balance
	resp["fee_summary"] = map[string]interface{}{
		"total_due":     totalDue,
		"total_paid":    totalPaid,
		"total_waivers": totalWaivers,
		"balance":       balance,
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func studentToMap(st Student) map[string]interface{} {
	return map[string]interface{}{
		"id": st.ID, "admission_no": st.AdmissionNo, "first_name": st.FirstName, "middle_name": st.MiddleName,
		"last_name": st.LastName, "full_name": st.FullName, "gender": st.Gender, "date_of_birth": st.DateOfBirth,
		"national_id": st.NationalID, "birth_certificate_no": st.BirthCertificateNo, "phone": st.Phone, "email": st.Email,
		"class_id": st.ClassID, "current_class_id": st.CurrentClassID, "stream_id": st.StreamID,
		"academic_year_id": st.AcademicYearID, "admission_date": st.AdmissionDate, "boarding_status": st.BoardingStatus,
		"blood_group": st.BloodGroup, "allergies": st.Allergies, "chronic_conditions": st.ChronicConditions,
		"residential_address": st.ResidentialAddress, "former_school": st.FormerSchool, "kcpe_marks": st.KcpeMarks,
		"kcpe_grade": st.KcpeGrade, "upi": st.UPI, "status": st.Status, "is_active": st.IsActive,
		"kemis_number": st.KemisNumber, "assessment_number": st.AssessmentNumber, "uses_transport": st.UsesTransport,
		"transport_route_id": st.TransportRouteID, "photo_path": st.PhotoPath,
		"birth_cert_document_path": st.BirthCertDocumentPath,
		"photo_consent":            st.PhotoConsent, "parental_status": st.ParentalStatus,
		"has_special_needs": st.HasSpecialNeeds, "special_needs": st.SpecialNeeds, "religion": st.Religion,
		"doctor_name": st.DoctorName, "doctor_phone": st.DoctorPhone, "preferred_hospital": st.PreferredHospital,
		"has_scholarship": st.HasScholarship, "scholarship_type": st.ScholarshipType, "sponsor_name": st.SponsorName,
		"sponsor_phone": st.SponsorPhone, "sponsor_location": st.SponsorLocation,
		"sponsor_contact_person": st.SponsorContactPerson, "sponsor_contact_details": st.SponsorContactDetails,
		"citizenship": st.Citizenship, "home_county": st.HomeCounty, "residence_county": st.ResidenceCounty,
		"sub_county": st.SubCounty,
	}
}

// studentUpdateBody is a locally-scoped, all-pointer/all-optional mirror of
// Student used only for PATCH/PUT — plain-decoded (no validate:"required")
// so an edit that only touches a couple of fields (e.g. just current_class_id
// for a promotion) doesn't 400 on missing required fields or, worse, silently
// zero out every other column via a full-struct bind. FK fields use
// *OptionalUUID + a CASE WHEN $bool pattern (see updateClass in academics.go)
// so "field omitted" and "field explicitly cleared to null" stay distinct.
type studentUpdateBody struct {
	AdmissionNo           *string       `json:"admission_no,omitempty"`
	FirstName             *string       `json:"first_name,omitempty"`
	MiddleName            *string       `json:"middle_name,omitempty"`
	LastName              *string       `json:"last_name,omitempty"`
	Gender                *string       `json:"gender,omitempty"`
	DateOfBirth           *string       `json:"date_of_birth,omitempty"`
	NationalID            *string       `json:"national_id,omitempty"`
	BirthCertificateNo    *string       `json:"birth_certificate_no,omitempty"`
	Phone                 *string       `json:"phone,omitempty"`
	Email                 *string       `json:"email,omitempty"`
	ClassID               *OptionalUUID `json:"class_id,omitempty"`
	CurrentClassID        *OptionalUUID `json:"current_class_id,omitempty"`
	StreamID              *OptionalUUID `json:"stream_id,omitempty"`
	AcademicYearID        *OptionalUUID `json:"academic_year_id,omitempty"`
	AdmissionDate         *string       `json:"admission_date,omitempty"`
	BoardingStatus        *string       `json:"boarding_status,omitempty"`
	BloodGroup            *string       `json:"blood_group,omitempty"`
	Allergies             *string       `json:"allergies,omitempty"`
	ChronicConditions     *string       `json:"chronic_conditions,omitempty"`
	ResidentialAddress    *string       `json:"residential_address,omitempty"`
	FormerSchool          *string       `json:"former_school,omitempty"`
	KcpeMarks             *int          `json:"kcpe_marks,omitempty"`
	KcpeGrade             *string       `json:"kcpe_grade,omitempty"`
	UPI                   *string       `json:"upi,omitempty"`
	Status                *string       `json:"status,omitempty"`
	IsActive              *bool         `json:"is_active,omitempty"`
	KemisNumber           *string       `json:"kemis_number,omitempty"`
	AssessmentNumber      *string       `json:"assessment_number,omitempty"`
	UsesTransport         *bool         `json:"uses_transport,omitempty"`
	TransportRouteID      *OptionalUUID `json:"transport_route_id,omitempty"`
	PhotoPath             *string       `json:"photo_path,omitempty"`
	BirthCertDocumentPath *string       `json:"birth_cert_document_path,omitempty"`
	studentWelfareUpdate
}

func (s *StudentsService) update(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var body studentUpdateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if errs := validWelfareUpdate(body.studentWelfareUpdate); len(errs) > 0 {
		fieldErrors(w, errs)
		return
	}

	var classPtr, currentClassPtr, streamPtr, yearPtr, routePtr *uuid.UUID
	if body.ClassID != nil {
		classPtr = body.ClassID.Ptr()
	}
	if body.CurrentClassID != nil {
		currentClassPtr = body.CurrentClassID.Ptr()
	}
	if body.StreamID != nil {
		streamPtr = body.StreamID.Ptr()
	}
	if body.AcademicYearID != nil {
		yearPtr = body.AcademicYearID.Ptr()
	}
	if body.TransportRouteID != nil {
		routePtr = body.TransportRouteID.Ptr()
	}

	var fullNamePtr, searchNamePtr *string
	if body.FirstName != nil || body.MiddleName != nil || body.LastName != nil {
		existing, err := scanStudent(s.pool.QueryRow(r.Context(), `SELECT `+studentCols+` FROM students WHERE id = $1`, id))
		if err != nil {
			httpx.ErrNotFound(w, "student not found")
			return
		}
		first, middle, last := existing.FirstName, existing.MiddleName, existing.LastName
		if body.FirstName != nil {
			first = *body.FirstName
		}
		if body.MiddleName != nil {
			middle = *body.MiddleName
		}
		if body.LastName != nil {
			last = *body.LastName
		}
		full := strings.TrimSpace(strings.Join(filterEmpty(first, middle, last), " "))
		search := strings.TrimSpace(first + " " + last)
		fullNamePtr, searchNamePtr = &full, &search
	}

	_, err = s.pool.Exec(r.Context(), `
		UPDATE students SET
			admission_no=COALESCE(NULLIF($1,''), admission_no),
			first_name=COALESCE(NULLIF($2,''), first_name),
			middle_name=COALESCE($3, middle_name),
			last_name=COALESCE(NULLIF($4,''), last_name),
			full_name=COALESCE($5, full_name), search_name=COALESCE($6, search_name),
			gender=COALESCE(NULLIF($7,''), gender), date_of_birth=COALESCE($8, date_of_birth),
			national_id=COALESCE($9, national_id), birth_certificate_no=COALESCE($10, birth_certificate_no),
			phone=COALESCE($11, phone), email=COALESCE($12, email),
			class_id=CASE WHEN $13::boolean THEN $14 ELSE class_id END,
			current_class_id=CASE WHEN $15::boolean THEN $16 ELSE current_class_id END,
			stream_id=CASE WHEN $17::boolean THEN $18 ELSE stream_id END,
			academic_year_id=CASE WHEN $19::boolean THEN $20 ELSE academic_year_id END,
			admission_date=COALESCE($21, admission_date),
			boarding_status=COALESCE(NULLIF($22,''), boarding_status),
			blood_group=COALESCE($23, blood_group), allergies=COALESCE($24, allergies),
			chronic_conditions=COALESCE($25, chronic_conditions), residential_address=COALESCE($26, residential_address),
			former_school=COALESCE($27, former_school), kcpe_marks=COALESCE($28, kcpe_marks),
			kcpe_grade=COALESCE($29, kcpe_grade), upi=COALESCE($30, upi),
			status=COALESCE(NULLIF($31,''), status), is_active=COALESCE($32, is_active),
			kemis_number=COALESCE($33, kemis_number), assessment_number=COALESCE($34, assessment_number),
			uses_transport=COALESCE($35, uses_transport),
			transport_route_id=CASE WHEN $36::boolean THEN $37 ELSE transport_route_id END,
			photo_path=COALESCE($38, photo_path), birth_cert_document_path=COALESCE($39, birth_cert_document_path),
			updated_at=NOW()
		WHERE id = $40`,
		body.AdmissionNo, body.FirstName, body.MiddleName, body.LastName, fullNamePtr, searchNamePtr,
		body.Gender, body.DateOfBirth, body.NationalID, body.BirthCertificateNo, body.Phone, body.Email,
		body.ClassID != nil, classPtr, body.CurrentClassID != nil, currentClassPtr,
		body.StreamID != nil, streamPtr, body.AcademicYearID != nil, yearPtr,
		body.AdmissionDate, body.BoardingStatus, body.BloodGroup, body.Allergies,
		body.ChronicConditions, body.ResidentialAddress, body.FormerSchool, body.KcpeMarks,
		body.KcpeGrade, body.UPI, body.Status, body.IsActive,
		body.KemisNumber, body.AssessmentNumber, body.UsesTransport,
		body.TransportRouteID != nil, routePtr,
		body.PhotoPath, body.BirthCertDocumentPath, id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	if _, err := s.pool.Exec(r.Context(), welfareUpdateSQL,
		append([]interface{}{id}, body.studentWelfareUpdate.args()...)...); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}

	updated, err := scanStudent(s.pool.QueryRow(r.Context(), `SELECT `+studentCols+` FROM students WHERE id = $1`, id))
	if err != nil {
		httpx.ErrNotFound(w, "student not found")
		return
	}
	// Rerunning the auto-bill here too (not just on create) means a class
	// assignment/change (promotion, transfer, initial assignment for a
	// student registered before a class existed) keeps the invoice in sync
	// with whatever fee structure now applies — billStudent's per-line
	// delete-then-insert makes this safe to repeat.
	s.billIfEnrolled(r.Context(), id, updated.CurrentClassID.Valid)
	httpx.JSON(w, http.StatusOK, updated)
}

func (s *StudentsService) delete(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM students WHERE id = $1`, id); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}

func (s *StudentsService) linkGuardian(w http.ResponseWriter, r *http.Request) {
	studentID, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid student id")
		return
	}
	var body GuardianLink
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback(r.Context())
	if err := linkOrCreateGuardianTx(r.Context(), tx, studentID, body); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]string{"status": "linked"})
}

func (s *StudentsService) unlinkGuardian(w http.ResponseWriter, r *http.Request) {
	studentID, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid student id")
		return
	}
	guardianID, err := uuid.Parse(chi.URLParam(r, "guardian_id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid guardian id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM student_guardians WHERE student_id = $1 AND guardian_id = $2`, studentID, guardianID); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "unlinked"})
}

type ClubLinkRequest struct {
	ClubID uuid.UUID `json:"club_id" validate:"required,uuid"`
}

func (s *StudentsService) linkClub(w http.ResponseWriter, r *http.Request) {
	studentID, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid student id")
		return
	}
	var body ClubLinkRequest
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	_, err = s.pool.Exec(r.Context(), `
		INSERT INTO student_clubs (student_id, club_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		studentID, body.ClubID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]string{"status": "linked"})
}

func (s *StudentsService) unlinkClub(w http.ResponseWriter, r *http.Request) {
	studentID, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid student id")
		return
	}
	clubID, err := uuid.Parse(chi.URLParam(r, "club_id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid club id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM student_clubs WHERE student_id = $1 AND club_id = $2`, studentID, clubID); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "unlinked"})
}
