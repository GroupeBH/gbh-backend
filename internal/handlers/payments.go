package handlers

import (
	"context"
	"net/http"
	"time"

	"gbh-backend/internal/models"
	"gbh-backend/internal/transport"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PaymentIntentRequest struct {
	AppointmentID string `json:"appointmentId" validate:"required"`
}

type PaymentIntentResponse struct {
	IntentID string `json:"intentId"`
	Status   string `json:"status"`
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
	Method   string `json:"method"`
}

func (s *Server) CreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	var req PaymentIntentRequest
	if err := decodeJSON(r, &req); err != nil {
		transport.WriteError(w, http.StatusBadRequest, "invalid json", nil)
		return
	}

	if err := s.Val.Struct(req); err != nil {
		details := validationDetails(s.Val.ValidationErrors(err))
		transport.WriteError(w, http.StatusBadRequest, "validation error", details)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var doc bson.M
	if err := s.Cols.Appointments.FindOne(ctx, bson.M{"_id": req.AppointmentID}).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			transport.WriteError(w, http.StatusNotFound, "appointment not found", nil)
			return
		}
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}

	method, _ := doc["paymentMethod"].(string)
	amount := extractInt(doc["total"])

	if method != models.PaymentOnline {
		transport.WriteJSON(w, http.StatusOK, PaymentIntentResponse{
			IntentID: "",
			Status:   "not_required",
			Amount:   amount,
			Currency: "CDF",
			Method:   method,
		})
		return
	}

	resp := PaymentIntentResponse{
		IntentID: primitive.NewObjectID().Hex(),
		Status:   "created",
		Amount:   amount,
		Currency: "CDF",
		Method:   method,
	}

	transport.WriteJSON(w, http.StatusOK, resp)
}
