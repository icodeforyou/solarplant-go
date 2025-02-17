package hours

import (
	"testing"
	"time"
)

func TestDateHourString(t *testing.T) {
	dh := DateHour{Date: "2025-01-01", Hour: 5}
	expected := "2025-01-01 05"
	if s := dh.String(); s != expected {
		t.Errorf("String() expected %q, got %q", expected, s)
	}
}

func TestDateHourIsoString(t *testing.T) {
	dh := DateHour{Date: "2025-01-01", Hour: 15}
	expected := "2025-01-01T15:00:00Z"
	if s := dh.IsoString(); s != expected {
		t.Errorf("IsoString() expected %q, got %q", expected, s)
	}
}

func TestDateHourAdd(t *testing.T) {
	tests := []struct {
		name     string
		input    DateHour
		addHours int
		expected DateHour
	}{
		{
			name:     "add within same day",
			input:    DateHour{Date: "2025-01-01", Hour: 10},
			addHours: 2,
			expected: DateHour{Date: "2025-01-01", Hour: 12},
		},
		{
			name:     "add crossing midnight",
			input:    DateHour{Date: "2025-01-01", Hour: 23},
			addHours: 2,
			expected: DateHour{Date: "2025-01-02", Hour: 1},
		},
		{
			name:     "add negative hours (subtract)",
			input:    DateHour{Date: "2025-01-01", Hour: 1},
			addHours: -2,
			expected: DateHour{Date: "2024-12-31", Hour: 23},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.Add(tt.addHours)
			if result != tt.expected {
				t.Errorf("Add(%d) expected %+v, got %+v", tt.addHours, tt.expected, result)
			}
		})
	}
}

func TestDateHourSub(t *testing.T) {
	tests := []struct {
		name     string
		input    DateHour
		subHours int
		expected DateHour
	}{
		{
			name:     "sub within same day",
			input:    DateHour{Date: "2025-01-01", Hour: 10},
			subHours: 2,
			expected: DateHour{Date: "2025-01-01", Hour: 8},
		},
		{
			name:     "sub crossing midnight",
			input:    DateHour{Date: "2025-01-01", Hour: 0},
			subHours: 1,
			expected: DateHour{Date: "2024-12-31", Hour: 23},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.Sub(tt.subHours)
			if result != tt.expected {
				t.Errorf("Sub(%d) expected %+v, got %+v", tt.subHours, tt.expected, result)
			}
		})
	}
}

func TestDateHourIsZero(t *testing.T) {
	// A zero value DateHour should be recognized as zero.
	var dh DateHour
	if !dh.IsZero() {
		t.Errorf("expected a zero value DateHour to be zero")
	}
	// A non-zero DateHour (even with Hour 0) should not be considered zero if Date is non-empty.
	dh = DateHour{Date: "2025-01-01", Hour: 0}
	if dh.IsZero() {
		t.Errorf("expected a non-zero DateHour (non-empty Date) not to be zero")
	}
}

func TestFromTime(t *testing.T) {
	// Test a valid time.
	tm := time.Date(2025, time.January, 1, 15, 30, 0, 0, time.UTC)
	dh := FromTime(tm)
	expected := DateHour{Date: "2025-01-01", Hour: 15}
	if dh != expected {
		t.Errorf("FromTime() expected %+v, got %+v", expected, dh)
	}

	// Test with a zero time.
	var zero time.Time
	dhZero := FromTime(zero)
	if !dhZero.IsZero() {
		t.Errorf("FromTime() with zero time expected a zero DateHour")
	}
}

func TestFromNow(t *testing.T) {
	// Since FromNow() uses the current time, we capture the expected values.
	now := time.Now().UTC()
	dh := FromNow()
	expectedDate := now.Format("2006-01-02")
	expectedHour := now.Hour()

	if dh.Date != expectedDate {
		t.Errorf("FromNow() expected date %q, got %q", expectedDate, dh.Date)
	}
	if int(dh.Hour) != expectedHour {
		t.Errorf("FromNow() expected hour %d, got %d", expectedHour, dh.Hour)
	}
}

func TestFromMidnight(t *testing.T) {
	now := time.Now().UTC()
	dh := FromMidnight()
	expectedDate := now.Format("2006-01-02")
	if dh.Date != expectedDate {
		t.Errorf("FromMidnight() expected date %q, got %q", expectedDate, dh.Date)
	}
	if dh.Hour != 0 {
		t.Errorf("FromMidnight() expected hour 0, got %d", dh.Hour)
	}
}

func TestFromIso(t *testing.T) {
	// Test a valid RFC3339 string.
	isoStr := "2025-01-01T15:00:00Z"
	parsed := FromIso(isoStr)
	expected := time.Date(2025, time.January, 1, 15, 0, 0, 0, time.UTC)
	if !parsed.Equal(expected) {
		t.Errorf("FromIso() expected %v, got %v", expected, parsed)
	}

	// Test an invalid string returns a zero time.
	invalid := "not a valid iso date"
	parsedInvalid := FromIso(invalid)
	if !parsedInvalid.IsZero() {
		t.Errorf("FromIso() expected zero time for an invalid date string")
	}
}

func TestLocationStockholm(t *testing.T) {
	// Test with a winter date when Stockholm is normally at UTC+1.
	tmWinter := time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)
	stockholmTimeWinter := LocationStockholm(tmWinter)
	_, offsetWinter := stockholmTimeWinter.Zone()
	if offsetWinter != 3600 {
		t.Errorf("LocationStockholm() on winter date expected offset 3600 seconds, got %d", offsetWinter)
	}

	// Test with a summer date when Stockholm is normally at UTC+2.
	tmSummer := time.Date(2025, time.July, 1, 12, 0, 0, 0, time.UTC)
	stockholmTimeSummer := LocationStockholm(tmSummer)
	_, offsetSummer := stockholmTimeSummer.Zone()
	if offsetSummer != 7200 {
		t.Errorf("LocationStockholm() on summer date expected offset 7200 seconds, got %d", offsetSummer)
	}
}
