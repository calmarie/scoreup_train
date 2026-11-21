package middleware

import (
	"context"
	"net/http"
	"strings"
)

type Middleware struct {
	verifier *Verifier
}

func New(verifier *Verifier) *Middleware {
	return &Middleware{verifier: verifier}
}

func (m *Middleware) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "expected bearer token", http.StatusUnauthorized)
			return
		}

		userID, err := m.verifier.VerifyToken(parts[1])

		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(
			r.Context(),
			"userID",
			userID,
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
