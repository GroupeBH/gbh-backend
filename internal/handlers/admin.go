package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"gbh-backend/internal/models"
	"gbh-backend/internal/schedule"
	"gbh-backend/internal/transport"
	"gbh-backend/internal/utils"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AdminServiceRequest struct {
	Name             string   `json:"name" validate:"required"`
	ShortDescription string   `json:"shortDescription"`
	Description      string   `json:"description" validate:"required"`
	Benefits         []string `json:"benefits" validate:"omitempty,dive,required"`
	Category         string   `json:"category" validate:"required"`
	ForAudience      string   `json:"forAudience" validate:"required"`
	Slug             string   `json:"slug"`
}

type AdminBlockRequest struct {
	Date   string `json:"date" validate:"required,date"`
	Time   string `json:"time" validate:"required,clock"`
	Reason string `json:"reason" validate:"required"`
}

type AdminStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=booked canceled"`
}

type AdminListQuery struct {
	Date string `validate:"omitempty,date"`
}

func (s *Server) AdminCreateService(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	var req AdminServiceRequest
	if err := decodeJSON(r, &req); err != nil {
		log.Warn("admin services create: invalid json")
		transport.WriteError(w, http.StatusBadRequest, "invalid json", nil)
		return
	}
	if err := s.Val.Struct(req); err != nil {
		log.Warn("admin services create: validation error")
		details := validationDetails(s.Val.ValidationErrors(err))
		transport.WriteError(w, http.StatusBadRequest, "validation error", details)
		return
	}

	slug := req.Slug
	if slug == "" {
		slug = utils.Slugify(req.Name)
	}

	service := models.Service{
		ID:               primitive.NewObjectID().Hex(),
		Name:             req.Name,
		ShortDescription: req.ShortDescription,
		Description:      req.Description,
		Benefits:         normalizeStringList(req.Benefits),
		Category:         req.Category,
		ForAudience:      req.ForAudience,
		Slug:             slug,
		CreatedAt:        time.Now().In(s.Cfg.Timezone),
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_, err := s.Cols.Services.InsertOne(ctx, service)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			log.Warn("admin services create: slug exists", slog.String("slug", slug))
			transport.WriteError(w, http.StatusConflict, "slug already exists", nil)
			return
		}
		log.Error("admin services create: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	if s.Cache != nil {
		_ = s.Cache.Delete(r.Context(), "services:all")
	}

	log.Info("admin services create: ok", slog.String("service_id", service.ID), slog.String("slug", slug))
	transport.WriteJSON(w, http.StatusCreated, service)
}

func (s *Server) AdminUpdateService(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	id := chi.URLParam(r, "id")
	if id == "" {
		log.Warn("admin services update: missing id")
		transport.WriteError(w, http.StatusBadRequest, "missing id", nil)
		return
	}

	var req AdminServiceRequest
	if err := decodeJSON(r, &req); err != nil {
		log.Warn("admin services update: invalid json")
		transport.WriteError(w, http.StatusBadRequest, "invalid json", nil)
		return
	}
	if err := s.Val.Struct(req); err != nil {
		log.Warn("admin services update: validation error")
		details := validationDetails(s.Val.ValidationErrors(err))
		transport.WriteError(w, http.StatusBadRequest, "validation error", details)
		return
	}

	slug := req.Slug
	if slug == "" {
		slug = utils.Slugify(req.Name)
	}

	update := bson.M{
		"$set": bson.M{
			"name":             req.Name,
			"shortDescription": req.ShortDescription,
			"description":      req.Description,
			"benefits":         normalizeStringList(req.Benefits),
			"category":         req.Category,
			"forAudience":      req.ForAudience,
			"slug":             slug,
		},
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res, err := s.Cols.Services.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			log.Warn("admin services update: slug exists", slog.String("slug", slug))
			transport.WriteError(w, http.StatusConflict, "slug already exists", nil)
			return
		}
		log.Error("admin services update: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}
	if res.MatchedCount == 0 {
		log.Warn("admin services update: not found", slog.String("service_id", id))
		transport.WriteError(w, http.StatusNotFound, "service not found", nil)
		return
	}

	var doc bson.M
	if err := s.Cols.Services.FindOne(ctx, bson.M{"_id": id}).Decode(&doc); err != nil {
		log.Error("admin services update: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	if s.Cache != nil {
		_ = s.Cache.Delete(r.Context(), "services:all")
	}

	log.Info("admin services update: ok", slog.String("service_id", id))
	transport.WriteJSON(w, http.StatusOK, normalizeID(doc))
}

func (s *Server) AdminDeleteService(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	id := chi.URLParam(r, "id")
	if id == "" {
		log.Warn("admin services delete: missing id")
		transport.WriteError(w, http.StatusBadRequest, "missing id", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res, err := s.Cols.Services.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		log.Error("admin services delete: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}
	if res.DeletedCount == 0 {
		log.Warn("admin services delete: not found", slog.String("service_id", id))
		transport.WriteError(w, http.StatusNotFound, "service not found", nil)
		return
	}

	if s.Cache != nil {
		_ = s.Cache.Delete(r.Context(), "services:all")
	}

	log.Info("admin services delete: ok", slog.String("service_id", id))
	transport.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) AdminCreateBlock(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	var req AdminBlockRequest
	if err := decodeJSON(r, &req); err != nil {
		log.Warn("admin blocks create: invalid json")
		transport.WriteError(w, http.StatusBadRequest, "invalid json", nil)
		return
	}
	if err := s.Val.Struct(req); err != nil {
		log.Warn("admin blocks create: validation error")
		details := validationDetails(s.Val.ValidationErrors(err))
		transport.WriteError(w, http.StatusBadRequest, "validation error", details)
		return
	}

	pastDate, err := schedule.IsDatePast(req.Date, s.Cfg.Timezone, time.Now())
	if err != nil {
		log.Warn("admin blocks create: invalid date", slog.String("date", req.Date))
		transport.WriteError(w, http.StatusBadRequest, "invalid date", nil)
		return
	}
	if pastDate {
		log.Warn("admin blocks create: date in the past", slog.String("date", req.Date))
		transport.WriteError(w, http.StatusBadRequest, "date in the past", nil)
		return
	}

	allowed, err := schedule.IsSlotAllowedWithDuration(req.Date, req.Time, schedule.SlotMinutes, s.Cfg.Timezone)
	if err != nil || !allowed {
		log.Warn("admin blocks create: slot not allowed", slog.String("date", req.Date), slog.String("time", req.Time))
		transport.WriteError(w, http.StatusBadRequest, "slot not available", nil)
		return
	}

	if dateIsToday(req.Date, s.Cfg.Timezone) {
		pastSlot, err := schedule.IsSlotPast(req.Date, req.Time, s.Cfg.Timezone, time.Now())
		if err != nil {
			log.Warn("admin blocks create: invalid time", slog.String("time", req.Time))
			transport.WriteError(w, http.StatusBadRequest, "invalid time", nil)
			return
		}
		if pastSlot {
			log.Warn("admin blocks create: slot already passed", slog.String("date", req.Date), slog.String("time", req.Time))
			transport.WriteError(w, http.StatusBadRequest, "slot already passed", nil)
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	reserved, err := s.reservedIntervals(ctx, req.Date)
	if err != nil {
		log.Error("admin blocks create: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}
	startMin, err := schedule.ParseClockToMinutes(req.Time)
	if err != nil {
		log.Warn("admin blocks create: invalid time", slog.String("time", req.Time))
		transport.WriteError(w, http.StatusBadRequest, "invalid time", nil)
		return
	}
	current := schedule.Interval{Start: startMin, End: startMin + schedule.SlotMinutes}
	for _, interval := range reserved {
		if schedule.Overlaps(current, interval) {
			log.Warn("admin blocks create: slot overlap", slog.String("date", req.Date), slog.String("time", req.Time))
			transport.WriteError(w, http.StatusConflict, "slot not available", nil)
			return
		}
	}

	block := models.ReservationBlock{
		ID:        primitive.NewObjectID().Hex(),
		Date:      req.Date,
		Time:      req.Time,
		Reason:    req.Reason,
		CreatedAt: time.Now().In(s.Cfg.Timezone),
	}

	if _, err := s.Cols.ReservationBlocks.InsertOne(ctx, block); err != nil {
		log.Error("admin blocks create: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	if s.Cache != nil {
		_ = s.Cache.DeletePrefix(r.Context(), "availability:"+req.Date+":")
	}

	log.Info("admin blocks create: ok", slog.String("block_id", block.ID), slog.String("date", block.Date), slog.String("time", block.Time))
	transport.WriteJSON(w, http.StatusCreated, block)
}

func (s *Server) AdminDeleteBlock(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	id := chi.URLParam(r, "id")
	if id == "" {
		log.Warn("admin blocks delete: missing id")
		transport.WriteError(w, http.StatusBadRequest, "missing id", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var doc bson.M
	if err := s.Cols.ReservationBlocks.FindOne(ctx, bson.M{"_id": id}).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn("admin blocks delete: not found", slog.String("block_id", id))
			transport.WriteError(w, http.StatusNotFound, "block not found", nil)
			return
		}
		log.Error("admin blocks delete: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	_, err := s.Cols.ReservationBlocks.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		log.Error("admin blocks delete: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	if date, ok := doc["date"].(string); ok {
		if s.Cache != nil {
			_ = s.Cache.DeletePrefix(r.Context(), "availability:"+date+":")
		}
	}

	log.Info("admin blocks delete: ok", slog.String("block_id", id))
	transport.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) AdminListAppointments(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	q := AdminListQuery{Date: r.URL.Query().Get("date")}
	if err := s.Val.Struct(q); err != nil {
		log.Warn("admin appointments list: invalid query")
		details := validationDetails(s.Val.ValidationErrors(err))
		transport.WriteError(w, http.StatusBadRequest, "invalid query", details)
		return
	}

	filter := bson.M{}
	if q.Date != "" {
		filter["date"] = q.Date
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "date", Value: 1}, {Key: "time", Value: 1}}).SetLimit(200)
	cursor, err := s.Cols.Appointments.Find(ctx, filter, opts)
	if err != nil {
		log.Error("admin appointments list: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}
	defer cursor.Close(ctx)

	var items []map[string]interface{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err == nil {
			items = append(items, normalizeID(doc))
		}
	}
	if err := cursor.Err(); err != nil {
		log.Error("admin appointments list: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("admin appointments list: ok", slog.Int("count", len(items)))
	transport.WriteJSON(w, http.StatusOK, map[string]interface{}{"appointments": items})
}

func (s *Server) AdminUpdateAppointmentStatus(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	id := chi.URLParam(r, "id")
	if id == "" {
		log.Warn("admin appointments status: missing id")
		transport.WriteError(w, http.StatusBadRequest, "missing id", nil)
		return
	}

	var req AdminStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		log.Warn("admin appointments status: invalid json")
		transport.WriteError(w, http.StatusBadRequest, "invalid json", nil)
		return
	}
	if err := s.Val.Struct(req); err != nil {
		log.Warn("admin appointments status: validation error")
		details := validationDetails(s.Val.ValidationErrors(err))
		transport.WriteError(w, http.StatusBadRequest, "validation error", details)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var doc bson.M
	if err := s.Cols.Appointments.FindOne(ctx, bson.M{"_id": id}).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn("admin appointments status: not found", slog.String("appointment_id", id))
			transport.WriteError(w, http.StatusNotFound, "appointment not found", nil)
			return
		}
		log.Error("admin appointments status: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	_, err := s.Cols.Appointments.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"status": req.Status}})
	if err != nil {
		log.Error("admin appointments status: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	if date, ok := doc["date"].(string); ok {
		if s.Cache != nil {
			_ = s.Cache.DeletePrefix(r.Context(), "availability:"+date+":")
		}
	}

	doc["status"] = req.Status
	log.Info("admin appointments status: ok", slog.String("appointment_id", id), slog.String("status", req.Status))
	transport.WriteJSON(w, http.StatusOK, normalizeID(doc))
}

func (s *Server) AdminListContacts(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(200)
	cursor, err := s.Cols.ContactMessages.Find(ctx, bson.M{}, opts)
	if err != nil {
		log.Error("admin contacts list: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}
	defer cursor.Close(ctx)

	var items []map[string]interface{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err == nil {
			items = append(items, normalizeID(doc))
		}
	}
	if err := cursor.Err(); err != nil {
		log.Error("admin contacts list: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("admin contacts list: ok", slog.Int("count", len(items)))
	transport.WriteJSON(w, http.StatusOK, map[string]interface{}{"contacts": items})
}
