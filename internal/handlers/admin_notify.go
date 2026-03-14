package handlers

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"log/slog"
)

func (s *Server) AdminEmails(ctx context.Context) ([]string, error) {
	if s == nil || s.Cols == nil || s.Cols.Users == nil {
		return nil, nil
	}
	cursor, err := s.Cols.Users.Find(ctx, bson.M{"role": "admin", "email": bson.M{"$ne": ""}}, nil)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	emails := make([]string, 0)
	for cursor.Next(ctx) {
		var doc struct {
			Email string `bson:"email"`
		}
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		if doc.Email != "" {
			emails = append(emails, doc.Email)
		}
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return emails, nil
}

func (s *Server) NotifyAdmins(ctx context.Context, subject, htmlBody string) {
	if s == nil || s.Mailer == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	emails, err := s.AdminEmails(ctx)
	if err != nil {
		s.Log.Warn("notify admins: failed to list admins", slog.String("error", err.Error()))
		return
	}
	for _, email := range emails {
		if email == "" {
			continue
		}
		_, err := s.Mailer.SendEmail(ctx, email, "Admin", subject, htmlBody)
		if err != nil {
			s.Log.Warn("notify admins: send failed", slog.String("email", email), slog.String("error", err.Error()))
		}
	}
}
