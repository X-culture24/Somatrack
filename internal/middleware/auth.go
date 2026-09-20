package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/cors"

	jwtutil "github.com/stmaryskabete/lms/internal/shared/jwt"
	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/types"
)

type ctxKey string

const (
	CtxClaims   ctxKey = "claims"
	CtxUserID   ctxKey = "user_id"
	CtxRole     ctxKey = "role"
	CtxPortal   ctxKey = "portal"
	CtxScope    ctxKey = "scope"
)

func Auth(jwtMgr *jwtutil.Manager, scope string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok, ok := httpx.ExtractBearerToken(r)
			if !ok {
				httpx.ErrUnauthorized(w, "missing bearer token")
				return
			}
			claims, err := jwtMgr.ValidateAccess(tok, scope)
			if err != nil {
				httpx.ErrUnauthorized(w, "invalid or expired token")
				return
			}
			ctx := r.Context()
			ctx = context.WithValue(ctx, CtxClaims, claims)
			ctx = context.WithValue(ctx, CtxUserID, claims.UserID)
			ctx = context.WithValue(ctx, CtxRole, claims.Role)
			ctx = context.WithValue(ctx, CtxPortal, claims.Portal)
			ctx = context.WithValue(ctx, CtxScope, claims.Scope)
			// Also set plain string keys: most api-gateway proxy handlers read
			// r.Context().Value("user_id") / .Value("role") with untyped string
			// literals, which do not match the ctxKey-typed constants above.
			ctx = context.WithValue(ctx, "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "role", claims.Role)
			ctx = context.WithValue(ctx, "portal", claims.Portal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func roleIn(r Role, list []types.Role) bool {
	for _, v := range list {
		if types.Role(r) == v {
			return true
		}
	}
	return false
}

type Role string

func (r Role) String() string { return string(r) }

func getRole(ctx context.Context) types.Role {
	v, ok := ctx.Value(CtxRole).(types.Role)
	if !ok {
		return ""
	}
	return v
}

func IsAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Value(CtxUserID).(stringOrUUID); !ok {
			// try uuid
			uid, ok2 := r.Context().Value(CtxUserID).(stringOrUUID)
			_ = uid
			if !ok2 {
				httpx.ErrUnauthorized(w, "user required")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

type stringOrUUID interface{}

func RequireRoles(allowed ...types.Role) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := getRole(r.Context())
			ok := false
			for _, a := range allowed {
				if a == role {
					ok = true
					break
				}
			}
			if !ok {
				httpx.ErrForbidden(w, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func IsAdminPortal(next http.Handler) http.Handler {
	return RequireAdminOrFinance(true, false, false, false)(next)
}

func IsFinancePortal(next http.Handler) http.Handler {
	return RequireAdminOrFinance(false, true, false, false)(next)
}

func IsTeacherPortal(next http.Handler) http.Handler {
	return RequireAdminOrFinance(false, false, true, false)(next)
}

func IsParent(next http.Handler) http.Handler {
	return RequireAdminOrFinance(false, false, false, true)(next)
}

func RequireAdminOrFinance(requireAdmin, requireFinance, requireTeacher, requireParent bool) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := getRole(r.Context())
			ok := false
			if requireAdmin {
				for _, a := range types.AdminRoles {
					if a == role {
						ok = true
						break
					}
				}
				if role == types.RoleSystemAdmin {
					ok = true
				}
			}
			if !ok && requireFinance {
				for _, a := range types.FinanceRoles {
					if a == role {
						ok = true
						break
					}
				}
			}
			if !ok && requireTeacher {
				for _, a := range types.TeacherRoles {
					if a == role {
						ok = true
						break
					}
				}
			}
			if !ok && requireParent {
				ok = role == types.RoleParent
			}
			if !ok {
				httpx.ErrForbidden(w, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func IsAdminOrReadOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m := strings.ToUpper(r.Method)
		if m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		role := getRole(r.Context())
		ok := false
		for _, a := range types.AdminRoles {
			if a == role {
				ok = true
				break
			}
		}
		if role == types.RoleSystemAdmin {
			ok = true
		}
		if !ok {
			httpx.ErrForbidden(w, "admin role required for write operations")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func CORS(allowedOrigins []string) func(next http.Handler) http.Handler {
	opts := cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           86400,
	}
	return cors.Handler(opts)
}

func Security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		next.ServeHTTP(w, r)
	})
}
