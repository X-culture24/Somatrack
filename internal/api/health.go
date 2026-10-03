package api

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
)

type ServiceHealth struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Status string `json:"status"` // "ok" | "degraded" | "unreachable"
	Error  string `json:"error,omitempty"`
}

type HealthResponse struct {
	Status   string          `json:"status"` // overall: "ok" if gateway DB + every downstream service is up
	Service  string          `json:"service"`
	Time     string          `json:"time"`
	Database string          `json:"database"`
	Services []ServiceHealth `json:"services"`
}

// HealthHandler checks the gateway's own DB connection plus every downstream
// service's /health endpoint (in parallel, with a short timeout each) so a
// single GET /api/health/ tells you what's actually up without hitting each
// service's port by hand.
func HealthHandler(pool *pgxpool.Pool, services map[string]string) http.HandlerFunc {
	client := &http.Client{Timeout: 2 * time.Second}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		dbStatus := "ok"
		if err := pool.Ping(ctx); err != nil {
			dbStatus = "degraded"
		}

		names := make([]string, 0, len(services))
		for name := range services {
			names = append(names, name)
		}
		sort.Strings(names)

		results := make([]ServiceHealth, len(names))
		var wg sync.WaitGroup
		for i, name := range names {
			i, name, url := i, name, services[name]
			wg.Add(1)
			go func() {
				defer wg.Done()
				results[i] = checkService(ctx, client, name, url)
			}()
		}
		wg.Wait()

		overall := "ok"
		if dbStatus != "ok" {
			overall = "degraded"
		}
		for _, s := range results {
			if s.Status != "ok" {
				overall = "degraded"
			}
		}

		httpx.JSON(w, http.StatusOK, HealthResponse{
			Status:   overall,
			Service:  "api-gateway",
			Time:     time.Now().UTC().Format(time.RFC3339),
			Database: dbStatus,
			Services: results,
		})
	}
}

func checkService(ctx context.Context, client *http.Client, name, baseURL string) ServiceHealth {
	sh := ServiceHealth{Name: name, URL: baseURL}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/health", nil)
	if err != nil {
		sh.Status = "unreachable"
		sh.Error = err.Error()
		return sh
	}
	resp, err := client.Do(req)
	if err != nil {
		sh.Status = "unreachable"
		sh.Error = err.Error()
		return sh
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		sh.Status = "ok"
	} else {
		sh.Status = "degraded"
		sh.Error = fmt.Sprintf("upstream returned status %d", resp.StatusCode)
	}
	return sh
}
