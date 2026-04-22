package notifications

import (
	"strings"
	"testing"

	"gbh-backend/internal/models"
)

const officeAddressSnippet = "Boulevard Sendwe, immeuble Adi Construct, quatrieme niveau, commune de Kalamu, quartier Matonge."

func TestBuildAppointmentConfirmationHTMLIncludesOfficeAddressForPresentiel(t *testing.T) {
	appointment := models.Appointment{
		ID:            "RDV-001",
		Name:          "Jean",
		Type:          models.ConsultationPresentiel,
		Date:          "2026-04-23",
		Time:          "10:00",
		Duration:      45,
		Price:         100,
		Total:         118,
		PaymentMethod: models.PaymentPlace,
	}
	service := models.Service{Name: "Consultation"}

	html, err := buildAppointmentConfirmationHTML(appointment, service)
	if err != nil {
		t.Fatalf("buildAppointmentConfirmationHTML() error = %v", err)
	}

	if !strings.Contains(html, officeAddressSnippet) {
		t.Fatalf("expected office address in confirmation email, got %q", html)
	}
}

func TestBuildAppointmentConfirmationHTMLOmitsOfficeAddressForOnline(t *testing.T) {
	appointment := models.Appointment{
		ID:            "RDV-002",
		Name:          "Amina",
		Type:          models.ConsultationOnline,
		Date:          "2026-04-23",
		Time:          "11:00",
		Duration:      45,
		Price:         100,
		Total:         118,
		PaymentMethod: models.PaymentOnline,
	}
	service := models.Service{Name: "Consultation"}

	html, err := buildAppointmentConfirmationHTML(appointment, service)
	if err != nil {
		t.Fatalf("buildAppointmentConfirmationHTML() error = %v", err)
	}

	if strings.Contains(html, officeAddressSnippet) {
		t.Fatalf("did not expect office address in online confirmation email, got %q", html)
	}
}
