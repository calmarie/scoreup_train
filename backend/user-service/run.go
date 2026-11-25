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
	"github.com/calmarie/scoreup_train/backend/user-service/internal/auth"
	"github.com/calmarie/scoreup_train/backend/user-service/internal/auth/access"
	"github.com/calmarie/scoreup_train/backend/user-service/internal/auth/refresh"
	"github.com/calmarie/scoreup_train/backend/user-service/internal/handler"
	"github.com/calmarie/scoreup_train/backend/user-service/internal/users"
	"github.com/calmarie/scoreup_train/backend/user-service/postgres"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// run drains in-flight RPCs before closing the database pool on shutdown.
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

	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		return fmt.Errorf("REDIS_URL is required")
	}
	rdb := refresh.New()
	defer rdb.Close()
	if err := refresh.CheckConn(ctx, rdb); err != nil {
		return err
	}

	privateKeyPEM, err := os.ReadFile("certs/private.pem")
	if err != nil {
		return fmt.Errorf("read JWT private key: %w", err)
	}
	signer, err := access.NewSigner(privateKeyPEM)
	if err != nil {
		return err
	}

	issuer := auth.NewIssuer(signer, refresh.NewRedisStore(rdb))
	usersService := users.NewService(users.NewRepository(db), issuer)

	creds, err := credentials.NewServerTLSFromFile(
		"certs/user.crt",
		"certs/user.key",
	)
	if err != nil {
		return fmt.Errorf("load user gRPC certificate: %w", err)
	}

	grpcServer := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterUserServer(grpcServer, handler.NewHandler(usersService))

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
