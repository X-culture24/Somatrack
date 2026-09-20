package fingerprint

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ScanType string

const (
	ScanEntry       ScanType = "entry"
	ScanExit        ScanType = "exit"
	ScanClassCheckin ScanType = "class_checkin"
)

type ScanRecord struct {
	ID                   uuid.UUID  `json:"id"`
	DeviceID             uuid.UUID  `json:"device_id"`
	StudentID            *uuid.UUID `json:"student_id,omitempty"`
	StaffID              *uuid.UUID `json:"staff_id,omitempty"`
	TemplateID           *uuid.UUID `json:"template_id,omitempty"`
	FingerprintMatchScore *float64   `json:"fingerprint_match_score,omitempty"`
	ScanType             ScanType   `json:"scan_type"`
	Timestamp            time.Time  `json:"timestamp"`
	RawScore             *float64   `json:"raw_score,omitempty"`
	IsDuplicate          bool       `json:"is_duplicate"`
	CreatedAt            time.Time  `json:"created_at"`
	SubjectID            *uuid.UUID `json:"subject_id,omitempty"`
	ClassID              *uuid.UUID `json:"class_id,omitempty"`
	Remarks              string     `json:"remarks,omitempty"`
}

type AttendanceRecord struct {
	ID         uuid.UUID  `json:"id"`
	StudentID  uuid.UUID  `json:"student_id"`
	Date       string     `json:"date"`
	ScanType   ScanType   `json:"scan_type"`
	Timestamp  time.Time  `json:"timestamp"`
	DeviceID   uuid.UUID  `json:"device_id"`
	RawScore   *float64   `json:"raw_score,omitempty"`
	MatchScore *float64   `json:"match_score,omitempty"`
}

const dedupWindow = 5 * time.Minute

type ScannerStore struct {
	pool       *pgxpool.Pool
	nairobiLoc *time.Location
}

func NewScannerStore(pool *pgxpool.Pool, nairobiLoc *time.Location) *ScannerStore {
	if nairobiLoc == nil {
		nairobiLoc = time.FixedZone("EAT", 3*3600)
	}
	return &ScannerStore{pool: pool, nairobiLoc: nairobiLoc}
}

func (s *ScannerStore) ProcessScan(ctx context.Context, rec *ScanRecord) (attendanceID *uuid.UUID, err error) {
	if rec.StudentID == nil && rec.StaffID == nil {
		return nil, errors.New("student_id or staff_id required")
	}
	if rec.StudentID != nil && rec.StaffID != nil {
		return nil, errors.New("provide only one of student_id or staff_id")
	}
	if rec.StaffID != nil {
		return s.processStaffScan(ctx, rec)
	}
	return s.processStudentScan(ctx, rec)
}

func (s *ScannerStore) processStudentScan(ctx context.Context, rec *ScanRecord) (attendanceID *uuid.UUID, err error) {
	date := rec.Timestamp.Truncate(24 * time.Hour)
	dupStart := rec.Timestamp.Add(-dedupWindow)
	dupEnd := rec.Timestamp.Add(dedupWindow)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var existingID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id FROM fingerprint_scans
		WHERE student_id = $1
		  AND scan_type = $2
		  AND date_trunc('day', timestamp) = $3
		  AND timestamp BETWEEN $4 AND $5
		LIMIT 1`, *rec.StudentID, rec.ScanType, date, dupStart, dupEnd).Scan(&existingID)
	if err == nil {
		rec.ID = existingID
		rec.IsDuplicate = true
		return nil, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	rec.ID = uuid.New()
	rec.CreatedAt = time.Now()
	rec.IsDuplicate = false

	if rec.ClassID == nil || rec.SubjectID == nil {
		var devClass, devSubject *uuid.UUID
		_ = tx.QueryRow(ctx, `
			SELECT class_id, subject_id FROM fingerprint_devices WHERE id = $1`, rec.DeviceID).Scan(&devClass, &devSubject)
		if rec.ClassID == nil {
			rec.ClassID = devClass
		}
		if rec.SubjectID == nil {
			rec.SubjectID = devSubject
		}
	}

	const insScan = `
		INSERT INTO fingerprint_scans (id, device_id, student_id, template_id, fingerprint_match_score, scan_type, timestamp, raw_score, is_duplicate, created_at, subject_id, class_id, remarks)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, false, $9, $10, $11, $12)`
	if _, err := tx.Exec(ctx, insScan,
		rec.ID, rec.DeviceID, rec.StudentID, rec.TemplateID, rec.FingerprintMatchScore,
		rec.ScanType, rec.Timestamp, rec.RawScore, rec.CreatedAt, rec.SubjectID, rec.ClassID, rec.Remarks); err != nil {
		return nil, fmt.Errorf("insert scan: %w", err)
	}

	var score *float64
	if rec.FingerprintMatchScore != nil {
		score = rec.FingerprintMatchScore
	} else if rec.RawScore != nil {
		score = rec.RawScore
	}

	var attID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO fingerprint_attendance (student_id, date, scan_type, timestamp, device_id, raw_score, match_score, scan_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (student_id, date, scan_type) DO UPDATE SET
			timestamp = EXCLUDED.timestamp,
			device_id = EXCLUDED.device_id,
			raw_score = COALESCE(EXCLUDED.raw_score, fingerprint_attendance.raw_score),
			match_score = COALESCE(EXCLUDED.match_score, fingerprint_attendance.match_score)
		RETURNING id`,
		*rec.StudentID, date, rec.ScanType, rec.Timestamp, rec.DeviceID, rec.RawScore, score, rec.ID,
	).Scan(&attID)
	if err != nil {
		return nil, fmt.Errorf("upsert attendance: %w", err)
	}

	if err := s.mirrorAcademicsAttendance(ctx, tx, rec); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &attID, nil
}

func (s *ScannerStore) processStaffScan(ctx context.Context, rec *ScanRecord) (attendanceID *uuid.UUID, err error) {
	if rec.ScanType != ScanEntry && rec.ScanType != ScanExit {
		return nil, errors.New("staff scans only support entry or exit")
	}
	date := rec.Timestamp.Truncate(24 * time.Hour)
	dupStart := rec.Timestamp.Add(-dedupWindow)
	dupEnd := rec.Timestamp.Add(dedupWindow)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var existingID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id FROM fingerprint_scans
		WHERE staff_id = $1
		  AND scan_type = $2
		  AND date_trunc('day', timestamp) = $3
		  AND timestamp BETWEEN $4 AND $5
		LIMIT 1`, *rec.StaffID, rec.ScanType, date, dupStart, dupEnd).Scan(&existingID)
	if err == nil {
		rec.ID = existingID
		rec.IsDuplicate = true
		return nil, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	rec.ID = uuid.New()
	rec.CreatedAt = time.Now()
	rec.IsDuplicate = false

	const insScan = `
		INSERT INTO fingerprint_scans (id, device_id, staff_id, template_id, fingerprint_match_score, scan_type, timestamp, raw_score, is_duplicate, created_at, remarks)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, false, $9, $10)`
	if _, err := tx.Exec(ctx, insScan,
		rec.ID, rec.DeviceID, rec.StaffID, rec.TemplateID, rec.FingerprintMatchScore,
		rec.ScanType, rec.Timestamp, rec.RawScore, rec.CreatedAt, rec.Remarks); err != nil {
		return nil, fmt.Errorf("insert staff scan: %w", err)
	}

	var score *float64
	if rec.FingerprintMatchScore != nil {
		score = rec.FingerprintMatchScore
	} else if rec.RawScore != nil {
		score = rec.RawScore
	}

	var attID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO staff_attendance (staff_id, date, scan_type, timestamp, device_id, raw_score, match_score, scan_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (staff_id, date, scan_type) DO UPDATE SET
			timestamp = EXCLUDED.timestamp,
			device_id = EXCLUDED.device_id,
			raw_score = COALESCE(EXCLUDED.raw_score, staff_attendance.raw_score),
			match_score = COALESCE(EXCLUDED.match_score, staff_attendance.match_score)
		RETURNING id`,
		*rec.StaffID, date, rec.ScanType, rec.Timestamp, rec.DeviceID, rec.RawScore, score, rec.ID,
	).Scan(&attID)
	if err != nil {
		return nil, fmt.Errorf("upsert staff attendance: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &attID, nil
}

func (s *ScannerStore) mirrorAcademicsAttendance(ctx context.Context, tx pgx.Tx, rec *ScanRecord) error {
	dateOnly := rec.Timestamp.Truncate(24 * time.Hour)
	timeOnly := rec.Timestamp.In(s.nairobiLoc)
	hourMin := fmt.Sprintf("%02d:%02d:00", timeOnly.Hour(), timeOnly.Minute())

	classID := rec.ClassID
	if classID == nil {
		var cid *uuid.UUID
		_ = tx.QueryRow(ctx, `SELECT current_class_id FROM students WHERE id = $1`, *rec.StudentID).Scan(&cid)
		classID = cid
	}

	switch rec.ScanType {
	case ScanEntry:
		lateCutoff := time.Date(dateOnly.Year(), dateOnly.Month(), dateOnly.Day(), 8, 15, 0, 0, s.nairobiLoc)
		status := "present"
		if rec.Timestamp.After(lateCutoff) {
			status = "late"
		}
		var existingID uuid.UUID
		err := tx.QueryRow(ctx, `
			SELECT id FROM attendance_records
			WHERE student_id = $1 AND date = $2 AND subject_id IS NULL
			LIMIT 1`, *rec.StudentID, dateOnly).Scan(&existingID)
		if err == nil {
			_, err = tx.Exec(ctx, `
				UPDATE attendance_records SET
					class_id = COALESCE($1, class_id),
					status = CASE WHEN status IN ('present','late') THEN status ELSE $2 END,
					time_in = COALESCE(time_in, $3::time)
				WHERE id = $4`, classID, status, hourMin, existingID)
		} else if errors.Is(err, pgx.ErrNoRows) {
			_, err = tx.Exec(ctx, `
				INSERT INTO attendance_records (student_id, class_id, date, status, time_in, remarks)
				VALUES ($1, $2, $3, $4, $5::time, 'fingerprint entry scan')`,
				*rec.StudentID, classID, dateOnly, status, hourMin)
		}
		if err != nil {
			return fmt.Errorf("upsert daily attendance entry: %w", err)
		}

	case ScanClassCheckin:
		subjectID := rec.SubjectID
		classCheckinCutoff := time.Date(dateOnly.Year(), dateOnly.Month(), dateOnly.Day(), 0, 0, 0, 0, s.nairobiLoc)
		_ = classCheckinCutoff
		status := "present"
		remarks := "fingerprint class_checkin"
		if rec.Remarks != "" {
			remarks = rec.Remarks
		}
		if subjectID != nil {
			_, err := tx.Exec(ctx, `
				INSERT INTO attendance_records (student_id, class_id, subject_id, date, status, time_in, remarks)
				VALUES ($1, $2, $3, $4, $5, $6::time, $7)
				ON CONFLICT (student_id, date, subject_id) DO UPDATE SET
					class_id = COALESCE(EXCLUDED.class_id, attendance_records.class_id),
					status = CASE WHEN attendance_records.status IN ('present','late')
					              THEN attendance_records.status ELSE EXCLUDED.status END,
					time_in = COALESCE(attendance_records.time_in, EXCLUDED.time_in),
					remarks = COALESCE(NULLIF(attendance_records.remarks,''), EXCLUDED.remarks)`,
				*rec.StudentID, classID, subjectID, dateOnly, status, hourMin, remarks)
			if err != nil {
				return fmt.Errorf("upsert subject attendance: %w", err)
			}
		} else {
			var existingID uuid.UUID
			err := tx.QueryRow(ctx, `
				SELECT id FROM attendance_records
				WHERE student_id = $1 AND date = $2 AND subject_id IS NULL
				LIMIT 1`, *rec.StudentID, dateOnly).Scan(&existingID)
			if err == nil {
				_, err = tx.Exec(ctx, `
					UPDATE attendance_records SET
						class_id = COALESCE($1, class_id),
						status = CASE WHEN status IN ('present','late') THEN status ELSE $2 END,
						time_in = COALESCE(time_in, $3::time)
					WHERE id = $4`, classID, status, hourMin, existingID)
			} else if errors.Is(err, pgx.ErrNoRows) {
				_, err = tx.Exec(ctx, `
					INSERT INTO attendance_records (student_id, class_id, date, status, time_in, remarks)
					VALUES ($1, $2, $3, $4, $5::time, $6)`,
					*rec.StudentID, classID, dateOnly, status, hourMin, remarks)
			}
			if err != nil {
				return fmt.Errorf("upsert daily attendance class_checkin: %w", err)
			}
		}

	case ScanExit:
		_, err := tx.Exec(ctx, `
			UPDATE attendance_records
			SET time_out = $1::time
			WHERE student_id = $2 AND date = $3 AND subject_id IS NULL`,
			hourMin, *rec.StudentID, dateOnly)
		if err != nil {
			return fmt.Errorf("update daily attendance exit: %w", err)
		}
	}
	return nil
}

func (s *ScannerStore) ListAttendance(ctx context.Context, studentID *uuid.UUID, startDate, endDate *time.Time, scanType *ScanType, page, pageSize int) ([]AttendanceRecord, int64, error) {
	where := []string{"true"}
	args := []interface{}{}
	argIdx := 1
	if studentID != nil {
		where = append(where, fmt.Sprintf("student_id = $%d", argIdx))
		args = append(args, *studentID)
		argIdx++
	}
	if startDate != nil {
		where = append(where, fmt.Sprintf("date >= $%d", argIdx))
		args = append(args, startDate.Truncate(24*time.Hour))
		argIdx++
	}
	if endDate != nil {
		where = append(where, fmt.Sprintf("date <= $%d", argIdx))
		args = append(args, endDate.Truncate(24*time.Hour))
		argIdx++
	}
	if scanType != nil {
		where = append(where, fmt.Sprintf("scan_type = $%d", argIdx))
		args = append(args, *scanType)
		argIdx++
	}
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM fingerprint_attendance WHERE %s`, joinAnd(where))
	var total int64
	if err := s.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 25
	}
	offset := (page - 1) * pageSize
	listQ := fmt.Sprintf(`
		SELECT id, student_id, date::text, scan_type, timestamp, device_id, raw_score, match_score
		FROM fingerprint_attendance
		WHERE %s
		ORDER BY date DESC, timestamp DESC
		LIMIT $%d OFFSET $%d`, joinAnd(where), argIdx, argIdx+1)
	args = append(args, pageSize, offset)
	rows, err := s.pool.Query(ctx, listQ, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []AttendanceRecord{}
	for rows.Next() {
		var r AttendanceRecord
		if err := rows.Scan(&r.ID, &r.StudentID, &r.Date, &r.ScanType, &r.Timestamp, &r.DeviceID, &r.RawScore, &r.MatchScore); err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	return out, total, nil
}

type DailyAttendanceSummary struct {
	Date          string  `json:"date"`
	PresentCount  int     `json:"present_count"`
	LateCount     int     `json:"late_count"`
	AbsentCount   int     `json:"absent_count"`
	TotalExpected int     `json:"total_expected"`
	AttendancePct float64 `json:"attendance_pct"`
}

type StudentAttendanceSummary struct {
	StudentID    uuid.UUID `json:"student_id"`
	StudentName  string    `json:"student_name"`
	AdmissionNo  string    `json:"admission_no"`
	ClassID      *uuid.UUID `json:"class_id,omitempty"`
	ClassName    string    `json:"class_name,omitempty"`
	PresentDays  int       `json:"present_days"`
	LateDays     int       `json:"late_days"`
	AbsentDays   int       `json:"absent_days"`
	ExpectedDays int       `json:"expected_days"`
	AttendancePct float64  `json:"attendance_pct"`
}

type StudentAttendanceEvent struct {
	Date       string    `json:"date"`
	Status     string    `json:"status"`
	TimeIn     *string   `json:"time_in,omitempty"`
	TimeOut    *string   `json:"time_out,omitempty"`
	Remarks    string    `json:"remarks,omitempty"`
	Subject    *string   `json:"subject,omitempty"`
}

func (s *ScannerStore) DailyClassSummary(ctx context.Context, classID *uuid.UUID, date time.Time) (*DailyAttendanceSummary, error) {
	dateTrunc := date.Truncate(24 * time.Hour)
	where := []string{"ar.date = $1", "ar.subject_id IS NULL"}
	args := []interface{}{dateTrunc}
	argIdx := 2
	if classID != nil {
		where = append(where, fmt.Sprintf("ar.class_id = $%d", argIdx))
		args = append(args, *classID)
		argIdx++
	}
	q := fmt.Sprintf(`
		SELECT
			COUNT(*) FILTER (WHERE ar.status = 'present') AS present_count,
			COUNT(*) FILTER (WHERE ar.status = 'late') AS late_count,
			COUNT(*) FILTER (WHERE ar.status = 'absent') AS absent_count,
			COUNT(*) AS total_expected
		FROM attendance_records ar
		WHERE %s`, joinAnd(where))
	sum := &DailyAttendanceSummary{Date: dateTrunc.Format("2006-01-02")}
	if err := s.pool.QueryRow(ctx, q, args...).Scan(
		&sum.PresentCount, &sum.LateCount, &sum.AbsentCount, &sum.TotalExpected,
	); err != nil {
		return nil, err
	}
	if sum.TotalExpected > 0 {
		sum.AttendancePct = float64(sum.PresentCount+sum.LateCount) / float64(sum.TotalExpected) * 100
	}
	return sum, nil
}

func (s *ScannerStore) ClassRosterAttendance(ctx context.Context, classID uuid.UUID, date time.Time) ([]StudentAttendanceSummary, error) {
	dateTrunc := date.Truncate(24 * time.Hour)
	q := `
		SELECT
			s.id,
			COALESCE(s.full_name, s.first_name || ' ' || s.last_name) AS student_name,
			s.admission_no,
			s.current_class_id,
			COALESCE(cg.grade || '-' || st.name, '') AS class_name,
			CASE WHEN ar.status IN ('present','late') THEN 1 ELSE 0 END AS present,
			CASE WHEN ar.status = 'late' THEN 1 ELSE 0 END AS late,
			CASE WHEN ar.status = 'absent' THEN 1 ELSE 0 END AS absent,
			1 AS expected,
			CASE WHEN ar.status IN ('present','late') THEN 100.0 ELSE 0.0 END AS pct
		FROM students s
		LEFT JOIN class_groups cg ON cg.id = s.current_class_id
		LEFT JOIN streams st ON st.id = cg.stream_id
		LEFT JOIN attendance_records ar
			ON ar.student_id = s.id AND ar.date = $1 AND ar.subject_id IS NULL
		WHERE s.status = 'active' AND s.is_active = TRUE
		  AND s.current_class_id = $2
		ORDER BY s.full_name, s.first_name, s.last_name`
	rows, err := s.pool.Query(ctx, q, dateTrunc, classID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StudentAttendanceSummary{}
	for rows.Next() {
		var r StudentAttendanceSummary
		var present, late, absent, expected int
		var pct float64
		if err := rows.Scan(&r.StudentID, &r.StudentName, &r.AdmissionNo,
			&r.ClassID, &r.ClassName, &present, &late, &absent, &expected, &pct); err != nil {
			continue
		}
		r.PresentDays = present
		r.LateDays = late
		r.AbsentDays = absent
		r.ExpectedDays = expected
		r.AttendancePct = pct
		out = append(out, r)
	}
	return out, nil
}

func (s *ScannerStore) StudentAttendanceRange(ctx context.Context, studentID uuid.UUID, startDate, endDate time.Time) ([]StudentAttendanceEvent, int, int, int, int, float64, error) {
	sd := startDate.Truncate(24 * time.Hour)
	ed := endDate.Truncate(24 * time.Hour)
	q := `
		SELECT
			ar.date::text,
			ar.status,
			ar.time_in::text,
			ar.time_out::text,
			COALESCE(ar.remarks, ''),
			COALESCE(sub.name, '')
		FROM attendance_records ar
		LEFT JOIN subjects sub ON sub.id = ar.subject_id
		WHERE ar.student_id = $1
		  AND ar.date BETWEEN $2 AND $3
		ORDER BY ar.date DESC, ar.subject_id NULLS FIRST`
	rows, err := s.pool.Query(ctx, q, studentID, sd, ed)
	if err != nil {
		return nil, 0, 0, 0, 0, 0, err
	}
	defer rows.Close()
	events := []StudentAttendanceEvent{}
	present, late, absent, expected := 0, 0, 0, 0
	seenDates := map[string]struct{}{}
	for rows.Next() {
		var e StudentAttendanceEvent
		var tin, tout, rem, subj *string
		if err := rows.Scan(&e.Date, &e.Status, &tin, &tout, &rem, &subj); err != nil {
			continue
		}
		e.TimeIn = tin
		e.TimeOut = tout
		if rem != nil && *rem != "" {
			e.Remarks = *rem
		}
		if subj != nil && *subj != "" {
			e.Subject = subj
		}
		events = append(events, e)
		if _, ok := seenDates[e.Date]; !ok {
			seenDates[e.Date] = struct{}{}
			expected++
			switch e.Status {
			case "present":
				present++
			case "late":
				late++
			case "absent":
				absent++
			}
		}
	}
	var pct float64
	if expected > 0 {
		pct = float64(present+late) / float64(expected) * 100
	}
	return events, present, late, absent, expected, pct, nil
}

type StaffAttendanceEvent struct {
	Date    string  `json:"date"`
	Status  string  `json:"status"`
	TimeIn  *string `json:"time_in,omitempty"`
	TimeOut *string `json:"time_out,omitempty"`
}

// StaffAttendanceRange returns one event per weekday in [startDate,endDate], deriving
// present/late/absent from staff_attendance entry/exit rows so it can drive a
// clock-in-time-vs-date graph the same way StudentAttendanceRange does for students.
func (s *ScannerStore) StaffAttendanceRange(ctx context.Context, staffID uuid.UUID, startDate, endDate time.Time) ([]StaffAttendanceEvent, int, int, int, int, float64, error) {
	sd := startDate.Truncate(24 * time.Hour)
	ed := endDate.Truncate(24 * time.Hour)
	q := `
		SELECT
			d::date::text,
			to_char(ent.timestamp AT TIME ZONE 'Africa/Nairobi', 'HH24:MI:SS'),
			to_char(ext.timestamp AT TIME ZONE 'Africa/Nairobi', 'HH24:MI:SS')
		FROM generate_series($2::date, $3::date, interval '1 day') d
		LEFT JOIN staff_attendance ent ON ent.staff_id = $1 AND ent.date = d::date AND ent.scan_type = 'entry'
		LEFT JOIN staff_attendance ext ON ext.staff_id = $1 AND ext.date = d::date AND ext.scan_type = 'exit'
		WHERE EXTRACT(ISODOW FROM d) < 6
		ORDER BY d DESC`
	rows, err := s.pool.Query(ctx, q, staffID, sd, ed)
	if err != nil {
		return nil, 0, 0, 0, 0, 0, err
	}
	defer rows.Close()
	const cutoff = "08:15:00"
	events := []StaffAttendanceEvent{}
	present, late, absent, expected := 0, 0, 0, 0
	for rows.Next() {
		var date string
		var tin, tout *string
		if err := rows.Scan(&date, &tin, &tout); err != nil {
			continue
		}
		status := "absent"
		if tin != nil {
			if *tin > cutoff {
				status = "late"
			} else {
				status = "present"
			}
		}
		events = append(events, StaffAttendanceEvent{Date: date, Status: status, TimeIn: tin, TimeOut: tout})
		expected++
		switch status {
		case "present":
			present++
		case "late":
			late++
		case "absent":
			absent++
		}
	}
	var pct float64
	if expected > 0 {
		pct = float64(present+late) / float64(expected) * 100
	}
	return events, present, late, absent, expected, pct, nil
}

func joinAnd(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " AND "
		}
		out += p
	}
	return out
}
