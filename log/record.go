package log

import (
	"encoding/json"
	"fmt"
	"time"
)

// LogRecord represents a single slog JSON log entry.
type LogRecord struct {
	Time    time.Time
	Level   string
	Msg     string
	Attrs   map[string]any
	RawJSON string
}

// ParseRecord parses a single JSON log line into a LogRecord.
func ParseRecord(line string) (LogRecord, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return LogRecord{}, fmt.Errorf("invalid JSON: %w", err)
	}

	rec := LogRecord{
		RawJSON: line,
		Attrs:   make(map[string]any),
	}

	if v, ok := raw["time"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			rec.Time = t
		}
	}
	if v, ok := raw["level"].(string); ok {
		rec.Level = v
	}
	if v, ok := raw["msg"].(string); ok {
		rec.Msg = v
	}

	for k, v := range raw {
		switch k {
		case "time", "level", "msg":
			continue
		default:
			rec.Attrs[k] = v
		}
	}

	return rec, nil
}
