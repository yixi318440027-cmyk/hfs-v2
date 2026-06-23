package server

import (
	"context"
	"net/http"
	"strings"
)

// authMiddleware validates the JWT Bearer token from the Authorization header.
// Valid claims are stored in the request context under the "user" key.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" || !strings.HasPrefix(token, "Bearer ") {
			http.Error(w, `{"ok":false,"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		claims, err := s.auth.ValidateToken(strings.TrimPrefix(token, "Bearer "))
		if err != nil {
			http.Error(w, `{"ok":false,"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "user", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
