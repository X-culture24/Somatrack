package api

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/time/rate"

	lmsmiddleware "github.com/stmaryskabete/lms/internal/middleware"
	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shortlink"
	"github.com/stmaryskabete/lms/internal/signedlink"
)

// LinksService owns the two kinds of link the school hands out:
//
//   - short SMS codes (random, stored hashed in short_links, expire, need a
//     portal login to use), and
//   - Ed25519-signed QR tokens printed on documents, checked by the public
//     /verify page against the published keys.
//
// Both travel in the URL fragment (…/r#code, …/verify#token). Browsers never
// send the fragment to a server, so neither ever lands in access logs or
// Referer headers; the portal page reads it and POSTs it here in the body.
type LinksService struct {
	pool      *pgxpool.Pool
	keys      *signedlink.Keyring // nil = document signing switched off
	portalURL string
}

func NewLinksService(pool *pgxpool.Pool, keys *signedlink.Keyring, portalURL string) *LinksService {
	return &LinksService{pool: pool, keys: keys, portalURL: strings.TrimRight(portalURL, "/")}
}

// RegisterPublicRoutes mounts the unauthenticated endpoints on the root router.
func (s *LinksService) RegisterPublicRoutes(r chi.Router) {
	r.Get("/.well-known/stmarys-keys.json", s.publicKeys)
	// A genuine check is one scan; 20/min with a small burst stops anyone
	// using the endpoint to probe for valid tokens.
	r.With(lmsmiddleware.RateLimit(rate.Limit(20.0/60.0), 10)).Post("/api/verify/", s.verify)
}

// RegisterRoutes mounts the endpoints that need a portal login.
func (s *LinksService) RegisterRoutes(r chi.Router) {
	r.With(lmsmiddleware.RateLimit(rate.Limit(10.0/60.0), 10)).Post("/links/resolve/", s.resolve)
}

// ---------------------------------------------------------------------------
// Signed document tokens (QR codes)

// ReceiptVerifyURL returns the URL to encode in a receipt's QR code. The
// amount is signed into the token, so an edited printout can't carry a valid
// code for a different figure.
func (s *LinksService) ReceiptVerifyURL(paymentID uuid.UUID, amount float64) (string, error) {
	extra := make([]byte, 8)
	binary.BigEndian.PutUint64(extra, uint64(math.Round(amount*100)))
	tok, err := s.keys.Issue(signedlink.PurposeReceipt, paymentID, 0, extra)
	if err != nil {
		return "", err
	}
	return s.portalURL + "/verify#" + tok, nil
}

// ResultsVerifyURL returns the QR URL for an academic record (transcript).
// The token carries the issue time, the number of terms and the latest term
// mean, so the verify page can say exactly which printout it vouches for.
func (s *LinksService) ResultsVerifyURL(studentID uuid.UUID, terms int, latestMean float64) (string, error) {
	extra := make([]byte, 7)
	binary.BigEndian.PutUint32(extra[0:4], uint32(time.Now().Unix()))
	extra[4] = byte(min(terms, 255))
	binary.BigEndian.PutUint16(extra[5:7], uint16(math.Round(latestMean*10)))
	tok, err := s.keys.Issue(signedlink.PurposeResults, studentID, 0, extra)
	if err != nil {
		return "", err
	}
	return s.portalURL + "/verify#" + tok, nil
}

type verifyResponse struct {
	Valid           bool    `json:"valid"`
	Reason          string  `json:"reason,omitempty"`
	Document        string  `json:"document,omitempty"`
	School          string  `json:"school,omitempty"`
	Number          string  `json:"number,omitempty"`
	Amount          float64 `json:"amount,omitempty"`
	Date            string  `json:"date,omitempty"`
	StudentInitials string  `json:"student_initials,omitempty"`
	Status          string  `json:"status,omitempty"`
}

func (s *LinksService) publicKeys(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=3600")
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"issuer": schoolName,
		"keys":   s.keys.PublicKeys(),
	})
}

// verify answers "is this document genuine?" for anyone holding it. It says
// as little as that question needs: no phone numbers, balances or full names.
func (s *LinksService) verify(w http.ResponseWriter, r *http.Request) {
	if s.keys == nil {
		httpx.Error(w, http.StatusServiceUnavailable, errors.New("document verification is not configured"))
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&body); err != nil || body.Token == "" {
		httpx.ErrBadRequest(w, "token required")
		return
	}

	c, err := s.keys.Verify(body.Token, signedlink.PurposeReceipt)
	if errors.Is(err, signedlink.ErrPurpose) {
		if rc, rerr := s.keys.Verify(body.Token, signedlink.PurposeResults); rerr == nil {
			s.verifyResults(w, r, rc)
			return
		}
	}
	if err != nil {
		reason := "This code was not issued by " + schoolName + ", or it has been altered."
		if errors.Is(err, signedlink.ErrExpired) {
			reason = "This code has expired."
		}
		httpx.JSON(w, http.StatusOK, verifyResponse{Valid: false, Reason: reason})
		return
	}

	var receiptNo, status, first, last string
	var amount float64
	var paidOn time.Time
	err = s.pool.QueryRow(r.Context(), `
		SELECT p.receipt_no, p.amount, p.payment_date, p.status,
		       COALESCE(st.first_name,''), COALESCE(st.last_name,'')
		FROM payments p JOIN students st ON st.id = p.student_id
		WHERE p.id = $1`, c.RecordID).Scan(&receiptNo, &amount, &paidOn, &status, &first, &last)
	if errors.Is(err, pgx.ErrNoRows) {
		// Signed by us but the record is gone: treat as not valid rather
		// than vouching for something the school can no longer back.
		httpx.JSON(w, http.StatusOK, verifyResponse{Valid: false, Reason: "This receipt is no longer on record. Contact the school bursar."})
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}

	signedAmount := amount
	if len(c.Extra) == 8 {
		signedAmount = float64(binary.BigEndian.Uint64(c.Extra)) / 100
	}
	resp := verifyResponse{
		Valid:           true,
		Document:        "Payment receipt",
		School:          schoolName,
		Number:          receiptNo,
		Amount:          signedAmount,
		Date:            paidOn.Format("02 Jan 2006"),
		StudentInitials: initials(first, last),
		Status:          "genuine",
	}
	// The signature proves we issued it; the live record decides whether it
	// still stands. A reversed/refunded payment invalidates its receipt
	// without needing a separate revocation list.
	switch {
	case status != "confirmed":
		resp.Valid = false
		resp.Status = status
		resp.Reason = fmt.Sprintf("This receipt was issued but the payment is now marked %s.", status)
	case math.Abs(signedAmount-amount) >= 0.005:
		resp.Valid = false
		resp.Status = "amount_changed"
		resp.Reason = "The amount on record no longer matches this receipt. Contact the school bursar."
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// verifyResults answers for a transcript QR code: whose record, when it was
// issued and what it covered — never the marks themselves.
func (s *LinksService) verifyResults(w http.ResponseWriter, r *http.Request, c *signedlink.Claims) {
	var first, last string
	err := s.pool.QueryRow(r.Context(), `SELECT COALESCE(first_name,''), COALESCE(last_name,'') FROM students WHERE id = $1`,
		c.RecordID).Scan(&first, &last)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusOK, verifyResponse{Valid: false, Reason: "This record is no longer on file. Contact the school."})
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	resp := verifyResponse{Valid: true, Document: "Academic record", School: schoolName,
		StudentInitials: initials(first, last), Status: "genuine"}
	if len(c.Extra) == 7 {
		resp.Date = time.Unix(int64(binary.BigEndian.Uint32(c.Extra[0:4])), 0).In(nairobiTZ).Format("02 Jan 2006")
		resp.Number = fmt.Sprintf("%d term(s), latest mean %.1f%%", c.Extra[4], float64(binary.BigEndian.Uint16(c.Extra[5:7]))/10)
	}
	httpx.JSON(w, http.StatusOK, resp)
}

var nairobiTZ = func() *time.Location {
	if l, err := time.LoadLocation("Africa/Nairobi"); err == nil {
		return l
	}
	return time.FixedZone("EAT", 3*3600)
}()

func initials(first, last string) string {
	var out []string
	for _, n := range []string{first, last} {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, strings.ToUpper(n[:1])+".")
		}
	}
	return strings.Join(out, " ")
}

// ---------------------------------------------------------------------------
// Short SMS codes

// IssueShortLink stores a new random code for record (belonging to student)
// and returns the URL to put in an SMS.
func (s *LinksService) IssueShortLink(ctx context.Context, purpose signedlink.Purpose, record, student uuid.UUID) (string, error) {
	return shortlink.Issue(ctx, s.pool, s.portalURL, purpose, record, student)
}

// RevokeShortLinks kills every outstanding SMS link for a record.
func (s *LinksService) RevokeShortLinks(ctx context.Context, purpose signedlink.Purpose, record uuid.UUID) error {
	return shortlink.Revoke(ctx, s.pool, purpose, record)
}

// resolve turns an SMS code into the portal page it points at. The caller
// must be logged in and allowed to see the student, so a forwarded or
// leaked SMS alone opens nothing.
func (s *LinksService) resolve(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256)).Decode(&body); err != nil || body.Code == "" {
		httpx.ErrBadRequest(w, "code required")
		return
	}
	hash := shortlink.Hash(body.Code)

	var purpose int16
	var record, student uuid.UUID
	var expires time.Time
	var revoked *time.Time
	err := s.pool.QueryRow(r.Context(), `
		SELECT purpose, record_id, student_id, expires_at, revoked_at
		FROM short_links WHERE code_hash = $1`, hash).Scan(&purpose, &record, &student, &expires, &revoked)
	// Unknown, expired and revoked all get the same answer so the endpoint
	// doesn't confirm which codes once existed.
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && (revoked != nil || time.Now().After(expires))) {
		httpx.ErrNotFound(w, "this link has expired or is no longer valid")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if !canViewStudent(r, s.pool, student) {
		httpx.ErrForbidden(w, "this link belongs to a different account")
		return
	}
	_, _ = s.pool.Exec(r.Context(), `
		UPDATE short_links SET use_count = use_count + 1, last_used_at = now() WHERE code_hash = $1`, hash)

	target := "/"
	switch signedlink.Purpose(purpose) {
	case signedlink.PurposeReceipt:
		target = "/parent/fees?receipt=" + record.String()
	case signedlink.PurposeStatement:
		target = "/parent/fees?invoice=" + record.String()
	case signedlink.PurposeResults:
		target = "/parent/progress"
	}
	httpx.JSON(w, http.StatusOK, map[string]string{
		"purpose":   signedlink.Purpose(purpose).String(),
		"record_id": record.String(),
		"target":    target,
	})
}
