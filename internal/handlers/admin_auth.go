package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"gbh-backend/internal/auth"
	"gbh-backend/internal/models"
	"gbh-backend/internal/transport"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AdminLoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type AdminLoginResponse struct {
	Status string `json:"status"`
}

func (s *Server) AdminLogin(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	var req AdminLoginRequest
	if err := decodeJSON(r, &req); err != nil {
		log.Warn("admin login: invalid json")
		transport.WriteError(w, http.StatusBadRequest, "invalid json", nil)
		return
	}
	if err := s.Val.Struct(req); err != nil {
		log.Warn("admin login: validation error")
		details := validationDetails(s.Val.ValidationErrors(err))
		transport.WriteError(w, http.StatusBadRequest, "validation error", details)
		return
	}

	if s.Cfg.JWTSecret == "" || s.Cols == nil || s.Cols.Users == nil {
		log.Warn("admin login: not configured")
		transport.WriteError(w, http.StatusServiceUnavailable, "admin auth not configured", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var user models.User
	filter := bson.M{
		"role": models.UserRoleAdmin,
		"$or": []bson.M{
			{"username": req.Username},
			{"email": req.Username},
		},
	}
	if err := s.Cols.Users.FindOne(ctx, filter).Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn("admin login: invalid credentials", slog.String("username", req.Username))
			transport.WriteError(w, http.StatusUnauthorized, "invalid credentials", nil)
			return
		}
		log.Error("admin login: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	if err := auth.ComparePassword(user.PasswordHash, req.Password); err != nil {
		log.Warn("admin login: invalid credentials", slog.String("username", req.Username))
		transport.WriteError(w, http.StatusUnauthorized, "invalid credentials", nil)
		return
	}

	manager := auth.Manager{
		Secret:     []byte(s.Cfg.JWTSecret),
		AccessTTL:  time.Duration(s.Cfg.AccessTTLMinutes) * time.Minute,
		RefreshTTL: time.Duration(s.Cfg.RefreshTTLMinutes) * time.Minute,
		Issuer:     "gbh-backend",
	}

	accessToken, err := manager.NewAccessToken("admin")
	if err != nil {
		transport.WriteError(w, http.StatusInternalServerError, "token error", nil)
		return
	}
	refreshToken, err := manager.NewRefreshToken("admin")
	if err != nil {
		transport.WriteError(w, http.StatusInternalServerError, "token error", nil)
		return
	}

	setAuthCookies(w, accessToken, refreshToken, manager.AccessTTL, manager.RefreshTTL, s.Cfg.CookieSecure)
	log.Info("admin login: ok", slog.String("username", req.Username))
	transport.WriteJSON(w, http.StatusOK, AdminLoginResponse{Status: "ok"})
}

func (s *Server) AdminRefresh(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	if s.Cfg.JWTSecret == "" {
		log.Warn("admin refresh: not configured")
		transport.WriteError(w, http.StatusServiceUnavailable, "admin auth not configured", nil)
		return
	}

	refreshCookie, err := r.Cookie("gbh_refresh")
	if err != nil || refreshCookie.Value == "" {
		log.Warn("admin refresh: missing refresh token")
		transport.WriteError(w, http.StatusUnauthorized, "missing refresh token", nil)
		return
	}

	manager := auth.Manager{
		Secret:     []byte(s.Cfg.JWTSecret),
		AccessTTL:  time.Duration(s.Cfg.AccessTTLMinutes) * time.Minute,
		RefreshTTL: time.Duration(s.Cfg.RefreshTTLMinutes) * time.Minute,
		Issuer:     "gbh-backend",
	}

	claims, err := manager.Parse(refreshCookie.Value)
	if err != nil || claims.Role != "admin" {
		log.Warn("admin refresh: invalid refresh token")
		transport.WriteError(w, http.StatusUnauthorized, "invalid refresh token", nil)
		return
	}

	accessToken, err := manager.NewAccessToken("admin")
	if err != nil {
		transport.WriteError(w, http.StatusInternalServerError, "token error", nil)
		return
	}
	refreshToken, err := manager.NewRefreshToken("admin")
	if err != nil {
		transport.WriteError(w, http.StatusInternalServerError, "token error", nil)
		return
	}

	setAuthCookies(w, accessToken, refreshToken, manager.AccessTTL, manager.RefreshTTL, s.Cfg.CookieSecure)
	log.Info("admin refresh: ok")
	transport.WriteJSON(w, http.StatusOK, AdminLoginResponse{Status: "ok"})
}

func (s *Server) AdminLogout(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	clearAuthCookies(w, s.Cfg.CookieSecure)
	log.Info("admin logout: ok")
	transport.WriteJSON(w, http.StatusOK, AdminLoginResponse{Status: "ok"})
}

func setAuthCookies(w http.ResponseWriter, access, refresh string, accessTTL, refreshTTL time.Duration, secure bool) {
	accessCookie := &http.Cookie{
		Name:     "gbh_access",
		Value:    access,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(accessTTL.Seconds()),
	}
	refreshCookie := &http.Cookie{
		Name:     "gbh_refresh",
		Value:    refresh,
		Path:     "/api/admin",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(refreshTTL.Seconds()),
	}
	http.SetCookie(w, accessCookie)
	http.SetCookie(w, refreshCookie)
}

func clearAuthCookies(w http.ResponseWriter, secure bool) {
	expire := time.Now().Add(-1 * time.Hour)
	accessCookie := &http.Cookie{
		Name:     "gbh_access",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expire,
		MaxAge:   -1,
	}
	refreshCookie := &http.Cookie{
		Name:     "gbh_refresh",
		Value:    "",
		Path:     "/api/admin",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expire,
		MaxAge:   -1,
	}
	http.SetCookie(w, accessCookie)
	http.SetCookie(w, refreshCookie)
}
