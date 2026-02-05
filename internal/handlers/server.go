package handlers

import (
	"log/slog"
	"net/http"

	"gbh-backend/internal/cache"
	"gbh-backend/internal/config"
	"gbh-backend/internal/db"
	"gbh-backend/internal/middleware"
	"gbh-backend/internal/validation"
)

type Server struct {
	Cfg  *config.Config
	Cols *db.Collections
	Val  *validation.Validator
	Log  *slog.Logger
	Cache cache.Cache
}

func (s *Server) logWithRequest(r *http.Request) *slog.Logger {
	if r == nil {
		return s.Log
	}
	if id := middleware.RequestIDFromContext(r.Context()); id != "" {
		return s.Log.With(slog.String("request_id", id))
	}
	return s.Log
}
