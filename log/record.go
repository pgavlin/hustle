package log

import "time"

// KeyValue is a single attribute key-value pair.
type KeyValue struct {
	Key   string
	Value any
}

// Attrs is a slice of key-value pairs in insertion order.
type Attrs []KeyValue

// Get returns the value for key and whether it was found.
// Linear scan; suitable for the small attribute sets typical of log records.
func (a Attrs) Get(key string) (any, bool) {
	for _, kv := range a {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return nil, false
}

// Set appends a key-value pair, or replaces the value if the key already exists.
func (a *Attrs) Set(key string, value any) {
	s := *a
	for i := range s {
		if s[i].Key == key {
			s[i].Value = value
			return
		}
	}
	*a = append(s, KeyValue{Key: key, Value: value})
}

// LogRecord represents a single structured log entry.
type LogRecord struct {
	Time    time.Time
	Level   string
	Msg     string
	Attrs   Attrs
	RawJSON string
}

// defaultJSON is used by ParseRecord for backward compatibility.
var defaultJSON = &JSONFormat{}

// ParseRecord parses a single JSON log line into a LogRecord.
// This is a convenience wrapper around JSONFormat for backward compatibility.
func ParseRecord(line string) (LogRecord, error) {
	return defaultJSON.ParseRecord(line)
}
