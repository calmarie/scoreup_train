package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	trainingpb "github.com/calmarie/scoreup_train/backend/gen/training"
	userpb "github.com/calmarie/scoreup_train/backend/gen/user"
	"github.com/calmarie/scoreup_train/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func run(logger *slog.Logger) error {
	httpPort, err := config.RequiredEnv("HTTP_PORT")
	if err != nil {
		return fmt.Errorf("read HTTP configuration: %w", err)
	}

	userClient, trainingClient, closeConns, err := connGRPCServers()
	if err != nil {
		return err
	}
	defer closeConns()

	server := &http.Server{
		Addr: ":" + httpPort,
		Handler: newRouter(
			userClient,
			trainingClient,
			logger,
		),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logger.Info("gateway started", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	return nil
}

func connGRPCServers() (userpb.UserClient, trainingpb.TrainingClient, func(), error) {
	creds, err := credentials.NewClientTLSFromFile("certs/ca.pem", "")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load gRPC CA: %w", err)
	}

	userTarget, err := config.RequiredEnv("USER_SERVICE_URL")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("read USER_SERVICE_URL: %w", err)
	}

	trainingTarget, err := config.RequiredEnv("TRAINING_SERVICE_URL")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("read TRAINING_SERVICE_URL: %w", err)
	}

	userConn, err := grpc.NewClient(userTarget, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create user grpc connection error: %w", err)
	}

	trainingConn, err := grpc.NewClient(trainingTarget, grpc.WithTransportCredentials(creds))
	if err != nil {
		_ = userConn.Close()
		return nil, nil, nil, fmt.Errorf("create training grpc connection error: %w", err)
	}

	userClient := userpb.NewUserClient(userConn)
	trainingClient := trainingpb.NewTrainingClient(trainingConn)
	closeConns := func() {
		_ = trainingConn.Close()
		_ = userConn.Close()
	}

	return userClient, trainingClient, closeConns, nil
}

func serviceURL(name string) (*url.URL, error) {
	raw, err := config.RequiredEnv(name)
	if err != nil {
		return nil, fmt.Errorf("read upstream configuration: %w", err)
	}
	target, err := url.Parse(raw)
	if err != nil || (target.Scheme != "http" && target.Scheme != "https") || target.Host == "" {
		return nil, fmt.Errorf("invalid %s: expected an HTTP URL", name)
	}
	return target, nil
}
