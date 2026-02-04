package schedule

import (
	"errors"
	"time"
)

const SlotMinutes = 45

var (
	ErrInvalidDate = errors.New("invalid date format")
	ErrInvalidTime = errors.New("invalid time format")
)

type TimeRange struct {
	Start string
	End   string
}

func ParseDate(dateStr string, loc *time.Location) (time.Time, error) {
	date, err := time.ParseInLocation("2006-01-02", dateStr, loc)
	if err != nil {
		return time.Time{}, ErrInvalidDate
	}
	return date, nil
}

func ParseDateTime(dateStr, timeStr string, loc *time.Location) (time.Time, error) {
	if _, err := time.Parse("15:04", timeStr); err != nil {
		return time.Time{}, ErrInvalidTime
	}
	_, err := ParseDate(dateStr, loc)
	if err != nil {
		return time.Time{}, err
	}

	parsed, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+timeStr, loc)
	if err != nil {
		return time.Time{}, ErrInvalidTime
	}

	return parsed, nil
}

func IsDatePast(dateStr string, loc *time.Location, now time.Time) (bool, error) {
	date, err := ParseDate(dateStr, loc)
	if err != nil {
		return false, err
	}
	startToday := time.Date(now.In(loc).Year(), now.In(loc).Month(), now.In(loc).Day(), 0, 0, 0, 0, loc)
	return date.Before(startToday), nil
}

func IsSlotPast(dateStr, timeStr string, loc *time.Location, now time.Time) (bool, error) {
	slot, err := ParseDateTime(dateStr, timeStr, loc)
	if err != nil {
		return false, err
	}
	return !slot.After(now.In(loc)), nil
}

func dayRanges(day time.Weekday) []TimeRange {
	switch day {
	case time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday:
		return []TimeRange{{Start: "09:00", End: "12:00"}, {Start: "14:00", End: "17:00"}}
	case time.Saturday:
		return []TimeRange{{Start: "09:00", End: "13:00"}}
	default:
		return nil
	}
}

func GenerateSlots(dateStr string, loc *time.Location) ([]string, error) {
	date, err := ParseDate(dateStr, loc)
	if err != nil {
		return nil, err
	}

	ranges := dayRanges(date.Weekday())
	if len(ranges) == 0 {
		return []string{}, nil
	}

	slots := make([]string, 0)
	for _, tr := range ranges {
		start, err := ParseDateTime(dateStr, tr.Start, loc)
		if err != nil {
			return nil, err
		}
		end, err := ParseDateTime(dateStr, tr.End, loc)
		if err != nil {
			return nil, err
		}

		for cursor := start; cursor.Add(time.Minute*SlotMinutes).Equal(end) || cursor.Add(time.Minute*SlotMinutes).Before(end); cursor = cursor.Add(time.Minute * SlotMinutes) {
			slots = append(slots, cursor.Format("15:04"))
		}
	}

	return slots, nil
}

func FilterReserved(slots []string, reserved map[string]bool) []string {
	filtered := make([]string, 0, len(slots))
	for _, s := range slots {
		if !reserved[s] {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func FilterPastSlots(dateStr string, slots []string, loc *time.Location, now time.Time) ([]string, error) {
	filtered := make([]string, 0, len(slots))
	for _, s := range slots {
		past, err := IsSlotPast(dateStr, s, loc, now)
		if err != nil {
			return nil, err
		}
		if !past {
			filtered = append(filtered, s)
		}
	}
	return filtered, nil
}

func IsSlotAllowed(dateStr, timeStr string, loc *time.Location) (bool, error) {
	slots, err := GenerateSlots(dateStr, loc)
	if err != nil {
		return false, err
	}
	for _, s := range slots {
		if s == timeStr {
			return true, nil
		}
	}
	return false, nil
}

func IsSlotAvailable(dateStr, timeStr string, loc *time.Location, now time.Time, reserved map[string]bool) (bool, error) {
	pastDate, err := IsDatePast(dateStr, loc, now)
	if err != nil {
		return false, err
	}
	if pastDate {
		return false, nil
	}

	allowed, err := IsSlotAllowed(dateStr, timeStr, loc)
	if err != nil || !allowed {
		return false, err
	}

	pastSlot, err := IsSlotPast(dateStr, timeStr, loc, now)
	if err != nil {
		return false, err
	}
	if pastSlot {
		return false, nil
	}

	if reserved != nil && reserved[timeStr] {
		return false, nil
	}
	return true, nil
}
