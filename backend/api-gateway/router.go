package main

import (
	"log/slog"
	"net/http"

	"github.com/calmarie/scoreup_train/backend/api-gateway/handler/stats_handler"
	training_handler "github.com/calmarie/scoreup_train/backend/api-gateway/handler/trainging_handler"
	"github.com/calmarie/scoreup_train/backend/api-gateway/handler/user_handler"
	"github.com/calmarie/scoreup_train/backend/api-gateway/middleware"
	statspb "github.com/calmarie/scoreup_train/backend/gen/stats"
	trainingpb "github.com/calmarie/scoreup_train/backend/gen/training"
	userpb "github.com/calmarie/scoreup_train/backend/gen/user"
)

func newRouter(
	userClient userpb.UserClient,
	trainingClient trainingpb.TrainingClient,
	statsClient statspb.StatsClient,
	verifier *middleware.Verifier,
	logger *slog.Logger,
) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	userHandler := user_handler.NewHandler(userClient, logger)
	userHandler.RegisterRoutes(mux)

	trainingHandler := training_handler.NewHandler(trainingClient, logger)
	protectedMux := http.NewServeMux()
	trainingHandler.RegisterRoutes(protectedMux)

	statsHandler := stats_handler.NewHandler(statsClient, logger)
	statsHandler.RegisterRoutes(protectedMux)

	auth := middleware.New(verifier)
	protected := auth.AuthMiddleware(protectedMux)
	mux.Handle("/api/v1/question", protected)
	mux.Handle("/api/v1/question/", protected)
	mux.Handle("/api/v1/stats/", protected)

	return mux
}
