package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
)

// ClubsService implements co-curricular clubs and student membership —
// "co-curricular, all clubs that were present" from the paper admission
// form. Clubs are simple reference data (like Streams/Subjects); the
// membership join is exposed both as its own route and inline on a
// student's detail view.
type ClubsService struct {
	pool *pgxpool.Pool
}

func NewClubsService(pool *pgxpool.Pool) *ClubsService {
	return &ClubsService{pool: pool}
}

func (s *ClubsService) RegisterRoutes(r chi.Router) {
	r.Route("/clubs/", func(r chi.Router) {
		r.Get("/", s.list)
		r.Post("/", adminOnly()(http.HandlerFunc(s.create)).ServeHTTP)
		r.Get("/{id}/", s.get)
		r.Put("/{id}/", adminOnly()(http.HandlerFunc(s.update)).ServeHTTP)
		r.Patch("/{id}/", adminOnly()(http.HandlerFunc(s.update)).ServeHTTP)
		r.Delete("/{id}/", adminOnly()(http.HandlerFunc(s.delete)).ServeHTTP)
	})
}

type Club struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name" validate:"required"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
}

func (s *ClubsService) list(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `SELECT id, name, description, is_active FROM clubs ORDER BY name`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []Club{}
	for rows.Next() {
		var c Club
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.IsActive); err != nil {
			continue
		}
		out = append(out, c)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *ClubsService) create(w http.ResponseWriter, r *http.Request) {
	var body Club
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	body.IsActive = true
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO clubs (name, description, is_active) VALUES ($1, $2, $3) RETURNING id`,
		body.Name, body.Description, body.IsActive).Scan(&body.ID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, body)
}

func (s *ClubsService) get(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var c Club
	err = s.pool.QueryRow(r.Context(), `SELECT id, name, description, is_active FROM clubs WHERE id = $1`, id).
		Scan(&c.ID, &c.Name, &c.Description, &c.IsActive)
	if err != nil {
		httpx.ErrNotFound(w, "club not found")
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (s *ClubsService) update(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var body Club
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	_, err = s.pool.Exec(r.Context(), `
		UPDATE clubs SET name=$1, description=$2, is_active=$3, updated_at=NOW() WHERE id = $4`,
		body.Name, body.Description, body.IsActive, id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	body.ID = id
	httpx.JSON(w, http.StatusOK, body)
}

func (s *ClubsService) delete(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM clubs WHERE id = $1`, id); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}
