package twamp

import (
	"time"
)

type Timestamp struct {
	Seconds  uint32
	Fraction uint32
}

// Converts a UNIX epoch time time.Time object into an RFC 1305 compliant time.
func NewTimestamp(t time.Time) Timestamp {
	// convert epoch from 1970 to 1900 per RFC 1305
	t = t.AddDate(70, 0, 0)
	return Timestamp{
		Seconds:  uint32(t.Unix()),
		Fraction: uint32(t.Nanosecond()),
	}
}

func ConvertTimestamp(sec uint32, nsec uint32) time.Time {
	t := time.Unix(int64(sec), int64(nsec))
	t = t.AddDate(-70, 0, 0) // convert epoch from 1970 to 1900 per RFC 1305
	return t
}

// Return a time.Time object representing Unix Epoch time since January 1st, 1970.
func (t *Timestamp) GetTime() time.Time {
	// convert epoch from 1900 back to 1970
	return time.Unix(int64(t.Seconds), int64(t.Fraction)).AddDate(-70, 0, 0)
}

func (t *Timestamp) String() string {
	return t.GetTime().String()
}
