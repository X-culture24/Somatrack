package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
)

// UploadsService handles document uploads for admission/registration
// (birth certificate, guardian ID, pupil passport photo). Deliberately
// narrow: a fixed category allowlist (never a client-supplied path
// segment), a server-generated filename (never the client's original
// filename — that's how path traversal / overwrite attacks happen), and
// the file's actual sniffed content type checked against an allowlist
// rather than trusting the client's Content-Type header or extension.
type UploadsService struct {
	mediaRoot string
	mediaURL  string
}

func NewUploadsService(mediaRoot, mediaURL string) *UploadsService {
	return &UploadsService{mediaRoot: mediaRoot, mediaURL: mediaURL}
}

func (s *UploadsService) RegisterRoutes(r chi.Router) {
	r.Post("/uploads/", s.upload)
}

const maxUploadBytes = 10 << 20 // 10MB

var allowedUploadCategories = map[string]bool{
	"birth_cert":  true,
	"guardian_id": true,
	"photo":       true,
	// staff registration documents
	"staff_id":           true,
	"certificate":        true,
	"appointment_letter": true,
	"good_conduct":       true,
}

var allowedUploadTypes = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"application/pdf": ".pdf",
}

func (s *UploadsService) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+1<<20)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		httpx.ErrBadRequest(w, "file too large or malformed upload (max 10MB)")
		return
	}

	category := r.FormValue("category")
	if !allowedUploadCategories[category] {
		httpx.ErrBadRequest(w, "category must be one of: birth_cert, guardian_id, photo, staff_id, certificate, appointment_letter, good_conduct")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		httpx.ErrBadRequest(w, "missing file field")
		return
	}
	defer file.Close()

	sniff := make([]byte, 512)
	n, _ := io.ReadFull(file, sniff)
	sniff = sniff[:n]
	contentType := http.DetectContentType(sniff)
	ext, ok := allowedUploadTypes[contentType]
	if !ok {
		httpx.ErrBadRequest(w, "file must be a JPEG, PNG, or PDF")
		return
	}

	dir := filepath.Join(s.mediaRoot, category)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	filename := uuid.New().String() + ext
	destPath := filepath.Join(dir, filename)

	dest, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer dest.Close()

	if _, err := dest.Write(sniff); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if _, err := io.Copy(dest, io.LimitReader(file, maxUploadBytes)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}

	relPath := category + "/" + filename
	httpx.JSON(w, http.StatusCreated, map[string]string{
		"path": relPath,
		"url":  s.mediaURL + relPath,
	})
}
