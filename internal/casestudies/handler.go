package casestudies

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gbh-backend/internal/httpx"
	"gbh-backend/internal/middleware"
	"gbh-backend/internal/transport"
	"gbh-backend/internal/validation"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
	val     *validation.Validator
	log     *slog.Logger
}

func NewHandler(service *Service, val *validation.Validator, log *slog.Logger) *Handler {
	return &Handler{
		service: service,
		val:     val,
		log:     log,
	}
}

func (h *Handler) PublicList(w http.ResponseWriter, r *http.Request) {
	log := h.logWithRequest(r)
	filter := PublicListFilter{
		Category: strings.TrimSpace(r.URL.Query().Get("category")),
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items, err := h.service.ListPublic(ctx, filter)
	if err != nil {
		log.Error("case studies public list: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("case studies public list: ok", slog.Int("count", len(items)))
	transport.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

func (h *Handler) PublicGetBySlug(w http.ResponseWriter, r *http.Request) {
	log := h.logWithRequest(r)
	slug := strings.TrimSpace(chi.URLParam(r, "slug"))
	if slug == "" {
		log.Warn("case studies public get: missing slug")
		transport.WriteError(w, http.StatusBadRequest, "missing slug", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := h.service.GetPublishedBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Warn("case studies public get: not found", slog.String("slug", slug))
			transport.WriteError(w, http.StatusNotFound, "case study not found", nil)
			return
		}
		log.Error("case studies public get: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("case studies public get: ok", slog.String("slug", slug))
	transport.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) AdminList(w http.ResponseWriter, r *http.Request) {
	log := h.logWithRequest(r)
	limit, offset, err := httpx.ParseLimitOffset(r.URL.Query(), 20, 100)
	if err != nil {
		log.Warn("admin case studies list: invalid query", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	filter := AdminListFilter{
		Category: strings.TrimSpace(r.URL.Query().Get("category")),
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	items, total, err := h.service.ListAdmin(ctx, filter, limit, offset)
	if err != nil {
		log.Error("admin case studies list: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("admin case studies list: ok", slog.Int("count", len(items)))
	transport.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"items":  items,
		"limit":  limit,
		"offset": offset,
		"total":  total,
	})
}

func (h *Handler) AdminCreate(w http.ResponseWriter, r *http.Request) {
	log := h.logWithRequest(r)

	var req UpsertRequest
	if err := httpx.DecodeJSON(r.Body, &req); err != nil {
		log.Warn("admin case studies create: invalid json")
		transport.WriteError(w, http.StatusBadRequest, "invalid json", nil)
		return
	}

	if err := h.val.Struct(req); err != nil {
		log.Warn("admin case studies create: validation error")
		transport.WriteError(w, http.StatusBadRequest, "validation error", httpx.ValidationDetails(h.val.ValidationErrors(err)))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	item, err := h.service.Create(ctx, req)
	if err != nil {
		if errors.Is(err, ErrSlugExists) {
			log.Warn("admin case studies create: slug exists")
			transport.WriteError(w, http.StatusConflict, "slug already exists", nil)
			return
		}
		if errors.Is(err, ErrInvalidSlug) {
			transport.WriteError(w, http.StatusBadRequest, "validation error", map[string]string{"slug": "invalid"})
			return
		}
		log.Error("admin case studies create: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("admin case studies create: ok", slog.String("case_study_id", item.ID), slog.String("slug", item.Slug))
	transport.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	log := h.logWithRequest(r)
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		log.Warn("admin case studies update: missing id")
		transport.WriteError(w, http.StatusBadRequest, "missing id", nil)
		return
	}

	var req UpsertRequest
	if err := httpx.DecodeJSON(r.Body, &req); err != nil {
		log.Warn("admin case studies update: invalid json")
		transport.WriteError(w, http.StatusBadRequest, "invalid json", nil)
		return
	}

	if err := h.val.Struct(req); err != nil {
		log.Warn("admin case studies update: validation error")
		transport.WriteError(w, http.StatusBadRequest, "validation error", httpx.ValidationDetails(h.val.ValidationErrors(err)))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	item, err := h.service.Update(ctx, id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Warn("admin case studies update: not found", slog.String("case_study_id", id))
			transport.WriteError(w, http.StatusNotFound, "case study not found", nil)
			return
		}
		if errors.Is(err, ErrSlugExists) {
			log.Warn("admin case studies update: slug exists")
			transport.WriteError(w, http.StatusConflict, "slug already exists", nil)
			return
		}
		if errors.Is(err, ErrInvalidSlug) {
			transport.WriteError(w, http.StatusBadRequest, "validation error", map[string]string{"slug": "invalid"})
			return
		}
		log.Error("admin case studies update: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("admin case studies update: ok", slog.String("case_study_id", id), slog.String("slug", item.Slug))
	transport.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) AdminDelete(w http.ResponseWriter, r *http.Request) {
	log := h.logWithRequest(r)
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		log.Warn("admin case studies delete: missing id")
		transport.WriteError(w, http.StatusBadRequest, "missing id", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.service.Delete(ctx, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Warn("admin case studies delete: not found", slog.String("case_study_id", id))
			transport.WriteError(w, http.StatusNotFound, "case study not found", nil)
			return
		}
		log.Error("admin case studies delete: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("admin case studies delete: ok", slog.String("case_study_id", id))
	transport.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) logWithRequest(r *http.Request) *slog.Logger {
	if r == nil {
		return h.log
	}
	if id := middleware.RequestIDFromContext(r.Context()); id != "" {
		return h.log.With(slog.String("request_id", id))
	}
	return h.log
}
