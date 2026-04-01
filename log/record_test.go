package log

import (
	"testing"
	"time"
)

func TestParseRecord_ValidFullRecord(t *testing.T) {
	line := `{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"server started","port":8080,"host":"localhost"}`
	rec, err := ParseRecord(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Level != "INFO" {
		t.Errorf("level = %q, want %q", rec.Level, "INFO")
	}
	if rec.Msg != "server started" {
		t.Errorf("msg = %q, want %q", rec.Msg, "server started")
	}
	expectedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if !rec.Time.Equal(expectedTime) {
		t.Errorf("time = %v, want %v", rec.Time, expectedTime)
	}
	if rec.Attrs["port"] != float64(8080) {
		t.Errorf("attrs[port] = %v, want %v", rec.Attrs["port"], float64(8080))
	}
	if rec.Attrs["host"] != "localhost" {
		t.Errorf("attrs[host] = %v, want %v", rec.Attrs["host"], "localhost")
	}
	if rec.RawJSON != line {
		t.Errorf("rawJSON mismatch")
	}
}

func TestParseRecord_MissingStandardFields(t *testing.T) {
	line := `{"extra":"value"}`
	rec, err := ParseRecord(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rec.Time.IsZero() {
		t.Errorf("time should be zero, got %v", rec.Time)
	}
	if rec.Level != "" {
		t.Errorf("level should be empty, got %q", rec.Level)
	}
	if rec.Msg != "" {
		t.Errorf("msg should be empty, got %q", rec.Msg)
	}
	if rec.Attrs["extra"] != "value" {
		t.Errorf("attrs[extra] = %v, want %q", rec.Attrs["extra"], "value")
	}
}

func TestParseRecord_NestedAttrs(t *testing.T) {
	line := `{"time":"2024-01-15T10:30:00Z","level":"DEBUG","msg":"request","headers":{"content-type":"application/json"}}`
	rec, err := ParseRecord(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	headers, ok := rec.Attrs["headers"].(map[string]any)
	if !ok {
		t.Fatalf("attrs[headers] is not a map, got %T", rec.Attrs["headers"])
	}
	if headers["content-type"] != "application/json" {
		t.Errorf("headers[content-type] = %v, want %q", headers["content-type"], "application/json")
	}
}

func TestParseRecord_MalformedJSON(t *testing.T) {
	_, err := ParseRecord("not json at all")
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestParseRecord_EmptyObject(t *testing.T) {
	rec, err := ParseRecord("{}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rec.Attrs) != 0 {
		t.Errorf("attrs should be empty, got %v", rec.Attrs)
	}
}
