package log

import "time"

// LogRecord represents a single structured log entry.
type LogRecord struct {
	Time    time.Time
	Level   string
	Msg     string
	Attrs   map[string]any
	RawJSON string
}

// defaultJSON is used by ParseRecord for backward compatibility.
var defaultJSON = &JSONFormat{}

// ParseRecord parses a single JSON log line into a LogRecord.
// This is a convenience wrapper around JSONFormat for backward compatibility.
func ParseRecord(line string) (LogRecord, error) {
	return defaultJSON.ParseRecord(line)
}
