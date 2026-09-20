package httpx

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
)

func ServiceKeyMiddleware(serviceKey string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			k := r.Header.Get("X-Service-Key")
			if serviceKey == "" || k == "" {
				ErrUnauthorized(w, "missing service key")
				return
			}
			if !constantTimeEqual(k, serviceKey) {
				ErrUnauthorized(w, "invalid service key")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// constantTimeEqual compares two strings without leaking their lengths (or
// the position of the first mismatching byte) via timing. Hashing both to a
// fixed size first, rather than comparing raw bytes, means the classic
// early-return-on-length-mismatch short-circuit can't leak how long the
// expected secret is either.
func constantTimeEqual(a, b string) bool {
	ah := sha256.Sum256([]byte(a))
	bh := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(ah[:], bh[:]) == 1
}

func ExtractBearerToken(r *http.Request) (string, bool) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", false
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	return parts[1], true
}
