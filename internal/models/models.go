package models

import "time"

const (
	ConsultationOnline     = "online"
	ConsultationPresentiel = "presentiel"

	PaymentOnline = "online"
	PaymentPlace  = "place"

	AppointmentStatusPending  = "pending"
	AppointmentStatusBooked   = "booked"
	AppointmentStatusCanceled = "canceled"
)

type Service struct {
	ID          string    `bson:"_id,omitempty" json:"id"`
	Name        string    `bson:"name" json:"name"`
	Description string    `bson:"description" json:"description"`
	Category    string    `bson:"category" json:"category"`
	ForAudience string    `bson:"forAudience" json:"forAudience"`
	Slug        string    `bson:"slug" json:"slug"`
	CreatedAt   time.Time `bson:"createdAt" json:"createdAt"`
}

type Appointment struct {
	ID            string    `bson:"_id,omitempty" json:"id"`
	ServiceID     string    `bson:"serviceId" json:"serviceId"`
	Name          string    `bson:"name" json:"name"`
	Email         string    `bson:"email" json:"email"`
	Phone         string    `bson:"phone" json:"phone"`
	Type          string    `bson:"type" json:"type"`
	Date          string    `bson:"date" json:"date"`
	Time          string    `bson:"time" json:"time"`
	Duration      int       `bson:"duration" json:"duration"`
	Price         int       `bson:"price" json:"price"`
	Tax           int       `bson:"tax" json:"tax"`
	Total         int       `bson:"total" json:"total"`
	Status        string    `bson:"status" json:"status"`
	PaymentMethod string    `bson:"paymentMethod" json:"paymentMethod"`
	CreatedAt     time.Time `bson:"createdAt" json:"createdAt"`
}

type ContactMessage struct {
	ID        string    `bson:"_id,omitempty" json:"id"`
	Name      string    `bson:"name" json:"name"`
	Email     string    `bson:"email" json:"email"`
	Phone     string    `bson:"phone" json:"phone"`
	Subject   string    `bson:"subject" json:"subject"`
	Message   string    `bson:"message" json:"message"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

type ReservationBlock struct {
	ID        string    `bson:"_id,omitempty" json:"id"`
	Date      string    `bson:"date" json:"date"`
	Time      string    `bson:"time" json:"time"`
	Reason    string    `bson:"reason" json:"reason"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}
