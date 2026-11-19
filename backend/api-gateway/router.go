package main

import (
	"log/slog"
	"net/http"

	"github.com/calmarie/scoreup_train/backend/api-gateway/handler/stats_handler"
	training_handler "github.com/calmarie/scoreup_train/backend/api-gateway/handler/trainging_handler"
	"github.com/calmarie/scoreup_train/backend/api-gateway/handler/user_handler"
	statspb "github.com/calmarie/scoreup_train/backend/gen/stats"
	trainingpb "github.com/calmarie/scoreup_train/backend/gen/training"
	userpb "github.com/calmarie/scoreup_train/backend/gen/user"
)

func newRouter(
	userClient userpb.UserClient,
	trainingClient trainingpb.TrainingClient,
	statsClient statspb.StatsClient,
	logger *slog.Logger,
) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	userHandler := user_handler.NewHandler(userClient, logger)
	userHandler.RegisterRoutes(mux)

	trainingHandler := training_handler.NewHandler(trainingClient, userClient, logger)
	trainingHandler.RegisterRoutes(mux)

	statsHandler := stats_handler.NewHandler(userClient, statsClient, logger)
	statsHandler.RegisterRoutes(mux)

	return mux
}
