package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRouterForwardsPublicRoutesToOwningService(t *testing.T) {
	userHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Service", "user")
		w.WriteHeader(http.StatusAccepted)
	})
	trainingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Service", "training")
		w.WriteHeader(http.StatusAccepted)
	})
	router := newRouter(userHandler, trainingHandler)

	tests := []struct {
		method  string
		path    string
		service string
	}{
		{http.MethodPost, "/api/v1/users", "user"},
		{http.MethodGet, "/api/v1/users/telegram/123", "user"},
		{http.MethodGet, "/api/v1/question", "training"},
		{http.MethodPost, "/api/v1/question/answer", "training"},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusAccepted {
				t.Errorf("status = %d, want %d", response.Code, http.StatusAccepted)
			}
			if got := response.Header().Get("X-Service"); got != test.service {
				t.Errorf("service = %q, want %q", got, test.service)
			}
		})
	}
}

func TestRouterHealthIsLocal(t *testing.T) {
	upstream := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("health request reached a service")
	})
	router := newRouter(upstream, upstream)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestGatewayProxiesRequestAndResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read upstream request: %v", err)
		}
		if r.Method != http.MethodPost || r.URL.RequestURI() != "/api/v1/question/answer?mode=test" || string(body) != `{"question_id":5}` {
			t.Errorf("upstream request = %s %s %s", r.Method, r.URL.RequestURI(), body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":9}`)
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouter(http.NotFoundHandler(), newServiceProxy("training-service", target, logger))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/question/answer?mode=test", strings.NewReader(`{"question_id":5}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("content type = %q, want application/json", got)
	}
	if got := response.Body.String(); got != `{"id":9}` {
		t.Errorf("body = %q, want %q", got, `{"id":9}`)
	}
}

func TestGatewayReturnsBadGatewayWhenServiceIsUnavailable(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	upstream.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouter(newServiceProxy("user-service", target, logger), http.NotFoundHandler())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/users", nil))

	if response.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", response.Code, http.StatusBadGateway)
	}
	if got := response.Body.String(); got != "{\"error\":\"service unavailable\"}\n" {
		t.Errorf("body = %q, want service unavailable JSON", got)
	}
}
