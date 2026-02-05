package middleware

import (
	"net/http"

	"gbh-backend/internal/auth"
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
					if err == nil && claims.Role == "admin" {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			transport.WriteError(w, http.StatusUnauthorized, "unauthorized", nil)
		})
	}
}
