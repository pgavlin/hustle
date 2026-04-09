package log

import (
	"math"
	"strings"
	"time"
	"unsafe"

	"github.com/segmentio/encoding/json"
)

// JSONFormat parses JSON log lines, auto-detecting common field name variants
// for time, level, and message across slog, zap, zerolog, logrus, bunyan, and docker.
type JSONFormat struct{}

func (f *JSONFormat) Name() string { return "json" }

func (f *JSONFormat) ParseRecord(line string) (LogRecord, error) {
	var raw map[string]any

	// Zero-copy parse: get a []byte view of the string without copying,
	// then use DontCopyString so decoded strings reference the original line.
	b := unsafe.Slice(unsafe.StringData(line), len(line))
	if _, err := json.Parse(b, &raw, json.DontCopyString|json.DontCopyNumber); err != nil {
		return LogRecord{}, err
	}

	rec := LogRecord{
		RawJSON: line,
		Attrs:   make(map[string]any, len(raw)),
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

func isUpperASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 'a' && s[i] <= 'z' {
			return false
		}
	}
	return true
}

// commonLevels maps lowercase (and mixed-case) level strings to their canonical
// uppercase forms. Avoids strings.ToUpper allocations for the common case.
var commonLevels = map[string]string{
	"trace":   "TRACE",
	"debug":   "DEBUG",
	"info":    "INFO",
	"warn":    "WARN",
	"warning": "WARNING",
	"error":   "ERROR",
	"fatal":   "FATAL",
	"panic":   "PANIC",
}

// normalizeLevel returns the canonical uppercase form of a level string,
// avoiding an allocation for the common case.
func normalizeLevel(s string) string {
	if isUpperASCII(s) {
		return s
	}
	if l, ok := commonLevels[s]; ok {
		return l
	}
	return strings.ToUpper(s)
}

// parseLevel normalizes level values to uppercase strings.
// Handles string levels and bunyan numeric levels.
func parseLevel(v any) string {
	switch v := v.(type) {
	case string:
		return normalizeLevel(v)
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
