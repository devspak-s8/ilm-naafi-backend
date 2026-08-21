package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ilmnafi/backend/internal/config"
)

type HealthChecker struct {
	db     *sql.DB
	config *config.Config
}

func NewHealthChecker(db *sql.DB, cfg *config.Config) *HealthChecker {
	return &HealthChecker{db: db, config: cfg}
}

func (h *HealthChecker) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	status := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
	}

	if err := h.db.PingContext(ctx); err != nil {
		status["status"] = "unhealthy"
		status["database"] = "unhealthy"
		status["error"] = "database connection failed"
		w.WriteHeader(http.StatusServiceUnavailable)
		writeJSON(w, status)
		return
	}

	status["database"] = "healthy"
	w.WriteHeader(http.StatusOK)
	writeJSON(w, status)
}

func (h *HealthChecker) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	status := map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	if err := h.db.PingContext(ctx); err != nil {
		status["status"] = "not ready"
		status["database"] = "not ready"
		w.WriteHeader(http.StatusServiceUnavailable)
		writeJSON(w, status)
		return
	}

	var version string
	if err := h.db.QueryRowContext(ctx, "SELECT version()").Scan(&version); err != nil {
		status["status"] = "not ready"
		status["database"] = "not ready"
		w.WriteHeader(http.StatusServiceUnavailable)
		writeJSON(w, status)
		return
	}

	status["database"] = "ready"
	status["database_version"] = version
	w.WriteHeader(http.StatusOK)
	writeJSON(w, status)
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}
