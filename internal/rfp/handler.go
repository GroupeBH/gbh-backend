package rfp

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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	log := h.logWithRequest(r)

	var req CreateRequest
	if err := httpx.DecodeJSON(r.Body, &req); err != nil {
		log.Warn("rfp create: invalid json")
		transport.WriteError(w, http.StatusBadRequest, "invalid json", nil)
		return
	}

	if err := h.val.Struct(req); err != nil {
		log.Warn("rfp create: validation error")
		transport.WriteError(w, http.StatusBadRequest, "validation error", httpx.ValidationDetails(h.val.ValidationErrors(err)))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	lead, err := h.service.Create(ctx, req)
	if err != nil {
		if errors.Is(err, ErrInvalidSource) {
			transport.WriteError(w, http.StatusBadRequest, "validation error", map[string]string{"source": "oneof"})
			return
		}
		log.Error("rfp create: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	go func(created Lead) {
		notifyCtx, notifyCancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer notifyCancel()
		if err := h.service.NotifyNewLead(notifyCtx, created); err != nil {
			h.log.Warn("rfp create: notification failed",
				slog.String("rfp_id", created.ID),
				slog.String("error", err.Error()),
			)
		}

		if err := h.service.NotifyLeadConfirmation(notifyCtx, created); err != nil {
			h.log.Warn("rfp create: user confirmation email failed",
				slog.String("rfp_id", created.ID),
				slog.String("email", created.Email),
				slog.String("error", err.Error()),
			)
		}
	}(lead)

	log.Info("rfp create: ok", slog.String("rfp_id", lead.ID), slog.String("source", lead.Source))
	transport.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "rfp submitted",
		"id":      lead.ID,
	})
}

func (h *Handler) AdminList(w http.ResponseWriter, r *http.Request) {
	log := h.logWithRequest(r)
	limit, offset, err := httpx.ParseLimitOffset(r.URL.Query(), 20, 100)
	if err != nil {
		log.Warn("admin rfp list: invalid query", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	filter := ListFilter{
		Status: strings.TrimSpace(r.URL.Query().Get("status")),
		Source: strings.TrimSpace(r.URL.Query().Get("source")),
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	items, total, err := h.service.ListAdmin(ctx, filter, limit, offset)
	if err != nil {
		if errors.Is(err, ErrInvalidStatus) {
			transport.WriteError(w, http.StatusBadRequest, "invalid query", map[string]string{"status": "oneof"})
			return
		}
		if errors.Is(err, ErrInvalidSource) {
			transport.WriteError(w, http.StatusBadRequest, "invalid query", map[string]string{"source": "oneof"})
			return
		}
		log.Error("admin rfp list: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("admin rfp list: ok", slog.Int("count", len(items)))
	transport.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"items":  items,
		"limit":  limit,
		"offset": offset,
		"total":  total,
	})
}

func (h *Handler) AdminGetByID(w http.ResponseWriter, r *http.Request) {
	log := h.logWithRequest(r)
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		log.Warn("admin rfp get: missing id")
		transport.WriteError(w, http.StatusBadRequest, "missing id", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	lead, err := h.service.GetAdminByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Warn("admin rfp get: not found", slog.String("rfp_id", id))
			transport.WriteError(w, http.StatusNotFound, "rfp not found", nil)
			return
		}
		log.Error("admin rfp get: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("admin rfp get: ok", slog.String("rfp_id", id))
	transport.WriteJSON(w, http.StatusOK, lead)
}

func (h *Handler) AdminUpdateStatus(w http.ResponseWriter, r *http.Request) {
	log := h.logWithRequest(r)
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		log.Warn("admin rfp status: missing id")
		transport.WriteError(w, http.StatusBadRequest, "missing id", nil)
		return
	}

	var req AdminStatusUpdateRequest
	if err := httpx.DecodeJSON(r.Body, &req); err != nil {
		log.Warn("admin rfp status: invalid json")
		transport.WriteError(w, http.StatusBadRequest, "invalid json", nil)
		return
	}

	if err := h.val.Struct(req); err != nil {
		log.Warn("admin rfp status: validation error")
		transport.WriteError(w, http.StatusBadRequest, "validation error", httpx.ValidationDetails(h.val.ValidationErrors(err)))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	lead, err := h.service.UpdateStatus(ctx, id, req.Status)
	if err != nil {
		if errors.Is(err, ErrInvalidStatus) {
			transport.WriteError(w, http.StatusBadRequest, "validation error", map[string]string{"status": "oneof"})
			return
		}
		if errors.Is(err, ErrNotFound) {
			log.Warn("admin rfp status: not found", slog.String("rfp_id", id))
			transport.WriteError(w, http.StatusNotFound, "rfp not found", nil)
			return
		}
		log.Error("admin rfp status: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("admin rfp status: ok", slog.String("rfp_id", id), slog.String("status", lead.Status))
	transport.WriteJSON(w, http.StatusOK, lead)
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
