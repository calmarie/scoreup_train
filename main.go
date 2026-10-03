package main

import (
	"log/slog"
	"os"
)

func main() {

	//standart logger for machine-readable text with standart output stream
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("service stopped", "error", err)
		os.Exit(1)
	}
}
