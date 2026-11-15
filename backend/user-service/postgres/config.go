package postgres

import (
	"fmt"
	"net"
	"net/url"
	"os"
)

func DatabaseURL() (string, error) {
	names := []string{
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"USER_POSTGRES_DB",
		"USER_POSTGRES_HOST",
		"USER_POSTGRES_PORT",
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
			values["USER_POSTGRES_HOST"],
			values["USER_POSTGRES_PORT"],
		),
		Path: values["USER_POSTGRES_DB"],
	}
	query := databaseURL.Query()
	query.Set("sslmode", "disable")
	databaseURL.RawQuery = query.Encode()

	return databaseURL.String(), nil
}

func requiredEnv(name string) (string, error) {
	value, ok := os.LookupEnv(name)
	if !ok || value == "" {
		return "", fmt.Errorf("environment variable %s is required", name)
	}
	return value, nil
}
