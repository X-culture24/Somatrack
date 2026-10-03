// Package shortlink issues the short random codes carried in SMS links
// (…/r#<code>). Codes are 128-bit random, stored only as a SHA-256 hash in
// short_links, single-purpose and expiring; opening one still needs a portal
// login (see internal/api/links.go). It's a package of its own so services
// that send SMS (payment-svc) can issue links without going through the API
// gateway — they share the database.
package shortlink

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/signedlink"
)

// TTL is how long an SMS link keeps working.
const TTL = 30 * 24 * time.Hour

// Hash is what short_links.code_hash stores for code.
func Hash(code string) []byte {
	h := sha256.Sum256([]byte(strings.TrimSpace(code)))
	return h[:]
}

// Issue stores a new code for record (belonging to student) and returns the
// URL to send: portalURL + "/r#" + code.
func Issue(ctx context.Context, pool *pgxpool.Pool, portalURL string, purpose signedlink.Purpose, record, student uuid.UUID) (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	code := base64.RawURLEncoding.EncodeToString(raw)
	_, err := pool.Exec(ctx, `
		INSERT INTO short_links (code_hash, purpose, record_id, student_id, expires_at)
		VALUES ($1, $2, $3, $4, $5)`,
		Hash(code), int16(purpose), record, student, time.Now().Add(TTL))
	if err != nil {
		return "", err
	}
	return strings.TrimRight(portalURL, "/") + "/r#" + code, nil
}

// Revoke kills every outstanding link for a record — e.g. a reversed payment.
func Revoke(ctx context.Context, pool *pgxpool.Pool, purpose signedlink.Purpose, record uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE short_links SET revoked_at = now()
		WHERE purpose = $1 AND record_id = $2 AND revoked_at IS NULL`, int16(purpose), record)
	return err
}
