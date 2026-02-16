package handlers

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gbh-backend/internal/auth"
	"gbh-backend/internal/models"
	"gbh-backend/internal/transport"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AdminUserCreateRequest struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"omitempty,email"`
	Password string `json:"password" validate:"required"`
}

type AdminRegisterRequest struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"omitempty,email"`
	Password string `json:"password" validate:"required"`
	SetupKey string `json:"setupKey" validate:"required"`
}

type AdminUserPasswordRequest struct {
	Password string `json:"password" validate:"required"`
}

func (s *Server) AdminRegister(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	var req AdminRegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		log.Warn("admin register: invalid json")
		transport.WriteError(w, http.StatusBadRequest, "invalid json", nil)
		return
	}
	req.Username, req.Email = normalizeAdminUserIdentity(req.Username, req.Email)
	if err := s.Val.Struct(req); err != nil {
		log.Warn("admin register: validation error")
		details := validationDetails(s.Val.ValidationErrors(err))
		transport.WriteError(w, http.StatusBadRequest, "validation error", details)
		return
	}
	if s.Cols == nil || s.Cols.Users == nil {
		log.Warn("admin register: not configured")
		transport.WriteError(w, http.StatusServiceUnavailable, "admin users not configured", nil)
		return
	}
	if s.Cfg.AdminSetupKey == "" {
		log.Warn("admin register: setup key missing")
		transport.WriteError(w, http.StatusServiceUnavailable, "admin registration not configured", nil)
		return
	}
	if s.Cfg.JWTSecret == "" {
		log.Warn("admin register: jwt secret missing")
		transport.WriteError(w, http.StatusServiceUnavailable, "admin auth not configured", nil)
		return
	}
	if subtle.ConstantTimeCompare([]byte(req.SetupKey), []byte(s.Cfg.AdminSetupKey)) != 1 {
		log.Warn("admin register: invalid setup key", slog.String("username", req.Username))
		transport.WriteError(w, http.StatusUnauthorized, "invalid setup key", nil)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Error("admin register: hash error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "password error", nil)
		return
	}

	now := time.Now().In(s.Cfg.Timezone)
	user := models.User{
		ID:           primitive.NewObjectID().Hex(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         models.UserRoleAdmin,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, err := s.Cols.Users.InsertOne(ctx, user); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			log.Warn("admin register: duplicate", slog.String("username", req.Username))
			transport.WriteError(w, http.StatusConflict, "username or email already exists", nil)
			return
		}
		log.Error("admin register: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("admin register: ok", slog.String("user_id", user.ID), slog.String("username", user.Username))
	accessToken, refreshToken, err := s.issueAdminSession(w)
	if err != nil {
		log.Error("admin register: token error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "token error", nil)
		return
	}
	transport.WriteJSON(w, http.StatusCreated, AdminLoginResponse{
		Status:       "ok",
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func (s *Server) AdminCreateUser(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	var req AdminUserCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		log.Warn("admin users create: invalid json")
		transport.WriteError(w, http.StatusBadRequest, "invalid json", nil)
		return
	}
	req.Username, req.Email = normalizeAdminUserIdentity(req.Username, req.Email)
	if err := s.Val.Struct(req); err != nil {
		log.Warn("admin users create: validation error")
		details := validationDetails(s.Val.ValidationErrors(err))
		transport.WriteError(w, http.StatusBadRequest, "validation error", details)
		return
	}
	if s.Cols == nil || s.Cols.Users == nil {
		log.Warn("admin users create: not configured")
		transport.WriteError(w, http.StatusServiceUnavailable, "admin users not configured", nil)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Error("admin users create: hash error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "password error", nil)
		return
	}

	now := time.Now().In(s.Cfg.Timezone)
	user := models.User{
		ID:           primitive.NewObjectID().Hex(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         models.UserRoleAdmin,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, err := s.Cols.Users.InsertOne(ctx, user); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			log.Warn("admin users create: duplicate", slog.String("username", req.Username))
			transport.WriteError(w, http.StatusConflict, "username or email already exists", nil)
			return
		}
		log.Error("admin users create: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("admin users create: ok", slog.String("user_id", user.ID), slog.String("username", user.Username))
	transport.WriteJSON(w, http.StatusCreated, user)
}

func (s *Server) AdminUpdateUserPassword(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	id := chi.URLParam(r, "id")
	if id == "" {
		log.Warn("admin users password: missing id")
		transport.WriteError(w, http.StatusBadRequest, "missing id", nil)
		return
	}

	var req AdminUserPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		log.Warn("admin users password: invalid json")
		transport.WriteError(w, http.StatusBadRequest, "invalid json", nil)
		return
	}
	if err := s.Val.Struct(req); err != nil {
		log.Warn("admin users password: validation error")
		details := validationDetails(s.Val.ValidationErrors(err))
		transport.WriteError(w, http.StatusBadRequest, "validation error", details)
		return
	}
	if s.Cols == nil || s.Cols.Users == nil {
		log.Warn("admin users password: not configured")
		transport.WriteError(w, http.StatusServiceUnavailable, "admin users not configured", nil)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Error("admin users password: hash error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "password error", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"passwordHash": hash,
			"updatedAt":    time.Now().In(s.Cfg.Timezone),
		},
	}
	res, err := s.Cols.Users.UpdateOne(ctx, bson.M{"_id": id, "role": models.UserRoleAdmin}, update)
	if err != nil {
		log.Error("admin users password: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}
	if res.MatchedCount == 0 {
		log.Warn("admin users password: not found", slog.String("user_id", id))
		transport.WriteError(w, http.StatusNotFound, "user not found", nil)
		return
	}

	log.Info("admin users password: ok", slog.String("user_id", id))
	transport.WriteJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func normalizeAdminUserIdentity(username, email string) (string, string) {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	if strings.Contains(username, "@") {
		username = strings.ToLower(username)
	}
	email = strings.ToLower(email)
	return username, email
}
