package api

import (
	"errors"
	"net/http"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/rpc"
)

// writeUpstreamError passes an internal microservice's real status code and
// JSON body straight through to the client (e.g. a 409 conflict stays a 409)
// instead of collapsing every RPC failure into a generic 502. A genuine
// connectivity failure (the service is down, DNS, timeout) isn't an
// *rpc.RPCError and still falls back to 502 Bad Gateway, which is correct.
func writeUpstreamError(w http.ResponseWriter, err error) {
	var rpcErr *rpc.RPCError
	if errors.As(err, &rpcErr) && rpcErr.StatusCode > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(rpcErr.StatusCode)
		_, _ = w.Write([]byte(rpcErr.Body))
		return
	}
	httpx.Error(w, http.StatusBadGateway, err)
}
