package log

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// JSONFormat parses JSON log lines, auto-detecting common field name variants
// for time, level, and message across slog, zap, zerolog, logrus, bunyan, and docker.
type JSONFormat struct{}

func (f *JSONFormat) Name() string { return "json" }

func (f *JSONFormat) ParseRecord(line string) (LogRecord, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return LogRecord{}, fmt.Errorf("invalid JSON: %w", err)
	}

	rec := LogRecord{
		RawJSON: line,
		Attrs:   make(map[string]any),
	}

	// Extract time from known field names
	for _, key := range []string{"time", "ts", "timestamp", "@timestamp"} {
		if v, ok := raw[key]; ok {
			rec.Time = parseTime(v)
			delete(raw, key)
			break
		}
	}

	// Extract level from known field names
	for _, key := range []string{"level", "severity", "lvl"} {
		if v, ok := raw[key]; ok {
			rec.Level = parseLevel(v)
			delete(raw, key)
			break
		}
	}

	// Extract message from known field names
	for _, key := range []string{"msg", "message", "log"} {
		if v, ok := raw[key]; ok {
			if s, ok := v.(string); ok {
				rec.Msg = s
			}
			delete(raw, key)
			break
		}
	}

	// Everything remaining goes into Attrs
	for k, v := range raw {
		rec.Attrs[k] = v
	}

	return rec, nil
}

// parseTime handles both string (RFC3339) and numeric (unix timestamp) time values.
func parseTime(v any) time.Time {
	switch v := v.(type) {
	case string:
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t
		}
	case float64:
		// Unix timestamp (zap style): seconds with fractional part
		sec, frac := math.Modf(v)
		return time.Unix(int64(sec), int64(frac*1e9))
	}
	return time.Time{}
}

// parseLevel normalizes level values to uppercase strings.
// Handles string levels and bunyan numeric levels.
func parseLevel(v any) string {
	switch v := v.(type) {
	case string:
		return strings.ToUpper(v)
	case float64:
		// Bunyan numeric levels
		switch {
		case v <= 10:
			return "TRACE"
		case v <= 20:
			return "DEBUG"
		case v <= 30:
			return "INFO"
		case v <= 40:
			return "WARN"
		case v <= 50:
			return "ERROR"
		default:
			return "FATAL"
		}
	}
	return ""
}
