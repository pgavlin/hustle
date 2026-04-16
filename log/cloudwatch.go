package log

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// CloudWatchFormat parses AWS CloudWatch Logs output.
// Handles both `aws logs filter-log-events` JSON output (document with events array)
// and `aws logs tail` line-by-line output (timestamp + stream + message).
type CloudWatchFormat struct{}

func (f *CloudWatchFormat) Name() string { return "cloudwatch" }

func (f *CloudWatchFormat) IsDocumentFormat() {}

// ParseRecord handles a single `aws logs tail` line:
// 2024-01-15T10:30:00+00:00 log-group/log-stream message...
func (f *CloudWatchFormat) ParseRecord(line string) (LogRecord, error) {
	// Try aws logs tail format: timestamp stream message
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 3 {
		return LogRecord{}, fmt.Errorf("not a cloudwatch tail line")
	}

	ts, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return LogRecord{}, fmt.Errorf("not a cloudwatch tail line: %w", err)
	}

	stream := parts[1]
	message := parts[2]

	// Try to parse the message with inner format auto-detection
	rec, innerFormat := parseInnerMessage(message)
	if rec.Time.IsZero() {
		rec.Time = ts
	}
	rec.RawJSON = line
	rec.Attrs.Set("log_stream", stream)
	if innerFormat != "" {
		rec.Attrs.Set("inner_format", innerFormat)
	}

	return rec, nil
}

// LoadCloudWatchJSON reads a complete CloudWatch JSON document (from aws logs
// filter-log-events or get-log-events) and returns parsed records.
func LoadCloudWatchJSON(r io.Reader) ([]LogRecord, int, Format, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("read cloudwatch json: %w", err)
	}

	// Try to parse as CloudWatch events document
	var doc cloudWatchDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, 0, nil, fmt.Errorf("parse cloudwatch json: %w", err)
	}

	if len(doc.Events) == 0 {
		return nil, 0, &CloudWatchFormat{}, nil
	}

	var records []LogRecord
	skipped := 0

	for _, evt := range doc.Events {
		rec, _ := parseInnerMessage(evt.Message)

		// Use CloudWatch timestamp if inner format didn't provide one
		if rec.Time.IsZero() && evt.Timestamp != 0 {
			rec.Time = time.UnixMilli(evt.Timestamp)
		}

		rec.RawJSON = evt.Message

		// Merge CloudWatch metadata
		if evt.LogStreamName != "" {
			rec.Attrs.Set("log_stream", evt.LogStreamName)
		}
		if evt.EventID != "" {
			rec.Attrs.Set("event_id", evt.EventID)
		}

		records = append(records, rec)
	}

	return records, skipped, &CloudWatchFormat{}, nil
}

// cloudWatchDoc represents the JSON output of aws logs filter-log-events / get-log-events.
type cloudWatchDoc struct {
	Events []cloudWatchEvent `json:"events"`
}

type cloudWatchEvent struct {
	LogStreamName string `json:"logStreamName"`
	Timestamp     int64  `json:"timestamp"`
	Message       string `json:"message"`
	IngestionTime int64  `json:"ingestionTime"`
	EventID       string `json:"eventId"`
}

// parseInnerMessage tries to parse a message string with known formats.
// Returns the parsed record and the format name, or a minimal record with
// the message as Msg if no format matches.
func parseInnerMessage(message string) (LogRecord, string) {
	// Trim trailing newline (CloudWatch often adds one)
	message = strings.TrimRight(message, "\n\r")

	// Try formats in detection priority order (same as Formats, minus cloudwatch)
	innerFormats := []Format{
		&JSONFormat{},
		&GlogFormat{},
		&CLFFormat{},
		&LogfmtFormat{},
	}

	for _, f := range innerFormats {
		if rec, err := f.ParseRecord(message); err == nil {
			return rec, f.Name()
		}
	}

	// No format matched — treat the whole message as plain text
	return LogRecord{
		Msg:   message,
		Attrs: make(Attrs, 0, 4),
	}, ""
}
