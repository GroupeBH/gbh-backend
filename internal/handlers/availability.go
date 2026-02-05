package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"gbh-backend/internal/schedule"
	"gbh-backend/internal/transport"
)

type availabilityQuery struct {
	Date string `validate:"required,date"`
}

func (s *Server) GetAvailability(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	q := availabilityQuery{Date: r.URL.Query().Get("date")}
	if err := s.Val.Struct(q); err != nil {
		log.Warn("availability: invalid query")
		details := validationDetails(s.Val.ValidationErrors(err))
		transport.WriteError(w, http.StatusBadRequest, "invalid query", details)
		return
	}

	duration, err := parseDurationParam(r.URL.Query().Get("duration"), schedule.SlotMinutes)
	if err != nil {
		log.Warn("availability: invalid duration")
		transport.WriteError(w, http.StatusBadRequest, "invalid duration", nil)
		return
	}

	cacheKey := "availability:" + q.Date + ":" + strconv.Itoa(duration)
	if s.Cache != nil {
		if cached, ok, err := s.Cache.Get(r.Context(), cacheKey); err == nil && ok {
			log.Info("availability: cache hit", slog.String("date", q.Date))
			writeCachedJSON(w, http.StatusOK, cached)
			return
		}
	}

	past, err := schedule.IsDatePast(q.Date, s.Cfg.Timezone, time.Now())
	if err != nil {
		log.Warn("availability: invalid date", slog.String("date", q.Date))
		transport.WriteError(w, http.StatusBadRequest, "invalid date", nil)
		return
	}
	if past {
		log.Warn("availability: date in the past", slog.String("date", q.Date))
		transport.WriteError(w, http.StatusBadRequest, "date in the past", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	slots, err := s.computeAvailableSlots(ctx, q.Date, duration, time.Now())
	if err != nil {
		log.Error("availability: compute error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "availability error", nil)
		return
	}

	response := map[string]interface{}{
		"date":     q.Date,
		"timezone": s.Cfg.Timezone.String(),
		"duration": duration,
		"slots":    slots,
	}

	if payload, err := encodeJSON(response); err == nil && s.Cache != nil {
		_ = s.Cache.Set(r.Context(), cacheKey, payload, time.Duration(s.Cfg.CacheTTLSeconds)*time.Second)
	}

	log.Info("availability: ok", slog.String("date", q.Date), slog.Int("duration", duration), slog.Int("slots", len(slots)))
	transport.WriteJSON(w, http.StatusOK, response)
}

func dateIsToday(dateStr string, loc *time.Location) bool {
	date, err := schedule.ParseDate(dateStr, loc)
	if err != nil {
		return false
	}
	now := time.Now().In(loc)
	return date.Year() == now.Year() && date.YearDay() == now.YearDay()
}
