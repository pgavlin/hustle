package log

import (
	"testing"
	"time"
)

func TestJSONFormat_Slog(t *testing.T) {
	f := &JSONFormat{}
	rec, err := f.ParseRecord(`{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"hello","port":8080}`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "INFO" {
		t.Errorf("level = %q, want INFO", rec.Level)
	}
	if rec.Msg != "hello" {
		t.Errorf("msg = %q, want hello", rec.Msg)
	}
	if rec.Time.IsZero() {
		t.Error("time should not be zero")
	}
	if v, _ := rec.Attrs.Get("port"); v != float64(8080) {
		t.Errorf("port = %v, want 8080", v)
	}
}

func TestJSONFormat_Zap(t *testing.T) {
	f := &JSONFormat{}
	rec, err := f.ParseRecord(`{"ts":1705312200.123,"level":"info","msg":"request","duration":42}`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "INFO" {
		t.Errorf("level = %q, want INFO", rec.Level)
	}
	if rec.Msg != "request" {
		t.Errorf("msg = %q, want request", rec.Msg)
	}
	if rec.Time.IsZero() {
		t.Error("time should not be zero (unix ts)")
	}
	// ts=1705312200.123 should parse to roughly 2024-01-15
	if rec.Time.Year() != 2024 {
		t.Errorf("year = %d, want 2024", rec.Time.Year())
	}
}

func TestJSONFormat_Zerolog(t *testing.T) {
	f := &JSONFormat{}
	rec, err := f.ParseRecord(`{"level":"debug","time":"2024-01-15T10:30:00Z","message":"starting"}`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "DEBUG" {
		t.Errorf("level = %q, want DEBUG", rec.Level)
	}
	if rec.Msg != "starting" {
		t.Errorf("msg = %q, want starting", rec.Msg)
	}
}

func TestJSONFormat_Logrus(t *testing.T) {
	f := &JSONFormat{}
	rec, err := f.ParseRecord(`{"level":"warning","msg":"deprecated","time":"2024-01-15T10:30:00Z"}`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "WARNING" {
		t.Errorf("level = %q, want WARNING", rec.Level)
	}
	if rec.Msg != "deprecated" {
		t.Errorf("msg = %q, want deprecated", rec.Msg)
	}
}

func TestJSONFormat_Bunyan(t *testing.T) {
	f := &JSONFormat{}
	rec, err := f.ParseRecord(`{"v":0,"level":50,"time":"2024-01-15T10:30:00Z","msg":"error occurred","pid":1234}`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "ERROR" {
		t.Errorf("level = %q, want ERROR", rec.Level)
	}
	if rec.Msg != "error occurred" {
		t.Errorf("msg = %q, want 'error occurred'", rec.Msg)
	}
}

func TestJSONFormat_DockerJSON(t *testing.T) {
	f := &JSONFormat{}
	rec, err := f.ParseRecord(`{"log":"container output","stream":"stdout","time":"2024-01-15T10:30:00.123456Z"}`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Msg != "container output" {
		t.Errorf("msg = %q, want 'container output'", rec.Msg)
	}
	if rec.Time.IsZero() {
		t.Error("time should not be zero")
	}
}

func TestJSONFormat_GenericNDJSON(t *testing.T) {
	f := &JSONFormat{}
	rec, err := f.ParseRecord(`{"severity":"ERROR","timestamp":"2024-01-15T10:30:00Z","message":"fail","code":500}`)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "ERROR" {
		t.Errorf("level = %q, want ERROR", rec.Level)
	}
	if rec.Msg != "fail" {
		t.Errorf("msg = %q, want fail", rec.Msg)
	}
}

func TestJSONFormat_InvalidJSON(t *testing.T) {
	f := &JSONFormat{}
	_, err := f.ParseRecord("not json")
	if err == nil {
		t.Error("expected error")
	}
}

func TestJSONFormat_Name(t *testing.T) {
	f := &JSONFormat{}
	if f.Name() != "json" {
		t.Errorf("name = %q, want json", f.Name())
	}
}

// Ensure time package is used (it's imported for the zero-check helpers above).
var _ = time.Time{}
