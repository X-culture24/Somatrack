package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
)

// LibraryService is direct-to-DB CRUD for books plus the loan lifecycle
// (checkout/return/renew) — replacing the placeholder stubs. Reads are open
// to any authenticated user; catalogue writes (add/edit/remove a book) and
// checkout/return/renew are admin-gated, matching the other direct-to-DB
// resources in this package.
type LibraryService struct {
	pool *pgxpool.Pool
}

func NewLibraryService(pool *pgxpool.Pool) *LibraryService {
	return &LibraryService{pool: pool}
}

func (s *LibraryService) RegisterRoutes(r chi.Router) {
	r.Route("/library/books/", func(r chi.Router) {
		r.Get("/", s.listBooks)
		r.Get("/{id}/", s.getBook)
		r.Group(func(r chi.Router) {
			r.Use(adminOnly())
			r.Post("/", s.createBook)
			r.Put("/{id}/", s.updateBook)
			r.Patch("/{id}/", s.updateBook)
			r.Delete("/{id}/", s.deleteBook)
		})
	})
	r.Route("/library/loans/", func(r chi.Router) {
		r.Get("/", s.listLoans)
		r.Get("/overdue/", s.listOverdue)
		r.Group(func(r chi.Router) {
			r.Use(adminOnly())
			r.Post("/", s.checkout)
			r.Post("/{id}/return/", s.returnLoan)
			r.Post("/{id}/renew/", s.renewLoan)
		})
	})
}

type Book struct {
	ID              uuid.UUID `json:"id"`
	Title           string    `json:"title" validate:"required"`
	Author          string    `json:"author"`
	ISBN            string    `json:"isbn"`
	Publisher       string    `json:"publisher"`
	PublicationYear *int      `json:"publication_year,omitempty"`
	Edition         string    `json:"edition"`
	ShelfLocation   string    `json:"shelf_location"`
	TotalCopies     int       `json:"total_copies"`
	AvailableCopies int       `json:"available_copies"`
	ReferenceOnly   bool      `json:"reference_only"`
	Condition       string    `json:"condition" validate:"omitempty,oneof=new good fair poor damaged lost"`
	Price           float64   `json:"price"`
	IsActive        bool      `json:"is_active"`
}

const bookCols = `id, title, author, isbn, publisher, publication_year, edition, shelf_location,
	total_copies, available_copies, reference_only, condition, price, is_active`

func scanBook(row interface {
	Scan(dest ...interface{}) error
}) (Book, error) {
	var b Book
	err := row.Scan(&b.ID, &b.Title, &b.Author, &b.ISBN, &b.Publisher, &b.PublicationYear, &b.Edition,
		&b.ShelfLocation, &b.TotalCopies, &b.AvailableCopies, &b.ReferenceOnly, &b.Condition, &b.Price, &b.IsActive)
	return b, err
}

func (s *LibraryService) listBooks(w http.ResponseWriter, r *http.Request) {
	where := []string{"is_active = TRUE"}
	args := []interface{}{}
	if q := strings.TrimSpace(r.URL.Query().Get("search")); q != "" {
		where = append(where, "(title ILIKE $1 OR author ILIKE $1)")
		args = append(args, "%"+q+"%")
	}
	rows, err := s.pool.Query(r.Context(), `SELECT `+bookCols+` FROM books WHERE `+joinAnd(where)+` ORDER BY title`, args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []Book{}
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			continue
		}
		out = append(out, b)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *LibraryService) getBook(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	b, err := scanBook(s.pool.QueryRow(r.Context(), `SELECT `+bookCols+` FROM books WHERE id = $1`, id))
	if err != nil {
		httpx.ErrNotFound(w, "book not found")
		return
	}
	httpx.JSON(w, http.StatusOK, b)
}

func (s *LibraryService) createBook(w http.ResponseWriter, r *http.Request) {
	var body Book
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if body.Condition == "" {
		body.Condition = "good"
	}
	if body.TotalCopies <= 0 {
		body.TotalCopies = 1
	}
	body.AvailableCopies = body.TotalCopies
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO books (id, title, author, isbn, publisher, publication_year, edition, shelf_location,
			total_copies, available_copies, reference_only, condition, price, is_active, date_acquired, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,true,CURRENT_DATE,NOW(),NOW())
		RETURNING id`,
		uuid.New(), body.Title, body.Author, body.ISBN, body.Publisher, body.PublicationYear, body.Edition,
		body.ShelfLocation, body.TotalCopies, body.AvailableCopies, body.ReferenceOnly, body.Condition, body.Price,
	).Scan(&body.ID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	body.IsActive = true
	httpx.JSON(w, http.StatusCreated, body)
}

func (s *LibraryService) updateBook(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var body struct {
		Title           *string  `json:"title,omitempty"`
		Author          *string  `json:"author,omitempty"`
		ISBN            *string  `json:"isbn,omitempty"`
		Publisher       *string  `json:"publisher,omitempty"`
		PublicationYear *int     `json:"publication_year,omitempty"`
		Edition         *string  `json:"edition,omitempty"`
		ShelfLocation   *string  `json:"shelf_location,omitempty"`
		TotalCopies     *int     `json:"total_copies,omitempty"`
		ReferenceOnly   *bool    `json:"reference_only,omitempty"`
		Condition       *string  `json:"condition,omitempty"`
		Price           *float64 `json:"price,omitempty"`
		IsActive        *bool    `json:"is_active,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	_, err = s.pool.Exec(r.Context(), `
		UPDATE books SET
			title = COALESCE($1, title), author = COALESCE($2, author), isbn = COALESCE($3, isbn),
			publisher = COALESCE($4, publisher), publication_year = COALESCE($5, publication_year),
			edition = COALESCE($6, edition), shelf_location = COALESCE($7, shelf_location),
			total_copies = COALESCE($8, total_copies), reference_only = COALESCE($9, reference_only),
			condition = COALESCE(NULLIF($10,''), condition), price = COALESCE($11, price),
			is_active = COALESCE($12, is_active), updated_at = NOW()
		WHERE id = $13`,
		body.Title, body.Author, body.ISBN, body.Publisher, body.PublicationYear, body.Edition,
		body.ShelfLocation, body.TotalCopies, body.ReferenceOnly, body.Condition, body.Price, body.IsActive, id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	b, err := scanBook(s.pool.QueryRow(r.Context(), `SELECT `+bookCols+` FROM books WHERE id = $1`, id))
	if err != nil {
		httpx.ErrNotFound(w, "book not found")
		return
	}
	httpx.JSON(w, http.StatusOK, b)
}

func (s *LibraryService) deleteBook(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM books WHERE id = $1`, id); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}

type Loan struct {
	ID          uuid.UUID  `json:"id"`
	BookID      uuid.UUID  `json:"book_id"`
	BookTitle   string     `json:"book_title"`
	StudentID   uuid.UUID  `json:"student_id"`
	StudentName string     `json:"student_name"`
	BorrowedAt  time.Time  `json:"borrowed_at"`
	DueAt       time.Time  `json:"due_at"`
	ReturnedAt  *time.Time `json:"returned_at,omitempty"`
	RenewCount  int        `json:"renew_count"`
	Status      string     `json:"status"`
	FineAmount  float64    `json:"fine_amount"`
}

const loanCols = `l.id, l.book_id, b.title, l.student_id, COALESCE(s.full_name,''), l.borrowed_at, l.due_at,
	l.returned_at, l.renew_count, l.status, l.fine_amount`

func scanLoan(row interface {
	Scan(dest ...interface{}) error
}) (Loan, error) {
	var l Loan
	err := row.Scan(&l.ID, &l.BookID, &l.BookTitle, &l.StudentID, &l.StudentName, &l.BorrowedAt, &l.DueAt,
		&l.ReturnedAt, &l.RenewCount, &l.Status, &l.FineAmount)
	return l, err
}

func (s *LibraryService) listLoans(w http.ResponseWriter, r *http.Request) {
	where := []string{"true"}
	if r.URL.Query().Get("active") == "true" {
		where = []string{"l.returned_at IS NULL"}
	}
	rows, err := s.pool.Query(r.Context(), `
		SELECT `+loanCols+` FROM loans l
		JOIN books b ON b.id = l.book_id
		LEFT JOIN students s ON s.id = l.student_id
		WHERE `+joinAnd(where)+` ORDER BY l.borrowed_at DESC LIMIT 200`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []Loan{}
	for rows.Next() {
		l, err := scanLoan(rows)
		if err != nil {
			continue
		}
		out = append(out, l)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *LibraryService) listOverdue(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT `+loanCols+` FROM loans l
		JOIN books b ON b.id = l.book_id
		LEFT JOIN students s ON s.id = l.student_id
		WHERE l.returned_at IS NULL AND l.due_at < NOW()
		ORDER BY l.due_at`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	type overdueLoan struct {
		Loan
		DaysOverdue int `json:"days_overdue"`
	}
	out := []overdueLoan{}
	for rows.Next() {
		l, err := scanLoan(rows)
		if err != nil {
			continue
		}
		days := int(time.Since(l.DueAt).Hours() / 24)
		out = append(out, overdueLoan{Loan: l, DaysOverdue: days})
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *LibraryService) checkout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BookID    uuid.UUID `json:"book"`
		StudentID uuid.UUID `json:"student"`
		DueAt     string    `json:"due_at" validate:"required"`
	}
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

	var available int
	if err := tx.QueryRow(r.Context(), `SELECT available_copies FROM books WHERE id = $1 FOR UPDATE`, body.BookID).Scan(&available); err != nil {
		httpx.ErrBadRequest(w, "book not found")
		return
	}
	if available <= 0 {
		httpx.ErrConflict(w, "no copies available to issue")
		return
	}
	id := uuid.New()
	if _, err := tx.Exec(r.Context(), `
		INSERT INTO loans (id, book_id, student_id, borrowed_at, due_at, status, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), $4, 'active', NOW(), NOW())`,
		id, body.BookID, body.StudentID, body.DueAt); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	if _, err := tx.Exec(r.Context(), `UPDATE books SET available_copies = available_copies - 1 WHERE id = $1`, body.BookID); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	l, err := scanLoan(s.pool.QueryRow(r.Context(), `
		SELECT `+loanCols+` FROM loans l JOIN books b ON b.id = l.book_id
		LEFT JOIN students s ON s.id = l.student_id WHERE l.id = $1`, id))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, l)
}

func (s *LibraryService) returnLoan(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback(r.Context())

	var bookID uuid.UUID
	var dueAt time.Time
	var returnedAt *time.Time
	if err := tx.QueryRow(r.Context(), `SELECT book_id, due_at, returned_at FROM loans WHERE id = $1 FOR UPDATE`, id).
		Scan(&bookID, &dueAt, &returnedAt); err != nil {
		if err == pgx.ErrNoRows {
			httpx.ErrNotFound(w, "loan not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if returnedAt != nil {
		httpx.ErrConflict(w, "loan already returned")
		return
	}
	fineDays := int(time.Since(dueAt).Hours() / 24)
	fineAmount := 0.0
	if fineDays > 0 {
		fineAmount = float64(fineDays) * 10 // KES 10/day overdue — matches a typical school library fine rate
	}
	status := "returned"
	if _, err := tx.Exec(r.Context(), `
		UPDATE loans SET returned_at = NOW(), status = $1, fine_days = $2, fine_amount = $3, updated_at = NOW()
		WHERE id = $4`, status, maxInt(fineDays, 0), fineAmount, id); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if _, err := tx.Exec(r.Context(), `UPDATE books SET available_copies = available_copies + 1 WHERE id = $1`, bookID); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	l, err := scanLoan(s.pool.QueryRow(r.Context(), `
		SELECT `+loanCols+` FROM loans l JOIN books b ON b.id = l.book_id
		LEFT JOIN students s ON s.id = l.student_id WHERE l.id = $1`, id))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, l)
}

func (s *LibraryService) renewLoan(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var extraDays int
	var body struct {
		ExtraDays int `json:"extra_days"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	extraDays = body.ExtraDays
	if extraDays <= 0 {
		extraDays = 7
	}
	_, err = s.pool.Exec(r.Context(), `
		UPDATE loans SET due_at = due_at + ($1 || ' days')::interval, renew_count = renew_count + 1, updated_at = NOW()
		WHERE id = $2 AND returned_at IS NULL`, extraDays, id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	l, err := scanLoan(s.pool.QueryRow(r.Context(), `
		SELECT `+loanCols+` FROM loans l JOIN books b ON b.id = l.book_id
		LEFT JOIN students s ON s.id = l.student_id WHERE l.id = $1`, id))
	if err != nil {
		httpx.ErrNotFound(w, "loan not found")
		return
	}
	httpx.JSON(w, http.StatusOK, l)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
