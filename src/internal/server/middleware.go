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
		ctx = context.WithValue(ctx, "username", claims.Username)
		ctx = context.WithValue(ctx, "role", claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// adminMiddleware checks that the user has the "admin" role.
// Must be used after authMiddleware.
func (s *Server) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value("role").(string)
		if role != "admin" {
			http.Error(w, `{"ok":"false","error":"admin access required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
