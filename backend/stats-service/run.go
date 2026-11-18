package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/calmarie/scoreup_train/backend/stats-service/internal/stats"
	"github.com/calmarie/scoreup_train/backend/stats-service/kafka"
	"github.com/calmarie/scoreup_train/backend/stats-service/postgres"
)

// run treats cancellation by SIGINT or SIGTERM as a clean shutdown.
func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	databaseURL, err := postgres.DatabaseURL()
	if err != nil {
		return fmt.Errorf("read stats database configuration: %w", err)
	}

	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect stats database: %w", err)
	}
	defer db.Close()

	statsService := stats.NewService(stats.NewRepository(db))
	logger.Info("starting stats consumer")
	if err := kafka.Consume(ctx, logger, statsService); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("run stats consumer: %w", err)
	}
	logger.Info("stats consumer stopped")
	return nil
}
