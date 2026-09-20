package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/rpc"
	"github.com/stmaryskabete/lms/internal/shared/types"
)

type FingerprintProxy struct {
	client *rpc.FingerprintClient
	pool   *pgxpool.Pool
}

func NewFingerprintProxy(c *rpc.FingerprintClient, pool *pgxpool.Pool) *FingerprintProxy {
	return &FingerprintProxy{client: c, pool: pool}
}

func (p *FingerprintProxy) RegisterRoutes(r chi.Router) {
	r.Post("/fingerprint/devices/register/", p.requireAdmin(p.handleDeviceRegister))
	r.Post("/fingerprint/devices/{id}/heartbeat/", p.handleDeviceHeartbeat)
	r.Post("/fingerprint/scan/", p.handleScan)
	r.Get("/fingerprint/students/{student_id}/template/", p.requireAdminOrTeacher(p.handleGetTemplate))
	r.Post("/fingerprint/students/{student_id}/template/", p.requireAdminOrTeacher(p.handleStoreTemplate))
	r.Delete("/fingerprint/students/{student_id}/template/", p.requireAdminOrTeacher(p.handleDeleteTemplate))
	r.Get("/fingerprint/staff/{staff_id}/template/", p.requireAdmin(p.handleGetStaffTemplate))
	r.Post("/fingerprint/staff/{staff_id}/template/", p.requireAdmin(p.handleStoreStaffTemplate))
	r.Delete("/fingerprint/staff/{staff_id}/template/", p.requireAdmin(p.handleDeleteStaffTemplate))
	r.Get("/fingerprint/attendance/", p.requireAuthenticated(p.handleAttendance))
	r.Get("/attendance/daily-summary/", p.requireAuthenticated(p.handleDailySummary))
	r.Get("/attendance/class/{class_id}/roster/", p.requireAdminOrTeacher(p.handleClassRosterAttendance))
	r.Get("/attendance/students/{student_id}/range/", p.requireAuthenticated(p.handleStudentAttendanceRange))
	r.Get("/attendance/staff/{staff_id}/range/", p.requireAuthenticated(p.handleStaffAttendanceRange))
	r.Get("/attendance/staff/me/range/", p.requireAuthenticated(p.handleMyStaffAttendanceRange))
}

func (p *FingerprintProxy) requireAuthenticated(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Value("user_id").(uuid.UUID); !ok {
			httpx.ErrUnauthorized(w, "auth required")
			return
		}
		next(w, r)
	}
}

func (p *FingerprintProxy) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value("role").(types.Role)
		admin := false
		for _, a := range types.AdminRoles {
			if a == role {
				admin = true
				break
			}
		}
		if role == types.RoleSystemAdmin {
			admin = true
		}
		if !admin {
			httpx.ErrForbidden(w, "admin required")
			return
		}
		next(w, r)
	}
}

func (p *FingerprintProxy) requireAdminOrTeacher(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value("role").(types.Role)
		ok := false
		for _, a := range types.AdminRoles {
			if a == role {
				ok = true
				break
			}
		}
		if !ok {
			for _, a := range types.TeacherRoles {
				if a == role {
					ok = true
					break
				}
			}
		}
		if role == types.RoleSystemAdmin {
			ok = true
		}
		if !ok {
			httpx.ErrForbidden(w, "admin or teacher required")
			return
		}
		next(w, r)
	}
}

func (p *FingerprintProxy) handleDeviceRegister(w http.ResponseWriter, r *http.Request) {
	var body rpc.DeviceRegisterRequest
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	res, err := p.client.RegisterDevice(r.Context(), &body)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}

func (p *FingerprintProxy) handleDeviceHeartbeat(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	_ = httpx.BindAndValidate(r, &body)
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "delegated"})
}

func (p *FingerprintProxy) handleScan(w http.ResponseWriter, r *http.Request) {
	var scan rpc.ScanEvent
	if err := httpx.BindAndValidate(r, &scan); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	res, err := p.client.ProcessScan(r.Context(), &scan)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (p *FingerprintProxy) handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	sidStr := chi.URLParam(r, "student_id")
	sid, err := uuid.Parse(sidStr)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid student id")
		return
	}
	res, err := p.client.GetTemplate(r.Context(), sid)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (p *FingerprintProxy) handleStoreTemplate(w http.ResponseWriter, r *http.Request) {
	sidStr := chi.URLParam(r, "student_id")
	sid, err := uuid.Parse(sidStr)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid student id")
		return
	}
	uid, _ := r.Context().Value("user_id").(uuid.UUID)
	var body struct {
		TemplateB64 string    `json:"template_b64" validate:"required"`
		DeviceID    uuid.UUID `json:"device_id" validate:"required,uuid"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	req := &rpc.StoreTemplateRequest{
		StudentID:   sid,
		TemplateB64: body.TemplateB64,
		EnrolledBy:  uid,
		DeviceID:    body.DeviceID,
	}
	if err := p.client.StoreTemplate(r.Context(), req); err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (p *FingerprintProxy) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	sidStr := chi.URLParam(r, "student_id")
	sid, err := uuid.Parse(sidStr)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid student id")
		return
	}
	if err := p.client.DeleteTemplate(r.Context(), sid); err != nil {
		writeUpstreamError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (p *FingerprintProxy) handleAttendance(w http.ResponseWriter, r *http.Request) {
	q := &rpc.AttendanceQuery{
		Page:     parseInt(r.URL.Query().Get("page"), 1),
		PageSize: parseInt(r.URL.Query().Get("page_size"), 25),
	}
	if s := r.URL.Query().Get("student_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid student_id")
			return
		}
		q.StudentID = &id
	}
	if s := r.URL.Query().Get("scan_type"); s != "" {
		q.ScanType = &s
	}
	res, err := p.client.ListAttendance(r.Context(), q)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	if n <= 0 {
		return def
	}
	return n
}

func (p *FingerprintProxy) handleDailySummary(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	req := &rpc.DailyAttendanceSummaryRequest{Date: date}
	if c := r.URL.Query().Get("class_id"); c != "" {
		id, err := uuid.Parse(c)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid class_id")
			return
		}
		req.ClassID = &id
	}
	res, err := p.client.DailyAttendanceSummary(r.Context(), req)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (p *FingerprintProxy) handleClassRosterAttendance(w http.ResponseWriter, r *http.Request) {
	classIDStr := chi.URLParam(r, "class_id")
	classID, err := uuid.Parse(classIDStr)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid class_id")
		return
	}
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	res, err := p.client.ClassRosterAttendance(r.Context(), &rpc.ClassRosterAttendanceRequest{
		ClassID: classID,
		Date:    date,
	})
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"data": res,
	})
}

func (p *FingerprintProxy) handleStudentAttendanceRange(w http.ResponseWriter, r *http.Request) {
	studentIDStr := chi.URLParam(r, "student_id")
	studentID, err := uuid.Parse(studentIDStr)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid student_id")
		return
	}
	role, _ := r.Context().Value("role").(types.Role)
	userID, _ := r.Context().Value("user_id").(uuid.UUID)
	isAllowed := false
	for _, a := range types.AdminRoles {
		if a == role {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		for _, a := range types.TeacherRoles {
			if a == role {
				isAllowed = true
				break
			}
		}
	}
	if role == types.RoleSystemAdmin {
		isAllowed = true
	}
	if !isAllowed && role == types.RoleStudent {
		isAllowed = (userID == studentID)
	}
	if !isAllowed && role == types.RoleParent {
		ok, err := p.isGuardianOf(r.Context(), userID, studentID)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		isAllowed = ok
	}
	if !isAllowed {
		httpx.ErrForbidden(w, "insufficient permissions for this student")
		return
	}
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	if startDate == "" || endDate == "" {
		now := time.Now()
		endDate = now.Format("2006-01-02")
		startDate = now.AddDate(0, -1, 0).Format("2006-01-02")
	}
	res, err := p.client.StudentAttendanceRange(r.Context(), &rpc.StudentAttendanceRangeRequest{
		StudentID: studentID,
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (p *FingerprintProxy) handleStoreStaffTemplate(w http.ResponseWriter, r *http.Request) {
	sidStr := chi.URLParam(r, "staff_id")
	sid, err := uuid.Parse(sidStr)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid staff id")
		return
	}
	uid, _ := r.Context().Value("user_id").(uuid.UUID)
	var body struct {
		TemplateB64 string    `json:"template_b64" validate:"required"`
		DeviceID    uuid.UUID `json:"device_id" validate:"required,uuid"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	req := &rpc.StoreStaffTemplateRequest{
		StaffID:     sid,
		TemplateB64: body.TemplateB64,
		EnrolledBy:  uid,
		DeviceID:    body.DeviceID,
	}
	if err := p.client.StoreStaffTemplate(r.Context(), req); err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (p *FingerprintProxy) handleGetStaffTemplate(w http.ResponseWriter, r *http.Request) {
	sidStr := chi.URLParam(r, "staff_id")
	sid, err := uuid.Parse(sidStr)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid staff id")
		return
	}
	res, err := p.client.GetStaffTemplate(r.Context(), sid)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (p *FingerprintProxy) handleDeleteStaffTemplate(w http.ResponseWriter, r *http.Request) {
	sidStr := chi.URLParam(r, "staff_id")
	sid, err := uuid.Parse(sidStr)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid staff id")
		return
	}
	if err := p.client.DeleteStaffTemplate(r.Context(), sid); err != nil {
		writeUpstreamError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (p *FingerprintProxy) handleStaffAttendanceRange(w http.ResponseWriter, r *http.Request) {
	staffIDStr := chi.URLParam(r, "staff_id")
	staffID, err := uuid.Parse(staffIDStr)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid staff_id")
		return
	}
	role, _ := r.Context().Value("role").(types.Role)
	userID, _ := r.Context().Value("user_id").(uuid.UUID)
	isAllowed := false
	for _, a := range types.AdminRoles {
		if a == role {
			isAllowed = true
			break
		}
	}
	if role == types.RoleSystemAdmin {
		isAllowed = true
	}
	if !isAllowed {
		ok, err := p.isSelfStaff(r.Context(), userID, staffID)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		isAllowed = ok
	}
	if !isAllowed {
		httpx.ErrForbidden(w, "insufficient permissions for this staff member")
		return
	}
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	if startDate == "" || endDate == "" {
		now := time.Now()
		endDate = now.Format("2006-01-02")
		startDate = now.AddDate(0, -1, 0).Format("2006-01-02")
	}
	res, err := p.client.StaffAttendanceRange(r.Context(), &rpc.StaffAttendanceRangeRequest{
		StaffID:   staffID,
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (p *FingerprintProxy) handleMyStaffAttendanceRange(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value("user_id").(uuid.UUID)
	staffID, err := p.resolveStaffID(r.Context(), userID)
	if err != nil {
		httpx.ErrNotFound(w, "no staff profile for this account")
		return
	}
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	if startDate == "" || endDate == "" {
		now := time.Now()
		endDate = now.Format("2006-01-02")
		startDate = now.AddDate(0, -1, 0).Format("2006-01-02")
	}
	res, err := p.client.StaffAttendanceRange(r.Context(), &rpc.StaffAttendanceRangeRequest{
		StaffID:   staffID,
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (p *FingerprintProxy) resolveStaffID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	if p.pool == nil {
		return uuid.Nil, errors.New("no db pool")
	}
	var id uuid.UUID
	err := p.pool.QueryRow(ctx, `SELECT id FROM staff_profiles WHERE user_id = $1`, userID).Scan(&id)
	return id, err
}

func (p *FingerprintProxy) isSelfStaff(ctx context.Context, userID uuid.UUID, staffID uuid.UUID) (bool, error) {
	if p.pool == nil {
		return false, nil
	}
	var id uuid.UUID
	err := p.pool.QueryRow(ctx, `
		SELECT id FROM staff_profiles WHERE id = $1 AND user_id = $2`, staffID, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (p *FingerprintProxy) isGuardianOf(ctx context.Context, guardianUserID uuid.UUID, studentID uuid.UUID) (bool, error) {
	if p.pool == nil {
		return false, nil
	}
	var gid uuid.UUID
	err := p.pool.QueryRow(ctx, `
		SELECT g.id
		FROM guardians g
		JOIN student_guardians sg ON sg.guardian_id = g.id
		WHERE g.user_id = $1 AND sg.student_id = $2
		LIMIT 1`, guardianUserID, studentID).Scan(&gid)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}
