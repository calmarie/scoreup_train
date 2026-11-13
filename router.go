package main

import (
	"log/slog"
	"net/http"

	"github.com/calmarie/scoreup_train/internal/healthcheck"
	"github.com/calmarie/scoreup_train/internal/question"
	"github.com/calmarie/scoreup_train/internal/users"
	"github.com/jackc/pgx/v5/pgxpool"
)

func new(
	db *pgxpool.Pool,
	logger *slog.Logger,
) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /health", healthcheck.NewHealthHandler(db, logger))

	usersService := users.NewService(users.NewRepository(db))
	usersHandler := users.NewHandler(usersService, logger)
	usersHandler.RegisterRoutes(mux)

	questionsHandler := question.NewHandler(
		question.NewService(question.NewRepository(db), usersService),
		logger,
	)
	questionsHandler.RegisterRoutes(mux)

	return mux
}
