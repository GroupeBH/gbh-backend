package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gbh-backend/internal/models"
	"go.mongodb.org/mongo-driver/bson"
)

// SendUpcomingAppointmentReminders finds appointments occurring within the next lookahead duration
// and not yet reminded, then notifies all admins and flags them as reminded.
func (s *Server) SendUpcomingAppointmentReminders(ctx context.Context, lookahead time.Duration) {
	if s == nil || s.Cols == nil || s.Cols.Appointments == nil {
		return
	}
	if s.Mailer == nil {
		return
	}

	now := time.Now().In(s.Cfg.Timezone)
	nowStr := now.Format("2006-01-02")
	timeFrom := now.Format("15:04")
	timeTo := now.Add(lookahead).Format("15:04")

	filter := bson.M{
		"date":           nowStr,
		"time":           bson.M{"$gte": timeFrom, "$lte": timeTo},
		"reminderSentAt": bson.M{"$exists": false},
	}

	cursor, err := s.Cols.Appointments.Find(ctx, filter)
	if err != nil {
		s.Log.Warn("appointment reminders: query failed", slog.String("error", err.Error()))
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var appt models.Appointment
		if err := cursor.Decode(&appt); err != nil {
			s.Log.Warn("appointment reminders: decode failed", slog.String("error", err.Error()))
			continue
		}

		subject := "Rendez-vous imminent"
		htmlBody := fmt.Sprintf("<p>Le rendez-vous de <strong>%s</strong> pour le service <strong>%s</strong> est prévu %s à %s.</p><p>ID: %s</p>", appt.Name, appt.ServiceID, appt.Date, appt.Time, appt.ID)
		s.NotifyAdmins(ctx, subject, htmlBody)

		// Mark as reminded
		update := bson.M{"$set": bson.M{"reminderSentAt": time.Now().In(s.Cfg.Timezone)}}
		if _, err := s.Cols.Appointments.UpdateOne(ctx, bson.M{"_id": appt.ID}, update); err != nil {
			s.Log.Warn("appointment reminders: update failed", slog.String("appointment_id", appt.ID), slog.String("error", err.Error()))
		}
	}
	if err := cursor.Err(); err != nil {
		s.Log.Warn("appointment reminders: cursor error", slog.String("error", err.Error()))
	}
}
