package fingerprint

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/rpc"
)

type Service struct {
	devices    *DeviceStore
	templates  *TemplateStore
	scans      *ScannerStore
	nairobiLoc *time.Location
	pool       *pgxpool.Pool
	notifCli   *rpc.NotificationClient
}

func NewService(devices *DeviceStore, templates *TemplateStore, scans *ScannerStore, loc *time.Location, pool *pgxpool.Pool, notifCli *rpc.NotificationClient) *Service {
	return &Service{devices: devices, templates: templates, scans: scans, nairobiLoc: loc, pool: pool, notifCli: notifCli}
}

func (svc *Service) RegisterPublicRoutes(r chi.Router) {
	r.Post("/devices/register", svc.handleDeviceRegister)
	r.Post("/devices/{id}/heartbeat", svc.handleDeviceHeartbeat)
	r.Post("/scan", svc.handleScan)
}

func (svc *Service) RegisterInternalRoutes(r chi.Router) {
	r.Post("/devices/register", svc.handleInternalDeviceRegister)
	r.Get("/students/{studentID}/template", svc.handleGetTemplate)
	r.Post("/students/{studentID}/template", svc.handleStoreTemplate)
	r.Delete("/students/{studentID}/template", svc.handleDeleteTemplate)
	r.Get("/staff/{staffID}/template", svc.handleGetStaffTemplate)
	r.Post("/staff/{staffID}/template", svc.handleStoreStaffTemplate)
	r.Delete("/staff/{staffID}/template", svc.handleDeleteStaffTemplate)
	r.Post("/scans/process", svc.handleInternalScan)
	r.Post("/attendance/list", svc.handleAttendanceList)
	r.Post("/attendance/summary/daily", svc.handleAttendanceDailySummary)
	r.Post("/attendance/class/roster", svc.handleClassRosterAttendance)
	r.Post("/attendance/student/range", svc.handleStudentAttendanceRange)
	r.Post("/attendance/staff/range", svc.handleStaffAttendanceRange)
}

func (svc *Service) handleDeviceRegister(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name         string `json:"name" validate:"required"`
		Location     string `json:"location" validate:"required"`
		DeviceType   string `json:"device_type" validate:"required"`
		SharedSecret string `json:"shared_secret" validate:"required,min=16"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.JSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error:   "validation_failed",
			Details: httpx.ValidationErrors(err),
			Message: err.Error(),
		})
		return
	}
	d, err := svc.devices.Register(r.Context(), body.Name, body.Location, body.DeviceType, body.SharedSecret)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{
		"device_id":   d.ID,
		"name":        d.Name,
		"location":    d.Location,
		"device_type": d.DeviceType,
		"is_active":   d.IsActive,
		"created_at":  d.CreatedAt,
	})
}

func (svc *Service) handleInternalDeviceRegister(w http.ResponseWriter, r *http.Request) {
	svc.handleDeviceRegister(w, r)
}

func (svc *Service) handleDeviceHeartbeat(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	deviceID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid device id")
		return
	}
	var body struct {
		Timestamp   int64  `json:"timestamp" validate:"required"`
		HMAC        string `json:"hmac" validate:"required"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	d, err := svc.devices.GetByID(r.Context(), deviceID)
	if err != nil {
		httpx.ErrNotFound(w, "device not found")
		return
	}
	if !IsTimestampFresh(body.Timestamp, time.Now(), 5*time.Minute) {
		httpx.ErrUnauthorized(w, "timestamp outside allowed window")
		return
	}
	msg := BuildHeartbeatMessage(idStr, body.Timestamp)
	if !VerifyDeviceHMAC(msg, body.HMAC, d.SecretHash) {
		httpx.ErrUnauthorized(w, "invalid hmac")
		return
	}
	now := time.Now()
	if err := svc.devices.Heartbeat(r.Context(), deviceID, now); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"status":       "ok",
		"last_seen_at": now,
		"is_active":    true,
	})
}

func (svc *Service) handleScan(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DeviceID             string   `json:"device_id" validate:"required,uuid"`
		StudentID            *string  `json:"student_id,omitempty"`
		StaffID              *string  `json:"staff_id,omitempty"`
		TemplateID           *string  `json:"template_id,omitempty"`
		FingerprintMatchScore *float64 `json:"fingerprint_match_score,omitempty"`
		ScanType             string   `json:"scan_type" validate:"required,oneof=entry exit class_checkin"`
		Timestamp            int64    `json:"timestamp" validate:"required"`
		HMAC                 string   `json:"hmac" validate:"required"`
		RawScore             *float64 `json:"raw_score,omitempty"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.JSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error:   "validation_failed",
			Details: httpx.ValidationErrors(err),
		})
		return
	}
	deviceID, _ := uuid.Parse(body.DeviceID)
	d, err := svc.devices.GetByID(r.Context(), deviceID)
	if err != nil {
		httpx.ErrUnauthorized(w, "unknown device")
		return
	}
	if !d.IsActive {
		httpx.ErrForbidden(w, "device not active")
		return
	}
	ts := body.Timestamp
	if !IsTimestampFresh(ts, time.Now(), 5*time.Minute) {
		httpx.ErrUnauthorized(w, "timestamp outside allowed window")
		return
	}
	studentIDRaw, staffIDRaw := "", ""
	if body.StudentID != nil {
		studentIDRaw = *body.StudentID
	}
	if body.StaffID != nil {
		staffIDRaw = *body.StaffID
	}
	msg := BuildScanMessage(body.DeviceID, body.ScanType, studentIDRaw, staffIDRaw, ts)
	if !VerifyDeviceHMAC(msg, body.HMAC, d.SecretHash) {
		httpx.ErrUnauthorized(w, "invalid hmac")
		return
	}
	rec := &ScanRecord{
		DeviceID:             deviceID,
		ScanType:             ScanType(body.ScanType),
		Timestamp:            time.Unix(ts, 0).In(svc.nairobiLoc),
		FingerprintMatchScore: body.FingerprintMatchScore,
		RawScore:             body.RawScore,
	}
	if body.StudentID != nil {
		uid, err := uuid.Parse(*body.StudentID)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid student_id")
			return
		}
		rec.StudentID = &uid
	}
	if body.StaffID != nil {
		uid, err := uuid.Parse(*body.StaffID)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid staff_id")
			return
		}
		rec.StaffID = &uid
	}
	if body.TemplateID != nil {
		tid, err := uuid.Parse(*body.TemplateID)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid template_id")
			return
		}
		rec.TemplateID = &tid
	}
	attID, err := svc.scans.ProcessScan(r.Context(), rec)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	resp := map[string]interface{}{
		"scan_id":   rec.ID,
		"duplicate": rec.IsDuplicate,
		"scan_type": rec.ScanType,
		"timestamp": rec.Timestamp,
	}
	if rec.StudentID != nil {
		resp["student_id"] = *rec.StudentID
	}
	if rec.StaffID != nil {
		resp["staff_id"] = *rec.StaffID
	}
	if attID != nil {
		resp["attendance_id"] = *attID
	}
	if !rec.IsDuplicate && rec.StudentID != nil {
		go svc.dispatchAttendanceAlert(r.Context(), *rec.StudentID, rec.ScanType, rec.Timestamp, rec.SubjectID)
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (svc *Service) handleInternalScan(w http.ResponseWriter, r *http.Request) {
	var rec ScanRecord
	if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if rec.Timestamp.IsZero() {
		rec.Timestamp = time.Now().In(svc.nairobiLoc)
	} else {
		rec.Timestamp = rec.Timestamp.In(svc.nairobiLoc)
	}
	attID, err := svc.scans.ProcessScan(r.Context(), &rec)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	sid := uuid.Nil
	if rec.StudentID != nil {
		sid = *rec.StudentID
	}
	stid := uuid.Nil
	if rec.StaffID != nil {
		stid = *rec.StaffID
	}
	if !rec.IsDuplicate && rec.StudentID != nil {
		go svc.dispatchAttendanceAlert(r.Context(), *rec.StudentID, rec.ScanType, rec.Timestamp, rec.SubjectID)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"scan_id":       rec.ID,
		"duplicate":     rec.IsDuplicate,
		"attendance_id": attID,
		"student_id":    sid,
		"staff_id":      stid,
	})
}

func (svc *Service) handleStoreTemplate(w http.ResponseWriter, r *http.Request) {
	studentID, err := uuid.Parse(chi.URLParam(r, "studentID"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid student id")
		return
	}
	var body struct {
		TemplateB64 string    `json:"template_b64" validate:"required"`
		EnrolledBy  uuid.UUID `json:"enrolled_by" validate:"required,uuid"`
		DeviceID    uuid.UUID `json:"device_id" validate:"required,uuid"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if err := svc.templates.Store(r.Context(), studentID, body.TemplateB64, body.EnrolledBy, body.DeviceID); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (svc *Service) handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	studentID, err := uuid.Parse(chi.URLParam(r, "studentID"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid student id")
		return
	}
	b64, ver, enrolledAt, deviceID, active, err := svc.templates.Export(r.Context(), studentID)
	if err != nil {
		httpx.ErrNotFound(w, "template not found")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"template_b64":    b64,
		"template_version": ver,
		"enrolled_at":     enrolledAt,
		"device_id":       deviceID,
		"is_active":       active,
	})
}

func (svc *Service) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	studentID, err := uuid.Parse(chi.URLParam(r, "studentID"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid student id")
		return
	}
	if err := svc.templates.Delete(r.Context(), studentID); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (svc *Service) handleStoreStaffTemplate(w http.ResponseWriter, r *http.Request) {
	staffID, err := uuid.Parse(chi.URLParam(r, "staffID"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid staff id")
		return
	}
	var body struct {
		TemplateB64 string    `json:"template_b64" validate:"required"`
		EnrolledBy  uuid.UUID `json:"enrolled_by" validate:"required,uuid"`
		DeviceID    uuid.UUID `json:"device_id" validate:"required,uuid"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if err := svc.templates.StoreStaff(r.Context(), staffID, body.TemplateB64, body.EnrolledBy, body.DeviceID); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (svc *Service) handleGetStaffTemplate(w http.ResponseWriter, r *http.Request) {
	staffID, err := uuid.Parse(chi.URLParam(r, "staffID"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid staff id")
		return
	}
	b64, ver, enrolledAt, deviceID, active, err := svc.templates.ExportStaff(r.Context(), staffID)
	if err != nil {
		httpx.ErrNotFound(w, "template not found")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"template_b64":    b64,
		"template_version": ver,
		"enrolled_at":     enrolledAt,
		"device_id":       deviceID,
		"is_active":       active,
	})
}

func (svc *Service) handleDeleteStaffTemplate(w http.ResponseWriter, r *http.Request) {
	staffID, err := uuid.Parse(chi.URLParam(r, "staffID"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid staff id")
		return
	}
	if err := svc.templates.DeleteStaff(r.Context(), staffID); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (svc *Service) handleStaffAttendanceRange(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StaffID   string `json:"staff_id" validate:"required,uuid"`
		StartDate string `json:"start_date" validate:"required"`
		EndDate   string `json:"end_date" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	stid, err := uuid.Parse(body.StaffID)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid staff_id")
		return
	}
	sd, err := time.ParseInLocation("2006-01-02", body.StartDate, svc.nairobiLoc)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid start_date")
		return
	}
	ed, err := time.ParseInLocation("2006-01-02", body.EndDate, svc.nairobiLoc)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid end_date")
		return
	}
	events, present, late, absent, expected, pct, err := svc.scans.StaffAttendanceRange(r.Context(), stid, sd, ed)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"events":         events,
		"present_days":   present,
		"late_days":      late,
		"absent_days":    absent,
		"expected_days":  expected,
		"attendance_pct": pct,
	})
}

func (svc *Service) handleAttendanceList(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StudentID *string  `json:"student_id,omitempty"`
		StartDate *string  `json:"start_date,omitempty"`
		EndDate   *string  `json:"end_date,omitempty"`
		ScanType  *string  `json:"scan_type,omitempty"`
		Page      int      `json:"page"`
		PageSize  int      `json:"page_size"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	var sid *uuid.UUID
	if body.StudentID != nil {
		id, err := uuid.Parse(*body.StudentID)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid student_id")
			return
		}
		sid = &id
	}
	var sd, ed *time.Time
	layout := "2006-01-02"
	if body.StartDate != nil {
		t, err := time.ParseInLocation(layout, *body.StartDate, svc.nairobiLoc)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid start_date (expect YYYY-MM-DD)")
			return
		}
		sd = &t
	}
	if body.EndDate != nil {
		t, err := time.ParseInLocation(layout, *body.EndDate, svc.nairobiLoc)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid end_date (expect YYYY-MM-DD)")
			return
		}
		ed = &t
	}
	var st *ScanType
	if body.ScanType != nil {
		switch *body.ScanType {
		case "entry", "exit", "class_checkin":
			v := ScanType(*body.ScanType)
			st = &v
		default:
			httpx.ErrBadRequest(w, "invalid scan_type")
			return
		}
	}
	rows, total, err := svc.scans.ListAttendance(r.Context(), sid, sd, ed, st, body.Page, body.PageSize)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if body.Page < 1 {
		body.Page = 1
	}
	if body.PageSize < 1 {
		body.PageSize = 25
	}
	tp := int((total + int64(body.PageSize) - 1) / int64(body.PageSize))
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"data":        rows,
		"page":        body.Page,
		"page_size":   body.PageSize,
		"total_count": total,
		"total_pages": tp,
	})
}

func (svc *Service) handleAttendanceDailySummary(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClassID *string `json:"class_id,omitempty"`
		Date    string  `json:"date" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	t, err := time.ParseInLocation("2006-01-02", body.Date, svc.nairobiLoc)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid date (expect YYYY-MM-DD)")
		return
	}
	var cid *uuid.UUID
	if body.ClassID != nil && *body.ClassID != "" {
		id, err := uuid.Parse(*body.ClassID)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid class_id")
			return
		}
		cid = &id
	}
	sum, err := svc.scans.DailyClassSummary(r.Context(), cid, t)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, sum)
}

func (svc *Service) handleClassRosterAttendance(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClassID string `json:"class_id" validate:"required,uuid"`
		Date    string `json:"date" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	classID, err := uuid.Parse(body.ClassID)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid class_id")
		return
	}
	t, err := time.ParseInLocation("2006-01-02", body.Date, svc.nairobiLoc)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid date (expect YYYY-MM-DD)")
		return
	}
	rows, err := svc.scans.ClassRosterAttendance(r.Context(), classID, t)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, rows)
}

func (svc *Service) handleStudentAttendanceRange(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StudentID string `json:"student_id" validate:"required,uuid"`
		StartDate string `json:"start_date" validate:"required"`
		EndDate   string `json:"end_date" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	sid, err := uuid.Parse(body.StudentID)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid student_id")
		return
	}
	sd, err := time.ParseInLocation("2006-01-02", body.StartDate, svc.nairobiLoc)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid start_date")
		return
	}
	ed, err := time.ParseInLocation("2006-01-02", body.EndDate, svc.nairobiLoc)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid end_date")
		return
	}
	events, present, late, absent, expected, pct, err := svc.scans.StudentAttendanceRange(r.Context(), sid, sd, ed)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"events":        events,
		"present_days":  present,
		"late_days":     late,
		"absent_days":   absent,
		"expected_days": expected,
		"attendance_pct": pct,
	})
}

func (svc *Service) dispatchAttendanceAlert(ctx context.Context, studentID uuid.UUID, scanType ScanType, timestamp time.Time, subjectID *uuid.UUID) {
	if svc.pool == nil || svc.notifCli == nil {
		return
	}
	type studentRow struct {
		StudentFirst string
		StudentLast  string
		ClassName    *string
		SubjectName  *string
	}
	var sr studentRow
	sq := `
		SELECT s.first_name, s.last_name,
		       CASE WHEN cg.id IS NOT NULL THEN cg.grade || '-' || st.name END AS class_name,
		       sub.name
		FROM students s
		LEFT JOIN class_groups cg ON cg.id = s.current_class_id
		LEFT JOIN streams st ON st.id = cg.stream_id
		LEFT JOIN subjects sub ON sub.id = $2
		WHERE s.id = $1`
	if err := svc.pool.QueryRow(ctx, sq, studentID, subjectID).Scan(
		&sr.StudentFirst, &sr.StudentLast, &sr.ClassName, &sr.SubjectName); err != nil {
		log.Printf("[fingerprint-attendance-alert] query student: %v", err)
		return
	}

	cutoff := time.Date(timestamp.Year(), timestamp.Month(), timestamp.Day(), 8, 15, 0, 0, svc.nairobiLoc)
	status := "present at school"
	switch scanType {
	case ScanEntry:
		if timestamp.After(cutoff) {
			status = "late arrival"
		} else {
			status = "arrived at school"
		}
	case ScanExit:
		status = "signed out of school"
	case ScanClassCheckin:
		if sr.SubjectName != nil {
			status = fmt.Sprintf("checked in to %s class", *sr.SubjectName)
		} else {
			status = "checked in to class"
		}
	}
	timeStr := timestamp.In(svc.nairobiLoc).Format("03:04 PM")
	if sr.ClassName != nil {
		status = fmt.Sprintf("%s (Class %s)", status, *sr.ClassName)
	}

	grows, err := svc.pool.Query(ctx, `
		SELECT sg.guardian_id, sg.is_primary
		FROM student_guardians sg
		WHERE sg.student_id = $1
		ORDER BY sg.is_primary DESC, sg.contact_priority ASC`, studentID)
	if err != nil {
		log.Printf("[fingerprint-attendance-alert] query guardians: %v", err)
		return
	}
	defer grows.Close()

	got := false
	for grows.Next() {
		var gid uuid.UUID
		var isPrimary bool
		if err := grows.Scan(&gid, &isPrimary); err != nil {
			continue
		}
		got = true
		td := map[string]string{
			"student_first": sr.StudentFirst,
			"student_last":  sr.StudentLast,
			"status":        status,
			"time":          timeStr,
		}
		req := &rpc.GuardianNotification{
			GuardianID:   gid,
			StudentID:    studentID,
			TemplateKey:  "attendance_alert",
			TemplateData: td,
			SMS:          true,
			Email:        false,
		}
		if err := svc.notifCli.DispatchGuardianNotification(ctx, req); err != nil {
			log.Printf("[fingerprint-attendance-alert] dispatch guardian=%s: %v", gid, err)
		}
		if isPrimary {
			break
		}
	}
	if !got {
		log.Printf("[fingerprint-attendance-alert] no guardians for student=%s", studentID)
	}
	_ = fmt.Sprintf("")
}

func (svc *Service) Health(ctx context.Context) error {
	if svc.devices == nil || svc.templates == nil || svc.scans == nil {
		panic("fingerprint service not initialized")
	}
	return nil
}
