package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
)

// StaffService and DepartmentsService are direct-to-DB CRUD replacing the
// placeholder stubs — payroll generation reads staff_profiles directly
// (internal/finance/payroll.go), so without this, every payroll run was
// silently generating zero entries: there was no way to create a staff
// member at all. A staff profile always attaches to an existing user
// account (see UsersService) — that's where name/email/phone/role live;
// this table only holds employment-specific fields.
type StaffService struct {
	pool *pgxpool.Pool
}

func NewStaffService(pool *pgxpool.Pool) *StaffService {
	return &StaffService{pool: pool}
}

func (s *StaffService) RegisterRoutes(r chi.Router) {
	r.Route("/staff/", func(r chi.Router) {
		r.Use(adminOnly())
		r.Get("/", s.list)
		r.Post("/", s.create)
		r.Get("/{id}/", s.get)
		r.Put("/{id}/", s.update)
		r.Patch("/{id}/", s.update)
		r.Delete("/{id}/", s.delete)
	})
}

type StaffProfile struct {
	ID                       uuid.UUID  `json:"id"`
	UserID                   uuid.UUID  `json:"user_id" validate:"required"`
	FullName                 string     `json:"full_name,omitempty"`
	Email                    string     `json:"email,omitempty"`
	Phone                    string     `json:"phone,omitempty"`
	Role                     string     `json:"role,omitempty"`
	EmployeeNo               string     `json:"employee_no" validate:"required"`
	NationalID               string     `json:"national_id"`
	TSCNo                    string     `json:"tsc_no"`
	Gender                   string     `json:"gender" validate:"omitempty,oneof=male female other"`
	DateOfBirth              *string    `json:"date_of_birth,omitempty"`
	EmploymentType           string     `json:"employment_type" validate:"omitempty,oneof=tsc bom contract intern volunteer"`
	DepartmentID             *uuid.UUID `json:"department_id,omitempty"`
	DepartmentName           string     `json:"department_name,omitempty"`
	Designation              string     `json:"designation"`
	DateOfJoining            *string    `json:"date_of_joining,omitempty"`
	HighestQualification     string     `json:"highest_qualification"`
	BasicSalary              float64    `json:"basic_salary"`
	KRAPin                   string     `json:"kra_pin"`
	NHIFNo                   string     `json:"nhif_no"`
	NSSFNo                   string     `json:"nssf_no"`
	BankName                 string     `json:"bank_name"`
	BankBranch               string     `json:"bank_branch"`
	BankAccountNo            string     `json:"bank_account_no"`
	EmergencyContactName     string     `json:"emergency_contact_name"`
	EmergencyContactPhone    string     `json:"emergency_contact_phone"`
	EmergencyContactRelation string     `json:"emergency_contact_relation"`
	ResidentialAddress       string     `json:"residential_address"`
	IsActive                 bool       `json:"is_active"`

	// Read: the classes and subjects this staff member teaches (empty for
	// non-teaching staff). Write (create only): assign them in the same call.
	Subjects    []string            `json:"subjects"`
	Assignments []SubjectAssignment `json:"assignments"`
	AssignTo    []assignmentInput   `json:"assign_subjects,omitempty"`

	StaffDetails
}

// StaffDetails is the rest of the staff registration form (personal and
// welfare details, medical contacts, home location, training background and
// scanned documents). Kept apart from the core columns so create/update
// write it with one statement of its own.
type StaffDetails struct {
	MiddleName            string  `json:"middle_name"`
	MaritalStatus         string  `json:"marital_status" validate:"omitempty,oneof=single married divorced widowed separated"`
	Religion              string  `json:"religion"`
	HasDisability         bool    `json:"has_disability"`
	DisabilityDetails     string  `json:"disability_details"`
	AlternatePhone        string  `json:"alternate_phone"`
	PostalAddress         string  `json:"postal_address"`
	Citizenship           string  `json:"citizenship"`
	HomeCounty            string  `json:"home_county"`
	ResidenceCounty       string  `json:"residence_county"`
	SubCounty             string  `json:"sub_county"`
	BloodGroup            string  `json:"blood_group"`
	Allergies             string  `json:"allergies"`
	DoctorName            string  `json:"doctor_name"`
	DoctorPhone           string  `json:"doctor_phone"`
	PreferredHospital     string  `json:"preferred_hospital"`
	TeachingSubjects      string  `json:"teaching_subjects"`
	TrainingInstitution   string  `json:"training_institution"`
	YearOfGraduation      string  `json:"year_of_graduation" validate:"omitempty,len=4,numeric"`
	PreviousEmployer      string  `json:"previous_employer"`
	ContractEndDate       *string `json:"contract_end_date,omitempty"`
	PhotoConsent          bool    `json:"photo_consent"`
	PhotoPath             string  `json:"photo_path"`
	IDDocumentPath        string  `json:"id_document_path"`
	CertificatesPath      string  `json:"certificates_path"`
	AppointmentLetterPath string  `json:"appointment_letter_path"`
	GoodConductPath       string  `json:"good_conduct_path"`
}

// staffDetailsUpdate is the PATCH mirror: nil leaves a column unchanged.
// contract_end_date "" clears the date.
type staffDetailsUpdate struct {
	MiddleName            *string `json:"middle_name,omitempty"`
	MaritalStatus         *string `json:"marital_status,omitempty"`
	Religion              *string `json:"religion,omitempty"`
	HasDisability         *bool   `json:"has_disability,omitempty"`
	DisabilityDetails     *string `json:"disability_details,omitempty"`
	AlternatePhone        *string `json:"alternate_phone,omitempty"`
	PostalAddress         *string `json:"postal_address,omitempty"`
	Citizenship           *string `json:"citizenship,omitempty"`
	HomeCounty            *string `json:"home_county,omitempty"`
	ResidenceCounty       *string `json:"residence_county,omitempty"`
	SubCounty             *string `json:"sub_county,omitempty"`
	BloodGroup            *string `json:"blood_group,omitempty"`
	Allergies             *string `json:"allergies,omitempty"`
	DoctorName            *string `json:"doctor_name,omitempty"`
	DoctorPhone           *string `json:"doctor_phone,omitempty"`
	PreferredHospital     *string `json:"preferred_hospital,omitempty"`
	TeachingSubjects      *string `json:"teaching_subjects,omitempty"`
	TrainingInstitution   *string `json:"training_institution,omitempty"`
	YearOfGraduation      *string `json:"year_of_graduation,omitempty"`
	PreviousEmployer      *string `json:"previous_employer,omitempty"`
	ContractEndDate       *string `json:"contract_end_date,omitempty"`
	PhotoConsent          *bool   `json:"photo_consent,omitempty"`
	PhotoPath             *string `json:"photo_path,omitempty"`
	IDDocumentPath        *string `json:"id_document_path,omitempty"`
	CertificatesPath      *string `json:"certificates_path,omitempty"`
	AppointmentLetterPath *string `json:"appointment_letter_path,omitempty"`
	GoodConductPath       *string `json:"good_conduct_path,omitempty"`
}

const staffDetailCols = `sp.middle_name, sp.marital_status, sp.religion, sp.has_disability, sp.disability_details, sp.alternate_phone, sp.postal_address, sp.citizenship, sp.home_county, sp.residence_county, sp.sub_county, sp.blood_group, sp.allergies, sp.doctor_name, sp.doctor_phone, sp.preferred_hospital, sp.teaching_subjects, sp.training_institution, sp.year_of_graduation, sp.previous_employer, sp.contract_end_date::text, sp.photo_consent, sp.photo_path, sp.id_document_path, sp.certificates_path, sp.appointment_letter_path, sp.good_conduct_path`

func (d *StaffDetails) scanDest() []interface{} {
	return []interface{}{&d.MiddleName, &d.MaritalStatus, &d.Religion, &d.HasDisability, &d.DisabilityDetails, &d.AlternatePhone, &d.PostalAddress, &d.Citizenship, &d.HomeCounty, &d.ResidenceCounty, &d.SubCounty, &d.BloodGroup, &d.Allergies, &d.DoctorName, &d.DoctorPhone, &d.PreferredHospital, &d.TeachingSubjects, &d.TrainingInstitution, &d.YearOfGraduation, &d.PreviousEmployer, &d.ContractEndDate, &d.PhotoConsent, &d.PhotoPath, &d.IDDocumentPath, &d.CertificatesPath, &d.AppointmentLetterPath, &d.GoodConductPath}
}

func (d StaffDetails) update() staffDetailsUpdate {
	return staffDetailsUpdate{&d.MiddleName, &d.MaritalStatus, &d.Religion, &d.HasDisability, &d.DisabilityDetails, &d.AlternatePhone, &d.PostalAddress, &d.Citizenship, &d.HomeCounty, &d.ResidenceCounty, &d.SubCounty, &d.BloodGroup, &d.Allergies, &d.DoctorName, &d.DoctorPhone, &d.PreferredHospital, &d.TeachingSubjects, &d.TrainingInstitution, &d.YearOfGraduation, &d.PreviousEmployer, d.ContractEndDate, &d.PhotoConsent, &d.PhotoPath, &d.IDDocumentPath, &d.CertificatesPath, &d.AppointmentLetterPath, &d.GoodConductPath}
}

func (u staffDetailsUpdate) args() []interface{} {
	return []interface{}{u.MiddleName, u.MaritalStatus, u.Religion, u.HasDisability, u.DisabilityDetails, u.AlternatePhone, u.PostalAddress, u.Citizenship, u.HomeCounty, u.ResidenceCounty, u.SubCounty, u.BloodGroup, u.Allergies, u.DoctorName, u.DoctorPhone, u.PreferredHospital, u.TeachingSubjects, u.TrainingInstitution, u.YearOfGraduation, u.PreviousEmployer, u.ContractEndDate, u.PhotoConsent, u.PhotoPath, u.IDDocumentPath, u.CertificatesPath, u.AppointmentLetterPath, u.GoodConductPath}
}

const staffDetailsUpdateSQL = `UPDATE staff_profiles SET
		middle_name = COALESCE($2, middle_name),
		marital_status = COALESCE($3, marital_status),
		religion = COALESCE($4, religion),
		has_disability = COALESCE($5, has_disability),
		disability_details = COALESCE($6, disability_details),
		alternate_phone = COALESCE($7, alternate_phone),
		postal_address = COALESCE($8, postal_address),
		citizenship = COALESCE($9, citizenship),
		home_county = COALESCE($10, home_county),
		residence_county = COALESCE($11, residence_county),
		sub_county = COALESCE($12, sub_county),
		blood_group = COALESCE($13, blood_group),
		allergies = COALESCE($14, allergies),
		doctor_name = COALESCE($15, doctor_name),
		doctor_phone = COALESCE($16, doctor_phone),
		preferred_hospital = COALESCE($17, preferred_hospital),
		teaching_subjects = COALESCE($18, teaching_subjects),
		training_institution = COALESCE($19, training_institution),
		year_of_graduation = COALESCE($20, year_of_graduation),
		previous_employer = COALESCE($21, previous_employer),
		contract_end_date = CASE WHEN $22::text = '' THEN NULL ELSE COALESCE($22::date, contract_end_date) END,
		photo_consent = COALESCE($23, photo_consent),
		photo_path = COALESCE($24, photo_path),
		id_document_path = COALESCE($25, id_document_path),
		certificates_path = COALESCE($26, certificates_path),
		appointment_letter_path = COALESCE($27, appointment_letter_path),
		good_conduct_path = COALESCE($28, good_conduct_path),
		updated_at = NOW()
	WHERE id = $1`

func validStaffDetailsUpdate(u staffDetailsUpdate) map[string]string {
	errs := map[string]string{}
	if u.MaritalStatus != nil && !containsStr([]string{"", "single", "married", "divorced", "widowed", "separated"}, *u.MaritalStatus) {
		errs["marital_status"] = "must be single, married, divorced, widowed or separated"
	}
	if u.YearOfGraduation != nil && *u.YearOfGraduation != "" {
		if _, err := strconv.Atoi(*u.YearOfGraduation); err != nil || len(*u.YearOfGraduation) != 4 {
			errs["year_of_graduation"] = "must be a 4-digit year"
		}
	}
	return errs
}

const staffCols = `sp.id, sp.user_id, concat_ws(' ', NULLIF(u.first_name,''), NULLIF(sp.middle_name,''), NULLIF(u.last_name,'')),
	COALESCE(u.email,''), COALESCE(u.phone,''), COALESCE(UPPER(u.role),''), sp.employee_no, sp.national_id,
	sp.tsc_no, sp.gender, sp.date_of_birth::text, sp.employment_type,
	sp.department_id, COALESCE(d.name,''), sp.designation, sp.date_of_joining::text, sp.highest_qualification,
	sp.basic_salary, sp.kra_pin, sp.nhif_no, sp.nssf_no, sp.bank_name, sp.bank_branch, sp.bank_account_no,
	sp.emergency_contact_name, sp.emergency_contact_phone, sp.emergency_contact_relation, sp.residential_address,
	sp.is_active, ` + staffDetailCols

const staffJoin = `staff_profiles sp
	JOIN users u ON u.id = sp.user_id
	LEFT JOIN departments d ON d.id = sp.department_id`

func scanStaff(row interface {
	Scan(dest ...interface{}) error
}) (StaffProfile, error) {
	var sp StaffProfile
	dest := []interface{}{&sp.ID, &sp.UserID, &sp.FullName, &sp.Email, &sp.Phone, &sp.Role, &sp.EmployeeNo,
		&sp.NationalID, &sp.TSCNo, &sp.Gender, &sp.DateOfBirth, &sp.EmploymentType,
		&sp.DepartmentID, &sp.DepartmentName, &sp.Designation, &sp.DateOfJoining, &sp.HighestQualification,
		&sp.BasicSalary, &sp.KRAPin, &sp.NHIFNo, &sp.NSSFNo, &sp.BankName, &sp.BankBranch, &sp.BankAccountNo,
		&sp.EmergencyContactName, &sp.EmergencyContactPhone, &sp.EmergencyContactRelation, &sp.ResidentialAddress,
		&sp.IsActive}
	err := row.Scan(append(dest, sp.StaffDetails.scanDest()...)...)
	return sp, err
}

// withAssignments fills in each staff member's teaching assignments.
func (s *StaffService) withAssignments(r *http.Request, staff []StaffProfile) error {
	ids := make([]uuid.UUID, len(staff))
	for i := range staff {
		ids[i] = staff[i].UserID
	}
	as, err := loadAssignments(r.Context(), s.pool, ids)
	if err != nil {
		return err
	}
	for i := range staff {
		staff[i].Assignments = as[staff[i].UserID]
		if staff[i].Assignments == nil {
			staff[i].Assignments = []SubjectAssignment{}
		}
		staff[i].Subjects = distinctSubjects(staff[i].Assignments)
	}
	return nil
}

func (s *StaffService) loadOne(r *http.Request, id uuid.UUID) (StaffProfile, error) {
	sp, err := scanStaff(s.pool.QueryRow(r.Context(), `SELECT `+staffCols+` FROM `+staffJoin+` WHERE sp.id = $1`, id))
	if err != nil {
		return sp, err
	}
	one := []StaffProfile{sp}
	err = s.withAssignments(r, one)
	return one[0], err
}

func (s *StaffService) list(w http.ResponseWriter, r *http.Request) {
	where := []string{"true"}
	if r.URL.Query().Get("is_active") == "true" {
		where = []string{"sp.is_active = TRUE"}
	}
	rows, err := s.pool.Query(r.Context(), `SELECT `+staffCols+` FROM `+staffJoin+`
		WHERE `+joinAnd(where)+` ORDER BY u.first_name, u.last_name`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	out := []StaffProfile{}
	for rows.Next() {
		sp, err := scanStaff(rows)
		if err != nil {
			rows.Close()
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		out = append(out, sp)
	}
	rows.Close()
	if err := s.withAssignments(r, out); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *StaffService) get(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	sp, err := s.loadOne(r, id)
	if err != nil {
		httpx.ErrNotFound(w, "staff member not found")
		return
	}
	httpx.JSON(w, http.StatusOK, sp)
}

func (s *StaffService) create(w http.ResponseWriter, r *http.Request) {
	var body StaffProfile
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
	id := uuid.New()
	_, err = tx.Exec(r.Context(), `
		INSERT INTO staff_profiles (id, user_id, employee_no, national_id, tsc_no, gender, date_of_birth,
			employment_type, department_id, designation, date_of_joining, highest_qualification, basic_salary,
			kra_pin, nhif_no, nssf_no, bank_name, bank_branch, bank_account_no, emergency_contact_name,
			emergency_contact_phone, emergency_contact_relation, residential_address, is_active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,true,NOW(),NOW())`,
		id, body.UserID, body.EmployeeNo, body.NationalID, body.TSCNo, body.Gender, body.DateOfBirth,
		body.EmploymentType, body.DepartmentID, body.Designation, body.DateOfJoining, body.HighestQualification,
		body.BasicSalary, body.KRAPin, body.NHIFNo, body.NSSFNo, body.BankName, body.BankBranch, body.BankAccountNo,
		body.EmergencyContactName, body.EmergencyContactPhone, body.EmergencyContactRelation, body.ResidentialAddress)
	if err != nil {
		if strings.Contains(err.Error(), "employee_no") {
			httpx.ErrConflict(w, "a staff member with this employee number already exists")
			return
		}
		if strings.Contains(err.Error(), "user_id") {
			httpx.ErrConflict(w, "this user account already has a staff profile")
			return
		}
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	if body.Citizenship == "" {
		body.Citizenship = "Kenya"
	}
	if _, err := tx.Exec(r.Context(), staffDetailsUpdateSQL,
		append([]interface{}{id}, body.StaffDetails.update().args()...)...); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	if len(body.AssignTo) > 0 {
		conflicts, fields, err := applyAssignments(r.Context(), tx, body.UserID, body.AssignTo, false)
		switch {
		case errors.Is(err, errNotTeacher):
			fieldErrors(w, map[string]string{"assign_subjects": "only teaching roles can be assigned subjects"})
			return
		case err != nil:
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		case len(fields) > 0:
			fieldErrors(w, fields)
			return
		case len(conflicts) > 0:
			httpx.JSON(w, http.StatusConflict, map[string]interface{}{
				"error": "conflict", "message": "Some of these subjects already have a teacher.", "conflicts": conflicts,
			})
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	sp, err := s.loadOne(r, id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, sp)
}

func (s *StaffService) update(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var body struct {
		EmployeeNo               *string    `json:"employee_no,omitempty"`
		NationalID               *string    `json:"national_id,omitempty"`
		TSCNo                    *string    `json:"tsc_no,omitempty"`
		Gender                   *string    `json:"gender,omitempty"`
		DateOfBirth              *string    `json:"date_of_birth,omitempty"`
		EmploymentType           *string    `json:"employment_type,omitempty"`
		DepartmentID             *uuid.UUID `json:"department_id,omitempty"`
		Designation              *string    `json:"designation,omitempty"`
		DateOfJoining            *string    `json:"date_of_joining,omitempty"`
		HighestQualification     *string    `json:"highest_qualification,omitempty"`
		BasicSalary              *float64   `json:"basic_salary,omitempty"`
		KRAPin                   *string    `json:"kra_pin,omitempty"`
		NHIFNo                   *string    `json:"nhif_no,omitempty"`
		NSSFNo                   *string    `json:"nssf_no,omitempty"`
		BankName                 *string    `json:"bank_name,omitempty"`
		BankBranch               *string    `json:"bank_branch,omitempty"`
		BankAccountNo            *string    `json:"bank_account_no,omitempty"`
		EmergencyContactName     *string    `json:"emergency_contact_name,omitempty"`
		EmergencyContactPhone    *string    `json:"emergency_contact_phone,omitempty"`
		EmergencyContactRelation *string    `json:"emergency_contact_relation,omitempty"`
		ResidentialAddress       *string    `json:"residential_address,omitempty"`
		IsActive                 *bool      `json:"is_active,omitempty"`
		staffDetailsUpdate
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if errs := validStaffDetailsUpdate(body.staffDetailsUpdate); len(errs) > 0 {
		fieldErrors(w, errs)
		return
	}
	_, err = s.pool.Exec(r.Context(), `
		UPDATE staff_profiles SET
			employee_no=COALESCE(NULLIF($1,''), employee_no), national_id=COALESCE($2, national_id),
			department_id=COALESCE($3, department_id), designation=COALESCE($4, designation),
			date_of_joining=COALESCE($5, date_of_joining), basic_salary=COALESCE($6, basic_salary),
			kra_pin=COALESCE($7, kra_pin), nhif_no=COALESCE($8, nhif_no), nssf_no=COALESCE($9, nssf_no),
			bank_name=COALESCE($10, bank_name), bank_account_no=COALESCE($11, bank_account_no),
			is_active=COALESCE($12, is_active), tsc_no=COALESCE($14, tsc_no), gender=COALESCE($15, gender),
			date_of_birth=COALESCE($16::date, date_of_birth), employment_type=COALESCE($17, employment_type),
			highest_qualification=COALESCE($18, highest_qualification), bank_branch=COALESCE($19, bank_branch),
			emergency_contact_name=COALESCE($20, emergency_contact_name),
			emergency_contact_phone=COALESCE($21, emergency_contact_phone),
			emergency_contact_relation=COALESCE($22, emergency_contact_relation),
			residential_address=COALESCE($23, residential_address), updated_at=NOW()
		WHERE id = $13`,
		body.EmployeeNo, body.NationalID, body.DepartmentID, body.Designation, body.DateOfJoining,
		body.BasicSalary, body.KRAPin, body.NHIFNo, body.NSSFNo, body.BankName, body.BankAccountNo, body.IsActive, id,
		body.TSCNo, body.Gender, body.DateOfBirth, body.EmploymentType, body.HighestQualification, body.BankBranch,
		body.EmergencyContactName, body.EmergencyContactPhone, body.EmergencyContactRelation, body.ResidentialAddress)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	if _, err := s.pool.Exec(r.Context(), staffDetailsUpdateSQL,
		append([]interface{}{id}, body.staffDetailsUpdate.args()...)...); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	sp, err := s.loadOne(r, id)
	if err != nil {
		httpx.ErrNotFound(w, "staff member not found")
		return
	}
	httpx.JSON(w, http.StatusOK, sp)
}

func (s *StaffService) delete(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM staff_profiles WHERE id = $1`, id); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}

// ---------------------------------------------------------------- Departments

type DepartmentsService struct {
	pool *pgxpool.Pool
}

func NewDepartmentsService(pool *pgxpool.Pool) *DepartmentsService {
	return &DepartmentsService{pool: pool}
}

func (s *DepartmentsService) RegisterRoutes(r chi.Router) {
	r.Route("/departments/", func(r chi.Router) {
		r.Get("/", s.list)
		r.Group(func(r chi.Router) {
			r.Use(adminOnly())
			r.Post("/", s.create)
			r.Delete("/{id}/", s.delete)
		})
	})
}

type Department struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name" validate:"required"`
	Code        string    `json:"code" validate:"required"`
	Description string    `json:"description"`
}

func (s *DepartmentsService) list(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `SELECT id, name, code, description FROM departments ORDER BY name`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []Department{}
	for rows.Next() {
		var d Department
		if err := rows.Scan(&d.ID, &d.Name, &d.Code, &d.Description); err != nil {
			continue
		}
		out = append(out, d)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *DepartmentsService) create(w http.ResponseWriter, r *http.Request) {
	var body Department
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO departments (id, name, code, description, created_at, updated_at)
		VALUES ($1,$2,$3,$4,NOW(),NOW()) RETURNING id`,
		uuid.New(), body.Name, body.Code, body.Description).Scan(&body.ID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, body)
}

func (s *DepartmentsService) delete(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM departments WHERE id = $1`, id); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}
