package hours

import (
	"fmt"
	"time"
)

const (
	dateLayout = "2006-01-02"
	hourLayout = "2006-01-02 15"
)

var (
	stockholmLoc *time.Location
	guiLocation  *time.Location = time.UTC
)

func init() {
	var err error
	stockholmLoc, err = time.LoadLocation("Europe/Stockholm")
	if err != nil {
		panic(fmt.Sprintf("failed to load Stockholm location: %v", err))
	}
}

func SetGuiTimezone(timezone string) error {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return fmt.Errorf("failed to load timezone %s: %v", timezone, err)
	}
	guiLocation = loc
	return nil
}

type DateHour struct {
	Date string
	Hour uint8
}

func (dh DateHour) String() string {
	return fmt.Sprintf("%s %02d", dh.Date, dh.Hour)
}

func (dh DateHour) LocalizedString() string {
	t, err := time.ParseInLocation(hourLayout, dh.String(), time.UTC)
	if err != nil {
		return dh.String()
	}
	localTime := t.In(guiLocation)
	return fmt.Sprintf("%s %02d", localTime.Format(dateLayout), localTime.Hour())
}

func (dh DateHour) IsoString() string {
	return fmt.Sprintf("%sT%02d:00:00Z", dh.Date, dh.Hour)
}

func (dh DateHour) Add(hours int) DateHour {
	t, err := time.ParseInLocation(hourLayout, dh.String(), time.UTC)
	if err != nil {
		return dh
	}

	t = t.Add(time.Duration(hours) * time.Hour)
	return DateHour{
		Date: t.Format(dateLayout),
		Hour: uint8(t.Hour()),
	}
}

func (dh DateHour) Sub(hours int) DateHour {
	return dh.Add(-hours)
}

func (dh DateHour) Compare(other DateHour) int {
	if dh == other {
		return 0
	}
	if dh.Date < other.Date {
		return -1
	}
	if dh.Date > other.Date {
		return 1
	}
	if dh.Hour < other.Hour {
		return -1
	}
	return 1
}

func (dh DateHour) IsZero() bool {
	return dh.Date == "" && dh.Hour == 0
}

func FromTime(t time.Time) DateHour {
	if t.IsZero() {
		return DateHour{}
	}
	t = t.UTC()
	return DateHour{
		Date: t.Format(dateLayout),
		Hour: uint8(t.Hour()),
	}
}

func FromNow() DateHour {
	now := time.Now().UTC()
	return DateHour{
		Date: now.Format(dateLayout),
		Hour: uint8(now.Hour()),
	}
}

func FromMidnight() DateHour {
	now := time.Now().UTC()
	return DateHour{
		Date: now.Format(dateLayout),
		Hour: 0,
	}
}

func FromIso(str string) time.Time {
	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func LocationStockholm(t time.Time) time.Time {
	return t.In(stockholmLoc)
}

func FormatTimeInGuiTimezone(t time.Time) string {
	return t.In(guiLocation).Format("2006-01-02 15:04:05")
}
