package api

import (
	"errors"
	"net/http"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/rpc"
	"github.com/stmaryskabete/lms/internal/shared/types"
)

// WriteUpstreamError passes an internal microservice's real status code and
// JSON body straight through to the client (e.g. a 409 conflict stays a 409)
// instead of collapsing every RPC failure into a generic 502. A genuine
// connectivity failure (the service is down, DNS, timeout) isn't an
// *rpc.RPCError and still falls back to 502 Bad Gateway, which is correct.
func WriteUpstreamError(w http.ResponseWriter, err error) {
	var rpcErr *rpc.RPCError
	if errors.As(err, &rpcErr) && rpcErr.StatusCode > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(rpcErr.StatusCode)
		_, _ = w.Write([]byte(rpcErr.Body))
		return
	}
	httpx.Error(w, http.StatusBadGateway, err)
}

// adminOnly gates a handler to admin-tier roles. Shared by the direct-to-DB
// gateway resources (academics, students, guardians, admissions) — mirrors
// the same check fpProxy/financeProxy each keep locally.
func adminOnly() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value("role").(types.Role)
			ok := role == types.RoleSystemAdmin
			if !ok {
				for _, a := range types.AdminRoles {
					if a == role {
						ok = true
						break
					}
				}
			}
			if !ok {
				httpx.ErrForbidden(w, "admin role required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// financeOnly mirrors adminOnly above but for finance-tier roles — used by
// the reports/fee-collection endpoint (internal/api/reports.go).
func financeOnly() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value("role").(types.Role)
			ok := false
			for _, a := range types.FinanceRoles {
				if a == role {
					ok = true
					break
				}
			}
			if !ok {
				httpx.ErrForbidden(w, "finance role required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
