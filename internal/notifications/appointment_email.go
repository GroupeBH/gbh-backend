package notifications

import (
	"bytes"
	"html/template"

	"gbh-backend/internal/models"
)

const appointmentConfirmationTemplate = `<!DOCTYPE html>
<html>
<body>
  <p>Bonjour {{.Name}},</p>
  <p>Votre reservation est confirmee. Voici les details :</p>
  <ul>
    <li>Service : {{.ServiceName}}</li>
    <li>Date : {{.Date}}</li>
    <li>Heure : {{.Time}}</li>
    <li>Duree : {{.DurationMinutes}} minutes</li>
    <li>Type : {{.TypeLabel}}</li>
    <li>Paiement : {{.PaymentLabel}}</li>
    <li>Prix : {{.Price}}</li>
    <li>Total : {{.Total}}</li>
    <li>Numero de reservation : {{.AppointmentID}}</li>
  </ul>
  <p>A apporter le jour du rendez-vous :</p>
  <ul>
    <li>Carte d'identite</li>
    <li>Cet email imprime</li>
  </ul>
  <p>Merci.</p>
</body>
</html>`

var appointmentConfirmationTmpl = template.Must(template.New("appointment_confirmation").Parse(appointmentConfirmationTemplate))

type appointmentConfirmationData struct {
	Name            string
	ServiceName     string
	Date            string
	Time            string
	DurationMinutes int
	TypeLabel       string
	PaymentLabel    string
	Price           int
	Total           int
	AppointmentID   string
}

func buildAppointmentConfirmationHTML(appointment models.Appointment, service models.Service) (string, error) {
	data := appointmentConfirmationData{
		Name:            appointment.Name,
		ServiceName:     service.Name,
		Date:            appointment.Date,
		Time:            appointment.Time,
		DurationMinutes: appointment.Duration,
		TypeLabel:       appointmentTypeLabel(appointment.Type),
		PaymentLabel:    paymentMethodLabel(appointment.PaymentMethod),
		Price:           appointment.Price,
		Total:           appointment.Total,
		AppointmentID:   appointment.ID,
	}
	var buf bytes.Buffer
	if err := appointmentConfirmationTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func appointmentTypeLabel(value string) string {
	switch value {
	case models.ConsultationOnline:
		return "En ligne"
	case models.ConsultationPresentiel:
		return "Presentiel"
	default:
		return value
	}
}

func paymentMethodLabel(value string) string {
	switch value {
	case models.PaymentOnline:
		return "En ligne"
	case models.PaymentPlace:
		return "Sur place"
	default:
		return value
	}
}
