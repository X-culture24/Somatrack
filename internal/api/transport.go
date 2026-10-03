package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
)

// TransportRoutesService is a minimal real implementation of just
// transport_routes — enough to back the "uses transport / which route"
// field on student registration. Vehicles and transport_attendance stay
// placeholder stubs; full transport management wasn't in scope here.
type TransportRoutesService struct {
	pool *pgxpool.Pool
}

func NewTransportRoutesService(pool *pgxpool.Pool) *TransportRoutesService {
	return &TransportRoutesService{pool: pool}
}

func (s *TransportRoutesService) RegisterRoutes(r chi.Router) {
	r.Route("/transport-routes/", func(r chi.Router) {
		r.Get("/", s.list)
		r.Post("/", adminOnly()(http.HandlerFunc(s.create)).ServeHTTP)
		r.Get("/{id}/", s.get)
		r.Put("/{id}/", adminOnly()(http.HandlerFunc(s.update)).ServeHTTP)
		r.Patch("/{id}/", adminOnly()(http.HandlerFunc(s.update)).ServeHTTP)
		r.Delete("/{id}/", adminOnly()(http.HandlerFunc(s.delete)).ServeHTTP)
	})
}

type TransportRoute struct {
	ID            uuid.UUID `json:"id"`
	RouteName     string    `json:"route_name" validate:"required"`
	RouteCode     string    `json:"route_code" validate:"required"`
	Description   string    `json:"description"`
	StartLocation string    `json:"start_location" validate:"required"`
	EndLocation   string    `json:"end_location" validate:"required"`
	Fare          float64   `json:"fare"`
	DriverName    string    `json:"driver_name"`
	DriverPhone   string    `json:"driver_phone"`
	IsActive      bool      `json:"is_active"`
}

const routeCols = `id, route_name, route_code, description, start_location, end_location,
	fare, driver_name, driver_phone, is_active`

func scanRoute(row interface {
	Scan(dest ...interface{}) error
}) (TransportRoute, error) {
	var t TransportRoute
	err := row.Scan(&t.ID, &t.RouteName, &t.RouteCode, &t.Description, &t.StartLocation, &t.EndLocation,
		&t.Fare, &t.DriverName, &t.DriverPhone, &t.IsActive)
	return t, err
}

func (s *TransportRoutesService) list(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `SELECT `+routeCols+` FROM transport_routes WHERE is_active = TRUE ORDER BY route_name`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []TransportRoute{}
	for rows.Next() {
		t, err := scanRoute(rows)
		if err != nil {
			continue
		}
		out = append(out, t)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *TransportRoutesService) create(w http.ResponseWriter, r *http.Request) {
	var body TransportRoute
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	body.IsActive = true
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO transport_routes (route_name, route_code, description, start_location, end_location,
			fare, driver_name, driver_phone, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
		body.RouteName, body.RouteCode, body.Description, body.StartLocation, body.EndLocation,
		body.Fare, body.DriverName, body.DriverPhone, body.IsActive).Scan(&body.ID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, body)
}

func (s *TransportRoutesService) get(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	t, err := scanRoute(s.pool.QueryRow(r.Context(), `SELECT `+routeCols+` FROM transport_routes WHERE id = $1`, id))
	if err != nil {
		httpx.ErrNotFound(w, "transport route not found")
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

func (s *TransportRoutesService) update(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	// Plain-decoded, all-pointer body + COALESCE below: the previous version
	// full-struct-bound this against TransportRoute (validate:"required" on
	// route_name/route_code/start_location/end_location) and always wrote
	// is_active from the request body. Since the edit form never sends
	// is_active, every save silently set it to false — and list() filters
	// WHERE is_active = TRUE, so the route vanished from every dropdown and
	// listing on the very first edit. This was the root cause behind
	// "transport not working".
	var body struct {
		RouteName     *string  `json:"route_name,omitempty"`
		RouteCode     *string  `json:"route_code,omitempty"`
		Description   *string  `json:"description,omitempty"`
		StartLocation *string  `json:"start_location,omitempty"`
		EndLocation   *string  `json:"end_location,omitempty"`
		Fare          *float64 `json:"fare,omitempty"`
		DriverName    *string  `json:"driver_name,omitempty"`
		DriverPhone   *string  `json:"driver_phone,omitempty"`
		IsActive      *bool    `json:"is_active,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	_, err = s.pool.Exec(r.Context(), `
		UPDATE transport_routes SET
			route_name=COALESCE(NULLIF($1,''), route_name), route_code=COALESCE(NULLIF($2,''), route_code),
			description=COALESCE($3, description), start_location=COALESCE(NULLIF($4,''), start_location),
			end_location=COALESCE(NULLIF($5,''), end_location), fare=COALESCE($6, fare),
			driver_name=COALESCE($7, driver_name), driver_phone=COALESCE($8, driver_phone),
			is_active=COALESCE($9, is_active), updated_at=NOW()
		WHERE id = $10`,
		body.RouteName, body.RouteCode, body.Description, body.StartLocation, body.EndLocation,
		body.Fare, body.DriverName, body.DriverPhone, body.IsActive, id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	t, err := scanRoute(s.pool.QueryRow(r.Context(), `SELECT `+routeCols+` FROM transport_routes WHERE id = $1`, id))
	if err != nil {
		httpx.ErrNotFound(w, "transport route not found")
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

func (s *TransportRoutesService) delete(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM transport_routes WHERE id = $1`, id); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}
