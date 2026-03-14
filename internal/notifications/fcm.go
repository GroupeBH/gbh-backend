package notifications

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gbh-backend/internal/models"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// FCMClient sends Firebase Cloud Messaging (FCM) push notifications.
// It is designed to be safe to call from multiple goroutines.
type FCMClient struct {
	client *messaging.Client
}

// NewFCMClient creates a new FCMClient.
// If credentialsFile is empty, the SDK will instead attempt to use
// the default application credentials (e.g. via GOOGLE_APPLICATION_CREDENTIALS).
func NewFCMClient(ctx context.Context, credentialsFile string) (*FCMClient, error) {
	var opts []option.ClientOption
	if strings.TrimSpace(credentialsFile) != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsFile))
	}

	app, err := firebase.NewApp(ctx, nil, opts...)
	if err != nil {
		return nil, fmt.Errorf("firebase new app: %w", err)
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase messaging client: %w", err)
	}
	return &FCMClient{client: client}, nil
}

// NewFCMClientFromJSON creates a new FCMClient from JSON credentials content.
func NewFCMClientFromJSON(ctx context.Context, credentialsJSON []byte) (*FCMClient, error) {
	opts := []option.ClientOption{option.WithCredentialsJSON(credentialsJSON)}

	app, err := firebase.NewApp(ctx, nil, opts...)
	if err != nil {
		return nil, fmt.Errorf("firebase new app from json: %w", err)
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase messaging client: %w", err)
	}
	return &FCMClient{client: client}, nil
}

// SendAppointmentConfirmation sends a push notification to the provided device token.
func (c *FCMClient) SendAppointmentConfirmation(ctx context.Context, deviceToken string, appointment models.Appointment, service models.Service) (string, error) {
	if c == nil || c.client == nil {
		return "", errors.New("fcm client is nil")
	}
	deviceToken = strings.TrimSpace(deviceToken)
	if deviceToken == "" {
		return "", errors.New("missing device token")
	}

	title := "Votre rendez-vous est confirmé"
	body := fmt.Sprintf("%s le %s à %s", service.Name, appointment.Date, appointment.Time)

	msg := &messaging.Message{
		Token: deviceToken,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: map[string]string{
			"appointmentId": appointment.ID,
			"serviceId":     appointment.ServiceID,
			"date":          appointment.Date,
			"time":          appointment.Time,
		},
	}

	return c.client.Send(ctx, msg)
}
