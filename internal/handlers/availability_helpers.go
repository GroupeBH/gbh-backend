package handlers

import (
	"context"
	"time"

	"gbh-backend/internal/schedule"
	"go.mongodb.org/mongo-driver/bson"
)

func (s *Server) reservedIntervals(ctx context.Context, date string) ([]schedule.Interval, error) {
	intervals := make([]schedule.Interval, 0)

	appCursor, err := s.Cols.Appointments.Find(ctx, bson.M{"date": date})
	if err != nil {
		return nil, err
	}
	for appCursor.Next(ctx) {
		var doc bson.M
		if err := appCursor.Decode(&doc); err != nil {
			continue
		}
		timeStr, ok := doc["time"].(string)
		if !ok || timeStr == "" {
			continue
		}
		start, err := schedule.ParseClockToMinutes(timeStr)
		if err != nil {
			continue
		}
		duration := extractInt(doc["duration"])
		if duration <= 0 {
			duration = schedule.SlotMinutes
		}
		intervals = append(intervals, schedule.Interval{Start: start, End: start + duration})
	}
	if err := appCursor.Err(); err != nil {
		return nil, err
	}
	appCursor.Close(ctx)

	blockCursor, err := s.Cols.ReservationBlocks.Find(ctx, bson.M{"date": date})
	if err != nil {
		return nil, err
	}
	for blockCursor.Next(ctx) {
		var doc bson.M
		if err := blockCursor.Decode(&doc); err != nil {
			continue
		}
		timeStr, ok := doc["time"].(string)
		if !ok || timeStr == "" {
			continue
		}
		start, err := schedule.ParseClockToMinutes(timeStr)
		if err != nil {
			continue
		}
		intervals = append(intervals, schedule.Interval{Start: start, End: start + schedule.SlotMinutes})
	}
	if err := blockCursor.Err(); err != nil {
		return nil, err
	}
	blockCursor.Close(ctx)

	return intervals, nil
}

func (s *Server) computeAvailableSlots(ctx context.Context, date string, duration int, now time.Time) ([]string, error) {
	slots, err := schedule.GenerateSlotsWithDuration(date, duration, s.Cfg.Timezone)
	if err != nil {
		return nil, err
	}

	intervals, err := s.reservedIntervals(ctx, date)
	if err != nil {
		return nil, err
	}

	slots, err = schedule.FilterOverlapping(slots, duration, intervals)
	if err != nil {
		return nil, err
	}

	if dateIsToday(date, s.Cfg.Timezone) {
		slots, err = schedule.FilterPastSlots(date, slots, s.Cfg.Timezone, now)
		if err != nil {
			return nil, err
		}
	}

	return slots, nil
}
