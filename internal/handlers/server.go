package handlers

import (
	"log/slog"

	"gbh-backend/internal/cache"
	"gbh-backend/internal/config"
	"gbh-backend/internal/db"
	"gbh-backend/internal/validation"
)

type Server struct {
	Cfg  *config.Config
	Cols *db.Collections
	Val  *validation.Validator
	Log  *slog.Logger
	Cache cache.Cache
}
