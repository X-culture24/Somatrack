package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
)

// VehiclesService is the real backend for the school transport fleet — it
// replaced a registerPlaceholderCRUD stub whose GET /vehicles/ returned a
// plain {"resource":...} object instead of an array, which crashed
// TransportManagementPage.tsx's render (.filter/.map called on a
// non-array) the moment the page loaded, taking down the whole transport
// screen, not just the vehicles section.
type VehiclesService struct {
	pool *pgxpool.Pool
}

func NewVehiclesService(pool *pgxpool.Pool) *VehiclesService {
	return &VehiclesService{pool: pool}
}

func (s *VehiclesService) RegisterRoutes(r chi.Router) {
	r.Route("/vehicles/", func(r chi.Router) {
		r.Get("/", s.list)
		r.Post("/", adminOnly()(http.HandlerFunc(s.create)).ServeHTTP)
		r.Get("/{id}/", s.get)
		r.Put("/{id}/", adminOnly()(http.HandlerFunc(s.update)).ServeHTTP)
		r.Patch("/{id}/", adminOnly()(http.HandlerFunc(s.update)).ServeHTTP)
		r.Delete("/{id}/", adminOnly()(http.HandlerFunc(s.delete)).ServeHTTP)
	})
}

type Vehicle struct {
	ID                uuid.UUID  `json:"id"`
	PlateNo           string     `json:"plate_no" validate:"required"`
	VehicleType       string     `json:"vehicle_type" validate:"omitempty,oneof=bus van car other"`
	Capacity          int        `json:"capacity"`
	RouteID           *uuid.UUID `json:"route_id,omitempty"`
	DriverName        string     `json:"driver_name"`
	DriverPhone       string     `json:"driver_phone"`
	LastServiceDate   *string    `json:"last_service_date,omitempty"`
	NextServiceDate   *string    `json:"next_service_date,omitempty"`
	InsuranceExpiry   *string    `json:"insurance_expiry,omitempty"`
	RoadLicenseExpiry *string    `json:"road_license_expiry,omitempty"`
	Status            string     `json:"status" validate:"omitempty,oneof=active maintenance grounded decommissioned"`
	RouteName         string     `json:"route_name,omitempty"`
}

const vehicleCols = `v.id, v.plate_no, v.vehicle_type, v.capacity, v.route_id, v.driver_name, v.driver_phone,
	v.last_service_date::text, v.next_service_date::text, v.insurance_expiry::text, v.road_license_expiry::text,
	v.status, COALESCE(tr.route_name, '')`

func scanVehicle(row interface {
	Scan(dest ...interface{}) error
}) (Vehicle, error) {
	var v Vehicle
	err := row.Scan(&v.ID, &v.PlateNo, &v.VehicleType, &v.Capacity, &v.RouteID, &v.DriverName, &v.DriverPhone,
		&v.LastServiceDate, &v.NextServiceDate, &v.InsuranceExpiry, &v.RoadLicenseExpiry, &v.Status, &v.RouteName)
	return v, err
}

func (s *VehiclesService) list(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT `+vehicleCols+`
		FROM vehicles v LEFT JOIN transport_routes tr ON tr.id = v.route_id
		ORDER BY v.plate_no`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []Vehicle{}
	for rows.Next() {
		v, err := scanVehicle(rows)
		if err != nil {
			continue
		}
		out = append(out, v)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *VehiclesService) create(w http.ResponseWriter, r *http.Request) {
	var body Vehicle
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if body.VehicleType == "" {
		body.VehicleType = "bus"
	}
	if body.Status == "" {
		body.Status = "active"
	}
	if body.Capacity == 0 {
		body.Capacity = 40
	}
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO vehicles (plate_no, vehicle_type, capacity, route_id, driver_name, driver_phone,
			last_service_date, next_service_date, insurance_expiry, road_license_expiry, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		body.PlateNo, body.VehicleType, body.Capacity, body.RouteID, body.DriverName, body.DriverPhone,
		body.LastServiceDate, body.NextServiceDate, body.InsuranceExpiry, body.RoadLicenseExpiry, body.Status).Scan(&body.ID)
	if err != nil {
		if strings.Contains(err.Error(), "plate_no") {
			httpx.ErrConflict(w, "a vehicle with that plate number already exists")
			return
		}
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, body)
}

func (s *VehiclesService) get(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	v, err := scanVehicle(s.pool.QueryRow(r.Context(), `
		SELECT `+vehicleCols+` FROM vehicles v LEFT JOIN transport_routes tr ON tr.id = v.route_id WHERE v.id = $1`, id))
	if err != nil {
		httpx.ErrNotFound(w, "vehicle not found")
		return
	}
	httpx.JSON(w, http.StatusOK, v)
}

func (s *VehiclesService) update(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var body struct {
		PlateNo           *string       `json:"plate_no,omitempty"`
		VehicleType       *string       `json:"vehicle_type,omitempty"`
		Capacity          *int          `json:"capacity,omitempty"`
		RouteID           *OptionalUUID `json:"route_id,omitempty"`
		DriverName        *string       `json:"driver_name,omitempty"`
		DriverPhone       *string       `json:"driver_phone,omitempty"`
		LastServiceDate   *string       `json:"last_service_date,omitempty"`
		NextServiceDate   *string       `json:"next_service_date,omitempty"`
		InsuranceExpiry   *string       `json:"insurance_expiry,omitempty"`
		RoadLicenseExpiry *string       `json:"road_license_expiry,omitempty"`
		Status            *string       `json:"status,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	var routePtr *uuid.UUID
	if body.RouteID != nil {
		routePtr = body.RouteID.Ptr()
	}
	_, err = s.pool.Exec(r.Context(), `
		UPDATE vehicles SET
			plate_no=COALESCE(NULLIF($1,''), plate_no), vehicle_type=COALESCE(NULLIF($2,''), vehicle_type),
			capacity=COALESCE($3, capacity),
			route_id=CASE WHEN $4::boolean THEN $5 ELSE route_id END,
			driver_name=COALESCE($6, driver_name), driver_phone=COALESCE($7, driver_phone),
			last_service_date=COALESCE($8, last_service_date), next_service_date=COALESCE($9, next_service_date),
			insurance_expiry=COALESCE($10, insurance_expiry), road_license_expiry=COALESCE($11, road_license_expiry),
			status=COALESCE(NULLIF($12,''), status), updated_at=NOW()
		WHERE id = $13`,
		body.PlateNo, body.VehicleType, body.Capacity, body.RouteID != nil, routePtr,
		body.DriverName, body.DriverPhone, body.LastServiceDate, body.NextServiceDate,
		body.InsuranceExpiry, body.RoadLicenseExpiry, body.Status, id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	v, err := scanVehicle(s.pool.QueryRow(r.Context(), `
		SELECT `+vehicleCols+` FROM vehicles v LEFT JOIN transport_routes tr ON tr.id = v.route_id WHERE v.id = $1`, id))
	if err != nil {
		httpx.ErrNotFound(w, "vehicle not found")
		return
	}
	httpx.JSON(w, http.StatusOK, v)
}

func (s *VehiclesService) delete(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM vehicles WHERE id = $1`, id); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}
