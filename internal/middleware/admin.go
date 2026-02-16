package middleware

import (
	"net/http"
	"strings"

	"gbh-backend/internal/auth"
	"gbh-backend/internal/models"
	"gbh-backend/internal/transport"
)

func AdminAuth(adminKey string, manager *auth.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if adminKey == "" && manager == nil {
				transport.WriteError(w, http.StatusServiceUnavailable, "admin auth not configured", nil)
				return
			}

			if adminKey != "" && r.Header.Get("X-Admin-Key") == adminKey {
				next.ServeHTTP(w, r)
				return
			}

			if manager != nil {
				cookie, err := r.Cookie("gbh_access")
				if err == nil && cookie.Value != "" {
					claims, err := manager.Parse(cookie.Value)
					if err == nil && claims.Role == models.UserRoleAdmin {
						next.ServeHTTP(w, r)
						return
					}
				}

				if token := bearerToken(r.Header.Get("Authorization")); token != "" {
					claims, err := manager.Parse(token)
					if err == nil && claims.Role == models.UserRoleAdmin {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			transport.WriteError(w, http.StatusUnauthorized, "unauthorized", nil)
		})
	}
}

func bearerToken(authHeader string) string {
	authHeader = strings.TrimSpace(authHeader)
	if authHeader == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
}
