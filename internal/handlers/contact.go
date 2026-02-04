package handlers

import (
	"context"
	"net/http"
	"time"

	"gbh-backend/internal/models"
	"gbh-backend/internal/transport"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ContactRequest struct {
	Name    string `json:"name" validate:"required"`
	Email   string `json:"email" validate:"required,email"`
	Phone   string `json:"phone" validate:"required,phone"`
	Subject string `json:"subject" validate:"required"`
	Message string `json:"message" validate:"required"`
}

func (s *Server) CreateContact(w http.ResponseWriter, r *http.Request) {
	var req ContactRequest
	if err := decodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest, "invalid json", nil)
		return
	}

	if err := s.Val.Struct(req); err != nil {
		details := validationDetails(s.Val.ValidationErrors(err))
		transport.WriteError(w, http.StatusBadRequest, "validation error", details)
		return
	}

	msg := models.ContactMessage{
		ID:        primitive.NewObjectID().Hex(),
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
		Subject:   req.Subject,
		Message:   req.Message,
		CreatedAt: time.Now().In(s.Cfg.Timezone),
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, err := s.Cols.ContactMessages.InsertOne(ctx, msg); err != nil {
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	transport.WriteJSON(w, http.StatusCreated, msg)
}
