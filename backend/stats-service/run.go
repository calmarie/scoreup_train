package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/calmarie/scoreup_train/backend/stats-service/kafka"
	"github.com/calmarie/scoreup_train/backend/stats-service/postgres"
)

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

	logger.Info("starting stats consumer")
	if err := kafka.Consume(ctx, logger, db); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("run stats consumer: %w", err)
	}
	logger.Info("stats consumer stopped")
	return nil
}
