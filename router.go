package main

import (
	"log/slog"
	"net/http"

	"github.com/calmarie/scoreup_train/internal/healthcheck"
	"github.com/calmarie/scoreup_train/internal/question"
	"github.com/jackc/pgx/v5/pgxpool"
)

func new(
	db *pgxpool.Pool,
	logger *slog.Logger,
) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /health", healthcheck.NewHealthHandler(db, logger))
	handler := question.NewHandler(question.NewService(question.NewRepository(db)), logger)
	handler.RegisterRoutes(mux)

	return mux
}
