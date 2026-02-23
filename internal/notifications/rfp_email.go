package notifications

import (
	"bytes"
	"html/template"

	"gbh-backend/internal/rfp"
)

const rfpLeadNotificationTemplate = `<!DOCTYPE html>
<html>
<body>
  <h3>Nouvelle demande RFP B2B</h3>
  <p><strong>Organisation:</strong> {{.Organization}}</p>
  <p><strong>Domaine:</strong> {{.Domain}}</p>
  <p><strong>Secteur:</strong> {{.Sector}}</p>
  <p><strong>Date limite:</strong> {{.Deadline}}</p>
  <p><strong>Budget:</strong> {{.BudgetRange}}</p>
  <p><strong>Contact:</strong> {{.ContactName}}</p>
  <p><strong>Telephone:</strong> {{.Phone}}</p>
  <p><strong>Email:</strong> {{.Email}}</p>
  <p><strong>Source:</strong> {{.Source}}</p>
  <p><strong>ID:</strong> {{.ID}}</p>
  <p><strong>Description:</strong><br/>{{.Description}}</p>
</body>
</html>`

var rfpLeadNotificationTmpl = template.Must(template.New("rfp_lead_notification").Parse(rfpLeadNotificationTemplate))
var rfpLeadConfirmationTmpl = template.Must(template.New("rfp_lead_confirmation").Parse(rfpLeadConfirmationTemplate))

func buildRFPLeadNotificationHTML(lead rfp.Lead) (string, error) {
	var buf bytes.Buffer
	if err := rfpLeadNotificationTmpl.Execute(&buf, lead); err != nil {
		return "", err
	}
	return buf.String(), nil
}

const rfpLeadConfirmationTemplate = `<!DOCTYPE html>
<html>
<body>
  <p>Bonjour {{.ContactName}},</p>
  <p>Votre demande RFP a bien ete recue.</p>
  <p><strong>ID de verification: {{.ID}}</strong></p>
  <p>Conservez cet ID. Il sera demande pour le suivi de votre dossier.</p>
  <ul>
    <li>Organisation: {{.Organization}}</li>
    <li>Domaine: {{.Domain}}</li>
    <li>Secteur: {{.Sector}}</li>
    <li>Telephone: {{.Phone}}</li>
    <li>Email: {{.Email}}</li>
    <li>Source: {{.Source}}</li>
  </ul>
  <p>Description:</p>
  <p>{{.Description}}</p>
  <p>Merci.</p>
</body>
</html>`

func buildRFPLeadConfirmationHTML(lead rfp.Lead) (string, error) {
	if lead.ContactName == "" {
		lead.ContactName = lead.Organization
	}
	var buf bytes.Buffer
	if err := rfpLeadConfirmationTmpl.Execute(&buf, lead); err != nil {
		return "", err
	}
	return buf.String(), nil
}
