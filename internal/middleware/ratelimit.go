package middleware

import (
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
)

// ipRateLimiter tracks one token-bucket limiter per client IP. It's a simple
// in-memory guard against naive brute forcing of low-frequency, high-value
// endpoints like login — it does not share state across api-gateway
// replicas, so a horizontally scaled deployment should back this with Redis
// instead once that matters.
type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	lastSeen map[string]time.Time
	r        rate.Limit
	b        int
}

func newIPRateLimiter(r rate.Limit, b int) *ipRateLimiter {
	l := &ipRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		lastSeen: make(map[string]time.Time),
		r:        r,
		b:        b,
	}
	go l.cleanupLoop()
	return l
}

func (l *ipRateLimiter) getLimiter(key string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	lim, ok := l.limiters[key]
	if !ok {
		lim = rate.NewLimiter(l.r, l.b)
		l.limiters[key] = lim
	}
	l.lastSeen[key] = time.Now()
	return lim
}

func (l *ipRateLimiter) cleanupLoop() {
	for {
		time.Sleep(10 * time.Minute)
		cutoff := time.Now().Add(-30 * time.Minute)
		l.mu.Lock()
		for key, seen := range l.lastSeen {
			if seen.Before(cutoff) {
				delete(l.limiters, key)
				delete(l.lastSeen, key)
			}
		}
		l.mu.Unlock()
	}
}

// RateLimit throttles requests per client IP to r events/sec with burst b.
// Apply it to sensitive, low-frequency routes (login, password reset) where
// brute forcing is a realistic threat, not to general API traffic.
func RateLimit(r rate.Limit, b int) func(next http.Handler) http.Handler {
	limiter := newIPRateLimiter(r, b)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if !limiter.getLimiter(clientIP(req)).Allow() {
				w.Header().Set("Retry-After", "30")
				httpx.Error(w, http.StatusTooManyRequests, errors.New("too many requests, try again later"))
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
