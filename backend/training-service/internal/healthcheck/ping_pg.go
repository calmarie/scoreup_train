package healthcheck

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const healthCheckTimeout = 2 * time.Second

type HealthHandler struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

func NewHealthHandler(db *pgxpool.Pool, logger *slog.Logger) *HealthHandler {
	return &HealthHandler{
		db:     db,
		logger: logger,
	}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), healthCheckTimeout)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		h.logger.Error("postgres health check failed", "error", err)
		writeHealthResponse(w, http.StatusServiceUnavailable, "unhealthy")
		return
	}

	writeHealthResponse(w, http.StatusOK, "ok")
}

func writeHealthResponse(w http.ResponseWriter, statusCode int, status string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(map[string]string{"status": status}); err != nil {
		slog.Error("write health response", "error", err)
	}
}
