package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/calmarie/scoreup_train/internal/config"
)

func run(logger *slog.Logger) error {
	httpPort, err := config.RequiredEnv("HTTP_PORT")
	if err != nil {
		return fmt.Errorf("read HTTP configuration: %w", err)
	}
	userTarget, err := serviceURL("USER_SERVICE_URL")
	if err != nil {
		return err
	}
	trainingTarget, err := serviceURL("TRAINING_SERVICE_URL")
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr: ":" + httpPort,
		Handler: newRouter(
			newServiceProxy("user-service", userTarget, logger),
			newServiceProxy("training-service", trainingTarget, logger),
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
