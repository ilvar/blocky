//go:generate go tool go-enum -f=$GOFILE --marshal --names --values
package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/0xERR0R/blocky/log"
	"github.com/sirupsen/logrus"
)

// ScheduleAction defines the action to take during a scheduled window.
// ENUM(
// blockGroups // Add specified groups to blocking check during window
// blockAll // Block ALL DNS requests during window (returns NXDOMAIN)
// allowGroupsOnly // Only allow domains matching specified allowlist groups; block everything else
// )
type ScheduleAction int

// TimeOfDay represents a time in HH:MM format
type TimeOfDay struct {
	Hour   int
	Minute int
}

// ToMinutes returns the total minutes since midnight
func (t TimeOfDay) ToMinutes() int {
	return t.Hour*60 + t.Minute
}

// String implements fmt.Stringer
func (t TimeOfDay) String() string {
	return fmt.Sprintf("%02d:%02d", t.Hour, t.Minute)
}

// UnmarshalText implements encoding.TextUnmarshaler
func (t *TimeOfDay) UnmarshalText(data []byte) error {
	input := string(data)
	parts := strings.Split(input, ":")

	if len(parts) != 2 {
		return fmt.Errorf("invalid time format '%s': expected HH:MM", input)
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return fmt.Errorf("invalid hour in time '%s': must be 00-23", input)
	}

	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return fmt.Errorf("invalid minute in time '%s': must be 00-59", input)
	}

	t.Hour = hour
	t.Minute = minute

	return nil
}

// MarshalText implements encoding.TextMarshaler
func (t TimeOfDay) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// DayPattern represents which days a schedule applies to
type DayPattern struct {
	days map[time.Weekday]bool
}

// Contains checks if a weekday is in the pattern
func (d DayPattern) Contains(day time.Weekday) bool {
	return d.days[day]
}

// String implements fmt.Stringer
func (d DayPattern) String() string {
	if len(d.days) == 7 {
		return "everyday"
	}

	// Check for weekdays pattern
	isWeekdays := true
	for day := time.Monday; day <= time.Friday; day++ {
		if !d.days[day] {
			isWeekdays = false

			break
		}
	}

	if isWeekdays && !d.days[time.Saturday] && !d.days[time.Sunday] && len(d.days) == 5 {
		return "weekdays"
	}

	// Check for weekends pattern
	if d.days[time.Saturday] && d.days[time.Sunday] && len(d.days) == 2 {
		return "weekends"
	}

	// List specific days
	var dayNames []string
	for day := time.Sunday; day <= time.Saturday; day++ {
		if d.days[day] {
			dayNames = append(dayNames, strings.ToLower(day.String()))
		}
	}

	return strings.Join(dayNames, ", ")
}

// parseDayName converts a day name string to time.Weekday
func parseDayName(name string) (time.Weekday, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "sunday", "sun":
		return time.Sunday, nil
	case "monday", "mon":
		return time.Monday, nil
	case "tuesday", "tue":
		return time.Tuesday, nil
	case "wednesday", "wed":
		return time.Wednesday, nil
	case "thursday", "thu":
		return time.Thursday, nil
	case "friday", "fri":
		return time.Friday, nil
	case "saturday", "sat":
		return time.Saturday, nil
	default:
		return 0, fmt.Errorf("unknown day name '%s'", name)
	}
}

// UnmarshalText implements encoding.TextUnmarshaler
func (d *DayPattern) UnmarshalText(data []byte) error {
	input := strings.TrimSpace(string(data))
	d.days = make(map[time.Weekday]bool)

	switch strings.ToLower(input) {
	case "everyday":
		for day := time.Sunday; day <= time.Saturday; day++ {
			d.days[day] = true
		}

		return nil
	case "weekdays":
		for day := time.Monday; day <= time.Friday; day++ {
			d.days[day] = true
		}

		return nil
	case "weekends":
		d.days[time.Saturday] = true
		d.days[time.Sunday] = true

		return nil
	}

	// Parse as comma-separated list of days
	for part := range strings.SplitSeq(input, ",") {
		day, err := parseDayName(part)
		if err != nil {
			return fmt.Errorf("invalid day pattern '%s': %w", input, err)
		}

		d.days[day] = true
	}

	if len(d.days) == 0 {
		return fmt.Errorf("invalid day pattern '%s': no valid days", input)
	}

	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler to handle both string and array formats
func (d *DayPattern) UnmarshalYAML(unmarshal func(any) error) error {
	// Try string first
	var str string
	if err := unmarshal(&str); err == nil {
		return d.UnmarshalText([]byte(str))
	}

	// Try array of strings
	var arr []string
	if err := unmarshal(&arr); err != nil {
		return fmt.Errorf("day pattern must be a string or array of strings: %w", err)
	}

	d.days = make(map[time.Weekday]bool)

	for _, name := range arr {
		day, err := parseDayName(name)
		if err != nil {
			return err
		}

		d.days[day] = true
	}

	if len(d.days) == 0 {
		return fmt.Errorf("day pattern array is empty")
	}

	return nil
}

// ScheduleWindow represents a time window when a schedule is active
type ScheduleWindow struct {
	Start TimeOfDay  `yaml:"start"`
	End   TimeOfDay  `yaml:"end"`
	Days  DayPattern `yaml:"days"`
}

// IsActive returns true if the schedule window is active at the given time
func (w *ScheduleWindow) IsActive(t time.Time) bool {
	if !w.Days.Contains(t.Weekday()) {
		return false
	}

	current := t.Hour()*60 + t.Minute()
	start := w.Start.ToMinutes()
	end := w.End.ToMinutes()

	if end <= start {
		// Overnight window (e.g., 22:00 to 06:00)
		return current >= start || current < end
	}

	return current >= start && current < end
}

// ClientSchedule represents a parental control schedule for a client
type ClientSchedule struct {
	Groups   []string         `yaml:"groups"`
	Action   ScheduleAction   `yaml:"action"`
	Schedule []ScheduleWindow `yaml:"schedule"`
}

// ParentalControl holds the parental control configuration
type ParentalControl struct {
	Timezone string                      `yaml:"timezone"`
	Clients  map[string][]ClientSchedule `yaml:"clients"`
}

// IsEnabled implements config.Configurable
func (c *ParentalControl) IsEnabled() bool {
	return len(c.Clients) > 0
}

// LogConfig implements config.Configurable
func (c *ParentalControl) LogConfig(logger *logrus.Entry) {
	if c.Timezone != "" {
		logger.Infof("timezone = %s", c.Timezone)
	} else {
		logger.Info("timezone = system default")
	}

	logger.Info("clients:")

	for client, schedules := range c.Clients {
		log.WithIndent(logger, "  ", func(logger *logrus.Entry) {
			logger.Infof("%s:", client)

			for i, schedule := range schedules {
				log.WithIndent(logger, "  ", func(logger *logrus.Entry) {
					logger.Infof("schedule %d:", i+1)
					logger.Infof("  action = %s", schedule.Action)

					if len(schedule.Groups) > 0 {
						logger.Infof("  groups = %v", schedule.Groups)
					}

					for j, window := range schedule.Schedule {
						logger.Infof("  window %d: %s-%s on %s", j+1, window.Start, window.End, window.Days)
					}
				})
			}
		})
	}
}
