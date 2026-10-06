package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
)

// converts environment veriables to connection link to postgres database
func DbURL(dbEnv string) (string, error) {
	names := []string{
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		dbEnv,
		"POSTGRES_HOST",
		"POSTGRES_PORT",
	}
	values := make(map[string]string, len(names))

	for _, name := range names {
		value, err := RequiredEnv(name)
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
		Path: values[dbEnv],
	}
	query := databaseURL.Query()
	query.Set("sslmode", "disable")
	databaseURL.RawQuery = query.Encode()

	return databaseURL.String(), nil
}

// finds environment variable from .env file
func RequiredEnv(name string) (string, error) {
	value, ok := os.LookupEnv(name)
	if !ok || value == "" {
		return "", fmt.Errorf("environment variable %s is required", name)
	}

	return value, nil
}
