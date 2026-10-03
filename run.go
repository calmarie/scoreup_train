package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/calmarie/scoreup_train/internal/postgres"
)

// listen and serve func
func run(logger *slog.Logger) error {
	databaseURL, err := dbURL()
	if err != nil {
		return fmt.Errorf("read postgres configuration: %w", err)
	}

	httpPort, err := requiredEnv("HTTP_PORT")
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

// converts environment veriables to connection link to postgres database
func dbURL() (string, error) {
	names := []string{
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_DB",
		"POSTGRES_HOST",
		"POSTGRES_PORT",
	}
	values := make(map[string]string, len(names))

	for _, name := range names {
		value, err := requiredEnv(name)
		if err != nil {
			return "", err
		}
		values[name] = value
	}

	databaseURL := &url.URL{
		Scheme: "postgresql",
		User: url.UserPassword(
			values["POSTGRES_USER"],
			values["POSTGRES_PASSWORD"],
		),
		Host: net.JoinHostPort(
			values["POSTGRES_HOST"],
			values["POSTGRES_PORT"],
		),
		Path: values["POSTGRES_DB"],
	}
	query := databaseURL.Query()
	query.Set("sslmode", "disable")
	databaseURL.RawQuery = query.Encode()

	return databaseURL.String(), nil
}

// finds environment variable from .env file
func requiredEnv(name string) (string, error) {
	value, ok := os.LookupEnv(name)
	if !ok || value == "" {
		return "", fmt.Errorf("environment variable %s is required", name)
	}

	return value, nil
}
