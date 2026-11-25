package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/calmarie/scoreup_train/backend/gen/training"
	"github.com/calmarie/scoreup_train/backend/training-service/internal/answer"
	"github.com/calmarie/scoreup_train/backend/training-service/internal/handler"
	"github.com/calmarie/scoreup_train/backend/training-service/internal/outbox"
	"github.com/calmarie/scoreup_train/backend/training-service/internal/question"
	"github.com/calmarie/scoreup_train/backend/training-service/kafka"
	"github.com/calmarie/scoreup_train/backend/training-service/postgres"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func run(logger *slog.Logger) error {

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		return fmt.Errorf("error on listener :50052 creation: %w", err)
	}
	defer lis.Close()

	url, err := postgres.DatabaseURL()
	if err != nil {
		return fmt.Errorf("error get database URL: %w", err)
	}

	db, err := postgres.Connect(ctx, url)
	if err != nil {
		return fmt.Errorf("error connect database: %w", err)
	}
	defer db.Close()

	kafkaClient, err := kafka.NewProducerClient()
	if err != nil {
		return fmt.Errorf("initialize Kafka producer: %w", err)
	}
	defer kafkaClient.Close()

	relay := outbox.NewRelay(outbox.NewRepository(db))
	go relay.Relay(ctx, kafkaClient, logger)

	questionRepo := question.NewRepository(db)
	questionService := question.NewService(questionRepo)
	answerService := answer.NewService(answer.NewRepository(db), questionRepo)

	creds, err := credentials.NewServerTLSFromFile(
		"certs/training.crt",
		"certs/training.key",
	)
	if err != nil {
		return fmt.Errorf("load training gRPC certificate: %w", err)
	}

	grpcServer := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterTrainingServer(grpcServer, handler.NewHandler(questionService, answerService))

	serveErr := make(chan error, 1)

	go func() {
		serveErr <- grpcServer.Serve(lis)
	}()

	logger.Info("training gRPC server started", "address", lis.Addr())

	select {
	case <-ctx.Done():
		logger.Info("stopping training gRPC server")
		grpcServer.GracefulStop()
		return nil

	case err := <-serveErr:
		return fmt.Errorf("serve training gRPC: %w", err)
	}
}
