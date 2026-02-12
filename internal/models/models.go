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

	UserRoleAdmin = "admin"
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

type User struct {
	ID           string    `bson:"_id,omitempty" json:"id"`
	Username     string    `bson:"username" json:"username"`
	Email        string    `bson:"email,omitempty" json:"email,omitempty"`
	PasswordHash string    `bson:"passwordHash" json:"-"`
	Role         string    `bson:"role" json:"role"`
	CreatedAt    time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time `bson:"updatedAt" json:"updatedAt"`
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
