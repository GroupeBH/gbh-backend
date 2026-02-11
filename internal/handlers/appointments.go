package handlers

import (
	"context"
	"log/slog"
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
	Duration      int    `json:"duration" validate:"omitempty,gte=15,lte=240,minutes15"`
	PaymentMethod string `json:"paymentMethod" validate:"required,oneof=online place"`
	Price         int    `json:"price" validate:"gte=0"`
}

func (s *Server) CreateAppointment(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	var req CreateAppointmentRequest
	if err := decodeJSON(r, &req); err != nil {
		log.Warn("appointments create: invalid json")
		transport.WriteError(w, http.StatusBadRequest, "invalid json", nil)
		return
	}

	if err := s.Val.Struct(req); err != nil {
		log.Warn("appointments create: validation error")
		details := validationDetails(s.Val.ValidationErrors(err))
		transport.WriteError(w, http.StatusBadRequest, "validation error", details)
		return
	}

	duration := req.Duration
	if duration == 0 {
		duration = schedule.SlotMinutes
	}

	past, err := schedule.IsDatePast(req.Date, s.Cfg.Timezone, time.Now())
	if err != nil {
		log.Warn("appointments create: invalid date", slog.String("date", req.Date))
		transport.WriteError(w, http.StatusBadRequest, "invalid date", nil)
		return
	}
	if past {
		log.Warn("appointments create: date in the past", slog.String("date", req.Date))
		transport.WriteError(w, http.StatusBadRequest, "date in the past", nil)
		return
	}

	allowed, err := schedule.IsSlotAllowedWithDuration(req.Date, req.Time, duration, s.Cfg.Timezone)
	if err != nil {
		log.Warn("appointments create: invalid time", slog.String("time", req.Time))
		transport.WriteError(w, http.StatusBadRequest, "invalid time", nil)
		return
	}
	if !allowed {
		log.Warn("appointments create: slot not allowed", slog.String("date", req.Date), slog.String("time", req.Time))
		transport.WriteError(w, http.StatusBadRequest, "slot not available", nil)
		return
	}

	if dateIsToday(req.Date, s.Cfg.Timezone) {
		pastSlot, err := schedule.IsSlotPast(req.Date, req.Time, s.Cfg.Timezone, time.Now())
		if err != nil {
			log.Warn("appointments create: invalid time", slog.String("time", req.Time))
			transport.WriteError(w, http.StatusBadRequest, "invalid time", nil)
			return
		}
		if pastSlot {
			log.Warn("appointments create: slot already passed", slog.String("date", req.Date), slog.String("time", req.Time))
			transport.WriteError(w, http.StatusBadRequest, "slot already passed", nil)
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	var service models.Service
	if err := s.Cols.Services.FindOne(ctx, bson.M{"_id": req.ServiceID}).Decode(&service); err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn("appointments create: service not found", slog.String("service_id", req.ServiceID))
			transport.WriteError(w, http.StatusBadRequest, "service not found", nil)
			return
		}
		log.Error("appointments create: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	reserved, err := s.reservedIntervals(ctx, req.Date)
	if err != nil {
		log.Error("appointments create: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	startMin, err := schedule.ParseClockToMinutes(req.Time)
	if err != nil {
		log.Warn("appointments create: invalid time", slog.String("time", req.Time))
		transport.WriteError(w, http.StatusBadRequest, "invalid time", nil)
		return
	}
	current := schedule.Interval{Start: startMin, End: startMin + duration}
	for _, interval := range reserved {
		if schedule.Overlaps(current, interval) {
			log.Warn("appointments create: slot overlap", slog.String("date", req.Date), slog.String("time", req.Time))
			transport.WriteError(w, http.StatusConflict, "slot not available", nil)
			return
		}
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
		Duration:      duration,
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
			log.Warn("appointments create: duplicate key", slog.String("date", req.Date), slog.String("time", req.Time))
			transport.WriteError(w, http.StatusConflict, "slot already booked", nil)
			return
		}
		log.Error("appointments create: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	if s.Cache != nil {
		_ = s.Cache.DeletePrefix(r.Context(), "availability:"+req.Date+":")
	}

	if s.Mailer != nil {
		appointmentCopy := appointment
		serviceCopy := service
		go s.sendAppointmentConfirmationEmail(log, appointmentCopy, serviceCopy)
	}

	log.Info("appointments create: booked",
		slog.String("appointment_id", appointment.ID),
		slog.String("service_id", appointment.ServiceID),
		slog.String("date", appointment.Date),
		slog.String("time", appointment.Time),
	)
	availableSlots, err := s.computeAvailableSlots(ctx, req.Date, duration, time.Now())
	if err != nil {
		log.Warn("appointments create: availability compute error", slog.String("error", err.Error()))
	}
	transport.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"appointment":    appointment,
		"availableSlots": availableSlots,
	})
}

func (s *Server) sendAppointmentConfirmationEmail(log *slog.Logger, appointment models.Appointment, service models.Service) {
	if s.Mailer == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	messageID, err := s.Mailer.SendAppointmentConfirmation(ctx, appointment, service)
	if err != nil {
		log.Warn("appointments email: send failed",
			slog.String("appointment_id", appointment.ID),
			slog.String("email", appointment.Email),
			slog.String("error", err.Error()),
		)
		return
	}

	log.Info("appointments email: sent",
		slog.String("appointment_id", appointment.ID),
		slog.String("email", appointment.Email),
		slog.String("message_id", messageID),
	)
}

func (s *Server) GetAppointment(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	id := chi.URLParam(r, "id")
	if id == "" {
		log.Warn("appointments get: missing id")
		transport.WriteError(w, http.StatusBadRequest, "missing id", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var doc bson.M
	if err := s.Cols.Appointments.FindOne(ctx, bson.M{"_id": id}).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn("appointments get: not found", slog.String("appointment_id", id))
			transport.WriteError(w, http.StatusNotFound, "appointment not found", nil)
			return
		}
		log.Error("appointments get: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("appointments get: ok", slog.String("appointment_id", id))
	transport.WriteJSON(w, http.StatusOK, normalizeID(doc))
}
