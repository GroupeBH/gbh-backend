package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gbh-backend/internal/models"
	"gbh-backend/internal/transport"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ServiceTestimonialRequest struct {
	Name    string `json:"name" validate:"required,max=120"`
	Rating  int    `json:"rating" validate:"required,gte=1,lte=5"`
	Message string `json:"message" validate:"required,max=2000"`
}

func (s *Server) GetServiceTestimonials(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	serviceID := chi.URLParam(r, "id")
	if serviceID == "" {
		log.Warn("service testimonials list: missing service id")
		transport.WriteError(w, http.StatusBadRequest, "missing service id", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if exists, err := s.serviceExists(ctx, serviceID); err != nil {
		log.Error("service testimonials list: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	} else if !exists {
		log.Warn("service testimonials list: service not found", slog.String("service_id", serviceID))
		transport.WriteError(w, http.StatusNotFound, "service not found", nil)
		return
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(200)
	cursor, err := s.Cols.ServiceTestimonials.Find(ctx, bson.M{"serviceId": serviceID}, opts)
	if err != nil {
		log.Error("service testimonials list: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}
	defer cursor.Close(ctx)

	var items []map[string]interface{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			log.Error("service testimonials list: decode error", slog.String("error", err.Error()))
			transport.WriteError(w, http.StatusInternalServerError, "decode error", nil)
			return
		}
		items = append(items, normalizeID(doc))
	}
	if err := cursor.Err(); err != nil {
		log.Error("service testimonials list: cursor error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "cursor error", nil)
		return
	}

	log.Info("service testimonials list: ok", slog.String("service_id", serviceID), slog.Int("count", len(items)))
	transport.WriteJSON(w, http.StatusOK, map[string]interface{}{"testimonials": items})
}

func (s *Server) CreateServiceTestimonial(w http.ResponseWriter, r *http.Request) {
	log := s.logWithRequest(r)
	serviceID := chi.URLParam(r, "id")
	if serviceID == "" {
		log.Warn("service testimonials create: missing service id")
		transport.WriteError(w, http.StatusBadRequest, "missing service id", nil)
		return
	}

	var req ServiceTestimonialRequest
	if err := decodeJSON(r, &req); err != nil {
		log.Warn("service testimonials create: invalid json")
		transport.WriteError(w, http.StatusBadRequest, "invalid json", nil)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Message = strings.TrimSpace(req.Message)
	if err := s.Val.Struct(req); err != nil {
		log.Warn("service testimonials create: validation error")
		details := validationDetails(s.Val.ValidationErrors(err))
		transport.WriteError(w, http.StatusBadRequest, "validation error", details)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if exists, err := s.serviceExists(ctx, serviceID); err != nil {
		log.Error("service testimonials create: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	} else if !exists {
		log.Warn("service testimonials create: service not found", slog.String("service_id", serviceID))
		transport.WriteError(w, http.StatusNotFound, "service not found", nil)
		return
	}

	testimonial := models.ServiceTestimonial{
		ID:        primitive.NewObjectID().Hex(),
		ServiceID: serviceID,
		Name:      req.Name,
		Rating:    req.Rating,
		Message:   req.Message,
		CreatedAt: time.Now().In(s.Cfg.Timezone),
	}

	if _, err := s.Cols.ServiceTestimonials.InsertOne(ctx, testimonial); err != nil {
		log.Error("service testimonials create: database error", slog.String("error", err.Error()))
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	log.Info("service testimonials create: ok", slog.String("service_id", serviceID), slog.String("testimonial_id", testimonial.ID))
	transport.WriteJSON(w, http.StatusCreated, testimonial)
}

func (s *Server) serviceExists(ctx context.Context, serviceID string) (bool, error) {
	if err := s.Cols.Services.FindOne(ctx, bson.M{"_id": serviceID}).Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
