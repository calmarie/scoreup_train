package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/calmarie/scoreup_train/backend/gen/stats"
	"github.com/calmarie/scoreup_train/backend/stats-service/internal/handler"
	"github.com/calmarie/scoreup_train/backend/stats-service/internal/stats"
	"github.com/calmarie/scoreup_train/backend/stats-service/kafka"
	"github.com/calmarie/scoreup_train/backend/stats-service/postgres"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// run treats cancellation by SIGINT or SIGTERM as a clean shutdown.
func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	listener, err := net.Listen("tcp", ":50054")
	if err != nil {
		return fmt.Errorf("listen for stats gRPC: %w", err)
	}
	defer listener.Close()

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
	creds, err := credentials.NewServerTLSFromFile("certs/stats.crt", "certs/stats.key")
	if err != nil {
		return fmt.Errorf("load stats gRPC certificate: %w", err)
	}
	grpcServer := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterStatsServer(grpcServer, handler.NewHandler(statsService))

	workers, runCtx := errgroup.WithContext(ctx)
	workers.Go(func() error {
		logger.Info("stats gRPC server started", "address", listener.Addr())
		if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			return fmt.Errorf("serve stats gRPC: %w", err)
		}
		return nil
	})
	workers.Go(func() error {
		logger.Info("starting stats consumer")
		if err := kafka.Consume(runCtx, logger, statsService); err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("run stats consumer: %w", err)
		}
		logger.Info("stats consumer stopped")
		return nil
	})
	workers.Go(func() error {
		<-runCtx.Done()
		logger.Info("stopping stats gRPC server")
		grpcServer.GracefulStop()
		return nil
	})
	return workers.Wait()
}
