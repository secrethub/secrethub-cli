package secrethub

import (
	"time"

	"fmt"

	units "github.com/docker/go-units"
)

// TimeFormatter can format a time to a string.
type TimeFormatter interface {
	Format(t time.Time) string
}

// NewTimeFormatter creates a new timeFormatter.
func NewTimeFormatter(timestamps bool) TimeFormatter {
	timeFormatter := timeFormatter(timestamps)
	return &timeFormatter
}

// NewTimestampFormatter is a convenience function to create a TimeFormatter that uses timestamps.
func NewTimestampFormatter() TimeFormatter {
	return NewTimeFormatter(true)
}

type timeFormatter bool

// Format returns a string representation of the time.
func (tf timeFormatter) Format(t time.Time) string {
	if tf {
		return t.Format(time.RFC3339)
	}
	return fmt.Sprintf("%s ago", units.HumanDuration(time.Now().UTC().Sub(t.UTC())))
}
