package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gbh-backend/internal/models"
)

const defaultBrevoEndpoint = "https://api.brevo.com/v3/smtp/email"

type BrevoClient struct {
	apiKey      string
	senderEmail string
	senderName  string
	sandbox     bool
	endpoint    string
	httpClient  *http.Client
}

func NewBrevoClient(apiKey, senderEmail, senderName string, sandbox bool) *BrevoClient {
	if strings.TrimSpace(apiKey) == "" || strings.TrimSpace(senderEmail) == "" {
		return nil
	}
	if strings.TrimSpace(senderName) == "" {
		senderName = senderEmail
	}
	fmt.Println("brevo api keys:", apiKey)
	return &BrevoClient{
		apiKey:      apiKey,
		senderEmail: senderEmail,
		senderName:  senderName,
		sandbox:     sandbox,
		endpoint:    defaultBrevoEndpoint,
		httpClient:  &http.Client{Timeout: 8 * time.Second},
	}
}

func (c *BrevoClient) SendAppointmentConfirmation(ctx context.Context, appointment models.Appointment, service models.Service) (string, error) {
	if c == nil {
		return "", errors.New("brevo client is nil")
	}
	subject := fmt.Sprintf("Confirmation de reservation - %s", service.Name)
	htmlBody, err := buildAppointmentConfirmationHTML(appointment, service)
	if err != nil {
		return "", err
	}
	return c.sendHTML(ctx, appointment.Email, appointment.Name, subject, htmlBody)
}

func (c *BrevoClient) sendHTML(ctx context.Context, toEmail, toName, subject, htmlBody string) (string, error) {
	if c == nil {
		return "", errors.New("brevo client is nil")
	}
	if strings.TrimSpace(toEmail) == "" {
		return "", errors.New("missing recipient email")
	}
	if strings.TrimSpace(subject) == "" {
		return "", errors.New("missing subject")
	}
	if strings.TrimSpace(htmlBody) == "" {
		return "", errors.New("missing html body")
	}

	payload := brevoSendRequest{
		Sender: brevoSender{
			Name:  c.senderName,
			Email: c.senderEmail,
		},
		To: []brevoRecipient{
			{
				Email: toEmail,
				Name:  toName,
			},
		},
		Subject:     subject,
		HtmlContent: htmlBody,
	}
	if c.sandbox {
		payload.Headers = map[string]string{
			"X-Sib-Sandbox": "drop",
		}
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("brevo marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("brevo create request: %w", err)
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("brevo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("brevo send failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out brevoSendResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("brevo decode response: %w", err)
	}
	if strings.TrimSpace(out.MessageID) == "" {
		return "", errors.New("brevo response missing messageId")
	}
	return out.MessageID, nil
}

type brevoSendRequest struct {
	Sender      brevoSender       `json:"sender"`
	To          []brevoRecipient  `json:"to"`
	Subject     string            `json:"subject"`
	HtmlContent string            `json:"htmlContent,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

type brevoSender struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type brevoRecipient struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type brevoSendResponse struct {
	MessageID string `json:"messageId"`
}
