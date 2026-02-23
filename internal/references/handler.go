package references

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
		log.Error("references public list: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("references public list: ok", slog.Int("count", len(items)))
	transport.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

func (h *Handler) AdminList(w http.ResponseWriter, r *http.Request) {
	log := h.logWithRequest(r)
	limit, offset, err := httpx.ParseLimitOffset(r.URL.Query(), 20, 100)
	if err != nil {
		log.Warn("admin references list: invalid query", slog.String("error", err.Error()))
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
		log.Error("admin references list: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("admin references list: ok", slog.Int("count", len(items)))
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
		log.Warn("admin references create: invalid json")
		transport.WriteError(w, http.StatusBadRequest, "invalid json", nil)
		return
	}

	if err := h.val.Struct(req); err != nil {
		log.Warn("admin references create: validation error")
		transport.WriteError(w, http.StatusBadRequest, "validation error", httpx.ValidationDetails(h.val.ValidationErrors(err)))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	item, err := h.service.Create(ctx, req)
	if err != nil {
		log.Error("admin references create: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("admin references create: ok", slog.String("reference_id", item.ID))
	transport.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	log := h.logWithRequest(r)
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		log.Warn("admin references update: missing id")
		transport.WriteError(w, http.StatusBadRequest, "missing id", nil)
		return
	}

	var req UpsertRequest
	if err := httpx.DecodeJSON(r.Body, &req); err != nil {
		log.Warn("admin references update: invalid json")
		transport.WriteError(w, http.StatusBadRequest, "invalid json", nil)
		return
	}

	if err := h.val.Struct(req); err != nil {
		log.Warn("admin references update: validation error")
		transport.WriteError(w, http.StatusBadRequest, "validation error", httpx.ValidationDetails(h.val.ValidationErrors(err)))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	item, err := h.service.Update(ctx, id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Warn("admin references update: not found", slog.String("reference_id", id))
			transport.WriteError(w, http.StatusNotFound, "reference not found", nil)
			return
		}
		log.Error("admin references update: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("admin references update: ok", slog.String("reference_id", id))
	transport.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) AdminDelete(w http.ResponseWriter, r *http.Request) {
	log := h.logWithRequest(r)
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		log.Warn("admin references delete: missing id")
		transport.WriteError(w, http.StatusBadRequest, "missing id", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.service.Delete(ctx, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Warn("admin references delete: not found", slog.String("reference_id", id))
			transport.WriteError(w, http.StatusNotFound, "reference not found", nil)
			return
		}
		log.Error("admin references delete: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("admin references delete: ok", slog.String("reference_id", id))
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
