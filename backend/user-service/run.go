package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/calmarie/scoreup_train/backend/gen/user"
	"github.com/calmarie/scoreup_train/backend/user-service/internal/users"
	"github.com/calmarie/scoreup_train/backend/user-service/postgres"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type userGRPCserver struct {
	pb.UnimplementedUserServer
	usersService *users.Service
}

func run(logger *slog.Logger) error {

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		return fmt.Errorf("error on listener :50053 creation: %w", err)
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

	usersService := users.NewService(users.NewRepository(db))
	srv := &userGRPCserver{usersService: usersService}

	creds, err := credentials.NewServerTLSFromFile(
		"certs/user.crt",
		"certs/user.key",
	)
	if err != nil {
		return fmt.Errorf("load user gRPC certificate: %w", err)
	}

	grpcServer := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterUserServer(grpcServer, srv)

	serveErr := make(chan error, 1)

	go func() {
		serveErr <- grpcServer.Serve(lis)
	}()

	logger.Info("user gRPC server started", "address", lis.Addr())

	select {
	case <-ctx.Done():
		logger.Info("stopping user gRPC server")
		grpcServer.GracefulStop()
		return nil

	case err := <-serveErr:
		return fmt.Errorf("serve user gRPC: %w", err)
	}
}
