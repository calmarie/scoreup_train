package postgres

import (
	"net"
	"net/url"

	"github.com/calmarie/scoreup_train/internal/config"
)

func DatabaseURL() (string, error) {
	names := []string{
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_DB",
		"POSTGRES_HOST",
		"POSTGRES_PORT",
	}
	values := make(map[string]string, len(names))
	for _, name := range names {
		value, err := config.RequiredEnv(name)
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
