package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterRegistersStatsXP(t *testing.T) {
	router := newRouter(nil, nil, nil, slog.New(slog.DiscardHandler))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/xp/123", nil)
	_, pattern := router.Handler(req)
	if pattern != "GET /api/v1/stats/xp/{telegramID}" {
		t.Fatalf("matched pattern = %q, want the Stats XP route", pattern)
	}
}
