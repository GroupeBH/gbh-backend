package handlers

import (
	"context"
	"net/http"
	"time"

	"gbh-backend/internal/models"
	"gbh-backend/internal/schedule"
	"gbh-backend/internal/transport"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type CreateAppointmentRequest struct {
	ServiceID     string `json:"serviceId" validate:"required"`
	Name          string `json:"name" validate:"required"`
	Email         string `json:"email" validate:"required,email"`
	Phone         string `json:"phone" validate:"required,phone"`
	Type          string `json:"type" validate:"required,oneof=online presentiel"`
	Date          string `json:"date" validate:"required,date"`
	Time          string `json:"time" validate:"required,clock"`
	PaymentMethod string `json:"paymentMethod" validate:"required,oneof=online place"`
	Price         int    `json:"price" validate:"gte=0"`
}

func (s *Server) CreateAppointment(w http.ResponseWriter, r *http.Request) {
	var req CreateAppointmentRequest
	if err := decodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest, "invalid json", nil)
		return
	}

	if err := s.Val.Struct(req); err != nil {
		details := validationDetails(s.Val.ValidationErrors(err))
		transport.WriteError(w, http.StatusBadRequest, "validation error", details)
		return
	}

	past, err := schedule.IsDatePast(req.Date, s.Cfg.Timezone, time.Now())
	if err != nil {
		transport.WriteError(w, http.StatusBadRequest, "invalid date", nil)
		return
	}
	if past {
		transport.WriteError(w, http.StatusBadRequest, "date in the past", nil)
		return
	}

	allowed, err := schedule.IsSlotAllowed(req.Date, req.Time, s.Cfg.Timezone)
	if err != nil {
		transport.WriteError(w, http.StatusBadRequest, "invalid time", nil)
		return
	}
	if !allowed {
		transport.WriteError(w, http.StatusBadRequest, "slot not available", nil)
		return
	}

	if dateIsToday(req.Date, s.Cfg.Timezone) {
		pastSlot, err := schedule.IsSlotPast(req.Date, req.Time, s.Cfg.Timezone, time.Now())
		if err != nil {
			transport.WriteError(w, http.StatusBadRequest, "invalid time", nil)
			return
		}
		if pastSlot {
			transport.WriteError(w, http.StatusBadRequest, "slot already passed", nil)
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	if err := s.Cols.Services.FindOne(ctx, bson.M{"_id": req.ServiceID}).Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			transport.WriteError(w, http.StatusBadRequest, "service not found", nil)
			return
		}
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	blockCount, err := s.Cols.ReservationBlocks.CountDocuments(ctx, bson.M{"date": req.Date, "time": req.Time})
	if err != nil {
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}
	if blockCount > 0 {
		transport.WriteError(w, http.StatusConflict, "slot blocked", nil)
		return
	}

	existingCount, err := s.Cols.Appointments.CountDocuments(ctx, bson.M{"date": req.Date, "time": req.Time})
	if err != nil {
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}
	if existingCount > 0 {
		transport.WriteError(w, http.StatusConflict, "slot already booked", nil)
		return
	}

	appointment := models.Appointment{
		ID:            primitive.NewObjectID().Hex(),
		ServiceID:     req.ServiceID,
		Name:          req.Name,
		Email:         req.Email,
		Phone:         req.Phone,
		Type:          req.Type,
		Date:          req.Date,
		Time:          req.Time,
		Duration:      schedule.SlotMinutes,
		Price:         req.Price,
		Tax:           0,
		Total:         req.Price,
		Status:        models.AppointmentStatusBooked,
		PaymentMethod: req.PaymentMethod,
		CreatedAt:     time.Now().In(s.Cfg.Timezone),
	}

	_, err = s.Cols.Appointments.InsertOne(ctx, appointment)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			transport.WriteError(w, http.StatusConflict, "slot already booked", nil)
			return
		}
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	if s.Cache != nil {
		_ = s.Cache.Delete(r.Context(), "availability:"+req.Date)
	}

	transport.WriteJSON(w, http.StatusCreated, appointment)
}

func (s *Server) GetAppointment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		transport.WriteError(w, http.StatusBadRequest, "missing id", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var doc bson.M
	if err := s.Cols.Appointments.FindOne(ctx, bson.M{"_id": id}).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			transport.WriteError(w, http.StatusNotFound, "appointment not found", nil)
			return
		}
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	transport.WriteJSON(w, http.StatusOK, normalizeID(doc))
}

