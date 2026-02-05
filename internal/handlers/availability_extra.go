package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"gbh-backend/internal/schedule"
	"gbh-backend/internal/transport"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (s *Server) GetServiceAvailability(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	serviceID := chi.URLParam(r, "id")
	if serviceID == "" {
		log.Warn("service availability: missing service id")
		transport.WriteError(w, http.StatusBadRequest, "missing service id", nil)
		return
	}

	q := availabilityQuery{Date: r.URL.Query().Get("date")}
	if err := s.Val.Struct(q); err != nil {
		log.Warn("service availability: invalid query")
		details := validationDetails(s.Val.ValidationErrors(err))
		transport.WriteError(w, http.StatusBadRequest, "invalid query", details)
		return
	}

	duration, err := parseDurationParam(r.URL.Query().Get("duration"), schedule.SlotMinutes)
	if err != nil {
		log.Warn("service availability: invalid duration")
		transport.WriteError(w, http.StatusBadRequest, "invalid duration", nil)
		return
	}

	past, err := schedule.IsDatePast(q.Date, s.Cfg.Timezone, time.Now())
	if err != nil {
		transport.WriteError(w, http.StatusBadRequest, "invalid date", nil)
		return
	}
	if past {
		transport.WriteError(w, http.StatusBadRequest, "date in the past", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.Cols.Services.FindOne(ctx, bson.M{"_id": serviceID}).Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn("service availability: service not found", slog.String("service_id", serviceID))
			transport.WriteError(w, http.StatusNotFound, "service not found", nil)
			return
		}
		log.Error("service availability: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	slots, err := s.computeAvailableSlots(ctx, q.Date, duration, time.Now())
	if err != nil {
		log.Error("service availability: compute error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "availability error", nil)
		return
	}

	response := map[string]interface{}{
		"serviceId": serviceID,
		"date":      q.Date,
		"timezone":  s.Cfg.Timezone.String(),
		"duration":  duration,
		"slots":     slots,
	}

	log.Info("service availability: ok", slog.String("service_id", serviceID), slog.String("date", q.Date), slog.Int("slots", len(slots)))
	transport.WriteJSON(w, http.StatusOK, response)
}

func (s *Server) GetNextAvailability(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	from := r.URL.Query().Get("from")
	if from == "" {
		from = time.Now().In(s.Cfg.Timezone).Format("2006-01-02")
	}
	if err := s.Val.Struct(availabilityQuery{Date: from}); err != nil {
		log.Warn("availability next: invalid date")
		transport.WriteError(w, http.StatusBadRequest, "invalid date", nil)
		return
	}

	duration, err := parseDurationParam(r.URL.Query().Get("duration"), schedule.SlotMinutes)
	if err != nil {
		log.Warn("availability next: invalid duration")
		transport.WriteError(w, http.StatusBadRequest, "invalid duration", nil)
		return
	}

	past, err := schedule.IsDatePast(from, s.Cfg.Timezone, time.Now())
	if err != nil {
		transport.WriteError(w, http.StatusBadRequest, "invalid date", nil)
		return
	}
	if past {
		transport.WriteError(w, http.StatusBadRequest, "date in the past", nil)
		return
	}

	startDate, err := schedule.ParseDate(from, s.Cfg.Timezone)
	if err != nil {
		transport.WriteError(w, http.StatusBadRequest, "invalid date", nil)
		return
	}
	for i := 0; i < 30; i++ {
		current := startDate.AddDate(0, 0, i)
		dateStr := current.Format("2006-01-02")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		slots, err := s.computeAvailableSlots(ctx, dateStr, duration, time.Now())
		cancel()
		if err != nil {
			log.Error("availability next: compute error", slog.String("error", err.Error()))
			transport.WriteError(w, http.StatusInternalServerError, "availability error", nil)
			return
		}
		if len(slots) > 0 {
			response := map[string]interface{}{
				"date":     dateStr,
				"time":     slots[0],
				"timezone": s.Cfg.Timezone.String(),
				"duration": duration,
			}
			log.Info("availability next: ok", slog.String("date", dateStr), slog.String("time", slots[0]))
			transport.WriteJSON(w, http.StatusOK, response)
			return
		}
	}

	transport.WriteError(w, http.StatusNotFound, "no availability found", map[string]string{"days": strconv.Itoa(30)})
}
