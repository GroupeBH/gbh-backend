package handlers

import (
	"context"
	"net/http"
	"time"

	"gbh-backend/internal/schedule"
	"gbh-backend/internal/transport"
	"go.mongodb.org/mongo-driver/bson"
)

type availabilityQuery struct {
	Date string `validate:"required,date"`
}

func (s *Server) GetAvailability(w http.ResponseWriter, r *http.Request) {
	q := availabilityQuery{Date: r.URL.Query().Get("date")}
	if err := s.Val.Struct(q); err != nil {
		details := validationDetails(s.Val.ValidationErrors(err))
		transport.WriteError(w, http.StatusBadRequest, "invalid query", details)
		return
	}

	cacheKey := "availability:" + q.Date
	if s.Cache != nil {
		if cached, ok, err := s.Cache.Get(r.Context(), cacheKey); err == nil && ok {
			writeCachedJSON(w, http.StatusOK, cached)
			return
		}
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

	slots, err := schedule.GenerateSlots(q.Date, s.Cfg.Timezone)
	if err != nil {
		transport.WriteError(w, http.StatusBadRequest, "invalid date", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	reserved := make(map[string]bool)

	appCursor, err := s.Cols.Appointments.Find(ctx, bson.M{"date": q.Date})
	if err != nil {
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}
	for appCursor.Next(ctx) {
		var doc bson.M
		if err := appCursor.Decode(&doc); err == nil {
			if t, ok := doc["time"].(string); ok {
				reserved[t] = true
			}
		}
	}
	if err := appCursor.Err(); err != nil {
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}
	appCursor.Close(ctx)

	blockCursor, err := s.Cols.ReservationBlocks.Find(ctx, bson.M{"date": q.Date})
	if err != nil {
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}
	for blockCursor.Next(ctx) {
		var doc bson.M
		if err := blockCursor.Decode(&doc); err == nil {
			if t, ok := doc["time"].(string); ok {
				reserved[t] = true
			}
		}
	}
	if err := blockCursor.Err(); err != nil {
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}
	blockCursor.Close(ctx)

	slots = schedule.FilterReserved(slots, reserved)

	if dateIsToday(q.Date, s.Cfg.Timezone) {
		slots, err = schedule.FilterPastSlots(q.Date, slots, s.Cfg.Timezone, time.Now())
		if err != nil {
			transport.WriteError(w, http.StatusInternalServerError, "slot filtering error", nil)
			return
		}
	}

	response := map[string]interface{}{
		"date":     q.Date,
		"timezone": s.Cfg.Timezone.String(),
		"slots":    slots,
	}

	if payload, err := encodeJSON(response); err == nil && s.Cache != nil {
		_ = s.Cache.Set(r.Context(), cacheKey, payload, time.Duration(s.Cfg.CacheTTLSeconds)*time.Second)
	}

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
