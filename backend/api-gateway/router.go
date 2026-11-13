package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func newRouter(userService, trainingService http.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.Handle("POST /api/v1/users", userService)
	mux.Handle("GET /api/v1/users/telegram/{telegramID}", userService)

	mux.Handle("GET /api/v1/question", trainingService)
	mux.Handle("POST /api/v1/question/answer", trainingService)

	return mux
}

func newServiceProxy(name string, target *url.URL, logger *slog.Logger) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.ErrorContext(r.Context(), "upstream request failed", "service", name, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "{\"error\":\"service unavailable\"}\n")
	}
	return proxy
}
