package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gbh-backend/internal/auth"
	"gbh-backend/internal/models"
	"gbh-backend/internal/transport"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

	req.Username = strings.TrimSpace(req.Username)
	if strings.Contains(req.Username, "@") {
		req.Username = strings.ToLower(req.Username)
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
	findOpts := options.FindOne().SetCollation(&options.Collation{Locale: "en", Strength: 2})
	if err := s.Cols.Users.FindOne(ctx, filter, findOpts).Decode(&user); err != nil {
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

	_, _, err := s.issueAdminSession(w)
	if err != nil {
		log.Error("admin login: token error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "token error", nil)
		return
	}
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

	refreshToken := extractRefreshToken(r)
	if refreshToken == "" {
		log.Warn("admin refresh: missing refresh token")
		transport.WriteError(w, http.StatusUnauthorized, "missing refresh token", nil)
		return
	}

	manager := s.newAdminJWTManager()

	claims, err := manager.Parse(refreshToken)
	if err != nil || claims.Role != models.UserRoleAdmin {
		log.Warn("admin refresh: invalid refresh token")
		transport.WriteError(w, http.StatusUnauthorized, "invalid refresh token", nil)
		return
	}

	_, _, err = s.issueAdminSession(w)
	if err != nil {
		log.Error("admin refresh: token error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "token error", nil)
		return
	}
	log.Info("admin refresh: ok")
	transport.WriteJSON(w, http.StatusOK, AdminLoginResponse{Status: "ok"})
}

func (s *Server) AdminLogout(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	clearAuthCookies(w, s.Cfg.CookieSecure)
	log.Info("admin logout: ok")
	transport.WriteJSON(w, http.StatusOK, AdminLoginResponse{Status: "ok"})
}

func (s *Server) newAdminJWTManager() auth.Manager {
	return auth.Manager{
		Secret:     []byte(s.Cfg.JWTSecret),
		AccessTTL:  time.Duration(s.Cfg.AccessTTLMinutes) * time.Minute,
		RefreshTTL: time.Duration(s.Cfg.RefreshTTLMinutes) * time.Minute,
		Issuer:     "gbh-backend",
	}
}

func (s *Server) issueAdminSession(w http.ResponseWriter) (string, string, error) {
	manager := s.newAdminJWTManager()

	accessToken, err := manager.NewAccessToken(models.UserRoleAdmin)
	if err != nil {
		return "", "", err
	}
	refreshToken, err := manager.NewRefreshToken(models.UserRoleAdmin)
	if err != nil {
		return "", "", err
	}

	setAuthCookies(w, accessToken, refreshToken, manager.AccessTTL, manager.RefreshTTL, s.Cfg.CookieSecure)
	return accessToken, refreshToken, nil
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

func extractRefreshToken(r *http.Request) string {
	if r == nil {
		return ""
	}
	if cookie, err := r.Cookie("gbh_refresh"); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return strings.TrimSpace(cookie.Value)
	}
	return extractBearerToken(r.Header.Get("Authorization"))
}

func extractBearerToken(authHeader string) string {
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
