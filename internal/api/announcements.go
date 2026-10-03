package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
)

// AnnouncementsService is direct-to-DB, same pattern as students/guardians —
// simple CRUD over one table, no need for a separate microservice.
type AnnouncementsService struct {
	pool *pgxpool.Pool
}

func NewAnnouncementsService(pool *pgxpool.Pool) *AnnouncementsService {
	return &AnnouncementsService{pool: pool}
}

func (s *AnnouncementsService) RegisterRoutes(r chi.Router) {
	r.Route("/announcements/", func(r chi.Router) {
		r.Get("/", s.list)
		r.Get("/{id}/", s.get)
		r.Group(func(r chi.Router) {
			r.Use(adminOnly())
			r.Post("/", s.create)
			r.Put("/{id}/", s.update)
			r.Patch("/{id}/", s.update)
			r.Delete("/{id}/", s.delete)
		})
	})
}

type Announcement struct {
	ID             uuid.UUID  `json:"id"`
	Title          string     `json:"title" validate:"required"`
	Content        string     `json:"content" validate:"required"`
	Summary        string     `json:"summary"`
	Audience       string     `json:"audience" validate:"omitempty,oneof=all staff parents students teachers finance administration class"`
	TargetClassID  *uuid.UUID `json:"target_class_id,omitempty"`
	Priority       string     `json:"priority" validate:"omitempty,oneof=low normal high urgent"`
	PublishDate    time.Time  `json:"publish_date"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	IsPublished    bool       `json:"is_published"`
	CreatedByID    *uuid.UUID `json:"created_by_id,omitempty"`
	CreatedByName  string     `json:"created_by_name,omitempty"`
}

const announcementCols = `a.id, a.title, a.content, a.summary, a.audience, a.target_class_id,
	a.priority, a.publish_date, a.expires_at, a.is_published, a.created_by_id,
	COALESCE(u.first_name || ' ' || u.last_name, '')`

func scanAnnouncement(row interface{ Scan(dest ...interface{}) error }) (Announcement, error) {
	var a Announcement
	err := row.Scan(&a.ID, &a.Title, &a.Content, &a.Summary, &a.Audience, &a.TargetClassID,
		&a.Priority, &a.PublishDate, &a.ExpiresAt, &a.IsPublished, &a.CreatedByID, &a.CreatedByName)
	return a, err
}

func (s *AnnouncementsService) list(w http.ResponseWriter, r *http.Request) {
	where := []string{"a.is_published = TRUE", "(a.expires_at IS NULL OR a.expires_at > NOW())"}
	args := []interface{}{}
	if v := r.URL.Query().Get("audience"); v != "" {
		where = append(where, "a.audience = $1")
		args = append(args, v)
	}
	q := `SELECT ` + announcementCols + ` FROM announcements a
		LEFT JOIN users u ON u.id = a.created_by_id
		WHERE ` + joinAnd(where) + ` ORDER BY a.publish_date DESC LIMIT 100`
	rows, err := s.pool.Query(r.Context(), q, args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []Announcement{}
	for rows.Next() {
		a, err := scanAnnouncement(rows)
		if err != nil {
			continue
		}
		out = append(out, a)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *AnnouncementsService) get(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	a, err := scanAnnouncement(s.pool.QueryRow(r.Context(), `
		SELECT `+announcementCols+` FROM announcements a LEFT JOIN users u ON u.id = a.created_by_id
		WHERE a.id = $1`, id))
	if err != nil {
		httpx.ErrNotFound(w, "announcement not found")
		return
	}
	httpx.JSON(w, http.StatusOK, a)
}

func (s *AnnouncementsService) create(w http.ResponseWriter, r *http.Request) {
	var body Announcement
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if body.Audience == "" {
		body.Audience = "all"
	}
	if body.Priority == "" {
		body.Priority = "normal"
	}
	userID, _ := r.Context().Value("user_id").(uuid.UUID)
	id := uuid.New()
	_, err := s.pool.Exec(r.Context(), `
		INSERT INTO announcements (id, title, content, summary, audience, target_class_id,
			priority, publish_date, expires_at, is_published, created_by_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), $8, true, $9, NOW(), NOW())`,
		id, body.Title, body.Content, body.Summary, body.Audience, body.TargetClassID,
		body.Priority, body.ExpiresAt, userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	a, err := scanAnnouncement(s.pool.QueryRow(r.Context(), `
		SELECT `+announcementCols+` FROM announcements a LEFT JOIN users u ON u.id = a.created_by_id
		WHERE a.id = $1`, id))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, a)
}

func (s *AnnouncementsService) update(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var body struct {
		Title       *string    `json:"title,omitempty"`
		Content     *string    `json:"content,omitempty"`
		Summary     *string    `json:"summary,omitempty"`
		Audience    *string    `json:"audience,omitempty"`
		Priority    *string    `json:"priority,omitempty"`
		ExpiresAt   *time.Time `json:"expires_at,omitempty"`
		IsPublished *bool      `json:"is_published,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	_, err = s.pool.Exec(r.Context(), `
		UPDATE announcements SET
			title = COALESCE($1, title),
			content = COALESCE($2, content),
			summary = COALESCE($3, summary),
			audience = COALESCE($4, audience),
			priority = COALESCE($5, priority),
			expires_at = COALESCE($6, expires_at),
			is_published = COALESCE($7, is_published),
			updated_at = NOW()
		WHERE id = $8`,
		body.Title, body.Content, body.Summary, body.Audience, body.Priority, body.ExpiresAt, body.IsPublished, id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	a, err := scanAnnouncement(s.pool.QueryRow(r.Context(), `
		SELECT `+announcementCols+` FROM announcements a LEFT JOIN users u ON u.id = a.created_by_id
		WHERE a.id = $1`, id))
	if err != nil {
		httpx.ErrNotFound(w, "announcement not found")
		return
	}
	httpx.JSON(w, http.StatusOK, a)
}

func (s *AnnouncementsService) delete(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), "DELETE FROM announcements WHERE id = $1", id); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}
