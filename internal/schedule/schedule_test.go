package schedule

import (
	"testing"
	"time"
)

func mustLoadLoc(t *testing.T) *time.Location {
	loc, err := time.LoadLocation("Africa/Kinshasa")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	return loc
}

func TestGenerateSlotsWeekday(t *testing.T) {
	loc := mustLoadLoc(t)
	slots, err := GenerateSlots("2026-02-02", loc)
	if err != nil {
		t.Fatalf("GenerateSlots error: %v", err)
	}
	if len(slots) != 8 {
		t.Fatalf("expected 8 slots, got %d", len(slots))
	}
	if slots[0] != "09:00" || slots[len(slots)-1] != "16:15" {
		t.Fatalf("unexpected boundary slots: %v", slots)
	}
}

func TestGenerateSlotsSaturday(t *testing.T) {
	loc := mustLoadLoc(t)
	slots, err := GenerateSlots("2026-02-07", loc)
	if err != nil {
		t.Fatalf("GenerateSlots error: %v", err)
	}
	if len(slots) != 5 {
		t.Fatalf("expected 5 slots, got %d", len(slots))
	}
	if slots[0] != "09:00" || slots[len(slots)-1] != "12:00" {
		t.Fatalf("unexpected boundary slots: %v", slots)
	}
}

func TestGenerateSlotsSundayClosed(t *testing.T) {
	loc := mustLoadLoc(t)
	slots, err := GenerateSlots("2026-02-01", loc)
	if err != nil {
		t.Fatalf("GenerateSlots error: %v", err)
	}
	if len(slots) != 0 {
		t.Fatalf("expected 0 slots, got %d", len(slots))
	}
}

func TestIsDatePast(t *testing.T) {
	loc := mustLoadLoc(t)
	now := time.Date(2026, 2, 4, 10, 0, 0, 0, loc)
	past, err := IsDatePast("2026-02-03", loc, now)
	if err != nil {
		t.Fatalf("IsDatePast error: %v", err)
	}
	if !past {
		t.Fatalf("expected date to be past")
	}

	past, err = IsDatePast("2026-02-04", loc, now)
	if err != nil {
		t.Fatalf("IsDatePast error: %v", err)
	}
	if past {
		t.Fatalf("expected date to be not past")
	}
}

func TestFilterReserved(t *testing.T) {
	slots := []string{"09:00", "09:45", "10:30"}
	reserved := map[string]bool{"09:45": true}
	filtered := FilterReserved(slots, reserved)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(filtered))
	}
	if filtered[1] != "10:30" {
		t.Fatalf("unexpected slots: %v", filtered)
	}
}

func TestIsSlotPast(t *testing.T) {
	loc := mustLoadLoc(t)
	now := time.Date(2026, 2, 4, 10, 0, 0, 0, loc)
	past, err := IsSlotPast("2026-02-04", "09:00", loc, now)
	if err != nil {
		t.Fatalf("IsSlotPast error: %v", err)
	}
	if !past {
		t.Fatalf("expected slot to be past")
	}
	past, err = IsSlotPast("2026-02-04", "10:30", loc, now)
	if err != nil {
		t.Fatalf("IsSlotPast error: %v", err)
	}
	if past {
		t.Fatalf("expected slot to be future")
	}
}

func TestIsSlotAllowed(t *testing.T) {
	loc := mustLoadLoc(t)
	ok, err := IsSlotAllowed("2026-02-04", "14:45", loc)
	if err != nil {
		t.Fatalf("IsSlotAllowed error: %v", err)
	}
	if !ok {
		t.Fatalf("expected slot to be allowed")
	}

	ok, err = IsSlotAllowed("2026-02-04", "13:00", loc)
	if err != nil {
		t.Fatalf("IsSlotAllowed error: %v", err)
	}
	if ok {
		t.Fatalf("expected slot to be not allowed")
	}
}

func TestIsSlotAvailableWithConflict(t *testing.T) {
	loc := mustLoadLoc(t)
	now := time.Date(2026, 2, 4, 8, 0, 0, 0, loc)
	reserved := map[string]bool{"09:00": true}

	ok, err := IsSlotAvailable("2026-02-04", "09:00", loc, now, reserved)
	if err != nil {
		t.Fatalf("IsSlotAvailable error: %v", err)
	}
	if ok {
		t.Fatalf("expected slot to be unavailable due to conflict")
	}
}

func TestIsSlotAvailableHappyPath(t *testing.T) {
	loc := mustLoadLoc(t)
	now := time.Date(2026, 2, 4, 8, 0, 0, 0, loc)
	reserved := map[string]bool{}

	ok, err := IsSlotAvailable("2026-02-04", "09:45", loc, now, reserved)
	if err != nil {
		t.Fatalf("IsSlotAvailable error: %v", err)
	}
	if !ok {
		t.Fatalf("expected slot to be available")
	}
}
