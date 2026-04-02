package log

import (
	"strings"
	"testing"
	"time"
)

func TestCloudWatch_TailFormatJSON(t *testing.T) {
	f := &CloudWatchFormat{}
	rec, err := f.ParseRecord(`2024-01-15T10:30:00+00:00 my-group/my-stream/abc123 {"level":"INFO","msg":"hello","port":8080}`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "INFO" {
		t.Errorf("level = %q, want INFO", rec.Level)
	}
	if rec.Msg != "hello" {
		t.Errorf("msg = %q, want hello", rec.Msg)
	}
	if rec.Attrs["port"] != float64(8080) {
		t.Errorf("port = %v, want 8080", rec.Attrs["port"])
	}
	if rec.Attrs["log_stream"] != "my-group/my-stream/abc123" {
		t.Errorf("log_stream = %v", rec.Attrs["log_stream"])
	}
	if rec.Time.Year() != 2024 {
		t.Errorf("time = %v", rec.Time)
	}
}

func TestCloudWatch_TailFormatLogfmt(t *testing.T) {
	f := &CloudWatchFormat{}
	rec, err := f.ParseRecord(`2024-01-15T10:30:00+00:00 my-stream level=error msg="db timeout" retries=3`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "ERROR" {
		t.Errorf("level = %q, want ERROR", rec.Level)
	}
	if rec.Msg != "db timeout" {
		t.Errorf("msg = %q", rec.Msg)
	}
}

func TestCloudWatch_TailFormatPlainText(t *testing.T) {
	f := &CloudWatchFormat{}
	rec, err := f.ParseRecord(`2024-01-15T10:30:00+00:00 my-stream This is just a plain text log line`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Msg != "This is just a plain text log line" {
		t.Errorf("msg = %q", rec.Msg)
	}
}

func TestCloudWatch_TailFormatInvalid(t *testing.T) {
	f := &CloudWatchFormat{}
	_, err := f.ParseRecord("not a cloudwatch line at all")
	if err == nil {
		t.Error("expected error")
	}
}

func TestCloudWatch_JSONDocument(t *testing.T) {
	doc := `{
		"events": [
			{
				"logStreamName": "my-service/abc123",
				"timestamp": 1705312200000,
				"message": "{\"time\":\"2024-01-15T10:30:00Z\",\"level\":\"INFO\",\"msg\":\"hello\",\"port\":8080}",
				"ingestionTime": 1705312201000,
				"eventId": "evt-001"
			},
			{
				"logStreamName": "my-service/abc123",
				"timestamp": 1705312201000,
				"message": "{\"time\":\"2024-01-15T10:30:01Z\",\"level\":\"ERROR\",\"msg\":\"db timeout\"}",
				"ingestionTime": 1705312202000,
				"eventId": "evt-002"
			}
		]
	}`

	records, skipped, format, err := LoadCloudWatchJSON(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	if format.Name() != "cloudwatch" {
		t.Errorf("format = %q, want cloudwatch", format.Name())
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	if skipped != 0 {
		t.Errorf("skipped = %d, want 0", skipped)
	}

	// First record
	if records[0].Level != "INFO" {
		t.Errorf("records[0].level = %q, want INFO", records[0].Level)
	}
	if records[0].Msg != "hello" {
		t.Errorf("records[0].msg = %q, want hello", records[0].Msg)
	}
	if records[0].Attrs["port"] != float64(8080) {
		t.Errorf("records[0].port = %v, want 8080", records[0].Attrs["port"])
	}
	if records[0].Attrs["log_stream"] != "my-service/abc123" {
		t.Errorf("records[0].log_stream = %v", records[0].Attrs["log_stream"])
	}
	if records[0].Attrs["event_id"] != "evt-001" {
		t.Errorf("records[0].event_id = %v", records[0].Attrs["event_id"])
	}

	// Second record
	if records[1].Level != "ERROR" {
		t.Errorf("records[1].level = %q, want ERROR", records[1].Level)
	}
}

func TestCloudWatch_JSONDocumentWithTimestampFallback(t *testing.T) {
	// Inner message has no timestamp — should use CloudWatch timestamp
	doc := `{
		"events": [
			{
				"timestamp": 1705312200000,
				"message": "{\"level\":\"INFO\",\"msg\":\"no time in message\"}"
			}
		]
	}`

	records, _, _, err := LoadCloudWatchJSON(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if records[0].Time.IsZero() {
		t.Error("time should not be zero — should fall back to CloudWatch timestamp")
	}
	// 1705312200000 ms = 2024-01-15T10:30:00Z
	if records[0].Time.Year() != 2024 || records[0].Time.Month() != time.January {
		t.Errorf("time = %v, expected 2024-01", records[0].Time)
	}
}

func TestCloudWatch_JSONDocumentEmpty(t *testing.T) {
	doc := `{"events": []}`
	records, _, _, err := LoadCloudWatchJSON(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Errorf("got %d records, want 0", len(records))
	}
}

func TestCloudWatch_Name(t *testing.T) {
	f := &CloudWatchFormat{}
	if f.Name() != "cloudwatch" {
		t.Errorf("name = %q, want cloudwatch", f.Name())
	}
}
