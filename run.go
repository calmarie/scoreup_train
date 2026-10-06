package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/calmarie/scoreup_train/internal/config"
	"github.com/calmarie/scoreup_train/internal/postgres"
)

// listen and serve func
func run(logger *slog.Logger) error {
	databaseURL, err := config.DbURL("POSTGRES_DB")
	if err != nil {
		return fmt.Errorf("read postgres configuration: %w", err)
	}

	httpPort, err := config.RequiredEnv("HTTP_PORT")
	if err != nil {
		return fmt.Errorf("read HTTP configuration: %w", err)
	}

	db, err := postgres.Connect(context.Background(), databaseURL)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer db.Close()

	server := &http.Server{
		Addr:              ":" + httpPort,
		Handler:           new(db, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logger.Info("service started", "address", server.Addr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serve HTTP: %w", err)
	}

	return nil
}
