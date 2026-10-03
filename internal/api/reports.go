package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/rpc"
)

// ReportsService exposes report-svc's real query engine at the gateway.
// The RPC client and report-svc side were both already built, but the
// gateway routes were left as placeholder stubs — report-svc's own SQL was
// also still written against a pre-refactor schema (a "classes" table and
// "s.class_id" that no longer exist, "i.total_due" that was renamed to
// total_amount, and a Postgres-illegal COUNT(DISTINCT ...) OVER (...)) and
// would have 500'd immediately if wired up as-is, which is presumably why
// it never was. See internal/reports/service.go for those fixes.
type ReportsService struct {
	client *rpc.ReportClient
}

func NewReportsService(client *rpc.ReportClient) *ReportsService {
	return &ReportsService{client: client}
}

func (s *ReportsService) RegisterRoutes(r chi.Router) {
	r.Route("/reports/", func(r chi.Router) {
		r.Route("/fee-collection/", func(r chi.Router) {
			r.Use(financeOnly())
			r.Get("/", s.feeCollection)
		})
		// These list every learner's attendance/results, so parents and
		// students use the per-student results trail instead.
		r.Group(func(r chi.Router) {
			r.Use(academicStaffOnly())
			r.Get("/attendance/", s.attendance)
			r.Get("/grades/", s.grades)
			r.Get("/class-performance/", s.classPerformance)
		})
	})
}

func queryUUID(r *http.Request, key string) *uuid.UUID {
	v := r.URL.Query().Get(key)
	if v == "" {
		return nil
	}
	id, err := uuid.Parse(v)
	if err != nil {
		return nil
	}
	return &id
}

func (s *ReportsService) attendance(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("start_date") == "" || q.Get("end_date") == "" {
		httpx.ErrBadRequest(w, "start_date and end_date (YYYY-MM-DD) are required")
		return
	}
	out, err := s.client.AttendanceReport(r.Context(), &rpc.AttendanceReportRequest{
		StartDate: q.Get("start_date"),
		EndDate:   q.Get("end_date"),
		ClassID:   queryUUID(r, "class_id"),
		StreamID:  queryUUID(r, "stream_id"),
		StudentID: queryUUID(r, "student_id"),
	})
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (s *ReportsService) grades(w http.ResponseWriter, r *http.Request) {
	termID := queryUUID(r, "term_id")
	if termID == nil {
		httpx.ErrBadRequest(w, "term_id is required")
		return
	}
	out, err := s.client.GradesReport(r.Context(), &rpc.GradesReportRequest{
		TermID:    *termID,
		ClassID:   queryUUID(r, "class_id"),
		SubjectID: queryUUID(r, "subject_id"),
		StudentID: queryUUID(r, "student_id"),
	})
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (s *ReportsService) feeCollection(w http.ResponseWriter, r *http.Request) {
	out, err := s.client.FeeCollectionReport(r.Context(), &rpc.FeeCollectionRequest{
		TermID: queryUUID(r, "term_id"),
	})
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (s *ReportsService) classPerformance(w http.ResponseWriter, r *http.Request) {
	termID := queryUUID(r, "term_id")
	classID := queryUUID(r, "class_id")
	if termID == nil || classID == nil {
		httpx.ErrBadRequest(w, "term_id and class_id are required")
		return
	}
	out, err := s.client.ClassPerformanceReport(r.Context(), &rpc.ClassPerformanceRequest{
		TermID: *termID, ClassID: *classID,
	})
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}
